// This file is part of Testaroli project, available at https://github.com/qrdl/testaroli
// Copyright (c) 2024-2026 Ilya Caramishev. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at https://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build (unix || windows) && (amd64 || arm64)

package testaroli

import (
	"reflect"
	"strings"
	"unsafe"
)

// shapedScanRange is how many bytes of a generic trampoline's code
// findShapedImpl scans looking for the call to the shaped implementation. The
// call sits in the trampoline's short body, well within this range.
const shapedScanRange = 256

// isGenericName reports whether a runtime function name denotes a generic
// instantiation whose shaped implementation can be patched. The Go compiler
// embeds "[...]" in the name of both the per-type instantiation trampoline and
// the shaped implementation:
//
//	generic function: pkg.Func[...]
//	method on a generic receiver type: pkg.Type[...].Method
//
// so the marker is at the end for functions but in the middle for methods (a
// pointer receiver adds a closing paren, pkg.(*Type[...]).Method).
//
// The marker alone is not a sufficient test, though: compiler-generated helpers
// defined inside a generic function inherit its name, for example
// pkg.Func[...].func1 for a closure, pkg.Func[...].deferwrap1 for a deferred
// call and pkg.Type[...].Method-fm for a method value. Those are ordinary entry
// points that receive no type dictionary (and whose bodies are far too small to
// host the shim), so anything other than a plain method selector after the
// marker is rejected here and handled by the regular strategy.
func isGenericName(name string) bool {
	i := strings.LastIndex(name, "[...]")
	if i < 0 {
		return false
	}
	rest := methodSelectorOf(name[i+len("[...]"):])
	if rest == "" {
		return true // plain generic function: pkg.Func[...]
	}
	sel, isSelector := strings.CutPrefix(rest, ".")
	return isSelector && isMethodSelector(sel)
}

// methodSelectorOf returns what follows the "[...]" marker in a generic
// instantiation name, without the closing paren that a pointer receiver adds:
// both pkg.Type[...].Method and pkg.(*Type[...]).Method yield ".Method".
func methodSelectorOf(afterMarker string) string {
	return strings.TrimPrefix(afterMarker, ")")
}

// isMethodSelector reports whether <s> is a plain method name rather than a
// compiler-generated wrapper. Closures are named funcN (nested ones funcN.M),
// deferred and go-statement wrappers deferwrapN/gowrapN, and method values carry
// an "-fm" suffix - none of them take a hidden type dictionary. An unexported
// method that happens to be named like a wrapper is misclassified as one, which
// only costs it the shaped patch (direct calls then run the original).
func isMethodSelector(s string) bool {
	if s == "" || strings.ContainsAny(s, ".-") {
		return false
	}
	for _, wrapper := range [...]string{"func", "deferwrap", "gowrap"} {
		if digits, ok := strings.CutPrefix(s, wrapper); ok && isDigits(digits) {
			return false
		}
	}
	return true
}

// isDigits reports whether <s> is a non-empty run of decimal digits.
func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// isGenericMethodName reports whether a generic instantiation name is a method
// on a generic receiver type (pkg.Type[...].Method) rather than a plain generic
// function (pkg.Func[...]) - the marker is followed by a method selector.
func isGenericMethodName(name string) bool {
	i := strings.LastIndex(name, "[...]")
	return i >= 0 && strings.HasPrefix(methodSelectorOf(name[i+len("[...]"):]), ".")
}

// dictDropSlot returns the integer-register argument slot that holds the hidden
// type dictionary for a shaped generic implementation, given the org/mock
// signature <sig>. For a plain function the dictionary is the first argument
// (slot 0). For a method on a generic type the receiver comes first and the
// dictionary immediately follows it, so the slot equals the number of integer
// registers the receiver occupies. ok is false if that footprint cannot be
// determined (an exotic, stack-passed receiver), in which case the caller must
// not attempt the shaped patch.
func dictDropSlot(sig reflect.Type, isMethod bool) (slot int, ok bool) {
	if !isMethod {
		return 0, true
	}
	if sig.NumIn() == 0 {
		return 0, false
	}
	return intRegs(sig.In(0)) // receiver is the first parameter
}

// intRegs returns the number of integer registers a value of type t occupies
// under Go's register-based calling convention, and whether that count is
// well-defined (false for types that are passed on the stack, such as
// multi-element arrays).
func intRegs(t reflect.Type) (int, bool) {
	ints, _, ok := regFootprint(t)
	return ints, ok
}

// regFootprint returns how many integer and floating-point registers a value of
// type t occupies under Go's register-based calling convention. ok is false for
// types that are passed on the stack rather than in registers (such as
// multi-element arrays), which the generic shim cannot reshape.
func regFootprint(t reflect.Type) (ints, floats int, ok bool) {
	switch t.Kind() {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Uintptr, reflect.Pointer, reflect.UnsafePointer,
		reflect.Chan, reflect.Map, reflect.Func:
		return 1, 0, true
	case reflect.Float32, reflect.Float64:
		return 0, 1, true
	case reflect.Complex64, reflect.Complex128:
		return 0, 2, true // real + imaginary
	case reflect.String:
		return 2, 0, true // data pointer + length
	case reflect.Slice:
		return 3, 0, true // data pointer + length + capacity
	case reflect.Interface:
		return 2, 0, true // type word + data word
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			fi, ff, fok := regFootprint(t.Field(i).Type)
			if !fok {
				return 0, 0, false
			}
			ints += fi
			floats += ff
		}
		return ints, floats, true
	case reflect.Array:
		switch t.Len() {
		case 0:
			return 0, 0, true
		case 1:
			return regFootprint(t.Elem())
		}
		// a multi-element array falls through - it is passed on the stack
	}
	return 0, 0, false
}

// shimFitsSignature reports whether the generic shim can faithfully reshape a
// call to a function/method with signature sig, and how many integer registers
// that signature occupies. The shim only shifts integer registers, so it works
// only when every argument - plus the hidden dictionary, which claims one extra
// integer register - is passed in registers. If any argument would spill to the
// stack (because it is a stack-passed type or the register budget is exceeded),
// the shim cannot reproduce the argument layout the mock expects, and the shaped
// implementation must be left unpatched.
func shimFitsSignature(sig reflect.Type) (intWords int, ok bool) {
	totalFloat := 0
	for i := 0; i < sig.NumIn(); i++ {
		ints, floats, ok := regFootprint(sig.In(i))
		if !ok {
			return 0, false
		}
		intWords += ints
		totalFloat += floats
	}
	if intWords+1 > intRegBudget || totalFloat > floatRegBudget {
		return 0, false
	}
	return intWords, true
}

// shapedImpls caches the shaped implementation of a generic trampoline (nil when
// it could not be located). The lookup scans the trampoline's code for the call
// it makes, so the address is resolved once, before the trampoline is ever
// patched: scanning it later means scanning a prologue overwritten with a jump
// and a shim, which may hide the call or be mistaken for one. Function addresses
// never move, so an entry stays valid for the lifetime of the process.
var shapedImpls = map[unsafe.Pointer]unsafe.Pointer{}

// shapedImplOf returns the shaped implementation of the generic trampoline at
// <tramp>, or nil if it cannot be located.
func shapedImplOf(tramp unsafe.Pointer) unsafe.Pointer {
	if shaped, cached := shapedImpls[tramp]; cached {
		return shaped
	}
	shaped := findShapedImpl(tramp)
	shapedImpls[tramp] = shaped
	return shaped
}

// applyOverride installs the override described by <e>, choosing the generic or
// the regular strategy depending on the kind of function being overridden.
func applyOverride(e *Expect) {
	// A generic instantiation is intercepted at both its trampoline and its
	// shaped implementation only when that implementation was located and the
	// shim can reshape the shaped calling convention (see shimFitsSignature).
	// Otherwise fall back to a plain trampoline patch, which still intercepts
	// reference-form calls exactly like a regular function; direct calls then run
	// the original (never corrupted). The fallback also leaves the trampoline
	// body untouched, which matters when the function is not the generic
	// trampoline we took it for and may be too short to host the shim.
	if e.isGeneric && e.canShim && e.shapedAddr != nil {
		e.orgPrologue, e.shapedPrologue = overrideGeneric(
			e.orgAddr, e.mockAddr, e.shapedAddr, e.dropSlot, e.intWords)
		return
	}
	e.orgPrologue = override(e.orgAddr, e.mockAddr)
}

// resetExpect restores the original state of an overridden function, including
// the shaped implementation for generic functions.
func resetExpect(e *Expect) {
	reset(e.orgAddr, e.orgPrologue)
	if e.shapedAddr != nil && len(e.shapedPrologue) > 0 {
		reset(e.shapedAddr, e.shapedPrologue)
	}
}

// overrideGeneric intercepts every call form of a generic function.
//
// A generic function can be reached two ways after instantiation:
//
//   - Reference form (fn := pointer[int]; fn(x)) goes through the instantiation
//     trampoline, so patching the trampoline entry with a jump to the mock is
//     enough, exactly like a regular function.
//   - Direct form (pointer[int](x) or pointer(x)) is compiled into a direct
//     call to the shaped implementation, bypassing the trampoline. That
//     implementation is invoked with an extra leading dictionary pointer, so it
//     is routed through a small shim that drops the dictionary and tail-calls
//     the mock.
//
// The shim is written into the body of the trampoline itself, right after the
// jump to the mock. The trampoline body is dead code once its entry jumps away,
// and it is larger than the shim - which holds no more than one register move
// per integer argument - so no separate executable memory has to be allocated;
// the shim is restored together with the trampoline.
//
// <shaped> is the address of the shaped implementation and must not be nil (the
// caller falls back to a plain trampoline patch when it could not be located).
// It returns the saved original bytes of the trampoline and of the shaped
// implementation, for later reset.
func overrideGeneric(orgPointer, mockPointer, shaped unsafe.Pointer, dropSlot, intWords int) (trampSaved, shapedSaved []byte) {
	jmpToMock := buildJump(orgPointer, mockPointer)
	shim := buildShim(uintptr(mockPointer), dropSlot, intWords)
	combined := make([]byte, 0, len(jmpToMock)+len(shim))
	combined = append(combined, jmpToMock...)
	combined = append(combined, shim...)

	trampSaved = saveCode(orgPointer, len(combined))
	replacePrologue(orgPointer, combined)
	flushICache(orgPointer, len(combined))

	shimAddr := unsafe.Pointer(uintptr(orgPointer) + uintptr(len(jmpToMock)))
	jmpToShim := buildJump(shaped, shimAddr)
	shapedSaved = saveCode(shaped, len(jmpToShim))
	replacePrologue(shaped, jmpToShim)
	flushICache(shaped, len(jmpToShim))

	return trampSaved, shapedSaved
}

// saveCode returns a copy of <n> bytes of machine code starting at <ptr>.
func saveCode(ptr unsafe.Pointer, n int) []byte {
	dst := make([]byte, n)
	copy(dst, unsafe.Slice((*byte)(ptr), n))
	return dst
}

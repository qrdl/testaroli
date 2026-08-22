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
	"strings"
	"unsafe"
)

// isGenericName reports whether a runtime function name denotes a generic
// instantiation. The Go compiler names both the per-type instantiation
// trampoline and the shaped implementation of a generic function with a
// trailing "[...]", while regular functions, methods and closures never carry
// that suffix.
func isGenericName(name string) bool {
	return strings.HasSuffix(name, "[...]")
}

// applyOverride installs the override described by <e>, choosing the generic or
// the regular strategy depending on the kind of function being overridden.
func applyOverride(e *Expect) {
	if e.isGeneric {
		e.shapedAddr, e.orgPrologue, e.shapedPrologue = overrideGeneric(e.orgAddr, e.mockAddr)
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
// and it is comfortably larger than the shim, so no separate executable memory
// has to be allocated - the shim is restored together with the trampoline.
//
// It returns the shaped implementation address (nil if it could not be located,
// in which case only the reference form is intercepted) and the saved original
// bytes of the trampoline and the shaped implementation, for later reset.
func overrideGeneric(orgPointer, mockPointer unsafe.Pointer) (shaped unsafe.Pointer, trampSaved, shapedSaved []byte) {
	// Locate the shaped implementation before overwriting the trampoline, as
	// the search scans the trampoline's original code for the call to it.
	shaped = findShapedImpl(orgPointer)

	jmpToMock := buildJump(orgPointer, mockPointer)
	shim := buildShim(uintptr(mockPointer))
	combined := make([]byte, 0, len(jmpToMock)+len(shim))
	combined = append(combined, jmpToMock...)
	combined = append(combined, shim...)

	trampSaved = saveCode(orgPointer, len(combined))
	replacePrologue(orgPointer, combined)
	flushICache(orgPointer, len(combined))

	if shaped == nil {
		return shaped, trampSaved, nil
	}

	shimAddr := unsafe.Pointer(uintptr(orgPointer) + uintptr(len(jmpToMock)))
	jmpToShim := buildJump(shaped, shimAddr)
	shapedSaved = saveCode(shaped, len(jmpToShim))
	replacePrologue(shaped, jmpToShim)
	flushICache(shaped, len(jmpToShim))

	return shaped, trampSaved, shapedSaved
}

// saveCode returns a copy of <n> bytes of machine code starting at <ptr>.
func saveCode(ptr unsafe.Pointer, n int) []byte {
	dst := make([]byte, n)
	copy(dst, unsafe.Slice((*byte)(ptr), n))
	return dst
}

package testaroli

import (
	"reflect"
	"testing"
	"unsafe"
)

// covData is a chunk of non-code memory used to drive findShapedImpl down its
// "address is not a function" path (a read-only lookup, no patching). It is as
// large as the range the scanner reads, so the lookup stays inside it.
var covData [shapedScanRange]byte

//go:noinline
func covCallee() int { return 7 }

//go:noinline
func covRecur(n int) int {
	if n <= 0 {
		return 0
	}
	return covRecur(n-1) + covCallee()
}

// covLargeNonGeneric is a deliberately large, never-executed function used as a
// patch target in TestApplyOverrideNoShapedFallback. It is not generic, so
// findShapedImpl finds no shaped implementation for it, and its compiled body is
// far larger than what an override writes, so patching and restoring its
// prologue cannot spill into an adjacent function.
//
//go:noinline
func covLargeNonGeneric(a int) int {
	b := a
	b += a
	b -= a
	b *= a
	b ^= a
	b |= a
	b &= a
	b <<= 1
	b >>= 1
	b += a
	b -= a
	b *= a
	b ^= a
	b |= a
	b &= a
	b <<= 1
	b >>= 1
	b += a
	b -= a
	b *= a
	b ^= a
	b |= a
	b &= a
	b <<= 1
	b >>= 1
	b += a
	b -= a
	b *= a
	b ^= a
	b |= a
	b &= a
	b <<= 1
	b >>= 1
	return b
}

// TestFindShapedImplNotAFunction covers the guard for an address that does not
// belong to any function (runtime.FuncForPC returns nil).
func TestFindShapedImplNotAFunction(t *testing.T) {
	if got := findShapedImpl(unsafe.Pointer(&covData[0])); got != nil {
		t.Errorf("expected nil for non-function address, got %p", got)
	}
}

// TestFindShapedImplNonGeneric covers the case where the scanned function makes
// calls but none of them target a same-named shaped implementation, so the scan
// falls through and returns nil. covRecur also calls itself, exercising the
// guard that skips a call whose target is the scanned function's own entry.
func TestFindShapedImplNonGeneric(t *testing.T) {
	if got := findShapedImpl(reflect.ValueOf(covRecur).UnsafePointer()); got != nil {
		t.Errorf("expected nil for non-generic function, got %p", got)
	}
	if got := findShapedImpl(reflect.ValueOf(covCallee).UnsafePointer()); got != nil {
		t.Errorf("expected nil for non-generic function, got %p", got)
	}
}

// TestApplyOverrideNoShapedFallback covers the fallback taken when the shaped
// implementation cannot be located: the override degrades to a plain entry patch
// and no shim is written into the function body, which would otherwise overwrite
// code the shaped entry can never branch to. It patches a real (but never
// called) function so that the memory write targets the TEXT segment, as it does
// in normal use; the original bytes are restored immediately.
func TestApplyOverrideNoShapedFallback(t *testing.T) {
	mock := func(int) int { return 0 }
	ptr := reflect.ValueOf(covLargeNonGeneric).UnsafePointer()

	e := Expect{
		mockAddr:  reflect.ValueOf(mock).UnsafePointer(),
		orgAddr:   ptr,
		isGeneric: true,
		canShim:   true,
		intWords:  1,
		// shapedAddr stays nil - findShapedImpl located no implementation
	}
	applyOverride(&e)
	reset(ptr, e.orgPrologue) // restore before the function could ever be called

	if want := len(buildJump(ptr, ptr)); len(e.orgPrologue) != want {
		t.Errorf("patched %d bytes, want %d (the entry jump alone, no shim)",
			len(e.orgPrologue), want)
	}
	if e.shapedPrologue != nil {
		t.Error("expected nil shaped prologue when shaped impl is absent")
	}
	if e.shapedAddr != nil {
		t.Errorf("expected nil shaped address, got %p", e.shapedAddr)
	}
}

// TestBuildShimLength checks that the shim only shifts the register slots the
// signature actually uses: each integer word above the dictionary slot costs one
// move, and a signature that uses none produces the shortest possible shim.
func TestBuildShimLength(t *testing.T) {
	const mock = uintptr(0x1000)
	base := len(buildShim(mock, 0, 0)) // tail call to the mock, no moves at all

	for _, c := range []struct{ dropSlot, intWords, moves int }{
		{0, 0, 0},
		{0, 1, 1},
		{0, 3, 3},
		{1, 3, 2}, // a one-word receiver keeps its own slot
		{2, 3, 1},
		{3, 3, 0},
	} {
		got := len(buildShim(mock, c.dropSlot, c.intWords))
		if c.moves == 0 {
			if got != base {
				t.Errorf("buildShim(dropSlot=%d, intWords=%d) = %d bytes, want %d",
					c.dropSlot, c.intWords, got, base)
			}
			continue
		}
		perMove := (len(buildShim(mock, 0, 1)) - base)
		if want := base + c.moves*perMove; got != want {
			t.Errorf("buildShim(dropSlot=%d, intWords=%d) = %d bytes, want %d (%d moves)",
				c.dropSlot, c.intWords, got, want, c.moves)
		}
	}
}

// TestFindShapedImplGeneric confirms the positive path: the shaped
// implementation of a real generic is found and differs from the trampoline.
func TestFindShapedImplGeneric(t *testing.T) {
	tramp := reflect.ValueOf(genericFunc[int]).UnsafePointer()
	shaped := findShapedImpl(tramp)
	if shaped == nil {
		t.Fatal("expected to locate shaped implementation of genericFunc[int]")
	}
	if shaped == tramp {
		t.Error("shaped implementation must differ from the trampoline")
	}
}

// TestIsGenericName checks that the "[...]" marker alone does not classify a
// name as a patchable generic instantiation: compiler-generated helpers defined
// inside a generic function inherit its name but take no type dictionary.
func TestIsGenericName(t *testing.T) {
	cases := map[string]bool{
		"pkg.Func[...]":                 true,
		"pkg.Type[...].Method":          true,
		"pkg.PlainFunc":                 false,
		"pkg.Type.Method":               false,
		"pkg.Func[...].func1":           false, // closure inside a generic function
		"pkg.Func[...].func2.1":         false, // nested closure
		"pkg.Type[...].Method.func1":    false, // closure inside a generic method
		"pkg.Func[...].deferwrap1":      false, // deferred call wrapper
		"pkg.Func[...].gowrap1":         false, // go statement wrapper
		"pkg.Type[...].Method-fm":       false, // method value wrapper
		"pkg.Type[...].Func1":           true,  // exported method, not a closure
		"pkg.Type[...].Method.Nested":   false,
		"pkg.Outer[...].Inner[...].Get": true,
		"pkg.(*Type[...]).Method":       true,  // pointer receiver
		"pkg.(*Type[...]).Method.func1": false, // closure inside a pointer-receiver method
	}
	for name, want := range cases {
		if got := isGenericName(name); got != want {
			t.Errorf("isGenericName(%q) = %v, want %v", name, got, want)
		}
	}
}

// TestIsGenericMethodName distinguishes generic methods from generic functions.
func TestIsGenericMethodName(t *testing.T) {
	cases := map[string]bool{
		"pkg.Func[...]":        false,
		"pkg.Type[...].Method": true,
		"pkg.Type[...].Get":    true,
		"pkg.PlainFunc":        false,
		"pkg.Type.Method":      false,
		"a[...]b[...].M":       true,
		"pkg.(*Type[...]).Set": true,
	}
	for name, want := range cases {
		if got := isGenericMethodName(name); got != want {
			t.Errorf("isGenericMethodName(%q) = %v, want %v", name, got, want)
		}
	}
}

// TestIntRegs checks the integer-register footprint computed for the argument
// types that determine a generic method's dictionary slot.
func TestIntRegs(t *testing.T) {
	type twoInts struct{ a, b int }
	type stringAndInt struct {
		s string
		v int
	}
	cases := []struct {
		v    any
		want int
		ok   bool
	}{
		{int(0), 1, true},
		{true, 1, true},
		{(*int)(nil), 1, true},
		{make(chan int), 1, true},
		{map[int]int(nil), 1, true},
		{func() {}, 1, true},
		{float64(0), 0, true},
		{complex128(0), 0, true},
		{"", 2, true},
		{[]int(nil), 3, true},
		{twoInts{}, 2, true},
		{stringAndInt{}, 3, true},
		{[0]int{}, 0, true},
		{[1]int{}, 1, true},
		{[3]int{}, 0, false},               // multi-element arrays are stack-passed
		{struct{ arr [3]int }{}, 0, false}, // a struct inherits an unsupported field
	}
	for _, c := range cases {
		got, ok := intRegs(reflect.TypeOf(c.v))
		if got != c.want || ok != c.ok {
			t.Errorf("intRegs(%T) = (%d, %v), want (%d, %v)", c.v, got, ok, c.want, c.ok)
		}
	}
	// interface kind (reflect.TypeOf unwraps concrete values, so build it via a field)
	iface := reflect.TypeOf(struct{ e error }{}).Field(0).Type
	if got, ok := intRegs(iface); got != 2 || !ok {
		t.Errorf("intRegs(interface) = (%d, %v), want (2, true)", got, ok)
	}
}

// TestDictDropSlot covers the slot computation, including the guard for a method
// signature with no parameters.
func TestDictDropSlot(t *testing.T) {
	// function: dictionary is always the first argument
	if slot, ok := dictDropSlot(reflect.TypeOf(func(int) {}), false); slot != 0 || !ok {
		t.Errorf("function slot = (%d, %v), want (0, true)", slot, ok)
	}
	// method: dictionary follows the receiver (a pointer -> slot 1)
	if slot, ok := dictDropSlot(reflect.TypeOf(func(*int, int) {}), true); slot != 1 || !ok {
		t.Errorf("method slot = (%d, %v), want (1, true)", slot, ok)
	}
	// method signature without parameters is malformed and rejected
	if _, ok := dictDropSlot(reflect.TypeOf(func() {}), true); ok {
		t.Error("expected ok=false for a method signature with no receiver")
	}
}

// TestShimFitsSignature checks the register-budget guard: a signature whose
// arguments plus the injected dictionary fit in registers is shim-able, while
// one that would spill an argument to the stack is not. It uses the arch-specific
// budgets so it holds on both amd64 and arm64.
func TestShimFitsSignature(t *testing.T) {
	funcOf := func(param reflect.Type, n int) reflect.Type {
		in := make([]reflect.Type, n)
		for i := range in {
			in[i] = param
		}
		return reflect.FuncOf(in, nil, false)
	}
	intType := reflect.TypeOf(int(0))
	floatType := reflect.TypeOf(float64(0))

	// (budget-1) integer args + the dictionary exactly fill the integer budget
	if words, ok := shimFitsSignature(funcOf(intType, intRegBudget-1)); !ok || words != intRegBudget-1 {
		t.Errorf("%d integer args: got (%d, %v), want (%d, true)",
			intRegBudget-1, words, ok, intRegBudget-1)
	}
	// budget integer args + the dictionary overflow the integer budget
	if _, ok := shimFitsSignature(funcOf(intType, intRegBudget)); ok {
		t.Errorf("%d integer args should not be shim-able", intRegBudget)
	}
	// floats use a separate budget and do not compete with the dictionary, and
	// occupy no integer slot, so no register has to be shifted
	if words, ok := shimFitsSignature(funcOf(floatType, floatRegBudget)); !ok || words != 0 {
		t.Errorf("%d float args: got (%d, %v), want (0, true)", floatRegBudget, words, ok)
	}
	if _, ok := shimFitsSignature(funcOf(floatType, floatRegBudget+1)); ok {
		t.Errorf("%d float args should not be shim-able", floatRegBudget+1)
	}
	// a stack-passed argument type is never shim-able
	if _, ok := shimFitsSignature(reflect.TypeOf(func(a [3]int) {})); ok {
		t.Error("a multi-element array arg should not be shim-able")
	}
	// a mixed signature reports the integer words only
	if words, ok := shimFitsSignature(reflect.TypeOf(func(string, float64, int) {})); !ok || words != 3 {
		t.Errorf("string+float64+int: got (%d, %v), want (3, true)", words, ok)
	}
}

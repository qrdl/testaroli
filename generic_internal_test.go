package testaroli

import (
	"reflect"
	"testing"
	"unsafe"
)

// covData is a chunk of non-code memory used to drive findShapedImpl down its
// "address is not a function" path (a read-only lookup, no patching).
var covData [128]byte

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
// patch target in TestOverrideGenericNoShaped. It is not generic, so
// findShapedImpl finds no shaped implementation for it, and its compiled body is
// far larger than the jump+shim that overrideGeneric writes, so patching and
// restoring its prologue cannot spill into an adjacent function.
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

// TestOverrideGenericNoShaped covers overrideGeneric's fallback when the shaped
// implementation cannot be located: it still patches the primary entry and
// reports a nil shaped address. It patches a real (but never-called) function so
// that the memory write targets the TEXT segment, as it does in normal use; the
// original bytes are restored immediately.
func TestOverrideGenericNoShaped(t *testing.T) {
	mock := func(int) int { return 0 }
	ptr := reflect.ValueOf(covLargeNonGeneric).UnsafePointer()

	shaped, trampSaved, shapedSaved := overrideGeneric(ptr, reflect.ValueOf(mock).UnsafePointer(), 0)
	reset(ptr, trampSaved) // restore before the function could ever be called

	if shaped != nil {
		t.Errorf("expected nil shaped address, got %p", shaped)
	}
	if shapedSaved != nil {
		t.Error("expected nil shaped prologue when shaped impl is absent")
	}
	if len(trampSaved) == 0 {
		t.Error("expected saved bytes for the primary entry")
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

// TestIsGenericMethodName distinguishes generic methods from generic functions.
func TestIsGenericMethodName(t *testing.T) {
	cases := map[string]bool{
		"pkg.Func[...]":        false,
		"pkg.Type[...].Method": true,
		"pkg.Type[...].Get":    true,
		"pkg.PlainFunc":        false,
		"pkg.Type.Method":      false,
		"a[...]b[...].M":       true,
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
	if !shimFitsSignature(funcOf(intType, intRegBudget-1)) {
		t.Errorf("%d integer args should be shim-able", intRegBudget-1)
	}
	// budget integer args + the dictionary overflow the integer budget
	if shimFitsSignature(funcOf(intType, intRegBudget)) {
		t.Errorf("%d integer args should not be shim-able", intRegBudget)
	}
	// floats use a separate budget and do not compete with the dictionary
	if !shimFitsSignature(funcOf(floatType, floatRegBudget)) {
		t.Errorf("%d float args should be shim-able", floatRegBudget)
	}
	if shimFitsSignature(funcOf(floatType, floatRegBudget+1)) {
		t.Errorf("%d float args should not be shim-able", floatRegBudget+1)
	}
	// a stack-passed argument type is never shim-able
	if shimFitsSignature(reflect.TypeOf(func(a [3]int) {})) {
		t.Error("a multi-element array arg should not be shim-able")
	}
}

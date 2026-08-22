package testaroli

import (
	"reflect"
	"testing"
	"unsafe"
)

// covData is a chunk of non-code memory used to drive findShapedImpl and
// overrideGeneric down their "not a generic function" paths.
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
// reports a nil shaped address. It runs against a scratch buffer so no real code
// is executed; the original bytes are restored afterwards.
func TestOverrideGenericNoShaped(t *testing.T) {
	mock := func(int) int { return 0 }
	ptr := unsafe.Pointer(&covData[0])

	shaped, trampSaved, shapedSaved := overrideGeneric(ptr, reflect.ValueOf(mock).UnsafePointer())
	defer reset(ptr, trampSaved) // restore scratch bytes

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

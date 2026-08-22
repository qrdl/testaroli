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

	shaped, trampSaved, shapedSaved := overrideGeneric(ptr, reflect.ValueOf(mock).UnsafePointer())
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

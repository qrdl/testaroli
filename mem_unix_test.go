package testaroli

import (
	"os"
	"testing"
	"unsafe"
)

func TestSinglePage(t *testing.T) {
	ptr, size := calcBoundaries(unsafe.Pointer(uintptr(0x10)), 0x10)
	if ptr != unsafe.Pointer(uintptr(0x00)) {
		t.Error("incorrect page start")
	}
	if size != 32 {
		t.Errorf("expected %x, got %x as area size", 20, size)
	}
}

func TestEndOfPage(t *testing.T) {
	pageSize := uintptr(os.Getpagesize())

	ptr, size := calcBoundaries(unsafe.Pointer(pageSize-uintptr(0x10)), 0x10)
	if ptr != unsafe.Pointer(uintptr(0x00)) {
		t.Error("incorrect page start")
	}
	if size != pageSize {
		t.Errorf("expected %x, got %x as area size", pageSize, size)
	}
}

func TestTwoPages(t *testing.T) {
	pageSize := uintptr(os.Getpagesize())

	ptr, size := calcBoundaries(unsafe.Pointer(pageSize-0x4), 0x10)
	if ptr != unsafe.Pointer(uintptr(0x00)) {
		t.Error("incorrect page start")
	}
	expectedsize := pageSize + 0x10 - 0x4
	if size != expectedsize {
		t.Errorf("expected %x, got %x as area size", expectedsize, size)
	}
}

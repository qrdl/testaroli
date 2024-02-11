package testaroli

import (
	"os"
	"testing"
	"unsafe"
)

func TestSinglePage(t *testing.T) {
	//	pageSize := uintptr(os.Getpagesize())

	ptr, size := calcBoundaries(unsafe.Pointer(uintptr(0x10)), 0x10)
	if ptr != unsafe.Pointer(uintptr(0x00)) {
		t.Error("incorrect page start")
	}
	if size != 32 {
		t.Errorf("expected %x, got %x as area size", 20, size)
	}
}

func TestTwoPages(t *testing.T) {
	pageSize := uintptr(os.Getpagesize())

	ptr, size := calcBoundaries(unsafe.Pointer(pageSize-0x4), 0x10)
	if ptr != unsafe.Pointer(uintptr(0x00)) {
		t.Error("incorrect page start")
	}
	if size != 4108 {
		t.Errorf("expected %x, got %x as area size", 4108, size)
	}
}

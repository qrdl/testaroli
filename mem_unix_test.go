//go:build unix && !darwin

package testaroli

import (
	"os"
	"testing"
	"unsafe"

	"golang.org/x/sys/unix"
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

func TestPanicInReplacePrologue(t *testing.T) {
	Override(TestingContext(t), makeMemRX, Always, func(ptr unsafe.Pointer, size int) error {
		//	Don't call expectation here to avoid infinite recursion
		if ptr == unsafe.Pointer(uintptr(0)) {
			return unix.EPERM // simulate failure in unix.Mprotect()
		}
		// the code of original makeMemRX
		start, sz := calcBoundaries(ptr, size)
		page := unsafe.Slice((*uint8)(start), sz)
		return unix.Mprotect(page, unix.PROT_WRITE|unix.PROT_READ|unix.PROT_EXEC)

	})

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		} else {
			// Expectaion() didn't finish and left the system in an inconsistent state, so we need to reset it to avoid affecting other tests
			Reset(makeMemRX)
		}
	}()

	replacePrologue(unsafe.Pointer(uintptr(0)), []byte{})
}

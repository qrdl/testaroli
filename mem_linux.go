package testaroli

import (
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

func makeMemWritable(ptr unsafe.Pointer, size int) error {
	start, sz := calcBoundaries(ptr, size)

	page := unsafe.Slice((*uint8)(start), sz)
	return unix.Mprotect(page, unix.PROT_WRITE|unix.PROT_READ|unix.PROT_EXEC)
}

func calcBoundaries(ptr unsafe.Pointer, size int) (unsafe.Pointer, uintptr) {
	pageSize := uintptr(os.Getpagesize())
	areaStart := unsafe.Pointer(uintptr(ptr) &^ (pageSize - 1))
	areaSize := (uintptr(ptr) + uintptr(size)) - uintptr(areaStart)

	return areaStart, areaSize
}

func makePageWritable(start unsafe.Pointer, size uintptr) error {
	page := unsafe.Slice((*uint8)(start), size)
	return syscall.Mprotect(page, syscall.PROT_WRITE|syscall.PROT_READ|syscall.PROT_EXEC)
}

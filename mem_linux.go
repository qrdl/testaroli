package testaroli

import (
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

func replacePrologue(ptr unsafe.Pointer, buf []byte) {
	err := makeMemRX(ptr, len(buf))
	if err != nil {
		panic(err)
	}
	funcPrologue := unsafe.Slice((*uint8)(ptr), len(buf))
	copy(funcPrologue, buf)
}

func makeMemRX(ptr unsafe.Pointer, size int) error {
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

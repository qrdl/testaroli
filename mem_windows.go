package testaroli

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

func replacePrologue(ptr unsafe.Pointer, buf []byte) error {
	err := makeMemRX(ptr, len(buf))
	if err != nil {
		return err
	}
	funcPrologue := unsafe.Slice((*uint8)(ptr), len(buf))
	copy(funcPrologue, buf)

	return nil
}

func makeMemRX(ptr unsafe.Pointer, size int) error {
	var oldPerms uint32
	return windows.VirtualProtect(
		uintptr(ptr),
		uintptr(size),
		windows.PAGE_EXECUTE_READWRITE,
		&oldPerms)
}

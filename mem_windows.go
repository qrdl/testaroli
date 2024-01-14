package testaroli

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

func makeWritable(ptr unsafe.Pointer, size int) error {
	var oldPerms uint32
	return windows.VirtualProtect(
		uintptr(ptr),
		uintptr(size),
		windows.PAGE_EXECUTE_READWRITE,
		&oldPerms)
}

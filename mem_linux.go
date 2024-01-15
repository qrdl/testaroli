package testaroli

import (
	"os"
	"syscall"
	"unsafe"
)

func makeMemWritable(ptr unsafe.Pointer, size int) error {
	pageSize := os.Getpagesize()
	page := unsafe.Slice(
		(*uint8)(
			// Have to go long way to avoid "possible misuse of unsafe.Pointer" warning
			unsafe.Pointer(
				uintptr(ptr)&^uintptr(pageSize-1),
			),
		),
		pageSize)
	err := syscall.Mprotect(page, syscall.PROT_WRITE|syscall.PROT_READ|syscall.PROT_EXEC)
	if err != nil {
		return err
	}

	// if buffer spans more then one page, make next page writable as well
	if uintptr(ptr)+uintptr(size) > uintptr(ptr)|uintptr(pageSize-1) {
		return makeMemWritable(
			unsafe.Pointer(uintptr(ptr)&^uintptr(pageSize-1)+uintptr(pageSize)), // next page start
			int(uintptr(ptr)+uintptr(size)-uintptr(ptr)|uintptr(pageSize-1)))    // buf overlap with next page
	}

	return nil
}

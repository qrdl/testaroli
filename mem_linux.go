package testaroli

import (
	"os"
	"syscall"
	"unsafe"
)

func makeMemWritable(ptr unsafe.Pointer, size int) error {
	pageSize := uintptr(os.Getpagesize())
	pageStart := unsafe.Pointer(uintptr(ptr) &^ (pageSize - 1))
	err := makePageWritable(pageStart, pageSize)
	if err != nil {
		return err
	}

	nextPageStart := unsafe.Add(pageStart, pageSize)
	if uintptr(ptr)+uintptr(size) > uintptr(nextPageStart) {
		return makePageWritable(nextPageStart, pageSize)
	}

	return nil
}

func makePageWritable(start unsafe.Pointer, size uintptr) error {
	page := unsafe.Slice((*uint8)(start), size)
	return syscall.Mprotect(page, syscall.PROT_WRITE|syscall.PROT_READ|syscall.PROT_EXEC)
}

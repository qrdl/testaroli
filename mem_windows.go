// Copyright (c) 2024 Ilya Caramishev. All rights reserved.
//
// This work is licensed under the terms of the Apache License, Version 2.0
// For a copy, see <https://opensource.org/license/apache-2-0>.

package testaroli

import (
	"unsafe"

	"golang.org/x/sys/windows"
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
	var oldPerms uint32
	return windows.VirtualProtect(
		uintptr(ptr),
		uintptr(size),
		windows.PAGE_EXECUTE_READWRITE,
		&oldPerms)
}

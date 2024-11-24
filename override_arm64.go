// Copyright (c) 2024 Ilya Caramishev. All rights reserved.
//
// This work is licensed under the terms of the Apache License, Version 2.0
// For a copy, see <https://opensource.org/license/apache-2-0>.

package testaroli

/*
// ARM doesn't automatically invalidate instruction cache so manual flushing needed
// after changing memory page with executable code

#include <stdint.h>
void flush_cache(uint64_t addr, size_t len) {
	char *target = (char *)addr;
	__builtin___clear_cache(target, target + len);
}
*/
import "C"

import (
	"encoding/binary"
	"unsafe"
)

const instrLength = 4
const jmpInstrCode = uint8(0x14) // B instruction

func override(orgPointer, mockPointer unsafe.Pointer) []byte {
	funcPrologue := unsafe.Slice((*uint8)(orgPointer), instrLength)
	orgPrologue := make([]byte, instrLength)
	copy(orgPrologue, funcPrologue)

	newPrologue := make([]byte, instrLength)
	jumpLocation := (uintptr(mockPointer) - (uintptr(orgPointer))) / uintptr(instrLength)
	binary.NativeEndian.PutUint32(newPrologue, uint32(jumpLocation))
	newPrologue[3] = jmpInstrCode

	replacePrologue(orgPointer, newPrologue) // OS-specific

	C.flush_cache(C.uint64_t(uintptr(orgPointer)), C.size_t(instrLength))

	return orgPrologue
}

func reset(ptr unsafe.Pointer, buf []byte) {
	replacePrologue(ptr, buf) // OS-specific

	C.flush_cache(C.uint64_t(uintptr(ptr)), C.size_t(instrLength))
}

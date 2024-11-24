// This file is part of Testaroli project, available at https://github.com/qrdl/testaroli
// Copyright (c) 2024 Ilya Caramishev. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at https://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testaroli

/*
// Arm64 doesn't automatically invalidate instruction cache so manual flushing is needed
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

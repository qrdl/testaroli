package testaroli

/*
// I'm not sure how it works, but sometimes after updating binary in memory CPU
// still executes old version (especially when you are using several mocks for
// the same function, one after another), most likely it happens because
// instruction cache wasn't flushed and execution switches to different core.
// Using Go's atomics, C volatile and changing memory page protection don't help,
// but flushing cache does.

#cgo CFLAGS: -g -Wall
#include <stdint.h>
void flush_cache(uint64_t addr, size_t len) {
	uint32_t *target = (uint32_t *)addr;
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
	// allow updating memory page with orig function code and leave it like this to allow further
	// restoration of original function prologue
	err := makeMemWritable(orgPointer, instrLength) // call OS-specific function
	if err != nil {
		panic(err)
	}

	funcPrologue := unsafe.Slice((*uint8)(orgPointer), instrLength)
	orgPrologue := make([]byte, instrLength)
	copy(orgPrologue, funcPrologue)

	// replace original content with B <mock func relative offset (in instructions) >
	jumpLocation := (uintptr(mockPointer) - (uintptr(orgPointer))) / uintptr(instrLength)
	binary.NativeEndian.PutUint32(funcPrologue, uint32(jumpLocation))
	funcPrologue[3] = jmpInstrCode

	C.flush_cache(C.uint64_t(uintptr(orgPointer)), C.size_t(instrLength))

	return orgPrologue
}

func reset(ptr unsafe.Pointer, buf []byte) {
	funcBegin := unsafe.Slice((*uint8)(ptr), instrLength)
	copy(funcBegin, buf)

	C.flush_cache(C.uint64_t(uintptr(ptr)), C.size_t(instrLength))
}

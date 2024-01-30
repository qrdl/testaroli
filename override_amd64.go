package testaroli

import (
	"encoding/binary"
	"unsafe"
)

const jmpInstrLength = 5 // length of local JMP instruction with operand
const jmpInstrCode = uint8(0xE9)

func override(orgPointer, mockPointer unsafe.Pointer) []byte {
	// allow updating memory page with orig function code and leave it like this to allow further
	// restoration of original function prologue
	err := makeMemWritable(orgPointer, jmpInstrLength) // call OS-specific function
	if err != nil {
		panic(err)
	}

	funcPrologue := unsafe.Slice((*uint8)(orgPointer), jmpInstrLength)
	orgPrologue := make([]byte, jmpInstrLength)
	copy(orgPrologue, funcPrologue)

	// replace original content with JMP <mock func relative address>
	funcPrologue[0] = jmpInstrCode
	jumpLocation := uintptr(mockPointer) - (uintptr(orgPointer) + jmpInstrLength)
	binary.NativeEndian.PutUint32(funcPrologue[1:], uint32(jumpLocation))

	return orgPrologue
}

func reset(ptr unsafe.Pointer, buf []byte) {
	funcBegin := unsafe.Slice((*uint8)(ptr), jmpInstrLength)
	copy(funcBegin, buf)
}

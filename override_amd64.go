package testaroli

import (
	"encoding/binary"
	"unsafe"
)

const jmpInstrLength = 5 // length of local JMP instruction with operand
const jmpInstrCode = uint8(0xE9)

func override(orgPointer, mockPointer unsafe.Pointer) []byte {
	funcPrologue := unsafe.Slice((*uint8)(orgPointer), jmpInstrLength)
	orgPrologue := make([]byte, jmpInstrLength)
	copy(orgPrologue, funcPrologue)

	// replace original content with JMP <mock func relative address>
	newPrologue := make([]byte, jmpInstrLength)
	newPrologue[0] = jmpInstrCode
	jumpLocation := uintptr(mockPointer) - (uintptr(orgPointer) + jmpInstrLength)
	binary.NativeEndian.PutUint32(newPrologue[1:], uint32(jumpLocation))

	replacePrologue(orgPointer, newPrologue) // OS-specific

	return orgPrologue
}

func reset(ptr unsafe.Pointer, buf []byte) {
	replacePrologue(ptr, buf) // OS-specific
}

package testaroli

import (
	"context"
	"encoding/binary"
	"unsafe"
)

const jmpInstrLength = 5 // length of local JMP instruction with operand
const jmpInstrCode = uint8(0xE9)

var orgContent = map[uintptr][]byte{}

func override(ctx context.Context, orgPointer, mockPointer unsafe.Pointer) {
	// allow updating memory page with orig function code and leave it like this to allow further
	// restoration of original function prologue
	err := makeMemWritable(orgPointer, jmpInstrLength) // call OS-specific function
	if err != nil {
		panic(err)
	}

	funcPrologue := unsafe.Slice((*uint8)(orgPointer), jmpInstrLength)

	// make a backup of original prologue, but only once
	if _, ok := orgContent[uintptr(orgPointer)]; !ok {
		backup := make([]byte, jmpInstrLength)
		copy(backup, funcPrologue)
		orgContent[uintptr(orgPointer)] = backup
	}

	// replace original content with JMP <mock func relative address>
	funcPrologue[0] = jmpInstrCode
	jumpLocation := uintptr(mockPointer) - (uintptr(orgPointer) + jmpInstrLength)
	binary.NativeEndian.PutUint32(funcPrologue[1:], uint32(jumpLocation))
}

func reset(ptr unsafe.Pointer) {
	funcBegin := unsafe.Slice((*uint8)(ptr), jmpInstrLength)
	copy(funcBegin, orgContent[uintptr(ptr)])
}

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

import (
	"encoding/binary"
	"runtime"
	"unsafe"
)

const jmpInstrLength = 5 // length of local JMP instruction with operand
const jmpInstrCode = uint8(0xE9)

// Go internal ABI argument-register budget on amd64: integer registers AX, BX,
// CX, DI, SI, R8, R9, R10, R11 and floating-point registers X0..X14. The generic
// shim can only reshape arguments that stay within these, once the hidden
// dictionary has claimed one integer register.
const intRegBudget = 9
const floatRegBudget = 15

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

// buildJump returns the bytes of an unconditional relative JMP instruction that,
// placed at address <from>, transfers control to <to>.
func buildJump(from, to unsafe.Pointer) []byte {
	buf := make([]byte, jmpInstrLength)
	buf[0] = jmpInstrCode
	jumpLocation := uintptr(to) - (uintptr(from) + jmpInstrLength)
	binary.NativeEndian.PutUint32(buf[1:], uint32(jumpLocation))
	return buf
}

// intArgMoves holds, for each integer argument register slot, the machine code
// that copies the next slot's register into it: slot i gets reg[i] <- reg[i+1]
// for the register sequence AX, BX, CX, DI, SI, R8, R9, R10, R11.
var intArgMoves = [][]byte{
	{0x48, 0x89, 0xD8}, // MOVQ BX, AX
	{0x48, 0x89, 0xCB}, // MOVQ CX, BX
	{0x48, 0x89, 0xF9}, // MOVQ DI, CX
	{0x48, 0x89, 0xF7}, // MOVQ SI, DI
	{0x4C, 0x89, 0xC6}, // MOVQ R8, SI
	{0x4D, 0x89, 0xC8}, // MOVQ R9, R8
	{0x4D, 0x89, 0xD1}, // MOVQ R10, R9
	{0x4D, 0x89, 0xDA}, // MOVQ R11, R10
}

// buildShim returns position-independent machine code that adapts the shaped
// (dictionary-passing) calling convention to the mock's convention and then
// tail-calls the mock. A generic implementation receives a hidden dictionary
// pointer in one integer-register argument slot - slot 0 (AX) for a plain
// function, or the slot right after the receiver for a method on a generic type
// (dropSlot) - which pushes every following argument one integer register down.
// The shim shifts those back up and jumps to the mock. Floating-point argument
// registers are untouched (the dictionary is never passed in one).
func buildShim(mockPointer uintptr, dropSlot int) []byte {
	var shim []byte
	for i := dropSlot; i < len(intArgMoves); i++ {
		shim = append(shim, intArgMoves[i]...)
	}
	shim = append(shim, 0x49, 0xBC, 0, 0, 0, 0, 0, 0, 0, 0) // MOVABS $mock, R12
	binary.NativeEndian.PutUint64(shim[len(shim)-8:], uint64(mockPointer))
	shim = append(shim, 0x41, 0xFF, 0xE4) // JMP R12
	return shim
}

// findShapedImpl locates the shaped implementation of a generic function given
// the address of its instantiation trampoline. The trampoline sets up the type
// dictionary and then makes a direct CALL to the shaped implementation, which
// carries the same "[...]" runtime name. Returns nil if it cannot be found.
func findShapedImpl(tramp unsafe.Pointer) unsafe.Pointer {
	f := runtime.FuncForPC(uintptr(tramp))
	if f == nil {
		return nil
	}
	name := f.Name()
	const scan = 256
	code := unsafe.Slice((*byte)(tramp), scan)
	for i := 0; i+jmpInstrLength <= scan; i++ {
		if code[i] != 0xE8 { // CALL rel32
			continue
		}
		rel := int32(binary.NativeEndian.Uint32(code[i+1:]))
		target := uintptr(tramp) + uintptr(i) + jmpInstrLength + uintptr(int64(rel))
		if target == uintptr(tramp) {
			continue
		}
		tf := runtime.FuncForPC(target)
		if tf != nil && tf.Entry() == target && tf.Name() == name {
			return unsafe.Pointer(target)
		}
	}
	return nil
}

// flushICache is a no-op on amd64, which keeps the instruction cache coherent
// with data writes automatically.
func flushICache(unsafe.Pointer, int) {}

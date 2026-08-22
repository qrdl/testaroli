// This file is part of Testaroli project, available at https://github.com/qrdl/testaroli
// Copyright (c) 2024-2026 Ilya Caramishev. All rights reserved.
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
// ARM64 doesn't automatically invalidate instruction cache so manual flushing is needed
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
	"runtime"
	"unsafe"
)

const instrLength = 4
const jmpInstrCode = uint8(0x14) // B instruction

// Go internal ABI argument-register budget on arm64: integer registers R0..R15
// and floating-point registers F0..F15. The generic shim can only reshape
// arguments that stay within these, once the hidden dictionary has claimed one
// integer register.
const intRegBudget = 16
const floatRegBudget = 16

func override(orgPointer, mockPointer unsafe.Pointer) []byte {
	funcPrologue := unsafe.Slice((*uint8)(orgPointer), instrLength)
	orgPrologue := make([]byte, instrLength)
	copy(orgPrologue, funcPrologue)

	newPrologue := make([]byte, instrLength)
	jumpLocation := (uintptr(mockPointer) - (uintptr(orgPointer))) / uintptr(instrLength)
	// ARM64 B (branch) instruction format:
	//   Bits [31:26]: opcode (0b000101 for unconditional branch)
	//   Bits [25:0]:  signed 26-bit offset in instructions (not bytes)
	// We must properly combine the opcode and offset without clobbering offset bits
	instruction := uint32(jumpLocation&0x03FFFFFF) | (uint32(jmpInstrCode) << 24)
	binary.NativeEndian.PutUint32(newPrologue, instruction)

	replacePrologue(orgPointer, newPrologue) // OS-specific

	C.flush_cache(C.uint64_t(uintptr(orgPointer)), C.size_t(instrLength))

	return orgPrologue
}

func reset(ptr unsafe.Pointer, buf []byte) {
	replacePrologue(ptr, buf) // OS-specific

	// Flush the whole restored range, not just the first instruction: a generic
	// override embeds a shim in the trampoline body, so restoring it rewrites
	// many instructions and every one of them must be evicted from the I-cache.
	C.flush_cache(C.uint64_t(uintptr(ptr)), C.size_t(len(buf)))
}

// buildJump returns the bytes of an unconditional B instruction that, placed at
// address <from>, transfers control to <to>.
func buildJump(from, to unsafe.Pointer) []byte {
	buf := make([]byte, instrLength)
	jumpLocation := (uintptr(to) - uintptr(from)) / uintptr(instrLength)
	instruction := uint32(jumpLocation&0x03FFFFFF) | (uint32(jmpInstrCode) << 24)
	binary.NativeEndian.PutUint32(buf, instruction)
	return buf
}

// buildShim returns position-independent machine code that adapts the shaped
// (dictionary-passing) calling convention to the mock's convention and then
// tail-calls the mock. A generic implementation receives a hidden dictionary
// pointer in one integer-register argument slot - slot 0 (R0) for a plain
// function, or the slot right after the receiver for a method on a generic type
// (dropSlot) - which pushes every following argument one integer register down
// the sequence R0..R15. The shim shifts those back up and branches to the mock.
// Floating-point argument registers are untouched.
func buildShim(mockPointer uintptr, dropSlot int) []byte {
	buf := make([]byte, 0, 15*instrLength+2*instrLength+8)
	var b [4]byte
	// MOV X{i}, X{i+1}  (encoded as ORR X{i}, XZR, X{i+1}) for i in dropSlot..14
	for i := uint32(dropSlot); i < 15; i++ {
		instr := uint32(0xAA0003E0) | ((i + 1) << 16) | i
		binary.NativeEndian.PutUint32(b[:], instr)
		buf = append(buf, b[:]...)
	}
	binary.NativeEndian.PutUint32(b[:], 0x58000050) // LDR X16, pc+8 (the literal below)
	buf = append(buf, b[:]...)
	binary.NativeEndian.PutUint32(b[:], 0xD61F0200) // BR X16
	buf = append(buf, b[:]...)
	var lit [8]byte
	binary.NativeEndian.PutUint64(lit[:], uint64(mockPointer)) // 8-byte mock address literal
	buf = append(buf, lit[:]...)
	return buf
}

// findShapedImpl locates the shaped implementation of a generic function given
// the address of its instantiation trampoline. The trampoline sets up the type
// dictionary and then makes a direct BL to the shaped implementation, which
// carries the same "[...]" runtime name. Returns nil if it cannot be found.
func findShapedImpl(tramp unsafe.Pointer) unsafe.Pointer {
	f := runtime.FuncForPC(uintptr(tramp))
	if f == nil {
		return nil
	}
	name := f.Name()
	const scan = 256
	code := unsafe.Slice((*byte)(tramp), scan)
	for i := 0; i+instrLength <= scan; i += instrLength {
		instr := binary.NativeEndian.Uint32(code[i:])
		if instr>>26 != 0x25 { // BL: top 6 bits are 0b100101
			continue
		}
		imm := int64(instr & 0x03FFFFFF)
		if imm&0x02000000 != 0 { // sign-extend 26-bit immediate
			imm -= 0x04000000
		}
		target := uintptr(int64(uintptr(tramp)) + int64(i) + imm*int64(instrLength))
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

// flushICache invalidates the instruction cache for the modified range, required
// on ARM64 after writing executable code.
func flushICache(ptr unsafe.Pointer, size int) {
	C.flush_cache(C.uint64_t(uintptr(ptr)), C.size_t(size))
}

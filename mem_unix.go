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

//go:build linux || dragonfly || freebsd || netbsd || openbsd

package testaroli

import (
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
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
	start, sz := calcBoundaries(ptr, size)

	page := unsafe.Slice((*uint8)(start), sz)
	return unix.Mprotect(page, unix.PROT_WRITE|unix.PROT_READ|unix.PROT_EXEC)
}

func calcBoundaries(ptr unsafe.Pointer, size int) (unsafe.Pointer, uintptr) {
	pageSize := uintptr(os.Getpagesize())
	areaStart := unsafe.Pointer(uintptr(ptr) &^ (pageSize - 1))
	areaSize := (uintptr(ptr) + uintptr(size)) - uintptr(areaStart)

	return areaStart, areaSize
}

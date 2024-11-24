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

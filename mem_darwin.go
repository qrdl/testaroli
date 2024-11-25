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

// #include "mem_darwin.h"
import "C"

import (
	"runtime"
	"unsafe"
)

func init() {
	// make sure no other goroutines are executed within this thread, so
	// suspending all other threads effectively stops all auxiliary goroutines
	runtime.LockOSThread()
	// re-create TEXT segment with max protection rwx
	res := C.make_text_writable()
	if res != 0 {
		panic("cannot make the code writable")
	}
}

func replacePrologue(ptr unsafe.Pointer, buf []byte) {
	res := C.overwrite_prolog(C.uint64_t(uintptr(ptr)), C.uint64_t(uintptr(unsafe.Pointer(&buf[0]))), C.uint64_t(len(buf)))
	if res != 0 {
		panic("cannot overwrite function prologue")
	}
}

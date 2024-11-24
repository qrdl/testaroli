// Copyright (c) 2024 Ilya Caramishev. All rights reserved.
//
// This work is licensed under the terms of the Apache License, Version 2.0
// For a copy, see <https://opensource.org/license/apache-2-0>.

package testaroli
/*
#include "mem_darwin.h"
*/
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

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

//go:build ((linux || darwin) && (amd64 || arm64)) || (windows && amd64)

package testaroli

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"slices"
	"testing"
	"unsafe"
)

/*
Expect holds information about overridden function and has methods to set and check arguments.
*/
type Expect struct {
	ctx         context.Context
	expCount    int
	actCount    int
	mockAddr    unsafe.Pointer
	orgAddr     unsafe.Pointer
	args        []reflect.Value
	orgName     string
	orgPrologue []byte
}

/*
Expectation can be called only from inside the mock (it panics otherwise), it checks whether function call
was expected at this point, and return matching expectation.

It is important to always call Expectation from the mock function, even if you don't want to check
arguments, because Expectation checks that function was called in order, and if it was the last expected
call for overridden function, it restores the original state and overrides next function in the chain.
*/
func Expectation() *Expect {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		panic("cannot identify calling function")
	}
	entry := runtime.FuncForPC(pc).Entry()

	var expect *Expect
	var order int
	// make sure we have called expected function
	for i, e := range expectations {
		if uintptr(e.mockAddr) == entry {
			// must be either Always or first on non-Always
			if e.expCount == Always || numLeadingAlways() == i {
				expect = e
				order = i
				break
			}
			panic("unexpected function call") // should never happen
		}
	}
	if expect == nil {
		panic("unexpected function call - not from mock function")
	}

	expect.actCount++
	if expect.actCount == expect.expCount && !(expect.expCount == Unlimited || expect.expCount == Always) {
		reset(expect.orgAddr, expect.orgPrologue)
		expectations = slices.Delete(expectations, order, order+1) // remove from expected chain
		overrideNextInChain()
	}

	return expect
}

func overrideNextInChain() {
	next := numLeadingAlways()
	if next < len(expectations) {
		expectations[next].orgPrologue = override( // call arch-specific function
			expectations[next].orgAddr,
			expectations[next].mockAddr)
	}
}

/*
RunNumber returns the sequence number of current run for the override. Count is zero-based,
so for the first run it returns 0.

It is useful when you need your mock to behave differently on different runs, for
example:

	//                 v-- function to be called twice
	Override(ctx, foo, 2, func (a int, b string) {
	    e := Expectation()
	    if e.RunNumber() == 0 {  // zero-based, so first run is number 0
	        e = e.Expect(42, "foo")
	    } else {
	        e = e.Expect(42, "bar")
	    }
	    e.CheckArgs(a, b)
	})
*/
func (e Expect) RunNumber() int {
	return e.actCount - 1
}

/*
Expect sets the expected argument values, that can be later checked with [Expect.CheckArgs].
See [Override] for better way (with compile-time type checks) of setting expected values.
*/
func (e *Expect) Expect(args ...any) *Expect {
	expArgs := make([]reflect.Value, len(args))
	for i := range args {
		expArgs[i] = reflect.ValueOf(args[i])
	}
	e.args = expArgs

	return e
}

/*
CheckArgs checks if actual values match the expected ones.

Please note that when reporting differences, this function always use zero-based
numbering - for array/slice elements, function arguments and run numbers, e.g. first
call (if function was overridden for several calls) is called `run 0`
*/
func (e Expect) CheckArgs(args ...any) {

	t := e.Testing()
	t.Helper()

	if len(args) != len(e.args) {
		if len(e.args) == 0 {
			t.Errorf("no extected args set")
		} else {
			t.Errorf("actual arg count %d doesn't match expected %d", len(args), len(e.args))
		}
		return
	}

	for i, a := range args {
		actualArg := reflect.ValueOf(a)
		expectedArg := e.args[i]
		if a == nil {
			// process situations when Expect(nil) is called
			if expectedArg.IsValid() && (!isNillable(expectedArg) || !expectedArg.IsNil()) {
				if e.expCount > 1 || e.expCount == Unlimited || e.expCount == Always {
					t.Errorf(
						"arg %d on the run %d actual value is nil while non-nil is expected",
						i,
						e.actCount-1) // 0-based
					return
				} else {
					t.Errorf(
						"arg %d actual value is nil while non-nil is expected",
						i)
					return
				}
			}
			continue
		}
		res, msg := equal(actualArg, expectedArg)
		if !res {
			if msg == "" {
				msg = fmt.Sprintf("actual value '%v' differs from expected '%v'",
					actualArg,
					expectedArg)
			}
			if e.expCount > 1 || e.expCount == Unlimited || e.expCount == Always {
				t.Errorf("arg %d on the run %d: %s",
					i+1,
					e.actCount-1, // 0-based
					msg)
			} else {
				t.Errorf("arg %d: %s", i, msg)
			}
			return
		}
	}
}

/*
Context returns [context.Context], passed to [Override] function.
*/
func (e Expect) Context() context.Context {
	return e.ctx
}

/*
Testing returns [testing.T], embedded into the context, passed to [Override] function.
*/
func (e Expect) Testing() *testing.T {
	return Testing(e.ctx)
}

func isNillable(val reflect.Value) bool {
	k := val.Kind()
	switch k {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Pointer, reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		return true
	default:
		return false
	}
}

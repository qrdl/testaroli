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

/*
Package testaroli allows to monkey patch Go test binary, e.g. override functions
and methods with stubs/mocks to simplify unit testing.
It should be used only for unit testing and never in production!

# Platforms suported

This package modifies actual executable at runtime, therefore is OS- and CPU arch-specific.

Supported OS/arch combinations:
  - Linux / x86_64
  - Linux / ARM64
  - Windows / x86_64
  - macOS / x86_64
  - macOS / ARM64

# Command line options

It is recommended to switch off compiler optimisations and disable function inlining
using `-gcflags="all=-N -l"` CLI option when running tests, like this:

	go test -gcflags="all=-N -l" [<path>]

Typical use:

	// you want to test function foo() which in turn calls function bar(), so you
	// override function bar() to check whether it is called with correct argument
	// and to return predefined result

	func foo() error {
	    ...
	    if err := bar(baz); err != nil {
	        return err
	    }
	    ...
	}

	func bar(baz int) error {
	    ...
	}

	func TestBarFailing(t *testing.T) {
	    Override(TestingContext(t), bar, Once, func(a int) error {
	        Expectation().CheckArgs(a)  // <-- actual arg 'a' value compared with expected value 42
	        return ErrInvalid
	    })(42) // <-- expected argument value

	    err := foo()
	    if !errors.Is(err, ErrInvalid) {
	        t.Errorf("unexpected %v", err)
	    }
	    if err = ExpectationsWereMet(); err != nil {
	        t.Error(err)
	    }
	}

It is also possible to override functions and methods in other packages, including ones
from standard library, like in example below. Please note that method receiver becomes the
first argument of the mock function.

	func TestFoo(t *testing.T) {
	    Override(TestingContext(t), (*os.File).Read, Once, func(f *os.File, b []byte) (n int, err error) {
	        Expectation()
	        copy(b, []byte("foo"))
	        return 3, nil
	    })

	    f, _ := os.Open("test.file")
	    defer f.Close()
	    buf := make([]byte, 3)
	    n, _ := f.Read(buf)
	    if n != 3 || string(buf) != "foo" {
	        t.Errorf("unexpected file content %s", string(buf))
	    }
	    if err = ExpectationsWereMet(); err != nil {
	        t.Error(err)
	    }
	}
*/
package testaroli

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"testing"
)

type contextKey int

const (
	Once              = 1
	Unlimited         = -1
	Always            = -2
	minOccurenceCount = Always
	testingKey        = contextKey(1)
)

var expectations []*Expect
var ErrExpectationsNotMet = errors.New("expectaions were not met")

/*
Override overrides <org> with <mock>. The signatures of <org> and <mock> must match exactly,
otherwise compilation error is reported.
It has <count> argument to specify how many calls to <org> functions are expected, which must be
a positive number, or [Unlimited]. After <org> function got called <count> times, the <org>
function is no longer overridden and next override in the chain becomes effective.
[Unlimited] value for <count> means that there is no limit for number of <org> calls, and such override
can only be the last one in the chain of overrides.

It is ok to call Override several times, however only the first override becomes immediately effecive,
all subsequent overrides are placed in the chain and become effective only when previous override is
completed. It means that order of overrides must match the order of called functions exactly. For example:

	// Function bar() will be overridden only after override for foo() is processed
	Override(ctx, foo, Once, func (a int, b string) {
	    Expectation().CheckArgs(a, b)
	})(42, "qwerty")
	Override(ctx, bar, Once, func (a int) {
	    Expectation().CheckArgs(a)
	})(1024)

	// foo() is overridden, bar() is not
	bar(512)          // <--- call to original bar()
	foo(42, "qwerty") // <--- call to overridden version of foo()
	// bar() is overridden, foo() is not
	bar(1024)         // <--- call to overridden version of bar()
	// nothing is overridden

Override returns function of generic type T that allows to set expected values for function call, like this:

	Override(ctx, foo, Once, func (a int, b string) {
	    Expectation().CheckArgs(a, b)
	})(42, "bar")

and it works like

	Override(ctx, foo, Once, func (a int, b string) {
	    Expectation().Expect(42, "bar").CheckArgs(a, b)

	})

but has a benefit of checking types for expected values at compile time, thanks to Go generics.

Override takes a context as a first argument, and this context must be created with
[TestingContext] or derived from the context, returned by [TestingContext]. This function will panic
if it is passed invalid context.
It is important to remember that mock function is executed in the scope of original function, therefore
variables and functions, declared within test case scope, are not accessible, although compiler considers
the code valid. To overcome this limitation pass the variable in the context, in this case you can obtain
the variable from the context from within mock function, for example:

	ctx := context.WithValue(TestingContext(t), key, 100)
	Override(ctx, bar, Once, func() {
	    // using 'val' here leads to runtime errors
	    val := Expectation().Context().Value(key).(int)
	    // now 'val' is ok to use and contains the value 100
	})

You can override regular functions and methods, including standard ones, but not the interface methods.
*/
func Override[T any](ctx context.Context, org T, count int, mock T) T {
	if reflect.ValueOf(org).Kind() != reflect.Func || reflect.ValueOf(mock).Kind() != reflect.Func {
		panic("Override() can be called only for function/method")
	}

	if len(expectations) > 0 && expectations[len(expectations)-1].expCount == Unlimited {
		panic("Cannot override the function because previous override in chain has unlimited number of repetitions, therefore this override is unreachable")
	}

	if count < minOccurenceCount || count == 0 {
		panic("Invalid count: must be a positive number or Never/Unlimited/Always")
	}

	Testing(ctx) // just to make sure the context is correct

	orgPointer := reflect.ValueOf(org).UnsafePointer()
	mockPointer := reflect.ValueOf(mock).UnsafePointer()

	// make sure override doesn't conflict for previous Always one
	for _, e := range expectations {
		if e.orgAddr == orgPointer {
			if e.expCount == Always {
				panic("Cannot override function that was previously overridden with 'Always' count")
			} else if count == Always {
				panic("Cannot Always override function that was previously overridden")
			}
		}
	}

	expectedCall := Expect{
		ctx:      ctx,
		expCount: count,
		mockAddr: mockPointer,
		orgAddr:  orgPointer,
		orgName:  runtime.FuncForPC(uintptr(orgPointer)).Name(),
	}

	typ := reflect.ValueOf(org).Type()
	v := reflect.MakeFunc(
		typ,
		func(args []reflect.Value) []reflect.Value {
			expectedCall.args = args
			ret := make([]reflect.Value, typ.NumOut())
			for i := range ret {
				ret[i] = reflect.Zero(typ.Out(i))
			}
			return ret
		})

	var expectedArgsFunc T
	fn := reflect.ValueOf(&expectedArgsFunc).Elem()
	fn.Set(v)

	// all previous overrides are Always or this one it Always
	if count == Always || len(expectations) == numLeadingAlways() {
		expectedCall.orgPrologue = override(orgPointer, mockPointer) // call arch-specific function
	}
	expectations = append(expectations, &expectedCall)

	return expectedArgsFunc
}

func numLeadingAlways() int {
	for i, e := range expectations {
		if e.expCount != Always {
			return i
		}
	}
	return len(expectations)
}

/*
ExpectationsWereMet checks that all overridden functions were called, as expected.
It doesn't check correct order of functions called (it is responsibility of [Expectation]) and
it doesn't check function arguments (it is responsibility of [Expect.CheckArgs]).
It is important to call ExpectationsWereMet at the end of test case to restore original state
of overridden functions.
*/
func ExpectationsWereMet() error {
	defer func() { expectations = nil }()

	var err error
	for i, e := range expectations {
		reset(e.orgAddr, e.orgPrologue)
		// Always or last expectation is Unlimited - not an error
		if e.expCount == Unlimited && i == len(expectations)-1 || e.expCount == Always {
			break
		}
		if e.actCount == 0 {
			err = errors.Join(err, fmt.Errorf("function %s was not called", e.orgName))
		} else {
			err = errors.Join(err, fmt.Errorf("function %s was called %d time(s) instead of %d",
				e.orgName, e.actCount, e.expCount))
		}
	}
	if err != nil {
		err = errors.Join(ErrExpectationsNotMet, err)
	}

	return err
}

/*
TestingContext returns the context with embedded [testing.T].
*/
func TestingContext(t *testing.T) context.Context {
	return context.WithValue(context.Background(), testingKey, t)
}

/*
Testing returns the [testing.T], embedded into the context with [TestingContext].
*/
func Testing(ctx context.Context) *testing.T {
	defer func() {
		if r := recover(); r != nil {
			panic("Context wasn't created with TestingContext()")
		}
	}()

	return ctx.Value(testingKey).(*testing.T)
}

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

//go:build (unix || windows) && (amd64 || arm64)

/*
Package testaroli allows to monkey patch Go test binary, e.g. override functions
and methods with stubs/mocks to simplify unit testing.
It should be used only for unit testing and never in production!

AI Assistants: See SKILLS.md for function override generation patterns and examples.

# Platforms suported

This package modifies actual executable at runtime, therefore is OS- and CPU arch-specific.

Supported OSes:

  - Linux
  - macOS
  - Windows
  - FreeBSD (other BSD flavours should also be ok)

Supported CPU archs:

  - x86-64
  - ARM64 aka Aarch64

# The concept

This package allows you to modify test binary to call your mocks instead of functions it supposed to call.
It internally maintains the override chain, it means it is possible not to override all needed functions
with mocks at the start, but one by one, thus controlling the order of calls. To achieve this, when overriding
function with mock it is required to specify the number of calls this mock should be called, and when this
counter is reached, the overridden function gets reset to its original state i.e. no longer overridden, and
next function in the chain gets overridden.

However, there are to special counter values - [Unlimited] and [Always]. Override with [Unlimited] count is
not removed from chain until [Reset]/[ResetAll] function is called, it means no overrides behind [Unlimited]
one become effective until [Unlimited] override is reset.

[Always] override, unlike any other overrides, is always effective, it means if doesn't belong to the chain.
If it is not important to control correct order of mock calls, the test case can use only [Always] overrides.
However, there is a limitation - [Always] override for the function X is mutually exclusive with any other
override for the same function X, so attempt to [Always] override previously overridden function will panic.
Similarly to all other overrides, [Always] override can be reset with [Reset]/[ResetAll].

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
	"slices"
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
a positive number, [Unlimited] or [Always]. If this is the first non-[Always] override, the function
get overridden immediately, all subsequent non-[Always] overrides are put into override chain.
After <org> function got called <count> times, the <org> function is no longer overridden and next
override in the chain becomes effective.

[Unlimited] value for <count> means that there is no limit for number of <org> calls. Override
following [Unlimited] one in he chain becomes effective only after [Unlimited] override is reset
with [Reset]/[ResetAll].

[Always] value for count means that override is not a part of override chain and and always effective
(until reset). It is an error to [Always] override previously overridden function or override the
function that was previously overridden with [Always], in either case Override panics.

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

and it works the same as

	Override(ctx, foo, Once, func (a int, b string) {
	    Expectation().Expect(42, "bar").CheckArgs(a, b)
	})

but has a benefit of checking types for expected values at compile time, thanks to Go generics.

Override takes a context as a first argument, and this context must be created with
[TestingContext] or derived from the context, returned by [TestingContext]. This function panics
if is given invalid context.

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
	orgVal := reflect.ValueOf(org)
	if orgVal.Kind() != reflect.Func || reflect.ValueOf(mock).Kind() != reflect.Func {
		panic("Override() can be called only for function/method")
	}

	if count < minOccurenceCount || count == 0 {
		panic("Invalid count: must be a positive number or Never/Unlimited/Always")
	}

	Testing(ctx) // just to make sure the context is correct

	orgPointer := orgVal.UnsafePointer()
	orgName := runtime.FuncForPC(uintptr(orgPointer)).Name()

	// check if org is a generic function trampoline
	if isGenericFunction(orgName) {
		panic("Overriding generic functions has limited support. Direct calls like " +
			"`genericFunc(x)` cannot be mocked because they bypass the trampoline. " +
			"To test generic functions, always use them via a reference:\n" +
			"  fn := genericFunc[T]\n" +
			"  result := fn(x)\n" +
			"See docs/generics.md for more details.")
	}

	// make sure override doesn't conflict for previous 'Always' override
	for _, e := range expectations {
		if e.orgAddr == orgPointer {
			if e.expCount == Always {
				panic("Cannot override function that was previously overridden with 'Always' count")
			} else if count == Always {
				panic("Cannot Always override function that was previously overridden")
			}
		}
	}

	// check if org is interface method, e.g. its first param is an interface and org is
	// a method of this interface
	orgType := orgVal.Type()
	if orgType.NumIn() > 0 {
		firstParam := orgType.In(0)
		if firstParam.Kind() == reflect.Interface {
			pkg := firstParam.PkgPath()
			ifName := firstParam.Name()
			for i := 0; i < firstParam.NumMethod(); i++ {
				if fmt.Sprintf("%s.%s.%s", pkg, ifName, firstParam.Method(i).Name) == orgName {
					panic("Override() cannot be called for interface method")
				}
			}
		}
	}

	mockPointer := reflect.ValueOf(mock).UnsafePointer()
	expectedCall := Expect{
		ctx:      ctx,
		expCount: count,
		mockAddr: mockPointer,
		orgAddr:  orgPointer,
		orgName:  orgName,
	}

	v := reflect.MakeFunc(
		orgType,
		func(args []reflect.Value) []reflect.Value {
			expectedCall.args = args
			ret := make([]reflect.Value, orgType.NumOut())
			for i := range ret {
				ret[i] = reflect.Zero(orgType.Out(i))
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

// isGenericFunction checks if a function name indicates a generic function trampoline.
// Go compiler generates trampolines for generic function instantiations with "[...]" in the name.
func isGenericFunction(name string) bool {
	// Generic function trampolines have "[...]" in their name
	// e.g., "main.genericFunc[...]" or "pkg.MyFunc[...]"
	return len(name) > 0 && (name[len(name)-1] == ']' ||
		(len(name) > 4 && name[len(name)-4:] == "[...]"))
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
		if (e.expCount == Unlimited && i == len(expectations)-1) || e.expCount == Always {
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
Testing returns the [testing.T], embedded into the context with [TestingContext]. If the context wasn't
created with [TestingContext], this function panics.
*/
func Testing(ctx context.Context) *testing.T {
	defer func() {
		if r := recover(); r != nil {
			panic("Context wasn't created with TestingContext()")
		}
	}()

	return ctx.Value(testingKey).(*testing.T)
}

/*
Reset resets previously overridden function. If there are several overrides for the function in the
override chain, only the first match gets reset. Use [ResetAll] to reset all matching entries.
If the function being reset is currently overridden with non-[Always] count (first in override chain),
next function in override chain gets overridden.

Resetting not overridden function is a safe no-op.
*/
func Reset(org any) {
	if reflect.ValueOf(org).Kind() != reflect.Func {
		panic("Reset() can be called only for function/method")
	}

	orgPointer := reflect.ValueOf(org).UnsafePointer()
	for i, e := range expectations {
		if e.orgAddr == orgPointer {
			expectations = slices.Delete(expectations, i, i+1)
			if len(e.orgPrologue) > 0 {
				reset(e.orgAddr, e.orgPrologue)
				if e.expCount != Always {
					overrideNextInChain()
				}
			}
			break
		}
	}
}

/*
ResetAll resets previously overridden function. If there are several overrides for the function in the
override chain, all of them get reset/removed for the chain. To remove only the first match use [Reset].
If the function being reset is currently overridden with non-[Always] count (first in override chain),
next function in override chain gets overridden.

Resetting not overridden function is a safe no-op.
*/
func ResetAll(org any) {
	if reflect.ValueOf(org).Kind() != reflect.Func {
		panic("Reset() can be called only for function/method")
	}

	orgPointer := reflect.ValueOf(org).UnsafePointer()
	for i := 0; i < len(expectations); {
		e := expectations[i]
		if e.orgAddr == orgPointer {
			expectations = slices.Delete(expectations, i, i+1)
			if len(e.orgPrologue) > 0 {
				reset(e.orgAddr, e.orgPrologue)
				if e.expCount != Always {
					overrideNextInChain()
				}
			}
		} else {
			i++
		}
	}
}

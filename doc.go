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

/*
Package testaroli allows to monkey patch Go test binary, e.g. override functions
and methods with stubs/mocks to simplify unit testing.
It should be used only for unit testing and never in production!

AI Assistants: See SKILLS.md for function override generation patterns and examples.

# Platforms supported

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

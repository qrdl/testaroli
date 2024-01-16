/*
Package testaroli allows to monkey patch Go test binary, e.g. override functions
and methods with stubs/mocks to simplify unit testing.
It can be used only for unit testing and never in production.

# Platforms suported

This package modifies actual executable at runtime, therefore is OS- and CPU arch-specific.

Currently supported OS/arch combinations:
  - Linux / x86_64
  - Windows / x86_64

Planned OS/arch combinations:
  - Linux / ARM64
  - macOS / x86_64
  - macOS / ARM64

# Command line options

Due to patching of running binary there are certain limitations that may require extra CLI options:
  - inlined functions cannot be overridden, so to prevent inlining use `-gcflags=-l` CLI option when running tests.
  - [Override] function modifies the binary on the fly, so better to avoid it using `-p=1` CLI option

Recommended command to run tests:

	go test -gcflags=-l -p=1 [<path>]

Example:

    // you want to test function foo() which in turn calls function bar(), so you
    // override function bar() to check whether it is called with correct argument
    // and to return preferdined result

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
        testaroli.Override(testeroli.NewContext(t), bar, func(a int) error {
            if a != 42 {  // check arg
                testaroli.Testing(testaroli.LookupContext(bar)).
                    Errorf("unexpected arg value %v", a)
            }
            return ErrInvalid  // simulate failure
        })
        defer testaroli.Reset(bar)  // reset org function in order not to break other tests

        err := foo()
        if !errors.Is(err, ErrInvalid) {
            t.Errorf("unexpected %v", err)
        }
    }
*/
package testaroli

import (
	"context"
	"reflect"
)

/*
Override overrides org with mock. The signatures of org and mock must match exactly,
otherwise compilation error will be reported.

Override can be used with regular functions and methods, hoverever there is a caveat - mock method
should specify the method's object as a first argument, like in example below:

	// override function `func foo(a int) error`
	testaroli.Override(context.TODO(), foo, func(a int) error { return ErrNotFound })
	// override method `func (foo Foo) bar(a int) error`
	testaroli.Override(context.TODO(), Foo.bar, func(foo Foo, a int) error { return ErrNotFound })

Overriding interface methods, however, is an error, and may result in panic.

Please note that function/method remains overridden until [Reset] is called, therefore it may
impact other test cases. It is recommended to use deferred [Reset], like this:

	testaroli.Override(context.TODO(), foo, func(a int) error { return ErrNotFound })
	defer testeroli.Reset(foo)
*/
func Override[T any](ctx context.Context, org, mock T) {
	if reflect.ValueOf(org).Kind() != reflect.Func {
		panic("Override() can be called only for function/method")
	}

	funcPointer := reflect.ValueOf(org).UnsafePointer()
	mockPointer := reflect.ValueOf(mock).UnsafePointer()
	contexts[uintptr(funcPointer)] = ctx

	override(ctx, funcPointer, mockPointer) // call arch-specific function
}

// Reset resets original state of overridden function `org`.
// After Reset call function is no longer overridden.
func Reset(org any) {
	reset(reflect.ValueOf(org).UnsafePointer())
}

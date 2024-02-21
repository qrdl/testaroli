/*
Package testaroli allows to monkey patch Go test binary, e.g. override functions
and methods with stubs/mocks to simplify unit testing.
It can be used only for unit testing and never in production.

# Platforms suported

This package modifies actual executable at runtime, therefore is OS- and CPU arch-specific.

Currently supported OS/arch combinations:
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
	    series := NewSeries(context.TODO(), t)

	    Override(bar, Once, func(a int) error {
	        Expectation().CheckArgs(a)  // <-- actual arg 'a' value compared with expected value 42
	        return ErrInvalid
	    })(42) // <-- expected argument value

	    err := foo()
	    if !errors.Is(err, ErrInvalid) {
	        t.Errorf("unexpected %v", err)
	    }
	    it err = series.ExpectationsWereMet(); err != nil {
	        t.Error(err)
	    }
	}

For more complex examples see 'examples' directory
*/
package testaroli

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"testing"
)

const Once = 1
const Unlimited = -1

/*
Series holds all information about expectations and allows to query final result with [Series.ExpectationsWereMet].
It is important to finalise the Series with [Series.ExpectationsWereMet], it means you need to call it at the end
of each test case to make sure all overridden functions are reset to their initial state and can be used
by other test cases.
*/
type Series struct {
	ctx      context.Context
	t        *testing.T
	expected []*Expect
}

var globalSeries Series

/*
NewSeries creates a new instance of Series object.

It takes a context, which later can be accessed inside the mock using [Series.Context] or [Expect.Context],
and [testing.T] parameter to report detected errors.

It is important to understand that although mock is defined within test function scope, in fact it is executed
in the scope of overridden function, it means that the only way for mock to access variables, defined in the test
function scope, it to pass them in this context. Accessing such variables directly results in Undefined Behaviour.

Example of using context for passing data to the mock function:

	mock := NewSeries(context.WithValue(context.TODO(), 1, "foo"), t)

	Override(foo, Once, func(a string) {
	    e := Expectation()
	    e.Expect(e.Context().Value(1).(string)).CheckArgs(a)
	})

NewSeries panics if there is another non-finalized Series object, because having several active Series objects (each modifying
running binary) can lead to undefined behaviour.
*/
func NewSeries(ctx context.Context, t *testing.T) *Series {
	if len(globalSeries.expected) != 0 {
		panic("Other Series instance is active, cannot have two instances")
	}
	globalSeries = Series{
		ctx:      ctx,
		t:        t,
		expected: make([]*Expect, 0),
	}
	return &globalSeries
}

/*
ExpectationsWereMet checks that all expected overridden functions were called.
It doesn't check correct order of functions called (it is responsibility of [Expectation]) and
it doesn't check function arguments (it is responsibility of [Expect.CheckArgs]).
It is important to call ExpectationsWereMet at the end of test case to restore original state
of overridden functions.
*/
func (s *Series) ExpectationsWereMet() error {
	defer func() { s.expected = nil }()
	if len(s.expected) != 0 {
		if len(s.expected[0].orgPrologue) > 0 {
			// reset last override
			reset(s.expected[0].orgAddr, s.expected[0].orgPrologue)
		}
		// special case - last expectation has unlimited number of repetitions, so it is not an error
		if s.expected[0].expCount == Unlimited {
			return nil
		}
		return fmt.Errorf("some expectations weren't met - function %s was not called", s.expected[0].orgName)
	}
	return nil
}

/*
Context returns [context.Context], passed to [NewSeries] function.
*/
func (s Series) Context() context.Context {
	return s.ctx
}

/*
Testing returns [testing.T], passed to [NewSeries] function.
*/
func (s Series) Testing() *testing.T {
	return s.t
}

/*
Override overrides <org> with <mock>. The signatures of <org> and <mock> must match exactly,
otherwise compilation error will be reported.
It has <count> argument to specify how many calls to <org> functions are expected, which must be
a positive number, or -1 for Unlimited. After <org> function got called <count> times, the <org>
function is no longer overridden and next override in the chain becomes effective.
[Unlimited] value for <count> means that there is no limit for number of <org> calls, and such override
can only be the last one in the chain of overrides.

Override returns function of generic type T that allows to set expected values for function call, like this:

	Override(foo, Once, func (a int, b string) { Expectation().CheckArgs(a, b) })(42, "bar")

It has the same effect as

	Override(foo, Once, func (a int, b string) { Expectation().Expect(42, "bar").CheckArgs(a, b) })

but has a benefit of checking types for expected values at compile time, thanks to Go generics.

You can override regular functions and methods, but not interface methods.
*/
func Override[T any](org T, count int, mock T) T {
	if reflect.ValueOf(org).Kind() != reflect.Func || reflect.ValueOf(mock).Kind() != reflect.Func {
		panic("Override() can be called only for function/method")
	}

	if len(globalSeries.expected) > 0 && globalSeries.expected[len(globalSeries.expected)-1].expCount == Unlimited {
		panic("Cannot override the function because previous override in chain has unlimited number of repetitions, therefore this override is unreachable")
	}

	if count <= 0 && count != Unlimited {
		panic("Invalid count: must be a positive number or Unlimited")
	}

	orgPointer := reflect.ValueOf(org).UnsafePointer()
	mockPointer := reflect.ValueOf(mock).UnsafePointer()

	expectedCall := Expect{
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

	if len(globalSeries.expected) == 0 {
		// first mock - change function prologue
		expectedCall.orgPrologue = override(orgPointer, mockPointer) // call arch-specific function
	}
	globalSeries.expected = append(globalSeries.expected, &expectedCall)

	return expectedArgsFunc
}

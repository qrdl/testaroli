//go:build (amd64 && (linux || windows || darwin)) || (arm64 && (linux || macos))

package testaroli

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
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
Expectation can be called only from inside the mock, it checks whether function call was expected at this point,
and return matching expectation.

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

	if len(expectations) == 0 {
		panic("unexpected function call")
	}

	e := expectations[0]
	t := e.Testing()
	t.Helper()

	// make sure we have called expected function
	if uintptr(e.mockAddr) != entry {
		t.Errorf("unexpected function call (expected %s)", e.orgName) // should never happen
		return &Expect{}
	}

	e.actCount++
	if e.actCount == e.expCount && e.expCount != Unlimited {
		reset(e.orgAddr, e.orgPrologue)
		expectations = expectations[1:] // remove from expected chain
		if len(expectations) > 0 {
			// override next expected function
			expectations[0].orgPrologue = override( // call arch-specific function
				expectations[0].orgAddr,
				expectations[0].mockAddr)
		}
	}

	return e
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
				if e.expCount > 1 || e.expCount == Unlimited {
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
			if e.expCount > 1 || e.expCount == Unlimited {
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

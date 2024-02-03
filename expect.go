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
Expect holds information about overridden function and has methods to set and check arguments
*/
type Expect struct {
	expCount    int
	actCount    int
	mockAddr    unsafe.Pointer
	orgAddr     unsafe.Pointer
	args        []reflect.Value
	orgName     string
	orgPrologue []byte
}

/*
Expectation can be called only from mock, it checks whether function call was expected at this point,
and return matching expectation.
*/
func Expectation() *Expect {
	globalMock.t.Helper()

	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		panic("cannot identify calling function")
	}
	entry := runtime.FuncForPC(pc).Entry()

	if len(globalMock.expected) == 0 {
		globalMock.t.Errorf("unexpected function call")
		return &Expect{}
	}
	e := globalMock.expected[0]

	e.actCount++
	if e.actCount == e.expCount && e.expCount != Unlimited {
		reset(e.orgAddr, e.orgPrologue)
		globalMock.expected = globalMock.expected[1:] // remove from expected chain
		if len(globalMock.expected) > 0 {
			// override next expected function
			globalMock.expected[0].orgPrologue = override( // call arch-specific function
				globalMock.expected[0].orgAddr,
				globalMock.expected[0].mockAddr)
		}
	}

	// check that we have called expected function
	if uintptr(e.mockAddr) != entry {
		globalMock.t.Errorf("unexpected function call (expected %s)", e.orgName)
		return &Expect{}
	}

	return e
}

/*
RunNumber return the number of current run for the override. Count starts from 1, not 0,
so for the first run it returns 1.
*/
func (e Expect) RunNumber() int {
	return e.actCount
}

/*
RemainingRuns returns the number of remaining runs for the override, or [Unlimited].
*/
func (e Expect) RemainingRuns() int {
	if e.expCount == Unlimited {
		return Unlimited
	}
	return e.expCount - e.actCount
}

/*
Expect sets the expected argument values, that can be later checked with [Args].
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
*/
func (e Expect) CheckArgs(args ...any) {
	globalMock.t.Helper()

	if len(args) != len(e.args) {
		if len(e.args) == 0 {
			globalMock.t.Errorf("no extected args set")
		} else {
			globalMock.t.Errorf("actual arg count %d doesn't match expected %d", len(args), len(e.args))
		}
		return
	}

	for i, a := range args {
		actualArg := reflect.ValueOf(a)
		expectedArg := e.args[i]
		if a == nil {
			// no risk in calling IsNil here since we already established that type is nilable
			if !expectedArg.IsNil() {
				if e.expCount > 1 || e.expCount == 0 {
					globalMock.t.Errorf(
						"%s arg on the %s run actual value is nil while non-nil is expected",
						ordinal(i+1),
						ordinal(e.actCount))
				} else {
					globalMock.t.Errorf(
						"%s arg run actual value is nil while non-nil is expected",
						ordinal(i+1))
				}
			}
			return
		}
		if actualArg.Type() != expectedArg.Type() {
			if e.expCount > 1 || e.expCount == 0 {
				globalMock.t.Errorf(
					"%s arg on the %s run actual type (%s) differs from expected (%s)",
					ordinal(i+1),
					ordinal(e.actCount),
					actualArg.Type(),
					expectedArg.Type())
			} else {
				globalMock.t.Errorf(
					"%s arg actual (%s) type differs from expected (%s)",
					ordinal(i+1),
					actualArg.Type(),
					expectedArg.Type())
			}
			return
		}
		if !actualArg.Equal(expectedArg) {
			if e.expCount > 1 || e.expCount == 0 {
				globalMock.t.Errorf(
					"%s arg on the %s run actual value '%v' differs from expected '%v'",
					ordinal(i+1),
					ordinal(e.actCount),
					actualArg,
					expectedArg)
			} else {
				globalMock.t.Errorf(
					"%s arg actual value '%v' differs from expected '%v'",
					ordinal(i+1),
					actualArg,
					expectedArg)
			}
			return
		}
	}
}

/*
Context returns [context.Context], passed to [New] function.
*/
func (e Expect) Context() context.Context {
	return globalMock.ctx
}

/*
Testing returns [testing.T], passed to [New] function.
*/
func (e Expect) Testing() *testing.T {
	return globalMock.t
}

func ordinal(i int) string {
	switch i % 10 {
	case 1:
		return fmt.Sprintf("%dst", i)
	case 2:
		return fmt.Sprintf("%dnd", i)
	case 3:
		return fmt.Sprintf("%drd", i)
	default:
		return fmt.Sprintf("%dth", i)
	}
}

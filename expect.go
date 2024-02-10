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

It is important to always call Expectation from the mock function, even if you don't want to check
arguments, because Expectation check that function was called in order, anf if it was the last expected
call for overridden function, it restores the original state and overrides next function in the chain.
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
RunNumber return the number of current run for the override. Count is zero-based,
so for the first run it returns 0.
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
				if e.expCount > 1 || e.expCount == Unlimited {
					globalMock.t.Errorf(
						"arg %d on the run %d actual value is nil while non-nil is expected",
						i,
						e.actCount-1) // 0-based
					return
				} else {
					globalMock.t.Errorf(
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
				globalMock.t.Errorf("arg %d on the run %d: %s",
					i+1,
					e.actCount-1, // 0-based
					msg)
			} else {
				globalMock.t.Errorf("arg %d: %s", i, msg)
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

// standard reflect.Value.Equal has several issues:
// - it compares pointers only as addresses
// - it doesn't compare maps
// - it doesn't compare slices
// - it doesn't explain what exactly has failed
// - it panics
// so I've rolled my own, based on reflect's implementation
func equal(a, e reflect.Value) (bool, string) {
	if a.Kind() == reflect.Interface {
		a = a.Elem()
	}
	if e.Kind() == reflect.Interface {
		e = e.Elem()
	}

	if !a.IsValid() || !e.IsValid() {
		return a.IsValid() == e.IsValid(), "cannot compare invalid value with valid one"
	}

	if a.Kind() != e.Kind() || a.Type() != e.Type() {
		return false, fmt.Sprintf("actual type '%s' differs from expected '%s'", a.Type(), e.Type())
	}

	switch a.Kind() {
	case reflect.Bool:
		return a.Bool() == e.Bool(), ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return a.Int() == e.Int(), ""
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return a.Uint() == e.Uint(), ""
	case reflect.Float32, reflect.Float64:
		return a.Float() == e.Float(), ""
	case reflect.Complex64, reflect.Complex128:
		return a.Complex() == e.Complex(), ""
	case reflect.String:
		return a.String() == e.String(), ""
	case reflect.Chan:
		return a.Pointer() == e.Pointer(), ""
	case reflect.Pointer, reflect.UnsafePointer: // my change
		if a.Pointer() == e.Pointer() {
			return true, ""
		}
		res, str := equal(reflect.Indirect(a), reflect.Indirect(e))
		if !res && str == "" {
			str = fmt.Sprintf("actual value '%v' differs from expected '%v'", reflect.Indirect(a), reflect.Indirect(e))
		}
		return res, str
	case reflect.Array:
		// u and v have the same type so they have the same length
		vl := a.Len()
		if vl == 0 {
			return true, ""
		}
		for i := 0; i < vl; i++ {
			res, str := equal(a.Index(i), e.Index(i))
			if !res {
				if str == "" {
					str = fmt.Sprintf("actual value '%v' differs from expected '%v'",
						a.Index(i), e.Index(i))
				}
				return false, fmt.Sprintf("array elem %d: %s", i, str)
			}
		}
		return true, ""
	case reflect.Struct:
		// u and v have the same type so they have the same fields
		nf := a.NumField()
		for i := 0; i < nf; i++ {
			res, str := equal(a.Field(i), e.Field(i))
			if !res {
				if str == "" {
					str = fmt.Sprintf("actual value '%v' differs from expected '%v'",
						a.Field(i), e.Field(i))
				}
				return false, fmt.Sprintf("struct field '%s': %s", a.Type().Field(i).Name, str)
			}
		}
		return true, ""
	case reflect.Map: // my change
		if a.Pointer() == e.Pointer() {
			return true, ""
		}
		keys := a.MapKeys()
		if len(keys) != len(e.MapKeys()) {
			return false, "map lengths differ"
		}
		for _, k := range keys {
			res, str := equal(a.MapIndex(k), e.MapIndex(k))
			if !res {
				if str == "" {
					str = fmt.Sprintf("actual value '%v' differs from expected '%v'",
						a.MapIndex(k), e.MapIndex(k))
				}
				return false, fmt.Sprintf("map value for key '%v': %s", k, str)
			}
		}
		return true, ""
	case reflect.Func:
		return a.Pointer() == e.Pointer(), ""
		// function can be equal only to itself
	case reflect.Slice: // my change
		if a.Pointer() == e.Pointer() {
			return true, ""
		}
		vl := a.Len()
		if vl != e.Len() {
			return false, "slice lengths differ"
		}
		if vl == 0 {
			return true, ""
		}
		for i := 0; i < vl; i++ {
			res, str := equal(a.Index(i), e.Index(i))
			if !res {
				if str == "" {
					str = fmt.Sprintf("actual value '%v' differs from expected '%v'",
						a.Index(i), e.Index(i))
				}
				return false, fmt.Sprintf("slice elem %d: %s", i, str)
			}
		}
		return true, ""
	}
	return false, "invalid variable Kind"
}

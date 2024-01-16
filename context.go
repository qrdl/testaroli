package testaroli

import (
	"context"
	"reflect"
	"testing"
)

type key int

const (
	_ = key(iota)
	testingKey
	callCounter
)

var contexts = map[uintptr]context.Context{}

// LookupContext returns a context for the overridden function `fn`. It returns the same context that was
// passed to corresponding [Override] call.
// It panics if the function `fn` was not previously overridden with [Override].
func LookupContext(fn any) context.Context {
	funcPointer := reflect.ValueOf(fn).UnsafePointer()
	return contexts[uintptr(funcPointer)]
}

/*
NewContext creates new testing context that embeds `t` and a counter.
Mock function is executed withing the scope of original (overridden) function therefore it has
no access to variables, defined outside that scope, e.g. by the function that calls [Override].
To pass any custom variable, add it to the context.

This is the example how it can be used:

	func foo(a int) error { ... }

	func TestFoo(t *testing.T) {
	    ctx := context.WithValue(testeroli.NewContext(t), myKey, "value")
	    testaroli.Override(testeroli.NewContext(t), foo, func(a int) error {
	        ctx := testaroli.LookupContext(foo)    // the function here must match the function beign overridden!
	        value := ctx.Value(myKey).(string)
	        // use value here ...
	        if testaroli.Increment(ctx) == 0 {
	            if a != 42 {  // check arg value on first call
	                testaroli.Testing(ctx).Errorf("unexpected arg value %v", a)
	            }
	            return nil
	        }
	        return ErrInvalid  // fail on all subsequest calls
	    })
	    defer testaroli.Reset(foo)  // reset original function in order not to break other tests
	    ...
	}
*/
func NewContext(t *testing.T) context.Context {
	var counter int
	return context.WithValue(
		context.WithValue(
			context.Background(),
			testingKey,
			t,
		),
		callCounter,
		&counter,
	)
}

// Testing returns [testing.T], embedded into the context `ctx`.
// The context must be created with [NewContext], otherwise this function panics.
func Testing(ctx context.Context) *testing.T {
	return ctx.Value(testingKey).(*testing.T)
}

// Counter returns the counter, embedded into the context `ctx`.
// The context must be created with [NewContext], otherwise this function panics.
func Counter(ctx context.Context) int {
	return *ctx.Value(callCounter).(*int)
}

// Increment increments the counter, embedded into the context `ctx`, and
// returns the value of the counter before the increment.
// The context must be created with [NewContext], otherwise this function panics.
func Increment(ctx context.Context) int {
	counter := ctx.Value(callCounter).(*int)
	defer func() { *counter++ }()
	return *counter
}

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

// LookupContext returns a context by the function being overridden.
// It panics if the function `fn` was not overridden before with [Instead].
func LookupContext(fn any) context.Context {
	funcPointer := reflect.ValueOf(fn).UnsafePointer()
	return contexts[uintptr(funcPointer)]
}

// Context creates new testing context that embeds [testing.T] and a counter.
// This is the example how it can be used:
//
//	func foo(a int) error { ... }
//
//	func TestFoo(t *testing.T) {
//	    testaroli.Instead(testeroli.Context(t), foo, func(a int) error {
//	        ctx := testaroli.LookupContext(foo)	// the function here must match the function beign overridden!
//	        t := testaroli.Testing(ctx)
//	        defer testaroli.Increment(ctx)
//	        if testaroli.Counter(ctx) == 0 {
//	            if a != 42 {  // check arg value on first call
//	                t.Errorf("unexpected arg value %v", a)
//	            }
//	            return nil
//	        }
//	        return ErrInvalid  // fail on all subsequest calls
//	    })
//	    defer testaroli.Restore(foo)  // restore original function in order not to break other tests
//	    ...
//	}
func Context(t *testing.T) context.Context {
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

// Testing returns context that embeds [testing.T] and a counter.
// The context must be created with [Context], otherwise thes function panics.
func Testing(ctx context.Context) *testing.T {
	return ctx.Value(testingKey).(*testing.T)
}

// Counter returns the counter, embedded into the context.
// The context must be created with [Context], otherwise thes function panics.
func Counter(ctx context.Context) int {
	return *ctx.Value(callCounter).(*int)
}

// Increment increments the counter, embedded into the context.
// It returns the value of the counter before the increment operation.
// The context must be created with [Context], otherwise thes function panics.
func Increment(ctx context.Context) int {
	counter := ctx.Value(callCounter).(*int)
	defer func() { *counter++ }()
	return *counter
}

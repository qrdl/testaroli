package testaroli

import (
	"testing"
)

func genericFunc[T any](a T) *T {
	return &a
}

// genericMix exercises a mix of integer, string and floating-point arguments
// together with multiple return values.
func genericMix[T any](a T, s string, f float64, n int) (string, T) {
	return s, a
}

// callDirectInferred / callDirectInstantiated call the generic function with a
// direct call expression (bypassing any reference), from a separate function so
// the compiler cannot fold them into the test body.
func callDirectInferred(a int) *int     { return genericFunc(a) }
func callDirectInstantiated(a int) *int { return genericFunc[int](a) }

// genericBox is a generic type with value- and pointer-receiver methods, used to
// verify overriding methods on instantiated generic receiver types.
type genericBox[T any] struct{ v T }

func (b genericBox[T]) Get() T     { return b.v }
func (b *genericBox[T]) Set(x T) T { b.v = x; return x }

// direct method calls from separate functions so they aren't folded into tests.
func callBoxGet(b genericBox[int]) int         { return b.Get() }
func callBoxSet(b *genericBox[int], x int) int { return b.Set(x) }

// genericPair has a value receiver occupying two integer registers, so the type
// dictionary lands in register slot 2 for its shaped method.
type genericPair[T any] struct{ a, b T }

func (p genericPair[T]) First(extra T) T { return p.a }

func callPairFirst(p genericPair[int], extra int) int { return p.First(extra) }

// genericNamed has a value receiver with a string field (two registers) plus a
// value field, so the dictionary lands in register slot 3 for its shaped method.
type genericNamed[T any] struct {
	name string
	v    T
}

func (n genericNamed[T]) Value() T { return n.v }

func callNamedValue(n genericNamed[int]) int { return n.Value() }

// TestGenericReference overrides a generic function and calls it through a
// reference (fn := genericFunc[int]).
func TestGenericReference(t *testing.T) {
	intPointer := genericFunc[int]

	Override(TestingContext(t), intPointer, Once, func(arg int) *int {
		Expectation().CheckArgs(arg)
		return nil
	})(42)

	if intPointer(42) != nil {
		t.Error("Expected overridden function to return nil")
	}

	testError(t, nil, ExpectationsWereMet())
}

// TestGenericDirectInstantiated overrides a generic function and calls it with
// an explicit instantiation expression genericFunc[int](x).
func TestGenericDirectInstantiated(t *testing.T) {
	Override(TestingContext(t), genericFunc[int], Once, func(arg int) *int {
		Expectation().CheckArgs(arg)
		return nil
	})(42)

	if callDirectInstantiated(42) != nil {
		t.Error("Expected overridden function to return nil for genericFunc[int](x)")
	}

	testError(t, nil, ExpectationsWereMet())
}

// TestGenericDirectInferred overrides a generic function and calls it with type
// inference genericFunc(x).
func TestGenericDirectInferred(t *testing.T) {
	Override(TestingContext(t), genericFunc[int], Once, func(arg int) *int {
		Expectation().CheckArgs(arg)
		return nil
	})(42)

	if callDirectInferred(42) != nil {
		t.Error("Expected overridden function to return nil for genericFunc(x)")
	}

	testError(t, nil, ExpectationsWereMet())
}

// TestGenericMixedArgs checks that overriding works with a mix of integer,
// string, floating-point and multi-value signatures for both call forms.
func TestGenericMixedArgs(t *testing.T) {
	Override(TestingContext(t), genericMix[int], Once, func(a int, s string, f float64, n int) (string, int) {
		Expectation().CheckArgs(a, s, f, n)
		return "mocked", -a
	})(7, "hello", 3.5, 99)

	s, v := genericMix(7, "hello", 3.5, 99)
	if s != "mocked" || v != -7 {
		t.Errorf("direct call: got (%q, %d), want (\"mocked\", -7)", s, v)
	}

	testError(t, nil, ExpectationsWereMet())
}

// TestGenericRestored verifies that after the override expires the original
// generic function is restored for both call forms.
func TestGenericRestored(t *testing.T) {
	Override(TestingContext(t), genericFunc[int], Once, func(arg int) *int {
		Expectation().CheckArgs(arg)
		return nil
	})(42)

	if callDirectInferred(42) != nil {
		t.Error("Expected overridden function to return nil")
	}
	testError(t, nil, ExpectationsWereMet())

	// original behaviour must be back for every call form
	if r := callDirectInferred(1); r == nil || *r != 1 {
		t.Error("Expected original genericFunc(x) behaviour after reset")
	}
	if r := callDirectInstantiated(2); r == nil || *r != 2 {
		t.Error("Expected original genericFunc[int](x) behaviour after reset")
	}
	fn := genericFunc[int]
	if r := fn(3); r == nil || *r != 3 {
		t.Error("Expected original reference behaviour after reset")
	}
}

// TestGenericMethodValue overrides a value-receiver method on an instantiated
// generic type and checks that a direct method call is intercepted. The receiver
// becomes the mock's first argument, as with any method override.
func TestGenericMethodValue(t *testing.T) {
	Override(TestingContext(t), genericBox[int].Get, Once, func(b genericBox[int]) int {
		Expectation().CheckArgs(b)
		return 777
	})(genericBox[int]{v: 5})

	if got := callBoxGet(genericBox[int]{v: 5}); got != 777 {
		t.Errorf("direct method call: got %d, want 777", got)
	}

	testError(t, nil, ExpectationsWereMet())

	// original behaviour is restored for every call form
	if got := callBoxGet(genericBox[int]{v: 9}); got != 9 {
		t.Errorf("restored direct call: got %d, want 9", got)
	}
	m := genericBox[int].Get
	if got := m(genericBox[int]{v: 8}); got != 8 {
		t.Errorf("restored reference call: got %d, want 8", got)
	}
}

// TestGenericMethodPointer overrides a pointer-receiver method on an
// instantiated generic type and checks that a direct method call is intercepted.
func TestGenericMethodPointer(t *testing.T) {
	Override(TestingContext(t), (*genericBox[int]).Set, Once, func(b *genericBox[int], x int) int {
		Expectation().CheckArgs(b, x)
		return -x
	})(&genericBox[int]{}, 3)

	box := &genericBox[int]{}
	if got := callBoxSet(box, 3); got != -3 {
		t.Errorf("direct method call: got %d, want -3", got)
	}

	testError(t, nil, ExpectationsWereMet())
}

// TestGenericMethodMultiRegReceiver overrides methods whose receivers span more
// than one integer register, verifying that the dictionary is dropped from the
// correct slot (2 for a two-word receiver, 3 for a string+value receiver).
func TestGenericMethodMultiRegReceiver(t *testing.T) {
	Override(TestingContext(t), genericPair[int].First, Once,
		func(p genericPair[int], extra int) int {
			Expectation().CheckArgs(p, extra)
			return 555
		})(genericPair[int]{a: 1, b: 2}, 9)

	if got := callPairFirst(genericPair[int]{a: 1, b: 2}, 9); got != 555 {
		t.Errorf("two-word receiver: got %d, want 555", got)
	}
	testError(t, nil, ExpectationsWereMet())

	Override(TestingContext(t), genericNamed[int].Value, Once,
		func(n genericNamed[int]) int {
			Expectation().CheckArgs(n)
			return 444
		})(genericNamed[int]{name: "x", v: 7})

	if got := callNamedValue(genericNamed[int]{name: "x", v: 7}); got != 444 {
		t.Errorf("string+value receiver: got %d, want 444", got)
	}
	testError(t, nil, ExpectationsWereMet())
}

// manyArgs has sixteen integer arguments; together with the hidden dictionary
// that exceeds the integer-register budget on both amd64 (9) and arm64 (16), so
// the shim cannot reshape the shaped calling convention and the shaped
// implementation is left unpatched.
func manyArgs[T ~int](a1, a2, a3, a4, a5, a6, a7, a8, a9, a10, a11, a12, a13, a14, a15, a16 T) T {
	return a1 + a2 + a3 + a4 + a5 + a6 + a7 + a8 + a9 + a10 + a11 + a12 + a13 + a14 + a15 + a16
}

func callManyArgs() int {
	return manyArgs[int](1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16)
}

// arrBox is a generic type whose value receiver is a multi-element array, which
// Go passes on the stack, so its shaped method cannot be reshaped by the shim.
type arrBox[T any] [3]T

func (a arrBox[T]) First() T { return a[0] }

func callArrFirst(a arrBox[int]) int { return a.First() }

// TestGenericUnshimmableDegrades verifies graceful degradation for signatures the
// shim cannot reshape (register overflow, or a stack-passed receiver): the
// reference form is still intercepted, while the direct form runs the original
// function with correct arguments rather than being called with corrupted ones.
func TestGenericUnshimmableDegrades(t *testing.T) {
	// register overflow: sixteen integer args + dictionary
	Override(TestingContext(t), manyArgs[int], Always,
		func(a1, a2, a3, a4, a5, a6, a7, a8, a9, a10, a11, a12, a13, a14, a15, a16 int) int {
			Expectation()
			return -1
		})
	if got := callManyArgs(); got != 136 {
		t.Errorf("direct call: got %d, want 136 (original, not intercepted or corrupted)", got)
	}
	fn := manyArgs[int]
	if got := fn(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16); got != -1 {
		t.Errorf("reference call: got %d, want -1 (mocked)", got)
	}
	ResetAll(manyArgs[int])

	// stack-passed receiver: multi-element array
	Override(TestingContext(t), arrBox[int].First, Always, func(a arrBox[int]) int {
		Expectation()
		return 99
	})
	if got := callArrFirst(arrBox[int]{7, 8, 9}); got != 7 {
		t.Errorf("direct method call: got %d, want 7 (original, not intercepted or corrupted)", got)
	}
	m := arrBox[int].First
	if got := m(arrBox[int]{7, 8, 9}); got != 99 {
		t.Errorf("reference method call: got %d, want 99 (mocked)", got)
	}
	ResetAll(arrBox[int].First)
}

// genericClosure returns a closure defined inside a generic function. The
// closure inherits the "[...]" marker of the enclosing instantiation in its
// runtime name (pkg.genericClosure[...].func1), but it is an ordinary entry
// point that receives no type dictionary.
func genericClosure[T any](a T) func(T) T {
	return func(b T) T { return a }
}

// TestGenericEnclosedClosure checks that a closure defined inside a generic
// function is overridden with the regular strategy. The marker in its name must
// not route it through the shaped-implementation strategy, which would mistake
// its first argument for a receiver and write a shim into its (very short) body.
func TestGenericEnclosedClosure(t *testing.T) {
	fn := genericClosure(1)

	Override(TestingContext(t), fn, Once, func(b int) int {
		Expectation().CheckArgs(b)
		return 42
	})(7)

	if got := fn(7); got != 42 {
		t.Errorf("closure call: got %d, want 42 (mocked)", got)
	}
	testError(t, nil, ExpectationsWereMet())

	if got := fn(7); got != 1 {
		t.Errorf("restored closure call: got %d, want 1 (original)", got)
	}
}

// shapeSibling has int as its underlying type, so genericFunc[shapeSibling]
// shares its shaped implementation with genericFunc[int] while having its own
// instantiation trampoline.
type shapeSibling int

func callSiblingDirect(a shapeSibling) *shapeSibling { return genericFunc(a) }

// mustPanic runs <f> and reports a failure unless it panics.
func mustPanic(t *testing.T, what string, f func()) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic %s", what)
		}
	}()
	f()
}

// TestGenericShapeSharingConflict verifies that overrides of shape-compatible
// instantiations that would be effective at the same time are rejected. Both
// patch the same shaped implementation, so the second one would record the first
// one's jump as the original code and resetting either would restore the wrong
// bytes.
func TestGenericShapeSharingConflict(t *testing.T) {
	ctx := TestingContext(t)

	Override(ctx, genericFunc[int], Always, func(a int) *int {
		Expectation()
		return nil
	})
	mustPanic(t, "on an Always override of a shape-compatible instantiation", func() {
		Override(ctx, genericFunc[shapeSibling], Always, func(a shapeSibling) *shapeSibling {
			Expectation()
			return nil
		})
	})
	mustPanic(t, "on any override of an Always-overridden shape-compatible instantiation", func() {
		Override(ctx, genericFunc[shapeSibling], Once, func(a shapeSibling) *shapeSibling {
			Expectation()
			return nil
		})
	})
	ResetAll(genericFunc[int])

	// the reverse order is rejected too
	Override(ctx, genericFunc[int], Once, func(a int) *int {
		Expectation()
		return nil
	})
	mustPanic(t, "on an Always override joining an overridden shape-compatible instantiation", func() {
		Override(ctx, genericFunc[shapeSibling], Always, func(a shapeSibling) *shapeSibling {
			Expectation()
			return nil
		})
	})
	ResetAll(genericFunc[int])

	if len(expectations) != 0 {
		t.Errorf("%d expectation(s) left behind", len(expectations))
	}
}

// TestGenericShapeSharingChained checks the conflict rule does not reject the
// legitimate case: shape-compatible instantiations can still be overridden one
// after another through the override chain, where only one of them patches the
// shared shaped implementation at a time.
func TestGenericShapeSharingChained(t *testing.T) {
	ctx := TestingContext(t)

	Override(ctx, genericFunc[int], Once, func(a int) *int {
		Expectation().CheckArgs(a)
		return nil
	})(1)
	Override(ctx, genericFunc[shapeSibling], Once, func(a shapeSibling) *shapeSibling {
		Expectation().CheckArgs(a)
		return nil
	})(2)

	if callDirectInferred(1) != nil {
		t.Error("first override: expected overridden genericFunc[int] to return nil")
	}
	if callSiblingDirect(2) != nil {
		t.Error("second override: expected overridden genericFunc[shapeSibling] to return nil")
	}
	testError(t, nil, ExpectationsWereMet())

	// original behaviour is back for both instantiations
	if r := callDirectInferred(5); r == nil || *r != 5 {
		t.Error("genericFunc[int] was not restored")
	}
	if r := callSiblingDirect(6); r == nil || *r != 6 {
		t.Error("genericFunc[shapeSibling] was not restored")
	}
}

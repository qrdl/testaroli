package main

import (
	"errors"
	"testing"

	. "github.com/qrdl/testaroli"
)

// TestReport overrides two generic helpers and checks that both call forms used
// by Report - type inference (average) and explicit instantiation (maxOf[int]) -
// are intercepted. Pass the instantiated function to Override; you do NOT need
// to route the call through a reference variable.
func TestReport(t *testing.T) {
	ctx := TestingContext(t)

	Override(ctx, average[int], Once, func(samples ...int) float64 {
		Expectation().CheckArgs(samples)
		return 12.5
	})(10, 15)

	Override(ctx, maxOf[int], Once, func(vals ...int) int {
		Expectation().CheckArgs(vals)
		return 99
	})(10, 15)

	result := Report(10, 15)
	if result != "avg=12.5ms peak=99ms" {
		t.Errorf("got [%s] when [avg=12.5ms peak=99ms] expected", result)
	}

	testError(t, nil, ExpectationsWereMet())
}

// TestExplicitInstantiation calls the generic function with an explicit type
// argument, maxOf[int](...), and confirms the override takes effect.
func TestExplicitInstantiation(t *testing.T) {
	Override(TestingContext(t), maxOf[int], Once, func(vals ...int) int {
		Expectation().CheckArgs(vals)
		return -1
	})(3, 7, 2)

	if result := maxOf[int](3, 7, 2); result != -1 {
		t.Errorf("got [%d] when [-1] expected", result)
	}

	testError(t, nil, ExpectationsWereMet())
}

// TestTypeInference calls the generic function relying on type inference,
// average(...), and confirms the override takes effect.
func TestTypeInference(t *testing.T) {
	Override(TestingContext(t), average[int], Once, func(samples ...int) float64 {
		Expectation().CheckArgs(samples)
		return 0
	})(1, 2, 3)

	if result := average(1, 2, 3); result != 0 {
		t.Errorf("got [%v] when [0] expected", result)
	}

	testError(t, nil, ExpectationsWereMet())
}

// TestReference calls the generic function through a stored reference,
// fn := deref[int]; fn(...), which also works (and was the only form supported
// before direct calls were handled).
func TestReference(t *testing.T) {
	fn := deref[int]

	Override(TestingContext(t), fn, Once, func(p *int) int {
		Expectation().CheckArgs(p)
		return 42
	})(nil)

	if result := fn(nil); result != 42 {
		t.Errorf("got [%d] when [42] expected", result)
	}

	testError(t, nil, ExpectationsWereMet())
}

// TestChain places overrides for generic functions in a chain and verifies they
// become effective in the expected order.
func TestChain(t *testing.T) {
	ctx := TestingContext(t)

	Override(ctx, average[int], Once, func(samples ...int) float64 {
		Expectation().CheckArgs(samples)
		return 1
	})(10)

	Override(ctx, average[int], Once, func(samples ...int) float64 {
		Expectation().CheckArgs(samples)
		return 2
	})(20)

	if result := average(10); result != 1 {
		t.Errorf("first call: got [%v] when [1] expected", result)
	}
	if result := average(20); result != 2 {
		t.Errorf("second call: got [%v] when [2] expected", result)
	}

	testError(t, nil, ExpectationsWereMet())
}

// TestRestored confirms the original generic function is restored once the
// override expires, for every call form.
func TestRestored(t *testing.T) {
	Override(TestingContext(t), maxOf[int], Once, func(vals ...int) int {
		Expectation()
		return 0
	})(5)

	if result := maxOf[int](5); result != 0 {
		t.Errorf("override: got [%d] when [0] expected", result)
	}
	testError(t, nil, ExpectationsWereMet())

	// original behaviour is back for both direct and reference calls
	if result := maxOf(3, 9, 1); result != 9 {
		t.Errorf("restored direct call: got [%d] when [9] expected", result)
	}
	fn := maxOf[int]
	if result := fn(4, 8); result != 8 {
		t.Errorf("restored reference call: got [%d] when [8] expected", result)
	}
}

// CAVEAT: Go shares one compiled implementation across shape-compatible type
// arguments (all pointer types, or a named type and its underlying type).
// Overriding deref[int] therefore also affects direct calls to deref[MyInt],
// because both use the same shaped implementation. This is rarely a problem in
// practice, but keep it in mind when a generic is called with several
// shape-compatible type arguments in one test.

func testError(t *testing.T, expected, actual error) {
	t.Helper()
	if expected == nil && actual != nil {
		t.Errorf("got [%v] error when no error expected", actual)
		return
	}
	if expected != nil && actual == nil {
		t.Errorf("no error reported when [%v] error expected", expected)
		return
	}
	if !errors.Is(expected, actual) {
		t.Errorf("got [%v] error when [%v] error expected", actual, expected)
		return
	}
}

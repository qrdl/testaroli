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

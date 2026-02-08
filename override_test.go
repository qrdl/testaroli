package testaroli

import (
	"context"
	"errors"
	"strings"
	"testing"
)

const key = contextKey(2)

func foo(i int) error {
	if i <= 100 {
		return bar(i + 1)
	}
	for j := 100; j < i; j++ {
		if err := baz(j); err != nil {
			return err
		}
	}
	return bar(i - 100)
}

func bar(i int) error {
	if i%2 == 0 {
		return qux(nil)
	}
	return qux(errors.New("odd"))
}

func baz(i int) error {
	return nil
}

func qux(err error) error {
	return err
}

func corge(a struct {
	a int
	b string
}) int {
	return a.a
}

func TestSingleCall(t *testing.T) {
	Override(TestingContext(t), bar, Once, func(i int) error {
		Expectation().CheckArgs(i)
		return nil
	})(2)

	err := foo(1)

	testError(t, nil, err)
	testError(t, nil, ExpectationsWereMet())
}

func TestSeveralCalls(t *testing.T) {
	ctx := context.WithValue(TestingContext(t), key, 100)

	Override(ctx, baz, Once, func(i int) error {
		e := Expectation()
		e.Expect(e.RunNumber() + e.Context().Value(key).(int))
		e.CheckArgs(i)
		return nil
	})

	err := foo(102)

	testError(t, nil, err)
	testError(t, nil, ExpectationsWereMet())
}

func TestSeveralCallsSeparateMocks(t *testing.T) {
	ctx := TestingContext(t)

	Override(ctx, baz, Once, func(i int) error {
		Expectation().CheckArgs(i)
		return nil
	})(100)

	Override(ctx, baz, Once, func(i int) error {
		Expectation().CheckArgs(i)
		return nil
	})(101)

	err := foo(102)

	testError(t, nil, err)
	testError(t, nil, ExpectationsWereMet())
}

func TestWrongExpectedArg(t *testing.T) {
	var t1 testing.T

	Override(TestingContext(&t1), bar, Unlimited, func(i int) error {
		Expectation().CheckArgs(i)
		return nil
	})(1)

	err := foo(1)

	testError(t, nil, err)
	testError(t, nil, ExpectationsWereMet())
	if !t1.Failed() {
		t.Errorf("expected error")
	}
}

func TestExpect(t *testing.T) {
	Override(TestingContext(t), bar, Once, func(i int) error {
		Expectation().Expect(2).CheckArgs(i)
		return nil
	})

	err := foo(1)

	testError(t, nil, err)
	testError(t, nil, ExpectationsWereMet())
}

func TestModifyContext(t *testing.T) {
	val := 100
	ctx := context.WithValue(TestingContext(t), key, &val)

	Override(ctx, bar, Once, func(i int) error {
		e := Expectation()
		val := e.Context().Value(key).(*int)
		t := e.Testing()
		if *val != 100 {
			t.Errorf("unexpected context value")
		}
		*val = 42
		return nil
	})

	err := foo(1)

	testError(t, nil, err)
	testError(t, nil, ExpectationsWereMet())
	if *(ctx.Value(key).(*int)) != 42 {
		t.Errorf("context not changed")
	}
}

func TestWrongArgTypes(t *testing.T) {
	var t1 testing.T

	Override(TestingContext(&t1), bar, Once, func(i int) error {
		Expectation().Expect("foo").CheckArgs(i)
		return nil
	})

	err := foo(1)

	testError(t, nil, err)
	testError(t, nil, ExpectationsWereMet())
	if !t1.Failed() {
		t.Errorf("expected error")
	}
}

func TestWrongArgCount(t *testing.T) {
	var t1 testing.T

	Override(TestingContext(&t1), bar, Once, func(i int) error {
		Expectation().Expect(1, "foo").CheckArgs(i)
		return nil
	})

	err := foo(1)

	testError(t, nil, err)
	testError(t, nil, ExpectationsWereMet())
	if !t1.Failed() {
		t.Errorf("expected error")
	}
}

func TestWrongArgCount2(t *testing.T) {
	var t1 testing.T

	Override(TestingContext(&t1), bar, Once, func(i int) error {
		Expectation().CheckArgs(i)
		return nil
	})

	err := foo(1)

	testError(t, nil, err)
	testError(t, nil, ExpectationsWereMet())
	if !t1.Failed() {
		t.Errorf("expected error")
	}
}

func TestWrongCallCount(t *testing.T) {
	var t1 testing.T

	Override(TestingContext(&t1), bar, 2, func(i int) error {
		Expectation().CheckArgs(i)
		return nil
	})(2)

	err := foo(1)

	testError(t, nil, err)
	if ExpectationsWereMet() == nil {
		t.Errorf("expected error")
	}
}

func TestUnlimitedCallCount(t *testing.T) {
	Override(TestingContext(t), bar, Unlimited, func(i int) error {
		Expectation().CheckArgs(i)
		return nil
	})(2)

	err := foo(1)

	testError(t, nil, err)
	testError(t, nil, ExpectationsWereMet())
}

func TestExpectNil(t *testing.T) {
	Override(TestingContext(t), qux, Once, func(err error) error {
		Expectation().CheckArgs(err)
		return nil
	})(nil)

	err := foo(1)

	testError(t, nil, err)
	testError(t, nil, ExpectationsWereMet())
}

func TestExpectNilFail(t *testing.T) {
	var t1 testing.T

	Override(TestingContext(&t1), qux, Unlimited, func(err error) error {
		Expectation().CheckArgs(err)
		return nil
	})(errors.New("dummy"))

	err := foo(1)

	testError(t, nil, err)
	testError(t, nil, ExpectationsWereMet())
	if !t1.Failed() {
		t.Errorf("expected error")
	}
}

func TestExpectNilFail2(t *testing.T) {
	var t1 testing.T

	Override(TestingContext(&t1), qux, Once, func(err error) error {
		Expectation().CheckArgs(err)
		return nil
	})(errors.New("dummy"))

	err := foo(1)

	testError(t, nil, err)
	testError(t, nil, ExpectationsWereMet())
	if !t1.Failed() {
		t.Errorf("expected error")
	}
}

func TestInvalidExpectationCall(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	Expectation()
}

func TestInvalidCount(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	Override(TestingContext(t), bar, 0, func(i int) error {
		return nil
	})
}

func TestInvalidOverride(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	var a, b int
	Override(TestingContext(t), 1, a, b) // not functions
}

func TestWrongContext(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	Override(context.Background(), bar, Once, func(i int) error {
		return nil
	})
}

func TestNilComparison(t *testing.T) {
	Override(TestingContext(t), qux, Once, func(err error) error {
		Expectation().Expect(nil).CheckArgs(nil)
		return nil
	})

	qux(nil)
}

func TestCompareWithNil(t *testing.T) {
	var t1 testing.T

	Override(TestingContext(&t1), corge, Once, func(a struct {
		a int
		b string
	}) int {
		Expectation().CheckArgs(nil)
		return 10
	})(struct {
		a int
		b string
	}{a: 1, b: "foo"})

	_ = corge(struct {
		a int
		b string
	}{a: 1, b: "foo"})

	testError(t, nil, ExpectationsWereMet())
	if !t1.Failed() {
		t.Errorf("expected error")
	}
}

func TestOverriddenNotCalled(t *testing.T) {
	var t1 testing.T

	Override(TestingContext(&t1), qux, Once, func(err error) error {
		Expectation().CheckArgs(err)
		return nil
	})(nil)

	err := ExpectationsWereMet()
	if !errors.Is(err, ErrExpectationsNotMet) {
		t.Errorf("expected error")
	}
	if !strings.Contains(err.Error(), "function github.com/qrdl/testaroli.qux was not called") {
		t.Errorf("expected error")
	}
}

func TestOverriddenCalledOnce(t *testing.T) {
	var t1 testing.T

	Override(TestingContext(&t1), qux, 2, func(err error) error {
		Expectation().CheckArgs(err)
		return nil
	})(nil)

	qux(nil)

	err := ExpectationsWereMet()
	testError(t, ErrExpectationsNotMet, err)
	if !strings.Contains(err.Error(), "function github.com/qrdl/testaroli.qux was called 1 time(s) instead of 2") {
		t.Errorf("expected error")
	}
}

func TestAlwaysDoubleOverride(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
		ExpectationsWereMet()
	}()

	ctx := TestingContext(t)

	Override(ctx, baz, Always, func(i int) error {
		e := Expectation()
		e.Expect(e.RunNumber() + e.Context().Value(key).(int))
		e.CheckArgs(i)
		return nil
	})

	Override(ctx, baz, Once, func(i int) error {
		e := Expectation()
		e.Expect(e.RunNumber() + e.Context().Value(key).(int))
		e.CheckArgs(i)
		return nil
	})

	foo(1)
}

func TestAlwaysDoubleOverride2(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
		ExpectationsWereMet()
	}()

	ctx := TestingContext(t)

	Override(ctx, baz, Once, func(i int) error {
		e := Expectation()
		e.Expect(e.RunNumber() + e.Context().Value(key).(int))
		e.CheckArgs(i)
		return nil
	})

	Override(ctx, baz, Always, func(i int) error {
		e := Expectation()
		e.Expect(e.RunNumber() + e.Context().Value(key).(int))
		e.CheckArgs(i)
		return nil
	})

	foo(1)
}

func TestAlways(t *testing.T) {
	ctx := TestingContext(t)

	Override(ctx, baz, Always, func(i int) error {
		e := Expectation()
		e.Expect(e.RunNumber() + 100)
		e.CheckArgs(i)
		return nil
	})(100)

	err := foo(102)

	testError(t, nil, err)
	testError(t, nil, ExpectationsWereMet())
}

func TestFuncVar(t *testing.T) {
	incFunc := func(a int) int {
		return a + 1
	}

	Override(TestingContext(t), incFunc, Once, func(a int) int {
		Expectation().CheckArgs(a)
		return a - 1
	})(100)

	if incFunc(100) != 99 {
		t.Error("unexpected")
	}
	testError(t, nil, ExpectationsWereMet())
}

func TestFuncVar2(t *testing.T) {
	barVar := bar
	Override(TestingContext(t), barVar, Once, func(i int) error {
		Expectation().CheckArgs(i)
		return nil
	})(2)

	err := foo(1)

	testError(t, nil, err)
	testError(t, nil, ExpectationsWereMet())
}

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
	if !errors.Is(actual, expected) {
		t.Errorf("got [%v] error when [%v] error expected", actual, expected)
		return
	}
}

// riskyOperation simulates a function that might panic
func riskyOperation(input string) string {
	return "success: " + input
}

// safeWrapper catches panics from riskyOperation
func safeWrapper(input string) (result string, recovered bool, panicValue interface{}) {
	defer func() {
		if r := recover(); r != nil {
			recovered = true
			panicValue = r
		}
	}()
	result = riskyOperation(input)
	return
}

func TestMockWithPanic(t *testing.T) {
	ctx := TestingContext(t)

	// Override riskyOperation to panic
	Override(ctx, riskyOperation, Once, func(input string) string {
		Expectation().CheckArgs(input)
		panic("simulated critical failure")
	})("test-input")

	// Call the safe wrapper which should catch the panic
	result, recovered, panicValue := safeWrapper("test-input")

	// Verify the panic was caught
	if !recovered {
		t.Errorf("Expected panic to be recovered")
	}

	if result != "" {
		t.Errorf("Expected empty result after panic, got: %s", result)
	}

	if panicValue != "simulated critical failure" {
		t.Errorf("Expected panic value 'simulated critical failure', got: %v", panicValue)
	}

	// Verify all expectations were met
	testError(t, nil, ExpectationsWereMet())
}

func TestMockWithPanicMultipleCalls(t *testing.T) {
	ctx := TestingContext(t)

	// First call panics
	Override(ctx, riskyOperation, Once, func(input string) string {
		Expectation().CheckArgs(input)
		panic("first panic")
	})("first")

	// Second call returns normally
	Override(ctx, riskyOperation, Once, func(input string) string {
		Expectation().CheckArgs(input)
		return "recovered"
	})("second")

	// First call - should panic
	_, recovered1, panicValue1 := safeWrapper("first")
	if !recovered1 {
		t.Errorf("Expected first call to panic")
	}
	if panicValue1 != "first panic" {
		t.Errorf("Expected panic value 'first panic', got: %v", panicValue1)
	}

	// Second call - should succeed
	result2, recovered2, _ := safeWrapper("second")
	if recovered2 {
		t.Errorf("Expected second call to not panic")
	}
	if result2 != "recovered" {
		t.Errorf("Expected result 'recovered', got: %s", result2)
	}

	// Verify all expectations were met
	testError(t, nil, ExpectationsWereMet())
}

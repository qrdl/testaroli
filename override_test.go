package testaroli

import (
	"context"
	"errors"
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
	return qux(errors.New("even"))
}

func baz(i int) error {
	return nil
}

func qux(err error) error {
	return err
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

func TestOverrideAfterUnlimited(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
		ExpectationsWereMet()
	}()

	ctx := TestingContext(t)
	Override(ctx, bar, Unlimited, func(i int) error {
		return nil
	})

	Override(ctx, baz, Once, func(i int) error {
		return nil
	})
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

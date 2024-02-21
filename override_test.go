package testaroli

import (
	"context"
	"errors"
	"testing"
)

type contextKey int

const key = contextKey(1)

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
	series := NewSeries(context.Background(), t)

	Override(bar, Once, func(i int) error {
		Expectation().CheckArgs(i)
		return nil
	})(2)

	err := foo(1)

	testError(t, nil, err)
	testError(t, nil, series.ExpectationsWereMet())
}

func TestSeveralCalls(t *testing.T) {
	series := NewSeries(context.WithValue(context.Background(), key, 100), t)

	Override(baz, Once, func(i int) error {
		e := Expectation()
		e.Expect(e.RunNumber() + e.Context().Value(key).(int))
		e.CheckArgs(i)
		return nil
	})

	err := foo(102)

	testError(t, nil, err)
	testError(t, nil, series.ExpectationsWereMet())
}

func TestSeveralCallsSeparateMocks(t *testing.T) {
	series := NewSeries(context.Background(), t)

	Override(baz, Once, func(i int) error {
		Expectation().CheckArgs(i)
		return nil
	})(100)

	Override(baz, Once, func(i int) error {
		Expectation().CheckArgs(i)
		return nil
	})(101)

	err := foo(102)

	testError(t, nil, err)
	testError(t, nil, series.ExpectationsWereMet())
}

func TestWrongExpectedArg(t *testing.T) {
	var t1 testing.T
	series := NewSeries(context.Background(), &t1)

	Override(bar, Unlimited, func(i int) error {
		Expectation().CheckArgs(i)
		return nil
	})(1)

	err := foo(1)

	testError(t, nil, err)
	testError(t, nil, series.ExpectationsWereMet())
	if !series.Testing().Failed() {
		t.Errorf("expected error")
	}
}

func TestExpect(t *testing.T) {
	series := NewSeries(context.Background(), t)

	Override(bar, Once, func(i int) error {
		Expectation().Expect(2).CheckArgs(i)
		return nil
	})

	err := foo(1)

	testError(t, nil, err)
	testError(t, nil, series.ExpectationsWereMet())
}

func TestModifyContext(t *testing.T) {
	val := 100
	series := NewSeries(context.WithValue(context.Background(), key, &val), t)

	Override(bar, Once, func(i int) error {
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
	testError(t, nil, series.ExpectationsWereMet())
	if *(series.Context().Value(key).(*int)) != 42 {
		t.Errorf("context not changed")
	}
}

func TestWrongArgTypes(t *testing.T) {
	var t1 testing.T
	series := NewSeries(context.Background(), &t1)

	Override(bar, Once, func(i int) error {
		Expectation().Expect("foo").CheckArgs(i)
		return nil
	})

	err := foo(1)

	testError(t, nil, err)
	testError(t, nil, series.ExpectationsWereMet())
	if !series.Testing().Failed() {
		t.Errorf("expected error")
	}
}

func TestWrongArgCount(t *testing.T) {
	var t1 testing.T
	series := NewSeries(context.Background(), &t1)

	Override(bar, Once, func(i int) error {
		Expectation().Expect(1, "foo").CheckArgs(i)
		return nil
	})

	err := foo(1)

	testError(t, nil, err)
	testError(t, nil, series.ExpectationsWereMet())
	if !series.Testing().Failed() {
		t.Errorf("expected error")
	}
}

func TestWrongArgCount2(t *testing.T) {
	var t1 testing.T
	series := NewSeries(context.Background(), &t1)

	Override(bar, Once, func(i int) error {
		Expectation().CheckArgs(i)
		return nil
	})

	err := foo(1)

	testError(t, nil, err)
	testError(t, nil, series.ExpectationsWereMet())
	if !series.Testing().Failed() {
		t.Errorf("expected error")
	}
}

func TestWrongCallCount(t *testing.T) {
	var t1 testing.T
	series := NewSeries(context.Background(), &t1)

	Override(bar, 2, func(i int) error {
		Expectation().CheckArgs(i)
		return nil
	})(2)

	err := foo(1)

	testError(t, nil, err)
	if series.ExpectationsWereMet() == nil {
		t.Errorf("expected error")
	}
}

func TestUnlimitedCallCount(t *testing.T) {
	series := NewSeries(context.Background(), t)

	Override(bar, Unlimited, func(i int) error {
		Expectation().CheckArgs(i)
		return nil
	})(2)

	err := foo(1)

	testError(t, nil, err)
	testError(t, nil, series.ExpectationsWereMet())
}

func TestExpectNil(t *testing.T) {
	series := NewSeries(context.Background(), t)

	Override(qux, Once, func(err error) error {
		Expectation().CheckArgs(err)
		return nil
	})(nil)

	err := foo(1)

	testError(t, nil, err)
	testError(t, nil, series.ExpectationsWereMet())
}

func TestExpectNilFail(t *testing.T) {
	var t1 testing.T
	series := NewSeries(context.Background(), &t1)

	Override(qux, Unlimited, func(err error) error {
		Expectation().CheckArgs(err)
		return nil
	})(errors.New("dummy"))

	err := foo(1)

	testError(t, nil, err)
	testError(t, nil, series.ExpectationsWereMet())
	if !series.Testing().Failed() {
		t.Errorf("expected error")
	}
}

func TestExpectNilFail2(t *testing.T) {
	var t1 testing.T
	series := NewSeries(context.Background(), &t1)

	Override(qux, Once, func(err error) error {
		Expectation().CheckArgs(err)
		return nil
	})(errors.New("dummy"))

	err := foo(1)

	testError(t, nil, err)
	testError(t, nil, series.ExpectationsWereMet())
	if !series.Testing().Failed() {
		t.Errorf("expected error")
	}
}

func TestInvalidExpectationCall(t *testing.T) {
	var t1 testing.T
	series := NewSeries(context.Background(), &t1)

	Expectation()

	if !series.Testing().Failed() {
		t.Errorf("expected error")
	}
}

func TestInvalidCount(t *testing.T) {
	_ = NewSeries(context.Background(), t)
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	Override(bar, 0, func(i int) error {
		return nil
	})
}

func TestInvalidOverride(t *testing.T) {
	_ = NewSeries(context.Background(), t)
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	var a, b int
	Override(1, a, b) // not functions
}

func TestOverrideAfterUnlimited(t *testing.T) {
	series := NewSeries(context.Background(), t)
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
		series.ExpectationsWereMet()
	}()

	Override(bar, Unlimited, func(i int) error {
		return nil
	})

	Override(baz, Once, func(i int) error {
		return nil
	})
}

func TestTwoMocks(t *testing.T) {
	series := NewSeries(context.Background(), t)
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
		series.ExpectationsWereMet()
	}()

	Override(bar, Once, func(i int) error {
		return nil
	})

	_ = NewSeries(context.Background(), t)
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

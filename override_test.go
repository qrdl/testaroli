package testaroli

import (
	"context"
	"errors"
	"testing"
)

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
	mock := New(context.Background(), t)

	Override(1, bar, func(i int) error {
		Expectation().CheckArgs(i)
		return nil
	})(2)

	err := foo(1)

	testError(t, nil, err)
	testError(t, nil, mock.ExpectationsWereMet())
}

func TestSeveralCalls(t *testing.T) {
	mock := New(context.WithValue(context.Background(), 1, 100), t)

	Override(2, baz, func(i int) error {
		e := Expectation()
		e.Expect(e.RunNumber() + e.Context().Value(1).(int))
		e.CheckArgs(i)
		return nil
	})

	err := foo(102)

	testError(t, nil, err)
	testError(t, nil, mock.ExpectationsWereMet())
}

func TestSeveralCallsSeparateMocks(t *testing.T) {
	mock := New(context.Background(), t)

	Override(1, baz, func(i int) error {
		Expectation().CheckArgs(i)
		return nil
	})(100)

	Override(1, baz, func(i int) error {
		Expectation().CheckArgs(i)
		return nil
	})(101)

	err := foo(102)

	testError(t, nil, err)
	testError(t, nil, mock.ExpectationsWereMet())
}

func TestWrongExpectedArg(t *testing.T) {
	var t1 testing.T
	mock := New(context.Background(), &t1)

	Override(Unlimited, bar, func(i int) error {
		Expectation().CheckArgs(i)
		return nil
	})(1)

	err := foo(1)

	testError(t, nil, err)
	testError(t, nil, mock.ExpectationsWereMet())
	if !mock.Testing().Failed() {
		t.Errorf("expected error")
	}
}

func TestExpect(t *testing.T) {
	mock := New(context.Background(), t)

	Override(1, bar, func(i int) error {
		Expectation().Expect(2).CheckArgs(i)
		return nil
	})

	err := foo(1)

	testError(t, nil, err)
	testError(t, nil, mock.ExpectationsWereMet())
}

func TestModifyContext(t *testing.T) {
	val := 100
	mock := New(context.WithValue(context.Background(), 1, &val), t)

	Override(1, bar, func(i int) error {
		e := Expectation()
		val := e.Context().Value(1).(*int)
		t := e.Testing()
		if *val != 100 {
			t.Errorf("unexpected context value")
		}
		*val = 42
		return nil
	})

	err := foo(1)

	testError(t, nil, err)
	testError(t, nil, mock.ExpectationsWereMet())
	if *(mock.Context().Value(1).(*int)) != 42 {
		t.Errorf("context not changed")
	}
}

func TestWrongArgTypes(t *testing.T) {
	var t1 testing.T
	mock := New(context.Background(), &t1)

	Override(1, bar, func(i int) error {
		Expectation().Expect("foo").CheckArgs(i)
		return nil
	})

	err := foo(1)

	testError(t, nil, err)
	testError(t, nil, mock.ExpectationsWereMet())
	if !mock.Testing().Failed() {
		t.Errorf("expected error")
	}
}

func TestWrongArgCount(t *testing.T) {
	var t1 testing.T
	mock := New(context.Background(), &t1)

	Override(1, bar, func(i int) error {
		Expectation().Expect(1, "foo").CheckArgs(i)
		return nil
	})

	err := foo(1)

	testError(t, nil, err)
	testError(t, nil, mock.ExpectationsWereMet())
	if !mock.Testing().Failed() {
		t.Errorf("expected error")
	}
}

func TestWrongArgCount2(t *testing.T) {
	var t1 testing.T
	mock := New(context.Background(), &t1)

	Override(1, bar, func(i int) error {
		Expectation().CheckArgs(i)
		return nil
	})

	err := foo(1)

	testError(t, nil, err)
	testError(t, nil, mock.ExpectationsWereMet())
	if !mock.Testing().Failed() {
		t.Errorf("expected error")
	}
}

func TestWrongCallCount(t *testing.T) {
	var t1 testing.T
	mock := New(context.Background(), &t1)

	Override(2, bar, func(i int) error {
		Expectation().CheckArgs(i)
		return nil
	})(2)

	err := foo(1)

	testError(t, nil, err)
	if mock.ExpectationsWereMet() == nil {
		t.Errorf("expected error")
	}
}

func TestUnlimitedCallCount(t *testing.T) {
	mock := New(context.Background(), t)

	Override(Unlimited, bar, func(i int) error {
		Expectation().CheckArgs(i)
		return nil
	})(2)

	err := foo(1)

	testError(t, nil, err)
	testError(t, nil, mock.ExpectationsWereMet())
}

func TestExpectNil(t *testing.T) {
	mock := New(context.Background(), t)

	Override(1, qux, func(err error) error {
		Expectation().CheckArgs(err)
		return nil
	})(nil)

	err := foo(1)

	testError(t, nil, err)
	testError(t, nil, mock.ExpectationsWereMet())
}

func TestExpectNilFail(t *testing.T) {
	var t1 testing.T
	mock := New(context.Background(), &t1)

	Override(Unlimited, qux, func(err error) error {
		Expectation().CheckArgs(err)
		return nil
	})(errors.New("dummy"))

	err := foo(1)

	testError(t, nil, err)
	testError(t, nil, mock.ExpectationsWereMet())
	if !mock.Testing().Failed() {
		t.Errorf("expected error")
	}
}

func TestExpectNilFail2(t *testing.T) {
	var t1 testing.T
	mock := New(context.Background(), &t1)

	Override(1, qux, func(err error) error {
		Expectation().CheckArgs(err)
		return nil
	})(errors.New("dummy"))

	err := foo(1)

	testError(t, nil, err)
	testError(t, nil, mock.ExpectationsWereMet())
	if !mock.Testing().Failed() {
		t.Errorf("expected error")
	}
}

func TestInvalidExpectationCall(t *testing.T) {
	var t1 testing.T
	mock := New(context.Background(), &t1)

	Expectation()

	if !mock.Testing().Failed() {
		t.Errorf("expected error")
	}
}

func TestInvalidCount(t *testing.T) {
	_ = New(context.Background(), t)
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	Override(0, bar, func(i int) error {
		return nil
	})
}

func TestInvalidOverride(t *testing.T) {
	_ = New(context.Background(), t)
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	var a, b int
	Override(1, a, b) // not functions
}

func TestOverrideAfterUnlimited(t *testing.T) {
	mock := New(context.Background(), t)
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
		mock.ExpectationsWereMet()
	}()

	Override(Unlimited, bar, func(i int) error {
		return nil
	})

	Override(1, baz, func(i int) error {
		return nil
	})
}

func TestTwoMocks(t *testing.T) {
	mock := New(context.Background(), t)
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
		mock.ExpectationsWereMet()
	}()

	Override(1, bar, func(i int) error {
		return nil
	})

	_ = New(context.Background(), t)
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

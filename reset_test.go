package testaroli

import (
	"errors"
	"testing"
)

func TestResetOnceFirst(t *testing.T) {
	Override(TestingContext(t), qux, Once, func(err error) error {
		Expectation()
		return nil
	})

	err := bar(3)
	testError(t, nil, err)

	Override(TestingContext(t), qux, Once, func(err error) error {
		Expectation()
		return nil
	})
	Reset(qux)

	err = bar(3)
	if err.Error() != "odd" {
		t.Errorf("unexpected")
	}
	testError(t, nil, ExpectationsWereMet())
}

func TestResetOnceNotFirst(t *testing.T) {
	Override(TestingContext(t), bar, Once, func(i int) error {
		Expectation()
		if i%2 != 0 {
			return qux(nil)
		}
		return qux(errors.New("even"))
	})
	Override(TestingContext(t), qux, Once, func(err error) error {
		Expectation()
		return nil
	})

	Reset(qux)

	err := bar(2)
	if err.Error() != "even" {
		t.Errorf("unexpected")
	}
	testError(t, nil, ExpectationsWereMet())
}

func TestResetAlways(t *testing.T) {
	Override(TestingContext(t), qux, Always, func(err error) error {
		Expectation()
		return nil
	})

	err := bar(3)
	testError(t, nil, err)

	err = bar(5)
	testError(t, nil, err)

	Reset(qux)
	err = bar(3)
	if err.Error() != "odd" {
		t.Errorf("unexpected")
	}
	testError(t, nil, ExpectationsWereMet())
}

func TestResetUnlimited(t *testing.T) {
	Override(TestingContext(t), qux, Unlimited, func(err error) error {
		Expectation()
		return nil
	})

	Override(TestingContext(t), qux, Once, func(err error) error {
		Expectation()
		return errors.New("test error")
	})

	err := bar(3)
	testError(t, nil, err)

	err = bar(5)
	testError(t, nil, err)

	Reset(qux) // Unlimited override is reset, so next Once override is effective

	err = bar(3)
	if err.Error() != "test error" {
		t.Errorf("unexpected")
	}
	testError(t, nil, ExpectationsWereMet())
}

func TestInvalidReset(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	Reset(1) // not functions
}

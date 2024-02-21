package main

import (
	"context"
	"errors"
	"testing"

	. "github.com/qrdl/testaroli"
)

func TestSuccess(t *testing.T) {
	series := NewSeries(context.TODO(), t)

	Override(accStatus, Once, func(acc string) AccStatus {
		Expectation().CheckArgs(acc)
		return AccStatusDebitable
	})("1024")

	Override(accStatus, Once, func(acc string) AccStatus {
		Expectation().CheckArgs(acc)
		return AccStatusCreditable
	})("2048")

	Override(accBalance, Once, func(acc string) float64 {
		Expectation().CheckArgs(acc)
		return 1000
	})("1024")

	Override(debit, Once, func(acc string, amount float64) {
		Expectation().CheckArgs(acc, amount)
	})("1024", 200)

	Override(credit, Once, func(acc string, amount float64) {
		Expectation().CheckArgs(acc, amount)
	})("2048", 200)

	err := transfer("1024", "2048", 200)
	testError(t, nil, err)
	testError(t, nil, series.ExpectationsWereMet())
}

func TestNoEnoughFunds(t *testing.T) {
	series := NewSeries(context.TODO(), t)

	Override(accStatus, Once, func(acc string) AccStatus {
		Expectation().CheckArgs(acc)
		return AccStatusDebitable
	})("1024")

	Override(accStatus, Once, func(acc string) AccStatus {
		Expectation().CheckArgs(acc)
		return AccStatusCreditable
	})("2048")

	Override(accBalance, Once, func(acc string) float64 {
		Expectation().Expect("1024").CheckArgs(acc)
		return 100
	})

	err := transfer("1024", "2048", 200)
	testError(t, ErrNotEnoughFunds, err)
	testError(t, nil, series.ExpectationsWereMet())
}

type contextKey int

const key = contextKey(1)

func TestNotCreditable(t *testing.T) {
	series := NewSeries(context.WithValue(context.TODO(), key, "1024"), t)
	defer func() { testError(t, nil, series.ExpectationsWereMet()) }()

	Override(accStatus, 2, func(acc string) AccStatus {
		f := Expectation()
		if f.RunNumber() == 0 {
			f.Expect(f.Context().Value(key).(string))
		} else {
			f.Expect("2048")
		}
		f.CheckArgs(acc)
		return AccStatusDebitable
	})

	err := transfer("1024", "2048", 200)
	testError(t, ErrInvalid, err)
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

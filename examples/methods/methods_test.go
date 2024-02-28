package main

import (
	"errors"
	"testing"

	. "github.com/qrdl/testaroli"
)

func TestTransferOK(t *testing.T) {
	err := Transfer("1024", "2048", 2.0)
	testError(t, nil, err)
}

func TestTransferDebitAccountNotOK(t *testing.T) {
	Override(TestingContext(t), Acc.IsDebitable, Once, func(Acc) bool {
		Expectation()
		return false
	})

	err := Transfer("1024", "2048", 2.0)
	testError(t, ErrInvalid, err)
	testError(t, nil, ExpectationsWereMet())
}

func TestTransferNotEnoughFunds(t *testing.T) {
	Override(TestingContext(t), Acc.Balance, Once, func(acc Acc) float64 {
		Expectation().CheckArgs(acc)
		return acc.balance * -1
	})(Acc{status: AccStatusDebitable | AccStatusCreditable, balance: 123.45, number: "1024"})

	err := Transfer("1024", "2048", 2.0)
	testError(t, ErrNotEnoughFunds, err)
	testError(t, nil, ExpectationsWereMet())
}

func TestTransferFail(t *testing.T) {
	Override(TestingContext(t), interAccountTransfer, Once, func(from, to *Acc, amount float64) error {
		Expectation().Expect("1024", "2048", 2.0).CheckArgs(from.number, to.number, amount)
		return ErrInvalid
	})

	err := Transfer("1024", "2048", 2.0)
	testError(t, ErrInvalid, err)
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
	if !errors.Is(expected, actual) {
		t.Errorf("got [%v] error when [%v] error expected", actual, expected)
		return
	}
}

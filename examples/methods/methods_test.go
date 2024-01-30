package main

import (
	"context"
	"errors"
	"testing"

	"github.com/qrdl/testaroli"
)

func TestTransferOK(t *testing.T) {
	err := Transfer("1024", "2048", 2.0)
	testError(t, nil, err)
}

func TestTransferDebitAccountNotOK(t *testing.T) {
	mock := testaroli.New(context.TODO(), t)

	testaroli.Override(1, Acc.IsDebitable, func(Acc) bool {
		testaroli.Expectation()
		return false
	})

	err := Transfer("1024", "2048", 2.0)
	testError(t, ErrInvalid, err)
	testError(t, nil, mock.ExpectationsWereMet())
}

func TestTransferNotEnoughFunds(t *testing.T) {
	mock := testaroli.New(context.TODO(), t)

	testaroli.Override(1, Acc.Balance, func(acc Acc) float64 {
		testaroli.Expectation().CheckArgs(acc)
		return acc.balance * -1
	})(Acc{status: AccStatusDebitable | AccStatusCreditable, balance: 123.45, number: "1024"})

	err := Transfer("1024", "2048", 2.0)
	testError(t, ErrNotEnoughFunds, err)
	testError(t, nil, mock.ExpectationsWereMet())
}

func TestTransferFail(t *testing.T) {
	mock := testaroli.New(context.TODO(), t)

	testaroli.Override(1, interAccountTransfer, func(from, to *Acc, amount float64) error {
		testaroli.Expectation().Expect("1024", "2048", 2.0).CheckArgs(from.number, to.number, amount)
		return ErrInvalid
	})

	err := Transfer("1024", "2048", 2.0)
	testError(t, ErrInvalid, err)
	testError(t, nil, mock.ExpectationsWereMet())
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

package main

import (
	"context"
	"testing"

	"github.com/qrdl/testaroli"
	"github.com/stretchr/testify/assert"
)

func TestTransferOK(t *testing.T) {
	err := Transfer("111", "222", 2.0)
	assert.NoError(t, err)
}

func TestTransferDebitAccountNotOK(t *testing.T) {
	testaroli.Instead(context.TODO(), Acc.IsDebitable, func(Acc) bool {
		defer testaroli.Restore(Acc.IsDebitable)
		return false
	})

	err := Transfer("111", "222", 2.0)
	assert.ErrorIs(t, err, ErrInvalid)
}

func TestTransferNotEnoughFunds(t *testing.T) {
	testaroli.Instead(context.TODO(), Acc.Balance, func(acc Acc) float64 {
		defer testaroli.Restore(Acc.Balance)
		return acc.balance * -1
	})

	err := Transfer("111", "222", 2.0)
	assert.ErrorIs(t, err, ErrNotEnoughFunds)
}

func TestTransferFail(t *testing.T) {
	testaroli.Instead(testaroli.Context(t), interAccountTransfer, func(from, to *Acc, amount float64) error {
		defer testaroli.Restore(interAccountTransfer)
		t := testaroli.Testing(testaroli.LookupContext(interAccountTransfer))
		assert.Equal(t, 2.0, amount)
		return ErrInvalid
	})

	err := Transfer("111", "222", 2.0)
	assert.ErrorIs(t, err, ErrInvalid)
}

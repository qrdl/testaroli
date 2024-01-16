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
	testaroli.Override(context.TODO(), Acc.IsDebitable, func(Acc) bool {
		defer testaroli.Reset(Acc.IsDebitable)
		return false
	})

	err := Transfer("111", "222", 2.0)
	assert.ErrorIs(t, err, ErrInvalid)
}

func TestTransferNotEnoughFunds(t *testing.T) {
	testaroli.Override(context.TODO(), Acc.Balance, func(acc Acc) float64 {
		defer testaroli.Reset(Acc.Balance)
		return acc.balance * -1
	})

	err := Transfer("111", "222", 2.0)
	assert.ErrorIs(t, err, ErrNotEnoughFunds)
}

func TestTransferFail(t *testing.T) {
	testaroli.Override(testaroli.NewContext(t), interAccountTransfer, func(from, to *Acc, amount float64) error {
		defer testaroli.Reset(interAccountTransfer)
		t := testaroli.Testing(testaroli.LookupContext(interAccountTransfer))
		assert.Equal(t, 2.0, amount)
		return ErrInvalid
	})

	err := Transfer("111", "222", 2.0)
	assert.ErrorIs(t, err, ErrInvalid)
}

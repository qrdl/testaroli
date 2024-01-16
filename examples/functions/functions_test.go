package main

import (
	"testing"

	"github.com/qrdl/testaroli"
	"github.com/stretchr/testify/assert"
)

func TestTransferOK(t *testing.T) {
	err := transfer("111", "222", 2.0)
	assert.NoError(t, err)
}

func TestTransferDebitAccountNotOK(t *testing.T) {
	testaroli.Override(testaroli.NewContext(t), isDebitable, func(acc string) bool {
		t := testaroli.Testing(testaroli.LookupContext(isDebitable))
		assert.Equal(t, "111", acc)
		return false
	})
	defer testaroli.Reset(isDebitable)

	err := transfer("111", "222", 2.0)
	assert.ErrorIs(t, err, ErrInvalid)
}

func TestTransferCreditAccountNotOK(t *testing.T) {
	testaroli.Override(testaroli.NewContext(t), isCreditable, func(acc string) bool {
		t := testaroli.Testing(testaroli.LookupContext(isCreditable))
		assert.Equal(t, "222", acc)
		return false
	})
	defer testaroli.Reset(isCreditable)

	err := transfer("111", "222", 2.0)
	assert.ErrorIs(t, err, ErrInvalid)
}

func TestTransferNotEnoughFunds(t *testing.T) {
	testaroli.Override(testaroli.NewContext(t), accBalance, func(acc string) float64 {
		t := testaroli.Testing(testaroli.LookupContext(accBalance))
		assert.Equal(t, "111", acc)
		return 1.0
	})
	defer testaroli.Reset(accBalance)

	err := transfer("111", "222", 2.0)
	assert.ErrorIs(t, err, ErrNotEnoughFunds)
}

func TestAccStatus(t *testing.T) {
	testaroli.Override(testaroli.NewContext(t), accStatus, func(acc string) AccStatus {
		ctx := testaroli.LookupContext(accStatus)
		t := testaroli.Testing(ctx)
		counter := testaroli.Increment(ctx)
		if counter == 0 {
			assert.Equal(t, "111", acc)
			return AccStatusDebitable
		} else {
			assert.Equal(t, "222", acc)
			return AccStatusCreditable
		}
	})
	defer testaroli.Reset(accStatus)

	err := transfer("111", "222", 2.0)
	assert.NoError(t, err)
}

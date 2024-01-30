package main

import (
	"errors"
)

type AccStatus int

const (
	AccStatusNone AccStatus = iota << 1
	AccStatusDebitable
	AccStatusCreditable
)

type Acc struct {
	number  string
	status  AccStatus
	balance float64
}

var ErrInvalid = errors.New("invalid account status")
var ErrNotEnoughFunds = errors.New("not enough funds")
var ErrNotFound = errors.New("unknown account")
var accounts = map[string]float64{
	"1024": 123.45,
	"2048": 234.56,
}

func Account(acc string) (*Acc, error) {
	if balance, ok := accounts[acc]; ok {
		return &Acc{number: acc, status: AccStatusDebitable | AccStatusCreditable, balance: balance}, nil
	}
	return nil, ErrNotFound
}

func (a Acc) Balance() float64 { return a.balance }

func (a Acc) IsDebitable() bool { return a.status&AccStatusDebitable != 0 }

func (a Acc) IsCreditable() bool { return a.status&AccStatusCreditable != 0 }

func (a *Acc) Debit(amt float64) { a.balance -= amt }

func (a *Acc) Credit(amt float64) { a.balance += amt }

func Transfer(from, to string, amount float64) error {
	acc1, err := Account(from)
	if err != nil {
		return err
	}
	if !acc1.IsDebitable() {
		return ErrInvalid
	}

	acc2, err := Account(to)
	if err != nil {
		return err
	}
	if !acc2.IsCreditable() {
		return ErrInvalid
	}

	if amount > acc1.Balance() {
		return ErrNotEnoughFunds
	}

	return interAccountTransfer(acc1, acc2, amount)
}

func interAccountTransfer(from, to *Acc, amount float64) error {
	from.Debit(amount)
	to.Credit(amount)
	return nil
}

package main

import (
	"errors"
	"fmt"
)

type AccStatus int

const (
	AccStatusNone AccStatus = iota << 1
	AccStatusDebitable
	AccStatusCreditable
)

var ErrInvalid = errors.New("invalid account status")
var ErrNotEnoughFunds = errors.New("not enough funds")
var accounts = map[string]float64{
	"111": 123.45,
	"222": 234.56,
}

func main() {
	if err := transfer("111", "222", 1.23); err != nil {
		fmt.Printf("Transfer failed: %v", err)
	} else {
		fmt.Println("Transfer successful")
	}
}

func transfer(from, to string, amount float64) error {
	if !isDebitable(from) || !isCreditable(to) {
		return ErrInvalid
	}

	if amount > accBalance(from) {
		return ErrNotEnoughFunds
	}

	debit(from, amount)
	credit(to, amount)

	return nil
}

func debit(acc string, amount float64) {
	accounts[acc] -= amount
}

func credit(acc string, amount float64) {
	accounts[acc] += amount
}

func isDebitable(acc string) bool {
	return accStatus(acc)&AccStatusDebitable != 0
}

func isCreditable(acc string) bool {
	return accStatus(acc)&AccStatusCreditable != 0
}

func accStatus(acc string) AccStatus {
	if _, ok := accounts[acc]; ok {
		return AccStatusDebitable | AccStatusCreditable
	}
	return AccStatusNone
}

func accBalance(acc string) float64 {
	if balance, ok := accounts[acc]; ok {
		return balance
	}
	return 0
}

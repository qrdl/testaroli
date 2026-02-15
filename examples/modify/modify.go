package main

import (
	"math/rand"
	"os"
	"time"
)

func genRandom() int {
	timestamp := time.Now().Unix()
	src := rand.NewSource(timestamp)
	r := rand.New(src)
	return r.Intn(100)
}

func openFile(filename string) (*os.File, error) {
	fs, err := os.Open(filename)
	if err == os.ErrPermission {
		return nil, os.ErrNotExist // hide file existance if user has no permissions
	}

	return fs, nil
}

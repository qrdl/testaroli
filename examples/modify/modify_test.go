package main

import (
	"errors"
	"math/rand"
	"os"
	"testing"
	"time"

	. "github.com/qrdl/testaroli"
)

func TestReplaceArg(t *testing.T) {
	Override(TestingContext(t), rand.NewSource, Once, func(seed int64) rand.Source {
		Expectation()
		// because Override count is Once, the original rand.NewSource is restored
		return rand.NewSource(12345) // use fixed seed to get deterministic results
	})

	res := genRandom()
	if res != 83 {
		t.Errorf("got [%d] when [83] expected", res)
	}

	testError(t, nil, ExpectationsWereMet())
}

func TestReplaceReturnValue(t *testing.T) {
	// for method object is the first argument ------------vvvvvvvvv
	Override(TestingContext(t), time.Time.Unix, Once, func(time.Time) int64 {
		Expectation()
		return 12345 // return fixed value
	})

	res := genRandom()
	if res != 83 {
		t.Errorf("got [%d] when [83] expected", res)
	}

	testError(t, nil, ExpectationsWereMet())
}

func TestRareCondition(t *testing.T) {
	Override(TestingContext(t), os.Open, Once, func(filename string) (*os.File, error) {
		Expectation().Expect("testdata/test.txt").CheckArgs(filename)
		return nil, os.ErrPermission
	})

	fs, err := openFile("testdata/test.txt")
	if fs != nil {
		t.Errorf("got non nil file when nil expected")
	}
	testError(t, os.ErrNotExist, err)
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

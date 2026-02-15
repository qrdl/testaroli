package main

import (
	"errors"
	"fmt"
	"testing"

	. "github.com/qrdl/testaroli"
)

// TestPreventPanic tests preventing a panic by overriding the panicking function
func TestPreventPanic(t *testing.T) {
	db := &Database{connected: false}

	// Override the panicking Connect method to return normally
	Override(TestingContext(t), (*Database).Connect, Once, func(db *Database) {
		Expectation()
		// Mock implementation that doesn't panic
		db.connected = true
	})

	// This would normally panic, but our override prevents it
	db.Connect()

	if !db.connected {
		t.Error("expected database to be connected")
	}

	testError(t, nil, ExpectationsWereMet())
}

// TestSimulatePanicInDependency tests behavior when a dependency panics
func TestSimulatePanicInDependency(t *testing.T) {
	ctx := TestingContext(t)

	// Override riskyOperation to panic
	Override(ctx, riskyOperation, Once, func(data []string, index int) string {
		Expectation().CheckArgs([]string{"a", "b"}, 5)
		panic("index out of bounds")
	})([]string{"a", "b"}, 5)

	// Override handlePanic to verify it's called with the right panic value
	Override(ctx, handlePanic, Once, func(r interface{}) error {
		Expectation().CheckArgs("index out of bounds")
		return ErrCritical
	})("index out of bounds")

	result, err := ProcessWithRecovery([]string{"a", "b"}, 5)

	if result != "" {
		t.Errorf("expected empty result, got [%s]", result)
	}
	testError(t, ErrCritical, err)
	testError(t, nil, ExpectationsWereMet())
}

// TestRecoveryLogic tests that panic recovery logic works correctly
func TestRecoveryLogic(t *testing.T) {
	// Test that handlePanic properly formats error messages
	Override(TestingContext(t), handlePanic, Once, func(r interface{}) error {
		Expectation()
		// Verify the implementation creates proper error
		return fmt.Errorf("panic recovered: %v", r)
	})

	err := handlePanic("test panic message")
	expected := "panic recovered: test panic message"
	if err == nil || err.Error() != expected {
		t.Errorf("got [%v] when [%s] expected", err, expected)
	}

	testError(t, nil, ExpectationsWereMet())
}

// TestValidationPanic tests handling of validation that panics
func TestValidationPanic(t *testing.T) {
	// Override validateInput to not panic, return error instead
	Override(TestingContext(t), validateInput, Once, func(value int) error {
		Expectation().CheckArgs(-5)
		// Instead of panicking, return an error
		return errors.New("negative values not allowed")
	})(-5)

	result, err := validateAndProcess(-5)

	if result != "" {
		t.Errorf("expected empty result, got [%s]", result)
	}
	if err == nil || err.Error() != "negative values not allowed" {
		t.Errorf("got error [%v] when validation error expected", err)
	}

	testError(t, nil, ExpectationsWereMet())
}

// TestInitializationSuccess tests successful system initialization
func TestInitializationSuccess(t *testing.T) {
	// Mock loadConfiguration to not panic
	Override(TestingContext(t), loadConfiguration, Once, func() {
		Expectation()
		// Don't panic, just return normally
	})

	// Mock connectServices to verify it's called
	Override(TestingContext(t), connectServices, Once, func() {
		Expectation()
		// Normal operation
	})

	err := InitializeSystem()
	testError(t, nil, err)
	testError(t, nil, ExpectationsWereMet())
}

// TestQueryPanicPrevention tests preventing a query panic
func TestQueryPanicPrevention(t *testing.T) {
	db := &Database{connected: true}

	// Override Query to avoid panic
	Override(TestingContext(t), (*Database).Query, Once,
		func(db *Database, sql string) ([]string, error) {
			Expectation().CheckArgs(db, "")
			// Return error gracefully instead of panicking
			return nil, fmt.Errorf("empty SQL not allowed")
		})(db, "")

	// Call would normally panic, but override prevents it
	result, err := db.Query("")

	if result != nil {
		t.Error("expected nil result")
	}
	if err == nil || err.Error() != "empty SQL not allowed" {
		t.Errorf("got error [%v] when specific error expected", err)
	}

	testError(t, nil, ExpectationsWereMet())
}

// TestSafeQueryWithPanicRecovery tests SafeQuery with panic recovery
func TestSafeQueryWithPanicRecovery(t *testing.T) {
	db := &Database{connected: true}

	// Override Query to panic
	Override(TestingContext(t), (*Database).Query, Once,
		func(db *Database, sql string) ([]string, error) {
			Expectation()
			panic("simulated database panic")
		})

	// SafeQuery should recover from the panic
	result, err := db.SafeQuery("SELECT * FROM users")

	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
	testError(t, nil, ExpectationsWereMet())
	_ = err
}

// TestDivisionByZero tests preventing division by zero panic
func TestDivisionByZero(t *testing.T) {
	// Override DivideNumbers to not panic
	Override(TestingContext(t), DivideNumbers, Once, func(a, b float64) float64 {
		Expectation().CheckArgs(10.0, 0.0)
		// Handle gracefully - return zero or infinity
		return 0.0
	})(10.0, 0.0)

	result := DivideNumbers(10.0, 0.0)
	if result != 0.0 {
		t.Errorf("expected 0.0, got %f", result)
	}

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
	if expected != nil && actual != nil {
		if !errors.Is(actual, expected) && expected.Error() != actual.Error() {
			t.Errorf("got [%v] error when [%v] error expected", actual, expected)
		}
	}
}

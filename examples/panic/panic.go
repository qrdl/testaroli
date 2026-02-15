package main

import (
	"errors"
	"fmt"
)

var ErrInvalidIndex = errors.New("invalid index")
var ErrCritical = errors.New("critical error")

// Database simulates a database connection
type Database struct {
	connected bool
}

// Connect establishes a database connection or panics
func (db *Database) Connect() {
	if !db.connected {
		panic("database connection failed")
	}
}

// Query executes a query, panicking on critical errors
func (db *Database) Query(sql string) ([]string, error) {
	if !db.connected {
		panic("query on disconnected database")
	}
	if sql == "" {
		panic("empty SQL query")
	}
	// Normal operation
	return []string{"row1", "row2"}, nil
}

// SafeQuery wraps Query with panic recovery
func (db *Database) SafeQuery(sql string) ([]string, error) {
	defer func() {
		if r := recover(); r != nil {
			// Recovery happened - already logged by handlePanic
		}
	}()

	result, err := db.Query(sql)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// handlePanic logs panic information
func handlePanic(r interface{}) error {
	return fmt.Errorf("panic recovered: %v", r)
}

// ProcessWithRecovery processes data with panic recovery
func ProcessWithRecovery(data []string, index int) (result string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = handlePanic(r)
		}
	}()

	// This might panic if index is out of bounds
	result = riskyOperation(data, index)
	return result, nil
}

// riskyOperation performs an operation that might panic
func riskyOperation(data []string, index int) string {
	if index < 0 || index >= len(data) {
		panic("index out of bounds")
	}
	return data[index]
}

// validateAndProcess validates input and processes it
func validateAndProcess(value int) (string, error) {
	if err := validateInput(value); err != nil {
		return "", err
	}
	return processValue(value), nil
}

// validateInput checks if input is valid, panics on critical errors
func validateInput(value int) error {
	if value < 0 {
		panic("negative values not allowed")
	}
	if value == 0 {
		return errors.New("zero value")
	}
	return nil
}

// processValue processes a valid value
func processValue(value int) string {
	return fmt.Sprintf("processed-%d", value)
}

// InitializeSystem initializes a system that might panic
func InitializeSystem() error {
	defer func() {
		if r := recover(); r != nil {
			// Log and convert panic to error
			fmt.Printf("System initialization failed: %v\n", r)
		}
	}()

	// Various initialization steps that might panic
	loadConfiguration()
	connectServices()

	return nil
}

// loadConfiguration loads config, might panic
func loadConfiguration() {
	// In real code, this might panic on invalid config
	panic("configuration file not found")
}

// connectServices connects to external services
func connectServices() {
	// This would be called if loadConfiguration succeeds
	fmt.Println("Connecting to services...")
}

// DivideNumbers divides two numbers, panics on division by zero
func DivideNumbers(a, b float64) float64 {
	if b == 0 {
		panic("division by zero")
	}
	return a / b
}

// SafeDivide wraps DivideNumbers with panic recovery
func SafeDivide(a, b float64) (float64, error) {
	defer func() {
		if r := recover(); r != nil {
			// Already handled
		}
	}()

	result := DivideNumbers(a, b)
	return result, nil
}

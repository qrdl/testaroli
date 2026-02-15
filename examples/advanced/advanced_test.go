package advanced

import (
	"context"
	"errors"
	"fmt"
	"testing"

	. "github.com/qrdl/testaroli"
)

// testError is a helper function to compare expected and actual errors
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

// Helper functions to be mocked in tests

// fetchData simulates fetching data from a remote service
func fetchData(id string) (string, error) {
	return fmt.Sprintf("real data for %s", id), nil
}

// processValue simulates processing a value
func processValue(val int) int {
	return val * 2
}

// stateMachine simulates a state machine transition function
func stateMachine(currentState string) (string, error) {
	switch currentState {
	case "init":
		return "processing", nil
	case "processing":
		return "complete", nil
	default:
		return "", errors.New("invalid state")
	}
}

// retry simulates a function that may fail and needs retrying
func retry(operation string) error {
	return errors.New("operation failed")
}

// calculateWithFallback tries primary calculation, falls back to secondary
func calculateWithFallback(x int) (int, error) {
	primary := primaryCalc(x)
	if primary < 0 {
		return secondaryCalc(x), nil
	}
	return primary, nil
}

func primaryCalc(x int) int {
	return x * 2
}

func secondaryCalc(x int) int {
	return x + 10
}

// TestMultiCountWithRunNumber demonstrates using multi-count override
// with RunNumber() to return different values on different calls
func TestMultiCountWithRunNumber(t *testing.T) {
	ctx := TestingContext(t)

	// Override fetchData to be called 3 times with different return values
	Override(ctx, fetchData, 3, func(id string) (string, error) {
		e := Expectation()
		e.CheckArgs(id)

		// Return different values based on which call this is
		switch e.RunNumber() {
		case 0:
			return "first call data", nil
		case 1:
			return "second call data", nil
		case 2:
			return "third call data", nil
		default:
			return "", errors.New("unexpected call")
		}
	})("user123")

	// Make three calls
	data1, err := fetchData("user123")
	testError(t, nil, err)
	if data1 != "first call data" {
		t.Errorf("expected 'first call data', got '%s'", data1)
	}

	data2, err := fetchData("user123")
	testError(t, nil, err)
	if data2 != "second call data" {
		t.Errorf("expected 'second call data', got '%s'", data2)
	}

	data3, err := fetchData("user123")
	testError(t, nil, err)
	if data3 != "third call data" {
		t.Errorf("expected 'third call data', got '%s'", data3)
	}

	testError(t, nil, ExpectationsWereMet())
}

// TestContextValuePassing demonstrates passing data from test to mock via context
func TestContextValuePassing(t *testing.T) {
	// Create context with values
	ctx := context.WithValue(TestingContext(t), "expected_value", 42)
	ctx = context.WithValue(ctx, "multiplier", 3)

	Override(ctx, processValue, Once, func(val int) int {
		e := Expectation()

		// Retrieve values from context
		expected := e.Context().Value("expected_value").(int)
		multiplier := e.Context().Value("multiplier").(int)

		e.CheckArgs(val)

		// Use context values in mock logic
		if val != expected {
			e.Testing().Errorf("expected value %d, got %d", expected, val)
		}

		return val * multiplier
	})(42)

	result := processValue(42)
	if result != 126 { // 42 * 3
		t.Errorf("expected 126, got %d", result)
	}

	testError(t, nil, ExpectationsWereMet())
}

// TestContextValueInMultiCall demonstrates using context values with RunNumber()
func TestContextValueInMultiCall(t *testing.T) {
	// Store expected values for each call in context
	expectedValues := map[int]string{
		0: "first",
		1: "second",
		2: "third",
	}
	ctx := context.WithValue(TestingContext(t), "expected_values", expectedValues)

	Override(ctx, fetchData, 3, func(id string) (string, error) {
		e := Expectation()
		e.CheckArgs(id)

		// Get expected value for current run from context
		values := e.Context().Value("expected_values").(map[int]string)
		expectedValue := values[e.RunNumber()]

		return fmt.Sprintf("%s: %s", id, expectedValue), nil
	})("item")

	// Make three calls
	for i := 0; i < 3; i++ {
		data, err := fetchData("item")
		testError(t, nil, err)
		expected := fmt.Sprintf("item: %s", expectedValues[i])
		if data != expected {
			t.Errorf("call %d: expected '%s', got '%s'", i, expected, data)
		}
	}

	testError(t, nil, ExpectationsWereMet())
}

// TestCombinedPatterns demonstrates combining RunNumber() and context values
func TestCombinedPatterns(t *testing.T) {
	// Define behavior for each run
	type behavior struct {
		shouldError bool
		multiplier  int
	}
	behaviors := map[int]behavior{
		0: {shouldError: false, multiplier: 2},
		1: {shouldError: true, multiplier: 0},
		2: {shouldError: false, multiplier: 5},
	}

	ctx := context.WithValue(TestingContext(t), "behaviors", behaviors)

	Override(ctx, processValue, 3, func(val int) int {
		e := Expectation()
		e.CheckArgs(val)

		// Get behavior for current run
		behaviors := e.Context().Value("behaviors").(map[int]behavior)
		b := behaviors[e.RunNumber()]

		if b.shouldError {
			// Simulate error by returning negative value
			return -1
		}

		return val * b.multiplier
	})(10)

	// First call: succeeds with multiplier 2
	result1 := processValue(10)
	if result1 != 20 {
		t.Errorf("call 1: expected 20, got %d", result1)
	}

	// Second call: "fails" (returns -1)
	result2 := processValue(10)
	if result2 != -1 {
		t.Errorf("call 2: expected -1, got %d", result2)
	}

	// Third call: succeeds with multiplier 5
	result3 := processValue(10)
	if result3 != 50 {
		t.Errorf("call 3: expected 50, got %d", result3)
	}

	testError(t, nil, ExpectationsWereMet())
}

// TestStateMachineTransitions demonstrates state machine pattern with multi-count
func TestStateMachineTransitions(t *testing.T) {
	ctx := TestingContext(t)

	// Define expected state transitions
	Override(ctx, stateMachine, 3, func(currentState string) (string, error) {
		e := Expectation()

		// Define expected transitions based on run number
		switch e.RunNumber() {
		case 0:
			e.Expect("init")
			e.CheckArgs(currentState)
			return "processing", nil
		case 1:
			e.Expect("processing")
			e.CheckArgs(currentState)
			return "validating", nil
		case 2:
			e.Expect("validating")
			e.CheckArgs(currentState)
			return "complete", nil
		default:
			return "", errors.New("unexpected state transition")
		}
	})

	// Simulate state machine execution
	state := "init"
	var err error

	// Transition 1: init -> processing
	state, err = stateMachine(state)
	testError(t, nil, err)
	if state != "processing" {
		t.Errorf("expected state 'processing', got '%s'", state)
	}

	// Transition 2: processing -> validating
	state, err = stateMachine(state)
	testError(t, nil, err)
	if state != "validating" {
		t.Errorf("expected state 'validating', got '%s'", state)
	}

	// Transition 3: validating -> complete
	state, err = stateMachine(state)
	testError(t, nil, err)
	if state != "complete" {
		t.Errorf("expected state 'complete', got '%s'", state)
	}

	testError(t, nil, ExpectationsWereMet())
}

// TestFallbackPatternWithRunNumber demonstrates fallback pattern using RunNumber()
func TestFallbackPatternWithRunNumber(t *testing.T) {
	ctx := TestingContext(t)

	// primaryCalc will fail (return negative), causing fallback to secondaryCalc
	Override(ctx, primaryCalc, Once, func(x int) int {
		Expectation().CheckArgs(x)
		return -1 // Indicate failure
	})(5)

	Override(ctx, secondaryCalc, Once, func(x int) int {
		Expectation().CheckArgs(x)
		return 15 // Fallback value
	})(5)

	result, err := calculateWithFallback(5)
	testError(t, nil, err)
	if result != 15 {
		t.Errorf("expected fallback result 15, got %d", result)
	}

	testError(t, nil, ExpectationsWereMet())
}

// TestMultiCountDifferentArguments demonstrates multi-count with varying arguments
func TestMultiCountDifferentArguments(t *testing.T) {
	ctx := TestingContext(t)

	// Define expected arguments for each call
	type callInfo struct {
		id       string
		expected string
	}
	calls := []callInfo{
		{"user1", "data for user1"},
		{"user2", "data for user2"},
		{"user3", "data for user3"},
	}

	ctx = context.WithValue(ctx, "calls", calls)

	Override(ctx, fetchData, 3, func(id string) (string, error) {
		e := Expectation()

		// Get expected info for this run
		calls := e.Context().Value("calls").([]callInfo)
		info := calls[e.RunNumber()]

		// Set expectation before checking
		e.Expect(info.id)
		e.CheckArgs(info.id)
		return info.expected, nil
	})

	// Make calls with different arguments
	for _, info := range calls {
		data, err := fetchData(info.id)
		testError(t, nil, err)
		if data != info.expected {
			t.Errorf("expected '%s', got '%s'", info.expected, data)
		}
	}

	testError(t, nil, ExpectationsWereMet())
}

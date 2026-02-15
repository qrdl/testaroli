package main

import (
	"errors"
	"fmt"
	"testing"

	. "github.com/qrdl/testaroli"
)

// TestVariadicNoArgs tests overriding a variadic function with no variadic arguments
func TestVariadicNoArgs(t *testing.T) {
	Override(TestingContext(t), formatList, Once, func(sep string, items ...string) string {
		// When checking args with variadic parameters, pass the variadic part as a slice
		Expectation().CheckArgs(sep, []string{})
		return "mocked-empty"
	})(", ") // Call with just the separator, no variadic args

	result := formatList(", ")
	if result != "mocked-empty" {
		t.Errorf("got [%s] when [mocked-empty] expected", result)
	}

	testError(t, nil, ExpectationsWereMet())
}

// TestVariadicSingleArg tests overriding with a single variadic argument
func TestVariadicSingleArg(t *testing.T) {
	Override(TestingContext(t), formatList, Once, func(sep string, items ...string) string {
		Expectation().CheckArgs(sep, []string{"one"})
		return "mocked-single"
	})(", ", "one") // Call with separator and one item

	result := formatList(", ", "one")
	if result != "mocked-single" {
		t.Errorf("got [%s] when [mocked-single] expected", result)
	}

	testError(t, nil, ExpectationsWereMet())
}

// TestVariadicMultipleArgs tests overriding with multiple variadic arguments
func TestVariadicMultipleArgs(t *testing.T) {
	Override(TestingContext(t), formatList, Once, func(sep string, items ...string) string {
		Expectation().CheckArgs(sep, []string{"alpha", "beta", "gamma"})
		return "mocked-multiple"
	})(" | ", "alpha", "beta", "gamma") // Call with separator and multiple items

	result := formatList(" | ", "alpha", "beta", "gamma")
	if result != "mocked-multiple" {
		t.Errorf("got [%s] when [mocked-multiple] expected", result)
	}

	testError(t, nil, ExpectationsWereMet())
}

// TestVariadicMethod tests overriding a variadic method
func TestVariadicMethod(t *testing.T) {
	logger := NewLogger("[TEST] ")

	// For methods, receiver is first arg, then regular args, then variadic args individually
	Override(TestingContext(t), (*Logger).Log, Once,
		func(l *Logger, format string, args ...interface{}) string {
			Expectation().CheckArgs(l, format, []interface{}{"REQ-123", 3})
			return "[TEST] mocked log"
		})(logger, "Processing request %s with %d parameters", "REQ-123", 3)

	result := logger.Log("Processing request %s with %d parameters", "REQ-123", 3)
	if result != "[TEST] mocked log" {
		t.Errorf("got [%s] when [[TEST] mocked log] expected", result)
	}

	testError(t, nil, ExpectationsWereMet())
}

// TestStdlibVariadic tests overriding standard library variadic function
func TestStdlibVariadic(t *testing.T) {
	Override(TestingContext(t), fmt.Sprintf, Once, func(format string, args ...interface{}) string {
		Expectation().CheckArgs(format, []interface{}{"test", 42})
		return "mocked sprintf"
	})("Format: %s = %d", "test", 42)

	result := fmt.Sprintf("Format: %s = %d", "test", 42)
	if result != "mocked sprintf" {
		t.Errorf("got [%s] when [mocked sprintf] expected", result)
	}

	testError(t, nil, ExpectationsWereMet())
}

// TestVariadicInContext tests calling variadic function in a larger context
func TestProcessRequestSuccess(t *testing.T) {
	ctx := TestingContext(t)

	// Mock the logger method
	Override(ctx, (*Logger).Log, Once,
		func(l *Logger, format string, args ...interface{}) string {
			Expectation()
			return l.prefix + fmt.Sprintf(format, args...)
		})

	// Mock formatList to verify it's called correctly
	Override(ctx, formatList, Once, func(sep string, items ...string) string {
		Expectation().CheckArgs(sep, []string{"param1", "param2", "param3"})
		return "param1, param2, param3"
	})(", ", "param1", "param2", "param3")

	result, err := ProcessRequest("REQ-001", "param1", "param2", "param3")
	testError(t, nil, err)

	expected := "Processed: REQ-001 [param1, param2, param3]"
	if result != expected {
		t.Errorf("got [%s] when [%s] expected", result, expected)
	}

	testError(t, nil, ExpectationsWereMet())
}

// TestVariadicErrorContext tests variadic error logging with different context lengths
func TestVariadicErrorContext(t *testing.T) {
	err := errors.New("connection failed")

	// Test with no context
	Override(TestingContext(t), logError, Once, func(err error, context ...string) string {
		Expectation().CheckArgs(err, []string{})
		return "mocked: no context"
	})(err) // No context args

	result := HandleError(err)
	if result != "mocked: no context" {
		t.Errorf("got [%s] when [mocked: no context] expected", result)
	}

	// Test with single context item
	Override(TestingContext(t), logError, Once, func(err error, context ...string) string {
		Expectation().CheckArgs(err, []string{"database"})
		return "mocked: single context"
	})(err, "database") // One context arg

	result = HandleError(err, "database")
	if result != "mocked: single context" {
		t.Errorf("got [%s] when [mocked: single context] expected", result)
	}

	// Test with multiple context items
	Override(TestingContext(t), logError, Once, func(err error, context ...string) string {
		Expectation().CheckArgs(err, []string{"database", "user-service", "retry-3"})
		return "mocked: multiple context"
	})(err, "database", "user-service", "retry-3") // Multiple context args

	result = HandleError(err, "database", "user-service", "retry-3")
	if result != "mocked: multiple context" {
		t.Errorf("got [%s] when [mocked: multiple context] expected", result)
	}

	testError(t, nil, ExpectationsWereMet())
}

// TestVariadicChain tests multiple overrides of a variadic function in sequence
func TestVariadicChain(t *testing.T) {
	ctx := TestingContext(t)

	// First call - no items
	Override(ctx, formatList, Once, func(sep string, items ...string) string {
		Expectation().CheckArgs(sep, []string{})
		return "first"
	})(", ")

	// Second call - one item
	Override(ctx, formatList, Once, func(sep string, items ...string) string {
		Expectation().CheckArgs(sep, []string{"item1"})
		return "second"
	})(", ", "item1")

	// Third call - multiple items
	Override(ctx, formatList, Once, func(sep string, items ...string) string {
		Expectation().CheckArgs(sep, []string{"a", "b", "c"})
		return "third"
	})(", ", "a", "b", "c")

	// Execute in order
	if result := formatList(", "); result != "first" {
		t.Errorf("first call: got [%s] when [first] expected", result)
	}

	if result := formatList(", ", "item1"); result != "second" {
		t.Errorf("second call: got [%s] when [second] expected", result)
	}

	if result := formatList(", ", "a", "b", "c"); result != "third" {
		t.Errorf("third call: got [%s] when [third] expected", result)
	}

	testError(t, nil, ExpectationsWereMet())
}

// TestVariadicWithoutExpectation tests override without argument validation
func TestVariadicWithoutExpectation(t *testing.T) {
	Override(TestingContext(t), concat, Once, func(strs ...string) string {
		Expectation() // Just mark as called, don't validate args
		return "concatenated"
	})

	// We can call with any arguments since we're not checking them
	result := concat("any", "number", "of", "args")
	if result != "concatenated" {
		t.Errorf("got [%s] when [concatenated] expected", result)
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
	if !errors.Is(expected, actual) {
		t.Errorf("got [%v] error when [%v] error expected", actual, expected)
		return
	}
}

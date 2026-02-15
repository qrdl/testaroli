package main

import (
	"fmt"
	"strings"
)

// Logger provides logging functionality with variadic methods
type Logger struct {
	prefix string
}

func NewLogger(prefix string) *Logger {
	return &Logger{prefix: prefix}
}

// Log formats and logs a message with optional arguments
func (l *Logger) Log(format string, args ...interface{}) string {
	msg := fmt.Sprintf(format, args...)
	return l.prefix + msg
}

// logError is a helper that logs errors with context
func logError(err error, context ...string) string {
	if err == nil {
		return "no error"
	}

	parts := []string{"ERROR:", err.Error()}
	if len(context) > 0 {
		parts = append(parts, "-", strings.Join(context, ", "))
	}
	return strings.Join(parts, " ")
}

// formatList formats a list of items with a separator
func formatList(separator string, items ...string) string {
	if len(items) == 0 {
		return ""
	}
	return strings.Join(items, separator)
}

// ProcessRequest simulates processing a request with logging
func ProcessRequest(requestID string, params ...string) (string, error) {
	logger := NewLogger("[REQUEST] ")

	// Log the request
	logger.Log("Processing request %s with %d parameters", requestID, len(params))

	// Simulate validation
	if len(params) == 0 {
		return "", fmt.Errorf("no parameters provided")
	}

	// Format parameters
	paramStr := formatList(", ", params...)

	// Return formatted result
	return fmt.Sprintf("Processed: %s [%s]", requestID, paramStr), nil
}

// HandleError handles errors with optional context information
func HandleError(err error, context ...string) string {
	msg := logError(err, context...)

	// In real code, this might write to a file or send to monitoring
	return msg
}

// concat concatenates strings with no separator
func concat(strs ...string) string {
	var result strings.Builder
	for _, s := range strs {
		result.WriteString(s)
	}
	return result.String()
}

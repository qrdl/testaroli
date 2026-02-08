package testaroli

import (
	"strings"
	"testing"
)

func genericFunc[T any](a T) *T {
	return &a
}

func TestGenericOverrideDetection(t *testing.T) {
	// This should panic with a helpful message when trying to override a generic function
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when overriding generic function")
		} else {
			msg, ok := r.(string)
			if !ok {
				t.Errorf("Expected string panic message, got %T: %v", r, r)
				return
			}

			t.Logf("Got expected panic: %s", msg)

			// Check that the message is helpful and contains key information
			if !strings.Contains(msg, "generic") {
				t.Errorf("Panic message should mention 'generic': %s", msg)
			}
			if !strings.Contains(msg, "reference") {
				t.Errorf("Panic message should mention 'reference': %s", msg)
			}
			if !strings.Contains(msg, "docs/generics.md") {
				t.Errorf("Panic message should reference documentation: %s", msg)
			}
		}
	}()

	// This should panic with a helpful error message
	Override(TestingContext(t), genericFunc[int], Once, func(a int) *int {
		Expectation()
		return nil
	})

	t.Error("Should not reach here - Override should have panicked")
}

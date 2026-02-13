package testaroli

import (
	"testing"
)

func genericFunc[T any](a T) *T {
	return &a
}

func TestGenericProper(t *testing.T) {
	intPointer := genericFunc[int]

	Override(TestingContext(t), intPointer, Once, func(arg int) *int {
		Expectation().CheckArgs(arg)
		return nil
	})(42)

	if intPointer(42) != nil {
		t.Error("Expected overridden function to return nil")
	}

	testError(t, nil, ExpectationsWereMet())
}

func TestGenericImproper(t *testing.T) {
	// Override the generic function (this patches the trampoline)
	Override(TestingContext(t), genericFunc[int], Once, func(arg int) *int {
		Expectation().CheckArgs(arg)
		return nil
	})(42)

	// ❌ IMPROPER: Direct call bypasses the trampoline
	// The compiler may optimize this to call the generic implementation directly
	// with the dictionary parameter, skipping the patched trampoline
	result := genericFunc(42)

	// This demonstrates the problem - the override doesn't work!
	if result == nil {
		// If this passes, we got lucky and the compiler used the trampoline
		t.Log("Override worked - compiler happened to use the trampoline")
	} else {
		// This is the expected behavior - direct calls bypass the override
		if *result != 42 {
			t.Errorf("Expected original function to return pointer to 42, got %v", *result)
		}
		t.Log("Override bypassed - direct call used original function (expected)")
	}

	// ExpectationsWereMet will fail because the override was never called
	err := ExpectationsWereMet()
	if err == nil {
		t.Error("Expected ExpectationsWereMet to fail - override was bypassed")
	} else {
		t.Logf("ExpectationsWereMet correctly failed: %v", err)
	}
}

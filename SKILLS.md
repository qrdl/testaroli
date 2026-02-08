# SKILLS.md - Function Override Generation Guide

## Overview

This guide provides step-by-step instructions for generating function and method overrides using the testaroli library. Follow these patterns to create effective mocks and stubs for Go unit testing.

## Prerequisites

Before generating overrides, ensure:
1. **Test flags are configured:** Add `-gcflags="all=-N -l"` to disable optimizations
   - VS Code: Add `"go.testFlags": [ "-gcflags", "all=-N -l" ]` to settings.json
   - Command line: `go test -gcflags="all=-N -l"`
2. **Import testaroli:** `import "github.com/qrdl/testaroli"`
3. **Testing context available:** Use `testing.T` or `testing.B`

## Core Override Pattern

### Basic Structure
```go
Override(
    TestingContext(t),        // Testing context wrapper
    targetFunc,               // Function/method to override
    callCount,                // How many times to apply (Once, N, Unlimited, Always)
    mockImplementation,       // Replacement function
)(expectedArgs...)            // Expected arguments (optional)
```

### Call Count Modes
- **Once** - Override applies one time only
- **N (positive integer)** - Override applies N times (e.g., 1, 2, 3, ...)
- **Unlimited** - Override active until manual reset
- **Always** - Override always effective (outside the chain)

## Step-by-Step Override Generation

### Step 1: Identify Target Function
Determine what needs to be mocked:
- Package function: `packageName.FunctionName`
- Method: `(*Type).MethodName`
- Standard library: `(*os.File).Read`, `fmt.Println`, etc.

### Step 2: Determine Call Pattern
Decide how many times the override should apply:
- Single call in test? → Use `Once`
- Multiple calls with different returns? → Chain overrides (see below)
- All calls in test? → Use `Unlimited`
- Permanent override? → Use `Always`

### Step 3: Create Mock Implementation
Write replacement function matching the target's signature:
```go
func(args...) returnTypes {
    // Optional: Verify arguments
    Expectation().CheckArgs(expectedArgs...)

    // Return mock data
    return mockValues...
}
```

### Step 4: Set Up Expectations (Optional)
Call the override result with expected arguments:
```go
Override(TestingContext(t), targetFunc, Once, mockFunc)(expectedArgs...)
```

## Pattern Library

### Pattern 1: Simple Function Override
```go
func TestExample(t *testing.T) {
    // Override a package function
    Override(TestingContext(t), mypackage.GetData, Once,
        func() string {
            Expectation()  // Track that mock was called
            return "mock data"
        })()

    // Test code that calls mypackage.GetData()
    result := mypackage.GetData()
    // Assertions...
}
```

### Pattern 2: Function with Argument Validation
```go
func TestWithArgs(t *testing.T) {
    expectedID := 42

    Override(TestingContext(t), mypackage.FetchUser, Once,
        func(id int) (*User, error) {
            Expectation().CheckArgs(id)  // Verify argument matches
            return &User{ID: id, Name: "Mock User"}, nil
        })(expectedID)  // Declare expected argument

    // Test code...
}
```

### Pattern 3: Method Override
```go
func TestMethodOverride(t *testing.T) {
    // Method receiver becomes first argument
    Override(TestingContext(t), (*Database).Query, Once,
        func(db *Database, sql string) (*Result, error) {
            Expectation().CheckArgs(sql)
            return &Result{Rows: []Row{mockRow}}, nil
        })("SELECT * FROM users")

    // Test code using Database.Query()...
}
```

### Pattern 4: Standard Library Override
```go
func TestFileRead(t *testing.T) {
    mockData := []byte("test content")

    Override(TestingContext(t), (*os.File).Read, Once,
        func(f *os.File, b []byte) (int, error) {
            Expectation()
            copy(b, mockData)
            return len(mockData), nil
        })()

    // Test code that opens and reads a file...
}
```

### Pattern 5: Error Simulation
```go
func TestErrorHandling(t *testing.T) {
    Override(TestingContext(t), mypackage.SaveData, Once,
        func(data string) error {
            Expectation().CheckArgs(data)
            return errors.New("mock error: database unavailable")
        })("test data")

    // Test error handling code...
}
```

### Pattern 6: Override Chain (Multiple Calls)
```go
func TestMultipleCalls(t *testing.T) {
    // First call returns "first"
    Override(TestingContext(t), mypackage.GetNext, Once,
        func() string {
            Expectation()
            return "first"
        })()

    // Second call returns "second"
    Override(TestingContext(t), mypackage.GetNext, Once,
        func() string {
            Expectation()
            return "second"
        })()

    // Third call returns "third"
    Override(TestingContext(t), mypackage.GetNext, Once,
        func() string {
            Expectation()
            return "third"
        })()

    // Test code that calls GetNext() three times...
    first := mypackage.GetNext()   // Returns "first"
    second := mypackage.GetNext()  // Returns "second"
    third := mypackage.GetNext()   // Returns "third"
}
```

### Pattern 7: Multiple Times Override
```go
func TestRepeatedCalls(t *testing.T) {
    callCount := 5

    // Override applies for 5 calls
    Override(TestingContext(t), mypackage.GetValue, callCount,
        func() int {
            Expectation()
            return 42
        })()

    // First 5 calls return 42, then original function is restored
    for i := 0; i < 5; i++ {
        value := mypackage.GetValue()  // Returns 42
        // Assertions...
    }
}
```

### Pattern 8: Unlimited Override
```go
func TestUnlimited(t *testing.T) {
    Override(TestingContext(t), mypackage.Random, Unlimited,
        func() int {
            Expectation()
            return 42  // Deterministic for testing
        })()

    // All calls in this test return 42
    // Override automatically restored when test ends
}
```

### Pattern 9: Always Override (Outside Chain)
```go
func TestAlways(t *testing.T) {
    // This override is always active
    Override(TestingContext(t), mypackage.GetConfig, Always,
        func() *Config {
            return &Config{Debug: true}
        })

    // No need to call Expectation() for Always overrides
    // They don't participate in the call chain
}
```

## Advanced Patterns

### Stateful Mock
```go
func TestStateful(t *testing.T) {
    callCount := 0

    Override(TestingContext(t), mypackage.Counter, Unlimited,
        func() int {
            Expectation()
            callCount++
            return callCount
        })()

    // Each call returns incrementing value
}
```

### Conditional Mock
```go
func TestConditional(t *testing.T) {
    Override(TestingContext(t), mypackage.Process, Unlimited,
        func(input string) (string, error) {
            Expectation()
            if input == "error" {
                return "", errors.New("mock error")
            }
            return "processed: " + input, nil
        })()

    // Mock behaves differently based on input
}
```

### Capturing Arguments
```go
func TestCapture(t *testing.T) {
    var capturedArgs []string

    Override(TestingContext(t), mypackage.LogMessage, Unlimited,
        func(msg string) {
            Expectation()
            capturedArgs = append(capturedArgs, msg)
        })()

    // Test code...

    // Verify captured arguments
    if len(capturedArgs) != 3 {
        t.Errorf("Expected 3 calls, got %d", len(capturedArgs))
    }
}
```

## Validation and Assertions

### Verify All Expectations Met
```go
func TestWithValidation(t *testing.T) {
    Override(TestingContext(t), mypackage.DoSomething, Once,
        func(arg string) {
            Expectation().CheckArgs(arg)
        })("expected")

    // Test code...

    // Verify all overrides were called as expected
    ExpectationsWereMet(t)
}
```

### Manual Validation
```go
func TestManual(t *testing.T) {
    override := Override(TestingContext(t), mypackage.Work, Once,
        func() {
            Expectation()
        })()

    mypackage.Work()

    // Check if override was called
    if !override.WasCalled() {
        t.Error("Expected Work() to be called")
    }
}
```

## Troubleshooting Guide

### Common Issues

**Issue:** "function was called but override not applied"
- **Solution:** Ensure `-gcflags="all=-N -l"` flag is set (disables inlining)

**Issue:** "override not restored after test"
- **Solution:** Use proper call count (Once, N, Unlimited) - avoid Always unless necessary

**Issue:** "arguments don't match"
- **Solution:** Check `CheckArgs()` receives same types and values as actual call

**Issue:** "cannot override generic function"
- **Solution:** Generic functions only work when called via reference (see docs/generics.md)

**Issue:** "panic: permission denied (memory protection)"
- **Solution:** Platform-specific issue - check OS/architecture support

## Best Practices

1. **Use Once for single calls** - Clearest intent and automatic restoration
2. **Chain overrides for sequences** - Create ordered behavior for multiple calls
3. **Call Expectation() in every mock** - Tracks call and enables validation
4. **Use CheckArgs() for critical tests** - Ensures function receives correct data
5. **Prefer Unlimited over Always** - Always overrides bypass the chain system
6. **Test one behavior per override** - Keep mocks simple and focused
7. **Clean up with ExpectationsWereMet()** - Validates test integrity
8. **Avoid overriding in production code** - TESTING ONLY

## Quick Reference

### Import and Setup
```go
import "github.com/qrdl/testaroli"
// Test flags: -gcflags="all=-N -l"
```

### Basic Override
```go
Override(TestingContext(t), targetFunc, Once, mockFunc)()
```

### With Arguments
```go
Override(TestingContext(t), targetFunc, Once, mockFunc)(expectedArgs...)
```

### Expectation Tracking
```go
Expectation()              // Track call
Expectation().CheckArgs()  // Track and validate
```

### Validation
```go
ExpectationsWereMet(t)  // Verify all expectations
```

## Related Documentation

- [Main Documentation](docs/README.md) - Comprehensive usage guide
- [Generics Limitations](docs/generics.md) - Generic function constraints
- [Interface Handling](docs/interfaces.md) - Working with interfaces
- [AGENTS.md](AGENTS.md) - Project architecture and design

---

*Generated: 2026-02-07*
*Purpose: AI Agent and Developer Reference for Function Override Generation*

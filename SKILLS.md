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
            Expectation()  // MUST call even for Always overrides
            return &Config{Debug: true}
        })

    // Always overrides don't participate in the call chain
    // but Expectation() must still be called to track execution
}
```

## Advanced Patterns

### Pattern 10: Variadic Function Override
```go
func TestVariadic(t *testing.T) {
    // Function: func formatList(sep string, items ...string) string
    
    Override(TestingContext(t), formatList, Once,
        func(sep string, items ...string) string {
            // Inside mock: items is a []string slice
            // In CheckArgs: pass variadic as slice
            Expectation().CheckArgs(sep, []string{"a", "b", "c"})
            return "mocked"
        })(", ", "a", "b", "c")  // Trailing call: individual args

    result := formatList(", ", "a", "b", "c")
    // result == "mocked"
}
```

**Key Rules for Variadic:**
- Inside mock: variadic parameter is a slice (`items ...string` → `items` is `[]string`)
- In `CheckArgs()`: pass variadic as slice (`[]string{"a", "b", "c"}`)
- In trailing call: pass as individual arguments (`"a", "b", "c"`)

### Pattern 11: Variadic Method Override
```go
func TestVariadicMethod(t *testing.T) {
    logger := &Logger{}
    
    // Method: func (l *Logger) Log(format string, args ...interface{}) string
    Override(TestingContext(t), (*Logger).Log, Once,
        func(l *Logger, format string, args ...interface{}) string {
            // Receiver first, then regular args, then variadic as slice
            Expectation().CheckArgs(l, format, []interface{}{"val1", 42})
            return "logged"
        })(logger, "Format: %s %d", "val1", 42)

    result := logger.Log("Format: %s %d", "val1", 42)
}
```

### Pattern 12: Passing Data to Mock via Context
```go
func TestContextData(t *testing.T) {
    // Problem: Cannot access outer variables from inside mock
    // Solution: Use context to pass data
    
    expectedValue := 100
    ctx := context.WithValue(TestingContext(t), "expected", expectedValue)
    
    Override(ctx, mypackage.Calculate, Once,
        func(x int) int {
            // Retrieve value from context
            expected := Expectation().Context().Value("expected").(int)
            Expectation().CheckArgs(x)
            return expected
        })(50)

    result := mypackage.Calculate(50)
    // result == 100
}
```

**Use context when:**
- Mock needs access to test data
- Combining with `RunNumber()` for complex scenarios
- Passing configuration or state to mock

### Pattern 13: Generic Function Override
```go
func TestGeneric(t *testing.T) {
    // Generic: func Max[T constraints.Ordered](a, b T) T
    
    // CRITICAL: Create function reference first
    maxInt := Max[int]  // Reference to instantiated generic
    
    Override(TestingContext(t), maxInt, Once,
        func(a, b int) int {
            Expectation().CheckArgs(a, b)
            return 999  // Mocked result
        })(10, 20)

    // MUST call via reference (not directly)
    result := maxInt(10, 20)  // ✅ Works
    // Max[int](10, 20)        // ❌ Would bypass override
}
```

**Generic limitations:**
- Cannot override generic functions directly
- Must create reference to type-instantiated function
- Call through reference, not direct generic call
- See [docs/generics.md](docs/generics.md) for details

### Pattern 14: Interface Implementation Override
```go
func TestInterface(t *testing.T) {
    // Interface: type Shape interface { Area() float64 }
    // Concrete: type square struct { side float64 }
    //           func (s square) Area() float64 { return s.side * s.side }
    
    // CRITICAL: Override concrete type's method, not interface
    Override(TestingContext(t), square.Area, Once,
        func(s square) float64 {
            Expectation()
            return 100.0  // Mocked area
        })

    s := square{side: 5}
    var shape Shape = s
    
    result := shape.Area()  // Calls mocked implementation
    // result == 100.0
}
```

**Interface limitations:**
- Cannot override interface methods directly
- Must override the concrete type's method that implements the interface
- Works for both value and pointer receivers: `square.Area` or `(*square).Area`
- See [docs/interfaces.md](docs/interfaces.md) for details

## Additional Techniques

These examples show techniques that can be combined with any pattern above:

### Stateful Mock (Using Variables)
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

### Conditional Logic in Mocks
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

### Capturing/Collecting Arguments
```go
func TestCapture(t *testing.T) {
    var capturedArgs []string

    Override(TestingContext(t), mypackage.LogMessage, Unlimited,
        func(msg string) {
            Expectation()
            capturedArgs = append(capturedArgs, msg)
        })()

    // Test code...

    // Verify captured arguments after
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

**Issue:** "expectations not met" or overrides never restore
- **Solution:** You forgot to call `Expectation()` inside the mock - it's REQUIRED for all overrides

## Best Practices

1. **Use Once for single calls** - Clearest intent and automatic restoration
2. **Chain overrides for sequences** - Create ordered behavior for multiple calls
3. **ALWAYS call Expectation() in every mock** - Required to track calls and enable proper chain management
4. **Use CheckArgs() for critical tests** - Ensures function receives correct data
5. **Prefer Unlimited over Always** - Always overrides bypass the chain system
6. **Test one behavior per override** - Keep mocks simple and focused
7. **Clean up with ExpectationsWereMet()** - Validates test integrity
8. **Avoid overriding in production code** - TESTING ONLY

## Quick Reference

**Core patterns:** Pattern Library (1-14) contains full examples  
**Techniques:** Additional Techniques section shows stateful, conditional, and capturing patterns

**Setup:**
```go
import "github.com/qrdl/testaroli"
// Required test flags: -gcflags="all=-N -l"
```

**Basic syntax:**
```go
Override(TestingContext(t), targetFunc, Once, mockFunc)(expectedArgs...)
```

**Key rules:**
- Always call `Expectation()` inside mock
- Methods: receiver becomes first argument
- Variadic: pass as slice in `CheckArgs`, individual args in trailing call
- Generics: create reference first (`fnRef := GenericFunc[int]`)
- Interfaces: override concrete type, not interface
- Context: use `context.WithValue` to pass data to mocks
- End test with `ExpectationsWereMet(t)`

## Related Documentation

- [Main Documentation](docs/README.md) - Comprehensive usage guide
- [Generics Limitations](docs/generics.md) - Generic function constraints
- [Interface Handling](docs/interfaces.md) - Working with interfaces
- [AGENTS.md](AGENTS.md) - Project architecture and design

---

*Generated: 2026-02-15*
*Purpose: AI Agent and Developer Reference for Function Override Generation*

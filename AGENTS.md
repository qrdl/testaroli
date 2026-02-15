# AGENTS.md - Testaroli Project Analysis

## Project Overview

**Name:** Testaroli
**Repository:** github.com/qrdl/testaroli
**Language:** Go (1.24.0+)
**License:** Apache License 2.0
**Purpose:** Monkey patching library for Go unit testing

Testaroli is a specialized Go package that enables runtime binary modification to override functions and methods with stubs/mocks during unit testing. It achieves this through low-level memory manipulation and executable patching.

## Core Functionality

### Primary Capabilities
- **Function Override:** Replace any function (including standard library functions) with mock implementations
- **Method Override:** Replace methods with mocks (method receiver becomes first argument)
- **Call Chain Management:** Maintain ordered override chains with call counting
- **Argument Verification:** Check that overridden functions receive expected arguments
- **Cross-Package Mocking:** Override functions from any package, including standard library

### Key Mechanisms
1. **Runtime Memory Patching:** Modifies executable code at runtime
2. **Override Chains:** Sequential override management with counters (Once, Unlimited, Always)
3. **Expectation Tracking:** Validates that mocks are called in expected order with correct arguments
4. **Automatic Restoration:** Resets overridden functions to original state after specified calls

## Architecture

### Core Components

#### 1. Override System (`override.go`)
- Main entry point for function/method overriding
- Manages override chain with different call count modes:
  - **Once** One time
  - **Fixed count (N > 0):** Any positive integer number of calls before restoration (e.g., 1, 2, 3, ...)
  - **Unlimited:** Active until manual reset
  - **Always:** Always effective, outside the chain
- Platform-specific implementations for memory protection

#### 2. Expectation System (`expect.go`)
- `Expect` struct tracks override metadata and call counts
- `Expectation()` function validates mock calls
- `CheckArgs()` method verifies function arguments
- `ExpectationsWereMet()` validates all expectations satisfied

#### 3. Equality Engine (`equal.go`)
- Custom equality comparison for argument validation
- Handles complex types that `reflect.Value.Equal` doesn't:
  - Pointers (compares values, not just addresses)
  - Maps and slices
  - Nested structures
- Provides detailed error messages for failures

#### 4. Platform-Specific Memory Management
- **Unix/Linux:** `mem_unix.go` - mprotect-based memory protection
- **Darwin/macOS:** `mem_darwin.go`, `mem_darwin.h` - Special handling for code signing
- **Windows:** `mem_windows.go` - VirtualProtect-based protection
- **Architecture-specific:** `override_amd64.go`, `override_arm64.go` - CPU instruction handling

## Platform Support

### Operating Systems
- ✅ Linux (x86-64, ARM64)
- ✅ Windows (x86-64, ARM64 via MSYS CLANGARM64)
- ✅ macOS (x86-64, ARM64)
- ✅ BSD (FreeBSD tested, others should work: NetBSD, OpenBSD, DragonFly BSD)

### Architecture Requirements
- x86-64 (amd64)
- ARM64 (aarch64)

Build tags ensure code only compiles on supported platforms: `//go:build (unix || windows) && (amd64 || arm64)`

## Technical Implementation Details

### Memory Manipulation Strategy
1. **Function Prologue Backup:** Save original function's first bytes
2. **Jump Injection:** Overwrite function start with jump to mock
3. **Memory Protection:** Temporarily make code memory writable
4. **Restoration:** Replace jump with original prologue when done

### Testing Requirements
- **Disable Optimizations:** Use `-gcflags="all=-N -l"` flag
- **Disable Inlining:** Prevents compiler from eliminating target functions
- VS Code integration: Add `"go.testFlags": [ "-gcflags", "all=-N -l" ]` to settings.json

## Override Count Modes

### Count Types Explained

**Once (or fixed number N > 0):**
- Override is active for exactly N calls
- After N calls, automatically restores original function
- Removed from chain after completion
- Example: `Once` (1 call), `2` (2 calls), `5` (5 calls)

**Unlimited:**
- Override is active until manually reset with `Reset()` or `ResetAll()`
- Part of the override chain (must be first non-Always override to be active)
- Blocks subsequent overrides until reset
- Use for overrides that should stay active for an indeterminate number of calls

**Always:**
- Override is ALWAYS active, independent of the chain
- Never automatically restored
- Sits outside the normal override chain
- Active simultaneously with other overrides
- Must be manually reset with `Reset()` or `ResetAll()`
- Use for global test fixtures or when you need an override active throughout entire test

### Override Chain Behavior

Overrides form a chain where only the first non-Always override is active:

```go
Override(ctx, fn, Once, mock1)     // Active immediately (first in chain)
Override(ctx, fn, Once, mock2)     // Queued (second in chain)
Override(ctx, fn, Always, mock3)   // Active immediately (Always is outside chain)

fn()  // Calls mock1 AND mock3 (both active)
fn()  // Calls mock2 AND mock3 (mock1 completed, mock2 now active)
fn()  // Calls original AND mock3 (mock2 completed, mock3 still Always)
```

## Expectation Management

### ExpectationsWereMet()
**Critical:** Always call at the end of tests to verify all expectations were satisfied:

```go
func TestFoo(t *testing.T) {
    Override(TestingContext(t), bar, Once, func(i int) error {
        Expectation().CheckArgs(42)
        return nil
    })(42)
    
    foo()  // Internally calls bar(42)
    
    // CRITICAL: Verify all overrides were called as expected
    if err := ExpectationsWereMet(); err != nil {
        t.Error(err)  // Reports: function not called, or called wrong number of times
    }
}
```

### Argument Validation Patterns

**With Expected Arguments:**
```go
Override(ctx, fn, Once, func(a int, b string) error {
    // Set expectations, then check
    Expectation().Expect(42, "test").CheckArgs(a, b)
    return nil
})
```

**Without Setting Expectations (validate in chain):**
```go
Override(ctx, fn, Once, func(a int, b string) error {
    Expectation().CheckArgs(42, "test")  // Expected values inline
    return nil
})(42, "test")  // Also set in trailing call
```

**No Argument Validation:**
```go
Override(ctx, fn, Once, func(a int, b string) error {
    Expectation()  // Just mark as called, don't validate
    return nil
})
```

### Reset Functions

**Reset(fn)** - Remove the first non-Always override:
```go
Override(ctx, bar, Once, mock1)
Override(ctx, bar, Unlimited, mock2)
Reset(bar)  // Removes mock1, mock2 becomes active
```

**ResetAll(fn)** - Remove ALL overrides (including Always):
```go
Override(ctx, bar, Once, mock1)
Override(ctx, bar, Always, mock2)
ResetAll(bar)  // Removes both overrides, restores original
```

## Limitations

### Generics
- **Limited support** for generic functions
- **Works:** When generic function is called via reference variable
- **Doesn't work:** Direct generic function calls
- **Pattern:**
  ```go
  fn := genericFunc[int]  // Create reference
  Override(ctx, fn, Once, mock)
  result := fn(42)  // Call via reference ✅
  // result := genericFunc[int](42)  // Direct call ❌
  ```
- See [docs/generics.md](docs/generics.md) for details

### Interface Methods
- **Cannot override interface methods directly**
- **Must override concrete type's method** that implements the interface
- **Pattern:**
  ```go
  // ❌ Wrong:
  Override(ctx, Shape.Area, ...)  // Interface method
  
  // ✅ Correct:
  Override(ctx, square.Area, ...)  // Concrete type method
  Override(ctx, (*square).Area, ...)  // Pointer receiver variant
  ```
- See [docs/interfaces.md](docs/interfaces.md) for details

### Other Constraints
- **Testing Only:** Never use in production code
- **Platform Dependent:** OS/architecture-specific implementation
- **Compiler Settings:** Requires specific compiler flags (`-gcflags="all=-N -l"`)
- **Inline Functions:** Cannot override inlined functions (reason for compiler flags)

## Project Structure

```
testaroli/
├── Core Library Files
│   ├── override.go          # Main override logic
│   ├── expect.go            # Expectation tracking and validation
│   ├── equal.go             # Custom equality comparison
│   └── interfaces_test.go   # Interface validation tests
│
├── Platform-Specific
│   ├── mem_unix.go          # Unix/Linux memory management
│   ├── mem_darwin.go        # macOS memory management
│   ├── mem_darwin.h         # macOS C headers
│   ├── mem_windows.go       # Windows memory management
│   ├── override_amd64.go    # x86-64 specific code
│   └── override_arm64.go    # ARM64 specific code
│
├── Tests
│   ├── equal_test.go        # Equality tests
│   ├── override_test.go     # Override functionality tests
│   ├── reset_test.go        # Reset functionality tests
│   └── mem_unix_test.go     # Unix memory tests
│
├── Documentation
│   ├── docs/README.md       # Main documentation
│   ├── docs/generics.md     # Generic functions limitations
│   ├── docs/interfaces.md   # Interface handling
│   └── docs/macOS.md        # macOS-specific notes
│
├── Examples
│   ├── examples/functions/  # Function override examples
│   ├── examples/methods/    # Method override examples
│   ├── examples/variadic/   # Variadic function examples
│   ├── examples/panic/      # Panic testing patterns
│   ├── examples/modify/     # Behavior modification examples
│   └── examples/advanced/   # Advanced patterns (RunNumber, context)
│
└── Build/Release
    └── go.mod               # Module definition
```

## Usage Patterns

### Basic Function Override
```go
Override(TestingContext(t), targetFunc, Once, func(args...) returnType {
    Expectation().CheckArgs(expectedArgs...)
    return mockResult
})(expectedArgs...)
```

### Method Override
**Important:** Method receiver becomes the first argument of the mock function.

```go
Override(TestingContext(t), (*Type).Method, Once,
    func(receiver *Type, args...) returnType {
        Expectation().CheckArgs(receiver, args...)  // receiver is first arg
        return mockResult
    })(receiverInstance, expectedArgs...)
```

### Standard Library Override
```go
Override(TestingContext(t), (*os.File).Read, Once,
    func(f *os.File, b []byte) (int, error) {
        Expectation()
        copy(b, []byte("mock data"))
        return len("mock data"), nil
    })
```

### Variadic Function Override
**Inside mock:** Variadic parameters become slices
**In CheckArgs:** Pass variadic as slice
**In Expectation call:** Pass as individual arguments

```go
// Function: func formatList(sep string, items ...string) string
Override(TestingContext(t), formatList, Once, 
    func(sep string, items ...string) string {
        // items is []string inside the mock
        Expectation().CheckArgs(sep, []string{"a", "b", "c"})  // Pass as slice
        return "mocked"
    })(", ", "a", "b", "c")  // Call with individual args

result := formatList(", ", "a", "b", "c")
```

### Variadic Method Override
```go
// Method: func (l *Logger) Log(format string, args ...interface{}) string
Override(TestingContext(t), (*Logger).Log, Once,
    func(l *Logger, format string, args ...interface{}) string {
        // Receiver first, then regular args, then variadic as slice
        Expectation().CheckArgs(l, format, []interface{}{"val1", 42})
        return "mocked"
    })(logger, "format %s %d", "val1", 42)
```

### Multi-Count Override with RunNumber()
Use `RunNumber()` to return different values on sequential calls:

```go
Override(TestingContext(t), fetchData, 3, func(id string) (string, error) {
    e := Expectation()
    e.CheckArgs(id)
    
    // Return different values based on call number
    switch e.RunNumber() {
    case 0:
        return "first call", nil
    case 1:
        return "second call", nil
    case 2:
        return "third call", nil
    default:
        return "", errors.New("unexpected call")
    }
})("userID")

// Now call fetchData 3 times - each gets different result
```

### Context Values with Multi-Count
Combine context values with `RunNumber()` for complex scenarios:

```go
expectedValues := map[int]string{
    0: "first",
    1: "second",
    2: "third",
}
ctx := context.WithValue(TestingContext(t), "expected", expectedValues)

Override(ctx, fetchData, 3, func(id string) (string, error) {
    e := Expectation()
    e.CheckArgs(id)
    
    // Get expected value from context based on run number
    values := e.Context().Value("expected").(map[int]string)
    return values[e.RunNumber()], nil
})("userID")
```

### Panic Prevention
Override panicking functions to prevent panics in tests:

```go
// db.Connect() normally panics when not connected
Override(TestingContext(t), (*Database).Connect, Once, 
    func(db *Database) {
        Expectation()
        // Mock implementation that doesn't panic
        db.connected = true
    })

db.Connect()  // No panic
```

### Panic Simulation
Simulate panics in dependencies to test recovery logic:

```go
Override(TestingContext(t), riskyOperation, Once, 
    func(data []string, index int) string {
        Expectation().CheckArgs([]string{"a", "b"}, 5)
        panic("index out of bounds")
    })([]string{"a", "b"}, 5)

result, err := ProcessWithRecovery(data, 5)  // Should recover from panic
```

## Dependencies

- **golang.org/x/sys v0.40.0** - System-level operations (memory protection, syscalls)
- **testing** (stdlib) - Integration with Go testing framework
- **reflect** (stdlib) - Runtime type inspection and manipulation
- **runtime** (stdlib) - Access to runtime internals
- **unsafe** (stdlib) - Low-level memory operations

## Development Workflow

### Testing
```bash
go test -gcflags="all=-N -l" ./...  # Run tests with optimizations disabled
```

### Testing Strategy
1. Unit tests for each component
2. Platform-specific test files
3. Integration tests in examples directory
4. Extensive test coverage (badge in README)

### Quality Assurance
- Go Report Card integration
- CodeQL security scanning
- Automated testing workflow (GitHub Actions)
- FOSSA license compliance

## Testing Scenarios and Patterns

### 1. Error Path Testing
Test error handling without requiring actual error conditions:

```go
// Test file read failure without needing unreadable file
Override(TestingContext(t), (*os.File).Read, Once,
    func(f *os.File, b []byte) (int, error) {
        Expectation()
        return 0, io.ErrUnexpectedEOF
    })
```

### 2. Rare Condition Testing
Simulate rare conditions (permissions, race conditions):

```go
// Test permission denied without changing OS permissions
Override(TestingContext(t), os.Open, Once,
    func(filename string) (*os.File, error) {
        Expectation().CheckArgs("secret.txt")
        return nil, os.ErrPermission
    })("secret.txt")
```

### 3. Deterministic Testing
Make non-deterministic code deterministic:

```go
// Fixed random seed
Override(TestingContext(t), rand.NewSource, Once,
    func(seed int64) rand.Source {
        Expectation()
        return rand.NewSource(12345)  // Fixed seed
    })

// Fixed time
Override(TestingContext(t), time.Time.Unix, Once,
    func(time.Time) int64 {
        Expectation()
        return 1234567890  // Fixed timestamp
    })
```

### 4. Sequential Behavior Testing
Test functions that call dependencies multiple times with different results:

```go
// First call succeeds, second fails
Override(ctx, apiCall, Once, mock1)  // Returns success
Override(ctx, apiCall, Once, mock2)  // Returns error

// Or use multi-count with RunNumber()
Override(ctx, apiCall, 2, func(arg string) error {
    e := Expectation()
    switch e.RunNumber() {
    case 0:
        return nil  // Success
    case 1:
        return ErrTimeout  // Failure
    }
})
```

### 5. State Machine Testing
Test complex state transitions:

```go
Override(ctx, transition, 3, func(state string) (string, error) {
    e := Expectation()
    switch e.RunNumber() {
    case 0:
        e.Expect("init")
        return "processing", nil
    case 1:
        e.Expect("processing")
        return "complete", nil
    case 2:
        e.Expect("complete")
        return "archived", nil
    }
})
```

### 6. Panic Recovery Testing
Test that panic handlers work correctly:

```go
// Simulate panic
Override(ctx, riskyFunc, Once, func() {
    Expectation()
    panic("simulated error")
})

// Verify recovery handler is called
Override(ctx, handlePanic, Once, func(r interface{}) error {
    Expectation().CheckArgs("simulated error")
    return ErrRecovered
})("simulated error")

// Test that code recovers properly
result, err := SafeOperation()
```

### 7. Call Order Verification
Verify functions are called in expected order:

```go
// Set up expected call order
Override(ctx, step1, Once, mock1)
Override(ctx, step2, Once, mock2)
Override(ctx, step3, Once, mock3)

// Execute function that should call step1, step2, step3 in order
workflow()

// ExpectationsWereMet() verifies order
if err := ExpectationsWereMet(); err != nil {
    t.Error(err)  // Will fail if wrong order
}
```

### 8. Conditional Path Testing
Test different execution paths based on conditions:

```go
// Test success path
Override(ctx, validate, Once, func(input string) error {
    Expectation().CheckArgs(input)
    return nil  // Validation passes
})("valid-input")

result := process("valid-input")  // Takes success path

// Test error path
Override(ctx, validate, Once, func(input string) error {
    Expectation().CheckArgs(input)
    return ErrInvalid  // Validation fails
})("invalid-input")

result := process("invalid-input")  // Takes error path
```

## Development Workflow

### Testing
```bash
go test -gcflags="all=-N -l" ./...  # Run tests with optimizations disabled
```

### Testing Strategy
1. Unit tests for each component
2. Platform-specific test files
3. Integration tests in examples directory
4. Extensive test coverage (badge in README)

### Quality Assurance
- Go Report Card integration
- CodeQL security scanning
- Automated testing workflow (GitHub Actions)
- FOSSA license compliance

## Key Insights for AI Agents

### When to Use This Project
- Unit testing Go code with hard-to-test dependencies
- Mocking functions from external packages or standard library
- Testing error paths that are difficult to trigger naturally
- Controlling execution order of dependent function calls
- Testing panic/recovery logic
- Making non-deterministic code deterministic (time, random)
- Testing rare conditions (filesystem errors, permission denials)

### When NOT to Use
- Production code (testing only!)
- With heavily inlined code
- Generic functions (limited support - needs reference pattern)
- On unsupported platforms
- When proper dependency injection is possible
- Interface methods directly (must use concrete types)

### Critical Patterns for AI Agents to Remember

#### 1. Method Signatures
**Methods:** Receiver becomes first argument:
```go
// Original: func (db *Database) Query(sql string) ([]string, error)
// Override:
Override(ctx, (*Database).Query, Once, 
    func(db *Database, sql string) ([]string, error) {
        // db is first parameter!
        Expectation().CheckArgs(db, sql)
        return []string{"mock"}, nil
    })(dbInstance, "SELECT *")
```

#### 2. Variadic Functions
Three key rules:
- Inside mock: variadic params are slices (`items ...string` → `items` is `[]string`)
- In CheckArgs: pass variadic as slice (`CheckArgs(sep, []string{"a", "b"})`)
- In expectation call: pass as individual args (`Override(...)(sep, "a", "b")`)

```go
// func formatList(sep string, items ...string) string
Override(ctx, formatList, Once, 
    func(sep string, items ...string) string {
        Expectation().CheckArgs(sep, []string{"a", "b"})  // As slice
        return "mocked"
    })(", ", "a", "b")  // Individual args
```

#### 3. Always Call ExpectationsWereMet()
Every test using Override() should end with:
```go
if err := ExpectationsWereMet(); err != nil {
    t.Error(err)
}
```

#### 4. Multi-Count with RunNumber()
For different behavior per call:
```go
Override(ctx, fn, 3, func(arg string) string {
    e := Expectation()
    e.CheckArgs(arg)
    
    switch e.RunNumber() {  // 0, 1, 2 for three calls
    case 0:
        return "first"
    case 1:
        return "second"
    case 2:
        return "third"
    }
})("input")
```

#### 5. Context for Data Passing
Cannot access outer variables from mock - use context:
```go
ctx := context.WithValue(TestingContext(t), "key", value)
Override(ctx, fn, Once, func(arg int) int {
    val := Expectation().Context().Value("key").(int)
    return val
})
```

### Mock Implementation Scope: Passing Data

**Important:** Mock functions are executed in the scope of the replaced function, not the test's lexical scope. This means you cannot access variables from the outer (test) function directly inside the mock.

**Best Practice:** To pass data from your test to the mock, use the context argument. Store values in the context and retrieve them inside the mock using `Expectation().Context()`.

**Example:**
```go
ctx := context.WithValue(TestingContext(t), "expected", 42)
Override(ctx, fn, Once, func(a int) int {
    expected := Expectation().Context().Value("expected").(int)
    Expectation().CheckArgs(a)
    return expected
})
```

This ensures your mock has access to any required data, even though it cannot see outer variables.

### Common Mistakes to Avoid

1. **Forgetting receiver in method mocks:**
   ```go
   // ❌ Wrong:
   Override(ctx, (*Type).Method, Once, func(arg int) int { ... })
   
   // ✅ Correct:
   Override(ctx, (*Type).Method, Once, func(t *Type, arg int) int { ... })
   ```

2. **Wrong variadic argument handling:**
   ```go
   // ❌ Wrong:
   Expectation().CheckArgs(sep, "a", "b", "c")  // Variadic as individual args
   
   // ✅ Correct:
   Expectation().CheckArgs(sep, []string{"a", "b", "c"})  // As slice
   ```

3. **Not calling ExpectationsWereMet():**
   - Without this, you won't know if mocks were called correctly

4. **Trying to override interface methods:**
   ```go
   // ❌ Wrong:
   Override(ctx, Shape.Area, ...)
   
   // ✅ Correct:
   Override(ctx, square.Area, ...)  // Concrete type
   ```

5. **Missing compiler flags:**
   - Tests will fail mysteriously without `-gcflags="all=-N -l"`

6. **Accessing outer variables in mocks:**
   ```go
   expected := 42
   // ❌ Wrong:
   Override(ctx, fn, Once, func(a int) int {
       return expected  // Won't work - different scope!
   })
   
   // ✅ Correct:
   ctx := context.WithValue(TestingContext(t), "expected", 42)
   Override(ctx, fn, Once, func(a int) int {
       val := Expectation().Context().Value("expected").(int)
       return val
   })
   ```

7. **Confusing Always and Unlimited:**
   - **Unlimited:** In chain, blocks others until reset
   - **Always:** Outside chain, active alongside others

### Best Practices for AI Agents

1. **Always generate complete test structure:**
   ```go
   func TestFoo(t *testing.T) {
       Override(TestingContext(t), fn, Once, mock)(expectedArgs)
       // Test code
       if err := ExpectationsWereMet(); err != nil {
           t.Error(err)
       }
   }
   ```

2. **Use appropriate count modes:**
   - `Once` for single calls
   - Fixed number (2, 3, etc.) when you know exact count
   - `Unlimited` when you need manual control
   - `Always` for test-wide fixtures

3. **Validate arguments when meaningful:**
   - Use `CheckArgs()` when argument values matter
   - Skip validation when only call count matters

4. **Suggest context for complex data:**
   - When mock needs multiple values
   - When combining with RunNumber()
   - For state that changes between calls

5. **Chain overrides for sequential behavior:**
   ```go
   Override(ctx, fn, Once, mock1)  // First call
   Override(ctx, fn, Once, mock2)  // Second call
   Override(ctx, fn, Once, mock3)  // Third call
   ```

6. **Use RunNumber() for varying behavior:**
   - When same function needs different results per call
   - Cleaner than chaining separate overrides

### AI Agent Considerations
1. **Memory Safety:** This code manipulates raw memory - suggest careful review
2. **Platform Awareness:** Recommend checking platform before suggesting usage
3. **Compiler Flags:** Always mention required `-gcflags` flags
4. **Testing Context:** Emphasize this is testing infrastructure only
5. **Alternative Solutions:** Consider suggesting dependency injection as alternative
6. **Method Receivers:** Always remind about receiver as first argument
7. **Variadic Functions:** Explain the three-part rule (mock as slice, CheckArgs as slice, call as individual)
8. **Panic Patterns:** Useful for testing recovery logic or preventing test failures

## Code Generation Guidelines for AI Agents

### Template for Basic Test
```go
func Test<Name>(t *testing.T) {
    // Setup: Create context
    ctx := TestingContext(t)
    
    // Override: Set up mock function(s)
    Override(ctx, targetFunc, Once, func(args...) returnType {
        Expectation().CheckArgs(expectedArgs...)
        return mockResult
    })(expectedArgs...)
    
    // Execute: Call function under test
    result := functionUnderTest(...)
    
    // Assert: Verify result
    if result != expected {
        t.Errorf("got %v, expected %v", result, expected)
    }
    
    // Verify: Check all mocks were called
    if err := ExpectationsWereMet(); err != nil {
        t.Error(err)
    }
}
```

### Template for Method Override
```go
func Test<Name>(t *testing.T) {
    instance := &Type{...}
    
    // NOTE: Receiver becomes first argument
    Override(TestingContext(t), (*Type).Method, Once,
        func(receiver *Type, args...) returnType {
            // Validate receiver if needed
            Expectation().CheckArgs(receiver, expectedArgs...)
            return mockResult
        })(instance, expectedArgs...)
    
    result := instance.Method(...)
    
    if err := ExpectationsWereMet(); err != nil {
        t.Error(err)
    }
}
```

### Template for Variadic Function
```go
func Test<Name>(t *testing.T) {
    Override(TestingContext(t), variadicFunc, Once,
        func(arg1 Type1, varArgs ...Type2) ReturnType {
            // NOTE: Pass variadic as slice in CheckArgs
            Expectation().CheckArgs(arg1, []Type2{...})
            return mockResult
        })(arg1Value, varArg1, varArg2, varArg3)  // Individual args
    
    result := variadicFunc(arg1Value, varArg1, varArg2, varArg3)
    
    if err := ExpectationsWereMet(); err != nil {
        t.Error(err)
    }
}
```

### Template for Multi-Count with RunNumber
```go
func Test<Name>(t *testing.T) {
    Override(TestingContext(t), targetFunc, 3, func(arg Type) ReturnType {
        e := Expectation()
        e.CheckArgs(arg)
        
        switch e.RunNumber() {
        case 0:
            return firstResult
        case 1:
            return secondResult
        case 2:
            return thirdResult
        default:
            panic("unexpected call")
        }
    })(expectedArg)
    
    // Call function 3 times
    result1 := targetFunc(expectedArg)
    result2 := targetFunc(expectedArg)
    result3 := targetFunc(expectedArg)
    
    // Verify results
    // ... assertions ...
    
    if err := ExpectationsWereMet(); err != nil {
        t.Error(err)
    }
}
```

### Template for Context Values
```go
func Test<Name>(t *testing.T) {
    // Store data in context
    expectedData := map[string]interface{}{
        "key1": value1,
        "key2": value2,
    }
    ctx := context.WithValue(TestingContext(t), "testData", expectedData)
    
    Override(ctx, targetFunc, Once, func(arg Type) ReturnType {
        // Retrieve data from context
        data := Expectation().Context().Value("testData").(map[string]interface{})
        
        // Use data in mock logic
        return data["key1"].(ReturnType)
    })
    
    result := targetFunc(arg)
    
    if err := ExpectationsWereMet(); err != nil {
        t.Error(err)
    }
}
```

### Checklist for AI Agents When Generating Override Code

1. **Context Creation**
   - [ ] Use `TestingContext(t)` or `context.WithValue(TestingContext(t), ...)`
   - [ ] Never use `context.Background()`

2. **Function Signature**
   - [ ] Match exact signature of target function
   - [ ] For methods: receiver is first parameter
   - [ ] For variadic: use `...Type` in mock signature

3. **Inside Mock Function**
   - [ ] Always call `Expectation()` (or assign to variable if using methods)
   - [ ] For variadic: pass as slice to `CheckArgs()`
   - [ ] Access context values via `Expectation().Context().Value()`
   - [ ] Use `RunNumber()` for multi-count overrides with varying behavior

4. **Trailing Call**
   - [ ] Add trailing call with expected args: `Override(...)(args...)`
   - [ ] For variadic: pass individual args (not slice)
   - [ ] Match what you expect in `CheckArgs()`

5. **Test Completion**
   - [ ] Call `ExpectationsWereMet()` at end
   - [ ] Check error and call `t.Error(err)` if not nil

6. **Count Mode Selection**
   - [ ] `Once` for single call
   - [ ] Positive integer (2, 3, ...) for fixed multiple calls
   - [ ] `Unlimited` for manual reset control
   - [ ] `Always` for always-active overrides

### When to Suggest What

**Suggest `Once`** when:
- Test involves a single call to the mocked function
- Clear, simple test case

**Suggest fixed count (2, 3, ...)** when:
- Known number of calls
- Use with `RunNumber()` for varying behavior

**Suggest `Unlimited`** when:
- Unknown number of calls
- Need manual control over when mock is disabled
- Loop with variable iteration count

**Suggest `Always`** when:
- Override should apply to entire test
- Global test fixture pattern
- Never want automatic restoration

**Suggest chaining** when:
- Need different behavior on sequential calls
- Don't want to use RunNumber()
- Clear sequence of behaviors

**Suggest RunNumber()** when:
- Same function, different results per call
- Cleaner than multiple Override calls
- State machine or sequential behavior pattern

**Suggest context values** when:
- Need to pass data to mock
- Complex configuration
- Multiple related values
- Combining with RunNumber() for complex scenarios

## Retracted Versions
Versions v0.1.0 through v0.3.2 are retracted - avoid using these versions.

## Community & Support
- GitHub: https://github.com/qrdl/testaroli
- Go Package Reference: https://pkg.go.dev/github.com/qrdl/testaroli
- Issues: Track platform-specific issues on GitHub
- Examples: See `examples/` directory for practical usage patterns

---

*Generated: 2026-02-14*
*Analysis Type: Project Structure, Functionality, and AI Agent Guidance*
*Comprehensive Instructions: All Testing Scenarios and Patterns*

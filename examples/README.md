# Testaroli Examples

This directory contains practical examples demonstrating how to use Testaroli for mocking and testing in Go.

## Prerequisites

All examples must be run with compiler optimizations and inlining disabled:

```bash
go test -gcflags="all=-N -l" ./...
```

Or from VS Code, add to your `settings.json`:
```json
"go.testFlags": [ "-gcflags", "all=-N -l" ]
```

## Examples Overview

### 1. [functions/](functions/) - Function Overriding

Demonstrates overriding regular functions in a banking transfer scenario. Shows:
- Basic function override using `Override()` with `Once` count
- Multiple sequential overrides forming a chain
- Argument validation using `CheckArgs()`
- Testing different execution paths (success, insufficient funds, invalid accounts)
- Using context values to pass data to mock functions

**Key Concepts:**
- `Override(ctx, targetFunc, Once, mockFunc)(expectedArgs...)`
- `Expectation().CheckArgs(args...)` for argument verification
- `ExpectationsWereMet()` to ensure all overrides were called
- `RunNumber()` to handle different behavior per call in multi-count overrides
- Context values for passing data to mocks

**Use Cases:**
- Testing business logic with multiple function dependencies
- Simulating different states without complex setup
- Verifying function call order and arguments

### 2. [methods/](methods/) - Method Overriding

Demonstrates overriding methods on structs, using the same banking scenario as functions example.

**Key Concepts:**
- Methods use the receiver as the first argument: `Override(ctx, (*Type).Method, ...)`
- Works with both pointer and value receivers
- Can override methods from any package, including your own

**Use Cases:**
- Mocking object behavior without interfaces
- Testing code that uses concrete types
- Isolating units that depend on complex objects

### 3. [modify/](modify/) - Behavior Modification

Demonstrates modifying behavior by replacing arguments and return values, plus testing rare conditions.

**Key Concepts:**
- Replacing function arguments to make tests deterministic
- Modifying return values to control execution flow
- Testing rare error conditions (like `os.ErrPermission`)
- Overriding standard library functions (`rand.NewSource`, `time.Time.Unix`, `os.Open`)

**Use Cases:**
- Making non-deterministic code deterministic (random, time)
- Testing error paths that are difficult to trigger naturally
- Controlling external dependencies (filesystem, network)

### 4. [variadic/](variadic/) - Variadic Function Testing

Demonstrates overriding functions with variadic parameters (functions that accept variable numbers of arguments).

**Key Concepts:**
- Overriding functions with `...` variadic parameters
- Variadic parameters become slices inside the mock function: `func(args ...string)` → `args` is `[]string`
- When checking arguments, variadic parameters are compared as slices: `CheckArgs(arg1, []string{...})`
- When setting expectations, call with individual arguments: `Override(...)(arg1, "item1", "item2")`
- Works with stdlib variadic functions like `fmt.Sprintf`
- Method overrides with variadic parameters (receiver is first arg)

**Use Cases:**
- Mocking logging functions (`Log(format string, args ...interface{})`)
- Mocking formatting functions (`fmt.Sprintf`, `strings.Join`)
- Testing functions that accept variable-length argument lists
- Validating different numbers of arguments (zero, one, many)

### 5. [panic/](panic/) - Panic Testing and Prevention

Demonstrates testing code that uses panic/recover patterns and preventing panics during tests.

**Key Concepts:**
- Override functions that would panic to prevent panics in tests
- Test panic recovery logic by simulating panics in mocked dependencies
- Verify that `defer`/`recover` handlers are called correctly
- Test error paths that normally involve panics (division by zero, invalid input, database errors)
- Chain overrides to test panic → recovery → success sequences

**Use Cases:**
- Testing code with defensive panic recovery
- Preventing third-party library panics from breaking tests
- Validating that panic handlers log/report correctly
- Testing critical failure scenarios safely
- Making panic-prone code testable without modifying it

### 6. [advanced/](advanced/) - Advanced Patterns

Demonstrates advanced Testaroli patterns including multi-count overrides and context-based data passing.

**Key Concepts:**
- **Multi-Count Overrides**: Use `RunNumber()` to return different values on sequential calls
- **Context Data Passing**: Pass test data to mocks via `context.WithValue` and access in mocks via `Expectation().Context().Value(...)`
- **State Machines**: Test multiple state transitions with multi-count overrides
- **Dynamic Expectations**: Use context to configure expected values dynamically
- **Combining Patterns**: Use both multi-count and context passing together

**Use Cases:**
- Testing retry logic with different outcomes per attempt
- State machine testing with multiple transitions
- Passing configuration or expected values to mocks without closure variables
- Testing sequences of function calls with varying behavior
- Complex data pipelines with multiple stages

## Testing Best Practices

1. **Always check expectations**: Call `ExpectationsWereMet()` at the end of tests
3. **Validate arguments**: Use `CheckArgs()` to ensure mocks are called correctly
4. **Keep mocks simple**: Mock functions should be focused and easy to understand
5. **Test one thing at a time**: Each test should verify a single behavior
6. **Use context for data**: Pass test data to mocks via context, not closures

## Common Pitfalls

1. **Forgetting `-gcflags`**: Tests will fail mysteriously without the flags
2. **Inline functions**: Cannot override inlined functions
4. **Generic functions**: Limited support - see [docs/generics.md](../docs/generics.md)
5. **Scope limitations**: Mocks execute in replaced function's scope, use context to pass data
6. **Variadic functions**: Inside mock, variadic params are slices; when checking args, pass as slice; when setting expectations, pass as individual args

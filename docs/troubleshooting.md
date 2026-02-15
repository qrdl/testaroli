# Troubleshooting

Common issues when using testaroli and their solutions.

## Tests Crash with Segmentation Fault

**Symptom:** Tests crash with segfault or similar memory errors when running overrides.

**Common causes:**
1. **Missing compiler flags** - Optimizations can inline functions, making them impossible to override
2. **Platform mismatch** - Testaroli only works on supported OS/architecture combinations

**Solutions:**
- Add `-gcflags="all=-N -l"` flag when running tests:
  ```bash
  go test -gcflags="all=-N -l" ./...
  ```
- For VS Code, add to settings.json:
  ```json
  "go.testFlags": [ "-gcflags", "all=-N -l" ]
  ```
- Verify your OS/arch is supported (see [platform table](README.md#platforms-supported))

## Override Doesn't Work / Original Function Still Called

**Symptom:** Override appears to succeed but the original function is still executed.

**Common causes:**
1. **Function was inlined** - Compiler optimized away the function call
2. **Wrong reference** - Overriding instance method instead of type method

**Solutions:**
- Use `-gcflags="all=-N -l"` to disable inlining (see above)
- For methods, override the type method, not instance:
  ```go
  // ❌ Wrong - instance reference
  s := square{side: 5}
  Override(ctx, s.Area, ...)

  // ✅ Correct - type method
  Override(ctx, square.Area, ...)
  ```

## Panic: "Override() cannot be called for generic function"

**Symptom:** Tests panic with message about generic functions.

**Cause:** Generic functions cannot be overridden directly due to Go's compile-time type instantiation.

**Solution:** Use a function reference instead - see [Generic Functions Guide](generics.md) for detailed workarounds.

## Panic: "Override() cannot be called for interface method"

**Symptom:** Tests panic with message about interface methods.

**Cause:** You're trying to override an interface method instead of a concrete type's method.

**Solution:** Override the concrete type's method that implements the interface - see [Interface Methods Guide](interfaces.md) for detailed patterns.

## Override Works Once Then Stops

**Symptom:** First override works but subsequent calls use original function.

**Cause:** You used `Once` call count, which restores original after one call.

**Solution:** Use appropriate call count:
```go
Override(ctx, fn, Once, mock)      // Single call
Override(ctx, fn, 3, mock)          // Three calls
Override(ctx, fn, Unlimited, mock)  // Until manual reset
Override(ctx, fn, Always, mock)     // Always effective
```

## Expectations Not Met Error

**Symptom:** `ExpectationsWereMet()` returns error about mocks not being called.

**Common causes:**
1. **Mock not called** - Code path didn't execute as expected
2. **Wrong call count** - Used `Once` but function called multiple times
3. **Override order** - Multiple overrides set up in wrong order

**Solutions:**
- Verify code path executes the mocked function
- Check call count matches actual usage:
  ```go
  // If function is called 3 times in your code
  Override(ctx, fn, 3, mock)(arg1, arg2, arg3)
  ```
- Set up overrides in the order they'll be called (chain executes LIFO)

## Mock Called But Expectations Not Met

**Symptom:** Function is clearly being called (you see mock behavior) but `ExpectationsWereMet()` reports it wasn't called, or subsequent overrides in chain never activate.

**Cause:** You forgot to call `Expectation()` inside the mock function.

**Critical:** `Expectation()` MUST be called inside every mock function, even if you don't check arguments. It:
- Increments the call counter
- Enables automatic restoration after specified calls
- Advances the override chain to next mock
- Validates call order

**Solution:**
```go
// ❌ Wrong - no Expectation() call
Override(ctx, fn, Once, func(a int) int {
    return 42  // Missing Expectation()!
})(42)

// ✅ Correct - always call Expectation()
Override(ctx, fn, Once, func(a int) int {
    Expectation()  // Required!
    return 42
})(42)

// ✅ Also correct - with argument checking
Override(ctx, fn, Once, func(a int) int {
    Expectation().CheckArgs(a)  // Expectation() is called before CheckArgs()
    return 42
})(42)
```

**Note:** This applies to ALL count modes including `Always` and `Unlimited`.

## Arguments Don't Match Error

**Symptom:** `CheckArgs()` panics with message about argument mismatch.

**Cause:** Function was called with different arguments than expected.

**Solutions:**
- Verify the expected arguments:
  ```go
  Override(ctx, fn, Once, func(a int, b string) {
      Expectation().CheckArgs(a, b)  // Checks both arguments
      return
  })(42, "expected")  // Expected values here
  ```
- For complex types (structs, slices), ensure deep equality:
  ```go
  expected := MyStruct{Field: "value"}
  Override(ctx, fn, Once, func(s MyStruct) {
      Expectation().CheckArgs(s)
  })(expected)  // Must match exactly
  ```

## Build Fails on Unsupported Platform

**Symptom:** Build errors mentioning build tags or undefined functions.

**Cause:** Testaroli only supports specific OS/architecture combinations.

**Supported platforms:**
- Linux (x86-64, ARM64)
- Windows (x86-64, ARM64)
- macOS (x86-64, ARM64)
- BSD (x86-64, ARM64)

**Solution:** Check that your platform is supported. For ARM64 Windows, use [MSYS CLANGARM64](https://www.msys2.org/docs/arm64/) shell.

## Cannot Override Function from Another Module

**Symptom:** Override compiles but doesn't affect the function from external module.

**Possible causes:**
1. **Function is inlined** - Even with `-N` flag, some standard library functions might be inlined
2. **Wrong function reference** - Importing different version or different function

**Solutions:**
- Verify you're referencing the correct function
- Check if function has `//go:noinline` directive (might prevent override)
- For standard library functions, ensure you're not overriding compiler intrinsics

## Cannot Use Outer Variables in Mock Functions

**Symptom:** Mock function cannot access variables from the outer test scope; panics or unexpected values occur.

**Cause:** The mock implementation is executed in the context of the replaced function, not the test's lexical scope. As a result, it cannot access variables from the outer (test) function.

**Solution:** Pass any required data via the context argument. Store values in the context and retrieve them inside the mock. This ensures the mock has access to needed data.

**Example:**
```go
// In your test:
ctx := context.WithValue(TestingContext(t), "expected", 42)
Override(ctx, fn, Once, func(a int) int {
  expected := Expectation().Context().Value("expected").(int)
  Expectation().CheckArgs(a)
  return expected
})
```

This pattern allows you to pass any value from the test to the mock safely.

## Memory Leak / Unexpected Memory Usage

**Symptom:** Tests consume excessive memory or leak memory.

**Cause:** Overrides not properly reset between tests or `Unlimited` overrides accumulating.

**Solutions:**
- Always use `TestingContext(t)` which automatically resets overrides after test:
  ```go
  Override(TestingContext(t), fn, ...)  // Auto-reset
  ```
- For `Unlimited` overrides, manually reset when done:
  ```go
  reset := Override(ctx, fn, Unlimited, mock)
  defer reset()  // Clean up when done
  ```
- Avoid using `Always` unless necessary (it stays outside the chain)

---

For more information, see:
- [Main Documentation](README.md)
- [Generic Functions](generics.md)
- [Interface Methods](interfaces.md)
- [Go Package Reference](https://pkg.go.dev/github.com/qrdl/testaroli)

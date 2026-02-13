# Overriding Generic Functions

**TL;DR:** Generic functions CAN be overridden, but you must use them via a reference, not direct calls.

## Quick Start

To override a generic function, always use it through a reference:

```go
func pointer[T any](a T) *T {
	return &a
}

func TestFoo(t *testing.T) {
	// ✅ WORKS - Create a reference first
	fn := pointer[int]

	Override(TestingContext(t), fn, Once, func(a int) *int {
		Expectation().CheckArgs(a)
		return nil
	})(1)

	// Call through the reference
	result := fn(1)
	if result != nil {
		t.Errorf("Got unexpected result %v", result)
	}
}
```

## What Works and What Doesn't

The key is **how you call** the function after overriding, not what you pass to Override:

| ✅ Works (call via reference) | ❌ Doesn't Work (call via expression) |
|--------------------------|-------------------------------|
| `fn := genericFunc[int]`<br>`Override(ctx, fn, ...)`<br>`result := fn(42)` | `fn := genericFunc[int]`<br>`Override(ctx, fn, ...)`<br>`result := genericFunc[int](42)` |

**Note:** Both `fn` and `genericFunc[int]` refer to the same trampoline. The Override itself always works. The difference is in the call - `fn(42)` uses the patched trampoline, while `genericFunc[int](42)` or `genericFunc(42)` bypass it.

### Example: Working Pattern

```go
// ✅ This works
func TestWithReference(t *testing.T) {
	pointerInt := pointer[int]  // Create reference
	Override(TestingContext(t), pointerInt, Once, mockFunc)
	result := pointerInt(42)    // Call via reference
}
```

### Example: Non-Working Pattern

```go
// ❌ This doesn't work
func TestDirectCall(t *testing.T) {
	fn := pointer[int]
	Override(TestingContext(t), fn, Once, mockFunc)
	result := pointer[int](42)  // Direct call expression bypasses override!
}
```

```go
// ❌ This also doesn't work
func TestNoReference(t *testing.T) {
	Override(TestingContext(t), pointer[int], Once, mockFunc)
	// Now you have no reference variable to call through!
	// You can only call pointer[int](42) or pointer(42)
	// Both bypass the override
	result := pointer(42)  // Direct call bypasses override!
}
```

## Key Insight

The limitation is in **how the compiler generates code** for different call patterns:

- **Call via stored reference** (`fn(42)`): Compiler generates an indirect call through the function pointer, which uses your patched trampoline ✅
- **Call via expression** (`genericFunc[int](42)`): Compiler generates a direct call to the shaped implementation, bypassing the trampoline ❌

This cannot be detected at runtime because both patterns pass the same trampoline to Override. The difference only appears in the calling code's assembly.

## Why This Limitation Exists

### Technical Background

Go uses a special calling convention for generics that includes a hidden "dictionary" parameter with type information. When you call a generic function directly (like `pointer(42)`), the compiler passes this dictionary automatically.

However, when you take a reference to a generic function (like `fn := pointer[int]`), Go generates a **trampoline function** that converts between the normal calling convention and the generic calling convention.

### How Override Works

When you call `Override(ctx, pointer[int], ...)`:
- Go passes the type-specific **trampoline** to Override
- Override patches the trampoline function
- Calls through the reference use the trampoline ✅
- Direct calls bypass the trampoline entirely ❌

This is why the reference pattern is required.

## Best Practices

1. **Always use references** when testing generic functions:
   ```go
   fn := genericFunc[TypeParam]
   Override(ctx, fn, ...)
   ```

2. **Structure your code** to make testing easier:
   ```go
   // In production code
   type GenericInvoker[T any] func(T) *T

   func DoWork[T any](invoker GenericInvoker[T], val T) {
       result := invoker(val)  // Uses reference
   }

   // In test
   mockInvoker := pointer[int]
   Override(ctx, mockInvoker, Once, mock)
   DoWork(mockInvoker, 42)  // ✅ Works
   ```

3. **Document the requirement** in your test code:
   ```go
   // Override requires a reference to the generic function
   fn := pointer[int]
   Override(ctx, fn, Once, mock)
   ```

## Further Reading

For those interested in the low-level details of Go's generic implementation:
- [Go 1.18 Generics Implementation](https://deepsource.com/blog/go-1-18-generics-implementation)
- Go compiler source: `cmd/compile/internal/gc/reflect.go` (dictionary generation)

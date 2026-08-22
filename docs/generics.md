# Overriding Generic Functions

**TL;DR:** Generic functions can be overridden and called in any form — through a
reference, through an explicit instantiation, or with type inference. You no
longer need to route calls through a stored reference.

## Quick Start

```go
func pointer[T any](a T) *T {
	return &a
}

func TestFoo(t *testing.T) {
	Override(TestingContext(t), pointer[int], Once, func(a int) *int {
		Expectation().CheckArgs(a)
		return nil
	})(1)

	// All of these now use the override:
	result := pointer(1)         // type inference
	_ = pointer[int](1)          // explicit instantiation
	fn := pointer[int]; _ = fn(1) // stored reference

	if result != nil {
		t.Errorf("Got unexpected result %v", result)
	}

	testError(t, nil, ExpectationsWereMet())
}
```

Pass the instantiated function (`pointer[int]`) to `Override` as usual. The
override is effective regardless of how the function is later called.

## Why This Used To Be a Problem

Go compiles a call to a generic function in one of two ways:

- **Through the instantiation trampoline.** When you take a reference
  (`fn := pointer[int]`), the compiler materialises a small per-type *trampoline*
  that adapts the normal calling convention to the generic one and then calls the
  shared implementation.
- **Directly to the shaped implementation.** When you write a call expression
  (`pointer[int](x)` or `pointer(x)`), the compiler emits a direct call to the
  *shaped* implementation, passing a hidden type *dictionary* as an extra leading
  argument. This path never touches the trampoline.

`Override` receives the trampoline (that is what `pointer[int]` evaluates to), so
patching only the trampoline used to leave direct call expressions unpatched —
hence the old "always call through a reference" rule.

## How Both Forms Are Now Intercepted

When `Override` detects a generic function (its runtime name ends in `[...]`), it
patches **both** entry points:

1. **The trampoline** is patched to jump straight to the mock, exactly like a
   regular function. This covers reference-form calls.
2. **The shaped implementation** is located by scanning the trampoline for the
   direct call it makes, and its entry is patched to jump to a tiny **shim**. The
   shim drops the hidden dictionary argument (shifting the remaining argument
   registers back into place) and tail-calls the mock. This covers direct call
   expressions.

The shim is written into the body of the trampoline itself — dead code once the
trampoline entry jumps away — so no extra executable memory is allocated, and
both patches are restored together when the override is reset.

## Limitation: Shape Sharing

Go does not generate a separate machine-code implementation for every type
argument. Instead it groups type arguments into *shapes* (a "gcshape") and shares
one implementation across a shape. In particular:

- **All pointer types share one shape.** `pointer[*Foo]` and `pointer[*Bar]` use
  the same shaped implementation.
- **Named types share the shape of their underlying type.** `pointer[int]` and
  `pointer[MyInt]` (where `type MyInt int`) use the same shaped implementation.

Because the shaped implementation is shared, overriding `pointer[int]` also
intercepts direct calls to `pointer[MyInt]`, and overriding `pointer[*Foo]` also
intercepts direct calls to `pointer[*Bar]`. Your mock will then receive an
argument whose static type differs from what it declared.

This is rarely an issue in practice — a single test overriding a single
instantiation — but keep it in mind when a generic function is called with
several shape-compatible type arguments in the same test.

## Requirements

The usual requirement still applies: run tests with optimizations and inlining
disabled, otherwise direct calls may be inlined and there is no call to patch:

```
go test -gcflags="all=-N -l" ./...
```

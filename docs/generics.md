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
	// count 3: the override stays effective for all three calls below
	Override(TestingContext(t), pointer[int], 3, func(a int) *int {
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

	if err := ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
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

When `Override` detects a generic instantiation (its runtime name contains
`[...]`), it patches **both** entry points:

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

## Methods on Generic Types

Methods on instantiated generic types work the same way — override the method
expression and every call form is intercepted:

```go
type Box[T any] struct{ v T }
func (b *Box[T]) Set(x T) T { b.v = x; return x }

func TestBox(t *testing.T) {
	Override(TestingContext(t), (*Box[int]).Set, Once, func(b *Box[int], x int) int {
		Expectation().CheckArgs(b, x)
		return -x
	})(&Box[int]{}, 3)

	// b.Set(3) — a direct method call — is intercepted
}
```

As with any method override, the receiver becomes the mock's first argument.
Internally the dictionary is passed right after the receiver rather than first,
so `Override` locates it from the receiver's register footprint.

## Limitation: Register-Passed Arguments Only

The shim that intercepts direct calls works by shifting the shaped
implementation's integer argument registers to remove the hidden dictionary. It
can only do this when every argument — *plus* the dictionary, which claims one
extra integer register — is passed in registers. Go's calling convention has a
fixed budget of argument registers (on amd64: 9 integer and 15 floating-point;
on arm64: 16 and 16). A signature is not shim-able when:

- its integer arguments plus the dictionary exceed the integer-register budget
  (for example nine or more integer-word arguments on amd64), or
- an argument is of a type Go passes on the stack (such as a multi-element array,
  including a method receiver of that kind, `type Ring[T any] [8]T`).

For such a signature only the **reference form** is intercepted; a direct call
runs the original function. Nothing is silently miscompiled — the arguments are
never corrupted — but if you rely on intercepting a direct call, its expectation
will simply go unmet. Call the generic through a stored reference in that case.

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

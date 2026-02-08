# Overriding Interface Methods

**TL;DR:** You cannot override interface methods directly, but you CAN override the concrete type's method that implements the interface.

## Quick Start

To override a method that implements an interface, override the **type method**, not the interface method:

```go
type Shape interface {
	Area() float64
}

type square struct {
	side float64
}

func (s square) Area() float64 {
	return s.side * s.side
}

func TestSquareArea(t *testing.T) {
	s := square{side: 5}

	// ✅ CORRECT - Override the type method (square.Area)
	Override(TestingContext(t), square.Area, Once, func(s square) float64 {
		Expectation()
		return 10
	})(s)

	if s.Area() != 10 {
		t.Errorf("Got unexpected result %v", s.Area())
	}
	if err := ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}
```

**Key point:** The method receiver becomes the **first argument** of the mock function.

## What Works and What Doesn't

| ✅ Works | ❌ Doesn't Work |
|----------|-----------------|
| `Override(ctx, square.Area, ...)` | `Override(ctx, Shape.Area, ...)` |
| Override concrete type method | Override interface method |
| Receiver as first argument | Interface as first argument |

### Working Pattern

```go
// ✅ Override the concrete type's method
Override(TestingContext(t), square.Area, Once, func(s square) float64 {
	Expectation()
	return 10
})(s)

result := s.Area()  // Calls mock, returns 10
```

### Non-Working Patterns

```go
// ❌ Cannot override interface method
Override(TestingContext(t), Shape.Area, Once, func(s Shape) float64 {
	return 10
})(square{side: 5})
// Panic: "Override() cannot be called for interface method"

// ❌ Cannot override instance method reference
s := square{side: 5}
Override(TestingContext(t), s.Area, Once, func() float64 {
	return 10
})()
// Overrides trampoline, not the actual method - doesn't work as expected
```

## Common Errors

### Error 1: Interface Method Override

**What you see:**
```
panic: Override() cannot be called for interface method
```

**What it means:**
You tried to override a method defined in an interface (like `Shape.Area`).

**Solution:**
Override the concrete type's method instead:
```go
// ❌ Wrong
Override(ctx, Shape.Area, ...)

// ✅ Correct
Override(ctx, square.Area, ...)
```

### Error 2: Instance Method Reference

**Symptom:**
Override seems to succeed but doesn't actually work - original method is called.

**What happened:**
When you write `s.Area`, Go creates a trampoline function. Override patches the trampoline, but direct calls like `s.Area()` bypass it.

**Solution:**
Always use the type method, not instance method reference:
```go
s := square{side: 5}

// ❌ Wrong - Creates trampoline
Override(ctx, s.Area, Once, func() float64 { return 10 })()

// ✅ Correct - Override type method
Override(ctx, square.Area, Once, func(s square) float64 {
	return 10
})(s)
```

## Multiple Interface Implementations

If multiple types implement the same interface, override each type's method separately:

```go
type circle struct {
	radius float64
}

func (c circle) Area() float64 {
	return math.Pi * c.radius * c.radius
}

func TestMultipleShapes(t *testing.T) {
	// Override square implementation
	Override(TestingContext(t), square.Area, Once, func(s square) float64 {
		Expectation()
		return 100
	})(square{side: 10})

	// Override circle implementation
	Override(TestingContext(t), circle.Area, Once, func(c circle) float64 {
		Expectation()
		return 200
	})(circle{radius: 10})

	sq := square{side: 10}
	if sq.Area() != 100 {
		t.Error("Square override failed")
	}

	circ := circle{radius: 10}
	if circ.Area() != 200 {
		t.Error("Circle override failed")
	}

	if err := ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}
```

## Why This Limitation Exists

### Interface Methods Are Virtual

In Go, interface methods are resolved dynamically at runtime. When you call a method through an interface:

```go
var s Shape = square{side: 5}
s.Area()  // Runtime dispatch to square.Area
```

The runtime looks up which concrete implementation to call. There's no single "interface method" to override - it's just a contract that multiple types fulfill.

### Type Methods Are Concrete

When you override a type method:

```go
Override(ctx, square.Area, ...)
```

You're patching the actual compiled function for `square.Area`. All calls to this method (whether through the interface or directly) now go through your override.

### Detection

Testaroli detects interface method override attempts by checking if the first parameter is an interface type and if the method belongs to that interface. When detected, it panics with a clear error message to guide you to the correct pattern.

## Best Practices

1. **Always override concrete types:**
   ```go
   Override(ctx, ConcreteType.Method, ...)  // ✅
   ```

2. **Remember receiver is first argument:**
   ```go
   func (s square) Area() float64 { ... }

   Override(ctx, square.Area, Once, func(s square) float64 {
       // 's' is the receiver, now a regular parameter
   })
   ```

3. **Use type assertions in tests when needed:**
   ```go
   var shape Shape = square{side: 5}
   if sq, ok := shape.(square); ok {
       // Now you can override square.Area
       Override(ctx, square.Area, Once, mock)(sq)
   }
   ```

4. **Test concrete types, not interfaces:**
   - Write tests for `square.Area()`, `circle.Area()`, etc.
   - Don't try to test "any Shape" - test specific implementations

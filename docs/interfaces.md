# How to override interface methods

Testaroli package doesn't allow to directly override interface methods, however you can override a method of the type that implement the interface.

Assuming there is an interface and a type, implementing the interface:

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
```

The correct way to override `Area()` method would be to override type method (with method receiver as method's first argument):
```go
func TestSquareArea(t *testing.T) {
	s := square{side: 5}

	Override(TestingContext(t), square.Area, Once, func(s square) float64 {
		return 10
	})(s)

	if s.Area() != 10 {
		t.Errorf("Got unexpected result %v", s.Area())
	}
}
```

Overriding instance method directly does not work:
```go
func TestSquareArea(t *testing.T) {
	s := square{side: 5}

	Override(TestingContext(t), s.Area, Once, func() float64 {
		return 10
	})()

	if s.Area() != 10 {
		t.Errorf("Got unexpected result %v", s.Area())
	}
}

```
This is due to the fact that Go creates trampoline functions for method references, and when method is passed as an argument, go implicitly creates such a trampoline, as a result Testaroli overrides the trampoline function, not the actual method. Unfortunately there is no way to detect incorrect use of instance method override.

However it is possible to detect interface method overrides, and the code below would panic:
```go
	Override(TestingContext(t), Shape.Area, Once, func(s Shape) float64 {
		return 10
	})(square{side: 5})
```

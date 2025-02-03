# How to override interface methods

Testaroli allow to override interface methods only for specific object types (but not instances!) that implement the interface.

Assuming there is following interface and type, implementing said interface:

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

The correct way to override `Area()` method would be
```go
func TestSquareArea(t *testing.T) {
	s := square{side: 5}

	Override(TestingContext(t), square.Area, Once, func(s square) float64 {
		return 10
	})(square{side: 5})

	if s.Area() != 10 {
		t.Errorf("Got unexpected result %v", s.Area())
	}
}
```

Overriding instance method with
```go
	Override(TestingContext(t), s.Area, Once, func() float64 {
		return 10
	})()
```
or interface method with
```go
	Override(TestingContext(t), Shape.Area, Once, func(s Shape) float64 {
		return 10
	})(square{side: 5})
```
both don't work. Testaroli doesn't throw errors for the incorrect cases of interface method overrides because Go's `reflect` package doesn't provide a mean the detect such cases.
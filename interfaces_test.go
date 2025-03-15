package testaroli

import "testing"

type Shape interface {
	Area() float64
}

type square struct {
	side float64
}

func (s square) Area() float64 {
	return s.side * s.side
}

func TestOverrideTypeMethod(t *testing.T) {
	s := square{side: 5}

	// override type method (square.Area), not instance method (s.Area) !
	Override(TestingContext(t), square.Area, Once, func(s square) float64 {
		Expectation()
		return 10
	})(s)

	if s.Area() != 10 {
		t.Errorf("Got unexpected result %v", s.Area())
	}
	testError(t, nil, ExpectationsWereMet())
}

func TestOverrideInterfaceMethod(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
		ExpectationsWereMet()
	}()

	Override(TestingContext(t), Shape.Area, Once, func(s Shape) float64 {
		Expectation()
		return 10
	})(square{side: 5})
}

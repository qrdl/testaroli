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

func TestOverrideInstanceMethodWithReference(t *testing.T) {
	s1 := square{side: 5}
	s2 := square{side: 7}

	Override(TestingContext(t), s1.Area, Always, func() float64 {
		return 10
	})()

	// use a reference to call the overridden trampoline function
	method := s1.Area
	if method() != 10 {
		t.Errorf("Got unexpected result")
	}

	// s2 remains unchanged
	if s2.Area() != 49 {
		t.Errorf("Got unexpected result")
	}

	testError(t, nil, ExpectationsWereMet())
}

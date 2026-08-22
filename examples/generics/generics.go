package main

import "fmt"

// Number is a constraint for the numeric types used by the metric helpers.
type Number interface {
	~int | ~int64 | ~float64
}

// average returns the arithmetic mean of the samples, or 0 for empty input.
// Production code calls it with type inference: average(samples...).
func average[T Number](samples ...T) float64 {
	if len(samples) == 0 {
		return 0
	}
	var total float64
	for _, s := range samples {
		total += float64(s)
	}
	return total / float64(len(samples))
}

// maxOf returns the largest of the given values, or the zero value for empty
// input. Production code calls it with an explicit type argument: maxOf[int](...).
func maxOf[T Number](vals ...T) T {
	var top T
	for i, v := range vals {
		if i == 0 || v > top {
			top = v
		}
	}
	return top
}

// deref returns the value a pointer points to, or the zero value of T for nil.
func deref[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}

// Report summarizes a set of latency samples (in milliseconds). It calls the
// generic helpers in two different forms - type inference and explicit
// instantiation - both of which are intercepted when the helpers are overridden.
func Report(samples ...int) string {
	avg := average(samples...)     // type inference: average(...)
	peak := maxOf[int](samples...) // explicit instantiation: maxOf[int](...)
	return fmt.Sprintf("avg=%.1fms peak=%dms", avg, peak)
}

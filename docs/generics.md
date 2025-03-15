# Overriding generics

TL;DR: Generics are not supported and probably never will be.

## What is possible and what is not
Testaroli package provides very limited functionality for overriding generic functions.

Assuming there is a generic function that returns the pointer to its argument:
```go
func pointer[T any](a T) *T {
	return &a
}
```
It is possible to test it when tested code calls the generic function via the reference, like this:
```go
func TestFoo(t *testing.T) {
	Override(TestingContext(t), pointer[int], Once, func(a int) *int {
		Expectation().CheckArgs(a)
		return nil
	})(1)

	pointerInt := pointer[int]
	res := pointerInt(1)
	if res != nil {
		t.Errorf("Got unexpected result %v", res)
	}
}
```
However this is not how the code is typically written, more common approach to call generic function in this scenario would be
```go
	res := pointer(1)
	if res != nil {
		t.Errorf("Got unexpected result %v", res)
	}
```
and it doesn't work.

## Explanation
Go uses special calling convention for generics - it passes extra parameter `dictionary` with type information (there are some articles explaining generic internals, such as [this one](https://deepsource.com/blog/go-1-18-generics-implementation)).
When taking reference of the generic function (which happens every time when code instantiates generic function, apart from direct call), like `pointerInt := pointer[int]` in example above, Go generates trampoline function that converts regular function call for `func (int) int` to generic-specific calling convention.

In a call to `Override` Go passes the type-specific instance (trampoline) function instead of generic function as an argument, so the trampoline function got overridden. As a result mock is only executed when trampoline is called with `pointerInt(...)`, but code `res := pointer(1)` calls the generic function directly, bypassing the trampoline function.
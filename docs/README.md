# testaroli

[![Go Reference](https://pkg.go.dev/badge/github.com/qrdl/testaroli.svg)](https://pkg.go.dev/github.com/qrdl/testaroli)
[![Go Report Card](https://goreportcard.com/badge/github.com/qrdl/testaroli)](https://goreportcard.com/report/github.com/qrdl/testaroli)
![Tests](https://github.com/qrdl/testaroli/actions/workflows/go.yml/badge.svg)
![CodeQL](https://github.com/qrdl/testaroli/workflows/CodeQL/badge.svg)

Package `testaroli` allows to monkey patch Go test binary, e.g. override functions and methods with stubs/mocks to simplify unit testing.
It can be used only for unit testing and never in production.

## Platforms suported

This package modifies actual executable at runtime, therefore is OS- and CPU arch-specific.

OS/arch combinations:

|         | x86_64    | ARM64   |
|---------|-----------|---------|
| Linux   | Supported | Planned |
| Windows | Supported | -       |
| macOS   | Planned   | Planned |


## Command line options

Inlined functions cannot be overridden, so to prevent inlining use `-gcflags=all=-l` CLI option when running tests, like this:

`go test -gcflags=all=-l [<path>]`

Typical use:
```
// you want to test function foo() which in turn calls function bar(), so you
// override function bar() to check whether it is called with correct argument
// and to return preferdined result

func foo() error {
    ...
    if err := bar(baz); err != nil {
        return err
    }
    ...
}

func bar(baz int) error {
    ...
}

func TestBarFailing(t *testing.T) {
    mock := testaroli.New(context.TODO(), t)

    //                 v-- how many runs expected
    testaroli.Override(1, bar, func(a int) error {
        testaroli.Expectation().CheckArgs(a)  // <-- arg value checked here
        return ErrInvalid
    })(42) // <-- expected argument value

    err := foo()
    if !errors.Is(err, ErrInvalid) {
        t.Errorf("unexpected %v", err)
    }
    it err = mock.ExpectationsWereMet(); err != nil {
        t.Error(err)
    }
}
```

See more advanced usage examples in [examples](examples) directory.

# testaroli

[![Go Reference](https://pkg.go.dev/badge/github.com/qrdl/testaroli.svg)](https://pkg.go.dev/github.com/qrdl/testaroli)
[![Go Report Card](https://goreportcard.com/badge/github.com/qrdl/testaroli)](https://goreportcard.com/report/github.com/qrdl/testaroli)

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

Due to live patching of running binary there are certain limitations that may require extra CLI options:
- inlined functions cannot be overridden, so to prevent inlining use `-gcflags=-l` CLI option when running tests
- Instead() function modifies the binary on the fly, therefore running tests in parallel may produce unpredictible results, so better to avoid it using `-p=1` CLI option

Recommended command to run tests:

`go test -gcflags=-l -p=1 [<path>]`

Example:

```
// you want to test function foo() which in turn calls function bar(), so you
// override function bar() to check whether it is called with correct arguments
// and to return preferdined result.

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
    testaroli.Instead(testeroli.Context(t), bar, func(baz int) error {
        if baz != 42 {  // check arg
            testaroli.Testing(testaroli.LookupContext(bar)).Errorf("unexpected arg value %v", a)
        }
        return ErrInvalid  // simulate failure
    })
    defer testaroli.Restore(bar)  // restore original function in order not to break other tests
    
    err := foo()
    if !errors.Is(err, ErrInvalid) {
        t.Errorf("unexpected %v", err)
    }
}
```

See more advanced usage examples in [examples](examples) directory.
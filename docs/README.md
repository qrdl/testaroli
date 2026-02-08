# testaroli

[![Go Reference](https://pkg.go.dev/badge/github.com/qrdl/testaroli.svg)](https://pkg.go.dev/github.com/qrdl/testaroli)
[![Go Report Card](https://goreportcard.com/badge/github.com/qrdl/testaroli)](https://goreportcard.com/report/github.com/qrdl/testaroli)
[![Tests](https://github.com/qrdl/testaroli/actions/workflows/go.yml/badge.svg)](https://github.com/qrdl/testaroli/actions/workflows/go.yml)
[![CodeQL](https://github.com/qrdl/testaroli/workflows/CodeQL/badge.svg)](https://github.com/qrdl/testaroli/actions/workflows/github-code-scanning/codeql)
[![codecov](https://codecov.io/github/qrdl/testaroli/graph/badge.svg?token=V51OL05VQ1)](https://codecov.io/github/qrdl/testaroli)
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fqrdl%2Ftestaroli.svg?type=small)](https://app.fossa.io/projects/git%2Bgithub.com%2Fqrdl%2Ftestaroli?ref=badge_small)

Package `testaroli` allows to [monkey patch](https://en.wikipedia.org/wiki/Monkey_patch) Go test binary, e.g. override functions and methods with stubs/mocks to simplify unit testing.
It can be used only for unit testing and never in production.

## Platforms supported

This package modifies actual executable at runtime, therefore is OS- and CPU arch-specific.

OS/arch combinations:

|         | x86-64 | ARM64  |
|---------|:------:|:------:|
| Linux   | ✅     | ✅     |
| Windows | ✅     | ✅[^1] |
| macOS   | ✅     | ✅     |
| BSD[^2] | ✅     | ✅     |

[^1]: I only could get it working using [MSYS CLANGARM64](https://www.msys2.org/docs/arm64/) shell, see [this issue](https://github.com/qrdl/testaroli/issues/44) for more details.

[^2]: This package was tested on FreeBSD but it should work on other BSD flavours (NetBSD, OpenBSD and DragonFly BSD) as well.

## Command line options

It is recommended to switch off compiler optimisations and disable function inlining using `-gcflags="all=-N -l"` CLI option when running tests, like this:

`go test -gcflags="all=-N -l" ./...`

If you plan to run tests from VS Code, add `"go.testFlags": [ "-gcflags", "all=-N -l" ]` to [settings.json](https://code.visualstudio.com/docs/getstarted/settings#_settings-json-file) file.

Typical use:
```go
import . "github.com/qrdl/testaroli"

// you want to test function foo() which in turn calls function bar(), so you
// override function bar() to check whether it is called with correct argument
// and to return predefined result

func foo() error {
    ...
    if err := bar(42); err != nil {
        return err
    }
    ...
}

func bar(baz int) error {
    ...
}

func TestBarFailing(t *testing.T) {
    Override(TestingContext(t), bar, Once, func(a int) error {
        Expectation().CheckArgs(a)  // <-- arg value checked here
        return ErrInvalid
    })(42) // <-- expected argument value

    err := foo()
    if !errors.Is(err, ErrInvalid) {
        t.Errorf("unexpected %v", err)
    }
    if err = ExpectationsWereMet(); err != nil {
        t.Error(err)
    }
}
```

It is also possible to override functions and methods in other packages, including ones
from standard library, like in example below. Please note that method receiver becomes the
first argument of the mock function.

```go
func TestFoo(t *testing.T) {
    Override(TestingContext(t), (*os.File).Read, Once, func(f *os.File, b []byte) (n int, err error) {
        Expectation()
        copy(b, []byte("foo"))
        return 3, nil
    })

    f, _ := os.Open("test.file")
    defer f.Close()
    buf := make([]byte, 3)
    n, _ := f.Read(buf)
    if n != 3 || string(buf) != "foo" {
        t.Errorf("unexpected file content %s", string(buf))
    }
    if err = ExpectationsWereMet(); err != nil {
        t.Error(err)
    }
}
```
See more advanced usage examples in [examples](../examples) directory. For detailed documentaion see [Go reference](https://pkg.go.dev/github.com/qrdl/testaroli).

## Troubleshooting

Having issues? Check the [Troubleshooting Guide](troubleshooting.md) for solutions to common problems.

## For AI Assistants

This repository includes machine-readable documentation to help AI coding assistants generate testaroli overrides:

- **[SKILLS.md](../SKILLS.md)** - Comprehensive function override generation patterns and step-by-step instructions
- **[AGENTS.md](../AGENTS.md)** - Project architecture, design patterns, and technical implementation details
- **[skill.yaml](../skill.yaml)** - Structured metadata for skill discovery and AI integration

AI assistants can reference these files to:
- Generate proper testaroli override code following established patterns
- Understand the library architecture and capabilities
- Suggest appropriate override strategies for different testing scenarios
- Validate argument expectations and track mock calls correctly

Developers can also use SKILLS.md as a quick reference guide with copy-paste patterns for common testing scenarios.

## Limitations

Generic functions cannot be overridden, see [details](generics.md).

There are certain rules how to override interface methods, [details](interfaces.md).

# testaroli

[![Go Reference](https://pkg.go.dev/badge/github.com/qrdl/testaroli.svg)](https://pkg.go.dev/github.com/qrdl/testaroli)
[![Go Report Card](https://goreportcard.com/badge/github.com/qrdl/testaroli)](https://goreportcard.com/report/github.com/qrdl/testaroli)
[![Tests](https://github.com/qrdl/testaroli/actions/workflows/go.yml/badge.svg)](https://github.com/qrdl/testaroli/actions/workflows/go.yml)
[![CodeQL](https://github.com/qrdl/testaroli/workflows/CodeQL/badge.svg)](https://github.com/qrdl/testaroli/actions/workflows/github-code-scanning/codeql)
[![codecov](https://codecov.io/github/qrdl/testaroli/graph/badge.svg?token=V51OL05VQ1)](https://codecov.io/github/qrdl/testaroli)
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fqrdl%2Ftestaroli.svg?type=small)](https://app.fossa.io/projects/git%2Bgithub.com%2Fqrdl%2Ftestaroli?ref=badge_small)

Package `testaroli` provides runtime function and method overriding for Go unit tests through binary patching. This enables testing of error paths and exceptional conditions without requiring dependency injection or interface wrappers.

This is a testing utility only. It modifies executable code at runtime and must not be used in production.

## Use Cases

`Testaroli` addresses several testing challenges in Go:

- Testing error handling when external dependencies fail (network, filesystem, database)
- Exercising exceptional code paths that are hard to trigger naturally
- Testing functions that call standard library or third-party code directly
- Isolating units under test from their dependencies without modifying production code
- Testing legacy codebases where adding dependency injection would require extensive refactoring
- Reducing test fragility - changes to internal implementations (like SQL queries or API calls) don't require updating mocks in every affected test

As a practical demonstration, `testaroli`'s own test suite uses itself to test exceptional code paths, achieving 99%+ test coverage.

## How It Works

`Testaroli` operates by:
1. Saving the original function prologue (first bytes of machine code)
2. Overwriting the function entry point with a jump to the mock implementation
3. Temporarily modifying memory page protections to allow code modification
4. Restoring the original prologue after the specified number of calls

This approach requires platform-specific implementations for memory protection APIs (mprotect on Unix, VirtualProtect on Windows) and CPU architecture-specific instruction handling (x86-64, ARM64).

Note that macOS requires additional complexity due to security restrictions. See [macOS.md](macOS.md) for implementation details.

## Platforms supported

`Testaroli` modifies the actual executable at runtime, therefore is OS and CPU architecture-specific.

OS/arch combinations:

|         | x86-64 | ARM64  |
|---------|:------:|:------:|
| Linux   | ✅     | ✅     |
| Windows | ✅     | ✅[^1] |
| macOS   | ✅     | ✅     |
| BSD[^2] | ✅     | ✅     |

[^1]: I could only get it working using [MSYS CLANGARM64](https://www.msys2.org/docs/arm64/) shell, see [this issue](https://github.com/qrdl/testaroli/issues/44) for more details.

[^2]: This package was tested on FreeBSD but it should work on other BSD flavours (NetBSD, OpenBSD and DragonFly BSD) as well.

## Command line options

It is recommended to switch off compiler optimisations and disable function inlining using `-gcflags="all=-N -l"` CLI option when running tests, like this:

`go test -gcflags="all=-N -l" ./...`

If you plan to run tests from VS Code, add `"go.testFlags": [ "-gcflags", "all=-N -l" ]` to [settings.json](https://code.visualstudio.com/docs/getstarted/settings#_settings-json-file) file.

Typical use:
```go
import . "github.com/qrdl/testaroli"

// Test function foo() which calls bar() internally.
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
    // Override bar() to verify it receives the correct
    // argument and returns a predefined result.
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

It is also possible to override functions and methods in other packages, including ones from standard library, as in the example below. Note that method receiver becomes the
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
See more advanced usage examples in [examples](../examples) directory. For detailed documentation see [Go reference](https://pkg.go.dev/github.com/qrdl/testaroli).

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

### Generic Functions

Override generic functions by using function references instead of direct calls. The [Generic Functions Guide](generics.md) shows complete patterns and examples for working with generics.

### Interface Methods

Interface methods must be overridden at the concrete type level, not the interface level. Override the type method that implements the interface instead. See the [Interface Methods Guide](interfaces.md) for patterns and examples.

### Other Constraints

- **Testing only** - Never use testaroli in production code
- **Compiler flags required** - Must use `-gcflags="all=-N -l"` to disable optimizations and inlining
- **Platform-specific** - Only works on supported OS/architecture combinations (see table above)


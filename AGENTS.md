# AGENTS.md - Testaroli Project Analysis

## Project Overview

**Name:** Testaroli
**Repository:** github.com/qrdl/testaroli
**Language:** Go (1.24.0+)
**License:** Apache License 2.0
**Purpose:** Monkey patching library for Go unit testing

Testaroli is a specialized Go package that enables runtime binary modification to override functions and methods with stubs/mocks during unit testing. It achieves this through low-level memory manipulation and executable patching.

## Core Functionality

### Primary Capabilities
- **Function Override:** Replace any function (including standard library functions) with mock implementations
- **Method Override:** Replace methods with mocks (method receiver becomes first argument)
- **Call Chain Management:** Maintain ordered override chains with call counting
- **Argument Verification:** Check that overridden functions receive expected arguments
- **Cross-Package Mocking:** Override functions from any package, including standard library

### Key Mechanisms
1. **Runtime Memory Patching:** Modifies executable code at runtime
2. **Override Chains:** Sequential override management with counters (Once, Unlimited, Always)
3. **Expectation Tracking:** Validates that mocks are called in expected order with correct arguments
4. **Automatic Restoration:** Resets overridden functions to original state after specified calls

## Architecture

### Core Components

#### 1. Override System (`override.go`)
- Main entry point for function/method overriding
- Manages override chain with different call count modes:
  - **Once** One time
  - **Fixed count (N > 0):** Any positive integer number of calls before restoration (e.g., 1, 2, 3, ...)
  - **Unlimited:** Active until manual reset
  - **Always:** Always effective, outside the chain
- Platform-specific implementations for memory protection

#### 2. Expectation System (`expect.go`)
- `Expect` struct tracks override metadata and call counts
- `Expectation()` function validates mock calls
- `CheckArgs()` method verifies function arguments
- `ExpectationsWereMet()` validates all expectations satisfied

#### 3. Equality Engine (`equal.go`)
- Custom equality comparison for argument validation
- Handles complex types that `reflect.Value.Equal` doesn't:
  - Pointers (compares values, not just addresses)
  - Maps and slices
  - Nested structures
- Provides detailed error messages for failures

#### 4. Platform-Specific Memory Management
- **Unix/Linux:** `mem_unix.go` - mprotect-based memory protection
- **Darwin/macOS:** `mem_darwin.go`, `mem_darwin.h` - Special handling for code signing
- **Windows:** `mem_windows.go` - VirtualProtect-based protection
- **Architecture-specific:** `override_amd64.go`, `override_arm64.go` - CPU instruction handling

## Platform Support

### Operating Systems
- ✅ Linux (x86-64, ARM64)
- ✅ Windows (x86-64, ARM64 via MSYS CLANGARM64)
- ✅ macOS (x86-64, ARM64)
- ✅ BSD (FreeBSD tested, others should work: NetBSD, OpenBSD, DragonFly BSD)

### Architecture Requirements
- x86-64 (amd64)
- ARM64 (aarch64)

Build tags ensure code only compiles on supported platforms: `//go:build (unix || windows) && (amd64 || arm64)`

## Technical Implementation Details

### Memory Manipulation Strategy
1. **Function Prologue Backup:** Save original function's first bytes
2. **Jump Injection:** Overwrite function start with jump to mock
3. **Memory Protection:** Temporarily make code memory writable
4. **Restoration:** Replace jump with original prologue when done

### Testing Requirements
- **Disable Optimizations:** Use `-gcflags="all=-N -l"` flag
- **Disable Inlining:** Prevents compiler from eliminating target functions
- VS Code integration: Add `"go.testFlags": [ "-gcflags", "all=-N -l" ]` to settings.json

## Limitations

### Generics
- Limited support for generic functions
- Works only when generic function is called via reference
- Direct generic function calls cannot be overridden
- See [docs/generics.md](docs/generics.md) for details

### Other Constraints
- **Testing Only:** Never use in production code
- **Platform Dependent:** OS/architecture-specific implementation
- **Compiler Settings:** Requires specific compiler flags
- **Inline Functions:** Cannot override inlined functions

## Project Structure

```
testaroli/
├── Core Library Files
│   ├── override.go          # Main override logic
│   ├── expect.go            # Expectation tracking and validation
│   ├── equal.go             # Custom equality comparison
│   └── interfaces_test.go   # Interface validation tests
│
├── Platform-Specific
│   ├── mem_unix.go          # Unix/Linux memory management
│   ├── mem_darwin.go        # macOS memory management
│   ├── mem_darwin.h         # macOS C headers
│   ├── mem_windows.go       # Windows memory management
│   ├── override_amd64.go    # x86-64 specific code
│   └── override_arm64.go    # ARM64 specific code
│
├── Tests
│   ├── equal_test.go        # Equality tests
│   ├── override_test.go     # Override functionality tests
│   ├── reset_test.go        # Reset functionality tests
│   └── mem_unix_test.go     # Unix memory tests
│
├── Documentation
│   ├── docs/README.md       # Main documentation
│   ├── docs/generics.md     # Generic functions limitations
│   ├── docs/interfaces.md   # Interface handling
│   └── docs/macOS.md        # macOS-specific notes
│
├── Examples
│   ├── examples/functions/  # Function override examples
│   └── examples/methods/    # Method override examples
│
└── Build/Release
    └── go.mod               # Module definition
```

## Usage Patterns

### Basic Function Override
```go
Override(TestingContext(t), targetFunc, Once, func(args...) returnType {
    Expectation().CheckArgs(expectedArgs...)
    return mockResult
})(expectedArgs...)
```

### Method Override
```go
Override(TestingContext(t), (*Type).Method, Once,
    func(receiver *Type, args...) returnType {
        Expectation()
        return mockResult
    })
```

### Standard Library Override
```go
Override(TestingContext(t), (*os.File).Read, Once,
    func(f *os.File, b []byte) (int, error) {
        Expectation()
        copy(b, []byte("mock data"))
        return len("mock data"), nil
    })
```

## Dependencies

- **golang.org/x/sys v0.40.0** - System-level operations (memory protection, syscalls)
- **testing** (stdlib) - Integration with Go testing framework
- **reflect** (stdlib) - Runtime type inspection and manipulation
- **runtime** (stdlib) - Access to runtime internals
- **unsafe** (stdlib) - Low-level memory operations

## Development Workflow

### Testing
```bash
go test -gcflags="all=-N -l" ./...  # Run tests with optimizations disabled
```

### Testing Strategy
1. Unit tests for each component
2. Platform-specific test files
3. Integration tests in examples directory
4. Extensive test coverage (badge in README)

### Quality Assurance
- Go Report Card integration
- CodeQL security scanning
- Automated testing workflow (GitHub Actions)
- FOSSA license compliance

## Key Insights for AI Agents

### When to Use This Project
- Unit testing Go code with hard-to-test dependencies
- Mocking functions from external packages or standard library
- Testing error paths that are difficult to trigger naturally
- Controlling execution order of dependent function calls

### When NOT to Use
- Production code (testing only!)
- With heavily inlined code
- Generic functions (limited support)
- On unsupported platforms
- When proper dependency injection is possible


### Mock Implementation Scope: Passing Data

**Important:** Mock functions are executed in the scope of the replaced function, not the test's lexical scope. This means you cannot access variables from the outer (test) function directly inside the mock.

**Best Practice:** To pass data from your test to the mock, use the context argument. Store values in the context and retrieve them inside the mock using `Expectation().Context()`.

**Example:**
```go
ctx := context.WithValue(TestingContext(t), "expected", 42)
Override(ctx, fn, Once, func(a int) int {
    expected := Expectation().Context().Value("expected").(int)
    Expectation().CheckArgs(a)
    return expected
})
```

This ensures your mock has access to any required data, even though it cannot see outer variables.

### AI Agent Considerations
1. **Memory Safety:** This code manipulates raw memory - suggest careful review
2. **Platform Awareness:** Recommend checking platform before suggesting usage
3. **Compiler Flags:** Always mention required `-gcflags` flags
4. **Testing Context:** Emphasize this is testing infrastructure only
5. **Alternative Solutions:** Consider suggesting dependency injection as alternative

## Retracted Versions
Versions v0.1.0 through v0.3.2 are retracted - avoid using these versions.

## Community & Support
- GitHub: https://github.com/qrdl/testaroli
- Go Package Reference: https://pkg.go.dev/github.com/qrdl/testaroli
- Issues: Track platform-specific issues on GitHub
- Examples: See `examples/` directory for practical usage patterns

---

*Generated: 2026-01-31*
*Analysis Type: Project Structure, Functionality, and AI Agent Guidance*

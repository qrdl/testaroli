# Patching binary in macOS on Apple silicon M series CPUs

Testaroli works by patching test binary in memory at runtime, which works without issues on Linux and Windows, however macOS implements additional security features, such as [Hardened Runtime](https://developer.apple.com/documentation/security/hardened-runtime), that makes the whole concept of monkey patching much more difficult, although not impossible.

## Problem 1
macOS kernel applies stricter permission control on memory segments. In addition to standard (for Linux world) effective permissions it has max allowed permissions, so effective permissions never can go beyond max permissions. In practice it means that `TEXT` segment has both effective and max permissions set to `r-x` so `w` permissions cannot be set, as a result `TEXT` segment cannot be made writable.

#### Solution
If `TEXT` segment cannot be modified, it has to be re-created with different permissions. To keep all addresses to point to correct functions and data, the segment must be created at the same address, which means that old segment has to be removed first, but it cannot be removed because it contains the code that is being executed, i.e. `PC` (program counter) points to instructions inside the segment. To overcome this issue Testaroli at startup creates a copy of `TEXT` segment (let's call it `TEMP` segment) and calls function in `TEMP` segment (so now `PC` points to `TEMP`, not `TEXT`) that removes original `TEXT` segment and re-creates it at the same address, but with `rwx` max permissions. When this function returns (back to `TEXT` segment), the program continues as nothing happened, but now `TEXT` segment can be made writable.

See `make_text_writable()` in [mem_darwin.go](../mem_darwin.go) for the implementation.

## Problem 2
Although new `TEXT` segment is created with `rwx` max permissions, macOS has some extra checks and it doesn't allow to execute instructions from writable segment, so `TEXT` must have `r-x` effective permissions while it is executed. It means Testaroli cannot modify `TEXT` from within `TEXT` itself.

#### Solution
Because Testaroli has already created a copy of `TEXT` segment at startup (remember `TEMP`?), it just switches the execution to `TEMP` and modifies `TEXT` from there.

See `overwrite_prolog()` in [mem_darwin.go](../mem_darwin.go) for the implementation.

## Problem 3
However, there is one issue with this approach - because Go is a garbage-collected language, Go's runtime has a separate goroutine which wakes up regularly to check if there is enough garbage to collect. If it happens at the moment when `TEXT` segment doesn't exist, it obviously results in segfault.

#### Solution
The solution to this was to force garbage collector to run in a separate thread from main using [runtime.LockOSThread()](https://pkg.go.dev/runtime#LockOSThread), and then temporarily suspend all process threads, except main, while segment is being recreated.

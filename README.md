# gitconfig for Go

[![GoDoc](https://godoc.org/github.com/gopasspw/gitconfig?status.svg)](http://godoc.org/github.com/gopasspw/gitconfig)
[![Go Report Card](https://goreportcard.com/badge/github.com/gopasspw/gitconfig)](https://goreportcard.com/report/github.com/gopasspw/gitconfig)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A pure Go library for reading and writing Git configuration files without depending on the `git` CLI. This library is particularly useful when:

- Git might not be installed or available
- You need consistent config parsing across platforms
- You want to avoid breaking changes when Git's behavior changes
- You need fine-grained control over config scopes and precedence

Originally developed to support [gopass](https://github.com/gopasspw/gopass), this library aims for full Git compatibility while maintaining a simple, clean API.

## Features

- ✅ **Multi-scope support** - System, global, local, worktree, and environment configs
- ✅ **Include directives** - Basic `[include]` and `[includeIf]` support (gitdir conditions)
- ✅ **Round-trip preservation** - Maintains comments, whitespace, and formatting when writing
- ✅ **Cross-platform** - Works on Linux, macOS, Windows, and other Unix systems
- ✅ **Customizable** - Override config paths and environment prefixes for your application
- ✅ **Pure Go** - No CGo dependencies, easy cross-compilation
- ✅ **Well-tested** - Comprehensive test coverage including edge cases

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/gopasspw/gitconfig"
)

func main() {
    // Load all config scopes (system, global, local)
    cfg := gitconfig.New()
    if err := cfg.LoadAll("."); err != nil {
        panic(err)
    }

    // Read a configuration value
    name, ok := cfg.Get("user.name")
    if ok {
        fmt.Printf("User name: %s\n", name)
    }

    // Write a configuration value (to local scope by default)
    cfg.Set("user.email", "example@example.com")
    if err := cfg.Write(); err != nil {
        panic(err)
    }
}
```

See the [examples/](examples/) directory for more detailed usage patterns.

## Reference Documentation

The reference for this implementation is the [Git config documentation](https://mirrors.edge.kernel.org/pub/software/scm/git/docs/git-config.html).

## Installation

```bash
go get github.com/gopasspw/gitconfig
```

## Usage

### Loading Configuration

Use `gitconfig.LoadAll` with an optional workspace argument to process configuration from these locations in order (later ones take precedence):

1. **System** - `/etc/gitconfig`
2. **Global** - `$XDG_CONFIG_HOME/git/config` or `~/.gitconfig`
3. **Local** - `<workdir>/.git/config`
4. **Worktree** - `<workdir>/.git/config.worktree`
5. **Environment** - `GIT_CONFIG_{COUNT,KEY,VALUE}` environment variables

```go
cfg := gitconfig.New()
if err := cfg.LoadAll("/path/to/repo"); err != nil {
    log.Fatal(err)
}

// Read from unified config (respects precedence)
value, ok := cfg.Get("core.editor")
```

### Reading Values

```go
// Get single value (returns last matching value)
editor, ok := cfg.Get("core.editor")
if !ok {
    editor = "vi" // default
}

// Get all values for a key (for multi-valued config)
remotes, ok := cfg.GetAll("remote.origin.fetch")

// Read from specific scope
email := cfg.GetGlobal("user.email")
autocrlf := cfg.GetLocal("core.autocrlf")
```

### Writing Values

```go
// Write to default scope (local)
cfg.Set("user.name", "Jane Doe")
cfg.Set("user.email", "jane@example.com")

// Write to specific scope
cfg.SetGlobal("core.editor", "vim")
cfg.SetSystem("core.autocrlf", "false")

// Save changes
if err := cfg.Write(); err != nil {
    log.Fatal(err)
}
```

### Scope-Specific Operations

```go
// Load only a specific config file
localCfg, err := gitconfig.LoadConfig("/path/to/repo/.git/config")
if err != nil {
    log.Fatal(err)
}

// Work with single scope
localCfg.Set("branch.main.remote", "origin")
localCfg.Write()
```

## Customization for Your Application

Applications like `gopass` can easily customize file paths and environment variable prefixes:

```go
cfg := gitconfig.New()

// Customize config file locations
cfg.SystemConfig = "/etc/gopass/config"
cfg.GlobalConfig = "~/.config/gopass/config"
cfg.LocalConfig = ".gopass-config"
cfg.WorktreeConfig = ""  // Disable worktree config

// Customize environment variable prefix
cfg.EnvPrefix = "GOPASS_CONFIG"  // Uses GOPASS_CONFIG_COUNT, etc.

// For testing: prevent accidentally overwriting real configs
cfg.NoWrites = true

// Load with custom settings
cfg.LoadAll(".")
```

### Customization Options

- **SystemConfig** - Path to system-wide configuration file
- **GlobalConfig** - Path to user's global configuration (set to `""` to disable)
- **LocalConfig** - Filename for repository-local config
- **WorktreeConfig** - Filename for worktree-specific config (or `""` to disable)
- **EnvPrefix** - Prefix for environment variables (e.g., `MYAPP_CONFIG`)
- **NoWrites** - Set to true to prevent Write() from modifying files (useful for testing)

## Advanced Features

### Include Directives

The library supports Git's `[include]` and `[includeIf]` directives:

```ini
# .git/config
[include]
    path = /path/to/common.gitconfig

[includeIf "gitdir:/path/to/work/"]
    path = ~/.gitconfig-work
```

**Supported `includeIf` conditions:**

- `gitdir:` - Include if git directory matches pattern
- `gitdir/i:` - Case-insensitive gitdir match

**Current limitations:**

- Other conditional types (onbranch, hasconfig) are not yet supported
- Relative paths in includes are resolved from the config file's directory

### Subsections

Access subsections using dot notation:

```go
// Set remote URL
cfg.Set("remote.origin.url", "https://github.com/user/repo.git")

// Set branch tracking
cfg.Set("branch.main.remote", "origin")
cfg.Set("branch.main.merge", "refs/heads/main")

// Access subsections
url, _ := cfg.Get("remote.origin.url")
```

### Multi-valued Keys

Some config keys can have multiple values:

```go
// Add multiple fetch refspecs
cfg.Set("remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
cfg.Set("remote.origin.fetch", "+refs/pull/*/head:refs/remotes/origin/pr/*")

// Retrieve all values
fetchSpecs, ok := cfg.GetAll("remote.origin.fetch")
// fetchSpecs = ["+refs/heads/*:refs/remotes/origin/*", "+refs/pull/*/head:refs/remotes/origin/pr/*"]
```

## Known Limitations

Current implementation has the following known limitations:

- **Bare boolean values** - Keys without values (bare booleans) are not supported
- **Worktree support** - Only partial worktree config support
- **includeIf conditions** - Only `gitdir` and `gitdir/i` are supported
- **URL matching** - `url.<base>.insteadOf` patterns are not yet implemented
- **Multivar operations** - No special handling for replacing specific multivar instances
- **Whitespace preservation** - Insignificant whitespace is not always perfectly preserved

These limitations reflect the primary use case supporting [gopass](https://github.com/gopasspw/gopass). Contributions to address these are welcome!

## Project Documentation

- **[ARCHITECTURE.md](ARCHITECTURE.md)** - Design decisions and internal architecture
- **[CONTRIBUTING.md](CONTRIBUTING.md)** - Development guidelines and how to contribute
- **[CHANGELOG.md](CHANGELOG.md)** - Version history and release notes
- **[examples/](examples/)** - Runnable code examples demonstrating various features

## Versioning and Compatibility

This library aims to support the latest stable release of Git. We currently do not provide semantic versioning guarantees but aim to maintain backwards compatibility where possible.

**Compatibility goals:**

- Parse any valid Git config file correctly
- Preserve config file structure when writing
- Handle edge cases gracefully (malformed input, missing files, etc.)

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for:

- Development setup instructions
- Code style guidelines
- Testing requirements
- Pull request process

### Quick Development Setup

```bash
# Clone the repository
git clone https://github.com/gopasspw/gitconfig.git
cd gitconfig

# Run tests
make test

# Run code quality checks
make codequality

# Format code
make fmt
```

## Testing

Run the full test suite:

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run with coverage
go test -cover ./...

# Generate a coverage report
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | tail -1

# Current coverage (2026-02-17): 89.9%

# Run specific test
go test -run TestLoadConfig
```

## License and Credit

This package is licensed under the [MIT License](https://opensource.org/licenses/MIT).

This repository is maintained to support the needs of [gopass](https://github.com/gopasspw/gopass), a password manager for the command line. We aim to make it universally useful for all Go projects that need Git config parsing.

**Maintainers:**

- Primary development and maintenance for gopass integration

**Contributing:**
Contributions are welcome! Please read our [contributing guidelines](CONTRIBUTING.md) before submitting pull requests.

## Support

- **Issues**: [GitHub Issues](https://github.com/gopasspw/gitconfig/issues)
- **Documentation**: [GoDoc](https://godoc.org/github.com/gopasspw/gitconfig)
- **Examples**: [examples/](examples/) directory in this repository

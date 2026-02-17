# Development Guide

This guide covers the development workflow, architecture decisions, and implementation details for contributors working on the gitconfig library.

## Table of Contents

- [Getting Started](#getting-started)
- [Project Architecture](#project-architecture)
- [Development Workflow](#development-workflow)
- [Code Organization](#code-organization)
- [Testing Strategy](#testing-strategy)
- [Common Development Tasks](#common-development-tasks)
- [Debugging Tips](#debugging-tips)
- [Performance Considerations](#performance-considerations)
- [Release Process](#release-process)

## Getting Started

### Prerequisites

- **Go**: 1.24 or later
- **golangci-lint**: For code quality checks
- **make**: For build automation
- **git**: For version control

### Initial Setup

```bash
# Clone the repository
git clone https://github.com/gopasspw/gitconfig.git
cd gitconfig

# Run tests to verify setup
make test

# Run code quality checks
make codequality

# Format code
make fmt
```

The project should build and all tests should pass on a fresh clone.

## Project Architecture

### Core Components

The library is organized into three main components:

#### 1. Config (Single Scope)

**File**: `config.go`

Handles a single configuration file with format preservation:

```go
type Config struct {
    path string              // File path
    parsed map[string]string // Parsed key-value pairs
    raw []string            // Original file lines
}
```

**Key design decision**: Maintains both `parsed` (for quick lookups) and `raw` (for format preservation) representations. When writing, the library updates the raw representation to preserve comments, whitespace, and structure.

**Methods**:

- `LoadConfig(path)` - Load from file
- `Get(key)` / `GetAll(key)` - Read values
- `Set(key, value)` - Write values
- `Write()` - Persist changes

#### 2. Configs (Multi-Scope)

**File**: `configs.go`

Manages multiple configuration scopes with precedence:

```go
type Configs struct {
    env      *Config  // Environment variables (highest precedence)
    worktree *Config  // Worktree-specific config
    local    *Config  // Repository config
    global   *Config  // User config
    system   *Config  // System-wide config
    preset   *Config  // Built-in defaults (lowest precedence)
}
```

**Precedence order**: env > worktree > local > global > system > preset

**Methods**:

- `LoadAll(workdir)` - Load all scopes
- `Get(key)` - Read from combined config (respects precedence)
- `GetLocal(key)`, `GetGlobal(key)`, etc. - Scope-specific reads
- `Set(key, value)` - Write to local scope (default)
- `SetLocal()`, `SetGlobal()`, etc. - Scope-specific writes

#### 3. Utilities

**File**: `utils.go`

Helper functions for parsing and formatting:

- `parseKey(key)` - Extract section, subsection, key from dotted notation
- `trim(lines)` - Clean whitespace
- `unescapeValue(val)` - Handle escape sequences
- `globMatch(pattern, path)` - Pattern matching for includeIf

### Data Flow

```text
┌─────────────────────────────────────────────────────┐
│  User API Call (Get/Set)                            │
└──────────────────┬──────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────┐
│  Configs (Multi-scope coordinator)                  │
│  - Checks each scope in precedence order            │
│  - Returns first match for reads                    │
│  - Routes writes to appropriate scope               │
└──────────────────┬──────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────┐
│  Config (Single file handler)                       │
│  - Parses file into map + raw lines                 │
│  - Handles includes recursively                     │
│  - Updates both map and raw on changes              │
└──────────────────┬──────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────┐
│  File I/O                                           │
│  - Read: os.ReadFile                                │
│  - Write: os.WriteFile with preservation            │
└─────────────────────────────────────────────────────┘
```

## Development Workflow

### Making Changes

1. **Create a feature branch**

   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes**
   - Follow existing code style
   - Add tests for new functionality
   - Update documentation

3. **Run quality checks**

   ```bash
   make fmt          # Format code
   make test         # Run tests
   make codequality  # Lint checks
   ```

4. **Commit with conventional commits**

   ```bash
   git commit -m "feat: add support for X"
   git commit -m "fix: handle edge case Y"
   git commit -m "docs: update readme for Z"
   ```

5. **Push and create PR**

   ```bash
   git push origin feature/your-feature-name
   ```

### Commit Message Format

Follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `test:` - Test additions or changes
- `refactor:` - Code restructuring
- `style:` - Formatting, whitespace
- `chore:` - Build, dependencies

## Code Organization

### File Structure

```text
gitconfig/
├── config.go              # Single config file handling
├── configs.go             # Multi-scope coordination
├── utils.go               # Helper functions
├── gitconfig.go           # Default settings for git
├── gitconfig_windows.go   # Windows-specific code
├── gitconfig_others.go    # Unix-specific code
├── doc.go                 # Package documentation
├── config_test.go         # Config tests
├── configs_test.go        # Configs tests
├── utils_test.go          # Utility tests
├── config_errors_test.go  # Error handling tests
├── config_parse_errors_test.go  # Parse error tests
├── config_include_errors_test.go # Include tests
├── config_edge_cases_test.go     # Edge case tests
├── examples/              # Usage examples
├── ARCHITECTURE.md        # Design documentation
├── CONTRIBUTING.md        # Contribution guidelines
├── CONFIG_FORMAT.md       # Format reference
└── DEVELOPMENT.md         # This file
```

### Naming Conventions

- **Exported functions**: Start with capital letter, use PascalCase
  - `LoadConfig()`, `Get()`, `SetGlobal()`

- **Unexported functions**: Start with lowercase, use camelCase
  - `parseKey()`, `getEffectiveIncludes()`

- **Test functions**: `Test<FunctionName>`
  - `TestLoadConfig()`, `TestGet()`

- **Types**: PascalCase
  - `Config`, `Configs`

- **Variables**: camelCase (short names acceptable for local variables)
  - `cfg`, `key`, `value`

### Code Style Guidelines

1. **Function length**: Keep functions focused and under ~50 lines
2. **Comments**:
   - All exported functions need godoc comments
   - Complex logic needs inline comments
   - Use complete sentences with periods
3. **Error handling**: Always check and handle errors explicitly
4. **Testing**: Use table-driven tests with `t.Run()` and `t.Parallel()`

## Testing Strategy

### Test Organization

Tests are organized by component and purpose:

- **`config_test.go`**: Basic Config functionality
- **`configs_test.go`**: Multi-scope behavior and precedence
- **`utils_test.go`**: Utility function tests
- **`config_errors_test.go`**: File system and general error handling
- **`config_parse_errors_test.go`**: Malformed syntax handling
- **`config_include_errors_test.go`**: Include directive edge cases
- **`config_edge_cases_test.go`**: Unusual but valid configurations

### Writing Tests

Use table-driven tests:

```go
func TestFunctionName(t *testing.T) {
    t.Parallel()
    
    testCases := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {
            name:     "basic case",
            input:    "value",
            expected: "value",
            wantErr:  false,
        },
        // ... more cases
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()
            
            result, err := FunctionName(tc.input)
            
            if tc.wantErr {
                require.Error(t, err)
            } else {
                require.NoError(t, err)
                assert.Equal(t, tc.expected, result)
            }
        })
    }
}
```

### Testing Best Practices

1. **Use temp directories**: Always use `t.TempDir()` for file operations
2. **Parallel tests**: Add `t.Parallel()` unless tests modify global state
3. **Assertions**: Use `testify/assert` and `testify/require`
   - `require`: For critical checks (test cannot continue if fails)
   - `assert`: For non-critical checks
4. **Coverage**: Aim for >80% coverage, especially for new code
5. **Error paths**: Test both success and failure scenarios

### Running Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific test
go test -run TestLoadConfig

# Run with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Current coverage (2026-02-17): 89.9%
```

## Common Development Tasks

### Adding a New Configuration Method

1. **Determine scope** (Config or Configs)
2. **Add method**:

   ```go
   // GetBool retrieves a boolean value from config.
   func (c *Configs) GetBool(key string) (bool, bool) {
       val, ok := c.Get(key)
       if !ok {
           return false, false
       }
       // Parse boolean
       return parseBool(val), true
   }
   ```

3. **Add tests**:

   ```go
   func TestGetBool(t *testing.T) {
       t.Parallel()
       // Test cases...
   }
   ```

4. **Update documentation** in `doc.go` if user-facing

### Adding Support for a New Section Type

1. **Understand Git behavior** (test with actual git)
2. **Update parser** in `config.go` if syntax changes needed
3. **Add test cases** with real-world examples
4. **Document** in `CONFIG_FORMAT.md`

### Handling a New Include Type

1. **Check Git documentation** for semantics
2. **Add condition parser** in `getConditionalIncludes()`
3. **Add tests** in `config_include_errors_test.go`
4. **Update** `CONFIG_FORMAT.md` with examples

### Cross-Platform Considerations

Platform-specific code goes in:

- `gitconfig_windows.go` - Windows (`//go:build windows`)
- `gitconfig_others.go` - Unix/Linux (`//go:build !windows`)

Example:

```go
// gitconfig_windows.go
//go:build windows

package gitconfig

func getDefaultSystemConfig() string {
    return "C:\\ProgramData\\Git\\config"
}
```

## Debugging Tips

### Debugging Tests

```bash
# Run single test with verbose output
go test -v -run TestSpecificTest

# Add debug prints in tests
t.Logf("Config state: %+v", cfg)

# Use delve debugger
dlv test -- -test.run TestSpecificTest
```

### Debugging File Parsing

Add temporary debug output:

```go
func (c *Config) Load() error {
    // ... parsing logic
    
    // Debug: print parsed state
    for k, v := range c.parsed {
        fmt.Printf("DEBUG: %s = %s\n", k, v)
    }
    
    // ... rest of function
}
```

### Common Issues

**Issue**: Tests fail with permission errors

- **Solution**: Ensure using `t.TempDir()` for test files
- **Solution**: Check file permissions in test setup

**Issue**: Include tests fail inconsistently

- **Solution**: Check for absolute vs relative path handling
- **Solution**: Verify include depth limits

**Issue**: Write operations don't preserve format

- **Solution**: Check that `raw` slice is being updated
- **Solution**: Verify line matching logic in write operations

## Performance Considerations

### Parsing Performance

- **Current**: File is fully parsed on load
- **Optimization**: Consider lazy loading for large configs
- **Benchmark**: Add benchmarks before optimizing

### Memory Usage

- **Trade-off**: We keep both `parsed` map and `raw` lines
- **Benefit**: Enables format preservation
- **Cost**: ~2x memory of parsed-only approach
- **Acceptable**: Config files are typically small (<100KB)

### Include Performance

- **Current**: Includes are loaded recursively
- **Watch for**: Circular includes (we have depth limits)
- **Future**: Could cache included files

### Benchmarking

Create benchmarks for performance-critical code:

```go
func BenchmarkParseKey(b *testing.B) {
    for i := 0; i < b.N; i++ {
        parseKey("section.subsection.key")
    }
}
```

Run benchmarks:

```bash
go test -bench=. -benchmem
```

## Release Process

### Version Numbering

The project does use semantic versioning.

### Release Checklist

1. **Update CHANGELOG.md**

   - Add new version section
   - List all changes since last release
   - Categorize: Added, Changed, Deprecated, Removed, Fixed, Security

2. **Run full test suite**

   ```bash
   make test
   make codequality
   ```

3. **Test cross-compilation**

   ```bash
   make crosscompile
   ```

4. **Update documentation**

   - Ensure README is current
   - Check godoc examples
   - Verify all links work

5. **Tag release**

   ```bash
   git tag -a v0.x.y -m "Release v0.x.y"
   git push origin v0.x.y
   ```

6. **Announce**
   - Update dependent projects (gopass)

## Additional Resources

- **Architecture**: See [ARCHITECTURE.md](ARCHITECTURE.md) for design decisions
- **Contributing**: See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines
- **Format**: See [CONFIG_FORMAT.md](CONFIG_FORMAT.md) for Git config format details
- **Examples**: See [examples/](examples/) for usage examples
- **Git Docs**: <https://git-scm.com/docs/git-config>

## Getting Help

- **Issues**: Open an issue on GitHub for bugs or questions
- **Discussions**: Use GitHub Discussions for general questions
- **Gopass**: This library primarily supports gopass; consult gopass maintainers for integration questions

## Maintainer Notes

### Code Review Checklist

When reviewing PRs:

- [ ] Tests added for new functionality
- [ ] Tests pass and coverage doesn't decrease
- [ ] Code follows existing style
- [ ] Godoc added for exported functions
- [ ] CHANGELOG updated if user-facing change
- [ ] Cross-platform considerations addressed
- [ ] No breaking changes (or clearly documented)

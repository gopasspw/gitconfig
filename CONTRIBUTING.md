# Contributing to gitconfig

Thank you for your interest in contributing to gitconfig! This document provides guidelines and instructions for contributing.

## Code of Conduct

Please be respectful and constructive in all interactions within this project.

## Getting Started

### Prerequisites

- Go 1.22 or later
- Git
- Make

### Development Setup

1. Clone the repository:

```bash
git clone https://github.com/gopasspw/gitconfig.git
cd gitconfig
```

1. Install dependencies:

```bash
go mod tidy
```

1. Verify your setup:

```bash
make test
make codequality
```

All tests should pass and no linting errors should be reported.

## Making Changes

### Branch Strategy

Create a feature branch for your work:

```bash
git checkout -b feature/my-feature
# or
git checkout -b fix/my-fix
```

Use descriptive branch names that indicate the type of change.

### Code Style

1. **Format your code:**

   ```bash
   make fmt
   ```

   This runs:
   - `keep-sorted` for import organization
   - `gofumpt` for aggressive Go formatting
   - `go mod tidy`

2. **Follow Go conventions:**
   - Use clear, descriptive names
   - Document exported functions with godoc comments
   - Keep functions focused and testable
   - Common abbreviations: cfg, err, ok, v, vs (values)

3. **Linting:**

   ```bash
   make codequality
   ```

   All linting errors must be resolved before submitting a pull request.

### Testing

#### Running Tests

```bash
# Run all tests
make test

# Run specific test
go test -v -run TestName

# Run with race detection
go test -race ./...
```

#### Writing Tests

1. **Test location:** Add tests to `*_test.go` files next to source code
2. **Test naming:** Use `TestFunctionName` or `TestFunctionName_Scenario`
3. **Test structure:** Use table-driven tests where practical
4. **Parallel tests:** Add `t.Parallel()` to tests that don't rely on shared state
5. **Assertions:** Use `testify/assert` and `testify/require`

Example test:

```go
func TestMyFeature(t *testing.T) {
 t.Parallel()

 testCases := []struct {
  name    string
  input   string
  want    string
  wantErr bool
 }{
  {
   name:    "simple case",
   input:   "test",
   want:    "result",
   wantErr: false,
  },
 }

 for _, tc := range testCases {
  t.Run(tc.name, func(t *testing.T) {
   got, err := MyFunction(tc.input)
   if tc.wantErr {
    assert.Error(t, err)
   } else {
    assert.NoError(t, err)
    assert.Equal(t, tc.want, got)
   }
  })
 }
}
```

#### Test Coverage

When adding new functionality:

- Aim for >80% code coverage
- Test both the success path and error cases
- Include edge case tests
- Document why certain edge cases are important

### Commit Messages

Use conventional commit format:

```
type(scope): short description

Longer explanation if needed. Wrap at 72 characters.

Additional context and motivation.

Fixes #123
```

Common types:

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation change
- `test`: Test addition or modification
- `refactor`: Code refactoring
- `perf`: Performance improvement
- `chore`: Build, dependencies, or tooling

Example:

```text
feat(parser): add support for bare boolean values

Add parsing support for bare boolean values in gitconfig files
as per git-config specification. Previously these were silently
ignored.

Fixes #42
```

## Submitting Changes

### Before Submitting

1. Ensure all tests pass:

   ```bash
   make test
   ```

2. Run code quality checks:

   ```bash
   make codequality
   make fmt
   ```

3. Verify cross-compilation works (if changing platform-specific code):

   ```bash
   make crosscompile
   ```

4. Update documentation if you changed:
   - Public API (update godoc comments)
   - User-facing behavior (update README.md, doc.go)
   - Configuration handling (update CONFIG_FORMAT.md)

### Pull Request Process

1. Push your branch to your fork
2. Create a pull request with a clear title and description
3. Link related issues using `Closes #123` or `Fixes #456`
4. Include any relevant documentation changes
5. Be ready to respond to code review comments

### What to Include in PR Description

```markdown
## Description
Clear description of what changes and why.

## Type of Change
- [ ] Bug fix (non-breaking)
- [ ] New feature (non-breaking)
- [ ] Breaking change

## How to Test
Steps to verify the changes work.

## Checklist
- [ ] Tests pass locally
- [ ] Code formatted with `make fmt`
- [ ] Linting passes with `make codequality`
- [ ] Documentation updated
- [ ] No new dependencies added (or justified)
```

## Code Review

Expect constructive feedback on:

- Code clarity and maintainability
- Test coverage
- Documentation completeness
- API design consistency
- Performance implications

## Common Patterns

### Adding a New Function

1. Implement the function
2. Add godoc comment with:
   - What it does
   - Parameters and return values
   - Examples
   - Any error conditions
3. Add tests covering normal and error cases
4. Run `make fmt` and `make codequality`

### Modifying Existing Functions

1. Check if behavior change is breaking
2. Update godoc if behavior changed
3. Add/update tests
4. Update related documentation

### Adding Dependencies

Before adding a new dependency:

1. Check if stdlib or existing deps can solve the problem
2. Verify license compatibility (must be MIT or compatible)
3. Ensure it's pure Go (no CGo)
4. Document why it's needed in DEPENDENCIES.md
5. Open an issue to discuss before implementing

## Getting Help

- **Questions:** Open a discussion or issue
- **Bug reports:** Include reproduction steps, environment, Go version
- **Feature requests:** Describe the use case and motivation
- **Design discussions:** Open an issue to discuss before implementing

## Additional Resources

- [git-config documentation](https://mirrors.edge.kernel.org/pub/software/scm/git/docs/git-config.html)
- [ARCHITECTURE.md](./ARCHITECTURE.md) - Design and structure
- [CONFIG_FORMAT.md](./CONFIG_FORMAT.md) - Supported syntax
- [DEVELOPMENT.md](./DEVELOPMENT.md) - Deeper technical details

## License

By contributing, you agree that your contributions will be licensed under the MIT License, consistent with the project's license.

## Recognition

Contributors are valued and important to this project. Your contributions help make gitconfig better for everyone.

Thank you for contributing! 🎉

# Dependencies

This document describes the external dependencies used by the gitconfig project.

## Direct Dependencies

### Production Dependencies

#### gobwas/glob (v0.2.3)

- **Purpose:** Glob pattern matching for conditional include resolution
- **Used for:** Matching `onbranch:*` patterns in includeIf conditions
- **Why needed:** Provides efficient glob matching with `**` support
- **License:** MIT
- **Note:** Minimal dependency; could be replaced with stdlib if glob features not needed

#### gopasspw/gopass (v1.16.1)

- **Purpose:** Provides utility functions and package infrastructure
- **Used for:** Debug logging, applicaton directory detection, set utilities
- **Why needed:** Used by parent project; provides common utilities
- **License:** MIT
- **Future:** Consider reducing this dependency in future versions

### Test Dependencies

#### stretchr/testify (v1.11.1)

- **Purpose:** Assertion and mocking library for tests
- **Used for:** `assert` and `require` functions in test files
- **Why needed:** Provides cleaner, more expressive test assertions
- **License:** MIT

## Indirect Dependencies

All indirect dependencies are test-related infrastructure:

- **blang/semver:** Semantic version parsing (from testify)
- **davecgh/go-spew:** Pretty-printing for debugging (from testify)
- **kr/pretty:** Pretty-printing utilities (from testify)
- **pmezard/go-difflib:** Diff generation for assertions (from testify)
- **golang.org/x/exp:** Experimental standard library features
- **gopkg.in/yaml.v3:** YAML parsing (from testify)

## No CGo Dependency

⚠️ **Important:** This project explicitly does NOT use CGo. All dependencies are pure Go, which enables:

- Cross-platform compilation (Windows, macOS, Linux)
- No C/C++ compiler required
- Static binary generation
- Simplified deployment

## Dependency Rationale

### Why not use standard library only?

1. **glob package:** Go's filepath.Match is too limited. We need:
   - Double-asterisk (`**`) support for path matching
   - Proper path component handling
   - Character classes and ranges

2. **gopass utilities:** Existing integration with gopass parent project requires these utilities

### Future Optimization Opportunities

- **Consider:** Reducing gopass dependency or making it optional
- **Consider:** Implementing lightweight glob matching if performance is critical
- **Note:** Keep testify as test-only dependency; it's well-maintained and improves test clarity

## Licensing

All dependencies use licenses compatible with gitconfig's MIT license:

- gobwas/glob: MIT
- gopasspw/gopass: MIT  
- stretchr/testify: MIT (Apache 2.0 compatible)

## Updating Dependencies

To update dependencies:

```bash
# Check for available updates
go list -u -m all

# Update a specific dependency
go get -u github.com/package/name

# Update all dependencies
go get -u ./...

# Tidy the go.mod file
go mod tidy
```

After updating dependencies, always:

1. Run tests: `make test`
2. Run linting: `make codequality`
3. Verify cross-compilation: `make crosscompile`

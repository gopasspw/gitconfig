# Dependencies

This document describes the external dependencies used by the gitconfig project.

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

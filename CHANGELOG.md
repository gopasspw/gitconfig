# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

### Changed

### Fixed

## [0.0.4] - 2026-02-17

### Added

- Improved documentation with CONTRIBUTING.md guide
- Example programs in examples/ directory
- Comprehensive API documentation in doc.go
- ARCHITECTURE.md explaining design decisions
- Additional test coverage for error handling
- Better test coverage for edge cases
- Support for onbranch conditional includes
- Support for gitdir/i (case-insensitive) conditionals

### Changed

- Enhanced error messages
- Improved parsing logic for edge cases
- Better handling of escape sequences

### Fixed

- Improved stability in include file resolution
- Better validation of key formats

## [0.1.0] - 2024-01-01

### Added

- Initial release with core gitconfig parsing
- Support for multiple config scopes (system, global, local, worktree, env)
- Config file mutation while preserving comments and whitespace
- Support for include and conditional include
- Environment variable override support (GIT_CONFIG_*)
- Cross-platform support (Windows, macOS, Linux)
- Comprehensive test suite

### Features

- Parse git configuration files without git CLI dependency
- Read/write config values with scope management
- Support for subsections and special characters in keys
- Conditional includes based on gitdir and onbranch patterns
- Value unescaping for standard escape sequences (\n, \t, \b, \\, \")
- Config merging from multiple sources

### Known Limitations

- Worktree support is only partial
- Bare boolean values not supported
- includeIf support only includes gitdir and onbranch conditions
- Does not support all git-config features (urlmatch, etc.)

[Unreleased]: https://github.com/gopasspw/gitconfig/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/gopasspw/gitconfig/releases/tag/v0.1.0

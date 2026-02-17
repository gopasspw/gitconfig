# Architecture

This document describes the architecture and design of the gitconfig library.

## Overview

gitconfig is a Go library for parsing and manipulating git configuration files without depending on the git CLI tool. The library maintains the structure of the original config file (including comments and whitespace) while allowing programmatic access and modification.

## Core Concepts

### Configuration Scopes

Git config has a hierarchical scope system. Each scope corresponds to a different level of configuration:

```
Priority (highest to lowest):
  Environment Variables (GIT_CONFIG_*)
    ↓
  Per-Worktree Config (.git/config.worktree)
    ↓
  Local/Repository Config (.git/config)
    ↓
  Global/User Config (~/.gitconfig or ~/.config/git/config)
    ↓
  System-wide Config (/etc/gitconfig)
    ↓
  Presets (built-in defaults)
```

When a key is requested, the library searches through scopes in priority order and returns the first match found. This allows settings at higher-priority scopes to override lower ones.

### Key Structure

Git config keys follow a hierarchical structure:

```
section.key                 → Simple value
section.subsection.key      → Value in a subsection
array.values[0]             → Array element (internally represented)
```

Keys are normalized according to git rules:
- Section names: case-insensitive, typically lowercase
- Subsection names: case-sensitive
- Key names: case-insensitive, typically lowercase

### Configuration Format

Git config files follow an INI-like format:

```ini
[section]
    key = value
    
[section "subsection"]
    key = value
    
; Comments
# Another comment

[section]
    multivalue = first
    multivalue = second
```

Special considerations:
- Subsections in quotes preserve case
- Multiple values for same key are supported
- Comments and whitespace are preserved during modifications
- Boolean values can be implicit (presence indicates true)

## Architecture Components

### 1. Config Structure

**Purpose:** Represents a single configuration file (one scope)

**File:** `config.go`

**Key Responsibilities:**
- Parse a single config file
- Maintain both parsed representation (`vars` map) and raw text representation  
- Support reading (Get, GetAll, IsSet)
- Support writing (Set, Unset)
- Preserve formatting during modifications

**Design Pattern: Round-Trip Preservation**

The Config struct maintains two parallel representations:

1. **Parsed representation** (`vars map[string][]string`)
   - Fast lookups: O(1)
   - Ordered values for multi-value keys
   - Structure: `section.subsection.key` → `[]string`

2. **Raw text representation** (strings.Builder)
   - Original file content
   - Preserves comments and whitespace
   - Modified in-place on write operations

When reading, the library uses the parsed `vars` map. When writing, it modifies the raw text representation to maintain the original file structure.

```go
type Config struct {
    vars raw string // Original file content
    // Internal parsed structure
}
```

**Write Algorithm:**
1. Find existing key location in raw text
2. Update value in-place if exists, append if new
3. Reconstruct raw text preserving all other content
4. Flush to disk

**Complexity Analysis:**
- Get: O(1)
- Set: O(n) where n = file size (due to raw text rewriting)
- Unset: O(n)

### 2. Configs Structure

**Purpose:** Unified interface for all configuration scopes

**File:** `configs.go`

**Key Responsibilities:**
- Load and manage multiple Config objects (one per scope)
- Implement scope precedence/priority
- Provide unified Get/Set/Unset interface
- Route writes to specific scopes
- Handle scope-aware operations

**Design Pattern: Scope Delegation**

Configs acts as a facade over multiple Config objects:

```go
type Configs struct {
    env Config      // Environment variables
    worktree Config // Worktree-specific
    local Config    // Repository-specific
    global Config   // User-specific
    system Config   // System-wide
    preset Config   // Built-in defaults
}
```

**Hierarchy Implementation:**

When calling `Get(key)`:
1. Check environment variables
2. Check worktree config
3. Check local config
4. Check global config
5. Check system config
6. Check presets
7. Return first match or error

This is implemented as a simple linear search with early termination. Optimization considerations were examined but deemed unnecessary given typical config module sizes (< 10KB).

**Write API:**
- `SetLocal()` → writes to local scope
- `SetGlobal()` → writes to global scope
- `SetWorktree()` → writes to worktree scope
- `Set()` → writes to local scope (default)

This prevents silent surprises where Set() might write to unexpected scope based on environment state.

### 3. Utility Functions

**Purpose:** Common parsing and matching operations

**File:** `utils.go`

**Key Functions:**

| Function | Purpose | Implementation |
|----------|---------|-----------------|
| `splitKey()` | Parse key into section, subsection, key | String splitting logic |
| `canonicalizeKey()` | Normalize key per git rules | Case normalization |
| `globMatch()` | Pattern matching for includes | gobwas/glob wrapper |
| `parseLineForComment()` | Handle quoted strings in values | State machine parser |
| `trim()` | Whitespace handling | Standard library wrapper |

### 4. Platform-Specific Code

**Purpose:** Handle differences between operating systems

**Files:** 
- `gitconfig.go` - Common functions
- `gitconfig_windows.go` - Windows-specific paths
- `gitconfig_others.go` - Unix/Linux/macOS paths

**Key Differences:**
- Home directory detection (environment variables)
- Path separators (\ vs /)
- Config file locations (Windows vs Unix conventions)
- Permission handling

## Design Decisions

### 1. No External Dependencies (Except gobwas/glob)

**Decision:** Minimize external dependencies

**Reasoning:**
- Improves portability and cross-compilation
- Reduces build complexity
- Avoids dependency version conflicts
- Easier to maintain long-term

**Exception - gobwas/glob:**
- Required for include conditional pattern matching
- Minimal pure-Go library
- No dependencies of its own

### 2. Round-Trip Preservation

**Decision:** Maintain original file formatting during modifications

**Reasoning:**
- Preserves user formatting intentions
- Retains comments which may contain important notes
- Matches git behavior of preserving structure
- Enables collaborative workflows where formatting matters

**Trade-off:**
- File write is O(n) instead of O(1) (n = file size)
- Acceptable because: config files are small (< 10KB typical), write frequency is low

### 3. Scope Separation in API

**Decision:** Separate local/global/worktree in public API

**Reasoning:**
- Makes scope explicit (no hidden behavior)
- Prevents bugs where wrong scope is written
- Self-documenting code (SetLocal() clearly means local)
- Aligns with git subcommands (git config --local vs --global)

**Trade-off:**
- Slightly more verbose API
- Benefit: correctness and clarity outweigh verbosity

### 4. Single String per Get()

**Decision:** Get() returns single string, GetAll() for multiple values

**Reasoning:**
- Simpler common case (most keys have one value)
- Clear distinction between single and multi-value patterns
- Prevents accidental data loss

**Trade-off:**
- Extra method for multi-value keys
- Benefit: prevents silent data truncation bugs

### 5. Parsed Map with String Keys

**Decision:** Store parsed config as `map[string][]string`

**Reasoning:**
- Fast lookups: O(1)
- Simple structure
- Easy to reason about
- Compatible with standard library patterns

**Trade-off:**
- Loses structural hierarchy (flat namespace)
- Benefit: simplicity and performance

## Thread Safety

**Current:** The library is NOT thread-safe by default.

**Reason:** 
- Config file format is relatively simple
- Most applications load config once at startup
- Proper synchronization is application responsibility
- Adds complexity for uncommon use case

**Recommendation for concurrent use:**
- Use sync.RWMutex to protect Config/Configs objects
- Serialize writes to prevent corruption
- Example:

```go
type ThreadSafeConfig struct {
    mu  sync.RWMutex
    cfg *gitconfig.Config
}

func (tc *ThreadSafeConfig) Get(key string) (string, bool) {
    tc.mu.RLock()
    defer tc.mu.RUnlock()
    return tc.cfg.Get(key)
}
```

## Performance Characteristics

### Time Complexity

| Operation | Complexity | Notes |
|-----------|-------------|-------|
| Get(key) | O(1) | Map lookup |
| GetAll(key) | O(1) | Map lookup |
| Set(key) | O(n) | n = file size (rewrite required) |
| Unset(key) | O(n) | n = file size |
| IsSet(key) | O(1) | Map lookup |
| LoadAll() | O(m) | m = number of config files |

### Space Complexity

- **Parsed representation:** O(k) where k = number of keys
- **Raw representation:** O(f) where f = file size
- **Total:** O(k + f)

For typical git configs: < 10KB, so negligible impact.

### Real-world Performance

On typical systems:
- Loading a config: < 1ms
- Reading a value: < 0.1ms 
- Writing a value: 1-5ms
- Loading all scopes: 5-10ms

Performance is not a bottleneck for config operations since they typically happen at application startup.

## Include File Handling

gitconfig supports the `[include]` directive:

```ini
[include]
    path = /path/to/common.conf
```

**Implementation:**
1. Parser detects include directives
2. Recursively loads included files
3. Values from includes are merged into the same Config object
4. Later values override earlier ones (path order matters)

**Use Cases:**
- Base configurations (DRY principle)
- Environment-specific overrides
- Team shared settings
- Machine-specific secrets (with .gitignore)

## Conditional Includes

Git also supports conditional includes (gitconfig 2.13+):

```ini
[includeIf "gitdir:~/work/"]
    path = ~/.gitconfig-work
```

**Implementation:**
- Uses glob pattern matching (globMatch)
- Conditional evaluated at load time
- Only matching includes are processed

## Future Extensibility

### Potential Enhancements

1. **Streaming large files**
   - Current: Load entire file into memory
   - Future: Stream mode for very large files
   - Would reduce memory usage but complicate API

2. **Type system**
   - Current: All values are strings
   - Future: Optional type coercion (string, bool, int)
   - Would improve usability but add API complexity

3. **Schema validation**
   - Current: No validation of keys/values
   - Future: Optional schema to validate allowed keys
   - Would catch errors earlier but may be too opinionated

4. **Watch mode**
   - Current: No detection of external changes
   - Future: File watcher for external modifications
   - Would require async APIs

### Design Stability

The core API is stable and unlikely to change significantly because:
- Strongly tied to git config semantics
- Already covers primary use cases
- Simple, minimal API reduces change surface area

## Testing Strategy

### Test Organization

```
config_test.go          → Config struct tests
configs_test.go         → Configs struct tests
utils_test.go           → Utility function tests
gitconfig_test.go       → Integration tests
```

### Test Coverage

Target coverage: > 80%

Test categories:
1. **Happy path:** Normal operations
2. **Error cases:** Missing files, parsing errors
3. **Edge cases:** Empty configs, special characters, multi-values
4. **Integration:** Multiple scopes, includes, real-world scenarios

## Summary

The gitconfig library implements a minimal, focused approach to git configuration:

- **Single responsibility:** Parse and manipulate git config files
- **Preservation:** Maintains formatting and comments
- **Simplicity:** Minimal API, no hidden behavior
- **Compatibility:** Follows git semantics closely
- **Performance:** Adequate for typical use cases (startup-time config loading)

The design prioritizes correctness, clarity, and compatibility with git over raw performance optimization.

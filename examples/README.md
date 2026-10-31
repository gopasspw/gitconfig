# gitconfig Examples

This directory contains practical examples demonstrating how to use the gitconfig library.

## Examples

### 1. [Basic Read](01-basic-read.go)

Demonstrates reading configuration values from git config using the simple Config API.

**Topics:**

- Loading a config file
- Reading string values
- Handling missing keys

**Run:**

```bash
go run -tags=examples examples/01-basic-read.go
```

### 2. [Write and Persist](02-write-persist.go)

Shows how to modify configuration values and persist changes back to disk.

**Topics:**

- Setting configuration values
- Persistence to file
- Verifying changes on disk

**Run:**

```bash
go run -tags=examples examples/02-write-persist.go
```

### 3. [Understanding Scopes](03-scopes.go)

Demonstrates the configuration scope hierarchy and how gitconfig resolves values across scopes.

**Topics:**

- System-wide config
- User config
- Local repository config
- Environment variables
- Scope priority/precedence

**Run:**

```bash
go run -tags=examples examples/03-scopes.go
```

### 4. [Custom Paths](04-custom-paths.go)

Shows how to work with custom configuration file paths instead of default Git locations.

**Topics:**

- Custom config paths
- Non-standard locations
- Loading from arbitrary files

**Run:**

```bash
go run -tags=examples examples/04-custom-paths.go
```

### 5. [Error Handling](05-error-handling.go)

Demonstrates proper error handling patterns when working with gitconfig.

**Topics:**

- Parse errors
- File not found
- Permission errors
- Invalid key formats

**Run:**

```bash
go run -tags=examples examples/05-error-handling.go
```

### 6. [Include Files](06-includes.go)

Shows how gitconfig handles include directives for modular configuration.

**Topics:**

- Including other config files
- Conditional includes
- External config organization

**Run:**

```bash
go run -tags=examples examples/06-includes.go
```

### 7. [Iterating Keys](07-iterate-keys.go)

Shows how to enumerate configuration keys across all scopes or within a single scope.

**Topics:**

- Listing all keys with `Configs.Keys()`
- Prefix filtering with `Configs.List()`
- Accessing a single scope with `Configs.Scope()`
- Iterating keys and values with `Config.Keys()` and `Config.Entries()`
- Copy semantics of `Config.Entries()`

**Run:**

```bash
go run -tags=examples examples/07-iterate-keys.go
```

### 8. [Reordering Sections and Keys](08-sort.go)

Shows how to reorder the sections and keys of a config file.

**Topics:**

- Grouping sections with the same name via `SortOptions.GroupSections`
- Sorting sections alphabetically via `SortOptions.SortSections`
- Sorting keys within a section via `SortOptions.SortKeys`
- Preserving comments and whitespace while reordering

**Run:**

```bash
go run -tags=examples examples/08-sort.go
```

## Prerequisites

Before running these examples, ensure you have Go 1.24 or later installed:

```bash
go version
```

## How to Use These Examples

1. Each example is a standalone Go file
2. Run with `go run -tags=examples examples/<number>-<name>.go`
3. Some examples create temporary files for demonstration
4. Review the source code to understand each pattern
5. Modify and experiment to learn more

## Learning Path

Recommended order for learning:

1. Start with **01-basic-read** to understand basic usage
2. Move to **02-write-persist** to learn mutation
3. Learn scope hierarchy with **03-scopes**
4. Explore flexibility with **04-custom-paths**
5. Master error handling with **05-error-handling**
6. Build modular configs with **06-includes**
7. Enumerate keys with **07-iterate-keys**

## Common Patterns

### Reading a Single Value

```go
cfg, _ := gitconfig.NewConfig("path/to/.git/config")
value, ok := cfg.Get("user.name")
if ok {
    fmt.Println("Name:", value)
}
```

### Setting and Saving

```go
cfg, _ := gitconfig.NewConfig("path/to/.git/config")
cfg.Set("core.editor", "vim")
_ = cfg.String()  // Persist changes
```

### Working with All Scopes

```go
cfg, _ := gitconfig.NewConfigs()  // Load all scopes
value := cfg.Get("user.name")      // Respects scope priority
cfg.SetLocal("core.pager", "less") // Write to local only
```

## More Information

- [README](../README.md) - Overview and quick start
- [ARCHITECTURE](../ARCHITECTURE.md) - Design and structure
- [CONTRIBUTING](../CONTRIBUTING.md) - How to contribute
- [godoc](../doc.go) - API documentation

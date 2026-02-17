# Git Configuration File Format

This document describes the Git configuration file format as implemented by this library, based on the [official Git documentation](https://git-scm.com/docs/git-config#_configuration_file).

## File Structure

A Git config file consists of sections and variables:

```ini
# Comment
[section]
    key = value

[section "subsection"]
    key = value
```

## Syntax Rules

### Sections

Sections are defined using square brackets:

```ini
[core]
[remote "origin"]
[branch "main"]
```

- **Simple section**: `[section]`
- **Subsection**: `[section "subsection"]`
- Section names are case-insensitive
- Subsection names are case-sensitive

### Keys

Keys appear within sections:

```ini
[user]
    name = John Doe
    email = john@example.com
```

- Key names are case-insensitive
- Keys can contain alphanumeric characters and dashes
- Whitespace before `=` is ignored
- Keys can appear multiple times (multivars)

### Values

Values follow the `=` sign:

```ini
[core]
    editor = vim
    autocrlf = true
    excludesfile = ~/.gitignore_global
```

**Value types:**

- **String**: Any text (quotes optional unless special characters present)
- **Boolean**: `true`, `false`, `yes`, `no`, `on`, `off`, `1`, `0`
- **Integer**: Numeric value (parsed as string by this library)

**Special characters in values:**

```ini
[section]
    # Quoted string with spaces
    key = "value with spaces"
    
    # Escaped characters
    path = "C:\\Users\\Name\\Path"
    
    # Empty value
    emptykey =
```

### Escape Sequences

Within double-quoted strings:

| Sequence | Meaning |
|----------|---------|
| `\\` | Backslash |
| `\"` | Double quote |
| `\n` | Newline |
| `\t` | Tab |
| `\b` | Backspace |

Example:

```ini
[alias]
    log1 = "log --pretty=format:\"%h %s\""
```

### Comments

Comments start with `#` or `;`:

```ini
# This is a comment
; This is also a comment

[section]
    key = value  # Inline comments are NOT standard Git behavior
```

**Note**: This library may not handle inline comments correctly. Use full-line comments.

### Whitespace

- Leading and trailing whitespace in values is trimmed
- Internal whitespace in unquoted values is preserved
- Use quotes to preserve leading/trailing whitespace

```ini
[section]
    # These are equivalent:
    key1 = value
    key1=value
    key1  =  value
    
    # These preserve whitespace:
    key2 = "  value with spaces  "
```

## Multi-valued Keys

Some keys can have multiple values:

```ini
[remote "origin"]
    fetch = +refs/heads/*:refs/remotes/origin/*
    fetch = +refs/pull/*/head:refs/remotes/origin/pr/*
```

Access with:

```go
values, ok := cfg.GetAll("remote.origin.fetch")
// values = ["+refs/heads/*...", "+refs/pull/*..."]
```

## Include Directives

### Basic Includes

Include other config files:

```ini
[include]
    path = /path/to/other.gitconfig
    path = ~/.gitconfig-extras
```

**Path resolution:**

- Relative paths are resolved from the directory of the current config file
- `~` expands to user home directory
- Absolute paths work as expected

### Conditional Includes

Include files based on conditions:

```ini
[includeIf "gitdir:~/work/"]
    path = ~/.gitconfig-work

[includeIf "gitdir/i:C:/projects/"]
    path = ~/gitconfig-windows
```

**Supported conditions:**

- `gitdir:<pattern>` - Include if git directory matches pattern (case-sensitive)
- `gitdir/i:<pattern>` - Include if git directory matches pattern (case-insensitive)

**Current limitations:**

- `onbranch:<pattern>` - Not supported
- `hasconfig:remote.*.url:<pattern>` - Not supported

### Include Precedence

Later includes override earlier ones:

```ini
[user]
    email = personal@example.com

[include]
    path = ~/.gitconfig-work  # May override user.email
```

Settings in included files follow normal override rules.

## Key Naming Conventions

### Section Hierarchy

Keys use dot notation:

```
section.key
section.subsection.key
```

Examples:

```ini
[core]
    editor = vim
# Accessed as: core.editor

[remote "origin"]
    url = https://github.com/user/repo.git
# Accessed as: remote.origin.url

[branch "main"]
    remote = origin
# Accessed as: branch.main.remote
```

### Case Sensitivity

- **Section names**: Case-insensitive (`[Core]` = `[core]`)
- **Subsection names**: Case-sensitive (`[remote "Origin"]` ≠ `[remote "origin"]`)
- **Key names**: Case-insensitive (`Name` = `name`)

## Common Patterns

### User Information

```ini
[user]
    name = Jane Doe
    email = jane@example.com
    signingkey = ABC123
```

### Core Settings

```ini
[core]
    editor = vim
    autocrlf = input
    filemode = true
    ignorecase = false
    quotepath = false
```

### Remote Repositories

```ini
[remote "origin"]
    url = https://github.com/user/repo.git
    fetch = +refs/heads/*:refs/remotes/origin/*
    pushurl = git@github.com:user/repo.git

[remote "upstream"]
    url = https://github.com/project/repo.git
    fetch = +refs/heads/*:refs/remotes/upstream/*
```

### Branch Configuration

```ini
[branch "main"]
    remote = origin
    merge = refs/heads/main
    rebase = true

[branch "develop"]
    remote = origin
    merge = refs/heads/develop
```

### Aliases

```ini
[alias]
    st = status
    co = checkout
    br = branch
    ci = commit
    unstage = reset HEAD --
    last = log -1 HEAD
    visual = log --graph --oneline --decorate --all
```

### URL Rewrites

```ini
[url "git@github.com:"]
    insteadOf = https://github.com/
```

**Note**: URL rewrites are not fully supported by this library.

## Configuration Scopes

Git supports multiple configuration scopes with defined precedence:

### Scope Hierarchy (Highest to Lowest)

1. **Environment variables** - `GIT_CONFIG_COUNT`, `GIT_CONFIG_KEY_*`, `GIT_CONFIG_VALUE_*`
2. **Worktree** - `.git/config.worktree`
3. **Local** - `.git/config` (repository-specific)
4. **Global** - `~/.gitconfig` or `$XDG_CONFIG_HOME/git/config` (user-specific)
5. **System** - `/etc/gitconfig` (system-wide)

### Scope File Locations

Default locations by scope:

| Scope | Linux/macOS | Windows | Customizable |
|-------|-------------|---------|--------------|
| System | `/etc/gitconfig` | `C:\ProgramData\Git\config` | Yes |
| Global | `~/.gitconfig` | `C:\Users\<user>\.gitconfig` | Yes |
| Local | `.git/config` | `.git\config` | No |
| Worktree | `.git/config.worktree` | `.git\config.worktree` | No |

## Library-Specific Behavior

### Round-Trip Preservation

This library attempts to preserve the original file structure when writing:

**Preserved:**

- Comments (full-line)
- Blank lines
- Section order
- Key order within sections

**Not Always Preserved:**

- Exact whitespace formatting (tabs vs spaces)
- Inline comments
- Specific indentation

### Limitations

**Not Supported:**

- Bare boolean values (keys without `=` sign)
- Some advanced include conditions (onbranch, hasconfig)
- URL rewrite patterns
- Replacing specific instances of multivars

**Partial Support:**

- Worktree configurations
- Some escape sequences
- Inline comments

### Differences from Git

- **Error handling**: This library may be more lenient with malformed configs
- **Whitespace**: Minor differences in whitespace preservation
- **Extensions**: Git supports more conditional include types
- **URL matching**: Git's URL insteadOf patterns are not implemented

## Validation and Error Handling

### Valid Configuration

```ini
[section]
    key = value
```

### Common Errors

**1. Missing section header:**

```ini
# ERROR: key without section
key = value
```

**2. Invalid section syntax:**

```ini
# ERROR: unmatched quotes
[section "subsection]
```

**3. Circular includes:**

```ini
# config-a
[include]
    path = config-b

# config-b
[include]
    path = config-a  # ERROR: circular reference
```

**4. Invalid escape sequences:**

```ini
[section]
    # ERROR: unknown escape sequence
    key = "value\x"
```

## Best Practices

1. **Use appropriate scopes**
   - User preferences → Global (`~/.gitconfig`)
   - Repository settings → Local (`.git/config`)
   - System defaults → System (`/etc/gitconfig`)

2. **Quote values with special characters**

   ```ini
   [section]
       # Good
       path = "C:\\Users\\Name\\Documents"
       
       # May cause issues
       path = C:\Users\Name\Documents
   ```

3. **Comment your configuration**

   ```ini
   [core]
       # Use vim for commit messages
       editor = vim
   ```

4. **Organize related settings**

   ```ini
   [user]
       name = Jane Doe
       email = jane@example.com
   
   [commit]
       gpgsign = true
   
   [gpg]
       program = gpg2
   ```

5. **Use includes for environment-specific settings**

   ```ini
   # ~/.gitconfig
   [include]
       path = ~/.gitconfig-personal
   
   [includeIf "gitdir:~/work/"]
       path = ~/.gitconfig-work
   ```

## Examples

### Complete Configuration File

```ini
# Global Git configuration
# ~/.gitconfig

[user]
    name = Jane Doe
    email = jane@example.com
    signingkey = ABC123DEF456

[core]
    editor = vim
    autocrlf = input
    excludesfile = ~/.gitignore_global
    
[commit]
    gpgsign = true
    verbose = true

[alias]
    st = status -sb
    co = checkout
    br = branch
    ci = commit
    unstage = reset HEAD --
    last = log -1 HEAD
    lg = log --graph --pretty=format:'%h %s'

[push]
    default = current
    followTags = true

[pull]
    rebase = true

[remote "origin"]
    fetch = +refs/heads/*:refs/remotes/origin/*

[includeIf "gitdir:~/work/"]
    path = ~/.gitconfig-work
```

### Environment-Specific Configuration

**Personal** (`~/.gitconfig-personal`):

```ini
[user]
    email = personal@gmail.com

[core]
    editor = code --wait
```

**Work** (`~/.gitconfig-work`):

```ini
[user]
    email = jane.doe@company.com
    signingkey = WORK123KEY456

[core]
    editor = vim

[commit]
    gpgsign = true
```

## References

- **Official Git Documentation**: <https://git-scm.com/docs/git-config>
- **Configuration File Format**: <https://mirrors.edge.kernel.org/pub/software/scm/git/docs/git-config.html>
- **Git Book - Configuration**: <https://git-scm.com/book/en/v2/Customizing-Git-Git-Configuration>

# Project Overview

gitconfig is a library to interact with git configuration files without a dependency on the git cli tool. It is useful when the git application
might not be available and to avoid breaking changes when the git behaviour should change. It is yet another configuration format, but it does
fit very well to the use cases of the [gopass project](https://github.com/gopasspw/gopass) where this did originate from.

The primary use case of this library is to support the use cases of gopass, but we aim for full git compatability. See [the git documentation](https://mirrors.edge.kernel.org/pub/software/scm/git/docs/git-config.html) for reference.

The project is specifically targeting users on all major platform, i.e. Linux, Unix, MacOS and Windows.

## Project Structure

- `config.go` contains the actual config parser. When making changes to the config through this library the parser tries to maintain the input file as much as possible, including whitespace and comments. So we have to maintain the `raw` representation of the input and update that accordingly when applying mutations. In a given git repository there can be multiple config scopes. Each config struct handles one scope.
- `configs.go` contains a struct representing all possible Git configs in a given repository. The different scopes have clearly defined scopes. We support the following scopes in decreasing order of priority: Environment variables (env), per-worktree configs (worktree), per-repository configs (local), per-user configs (global), system-wide configs (system) and as a pseudo-config presets that define built-in default values.
- `doc.go` contains godoc documentation and examples. Update this if you make user-visible changes.
- `gitconfig.go` contains default settings that pre-configure the library to work with Git repos. Other applications, like gopass, override these through global variables.

## Libraries and Frameworks

- Avoid introducing new external dependencies unless absolutely necessary.
- If a new dependency is required, please state the reason.
- The project is licensed under the terms of the MIT license and we can only add compatible licenses.
- We must avoid introducing CGo dependencies since this make cross-compiling infeasible.

## Testing instructions

- Always run `make test` and `make codequality` before submitting.
- Run `make fmt` to properly format the code. Run this before `make codequality`.
- Before mailing a PR run `make test crosscompile`

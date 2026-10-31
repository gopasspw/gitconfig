// Package gitconfig implements a pure Go parser of Git SCM config files. The support
// is currently not matching git exactly, e.g. includes, urlmatches and multivars are currently
// not supported. And while we try to preserve the original file a much as possible
// when writing we currently don't exactly retain (insignificant) whitespaces.
//
// The reference for this implementation is https://mirrors.edge.kernel.org/pub/software/scm/git/docs/git-config.html
//
// # Usage
//
// Use gitconfig.LoadAll with an optional workspace argument to process configuration
// input from these locations in order (i.e. the later ones take precedence):
//
//   - `system` - /etc/gitconfig
//   - `global` - `$XDG_CONFIG_HOME/git/config` or `~/.gitconfig`
//   - `local` - `<workdir>/config`
//   - `worktree` - `<workdir>/config.worktree`
//   - `command` - GIT_CONFIG_{COUNT,KEY,VALUE} environment variables
//
// Note: We do not support parsing command line flags directly, but one
// can use the SetEnv method to set flags from the command line in the config.
//
// # Customization
//
// `gopass` and other users of this package can easily customize file and environment
// names by utilizing the exported variables from the Configs struct:
//
//   - SystemConfig - Path to system-wide config (e.g., /etc/gitconfig)
//   - GlobalConfig - Path to user config, absolute or relative to the user's
//     home directory (e.g., .gitconfig), or "" to disable
//   - LocalConfig - Per-repository config name (e.g., .git/config)
//   - WorktreeConfig - Per-worktree config name (e.g., .git/config.worktree)
//   - EnvPrefix - Environment variable prefix (defaults to GIT_CONFIG)
//
// Note: For tests users will want to set `NoWrites = true` to avoid overwriting
// their real configs. `NoWrites` is propagated to the global, local and
// worktree scopes on LoadAll. To allow writes to a single scope only, override
// the scope's own `NoWrites` flag after loading:
//
//	cfg := gitconfig.New()
//	cfg.NoWrites = true
//	cfg.LoadAll(".")
//	if local, ok := cfg.Scope("local"); ok {
//	  local.NoWrites = false
//	}
//	cfg.SetLocal("core.pager", "less") // persisted, other scopes stay untouched
//
// # Examples
//
// ## Loading and Reading Configuration
//
// Basic reading from all scopes (respects precedence):
//
//	cfg := gitconfig.New()
//	cfg.LoadAll(".")
//	value := cfg.Get("user.name")
//	fmt.Println(value)  // Reads from highest priority scope available
//
// ## Reading from Specific Scopes
//
// Access configuration from a specific scope:
//
//	cfg := gitconfig.New()
//	cfg.LoadAll(".")
//	local := cfg.GetLocal("core.editor")
//	global := cfg.GetGlobal("user.email")
//	system := cfg.GetSystem("core.pager")
//
// ## Enumerating Keys
//
// List the keys of all scopes, filtered by prefix, or of a single scope:
//
//	cfg := gitconfig.New()
//	cfg.LoadAll(".")
//
//	for _, key := range cfg.Keys() {         // keys of all scopes, sorted
//		fmt.Printf("%s=%s\n", key, cfg.Get(key))
//	}
//	for _, key := range cfg.List("remote.") { // prefix filtered
//		fmt.Println(key)
//	}
//
//	local, ok := cfg.Scope("local") // reach a single scope
//	if ok {
//		for key, values := range local.Entries() { // key -> all values
//			fmt.Printf("%s: %v\n", key, values)
//		}
//	}
//
// Keys returns canonical keys (section[.subsection].key) with sections and
// key names lowercased and subsections case preserved. Entries returns a
// deep copy, so callers may modify the result without affecting the config.
//
// ## Customization for Other Applications
//
// Configure for a different application (like gopass):
//
//	cfg := gitconfig.New()
//	cfg.SystemConfig = "/etc/gopass/config"
//	cfg.GlobalConfig = ""
//	cfg.LocalConfig = ".gopass-config"
//	cfg.EnvPrefix = "GOPASS_CONFIG"
//	cfg.LoadAll(".")
//	notifications := cfg.Get("core.notifications")
//
// ## Writing Configuration
//
// Modify and persist changes:
//
//	cfg, _ := gitconfig.LoadConfig(".git/config")
//	cfg.Set("user.name", "John Doe")
//	cfg.Set("user.email", "john@example.com")
//	cfg.Write()  // Persist changes to disk
//
// Set and Unset already persist to disk. Write is only needed to persist
// changes that were made while NoWrites was set, or to retry a failed write.
//
// ## Removing Sections
//
// Unset removes individual keys and may leave an empty section header behind.
// Use UnsetSection to remove a whole section including all of its keys, or
// PruneEmptySections to drop all sections that no longer contain any keys:
//
//	cfg, _ := gitconfig.LoadConfig(".git/config")
//	cfg.UnsetSection("remote.origin") // remove [remote "origin"] and its keys
//	cfg.PruneEmptySections()          // drop any remaining empty sections
//
// UnsetSection accepts "section", "section.subsection" or a section header
// such as `[remote "origin"]`. Section names are matched case-insensitively,
// subsection names case-sensitively. Configs.UnsetSectionLocal,
// Configs.UnsetSectionGlobal and Configs.PruneEmptySections apply the same
// operations to the corresponding scopes.
//
// ## Reordering Sections and Keys
//
// By default the library preserves the order of the original file, like the
// git CLI does. Sorting is opt-in via Sort and SortOptions:
//
//	cfg, _ := gitconfig.LoadConfig(".git/config")
//	err := cfg.Sort(gitconfig.SortOptions{
//		GroupSections: true, // keep all [remote "..."] sections together
//		SortKeys:      true, // sort keys within each section
//	})
//
// SortSections additionally sorts the sections alphabetically. Multi-valued
// keys keep their relative order and comments directly above a key move with
// it. Configs.Sort applies the same options to all writable scopes.
//
// Note that reordering can change the effective configuration, since the order
// of multi-valued keys and of include directives matters.
//
// ## Scope-Specific Writes
//
// Write to specific scopes in multi-scope configs:
//
//	cfg := gitconfig.New()
//	cfg.LoadAll(".")
//	cfg.SetLocal("core.autocrlf", "true")   // Write to .git/config
//	cfg.SetGlobal("user.signingkey", "...")  // Write to ~/.gitconfig
//	cfg.SetSystem("core.pager", "less")      // Write to /etc/gitconfig
//
// ## Error Handling
//
// Use errors.Is to detect common error categories:
//
//	if err := cfg.Set("invalid", "value"); err != nil {
//		if errors.Is(err, gitconfig.ErrInvalidKey) {
//			// handle invalid key
//		}
//	}
//
//	if err := cfgs.SetLocal("core.editor", "vim"); err != nil {
//		if errors.Is(err, gitconfig.ErrWorkdirNotSet) {
//			// call LoadAll or provide a workdir
//		}
//	}
//
// # Versioning and Compatibility
//
// We aim to support the latest stable release of Git only.
// Currently we do not provide any backwards compatibility
// and semantic versioning.
//
// # Known limitations
//
// * Worktree support is only partial
// * Bare boolean values are not supported (e.g. a setting were only the key is present)
// * includeIf suppport is only partial, i.e. we only support the gitdir option
package gitconfig

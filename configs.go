package gitconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gopasspw/gopass/pkg/appdir"
	"github.com/gopasspw/gopass/pkg/debug"
	"github.com/gopasspw/gopass/pkg/set"
)

// Configs represents all git configuration files for a repository.
//
// Configs manages multiple Config objects from different scopes with a unified
// interface. It handles loading and merging configurations from multiple sourc with priority.
//
// Scope Priority (highest to lowest):
// 1. Environment variables (GIT_CONFIG_*)
// 2. Worktree-specific config (.git/config.worktree)
// 3. Local/repository config (.git/config)
// 4. Global/user config (~/.gitconfig)
// 5. System config (/etc/gitconfig)
// 6. Preset/built-in defaults
//
// Fields:
// - Preset: Built-in default configuration (optional)
// - system, global, local, worktree, env: Config objects for each scope
// - workdir: Working directory (used to locate local and worktree configs)
// - Name: Configuration set name (e.g., "git" or "gopass")
// - SystemConfig, GlobalConfig, LocalConfig, WorktreeConfig: File paths
// - EnvPrefix: Prefix for environment variables (e.g., "GIT_CONFIG")
// - NoWrites: If true, prevents all writes to disk
//
// Usage:
//
//	cfg := New()
//	cfg.LoadAll(".")  // Load from current directory
//	value := cfg.Get("core.editor")  // Reads from all scopes
//	cfg.SetLocal("core.pager", "less")  // Write to local only
type Configs struct {
	Preset   *Config
	system   *Config
	global   *Config
	local    *Config
	worktree *Config
	env      *Config
	workdir  string

	Name           string
	SystemConfig   string
	GlobalConfig   string
	LocalConfig    string
	WorktreeConfig string
	EnvPrefix      string
	NoWrites       bool
}

// New creates a new Configs instance with default configuration.
//
// The returned instance is not yet loaded. Call LoadAll() to load configurations.
//
// Default settings:
// - Name: "git"
// - SystemConfig: "/etc/gitconfig" (Unix) or auto-detected (Windows)
// - GlobalConfig: "~/.gitconfig"
// - LocalConfig: "config" (relative to workdir)
// - WorktreeConfig: "config.worktree" (relative to workdir)
// - EnvPrefix: "GIT_CONFIG"
// - NoWrites: false (allows persisting changes)
//
// These settings can be customized before calling LoadAll():
//
//	cfg := New()
//	cfg.SystemConfig = "/etc/myapp/config"
//	cfg.EnvPrefix = "MYAPP_CONFIG"
//	cfg.LoadAll(".")
func New() *Configs {
	return &Configs{
		system: &Config{
			readonly: true,
		},
		global: &Config{
			path: globalConfigFile(name),
		},
		local:    &Config{},
		worktree: &Config{},
		env: &Config{
			noWrites: true,
		},

		Name:           name,
		SystemConfig:   systemConfig,
		GlobalConfig:   globalConfig,
		LocalConfig:    localConfig,
		WorktreeConfig: worktreeConfig,
		EnvPrefix:      envPrefix,
	}
}

// Reload reloads all configuration files from disk.
//
// This is useful when configuration files have been modified externally.
// Uses the same workdir that was provided to the last LoadAll call.
func (cs *Configs) Reload() {
	cs.LoadAll(cs.workdir)
}

// String implements fmt.Stringer for debugging.
func (cs *Configs) String() string {
	return fmt.Sprintf("GitConfigs{Name: %s - Workdir: %s - Env: %s - System: %s - Global: %s - Local: %s - Worktree: %s}", cs.Name, cs.workdir, cs.EnvPrefix, cs.SystemConfig, cs.GlobalConfig, cs.LocalConfig, cs.WorktreeConfig)
}

// LoadAll loads all known configuration files from their configured locations.
//
// Behavior:
// - Loads configs from all scopes (system, global, local, worktree, env)
// - Missing or invalid files are silently ignored
// - Never returns an error (always returns &cs for chaining)
// - workdir is optional; if empty, local and worktree configs are not loaded
// - Processes include and includeIf directives
// - Merges all configs with proper scope priority
//
// Parameters:
// - workdir: Working directory (usually repo root) to locate local/worktree configs
//
// Example:
//   cfg := New()
//   cfg.LoadAll("/path/to/repo")
//   // Now ready to use Get, Set, etc.
func (cs *Configs) LoadAll(workdir string) *Configs {
	cs.workdir = workdir

	debug.Log("Loading gitconfigs for %s", cs.Name)

	// load the system config, if any
	if os.Getenv(cs.EnvPrefix+"_NOSYSTEM") == "" {
		c, err := LoadConfig(cs.SystemConfig)
		if err != nil {
			debug.V(1).Log("[%s] failed to load system config: %s", cs.Name, err)
		} else {
			debug.V(1).Log("[%s] loaded system config from %s", cs.Name, cs.SystemConfig)
			cs.system = c
			// the system config should generally not be written from gopass.
			// in almost any scenario gopass shouldn't have write access
			// and even if it does we shouldn't accidentially change it.
			// It's for operators and package mainatiners.
			cs.system.readonly = true
		}
	}

	// load the "global" (per user) config, if any
	cs.loadGlobalConfigs()
	cs.global.noWrites = cs.NoWrites

	// load the local config, if any
	if workdir != "" {
		localConfigPath := filepath.Join(workdir, cs.LocalConfig)
		c, err := LoadConfig(localConfigPath)
		if err != nil {
			debug.V(1).Log("[%s] failed to load local config from %s: %s", cs.Name, localConfigPath, err)
			// set the path just in case we want to modify / write to it later
			cs.local.path = localConfigPath
		} else {
			debug.V(1).Log("[%s] loaded local config from %s", cs.Name, localConfigPath)
			cs.local = c
		}
	}
	cs.local.noWrites = cs.NoWrites

	// load the worktree config, if any
	if workdir != "" {
		worktreeConfigPath := filepath.Join(workdir, cs.WorktreeConfig)
		c, err := LoadConfig(worktreeConfigPath)
		if err != nil {
			debug.V(3).Log("[%s] failed to load worktree config from %s: %s", cs.Name, worktreeConfigPath, err)
			// set the path just in case we want to modify / write to it later
			cs.worktree.path = worktreeConfigPath
		} else {
			debug.V(1).Log("[%s] loaded worktree config from %s", cs.Name, worktreeConfigPath)
			cs.worktree = c
		}
	}
	cs.worktree.noWrites = cs.NoWrites

	// load any env vars
	cs.env = LoadConfigFromEnv(cs.EnvPrefix)

	return cs
}

func globalConfigFile(name string) string {
	// $XDG_CONFIG_HOME/git/config
	return filepath.Join(appdir.New(name).UserConfig(), "config")
}

// loadGlobalConfigs will try to load the per-user (Git calls them "global") configs.
// Since we might need to try different locations but only want to use the first one
// it's easier to handle this in its own method.
func (cs *Configs) loadGlobalConfigs() string {
	locs := []string{
		globalConfigFile(cs.Name),
	}

	if cs.GlobalConfig != "" {
		// ~/.gitconfig
		locs = append(locs, filepath.Join(appdir.UserHome(), cs.GlobalConfig))
	}

	// if we already have a global config we can just reload it instead of trying all locations
	if !cs.global.IsEmpty() {
		if p := cs.global.path; p != "" {
			debug.V(1).Log("[%s] reloading existing global config from %s", cs.Name, p)
			cfg, err := LoadConfig(p)
			if err != nil {
				debug.V(1).Log("[%s] failed to reload global config from %s", cs.Name, p)
			} else {
				cs.global = cfg

				return p
			}
		}
	}

	debug.V(1).Log("[%s] trying to find global configs in %v", cs.Name, locs)
	for _, p := range locs {
		// GlobalConfig might be set to an empty string to disable it
		// and instead of the XDG_CONFIG_HOME path only.
		if p == "" {
			continue
		}
		cfg, err := LoadConfig(p)
		if err != nil {
			debug.V(1).Log("[%s] failed to load global config from %s: %s", cs.Name, p, err)

			continue
		}

		debug.V(1).Log("[%s] loaded global config from %s", cs.Name, p)
		cs.global = cfg

		return p
	}

	debug.V(1).Log("[%s] no global config found", cs.Name)

	// set the path to the default one in case we want to write to it (create it) later
	cs.global = &Config{
		path: globalConfigFile(cs.Name),
	}

	return ""
}

// HasGlobalConfig indicates if a per-user config can be found.
//
// Returns true if a global config file exists at one of the configured locations.
func (cs *Configs) HasGlobalConfig() bool {
	return cs.loadGlobalConfigs() != ""
}

// Get returns the value for the given key from the first scope that contains it.
//
// Lookup Order (by scope priority):
// 1. Environment variables (GIT_CONFIG_*)
// 2. Worktree config (.git/config.worktree)
// 3. Local config (.git/config)
// 4. Global config (~/.gitconfig)
// 5. System config (/etc/gitconfig)
// 6. Preset/defaults
//
// The search stops at the first scope that has the key. Earlier scopes override later ones.
//
// Returns the value as a string. For keys with multiple values, returns the first one.
// Returns empty string if key not found in any scope.
//
// Example:
//   editor := cfg.Get("core.editor")
//   if editor != "" {
//     fmt.Printf("Using editor: %s\n", editor)
//   }
func (cs *Configs) Get(key string) string {
	for _, cfg := range []*Config{
		cs.env,
		cs.worktree,
		cs.local,
		cs.global,
		cs.system,
		cs.Preset,
	} {
		if cfg == nil || cfg.vars == nil {
			continue
		}
		if v, found := cfg.Get(key); found {
			return v
		}
	}

	debug.V(3).Log("[%s] no value for %s found", cs.Name, key)

	return ""
}

// GetAll returns all values for the given key from the first scope that contains it.
//
// Like Get but returns all values for keys that can have multiple entries.
// See Get documentation for scope priority.
//
// Returns nil if key not found in any scope.
func (cs *Configs) GetAll(key string) []string {
	for _, cfg := range []*Config{
		cs.env,
		cs.worktree,
		cs.local,
		cs.global,
		cs.system,
		cs.Preset,
	} {
		if cfg == nil || cfg.vars == nil {
			continue
		}
		if vs, found := cfg.GetAll(key); found {
			return vs
		}
	}

	debug.V(3).Log("[%s] no value for %s found", cs.Name, key)

	return nil
}

// GetFrom returns the value for the given key from the given scope. Valid scopes are:
// env, worktree, local, global, system and preset.
func (cs *Configs) GetFrom(key string, scope string) (string, bool) {
	switch strings.ToLower(scope) {
	case "env":
		return cs.env.Get(key)
	case "worktree":
		return cs.worktree.Get(key)
	case "local":
		return cs.local.Get(key)
	case "global":
		return cs.global.Get(key)
	case "system":
		return cs.system.Get(key)
	case "preset":
		return cs.Preset.Get(key)
	default:
		debug.V(3).Log("[%s] unknown config scope %s for key %s", cs.Name, scope, key)

		return "", false
	}
}

// GetGlobal specifically asks the per-user (global) config for a key.
//
// This bypasses the scope priority and only reads from the global config.
// Useful when you specifically want settings from ~/.gitconfig.
//
// Returns empty string if the key is not found in the global config.
//
// Example:
//   name, _ := cfg.GetGlobal("user.name")
func (cs *Configs) GetGlobal(key string) string {
	if cs.global == nil {
		return ""
	}

	if v, found := cs.global.Get(key); found {
		return v
	}

	debug.V(3).Log("[%s] no value for %s found", cs.Name, key)

	return ""
}

// GetLocal specifically asks the per-directory (local) config for a key.
//
// This bypasses the scope priority and only reads from the local config (.git/config).
// Useful when you specifically want settings from the repository's config.
//
// Returns empty string if the key is not found in the local config.
//
// Example:
//   url, _ := cfg.GetLocal("remote.origin.url")
func (cs *Configs) GetLocal(key string) string {
	if cs.local == nil {
		return ""
	}

	if v, found := cs.local.Get(key); found {
		return v
	}

	debug.V(3).Log("[%s] no value for %s found", cs.Name, key)

	return ""
}

// IsSet returns true if this key is set in any of our configs.
func (cs *Configs) IsSet(key string) bool {
	for _, cfg := range []*Config{
		cs.env,
		cs.worktree,
		cs.local,
		cs.global,
		cs.system,
		cs.Preset,
	} {
		if cfg != nil && cfg.IsSet(key) {
			return true
		}
	}

	return false
}

// SetLocal sets (or adds) a key only in the per-directory (local) config.
func (cs *Configs) SetLocal(key, value string) error {
	if cs.local == nil {
		if cs.workdir == "" {
			return fmt.Errorf("no workdir set")
		}
		cs.local = &Config{
			path: filepath.Join(cs.workdir, cs.LocalConfig),
		}
	}

	return cs.local.Set(key, value)
}

// SetGlobal sets (or adds) a key only in the per-user (global) config.
func (cs *Configs) SetGlobal(key, value string) error {
	if cs.global == nil {
		cs.global = &Config{
			path: globalConfigFile(cs.Name),
		}
	}

	return cs.global.Set(key, value)
}

// SetEnv sets (or adds) a key in the per-process (env) config. Useful
// for persisting flags during the invocation.
func (cs *Configs) SetEnv(key, value string) error {
	if cs.env == nil {
		cs.env = &Config{
			noWrites: true,
		}
	}

	return cs.env.Set(key, value)
}

// UnsetLocal deletes a key from the local config.
func (cs *Configs) UnsetLocal(key string) error {
	if cs.local == nil {
		return nil
	}

	return cs.local.Unset(key)
}

// UnsetGlobal deletes a key from the global config.
func (cs *Configs) UnsetGlobal(key string) error {
	if cs.global == nil {
		return nil
	}

	return cs.global.Unset(key)
}

// Keys returns a list of all keys from all available scopes. Every key has section and possibly
// a subsection. They are separated by dots. The subsection itself may contain dots. The final
// key name and the section MUST NOT contain dots.
//
// Examples
//   - remote.gist.gopass.pw.path -> section: remote, subsection: gist.gopass.pw, key: path
//   - core.timeout -> section: core, key: timeout
func (cs *Configs) Keys() []string {
	keys := make([]string, 0, 128)

	for _, cfg := range []*Config{
		cs.Preset,
		cs.system,
		cs.global,
		cs.local,
		cs.worktree,
		cs.env,
	} {
		if cfg == nil {
			continue
		}
		for k := range cfg.vars {
			keys = append(keys, k)
		}
	}

	return set.Sorted(keys)
}

// List returns all keys matching the given prefix. The prefix can be empty,
// then this is identical to Keys().
func (cs *Configs) List(prefix string) []string {
	return set.SortedFiltered(cs.Keys(), func(k string) bool {
		return strings.HasPrefix(k, prefix)
	})
}

// ListSections returns a sorted list of all sections.
func (cs *Configs) ListSections() []string {
	return set.Sorted(set.Apply(cs.Keys(), func(k string) string {
		section, _, _ := splitKey(k)

		return section
	}))
}

// ListSubsections returns a sorted list of all subsections
// in the given section.
func (cs *Configs) ListSubsections(wantSection string) []string {
	// apply extracts the subsection and matches it to the empty string
	// if it doesn't belong to the section we're looking for. Then the
	// filter func filters out any empty string.
	return set.SortedFiltered(set.Apply(cs.Keys(), func(k string) string {
		section, subsection, _ := splitKey(k)
		if section != wantSection {
			return ""
		}

		return subsection
	}), func(s string) bool {
		return s != ""
	})
}

// KVList returns a list of all keys and values matching the given prefix.
func (cs *Configs) KVList(prefix, sep string) []string {
	if sep == "" {
		sep = "="
	}
	keys := cs.List(prefix)
	kv := make([]string, 0, len(keys))
	for _, k := range keys {
		vs := cs.GetAll(k)
		for _, v := range vs {
			if v == "" {
				continue
			}
			kv = append(kv, fmt.Sprintf("%s%s%s", k, sep, v))
		}
	}

	sort.Strings(kv)

	return kv
}

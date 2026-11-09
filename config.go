package gitconfig

import (
	"bufio"
	"fmt"
	"io"
	"maps"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/gopasspw/gopass/pkg/debug"
)

var (
	keyValueTpl     = "\t%s = %s%s"
	keyTpl          = "\t%s%s"
	reQuotedComment = regexp.MustCompile(`"[^"]*[#;][^"]*"`)
	// "The variable names are case-insensitive, allow only alphanumeric characters and -, and must start with an alphabetic character."".
	reValidKey = regexp.MustCompile(`^[a-z]+[a-z0-9-]*$`)

	// CompatMode enables compatibility mode, which disables certain features like value unescaping.
	CompatMode bool
)

// Config represents a single git configuration file from one scope.
//
// Config handles reading and writing a single configuration file while attempting
// to preserve the original formatting (comments, whitespace, section order).
//
// Fields:
// - path: File path of this config file
// - readonly: If true, prevents any modifications (even in-memory)
// - NoWrites: If true, prevents persisting changes to disk (useful for testing)
// - raw: Maintains the raw text representation for round-trip fidelity
// - vars: Map of normalized keys to their values (may be multiple values per key)
// - branch: Current git branch name (for onbranch conditionals)
//
// Note: Config is not thread-safe. Concurrent access from multiple goroutines
// is not supported. Callers must provide synchronization if needed.
//
// Typical Usage:
//
//	cfg, err := LoadConfig("~/.gitconfig")
//	if err != nil { ... }
//	value, ok := cfg.Get("core.editor")
//	if err := cfg.Set("core.pager", "less"); err != nil { ... }
//	if err := cfg.Write(); err != nil { ... }
type Config struct {
	path     string
	readonly bool // do not allow modifying values (even in memory)
	raw      strings.Builder
	vars     map[string][]string
	branch   string

	// NoWrites prevents persisting changes to disk when set to true. Set and
	// Unset still update the in-memory representation, but Write (and the
	// implicit write performed by Set and Unset) becomes a no-op. This is
	// useful for tests that must not touch the user's real config files.
	//
	// Note that a Config obtained from Configs inherits the value of
	// Configs.NoWrites when the configs are loaded.
	NoWrites bool
}

// IsEmpty returns true if the config is empty (no configuration loaded).
//
// An empty config is one that:
// - Is nil
// - Has no variables loaded
// - Has no raw content (not just missing path reference)
//
// This is used to distinguish between "not yet loaded" and "loaded but empty file".
func (c *Config) IsEmpty() bool {
	if c == nil || c.vars == nil {
		return true
	}

	if c.raw.Len() > 0 {
		return false
	}

	return true
}

// Unset deletes a key from the config.
//
// Behavior:
// - If the key exists, it's removed from vars and the raw config string
// - If the key doesn't exist, this is a no-op (no error)
// - The underlying config file is updated if possible
// - Readonly configs silently ignore the unset operation
//
// Note: Unset only removes individual keys. If this leaves an empty section
// behind, use PruneEmptySections to drop it, or UnsetSection to remove a whole
// section at once.
//
// Example:
//
//	if err := cfg.Unset("core.pager"); err != nil {
//	  log.Fatal(err)
//	}
func (c *Config) Unset(key string) error {
	if c.readonly {
		return nil
	}

	section, _, subkey := splitKey(key)
	if section == "" || subkey == "" {
		return fmt.Errorf("%w: %s", ErrInvalidKey, key)
	}

	key = canonicalizeKey(key)

	_, present := c.vars[key]
	if !present {
		return nil
	}

	delete(c.vars, key)

	return c.rewriteRaw(key, "", func(fKey, key, value, comment, _ string) (string, bool) {
		return "", true
	})
}

// UnsetSection deletes a whole section, including all of its keys, from the
// config.
//
// The section may be given as "section" or "section.subsection", e.g. "core"
// or "remote.origin". A section header as it appears in the file, e.g.
// `[remote "origin"]`, is accepted as well. Section names are matched
// case-insensitively, subsection names case-sensitively, as usual for git
// config. If the section appears multiple times in the file, all occurrences
// are removed.
//
// Behavior:
// - All keys of the section are removed from the in-memory representation
// - All matching section headers and their keys are removed from the raw config
// - If the section doesn't exist, this is a no-op (no error)
// - Readonly configs silently ignore the operation
// - The underlying config file is updated if possible
//
// Note: This only removes the section given by name. To remove all
// subsections of a section, call UnsetSection for each of them, or use
// PruneEmptySections after unsetting the individual keys.
//
// Example:
//
//	if err := cfg.UnsetSection("remote.origin"); err != nil {
//	  log.Fatal(err)
//	}
func (c *Config) UnsetSection(section string) error {
	if c.readonly {
		return nil
	}

	wSection, wSubsection, ok := splitSection(section)
	if !ok {
		return fmt.Errorf("%w: %s", ErrInvalidKey, section)
	}

	// remove all keys of the section from the in-memory representation
	changed := false
	for k := range c.vars {
		s, sub, _ := splitKey(k)
		if strings.EqualFold(s, wSection) && sub == wSubsection {
			delete(c.vars, k)
			changed = true
		}
	}

	blocks := splitConfigBlocks(c.raw.String())
	kept := make([]*configBlock, 0, len(blocks))
	for _, b := range blocks {
		if !b.preamble && strings.EqualFold(b.section, wSection) && b.subsection == wSubsection {
			changed = true

			continue
		}
		kept = append(kept, b)
	}

	if !changed {
		return nil
	}

	c.raw = strings.Builder{}
	c.raw.WriteString(blocksToString(kept))

	return c.flushRaw()
}

// PruneEmptySections removes all sections that don't contain any keys from the
// config. This is useful after unsetting individual keys, which may leave
// empty section headers behind.
//
// Comments that were part of a removed section are preserved and moved to the
// preceding section (or to the top of the file), so no user comments are lost.
// Readonly configs silently ignore the operation, and NoWrites only prevents
// persisting the result to disk.
//
// Example:
//
//	if err := cfg.Unset("remote.origin.url"); err != nil {
//	  log.Fatal(err)
//	}
//	if err := cfg.PruneEmptySections(); err != nil { // drop the now empty [remote "origin"]
//	  log.Fatal(err)
//	}
func (c *Config) PruneEmptySections() error {
	if c.readonly {
		return nil
	}

	blocks := splitConfigBlocks(c.raw.String())
	kept := make([]*configBlock, 0, len(blocks))
	changed := false
	for _, b := range blocks {
		if b.preamble || len(b.chunks) > 0 {
			kept = append(kept, b)

			continue
		}

		changed = true

		// keep comments of the removed section, but drop blank lines
		comments := make([]string, 0, len(b.trailing))
		for _, line := range b.trailing {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ";") {
				comments = append(comments, line)
			}
		}
		if len(comments) == 0 {
			continue
		}
		if len(kept) == 0 {
			kept = append(kept, &configBlock{preamble: true})
		}
		prev := kept[len(kept)-1]
		prev.trailing = append(prev.trailing, comments...)
	}

	if !changed {
		return nil
	}

	c.raw = strings.Builder{}
	c.raw.WriteString(blocksToString(kept))

	return c.flushRaw()
}

// Get returns the last value of the key in this config scope, matching
// canonical `git config --get` (last-wins) semantics for multi-valued keys.
//
// Use GetAll to retrieve all values for a key.
//
// The key is case-insensitive for sections and key names but case-sensitive
// for subsection names (per git-config specification).
//
// Returns (value, true) if the key is found, ("", false) otherwise.
//
// Example:
//
//	v, ok := cfg.Get("core.editor")
//	if ok {
//	  fmt.Printf("Editor: %s\n", v)
//	}
func (c *Config) Get(key string) (string, bool) {
	if c == nil {
		return "", false
	}
	key = canonicalizeKey(key)
	vs, found := c.vars[key]
	if !found || len(vs) < 1 {
		return "", false
	}

	return vs[len(vs)-1], true
}

// GetBool returns the boolean interpretation of the key's value per git-config rules.
//
// Truthy values: "true", "yes", "on", "1", or a bare key without '='.
// Falsy values: "false", "no", "off", "0", or an empty value after '='.
// Returns (false, false) if the key is not set or cannot be parsed as a boolean.
func (c *Config) GetBool(key string) (bool, bool) {
	if c == nil {
		return false, false
	}
	v, ok := c.Get(key)
	if !ok {
		return false, false
	}

	return parseBoolValue(v)
}

func parseBoolValue(v string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "true", "yes", "on", "1":
		return true, true
	case "false", "no", "off", "0", "":
		return false, true
	default:
		return false, false
	}
}

// GetInt returns the integer interpretation of the key's value per git-config rules.
//
// Supports optional case-insensitive unit suffixes:
// - k: multiplies by 1024
// - m: multiplies by 1048576 (1024^2)
// - g: multiplies by 1073741824 (1024^3)
func (c *Config) GetInt(key string) (int64, bool) {
	if c == nil {
		return 0, false
	}
	v, ok := c.Get(key)
	if !ok {
		return 0, false
	}

	return parseIntValue(v)
}

func parseIntValue(v string) (int64, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, false
	}

	multiplier := int64(1)
	switch last := v[len(v)-1]; last {
	case 'k', 'K':
		multiplier = 1024
		v = v[:len(v)-1]
	case 'm', 'M':
		multiplier = 1024 * 1024
		v = v[:len(v)-1]
	case 'g', 'G':
		multiplier = 1024 * 1024 * 1024
		v = v[:len(v)-1]
	}

	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, false
	}

	return n * multiplier, true
}

// GetPath returns the path value of the key, expanding a leading "~/" to the
// user's home directory per git-config path type rules.
func (c *Config) GetPath(key string) (string, bool) {
	if c == nil {
		return "", false
	}
	v, ok := c.Get(key)
	if !ok {
		return "", false
	}

	return expandPathValue(v), true
}

func expandPathValue(v string) string {
	if v == "~" {
		if home := userHomeDir(); home != "" {
			return home
		}
	}
	if strings.HasPrefix(v, "~/") || strings.HasPrefix(v, `~\`) {
		if home := userHomeDir(); home != "" {
			return filepath.Join(home, v[2:])
		}
	}

	return v
}

func userHomeDir() string {
	if hd := os.Getenv("GOPASS_HOMEDIR"); hd != "" {
		return hd
	}
	if hd, exists := os.LookupEnv("HOME"); exists && hd != "" {
		return hd
	}
	hd, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return hd
}

// GetAll returns all values of the key.
//
// Git config allows multiple values for the same key. This is common for:
// - Multiple include paths
// - Multiple aliases
// - Arrays in custom configurations
//
// Returns (values, true) if the key is found, (nil, false) otherwise.
// If found, values will be non-nil but may be empty.
//
// The returned slice is a copy; modifying it does not affect the Config.
//
// Example:
//
//	paths, ok := cfg.GetAll("include.path")
//	if ok {
//	  for _, path := range paths {
//	    fmt.Printf("Include: %s\n", path)
//	  }
//	}
func (c *Config) GetAll(key string) ([]string, bool) {
	key = canonicalizeKey(key)
	vs, found := c.vars[key]
	if !found {
		return nil, false
	}

	return slices.Clone(vs), true
}

// Keys returns the sorted list of all keys in this config scope.
//
// Each key has the form section[.subsection].key, as described by splitKey.
// Subsection names retain their original case, section and key names are
// lowercased (see canonicalizeKey). Multi-value keys are returned only once.
//
// Returns an empty (non-nil) slice for an empty or nil config.
//
// Example:
//
//	for _, key := range cfg.Keys() {
//	  value, _ := cfg.Get(key)
//	  fmt.Printf("%s=%s\n", key, value)
//	}
func (c *Config) Keys() []string {
	if c == nil || len(c.vars) == 0 {
		return []string{}
	}

	return slices.Sorted(maps.Keys(c.vars))
}

// List returns all keys matching the given prefix. The prefix can be empty,
// then this is identical to Keys().
//
// Example:
//
//	for _, key := range cfg.List("remote.") {
//	  fmt.Println(key)
//	}
func (c *Config) List(prefix string) []string {
	keys := c.Keys()
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		if strings.HasPrefix(k, prefix) {
			out = append(out, k)
		}
	}

	return out
}

// Entries returns a copy of all keys and their values in this config scope.
//
// The result is a deep copy. Modifying the returned map or any of its value
// slices does not affect the Config. Keys are returned in canonical form and
// the values of a key retain their order of appearance in the config file.
//
// Returns an empty (non-nil) map for an empty or nil config.
//
// Example:
//
//	for key, values := range cfg.Entries() {
//	  fmt.Printf("%s: %v\n", key, values)
//	}
func (c *Config) Entries() map[string][]string {
	if c == nil {
		return map[string][]string{}
	}

	entries := make(map[string][]string, len(c.vars))
	for k, vs := range c.vars {
		entries[k] = slices.Clone(vs)
	}

	return entries
}

// IsSet returns true if the key was set in this config.
//
// Returns true even if the value is empty string (unlike checking Get with ok).
//
// Example:
//
//	if cfg.IsSet("core.editor") {
//	  fmt.Println("Editor is configured")
//	}
func (c *Config) IsSet(key string) bool {
	key = canonicalizeKey(key)
	_, present := c.vars[key]

	return present
}

// Set updates or adds a key in the config.
//
// Behavior:
// - If the key exists, the first value is updated
// - If the key doesn't exist, it's added to an existing section or a new section
// - If possible, the underlying config file is written to disk
// - Original formatting (comments, whitespace) is preserved where possible
//
// Errors:
// - Returns error if readonly or key is invalid (missing section or key name)
// - Returns error if file write fails (but in-memory value may be set)
//
// This method normalizes the key (lowercase sections and key names) but preserves
// subsect names' case.
//
// Example:
//
//	if err := cfg.Set("core.pager", "less"); err != nil {
//	  log.Fatal(err)
//	}
func (c *Config) Set(key, value string) error {
	section, _, subkey := splitKey(key)
	if section == "" || subkey == "" {
		return fmt.Errorf("%w: %s", ErrInvalidKey, key)
	}
	key = canonicalizeKey(key)

	// can't set env vars
	if c.readonly {
		debug.Log("can not write to a readonly config")

		return nil
	}

	if c.vars == nil {
		c.vars = make(map[string][]string, 16)
	}

	// already present at the same value, no need to rewrite the config
	if vs, found := c.vars[key]; found {
		if slices.Contains(vs, value) {
			debug.V(1).Log("key %q with value %q already present. Not re-writing.", key, value)

			return nil
		}
	}

	vs, present := c.vars[key]
	if vs == nil {
		vs = make([]string, 1)
	}
	vs[0] = value
	c.vars[key] = vs

	debug.V(3).Log("set %q to %q", key, value)

	// a new key, insert it into an existing section, if any
	if !present {
		debug.V(3).Log("inserting value")

		return c.insertValue(key, value)
	}

	debug.V(3).Log("updating value")

	var updated bool

	return c.rewriteRaw(key, value, func(fKey, sKey, value, comment, line string) (string, bool) {
		if updated {
			return line, false
		}
		updated = true

		return formatKeyValue(sKey, value, comment), false
	})
}

// Add appends a new value for the key in the config (`git config --add`).
// Unlike Set, which updates the first value if the key already exists,
// Add always appends an additional entry for the key.
func (c *Config) Add(key, value string) error {
	section, _, subkey := splitKey(key)
	if section == "" || subkey == "" {
		return fmt.Errorf("%w: %s", ErrInvalidKey, key)
	}
	key = canonicalizeKey(key)

	if c.readonly {
		debug.Log("can not write to a readonly config")

		return nil
	}

	if c.vars == nil {
		c.vars = make(map[string][]string, 16)
	}

	vs, present := c.vars[key]
	prevCount := len(vs)
	c.vars[key] = append(vs, value)

	if !present || prevCount == 0 {
		return c.insertValue(key, value)
	}

	var seen int

	return c.rewriteRaw(key, value, func(fKey, sKey, val, comment, line string) (string, bool) {
		seen++
		if seen == prevCount {
			return line + "\n" + formatKeyValue(sKey, val, ""), false
		}

		return line, false
	})
}

// SetAll replaces all values of a multi-valued key with a single new value
// (`git config --replace-all`). If the key does not exist yet, it is inserted.
func (c *Config) SetAll(key, value string) error {
	section, _, subkey := splitKey(key)
	if section == "" || subkey == "" {
		return fmt.Errorf("%w: %s", ErrInvalidKey, key)
	}
	key = canonicalizeKey(key)

	if c.readonly {
		debug.Log("can not write to a readonly config")

		return nil
	}

	if c.vars == nil {
		c.vars = make(map[string][]string, 16)
	}

	_, present := c.vars[key]
	c.vars[key] = []string{value}

	if !present {
		return c.insertValue(key, value)
	}

	var updated bool

	return c.rewriteRaw(key, value, func(fKey, sKey, val, comment, line string) (string, bool) {
		if !updated {
			updated = true

			return formatKeyValue(sKey, val, comment), false
		}

		return "", true
	})
}

func (c *Config) insertValue(key, value string) error {
	debug.V(3).Log("input (%s: %s): \n--------------\n%s\n--------------\n", key, value, strings.Join(strings.Split("- "+c.raw.String(), "\n"), "\n- "))

	wSection, wSubsection, wKey := splitKey(key)

	s := bufio.NewScanner(strings.NewReader(c.raw.String()))

	lines := make([]string, 0, 128)
	var section string
	var subsection string
	var written bool
	for s.Scan() {
		line := s.Text()

		lines = append(lines, line)

		if written {
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			s, subs, skip := parseSectionHeader(line)
			if skip {
				continue
			}
			section = s
			subsection = subs
		}

		if section != wSection {
			continue
		}
		if subsection != wSubsection {
			continue
		}

		lines = append(lines, formatKeyValue(wKey, value, ""))
		written = true
	}

	// not added to an existing section, so add it at the end
	if !written {
		sect := fmt.Sprintf("[%s]", wSection)
		if wSubsection != "" {
			sect = fmt.Sprintf("[%s \"%s\"]", wSection, wSubsection)
		}
		lines = append(lines, sect)
		lines = append(lines, formatKeyValue(wKey, value, ""))
	}

	c.raw = strings.Builder{}
	c.raw.WriteString(strings.Join(lines, "\n"))
	c.raw.WriteString("\n")

	debug.V(3).Log("output: \n--------------\n%s\n--------------\n", strings.Join(strings.Split("+ "+c.raw.String(), "\n"), "\n+ "))

	return c.flushRaw()
}

// formatKeyValue formats a configuration key-value pair for writing to file.
// If the value is empty, only the key is written.
// Values containing special characters or leading/trailing whitespace are
// escaped and quoted so they round-trip cleanly when parsed back.
// The comment parameter preserves any trailing comment from the original line.
func formatKeyValue(key, value, comment string) string {
	if value == "" {
		return fmt.Sprintf(keyTpl, key, comment)
	}
	if !CompatMode {
		value = escapeValueForWrite(value)
	}

	return fmt.Sprintf(keyValueTpl, key, value, comment)
}

func escapeValueForWrite(v string) string {
	if v == "" {
		return v
	}

	needsQuotes := strings.TrimSpace(v) != v || strings.ContainsAny(v, "#;\"")
	hasEscapable := strings.ContainsAny(v, "\\\"\n\t\b")

	if !needsQuotes && !hasEscapable {
		return v
	}

	var b strings.Builder
	b.Grow(len(v) + 4)
	if needsQuotes {
		b.WriteByte('"')
	}
	for i := range len(v) {
		switch v[i] {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		case '\b':
			b.WriteString(`\b`)
		default:
			b.WriteByte(v[i])
		}
	}
	if needsQuotes {
		b.WriteByte('"')
	}

	return b.String()
}

// parseSectionHeader extracts the section and subsection from a config file section header line.
// For example:
//
//	"[core]" returns ("core", "", false)
//	"[remote \"origin\"]" returns ("remote", "origin", false)
//	"[]" returns ("", "", true) to indicate skip
//
// The skip return value indicates whether this line should be ignored.
func parseSectionHeader(line string) (section, subsection string, skip bool) { //nolint:nonamedreturns
	line = stripHeaderComment(strings.TrimSpace(line))
	if !strings.HasPrefix(line, "[") || !strings.HasSuffix(line, "]") {
		line = strings.Trim(line, "[]")
	} else {
		line = line[1 : len(line)-1]
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return "", "", true
	}
	wsp := strings.IndexAny(line, " \t")
	if wsp < 0 {
		if sec, sub, ok := strings.Cut(line, "."); ok {
			return strings.ToLower(sec), strings.ToLower(sub), false
		}

		return strings.ToLower(line), "", false
	}

	section = strings.ToLower(strings.TrimSpace(line[:wsp]))
	subsection = strings.TrimSpace(line[wsp+1:])
	subsection = strings.TrimPrefix(subsection, "\"")
	subsection = strings.TrimSuffix(subsection, "\"")
	subsection = unescapeSubsection(subsection)

	return section, subsection, false
}

// unescapeSubsection unescapes a quoted subsection name according to git-config rules:
// backslash followed by any character evaluates to that character (\\ -> \, \" -> ", \x -> x).
func unescapeSubsection(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			i++
			b.WriteByte(s[i])

			continue
		}
		if s[i] == '\\' {
			continue
		}
		b.WriteByte(s[i])
	}

	return b.String()
}

// rewriteRaw is used to rewrite the raw config copy. It is used for set and unset operations
// with different callbacks each.
func (c *Config) rewriteRaw(key, value string, cb parseFunc) error {
	debug.V(3).Log("input (%s: %s): \n--------------\n%s\n--------------\n", key, value, strings.Join(strings.Split("- "+c.raw.String(), "\n"), "\n- "))

	lines := parseConfig(strings.NewReader(c.raw.String()), key, value, cb)

	c.raw = strings.Builder{}
	c.raw.WriteString(strings.Join(lines, "\n"))
	c.raw.WriteString("\n")

	debug.V(3).Log("output: \n--------------\n%s\n--------------\n", strings.Join(strings.Split("+ "+c.raw.String(), "\n"), "\n+ "))

	return c.flushRaw()
}

// Write persists the current in-memory representation of the config to disk.
//
// Set and Unset already write to disk implicitly, so Write is only needed to
// persist changes that were made while NoWrites was set, or to retry a write
// that previously failed. It is a no-op if NoWrites is set or the config has
// no path.
//
// Errors:
// - Returns error if the config directory or file cannot be written
//
// Example:
//
//	cfg, err := gitconfig.LoadConfig(".git/config")
//	if err != nil {
//	  log.Fatal(err)
//	}
//	cfg.NoWrites = true // modify in memory only
//	if err := cfg.Set("core.pager", "less"); err != nil {
//	  log.Fatal(err)
//	}
//	cfg.NoWrites = false
//	if err := cfg.Write(); err != nil { // persist the change
//	  log.Fatal(err)
//	}
func (c *Config) Write() error {
	return c.flushRaw()
}

func (c *Config) flushRaw() error {
	if c.NoWrites || c.path == "" {
		debug.V(3).Log("not writing changes to disk (noWrites %t, path %q)", c.NoWrites, c.path)

		return nil
	}

	if err := os.MkdirAll(filepath.Dir(c.path), 0o700); err != nil {
		return fmt.Errorf("%w: %s: %w", ErrCreateConfigDir, filepath.Dir(c.path), err)
	}

	debug.V(3).Log("writing config to %s: \n--------------\n%s\n--------------", c.path, c.raw.String())

	if err := os.WriteFile(c.path, []byte(c.raw.String()), 0o600); err != nil {
		return fmt.Errorf("%w: %s: %w", ErrWriteConfig, c.path, err)
	}

	debug.V(1).Log("wrote config to %s", c.path)

	return nil
}

type parseFunc func(fqkn, skn, value, comment, fullLine string) (newLine string, skipLine bool)

// parseConfig implements a simple parser for the gitconfig subset we support.
// The idea is to save all lines unaltered so we can reproduce the config
// almost exactly. Then we skip comments and extract section and subsection
// header. The next steps depend on the mode. Either we want to extract the
// values when loading (key and value empty, parseFunc adds the key-value pairs
// to the vars map), update a key (key is the target key, value the new value)
// or delete a key (parseFunc returns skip).
func parseConfig(in io.Reader, key, value string, cb parseFunc) []string {
	wSection, wSubsection, wKey := splitKey(key)

	s := bufio.NewScanner(in)

	lines := make([]string, 0, 128)
	var section string
	var subsection string
	for s.Scan() {
		fullLine := s.Text()
		for hasLineContinuation(fullLine) && s.Scan() {
			fullLine = strings.TrimSuffix(fullLine, `\`) + s.Text()
		}

		lines = append(lines, fullLine)

		line := strings.TrimSpace(fullLine)
		// Handle full-line comments
		if strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, ";") {
			continue
		}

		// Handle section headers
		if strings.HasPrefix(line, "[") {
			s, subs, skip := parseSectionHeader(line)
			if skip {
				continue
			}
			section = s
			subsection = subs

			continue
		}

		if key != "" && (!strings.EqualFold(section, wSection) || subsection != wSubsection) {
			continue
		}

		// Reference: https://git-scm.com/docs/git-config#_syntax.
		k, v, found := strings.Cut(line, "=")
		var bareComment string
		var isBare bool
		// This is a special case for bare booleans ("name" without "=", short-hand for boolean "true").
		if !found && !strings.HasPrefix(line, "[") && strings.TrimSpace(line) != "" {
			if idx := strings.IndexAny(k, "#;"); idx >= 0 {
				bareComment = " " + k[idx:]
				k = k[:idx]
			}
			v = "true"
			found = true
			isBare = true
		}
		if !found {
			debug.V(3).Log("no valid KV-pair on line: %q", line)

			continue
		}
		// Remove whitespace from key and value that might be around the '='
		// "Whitespace characters surrounding name, = and value are discarded."
		// https://git-scm.com/docs/git-config#_syntax
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)

		// keep a copy of the original key for serialization.
		ok := k
		// "The variable names are case-insensitive"
		k = strings.ToLower(k)

		if !reValidKey.MatchString(k) {
			debug.V(3).Log("invalid key %q in line: %q", k, line)

			continue
		}

		fKey := section + "."
		if subsection != "" {
			fKey += subsection + "."
		}
		fKey += k
		if key == "" {
			wKey = ok
		}

		// extract possible comment from the value
		oValue, comment := splitValueComment(v)
		if isBare && comment == "" {
			comment = bareComment
		}

		// unescape value
		if !CompatMode {
			oValue = unescapeValue(oValue)
		}

		if key != "" && canonicalizeKey(key) != canonicalizeKey(fKey) {
			continue
		}
		if key != "" {
			oValue = value
		}

		newLine, skip := cb(fKey, wKey, oValue, comment, fullLine)
		debug.V(3).Log("parsed line: %q -> %q, skip: %t", fullLine, newLine, skip)
		if skip {
			// remove the last line
			lines = lines[:len(lines)-1]

			continue
		}
		lines[len(lines)-1] = newLine
	}

	return lines
}

func hasLineContinuation(line string) bool {
	if !strings.HasSuffix(line, `\`) {
		return false
	}
	var inQuotes bool
	var escaped bool
	for i := range len(line) {
		ch := line[i]
		if escaped {
			escaped = false

			continue
		}
		if ch == '\\' {
			if i == len(line)-1 {
				return true
			}
			escaped = true

			continue
		}
		if ch == '"' {
			inQuotes = !inQuotes

			continue
		}
		if !inQuotes && (ch == '#' || ch == ';') {
			return false
		}
	}

	return false
}

func hasEqualsBeforeComment(line string) bool {
	for i := range len(line) {
		switch line[i] {
		case '=':
			return true
		case '#', ';':
			return false
		}
	}

	return false
}

// splitValueComment separates a config value from any trailing comment.
// Handles unquoted values, partially or fully double-quoted values, escapes,
// and trailing # or ; comments.
// Returns the value (with surrounding/mid-value quotes stripped, preserving
// escape sequences for unescapeValue) and the comment portion (including # or ;).
func splitValueComment(rValue string) (string, string) {
	if rValue == "" {
		return "", ""
	}

	var val strings.Builder
	var pendingSpace strings.Builder
	var inQuotes bool
	var seenContent bool

	for i := 0; i < len(rValue); i++ {
		ch := rValue[i]

		if ch == '\\' {
			if pendingSpace.Len() > 0 {
				if seenContent {
					val.WriteString(pendingSpace.String())
				}
				pendingSpace.Reset()
			}
			seenContent = true
			val.WriteByte('\\')
			if i+1 < len(rValue) {
				i++
				val.WriteByte(rValue[i])
			}

			continue
		}

		if ch == '"' {
			if !inQuotes && pendingSpace.Len() > 0 {
				if seenContent {
					val.WriteString(pendingSpace.String())
				}
				pendingSpace.Reset()
			}
			inQuotes = !inQuotes
			seenContent = true

			continue
		}

		if !inQuotes {
			if ch == '#' || ch == ';' {
				return val.String(), " " + rValue[i:]
			}
			if ch == ' ' || ch == '\t' {
				if seenContent {
					pendingSpace.WriteByte(ch)
				}

				continue
			}
		}

		if pendingSpace.Len() > 0 {
			if seenContent {
				val.WriteString(pendingSpace.String())
			}
			pendingSpace.Reset()
		}
		seenContent = true
		val.WriteByte(ch)
	}

	return val.String(), ""
}

// unescapeValue processes escape sequences in configuration values in a single pass.
// Supports: \\, \", \n (newline), \t (tab), \b (backspace).
// Other escape sequences (including octal) are preserved unchanged.
func unescapeValue(value string) string {
	if !strings.Contains(value, `\`) {
		return value
	}

	var b strings.Builder
	b.Grow(len(value))
	for i := 0; i < len(value); i++ {
		if value[i] == '\\' && i+1 < len(value) {
			i++
			switch value[i] {
			case '\\':
				b.WriteByte('\\')
			case '"':
				b.WriteByte('"')
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'b':
				b.WriteByte('\b')
			default:
				b.WriteByte('\\')
				b.WriteByte(value[i])
			}

			continue
		}
		b.WriteByte(value[i])
	}

	return b.String()
}

// NewFromMap allows creating a new preset config from a map.
func NewFromMap(data map[string]string) *Config {
	c := &Config{
		readonly: true,
		vars:     make(map[string][]string, len(data)),
	}

	for k, v := range data {
		c.vars[canonicalizeKey(k)] = []string{v}
	}

	return c
}

// LoadConfig tries to load a gitconfig from the given path.
func LoadConfig(fn string) (*Config, error) {
	return loadConfigs(fn, "")
}

// LoadConfigWithWorkdir tries to load a gitconfig from the given path and
// a workdir. The workdir is used to resolve relative paths in the config.
func LoadConfigWithWorkdir(fn, workdir string) (*Config, error) {
	c, err := loadConfigs(fn, workdir)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// resolveGitDir resolves the per-worktree gitDir and shared commonDir for workdir.
// It supports standard .git directories, bare/direct .git paths, and .git files
// pointing to a worktree or submodule gitdir ("gitdir: <path>").
func resolveGitDir(workdir string) (gitDir, commonDir string) { //nolint:nonamedreturns
	if workdir == "" {
		return "", ""
	}

	dotGit := filepath.Join(workdir, ".git")
	fi, err := os.Stat(dotGit)
	if err != nil {
		// Check if workdir itself is a git directory (contains HEAD or config)
		if _, errHead := os.Stat(filepath.Join(workdir, "HEAD")); errHead == nil {
			return workdir, resolveCommonDir(workdir)
		}

		return "", ""
	}

	if fi.IsDir() {
		return dotGit, resolveCommonDir(dotGit)
	}

	// .git is a regular file: parse "gitdir: <path>"
	content, err := os.ReadFile(dotGit)
	if err != nil {
		return "", ""
	}
	line := strings.TrimSpace(string(content))
	target, ok := strings.CutPrefix(line, "gitdir:")
	if !ok {
		return "", ""
	}
	target = strings.TrimSpace(target)
	if !filepath.IsAbs(target) {
		target = filepath.Clean(filepath.Join(workdir, target))
	}

	return target, resolveCommonDir(target)
}

func resolveCommonDir(gitDir string) string {
	commonDirFile := filepath.Join(gitDir, "commondir")
	data, err := os.ReadFile(commonDirFile)
	if err != nil {
		return gitDir
	}
	rel := strings.TrimSpace(string(data))
	if rel == "" {
		return gitDir
	}
	if !filepath.IsAbs(rel) {
		return filepath.Clean(filepath.Join(gitDir, rel))
	}

	return filepath.Clean(rel)
}

func readGitBranch(workdir string) string {
	gitDir, _ := resolveGitDir(workdir)
	if gitDir == "" {
		return ""
	}

	headFile := filepath.Join(gitDir, "HEAD")
	content, err := os.ReadFile(headFile)
	if err != nil {
		return ""
	}

	// content is like "ref: refs/heads/main"
	if branch, found := strings.CutPrefix(string(content), "ref: refs/heads/"); found {
		return strings.TrimSpace(branch)
	}

	return "" // detached HEAD or other cases
}

// getEffectiveIncludes returns all include paths from the config, combining
// basic [include] directives with conditional [includeIf] directives.
// The workdir parameter is used to evaluate conditional includes and
// remoteURLs holds the remote URLs used for hasconfig conditions.
func getEffectiveIncludes(c *Config, workdir string, remoteURLs []string) ([]string, bool) {
	includePaths, includeExists := c.GetAll("include.path")

	if cIncludes := getConditionalIncludes(c, workdir, remoteURLs); len(cIncludes) > 0 {
		includePaths = append(includePaths, cIncludes...)
		includeExists = true
	}

	return includePaths, includeExists
}

// getConditionalIncludes processes [includeIf "condition"] directives and returns
// paths that match the current environment.
// Supported conditions:
//   - gitdir:<pattern> - Include if git directory matches pattern (case-sensitive)
//   - gitdir/i:<pattern> - Include if git directory matches pattern (case-insensitive)
//   - onbranch:<pattern> - Include if the current branch matches pattern
//   - hasconfig:remote.*.url:<pattern> - Include if any remote URL matches pattern
func getConditionalIncludes(c *Config, workdir string, remoteURLs []string) []string {
	candidates := []string{}
	for k := range c.vars {
		debug.V(3).Log("found config key: %q", k)
		// must have the form includeIf.<condition>.path
		// e.g. includeIf."gitdir:/path/to/group/".path
		// see https://git-scm.com/docs/git-config#_conditional_includes
		if !strings.HasPrefix(k, "includeif.") || !strings.HasSuffix(k, ".path") {
			continue
		}
		candidates = append(candidates, k)
	}

	out := make([]string, 0, len(candidates))
	for _, k := range filterCandidates(candidates, workdir, c, remoteURLs) {
		path, found := c.GetAll(k)
		if !found {
			debug.V(3).Log("skipping include candidate %q, no path found", k)

			continue
		}
		out = append(out, path...)
	}

	return out
}

// filterCandidates filters the candidates for include paths.
// See matchSubSection for the supported conditions.
func filterCandidates(candidates []string, workdir string, c *Config, remoteURLs []string) []string {
	out := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		sec, subsec, key := splitKey(candidate)
		if sec != "includeif" || subsec == "" || key != "path" {
			debug.V(3).Log("skipping invalid include candidate %q", candidate)

			continue
		}

		if matchSubSection(subsec, workdir, c, remoteURLs) {
			out = append(out, candidate)
		}
	}

	return out
}

// matchSubSection determines if a subsection condition matches the current environment.
// Handles gitdir, gitdir/i, onbranch, hasconfig:remote.*.url and other condition types.
// Returns true if the condition matches and the config should be included.
func matchSubSection(subsec, workdir string, c *Config, remoteURLs []string) bool {
	if strings.HasPrefix(subsec, "gitdir:") || strings.HasPrefix(subsec, "gitdir/i:") {
		if workdir == "" {
			return false
		}
		caseInsensitive := strings.HasPrefix(subsec, "gitdir/i:")
		_, dir, ok := strings.Cut(subsec, ":")
		if !ok || dir == "" {
			return false
		}

		var exactMatch bool
		if caseInsensitive {
			exactMatch = strings.EqualFold(strings.TrimSuffix(workdir, "/"), strings.TrimSuffix(dir, "/"))
		} else {
			exactMatch = strings.TrimSuffix(workdir, "/") == strings.TrimSuffix(dir, "/")
		}

		if exactMatch || prefixMatch(workdir, dir, caseInsensitive) {
			return true
		}

		var cfgPath string
		if c != nil {
			cfgPath = c.path
		}
		if matchGitDirPattern(dir, workdir, cfgPath, caseInsensitive) {
			return true
		}

		debug.V(3).Log("skipping include candidate, no match for subsec %q and workdir %q", subsec, workdir)

		return false
	}

	if strings.HasPrefix(subsec, "onbranch:") {
		p := strings.SplitN(subsec, ":", 2)
		branchPattern := p[1]
		if c.branch == "" {
			return false
		}

		match, err := globMatch(branchPattern, c.branch)
		if err != nil {
			debug.V(1).Log("invalid glob pattern in onbranch: %s", err)

			return false
		}
		if match {
			return true
		}

		return false
	}

	if strings.HasPrefix(subsec, hasConfigRemoteURLPrefix) {
		pattern := strings.TrimPrefix(subsec, hasConfigRemoteURLPrefix)
		for _, u := range remoteURLs {
			match, err := globMatch(pattern, u)
			if err != nil {
				debug.V(1).Log("invalid glob pattern in hasconfig:remote.*.url: %s", err)

				return false
			}
			if match {
				return true
			}
		}

		debug.V(3).Log("skipping include candidate, no remote URL matches %q", pattern)

		return false
	}

	debug.V(3).Log("skipping unsupported include candidate %q", subsec)

	return false
}

func matchGitDirPattern(pattern, workdir, configPath string, caseInsensitive bool) bool {
	pat := filepath.ToSlash(pattern)
	trailingSlash := strings.HasSuffix(pat, "/")

	switch {
	case strings.HasPrefix(pat, "~/"):
		home := userHomeDir()
		if home == "" {
			return false
		}
		pat = filepath.ToSlash(filepath.Join(home, pat[2:]))
		if trailingSlash && !strings.HasSuffix(pat, "/") {
			pat += "/"
		}
	case strings.HasPrefix(pat, "./"):
		if configPath == "" {
			return false
		}
		baseDir := filepath.Dir(configPath)
		pat = filepath.ToSlash(filepath.Join(baseDir, pat[2:]))
		if trailingSlash && !strings.HasSuffix(pat, "/") {
			pat += "/"
		}
	case !strings.HasPrefix(pat, "/") && !strings.HasPrefix(pat, "**/") && filepath.VolumeName(pattern) == "":
		pat = "**/" + pat
	}

	if strings.HasSuffix(pat, "/") {
		pat += "**"
	}

	cleanWorkdir := filepath.ToSlash(filepath.Clean(workdir))
	candidates := []string{cleanWorkdir}
	if !strings.HasSuffix(cleanWorkdir, "/.git") {
		candidates = append(candidates, cleanWorkdir+"/.git")
	}

	if caseInsensitive {
		pat = strings.ToLower(pat)
	}

	for _, cand := range candidates {
		target := cand
		if caseInsensitive {
			target = strings.ToLower(target)
		}
		if ok, err := globMatch(pat, target); err == nil && ok {
			return true
		}
	}

	return false
}

// hasConfigRemoteURLPrefix is the prefix of the hasconfig:remote.*.url
// conditional include. See
// https://git-scm.com/docs/git-config#_conditional_includes.
const hasConfigRemoteURLPrefix = "hasconfig:remote.*.url:"

// remoteURLsFromConfig returns all remote.<name>.url values of the given config.
func remoteURLsFromConfig(c *Config) []string {
	if c == nil {
		return nil
	}

	urls := make([]string, 0, 4)
	for k, vs := range c.vars {
		sec, sub, key := splitKey(k)
		if sec == "remote" && sub != "" && key == "url" {
			urls = append(urls, vs...)
		}
	}

	return urls
}

// configNeedsRemoteURLs reports whether the config may contain
// hasconfig:remote.*.url conditions, either directly or through includes.
func configNeedsRemoteURLs(c *Config) bool {
	if c == nil {
		return false
	}

	for k := range c.vars {
		if strings.HasPrefix(k, "includeif.") || k == "include.path" {
			return true
		}
	}

	return false
}

// collectRemoteURLsIfNeeded is like collectRemoteURLs but skips configs that
// cannot contain hasconfig:remote.*.url conditions, i.e. configs without any
// include directives.
func collectRemoteURLsIfNeeded(fn string) []string {
	c, err := loadConfig(fn)
	if err != nil || !configNeedsRemoteURLs(c) {
		return nil
	}

	return collectRemoteURLs(fn)
}

// collectRemoteURLs returns all remote.<name>.url values found in the given
// config file and all files it includes, regardless of any includeIf
// conditions. This mirrors git, which collects all remote URLs before it
// evaluates hasconfig:remote.*.url conditions.
func collectRemoteURLs(fn string) []string {
	urls := []string{}
	seen := map[string]struct{}{}
	queue := []string{fn}

	for len(queue) > 0 {
		head := queue[0]
		queue = queue[1:]

		if _, ok := seen[head]; ok {
			continue
		}
		seen[head] = struct{}{}

		c, err := loadConfig(head)
		if err != nil {
			debug.V(3).Log("failed to load config %q while collecting remote URLs: %s", head, err)

			continue
		}

		urls = append(urls, remoteURLsFromConfig(c)...)

		// Collect the paths of all includes, conditional or not. Git also
		// reads all conditionally included files when collecting remote URLs.
		includes, _ := c.GetAll("include.path")
		for k := range c.vars {
			if !strings.HasPrefix(k, "includeif.") || !strings.HasSuffix(k, ".path") {
				continue
			}
			if vs, ok := c.GetAll(k); ok {
				includes = append(includes, vs...)
			}
		}

		queue = append(queue, getPathsForNestedConfig(includes, head)...)
	}

	return urls
}

// prefixMatch checks if a path matches a prefix pattern, with optional case-folding.
// This is used for gitdir: and gitdir/i: conditional includes.
// The fold parameter controls case-insensitive matching.
func prefixMatch(path, prefix string, fold bool) bool {
	if !strings.HasSuffix(prefix, "/") {
		return false
	}
	if fold {
		return strings.HasPrefix(strings.ToLower(path), strings.ToLower(prefix))
	}

	return strings.HasPrefix(path, prefix)
}

// loadConfigs loads a config file and recursively processes all include directives.
// This is the main entry point for loading configs with include support.
// Returns the merged configuration from all included files.
func loadConfigs(fn, workdir string) (*Config, error) {
	return loadConfigsWithRemoteURLs(fn, workdir, collectRemoteURLsIfNeeded(fn))
}

// loadConfigsWithRemoteURLs is like loadConfigs but uses the given remote URLs
// to evaluate hasconfig:remote.*.url conditional includes. Callers that load
// multiple config scopes can collect the remote URLs of all scopes first, so
// that a condition in one scope can match a remote defined in another one.
func loadConfigsWithRemoteURLs(fn, workdir string, remoteURLs []string) (*Config, error) {
	c, err := loadConfig(fn)
	if err != nil {
		return nil, err
	}
	c.path = fn
	c.branch = readGitBranch(workdir)

	loadedConfigs := map[string]struct{}{
		fn: {},
	}
	configsToLoad := []string{}

	includePaths, includeExists := getEffectiveIncludes(c, workdir, remoteURLs)
	if includeExists {
		configsToLoad = append(configsToLoad, getPathsForNestedConfig(includePaths, c.path)...)
	}

	// load all nested configs
	// this is using a slice as a stack because when we load a config
	// it may include other configs
	// so we need to load them in the order they are found.
	for len(configsToLoad) > 0 {
		head := configsToLoad[0]
		configsToLoad = configsToLoad[1:]

		// check if we already loaded this config
		// this is needed to avoid infinite loops when loading nested configs
		_, ignore := loadedConfigs[head]
		if ignore {
			debug.V(3).Log("skipping already loaded config %q", head)

			continue
		}

		debug.V(2).Log("loading nested config %q", head)
		nc, err := loadConfig(head)
		if err != nil {
			return nil, err
		}

		c = mergeConfigs(c, nc)
		loadedConfigs[head] = struct{}{}

		includePaths, includeExists := getEffectiveIncludes(nc, workdir, remoteURLs)
		if includeExists {
			configsToLoad = append(configsToLoad, getPathsForNestedConfig(includePaths, nc.path)...)
		}
	}

	return c, nil
}

// loadConfig loads a single config file without processing includes.
// This is used internally by loadConfigs to load individual files.
func loadConfig(fn string) (*Config, error) {
	fh, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	defer fh.Close() //nolint:errcheck

	c := ParseConfig(fh)
	c.path = fn

	return c, nil
}

// mergeConfigs merge two configs, using first config as a base config extending it with vars, raw fields from the latter.
func mergeConfigs(base *Config, extension *Config) *Config {
	newConfig := Config{
		path:     base.path,
		readonly: base.readonly,
		NoWrites: base.NoWrites,
		raw:      strings.Builder{},
		vars:     map[string][]string{},
	}
	newConfig.raw.WriteString(base.raw.String())
	// Note: We can not append the included config raw to the base config raw, because it will
	// write the included config to the base config file when we write the base config.

	// populate the new config with the base config
	maps.Copy(newConfig.vars, base.vars)

	for k, v := range extension.vars {
		_, existing := newConfig.vars[k]
		if !existing {
			newConfig.vars[k] = []string{}
		}
		newConfig.vars[k] = append(newConfig.vars[k], v...)
	}

	return &newConfig
}

// getPathsForNestedConfig tries to convert paths of nested configs ('/absolute', '~/from/home', 'relative/to/base') to absolute paths.
func getPathsForNestedConfig(nestedConfigs []string, baseConfig string) []string {
	absolutePaths := []string{}
	for _, nc := range nestedConfigs {
		if path.IsAbs(nc) {
			absolutePaths = append(absolutePaths, nc)

			continue
		}
		if strings.HasPrefix(nc, "~/") {
			home := userHomeDir()
			if home == "" {
				// cannot resolve home directory
				debug.V(3).Log("cannot resolve home directory, skipping %q", nc)

				continue
			}
			absolutePaths = append(absolutePaths, path.Join(home, strings.Replace(nc, "~/", "", 1)))

			continue
		}
		absolutePaths = append(absolutePaths, path.Clean(path.Join(path.Dir(baseConfig), nc)))
	}

	return absolutePaths
}

// ParseConfig will try to parse a gitconfig from the given io.Reader. It never fails.
// Invalid configs will be silently rejected.
func ParseConfig(r io.Reader) *Config {
	c := &Config{
		vars: make(map[string][]string, 42),
	}

	lines := parseConfig(r, "", "", func(fk, k, v, comment, fullLine string) (string, bool) {
		fk = canonicalizeKey(fk)
		c.vars[fk] = append(c.vars[fk], v)
		if !hasEqualsBeforeComment(fullLine) {
			return formatKeyValue(k, "", comment), false
		}

		return formatKeyValue(k, v, comment), false
	})

	c.raw.WriteString(strings.Join(lines, "\n"))
	c.raw.WriteString("\n")

	debug.V(3).Log("processed config: %s\nvars: %+v", c.raw.String(), c.vars)

	return c
}

// LoadConfigFromEnv will try to parse an overlay config from the environment variables.
// It supports both <envPrefix>_PARAMETERS (like GIT_CONFIG_PARAMETERS) and
// <envPrefix>_COUNT / <envPrefix>_KEY_<i> / <envPrefix>_VALUE_<i>.
// If no environment variables are set the resulting config will be valid but empty.
// Either way it will not be writeable.
func LoadConfigFromEnv(envPrefix string) *Config {
	c := &Config{
		NoWrites: true,
		vars:     make(map[string][]string),
	}

	if params := os.Getenv(envPrefix + "_PARAMETERS"); params != "" {
		parseConfigParameters(params, c)
	}

	if countStr, hasCount := os.LookupEnv(envPrefix + "_COUNT"); hasCount && countStr != "" {
		count, err := strconv.Atoi(countStr)
		if err != nil || count < 0 {
			return &Config{
				NoWrites: true,
			}
		}

		for i := range count {
			keyVar := fmt.Sprintf("%s%d", envPrefix+"_KEY_", i)
			key := os.Getenv(keyVar)

			valVar := fmt.Sprintf("%s%d", envPrefix+"_VALUE_", i)
			value, found := os.LookupEnv(valVar)

			if key == "" || !found {
				return &Config{
					NoWrites: true,
				}
			}

			ck := canonicalizeKey(key)
			c.vars[ck] = append(c.vars[ck], value)
			debug.V(3).Log("added %s from env", ck)
		}
	}

	if len(c.vars) == 0 {
		c.vars = nil
	}

	return c
}

// parseConfigParameters parses GIT_CONFIG_PARAMETERS-formatted strings.
func parseConfigParameters(params string, c *Config) {
	s := strings.TrimSpace(params)
	for len(s) > 0 {
		s = strings.TrimLeft(s, " \t")
		if len(s) == 0 {
			break
		}

		var token strings.Builder
		for len(s) > 0 && s[0] != ' ' && s[0] != '\t' {
			if s[0] == '\'' {
				s = s[1:]
				for len(s) > 0 && s[0] != '\'' {
					token.WriteByte(s[0])
					s = s[1:]
				}
				if len(s) > 0 && s[0] == '\'' {
					s = s[1:]
				}
				if strings.HasPrefix(s, `\'`) {
					token.WriteByte('\'')
					s = s[2:]
				}

				continue
			}
			token.WriteByte(s[0])
			s = s[1:]
		}

		item := token.String()
		if item == "" {
			continue
		}
		k, v, hasEq := strings.Cut(item, "=")
		ck := canonicalizeKey(strings.TrimSpace(k))
		if ck == "" {
			continue
		}
		if !hasEq {
			v = "true"
		}
		c.vars[ck] = append(c.vars[ck], v)
	}
}

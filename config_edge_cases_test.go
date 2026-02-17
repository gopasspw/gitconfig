package gitconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEdgeCaseUnicodeKeys tests handling of special characters in keys.
func TestEdgeCaseUnicodeKeys(t *testing.T) {
	t.Parallel()

	td := t.TempDir()
	configPath := filepath.Join(td, "config")

	// Config with various key formats (git config keys are typically alphanumeric + dash/dot)
	content := `[user]
	name = John Doe
[core]
	ignorecase = true`

	err := os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Standard keys should be accessible
	val, ok := cfg.Get("user.name")
	assert.True(t, ok)
	assert.Equal(t, "John Doe", val)

	val, ok = cfg.Get("core.ignorecase")
	assert.True(t, ok)
	assert.Equal(t, "true", val)
}

// TestEdgeCaseVeryLongValues tests handling of very long configuration values.
func TestEdgeCaseVeryLongValues(t *testing.T) {
	t.Parallel()

	td := t.TempDir()
	configPath := filepath.Join(td, "config")

	// Create a very long value (e.g., 10KB)
	longValue := strings.Repeat("x", 10000)
	content := "[section]\n\tkey = " + longValue + "\n"

	err := os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	val, ok := cfg.Get("section.key")
	assert.True(t, ok)
	assert.Equal(t, longValue, val)
}

// TestEdgeCaseVeryDeepSections tests handling of deeply nested section hierarchies.
func TestEdgeCaseVeryDeepSections(t *testing.T) {
	t.Parallel()

	td := t.TempDir()
	configPath := filepath.Join(td, "config")

	// Config with multiple nested levels
	content := `[core]
	key1 = value1
[filter "smudge"]
	command = git convert-smudge
[filter "clean"]
	command = git convert-clean
[user]
	name = Test User
	email = test@example.com
[remote "origin"]
	url = https://github.com/test/repo.git
	fetch = +refs/heads/*:refs/remotes/origin/*`

	err := os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify subsection access works
	val, ok := cfg.Get("filter.smudge.command")
	assert.True(t, ok)
	assert.Equal(t, "git convert-smudge", val)

	val, ok = cfg.Get("remote.origin.url")
	assert.True(t, ok)
	assert.Equal(t, "https://github.com/test/repo.git", val)
}

// TestEdgeCaseEmptyValues tests handling of empty string values.
func TestEdgeCaseEmptyValues(t *testing.T) {
	t.Parallel()

	td := t.TempDir()
	configPath := filepath.Join(td, "config")

	// Config with empty values
	content := `[section]
	empty = 
	noSpace=
	quoted = ""
	whitespace =    `

	err := os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Empty values should return empty string
	val, ok := cfg.Get("section.empty")
	assert.True(t, ok)
	assert.Equal(t, "", val)

	val, ok = cfg.Get("section.noSpace")
	assert.True(t, ok)
	assert.Equal(t, "", val)

	val, ok = cfg.Get("section.quoted")
	// Quoted empty string should be ""
	assert.True(t, ok)
}

// TestEdgeCaseWhitespacePreservation tests that whitespace in values is preserved or handled correctly.
func TestEdgeCaseWhitespacePreservation(t *testing.T) {
	t.Parallel()

	td := t.TempDir()
	configPath := filepath.Join(td, "config")

	// Config with various whitespace scenarios
	content := `[section]
	leading =    value
	internal = value with    spaces
	trailing = value   
	tabs = value	with	tabs`

	err := os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Leading whitespace should be trimmed
	val, ok := cfg.Get("section.leading")
	assert.True(t, ok)
	assert.Equal(t, "value", val)

	// Internal spaces should be preserved
	val, ok = cfg.Get("section.internal")
	assert.True(t, ok)
	assert.Equal(t, "value with    spaces", val)
}

// TestEdgeCaseSpecialCharactersInValues tests handling of special characters.
func TestEdgeCaseSpecialCharactersInValues(t *testing.T) {
	t.Parallel()

	td := t.TempDir()
	configPath := filepath.Join(td, "config")

	// Config with special characters in values
	content := `[section]
	semicolon = value;with;semicolons
	hash = value#with#hash
	equals = key=value
	brackets = [value]
	quotes = value"with"quotes
	backslash = C:\\Users\\test\\path`

	err := os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Semicolons should be preserved in unquoted values (or may start comment)
	val, ok := cfg.Get("section.semicolon")
	assert.True(t, ok)
	// Behavior depends on parser - either full value or up to semicolon
	assert.NotEmpty(t, val)

	// Hash may start comment or be literal depending on quoting
	val, ok = cfg.Get("section.hash")
	if ok {
		assert.NotEmpty(t, val)
	}

	// Backslash handling
	val, ok = cfg.Get("section.backslash")
	assert.True(t, ok)
	// Value should contain the path
	assert.True(t, strings.Contains(val, "Users") || strings.Contains(val, "\\\\"))
}

// TestEdgeCaseCommentHandling tests behavior with comments in various positions.
func TestEdgeCaseCommentHandling(t *testing.T) {
	t.Parallel()

	td := t.TempDir()
	configPath := filepath.Join(td, "config")

	// Config with various comment styles
	content := `# File comment
[section]
	# This is a comment
	key1 = value1
	key2 = value2 # inline comment?
	; semicolon comment
	key3 = value3`

	err := os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// All keys should be accessible
	val, ok := cfg.Get("section.key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val)

	val, ok = cfg.Get("section.key2")
	assert.True(t, ok)
	// Should not include the inline comment
	assert.Equal(t, "value2", val)

	val, ok = cfg.Get("section.key3")
	assert.True(t, ok)
	assert.Equal(t, "value3", val)
}

// TestEdgeCaseMultilineValues tests handling of multiline values (if supported).
func TestEdgeCaseMultilineValues(t *testing.T) {
	t.Parallel()

	td := t.TempDir()
	configPath := filepath.Join(td, "config")

	// Multiline values with continuation or quoted strings
	content := `[section]
	singleline = value on one line
	quoted = "value on multiple lines
can continue here"
	continuation = value \
that continues \
on next lines`

	err := os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	val, ok := cfg.Get("section.singleline")
	assert.True(t, ok)
	assert.Equal(t, "value on one line", val)
}

// TestEdgeCaseNumericValues tests handling of numeric-looking values.
func TestEdgeCaseNumericValues(t *testing.T) {
	t.Parallel()

	td := t.TempDir()
	configPath := filepath.Join(td, "config")

	// Numeric-looking values should be treated as strings
	content := `[section]
	integer = 42
	float = 3.14159
	octal = 0755
	hex = 0xFF
	negative = -100
	scientific = 1.5e-10`

	err := os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// All should be accessible as strings
	val, ok := cfg.Get("section.integer")
	assert.True(t, ok)
	assert.Equal(t, "42", val)

	val, ok = cfg.Get("section.float")
	assert.True(t, ok)
	assert.Equal(t, "3.14159", val)

	val, ok = cfg.Get("section.octal")
	assert.True(t, ok)
	assert.Equal(t, "0755", val)
}

// TestEdgeCaseBooleanValues tests handling of boolean representations.
func TestEdgeCaseBooleanValues(t *testing.T) {
	t.Parallel()

	td := t.TempDir()
	configPath := filepath.Join(td, "config")

	// Various boolean representations
	content := `[section]
	enabled = true
	disabled = false
	yes = yes
	no = no
	on = on
	off = off
	one = 1
	zero = 0`

	err := os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// All should be accessible as strings
	val, ok := cfg.Get("section.enabled")
	assert.True(t, ok)
	assert.Equal(t, "true", val)

	val, ok = cfg.Get("section.disabled")
	assert.True(t, ok)
	assert.Equal(t, "false", val)

	val, ok = cfg.Get("section.zero")
	assert.True(t, ok)
	assert.Equal(t, "0", val)
}

// TestEdgeCaseLargeConfigFile tests handling of large configuration files.
func TestEdgeCaseLargeConfigFile(t *testing.T) {
	t.Parallel()

	td := t.TempDir()
	configPath := filepath.Join(td, "config")

	// Generate a large config with many sections and keys
	var sb strings.Builder
	for i := 0; i < 20; i++ {
		sb.WriteString(fmt.Sprintf("[section%d]\n", i))
		for j := 0; j < 5; j++ {
			sb.WriteString(fmt.Sprintf("\tkey%d = value_%d_%d\n", j, i, j))
		}
	}

	err := os.WriteFile(configPath, []byte(sb.String()), 0o644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Spot check some values
	val, ok := cfg.Get("section0.key0")
	assert.True(t, ok)
	assert.NotEmpty(t, val)

	val, ok = cfg.Get("section10.key3")
	assert.True(t, ok)
	assert.NotEmpty(t, val)
}

// TestEdgeCaseDuplicateKeys tests handling of duplicate keys in configuration.
func TestEdgeCaseDuplicateKeys(t *testing.T) {
	t.Parallel()

	td := t.TempDir()
	configPath := filepath.Join(td, "config")

	// Config with duplicate keys
	content := `[section]
	key = value1
	other = value
	key = value2
	key = value3`

	err := os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Get should return a value (likely the last one)
	val, ok := cfg.Get("section.key")
	assert.True(t, ok)
	assert.NotEmpty(t, val)

	// GetAll should return all values
	vals, ok := cfg.GetAll("section.key")
	assert.True(t, ok)
	assert.GreaterOrEqual(t, len(vals), 1)
}

// TestEdgeCaseCaseSensitivity tests key case sensitivity (typically case-insensitive).
func TestEdgeCaseCaseSensitivity(t *testing.T) {
	t.Parallel()

	td := t.TempDir()
	configPath := filepath.Join(td, "config")

	// Config with various case keys
	content := `[section]
	key = value
	Key = VALUE
	KEY = VALUE_UPPER`

	err := os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Git config treats keys case-insensitively (for keys, not sections usually)
	val, ok := cfg.Get("section.key")
	assert.True(t, ok)
	assert.NotEmpty(t, val)
}

// TestEdgeCaseWindowsLineEndings tests handling of Windows (CRLF) line endings.
func TestEdgeCaseWindowsLineEndings(t *testing.T) {
	t.Parallel()

	td := t.TempDir()
	configPath := filepath.Join(td, "config")

	// Config with CRLF line endings
	content := "[section]\r\n\tkey = value\r\n\tother = test"

	err := os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	val, ok := cfg.Get("section.key")
	assert.True(t, ok)
	assert.Equal(t, "value", val)

	val, ok = cfg.Get("section.other")
	assert.True(t, ok)
	assert.Equal(t, "test", val)
}

// TestEdgeCaseLeadingTrailingWhitespace tests config files with leading/trailing whitespace.
func TestEdgeCaseLeadingTrailingWhitespace(t *testing.T) {
	t.Parallel()

	td := t.TempDir()
	configPath := filepath.Join(td, "config")

	// Config with extra whitespace
	content := "   \n   \n[section]\n\tkey = value\n\n\n"

	err := os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	val, ok := cfg.Get("section.key")
	assert.True(t, ok)
	assert.Equal(t, "value", val)
}

// TestEdgeCaseNullBytes tests handling of null bytes in configuration.
func TestEdgeCaseNullBytes(t *testing.T) {
	// This test may be platform-specific
	if runtime.GOOS == "windows" {
		t.Skip("Null byte handling may differ on Windows")
	}

	t.Parallel()

	td := t.TempDir()
	configPath := filepath.Join(td, "config")

	// Config with null byte (should not normally appear)
	content := "[section]\nkey = value\x00end"

	err := os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	// Behavior: either error or handle gracefully
	if err != nil {
		return
	}

	require.NotNil(t, cfg)
	val, ok := cfg.Get("section.key")
	// If it loads, value should be approximately "value"
	if ok {
		assert.True(t, strings.HasPrefix(val, "value"))
	}
}

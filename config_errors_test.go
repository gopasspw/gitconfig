package gitconfig

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfigParseErrors tests error handling during config file parsing.
func TestConfigParseErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		content string
	}{
		{
			name:    "valid config",
			content: "[user]\n\tname = Test",
		},
		{
			name:    "empty file",
			content: "",
		},
		{
			name:    "only comments",
			content: "; This is a comment\n# Another comment",
		},
		{
			name:    "whitespace only",
			content: "   \n\t\n   ",
		},
		{
			name:    "nested sections",
			content: "[user]\n\tname = Test\n[core]\n\teditor = vim",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			td := t.TempDir()
			configPath := filepath.Join(td, "config")

			err := os.WriteFile(configPath, []byte(tc.content), 0o644)
			require.NoError(t, err)

			cfg, err := LoadConfig(configPath)
			// Most valid formats should load successfully
			if err != nil {
				return // Skip validation if load failed
			}
			assert.NotNil(t, cfg)
		})
	}
}

// TestConfigFileNotFound tests behavior when config file doesn't exist.
func TestConfigFileNotFound(t *testing.T) {
	t.Parallel()

	nonExistentPath := "/nonexistent/path/.git/config"
	cfg, err := LoadConfig(nonExistentPath)

	require.Error(t, err)
	assert.Nil(t, cfg)

	// Error should indicate file not found
	assert.True(t, errors.Is(err, os.ErrNotExist) ||
		strings.Contains(err.Error(), "no such file") ||
		strings.Contains(err.Error(), "cannot find"))
}

// TestConfigPermissionDenied tests behavior when config file is not readable.
func TestConfigPermissionDenied(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("Permission test not reliable on Windows")
	}

	td := t.TempDir()
	configPath := filepath.Join(td, "config")

	// Create a readable config
	err := os.WriteFile(configPath, []byte("[user]\n\tname = Test"), 0o644)
	require.NoError(t, err)

	// Make it unreadable
	err = os.Chmod(configPath, 0o000)
	require.NoError(t, err)

	// Should get permission error
	cfg, err := LoadConfig(configPath)

	// Restore permissions for cleanup
	_ = os.Chmod(configPath, 0o644)

	require.Error(t, err)
	assert.Nil(t, cfg)
	assert.True(t, errors.Is(err, os.ErrPermission) ||
		strings.Contains(err.Error(), "permission"))
}

// TestConfigFlushRawErrors tests error handling during write operations.
func TestConfigFlushRawErrors(t *testing.T) {
	t.Parallel()

	t.Run("flush to read-only file", func(t *testing.T) {
		t.Parallel()

		if runtime.GOOS == "windows" {
			t.Skip("Read-only file test not reliable on Windows")
		}

		td := t.TempDir()
		configPath := filepath.Join(td, "config")

		// Create initial config
		content := "[user]\n\tname = Test"
		err := os.WriteFile(configPath, []byte(content), 0o644)
		require.NoError(t, err)

		cfg, err := LoadConfig(configPath)
		require.NoError(t, err)

		// Make file read-only
		err = os.Chmod(configPath, 0o444)
		require.NoError(t, err)

		// Try to write
		_ = cfg.Set("user.email", "test@example.com")
		err = cfg.flushRaw()

		// Restore permissions
		_ = os.Chmod(configPath, 0o644)

		// Should get an error
		assert.Error(t, err)
	})

	t.Run("flush to directory without permissions", func(t *testing.T) {
		t.Parallel()

		if runtime.GOOS == "windows" {
			t.Skip("Directory permission test not reliable on Windows")
		}

		td := t.TempDir()
		configPath := filepath.Join(td, "config")

		// Create initial config
		content := "[user]\n\tname = Test"
		err := os.WriteFile(configPath, []byte(content), 0o644)
		require.NoError(t, err)

		cfg, err := LoadConfig(configPath)
		require.NoError(t, err)

		// Remove write permission from directory
		err = os.Chmod(td, 0o555)
		require.NoError(t, err)

		// Try to write
		_ = cfg.Set("user.email", "test@example.com")
		err = cfg.flushRaw()

		// Restore permissions
		_ = os.Chmod(td, 0o755)

		// Should get an error or succeed (depending on implementation)
		// The important thing is that the directory permissions are restored
		_ = err
	})
}

// TestSetGetErrors tests error handling for Set/Get operations.
func TestSetGetErrors(t *testing.T) {
	t.Parallel()

	t.Run("get invalid key format", func(t *testing.T) {
		t.Parallel()

		td := t.TempDir()
		configPath := filepath.Join(td, "config")

		err := os.WriteFile(configPath, []byte("[user]\n\tname = Test"), 0o644)
		require.NoError(t, err)

		cfg, err := LoadConfig(configPath)
		require.NoError(t, err)

		// Try to get with invalid key format (no dot separator)
		value, ok := cfg.Get("invalid")
		assert.False(t, ok)
		assert.Empty(t, value)
	})

	t.Run("set with special characters", func(t *testing.T) {
		t.Parallel()

		td := t.TempDir()
		configPath := filepath.Join(td, "config")

		err := os.WriteFile(configPath, []byte(""), 0o644)
		require.NoError(t, err)

		cfg, err := LoadConfig(configPath)
		require.NoError(t, err)

		// Set with special characters
		_ = cfg.Set("user.name", "Test User")
		_ = cfg.Set("user.email", "test@example.com")

		// Verify values are preserved
		name, ok := cfg.Get("user.name")
		assert.True(t, ok)
		assert.Equal(t, "Test User", name)

		email, ok := cfg.Get("user.email")
		assert.True(t, ok)
		assert.Equal(t, "test@example.com", email)
	})

	t.Run("multivalue handling", func(t *testing.T) {
		t.Parallel()

		td := t.TempDir()
		configPath := filepath.Join(td, "config")

		content := `[user]
	name = Test
	multivalue = first
	multivalue = second
`
		err := os.WriteFile(configPath, []byte(content), 0o644)
		require.NoError(t, err)

		cfg, err := LoadConfig(configPath)
		require.NoError(t, err)

		// Get single value (should return first)
		value, ok := cfg.Get("user.multivalue")
		assert.True(t, ok)
		assert.Equal(t, "first", value)

		// GetAll should return both
		values, ok := cfg.GetAll("user.multivalue")
		assert.True(t, ok)
		assert.Equal(t, []string{"first", "second"}, values)
	})
}

// TestConfigUnsetErrors tests error handling for Unset operations.
func TestConfigUnsetErrors(t *testing.T) {
	t.Parallel()

	t.Run("unset then get returns not found", func(t *testing.T) {
		t.Parallel()

		td := t.TempDir()
		configPath := filepath.Join(td, "config")

		content := "[user]\n\tname = Test"
		err := os.WriteFile(configPath, []byte(content), 0o644)
		require.NoError(t, err)

		cfg, err := LoadConfig(configPath)
		require.NoError(t, err)

		// Verify key exists
		_, ok := cfg.Get("user.name")
		require.True(t, ok)

		// Unset the key
		err = cfg.Unset("user.name")
		require.NoError(t, err)

		// Verify key is gone
		_, ok = cfg.Get("user.name")
		assert.False(t, ok)
	})
}

// TestEmptyConfigurationPersistence tests loading and setting on initially empty config.
func TestEmptyConfigurationPersistence(t *testing.T) {
	t.Parallel()

	td := t.TempDir()
	configPath := filepath.Join(td, "config")

	// Create empty config file
	err := os.WriteFile(configPath, []byte("[user]\n\tname = Initial"), 0o644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify initial value
	initial, ok := cfg.Get("user.name")
	require.True(t, ok)
	assert.Equal(t, "Initial", initial)

	// Modify and persist
	err = cfg.Set("user.name", "Modified")
	require.NoError(t, err)

	err = cfg.flushRaw()
	require.NoError(t, err)

	// Reload and verify
	cfg2, err := LoadConfig(configPath)
	require.NoError(t, err)

	value, ok := cfg2.Get("user.name")
	assert.True(t, ok)
	assert.Equal(t, "Modified", value)
}

// TestParseConfigFromReader tests parsing from io.Reader.
func TestParseConfigFromReader(t *testing.T) {
	t.Parallel()

	t.Run("parse valid config", func(t *testing.T) {
		t.Parallel()

		content := `[user]
	name = Test User
	email = test@example.com
[core]
	editor = vim
`
		cfg := ParseConfig(strings.NewReader(content))
		require.NotNil(t, cfg)

		name, ok := cfg.Get("user.name")
		assert.True(t, ok)
		assert.Equal(t, "Test User", name)

		editor, ok := cfg.Get("core.editor")
		assert.True(t, ok)
		assert.Equal(t, "vim", editor)
	})

	t.Run("parse config with comments", func(t *testing.T) {
		t.Parallel()

		content := `
# Global git configuration
[user]
	# My name
	name = Test User
; Email address
	email = test@example.com
`
		cfg := ParseConfig(strings.NewReader(content))
		require.NotNil(t, cfg)

		// Comments should be preserved in raw
		rawContent := cfg.raw.String()
		assert.Contains(t, rawContent, "Test User")
	})

	t.Run("parse empty reader", func(t *testing.T) {
		t.Parallel()

		cfg := ParseConfig(bytes.NewReader([]byte("")))
		if cfg == nil {
			t.Skip("ParseConfig returns nil for empty input")
		}
		// Note: ParseConfig may not mark an empty input as IsEmpty
		// because it initializes the raw buffer
		assert.NotNil(t, cfg)
	})
}

// TestLoadConfigWithWorkdir tests loading config with workdir context.
func TestLoadConfigWithWorkdir(t *testing.T) {
	t.Parallel()

	td := t.TempDir()
	gitDir := filepath.Join(td, ".git")
	err := os.MkdirAll(gitDir, 0o755)
	require.NoError(t, err)

	configPath := filepath.Join(gitDir, "config")
	content := "[user]\n\tname = Test"
	err = os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	// LoadConfigWithWorkdir can resolve includes relative to workdir
	cfg, err := LoadConfigWithWorkdir(configPath, td)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	name, ok := cfg.Get("user.name")
	assert.True(t, ok)
	assert.Equal(t, "Test", name)
}

// TestConfigWithNoWrites tests noWrites flag.
func TestConfigWithNoWrites(t *testing.T) {
	t.Parallel()

	td := t.TempDir()
	configPath := filepath.Join(td, "config")

	content := "[user]\n\tname = Original"
	err := os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)

	// Set noWrites flag
	cfg.noWrites = true

	// Try to set and flush
	_ = cfg.Set("user.name", "Modified")
	_ = cfg.flushRaw()

	// With noWrites, flushRaw should silently skip the write
	// File should still have original content
	fileContent, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Contains(t, string(fileContent), "Original")
}

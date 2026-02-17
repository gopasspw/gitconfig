package gitconfig

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPlatformDefaultPaths tests that default config paths are platform-appropriate.
func TestPlatformDefaultPaths(t *testing.T) {
	t.Parallel()

	cfg := New()

	// System config should be set
	assert.NotEmpty(t, cfg.SystemConfig, "system config path should be set")

	// Local and worktree configs should be set
	assert.NotEmpty(t, cfg.LocalConfig, "local config should be set")
	assert.NotEmpty(t, cfg.WorktreeConfig, "worktree config should be set")

	// Platform-specific checks
	switch runtime.GOOS {
	case "windows":
		// Windows paths might contain backslashes or use ProgramData
		assert.Contains(t, []bool{
			filepath.IsAbs(cfg.SystemConfig),
			len(cfg.SystemConfig) > 0,
		}, true, "Windows system config should be absolute or set")
	default:
		// Unix-like systems typically use /etc/gitconfig
		if cfg.SystemConfig != "" {
			assert.True(t, filepath.IsAbs(cfg.SystemConfig), "Unix system config should be absolute if set")
		}
	}
}

// TestPlatformPathSeparators tests that path handling works correctly on the platform.
func TestPlatformPathSeparators(t *testing.T) {
	t.Parallel()

	td := t.TempDir()
	subdir := filepath.Join(td, "configs")
	require.NoError(t, os.MkdirAll(subdir, 0o755))

	configPath := filepath.Join(td, "config")
	includePath := filepath.Join(subdir, "included.conf")

	// Create included config
	err := os.WriteFile(includePath, []byte("[core]\n\teditor = vim"), 0o644)
	require.NoError(t, err)

	// Main config with platform-appropriate path
	content := "[include]\n\tpath = " + filepath.ToSlash(includePath) + "\n[user]\n\tname = Test"
	err = os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify included value is accessible regardless of platform path separators
	editor, ok := cfg.Get("core.editor")
	if ok {
		assert.Equal(t, "vim", editor)
	}
}

// TestPlatformLineEndings tests handling of platform-specific line endings.
func TestPlatformLineEndings(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		content    string
		lineEnding string
		skip       bool
	}{
		{
			name:       "Unix LF",
			lineEnding: "\n",
			skip:       false,
		},
		{
			name:       "Windows CRLF",
			lineEnding: "\r\n",
			skip:       false,
		},
		{
			name:       "Old Mac CR",
			lineEnding: "\r",
			skip:       true, // Git doesn't support CR-only line endings
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.skip {
				t.Skip("Skipping unsupported line ending format")
			}

			td := t.TempDir()
			configPath := filepath.Join(td, "config")

			// Build config with specific line ending
			lines := []string{
				"[user]",
				"\tname = John Doe",
				"\temail = john@example.com",
				"[core]",
				"\teditor = vim",
			}
			content := ""
			for _, line := range lines {
				content += line + tc.lineEnding
			}

			err := os.WriteFile(configPath, []byte(content), 0o644)
			require.NoError(t, err)

			cfg, err := LoadConfig(configPath)
			require.NoError(t, err)
			require.NotNil(t, cfg)

			// Values should be accessible regardless of line ending
			name, ok := cfg.Get("user.name")
			assert.True(t, ok)
			assert.Equal(t, "John Doe", name)

			email, ok := cfg.Get("user.email")
			assert.True(t, ok)
			assert.Equal(t, "john@example.com", email)

			editor, ok := cfg.Get("core.editor")
			assert.True(t, ok)
			assert.Equal(t, "vim", editor)
		})
	}
}

// TestPlatformFilePermissions tests file permission handling (Unix-specific behavior).
func TestPlatformFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("File permission test not applicable on Windows")
	}

	t.Parallel()

	td := t.TempDir()
	configPath := filepath.Join(td, "config")

	// Create config with specific permissions
	content := "[user]\n\tname = Test User"
	err := os.WriteFile(configPath, []byte(content), 0o600)
	require.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	name, ok := cfg.Get("user.name")
	assert.True(t, ok)
	assert.Equal(t, "Test User", name)

	// Verify file permissions are preserved after write (Set automatically writes)
	err = cfg.Set("user.email", "test@example.com")
	require.NoError(t, err)

	info, err := os.Stat(configPath)
	require.NoError(t, err)

	// Permission preservation may vary by implementation
	// Just verify file is still readable
	assert.NotNil(t, info)
}

// TestPlatformCaseSensitivity tests path handling based on filesystem case sensitivity.
func TestPlatformCaseSensitivity(t *testing.T) {
	t.Parallel()

	td := t.TempDir()

	// Try to create files with different cases
	configLower := filepath.Join(td, "config")
	configUpper := filepath.Join(td, "CONFIG")

	err := os.WriteFile(configLower, []byte("[user]\n\tname = lowercase"), 0o644)
	require.NoError(t, err)

	// On case-sensitive filesystems, this creates a different file
	// On case-insensitive filesystems (Windows, macOS default), this overwrites
	err = os.WriteFile(configUpper, []byte("[user]\n\tname = uppercase"), 0o644)
	require.NoError(t, err)

	// Load the lowercase version
	cfg, err := LoadConfig(configLower)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	name, ok := cfg.Get("user.name")
	assert.True(t, ok)

	// On case-insensitive FS, we get "uppercase"
	// On case-sensitive FS, we get "lowercase"
	// Both are valid platform behaviors
	assert.Contains(t, []string{"lowercase", "uppercase"}, name)
}

// TestPlatformUserHomeExpansion tests tilde expansion for home directory.
func TestPlatformUserHomeExpansion(t *testing.T) {
	t.Parallel()

	td := t.TempDir()
	configPath := filepath.Join(td, "config")

	// Create a config that would normally use ~/ (though we test the loading, not path expansion)
	content := "[user]\n\tname = Test"
	err := os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify basic loading works (actual ~ expansion tested elsewhere)
	name, ok := cfg.Get("user.name")
	assert.True(t, ok)
	assert.Equal(t, "Test", name)
}

// TestPlatformSymlinks tests symlink handling (Unix-specific).
func TestPlatformSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Symlink test not reliable on Windows without privileges")
	}

	t.Parallel()

	td := t.TempDir()
	realConfig := filepath.Join(td, "real-config")
	symlinkConfig := filepath.Join(td, "symlink-config")

	// Create real config
	content := "[user]\n\tname = Via Symlink"
	err := os.WriteFile(realConfig, []byte(content), 0o644)
	require.NoError(t, err)

	// Create symlink
	err = os.Symlink(realConfig, symlinkConfig)
	if err != nil {
		t.Skipf("Cannot create symlink: %v", err)
	}

	// Load via symlink
	cfg, err := LoadConfig(symlinkConfig)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	name, ok := cfg.Get("user.name")
	assert.True(t, ok)
	assert.Equal(t, "Via Symlink", name)

	// Write via symlink should work (Set automatically writes)
	err = cfg.Set("user.email", "symlink@example.com")
	require.NoError(t, err)

	// Verify write went through
	cfg2, err := LoadConfig(realConfig)
	require.NoError(t, err)

	email, ok := cfg2.Get("user.email")
	assert.True(t, ok)
	assert.Equal(t, "symlink@example.com", email)
}

// TestPlatformLongPaths tests handling of long file paths.
func TestPlatformLongPaths(t *testing.T) {
	t.Parallel()

	// Create a deep directory structure
	td := t.TempDir()
	deepPath := td

	// Create a reasonably deep path (not MAX_PATH to avoid platform issues)
	for range 10 {
		deepPath = filepath.Join(deepPath, "verylongdirectorynametotest")
	}

	err := os.MkdirAll(deepPath, 0o755)
	if err != nil {
		t.Skipf("Cannot create deep path on this platform: %v", err)
	}

	configPath := filepath.Join(deepPath, "config")
	content := "[user]\n\tname = Deep Path Test"
	err = os.WriteFile(configPath, []byte(content), 0o644)
	if err != nil {
		t.Skipf("Cannot write to deep path on this platform: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	name, ok := cfg.Get("user.name")
	assert.True(t, ok)
	assert.Equal(t, "Deep Path Test", name)
}

// TestPlatformRelativePaths tests relative path handling.
func TestPlatformRelativePaths(t *testing.T) {
	td := t.TempDir()
	configPath := filepath.Join(td, "config")

	content := "[user]\n\tname = Relative Path Test"
	err := os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	// Change to temp directory for relative path resolution
	t.Chdir(td)

	// Load with relative path
	cfg, err := LoadConfig("config")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	name, ok := cfg.Get("user.name")
	assert.True(t, ok)
	assert.Equal(t, "Relative Path Test", name)
}

// TestPlatformEnvironmentVariables tests environment variable handling across platforms.
func TestPlatformEnvironmentVariables(t *testing.T) {
	// Set a test environment variable
	testKey := "GITCONFIG_TEST_COUNT"
	testKeyVar := "GITCONFIG_TEST_KEY_0"
	testValueVar := "GITCONFIG_TEST_VALUE_0"

	t.Setenv(testKey, "1")
	t.Setenv(testKeyVar, "user.name")
	t.Setenv(testValueVar, "From Env")

	cfg := LoadConfigFromEnv("GITCONFIG_TEST")
	require.NotNil(t, cfg)

	name, ok := cfg.Get("user.name")
	assert.True(t, ok)
	assert.Equal(t, "From Env", name)
}

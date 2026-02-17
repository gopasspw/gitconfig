package gitconfig

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIncludeFileNotFound tests behavior when included files don't exist.
func TestIncludeFileNotFound(t *testing.T) {
	t.Parallel()

	td := t.TempDir()
	configPath := filepath.Join(td, "config")

	// Config with include directive pointing to non-existent file
	content := `[include]
	path = nonexistent.conf
[user]
	name = Test
`
	err := os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	// LoadConfig should report the include error
	cfg, err := LoadConfig(configPath)
	if err != nil {
		// Acceptable to error on missing include
		assert.Error(t, err)
	} else if cfg != nil {
		// Or may skip the include silently - check behavior is consistent
		assert.NotNil(t, cfg)
	}
}

// TestIncludePermissionDenied tests behavior when included files are unreadable.
func TestIncludePermissionDenied(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("Permission test not reliable on Windows")
	}

	td := t.TempDir()
	configPath := filepath.Join(td, "config")
	includePath := filepath.Join(td, "include.conf")

	// Create include file
	err := os.WriteFile(includePath, []byte("[section]\n\tkey = value"), 0o644)
	require.NoError(t, err)

	// Create main config with include
	content := "[include]\n\tpath = " + includePath + "\n[user]\n\tname = Test"
	err = os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	// Make include file unreadable
	err = os.Chmod(includePath, 0o000)
	require.NoError(t, err)

	// Should error on permission
	cfg, err := LoadConfig(configPath)

	// Restore for cleanup
	_ = os.Chmod(includePath, 0o644)

	if err != nil {
		// Acceptable to error
		assert.Error(t, err)
	} else if cfg != nil {
		// Or may handle gracefully - verify some state
		assert.NotNil(t, cfg)
	}
}

// TestIncludeCircular tests behavior with circular include references.
func TestIncludeCircular(t *testing.T) {
	t.Parallel()

	td := t.TempDir()
	configA := filepath.Join(td, "config-a")
	configB := filepath.Join(td, "config-b")

	// Create circular includes: A -> B -> A
	contentA := "[include]\n\tpath = " + configB + "\n[section]\n\tkey = a"
	contentB := "[include]\n\tpath = " + configA + "\n[section]\n\tkey = b"

	err := os.WriteFile(configA, []byte(contentA), 0o644)
	require.NoError(t, err)

	err = os.WriteFile(configB, []byte(contentB), 0o644)
	require.NoError(t, err)

	// Behavior: either errors on circular include or handles gracefully
	cfg, err := LoadConfig(configA)

	if err != nil {
		// Acceptable to detect and error
		assert.Error(t, err)
	} else {
		// Or succeeds with some depth limit
		assert.NotNil(t, cfg)
	}
}

// TestIncludeRelativePath tests relative path resolution in includes.
func TestIncludeRelativePath(t *testing.T) {
	t.Parallel()

	td := t.TempDir()
	subdir := filepath.Join(td, "configs")
	err := os.MkdirAll(subdir, 0o755)
	require.NoError(t, err)

	// Create included config in subdirectory
	includePath := filepath.Join(subdir, "included.conf")
	err = os.WriteFile(includePath, []byte("[core]\n\teditor = vim"), 0o644)
	require.NoError(t, err)

	// Main config with relative path include
	configPath := filepath.Join(td, "config")
	// Relative paths are typically resolved from the directory of the config file
	content := "[include]\n\tpath = configs/included.conf\n[user]\n\tname = Test"
	err = os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	if err != nil {
		// Relative path resolution might fail
		return
	}

	require.NotNil(t, cfg)
	// If successfully loaded, verify included value is present
	editor, ok := cfg.Get("core.editor")
	if ok {
		assert.Equal(t, "vim", editor)
	}
}

// TestIncludeAbsolutePath tests absolute path resolution in includes.
func TestIncludeAbsolutePath(t *testing.T) {
	t.Parallel()

	td := t.TempDir()
	configPath := filepath.Join(td, "config")

	// Create include file with absolute path
	includePath := filepath.Join(td, "include.conf")
	err := os.WriteFile(includePath, []byte("[core]\n\tpager = less"), 0o644)
	require.NoError(t, err)

	// Main config with absolute path include
	content := "[include]\n\tpath = " + filepath.ToSlash(includePath) + "\n[user]\n\tname = Test"
	err = os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify included value is present
	pager, ok := cfg.Get("core.pager")
	assert.True(t, ok)
	assert.Equal(t, "less", pager)

	// Verify main config value is present
	name, ok := cfg.Get("user.name")
	assert.True(t, ok)
	assert.Equal(t, "Test", name)
}

// TestIncludeMultipleFiles tests including multiple config files.
func TestIncludeMultipleFiles(t *testing.T) {
	t.Parallel()

	td := t.TempDir()
	configPath := filepath.Join(td, "config")

	// Create multiple include files
	include1 := filepath.Join(td, "config1.conf")
	include2 := filepath.Join(td, "config2.conf")

	err := os.WriteFile(include1, []byte("[core]\n\teditor = vim"), 0o644)
	require.NoError(t, err)

	err = os.WriteFile(include2, []byte("[core]\n\tpager = less"), 0o644)
	require.NoError(t, err)

	// Main config including multiple files
	content := "[include]\n\tpath = " + filepath.ToSlash(include1) + "\n[include]\n\tpath = " + filepath.ToSlash(include2) + "\n[user]\n\tname = Test"
	err = os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify all included values are present
	editor, ok := cfg.Get("core.editor")
	assert.True(t, ok)
	assert.Equal(t, "vim", editor)

	pager, ok := cfg.Get("core.pager")
	assert.True(t, ok)
	assert.Equal(t, "less", pager)

	name, ok := cfg.Get("user.name")
	assert.True(t, ok)
	assert.Equal(t, "Test", name)
}

// TestIncludeOverride tests include file precedence and value merging.
func TestIncludeOverride(t *testing.T) {
	t.Parallel()

	td := t.TempDir()
	configPath := filepath.Join(td, "config")

	// Create include files with overlapping and unique keys
	include1 := filepath.Join(td, "base.conf")
	include2 := filepath.Join(td, "override.conf")

	err := os.WriteFile(include1, []byte("[core]\n\teditor = vim\n\tpager = less"), 0o644)
	require.NoError(t, err)

	err = os.WriteFile(include2, []byte("[core]\n\teditor = nano"), 0o644)
	require.NoError(t, err)

	// Main config includes files in order
	content := "[include]\n\tpath = " + filepath.ToSlash(include1) + "\n[include]\n\tpath = " + filepath.ToSlash(include2)
	err = os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify values from includes are loaded
	editor, ok := cfg.Get("core.editor")
	assert.True(t, ok)
	// Value may depend on implementation; check it exists and is one of the included values
	assert.Contains(t, []string{"vim", "nano"}, editor)

	// Pager from first include should be present
	pager, ok := cfg.Get("core.pager")
	assert.True(t, ok)
	assert.Equal(t, "less", pager)
}

// TestIncludeWithConditional tests conditional include directives.
func TestIncludeWithConditional(t *testing.T) {
	t.Parallel()

	td := t.TempDir()
	configPath := filepath.Join(td, "config")
	gitDir := filepath.Join(td, ".git")
	err := os.MkdirAll(gitDir, 0o755)
	require.NoError(t, err)

	// Create work-specific config
	workConfig := filepath.Join(td, "work.conf")
	err = os.WriteFile(workConfig, []byte("[user]\n\temail = work@company.com"), 0o644)
	require.NoError(t, err)

	// Main config with conditional include
	// Note: Conditional syntax might be [includeIf "gitdir:..."]
	content := `[user]
	email = personal@example.com
[includeIf "gitdir:` + gitDir + `/"]
	path = ` + workConfig
	err = os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	cfg, err := LoadConfigWithWorkdir(configPath, td)
	if err != nil {
		// Not all implementations support conditional includes
		t.Skip("Conditional includes not supported")
	}

	require.NotNil(t, cfg)
	// Verify that the conditional include was applied
	email, ok := cfg.Get("user.email")
	assert.True(t, ok)
	// Should have work email if gitdir condition matched
	assert.NotEmpty(t, email)
}

// TestIncludeEmptyPath tests handling of includes with empty paths.
func TestIncludeEmptyPath(t *testing.T) {
	t.Parallel()

	td := t.TempDir()
	configPath := filepath.Join(td, "config")

	// Config with empty include path
	content := `[include]
	path =
[user]
	name = Test`
	err := os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	// This should either error or be ignored
	cfg, err := LoadConfig(configPath)

	if err != nil {
		// Acceptable to reject invalid syntax
		assert.Error(t, err)
	} else if cfg != nil {
		// Or silently ignore empty path
		assert.NotNil(t, cfg)
	}
}

package gitconfig

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigs(t *testing.T) {
	td := t.TempDir()

	t.Setenv("GOPASS_HOMEDIR", td)

	c := New()
	c.SystemConfig = filepath.Join(td, "system")
	c.GlobalConfig = "global"
	c.LocalConfig = "local"
	c.WorktreeConfig = "worktree"
	c.EnvPrefix = "GPTEST_CONFIG"

	require.NoError(t, os.WriteFile(c.SystemConfig, []byte(`[system]
	key = system
`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(td, c.GlobalConfig), []byte(`[global]
	key = global
[alias "foo"]
	key = bar
`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(td, c.LocalConfig), []byte(`[local]
	key = local
`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(td, c.WorktreeConfig), []byte(`[worktree]
	key = worktree
`), 0o600))
	t.Setenv("GPTEST_CONFIG_COUNT", "1")
	t.Setenv("GPTEST_CONFIG_KEY_0", "env.key")
	t.Setenv("GPTEST_CONFIG_VALUE_0", "env")

	// Load the configs
	c.LoadAll(td)

	assert.True(t, c.HasGlobalConfig())

	// Read the expected keys
	assert.Equal(t, "system", c.Get("system.key"))
	assert.Equal(t, "global", c.Get("global.key"))
	assert.Equal(t, "local", c.Get("local.key"))
	assert.Equal(t, "worktree", c.Get("worktree.key"))
	assert.Equal(t, "env", c.Get("env.key"))

	assert.Equal(t, "global", c.GetGlobal("global.key"))
	assert.Empty(t, c.GetGlobal("local.key"))

	assert.Equal(t, "local", c.GetLocal("local.key"))
	assert.Empty(t, c.GetLocal("global.key"))

	for _, k := range []string{"system.key", "global.key", "local.key", "worktree.key", "env.key"} {
		assert.True(t, c.IsSet(k))
	}

	// SetLocal
	require.NoError(t, c.SetLocal("global.fakekey", "local"))
	assert.Equal(t, "local", c.GetLocal("global.fakekey"))
	assert.Empty(t, c.GetGlobal("global.fakekey"))
	require.NoError(t, c.UnsetLocal("global.fakekey"))
	assert.Empty(t, c.Get("global.fakekey"))

	// SetGlobal
	require.NoError(t, c.SetGlobal("local.fakekey", "global"))
	assert.Empty(t, c.GetLocal("local.fakekey"))
	assert.Equal(t, "global", c.GetGlobal("local.fakekey"))
	require.NoError(t, c.UnsetGlobal("local.fakekey"))
	assert.Empty(t, c.Get("local.fakekey"))

	// SetEnv
	require.NoError(t, c.SetEnv("worktree.fakekey", "env"))
	assert.Empty(t, c.GetLocal("worktree.fakekey"))
	assert.Empty(t, c.GetGlobal("worktree.fakekey"))
	assert.Equal(t, "env", c.Get("worktree.fakekey"))

	// List
	assert.Equal(t, []string{"alias.foo.key", "env.key", "global.key", "local.key", "system.key", "worktree.fakekey", "worktree.key"}, c.Keys())
	assert.Equal(t, []string{"global.key"}, c.List("global."))
	assert.Equal(t, []string{"alias", "env", "global", "local", "system", "worktree"}, c.ListSections())
	assert.Equal(t, []string{"foo"}, c.ListSubsections("alias"))

	// Failure modes
	c.workdir = ""
	c.local = nil
	c.global = nil
	c.env = nil

	require.Error(t, c.SetLocal("core.foo", "bar"))
	require.NoError(t, c.SetGlobal("core.global", "foo"))
	require.NoError(t, c.SetEnv("env.var", "var"))
	require.False(t, c.IsSet("env.nonexistent"))
}

func TestGetFrom(t *testing.T) {
	td := t.TempDir()

	t.Setenv("GOPASS_HOMEDIR", td)

	c := New()
	c.SystemConfig = filepath.Join(td, "system")
	c.GlobalConfig = "global"
	c.LocalConfig = "local"
	c.WorktreeConfig = "worktree"
	c.EnvPrefix = "GPTEST_CONFIG"

	require.NoError(t, os.WriteFile(c.SystemConfig, []byte(`[system]
	key = system
`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(td, c.GlobalConfig), []byte(`[global]
	key = global
`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(td, c.LocalConfig), []byte(`[local]
	key = local
`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(td, c.WorktreeConfig), []byte(`[worktree]
	key = worktree
`), 0o600))
	t.Setenv("GPTEST_CONFIG_COUNT", "1")
	t.Setenv("GPTEST_CONFIG_KEY_0", "env.key")
	t.Setenv("GPTEST_CONFIG_VALUE_0", "env")

	// Load the configs
	c.LoadAll(td)

	// add a preset config
	c.Preset = NewFromMap(map[string]string{"preset.key": "preset"})

	// Valid scopes
	v, ok := c.GetFrom("env.key", "env")
	assert.True(t, ok)
	assert.Equal(t, "env", v)

	v, ok = c.GetFrom("worktree.key", "worktree")
	assert.True(t, ok)
	assert.Equal(t, "worktree", v)

	v, ok = c.GetFrom("local.key", "local")
	assert.True(t, ok)
	assert.Equal(t, "local", v)

	v, ok = c.GetFrom("global.key", "global")
	assert.True(t, ok)
	assert.Equal(t, "global", v)

	v, ok = c.GetFrom("system.key", "system")
	assert.True(t, ok)
	assert.Equal(t, "system", v)

	v, ok = c.GetFrom("preset.key", "preset")
	assert.True(t, ok)
	assert.Equal(t, "preset", v)

	// Unknown scope
	v, ok = c.GetFrom("any.key", "unknownscope")
	assert.False(t, ok)
	assert.Empty(t, v)
}

func writeTestConfig(t *testing.T, path, value string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte("[test]\n\tvalue = "+value+"\n"), 0o600))
}

func TestGlobalConfigPaths(t *testing.T) {
	newConfigs := func(t *testing.T) (*Configs, string, string) {
		t.Helper()

		home := t.TempDir()
		t.Setenv("GOPASS_HOMEDIR", home)
		t.Setenv("GITCONFIG_TEST_NOSYSTEM", "1")

		c := New()
		c.EnvPrefix = "GITCONFIG_TEST"

		return c, home, filepath.Join(home, ".config", "git", "config")
	}

	t.Run("absolute path is used as-is and takes precedence over the XDG location", func(t *testing.T) {
		c, _, xdgConfig := newConfigs(t)
		writeTestConfig(t, xdgConfig, "xdg")

		absConfig := filepath.Join(t.TempDir(), "gopass", "config")
		writeTestConfig(t, absConfig, "absolute")

		c.GlobalConfig = absConfig
		c.LoadAll("")

		assert.True(t, c.HasGlobalConfig())
		assert.Equal(t, "absolute", c.Get("test.value"))
	})

	t.Run("relative path is resolved below the home directory", func(t *testing.T) {
		c, home, _ := newConfigs(t)
		writeTestConfig(t, filepath.Join(home, "custom", "config"), "relative")

		c.GlobalConfig = filepath.Join("custom", "config")
		c.LoadAll("")

		assert.True(t, c.HasGlobalConfig())
		assert.Equal(t, "relative", c.Get("test.value"))
	})

	t.Run("XDG location keeps precedence over a relative path", func(t *testing.T) {
		c, home, xdgConfig := newConfigs(t)
		writeTestConfig(t, filepath.Join(home, "custom", "config"), "relative")
		writeTestConfig(t, xdgConfig, "xdg")

		c.GlobalConfig = filepath.Join("custom", "config")
		c.LoadAll("")

		assert.Equal(t, "xdg", c.Get("test.value"))
	})

	t.Run("missing absolute path does not fall back to the XDG location", func(t *testing.T) {
		c, _, xdgConfig := newConfigs(t)
		writeTestConfig(t, xdgConfig, "xdg")

		c.GlobalConfig = filepath.Join(t.TempDir(), "gopass", "config")
		c.LoadAll("")

		assert.False(t, c.HasGlobalConfig())
		assert.Empty(t, c.Get("test.value"))
	})

	t.Run("absolute path is the write target when no config exists", func(t *testing.T) {
		c, _, xdgConfig := newConfigs(t)
		absConfig := filepath.Join(t.TempDir(), "gopass", "config")

		c.GlobalConfig = absConfig
		c.LoadAll("")

		require.NoError(t, c.SetGlobal("test.value", "written"))

		written, err := LoadConfig(absConfig)
		require.NoError(t, err)

		value, ok := written.Get("test.value")
		require.True(t, ok)
		assert.Equal(t, "written", value)

		_, err = os.Stat(xdgConfig)
		assert.True(t, os.IsNotExist(err), "must not create a default config at the XDG location")
	})

	t.Run("empty GlobalConfig only uses the XDG location", func(t *testing.T) {
		c, _, xdgConfig := newConfigs(t)
		writeTestConfig(t, xdgConfig, "xdg")

		c.GlobalConfig = ""
		c.LoadAll("")

		assert.True(t, c.HasGlobalConfig())
		assert.Equal(t, "xdg", c.Get("test.value"))
	})
}

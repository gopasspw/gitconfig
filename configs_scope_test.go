package gitconfig

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newScopeTestConfigs() *Configs {
	preset := ParseConfig(strings.NewReader("[preset]\n\tkey = preset\n"))
	system := ParseConfig(strings.NewReader("[system]\n\tkey = system\n"))
	global := ParseConfig(strings.NewReader("[global]\n\tkey = global\n"))
	local := ParseConfig(strings.NewReader("[local]\n\tkey = local\n[alias \"foo\"]\n\tkey = bar\n"))
	worktree := ParseConfig(strings.NewReader("[worktree]\n\tkey = worktree\n"))
	env := ParseConfig(strings.NewReader("[env]\n\tkey = env\n"))

	for _, c := range []*Config{preset, system, global, local, worktree, env} {
		c.NoWrites = true
	}

	return &Configs{
		Preset:   preset,
		system:   system,
		global:   global,
		local:    local,
		worktree: worktree,
		env:      env,
	}
}

func TestConfigsScope(t *testing.T) {
	t.Parallel()

	c := newScopeTestConfigs()

	tests := []struct {
		name string
		want *Config
	}{
		{"preset", c.Preset},
		{"system", c.system},
		{"global", c.global},
		{"local", c.local},
		{"worktree", c.worktree},
		{"env", c.env},
		// scope names are case-insensitive
		{"LOCAL", c.local},
		{"Global", c.global},
	}
	for _, tc := range tests {
		got, ok := c.Scope(tc.name)
		assert.True(t, ok, tc.name)
		assert.Same(t, tc.want, got, tc.name)
	}

	got, ok := c.Scope("doesnotexist")
	assert.False(t, ok)
	assert.Nil(t, got)

	got, ok = c.Scope("")
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestConfigsScopeNotLoaded(t *testing.T) {
	t.Parallel()

	c := New()

	// New pre-populates all scopes except the Preset, so they must be found
	// even before LoadAll is called.
	for _, name := range []string{"env", "worktree", "local", "global", "system"} {
		cfg, ok := c.Scope(name)
		assert.True(t, ok, name)
		require.NotNil(t, cfg, name)
		assert.Empty(t, cfg.Keys(), name)
	}

	_, ok := c.Scope("preset")
	assert.False(t, ok)

	// a Configs with all scopes nil must not panic
	c = &Configs{}
	for _, name := range []string{"env", "worktree", "local", "global", "system", "preset"} {
		cfg, ok := c.Scope(name)
		assert.False(t, ok, name)
		assert.Nil(t, cfg, name)
	}

	assert.Empty(t, c.Keys())
}

func TestConfigsScopeIteration(t *testing.T) {
	t.Parallel()

	c := newScopeTestConfigs()

	local, ok := c.Scope("local")
	require.True(t, ok)

	// Entries returns all keys and values of exactly one scope
	assert.Equal(t, map[string][]string{
		"local.key":     {"local"},
		"alias.foo.key": {"bar"},
	}, local.Entries())

	// Keys returns the same keys sorted
	assert.Equal(t, []string{"alias.foo.key", "local.key"}, local.Keys())
	assert.Equal(t, []string{"alias.foo.key"}, local.List("alias."))

	// the union across all scopes still works
	assert.Equal(t, []string{
		"alias.foo.key",
		"env.key",
		"global.key",
		"local.key",
		"preset.key",
		"system.key",
		"worktree.key",
	}, c.Keys())

	// the returned Config is the live scope, so mutations are visible
	require.NoError(t, local.Set("local.other", "value"))
	assert.True(t, c.IsSet("local.other"))
	assert.Equal(t, "value", c.GetLocal("local.other"))
}

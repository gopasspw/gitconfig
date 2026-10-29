package gitconfig

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSortGroupSections(t *testing.T) {
	t.Parallel()

	in := `[core]
	repositoryformatversion = 0
	filemode = true
[remote "origin"]
[remote "github"]
	url = https://github.com/gopasspw/gopass
	fetch = +refs/heads/*:refs/remotes/github/*
[branch "master"]
	remote = origin
	merge = refs/heads/master
[remote "forgejo"]
	fetch = +refs/heads/*:refs/remotes/origin/*
	url = https://forgejo.example.com/gopass
`
	c := ParseConfig(strings.NewReader(in))
	c.NoWrites = true

	require.NoError(t, c.Sort(SortOptions{GroupSections: true}))

	assert.Equal(t, `[core]
	repositoryformatversion = 0
	filemode = true
[remote "origin"]
[remote "github"]
	url = https://github.com/gopasspw/gopass
	fetch = +refs/heads/*:refs/remotes/github/*
[remote "forgejo"]
	fetch = +refs/heads/*:refs/remotes/origin/*
	url = https://forgejo.example.com/gopass
[branch "master"]
	remote = origin
	merge = refs/heads/master
`, c.raw.String())
}

func TestSortSections(t *testing.T) {
	t.Parallel()

	in := `[remote "origin"]
	url = https://example.com/origin
[core]
	bare = false
[remote "github"]
	url = https://example.com/github
[alias]
	co = checkout
`
	c := ParseConfig(strings.NewReader(in))
	c.NoWrites = true

	require.NoError(t, c.Sort(SortOptions{SortSections: true}))

	assert.Equal(t, `[alias]
	co = checkout
[core]
	bare = false
[remote "github"]
	url = https://example.com/github
[remote "origin"]
	url = https://example.com/origin
`, c.raw.String())
}

func TestSortKeys(t *testing.T) {
	t.Parallel()

	in := `[remote "origin"]
	url = https://example.com/origin
	fetch = +refs/heads/*:refs/remotes/origin/*
	fetch = +refs/pull/*/head:refs/remotes/origin/pr/*
	pushurl = https://example.com/push
`
	c := ParseConfig(strings.NewReader(in))
	c.NoWrites = true

	require.NoError(t, c.Sort(SortOptions{SortKeys: true}))

	// fetch keeps its relative order, url and pushurl are sorted
	assert.Equal(t, `[remote "origin"]
	fetch = +refs/heads/*:refs/remotes/origin/*
	fetch = +refs/pull/*/head:refs/remotes/origin/pr/*
	pushurl = https://example.com/push
	url = https://example.com/origin
`, c.raw.String())
}

func TestSortKeysKeepsComments(t *testing.T) {
	t.Parallel()

	in := `[core]
	# the editor
	editor = vim
	# whether to be bare
	bare = false
`
	c := ParseConfig(strings.NewReader(in))
	c.NoWrites = true

	require.NoError(t, c.Sort(SortOptions{SortKeys: true}))

	assert.Equal(t, `[core]
	# whether to be bare
	bare = false
	# the editor
	editor = vim
`, c.raw.String())
}

func TestSortKeepsPreamble(t *testing.T) {
	t.Parallel()

	in := `# global comment
; another one

[remote "origin"]
	url = https://example.com/origin
[core]
	bare = false
`
	c := ParseConfig(strings.NewReader(in))
	c.NoWrites = true

	require.NoError(t, c.Sort(SortOptions{SortSections: true}))

	assert.Equal(t, `# global comment
; another one

[core]
	bare = false
[remote "origin"]
	url = https://example.com/origin
`, c.raw.String())
}

func TestSortSectionHeaderComment(t *testing.T) {
	t.Parallel()

	in := `[remote "origin"] # the main remote
	url = https://example.com/origin
[core] ; basic settings
	bare = false
`
	c := ParseConfig(strings.NewReader(in))
	c.NoWrites = true

	require.NoError(t, c.Sort(SortOptions{SortSections: true}))

	assert.Equal(t, `[core] ; basic settings
	bare = false
[remote "origin"] # the main remote
	url = https://example.com/origin
`, c.raw.String())
}

func TestSortNoop(t *testing.T) {
	t.Parallel()

	in := `[remote "origin"]
	url = https://example.com/origin
[core]
	bare = false
`
	c := ParseConfig(strings.NewReader(in))
	c.NoWrites = true

	require.NoError(t, c.Sort(SortOptions{}))
	assert.Equal(t, in, c.raw.String())

	// readonly configs are not modified
	c.readonly = true
	require.NoError(t, c.Sort(SortOptions{SortSections: true}))
	assert.Equal(t, in, c.raw.String())
}

func TestSortEmptyAndNoWrites(t *testing.T) {
	t.Parallel()

	c := ParseConfig(strings.NewReader(""))
	c.NoWrites = true

	require.NoError(t, c.Sort(SortOptions{SortSections: true, SortKeys: true}))
	assert.Equal(t, "\n", c.raw.String())

	fn := t.TempDir() + "/config"
	in := `[remote "origin"]
	url = https://example.com/origin
[core]
	bare = false
`
	require.NoError(t, os.WriteFile(fn, []byte(in), 0o600))

	onDisk, err := LoadConfig(fn)
	require.NoError(t, err)
	onDisk.NoWrites = true

	require.NoError(t, onDisk.Sort(SortOptions{SortSections: true}))

	// in memory the config is sorted, but the file is untouched
	assert.Equal(t, `[core]
	bare = false
[remote "origin"]
	url = https://example.com/origin
`, onDisk.raw.String())

	written, err := LoadConfig(fn)
	require.NoError(t, err)
	assert.Equal(t, in, written.raw.String())
}

func TestSortWritesToDisk(t *testing.T) {
	t.Parallel()

	fn := t.TempDir() + "/config"
	require.NoError(t, os.WriteFile(fn, []byte(`[remote "origin"]
	url = https://example.com/origin
[core]
	bare = false
`), 0o600))

	c, err := LoadConfig(fn)
	require.NoError(t, err)

	require.NoError(t, c.Sort(SortOptions{SortSections: true}))

	written, err := LoadConfig(fn)
	require.NoError(t, err)
	assert.Equal(t, `[core]
	bare = false
[remote "origin"]
	url = https://example.com/origin
`, written.raw.String())
}

func TestConfigsSort(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GOPASS_HOMEDIR", home)
	t.Setenv("GITCONFIG_TEST_NOSYSTEM", "1")

	workdir := t.TempDir()

	c := New()
	c.EnvPrefix = "GITCONFIG_TEST"
	c.GlobalConfig = "global"
	c.LocalConfig = "local"
	c.WorktreeConfig = "worktree"

	require.NoError(t, os.WriteFile(workdir+"/"+c.LocalConfig, []byte(`[remote "origin"]
	url = https://example.com/origin
[core]
	bare = false
`), 0o600))

	c.LoadAll(workdir)

	require.NoError(t, c.Sort(SortOptions{SortSections: true}))

	written, err := LoadConfig(workdir + "/" + c.LocalConfig)
	require.NoError(t, err)
	assert.Equal(t, `[core]
	bare = false
[remote "origin"]
	url = https://example.com/origin
`, written.raw.String())
}

package gitconfig

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatTrimTrailingWhitespace(t *testing.T) {
	t.Parallel()

	in := "[core]   \n\teditor = vim\t\n\tbare = false \n"
	c := ParseConfig(strings.NewReader(in))
	c.NoWrites = true

	require.NoError(t, c.Format(FormatOptions{TrimTrailingWhitespace: true}))

	assert.Equal(t, "[core]\n\teditor = vim\n\tbare = false\n", c.raw.String())
}

func TestFormatCollapseBlankLines(t *testing.T) {
	t.Parallel()

	in := `[core]


	editor = vim


[user]
	name = John
`
	c := ParseConfig(strings.NewReader(in))
	c.NoWrites = true

	require.NoError(t, c.Format(FormatOptions{CollapseBlankLines: true}))

	assert.Equal(t, `[core]

	editor = vim

[user]
	name = John
`, c.raw.String())
}

func TestFormatRemoveBlankLinesAfterSectionHeader(t *testing.T) {
	t.Parallel()

	in := `[core]

	editor = vim
[user]

	name = John
`
	c := ParseConfig(strings.NewReader(in))
	c.NoWrites = true

	require.NoError(t, c.Format(FormatOptions{RemoveBlankLinesAfterSectionHeader: true}))

	assert.Equal(t, `[core]
	editor = vim
[user]
	name = John
`, c.raw.String())
}

func TestFormatRemoveBlankLinesBeforeSectionHeader(t *testing.T) {
	t.Parallel()

	in := `[core]
	editor = vim

[user]
	name = John

`
	c := ParseConfig(strings.NewReader(in))
	c.NoWrites = true

	require.NoError(t, c.Format(FormatOptions{RemoveBlankLinesBeforeSectionHeader: true}))

	// blank lines at the end of the file are kept
	assert.Equal(t, `[core]
	editor = vim
[user]
	name = John

`, c.raw.String())
}

func TestFormatTrimBlankLines(t *testing.T) {
	t.Parallel()

	in := `

[core]
	editor = vim

`
	c := ParseConfig(strings.NewReader(in))
	c.NoWrites = true

	require.NoError(t, c.Format(FormatOptions{TrimBlankLines: true}))

	assert.Equal(t, "[core]\n\teditor = vim\n", c.raw.String())
}

func TestFormatKeepsComments(t *testing.T) {
	t.Parallel()

	in := `# leading comment

[core]

	# the editor
	editor = vim

`
	c := ParseConfig(strings.NewReader(in))
	c.NoWrites = true

	require.NoError(t, c.Format(FormatOptions{
		CollapseBlankLines:                 true,
		RemoveBlankLinesAfterSectionHeader: true,
		TrimBlankLines:                     true,
	}))

	assert.Equal(t, `# leading comment

[core]
	# the editor
	editor = vim
`, c.raw.String())
}

func TestFormatKeepsValues(t *testing.T) {
	t.Parallel()

	in := `[core]

	editor = vim

[remote "origin"]

	url = https://example.com/origin
	fetch = +refs/heads/*:refs/remotes/origin/*
`
	c := ParseConfig(strings.NewReader(in))
	c.NoWrites = true

	before := c.Entries()

	require.NoError(t, c.Format(FormatOptions{
		CollapseBlankLines:                 true,
		RemoveBlankLinesAfterSectionHeader: true,
	}))

	assert.Equal(t, before, c.Entries())
}

func TestFormatNoop(t *testing.T) {
	t.Parallel()

	in := "[core]\n\n\teditor = vim\n\n\n"
	c := ParseConfig(strings.NewReader(in))
	c.NoWrites = true

	require.NoError(t, c.Format(FormatOptions{}))
	assert.Equal(t, in, c.raw.String())

	// readonly configs are not modified
	c.readonly = true
	require.NoError(t, c.Format(FormatOptions{CollapseBlankLines: true}))
	assert.Equal(t, in, c.raw.String())
}

func TestFormatEmptyAndNoWrites(t *testing.T) {
	t.Parallel()

	c := ParseConfig(strings.NewReader(""))
	c.NoWrites = true

	require.NoError(t, c.Format(FormatOptions{CollapseBlankLines: true, TrimBlankLines: true}))
	assert.Equal(t, "\n", c.raw.String())

	fn := t.TempDir() + "/config"
	in := "[core]\n\n\n\teditor = vim\n"
	require.NoError(t, os.WriteFile(fn, []byte(in), 0o600))

	onDisk, err := LoadConfig(fn)
	require.NoError(t, err)
	onDisk.NoWrites = true

	require.NoError(t, onDisk.Format(FormatOptions{CollapseBlankLines: true}))

	// in memory the config is formatted, but the file is untouched
	assert.Equal(t, "[core]\n\n\teditor = vim\n", onDisk.raw.String())

	written, err := LoadConfig(fn)
	require.NoError(t, err)
	assert.Equal(t, in, written.raw.String())
}

func TestFormatWritesToDisk(t *testing.T) {
	t.Parallel()

	fn := t.TempDir() + "/config"
	require.NoError(t, os.WriteFile(fn, []byte("[core]\n\n\n\teditor = vim\n"), 0o600))

	c, err := LoadConfig(fn)
	require.NoError(t, err)

	require.NoError(t, c.Format(FormatOptions{CollapseBlankLines: true}))

	written, err := LoadConfig(fn)
	require.NoError(t, err)
	assert.Equal(t, "[core]\n\n\teditor = vim\n", written.raw.String())
}

func TestConfigsFormat(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GOPASS_HOMEDIR", home)
	t.Setenv("GITCONFIG_TEST_NOSYSTEM", "1")

	workdir := t.TempDir()

	c := New()
	c.EnvPrefix = "GITCONFIG_TEST"
	c.GlobalConfig = "global"
	c.LocalConfig = "local"
	c.WorktreeConfig = "worktree"

	require.NoError(t, os.WriteFile(workdir+"/"+c.LocalConfig, []byte("[core]\n\n\n\teditor = vim\n"), 0o600))

	c.LoadAll(workdir)

	require.NoError(t, c.Format(FormatOptions{CollapseBlankLines: true}))

	written, err := LoadConfig(workdir + "/" + c.LocalConfig)
	require.NoError(t, err)
	assert.Equal(t, "[core]\n\n\teditor = vim\n", written.raw.String())
}

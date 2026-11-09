package gitconfig

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitCompatLineContinuation(t *testing.T) {
	t.Parallel()

	in := `[section]
	unquoted = hello \
world \
again
	quoted = "first \
second"
	escapedBackslash = trailing\\
	nextKey = afterEscaped
`
	c := ParseConfig(strings.NewReader(in))

	v, ok := c.Get("section.unquoted")
	assert.True(t, ok)
	assert.Equal(t, "hello world again", v)

	v, ok = c.Get("section.quoted")
	assert.True(t, ok)
	assert.Equal(t, "first second", v)

	v, ok = c.Get("section.escapedBackslash")
	assert.True(t, ok)
	assert.Equal(t, `trailing\`, v)

	v, ok = c.Get("section.nextKey")
	assert.True(t, ok)
	assert.Equal(t, "afterEscaped", v)
}

func TestGitCompatMidValueQuotesAndInlineComments(t *testing.T) {
	t.Parallel()

	in := `[alias]
	credit1 = "!f() { git commit --amend --author \"$1 <$2>\" -C HEAD; }; f" # Credit an author
	credit2 = commit --amend --author "$1 <$2>" -C HEAD
	d = !"git diff-index --quiet HEAD -- || clear; git diff --patch-with-stat" # Show diff
	p = !"git pull; git submodule foreach git pull origin master" ; pull all
	it = !"git init && git commit -m "root" --allow-empty"
	spaceQuote = prefix "  inside  " suffix # trailing comment
`
	c := ParseConfig(strings.NewReader(in))

	v, ok := c.Get("alias.credit1")
	assert.True(t, ok)
	assert.Equal(t, `!f() { git commit --amend --author "$1 <$2>" -C HEAD; }; f`, v)

	v, ok = c.Get("alias.credit2")
	assert.True(t, ok)
	assert.Equal(t, `commit --amend --author $1 <$2> -C HEAD`, v)

	v, ok = c.Get("alias.d")
	assert.True(t, ok)
	assert.Equal(t, `!git diff-index --quiet HEAD -- || clear; git diff --patch-with-stat`, v)

	v, ok = c.Get("alias.p")
	assert.True(t, ok)
	assert.Equal(t, `!git pull; git submodule foreach git pull origin master`, v)

	v, ok = c.Get("alias.it")
	assert.True(t, ok)
	assert.Equal(t, `!git init && git commit -m root --allow-empty`, v)

	v, ok = c.Get("alias.spaceQuote")
	assert.True(t, ok)
	assert.Equal(t, `prefix   inside   suffix`, v)
}

func TestGitCompatUnescapeOrderOfOperations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "escaped backslash before n",
			input: `\\n`,
			want:  `\n`,
		},
		{
			name:  "escaped backslash before t",
			input: `\\t`,
			want:  `\t`,
		},
		{
			name:  "escaped backslash before b",
			input: `\\b`,
			want:  `\b`,
		},
		{
			name:  "escaped backslash before quote",
			input: `\\\"`,
			want:  `\"`,
		},
		{
			name:  "double escaped backslash",
			input: `\\\\`,
			want:  `\\`,
		},
		{
			name:  "escaped backslash followed by real newline escape",
			input: `\\\n`,
			want:  "\\\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, unescapeValue(tc.input))
		})
	}
}

func TestGitCompatRoundTripEscapeOnWrite(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
	}{
		{
			name:  "value with hash comment char",
			value: "foo # not a comment",
		},
		{
			name:  "value with semicolon comment char",
			value: "foo ; not a comment",
		},
		{
			name:  "value with double quotes",
			value: `say "hello" world`,
		},
		{
			name:  "value with backslashes",
			value: `C:\Users\name\Documents`,
		},
		{
			name:  "value with newline and tab",
			value: "line1\nline2\tindented",
		},
		{
			name:  "value with leading and trailing spaces",
			value: "  padded value  ",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			td := t.TempDir()
			fn := filepath.Join(td, "config")

			c := &Config{path: fn}
			require.NoError(t, c.Set("section.key", tc.value))

			reloaded, err := LoadConfig(fn)
			require.NoError(t, err)

			got, ok := reloaded.Get("section.key")
			require.True(t, ok)
			assert.Equal(t, tc.value, got)
		})
	}
}

func TestGitCompatSectionHeaders(t *testing.T) {
	t.Parallel()

	in := `[core] # main core comment
	editor = vim
[remote "origin"] ; remote comment
	url = https://example.com/repo.git
[section "sub\\path\"name"] # comment after escaped quote
	key = val1
[Branch.Main]
	remote = origin
`
	c := ParseConfig(strings.NewReader(in))

	v, ok := c.Get("core.editor")
	assert.True(t, ok)
	assert.Equal(t, "vim", v)

	v, ok = c.Get("remote.origin.url")
	assert.True(t, ok)
	assert.Equal(t, "https://example.com/repo.git", v)

	v, ok = c.Get(`section.sub\path"name.key`)
	assert.True(t, ok)
	assert.Equal(t, "val1", v)

	// Deprecated [section.subsection] syntax lowercases subsection name per git spec
	v, ok = c.Get("branch.main.remote")
	assert.True(t, ok)
	assert.Equal(t, "origin", v)
}

func TestGitCompatBareBooleans(t *testing.T) {
	t.Parallel()

	in := `[http]
	sslVerify
	bareWithComment # inline comment on bare key
	emptyVal =
	explicitTrue = true
	explicitFalse = false
`
	c := ParseConfig(strings.NewReader(in))

	// Get returns ("true", true) for bare keys per git-config syntax ("short-hand to say that the variable is the boolean true")
	v, ok := c.Get("http.sslVerify")
	assert.True(t, ok)
	assert.Equal(t, "true", v)

	v, ok = c.Get("http.bareWithComment")
	assert.True(t, ok)
	assert.Equal(t, "true", v)

	v, ok = c.Get("http.emptyVal")
	assert.True(t, ok)
	assert.Empty(t, v)

	// GetBool distinguishes bare keys (true) from empty `key =` values (false)
	b, ok := c.GetBool("http.sslVerify")
	assert.True(t, ok)
	assert.True(t, b)

	b, ok = c.GetBool("http.bareWithComment")
	assert.True(t, ok)
	assert.True(t, b)

	b, ok = c.GetBool("http.emptyVal")
	assert.True(t, ok)
	assert.False(t, b)

	b, ok = c.GetBool("http.explicitTrue")
	assert.True(t, ok)
	assert.True(t, b)

	b, ok = c.GetBool("http.explicitFalse")
	assert.True(t, ok)
	assert.False(t, b)
}

func TestGitCompatMultiValueGetAndMutations(t *testing.T) {
	t.Parallel()

	in := `[remote "origin"]
	fetch = +refs/heads/first:refs/remotes/origin/first
	fetch = +refs/heads/second:refs/remotes/origin/second
`
	c := ParseConfig(strings.NewReader(in))
	c.NoWrites = true

	// Get returns last value (canonical git config --get behavior)
	last, ok := c.Get("remote.origin.fetch")
	assert.True(t, ok)
	assert.Equal(t, "+refs/heads/second:refs/remotes/origin/second", last)

	// Add appends a new value to the multivar
	require.NoError(t, c.Add("remote.origin.fetch", "+refs/heads/third:refs/remotes/origin/third"))
	vs, ok := c.GetAll("remote.origin.fetch")
	assert.True(t, ok)
	assert.Equal(t, []string{
		"+refs/heads/first:refs/remotes/origin/first",
		"+refs/heads/second:refs/remotes/origin/second",
		"+refs/heads/third:refs/remotes/origin/third",
	}, vs)

	last, ok = c.Get("remote.origin.fetch")
	assert.True(t, ok)
	assert.Equal(t, "+refs/heads/third:refs/remotes/origin/third", last)

	// SetAll replaces all occurrences with a single value (like git config --replace-all)
	require.NoError(t, c.SetAll("remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*"))
	vs, ok = c.GetAll("remote.origin.fetch")
	assert.True(t, ok)
	assert.Equal(t, []string{"+refs/heads/*:refs/remotes/origin/*"}, vs)

	// Reparse raw to ensure raw representation also has only one entry
	reparsed := ParseConfig(strings.NewReader(c.raw.String()))
	vs, ok = reparsed.GetAll("remote.origin.fetch")
	assert.True(t, ok)
	assert.Equal(t, []string{"+refs/heads/*:refs/remotes/origin/*"}, vs)
}

func TestGitCompatCrossScopeGetAll(t *testing.T) {
	t.Parallel()

	c := &Configs{
		Preset:   NewFromMap(map[string]string{"multi.key": "preset", "only.preset": "default"}),
		system:   ParseConfig(strings.NewReader("[multi]\n\tkey = system\n")),
		global:   ParseConfig(strings.NewReader("[multi]\n\tkey = global1\n\tkey = global2\n")),
		local:    ParseConfig(strings.NewReader("[multi]\n\tkey = local\n")),
		worktree: ParseConfig(strings.NewReader("[multi]\n\tkey = worktree\n")),
		env:      ParseConfig(strings.NewReader("[multi]\n\tkey = env\n")),
	}

	// GetAll aggregates across all configured scopes from lowest to highest precedence
	// (system -> global -> local -> worktree -> env), without including Preset when overridden.
	assert.Equal(t, []string{"system", "global1", "global2", "local", "worktree", "env"}, c.GetAll("multi.key"))

	// Preset is used as fallback when no other scope defines the key
	assert.Equal(t, []string{"default"}, c.GetAll("only.preset"))
}

func TestGitCompatGitDirPatterns(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on windows")
	}

	td := t.TempDir()
	home := filepath.Join(td, "home")
	require.NoError(t, os.MkdirAll(filepath.Join(home, "work", "project"), 0o755))
	t.Setenv("HOME", home)
	t.Setenv("GOPASS_HOMEDIR", home)

	cfgDir := filepath.Join(td, "cfg")
	require.NoError(t, os.MkdirAll(filepath.Join(cfgDir, "relrepo", "sub"), 0o755))

	fn := filepath.Join(cfgDir, "config")
	require.NoError(t, os.WriteFile(fn, []byte(`[includeIf "gitdir:~/work/"]
	path = home-work.config
[includeIf "gitdir:./relrepo/"]
	path = rel.config
[includeIf "gitdir:feature-repo/"]
	path = relative-glob.config
`), 0o600))

	require.NoError(t, os.WriteFile(filepath.Join(cfgDir, "home-work.config"), []byte("[matched]\n\thomework = true\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(cfgDir, "rel.config"), []byte("[matched]\n\trel = true\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(cfgDir, "relative-glob.config"), []byte("[matched]\n\trelglob = true\n"), 0o600))

	// 1. Match ~/work/ against ~/work/project
	c1, err := LoadConfigWithWorkdir(fn, filepath.Join(home, "work", "project"))
	require.NoError(t, err)
	v, ok := c1.Get("matched.homework")
	assert.True(t, ok)
	assert.Equal(t, "true", v)

	// 2. Match ./relrepo/ relative to cfgDir
	c2, err := LoadConfigWithWorkdir(fn, filepath.Join(cfgDir, "relrepo", "sub"))
	require.NoError(t, err)
	v, ok = c2.Get("matched.rel")
	assert.True(t, ok)
	assert.Equal(t, "true", v)

	// 3. Match relative pattern without ./ (auto-prepends **/)
	c3, err := LoadConfigWithWorkdir(fn, "/any/deep/path/feature-repo/subdir")
	require.NoError(t, err)
	v, ok = c3.Get("matched.relglob")
	assert.True(t, ok)
	assert.Equal(t, "true", v)
}

func TestGitCompatEnvCanonicalizationAndOverrides(t *testing.T) {
	td := t.TempDir()
	t.Setenv("GOPASS_HOMEDIR", td)

	sysFile := filepath.Join(td, "custom-system")
	globFile := filepath.Join(td, "custom-global")
	require.NoError(t, os.WriteFile(sysFile, []byte("[system]\n\tFromEnvPath = yes\n"), 0o600))
	require.NoError(t, os.WriteFile(globFile, []byte("[global]\n\tFromEnvPath = yes\n"), 0o600))

	prefix := "GPTEST_ENV_COMPAT"
	t.Setenv(prefix+"_SYSTEM", sysFile)
	t.Setenv(prefix+"_GLOBAL", globFile)
	// Setting NOSYSTEM to "false" or "0" must NOT disable system config per git rules
	t.Setenv(prefix+"_NOSYSTEM", "false")
	t.Setenv(prefix+"_COUNT", "1")
	t.Setenv(prefix+"_KEY_0", "Core.MixedCaseKey")
	t.Setenv(prefix+"_VALUE_0", "from-env")
	t.Setenv(prefix+"_PARAMETERS", `'alias.st=status -sb' 'user.name=Jane '\''Git'\'' Doe'`)

	cs := New()
	cs.EnvPrefix = prefix
	cs.LoadAll("")

	assert.Equal(t, "yes", cs.Get("system.fromenvpath"))
	assert.Equal(t, "yes", cs.Get("global.fromenvpath"))
	assert.Equal(t, "from-env", cs.Get("core.mixedcasekey"))
	assert.Equal(t, "status -sb", cs.Get("alias.st"))
	assert.Equal(t, "Jane 'Git' Doe", cs.Get("user.name"))
}

func TestGitCompatWorkdirAndGitFileDiscovery(t *testing.T) {
	td := t.TempDir()
	t.Setenv("GOPASS_HOMEDIR", td)
	t.Setenv("GPTEST_WD_NOSYSTEM", "1")

	// 1. Standard repository where workdir is repo root containing .git/ directory
	repoRoot := filepath.Join(td, "repo")
	gitDir := filepath.Join(repoRoot, ".git")
	require.NoError(t, os.MkdirAll(gitDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "config"), []byte(`[core]
	bare = false
[includeIf "onbranch:feat/worktree"]
	path = branch.config
`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "branch.config"), []byte(`[branchcfg]
	active = true
`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/feat/worktree\n"), 0o644))

	cs := New()
	cs.EnvPrefix = "GPTEST_WD"
	cs.LoadAll(repoRoot)

	assert.Equal(t, "false", cs.GetLocal("core.bare"))
	assert.Equal(t, "true", cs.GetLocal("branchcfg.active"))

	// 2. Linked worktree where .git is a file pointing to gitdir: <path> with commondir
	wtRoot := filepath.Join(td, "linked-worktree")
	wtGitDir := filepath.Join(gitDir, "worktrees", "linked")
	require.NoError(t, os.MkdirAll(wtRoot, 0o755))
	require.NoError(t, os.MkdirAll(wtGitDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(wtRoot, ".git"), []byte("gitdir: "+wtGitDir+"\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(wtGitDir, "commondir"), []byte("../..\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(wtGitDir, "HEAD"), []byte("ref: refs/heads/feat/worktree\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(wtGitDir, "config.worktree"), []byte("[worktree]\n\tspecific = yes\n"), 0o600))

	// Without extensions.worktreeConfig = true, config.worktree is ignored per git rules
	csWt := New()
	csWt.EnvPrefix = "GPTEST_WD"
	csWt.LoadAll(wtRoot)

	assert.Equal(t, "false", csWt.GetLocal("core.bare"))
	assert.Equal(t, "true", csWt.GetLocal("branchcfg.active"))
	assert.Empty(t, csWt.Get("worktree.specific"))

	// Enabling extensions.worktreeConfig causes config.worktree to be loaded
	require.NoError(t, csWt.SetLocal("extensions.worktreeConfig", "true"))
	csWt.Reload()
	assert.Equal(t, "yes", csWt.Get("worktree.specific"))
}

func TestGitCompatTypedGetters(t *testing.T) {
	t.Setenv("HOME", "/home/testuser")
	t.Setenv("GOPASS_HOMEDIR", "/home/testuser")

	in := `[bools]
	bTrue1 = true
	bTrue2 = yes
	bTrue3 = on
	bTrue4 = 1
	bTrueBare
	bFalse1 = false
	bFalse2 = no
	bFalse3 = off
	bFalse4 = 0
	bFalseEmpty =
	bInvalid = notabool
[ints]
	plain = 42
	negative = -10
	kilo = 2k
	mega = 3M
	giga = 1G
	invalid = 12x
[paths]
	homeRel = ~/.gitconfig-extra
	abs = /etc/gitconfig
`
	c := ParseConfig(strings.NewReader(in))

	for _, k := range []string{"bools.bTrue1", "bools.bTrue2", "bools.bTrue3", "bools.bTrue4", "bools.bTrueBare"} {
		v, ok := c.GetBool(k)
		assert.True(t, ok, k)
		assert.True(t, v, k)
	}
	for _, k := range []string{"bools.bFalse1", "bools.bFalse2", "bools.bFalse3", "bools.bFalse4", "bools.bFalseEmpty"} {
		v, ok := c.GetBool(k)
		assert.True(t, ok, k)
		assert.False(t, v, k)
	}
	_, ok := c.GetBool("bools.bInvalid")
	assert.False(t, ok)
	_, ok = c.GetBool("bools.missing")
	assert.False(t, ok)

	iv, ok := c.GetInt("ints.plain")
	assert.True(t, ok)
	assert.Equal(t, int64(42), iv)

	iv, ok = c.GetInt("ints.negative")
	assert.True(t, ok)
	assert.Equal(t, int64(-10), iv)

	iv, ok = c.GetInt("ints.kilo")
	assert.True(t, ok)
	assert.Equal(t, int64(2*1024), iv)

	iv, ok = c.GetInt("ints.mega")
	assert.True(t, ok)
	assert.Equal(t, int64(3*1024*1024), iv)

	iv, ok = c.GetInt("ints.giga")
	assert.True(t, ok)
	assert.Equal(t, int64(1*1024*1024*1024), iv)

	_, ok = c.GetInt("ints.invalid")
	assert.False(t, ok)

	pv, ok := c.GetPath("paths.homeRel")
	assert.True(t, ok)
	assert.Equal(t, filepath.Join("/home/testuser", ".gitconfig-extra"), pv)

	pv, ok = c.GetPath("paths.abs")
	assert.True(t, ok)
	assert.Equal(t, "/etc/gitconfig", pv)
}

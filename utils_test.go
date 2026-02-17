package gitconfig

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrim(t *testing.T) {
	t.Parallel()

	for _, tc := range [][]string{
		{" a ", "b       ", "\tc\n"},
	} {
		trim(tc)
		for _, e := range tc {
			assert.Equal(t, strings.TrimSpace(e), e)
		}
	}
}

func TestGlobMatch(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		pattern string
		input   string
		want    bool
		wantErr bool
	}{
		{
			name:    "single asterisk matches within component",
			pattern: "feat/*",
			input:   "feat/test",
			want:    true,
		},
		{
			name:    "double asterisk matches across components",
			pattern: "feat/**",
			input:   "feat/foo/bar/baz",
			want:    true,
		},
		{
			name:    "single asterisk no match",
			pattern: "feat/*",
			input:   "feat/foo/bar",
			want:    false,
		},
		{
			name:    "question mark matches single character",
			pattern: "?.js",
			input:   "a.js",
			want:    true,
		},
		{
			name:    "question mark multiple",
			pattern: "test_?_?.go",
			input:   "test_a_b.go",
			want:    true,
		},
		{
			name:    "character class matching",
			pattern: "[ab].txt",
			input:   "a.txt",
			want:    true,
		},
		{
			name:    "character range",
			pattern: "[a-z]*.txt",
			input:   "names.txt",
			want:    true,
		},
		{
			name:    "no match",
			pattern: "*.md",
			input:   "file.go",
			want:    false,
		},
		{
			name:    "exact match",
			pattern: "exact.txt",
			input:   "exact.txt",
			want:    true,
		},
		{
			name:    "invalid pattern - bad range",
			pattern: "[z-a].txt",
			input:   "a.txt",
			wantErr: true,
		},
		{
			name:    "invalid pattern - bad bracket",
			pattern: "[.txt",
			input:   "a.txt",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := globMatch(tc.pattern, tc.input)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.want, got)
			}
		})
	}
}

func TestSplitKey(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		in         string
		section    string
		subsection string
		key        string
	}{
		{
			in:         "url.git@gist.github.com:.pushinsteadof",
			section:    "url",
			subsection: "git@gist.github.com:",
			key:        "pushinsteadof",
		},
		{
			in:      "gc.auto",
			section: "gc",
			key:     "auto",
		},
	} {
		sec, sub, key := splitKey(tc.in)
		assert.Equal(t, tc.section, sec, sec)
		assert.Equal(t, tc.subsection, sub, sub)
		assert.Equal(t, tc.key, key, key)
	}
}

// TestParseLineForComment tests the parseLineForComment function with various inputs.
func TestParseLineForComment(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		wantContent string
		wantComment string
	}{
		{
			name:        "Double quotes with hash comment",
			input:       `"foo#bar#baz" # comment1`,
			wantContent: `foo#bar#baz`,
			wantComment: `comment1`,
		},
		{
			name:        "Single quotes with semicolon comment - single quotes are not valid",
			input:       `'foo;bar' ; comment2`,
			wantContent: `'foo`,
			wantComment: `bar' ; comment2`,
		},
		{
			name:        "No quotes with hash comment",
			input:       `no quotes here # comment3`,
			wantContent: `no quotes here`,
			wantComment: `comment3`,
		},
		{
			name:        "Nested single quotes with hash comment",
			input:       `"nested 'quotes' # works" # comment4`,
			wantContent: `nested 'quotes' # works`,
			wantComment: `comment4`,
		},
		{
			name:        "No comment present",
			input:       `no comment here`,
			wantContent: `no comment here`,
			wantComment: ``,
		},
		{
			name:        "Leading space content with semicolon comment",
			input:       `   "leading space content" ; comment6`,
			wantContent: `leading space content`,
			wantComment: `comment6`,
		},
		{
			name:        "Trailing space content and comment with hash",
			input:       `trailing space content # comment7   `,
			wantContent: `trailing space content`,
			wantComment: `comment7`,
		},
		{
			name:        "Hash comment line",
			input:       `# comment line`,
			wantContent: ``,
			wantComment: `comment line`,
		},
		{
			name:        "Semicolon comment line",
			input:       `; another comment line`,
			wantContent: ``,
			wantComment: `another comment line`,
		},
		{
			name:        "Quoted content spanning potential comment char",
			input:       ` "quotes spanning ; comment char" `,
			wantContent: `quotes spanning ; comment char`,
			wantComment: ``,
		},
		{
			name:        "Unterminated quote before hash comment",
			input:       ` "unterminated ' quote # comment"`,
			wantContent: `unterminated ' quote # comment`,
			wantComment: ``,
		},
		{
			name:        "Hash inside quotes with comment outside",
			input:       ` "hash # inside" # comment outside `,
			wantContent: `hash # inside`,
			wantComment: `comment outside`,
		},
		{
			name:        "Hash inside quotes part of string",
			input:       ` string with #"# hash inside quotes`,
			wantContent: `string with`,
			wantComment: `"# hash inside quotes`,
		},
		{
			name:        "Empty input string",
			input:       ``,
			wantContent: ``,
			wantComment: ``,
		},
		{
			name:        "Whitespace only input string",
			input:       `   `,
			wantContent: ``,
			wantComment: ``,
		},
		{
			name:        "Key value pair like structure",
			input:       `key = value # comment`,
			wantContent: `key = value`,
			wantComment: `comment`,
		},
		{
			name:        "Only double quoted content",
			input:       `"only quotes"`,
			wantContent: `only quotes`,
			wantComment: ``,
		},
		{
			name:        "Only single quoted content",
			input:       `'single quotes'`,
			wantContent: `'single quotes'`,
			wantComment: ``,
		},
		{
			name:        "Mismatched surrounding quotes 1",
			input:       ` " mismatched quote'`,
			wantContent: ` mismatched quote'`,
			wantComment: ``,
		},
		{
			name:        "Mismatched surrounding quotes 2",
			input:       ` 'mismatched quote"`,
			wantContent: `'mismatched quote"`,
			wantComment: ``,
		},
		{
			name:        "Single quote only content",
			input:       ` '`,
			wantContent: `'`,
			wantComment: ``,
		},
		{
			name:        "Double quote only content",
			input:       `"`,
			wantContent: ``,
			wantComment: ``,
		},
		{
			name:        "Empty double quotes",
			input:       `""`,
			wantContent: ``,
			wantComment: ``,
		},
		{
			name:        "Empty single quotes",
			input:       `''`,
			wantContent: `''`,
			wantComment: ``,
		},
		{
			name:        "Content followed immediately by hash",
			input:       `content#`,
			wantContent: `content`,
			wantComment: ``,
		},
		{
			name:        "Content followed immediately by semicolon",
			input:       `content;`,
			wantContent: `content`,
			wantComment: ``,
		},
		{
			name:        "Content followed by delimiter and spaces",
			input:       `content #  `,
			wantContent: `content`,
			wantComment: ``,
		},
	}

	// Iterate over the test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotContent, gotComment := parseLineForComment(tc.input)

			if gotContent != tc.wantContent {
				t.Errorf("parseLineForComment(%q) got content %q, want %q", tc.input, gotContent, tc.wantContent)
			}

			if gotComment != tc.wantComment {
				t.Errorf("parseLineForComment(%q) got comment %q, want %q", tc.input, gotComment, tc.wantComment)
			}
		})
	}
}

func TestCanonicalizeKey(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple key",
			input:    "core.autocrlf",
			expected: "core.autocrlf",
		},
		{
			name:     "Key with subsection",
			input:    "remote.origin.url",
			expected: "remote.origin.url",
		},
		{
			name:     "Key with mixed case section and key",
			input:    "Core.AutoCRLF",
			expected: "core.autocrlf",
		},
		{
			name:     "Key with mixed case section, subsection, and key",
			input:    "Remote.Origin.URL",
			expected: "remote.Origin.url",
		},
		{
			name:     "Key with subsection containing dots",
			input:    "url.git@github.com:.pushinsteadof",
			expected: "url.git@github.com:.pushinsteadof",
		},
		{
			name:     "Key with mixed case and subsection containing dots",
			input:    "Url.Git@github.com:.PushInsteadOf",
			expected: "url.Git@github.com:.pushinsteadof",
		},
		{
			name:     "Empty input - invalid",
			input:    "",
			expected: "",
		},
		{
			name:     "Single part input - invalid",
			input:    "section",
			expected: "",
		},
		{
			name:     "Key starting with dot - invalid",
			input:    ".key",
			expected: "",
		},
		{
			name:     "Key ending with dot - invalid",
			input:    "section.",
			expected: "",
		},
		{
			name:     "Key with multiple dots in subsection",
			input:    "section.sub.section.key",
			expected: "section.sub.section.key",
		},
		{
			name:     "Key with uppercase subsection",
			input:    "section.SUBSECTION.key",
			expected: "section.SUBSECTION.key",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actual := canonicalizeKey(tc.input)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

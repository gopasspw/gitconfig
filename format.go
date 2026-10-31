package gitconfig

import (
	"strings"

	"github.com/gopasspw/gopass/pkg/debug"
)

// FormatOptions controls how Config.Format normalizes the layout of a config
// file. The zero value is a no-op, i.e. the raw config is left untouched.
//
// Formatting is opt-in. By default the library preserves the layout of the
// original file, which matches the behaviour of the git CLI. Only enable the
// options whose result is acceptable for the config at hand, since formatting
// rewrites the file and may drop intentional blank lines.
type FormatOptions struct {
	// TrimTrailingWhitespace removes trailing whitespace from every line.
	TrimTrailingWhitespace bool

	// CollapseBlankLines replaces runs of consecutive blank lines with a
	// single blank line.
	CollapseBlankLines bool

	// RemoveBlankLinesAfterSectionHeader removes blank lines directly after a
	// section header, e.g. the empty line in
	//
	//	[core]
	//
	//		editor = vim
	RemoveBlankLinesAfterSectionHeader bool

	// RemoveBlankLinesBeforeSectionHeader removes blank lines directly before
	// a section header.
	RemoveBlankLinesBeforeSectionHeader bool

	// TrimBlankLines removes blank lines at the beginning and the end of the
	// file.
	TrimBlankLines bool
}

// isZero returns true if no formatting was requested.
func (o FormatOptions) isZero() bool {
	return !o.TrimTrailingWhitespace &&
		!o.CollapseBlankLines &&
		!o.RemoveBlankLinesAfterSectionHeader &&
		!o.RemoveBlankLinesBeforeSectionHeader &&
		!o.TrimBlankLines
}

// Format normalizes the layout of the config according to opts and persists
// the result.
//
// Only the layout is changed, the parsed keys and values are left untouched.
// Comments are always preserved. Readonly configs silently ignore the
// operation, and NoWrites only prevents persisting the result to disk.
//
// Example:
//
//	cfg, err := gitconfig.LoadConfig(".git/config")
//	if err != nil {
//	  log.Fatal(err)
//	}
//	// remove trailing whitespace, collapse multiple blank lines and drop
//	// blank lines directly after a section header
//	if err := cfg.Format(gitconfig.FormatOptions{
//	  TrimTrailingWhitespace:             true,
//	  CollapseBlankLines:                 true,
//	  RemoveBlankLinesAfterSectionHeader: true,
//	}); err != nil {
//	  log.Fatal(err)
//	}
func (c *Config) Format(opts FormatOptions) error {
	if c.readonly || opts.isZero() {
		return nil
	}

	lines := splitRawLines(c.raw.String())

	if opts.TrimTrailingWhitespace {
		for i, line := range lines {
			lines[i] = strings.TrimRight(line, " \t")
		}
	}

	if opts.RemoveBlankLinesAfterSectionHeader {
		lines = removeBlankLinesAfterSectionHeaders(lines)
	}

	if opts.RemoveBlankLinesBeforeSectionHeader {
		lines = removeBlankLinesBeforeSectionHeaders(lines)
	}

	if opts.CollapseBlankLines {
		lines = collapseBlankLines(lines)
	}

	if opts.TrimBlankLines {
		lines = trimBlankLines(lines)
	}

	c.raw = strings.Builder{}
	c.raw.WriteString(strings.Join(lines, "\n"))
	c.raw.WriteString("\n")

	debug.V(3).Log("formatted config: \n--------------\n%s\n--------------", c.raw.String())

	return c.flushRaw()
}

// splitRawLines splits the raw config into lines. The raw config always ends
// with a newline, so the resulting empty last element is dropped.
func splitRawLines(raw string) []string {
	lines := strings.Split(raw, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return lines
}

// isBlankLine returns true if the line is empty or only contains whitespace.
func isBlankLine(line string) bool {
	return strings.TrimSpace(line) == ""
}

// isSectionHeaderLine returns true if the line is a section header, e.g.
// `[core]` or `[remote "origin"]`.
func isSectionHeaderLine(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "[")
}

// removeBlankLinesAfterSectionHeaders drops all blank lines that directly
// follow a section header.
func removeBlankLinesAfterSectionHeaders(lines []string) []string {
	out := make([]string, 0, len(lines))
	afterHeader := false
	for _, line := range lines {
		if isBlankLine(line) {
			if afterHeader {
				continue
			}
			out = append(out, line)

			continue
		}
		afterHeader = isSectionHeaderLine(line)
		out = append(out, line)
	}

	return out
}

// removeBlankLinesBeforeSectionHeaders drops all blank lines that directly
// precede a section header. Blank lines at the end of the file are kept.
func removeBlankLinesBeforeSectionHeaders(lines []string) []string {
	out := make([]string, 0, len(lines))
	pending := make([]string, 0, 4)
	for _, line := range lines {
		if isBlankLine(line) {
			pending = append(pending, line)

			continue
		}
		if !isSectionHeaderLine(line) {
			out = append(out, pending...)
		}
		pending = pending[:0]
		out = append(out, line)
	}
	out = append(out, pending...)

	return out
}

// collapseBlankLines replaces runs of consecutive blank lines with a single
// blank line.
func collapseBlankLines(lines []string) []string {
	out := make([]string, 0, len(lines))
	prevBlank := false
	for _, line := range lines {
		blank := isBlankLine(line)
		if blank && prevBlank {
			continue
		}
		out = append(out, line)
		prevBlank = blank
	}

	return out
}

// trimBlankLines removes blank lines at the beginning and the end of the file.
func trimBlankLines(lines []string) []string {
	start := 0
	for start < len(lines) && isBlankLine(lines[start]) {
		start++
	}
	end := len(lines)
	for end > start && isBlankLine(lines[end-1]) {
		end--
	}

	return lines[start:end]
}

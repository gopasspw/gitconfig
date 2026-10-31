package gitconfig

import (
	"slices"
	"strings"

	"github.com/gopasspw/gopass/pkg/debug"
)

// SortOptions controls how Config.Sort reorders the sections and keys of a
// config file. The zero value is a no-op, i.e. the raw config is left
// untouched.
//
// Sorting is opt-in. By default the library preserves the order of the
// original file, which matches the behaviour of the git CLI.
//
// Note that reordering sections can change the effective configuration, since
// for multi-valued keys the order of the values matters and include directives
// are processed in the order they appear. Only enable sorting if the resulting
// order is acceptable for the config at hand.
type SortOptions struct {
	// GroupSections moves all sections with the same name next to each other,
	// e.g. all [remote "..."] sections. The order in which the section names
	// first appear in the file is preserved, so [core] stays first if it was
	// first.
	GroupSections bool

	// SortSections sorts the sections alphabetically by section name and, for
	// sections with the same name, by subsection name. It implies
	// GroupSections.
	SortSections bool

	// SortKeys sorts the keys within each section alphabetically. Keys with
	// the same name keep their relative order, so multi-valued keys (e.g. the
	// multiple fetch lines of a remote) are not reordered among themselves.
	// Comment lines directly above a key move together with that key.
	SortKeys bool
}

// isZero returns true if no reordering was requested.
func (o SortOptions) isZero() bool {
	return !o.GroupSections && !o.SortSections && !o.SortKeys
}

// Sort reorders the sections and keys of the config according to opts and
// persists the result.
//
// The raw representation of the config is rewritten, so comments and
// whitespace are preserved as far as possible. Lines before the first section
// header are never reordered. Readonly configs silently ignore the operation,
// and NoWrites only prevents persisting the result to disk.
//
// Example:
//
//	cfg, err := gitconfig.LoadConfig(".git/config")
//	if err != nil {
//	  log.Fatal(err)
//	}
//	// group all [remote "..."] sections and sort the keys within each section
//	if err := cfg.Sort(gitconfig.SortOptions{GroupSections: true, SortKeys: true}); err != nil {
//	  log.Fatal(err)
//	}
func (c *Config) Sort(opts SortOptions) error {
	if c.readonly || opts.isZero() {
		return nil
	}

	blocks := splitConfigBlocks(c.raw.String())

	if opts.SortKeys {
		for _, b := range blocks {
			if b.preamble {
				continue
			}
			sortBlockKeys(b)
		}
	}

	blocks = orderBlocks(blocks, opts)

	c.raw = strings.Builder{}
	c.raw.WriteString(blocksToString(blocks))

	debug.V(3).Log("sorted config: \n--------------\n%s\n--------------", c.raw.String())

	return c.flushRaw()
}

// blocksToString serializes the given blocks back into the raw config format.
func blocksToString(blocks []*configBlock) string {
	var sb strings.Builder
	for _, b := range blocks {
		if b.header != "" {
			sb.WriteString(b.header)
			sb.WriteString("\n")
		}
		for _, chunk := range b.chunks {
			for _, line := range chunk {
				sb.WriteString(line)
				sb.WriteString("\n")
			}
		}
		for _, line := range b.trailing {
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// configBlock is a single section of a config file, i.e. the section header
// line and all lines belonging to it. Lines before the first section header
// are kept in a block with preamble set to true.
type configBlock struct {
	header     string
	section    string
	subsection string
	preamble   bool

	// chunks holds the key lines of the section. Each chunk is a key line
	// preceded by the comment lines that directly belong to it.
	chunks [][]string

	// trailing holds comments and blank lines after the last key of the
	// section, or all lines of the preamble.
	trailing []string
}

// splitConfigBlocks splits the raw config into its sections.
func splitConfigBlocks(raw string) []*configBlock {
	lines := splitRawLines(raw)

	blocks := make([]*configBlock, 0, 16)
	cur := &configBlock{preamble: true}
	blocks = append(blocks, cur)

	pending := make([]string, 0, len(lines))

	flushPending := func() {
		if len(pending) == 0 {
			return
		}
		cur.trailing = append(cur.trailing, pending...)
		pending = nil
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "[") {
			section, subsection, skip := parseSectionHeader(stripHeaderComment(trimmed))
			if skip {
				// keep unparsable headers in place
				cur.trailing = append(cur.trailing, line)

				continue
			}
			flushPending()
			cur = &configBlock{
				header:     line,
				section:    strings.ToLower(section),
				subsection: subsection,
			}
			blocks = append(blocks, cur)

			continue
		}

		if cur.preamble {
			cur.trailing = append(cur.trailing, line)

			continue
		}

		if isKeyLine(trimmed) {
			cur.chunks = append(cur.chunks, append(pending, line))
			pending = nil

			continue
		}

		pending = append(pending, line)
	}

	flushPending()

	return blocks
}

// isKeyLine returns true if the given trimmed line is a key-value or a bare
// boolean key. Any non-empty line that is not a comment is treated as a key
// line, so that lines we cannot parse stay in place instead of being moved
// around.
func isKeyLine(line string) bool {
	return line != "" && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, ";")
}

// stripHeaderComment removes a trailing comment from a section header line,
// e.g. `[core] # main settings` becomes `[core]`. Brackets inside a quoted
// subsection are ignored.
func stripHeaderComment(line string) string {
	quoted := false
	for i, r := range line {
		switch r {
		case '"':
			quoted = !quoted
		case ']':
			if !quoted {
				return line[:i+1]
			}
		}
	}

	return line
}

// chunkKey returns the lowercased key name of a chunk.
func chunkKey(chunk []string) string {
	k, _, _ := strings.Cut(strings.TrimSpace(chunk[len(chunk)-1]), "=")

	return strings.ToLower(strings.TrimSpace(k))
}

// sortBlockKeys sorts the keys of a block alphabetically. The sort is stable,
// so multi-valued keys keep their relative order.
func sortBlockKeys(b *configBlock) {
	slices.SortStableFunc(b.chunks, func(a, c []string) int {
		return strings.Compare(chunkKey(a), chunkKey(c))
	})
}

// orderBlocks returns the blocks in the requested order. Preamble blocks are
// always kept at the front.
func orderBlocks(blocks []*configBlock, opts SortOptions) []*configBlock {
	preamble := make([]*configBlock, 0, 1)
	sections := make([]*configBlock, 0, len(blocks))

	for _, b := range blocks {
		if b.preamble {
			preamble = append(preamble, b)

			continue
		}
		sections = append(sections, b)
	}

	switch {
	case opts.SortSections:
		slices.SortStableFunc(sections, func(a, b *configBlock) int {
			if n := strings.Compare(a.section, b.section); n != 0 {
				return n
			}

			return strings.Compare(a.subsection, b.subsection)
		})
	case opts.GroupSections:
		first := make(map[string]int, len(sections))
		for i, b := range sections {
			if _, found := first[b.section]; !found {
				first[b.section] = i
			}
		}
		slices.SortStableFunc(sections, func(a, b *configBlock) int {
			return first[a.section] - first[b.section]
		})
	}

	out := make([]*configBlock, 0, len(blocks))
	out = append(out, preamble...)
	out = append(out, sections...)

	return out
}

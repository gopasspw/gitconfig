package gitconfig

import (
	"strings"

	"github.com/gobwas/glob"
)

// globMatch matches a string against a glob pattern.
// It uses the gobwas/glob package and supports:
// - single-asterisk (*) patterns for matching within a path component
// - double-asterisk (**) patterns for matching across path components
// - question mark (?) for single character matching
// - character classes [abc] and ranges [a-z]
//
// The pattern uses '/' as a path separator.
//
// Example:
//
//	globMatch("feat/*", "feat/test") // returns (true, nil)
//	globMatch("feat/**", "feat/foo/bar") // returns (true, nil)
//
// Returns:
// - (true, nil) if the string matches the pattern
// - (false, nil) if the string does not match
// - (false, error) if the pattern is invalid
func globMatch(pattern, s string) (bool, error) {
	g, err := glob.Compile(pattern, '/')
	if err != nil {
		return false, err
	}

	return g.Match(s), nil
}

// splitKey splits a fully qualified gitconfig key into two or three parts.
// A valid key consists of either a section and a key separated by a dot
// or section, subsection and key, all separated by a dot. Note that
// the subsection might contain dots itself.
//
// Valid examples:
// - core.push
// - insteadof.git@github.com.push.
func splitKey(key string) (section, subsection, skey string) { //nolint:nonamedreturns
	n := strings.Index(key, ".")
	if n > 0 {
		section = key[:n]
	}

	if m := strings.LastIndex(key, "."); n != m && m > 0 && len(key) > m+1 {
		subsection = key[n+1 : m]
		skey = key[m+1:]

		return
	}

	skey = key[n+1:]

	return
}

// canonicalizeKey normalizes a gitconfig key according to git rules.
//
// Canonicalization rules (per git-config):
// - Section names are converted to lowercase
// - Subsection names are kept as-is (case-sensitive per git spec)
// - Key names are converted to lowercase
//
// Returns an empty string if the key is invalid (missing section or key part).
//
// Examples:
//
//	canonicalizeKey("Core.Push") returns "core.push"
//	canonicalizeKey("remote.Origin.URL") returns "remote.Origin.url"
//	canonicalizeKey("valid.key") returns "valid.key"
//	canonicalizeKey("invalid") returns "" // missing key part
func canonicalizeKey(key string) string {
	if key == "" {
		// invalid key, return empty string
		return ""
	}

	section, subsection, skey := splitKey(key)
	// "Section names are case-insensitive.""
	section = strings.ToLower(section)
	// "Subsection names are case sensitive."
	// "The variable names are case-insensitive."
	skey = strings.ToLower(skey)

	if section == "" || skey == "" {
		// invalid key, return empty string
		return ""
	}

	if subsection == "" {
		return section + "." + skey
	}

	return section + "." + subsection + "." + skey
}

// trim removes leading and trailing whitespace from all strings in the slice.
// It modifies the slice in-place.
//
// This is a convenience function for cleaning up parsed lines.
func trim(s []string) {
	for i, e := range s {
		s[i] = strings.TrimSpace(e)
	}
}

// parseLineForComment separates a line into content and comment parts.
//
// Parsing rules:
// - Searches for the first unquoted comment character (# or ;)
// - Ignores comment characters inside double-quoted strings
// - Trims whitespace from both content and comment
// - Removes surrounding double quotes from content
//
// The returned comment string does NOT include the delimiter character itself.
//
// Examples:
//
//	parseLineForComment(`value # comment`) returns ("value", "comment")
//	parseLineForComment(`"content # not-comment" # comment`) returns ("content # not-comment", "comment")
//	parseLineForComment(`plain-value`) returns ("plain-value", "")
//	parseLineForComment(`"quoted value"`) returns ("quoted value", "")
func parseLineForComment(line string) (string, string) {
	line = strings.TrimSpace(line) // Trim whitespace from the line first
	if !strings.HasPrefix(line, `"`) {
		// no properly quoted value string, we shouldn't have ended up here.
		if value, comment, found := strings.Cut(line, "#"); found {
			return strings.TrimSpace(value), strings.TrimSpace(comment)
		}
		if value, comment, found := strings.Cut(line, ";"); found {
			return strings.TrimSpace(value), strings.TrimSpace(comment)
		}

		// no comment found, return the line as is
		return line, ""
	}
	commentStartIndex := -1 // Initialize to -1, indicating comment not found yet
	inQuotes := false
	foundComment := false // Flag to signal when to break the loop

	// Iterate through the string to find the first unquoted comment character
	for i, r := range line {
		switch r {
		case '"':
			inQuotes = !inQuotes
		case '#', ';':
			if !inQuotes {
				commentStartIndex = i
				foundComment = true
			}
		}
		if foundComment {
			break // Exit the for loop
		}
	}

	content := ""
	comment := ""

	// Determine initial content and comment parts based on index
	var initialContent string
	if commentStartIndex != -1 {
		// Comment character was found
		initialContent = line[:commentStartIndex]
		// Extract comment text *after* the delimiter and trim whitespace
		if commentStartIndex+1 <= len(line) { // Check index bounds
			comment = strings.TrimSpace(line[commentStartIndex+1:])
		} else {
			comment = "" // Delimiter was the very last character
		}
	} else {
		// No unquoted comment character found
		initialContent = line
		comment = ""
	}

	// Trim whitespace from the initial content part FIRST
	trimmedContent := strings.TrimSpace(initialContent)
	// Now, check for and remove surrounding quotes from the trimmed content
	content = strings.Trim(trimmedContent, `"`)

	// Return the processed content and the processed comment part
	return content, comment
}

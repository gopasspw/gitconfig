//go:build examples
// +build examples

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/gopasspw/gitconfig"
)

// Example 10: Formatting the Layout
//
// This example demonstrates how to normalize the layout of a config file
// without changing any keys or values. Formatting is opt-in, the default is to
// preserve the original layout. It shows:
// - Removing trailing whitespace
// - Collapsing multiple blank lines into one
// - Dropping blank lines directly after a section header
// - Trimming blank lines at the beginning and the end of the file
func main() {
	tmpDir, err := os.MkdirTemp("", "gitconfig-example-")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, ".git", "config")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		log.Fatal(err)
	}

	initialConfig := `
[core]


	repositoryformatversion = 0
	filemode = true   

[remote "origin"]

	url = https://github.com/gopasspw/gitconfig.git
	fetch = +refs/heads/*:refs/remotes/origin/*


[alias]
	co = checkout

`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0o644); err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Example 10: Formatting the Layout ===\n")

	cfg, err := gitconfig.LoadConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Initial config:")
	fmt.Println(initialConfig)

	// Normalize the layout. Only the layout changes, keys and values stay the
	// same and comments are always preserved.
	err = cfg.Format(gitconfig.FormatOptions{
		TrimTrailingWhitespace:              true,
		CollapseBlankLines:                  true,
		RemoveBlankLinesAfterSectionHeader:  true,
		RemoveBlankLinesBeforeSectionHeader: true,
		TrimBlankLines:                      true,
	})
	if err != nil {
		log.Fatal(err)
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("After Format:")
	fmt.Println(string(content))

	fmt.Println("=== Summary ===")
	fmt.Println("Format is opt-in and only changes the layout of the file.")
	fmt.Println("Comments are preserved and the parsed values stay untouched.")
}

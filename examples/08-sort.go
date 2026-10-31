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

// Example 8: Reordering Sections and Keys
//
// This example demonstrates how to reorder the sections and keys of a config
// file. Reordering is opt-in, the default is to preserve the original order.
// It shows:
// - Grouping sections with the same name, e.g. all [remote "..."] sections
// - Sorting sections alphabetically
// - Sorting keys within each section
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

	initialConfig := `[core]
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
	if err := os.WriteFile(configPath, []byte(initialConfig), 0o644); err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Example 8: Reordering Sections and Keys ===\n")

	cfg, err := gitconfig.LoadConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Initial config:")
	fmt.Println(initialConfig)

	// Group all [remote "..."] sections and sort the keys within each section.
	// The order in which section names first appear is preserved.
	err = cfg.Sort(gitconfig.SortOptions{
		GroupSections: true,
		SortKeys:      true,
	})
	if err != nil {
		log.Fatal(err)
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("After grouping sections and sorting keys:")
	fmt.Println(string(content))

	// SortSections additionally sorts the sections alphabetically.
	if err := cfg.Sort(gitconfig.SortOptions{SortSections: true}); err != nil {
		log.Fatal(err)
	}

	content, err = os.ReadFile(configPath)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("After sorting sections alphabetically:")
	fmt.Println(string(content))

	fmt.Println("=== Summary ===")
	fmt.Println("Sort is opt-in and preserves comments and whitespace.")
	fmt.Println("Note that reordering can change the effective configuration,")
	fmt.Println("since the order of multi-valued keys and includes matters.")
}

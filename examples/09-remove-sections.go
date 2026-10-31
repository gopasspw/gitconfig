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

// Example 9: Removing Sections
//
// This example demonstrates how to remove whole sections from a config file.
// Unset only removes individual keys and may leave an empty section header
// behind. It shows:
// - Removing a section and all of its keys with UnsetSection
// - Dropping empty sections with PruneEmptySections
// - The accepted section formats
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
[remote "origin"]
	url = https://github.com/gopasspw/gitconfig.git
	fetch = +refs/heads/*:refs/remotes/origin/*
[remote "github"]
	url = https://github.com/gopasspw/gitconfig
[alias]
	co = checkout
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0o644); err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Example 9: Removing Sections ===\n")

	cfg, err := gitconfig.LoadConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Initial config:")
	fmt.Println(initialConfig)

	// Remove a whole section, including all of its keys. The section can be
	// given as "section", "section.subsection" or as a section header.
	if err := cfg.UnsetSection("remote.origin"); err != nil {
		log.Fatal(err)
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("After UnsetSection(\"remote.origin\"):")
	fmt.Println(string(content))

	// Unset only removes individual keys, which may leave an empty section
	// header behind. PruneEmptySections drops all sections without keys.
	if err := cfg.Unset("alias.co"); err != nil {
		log.Fatal(err)
	}
	if err := cfg.PruneEmptySections(); err != nil {
		log.Fatal(err)
	}

	content, err = os.ReadFile(configPath)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("After Unset(\"alias.co\") and PruneEmptySections():")
	fmt.Println(string(content))

	fmt.Println("=== Summary ===")
	fmt.Println("UnsetSection removes a section and all of its keys in one go.")
	fmt.Println("PruneEmptySections cleans up sections left empty by Unset.")
}

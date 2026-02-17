//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/gopasspw/gitconfig"
)

// Example 1: Basic Read
//
// This example demonstrates how to read configuration values from a git config file.
// It shows:
// - Loading a config file
// - Reading string values
// - Handling missing keys
func main() {
	// Create a temporary git config file for this example
	tmpDir, err := os.MkdirTemp("", "gitconfig-example-")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, ".git", "config")
	os.MkdirAll(filepath.Dir(configPath), 0755)

	// Write a sample config file
	sampleConfig := `[user]
    name = John Doe
    email = john@example.com
[core]
    editor = vim
    pager = less
[branch "main"]
    remote = origin
    merge = refs/heads/main
`
	err = os.WriteFile(configPath, []byte(sampleConfig), 0644)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Example 1: Basic Read ===\n")

	// Load the config file
	cfg, err := gitconfig.NewConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}

	// Read simple values
	fmt.Println("Reading simple values:")
	if name, ok := cfg.Get("user.name"); ok {
		fmt.Printf("  user.name = %s\n", name)
	}

	if email, ok := cfg.Get("user.email"); ok {
		fmt.Printf("  user.email = %s\n", email)
	}

	if editor, ok := cfg.Get("core.editor"); ok {
		fmt.Printf("  core.editor = %s\n", editor)
	}

	// Try to read a non-existent key
	fmt.Println("\nAttempting to read non-existent key:")
	if value, ok := cfg.Get("nonexistent.key"); ok {
		fmt.Printf("  nonexistent.key = %s\n", value)
	} else {
		fmt.Println("  nonexistent.key not found (this is expected)")
	}

	// Read values from subsections (branch)
	fmt.Println("\nReading from subsections (branch.main.*):")
	if remote, ok := cfg.Get("branch.main.remote"); ok {
		fmt.Printf("  branch.main.remote = %s\n", remote)
	}

	if merge, ok := cfg.Get("branch.main.merge"); ok {
		fmt.Printf("  branch.main.merge = %s\n", merge)
	}

	fmt.Println("\n=== Summary ===")
	fmt.Println("Config file loaded and values read successfully.")
	fmt.Println("Use cfg.Get(key) to retrieve values.")
	fmt.Println("Returns (value, ok) - check ok to see if key exists.")
}

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

// Example 2: Write and Persist
//
// This example demonstrates how to modify configuration values and persist
// those changes back to the config file.
// It shows:
// - Setting configuration values
// - Persisting changes to disk
// - Verifying changes
func main() {
	// Create a temporary git config file for this example
	tmpDir, err := os.MkdirTemp("", "gitconfig-example-")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, ".git", "config")
	os.MkdirAll(filepath.Dir(configPath), 0o755)

	// Write initial config
	initialConfig := `[user]
    name = John Doe
[core]
    editor = vim
`
	err = os.WriteFile(configPath, []byte(initialConfig), 0o644)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Example 2: Write and Persist ===\n")

	// Load the config file
	cfg, err := gitconfig.NewConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Initial config:")
	printValues(cfg, []string{"user.name", "user.email", "core.editor", "core.pager"})

	// Modify values
	fmt.Println("\nModifying configuration...")
	cfg.Set("user.email", "john.doe@example.com")
	cfg.Set("core.pager", "less -R")
	cfg.Set("core.autocrlf", "false")

	fmt.Println("After modifications (in memory):")
	printValues(cfg, []string{"user.name", "user.email", "core.editor", "core.pager", "core.autocrlf"})

	// Persist changes to disk
	fmt.Println("\nPersisting changes to disk...")
	err = cfg.Write()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Changes persisted successfully!")

	// Reload from disk to verify
	fmt.Println("\nReloading config from disk...")
	cfg2, err := gitconfig.NewConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Config after reload (verifying persistence):")
	printValues(cfg2, []string{"user.name", "user.email", "core.editor", "core.pager", "core.autocrlf"})

	// Print the actual file contents
	fmt.Println("\nActual file contents:")
	content, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(content))

	fmt.Println("=== Summary ===")
	fmt.Println("Configuration values can be modified and persisted using Set() and Write().")
	fmt.Println("The library preserves formatting of the original file.")
}

func printValues(cfg *gitconfig.Config, keys []string) {
	for _, key := range keys {
		if value, ok := cfg.Get(key); ok {
			fmt.Printf("  %s = %s\n", key, value)
		}
	}
}

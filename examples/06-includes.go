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

// Example 6: Include Files
//
// This example demonstrates how gitconfig handles include directives.
// Git config supports including other config files:
//
//	[include]
//	    path = /path/to/other/config
//
// This enables modular configuration and code reuse.
func main() {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "gitconfig-example-")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	fmt.Println("=== Example 6: Include Files ===\n")

	// Create included config files
	commonPath := filepath.Join(tmpDir, "config-common")
	projectPath := filepath.Join(tmpDir, "config-project")
	mainPath := filepath.Join(tmpDir, "config-main")

	// Common settings (included by all projects)
	err = os.WriteFile(commonPath, []byte(`[core]
    # Common settings used across all projects
    sshCommand = ssh -i ~/.ssh/id_ed25519
    autocrlf = input
[init]
    defaultBranch = main
`), 0644)
	if err != nil {
		log.Fatal(err)
	}

	// Project-specific settings (included by main config)
	err = os.WriteFile(projectPath, []byte(`[user]
    name = Project Team
    email = team@project.com
[feature]
    enabled = true
`), 0644)
	if err != nil {
		log.Fatal(err)
	}

	// Main configuration that includes others
	err = os.WriteFile(mainPath, []byte(fmt.Sprintf(`# Main project config
[include]
    path = %s
    path = %s
[user]
    signingkey = ~/.ssh/id_ed25519.pub
[commit]
    gpgsign = true
`, commonPath, projectPath)), 0644)
	if err != nil {
		log.Fatal(err)
	}

	// Load the main config (which includes others)
	cfg, err := gitconfig.NewConfig(mainPath)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Configuration structure:")
	fmt.Printf("  Main config: %s\n", mainPath)
	fmt.Printf("  Includes: %s\n", commonPath)
	fmt.Printf("             %s\n", projectPath)

	fmt.Println("\nResolved configuration values:")

	// Values from common config (included)
	values := []string{
		"core.sshCommand",
		"core.autocrlf",
		"init.defaultBranch",
	}

	for _, key := range values {
		if v, ok := cfg.Get(key); ok {
			fmt.Printf("  %s = %s (from common config)\n", key, v)
		}
	}

	// Values from project config (included)
	values = []string{
		"user.name",
		"user.email",
		"feature.enabled",
	}

	for _, key := range values {
		if v, ok := cfg.Get(key); ok {
			fmt.Printf("  %s = %s (from project config)\n", key, v)
		}
	}

	// Values from main config itself
	values = []string{
		"user.signingkey",
		"commit.gpgsign",
	}

	for _, key := range values {
		if v, ok := cfg.Get(key); ok {
			fmt.Printf("  %s = %s (from main config)\n", key, v)
		}
	}

	fmt.Println("\n=== Include Path Resolution ===")
	fmt.Println("Include paths are relative to the config file location.")
	fmt.Println("Absolute paths are also supported.")
	fmt.Println("Included files can themselves include other files.")
	fmt.Println("Later includes override earlier values (last-write-wins).")

	fmt.Println("\n=== Real-world Example ===")
	fmt.Println("Typical Git config organization:")
	fmt.Println("  ~/.gitconfig           # Global user config")
	fmt.Println("  ~/.gitconfig-work      # Work-specific settings")
	fmt.Println("  ~/.gitconfig-personal  # Personal project settings")
	fmt.Println("")
	fmt.Println("With entries in ~/.gitconfig:")
	fmt.Println("  [include]")
	fmt.Println("      path = ~/.gitconfig-work")
	fmt.Println("      path = ~/.gitconfig-personal")

	fmt.Println("\n=== Summary ===")
	fmt.Println("Include directives enable modular configuration.")
	fmt.Println("Common patterns in included files:")
	fmt.Println("  - Base settings (core options)")
	fmt.Println("  - User-specific settings")
	fmt.Println("  - Project-specific settings")
	fmt.Println("  - Environment-specific settings (dev/prod)")
	fmt.Println("")
	fmt.Println("Git processes includes in order they appear.")
	fmt.Println("Later values override earlier ones from different files.")
}

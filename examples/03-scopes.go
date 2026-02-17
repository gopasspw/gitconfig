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

// Example 3: Understanding Scopes
//
// This example demonstrates the configuration scope hierarchy.
// Git config has multiple scopes with a clear precedence order:
//  1. Environment variables (highest priority)
//  2. Per-worktree config
//  3. Per-repository config (local)
//  4. Per-user config (global)
//  5. System-wide config
//  6. Presets (built-in defaults, lowest priority)
//
// When you call Get(), the library searches through scopes in order
// and returns the first value it finds.
func main() {
	// Create temporary directories for different config scopes
	tmpDir, err := os.MkdirTemp("", "gitconfig-example-")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	if err := os.Setenv("GOPASS_HOMEDIR", tmpDir); err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = os.Unsetenv("GOPASS_HOMEDIR")
	}()

	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Example 3: Understanding Scopes ===\n")

	// Create sample config files for different scopes
	systemConfig := filepath.Join(tmpDir, "gitconfig-system")
	globalConfig := filepath.Join(tmpDir, "gitconfig-global")
	localConfig := filepath.Join(gitDir, "config")

	// System config (lowest priority among files)
	err = os.WriteFile(systemConfig, []byte(`[user]
	name = System User
	email = system@example.com
[core]
	pager = less
`), 0o644)
	if err != nil {
		log.Fatal(err)
	}

	// Global/user config
	err = os.WriteFile(globalConfig, []byte(`[user]
	name = Global User
	email = global@example.com
[core]
	editor = emacs
`), 0o644)
	if err != nil {
		log.Fatal(err)
	}

	// Local/repository config (highest priority among files)
	err = os.WriteFile(localConfig, []byte(`[user]
	name = Local User
[core]
	autocrlf = true
`), 0o644)
	if err != nil {
		log.Fatal(err)
	}

	// Create a Configs object that loads all scopes
	// Note: This example uses custom paths since we don't have a real system setup
	cfg := gitconfig.New()

	// Manually customize paths for this example
	cfg.SystemConfig = systemConfig
	cfg.GlobalConfig = "gitconfig-global"
	cfg.LocalConfig = filepath.Join(".git", "config")

	fmt.Println("Config scope hierarchy (highest to lowest priority):")
	fmt.Println("  1. Environment variables")
	fmt.Println("  2. Per-worktree config")
	fmt.Println("  3. Local (per-repository) config")
	fmt.Println("  4. Global (per-user) config")
	fmt.Println("  5. System-wide config")
	fmt.Println("  6. Presets (built-in defaults)")

	// Load and display
	cfg.LoadAll(tmpDir)

	fmt.Println("\nResolved values (respecting scope priority):")
	fmt.Println("  user.name =", getOrDefault(cfg, "user.name"))
	fmt.Println("    ^ Comes from local config (highest priority)")
	fmt.Println("  user.email =", getOrDefault(cfg, "user.email"))
	fmt.Println("    ^ Comes from global config (not overridden locally)")
	fmt.Println("  core.editor =", getOrDefault(cfg, "core.editor"))
	fmt.Println("    ^ Comes from global config")
	fmt.Println("  core.pager =", getOrDefault(cfg, "core.pager"))
	fmt.Println("    ^ Comes from system config (not overridden)")
	fmt.Println("  core.autocrlf =", getOrDefault(cfg, "core.autocrlf"))
	fmt.Println("    ^ Comes from local config")

	// Show how to access specific scopes directly
	fmt.Println("\nAccessing specific scopes directly:")

	local, err := gitconfig.LoadConfig(localConfig)
	if err == nil {
		if name, ok := local.Get("user.name"); ok {
			fmt.Printf("  local user.name = %s\n", name)
		}
	}

	global, err := gitconfig.LoadConfig(globalConfig)
	if err == nil {
		if editor, ok := global.Get("core.editor"); ok {
			fmt.Printf("  global core.editor = %s\n", editor)
		}
	}

	fmt.Println("\n=== Summary ===")
	fmt.Println("Git has multiple config scopes with clear precedence.")
	fmt.Println("Use Configs.Get() to read values respecting all scopes.")
	fmt.Println("Use GetLocal(), GetGlobal(), etc. to read from specific scopes.")
	fmt.Println("Use SetLocal(), SetGlobal(), etc. to write to specific scopes.")
}

func getOrDefault(cfg *gitconfig.Configs, key string) string {
	if v := cfg.Get(key); v != "" {
		return v
	}
	return "(not set)"
}

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

// Example 5: Error Handling
//
// This example demonstrates proper error handling patterns when working
// with gitconfig. Common errors include:
// - File not found
// - Permission errors
// - Parse errors
// - Invalid key formats
func main() {
	fmt.Println("=== Example 5: Error Handling ===\n")

	// Error 1: File not found
	fmt.Println("Error 1: File not found")
	cfg, err := gitconfig.NewConfig("/nonexistent/path/.git/config")
	if err != nil {
		fmt.Printf("  Expected error: %v\n", err)
	}

	// Error 2: Permission denied
	fmt.Println("\nError 2: Permission denied")
	tmpDir, _ := os.MkdirTemp("", "gitconfig-example-")
	defer os.RemoveAll(tmpDir)

	restrictedPath := filepath.Join(tmpDir, "restricted-config")
	os.WriteFile(restrictedPath, []byte("[user]\n    name = Test"), 0644)
	os.Chmod(restrictedPath, 0000) // Remove all permissions

	cfg, err = gitconfig.NewConfig(restrictedPath)
	if err != nil {
		fmt.Printf("  Expected error: %v\n", err)
	}
	os.Chmod(restrictedPath, 0644) // Restore permissions for cleanup

	// Error 3: Parse error
	fmt.Println("\nError 3: Parse error (invalid config syntax)")
	badConfigPath := filepath.Join(tmpDir, "bad-config")
	os.WriteFile(badConfigPath, []byte(`[user
    name = John
`), 0644) // Missing closing bracket

	cfg, err = gitconfig.NewConfig(badConfigPath)
	if err != nil {
		fmt.Printf("  Parse error detected: %v\n", err)
	}

	// Error 4: Write error (permission denied)
	fmt.Println("\nError 4: Write error (permission denied)")
	writePath := filepath.Join(tmpDir, "write-test")
	os.WriteFile(writePath, []byte("[user]\n    name = Test"), 0644)

	cfg, err = gitconfig.NewConfig(writePath)
	if err == nil {
		cfg.Set("user.email", "test@example.com")

		os.Chmod(tmpDir, 0000) // Remove write permissions
		err = cfg.Write()
		if err != nil {
			fmt.Printf("  Expected write error: %v\n", err)
		}
		os.Chmod(tmpDir, 0755) // Restore permissions
	}

	// Error 5: Graceful error handling pattern
	fmt.Println("\nError 5: Graceful error handling pattern")
	goodConfigPath := filepath.Join(tmpDir, "good-config")
	os.WriteFile(goodConfigPath, []byte(`[user]
    name = John Doe
    email = john@example.com
[core]
    editor = vim
`), 0644)

	// Pattern: Load with error checking
	cfg, err = gitconfig.NewConfig(goodConfigPath)
	if err != nil {
		fmt.Printf("  Failed to load config: %v\n", err)
		fmt.Println("  Continuing with fallback...")
		return
	}

	// Pattern: Read with existence check
	name, ok := cfg.Get("user.name")
	if !ok {
		fmt.Println("  user.name not found, using default")
		name = "Unknown"
	}
	fmt.Printf("  user.name = %s\n", name)

	// Pattern: Write with error checking
	cfg.Set("user.email", "newemail@example.com")
	err = cfg.Write()
	if err != nil {
		fmt.Printf("  Failed to write config: %v\n", err)
		log.Printf("Warning: Could not persist changes")
	} else {
		fmt.Println("  Changes persisted successfully")
	}

	// Error 6: Multi-scope errors
	fmt.Println("\nError 6: Multi-scope errors (Configs)")
	configs := gitconfig.NewConfigs()

	// Set paths to non-existent files (this is okay for Configs)
	configs.SetConfigPath(gitconfig.ConfigLocal, filepath.Join(tmpDir, "nonexistent-local"))
	configs.SetConfigPath(gitconfig.ConfigGlobal, filepath.Join(tmpDir, "nonexistent-global"))

	// LoadAll might handle missing files differently
	err = configs.LoadAll()
	if err != nil {
		fmt.Printf("  LoadAll error: %v\n", err)
	} else {
		fmt.Println("  LoadAll succeeded (skipped missing files)")
	}

	fmt.Println("\n=== Common Error Patterns ===")
	fmt.Println("1. Check errors after NewConfig() and LoadAll()")
	fmt.Println("2. Use if ok := cfg.Get(key) pattern for optional values")
	fmt.Println("3. Check Write() errors when persisting changes")
	fmt.Println("4. Handle permission errors gracefully")
	fmt.Println("5. Provide fallback values for critical config")

	fmt.Println("\n=== Summary ===")
	fmt.Println("Always check errors when:")
	fmt.Println("  - Loading config files (might not exist or be unreadable)")
	fmt.Println("  - Writing config files (might lose permissions)")
	fmt.Println("  - Parsing config (malformed files)")
	fmt.Println("Use (value, ok) pattern for optional config values.")
}

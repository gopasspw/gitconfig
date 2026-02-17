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
	cfg, err := gitconfig.LoadConfig("/nonexistent/path/.git/config")
	if err != nil {
		fmt.Printf("  Expected error: %v\n", err)
	}

	// Error 2: Permission denied
	fmt.Println("\nError 2: Permission denied")
	tmpDir, _ := os.MkdirTemp("", "gitconfig-example-")
	defer os.RemoveAll(tmpDir)

	restrictedPath := filepath.Join(tmpDir, "restricted-config")
	os.WriteFile(restrictedPath, []byte("[user]\n    name = Test"), 0o644)
	os.Chmod(restrictedPath, 0o000) // Remove all permissions

	cfg, err = gitconfig.LoadConfig(restrictedPath)
	if err != nil {
		fmt.Printf("  Expected error: %v\n", err)
	}
	os.Chmod(restrictedPath, 0o644) // Restore permissions for cleanup

	// Error 3: Parse error
	fmt.Println("\nError 3: Parse error (invalid config syntax)")
	badConfigPath := filepath.Join(tmpDir, "bad-config")
	os.WriteFile(badConfigPath, []byte(`[user
    name = John
`), 0o644) // Missing closing bracket

	cfg, err = gitconfig.LoadConfig(badConfigPath)
	if err != nil {
		fmt.Printf("  Parse error detected: %v\n", err)
	}

	// Error 4: Write error (permission denied)
	fmt.Println("\nError 4: Write error (permission denied)")
	writePath := filepath.Join(tmpDir, "write-test")
	os.WriteFile(writePath, []byte("[user]\n    name = Test"), 0o644)

	cfg, err = gitconfig.LoadConfig(writePath)
	if err == nil {
		os.Chmod(tmpDir, 0o000) // Remove write permissions
		err = cfg.Set("user.email", "test@example.com")
		if err != nil {
			fmt.Printf("  Expected write error: %v\n", err)
		}
		os.Chmod(tmpDir, 0o755) // Restore permissions
	}

	// Error 5: Graceful error handling pattern
	fmt.Println("\nError 5: Graceful error handling pattern")
	goodConfigPath := filepath.Join(tmpDir, "good-config")
	os.WriteFile(goodConfigPath, []byte(`[user]
    name = John Doe
    email = john@example.com
[core]
    editor = vim
`), 0o644)

	// Pattern: Load with error checking
	cfg, err = gitconfig.LoadConfig(goodConfigPath)
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
	if err := cfg.Set("user.email", "newemail@example.com"); err != nil {
		fmt.Printf("  Failed to write config: %v\n", err)
		log.Printf("Warning: Could not persist changes")
	} else {
		fmt.Println("  Changes persisted successfully")
	}

	// Error 6: Multi-scope errors
	fmt.Println("\nError 6: Multi-scope errors (Configs)")
	configs := gitconfig.New()
	if err := os.Setenv("GOPASS_HOMEDIR", tmpDir); err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = os.Unsetenv("GOPASS_HOMEDIR")
	}()

	// Set paths to non-existent files (this is okay for Configs)
	configs.LocalConfig = filepath.Join(".git", "config")
	configs.GlobalConfig = "nonexistent-global"

	// LoadAll handles missing files gracefully
	configs.LoadAll(tmpDir)
	fmt.Println("  LoadAll succeeded (skipped missing files)")

	fmt.Println("\n=== Common Error Patterns ===")
	fmt.Println("1. Check errors after LoadConfig() and LoadAll()")
	fmt.Println("2. Use if ok := cfg.Get(key) pattern for optional values")
	fmt.Println("3. Check Set() errors when persisting changes")
	fmt.Println("4. Handle permission errors gracefully")
	fmt.Println("5. Provide fallback values for critical config")

	fmt.Println("\n=== Summary ===")
	fmt.Println("Always check errors when:")
	fmt.Println("  - Loading config files (might not exist or be unreadable)")
	fmt.Println("  - Updating config files (might lose permissions)")
	fmt.Println("  - Parsing config (malformed files)")
	fmt.Println("Use (value, ok) pattern for optional config values.")
}

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

// Example 4: Custom Paths
//
// This example demonstrates how to work with configuration files
// at custom locations instead of using the default Git paths.
// This is useful when:
// - Working with non-standard directory structures
// - Testing with temporary configs
// - Integrating with systems that use similar config formats
func main() {
	// Create temporary directory for this example
	tmpDir, err := os.MkdirTemp("", "gitconfig-example-")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	fmt.Println("=== Example 4: Custom Paths ===\n")

	// Example 1: Load config from custom location
	customPath1 := filepath.Join(tmpDir, "my-config")
	err = os.WriteFile(customPath1, []byte(`[app]
    name = MyApp
    version = 1.0
[features]
    logging = true
    debug = false
`), 0644)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Loading config from custom path:")
	fmt.Printf("  Path: %s\n", customPath1)

	cfg1, err := gitconfig.NewConfig(customPath1)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("  Values:")
	if name, ok := cfg1.Get("app.name"); ok {
		fmt.Printf("    app.name = %s\n", name)
	}
	if version, ok := cfg1.Get("app.version"); ok {
		fmt.Printf("    app.version = %s\n", version)
	}

	// Example 2: Multiple custom config files
	fmt.Println("\nWorking with multiple custom configs:")

	configA := filepath.Join(tmpDir, "config-a")
	configB := filepath.Join(tmpDir, "config-b")

	err = os.WriteFile(configA, []byte(`[database]
    host = localhost
    port = 5432
`), 0644)
	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile(configB, []byte(`[database]
    user = admin
    password = secret
[cache]
    enabled = true
`), 0644)
	if err != nil {
		log.Fatal(err)
	}

	cfgA, err := gitconfig.NewConfig(configA)
	if err != nil {
		log.Fatal(err)
	}

	cfgB, err := gitconfig.NewConfig(configB)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("  Config A (database.host):")
	if host, ok := cfgA.Get("database.host"); ok {
		fmt.Printf("    %s\n", host)
	}

	fmt.Println("  Config B (database.user):")
	if user, ok := cfgB.Get("database.user"); ok {
		fmt.Printf("    %s\n", user)
	}

	fmt.Println("  Config B (cache.enabled):")
	if enabled, ok := cfgB.Get("cache.enabled"); ok {
		fmt.Printf("    %s\n", enabled)
	}

	// Example 3: Creating and modifying custom config
	fmt.Println("\nCreating and modifying custom config:")

	customPath2 := filepath.Join(tmpDir, "new-config")
	fmt.Printf("  Creating new config at: %s\n", customPath2)

	// Create empty config in memory (doesn't need to exist yet)
	cfg3, err := gitconfig.NewConfig(customPath2)
	if err != nil {
		log.Fatal(err)
	}

	// Add values
	cfg3.Set("app.name", "NewApp")
	cfg3.Set("app.version", "2.0")
	cfg3.Set("environment", "production")

	// Write to disk
	err = cfg3.Write()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("  Config written successfully!")

	// Verify by loading it back
	cfg3Reloaded, err := gitconfig.NewConfig(customPath2)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("  Verifying persisted values:")
	if name, ok := cfg3Reloaded.Get("app.name"); ok {
		fmt.Printf("    app.name = %s\n", name)
	}
	if ver, ok := cfg3Reloaded.Get("app.version"); ok {
		fmt.Printf("    app.version = %s\n", ver)
	}
	if env, ok := cfg3Reloaded.Get("environment"); ok {
		fmt.Printf("    environment = %s\n", env)
	}

	fmt.Println("\n=== Summary ===")
	fmt.Println("NewConfig() accepts any file path as an argument.")
	fmt.Println("This allows using gitconfig for non-Git applications.")
	fmt.Println("Useful for config files with git-config format.")
}

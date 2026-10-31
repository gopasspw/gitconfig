//go:build examples
// +build examples

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/gopasspw/gitconfig"
)

// Example 7: Iterating Keys
//
// This example demonstrates how to enumerate configuration keys, either across
// all scopes or within a single scope:
//
//   - Configs.Keys() lists the union of all keys from all scopes, sorted
//   - Configs.List(prefix) filters that list by a prefix
//   - Configs.Scope(name) returns the live *Config of a single scope
//   - Config.Keys() and Config.Entries() enumerate the keys and values of a
//     single scope. Entries returns a deep copy.
func main() {
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

	fmt.Println("=== Example 7: Iterating Keys ===\n")

	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		log.Fatal(err)
	}

	localConfig := filepath.Join(gitDir, "config")
	if err := os.WriteFile(localConfig, []byte(`[core]
	repositoryformatversion = 0
[remote "origin"]
	url = https://github.com/gopasspw/gitconfig.git
	fetch = +refs/heads/*:refs/remotes/origin/*
	fetch = +refs/pull/*/head:refs/remotes/origin/pr/*
`), 0o600); err != nil {
		log.Fatal(err)
	}

	// GOPASS_HOMEDIR points at tmpDir, so the global config is looked up there
	// and we never touch the user's real configuration.
	cfg := gitconfig.New()
	cfg.SystemConfig = filepath.Join(tmpDir, "gitconfig-system")
	cfg.EnvPrefix = "GITCONFIG_EXAMPLE"
	cfg.NoWrites = true
	cfg.LoadAll(gitDir)

	fmt.Println("All keys across all scopes:")
	for _, key := range cfg.Keys() {
		fmt.Printf("  %-40s = %s\n", key, cfg.Get(key))
	}

	fmt.Println("\nKeys with the prefix \"remote.\":")
	for _, key := range cfg.List("remote.") {
		fmt.Printf("  %s\n", key)
	}

	local, ok := cfg.Scope("local")
	if !ok {
		log.Fatal("no local scope available")
	}

	fmt.Println("\nAll keys and values of the local scope:")
	entries := local.Entries()
	keys := make([]string, 0, len(entries))
	for key := range entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		fmt.Printf("  %-40s = %v\n", key, entries[key])
	}

	fmt.Println("\nEntries is a copy, mutations to it do not affect the config:")
	entries["remote.origin.fetch"][0] = "mutated"
	actual, _ := local.GetAll("remote.origin.fetch")
	fmt.Printf("  mutated copy:  %v\n", entries["remote.origin.fetch"])
	fmt.Printf("  actual config: %v\n", actual)
}

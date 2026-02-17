package gitconfig

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConcurrentReads tests that multiple goroutines can safely read from the same config.
func TestConcurrentReads(t *testing.T) {
	t.Parallel()

	td := t.TempDir()
	configPath := filepath.Join(td, "config")

	// Create a config with multiple keys
	content := `[user]
	name = John Doe
	email = john@example.com
[core]
	editor = vim
	autocrlf = true
	filemode = false
[remote "origin"]
	url = https://github.com/test/repo.git
	fetch = +refs/heads/*:refs/remotes/origin/*
`
	err := os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Launch multiple goroutines reading different keys
	var wg sync.WaitGroup
	iterations := 100
	goroutines := 10

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for i := 0; i < iterations; i++ {
				// Each goroutine reads different keys based on its ID
				switch id % 3 {
				case 0:
					name, ok := cfg.Get("user.name")
					assert.True(t, ok)
					assert.Equal(t, "John Doe", name)
				case 1:
					editor, ok := cfg.Get("core.editor")
					assert.True(t, ok)
					assert.Equal(t, "vim", editor)
				case 2:
					url, ok := cfg.Get("remote.origin.url")
					assert.True(t, ok)
					assert.Equal(t, "https://github.com/test/repo.git", url)
				}
			}
		}(g)
	}

	wg.Wait()
}

// TestConcurrentLoad tests that loading multiple configs concurrently is safe.
func TestConcurrentLoad(t *testing.T) {
	t.Parallel()

	td := t.TempDir()

	// Create multiple config files
	configs := make([]string, 5)
	for i := 0; i < len(configs); i++ {
		configPath := filepath.Join(td, "config"+string(rune('0'+i)))
		content := "[user]\n\tname = User" + string(rune('0'+i))
		err := os.WriteFile(configPath, []byte(content), 0o644)
		require.NoError(t, err)
		configs[i] = configPath
	}

	// Load all configs concurrently
	var wg sync.WaitGroup
	results := make([]*Config, len(configs))
	errors := make([]error, len(configs))

	for i := 0; i < len(configs); i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			cfg, err := LoadConfig(configs[index])
			results[index] = cfg
			errors[index] = err
		}(i)
	}

	wg.Wait()

	// Verify all loads succeeded
	for i := 0; i < len(configs); i++ {
		require.NoError(t, errors[i], "config %d should load without error", i)
		require.NotNil(t, results[i], "config %d should not be nil", i)

		name, ok := results[i].Get("user.name")
		assert.True(t, ok)
		assert.Equal(t, "User"+string(rune('0'+i)), name)
	}
}

// TestConcurrentReadsSameKey tests race conditions when reading the same key.
func TestConcurrentReadsSameKey(t *testing.T) {
	t.Parallel()

	td := t.TempDir()
	configPath := filepath.Join(td, "config")

	content := "[user]\n\tname = Concurrent Test"
	err := os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Many goroutines reading the same key
	var wg sync.WaitGroup
	iterations := 50
	goroutines := 20

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				name, ok := cfg.Get("user.name")
				assert.True(t, ok)
				assert.Equal(t, "Concurrent Test", name)
			}
		}()
	}

	wg.Wait()
}

// TestConcurrentGetAll tests concurrent access to multi-valued keys.
func TestConcurrentGetAll(t *testing.T) {
	t.Parallel()

	td := t.TempDir()
	configPath := filepath.Join(td, "config")

	content := `[remote "origin"]
	fetch = +refs/heads/*:refs/remotes/origin/*
	fetch = +refs/tags/*:refs/tags/*
	fetch = +refs/pull/*/head:refs/remotes/origin/pr/*
`
	err := os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	var wg sync.WaitGroup
	iterations := 50
	goroutines := 10

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				values, ok := cfg.GetAll("remote.origin.fetch")
				assert.True(t, ok)
				assert.Equal(t, 3, len(values))
			}
		}()
	}

	wg.Wait()
}

// TestSerialWrites tests that writes are properly serialized (no concurrent write support expected).
func TestSerialWrites(t *testing.T) {
	t.Parallel()

	td := t.TempDir()
	configPath := filepath.Join(td, "config")

	content := "[user]\n\tname = Initial"
	err := os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	// Load separate config instances for each write
	configs := make([]*Config, 5)
	for i := 0; i < len(configs); i++ {
		cfg, err := LoadConfig(configPath)
		require.NoError(t, err)
		configs[i] = cfg
	}

	// Write sequentially (not concurrently, as that would cause data loss)
	// Set automatically writes to disk
	for i, cfg := range configs {
		err := cfg.Set("user.id", string(rune('0'+i)))
		require.NoError(t, err)
	}

	// Load final state
	finalCfg, err := LoadConfig(configPath)
	require.NoError(t, err)

	// Last write should win
	id, ok := finalCfg.Get("user.id")
	assert.True(t, ok)
	assert.Equal(t, "4", id)
}

// TestConcurrentMultiScopeReads tests concurrent reads across multiple scopes.
func TestConcurrentMultiScopeReads(t *testing.T) {
	// Note: not using t.Parallel() because we need t.Setenv()

	td := t.TempDir()
	t.Setenv("GOPASS_HOMEDIR", td)

	gitDir := filepath.Join(td, ".git")
	require.NoError(t, os.MkdirAll(gitDir, 0o755))

	// Create local config
	localPath := filepath.Join(gitDir, "config")
	localContent := "[user]\n\tname = Local User\n\temail = local@example.com"
	err := os.WriteFile(localPath, []byte(localContent), 0o644)
	require.NoError(t, err)

	// Create global config
	globalPath := filepath.Join(td, "global-config")
	globalContent := "[user]\n\tname = Global User\n[core]\n\teditor = vim"
	err = os.WriteFile(globalPath, []byte(globalContent), 0o644)
	require.NoError(t, err)

	// Load configs
	cs := New()
	cs.GlobalConfig = "global-config"
	cs.LocalConfig = ".git/config"
	cs.NoWrites = true
	cs.LoadAll(td)

	// Concurrent reads from different scopes
	var wg sync.WaitGroup
	iterations := 50
	goroutines := 10

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				switch id % 3 {
				case 0:
					// Read value that exists in local scope
					name := cs.GetLocal("user.name")
					assert.Equal(t, "Local User", name)
				case 1:
					// Read value from global scope
					editor := cs.GetGlobal("core.editor")
					assert.Equal(t, "vim", editor)
				case 2:
					// Read with precedence (local wins)
					name := cs.Get("user.name")
					assert.Equal(t, "Local User", name)
				}
			}
		}(g)
	}

	wg.Wait()
}

// TestConcurrentConfigCreation tests creating multiple config instances concurrently.
func TestConcurrentConfigCreation(t *testing.T) {
	t.Parallel()

	var wg sync.WaitGroup
	goroutines := 10

	results := make([]*Configs, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			results[index] = New()
		}(i)
	}

	wg.Wait()

	// Verify all instances were created successfully
	for i := 0; i < goroutines; i++ {
		assert.NotNil(t, results[i], "config instance %d should not be nil", i)
		assert.NotEmpty(t, results[i].LocalConfig)
		assert.NotEmpty(t, results[i].WorktreeConfig)
	}
}

// TestConcurrentEnvConfigLoad tests loading environment configs concurrently.
func TestConcurrentEnvConfigLoad(t *testing.T) {
	t.Parallel()

	// Set up test environment variables
	testPrefix := "GITCONFIG_CONCURRENT"
	os.Setenv(testPrefix+"_COUNT", "2")
	os.Setenv(testPrefix+"_KEY_0", "user.name")
	os.Setenv(testPrefix+"_VALUE_0", "Env User")
	os.Setenv(testPrefix+"_KEY_1", "user.email")
	os.Setenv(testPrefix+"_VALUE_1", "env@example.com")

	defer func() {
		os.Unsetenv(testPrefix + "_COUNT")
		os.Unsetenv(testPrefix + "_KEY_0")
		os.Unsetenv(testPrefix + "_VALUE_0")
		os.Unsetenv(testPrefix + "_KEY_1")
		os.Unsetenv(testPrefix + "_VALUE_1")
	}()

	var wg sync.WaitGroup
	goroutines := 10
	results := make([]*Config, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			results[index] = LoadConfigFromEnv(testPrefix)
		}(i)
	}

	wg.Wait()

	// Verify all loads succeeded
	for i := 0; i < goroutines; i++ {
		require.NotNil(t, results[i], "env config %d should not be nil", i)

		name, ok := results[i].Get("user.name")
		assert.True(t, ok)
		assert.Equal(t, "Env User", name)

		email, ok := results[i].Get("user.email")
		assert.True(t, ok)
		assert.Equal(t, "env@example.com", email)
	}
}

// TestConcurrentReadDuringLoad tests reading while other configs are being loaded.
func TestConcurrentReadDuringLoad(t *testing.T) {
	t.Parallel()

	td := t.TempDir()
	configPath := filepath.Join(td, "config")

	content := "[user]\n\tname = Load Test User"
	err := os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	// Load initial config
	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	var wg sync.WaitGroup
	readGoroutines := 5
	loadGoroutines := 5
	duration := 100 * time.Millisecond

	// Goroutines continuously reading from existing config
	for i := 0; i < readGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			end := time.Now().Add(duration)
			for time.Now().Before(end) {
				name, ok := cfg.Get("user.name")
				assert.True(t, ok)
				assert.Equal(t, "Load Test User", name)
			}
		}()
	}

	// Goroutines loading new config instances
	for i := 0; i < loadGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			end := time.Now().Add(duration)
			for time.Now().Before(end) {
				newCfg, err := LoadConfig(configPath)
				assert.NoError(t, err)
				if newCfg != nil {
					name, ok := newCfg.Get("user.name")
					assert.True(t, ok)
					assert.Equal(t, "Load Test User", name)
				}
			}
		}()
	}

	wg.Wait()
}

// TestNoDataRacesInGet tests that Get operations don't cause data races.
func TestNoDataRacesInGet(t *testing.T) {
	t.Parallel()

	td := t.TempDir()
	configPath := filepath.Join(td, "config")

	content := `[user]
	name = Race Test
[core]
	editor = vim
[remote "origin"]
	url = https://github.com/test/repo.git
`
	err := os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Run with race detector enabled:  go test -race
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_, _ = cfg.Get("user.name")
				_, _ = cfg.Get("core.editor")
				_, _ = cfg.Get("remote.origin.url")
			}
		}()
	}

	wg.Wait()
}

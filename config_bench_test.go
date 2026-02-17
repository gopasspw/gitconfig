package gitconfig

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func BenchmarkLoadConfig(b *testing.B) {
	td := b.TempDir()
	configPath := filepath.Join(td, "config")
	content := "[user]\n\tname = Bench User\n\temail = bench@example.com\n[core]\n\teditor = vim\n"

	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		cfg, err := LoadConfig(configPath)
		if err != nil {
			b.Fatal(err)
		}
		if cfg == nil {
			b.Fatal("nil config")
		}
	}
}

func BenchmarkGet(b *testing.B) {
	td := b.TempDir()
	configPath := filepath.Join(td, "config")
	content := "[user]\n\tname = Bench User\n\temail = bench@example.com\n[core]\n\teditor = vim\n"

	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		b.Fatal(err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		_, ok := cfg.Get("user.name")
		if !ok {
			b.Fatal("missing key")
		}
	}
}

func BenchmarkSet(b *testing.B) {
	td := b.TempDir()
	configPath := filepath.Join(td, "config")
	content := "[user]\n\tname = Bench User\n\temail = bench@example.com\n[core]\n\teditor = vim\n"

	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		b.Fatal(err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		b.Fatal(err)
	}
	cfg.noWrites = true

	b.ResetTimer()

	for i := range b.N {
		if err := cfg.Set("user.name", strconv.Itoa(i)); err != nil {
			b.Fatal(err)
		}
	}
}

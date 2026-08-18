package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureEnvCreatesAndMigrates(t *testing.T) {
	dir := t.TempDir()
	example := "OPENWEBUI_IMAGE=openwebui/open-webui:main\nVALUE=test\n"
	if err := os.WriteFile(filepath.Join(dir, ".env.example"), []byte(example), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := EnsureEnv(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Created || !result.Migrated {
		t.Fatalf("expected created and migrated, got %+v", result)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".env"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "OPENWEBUI_IMAGE=openwebui/open-webui:latest") {
		t.Fatalf("migration missing: %s", text)
	}
}

func TestEnsureEnvPreservesExisting(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte("VALUE=keep\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := EnsureEnv(dir)
	if err != nil {
		t.Fatal(err)
	}
	if result.Created || result.Migrated {
		t.Fatalf("unexpected changes: %+v", result)
	}
	data, _ := os.ReadFile(envPath)
	if string(data) != "VALUE=keep\n" {
		t.Fatalf("existing .env changed: %q", data)
	}
}

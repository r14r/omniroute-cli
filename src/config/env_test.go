package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureEnvCreatesSecureValuesAndPermissions(t *testing.T) {
	dir := t.TempDir()
	template := strings.Join([]string{
		"OPENWEBUI_IMAGE=openwebui/open-webui:latest",
		"OMNIROUTE_JWT_SECRET=__GENERATE__",
		"OMNIROUTE_API_KEY_SECRET=__GENERATE__",
		"OMNIROUTE_INITIAL_PASSWORD=__GENERATE__",
		"OMNIROUTE_WS_BRIDGE_SECRET=__GENERATE__",
		"OMNIROUTE_STORAGE_ENCRYPTION_KEY=__GENERATE__",
		"OPENWEBUI_SECRET_KEY=__GENERATE__",
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(dir, ".env.example"), []byte(template), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := EnsureEnv(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Created || result.Migrated {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(result.InsecureKeys) != 0 {
		t.Fatalf("new env must not contain legacy secrets: %v", result.InsecureKeys)
	}

	envPath := filepath.Join(dir, ".env")
	data, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), generatePlaceholder) {
		t.Fatalf("generated placeholder remained in .env: %s", data)
	}
	info, err := os.Stat(envPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf(".env permissions = %o, want 600", got)
	}
}

func TestEnsureEnvPreservesExistingAndProtectsMode(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte("VALUE=keep\n"), 0o644); err != nil {
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
	info, _ := os.Stat(envPath)
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf(".env permissions = %o, want 600", got)
	}
}

func TestEnsureEnvMigratesObsoleteImageTag(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte("OPENWEBUI_IMAGE=openwebui/open-webui:main\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := EnsureEnv(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Migrated {
		t.Fatal("expected image tag migration")
	}
	data, _ := os.ReadFile(envPath)
	if !strings.Contains(string(data), "openwebui/open-webui:latest") {
		t.Fatalf("migration missing: %s", data)
	}
}

func TestFindHashedSecrets(t *testing.T) {
	values := map[string]string{
		"SECRET": "legacy-test-value",
		"OTHER":  "safe",
	}
	hashes := map[string]string{
		"SECRET": "bf5a9c74ae364a6aa62d07e1ddd8bc9b25e811bcdbee2ca1aa33ea3b79b131e6",
	}
	got := findHashedSecrets(values, hashes)
	if len(got) != 1 || got[0] != "SECRET" {
		t.Fatalf("unexpected insecure keys: %v", got)
	}
}

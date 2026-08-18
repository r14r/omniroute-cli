package compose

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateFiles(t *testing.T) {
	dir := t.TempDir()
	r := Runner{ProjectDir: dir, ComposeFile: "compose.yaml"}
	if err := r.ValidateFiles(); err == nil {
		t.Fatal("expected missing compose file error")
	}
	if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte("services: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := r.ValidateFiles(); err != nil {
		t.Fatalf("unexpected validate error: %v", err)
	}
}

func TestDryRunComposePrintsCommand(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte("services: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	r := Runner{ProjectDir: dir, ComposeFile: "compose.yaml", Stdout: &out, Stderr: &out, DryRun: true}
	if err := r.Compose("up", "-d", "--force-recreate"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "$ docker compose -f compose.yaml up -d --force-recreate") {
		t.Fatalf("unexpected dry-run output: %q", out.String())
	}
}

func TestShellJoinQuotesWhitespace(t *testing.T) {
	got := shellJoin([]string{"compose", "--project-name", "my stack"})
	if !strings.Contains(got, `"my stack"`) {
		t.Fatalf("expected quoted arg, got %q", got)
	}
}

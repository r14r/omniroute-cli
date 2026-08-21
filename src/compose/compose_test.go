package compose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestComposeArgsProjectName(t *testing.T) {
	d := t.TempDir()
	if err := os.WriteFile(filepath.Join(d, "docker-compose.yaml"), []byte("services: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := Runner{ProjectDir: d, ComposeFile: "docker-compose.yaml", ProjectName: "dev"}
	args, err := r.composeArgs("ps")
	if err != nil {
		t.Fatal(err)
	}
	s := strings.Join(args, " ")
	if s != "compose -f docker-compose.yaml -p dev ps" {
		t.Fatal(s)
	}
}

func TestAutoDetectPrefersDockerComposeYAML(t *testing.T) {
	d := t.TempDir()
	for _, name := range []string{"compose.yaml", "docker-compose.yaml"} {
		if err := os.WriteFile(filepath.Join(d, name), []byte("services: {}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r := Runner{ProjectDir: d}
	got, err := r.ResolveComposeFile()
	if err != nil {
		t.Fatal(err)
	}
	if got != "docker-compose.yaml" {
		t.Fatalf("got %q", got)
	}
}

func TestAutoDetectLegacyComposeYAML(t *testing.T) {
	d := t.TempDir()
	if err := os.WriteFile(filepath.Join(d, "compose.yaml"), []byte("services: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// The CLI currently supplies docker-compose.yaml as its standard default.
	// If that file is absent, standard Compose names are compatibility fallbacks.
	r := Runner{ProjectDir: d, ComposeFile: "docker-compose.yaml"}
	got, err := r.ResolveComposeFile()
	if err != nil {
		t.Fatal(err)
	}
	if got != "compose.yaml" {
		t.Fatalf("got %q", got)
	}
}

func TestExplicitComposeFileDoesNotFallback(t *testing.T) {
	d := t.TempDir()
	if err := os.WriteFile(filepath.Join(d, "docker-compose.yaml"), []byte("services: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := Runner{ProjectDir: d, ComposeFile: "missing.yaml"}
	if _, err := r.ResolveComposeFile(); err == nil || !strings.Contains(err.Error(), "missing.yaml") {
		t.Fatalf("expected explicit-file error, got %v", err)
	}
}

func TestDefaultTimeoutCanBeSet(t *testing.T) {
	r := Runner{Timeout: 3 * time.Second}
	if r.Timeout != 3*time.Second {
		t.Fatal()
	}
}

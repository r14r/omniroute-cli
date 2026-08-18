package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHelpAndVersionFlagsExitZero(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"-h"}, {"--version"}, {"-v"}, {"version"}, {"help"}} {
		var out, errOut bytes.Buffer
		if code := Run(args, "0.2.0", &out, &errOut); code != 0 {
			t.Fatalf("Run(%v) code=%d stderr=%q", args, code, errOut.String())
		}
	}
}

func TestUnknownCommandFails(t *testing.T) {
	var out, errOut bytes.Buffer
	code := Run([]string{"unknown"}, "0.2.0", &out, &errOut)
	if code == 0 || !strings.Contains(errOut.String(), "unknown command") {
		t.Fatalf("code=%d stderr=%q", code, errOut.String())
	}
}

func TestReadOnlyStatusDoesNotCreateEnv(t *testing.T) {
	dir := t.TempDir()
	writeProjectFiles(t, dir)
	var out, errOut bytes.Buffer
	code := Run([]string{"--project-dir", dir, "--dry-run", "status"}, "0.2.0", &out, &errOut)
	if code == 0 || !strings.Contains(errOut.String(), ".env does not exist") {
		t.Fatalf("code=%d stderr=%q", code, errOut.String())
	}
	if _, err := os.Stat(filepath.Join(dir, ".env")); !os.IsNotExist(err) {
		t.Fatalf("status created .env unexpectedly: %v", err)
	}
}

func TestInitCreatesEnvAndUpdateAvoidsExplicitDown(t *testing.T) {
	dir := t.TempDir()
	writeProjectFiles(t, dir)

	var out, errOut bytes.Buffer
	if code := Run([]string{"--project-dir", dir, "init"}, "0.2.0", &out, &errOut); code != 0 {
		t.Fatalf("init code=%d stderr=%q", code, errOut.String())
	}
	data, err := os.ReadFile(filepath.Join(dir, ".env"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "__GENERATE__") {
		t.Fatalf("generated placeholder remained: %s", data)
	}

	out.Reset()
	errOut.Reset()
	if code := Run([]string{"--project-dir", dir, "--dry-run", "update"}, "0.2.0", &out, &errOut); code != 0 {
		t.Fatalf("update code=%d stderr=%q", code, errOut.String())
	}
	text := out.String()
	if strings.Contains(text, " compose -f docker-compose.yaml down ") || strings.Contains(text, " compose -f docker-compose.yaml down\n") {
		t.Fatalf("update must not explicitly down the stack:\n%s", text)
	}
	if !strings.Contains(text, "up -d --force-recreate --remove-orphans") {
		t.Fatalf("update missing force recreate:\n%s", text)
	}
}

func TestServiceAliasInDryRun(t *testing.T) {
	dir := t.TempDir()
	writeProjectFiles(t, dir)
	if code := Run([]string{"--project-dir", dir, "init"}, "0.2.0", &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatal("init failed")
	}
	var out, errOut bytes.Buffer
	if code := Run([]string{"--project-dir", dir, "--dry-run", "restart", "openwebui"}, "0.2.0", &out, &errOut); code != 0 {
		t.Fatalf("restart code=%d stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "restart ai-tools-omniroute-openwebui") {
		t.Fatalf("service alias not resolved: %s", out.String())
	}
}

func writeProjectFiles(t *testing.T, dir string) {
	t.Helper()
	compose := "name: test\nservices:\n  ai-tools-omniroute:\n    image: example/omniroute:latest\n  ai-tools-omniroute-openwebui:\n    image: example/openwebui:latest\n"
	if err := os.WriteFile(filepath.Join(dir, "docker-compose.yaml"), []byte(compose), 0o644); err != nil {
		t.Fatal(err)
	}
	template := strings.Join([]string{
		"PUBLIC_BIND_ADDRESS=127.0.0.1",
		"OMNIROUTE_PUBLIC_PORT=20128",
		"OPENWEBUI_PUBLIC_PORT=20000",
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
}

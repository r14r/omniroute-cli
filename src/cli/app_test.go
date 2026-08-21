package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/r14r/omniroute-cli/src/semver"
)

var testVersion = semver.MustParse("0.6.0")

func TestHelpVersionAndGlobalFlagsAnywhere(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"-h"}, {"--version"}, {"-v"}, {"version"}, {"help"}, {"version", "--json"}} {
		var out, err bytes.Buffer
		if c := Run(args, testVersion, &out, &err); c != 0 {
			t.Fatalf("%v code=%d err=%s", args, c, err.String())
		}
	}
}
func TestInitConfigAndNoPullOnUp(t *testing.T) {
	d := fixture(t)
	var out, err bytes.Buffer
	if c := Run([]string{"init", "--project-dir", d}, testVersion, &out, &err); c != 0 {
		t.Fatal(err.String())
	}
	if c := Run([]string{"config", "set", "OMNIROUTE_PUBLIC_PORT", "21111", "--project-dir", d}, testVersion, &out, &err); c != 0 {
		t.Fatal(err.String())
	}
	m, _ := os.ReadFile(filepath.Join(d, ".env"))
	if !strings.Contains(string(m), "OMNIROUTE_PUBLIC_PORT=21111") {
		t.Fatal(string(m))
	}
	out.Reset()
	err.Reset()
	if c := Run([]string{"up", "--dry-run", "--project-dir", d}, testVersion, &out, &err); c != 0 {
		t.Fatal(err.String())
	}
	if strings.Contains(out.String(), " pull ") {
		t.Fatalf("up must not pull: %s", out.String())
	}
}
func TestUpdatePlanIsNonMutatingDryRun(t *testing.T) {
	d := fixture(t)
	Run([]string{"init", "--project-dir", d}, testVersion, &bytes.Buffer{}, &bytes.Buffer{})
	var out, err bytes.Buffer
	if c := Run([]string{"update", "--plan", "--dry-run", "--project-dir", d, "--json"}, testVersion, &out, &err); c != 0 {
		t.Fatal(err.String())
	}
	if !strings.Contains(out.String(), "automatic rollback") {
		t.Fatal(out.String())
	}
}
func TestSecretsCheckPublicWarning(t *testing.T) {
	d := fixture(t)
	Run([]string{"init", "--project-dir", d}, testVersion, &bytes.Buffer{}, &bytes.Buffer{})
	p := filepath.Join(d, ".env")
	b, _ := os.ReadFile(p)
	s := strings.Replace(string(b), "PUBLIC_BIND_ADDRESS=127.0.0.1", "PUBLIC_BIND_ADDRESS=0.0.0.0", 1)
	os.WriteFile(p, []byte(s), 0o600)
	var out, err bytes.Buffer
	if c := Run([]string{"secrets", "check", "--project-dir", d}, testVersion, &out, &err); c != 0 {
		t.Fatal(err.String())
	}
	if !strings.Contains(out.String(), "public-bind-without-api-key") {
		t.Fatal(out.String())
	}
}
func TestGlobalTimeoutParsedSemantically(t *testing.T) {
	o, rest, err := parseGlobals([]string{"status", "--timeout", "3s", "--json"})
	if err != nil {
		t.Fatal(err)
	}
	if o.timeout != 3*time.Second || !o.json || len(rest) != 1 || rest[0] != "status" {
		t.Fatalf("%+v %v", o, rest)
	}
}

func fixture(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	compose := `name: test
services:
  ai-tools-omniroute:
    image: ${OMNIROUTE_IMAGE:-example/omniroute:latest}
  ai-tools-omniroute-openwebui:
    image: ${OPENWEBUI_IMAGE:-example/openwebui:latest}
`
	if err := os.WriteFile(filepath.Join(d, "docker-compose.yaml"), []byte(compose), 0o644); err != nil {
		t.Fatal(err)
	}
	env := `PUBLIC_BIND_ADDRESS=127.0.0.1
OMNIROUTE_PUBLIC_PORT=20128
OPENWEBUI_PUBLIC_PORT=20000
OMNIROUTE_IMAGE=example/omniroute:latest
OPENWEBUI_IMAGE=example/openwebui:latest
OMNIROUTE_JWT_SECRET=__GENERATE__
OMNIROUTE_API_KEY_SECRET=__GENERATE__
OMNIROUTE_INITIAL_PASSWORD=123456
OMNIROUTE_WS_BRIDGE_SECRET=__GENERATE__
OMNIROUTE_STORAGE_ENCRYPTION_KEY=__GENERATE__
OMNIROUTE_REQUIRE_API_KEY=false
OMNIROUTE_AUTH_COOKIE_SECURE=false
OPENWEBUI_SECRET_KEY=__GENERATE__
OMNIROUTE_OPENAI_API_KEY=omniroute-local
OMNIROUTE_MANAGEMENT_TOKEN=
`
	if err := os.WriteFile(filepath.Join(d, ".env.example"), []byte(env), 0o644); err != nil {
		t.Fatal(err)
	}
	return d
}

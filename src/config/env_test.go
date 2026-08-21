package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureEnvSecure(t *testing.T) {
	d := t.TempDir()
	tpl := "OMNIROUTE_JWT_SECRET=__GENERATE__\nOMNIROUTE_INITIAL_PASSWORD=123456\nOPENWEBUI_SECRET_KEY=__GENERATE__\n"
	if err := os.WriteFile(filepath.Join(d, ".env.example"), []byte(tpl), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := EnsureEnv(d)
	if err != nil {
		t.Fatal(err)
	}
	if !r.Created {
		t.Fatal("not created")
	}
	b, _ := os.ReadFile(filepath.Join(d, ".env"))
	s := string(b)
	if strings.Contains(s, "__GENERATE__") || strings.Contains(s, "OMNIROUTE_INITIAL_PASSWORD=123456") {
		t.Fatalf("unsafe env: %s", s)
	}
	st, _ := os.Stat(filepath.Join(d, ".env"))
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("mode %o", st.Mode().Perm())
	}
}
func TestSetEnvValue(t *testing.T) {
	p := filepath.Join(t.TempDir(), ".env")
	os.WriteFile(p, []byte("A=1\n# B=old\n"), 0o600)
	if err := SetEnvValue(p, "A", "2"); err != nil {
		t.Fatal(err)
	}
	if err := SetEnvValue(p, "B", "3"); err != nil {
		t.Fatal(err)
	}
	m, _ := LoadEnv(p)
	if m["A"] != "2" || m["B"] != "3" {
		t.Fatal(m)
	}
}
func TestSecurityWarnings(t *testing.T) {
	w := SecurityWarnings(map[string]string{"PUBLIC_BIND_ADDRESS": "0.0.0.0", "OMNIROUTE_REQUIRE_API_KEY": "false", "OMNIROUTE_AUTH_COOKIE_SECURE": "false"})
	if len(w) != 2 {
		t.Fatal(w)
	}
}

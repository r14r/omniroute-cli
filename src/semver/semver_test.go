package semver

import "testing"

func TestParseCompare(t *testing.T) {
	cases := []string{"0.6.0", "1.2.3-alpha.1+build.5", "10.20.30"}
	for _, s := range cases {
		if _, err := Parse(s); err != nil {
			t.Fatalf("Parse(%q): %v", s, err)
		}
	}
	if MustParse("1.0.0").Compare(MustParse("1.0.0-rc.1")) <= 0 {
		t.Fatal("release must sort after prerelease")
	}
	if MustParse("1.2.0").Compare(MustParse("1.1.9")) <= 0 {
		t.Fatal("semantic compare failed")
	}
}
func TestRejectInvalid(t *testing.T) {
	for _, s := range []string{"1.2", "01.2.3", "1.2.3-01", "1.2.3+"} {
		if _, err := Parse(s); err == nil {
			t.Fatalf("expected %q invalid", s)
		}
	}
}

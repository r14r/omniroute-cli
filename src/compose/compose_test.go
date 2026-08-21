package compose

import (
	"strings"
	"testing"
	"time"
)

func TestComposeArgsProjectName(t *testing.T) {
	r := Runner{ComposeFile: "docker-compose.yaml", ProjectName: "dev"}
	s := strings.Join(r.composeArgs("ps"), " ")
	if s != "compose -f docker-compose.yaml -p dev ps" {
		t.Fatal(s)
	}
}
func TestDefaultTimeoutCanBeSet(t *testing.T) {
	r := Runner{Timeout: 3 * time.Second}
	if r.Timeout != 3*time.Second {
		t.Fatal()
	}
}

package state

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

type Rollback struct {
	CreatedAt           time.Time `json:"created_at"`
	OmniRouteConfigured string    `json:"omniroute_configured"`
	OpenWebUIConfigured string    `json:"openwebui_configured"`
	OmniRoutePrevious   string    `json:"omniroute_previous"`
	OpenWebUIPrevious   string    `json:"openwebui_previous"`
}

func Dir(project string) string          { return filepath.Join(project, ".omniroute-cli") }
func RollbackPath(project string) string { return filepath.Join(Dir(project), "rollback.json") }
func Save(project string, r Rollback) error {
	if err := os.MkdirAll(Dir(project), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(RollbackPath(project), append(b, '\n'), 0o600)
}
func Load(project string) (Rollback, error) {
	var r Rollback
	b, err := os.ReadFile(RollbackPath(project))
	if err != nil {
		return r, err
	}
	if err := json.Unmarshal(b, &r); err != nil {
		return r, err
	}
	if r.OmniRoutePrevious == "" && r.OpenWebUIPrevious == "" {
		return r, errors.New("rollback state has no previous images")
	}
	return r, nil
}

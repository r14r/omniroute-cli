package cli

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/r14r/omniroute-cli/src/config"
	"github.com/r14r/omniroute-cli/src/omni"
)

type healthResult struct {
	Healthy        bool        `json:"healthy"`
	OmniRoute      omni.Check  `json:"omniroute"`
	OpenWebUI      omni.Check  `json:"openwebui"`
	OpenWebUIReady *omni.Check `json:"openwebui_ready,omitempty"`
}

func (a *app) currentHealth(deep bool) healthResult {
	if a.dryRun {
		return healthResult{Healthy: true, OmniRoute: omni.Check{Name: "omniroute", Healthy: true, Detail: "dry-run"}, OpenWebUI: omni.Check{Name: "openwebui", Healthy: true, Detail: "dry-run"}}
	}
	m, err := a.requireEnv()
	if err != nil {
		return healthResult{Healthy: false, OmniRoute: omni.Check{Name: "omniroute", Detail: err.Error()}, OpenWebUI: omni.Check{Name: "openwebui", Detail: err.Error()}}
	}
	c := a.omniClient(m)
	obase := c.BaseURL
	wbase := m["OPENWEBUI_URL"]
	if wbase == "" {
		wbase = omni.LocalURL(m["PUBLIC_BIND_ADDRESS"], envDefault(m, "OPENWEBUI_PUBLIC_PORT", "20000"))
	}
	oh := omni.HTTPCheck("omniroute", strings.TrimRight(obase, "/")+"/api/monitoring/health", m["OMNIROUTE_MANAGEMENT_TOKEN"], a.timeout)
	// If management auth is unavailable, an inference catalog response still proves the service is reachable/ready.
	if !deep && !oh.Healthy && (oh.Status == http.StatusUnauthorized || oh.Status == http.StatusForbidden) {
		fallback := omni.HTTPCheck("omniroute", strings.TrimRight(obase, "/")+"/v1/models?prefix=alias", m["OMNIROUTE_OPENAI_API_KEY"], a.timeout)
		if fallback.Healthy {
			fallback.Detail = "management health requires OMNIROUTE_MANAGEMENT_TOKEN; inference endpoint is healthy"
			oh = fallback
		}
	}
	wh := omni.HTTPCheck("openwebui", strings.TrimRight(wbase, "/")+"/health", "", a.timeout)
	r := healthResult{Healthy: oh.Healthy && wh.Healthy, OmniRoute: oh, OpenWebUI: wh}
	if deep {
		ready := omni.HTTPCheck("openwebui-ready", strings.TrimRight(wbase, "/")+"/ready", "", a.timeout)
		r.OpenWebUIReady = &ready
		r.Healthy = r.Healthy && ready.Healthy
	}
	return r
}
func (a *app) waitHealth(wait time.Duration, deep bool) (healthResult, bool) {
	if wait <= 0 {
		r := a.currentHealth(deep)
		return r, r.Healthy
	}
	deadline := time.Now().Add(wait)
	var r healthResult
	for {
		r = a.currentHealth(deep)
		if r.Healthy {
			return r, true
		}
		if time.Now().After(deadline) {
			return r, false
		}
		time.Sleep(2 * time.Second)
	}
}
func (a *app) healthCommand(args []string) error {
	fs := flag.NewFlagSet("health", flag.ContinueOnError)
	deep := fs.Bool("deep", false, "")
	wait := fs.Duration("wait", 0, "")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) > 0 {
		return errors.New("health accepts --deep and --wait")
	}
	r, ok := a.waitHealth(*wait, *deep)
	if a.jsonOutput {
		return a.write(r, "")
	}
	fmt.Fprintf(a.out, "OmniRoute:  %s (%s)\nOpen WebUI: %s (%s)\n", healthWord(r.OmniRoute.Healthy), r.OmniRoute.Detail, healthWord(r.OpenWebUI.Healthy), r.OpenWebUI.Detail)
	if r.OpenWebUIReady != nil {
		fmt.Fprintf(a.out, "Open WebUI ready: %s (%s)\n", healthWord(r.OpenWebUIReady.Healthy), r.OpenWebUIReady.Detail)
	}
	if !ok {
		return errors.New("one or more health checks failed")
	}
	return nil
}
func healthWord(v bool) string {
	if v {
		return "healthy"
	}
	return "unhealthy"
}

func (a *app) statusCommand(args []string) error {
	if len(args) > 0 {
		return errors.New("status takes no command options")
	}
	if err := a.validateCompose(false); err != nil {
		return err
	}
	omniState, _ := a.runner.ServiceState(serviceOmniRoute)
	webState, _ := a.runner.ServiceState(serviceOpenWebUI)
	h := a.currentHealth(false)
	m, _ := a.requireEnv()
	warnings := config.SecurityWarnings(m)
	if w := config.EnvPermissionWarning(a.envPath()); w != "" {
		warnings = append(warnings, w)
	}
	result := map[string]any{"cli_version": a.version.String(), "project": func() string {
		if a.projectName != "" {
			return a.projectName
		}
		return "ai-tools-omniroute"
	}(), "services": map[string]any{"omniroute": map[string]any{"container": omniState, "health": h.OmniRoute}, "openwebui": map[string]any{"container": webState, "health": h.OpenWebUI}}, "urls": map[string]string{"omniroute": a.omniClient(m).BaseURL, "openwebui": omni.LocalURL(m["PUBLIC_BIND_ADDRESS"], envDefault(m, "OPENWEBUI_PUBLIC_PORT", "20000"))}, "security_warnings": warnings}
	if a.jsonOutput {
		return a.write(result, "")
	}
	fmt.Fprintf(a.out, "omniroute-cli %s\nProject: %s\nOmniRoute: %s\nOpen WebUI: %s\n", a.version.VString(), result["project"], healthWord(h.OmniRoute.Healthy), healthWord(h.OpenWebUI.Healthy))
	if ws := warnings; len(ws) > 0 {
		fmt.Fprintf(a.out, "Security warnings: %s\n", strings.Join(ws, ", "))
	}
	return nil
}

func (a *app) doctorCommand(args []string) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	deep := fs.Bool("deep", false, "")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) > 0 {
		return errors.New("doctor accepts only --deep")
	}
	result := map[string]any{"version": a.version.String(), "checks": map[string]any{}}
	checks := result["checks"].(map[string]any)
	if err := a.runner.ValidateFiles(); err != nil {
		checks["compose_file"] = err.Error()
		return err
	}
	checks["compose_file"] = "ok"
	if err := a.runner.CheckDocker(); err != nil {
		checks["docker"] = err.Error()
		return err
	}
	checks["docker"] = "ok"
	if err := a.runner.CheckCompose(); err != nil {
		checks["compose"] = err.Error()
		return err
	}
	checks["compose"] = "ok"
	m, err := a.requireEnv()
	if err != nil {
		return err
	}
	warnings := config.SecurityWarnings(m)
	if w := config.EnvPermissionWarning(a.envPath()); w != "" {
		warnings = append(warnings, w)
	}
	checks["security_warnings"] = warnings
	if err := a.runner.Compose("config", "--quiet"); err != nil {
		return err
	}
	checks["config"] = "ok"
	if *deep {
		h := a.currentHealth(true)
		checks["health"] = h
		if !h.Healthy {
			_ = a.writePretty(result)
			return errors.New("deep health check failed")
		}
	}
	if len(warnings) > 0 {
		_ = a.writePretty(result)
		return errors.New("security policy warnings detected")
	}
	if a.jsonOutput {
		return a.write(result, "")
	}
	fmt.Fprintln(a.out, "Doctor: OK")
	if *deep {
		fmt.Fprintln(a.out, "Deep application readiness: OK")
	}
	return nil
}

func (a *app) urls() error {
	m, _ := config.LoadEnv(a.envPath())
	obase := m["OMNIROUTE_URL"]
	if obase == "" {
		obase = omni.LocalURL(m["PUBLIC_BIND_ADDRESS"], envDefault(m, "OMNIROUTE_PUBLIC_PORT", "20128"))
	}
	wbase := m["OPENWEBUI_URL"]
	if wbase == "" {
		wbase = omni.LocalURL(m["PUBLIC_BIND_ADDRESS"], envDefault(m, "OPENWEBUI_PUBLIC_PORT", "20000"))
	}
	return a.write(map[string]string{"omniroute": obase, "openwebui": wbase}, fmt.Sprintf("OmniRoute:  %s\nOpen WebUI: %s\n", obase, wbase))
}
func (a *app) info() error {
	u := map[string]any{"version": a.version.String(), "project_dir": a.dir, "compose_file": a.file, "project_name": a.projectName, "env_file": a.envPath(), "timeout": a.timeout.String()}
	return a.writePretty(u)
}
func (a *app) composeConfig() error {
	if err := a.validateCompose(false); err != nil {
		return err
	}
	return a.runner.Compose("config")
}

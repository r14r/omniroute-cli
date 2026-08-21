package cli

import (
	"errors"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/r14r/omniroute-cli/src/config"
	"github.com/r14r/omniroute-cli/src/state"
)

func (a *app) composeServices(action string, args []string) error {
	if err := a.validateCompose(false); err != nil {
		return err
	}
	resolved, err := resolveServices(args)
	if err != nil {
		return err
	}
	x := []string{action}
	x = append(x, resolved...)
	return a.runner.Compose(x...)
}
func (a *app) up() error {
	if err := a.validateCompose(true); err != nil {
		return err
	}
	return a.runner.Compose("up", "-d", "--remove-orphans")
}
func (a *app) down() error {
	if err := a.validateCompose(false); err != nil {
		return err
	}
	return a.runner.Compose("down", "--remove-orphans")
}
func (a *app) pull() error {
	if err := a.validateCompose(true); err != nil {
		return err
	}
	return a.pullServices()
}
func (a *app) pullServices() error {
	if err := a.runner.Compose("pull", serviceOmniRoute); err != nil {
		return err
	}
	return a.runner.Compose("pull", serviceOpenWebUI)
}
func (a *app) recreate() error {
	if err := a.validateCompose(true); err != nil {
		return err
	}
	return a.runner.Compose("up", "-d", "--force-recreate", "--remove-orphans")
}
func (a *app) composeRead(args ...string) error {
	if err := a.validateCompose(false); err != nil {
		return err
	}
	return a.runner.Compose(args...)
}
func (a *app) clean() error { return a.down() }
func (a *app) prune() error {
	if !a.dryRun {
		if err := a.runner.CheckDocker(); err != nil {
			return err
		}
	}
	return a.runner.Run("image", "prune", "-f")
}
func (a *app) cleanAll(args []string) error {
	fs := flag.NewFlagSet("clean-all", flag.ContinueOnError)
	yes := fs.Bool("yes", false, "")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !*yes {
		return errors.New("clean-all deletes persistent volumes; rerun with --yes")
	}
	if err := a.validateCompose(false); err != nil {
		return err
	}
	return a.runner.Compose("down", "-v", "--remove-orphans")
}

func (a *app) updateCommand(args []string) error {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(a.errOut)
	plan := fs.Bool("plan", false, "preview update")
	wait := fs.Duration("wait", 2*time.Minute, "health wait after update")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) > 0 {
		return errors.New("update accepts only --plan and --wait")
	}
	if err := a.validateCompose(true); err != nil {
		return err
	}
	m, err := a.requireEnv()
	if err != nil {
		return err
	}
	// A persisted rollback pins immutable digests in .env. A later update restores
	// the configured moving refs recorded before the failed update, then retries.
	if rb, e := state.Load(a.dir); e == nil {
		changed := false
		if rb.OmniRouteConfigured != "" && m["OMNIROUTE_IMAGE"] == rb.OmniRoutePrevious {
			_ = config.SetEnvValue(a.envPath(), "OMNIROUTE_IMAGE", rb.OmniRouteConfigured)
			changed = true
		}
		if rb.OpenWebUIConfigured != "" && m["OPENWEBUI_IMAGE"] == rb.OpenWebUIPrevious {
			_ = config.SetEnvValue(a.envPath(), "OPENWEBUI_IMAGE", rb.OpenWebUIConfigured)
			changed = true
		}
		if changed {
			m, err = a.requireEnv()
			if err != nil {
				return err
			}
		}
	}
	omniRef := envDefault(m, "OMNIROUTE_IMAGE", "diegosouzapw/omniroute:latest")
	webRef := envDefault(m, "OPENWEBUI_IMAGE", "openwebui/open-webui:latest")
	oldOmni, _ := a.runner.ImageDigest(omniRef)
	oldWeb, _ := a.runner.ImageDigest(webRef)
	preview := map[string]any{"configured_images": map[string]string{"omniroute": omniRef, "openwebui": webRef}, "current_digests": map[string]string{"omniroute": oldOmni, "openwebui": oldWeb}, "steps": []string{"pull images", "force recreate", "wait for health", "automatic rollback on failed health"}}
	if *plan {
		return a.writePretty(preview)
	}
	if oldOmni != "" || oldWeb != "" {
		_ = state.Save(a.dir, state.Rollback{CreatedAt: time.Now(), OmniRouteConfigured: omniRef, OpenWebUIConfigured: webRef, OmniRoutePrevious: oldOmni, OpenWebUIPrevious: oldWeb})
	}
	if err := a.pullServices(); err != nil {
		return err
	}
	if err := a.runner.Compose("up", "-d", "--force-recreate", "--remove-orphans"); err != nil {
		return err
	}
	checks, healthy := a.waitHealth(*wait, false)
	if healthy {
		return a.write(map[string]any{"updated": true, "health": checks}, "Update complete; services healthy.\n")
	}
	if oldOmni == "" && oldWeb == "" {
		return fmt.Errorf("update health check failed and no previous image digest is available")
	}
	fmt.Fprintln(a.errOut, "Update health check failed; attempting automatic image rollback")
	if err := a.rollbackImages(oldOmni, oldWeb, true); err != nil {
		return fmt.Errorf("update failed health check and rollback failed: %w", err)
	}
	return errors.New("update failed health check; previous images were restored")
}

func (a *app) rollbackCommand(args []string) error {
	fs := flag.NewFlagSet("rollback", flag.ContinueOnError)
	prev := fs.Bool("previous", false, "")
	if err := fs.Parse(args); err != nil {
		return err
	}
	_ = prev
	if len(fs.Args()) > 0 {
		return errors.New("rollback accepts only --previous")
	}
	if err := a.validateCompose(false); err != nil {
		return err
	}
	r, err := state.Load(a.dir)
	if err != nil {
		return fmt.Errorf("load rollback state: %w", err)
	}
	if err := a.rollbackImages(r.OmniRoutePrevious, r.OpenWebUIPrevious, true); err != nil {
		return err
	}
	return a.write(map[string]any{"rolled_back": true, "images": map[string]string{"omniroute": r.OmniRoutePrevious, "openwebui": r.OpenWebUIPrevious}}, "Rollback complete; services healthy.\n")
}
func (a *app) rollbackImages(omniDigest, webDigest string, persist bool) error {
	m, err := a.requireEnv()
	if err != nil {
		return err
	}
	if persist {
		if omniDigest != "" {
			if err := config.SetEnvValue(a.envPath(), "OMNIROUTE_IMAGE", omniDigest); err != nil {
				return err
			}
		}
		if webDigest != "" {
			if err := config.SetEnvValue(a.envPath(), "OPENWEBUI_IMAGE", webDigest); err != nil {
				return err
			}
		}
		m, err = a.requireEnv()
		if err != nil {
			return err
		}
	}
	env := map[string]string{}
	for k, v := range a.runner.Env {
		env[k] = v
	}
	if omniDigest != "" {
		env["OMNIROUTE_IMAGE"] = omniDigest
	}
	if webDigest != "" {
		env["OPENWEBUI_IMAGE"] = webDigest
	}
	r := a.runner
	r.Env = env
	if err := r.Compose("up", "-d", "--force-recreate", "--remove-orphans"); err != nil {
		return err
	}
	_ = m
	checks, ok := a.waitHealth(90*time.Second, false)
	if !ok {
		return fmt.Errorf("rollback completed but health is not OK: %v", checks)
	}
	return nil
}

func (a *app) logs(followDefault bool, args []string) error {
	if err := a.validateCompose(false); err != nil {
		return err
	}
	fs := flag.NewFlagSet("logs", flag.ContinueOnError)
	follow := fs.Bool("f", followDefault, "")
	fs.BoolVar(follow, "follow", followDefault, "")
	tail := fs.Int("tail", 200, "")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *tail < 0 {
		return errors.New("--tail must be >=0")
	}
	x := []string{"logs", "--tail", fmt.Sprint(*tail)}
	if *follow {
		x = append(x, "-f")
	}
	rem := fs.Args()
	if len(rem) > 1 {
		return errors.New("logs accepts at most one service")
	}
	if len(rem) == 1 {
		s, err := resolveService(rem[0])
		if err != nil {
			return err
		}
		x = append(x, s)
	}
	return a.runner.Compose(x...)
}
func (a *app) shell(args []string) error {
	if err := a.validateCompose(false); err != nil {
		return err
	}
	if len(args) > 1 {
		return errors.New("shell accepts at most one service")
	}
	s := serviceOmniRoute
	if len(args) == 1 {
		var err error
		s, err = resolveService(args[0])
		if err != nil {
			return err
		}
	}
	return a.runner.Compose("exec", s, "sh")
}
func resolveServices(xs []string) ([]string, error) {
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		s, err := resolveService(x)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}
func resolveService(v string) (string, error) {
	switch strings.ToLower(v) {
	case "omniroute", serviceOmniRoute:
		return serviceOmniRoute, nil
	case "openwebui", "open-webui", serviceOpenWebUI:
		return serviceOpenWebUI, nil
	default:
		return "", fmt.Errorf("unknown service %q", v)
	}
}

package cli

import (
	"errors"
	"fmt"

	"github.com/r14r/omniroute-cli/src/omni"
)

func (a *app) apiResponse(r omni.Response, err error) error {
	if err != nil {
		if r.Status == 401 || r.Status == 403 {
			return fmt.Errorf("management API authorization failed; set OMNIROUTE_MANAGEMENT_TOKEN to an oma_live_… access token with sufficient scope: %w", err)
		}
		return err
	}
	if r.Body != nil {
		return a.writePretty(r.Body)
	}
	return a.write(map[string]any{"raw": r.Raw}, r.Raw+"\n")
}
func (a *app) modelsCommand(args []string) error {
	if len(args) > 1 || (len(args) == 1 && args[0] != "list") {
		return errors.New("usage: models list")
	}
	m, err := a.requireEnv()
	if err != nil {
		return err
	}
	r, err := a.omniClient(m).Models()
	if err != nil {
		return err
	}
	ids, err := omni.ModelIDs(r)
	if err != nil {
		return a.apiResponse(r, nil)
	}
	if a.jsonOutput {
		return a.write(map[string]any{"models": ids}, "")
	}
	for _, id := range ids {
		fmt.Fprintln(a.out, id)
	}
	return nil
}
func (a *app) providersCommand(args []string) error {
	if len(args) > 1 || (len(args) == 1 && args[0] != "status") {
		return errors.New("usage: providers status")
	}
	m, err := a.requireEnv()
	if err != nil {
		return err
	}
	r, e := a.omniClient(m).Providers()
	return a.apiResponse(r, e)
}
func (a *app) sessionsCommand(args []string) error {
	if len(args) > 0 {
		return errors.New("usage: sessions")
	}
	m, err := a.requireEnv()
	if err != nil {
		return err
	}
	r, e := a.omniClient(m).Sessions()
	return a.apiResponse(r, e)
}
func (a *app) usageCommand(args []string) error {
	if len(args) > 0 {
		return errors.New("usage: usage")
	}
	m, err := a.requireEnv()
	if err != nil {
		return err
	}
	r, e := a.omniClient(m).Usage()
	return a.apiResponse(r, e)
}
func (a *app) cacheCommand(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: cache stats | cache clear --yes")
	}
	m, err := a.requireEnv()
	if err != nil {
		return err
	}
	c := a.omniClient(m)
	switch args[0] {
	case "stats":
		if len(args) != 1 {
			return errors.New("usage: cache stats")
		}
		r, e := c.CacheStats()
		return a.apiResponse(r, e)
	case "clear":
		if len(args) != 2 || args[1] != "--yes" {
			return errors.New("cache clear deletes semantic cache entries; rerun as 'cache clear --yes'")
		}
		r, e := c.ClearCache()
		return a.apiResponse(r, e)
	default:
		return fmt.Errorf("unknown cache subcommand %q", args[0])
	}
}

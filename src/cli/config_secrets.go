package cli

import (
	"errors"
	"flag"
	"fmt"
	"sort"
	"strings"

	"github.com/r14r/omniroute-cli/src/config"
)

func (a *app) configCommand(args []string) error {
	sub := "list"
	if len(args) > 0 {
		sub = args[0]
		args = args[1:]
	}
	switch sub {
	case "list":
		m, err := a.requireEnv()
		if err != nil {
			return err
		}
		red := map[string]string{}
		for k, v := range m {
			if config.SecretKeys[k] && v != "" {
				red[k] = "<redacted>"
			} else {
				red[k] = v
			}
		}
		if a.jsonOutput {
			return a.write(red, "")
		}
		ks := sortedKeys(red)
		for _, k := range ks {
			fmt.Fprintf(a.out, "%s=%s\n", k, red[k])
		}
		return nil
	case "get":
		if len(args) != 1 {
			return errors.New("usage: config get KEY")
		}
		m, err := a.requireEnv()
		if err != nil {
			return err
		}
		v, ok := m[args[0]]
		if !ok {
			return fmt.Errorf("config key %s not found", args[0])
		}
		if config.SecretKeys[args[0]] {
			v = "<redacted>"
		}
		return a.write(map[string]any{"key": args[0], "value": v}, fmt.Sprintf("%s\n", v))
	case "set":
		if len(args) != 2 {
			return errors.New("usage: config set KEY VALUE")
		}
		if config.SecretKeys[args[0]] {
			return fmt.Errorf("%s is secret; use 'secrets rotate' or edit .env deliberately", args[0])
		}
		if _, err := a.requireEnv(); err != nil {
			return err
		}
		if err := config.SetEnvValue(a.envPath(), args[0], args[1]); err != nil {
			return err
		}
		return a.write(map[string]any{"key": args[0], "updated": true}, fmt.Sprintf("Updated %s\n", args[0]))
	case "validate":
		if err := a.validateCompose(false); err != nil {
			return err
		}
		return a.write(map[string]any{"valid": true}, "Configuration: OK\n")
	case "path":
		return a.write(map[string]any{"path": a.envPath()}, a.envPath()+"\n")
	default:
		return fmt.Errorf("unknown config subcommand %q", sub)
	}
}

func (a *app) secretsCommand(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: secrets check|rotate")
	}
	switch args[0] {
	case "check":
		m, err := a.requireEnv()
		if err != nil {
			return err
		}
		warnings := config.SecurityWarnings(m)
		if w := config.EnvPermissionWarning(a.envPath()); w != "" {
			warnings = append(warnings, w)
			sort.Strings(warnings)
		}
		result := map[string]any{"ok": len(warnings) == 0, "warnings": warnings}
		if len(warnings) > 0 {
			return a.write(result, "Security warnings: "+strings.Join(warnings, ", ")+"\n")
		}
		return a.write(result, "Secrets/security: OK\n")
	case "rotate":
		return a.rotateSecrets(args[1:])
	default:
		return fmt.Errorf("unknown secrets subcommand %q", args[0])
	}
}
func (a *app) rotateSecrets(args []string) error {
	fs := flag.NewFlagSet("secrets rotate", flag.ContinueOnError)
	fs.SetOutput(a.errOut)
	safe := fs.Bool("safe", false, "rotate all non-storage generated secrets")
	yes := fs.Bool("yes", false, "confirm disruptive rotation")
	if err := fs.Parse(args); err != nil {
		return err
	}
	names := fs.Args()
	aliases := map[string]string{"jwt": "OMNIROUTE_JWT_SECRET", "api": "OMNIROUTE_API_KEY_SECRET", "initial-password": "OMNIROUTE_INITIAL_PASSWORD", "ws": "OMNIROUTE_WS_BRIDGE_SECRET", "storage": "OMNIROUTE_STORAGE_ENCRYPTION_KEY", "openwebui": "OPENWEBUI_SECRET_KEY"}
	var keys []string
	if *safe {
		if !*yes {
			return errors.New("secrets rotate --safe changes sessions/tokens; rerun with --yes")
		}
		keys = []string{"OMNIROUTE_JWT_SECRET", "OMNIROUTE_API_KEY_SECRET", "OMNIROUTE_WS_BRIDGE_SECRET", "OPENWEBUI_SECRET_KEY"}
	} else {
		if len(names) != 1 {
			return errors.New("usage: secrets rotate <jwt|api|initial-password|ws|storage|openwebui> [--yes] or --safe --yes")
		}
		k, ok := aliases[names[0]]
		if !ok {
			return fmt.Errorf("unknown secret %q", names[0])
		}
		if k == "OMNIROUTE_STORAGE_ENCRYPTION_KEY" && !*yes {
			return errors.New("storage key rotation can make encrypted data unreadable; rerun with --yes only after migration planning")
		}
		keys = []string{k}
	}
	if _, err := a.requireEnv(); err != nil {
		return err
	}
	sort.Strings(keys)
	for _, k := range keys {
		v, err := config.GenerateSecret(k)
		if err != nil {
			return err
		}
		if err := config.SetEnvValue(a.envPath(), k, v); err != nil {
			return err
		}
	}
	return a.write(map[string]any{"rotated": keys, "restart_required": true}, "Rotated: "+strings.Join(keys, ", ")+"\nRestart/recreate the stack for changes to take effect.\n")
}

func (a *app) completion(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: completion bash|zsh|fish")
	}
	commands := "init up pull update rollback recreate start stop restart down status health doctor logs log shell top images services resolved-images urls info config secrets models providers sessions usage cache clean prune clean-all version help"
	switch args[0] {
	case "bash":
		fmt.Fprintf(a.out, "complete -W '%s' omniroute-cli\n", commands)
	case "zsh":
		fmt.Fprintf(a.out, "#compdef omniroute-cli\n_arguments '*:command:(%s)'\n", commands)
	case "fish":
		for _, c := range strings.Fields(commands) {
			fmt.Fprintf(a.out, "complete -c omniroute-cli -f -a %s\n", c)
		}
	default:
		return fmt.Errorf("unsupported shell %q", args[0])
	}
	return nil
}

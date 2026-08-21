package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/r14r/omniroute-cli/src/compose"
	"github.com/r14r/omniroute-cli/src/config"
	"github.com/r14r/omniroute-cli/src/omni"
	"github.com/r14r/omniroute-cli/src/semver"
)

const (
	serviceOmniRoute = "ai-tools-omniroute"
	serviceOpenWebUI = "ai-tools-omniroute-openwebui"
)

type app struct {
	version                semver.Version
	out, errOut            io.Writer
	dir, file, projectName string
	dryRun, jsonOutput     bool
	timeout                time.Duration
	runner                 compose.Runner
}

type globalOptions struct {
	dir, file, project string
	dry, json          bool
	timeout            time.Duration
}

func Run(args []string, version semver.Version, out, errOut io.Writer) int {
	a := &app{version: version, out: out, errOut: errOut}
	if err := a.run(args); err != nil {
		if a.jsonOutput {
			_ = json.NewEncoder(errOut).Encode(map[string]any{"ok": false, "error": err.Error()})
		} else {
			fmt.Fprintf(errOut, "ERROR: %v\n", err)
		}
		return 1
	}
	return 0
}

func (a *app) run(args []string) error {
	opts, rest, err := parseGlobals(args)
	if err != nil {
		return err
	}
	a.dir, a.file, a.projectName, a.dryRun, a.jsonOutput, a.timeout = opts.dir, opts.file, opts.project, opts.dry, opts.json, opts.timeout
	if a.timeout <= 0 {
		a.timeout = 2 * time.Minute
	}
	abs, err := filepath.Abs(a.dir)
	if err != nil {
		return err
	}
	a.dir = abs
	env := map[string]string{}
	if a.projectName != "" {
		env["COMPOSE_PROJECT_NAME"] = a.projectName
		env["OMNIROUTE_PREFIX"] = a.projectName
	}
	runnerOut := a.out
	if a.jsonOutput {
		runnerOut = a.errOut
	}
	a.runner = compose.Runner{ProjectDir: a.dir, ComposeFile: a.file, ProjectName: a.projectName, Stdout: runnerOut, Stderr: a.errOut, DryRun: a.dryRun, Timeout: a.timeout, Env: env}
	if len(rest) == 0 {
		return a.printUsage(a.out)
	}
	cmd := rest[0]
	sub := rest[1:]
	switch cmd {
	case "help", "--help", "-h":
		return a.printUsage(a.out)
	case "version":
		return a.printVersion()
	case "completion":
		return a.completion(sub)
	case "init":
		return a.initEnv()
	case "info":
		return a.info()
	case "urls":
		return a.urls()
	case "config":
		return a.configCommand(sub)
	case "secrets":
		return a.secretsCommand(sub)
	case "health":
		return a.healthCommand(sub)
	case "doctor":
		return a.doctorCommand(sub)
	case "status", "ps":
		return a.statusCommand(sub)
	case "models":
		return a.modelsCommand(sub)
	case "providers":
		return a.providersCommand(sub)
	case "sessions":
		return a.sessionsCommand(sub)
	case "usage":
		return a.usageCommand(sub)
	case "cache":
		return a.cacheCommand(sub)
	case "compose-config":
		return a.composeConfig()
	case "start", "stop", "restart":
		return a.composeServices(cmd, sub)
	case "up", "run":
		return a.up()
	case "down":
		return a.down()
	case "pull":
		return a.pull()
	case "update", "rebuild":
		return a.updateCommand(sub)
	case "rollback":
		return a.rollbackCommand(sub)
	case "recreate":
		return a.recreate()
	case "images":
		return a.composeRead("images")
	case "services":
		return a.composeRead("config", "--services")
	case "resolved-images":
		return a.composeRead("config", "--images")
	case "top":
		return a.composeRead("top")
	case "logs", "log":
		return a.logs(cmd == "logs", sub)
	case "shell":
		return a.shell(sub)
	case "clean":
		return a.clean()
	case "prune":
		return a.prune()
	case "clean-all":
		return a.cleanAll(sub)
	default:
		return fmt.Errorf("unknown command %q; run 'omniroute-cli help'", cmd)
	}
}

func parseGlobals(args []string) (globalOptions, []string, error) {
	o := globalOptions{dir: ".", file: "docker-compose.yaml", timeout: 2 * time.Minute}
	var rest []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--dry-run":
			o.dry = true
		case a == "--json":
			o.json = true
		case a == "--project-dir" || a == "--compose-file" || a == "--project-name" || a == "--prefix" || a == "--timeout":
			if i+1 >= len(args) {
				return o, nil, fmt.Errorf("%s requires a value", a)
			}
			i++
			v := args[i]
			switch a {
			case "--project-dir":
				o.dir = v
			case "--compose-file":
				o.file = v
			case "--project-name", "--prefix":
				o.project = v
			case "--timeout":
				d, e := time.ParseDuration(v)
				if e != nil || d <= 0 {
					return o, nil, fmt.Errorf("invalid --timeout %q", v)
				}
				o.timeout = d
			}
		case strings.HasPrefix(a, "--project-dir="):
			o.dir = strings.TrimPrefix(a, "--project-dir=")
		case strings.HasPrefix(a, "--compose-file="):
			o.file = strings.TrimPrefix(a, "--compose-file=")
		case strings.HasPrefix(a, "--project-name="):
			o.project = strings.TrimPrefix(a, "--project-name=")
		case strings.HasPrefix(a, "--prefix="):
			o.project = strings.TrimPrefix(a, "--prefix=")
		case strings.HasPrefix(a, "--timeout="):
			d, e := time.ParseDuration(strings.TrimPrefix(a, "--timeout="))
			if e != nil || d <= 0 {
				return o, nil, fmt.Errorf("invalid --timeout")
			}
			o.timeout = d
		case a == "--version" || a == "-v":
			rest = append(rest, "version")
		default:
			rest = append(rest, a)
		}
	}
	if strings.ContainsAny(o.project, "/\\ \t\n") {
		return o, nil, errors.New("--project-name/--prefix must not contain whitespace or path separators")
	}
	return o, rest, nil
}

func (a *app) printUsage(w io.Writer) error {
	fmt.Fprintf(w, `omniroute-cli %s - OmniRoute/Open WebUI operations CLI

Usage:
  omniroute-cli [global options] <command> [options]

Global options (accepted before or after the command):
  --project-dir DIR       Project directory (default .)
  --compose-file FILE     Compose file (default docker-compose.yaml)
  --project-name NAME     Compose project/container/volume prefix
  --prefix NAME           Alias for --project-name
  --timeout DURATION      Per-operation timeout (default 2m)
  --json                  Machine-readable JSON output
  --dry-run               Print Docker operations without executing
  --version, -v           Print semantic CLI version

Lifecycle:
  init                    Securely initialize .env
  up | run                Start configured stack without pulling images
  pull                    Pull configured images
  update [--plan]         Pull, recreate, health-check; auto-rollback on failure
  rollback [--previous]   Recreate with last known previous image digests
  recreate                Recreate without pulling
  start|stop|restart [SVC...]
  down                    Remove containers/network, keep volumes
  clean                   Alias-like safe down
  clean-all --yes         Delete stack including volumes
  prune                   Explicit global Docker image prune

Operations:
  status                  Rich container + HTTP health status
  health [--deep] [--wait DURATION]
  doctor [--deep]         Docker/config/security/application diagnostics
  logs|log [--tail N] [SVC]
  shell [SVC]             Open sh in a service
  top                     Show container processes
  images|resolved-images|services
  urls                    Show local URLs
  info                    Show CLI/project metadata
  compose-config          Render Docker Compose configuration

Configuration and API:
  config list|get|set|validate|path
  secrets check|rotate
  models list
  providers status
  sessions
  usage
  cache stats|clear
  completion bash|zsh|fish

Service aliases: omniroute, openwebui
`, a.version.VString())
	return nil
}
func (a *app) printVersion() error {
	return a.write(map[string]any{"version": a.version.String()}, fmt.Sprintf("omniroute-cli %s\n", a.version.VString()))
}
func (a *app) write(v any, human string) error {
	if a.jsonOutput {
		return json.NewEncoder(a.out).Encode(v)
	}
	_, err := io.WriteString(a.out, human)
	return err
}
func (a *app) writePretty(v any) error {
	if a.jsonOutput {
		return json.NewEncoder(a.out).Encode(v)
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(a.out, string(b))
	return err
}
func (a *app) envPath() string { return filepath.Join(a.dir, ".env") }
func (a *app) requireEnv() (map[string]string, error) {
	m, err := config.LoadEnv(a.envPath())
	if errors.Is(err, os.ErrNotExist) {
		return nil, errors.New(".env does not exist; run 'omniroute-cli init' first")
	}
	return m, err
}
func (a *app) ensureDocker() error {
	if err := a.runner.ValidateFiles(); err != nil {
		return err
	}
	if a.dryRun {
		return nil
	}
	return a.runner.CheckDocker()
}
func (a *app) validateCompose(create bool) error {
	if create {
		if _, err := config.EnsureEnv(a.dir); err != nil {
			return err
		}
	} else {
		if _, err := a.requireEnv(); err != nil {
			return err
		}
	}
	if err := a.ensureDocker(); err != nil {
		return err
	}
	return a.runner.Compose("config", "--quiet")
}
func envDefault(m map[string]string, k, d string) string {
	if strings.TrimSpace(m[k]) != "" {
		return m[k]
	}
	return d
}
func sortedKeys(m map[string]string) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}
func parseBool(s string) bool { v, _ := strconv.ParseBool(s); return v }

func (a *app) initEnv() error {
	r, err := config.EnsureEnv(a.dir)
	if err != nil {
		return err
	}
	msg := ".env already exists\n"
	if r.Created {
		msg = "Created secure .env from .env.example\n"
	}
	if r.Migrated {
		msg += "Migrated Open WebUI image tag main -> latest\n"
	}
	if len(r.InsecureKeys) > 0 {
		msg += "WARNING legacy/insecure values: " + strings.Join(r.InsecureKeys, ", ") + "\n"
	}
	return a.write(r, msg)
}

func (a *app) omniClient(m map[string]string) omni.Client {
	base := m["OMNIROUTE_URL"]
	if base == "" {
		base = omni.LocalURL(m["PUBLIC_BIND_ADDRESS"], envDefault(m, "OMNIROUTE_PUBLIC_PORT", "20128"))
	}
	return omni.Client{BaseURL: base, ManagementToken: m["OMNIROUTE_MANAGEMENT_TOKEN"], InferenceToken: m["OMNIROUTE_OPENAI_API_KEY"], Timeout: a.timeout}
}

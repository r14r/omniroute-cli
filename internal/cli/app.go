package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/r14r/omniroute-cli/internal/compose"
	"github.com/r14r/omniroute-cli/internal/config"
)

const (
	serviceOmniRoute = "ai-tools-omniroute"
	serviceOpenWebUI = "ai-tools-omniroute-openwebui"
)

type app struct {
	version string
	out     io.Writer
	errOut  io.Writer
	dir     string
	file    string
	dryRun  bool
	runner  compose.Runner
}

func Run(args []string, version string, out, errOut io.Writer) int {
	a := &app{version: version, out: out, errOut: errOut}
	if err := a.run(args); err != nil {
		fmt.Fprintf(errOut, "ERROR: %v\n", err)
		return 1
	}
	return 0
}

func (a *app) run(args []string) error {
	fs := flag.NewFlagSet("omniroute-cli", flag.ContinueOnError)
	fs.SetOutput(a.errOut)
	fs.StringVar(&a.dir, "project-dir", ".", "project directory containing compose.yaml")
	fs.StringVar(&a.file, "compose-file", "compose.yaml", "Docker Compose file relative to project directory")
	fs.BoolVar(&a.dryRun, "dry-run", false, "print Docker commands without executing them")
	fs.Usage = func() { a.printUsage(a.errOut) }

	if err := fs.Parse(args); err != nil {
		return err
	}

	rest := fs.Args()
	if len(rest) == 0 {
		a.printUsage(a.out)
		return nil
	}

	absDir, err := filepath.Abs(a.dir)
	if err != nil {
		return fmt.Errorf("resolve project directory: %w", err)
	}
	a.dir = absDir
	a.runner = compose.Runner{
		ProjectDir:  a.dir,
		ComposeFile: a.file,
		Stdout:      a.out,
		Stderr:      a.errOut,
		DryRun:      a.dryRun,
	}

	command := rest[0]
	commandArgs := rest[1:]
	if command != "help" && command != "version" && command != "init" && command != "urls" {
		if err := a.runner.ValidateFiles(); err != nil {
			return err
		}
		if !a.dryRun {
			if err := a.runner.CheckDocker(); err != nil {
				return err
			}
		}
	}

	switch command {
	case "help", "--help", "-h":
		a.printUsage(a.out)
		return nil
	case "version":
		fmt.Fprintf(a.out, "omniroute-cli %s\n", normalizeVersion(a.version))
		return nil
	case "init":
		return a.initEnv()
	case "config":
		if err := a.prepare(); err != nil {
			return err
		}
		return a.runner.Compose("config")
	case "doctor":
		return a.doctor()
	case "start":
		return a.composeWithOptionalServices("start", commandArgs)
	case "stop":
		return a.composeWithOptionalServices("stop", commandArgs)
	case "restart":
		return a.composeWithOptionalServices("restart", commandArgs)
	case "up", "run":
		return a.up()
	case "down":
		return a.down()
	case "pull":
		return a.pull()
	case "update", "rebuild":
		return a.update()
	case "recreate":
		return a.recreate()
	case "status", "ps":
		if err := a.prepare(); err != nil {
			return err
		}
		return a.runner.Compose("ps")
	case "images":
		if err := a.prepare(); err != nil {
			return err
		}
		return a.runner.Compose("images")
	case "services":
		if err := a.prepare(); err != nil {
			return err
		}
		return a.runner.Compose("config", "--services")
	case "resolved-images":
		if err := a.prepare(); err != nil {
			return err
		}
		return a.runner.Compose("config", "--images")
	case "logs", "log":
		return a.logs(command == "logs", commandArgs)
	case "shell":
		return a.shell(commandArgs)
	case "urls":
		return a.urls()
	case "clean":
		return a.clean()
	case "clean-all":
		return a.cleanAll(commandArgs)
	default:
		return fmt.Errorf("unknown command %q; run 'omniroute-cli help'", command)
	}
}

func (a *app) printUsage(w io.Writer) {
	fmt.Fprintf(w, `omniroute-cli %s - manage the ai-tools-omniroute Docker Compose stack

Usage:
  omniroute-cli [global options] <command> [command options]

Global options:
  --project-dir DIR      Project directory (default: .)
  --compose-file FILE    Compose file (default: compose.yaml)
  --dry-run              Print commands without executing Docker

Commands:
  init                   Create .env from .env.example when missing
  up | run               Pull images and start the stack
  start [SERVICE...]     Start existing containers
  stop [SERVICE...]      Stop containers
  restart [SERVICE...]   Restart containers
  down                   Remove containers/network; keep volumes
  pull                   Pull both service images
  update                 Pull, down and force-recreate the stack
  rebuild                Alias for update
  recreate               Force-recreate without pulling images
  status | ps            Show container status
  logs [options] [SVC]   Follow logs (default --tail 200)
  log [options] [SVC]    Show logs without following
  images                 Show local images used by the stack
  services               Show resolved Compose services
  resolved-images        Show exact images resolved from .env
  shell [SERVICE]        Open sh in a service (default: omniroute)
  urls                   Show public OmniRoute/Open WebUI URLs
  config                 Render resolved Docker Compose configuration
  doctor                 Validate Docker, Compose, config, services and images
  clean                  Down stack and prune unused images
  clean-all --yes        Delete stack INCLUDING persistent volumes
  version                Print CLI version
  help                   Show this help

Service aliases:
  omniroute              %s
  openwebui              %s

Examples:
  omniroute-cli up
  omniroute-cli status
  omniroute-cli logs -f openwebui
  omniroute-cli update
  omniroute-cli --project-dir /opt/omniroute status
`, normalizeVersion(a.version), serviceOmniRoute, serviceOpenWebUI)
}

func (a *app) prepare() error {
	if err := a.initEnv(); err != nil {
		return err
	}
	return nil
}

func (a *app) initEnv() error {
	result, err := config.EnsureEnv(a.dir)
	if err != nil {
		return err
	}
	switch {
	case result.Created && result.Migrated:
		fmt.Fprintf(a.out, "Created .env from %s and migrated Open WebUI tag main -> latest\n", filepath.Base(result.Source))
	case result.Created:
		fmt.Fprintf(a.out, "Created .env from %s\n", filepath.Base(result.Source))
	case result.Migrated:
		fmt.Fprintln(a.out, "Migrated .env Open WebUI tag main -> latest")
	default:
		fmt.Fprintln(a.out, ".env already exists")
	}
	return nil
}

func (a *app) validateConfig() error {
	if err := a.prepare(); err != nil {
		return err
	}
	return a.runner.Compose("config", "--quiet")
}

func (a *app) composeWithOptionalServices(action string, args []string) error {
	if err := a.validateConfig(); err != nil {
		return err
	}
	resolved, err := resolveServices(args)
	if err != nil {
		return err
	}
	composeArgs := []string{action}
	composeArgs = append(composeArgs, resolved...)
	return a.runner.Compose(composeArgs...)
}

func (a *app) up() error {
	if err := a.validateConfig(); err != nil {
		return err
	}
	if err := a.pullServices(); err != nil {
		return err
	}
	return a.runner.Compose("up", "-d", "--remove-orphans")
}

func (a *app) down() error {
	if err := a.validateConfig(); err != nil {
		return err
	}
	return a.runner.Compose("down", "--remove-orphans")
}

func (a *app) pull() error {
	if err := a.validateConfig(); err != nil {
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

func (a *app) update() error {
	if err := a.validateConfig(); err != nil {
		return err
	}
	if err := a.pullServices(); err != nil {
		return err
	}
	if err := a.runner.Compose("down", "--remove-orphans"); err != nil {
		return err
	}
	if err := a.runner.Compose("up", "-d", "--force-recreate", "--remove-orphans"); err != nil {
		return err
	}
	return a.runner.Compose("ps")
}

func (a *app) recreate() error {
	if err := a.validateConfig(); err != nil {
		return err
	}
	return a.runner.Compose("up", "-d", "--force-recreate", "--remove-orphans")
}

func (a *app) logs(followDefault bool, args []string) error {
	if err := a.validateConfig(); err != nil {
		return err
	}
	fs := flag.NewFlagSet("logs", flag.ContinueOnError)
	fs.SetOutput(a.errOut)
	follow := followDefault
	tail := 200
	fs.BoolVar(&follow, "f", followDefault, "follow log output")
	fs.BoolVar(&follow, "follow", followDefault, "follow log output")
	fs.IntVar(&tail, "tail", 200, "number of lines to show")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if tail < 0 {
		return errors.New("--tail must be >= 0")
	}

	composeArgs := []string{"logs", "--tail", strconv.Itoa(tail)}
	if follow {
		composeArgs = append(composeArgs, "-f")
	}
	remaining := fs.Args()
	if len(remaining) > 1 {
		return errors.New("logs accepts at most one service")
	}
	if len(remaining) == 1 {
		service, err := resolveService(remaining[0])
		if err != nil {
			return err
		}
		composeArgs = append(composeArgs, service)
	}
	return a.runner.Compose(composeArgs...)
}

func (a *app) shell(args []string) error {
	if err := a.validateConfig(); err != nil {
		return err
	}
	if len(args) > 1 {
		return errors.New("shell accepts at most one service")
	}
	service := serviceOmniRoute
	if len(args) == 1 {
		var err error
		service, err = resolveService(args[0])
		if err != nil {
			return err
		}
	}
	return a.runner.Compose("exec", service, "sh")
}

func (a *app) urls() error {
	if err := a.prepare(); err != nil {
		return err
	}
	values, err := config.LoadEnv(filepath.Join(a.dir, ".env"))
	if err != nil {
		return fmt.Errorf("read .env: %w", err)
	}
	omniPort := values["OMNIROUTE_PUBLIC_PORT"]
	if omniPort == "" {
		omniPort = "20128"
	}
	webUIPort := values["OPENWEBUI_PUBLIC_PORT"]
	if webUIPort == "" {
		webUIPort = "20000"
	}
	fmt.Fprintf(a.out, "OmniRoute:  http://localhost:%s\n", omniPort)
	fmt.Fprintf(a.out, "Open WebUI: http://localhost:%s\n", webUIPort)
	return nil
}

func (a *app) doctor() error {
	if err := a.runner.ValidateFiles(); err != nil {
		return err
	}
	if err := a.runner.CheckDocker(); err != nil {
		return err
	}
	if err := a.prepare(); err != nil {
		return err
	}
	fmt.Fprintln(a.out, "\nDocker Compose:")
	if err := a.runner.CheckCompose(); err != nil {
		return err
	}
	fmt.Fprintln(a.out, "\nConfiguration:")
	if err := a.runner.Compose("config", "--quiet"); err != nil {
		return err
	}
	fmt.Fprintln(a.out, "OK")
	fmt.Fprintln(a.out, "\nServices:")
	if err := a.runner.Compose("config", "--services"); err != nil {
		return err
	}
	fmt.Fprintln(a.out, "\nImages:")
	return a.runner.Compose("config", "--images")
}

func (a *app) clean() error {
	if err := a.validateConfig(); err != nil {
		return err
	}
	if err := a.runner.Compose("down", "--remove-orphans"); err != nil {
		return err
	}
	return a.runner.Run("image", "prune", "-f")
}

func (a *app) cleanAll(args []string) error {
	fs := flag.NewFlagSet("clean-all", flag.ContinueOnError)
	fs.SetOutput(a.errOut)
	yes := fs.Bool("yes", false, "confirm deletion of persistent volumes")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !*yes {
		return errors.New("clean-all deletes persistent volumes; rerun with --yes")
	}
	if err := a.validateConfig(); err != nil {
		return err
	}
	return a.runner.Compose("down", "-v", "--remove-orphans")
}

func resolveServices(values []string) ([]string, error) {
	resolved := make([]string, 0, len(values))
	for _, value := range values {
		service, err := resolveService(value)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, service)
	}
	return resolved, nil
}

func resolveService(value string) (string, error) {
	switch strings.ToLower(value) {
	case "omniroute", serviceOmniRoute:
		return serviceOmniRoute, nil
	case "openwebui", "open-webui", serviceOpenWebUI:
		return serviceOpenWebUI, nil
	default:
		return "", fmt.Errorf("unknown service %q (use omniroute or openwebui)", value)
	}
}

func normalizeVersion(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return "dev"
	}
	if version == "dev" || strings.HasPrefix(version, "v") {
		return version
	}
	return "v" + version
}

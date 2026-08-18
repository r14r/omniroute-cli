package compose

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Runner struct {
	ProjectDir  string
	ComposeFile string
	Stdout      io.Writer
	Stderr      io.Writer
	DryRun      bool
}

func (r Runner) ValidateFiles() error {
	path := r.ComposeFile
	if !filepath.IsAbs(path) {
		path = filepath.Join(r.ProjectDir, path)
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("compose file %s: %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("compose file %s is a directory", path)
	}
	return nil
}

func (r Runner) CheckDocker() error {
	if _, err := exec.LookPath("docker"); err != nil {
		return errors.New("docker command not found")
	}
	return nil
}

func (r Runner) CheckCompose() error {
	return r.Run("compose", "version")
}

func (r Runner) Compose(args ...string) error {
	full := []string{"compose", "-f", r.ComposeFile}
	full = append(full, args...)
	return r.Run(full...)
}

func (r Runner) ComposeOutput(args ...string) (string, error) {
	full := []string{"compose", "-f", r.ComposeFile}
	full = append(full, args...)
	return r.Output(full...)
}

func (r Runner) Run(args ...string) error {
	fmt.Fprintf(r.Stdout, "$ docker %s\n", shellJoin(args))
	if r.DryRun {
		return nil
	}
	cmd := exec.Command("docker", args...)
	cmd.Dir = r.ProjectDir
	cmd.Stdout = r.Stdout
	cmd.Stderr = r.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker %s: %w", strings.Join(args, " "), err)
	}
	return nil
}

func (r Runner) Output(args ...string) (string, error) {
	fmt.Fprintf(r.Stdout, "$ docker %s\n", shellJoin(args))
	if r.DryRun {
		return "", nil
	}
	var stdout bytes.Buffer
	cmd := exec.Command("docker", args...)
	cmd.Dir = r.ProjectDir
	cmd.Stdout = &stdout
	cmd.Stderr = r.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("docker %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func shellJoin(args []string) string {
	quoted := make([]string, len(args))
	for i, arg := range args {
		if strings.ContainsAny(arg, " \t\n\"'") {
			quoted[i] = fmt.Sprintf("%q", arg)
		} else {
			quoted[i] = arg
		}
	}
	return strings.Join(quoted, " ")
}

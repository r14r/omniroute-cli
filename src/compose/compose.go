package compose

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Runner struct {
	ProjectDir, ComposeFile, ProjectName string
	Stdout, Stderr                       io.Writer
	DryRun                               bool
	Timeout                              time.Duration
	Env                                  map[string]string
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
func (r Runner) CheckCompose() error          { return r.Run("compose", "version") }
func (r Runner) Compose(args ...string) error { return r.Run(r.composeArgs(args...)...) }
func (r Runner) ComposeOutput(args ...string) (string, error) {
	return r.Output(r.composeArgs(args...)...)
}
func (r Runner) composeArgs(args ...string) []string {
	full := []string{"compose", "-f", r.ComposeFile}
	if r.ProjectName != "" {
		full = append(full, "-p", r.ProjectName)
	}
	return append(full, args...)
}
func (r Runner) Run(args ...string) error              { _, err := r.run(false, args...); return err }
func (r Runner) Output(args ...string) (string, error) { return r.run(true, args...) }
func (r Runner) run(capture bool, args ...string) (string, error) {
	fmt.Fprintf(r.Stdout, "$ docker %s\n", shellJoin(args))
	if r.DryRun {
		return "", nil
	}
	timeout := r.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = r.ProjectDir
	cmd.Stdin = os.Stdin
	cmd.Stderr = r.Stderr
	cmd.Env = mergeEnv(os.Environ(), r.Env)
	var buf bytes.Buffer
	if capture {
		cmd.Stdout = &buf
	} else {
		cmd.Stdout = r.Stdout
	}
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("docker %s timed out after %s", strings.Join(args, " "), timeout)
	}
	if err != nil {
		return "", fmt.Errorf("docker %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(buf.String()), nil
}
func mergeEnv(base []string, extra map[string]string) []string {
	if len(extra) == 0 {
		return base
	}
	m := map[string]string{}
	for _, e := range base {
		k, v, ok := strings.Cut(e, "=")
		if ok {
			m[k] = v
		}
	}
	for k, v := range extra {
		m[k] = v
	}
	out := make([]string, 0, len(m))
	for k, v := range m {
		out = append(out, k+"="+v)
	}
	return out
}
func shellJoin(args []string) string {
	q := make([]string, len(args))
	for i, a := range args {
		if strings.ContainsAny(a, " \t\n\"'") {
			q[i] = fmt.Sprintf("%q", a)
		} else {
			q[i] = a
		}
	}
	return strings.Join(q, " ")
}

func (r Runner) ServiceState(service string) (map[string]any, error) {
	out, err := r.ComposeOutput("ps", "--format", "json", service)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(out) == "" {
		return map[string]any{"service": service, "running": false}, nil
	}
	var arr []map[string]any
	if json.Unmarshal([]byte(out), &arr) == nil && len(arr) > 0 {
		return arr[0], nil
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(out), &obj); err == nil {
		return obj, nil
	}
	// Some compose versions emit one JSON object per line.
	line := strings.Split(out, "\n")[0]
	if err := json.Unmarshal([]byte(line), &obj); err != nil {
		return nil, fmt.Errorf("parse compose ps JSON: %w", err)
	}
	return obj, nil
}
func (r Runner) ImageDigest(ref string) (string, error) {
	if strings.TrimSpace(ref) == "" {
		return "", errors.New("empty image reference")
	}
	return r.Output("image", "inspect", "--format", "{{if .RepoDigests}}{{index .RepoDigests 0}}{{else}}{{.Id}}{{end}}", ref)
}

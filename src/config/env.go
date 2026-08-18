package config

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type EnsureResult struct {
	Created  bool
	Migrated bool
	Source   string
	Path     string
}

func EnsureEnv(projectDir string) (EnsureResult, error) {
	envPath := filepath.Join(projectDir, ".env")
	result := EnsureResult{Path: envPath}

	if _, err := os.Stat(envPath); errors.Is(err, os.ErrNotExist) {
		source, err := findExample(projectDir)
		if err != nil {
			return result, err
		}
		if err := copyFile(source, envPath); err != nil {
			return result, fmt.Errorf("create .env: %w", err)
		}
		result.Created = true
		result.Source = source
	} else if err != nil {
		return result, fmt.Errorf("stat .env: %w", err)
	}

	data, err := os.ReadFile(envPath)
	if err != nil {
		return result, fmt.Errorf("read .env: %w", err)
	}

	old := "OPENWEBUI_IMAGE=openwebui/open-webui:main"
	current := "OPENWEBUI_IMAGE=openwebui/open-webui:latest"
	text := string(data)
	if strings.Contains(text, old) {
		text = strings.ReplaceAll(text, old, current)
		if err := os.WriteFile(envPath, []byte(text), 0o600); err != nil {
			return result, fmt.Errorf("migrate .env: %w", err)
		}
		result.Migrated = true
	}

	return result, nil
}

func LoadEnv(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	values := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key != "" {
			values[key] = value
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return values, nil
}

func findExample(projectDir string) (string, error) {
	candidates := []string{
		filepath.Join(projectDir, ".env.example"),
		filepath.Join(projectDir, "env.example"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", errors.New("neither .env.example nor env.example exists")
}

func copyFile(source, target string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

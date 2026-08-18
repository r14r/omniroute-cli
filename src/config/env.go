package config

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	generatePlaceholder     = "__GENERATE__"
	upstreamInitialPassword = "123456"
)

var generatedSecretBytes = map[string]int{
	"OMNIROUTE_JWT_SECRET":             48,
	"OMNIROUTE_API_KEY_SECRET":         32,
	"OMNIROUTE_WS_BRIDGE_SECRET":       32,
	"OMNIROUTE_STORAGE_ENCRYPTION_KEY": 32,
	"OPENWEBUI_SECRET_KEY":             32,
}

// SHA-256 fingerprints of values published in pre-0.2.0 example files.
// Storing only fingerprints avoids re-publishing the legacy secret material.
var legacyPublishedSecretHashes = map[string]string{
	"OMNIROUTE_JWT_SECRET":             "e525dc897265a4a8891b40b3c89557549dd7572a3f2cb030201c41f8547cb7ca",
	"OMNIROUTE_API_KEY_SECRET":         "1b2e487b8479e0794661c80a01a205c8ed44a9f8802278fa4225870e339a8657",
	"OMNIROUTE_INITIAL_PASSWORD":       "3b5cfc10aa1366456a2191437fc5410df0be7e5d76a5ce45f86c906c9a7b6926",
	"OMNIROUTE_WS_BRIDGE_SECRET":       "288aeb8f2aa9e69688ae9f422a7e9e76f8b13da63e76726d0ba95edf2d283927",
	"OMNIROUTE_STORAGE_ENCRYPTION_KEY": "00e56d10ed9ba532cfed068e740438980d42f3a60c99a248c4f08d14251218c1",
	"OPENWEBUI_SECRET_KEY":             "61220b8a94b4abe7abbddb44a196d1ee546feb0a523a0bd5bac2b9db11e9b38f",
}

type EnsureResult struct {
	Created      bool
	Migrated     bool
	Source       string
	Path         string
	InsecureKeys []string
}

func EnsureEnv(projectDir string) (EnsureResult, error) {
	envPath := filepath.Join(projectDir, ".env")
	result := EnsureResult{Path: envPath}

	if _, err := os.Stat(envPath); errors.Is(err, os.ErrNotExist) {
		source := filepath.Join(projectDir, ".env.example")
		data, err := os.ReadFile(source)
		if err != nil {
			return result, fmt.Errorf("read .env.example: %w", err)
		}
		rendered, err := renderSecureTemplate(string(data))
		if err != nil {
			return result, err
		}
		if err := writeExclusive(envPath, []byte(rendered)); err != nil {
			return result, fmt.Errorf("create .env: %w", err)
		}
		if err := os.Chmod(envPath, 0o600); err != nil {
			return result, fmt.Errorf("protect .env: %w", err)
		}
		result.Created = true
		result.Source = source
	} else if err != nil {
		return result, fmt.Errorf("stat .env: %w", err)
	} else if err := os.Chmod(envPath, 0o600); err != nil {
		return result, fmt.Errorf("protect .env: %w", err)
	}

	data, err := os.ReadFile(envPath)
	if err != nil {
		return result, fmt.Errorf("read .env: %w", err)
	}

	oldImage := "OPENWEBUI_IMAGE=openwebui/open-webui:main"
	currentImage := "OPENWEBUI_IMAGE=openwebui/open-webui:latest"
	text := string(data)
	if strings.Contains(text, oldImage) {
		text = strings.ReplaceAll(text, oldImage, currentImage)
		if err := os.WriteFile(envPath, []byte(text), 0o600); err != nil {
			return result, fmt.Errorf("migrate .env: %w", err)
		}
		result.Migrated = true
	}

	values, err := LoadEnv(envPath)
	if err != nil {
		return result, fmt.Errorf("inspect .env: %w", err)
	}
	result.InsecureKeys = FindLegacyPublishedSecrets(values)
	return result, nil
}

func FindLegacyPublishedSecrets(values map[string]string) []string {
	keys := findHashedSecrets(values, legacyPublishedSecretHashes)
	if values["OMNIROUTE_INITIAL_PASSWORD"] == upstreamInitialPassword {
		found := false
		for _, key := range keys {
			if key == "OMNIROUTE_INITIAL_PASSWORD" {
				found = true
				break
			}
		}
		if !found {
			keys = append(keys, "OMNIROUTE_INITIAL_PASSWORD")
			sort.Strings(keys)
		}
	}
	return keys
}

func findHashedSecrets(values map[string]string, hashes map[string]string) []string {
	var keys []string
	for key, expectedHash := range hashes {
		value, ok := values[key]
		if !ok || value == "" {
			continue
		}
		sum := sha256.Sum256([]byte(value))
		if hex.EncodeToString(sum[:]) == expectedHash {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
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

func renderSecureTemplate(template string) (string, error) {
	lines := strings.Split(template, "\n")
	for i, line := range lines {
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		shouldGenerate := value == generatePlaceholder || (key == "OMNIROUTE_INITIAL_PASSWORD" && value == upstreamInitialPassword)
		if !shouldGenerate {
			continue
		}
		var generated string
		var err error
		if key == "OMNIROUTE_INITIAL_PASSWORD" {
			generated, err = randomPassword(24)
		} else if size, found := generatedSecretBytes[key]; found {
			generated, err = randomHex(size)
		} else {
			return "", fmt.Errorf("unknown generated secret key %q", key)
		}
		if err != nil {
			return "", fmt.Errorf("generate %s: %w", key, err)
		}
		lines[i] = key + "=" + generated
	}
	rendered := strings.Join(lines, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		_, value, ok := strings.Cut(line, "=")
		if ok && strings.TrimSpace(value) == generatePlaceholder {
			return "", errors.New("unresolved generated secret placeholder remains in .env.example")
		}
	}
	return rendered, nil
}

func writeExclusive(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return err
	}
	return f.Close()
}

func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func randomPassword(length int) (string, error) {
	if length < 16 {
		return "", errors.New("password length must be at least 16")
	}
	// URL-safe alphabet avoids shell/.env quoting problems.
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(buf)
	if len(encoded) > length {
		encoded = encoded[:length]
	}
	return encoded, nil
}

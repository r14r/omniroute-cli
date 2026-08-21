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
	GeneratePlaceholder     = "__GENERATE__"
	UpstreamInitialPassword = "123456"
)

var generatedSecretBytes = map[string]int{
	"OMNIROUTE_JWT_SECRET":             48,
	"OMNIROUTE_API_KEY_SECRET":         32,
	"OMNIROUTE_WS_BRIDGE_SECRET":       32,
	"OMNIROUTE_STORAGE_ENCRYPTION_KEY": 32,
	"OPENWEBUI_SECRET_KEY":             32,
}

var legacyPublishedSecretHashes = map[string]string{
	"OMNIROUTE_JWT_SECRET":             "e525dc897265a4a8891b40b3c89557549dd7572a3f2cb030201c41f8547cb7ca",
	"OMNIROUTE_API_KEY_SECRET":         "1b2e487b8479e0794661c80a01a205c8ed44a9f8802278fa4225870e339a8657",
	"OMNIROUTE_INITIAL_PASSWORD":       "3b5cfc10aa1366456a2191437fc5410df0be7e5d76a5ce45f86c906c9a7b6926",
	"OMNIROUTE_WS_BRIDGE_SECRET":       "288aeb8f2aa9e69688ae9f422a7e9e76f8b13da63e76726d0ba95edf2d283927",
	"OMNIROUTE_STORAGE_ENCRYPTION_KEY": "00e56d10ed9ba532cfed068e740438980d42f3a60c99a248c4f08d14251218c1",
	"OPENWEBUI_SECRET_KEY":             "61220b8a94b4abe7abbddb44a196d1ee546feb0a523a0bd5bac2b9db11e9b38f",
}

var SecretKeys = map[string]bool{
	"OMNIROUTE_JWT_SECRET":             true,
	"OMNIROUTE_API_KEY_SECRET":         true,
	"OMNIROUTE_INITIAL_PASSWORD":       true,
	"OMNIROUTE_WS_BRIDGE_SECRET":       true,
	"OMNIROUTE_STORAGE_ENCRYPTION_KEY": true,
	"OPENWEBUI_SECRET_KEY":             true,
	"OMNIROUTE_OPENAI_API_KEY":         true,
	"OMNIROUTE_MANAGEMENT_TOKEN":       true,
}

type EnsureResult struct {
	Created, Migrated bool
	Source, Path      string
	InsecureKeys      []string
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
		rendered, err := RenderSecureTemplate(string(data))
		if err != nil {
			return result, err
		}
		if err := writeExclusive(envPath, []byte(rendered)); err != nil {
			return result, fmt.Errorf("create .env: %w", err)
		}
		result.Created, result.Source = true, source
	} else if err != nil {
		return result, fmt.Errorf("stat .env: %w", err)
	}
	if err := os.Chmod(envPath, 0o600); err != nil {
		return result, fmt.Errorf("protect .env: %w", err)
	}
	data, err := os.ReadFile(envPath)
	if err != nil {
		return result, err
	}
	oldImage := "OPENWEBUI_IMAGE=openwebui/open-webui:main"
	if strings.Contains(string(data), oldImage) {
		text := strings.ReplaceAll(string(data), oldImage, "OPENWEBUI_IMAGE=openwebui/open-webui:latest")
		if err := os.WriteFile(envPath, []byte(text), 0o600); err != nil {
			return result, err
		}
		result.Migrated = true
	}
	values, err := LoadEnv(envPath)
	if err != nil {
		return result, err
	}
	result.InsecureKeys = FindLegacyPublishedSecrets(values)
	return result, nil
}

func LoadEnv(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	values := map[string]string{}
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.Trim(strings.TrimSpace(v), `"'`)
		if k != "" {
			values[k] = v
		}
	}
	return values, s.Err()
}

func SetEnvValue(path, key, value string) error {
	if strings.TrimSpace(key) == "" || strings.ContainsAny(key, "=\n\r") {
		return errors.New("invalid environment key")
	}
	if strings.ContainsAny(value, "\n\r") {
		return errors.New("environment value must be one line")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	found := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		k, _, ok := strings.Cut(line, "=")
		if ok && strings.TrimSpace(k) == key {
			lines[i] = key + "=" + value
			found = true
		}
	}
	if !found {
		if len(lines) > 0 && lines[len(lines)-1] != "" {
			lines = append(lines, "")
		}
		lines = append(lines, key+"="+value)
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

func FindLegacyPublishedSecrets(values map[string]string) []string {
	keys := findHashedSecrets(values, legacyPublishedSecretHashes)
	if values["OMNIROUTE_INITIAL_PASSWORD"] == UpstreamInitialPassword {
		keys = appendUnique(keys, "OMNIROUTE_INITIAL_PASSWORD")
	}
	sort.Strings(keys)
	return keys
}

func EnvPermissionWarning(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return "env-permissions-unreadable"
	}
	if info.Mode().Perm()&0o077 != 0 {
		return fmt.Sprintf("env-permissions-%03o", info.Mode().Perm())
	}
	return ""
}

func SecurityWarnings(values map[string]string) []string {
	var warnings []string
	warnings = append(warnings, FindLegacyPublishedSecrets(values)...)
	if values["PUBLIC_BIND_ADDRESS"] == "0.0.0.0" && strings.EqualFold(values["OMNIROUTE_REQUIRE_API_KEY"], "false") {
		warnings = append(warnings, "public-bind-without-api-key")
	}
	if values["PUBLIC_BIND_ADDRESS"] == "0.0.0.0" && strings.EqualFold(values["OMNIROUTE_AUTH_COOKIE_SECURE"], "false") {
		warnings = append(warnings, "public-bind-with-insecure-cookie")
	}
	sort.Strings(warnings)
	return warnings
}

func findHashedSecrets(values, hashes map[string]string) []string {
	var keys []string
	for k, h := range hashes {
		v := values[k]
		if v == "" {
			continue
		}
		sum := sha256.Sum256([]byte(v))
		if hex.EncodeToString(sum[:]) == h {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys
}
func appendUnique(xs []string, s string) []string {
	for _, x := range xs {
		if x == s {
			return xs
		}
	}
	return append(xs, s)
}

func RenderSecureTemplate(template string) (string, error) {
	lines := strings.Split(template, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		should := value == GeneratePlaceholder || (key == "OMNIROUTE_INITIAL_PASSWORD" && value == UpstreamInitialPassword)
		if !should {
			continue
		}
		generated, err := GenerateSecret(key)
		if err != nil {
			return "", err
		}
		lines[i] = key + "=" + generated
	}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		_, v, ok := strings.Cut(line, "=")
		if ok && strings.TrimSpace(v) == GeneratePlaceholder {
			return "", errors.New("unresolved generated secret placeholder remains")
		}
	}
	return strings.Join(lines, "\n"), nil
}

func GenerateSecret(key string) (string, error) {
	if key == "OMNIROUTE_INITIAL_PASSWORD" {
		return randomPassword(24)
	}
	n, ok := generatedSecretBytes[key]
	if !ok {
		return "", fmt.Errorf("key %s is not a generated secret", key)
	}
	return randomHex(n)
}

func writeExclusive(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err = f.Write(data); err != nil {
		f.Close()
		os.Remove(path)
		return err
	}
	return f.Close()
}
func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
func randomPassword(length int) (string, error) {
	if length < 16 {
		return "", errors.New("password length must be at least 16")
	}
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	s := base64.RawURLEncoding.EncodeToString(b)
	if len(s) > length {
		s = s[:length]
	}
	return s, nil
}

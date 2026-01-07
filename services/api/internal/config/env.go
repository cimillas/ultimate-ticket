package config

import (
	"bufio"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// LoadEnvFile loads .env values from the current directory or parents if found.
func LoadEnvFile(logger *log.Logger) {
	path, err := findEnvFile()
	if err != nil {
		logger.Printf("WARN: failed to locate .env: %v", err)
		return
	}
	if path == "" {
		logger.Printf("WARN: .env not found in current or parent directories")
		return
	}

	file, err := os.Open(path)
	if err != nil {
		logger.Printf("WARN: failed to open %s: %v", path, err)
		return
	}
	if err := parseEnvFile(logger, file); err != nil {
		logger.Printf("WARN: failed to load %s: %v", path, err)
	} else {
		logger.Printf("loaded env from %s", path)
	}
	_ = file.Close()
}

func findEnvFile() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for i := 0; i < 6; i++ {
		path := filepath.Join(dir, ".env")
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", nil
}

func parseEnvFile(logger *log.Logger, file *os.File) error {
	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if lineNum == 1 {
			line = strings.TrimPrefix(line, "\ufeff")
		}
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			continue
		}
		value = trimQuotes(value)
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			logger.Printf("WARN: failed to set %s from env file", key)
		}
	}
	return scanner.Err()
}

func trimQuotes(value string) string {
	if len(value) < 2 {
		return value
	}
	if (value[0] == '"' && value[len(value)-1] == '"') ||
		(value[0] == '\'' && value[len(value)-1] == '\'') {
		return value[1 : len(value)-1]
	}
	return value
}

// ResolveSessionCookieSecure determines the secure flag for session cookies.
// Defaults to false in local and true otherwise, unless explicitly set.
func ResolveSessionCookieSecure(logger *log.Logger) bool {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}
	defaultSecure := true
	appEnv := strings.TrimSpace(os.Getenv("APP_ENV"))
	switch appEnv {
	case "":
		logger.Printf("WARN: APP_ENV not set, assuming non-local for SESSION_COOKIE_SECURE default")
	case "local":
		defaultSecure = false
	}

	raw := strings.TrimSpace(os.Getenv("SESSION_COOKIE_SECURE"))
	if raw == "" {
		logger.Printf("WARN: SESSION_COOKIE_SECURE not set, using default %t", defaultSecure)
		return defaultSecure
	}
	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		logger.Printf("WARN: SESSION_COOKIE_SECURE invalid, using default %t", defaultSecure)
		return defaultSecure
	}
	return parsed
}

// ResolveAllowPublicRegister determines if public registration is enabled.
// Defaults to true in local and false otherwise, unless explicitly set.
func ResolveAllowPublicRegister(logger *log.Logger) bool {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}
	defaultAllowed := false
	appEnv := strings.TrimSpace(os.Getenv("APP_ENV"))
	switch appEnv {
	case "":
		logger.Printf("WARN: APP_ENV not set, assuming non-local for ALLOW_PUBLIC_REGISTER default")
	case "local":
		defaultAllowed = true
	}

	raw := strings.TrimSpace(os.Getenv("ALLOW_PUBLIC_REGISTER"))
	if raw == "" {
		logger.Printf("WARN: ALLOW_PUBLIC_REGISTER not set, using default %t", defaultAllowed)
		return defaultAllowed
	}
	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		logger.Printf("WARN: ALLOW_PUBLIC_REGISTER invalid, using default %t", defaultAllowed)
		return defaultAllowed
	}
	return parsed
}

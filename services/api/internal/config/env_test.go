package config

import (
	"bytes"
	"log"
	"testing"
)

func TestResolveSessionCookieSecure_LocalDefaultsToFalse(t *testing.T) {
	t.Setenv("APP_ENV", "local")
	t.Setenv("SESSION_COOKIE_SECURE", "")

	logger := log.New(&bytes.Buffer{}, "", 0)
	if got := ResolveSessionCookieSecure(logger); got {
		t.Fatalf("expected false for local default, got true")
	}
}

func TestResolveSessionCookieSecure_NonLocalDefaultsToTrue(t *testing.T) {
	t.Setenv("APP_ENV", "staging")
	t.Setenv("SESSION_COOKIE_SECURE", "")

	logger := log.New(&bytes.Buffer{}, "", 0)
	if got := ResolveSessionCookieSecure(logger); !got {
		t.Fatalf("expected true for non-local default, got false")
	}
}

func TestResolveSessionCookieSecure_EmptyEnvDefaultsToTrue(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("SESSION_COOKIE_SECURE", "")

	logger := log.New(&bytes.Buffer{}, "", 0)
	if got := ResolveSessionCookieSecure(logger); !got {
		t.Fatalf("expected true when APP_ENV missing, got false")
	}
}

func TestResolveSessionCookieSecure_RespectsExplicitValue(t *testing.T) {
	t.Setenv("APP_ENV", "local")
	t.Setenv("SESSION_COOKIE_SECURE", "true")

	logger := log.New(&bytes.Buffer{}, "", 0)
	if got := ResolveSessionCookieSecure(logger); !got {
		t.Fatalf("expected true when explicitly set")
	}
}

func TestResolveSessionCookieSecure_InvalidValueFallsBack(t *testing.T) {
	t.Setenv("APP_ENV", "local")
	t.Setenv("SESSION_COOKIE_SECURE", "nope")

	logger := log.New(&bytes.Buffer{}, "", 0)
	if got := ResolveSessionCookieSecure(logger); got {
		t.Fatalf("expected false fallback for local, got true")
	}
}

func TestResolveAllowPublicRegister_LocalDefaultsToTrue(t *testing.T) {
	t.Setenv("APP_ENV", "local")
	t.Setenv("ALLOW_PUBLIC_REGISTER", "")

	logger := log.New(&bytes.Buffer{}, "", 0)
	if got := ResolveAllowPublicRegister(logger); !got {
		t.Fatalf("expected true for local default, got false")
	}
}

func TestResolveAllowPublicRegister_NonLocalDefaultsToFalse(t *testing.T) {
	t.Setenv("APP_ENV", "staging")
	t.Setenv("ALLOW_PUBLIC_REGISTER", "")

	logger := log.New(&bytes.Buffer{}, "", 0)
	if got := ResolveAllowPublicRegister(logger); got {
		t.Fatalf("expected false for non-local default, got true")
	}
}

func TestResolveAllowPublicRegister_ExplicitValue(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("ALLOW_PUBLIC_REGISTER", "true")

	logger := log.New(&bytes.Buffer{}, "", 0)
	if got := ResolveAllowPublicRegister(logger); !got {
		t.Fatalf("expected true when explicitly set")
	}
}

func TestResolveAllowPublicRegister_InvalidValueFallsBack(t *testing.T) {
	t.Setenv("APP_ENV", "local")
	t.Setenv("ALLOW_PUBLIC_REGISTER", "nope")

	logger := log.New(&bytes.Buffer{}, "", 0)
	if got := ResolveAllowPublicRegister(logger); !got {
		t.Fatalf("expected true fallback for local, got false")
	}
}

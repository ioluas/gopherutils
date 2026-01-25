package parse

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ioluas/gopherutils/utils/file/ls/internal/config"
)

func TestParseArgsTimeStyleEnv(t *testing.T) {
	t.Setenv("TIME_STYLE", "long-iso")
	var stderr bytes.Buffer
	cfg, err := ParseArgs([]string{"-l"}, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.TimeStyleSpec == nil {
		t.Fatal("expected TimeStyleSpec to be set from TIME_STYLE")
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no warnings, got %q", stderr.String())
	}
}

func TestParseArgsTimeStyleEnvInvalidWarns(t *testing.T) {
	t.Setenv("TIME_STYLE", "+%Q")
	var stderr bytes.Buffer

	cfg, err := ParseArgs([]string{"-l"}, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.TimeStyleSpec != nil {
		t.Fatalf("expected TimeStyleSpec to be nil for invalid TIME_STYLE, got %#v", cfg.TimeStyleSpec)
	}

	out := stderr.String()
	if !strings.Contains(out, "TIME_STYLE") || !strings.Contains(out, "unsupported TIME_STYLE token") {
		t.Fatalf("expected TIME_STYLE warning about unsupported token, got %q", out)
	}
}

func TestParseArgsTimeStyleInvalidWarns(t *testing.T) {
	var stderr bytes.Buffer
	cfg, err := ParseArgs([]string{"-l", "--time-style=+%Q"}, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.TimeStyleSpec != nil {
		t.Fatalf("expected TimeStyleSpec to be nil for invalid time-style")
	}
	if !strings.Contains(stderr.String(), "unsupported TIME_STYLE token") {
		t.Fatalf("expected warning, got %q", stderr.String())
	}
}

func TestParseArgsFullTime(t *testing.T) {
	var stderr bytes.Buffer
	cfg, err := ParseArgs([]string{"-l", "--full-time"}, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.TimeStyleSpec == nil || cfg.TimeStyleSpec.Kind != config.TimeStyleFullISO {
		t.Fatalf("expected full-iso time style, got %v", cfg.TimeStyleSpec)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no warnings, got %q", stderr.String())
	}
}

func TestParseArgsFullTimeWithTimeStyleWarns(t *testing.T) {
	var stderr bytes.Buffer
	cfg, err := ParseArgs([]string{"-l", "--full-time", "--time-style=iso"}, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.TimeStyleSpec == nil || cfg.TimeStyleSpec.Kind != config.TimeStyleISO {
		t.Fatalf("expected time-style iso to win, got %v", cfg.TimeStyleSpec)
	}
	if !strings.Contains(stderr.String(), "ls: warning: --full-time is ignored when --time-style is used") {
		t.Fatalf("expected warning, got %q", stderr.String())
	}
}

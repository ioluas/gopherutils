package parse

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/ioluas/gopherutils/utils/file/ls/internal/config"
	"github.com/spf13/pflag"
)

func TestParseArgsHelp(t *testing.T) {
	var stderr bytes.Buffer
	_, err := ParseArgs([]string{"--help"}, &stderr)
	if !errors.Is(err, pflag.ErrHelp) {
		t.Fatalf("expected ErrHelp, got %v", err)
	}
	if !strings.Contains(stderr.String(), "Usage: ls") {
		t.Fatalf("expected usage output, got %q", stderr.String())
	}
}

func TestParseArgsInvalidFlag(t *testing.T) {
	var stderr bytes.Buffer
	_, err := ParseArgs([]string{"--nope"}, &stderr)
	if err == nil {
		t.Fatal("expected error for invalid flag")
	}
}

func TestParseArgsDefaultDirectory(t *testing.T) {
	var stderr bytes.Buffer
	cfg, err := ParseArgs([]string{}, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Directories) != 1 || cfg.Directories[0] != "." {
		t.Fatalf("expected default directory '.', got %v", cfg.Directories)
	}
}

func TestParseArgsDirectories(t *testing.T) {
	var stderr bytes.Buffer
	cfg, err := ParseArgs([]string{"a", "b"}, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Directories) != 2 || cfg.Directories[0] != "a" || cfg.Directories[1] != "b" {
		t.Fatalf("expected directories [a b], got %v", cfg.Directories)
	}
}

func TestParseArgsTimeFieldSet(t *testing.T) {
	var stderr bytes.Buffer
	cfg, err := ParseArgs([]string{"--time=access"}, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.TimeField != config.TimeFieldAccess {
		t.Fatalf("expected TimeFieldAccess, got %v", cfg.TimeField)
	}
	if !cfg.TimeFieldSet {
		t.Fatal("expected TimeFieldSet to be true")
	}
}

func TestParseArgsBlockSizeValid(t *testing.T) {
	var stderr bytes.Buffer
	cfg, err := ParseArgs([]string{"--block-size=1K", "-l"}, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.BlockSize == nil || cfg.BlockSize.SizeBytes != 1024 {
		t.Fatalf("expected block size 1024, got %#v", cfg.BlockSize)
	}
}

func TestParseArgsBlockSizeInvalidWarns(t *testing.T) {
	var stderr bytes.Buffer
	cfg, err := ParseArgs([]string{"--block-size=1X", "-l"}, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.BlockSize != nil {
		t.Fatalf("expected BlockSize to be nil, got %#v", cfg.BlockSize)
	}
	if !strings.Contains(stderr.String(), "unknown SIZE suffix") {
		t.Fatalf("expected warning about size suffix, got %q", stderr.String())
	}
}

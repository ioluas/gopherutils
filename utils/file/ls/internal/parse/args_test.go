package parse

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/ioluas/gopherutils/utils/file/ls/internal/config"
	"github.com/spf13/pflag"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		checkConfig func(t *testing.T, config *config.Config)
		setup       func() func()
	}{
		{
			name:        "no arguments - use current directory",
			args:        []string{},
			expectError: false,
			checkConfig: func(t *testing.T, config *config.Config) {
				if config.ShowAll {
					t.Error("Expected ShowAll to be false")
				}
				if config.HumanReadable {
					t.Error("Expected HumanReadable to be false")
				}
				if len(config.Directories) != 1 || config.Directories[0] != "." {
					t.Errorf("Expected directory \".\", got %v", config.Directories)
				}
			},
		},
		{
			name:        "single directory argument",
			args:        []string{"/tmp"},
			expectError: false,
			checkConfig: func(t *testing.T, config *config.Config) {
				if config.ShowAll {
					t.Error("Expected ShowAll to be false")
				}
				if config.HumanReadable {
					t.Error("Expected HumanReadable to be false")
				}
				if len(config.Directories) != 1 || config.Directories[0] != "/tmp" {
					t.Errorf("Expected directory /tmp, got %v", config.Directories)
				}
			},
		},
		{
			name:        "multiple directory arguments",
			args:        []string{"/tmp", "/var"},
			expectError: false,
			checkConfig: func(t *testing.T, config *config.Config) {
				if len(config.Directories) != 2 {
					t.Errorf("Expected 2 directories, got %d", len(config.Directories))
				}
				if config.Directories[0] != "/tmp" {
					t.Errorf("Expected first directory /tmp, got %s", config.Directories[0])
				}
				if config.Directories[1] != "/var" {
					t.Errorf("Expected second directory /var, got %s", config.Directories[1])
				}
			},
		},
		{
			name:        "multiple directory arguments with flags",
			args:        []string{"-a", "/tmp", "/var", "-l"},
			expectError: false,
			checkConfig: func(t *testing.T, config *config.Config) {
				if !config.ShowAll {
					t.Error("Expected ShowAll to be true")
				}
				if !config.LongListing {
					t.Error("Expected LongListing to be true")
				}
				if len(config.Directories) != 2 {
					t.Errorf("Expected 2 directories, got %d", len(config.Directories))
				}
				if config.Directories[0] != "/tmp" {
					t.Errorf("Expected first directory /tmp, got %s", config.Directories[0])
				}
				if config.Directories[1] != "/var" {
					t.Errorf("Expected second directory /var, got %s", config.Directories[1])
				}
			},
		},
		{
			name:        "-a flag only",
			args:        []string{"-a"},
			expectError: false,
			checkConfig: func(t *testing.T, config *config.Config) {
				if !config.ShowAll {
					t.Error("Expected ShowAll to be true")
				}
				if len(config.Directories) != 1 || config.Directories[0] != "." {
					t.Errorf("Expected directory \".\", got %v", config.Directories)
				}
			},
		},
		{
			name:        "--all flag only",
			args:        []string{"--all"},
			expectError: false,
			checkConfig: func(t *testing.T, config *config.Config) {
				if !config.ShowAll {
					t.Error("Expected ShowAll to be true")
				}
				if len(config.Directories) != 1 || config.Directories[0] != "." {
					t.Errorf("Expected directory \".\", got %v", config.Directories)
				}
			},
		},
		{
			name:        "-a flag with directory",
			args:        []string{"-a", "/tmp"},
			expectError: false,
			checkConfig: func(t *testing.T, config *config.Config) {
				if !config.ShowAll {
					t.Error("Expected ShowAll to be true")
				}
				if len(config.Directories) != 1 || config.Directories[0] != "/tmp" {
					t.Errorf("Expected directory /tmp, got %v", config.Directories)
				}
			},
		},
		{
			name:        "directory after -a flag",
			args:        []string{"-a", "/var"},
			expectError: false,
			checkConfig: func(t *testing.T, config *config.Config) {
				if !config.ShowAll {
					t.Error("Expected ShowAll to be true")
				}
				if len(config.Directories) != 1 || config.Directories[0] != "/var" {
					t.Errorf("Expected directory /var, got %v", config.Directories)
				}
			},
		},
		{
			name:        "directory with --all flag",
			args:        []string{"/tmp", "--all"},
			expectError: false,
			checkConfig: func(t *testing.T, config *config.Config) {
				if !config.ShowAll {
					t.Error("Expected ShowAll to be true")
				}
				if len(config.Directories) != 1 || config.Directories[0] != "/tmp" {
					t.Errorf("Expected directory /tmp, got %v", config.Directories)
				}
			},
		},
		{
			name:        "--author flag only",
			args:        []string{"--author"},
			expectError: false,
			checkConfig: func(t *testing.T, config *config.Config) {
				if !config.ShowAuthor {
					t.Error("Expected ShowAuthor to be true")
				}
				if config.LongListing { // --author alone should not set -l
					t.Error("Expected LongListing to be false")
				}
			},
		},
		{
			name:        "-G flag only",
			args:        []string{"-G"},
			expectError: false,
			checkConfig: func(t *testing.T, config *config.Config) {
				if !config.NoGroup {
					t.Error("Expected NoGroup to be true")
				}
				if config.LongListing {
					t.Error("Expected LongListing to be false")
				}
			},
		},
		{
			name:        "--no-group flag only",
			args:        []string{"--no-group"},
			expectError: false,
			checkConfig: func(t *testing.T, config *config.Config) {
				if !config.NoGroup {
					t.Error("Expected NoGroup to be true")
				}
				if config.LongListing {
					t.Error("Expected LongListing to be false")
				}
			},
		},
		{
			name:        "--full-time flag only",
			args:        []string{"--full-time"},
			expectError: false,
			checkConfig: func(t *testing.T, config *config.Config) {
				if !config.FullTime {
					t.Error("Expected FullTime to be true")
				}
			},
		},
		{
			name:        "-l --author flags",
			args:        []string{"-l", "--author"},
			expectError: false,
			checkConfig: func(t *testing.T, config *config.Config) {
				if !config.LongListing {
					t.Error("Expected LongListing to be true")
				}
				if !config.ShowAuthor {
					t.Error("Expected ShowAuthor to be true")
				}
			},
		},
		{
			name:        "-A flag only",
			args:        []string{"-A"},
			expectError: false,
			checkConfig: func(t *testing.T, config *config.Config) {
				if !config.ShowAlmostAll {
					t.Error("Expected ShowAlmostAll to be true")
				}
				if config.ShowAll { // Ensure -a is not implicitly set
					t.Error("Expected ShowAll to be false")
				}
				if len(config.Directories) != 1 || config.Directories[0] != "." {
					t.Errorf("Expected directory \".\", got %v", config.Directories)
				}
			},
		},
		{
			name:        "--almost-all flag only",
			args:        []string{"--almost-all"},
			expectError: false,
			checkConfig: func(t *testing.T, config *config.Config) {
				if !config.ShowAlmostAll {
					t.Error("Expected ShowAlmostAll to be true")
				}
				if config.ShowAll { // Ensure -a is not implicitly set
					t.Error("Expected ShowAll to be false")
				}
				if len(config.Directories) != 1 || config.Directories[0] != "." {
					t.Errorf("Expected directory \".\", got %v", config.Directories)
				}
			},
		},
		{
			name:        "-h flag",
			args:        []string{"-h"},
			expectError: false,
			checkConfig: func(t *testing.T, config *config.Config) {
				if !config.HumanReadable {
					t.Error("Expected HumanReadable to be true")
				}
			},
		},
		{
			name:        "--human-readable flag",
			args:        []string{"--human-readable"},
			expectError: false,
			checkConfig: func(t *testing.T, config *config.Config) {
				if !config.HumanReadable {
					t.Error("Expected HumanReadable to be true")
				}
			},
		},
		{
			name:        "-b flag",
			args:        []string{"-b"},
			expectError: false,
			checkConfig: func(t *testing.T, config *config.Config) {
				if !config.Escape {
					t.Error("Expected Escape to be true")
				}
			},
		},
		{
			name:        "--escape flag",
			args:        []string{"--escape"},
			expectError: false,
			checkConfig: func(t *testing.T, config *config.Config) {
				if !config.Escape {
					t.Error("Expected Escape to be true")
				}
			},
		},
		{
			name:        "-B flag",
			args:        []string{"-B"},
			expectError: false,
			checkConfig: func(t *testing.T, config *config.Config) {
				if !config.IgnoreBackups {
					t.Error("Expected IgnoreBackups to be true")
				}
			},
		},
		{
			name:        "--ignore-backups flag",
			args:        []string{"--ignore-backups"},
			expectError: false,
			checkConfig: func(t *testing.T, config *config.Config) {
				if !config.IgnoreBackups {
					t.Error("Expected IgnoreBackups to be true")
				}
			},
		},
		{
			name:        "--si flag",
			args:        []string{"--si"},
			expectError: false,
			checkConfig: func(t *testing.T, config *config.Config) {
				if !config.SI {
					t.Error("Expected SI to be true")
				}
			},
		},
		{
			name:        "-d flag",
			args:        []string{"-d"},
			expectError: false,
			checkConfig: func(t *testing.T, config *config.Config) {
				if !config.ListDirectory {
					t.Error("Expected ListDirectory to be true")
				}
			},
		},
		{
			name:        "--directory flag",
			args:        []string{"--directory"},
			expectError: false,
			checkConfig: func(t *testing.T, config *config.Config) {
				if !config.ListDirectory {
					t.Error("Expected ListDirectory to be true")
				}
			},
		},
		{
			name:        "-C flag only",
			args:        []string{"-C"},
			expectError: false,
			checkConfig: func(t *testing.T, config *config.Config) {
				if !config.Columnate {
					t.Error("Expected Columnate to be true")
				}
			},
		},
		{
			name:        "unknown flag",
			args:        []string{"-x"},
			expectError: true,
		},
		{
			name:        "--help flag",
			args:        []string{"--help"},
			expectError: true,
			checkConfig: func(t *testing.T, config *config.Config) {
				// When help is requested, config should be nil
				if config != nil {
					t.Error("Expected config to be nil when help is requested")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(innerT *testing.T) {
			if tt.setup != nil {
				cleanup := tt.setup()
				defer cleanup()
			}
			config, err := ParseArgs(tt.args, io.Discard)

			if tt.expectError {
				if err == nil {
					innerT.Error("Expected error but got none")
				}
				if tt.name == "--help flag" && !errors.Is(err, pflag.ErrHelp) {
					innerT.Errorf("Expected pflag.ErrHelp for --help, but got %v", err)
				}
				// For help flag, config should be nil
				if tt.name == "--help flag" && config != nil {
					innerT.Error("Expected config to be nil when help is requested")
				}
			} else if err != nil {
				innerT.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError && tt.checkConfig != nil {
				tt.checkConfig(innerT, config)
			} else if tt.expectError && tt.checkConfig != nil {
				tt.checkConfig(innerT, config)
			}
		})
	}
}

func TestParseArgsLong(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		expectedLL bool
	}{
		{
			name:       "short flag -l",
			args:       []string{"-l"},
			expectedLL: true,
		},
		{
			name:       "long flag --long",
			args:       []string{"--long"},
			expectedLL: true,
		},
		{
			name:       "no flag",
			args:       []string{},
			expectedLL: false,
		},
		{
			name:       "combined -la",
			args:       []string{"-al"},
			expectedLL: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ParseArgs(tt.args, io.Discard)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if config.LongListing != tt.expectedLL {
				t.Errorf("Expected LongListing=%v, got %v", tt.expectedLL, config.LongListing)
			}
		})
	}
}

func TestParseArgsTimeFlags(t *testing.T) {
	var stderr bytes.Buffer
	cfg, err := ParseArgs([]string{"-t"}, &stderr)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !cfg.SortTime {
		t.Fatalf("Expected SortTime to be true")
	}

	cfg, err = ParseArgs([]string{"--time=access"}, &stderr)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if cfg.TimeField != config.TimeFieldAccess {
		t.Fatalf("Expected TimeField access, got %v", cfg.TimeField)
	}
	if !cfg.TimeFieldSet {
		t.Fatalf("Expected TimeFieldSet to be true")
	}
}

func TestParseArgsInvalidTimeWord(t *testing.T) {
	var stderr bytes.Buffer
	_, err := ParseArgs([]string{"--time=invalid"}, &stderr)
	if err == nil {
		t.Fatalf("Expected error for invalid time word")
	}
}

func TestParseArgsBlockSize(t *testing.T) {
	t.Run("valid block size", func(t *testing.T) {
		var stderr bytes.Buffer
		// shadowing config package here too?
		cfg, err := ParseArgs([]string{"--block-size=1K", "-l"}, &stderr)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if cfg.BlockSize == nil {
			t.Fatal("Expected BlockSize to be set")
		}
		if cfg.BlockSize.SizeBytes != 1024 {
			t.Errorf("Expected BlockSize sizeBytes=1024, got %d", cfg.BlockSize.SizeBytes)
		}
		if stderr.Len() != 0 {
			t.Errorf("Expected no warnings, got %q", stderr.String())
		}
	})

	t.Run("invalid block size warns and drops", func(t *testing.T) {
		var stderr bytes.Buffer
		cfg, err := ParseArgs([]string{"--block-size=1X", "-l"}, &stderr)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if cfg.BlockSize != nil {
			t.Fatal("Expected BlockSize to be nil for invalid size")
		}
		if !strings.Contains(stderr.String(), "unknown SIZE suffix") {
			t.Errorf("Expected warning about unknown SIZE suffix, got %q", stderr.String())
		}
	})
}

func TestParseArgsErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "invalid flag combination",
			args:        []string{"-xyz"},
			expectError: true,
		},
		{
			name:        "help short flag",
			args:        []string{"-?"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ParseArgs(tt.args, io.Discard)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if cfg != nil {
					t.Error("Expected config to be nil on error")
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

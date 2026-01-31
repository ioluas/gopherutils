package parse

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

func TestArgs(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		wantSources   []string
		wantDest      string
		wantVerbose   bool
		wantErr       bool
		wantErrString string
		wantHelp      bool
	}{
		{
			name:        "Basic usage",
			args:        []string{"src", "dest"},
			wantSources: []string{"src"},
			wantDest:    "dest",
			wantVerbose: false,
			wantErr:     false,
		},
		{
			name:        "Verbose flag short",
			args:        []string{"-v", "src", "dest"},
			wantSources: []string{"src"},
			wantDest:    "dest",
			wantVerbose: true,
			wantErr:     false,
		},
		{
			name:        "Verbose flag long",
			args:        []string{"--verbose", "src", "dest"},
			wantSources: []string{"src"},
			wantDest:    "dest",
			wantVerbose: true,
			wantErr:     false,
		},
		{
			name:        "Multiple sources",
			args:        []string{"src1", "src2", "dir"},
			wantSources: []string{"src1", "src2"},
			wantDest:    "dir",
			wantVerbose: false,
			wantErr:     false,
		},
		{
			name:     "Help flag",
			args:     []string{"--help"},
			wantErr:  true,
			wantHelp: true,
		},
		{
			name:     "Short help flag",
			args:     []string{"-?"},
			wantErr:  true,
			wantHelp: true,
		},
		{
			name:          "Missing operand (no args)",
			args:          []string{},
			wantErr:       true,
			wantErrString: "missing file operand",
		},
		{
			name:          "Missing operand (one arg)",
			args:          []string{"src"},
			wantErr:       true,
			wantErrString: "missing file operand",
		},
		{
			name:          "Invalid flag",
			args:          []string{"--invalid", "src", "dest"},
			wantErr:       true,
			wantErrString: "unknown flag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stderr bytes.Buffer
			cfg, err := Args(tt.args, &stderr)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Args() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if tt.wantHelp {
					if !errors.Is(err, pflag.ErrHelp) {
						t.Errorf("Args() error = %v, want pflag.ErrHelp", err)
					}
					// Check if usage was printed to stderr
					if !strings.Contains(stderr.String(), "Usage: cp") {
						t.Errorf("Args() stderr = %q, want usage info", stderr.String())
					}
				} else if tt.wantErrString != "" {
					if !strings.Contains(err.Error(), tt.wantErrString) {
						t.Errorf("Args() error = %v, want error containing %q", err, tt.wantErrString)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("Args() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(cfg.Sources, tt.wantSources) {
				t.Errorf("Args() Sources = %v, want %v", cfg.Sources, tt.wantSources)
			}
			if cfg.Dest != tt.wantDest {
				t.Errorf("Args() Dest = %v, want %v", cfg.Dest, tt.wantDest)
			}
			if cfg.Verbose != tt.wantVerbose {
				t.Errorf("Args() Verbose = %v, want %v", cfg.Verbose, tt.wantVerbose)
			}
		})
	}
}

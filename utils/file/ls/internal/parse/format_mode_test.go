package parse

import (
	"io"
	"testing"

	"github.com/ioluas/gopherutils/utils/file/ls/internal/config"
)

func TestFormatModeLastFlagWins(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		expectedMode config.FormatMode
	}{
		{
			name:         "-C only",
			args:         []string{"-C"},
			expectedMode: config.FormatColumnate,
		},
		{
			name:         "-1 only",
			args:         []string{"-1"},
			expectedMode: config.FormatOnePerLine,
		},
		{
			name:         "-C then -1 (last wins)",
			args:         []string{"-C", "-1"},
			expectedMode: config.FormatOnePerLine,
		},
		{
			name:         "-1 then -C (last wins)",
			args:         []string{"-1", "-C"},
			expectedMode: config.FormatColumnate,
		},
		{
			name:         "no format flags",
			args:         []string{"-a"},
			expectedMode: config.FormatDefault,
		},
		{
			name:         "-C -1 -C (last -C wins)",
			args:         []string{"-C", "-1", "-C"},
			expectedMode: config.FormatColumnate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ParseArgs(tt.args, io.Discard)
			if err != nil {
				t.Fatalf("ParseArgs() error = %v", err)
			}
			if cfg.FormatMode != tt.expectedMode {
				t.Errorf("FormatMode = %v, want %v", cfg.FormatMode, tt.expectedMode)
			}
		})
	}
}

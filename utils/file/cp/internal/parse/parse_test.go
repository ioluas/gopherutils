package parse

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/ioluas/gopherutils/utils/file/cp/internal/config"
	"github.com/spf13/pflag"
)

func TestArgs(t *testing.T) {
	tests := []struct {
		name             string
		args             []string
		wantSources      []string
		wantDest         string
		wantVerbose      bool
		wantBackup       bool
		wantBackupMethod config.BackupMethod
		wantUpdate       config.UpdateMode
		wantErr          bool
		wantErrString    string
		wantHelp         bool
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
		{
			name:             "Backup flag short",
			args:             []string{"-b", "src", "dest"},
			wantSources:      []string{"src"},
			wantDest:         "dest",
			wantVerbose:      false,
			wantBackup:       true,
			wantBackupMethod: config.BackupExisting, // default
			wantErr:          false,
		},
		{
			name:             "Backup flag long default",
			args:             []string{"--backup", "src", "dest"},
			wantSources:      []string{"src"},
			wantDest:         "dest",
			wantVerbose:      false,
			wantBackup:       true,
			wantBackupMethod: config.BackupExisting,
			wantErr:          false,
		},
		{
			name:             "Backup flag long numbered",
			args:             []string{"--backup=numbered", "src", "dest"},
			wantSources:      []string{"src"},
			wantDest:         "dest",
			wantVerbose:      false,
			wantBackup:       true,
			wantBackupMethod: config.BackupNumbered,
			wantErr:          false,
		},
		{
			name:             "Backup flag long t",
			args:             []string{"--backup=t", "src", "dest"},
			wantSources:      []string{"src"},
			wantDest:         "dest",
			wantBackup:       true,
			wantBackupMethod: config.BackupNumbered,
			wantErr:          false,
		},
		{
			name:             "Backup flag long none",
			args:             []string{"--backup=none", "src", "dest"},
			wantSources:      []string{"src"},
			wantDest:         "dest",
			wantBackup:       true,
			wantBackupMethod: config.BackupNone,
			wantErr:          false,
		},
		{
			name:             "Backup flag long off",
			args:             []string{"--backup=off", "src", "dest"},
			wantSources:      []string{"src"},
			wantDest:         "dest",
			wantBackup:       true,
			wantBackupMethod: config.BackupNone,
			wantErr:          false,
		},
		{
			name:             "Backup flag long simple",
			args:             []string{"--backup=simple", "src", "dest"},
			wantSources:      []string{"src"},
			wantDest:         "dest",
			wantBackup:       true,
			wantBackupMethod: config.BackupSimple,
			wantErr:          false,
		},
		{
			name:             "Backup flag long never",
			args:             []string{"--backup=never", "src", "dest"},
			wantSources:      []string{"src"},
			wantDest:         "dest",
			wantBackup:       true,
			wantBackupMethod: config.BackupSimple,
			wantErr:          false,
		},
		{
			name:             "Backup flag long existing",
			args:             []string{"--backup=existing", "src", "dest"},
			wantSources:      []string{"src"},
			wantDest:         "dest",
			wantBackup:       true,
			wantBackupMethod: config.BackupExisting,
			wantErr:          false,
		},
		{
			name:             "Backup flag long nil",
			args:             []string{"--backup=nil", "src", "dest"},
			wantSources:      []string{"src"},
			wantDest:         "dest",
			wantBackup:       true,
			wantBackupMethod: config.BackupExisting,
			wantErr:          false,
		},
		{
			name:          "Backup invalid",
			args:          []string{"--backup=exsting", "src", "dest"},
			wantErr:       true,
			wantErrString: "invalid argument 'exsting' for '--backup'",
		},
		{
			name:        "Suffix flag",
			args:        []string{"-S", ".bak", "src", "dest"},
			wantSources: []string{"src"},
			wantDest:    "dest",
			wantErr:     false,
		},
		{
			name:        "Update flag no arg (default older)",
			args:        []string{"--update", "src", "dest"},
			wantSources: []string{"src"},
			wantDest:    "dest",
			wantUpdate:  config.UpdateReplaceOlder,
			wantErr:     false,
		},
		{
			name:        "Update flag short (default older)",
			args:        []string{"-u", "src", "dest"},
			wantSources: []string{"src"},
			wantDest:    "dest",
			wantUpdate:  config.UpdateReplaceOlder,
			wantErr:     false,
		},
		{
			name:        "Update flag all",
			args:        []string{"--update=all", "src", "dest"},
			wantSources: []string{"src"},
			wantDest:    "dest",
			wantUpdate:  config.UpdateReplaceAll,
			wantErr:     false,
		},
		{
			name:        "Update flag none",
			args:        []string{"--update=none", "src", "dest"},
			wantSources: []string{"src"},
			wantDest:    "dest",
			wantUpdate:  config.UpdateNone,
			wantErr:     false,
		},
		{
			name:        "Update flag none-fail",
			args:        []string{"--update=none-fail", "src", "dest"},
			wantSources: []string{"src"},
			wantDest:    "dest",
			wantUpdate:  config.UpdateNoneFail,
			wantErr:     false,
		},
		{
			name:        "Update flag older",
			args:        []string{"--update=older", "src", "dest"},
			wantSources: []string{"src"},
			wantDest:    "dest",
			wantUpdate:  config.UpdateReplaceOlder,
			wantErr:     false,
		},
		{
			name:          "Update flag invalid",
			args:          []string{"--update=invalid", "src", "dest"},
			wantErr:       true,
			wantErrString: "invalid argument 'invalid' for '--update'",
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
			if cfg.Backup != tt.wantBackup {
				t.Errorf("Args() Backup = %v, want %v", cfg.Backup, tt.wantBackup)
			}
			if tt.wantBackup && cfg.BackupMethod != tt.wantBackupMethod {
				t.Errorf("Args() BackupMethod = %v, want %v", cfg.BackupMethod, tt.wantBackupMethod)
			}
			if cfg.UpdateMode != tt.wantUpdate {
				t.Errorf("Args() UpdateMode = %v, want %v", cfg.UpdateMode, tt.wantUpdate)
			}
		})
	}

	t.Run("BackupEnvInvalid", func(t *testing.T) {
		t.Setenv("VERSION_CONTROL", "exsting")
		var stderr bytes.Buffer
		cfg, err := Args([]string{"-b", "src", "dest"}, &stderr)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !cfg.Backup {
			t.Error("want Backup true")
		}
		if cfg.BackupMethod != config.BackupExisting {
			t.Errorf("BackupMethod = %v, want BackupExisting", cfg.BackupMethod)
		}
	})
}

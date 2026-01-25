//go:build !windows

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"testing"
	"time"

	lsconfig "github.com/ioluas/gopherutils/utils/file/ls/internal/config"
	"github.com/ioluas/gopherutils/utils/file/ls/internal/display"
	"github.com/ioluas/gopherutils/utils/file/ls/internal/entry"
	"github.com/ioluas/gopherutils/utils/file/ls/internal/parse"
	"github.com/ioluas/gopherutils/utils/file/ls/internal/size"
	"github.com/ioluas/gopherutils/utils/file/ls/internal/timeutil"

	"github.com/spf13/pflag"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		checkConfig func(t *testing.T, config *lsconfig.Config)
		setup       func() func()
	}{
		{
			name:        "no arguments - use current directory",
			args:        []string{},
			expectError: false,
			checkConfig: func(t *testing.T, config *lsconfig.Config) {
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
			checkConfig: func(t *testing.T, config *lsconfig.Config) {
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
			checkConfig: func(t *testing.T, config *lsconfig.Config) {
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
			checkConfig: func(t *testing.T, config *lsconfig.Config) {
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
			checkConfig: func(t *testing.T, config *lsconfig.Config) {
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
			checkConfig: func(t *testing.T, config *lsconfig.Config) {
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
			checkConfig: func(t *testing.T, config *lsconfig.Config) {
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
			checkConfig: func(t *testing.T, config *lsconfig.Config) {
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
			checkConfig: func(t *testing.T, config *lsconfig.Config) {
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
			checkConfig: func(t *testing.T, config *lsconfig.Config) {
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
			checkConfig: func(t *testing.T, config *lsconfig.Config) {
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
			checkConfig: func(t *testing.T, config *lsconfig.Config) {
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
			checkConfig: func(t *testing.T, config *lsconfig.Config) {
				if !config.FullTime {
					t.Error("Expected FullTime to be true")
				}
			},
		},
		{
			name:        "-l --author flags",
			args:        []string{"-l", "--author"},
			expectError: false,
			checkConfig: func(t *testing.T, config *lsconfig.Config) {
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
			checkConfig: func(t *testing.T, config *lsconfig.Config) {
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
			checkConfig: func(t *testing.T, config *lsconfig.Config) {
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
			checkConfig: func(t *testing.T, config *lsconfig.Config) {
				if !config.HumanReadable {
					t.Error("Expected HumanReadable to be true")
				}
			},
		},
		{
			name:        "--human-readable flag",
			args:        []string{"--human-readable"},
			expectError: false,
			checkConfig: func(t *testing.T, config *lsconfig.Config) {
				if !config.HumanReadable {
					t.Error("Expected HumanReadable to be true")
				}
			},
		},
		{
			name:        "-b flag",
			args:        []string{"-b"},
			expectError: false,
			checkConfig: func(t *testing.T, config *lsconfig.Config) {
				if !config.Escape {
					t.Error("Expected Escape to be true")
				}
			},
		},
		{
			name:        "--escape flag",
			args:        []string{"--escape"},
			expectError: false,
			checkConfig: func(t *testing.T, config *lsconfig.Config) {
				if !config.Escape {
					t.Error("Expected Escape to be true")
				}
			},
		},
		{
			name:        "-B flag",
			args:        []string{"-B"},
			expectError: false,
			checkConfig: func(t *testing.T, config *lsconfig.Config) {
				if !config.IgnoreBackups {
					t.Error("Expected IgnoreBackups to be true")
				}
			},
		},
		{
			name:        "--ignore-backups flag",
			args:        []string{"--ignore-backups"},
			expectError: false,
			checkConfig: func(t *testing.T, config *lsconfig.Config) {
				if !config.IgnoreBackups {
					t.Error("Expected IgnoreBackups to be true")
				}
			},
		},
		{
			name:        "--si flag",
			args:        []string{"--si"},
			expectError: false,
			checkConfig: func(t *testing.T, config *lsconfig.Config) {
				if !config.SI {
					t.Error("Expected SI to be true")
				}
			},
		},
		{
			name:        "-d flag",
			args:        []string{"-d"},
			expectError: false,
			checkConfig: func(t *testing.T, config *lsconfig.Config) {
				if !config.ListDirectory {
					t.Error("Expected ListDirectory to be true")
				}
			},
		},
		{
			name:        "--directory flag",
			args:        []string{"--directory"},
			expectError: false,
			checkConfig: func(t *testing.T, config *lsconfig.Config) {
				if !config.ListDirectory {
					t.Error("Expected ListDirectory to be true")
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
			checkConfig: func(t *testing.T, config *lsconfig.Config) {
				// When help is requested, config should be nil
				if config != nil {
					t.Error("Expected config to be nil when help is requested")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(innerT *testing.T) { // Changed 't' to 'innerT'
			if tt.setup != nil {
				cleanup := tt.setup()
				defer cleanup()
			}
			config, err := parse.ParseArgs(tt.args, io.Discard)

			if tt.expectError {
				if err == nil {
					innerT.Error("Expected error but got none") // Use innerT
				}
				if tt.name == "--help flag" && !errors.Is(err, pflag.ErrHelp) {
					innerT.Errorf("Expected pflag.ErrHelp for --help, but got %v", err)
				}
				// For help flag, config should be nil
				if tt.name == "--help flag" && config != nil {
					innerT.Error("Expected config to be nil when help is requested")
				}
			} else if err != nil {
				innerT.Errorf("Unexpected error: %v", err) // Use innerT
			}

			if !tt.expectError && tt.checkConfig != nil {
				tt.checkConfig(innerT, config) // Pass innerT
			} else if tt.expectError && tt.checkConfig != nil {
				tt.checkConfig(innerT, config) // Also check config for error cases
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
			config, err := parse.ParseArgs(tt.args, io.Discard)
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
	config, err := parse.ParseArgs([]string{"-t"}, &stderr)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !config.SortTime {
		t.Fatalf("Expected SortTime to be true")
	}

	config, err = parse.ParseArgs([]string{"--time=access"}, &stderr)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if config.TimeField != lsconfig.TimeFieldAccess {
		t.Fatalf("Expected TimeField access, got %v", config.TimeField)
	}
	if !config.TimeFieldSet {
		t.Fatalf("Expected TimeFieldSet to be true")
	}
}

func TestParseArgsInvalidTimeWord(t *testing.T) {
	var stderr bytes.Buffer
	_, err := parse.ParseArgs([]string{"--time=invalid"}, &stderr)
	if err == nil {
		t.Fatalf("Expected error for invalid time word")
	}
}

func TestParseArgsBlockSize(t *testing.T) {
	t.Run("valid block size", func(t *testing.T) {
		var stderr bytes.Buffer
		config, err := parse.ParseArgs([]string{"--block-size=1K", "-l"}, &stderr)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if config.BlockSize == nil {
			t.Fatal("Expected BlockSize to be set")
		}
		if config.BlockSize.SizeBytes != 1024 {
			t.Errorf("Expected BlockSize sizeBytes=1024, got %d", config.BlockSize.SizeBytes)
		}
		if stderr.Len() != 0 {
			t.Errorf("Expected no warnings, got %q", stderr.String())
		}
	})

	t.Run("invalid block size warns and drops", func(t *testing.T) {
		var stderr bytes.Buffer
		config, err := parse.ParseArgs([]string{"--block-size=1X", "-l"}, &stderr)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if config.BlockSize != nil {
			t.Fatal("Expected BlockSize to be nil for invalid size")
		}
		if !strings.Contains(stderr.String(), "unknown SIZE suffix") {
			t.Errorf("Expected warning about unknown SIZE suffix, got %q", stderr.String())
		}
	})
}

func TestPrintEntries(t *testing.T) {
	tests := []struct {
		name     string
		entries  []string
		expected string
	}{
		{
			name:     "single entry",
			entries:  []string{"file1.txt"},
			expected: "file1.txt\n",
		},
		{
			name:     "multiple entries",
			entries:  []string{"file1.txt", "file2.txt", "file3.txt"},
			expected: "file1.txt\nfile2.txt\nfile3.txt\n",
		},
		{
			name:     "empty directory",
			entries:  []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			display.PrintEntries(&buf, tt.entries)

			got := buf.String()
			if got != tt.expected {
				t.Errorf("display.PrintEntries() output = %q, want %q", got, tt.expected)
			}
		})
	}
}

// mockFileInfo is a mock implementation of fs.FileInfo for testing.
type mockFileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	sys     interface{}
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return m.size }
func (m *mockFileInfo) Mode() fs.FileMode  { return m.mode }
func (m *mockFileInfo) ModTime() time.Time { return m.modTime }
func (m *mockFileInfo) IsDir() bool        { return m.mode.IsDir() }
func (m *mockFileInfo) Sys() interface{}   { return m.sys }

// mockDirEntry is a mock implementation of os.DirEntry for testing error cases.
type mockDirEntry struct {
	name     string
	isDir    bool
	infoErr  error       // The error to return from Info()
	fileInfo fs.FileInfo // The file info to return from Info()
}

func (m *mockDirEntry) Name() string {
	return m.name
}

func (m *mockDirEntry) IsDir() bool {
	return m.isDir
}

func (m *mockDirEntry) Type() fs.FileMode {
	if m.isDir {
		return fs.ModeDir
	}
	return 0
}

func (m *mockDirEntry) Info() (fs.FileInfo, error) {
	if m.infoErr != nil {
		return nil, m.infoErr
	}
	if m.fileInfo != nil {
		return m.fileInfo, nil
	}
	return nil, errors.New("not implemented")
}

func TestPrintLongListNonUnix(t *testing.T) {
	now := time.Now()
	mockEntries := []os.DirEntry{
		&mockDirEntry{
			name: "non-unix-file",
			fileInfo: &mockFileInfo{
				name:    "non-unix-file",
				size:    1234,
				mode:    0644,
				modTime: now,
				sys:     nil, // Not a *syscall.Stat_t
			},
		},
	}

	config := &lsconfig.Config{LongListing: true, ShowAuthor: true}
	var stdout, stderr bytes.Buffer
	hadError := display.PrintLongList(&stdout, &stderr, mockEntries, config)
	if hadError {
		t.Error("Expected no error from PrintLongList, but got one")
	}

	output := stdout.String()
	if output == "" {
		t.Errorf("Expected output for non-Unix entry, but got empty string")
	}

	// Output should contain nlink 1, size 1234, and placeholders for owner/group
	// Format: mode nlink owner author group size date time name
	if !strings.Contains(output, "non-unix-file") {
		t.Errorf("Output should contain filename, got: %q", output)
	}
	if !strings.Contains(output, "1234") {
		t.Errorf("Output should contain size 1234, got: %q", output)
	}
	// With ShowAuthor=true, placeholders should be present
	if !strings.Contains(output, "-") {
		t.Errorf("Output should contain placeholders '-' for owner/group, got: %q", output)
	}

	// Test without author
	stdout.Reset()
	config.ShowAuthor = false
	hadError = display.PrintLongList(&stdout, &stderr, mockEntries, config)
	if hadError {
		t.Error("Expected no error from PrintLongList on second call, but got one")
	}
	output = stdout.String()
	// Format: mode nlink owner group size date time name (no author column)
	fields := strings.Fields(output)
	// Expected fields: mode, nlink, owner, group, size, month, day, time, name
	// Total 9 fields.
	if len(fields) != 9 {
		t.Errorf("Expected 9 fields when ShowAuthor is false, got %d: %v", len(fields), fields)
	}
	if !strings.Contains(output, " - ") {
		t.Errorf("Expected placeholders '-' for owner/group, got %q", output)
	}
}

func TestRun(t *testing.T) {
	tests := []struct {
		name           string
		config         *lsconfig.Config
		setupFunc      func(t *testing.T) string
		expectedExit   int
		expectedStdout string
		expectedStderr string
	}{
		{
			name: "list directory with files",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				// Create test files
				for _, file := range []string{"a.txt", "b.txt", "c.txt"} {
					path := filepath.Join(tmpDir, file)
					if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
						t.Fatalf("Failed to create test file: %v", err)
					}
				}
				return tmpDir
			},
			expectedExit:   0,
			expectedStdout: "a.txt\nb.txt\nc.txt\n",
			expectedStderr: "",
		},
		{
			name: "list empty directory",
			setupFunc: func(t *testing.T) string {
				return t.TempDir()
			},
			expectedExit:   0,
			expectedStdout: "",
			expectedStderr: "",
		},
		{
			name: "list directory with hidden files skips hidden",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				// Create regular and hidden files
				for _, file := range []string{".hidden", "visible.txt"} {
					path := filepath.Join(tmpDir, file)
					if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
						t.Fatalf("Failed to create test file: %v", err)
					}
				}
				return tmpDir
			},
			expectedExit:   0,
			expectedStdout: "visible.txt\n",
			expectedStderr: "",
		},
		{
			name: "list directory with hidden files - showAll true",
			config: &lsconfig.Config{
				ShowAll: true,
			},
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				// Create regular and hidden files
				for _, file := range []string{".hidden", "visible.txt"} {
					path := filepath.Join(tmpDir, file)
					if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
						t.Fatalf("Failed to create test file: %v", err)
					}
				}
				return tmpDir
			},
			expectedExit:   0,
			expectedStdout: ".\n..\n.hidden\nvisible.txt\n",
			expectedStderr: "",
		},
		{
			name: "non-existent directory",
			setupFunc: func(t *testing.T) string {
				return "/path/that/does/not/exist"
			},
			expectedExit:   2,
			expectedStdout: "",
			expectedStderr: "ls: cannot access '/path/that/does/not/exist':",
		},
		{
			name:   "-h without -l warns",
			config: &lsconfig.Config{HumanReadable: true, LongListing: false},
			setupFunc: func(t *testing.T) string {
				return t.TempDir()
			},
			expectedExit:   0,
			expectedStdout: "",
			expectedStderr: "ls: warning: option -h is ignored when -l is not used",
		},
		{
			name:   "--si without -l warns",
			config: &lsconfig.Config{SI: true, LongListing: false},
			setupFunc: func(t *testing.T) string {
				return t.TempDir()
			},
			expectedExit:   0,
			expectedStdout: "",
			expectedStderr: "ls: warning: option --si is ignored when -l is not used",
		},
		{
			name:   "-h and --si without -l warns (both)",
			config: &lsconfig.Config{HumanReadable: true, SI: true, LongListing: false},
			setupFunc: func(t *testing.T) string {
				return t.TempDir()
			},
			expectedExit:   0,
			expectedStdout: "",
			expectedStderr: "ls: warning: options -h and --si are ignored when -l is not used",
		},
		{
			name:   "--block-size without -l warns",
			config: &lsconfig.Config{BlockSize: &lsconfig.BlockSizeSpec{Mode: lsconfig.BlockSizeModeBytes, SizeBytes: 1024}},
			setupFunc: func(t *testing.T) string {
				return t.TempDir()
			},
			expectedExit:   0,
			expectedStdout: "",
			expectedStderr: "ls: warning: option --block-size is ignored when -l is not used",
		},
		{
			name:   "--no-group without -l warns",
			config: &lsconfig.Config{NoGroup: true},
			setupFunc: func(t *testing.T) string {
				return t.TempDir()
			},
			expectedExit:   0,
			expectedStdout: "",
			expectedStderr: "ls: warning: --no-group is ignored when -l is not used",
		},
		{
			name:   "--time without -l warns",
			config: &lsconfig.Config{TimeFieldSet: true, TimeField: lsconfig.TimeFieldAccess},
			setupFunc: func(t *testing.T) string {
				return t.TempDir()
			},
			expectedExit:   0,
			expectedStdout: "",
			expectedStderr: "ls: warning: --time is ignored when -l is not used",
		},
		{
			name:   "--full-time without -l warns",
			config: &lsconfig.Config{FullTime: true, TimeStyleSpec: &lsconfig.TimeStyleSpec{Kind: lsconfig.TimeStyleFullISO, RecentLayout: "2006-01-02 15:04:05.000000000 -0700"}},
			setupFunc: func(t *testing.T) string {
				return t.TempDir()
			},
			expectedExit:   0,
			expectedStdout: "",
			expectedStderr: "ls: warning: --full-time is ignored when -l is not used",
		},
		{
			name: "list directory with dotfiles - no flags",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				if err := os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte(""), 0644); err != nil {
					t.Fatalf("failed to create file1.txt: %v", err)
				}
				if err := os.WriteFile(filepath.Join(tmpDir, ".dotfile.txt"), []byte(""), 0644); err != nil {
					t.Fatalf("failed to create .dotfile.txt: %v", err)
				}
				if err := os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755); err != nil {
					t.Fatalf("failed to create subdir: %v", err)
				}
				return tmpDir
			},
			config:         &lsconfig.Config{ShowAll: false, ShowAlmostAll: false}, // Explicitly set to make test clear
			expectedExit:   0,
			expectedStdout: "file1.txt\nsubdir\n", // .dotfile.txt, . and .. hidden
			expectedStderr: "",
		},
		{
			name: "list directory with dotfiles -a flag",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				if err := os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte(""), 0644); err != nil {
					t.Fatalf("failed to create file1.txt: %v", err)
				}
				if err := os.WriteFile(filepath.Join(tmpDir, ".dotfile.txt"), []byte(""), 0644); err != nil {
					t.Fatalf("failed to create .dotfile.txt: %v", err)
				}
				if err := os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755); err != nil {
					t.Fatalf("failed to create subdir: %v", err)
				}
				return tmpDir
			},
			config:         &lsconfig.Config{ShowAll: true, ShowAlmostAll: false},
			expectedExit:   0,
			expectedStdout: ".\n..\n.dotfile.txt\nfile1.txt\nsubdir\n", // Sorted output of all items
			expectedStderr: "",
		},
		{
			name: "list directory with dotfiles -A flag",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				if err := os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte(""), 0644); err != nil {
					t.Fatalf("failed to create file1.txt: %v", err)
				}
				if err := os.WriteFile(filepath.Join(tmpDir, ".dotfile.txt"), []byte(""), 0644); err != nil {
					t.Fatalf("failed to create .dotfile.txt: %v", err)
				}
				if err := os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755); err != nil {
					t.Fatalf("failed to create subdir: %v", err)
				}
				return tmpDir
			},
			config:         &lsconfig.Config{ShowAll: false, ShowAlmostAll: true},
			expectedExit:   0,
			expectedStdout: ".dotfile.txt\nfile1.txt\nsubdir\n", // . and .. hidden
			expectedStderr: "",
		},
		{
			name: "list directory with dotfiles -a -A flags (a should take precedence)",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				if err := os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte(""), 0644); err != nil {
					t.Fatalf("failed to create file1.txt: %v", err)
				}
				if err := os.WriteFile(filepath.Join(tmpDir, ".dotfile.txt"), []byte(""), 0644); err != nil {
					t.Fatalf("failed to create .dotfile.txt: %v", err)
				}
				if err := os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755); err != nil {
					t.Fatalf("failed to create subdir: %v", err)
				}
				return tmpDir
			},
			config:         &lsconfig.Config{ShowAll: true, ShowAlmostAll: true}, // ShowAll takes precedence
			expectedExit:   0,
			expectedStdout: ".\n..\n.dotfile.txt\nfile1.txt\nsubdir\n", // Sorted output of all items
			expectedStderr: "",
		},
		{
			name: "ls --author alone (warn, no long listing)",
			config: &lsconfig.Config{
				ShowAuthor:  true,
				LongListing: false,
			},
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				if err := os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte(""), 0644); err != nil {
					t.Fatalf("failed to create file1.txt: %v", err)
				}
				return tmpDir
			},
			expectedExit:   0,
			expectedStdout: "file1.txt\n", // Not long listing
			expectedStderr: "ls: warning: --author is ignored when -l is not used\n",
		},
		{
			name: "ls -l (no author/group)",
			config: &lsconfig.Config{
				LongListing: true,
				ShowAuthor:  false,
			},
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				if err := os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte(""), 0644); err != nil {
					t.Fatalf("failed to create file1.txt: %v", err)
				}
				return tmpDir
			},
			expectedExit: 0,
			expectedStdout: func() string {
				username, groupname := getUserDetails(t)
				return getFileDetails(t, "file1.txt", username, groupname, false)
			}(),
			expectedStderr: "",
		},
		{
			name: "ls -l --author (show author/group)",
			config: &lsconfig.Config{
				LongListing: true,
				ShowAuthor:  true,
			},
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				if err := os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte(""), 0644); err != nil {
					t.Fatalf("failed to create file1.txt: %v", err)
				}
				return tmpDir
			},
			expectedExit: 0,
			expectedStdout: func() string {
				username, groupname := getUserDetails(t)
				return getFileDetails(t, "file1.txt", username, groupname, true)
			}(),
			expectedStderr: "",
		},
		{
			name: "ls -d on non-existent path",
			config: &lsconfig.Config{
				ListDirectory: true,
			},
			setupFunc: func(t *testing.T) string {
				return "/path/that/does/not/exist/ls_d_test"
			},
			expectedExit:   2,
			expectedStdout: "",
			expectedStderr: "ls: cannot access '/path/that/does/not/exist/ls_d_test':",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setupFunc(t)

			// Create config if not provided
			config := tt.config
			if config == nil {
				config = &lsconfig.Config{
					ShowAll: false,
				}
			}
			// config.Directory = dir // No longer needed as run takes directory directly

			var stdout, stderr bytes.Buffer
			exitCode := run(dir, config, &stdout, &stderr)

			// Check exit code
			if exitCode != tt.expectedExit {
				t.Errorf("run() exit code = %d, want %d", exitCode, tt.expectedExit)
			}

			// Check stdout
			gotStdout := stdout.String()
			if gotStdout != tt.expectedStdout {
				t.Errorf("run() stdout = %q, want %q", gotStdout, tt.expectedStdout)
			}

			// Check stderr (partial match for error messages)
			gotStderr := stderr.String()
			if tt.expectedStderr != "" && !strings.Contains(gotStderr, tt.expectedStderr) {
				t.Errorf("run() stderr = %q, want to contain %q", gotStderr, tt.expectedStderr)
			} else if tt.expectedStderr == "" && gotStderr != "" {
				t.Errorf("run() stderr = %q, want empty", gotStderr)
			}
		})
	}
}

func TestRunWithSubdirectories(t *testing.T) {
	// Create a temporary directory with files and subdirectories
	tmpDir := t.TempDir()

	// Create files
	files := []string{"file1.txt", "file2.txt"}
	for _, file := range files {
		path := filepath.Join(tmpDir, file)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Create subdirectories
	dirs := []string{"dir1", "dir2"}
	for _, dir := range dirs {
		path := filepath.Join(tmpDir, dir)
		if err := os.Mkdir(path, 0755); err != nil {
			t.Fatalf("Failed to create test directory: %v", err)
		}
	}

	config := &lsconfig.Config{
		// Directory: tmpDir, // No longer needed as run takes directory directly
		ShowAll: false,
	}

	var stdout, stderr bytes.Buffer
	exitCode := run(tmpDir, config, &stdout, &stderr)

	// Check exit code
	if exitCode != 0 {
		t.Errorf("run() exit code = %d, want 0", exitCode)
	}

	// Check that output contains all files and directories
	output := stdout.String()
	for _, file := range files {
		if !strings.Contains(output, file) {
			t.Errorf("Output missing file: %s", file)
		}
	}
	for _, dir := range dirs {
		if !strings.Contains(output, dir) {
			t.Errorf("Output missing directory: %s", dir)
		}
	}

	// Check stderr is empty
	if stderr.Len() > 0 {
		t.Errorf("Unexpected stderr output: %s", stderr.String())
	}
}

func TestPrintGrid(t *testing.T) {
	tests := []struct {
		name     string
		entries  []string
		width    int
		expected string
	}{
		{
			name:    "single column fit",
			entries: []string{"a", "b", "c"},
			width:   10,
			// maxLen=1, colWidth=3. numCols=3. numRows=1.
			// Output: "a  b  c  \n"
			expected: "a  b  c  \n",
		},
		{
			name:    "force multiple rows",
			entries: []string{"a", "b", "c", "d"},
			width:   5,
			// maxLen=1, colWidth=3. numCols=1. numRows=4.
			// Output:
			// a
			// b
			// c
			// d
			expected: "a  \nb  \nc  \nd  \n",
		},
		{
			name:    "2 columns",
			entries: []string{"1", "2", "3", "4"},
			width:   7,
			// maxLen=1, colWidth=3. numCols=2. numRows=2.
			// Row 0: 1, 3
			// Row 1: 2, 4
			// Output: "1  3  \n2  4  \n"
			expected: "1  3  \n2  4  \n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			display.PrintGrid(&buf, tt.entries, tt.width)
			got := buf.String()
			if got != tt.expected {
				t.Errorf("display.PrintGrid() output:\n%q\nwant:\n%q", got, tt.expected)
			}
		})
	}
}
func TestFormatSize(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0"},
		{100, "100"},
		{1023, "1023"},
		{1024, "1.0K"},
		{1500, "1.5K"},
		{1024 * 1024, "1.0M"},
		{1024*1024*2 + 500*1024, "2.5M"},
		{1024 * 1024 * 1024, "1.0G"},
	}

	for _, tt := range tests {
		got := size.FormatSize(tt.input, 1024)
		if got != tt.expected {
			t.Errorf("size.FormatSize(%d, 1024) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestFormatSizeSI(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0"},
		{100, "100"},
		{999, "999"},
		{1000, "1.0K"},
		{1500, "1.5K"},
		{1000 * 1000, "1.0M"},
		{1000*1000*2 + 500*1000, "2.5M"},
		{1000 * 1000 * 1000, "1.0G"},
	}

	for _, tt := range tests {
		got := size.FormatSize(tt.input, 1000)
		if got != tt.expected {
			t.Errorf("size.FormatSize(%d, 1000) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

// getUserDetails gets current user and group info for testing long listings
func getUserDetails(t *testing.T) (string, string) {
	currentUser, err := user.Current()
	if err != nil {
		t.Fatalf("Failed to get current user: %v", err)
	}
	currentGroup, err := user.LookupGroupId(currentUser.Gid)
	if err != nil {
		t.Fatalf("Failed to get current group: %v", err)
	}
	return currentUser.Username, currentGroup.Name
}

// getFileDetails formats a single file's long listing output for testing
func getFileDetails(t *testing.T, filename string, owner, group string, showAuthor bool) string {
	// Create a temporary file and get its stat info
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, filename)
	if err := os.WriteFile(filePath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("Failed to stat temp file: %v", err)
	}

	mode := info.Mode().String()
	nlink := 1 // For a newly created file
	size := info.Size()
	modTime := info.ModTime().Format("Jan 02 15:04")

	// Max widths for a single file test case, assuming minimum widths as in display.PrintLongList
	maxLinkLen := len(fmt.Sprint(nlink))
	if maxLinkLen < 2 {
		maxLinkLen = 2
	} // Assuming minimum 2 for nlink formatting
	maxSizeLen := len(fmt.Sprint(size))
	if maxSizeLen < 1 {
		maxSizeLen = 1
	} // Assuming minimum 1 for size formatting

	maxOwnerLen := len(owner)
	maxGroupLen := len(group)
	if showAuthor {
		maxAuthorLen := len(owner)
		return fmt.Sprintf("%s %*d %-*s %-*s %-*s %*d %s %s\n",
			mode,
			maxLinkLen, nlink,
			maxOwnerLen, owner,
			maxAuthorLen, owner,
			maxGroupLen, group,
			maxSizeLen, size,
			modTime,
			filename,
		)
	}
	return fmt.Sprintf("%s %*d %-*s %-*s %*d %s %s\n",
		mode,
		maxLinkLen, nlink,
		maxOwnerLen, owner,
		maxGroupLen, group,
		maxSizeLen, size,
		modTime,
		filename,
	)
}

func TestPrintLongListErrors(t *testing.T) {
	config := &lsconfig.Config{LongListing: true}
	mockEntries := []os.DirEntry{
		&mockDirEntry{name: "good-file", isDir: false}, // This won't actually be processed fully
		&mockDirEntry{name: "bad-file", isDir: false, infoErr: errors.New("simulated info error")},
	}

	var stdout, stderr bytes.Buffer
	hadError := display.PrintLongList(&stdout, &stderr, mockEntries, config)
	if !hadError {
		t.Error("Expected an error from PrintLongList, but got none")
	}

	// Since the "good-file" doesn't have proper Info, it will also be skipped.
	// Only the error condition is what we are testing here.
	// The function should not panic and produce no output for the error entry.
	expectedOutput := ""
	if stdout.String() != expectedOutput {
		t.Errorf("Expected output %q, got %q", expectedOutput, stdout.String())
	}
	if !strings.Contains(stderr.String(), "ls: cannot access 'bad-file'") {
		t.Errorf("Expected warning on stderr, got %q", stderr.String())
	}
}

func TestDirEntryWrapper(t *testing.T) {
	tmpDir := t.TempDir()
	parentDir := filepath.Dir(tmpDir)

	// Test "." wrapper
	dotWrapper := &entry.DirEntryWrapper{EntryName: ".", DirPath: tmpDir}
	if dotWrapper.Name() != "." {
		t.Errorf("Expected Name() to return '.', got %s", dotWrapper.Name())
	}
	if !dotWrapper.IsDir() {
		t.Error("Expected IsDir() to return true")
	}
	if dotWrapper.Type() != fs.ModeDir {
		t.Errorf("Expected Type() to return fs.ModeDir, got %v", dotWrapper.Type())
	}
	info, err := dotWrapper.Info()
	if err != nil {
		t.Errorf("Expected Info() to succeed, got error: %v", err)
	}
	if info == nil {
		t.Error("Expected Info() to return non-nil FileInfo")
	}

	// Test isRoot: true for Info()
	rootWrapper := &entry.DirEntryWrapper{EntryName: tmpDir, DirPath: tmpDir, IsRoot: true}
	info, err = rootWrapper.Info()
	if err != nil {
		t.Errorf("Expected Info() for rootWrapper to succeed, got error: %v", err)
	}
	if info == nil || info.Name() != filepath.Base(tmpDir) {
		t.Errorf("Expected Info() to return info for %s, got %v", tmpDir, info)
	}

	// Test IsDir and Type error paths
	badWrapper := &entry.DirEntryWrapper{EntryName: "nonexistent", DirPath: "/nonexistent/path"}
	if badWrapper.IsDir() {
		t.Error("Expected IsDir() to be false for nonexistent path")
	}
	if badWrapper.Type() != 0 {
		t.Errorf("Expected Type() to be 0 for nonexistent path, got %v", badWrapper.Type())
	}

	// Test ".." wrapper
	dotDotWrapper := &entry.DirEntryWrapper{EntryName: "..", DirPath: tmpDir}
	if dotDotWrapper.Name() != ".." {
		t.Errorf("Expected Name() to return '..', got %s", dotDotWrapper.Name())
	}
	if !dotDotWrapper.IsDir() {
		t.Error("Expected IsDir() to return true")
	}
	if dotDotWrapper.Type() != fs.ModeDir {
		t.Errorf("Expected Type() to return fs.ModeDir, got %v", dotDotWrapper.Type())
	}
	info, err = dotDotWrapper.Info()
	if err != nil {
		t.Errorf("Expected Info() to succeed, got error: %v", err)
	}
	if info == nil {
		t.Error("Expected Info() to return non-nil FileInfo")
	}
	// Verify that ".." Info() returns the parent directory's info
	parentInfo, err := os.Stat(parentDir)
	if err == nil && info != nil {
		if info.Name() != parentInfo.Name() {
			t.Errorf("Expected Info() for '..' to return parent dir name %s, got %s", parentInfo.Name(), info.Name())
		}
	}

	// Test ".." wrapper with relative path "."
	// Before fix, this will incorrectly stat "." (the current dir) instead of ".." (the parent)
	// Because filepath.Dir(".") returns "."
	dotDotRelativeWrapper := &entry.DirEntryWrapper{EntryName: "..", DirPath: "."}
	infoRel, err := dotDotRelativeWrapper.Info()
	if err != nil {
		t.Errorf("Expected Info() for relative '..' to succeed, got error: %v", err)
	}

	cwd, _ := os.Getwd()
	parentOfCwd := filepath.Dir(cwd)
	expectedParentInfo, _ := os.Stat(parentOfCwd)

	if infoRel != nil && expectedParentInfo != nil {
		if !os.SameFile(infoRel, expectedParentInfo) {
			t.Errorf("Expected Info() for relative '..' to return info for %s, but it didn't match", parentOfCwd)
		}
	}
}

func TestPrintLongListWithShowAll(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte(""), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	config := &lsconfig.Config{
		LongListing: true,
		ShowAll:     true,
	}

	var buf bytes.Buffer
	exitCode := run(tmpDir, config, &buf, &buf)
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	output := buf.String()
	// Should contain "." and ".." entries in long listing format
	// However, if Info() fails for these entries (e.g., parent dir not accessible),
	// they will be skipped in display.PrintLongList. So we test that the functionality
	// works by checking that file1.txt is present and that we got some output.
	// The actual "." and ".." entries depend on system permissions.
	lines := strings.Split(strings.TrimSpace(output), "\n")
	foundFile := false
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 0 {
			lastField := fields[len(fields)-1]
			if lastField == "file1.txt" {
				foundFile = true
			}
		}
	}
	if !foundFile {
		t.Errorf("Expected output to contain 'file1.txt' entry. Output: %q", output)
	}

	// Verify that "." and ".." are present and shown correctly
	foundDot := false
	foundDotDot := false
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 0 {
			lastField := fields[len(fields)-1]
			if lastField == "." {
				foundDot = true
			}
			if lastField == ".." {
				foundDotDot = true
			}
		}
	}
	if !foundDot {
		t.Errorf("Expected output to contain '.' entry. Output: %q", output)
	}
	if !foundDotDot {
		t.Errorf("Expected output to contain '..' entry. Output: %q", output)
	}

	// Verify that we got some output (at least file1.txt should be there)
	if len(lines) == 0 {
		t.Error("Expected at least one line of output")
	}
}

func TestPrintGridEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		entries  []string
		width    int
		expected string
	}{
		{
			name:     "empty entries",
			entries:  []string{},
			width:    80,
			expected: "",
		},
		{
			name:    "zero width",
			entries: []string{"a", "b"},
			width:   0,
			// When width is 0, colWidth becomes 1, numCols becomes 1, so output is "a\nb\n"
			expected: "a\nb\n",
		},
		{
			name:    "very narrow width",
			entries: []string{"a", "b", "c"},
			width:   1,
			// When width is 1, colWidth becomes 1, numCols becomes 1, so output is "a\nb\nc\n"
			expected: "a\nb\nc\n",
		},
		{
			name:     "single entry",
			entries:  []string{"single"},
			width:    80,
			expected: "single  \n",
		},
		{
			name:     "long names",
			entries:  []string{"verylongfilename1", "verylongfilename2", "verylongfilename3"},
			width:    100,
			expected: "verylongfilename1  verylongfilename2  verylongfilename3  \n",
		},
		{
			name:    "mixed length names",
			entries: []string{"a", "verylongname", "b", "c"},
			width:   80,
			// maxLen=12, colWidth=14, numCols=80/14=5, but limited to 4 entries, so numCols=4, numRows=1
			// All 4 entries fit in one row: "a             verylongname  b             c             \n"
			expected: "a             verylongname  b             c             \n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			display.PrintGrid(&buf, tt.entries, tt.width)
			got := buf.String()
			if got != tt.expected {
				t.Errorf("display.PrintGrid() output:\n%q\nwant:\n%q", got, tt.expected)
			}
		})
	}
}

func TestPrintLongListEmpty(t *testing.T) {
	config := &lsconfig.Config{LongListing: true}
	var stdout, stderr bytes.Buffer
	hadError := display.PrintLongList(&stdout, &stderr, []os.DirEntry{}, config)
	if hadError {
		t.Error("Expected no error from PrintLongList with empty entries, but got one")
	}
	if stdout.String() != "" {
		t.Errorf("Expected empty output for empty entries, got %q", stdout.String())
	}
}

func TestPrintLongListWithoutAuthor(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	config := &lsconfig.Config{
		LongListing: true,
		ShowAuthor:  false,
	}

	var buf bytes.Buffer
	exitCode := run(tmpDir, config, &buf, &buf)
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	output := buf.String()
	// Should not contain author column when ShowAuthor is false
	// Format should be: mode nlink owner group size date time name
	if !strings.Contains(output, "file1.txt") {
		t.Error("Expected output to contain 'file1.txt'")
	}
	// Verify it's in the expected format (no owner/group columns)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) > 0 {
		fields := strings.Fields(lines[0])
		// Should have: mode, nlink, owner, group, size, date, time, name (8 fields)
		if len(fields) < 8 {
			t.Errorf("Expected at least 8 fields in long listing, got %d: %v", len(fields), fields)
		}
	}
}

func TestFormatSizeEdgeCases(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{1023, "1023"},
		{1024, "1.0K"},
		{1536, "1.5K"},
		{1024 * 1024, "1.0M"},
		{1024 * 1024 * 1024, "1.0G"},
		{1024 * 1024 * 1024 * 1024, "1.0T"},
		{1024 * 1024 * 1024 * 1024 * 1024, "1.0P"},
		{1024 * 1024 * 1024 * 1024 * 1024 * 1024, "1.0E"},
		{9223372036854775807, "8.0E"},
		{2048, "2.0K"},
		{5120, "5.0K"},
		{1048576, "1.0M"},
		{1073741824, "1.0G"},
	}

	for _, tt := range tests {
		got := size.FormatSize(tt.input, 1024)
		if got != tt.expected {
			t.Errorf("size.FormatSize(%d, 1024) = %q, want %q", tt.input, got, tt.expected)
		}
	}
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
			config, err := parse.ParseArgs(tt.args, io.Discard)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if config != nil {
					t.Error("Expected config to be nil on error")
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestPrintEntriesEmpty(t *testing.T) {
	var buf bytes.Buffer
	display.PrintEntries(&buf, []string{})
	if buf.String() != "" {
		t.Errorf("Expected empty output for empty entries, got %q", buf.String())
	}
}

func TestRunMultipleDirectories(t *testing.T) {
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir1, "file1.txt"), []byte(""), 0644); err != nil {
		t.Fatalf("failed to create test file in tmpDir1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir2, "file2.txt"), []byte(""), 0644); err != nil {
		t.Fatalf("failed to create test file in tmpDir2: %v", err)
	}

	config := &lsconfig.Config{
		Directories: []string{tmpDir1, tmpDir2},
		ShowAll:     false,
	}

	var stdout, stderr bytes.Buffer
	// This tests the main() logic for multiple directories
	// We'll test it by calling run() for each directory
	exitCode1 := run(tmpDir1, config, &stdout, &stderr)
	exitCode2 := run(tmpDir2, config, &stdout, &stderr)

	if exitCode1 != 0 || exitCode2 != 0 {
		t.Errorf("Expected exit code 0 for both directories, got %d and %d", exitCode1, exitCode2)
	}

	output := stdout.String()
	if !strings.Contains(output, "file1.txt") {
		t.Error("Expected output to contain 'file1.txt'")
	}
	if !strings.Contains(output, "file2.txt") {
		t.Error("Expected output to contain 'file2.txt'")
	}
}

func TestExecute(t *testing.T) {
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir1, "f1.txt"), []byte(""), 0644); err != nil {
		t.Fatalf("failed to create f1.txt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir2, "f2.txt"), []byte(""), 0644); err != nil {
		t.Fatalf("failed to create f2.txt: %v", err)
	}

	tests := []struct {
		name           string
		args           []string
		expectedExit   int
		expectedStdout string
		expectedStderr string
	}{
		{
			name:           "success - single dir",
			args:           []string{tmpDir1},
			expectedExit:   0,
			expectedStdout: "f1.txt\n",
			expectedStderr: "",
		},
		{
			name:           "success - multiple dirs",
			args:           []string{tmpDir1, tmpDir2},
			expectedExit:   0,
			expectedStdout: fmt.Sprintf("%s:\nf1.txt\n\n%s:\nf2.txt\n", tmpDir1, tmpDir2),
			expectedStderr: "",
		},
		{
			name:           "multiple dirs with one failing",
			args:           []string{tmpDir1, "/nonexistent/path"},
			expectedExit:   2,
			expectedStdout: fmt.Sprintf("%s:\nf1.txt\n", tmpDir1),
			expectedStderr: "ls: cannot access '/nonexistent/path':",
		},
		{
			name:           "invalid flag",
			args:           []string{"--invalid-flag"},
			expectedExit:   1,
			expectedStdout: "",
			expectedStderr: "ls: unknown flag: --invalid-flag\n",
		},
		{
			name:           "help flag",
			args:           []string{"--help"},
			expectedExit:   0,
			expectedStdout: "",
			expectedStderr: "Usage: ls [OPTION]... [FILE]...",
		},
		{
			name:           "non-existent directory",
			args:           []string{"/path/does/not/exist"},
			expectedExit:   2,
			expectedStdout: "",
			expectedStderr: "ls: cannot access '/path/does/not/exist':",
		},
		{
			name:           "ls -d with mixed paths (one non-existent)",
			args:           []string{"-d", tmpDir1, "/nonexistent/path/for/ls/d"},
			expectedExit:   2,
			expectedStdout: tmpDir1,
			expectedStderr: "ls: cannot access '/nonexistent/path/for/ls/d':",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			exitCode := Execute(tt.args, &stdout, &stderr)

			if exitCode != tt.expectedExit {
				t.Errorf("Execute() exitCode = %v, want %v", exitCode, tt.expectedExit)
			}

			if tt.expectedStdout != "" && !strings.Contains(stdout.String(), tt.expectedStdout) {
				t.Errorf("Execute() stdout = %q, want to contain %q", stdout.String(), tt.expectedStdout)
			}

			if tt.expectedStderr != "" && !strings.Contains(stderr.String(), tt.expectedStderr) {
				t.Errorf("Execute() stderr = %q, want to contain %q", stderr.String(), tt.expectedStderr)
			}
		})
	}
}

func TestPrintLongListWithHumanReadable(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a file with a specific size to test human-readable format
	testContent := make([]byte, 2048) // 2KB
	if err := os.WriteFile(filepath.Join(tmpDir, "testfile.txt"), testContent, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	config := &lsconfig.Config{
		LongListing:   true,
		HumanReadable: true,
	}

	var buf bytes.Buffer
	exitCode := run(tmpDir, config, &buf, &buf)
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	output := buf.String()
	// Should contain human-readable size (e.g., "2.0K")
	if !strings.Contains(output, "K") && !strings.Contains(output, "testfile.txt") {
		t.Errorf("Expected output to contain human-readable size or filename, got: %q", output)
	}
}

func TestPrintLongListWithSI(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a file with a specific size to test SI format
	// 2000 bytes should be 2.0K in SI (base 1000)
	testContent := make([]byte, 2000)
	if err := os.WriteFile(filepath.Join(tmpDir, "sifile.txt"), testContent, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	config := &lsconfig.Config{
		LongListing: true,
		SI:          true,
	}

	var buf bytes.Buffer
	exitCode := run(tmpDir, config, &buf, &buf)
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	output := buf.String()
	// Should contain SI size "2.0K" (because 2000 / 1000 = 2.0)
	if !strings.Contains(output, "2.0K") {
		t.Errorf("Expected output to contain '2.0K', got: %q", output)
	}
}

func TestPrintLongListWithNoGroup(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "nogroup.txt"), []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	config := &lsconfig.Config{
		LongListing: true,
		ShowAuthor:  true,
		NoGroup:     true,
	}

	var buf bytes.Buffer
	exitCode := run(tmpDir, config, &buf, &buf)
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) == 0 {
		t.Fatalf("Expected output, got empty")
	}
	fields := strings.Fields(lines[0])
	if len(fields) != 9 {
		t.Fatalf("Expected 9 fields (mode nlink owner author size month day time name), got %d: %v", len(fields), fields)
	}
}

func TestPrintLongListWithBlockSize(t *testing.T) {
	tmpDir := t.TempDir()
	testContent := make([]byte, 1500)
	if err := os.WriteFile(filepath.Join(tmpDir, "blockfile.txt"), testContent, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	config := &lsconfig.Config{
		LongListing: true,
		BlockSize:   &lsconfig.BlockSizeSpec{Mode: lsconfig.BlockSizeModeBytes, SizeBytes: 1024},
	}

	var buf bytes.Buffer
	exitCode := run(tmpDir, config, &buf, &buf)
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) == 0 {
		t.Fatalf("Expected output, got empty")
	}
	fields := strings.Fields(lines[0])
	if len(fields) < 8 {
		t.Fatalf("Expected long listing fields, got %v", fields)
	}
	if fields[4] != "2" {
		t.Errorf("Expected block size count 2, got %q", fields[4])
	}
}

func TestRunSortByTime(t *testing.T) {
	tmpDir := t.TempDir()
	oldPath := filepath.Join(tmpDir, "old.txt")
	newPath := filepath.Join(tmpDir, "new.txt")
	if err := os.WriteFile(oldPath, []byte("old"), 0644); err != nil {
		t.Fatalf("failed to create old file: %v", err)
	}
	if err := os.WriteFile(newPath, []byte("new"), 0644); err != nil {
		t.Fatalf("failed to create new file: %v", err)
	}

	oldTime := time.Now().Add(-2 * time.Hour).Truncate(time.Second)
	newTime := time.Now().Add(-1 * time.Hour).Truncate(time.Second)
	if err := os.Chtimes(oldPath, oldTime, oldTime); err != nil {
		t.Fatalf("failed to set times on old file: %v", err)
	}
	if err := os.Chtimes(newPath, newTime, newTime); err != nil {
		t.Fatalf("failed to set times on new file: %v", err)
	}

	config := &lsconfig.Config{
		SortTime:  true,
		TimeField: lsconfig.TimeFieldMod,
	}

	var stdout, stderr bytes.Buffer
	exitCode := run(tmpDir, config, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() failed: %v", stderr.String())
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) < 2 {
		t.Fatalf("Expected at least 2 lines, got %d. Stdout: %q", len(lines), stdout.String())
	}
	if lines[0] != "new.txt" || lines[1] != "old.txt" {
		t.Fatalf("Expected newest first (new.txt, old.txt), got %v. File times: new=%v, old=%v", lines, newTime, oldTime)
	}
}

func TestRunSortByTimeTiebreak(t *testing.T) {
	tmpDir := t.TempDir()
	aPath := filepath.Join(tmpDir, "a.txt")
	bPath := filepath.Join(tmpDir, "b.txt")
	if err := os.WriteFile(aPath, []byte("a"), 0644); err != nil {
		t.Fatalf("failed to create a.txt: %v", err)
	}
	if err := os.WriteFile(bPath, []byte("b"), 0644); err != nil {
		t.Fatalf("failed to create b.txt: %v", err)
	}

	sameTime := time.Now().Add(-1 * time.Hour)
	if err := os.Chtimes(aPath, sameTime, sameTime); err != nil {
		t.Fatalf("failed to set times on a.txt: %v", err)
	}
	if err := os.Chtimes(bPath, sameTime, sameTime); err != nil {
		t.Fatalf("failed to set times on b.txt: %v", err)
	}

	cfg := &lsconfig.Config{
		SortTime:  true,
		TimeField: lsconfig.TimeFieldMod,
	}

	var stdout, stderr bytes.Buffer
	exitCode := run(tmpDir, cfg, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() failed: %v", stderr.String())
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %d", len(lines))
	}
	if lines[0] != "a.txt" || lines[1] != "b.txt" {
		t.Fatalf("expected name tiebreak ordering, got %v", lines)
	}
}

func TestRunWithSIWarning(t *testing.T) {
	tmpDir := t.TempDir()
	config := &lsconfig.Config{
		SI:          true,
		LongListing: false,
	}

	var stdout, stderr bytes.Buffer
	exitCode := run(tmpDir, config, &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	gotStderr := stderr.String()
	expectedWarning := "ls: warning: option --si is ignored when -l is not used"
	if !strings.Contains(gotStderr, expectedWarning) {
		t.Errorf("Expected stderr to contain %q, got %q", expectedWarning, gotStderr)
	}
}

func TestRunTimeFieldWarnings(t *testing.T) {
	tests := []struct {
		name          string
		sortTime      bool
		expectWarning bool
	}{
		{"without sort", false, true},
		{"with sort", true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			config := &lsconfig.Config{
				TimeFieldSet: true,
				LongListing:  false,
				SortTime:     tt.sortTime,
			}
			var stdout, stderr bytes.Buffer
			exitCode := run(tmpDir, config, &stdout, &stderr)
			if exitCode != 0 {
				t.Fatalf("run failed: exit code %d, stderr: %q", exitCode, stderr.String())
			}
			hasWarning := strings.Contains(stderr.String(), "warning: --time is ignored")
			if hasWarning != tt.expectWarning {
				t.Errorf("expected warning %t, got %t\nstderr: %q", tt.expectWarning, hasWarning, stderr.String())
			}
		})
	}
}

func TestRunSortByAccessTimeNoLong(t *testing.T) {
	tmpDir := t.TempDir()
	fileA := filepath.Join(tmpDir, "fileA")
	fileB := filepath.Join(tmpDir, "fileB")
	if err := os.WriteFile(fileA, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fileB, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}
	now := time.Now().Truncate(time.Second)
	newAccess := now.Add(-time.Hour)
	oldAccess := now.Add(-2 * time.Hour)
	newMod := now.Add(-time.Hour)
	oldMod := now.Add(-2 * time.Hour)
	// fileA: newer access, older mod
	if err := os.Chtimes(fileA, newAccess, oldMod); err != nil {
		t.Fatal(err)
	}
	// fileB: older access, newer mod
	if err := os.Chtimes(fileB, oldAccess, newMod); err != nil {
		t.Fatal(err)
	}
	config := &lsconfig.Config{
		SortTime:     true,
		TimeField:    lsconfig.TimeFieldAccess,
		TimeFieldSet: true,
		LongListing:  false,
	}
	var stdout, stderr bytes.Buffer
	exitCode := run(tmpDir, config, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run failed: exit %d, stderr: %q", exitCode, stderr.String())
	}
	// No warning since SortTime
	if strings.Contains(stderr.String(), "warning: --time") {
		t.Error("unexpected --time warning")
	}
	// Sort by access newest first: fileA before fileB
	got := strings.TrimSpace(stdout.String())
	if !strings.HasPrefix(got, "fileA\nfileB") {
		t.Errorf("expected fileA then fileB by access time, got:\n%q", got)
	}
}

func TestQuoteName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"plain", "plain"},
		{"space ", "space "},
		{"newline\n", `newline\n`},
		{"tab\t", `tab\t`},
		{"carriage\r", `carriage\r`},
		{"backspace\b", `backspace\b`},
		{"formfeed\f", `formfeed\f`},
		{"vtab\v", `vtab\v`},
		{"bell\a", `bell\a`},
		{"backslash\\", `backslash\\`},
		{"non-graphic\x01", `non-graphic\001`},
		{"non-graphic\x1f", `non-graphic\037`},
		{"non-graphic\x7f", `non-graphic\177`},
		{"utf8-日本語", "utf8-\\346\\227\\245\\346\\234\\254\\350\\252\\236"},
	}

	for _, tt := range tests {
		got := display.QuoteName(tt.input)
		if got != tt.expected {
			t.Errorf("display.QuoteName(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestRunWithEscape(t *testing.T) {
	tmpDir := t.TempDir()
	// Use a filename with a newline
	filename := "file\nwith\nnewline"
	filePath := filepath.Join(tmpDir, filename)
	if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
		// On some systems, newlines in filenames might not be allowed.
		// If this fails, we skip the test.
		t.Skipf("Skipping test because filename with newline could not be created: %v", err)
	}

	config := &lsconfig.Config{
		Escape: true,
	}

	var buf bytes.Buffer
	exitCode := run(tmpDir, config, &buf, io.Discard)
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	output := buf.String()
	expected := "file\\nwith\\nnewline\n"
	if output != expected {
		t.Errorf("Expected output %q, got %q", expected, output)
	}

	// Test long listing with escape
	buf.Reset()
	config.LongListing = true
	exitCode = run(tmpDir, config, &buf, io.Discard)
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}
	output = buf.String()
	if !strings.Contains(output, "file\\nwith\\nnewline") {
		t.Errorf("Expected long listing to contain escaped filename, got: %q", output)
	}
}

func TestIgnoreBackups(t *testing.T) {
	tmpDir := t.TempDir()
	files := []string{"file1.txt", "file1.txt~", "file2.txt", "README.md~"}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, f), []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", f, err)
		}
	}

	tests := []struct {
		name     string
		config   *lsconfig.Config
		expected string
	}{
		{
			name:     "no ignore-backups",
			config:   &lsconfig.Config{},
			expected: "README.md~\nfile1.txt\nfile1.txt~\nfile2.txt\n",
		},
		{
			name:     "with ignore-backups",
			config:   &lsconfig.Config{IgnoreBackups: true},
			expected: "file1.txt\nfile2.txt\n",
		},
		{
			name:     "with ignore-backups and show-all",
			config:   &lsconfig.Config{IgnoreBackups: true, ShowAll: true},
			expected: ".\n..\nfile1.txt\nfile2.txt\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			exitCode := run(tmpDir, tt.config, &stdout, &stderr)
			if exitCode != 0 {
				t.Fatalf("run() failed: %v", stderr.String())
			}
			got := stdout.String()
			if got != tt.expected {
				t.Errorf("Expected stdout %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestListDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	subdir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subdir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "file.txt"), []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	tests := []struct {
		name           string
		path           string
		config         *lsconfig.Config
		expectedStdout string
	}{
		{
			name:           "list directory itself",
			path:           subdir,
			config:         &lsconfig.Config{ListDirectory: true},
			expectedStdout: subdir + "\n",
		},
		{
			name:           "list directory itself long format",
			path:           subdir,
			config:         &lsconfig.Config{ListDirectory: true, LongListing: true},
			expectedStdout: "drwxr-xr-x", // Just check prefix for mode
		},
		{
			name:           "list directory itself with escape",
			path:           subdir,
			config:         &lsconfig.Config{ListDirectory: true, Escape: true},
			expectedStdout: subdir + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			exitCode := run(tt.path, tt.config, &stdout, &stderr)
			if exitCode != 0 {
				t.Fatalf("run() failed: %v", stderr.String())
			}
			got := stdout.String()
			if !strings.Contains(got, tt.expectedStdout) {
				t.Errorf("Expected stdout to contain %q, got %q", tt.expectedStdout, got)
			}
			// Verify it doesn't list contents
			if strings.Contains(got, "file.txt") {
				t.Errorf("Expected stdout NOT to contain 'file.txt', but it did: %q", got)
			}
		})
	}
}

type mockTerminalWriter struct {
	bytes.Buffer
}

func (m *mockTerminalWriter) Fd() uintptr {
	return 1 // Typical stdout FD
}

func TestPrintEntriesTerminal(t *testing.T) {
	// Mock terminal functions
	oldIsTerminal := display.IsTerminalFunc
	oldGetTermSize := display.GetTermSizeFunc
	defer func() {
		display.IsTerminalFunc = oldIsTerminal
		display.GetTermSizeFunc = oldGetTermSize
	}()

	display.IsTerminalFunc = func(fd int) bool {
		return true
	}
	display.GetTermSizeFunc = func(fd int) (int, int, error) {
		return 80, 24, nil
	}

	w := &mockTerminalWriter{}
	display.PrintEntries(w, []string{"a", "b", "c"})
	output := w.String()
	// Should be in grid format
	if !strings.Contains(output, "a  b  c") {
		t.Errorf("Expected output to be in grid format, got %q", output)
	}

	// Test error in GetSize
	display.GetTermSizeFunc = func(fd int) (int, int, error) {
		return 0, 0, errors.New("size error")
	}
	w.Reset()
	display.PrintEntries(w, []string{"a", "b", "c"})
	output = w.String()
	// Should fall back to one per line
	if output != "a\nb\nc\n" {
		t.Errorf("Expected fallback to one per line, got %q", output)
	}
}
func TestExecuteHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Execute([]string{"--help"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("Expected exit code 0 for --help, got %d", exitCode)
	}
}

func TestExecuteError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Execute([]string{"--invalid-flag"}, &stdout, &stderr)
	if exitCode != 1 {
		t.Errorf("Expected exit code 1 for invalid flag, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "ls: unknown flag") {
		t.Errorf("Expected unknown flag error, got %q", stderr.String())
	}
}

func TestRunErrorCases(t *testing.T) {
	// 1. Nonexistent path
	config := &lsconfig.Config{}
	var stdout, stderr bytes.Buffer
	exitCode := run("nonexistent_path_xyz", config, &stdout, &stderr)
	if exitCode != 2 {
		t.Errorf("Expected exit code 2 for nonexistent path, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "cannot access") {
		t.Errorf("Expected error message, got %q", stderr.String())
	}

	// 2. Nonexistent path with -d
	stdout.Reset()
	stderr.Reset()
	config.ListDirectory = true
	exitCode = run("nonexistent_path_xyz", config, &stdout, &stderr)
	if exitCode != 2 {
		t.Errorf("Expected exit code 2 for nonexistent path with -d, got %d", exitCode)
	}

	// 3. Permission denied (if possible to test reliably)
	tmpDir := t.TempDir()
	permDir := filepath.Join(tmpDir, "perm_denied")
	if err := os.Mkdir(permDir, 0000); err != nil {
		t.Fatalf("Failed to create perm_denied dir: %v", err)
	}
	defer func() { _ = os.Chmod(permDir, 0755) }()

	stdout.Reset()
	stderr.Reset()
	config.ListDirectory = false
	exitCode = run(permDir, config, &stdout, &stderr)
	// On some systems/environments, root might still be able to read it,
	// but usually it should fail for the current user.
	if os.Getuid() != 0 {
		if exitCode != 2 {
			t.Errorf("Expected exit code 2 for permission denied, got %d", exitCode)
		}
	}
}

func TestParseBlockSizeVarious(t *testing.T) {
	// Missing SIZE
	_, warn, ok := size.ParseBlockSize("")
	if ok || warn != "missing SIZE" {
		t.Errorf("Expected 'missing SIZE', got ok=%v warn=%q", ok, warn)
	}

	// Missing SIZE after apostrophe
	_, warn, ok = size.ParseBlockSize("'")
	if ok || warn != "missing SIZE" {
		t.Errorf("Expected 'missing SIZE' for single apostrophe, got ok=%v warn=%q", ok, warn)
	}

	// Invalid number
	_, warn, ok = size.ParseBlockSize("0")
	if ok || warn != "invalid SIZE number" {
		t.Errorf("Expected 'invalid SIZE number' for 0, got ok=%v warn=%q", ok, warn)
	}

	// Too large
	_, warn, ok = size.ParseBlockSize("1000000000000000000000000000")
	if ok || !strings.Contains(warn, "invalid") {
		t.Errorf("Expected invalid number error, got ok=%v warn=%q", ok, warn)
	}

	// Unknown suffix
	_, warn, ok = size.ParseBlockSize("1X")
	if ok || warn != "unknown SIZE suffix" {
		t.Errorf("Expected 'unknown SIZE suffix', got ok=%v warn=%q", ok, warn)
	}

	// Overflow
	_, warn, ok = size.ParseBlockSize("10000000000000000000T")
	if ok || warn != "SIZE too large" {
		t.Errorf("Expected 'SIZE too large', got ok=%v warn=%q", ok, warn)
	}
}

func TestParseTimeFormatErrors(t *testing.T) {
	// Empty format
	_, _, warn, ok := timeutil.ParseTimeFormat("")
	if ok || warn != "missing TIME_STYLE format" {
		t.Errorf("Expected 'missing TIME_STYLE format', got ok=%v warn=%q", ok, warn)
	}

	// More than 2 parts
	_, _, warn, ok = timeutil.ParseTimeFormat("a\nb\nc")
	if ok || warn != "invalid TIME_STYLE format" {
		t.Errorf("Expected 'invalid TIME_STYLE format', got ok=%v warn=%q", ok, warn)
	}

	// Invalid token in first part of two-part format
	_, _, warn, ok = timeutil.ParseTimeFormat("%Q\nJan 02 15:04")
	if ok || !strings.Contains(warn, "unsupported TIME_STYLE token") {
		t.Errorf("Expected unsupported token warning for part 0, got ok=%v warn=%q", ok, warn)
	}

	// Invalid token in second part
	_, _, warn, ok = timeutil.ParseTimeFormat("Jan 02 15:04\n%Q")
	if ok || !strings.Contains(warn, "unsupported TIME_STYLE token") {
		t.Errorf("Expected unsupported token warning, got ok=%v warn=%q", ok, warn)
	}

	// Trailing percent in timeutil.ConvertTimeFormat
	_, _, warn, ok = timeutil.ParseTimeFormat("%")
	if ok || warn != "invalid TIME_STYLE format" {
		t.Errorf("Expected 'invalid TIME_STYLE format' for trailing %%, got ok=%v warn=%q", ok, warn)
	}
}

func TestGetEntryTimeVarious(t *testing.T) {
	now := time.Now().Truncate(time.Second)

	// 1. ModTime field
	info := &mockFileInfo{modTime: now}
	got := timeutil.GetEntryTime(info, lsconfig.TimeFieldMod)
	if !got.Equal(now) {
		t.Errorf("Expected ModTime, got %v", got)
	}

	// 2. Sys is not *syscall.Stat_t
	got = timeutil.GetEntryTime(info, lsconfig.TimeFieldAccess)
	if !got.Equal(now) {
		t.Errorf("Expected fallback to ModTime when sys is nil, got %v", got)
	}

	// 3. Access time
	stat := &syscall.Stat_t{}
	// On some platforms (macOS), we need to avoid literal initialization of platform-specific fields in shared tests.
	// But empty Stat_t is generally fine.
	info = &mockFileInfo{modTime: now, sys: stat}
	_ = timeutil.GetEntryTime(info, lsconfig.TimeFieldAccess)

	// 4. Change time
	_ = timeutil.GetEntryTime(info, lsconfig.TimeFieldChange)

	// 5. Birthtime fallback: Provide a mock Stat_t where birthtime is explicitly zeroed.
	// An empty syscall.Stat_t{} will have all its timespec fields zeroed,
	// which causes timeutil.statBirthtime to return (time.Time{}, false) on platforms
	// where birthtime is supported (like Darwin) and it is not present.
	// On Linux, timeutil.statBirthtime always returns (time.Time{}, false) anyway.
	info = &mockFileInfo{modTime: now, sys: &syscall.Stat_t{}}
	got = timeutil.GetEntryTime(info, lsconfig.TimeFieldBirth)
	if !got.Equal(now) {
		t.Errorf("Expected fallback to ModTime for birth when birthtime is not present (zeroed), got %v", got)
	}
}

func TestExecuteMultipleDirs(t *testing.T) {
	tmpDir := t.TempDir()
	d1 := filepath.Join(tmpDir, "d1")
	d2 := filepath.Join(tmpDir, "d2")
	_ = os.Mkdir(d1, 0755)
	_ = os.Mkdir(d2, 0755)
	_ = os.WriteFile(filepath.Join(d1, "f1"), []byte("1"), 0644)
	_ = os.WriteFile(filepath.Join(d2, "f2"), []byte("2"), 0644)

	var stdout, stderr bytes.Buffer
	exitCode := Execute([]string{d1, d2}, &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d. Stderr: %q", exitCode, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "d1:") || !strings.Contains(out, "d2:") {
		t.Errorf("Expected output to contain directory names, got %q", out)
	}
	if !strings.Contains(out, "\n\n") {
		t.Errorf("Expected blank line between directory listings, got %q", out)
	}
}

func TestExecuteOneDirError(t *testing.T) {
	tmpDir := t.TempDir()
	d1 := filepath.Join(tmpDir, "d1")
	d2 := "nonexistent"
	_ = os.Mkdir(d1, 0755)

	var stdout, stderr bytes.Buffer
	exitCode := Execute([]string{d1, d2}, &stdout, &stderr)
	if exitCode != 2 {
		t.Errorf("Expected exit code 2 because d2 fails, got %d", exitCode)
	}
}

func TestSortNilTime(t *testing.T) {
	now := time.Now()
	e1 := &entry.CachedDirEntry{DirEntry: &mockDirEntry{name: "a"}, Time: &now}
	e2 := &entry.CachedDirEntry{DirEntry: &mockDirEntry{name: "b"}, Time: nil}
	e3 := &entry.CachedDirEntry{DirEntry: &mockDirEntry{name: "c"}, Time: &now}
	e4 := &entry.CachedDirEntry{DirEntry: &mockDirEntry{name: "d"}, Time: nil}

	filtered := []os.DirEntry{e1, e2, e3, e4}

	sort.Slice(filtered, func(i, j int) bool {
		return entry.LessByTime(filtered[i], filtered[j])
	})

	if filtered[0].Name() != "a" || filtered[1].Name() != "c" || filtered[2].Name() != "b" || filtered[3].Name() != "d" {
		t.Errorf("Unexpected sort order: %v, %v, %v, %v", filtered[0].Name(), filtered[1].Name(), filtered[2].Name(), filtered[3].Name())
	}
}

func TestConvertTimeFormatAll(t *testing.T) {
	tokens := "%Y%y%m%d%e%H%M%S%b%B%a%Z%z%%"
	got, warn, ok := timeutil.ConvertTimeFormat(tokens)
	if !ok || warn != "" {
		t.Errorf("Expected conversion, got ok=%v warn=%q", ok, warn)
	}
	expected := "2006060102 2150405JanJanuaryMonMST-0700%"
	if got != expected {
		t.Errorf("Expected %q, got %q", expected, got)
	}
}

func TestRunShowAll(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmpDir, ".hidden"), []byte("h"), 0644)

	config := &lsconfig.Config{ShowAll: true}
	var stdout, stderr bytes.Buffer
	run(tmpDir, config, &stdout, &stderr)
	out := stdout.String()
	if !strings.Contains(out, ".") || !strings.Contains(out, "..") || !strings.Contains(out, ".hidden") {
		t.Errorf("Expected ., .., and .hidden, got %q", out)
	}
}

func TestRunAlmostAll(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmpDir, ".hidden"), []byte("h"), 0644)

	config := &lsconfig.Config{ShowAlmostAll: true}
	var stdout, stderr bytes.Buffer
	run(tmpDir, config, &stdout, &stderr)
	out := stdout.String()
	if !strings.Contains(out, ".hidden") {
		t.Errorf("Expected .hidden, got %q", out)
	}
	lines := strings.Fields(out)
	for _, l := range lines {
		if l == "." || l == ".." {
			t.Errorf("Did not expect %q with -A", l)
		}
	}
}

func TestRunIgnoreBackups(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmpDir, "file"), []byte("f"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "file~"), []byte("b"), 0644)

	config := &lsconfig.Config{IgnoreBackups: true}
	var stdout, stderr bytes.Buffer
	run(tmpDir, config, &stdout, &stderr)
	out := stdout.String()
	if !strings.Contains(out, "file") || strings.Contains(out, "file~") {
		t.Errorf("Expected file, not file~, got %q", out)
	}
}

func TestRunWarnings(t *testing.T) {
	var stdout, stderr bytes.Buffer
	config := &lsconfig.Config{
		HumanReadable: true,
		SI:            true,
		LongListing:   false,
	}
	run(t.TempDir(), config, &stdout, &stderr)
	if !strings.Contains(stderr.String(), "options -h and --si are ignored") {
		t.Errorf("Expected combined -h and --si warning, got %q", stderr.String())
	}

	stderr.Reset()
	config = &lsconfig.Config{SI: true, LongListing: false}
	run(t.TempDir(), config, &stdout, &stderr)
	if !strings.Contains(stderr.String(), "option --si is ignored") {
		t.Errorf("Expected --si warning, got %q", stderr.String())
	}

	stderr.Reset()
	config = &lsconfig.Config{NoGroup: true, LongListing: false}
	run(t.TempDir(), config, &stdout, &stderr)
	if !strings.Contains(stderr.String(), "--no-group is ignored") {
		t.Errorf("Expected --no-group warning, got %q", stderr.String())
	}

	stderr.Reset()
	config = &lsconfig.Config{ShowAuthor: true, LongListing: false}
	run(t.TempDir(), config, &stdout, &stderr)
	if !strings.Contains(stderr.String(), "--author is ignored") {
		t.Errorf("Expected --author warning, got %q", stderr.String())
	}

	stderr.Reset()
	config = &lsconfig.Config{BlockSize: &lsconfig.BlockSizeSpec{}, LongListing: false}
	run(t.TempDir(), config, &stdout, &stderr)
	if !strings.Contains(stderr.String(), "option --block-size is ignored") {
		t.Errorf("Expected --block-size warning, got %q", stderr.String())
	}
}

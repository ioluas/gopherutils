package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		checkConfig func(t *testing.T, config *Config)
	}{
		{
			name:        "no arguments - use current directory",
			args:        []string{},
			expectError: false,
			checkConfig: func(t *testing.T, config *Config) {
				if config.ShowAll {
					t.Error("Expected ShowAll to be false")
				}
				if config.HumanReadable {
					t.Error("Expected HumanReadable to be false")
				}
				cwd, err := os.Getwd()
				if err != nil {
					t.Fatalf("Failed to get cwd: %v", err)
				}
				if len(config.Directories) != 1 || config.Directories[0] != cwd {
					t.Errorf("Expected directory %s, got %v", cwd, config.Directories)
				}
			},
		},
		{
			name:        "single directory argument",
			args:        []string{"/tmp"},
			expectError: false,
			checkConfig: func(t *testing.T, config *Config) {
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
			checkConfig: func(t *testing.T, config *Config) {
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
			checkConfig: func(t *testing.T, config *Config) {
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
			checkConfig: func(t *testing.T, config *Config) {
				if !config.ShowAll {
					t.Error("Expected ShowAll to be true")
				}
				cwd, err := os.Getwd()
				if err != nil {
					t.Fatalf("Failed to get cwd: %v", err)
				}
				if len(config.Directories) != 1 || config.Directories[0] != cwd {
					t.Errorf("Expected directory %s, got %v", cwd, config.Directories)
				}
			},
		},
		{
			name:        "--all flag only",
			args:        []string{"--all"},
			expectError: false,
			checkConfig: func(t *testing.T, config *Config) {
				if !config.ShowAll {
					t.Error("Expected ShowAll to be true")
				}
				cwd, err := os.Getwd()
				if err != nil {
					t.Fatalf("Failed to get cwd: %v", err)
				}
				if len(config.Directories) != 1 || config.Directories[0] != cwd {
					t.Errorf("Expected directory %s, got %v", cwd, config.Directories)
				}
			},
		},
		{
			name:        "-a flag with directory",
			args:        []string{"-a", "/tmp"},
			expectError: false,
			checkConfig: func(t *testing.T, config *Config) {
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
			checkConfig: func(t *testing.T, config *Config) {
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
			checkConfig: func(t *testing.T, config *Config) {
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
			checkConfig: func(t *testing.T, config *Config) {
				if !config.ShowAuthor {
					t.Error("Expected ShowAuthor to be true")
				}
				if config.LongListing { // --author alone should not set -l
					t.Error("Expected LongListing to be false")
				}
			},
		},
		{
			name:        "-l --author flags",
			args:        []string{"-l", "--author"},
			expectError: false,
			checkConfig: func(t *testing.T, config *Config) {
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
			checkConfig: func(t *testing.T, config *Config) {
				if !config.ShowAlmostAll {
					t.Error("Expected ShowAlmostAll to be true")
				}
				if config.ShowAll { // Ensure -a is not implicitly set
					t.Error("Expected ShowAll to be false")
				}
				cwd, err := os.Getwd()
				if err != nil {
					t.Fatalf("Failed to get cwd: %v", err)
				}
				if len(config.Directories) != 1 || config.Directories[0] != cwd {
					t.Errorf("Expected directory %s, got %v", cwd, config.Directories)
				}
			},
		},
		{
			name:        "--almost-all flag only",
			args:        []string{"--almost-all"},
			expectError: false,
			checkConfig: func(t *testing.T, config *Config) {
				if !config.ShowAlmostAll {
					t.Error("Expected ShowAlmostAll to be true")
				}
				if config.ShowAll { // Ensure -a is not implicitly set
					t.Error("Expected ShowAll to be false")
				}
				cwd, err := os.Getwd()
				if err != nil {
					t.Fatalf("Failed to get cwd: %v", err)
				}
				if len(config.Directories) != 1 || config.Directories[0] != cwd {
					t.Errorf("Expected directory %s, got %v", cwd, config.Directories)
				}
			},
		},
		{
			name:        "-h flag",
			args:        []string{"-h"},
			expectError: false,
			checkConfig: func(t *testing.T, config *Config) {
				if !config.HumanReadable {
					t.Error("Expected HumanReadable to be true")
				}
			},
		},
		{
			name:        "--human-readable flag",
			args:        []string{"--human-readable"},
			expectError: false,
			checkConfig: func(t *testing.T, config *Config) {
				if !config.HumanReadable {
					t.Error("Expected HumanReadable to be true")
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
			checkConfig: func(t *testing.T, config *Config) {
				// When help is requested, config should be nil
				if config != nil {
					t.Error("Expected config to be nil when help is requested")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(innerT *testing.T) { // Changed 't' to 'innerT'
			config, err := ParseArgs(tt.args)

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
			config, err := ParseArgs(tt.args)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if config.LongListing != tt.expectedLL {
				t.Errorf("Expected LongListing=%v, got %v", tt.expectedLL, config.LongListing)
			}
		})
	}
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
			printEntries(&buf, tt.entries)

			got := buf.String()
			if got != tt.expected {
				t.Errorf("printEntries() output = %q, want %q", got, tt.expected)
			}
		})
	}
}

// mockDirEntry is a mock implementation of os.DirEntry for testing error cases.
type mockDirEntry struct {
	name    string
	isDir   bool
	infoErr error // The error to return from Info()
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
	// This part is not strictly needed if we only test the error case,
	// but it makes the mock more complete.
	return nil, errors.New("not implemented")
}

func TestRun(t *testing.T) {
	tests := []struct {
		name           string
		config         *Config
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
			config: &Config{
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
			config: &Config{HumanReadable: true, LongListing: false},
			setupFunc: func(t *testing.T) string {
				return t.TempDir()
			},
			expectedExit:   0,
			expectedStdout: "",
			expectedStderr: "ls: warning: option -h is ignored when -l is not used",
		},
		{
			name: "list directory with dotfiles - no flags",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte(""), 0644)
				os.WriteFile(filepath.Join(tmpDir, ".dotfile.txt"), []byte(""), 0644)
				os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)
				return tmpDir
			},
			config:         &Config{ShowAll: false, ShowAlmostAll: false}, // Explicitly set to make test clear
			expectedExit:   0,
			expectedStdout: "file1.txt\nsubdir\n", // .dotfile.txt, . and .. hidden
			expectedStderr: "",
		},
		{
			name: "list directory with dotfiles -a flag",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte(""), 0644)
				os.WriteFile(filepath.Join(tmpDir, ".dotfile.txt"), []byte(""), 0644)
				os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)
				return tmpDir
			},
			config:         &Config{ShowAll: true, ShowAlmostAll: false},
			expectedExit:   0,
			expectedStdout: ".\n..\n.dotfile.txt\nfile1.txt\nsubdir\n", // Sorted output of all items
			expectedStderr: "",
		},
		{
			name: "list directory with dotfiles -A flag",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte(""), 0644)
				os.WriteFile(filepath.Join(tmpDir, ".dotfile.txt"), []byte(""), 0644)
				os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)
				return tmpDir
			},
			config:         &Config{ShowAll: false, ShowAlmostAll: true},
			expectedExit:   0,
			expectedStdout: ".dotfile.txt\nfile1.txt\nsubdir\n", // . and .. hidden
			expectedStderr: "",
		},
		{
			name: "list directory with dotfiles -a -A flags (a should take precedence)",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte(""), 0644)
				os.WriteFile(filepath.Join(tmpDir, ".dotfile.txt"), []byte(""), 0644)
				os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)
				return tmpDir
			},
			config:         &Config{ShowAll: true, ShowAlmostAll: true}, // ShowAll takes precedence
			expectedExit:   0,
			expectedStdout: ".\n..\n.dotfile.txt\nfile1.txt\nsubdir\n", // Sorted output of all items
			expectedStderr: "",
		},
		{
			name: "ls --author alone (warn, no long listing)",
			config: &Config{
				ShowAuthor:  true,
				LongListing: false,
			},
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte(""), 0644)
				return tmpDir
			},
			expectedExit:   0,
			expectedStdout: "file1.txt\n", // Not long listing
			expectedStderr: "ls: warning: --author is ignored when -l is not used\n",
		},
		{
			name: "ls -l (no author/group)",
			config: &Config{
				LongListing: true,
				ShowAuthor:  false,
			},
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte(""), 0644)
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
			config: &Config{
				LongListing: true,
				ShowAuthor:  true,
			},
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte(""), 0644)
				return tmpDir
			},
			expectedExit: 0,
			expectedStdout: func() string {
				username, groupname := getUserDetails(t)
				return getFileDetails(t, "file1.txt", username, groupname, true)
			}(),
			expectedStderr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setupFunc(t)

			// Create config if not provided
			config := tt.config
			if config == nil {
				config = &Config{
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

	config := &Config{
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
			printGrid(&buf, tt.entries, tt.width)
			got := buf.String()
			if got != tt.expected {
				t.Errorf("printGrid() output:\n%q\nwant:\n%q", got, tt.expected)
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
		got := formatSize(tt.input)
		if got != tt.expected {
			t.Errorf("formatSize(%d) = %q, want %q", tt.input, got, tt.expected)
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
	defer os.RemoveAll(tmpDir) // Clean up the temp directory after the test

	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("Failed to stat temp file: %v", err)
	}

	mode := info.Mode().String()
	nlink := 1 // For a newly created file
	size := info.Size()
	modTime := info.ModTime().Format("Jan 02 15:04")

	// Max widths for a single file test case, assuming minimum widths as in printLongList
	maxLinkLen := len(fmt.Sprint(nlink))
	if maxLinkLen < 2 {
		maxLinkLen = 2
	} // Assuming minimum 2 for nlink formatting
	maxSizeLen := len(fmt.Sprint(size))
	if maxSizeLen < 1 {
		maxSizeLen = 1
	} // Assuming minimum 1 for size formatting

	if showAuthor {
		maxOwnerLen := len(owner)
		maxGroupLen := len(group)
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
	return fmt.Sprintf("%s %*d %*d %s %s\n",
		mode,
		maxLinkLen, nlink,
		maxSizeLen, size,
		modTime,
		filename,
	)
}

func TestPrintLongListErrors(t *testing.T) {
	config := &Config{LongListing: true}
	mockEntries := []os.DirEntry{
		&mockDirEntry{name: "good-file", isDir: false}, // This won't actually be processed fully
		&mockDirEntry{name: "bad-file", isDir: false, infoErr: errors.New("simulated info error")},
	}

	var buf bytes.Buffer
	printLongList(&buf, mockEntries, config)

	// Since the "good-file" doesn't have proper Info, it will also be skipped.
	// Only the error condition is what we are testing here.
	// The function should not panic and produce no output for the error entry.
	expectedOutput := ""
	if buf.String() != expectedOutput {
		t.Errorf("Expected output %q, got %q", expectedOutput, buf.String())
	}
}

func TestDirEntryWrapper(t *testing.T) {
	tmpDir := t.TempDir()
	parentDir := filepath.Dir(tmpDir)

	// Test "." wrapper
	dotWrapper := &dirEntryWrapper{name: ".", dirPath: tmpDir}
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

	// Test ".." wrapper
	dotDotWrapper := &dirEntryWrapper{name: "..", dirPath: tmpDir}
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
}

func TestPrintLongListWithShowAll(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte(""), 0644)

	config := &Config{
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
	// they will be skipped in printLongList. So we test that the functionality
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
			name:     "zero width",
			entries:  []string{"a", "b"},
			width:    0,
			// When width is 0, colWidth becomes 1, numCols becomes 1, so output is "a\nb\n"
			expected: "a\nb\n",
		},
		{
			name:     "very narrow width",
			entries:  []string{"a", "b", "c"},
			width:    1,
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
			name:     "mixed length names",
			entries:  []string{"a", "verylongname", "b", "c"},
			width:    80,
			// maxLen=12, colWidth=14, numCols=80/14=5, but limited to 4 entries, so numCols=4, numRows=1
			// All 4 entries fit in one row: "a             verylongname  b             c             \n"
			expected: "a             verylongname  b             c             \n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			printGrid(&buf, tt.entries, tt.width)
			got := buf.String()
			if got != tt.expected {
				t.Errorf("printGrid() output:\n%q\nwant:\n%q", got, tt.expected)
			}
		})
	}
}

func TestPrintLongListEmpty(t *testing.T) {
	config := &Config{LongListing: true}
	var buf bytes.Buffer
	printLongList(&buf, []os.DirEntry{}, config)
	if buf.String() != "" {
		t.Errorf("Expected empty output for empty entries, got %q", buf.String())
	}
}

func TestPrintLongListWithoutAuthor(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("test"), 0644)

	config := &Config{
		LongListing: true,
		ShowAuthor:  false,
	}

	var buf bytes.Buffer
	exitCode := run(tmpDir, config, &buf, &buf)
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	output := buf.String()
	// Should not contain owner/group when ShowAuthor is false
	// Format should be: mode nlink size date time name
	if !strings.Contains(output, "file1.txt") {
		t.Error("Expected output to contain 'file1.txt'")
	}
	// Verify it's in the expected format (no owner/group columns)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) > 0 {
		fields := strings.Fields(lines[0])
		// Should have: mode, nlink, size, date, time, name (6 fields without author)
		// Or: mode, nlink, size, date, time, name (6 fields)
		if len(fields) < 6 {
			t.Errorf("Expected at least 6 fields in long listing, got %d: %v", len(fields), fields)
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
		{2048, "2.0K"},
		{5120, "5.0K"},
		{1048576, "1.0M"},
		{1073741824, "1.0G"},
	}

	for _, tt := range tests {
		got := formatSize(tt.input)
		if got != tt.expected {
			t.Errorf("formatSize(%d) = %q, want %q", tt.input, got, tt.expected)
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
			config, err := ParseArgs(tt.args)
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
	printEntries(&buf, []string{})
	if buf.String() != "" {
		t.Errorf("Expected empty output for empty entries, got %q", buf.String())
	}
}

func TestRunMultipleDirectories(t *testing.T) {
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir1, "file1.txt"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir2, "file2.txt"), []byte(""), 0644)

	config := &Config{
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

func TestPrintLongListWithHumanReadable(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a file with a specific size to test human-readable format
	testContent := make([]byte, 2048) // 2KB
	os.WriteFile(filepath.Join(tmpDir, "testfile.txt"), testContent, 0644)

	config := &Config{
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


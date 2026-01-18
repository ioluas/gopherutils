package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
				cwd, err := os.Getwd()
				if err != nil {
					t.Fatalf("Failed to get cwd: %v", err)
				}
				if config.Directory != cwd {
					t.Errorf("Expected directory %s, got %s", cwd, config.Directory)
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
				if config.Directory != "/tmp" {
					t.Errorf("Expected directory /tmp, got %s", config.Directory)
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
				if config.Directory != cwd {
					t.Errorf("Expected directory %s, got %s", cwd, config.Directory)
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
				if config.Directory != cwd {
					t.Errorf("Expected directory %s, got %s", cwd, config.Directory)
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
				if config.Directory != "/tmp" {
					t.Errorf("Expected directory /tmp, got %s", config.Directory)
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
				if config.Directory != "/var" {
					t.Errorf("Expected directory /var, got %s", config.Directory)
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
				if config.Directory != "/tmp" {
					t.Errorf("Expected directory /tmp, got %s", config.Directory)
				}
			},
		},
		{
			name:        "unknown flag",
			args:        []string{"-x"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ParseArgs(tt.args)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError && tt.checkConfig != nil {
				tt.checkConfig(t, config)
			}
		})
	}
}

func TestParseDirectory(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		checkResult func(t *testing.T, result string)
	}{
		{
			name:        "no arguments - use current directory",
			args:        []string{},
			expectError: false,
			checkResult: func(t *testing.T, result string) {
				// Result should be a valid directory path
				if result == "" {
					t.Error("Expected non-empty directory path")
				}
				// Should be the current working directory
				cwd, err := os.Getwd()
				if err != nil {
					t.Fatalf("Failed to get cwd: %v", err)
				}
				if result != cwd {
					t.Errorf("Expected cwd %s, got %s", cwd, result)
				}
			},
		},
		{
			name:        "single argument - use specified directory",
			args:        []string{"/tmp"},
			expectError: false,
			checkResult: func(t *testing.T, result string) {
				if result != "/tmp" {
					t.Errorf("Expected /tmp, got %s", result)
				}
			},
		},
		{
			name:        "relative path argument",
			args:        []string{"./test"},
			expectError: false,
			checkResult: func(t *testing.T, result string) {
				if result != "./test" {
					t.Errorf("Expected ./test, got %s", result)
				}
			},
		},
		{
			name:        "multiple arguments - use first one",
			args:        []string{"/tmp", "/var", "/usr"},
			expectError: false,
			checkResult: func(t *testing.T, result string) {
				if result != "/tmp" {
					t.Errorf("Expected /tmp, got %s", result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDirectory(tt.args)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError && tt.checkResult != nil {
				tt.checkResult(t, result)
			}
		})
	}
}

func TestListEntries(t *testing.T) {
	// Create a temporary directory with test files
	tmpDir := t.TempDir()

	// Create test files (including hidden files)
	testFiles := []string{"file1.txt", "file2.txt", "file3.txt", ".hidden"}
	for _, file := range testFiles {
		path := filepath.Join(tmpDir, file)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Read directory entries
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	t.Run("without showAll flag", func(t *testing.T) {
		// Test listEntries function without showAll
		names := listEntries(entries, false)

		// Verify we got only non-hidden files
		expectedCount := 3 // Only non-hidden files
		if len(names) != expectedCount {
			t.Errorf("Expected %d entries, got %d", expectedCount, len(names))
		}

		// Verify hidden file is NOT in the result
		for _, name := range names {
			if strings.HasPrefix(name, ".") {
				t.Errorf("Hidden file %s should not be in results", name)
			}
		}
	})

	t.Run("with showAll flag", func(t *testing.T) {
		// Test listEntries function with showAll
		names := listEntries(entries, true)

		// Verify we got all files including hidden
		if len(names) != len(testFiles) {
			t.Errorf("Expected %d entries, got %d", len(testFiles), len(names))
		}

		// Verify each file is in the result
		nameSet := make(map[string]bool)
		for _, name := range names {
			nameSet[name] = true
		}

		for _, expected := range testFiles {
			if !nameSet[expected] {
				t.Errorf("Expected file %s not found in results", expected)
			}
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
			expected: "file1.txt \n",
		},
		{
			name:     "multiple entries",
			entries:  []string{"file1.txt", "file2.txt", "file3.txt"},
			expected: "file1.txt file2.txt file3.txt \n",
		},
		{
			name:     "empty directory",
			entries:  []string{},
			expected: "\n",
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
			expectedStdout: "a.txt b.txt c.txt \n",
			expectedStderr: "",
		},
		{
			name: "list empty directory",
			setupFunc: func(t *testing.T) string {
				return t.TempDir()
			},
			expectedExit:   0,
			expectedStdout: "\n",
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
			expectedStdout: "visible.txt \n",
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
			expectedStdout: ".hidden visible.txt \n",
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
			config.Directory = dir

			var stdout, stderr bytes.Buffer
			exitCode := run(config, &stdout, &stderr)

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
		Directory: tmpDir,
		ShowAll:   false,
	}

	var stdout, stderr bytes.Buffer
	exitCode := run(config, &stdout, &stderr)

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

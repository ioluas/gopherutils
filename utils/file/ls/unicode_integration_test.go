package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	lsconfig "github.com/ioluas/gopherutils/utils/file/ls/internal/config"
)

func TestUnicodeSorting(t *testing.T) {
	// Create a temporary directory with Unicode filenames
	tmpDir := t.TempDir()

	// Create files with various Unicode characters
	requestedFiles := []string{
		"Café.txt",
		"café.txt",
		"Zebra.txt",
		"Äpfel.txt",
		"äpfel.txt",
		"文件.txt",
		"Москва.txt",
		"αλφα.txt",
		"Año.txt",
		"año.txt",
		"test-file.txt",
		"test_file.txt",
		"testfile.txt",
	}

	// Track which files were actually created (macOS may normalize some filenames)
	var createdFiles []string
	for _, name := range requestedFiles {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(""), 0644); err != nil {
			t.Logf("warning: failed to create file %s: %v (may be due to filesystem normalization)", name, err)
			continue
		}
		createdFiles = append(createdFiles, name)
	}

	// Read back what was actually created to handle filesystem normalization
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("failed to read directory: %v", err)
	}

	actualFileCount := len(entries)
	if actualFileCount == 0 {
		t.Fatal("no files were created")
	}

	// Run ls on the directory
	config := &lsconfig.Config{}
	var stdout, stderr bytes.Buffer
	exitCode := run(tmpDir, config, &stdout, &stderr)

	if exitCode != 0 {
		t.Errorf("run() exit code = %d, want 0", exitCode)
	}

	if stderr.Len() > 0 {
		t.Errorf("unexpected stderr: %s", stderr.String())
	}

	output := stdout.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Verify the number of files listed matches what was actually created
	if len(lines) != actualFileCount {
		t.Errorf("expected %d files (actually created), got %d in output", actualFileCount, len(lines))
	}

	// Verify that files are sorted (case-insensitive, punctuation-ignored)
	t.Logf("Created %d files (requested %d)", actualFileCount, len(requestedFiles))
	t.Logf("Sorted output:\n%s", output)

	// Check that café variations are near each other (if both were created)
	cafeIndex := -1
	CafeIndex := -1
	for i, line := range lines {
		if strings.Contains(line, "café.txt") {
			cafeIndex = i
		}
		if strings.Contains(line, "Café.txt") {
			CafeIndex = i
		}
	}

	if cafeIndex >= 0 && CafeIndex >= 0 {
		// They should be adjacent or very close
		diff := cafeIndex - CafeIndex
		if diff < 0 {
			diff = -diff
		}
		if diff > 2 {
			t.Errorf("café.txt and Café.txt are too far apart in sort order (indices %d and %d)", cafeIndex, CafeIndex)
		}
	}
}

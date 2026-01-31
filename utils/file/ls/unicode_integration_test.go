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
	files := []string{
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

	for _, name := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(""), 0644); err != nil {
			t.Fatalf("failed to create file %s: %v", name, err)
		}
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

	// Verify all files are listed
	if len(lines) != len(files) {
		t.Errorf("expected %d files, got %d", len(files), len(lines))
	}

	// Verify that files are sorted (case-insensitive, punctuation-ignored)
	// The exact order depends on Unicode collation, but we can verify
	// that similar names are grouped together
	t.Logf("Sorted output:\n%s", output)

	// Check that café variations are near each other
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

//go:build !windows

package display

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	lsconfig "github.com/ioluas/gopherutils/utils/file/ls/internal/config"
)

func TestPrintLongListDired(t *testing.T) {
	now := time.Now()
	// Create mock entries using the mockDirEntry defined in display_longlist_test.go
	entries := []os.DirEntry{
		mockDirEntry{
			name:    "file1",
			fileMod: 0644,
			fileSiz: 100,
			modTime: now,
		},
		mockDirEntry{
			name:    "file2",
			fileMod: 0644,
			fileSiz: 200,
			modTime: now,
		},
	}

	config := &lsconfig.Config{
		Dired:       true,
		LongListing: true,
	}

	var stdout, stderr bytes.Buffer

	PrintLongList(&stdout, &stderr, entries, config, true)

	output := stdout.String()
	// Using TrimRight to preserve leading indentation which is significant for -D
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	if len(lines) < 5 {
		t.Fatalf("Expected at least 5 lines of output (total + 2 files + 2 dired lines), got %d:\n%s", len(lines), output)
	}

	if !strings.HasPrefix(lines[0], "  total") {
		t.Errorf("Expected first line to be indented total block count, got: %q", lines[0])
	}

	diredLineFound := false
	for _, line := range lines {
		if strings.HasPrefix(line, "//DIRED//") {
			diredLineFound = true
			if fields := strings.Fields(line); len(fields) < 2 {
				t.Errorf("Expected offsets in //DIRED// line, got: %s", line)
			}
			break
		}
	}

	if !diredLineFound {
		t.Errorf("Expected //DIRED// line in output, but got:\n%s", output)
	}

	optionsLineFound := false
	for _, line := range lines {
		if strings.HasPrefix(line, "//DIRED-OPTIONS//") {
			optionsLineFound = true
			break
		}
	}

	if !optionsLineFound {
		t.Errorf("Expected //DIRED-OPTIONS// line in output, but got:\n%s", output)
	}
}

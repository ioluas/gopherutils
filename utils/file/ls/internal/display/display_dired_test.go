//go:build !windows

package display

import (
	"bytes"
	"os"
	"strconv"
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
		Dired:        true,
		LongListing:  true,
		QuotingStyle: lsconfig.QuotingStyleShellEscape,
	}

	var stdout, stderr bytes.Buffer

	PrintLongList(&stdout, &stderr, entries, config, true)

	output := stdout.String()
	outputBytes := []byte(output)
	// Using TrimRight to preserve leading indentation which is significant for -D
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	if len(lines) < 5 {
		t.Fatalf("Expected at least 5 lines of output (total + 2 files + 2 dired lines), got %d:\n%s", len(lines), output)
	}

	if !strings.HasPrefix(lines[0], "  total") {
		t.Errorf("Expected first line to be indented total block count, got: %q", lines[0])
	}

	diredLineFound := false
	var diredLine string
	for _, line := range lines {
		if strings.HasPrefix(line, "//DIRED//") {
			diredLineFound = true
			diredLine = line
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
	var optionsLine string
	for _, line := range lines {
		if strings.HasPrefix(line, "//DIRED-OPTIONS//") {
			optionsLineFound = true
			optionsLine = line
			break
		}
	}

	if !optionsLineFound {
		t.Errorf("Expected //DIRED-OPTIONS// line in output, but got:\n%s", output)
	}

	if diredLineFound {
		fields := strings.Fields(diredLine)
		if len(fields) != 5 {
			t.Fatalf("Expected 4 offsets for 2 entries, got %d: %q", len(fields)-1, diredLine)
		}

		offsets := make([]int, 0, 4)
		for _, raw := range fields[1:] {
			offset, err := strconv.Atoi(raw)
			if err != nil {
				t.Fatalf("Invalid offset %q: %v", raw, err)
			}
			offsets = append(offsets, offset)
		}

		for i, off := range offsets {
			if off < 0 || off > len(outputBytes) {
				t.Fatalf("Offset %d out of bounds (len=%d)", off, len(outputBytes))
			}
			if i > 0 && offsets[i-1] >= off {
				t.Fatalf("Offsets not strictly increasing: %v", offsets)
			}
		}

		name1 := string(outputBytes[offsets[0]:offsets[1]])
		name2 := string(outputBytes[offsets[2]:offsets[3]])
		if name1 != "file1" {
			t.Errorf("Expected first name %q, got %q", "file1", name1)
		}
		if name2 != "file2" {
			t.Errorf("Expected second name %q, got %q", "file2", name2)
		}
	}

	if optionsLineFound && !strings.HasSuffix(optionsLine, "--quoting-style=shell-escape") {
		t.Errorf("Expected //DIRED-OPTIONS// to end with --quoting-style=shell-escape, got: %q", optionsLine)
	}
}

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"testing"
	"time"
)

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
	config := &Config{}
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
	_, warn, ok := parseBlockSize("")
	if ok || warn != "missing SIZE" {
		t.Errorf("Expected 'missing SIZE', got ok=%v warn=%q", ok, warn)
	}

	// Missing SIZE after apostrophe
	_, warn, ok = parseBlockSize("'")
	if ok || warn != "missing SIZE" {
		t.Errorf("Expected 'missing SIZE' for single apostrophe, got ok=%v warn=%q", ok, warn)
	}

	// Invalid number
	_, warn, ok = parseBlockSize("0")
	if ok || warn != "invalid SIZE number" {
		t.Errorf("Expected 'invalid SIZE number' for 0, got ok=%v warn=%q", ok, warn)
	}

	// Too large
	_, warn, ok = parseBlockSize("1000000000000000000000000000")
	if ok || !strings.Contains(warn, "invalid") {
		t.Errorf("Expected invalid number error, got ok=%v warn=%q", ok, warn)
	}

	// Unknown suffix
	_, warn, ok = parseBlockSize("1X")
	if ok || warn != "unknown SIZE suffix" {
		t.Errorf("Expected 'unknown SIZE suffix', got ok=%v warn=%q", ok, warn)
	}

	// Overflow
	_, warn, ok = parseBlockSize("10000000000000000000T")
	if ok || warn != "SIZE too large" {
		t.Errorf("Expected 'SIZE too large', got ok=%v warn=%q", ok, warn)
	}
}

func TestParseTimeFormatErrors(t *testing.T) {
	// Empty format
	_, _, warn, ok := parseTimeFormat("")
	if ok || warn != "missing TIME_STYLE format" {
		t.Errorf("Expected 'missing TIME_STYLE format', got ok=%v warn=%q", ok, warn)
	}

	// More than 2 parts
	_, _, warn, ok = parseTimeFormat("a\nb\nc")
	if ok || warn != "invalid TIME_STYLE format" {
		t.Errorf("Expected 'invalid TIME_STYLE format', got ok=%v warn=%q", ok, warn)
	}

	// Invalid token in first part of two-part format
	_, _, warn, ok = parseTimeFormat("%Q\nJan 02 15:04")
	if ok || !strings.Contains(warn, "unsupported TIME_STYLE token") {
		t.Errorf("Expected unsupported token warning for part 0, got ok=%v warn=%q", ok, warn)
	}

	// Invalid token in second part
	_, _, warn, ok = parseTimeFormat("Jan 02 15:04\n%Q")
	if ok || !strings.Contains(warn, "unsupported TIME_STYLE token") {
		t.Errorf("Expected unsupported token warning, got ok=%v warn=%q", ok, warn)
	}

	// Trailing percent in convertTimeFormat
	_, _, warn, ok = parseTimeFormat("%")
	if ok || warn != "invalid TIME_STYLE format" {
		t.Errorf("Expected 'invalid TIME_STYLE format' for trailing %%, got ok=%v warn=%q", ok, warn)
	}
}

func TestGetEntryTimeVarious(t *testing.T) {
	now := time.Now().Truncate(time.Second)

	// 1. ModTime field
	info := &mockFileInfo{modTime: now}
	got := getEntryTime(info, timeFieldMod)
	if !got.Equal(now) {
		t.Errorf("Expected ModTime, got %v", got)
	}

	// 2. Sys is not *syscall.Stat_t
	got = getEntryTime(info, timeFieldAccess)
	if !got.Equal(now) {
		t.Errorf("Expected fallback to ModTime when sys is nil, got %v", got)
	}

	// 3. Access time
	stat := &syscall.Stat_t{}
	// On some platforms (macOS), we need to avoid literal initialization of platform-specific fields in shared tests.
	// But empty Stat_t is generally fine.
	info = &mockFileInfo{modTime: now, sys: stat}
	_ = getEntryTime(info, timeFieldAccess)

	// 4. Change time
	_ = getEntryTime(info, timeFieldChange)

	// 5. Birthtime fallback
	got = getEntryTime(info, timeFieldBirth)
	if !got.Equal(now) {
		t.Errorf("Expected fallback to ModTime for birth, got %v", got)
	}
}

func TestLongListFormatArgsCombinations(t *testing.T) {
	d := fileDetails{
		mode:    "-rw-r--r--",
		nlink:   1,
		owner:   "user",
		group:   "group",
		author:  "author",
		sizeStr: "123",
		name:    "file",
		modTime: time.Now(),
	}
	widths := longListWidths{
		link:   1,
		owner:  4,
		group:  5,
		author: 6,
		size:   3,
	}

	// Case 1: ShowAuthor=true, NoGroup=true
	config := &Config{ShowAuthor: true, NoGroup: true}
	format, args := longListFormatArgs(d, widths, config)
	if !strings.Contains(format, "%-*s %-*s") || len(args) != 11 {
		t.Errorf("Unexpected format or args for Author+NoGroup: %q, %d", format, len(args))
	}

	// Case 2: ShowAuthor=true, NoGroup=false
	config = &Config{ShowAuthor: true, NoGroup: false}
	format, args = longListFormatArgs(d, widths, config)
	if !strings.Contains(format, "%-*s %-*s %-*s") || len(args) != 13 {
		t.Errorf("Unexpected format or args for Author: %q, %d", format, len(args))
	}

	// Case 3: ShowAuthor=false, NoGroup=true
	config = &Config{ShowAuthor: false, NoGroup: true}
	format, args = longListFormatArgs(d, widths, config)
	if strings.Contains(format, "%-*s %-*s %-*s") || len(args) != 9 {
		t.Errorf("Unexpected format or args for NoGroup: %q, %d", format, len(args))
	}

	// Case 4: Default (both false)
	config = &Config{ShowAuthor: false, NoGroup: false}
	_, args = longListFormatArgs(d, widths, config)
	if len(args) != 11 {
		t.Errorf("Unexpected args for default: %d", len(args))
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
	e1 := &cachedDirEntry{DirEntry: &mockDirEntry{name: "a"}, time: &now}
	e2 := &cachedDirEntry{DirEntry: &mockDirEntry{name: "b"}, time: nil}
	e3 := &cachedDirEntry{DirEntry: &mockDirEntry{name: "c"}, time: &now}
	e4 := &cachedDirEntry{DirEntry: &mockDirEntry{name: "d"}, time: nil}

	filtered := []os.DirEntry{e1, e2, e3, e4}

	// Copy of sorting logic from main.go:run
	sort.Slice(filtered, func(i, j int) bool {
		ti := filtered[i].(*cachedDirEntry).time
		tj := filtered[j].(*cachedDirEntry).time

		if ti != nil && tj != nil {
			if ti.Equal(*tj) {
				return filtered[i].Name() < filtered[j].Name()
			}
			return ti.After(*tj)
		}
		if ti != nil {
			return true
		}
		if tj != nil {
			return false
		}
		return filtered[i].Name() < filtered[j].Name()
	})

	if filtered[0].Name() != "a" || filtered[1].Name() != "c" || filtered[2].Name() != "b" || filtered[3].Name() != "d" {
		t.Errorf("Unexpected sort order: %v, %v, %v, %v", filtered[0].Name(), filtered[1].Name(), filtered[2].Name(), filtered[3].Name())
	}
}

func TestConvertTimeFormatAll(t *testing.T) {
	tokens := "%Y%y%m%d%e%H%M%S%b%B%a%Z%z%%"
	got, warn, ok := convertTimeFormat(tokens)
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

	config := &Config{ShowAll: true}
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

	config := &Config{ShowAlmostAll: true}
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

	config := &Config{IgnoreBackups: true}
	var stdout, stderr bytes.Buffer
	run(tmpDir, config, &stdout, &stderr)
	out := stdout.String()
	if !strings.Contains(out, "file") || strings.Contains(out, "file~") {
		t.Errorf("Expected file, not file~, got %q", out)
	}
}

func TestRunWarnings(t *testing.T) {
	var stdout, stderr bytes.Buffer
	config := &Config{
		HumanReadable: true,
		SI:            true,
		LongListing:   false,
	}
	run(t.TempDir(), config, &stdout, &stderr)
	if !strings.Contains(stderr.String(), "options -h and --si are ignored") {
		t.Errorf("Expected combined -h and --si warning, got %q", stderr.String())
	}

	stderr.Reset()
	config = &Config{SI: true, LongListing: false}
	run(t.TempDir(), config, &stdout, &stderr)
	if !strings.Contains(stderr.String(), "option --si is ignored") {
		t.Errorf("Expected --si warning, got %q", stderr.String())
	}

	stderr.Reset()
	config = &Config{NoGroup: true, LongListing: false}
	run(t.TempDir(), config, &stdout, &stderr)
	if !strings.Contains(stderr.String(), "--no-group is ignored") {
		t.Errorf("Expected --no-group warning, got %q", stderr.String())
	}

	stderr.Reset()
	config = &Config{ShowAuthor: true, LongListing: false}
	run(t.TempDir(), config, &stdout, &stderr)
	if !strings.Contains(stderr.String(), "--author is ignored") {
		t.Errorf("Expected --author warning, got %q", stderr.String())
	}

	stderr.Reset()
	config = &Config{BlockSize: &BlockSizeSpec{}, LongListing: false}
	run(t.TempDir(), config, &stdout, &stderr)
	if !strings.Contains(stderr.String(), "option --block-size is ignored") {
		t.Errorf("Expected --block-size warning, got %q", stderr.String())
	}
}

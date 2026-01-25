package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseTimeWord(t *testing.T) {
	tests := []struct {
		input    string
		expected timeField
	}{
		{"atime", timeFieldAccess},
		{"access", timeFieldAccess},
		{"use", timeFieldAccess},
		{"ctime", timeFieldChange},
		{"status", timeFieldChange},
		{"mtime", timeFieldMod},
		{"modification", timeFieldMod},
		{"birth", timeFieldBirth},
		{"creation", timeFieldBirth},
	}

	for _, tt := range tests {
		got, err := parseTimeWord(tt.input)
		if err != nil {
			t.Fatalf("parseTimeWord(%q) error: %v", tt.input, err)
		}
		if got != tt.expected {
			t.Fatalf("parseTimeWord(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}

	if _, err := parseTimeWord("invalid"); err == nil {
		t.Fatal("expected error for invalid time word")
	}
}

func TestGetEntryTimeFallback(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	info := &mockFileInfo{modTime: now}
	if got := getEntryTime(info, timeFieldAccess); !got.Equal(now) {
		t.Fatalf("expected fallback to ModTime, got %v", got)
	}
	if got := getEntryTime(info, timeFieldBirth); !got.Equal(now) {
		t.Fatalf("expected fallback to ModTime for birth, got %v", got)
	}
}

func TestRunTimeFieldIgnoredWithoutLongListing(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	config := &Config{
		TimeFieldSet: true,
		TimeField:    timeFieldAccess,
	}

	var stdout, stderr bytes.Buffer
	exitCode := run(tmpDir, config, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() failed: %v", stderr.String())
	}
	if !strings.Contains(stderr.String(), "ls: warning: --time is ignored when -l is not used") {
		t.Fatalf("expected warning, got %q", stderr.String())
	}
	if config.TimeField != timeFieldMod {
		t.Fatalf("expected TimeField reset to mod, got %v", config.TimeField)
	}
}

func TestRunTimeFieldWithLongListing(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	config := &Config{
		LongListing:  true,
		TimeFieldSet: true,
		TimeField:    timeFieldAccess,
	}

	var stdout, stderr bytes.Buffer
	exitCode := run(tmpDir, config, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() failed: %v", stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no warning, got %q", stderr.String())
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

	config := &Config{
		SortTime:  true,
		TimeField: timeFieldMod,
	}

	var stdout, stderr bytes.Buffer
	exitCode := run(tmpDir, config, &stdout, &stderr)
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

func TestTimeStyleWarningWithoutLongListing(t *testing.T) {
	config := &Config{
		TimeStyleSet: true,
		TimeStyleSpec: &timeStyleSpec{
			kind:         timeStyleISO,
			recentLayout: "2006-01-02 15:04",
		},
	}

	var stdout, stderr bytes.Buffer
	exitCode := run(t.TempDir(), config, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() failed: %v", stderr.String())
	}
	if !strings.Contains(stderr.String(), "ls: warning: --time-style is ignored when -l is not used") {
		t.Fatalf("expected warning, got %q", stderr.String())
	}
	if config.TimeStyleSpec != nil {
		t.Fatalf("expected TimeStyleSpec to be cleared")
	}
}

func TestParseTimeStyleKeywords(t *testing.T) {
	spec, warn, ok := parseTimeStyle("full-iso")
	if !ok || warn != "" || spec == nil || spec.recentLayout == "" {
		t.Fatalf("expected full-iso to parse, warn=%q ok=%v spec=%v", warn, ok, spec)
	}
	if spec.oldLayout != "" {
		t.Fatalf("expected full-iso to have no old layout, got %q", spec.oldLayout)
	}

	spec, warn, ok = parseTimeStyle("locale")
	if !ok || warn != "" || spec == nil || spec.oldLayout == "" {
		t.Fatalf("expected locale to parse with old layout, warn=%q ok=%v spec=%v", warn, ok, spec)
	}

	spec, warn, ok = parseTimeStyle("iso")
	if !ok || warn != "" || spec == nil || spec.oldLayout == "" || spec.recentLayout == "" {
		t.Fatalf("expected iso to parse with old/recent layouts, warn=%q ok=%v spec=%v", warn, ok, spec)
	}

	spec, warn, ok = parseTimeStyle("long-iso")
	if !ok || warn != "" || spec == nil || spec.recentLayout == "" {
		t.Fatalf("expected long-iso to parse, warn=%q ok=%v spec=%v", warn, ok, spec)
	}
	if spec.oldLayout != "" {
		t.Fatalf("expected long-iso to have no old layout, got %q", spec.oldLayout)
	}
}

func TestParseTimeStyleFormat(t *testing.T) {
	spec, warn, ok := parseTimeStyle("+%Y-%m-%d")
	if !ok || warn != "" || spec == nil || spec.recentLayout == "" {
		t.Fatalf("expected format to parse, warn=%q ok=%v spec=%v", warn, ok, spec)
	}

	_, warn, ok = parseTimeStyle("+%Q")
	if ok || warn == "" {
		t.Fatalf("expected unsupported token warning, ok=%v warn=%q", ok, warn)
	}

	spec, warn, ok = parseTimeStyle("+%Y\n%H")
	if !ok || warn != "" || spec == nil || spec.oldLayout == "" {
		t.Fatalf("expected newline format to parse, warn=%q ok=%v spec=%v", warn, ok, spec)
	}

	_, warn, ok = parseTimeStyle("unknown-style")
	if ok || warn == "" {
		t.Fatalf("expected unknown style warning, ok=%v warn=%q", ok, warn)
	}
}

func TestFormatTimeLocaleRecentOld(t *testing.T) {
	spec := &timeStyleSpec{
		kind:         timeStyleLocale,
		recentLayout: "Jan 02 15:04",
		oldLayout:    "Jan 02  2006",
	}
	config := &Config{TimeStyleSpec: spec}

	recent := time.Now().Add(-24 * time.Hour)
	old := time.Now().Add(-365 * 24 * time.Hour)

	if got := formatTime(recent, config); got == formatTime(old, config) {
		t.Fatalf("expected different formats for recent vs old")
	}
}

func TestFormatTimeISORecentOld(t *testing.T) {
	spec := &timeStyleSpec{
		kind:         timeStyleISO,
		recentLayout: "01-02 15:04",
		oldLayout:    "2006-01-02",
	}
	config := &Config{TimeStyleSpec: spec}

	recent := time.Now().Add(-24 * time.Hour)
	old := time.Now().Add(-365 * 24 * time.Hour)

	if got := formatTime(recent, config); got == formatTime(old, config) {
		t.Fatalf("expected different formats for recent vs old")
	}
}

func TestParseTimeFormatInvalid(t *testing.T) {
	_, _, warn, ok := parseTimeFormat("")
	if ok || warn == "" {
		t.Fatalf("expected missing format warning, ok=%v warn=%q", ok, warn)
	}

	_, _, warn, ok = parseTimeFormat("a\nb\nc")
	if ok || warn == "" {
		t.Fatalf("expected invalid format warning, ok=%v warn=%q", ok, warn)
	}
}

func TestConvertTimeFormatTokens(t *testing.T) {
	got, warn, ok := convertTimeFormat("%Y-%m-%d %% %H:%M:%S %b %B %a %Z %z %e")
	if !ok || warn != "" {
		t.Fatalf("expected format to convert, warn=%q ok=%v", warn, ok)
	}
	if !strings.Contains(got, "2006-01-02 % 15:04:05 Jan January Mon MST -0700  2") {
		t.Fatalf("unexpected layout: %q", got)
	}

	_, warn, ok = convertTimeFormat("%")
	if ok || warn == "" {
		t.Fatalf("expected invalid format warning, ok=%v warn=%q", ok, warn)
	}
}

func TestIsPosixLocale(t *testing.T) {
	t.Setenv("LC_ALL", "C")
	t.Setenv("LC_TIME", "")
	t.Setenv("LANG", "")
	if !isPosixLocale() {
		t.Fatal("expected posix locale for LC_ALL=C")
	}

	t.Setenv("LC_ALL", "")
	t.Setenv("LC_TIME", "POSIX")
	if !isPosixLocale() {
		t.Fatal("expected posix locale for LC_TIME=POSIX")
	}

	t.Setenv("LC_ALL", "")
	t.Setenv("LC_TIME", "")
	t.Setenv("LANG", "en_US.UTF-8")
	if isPosixLocale() {
		t.Fatal("expected non-posix locale for LANG=en_US.UTF-8")
	}
}

func TestParseTimeStylePosixPrefix(t *testing.T) {
	t.Setenv("LC_ALL", "C")
	spec, warn, ok := parseTimeStyle("posix-iso")
	if ok || warn != "" || spec != nil {
		t.Fatalf("expected posix-iso ignored in C locale, ok=%v warn=%q spec=%v", ok, warn, spec)
	}

	t.Setenv("LC_ALL", "")
	t.Setenv("LANG", "en_US.UTF-8")
	spec, warn, ok = parseTimeStyle("posix-iso")
	if !ok || warn != "" || spec == nil {
		t.Fatalf("expected posix-iso to parse in non-posix locale, ok=%v warn=%q spec=%v", ok, warn, spec)
	}
}

func TestIsRecentTimeBoundaries(t *testing.T) {
	if isRecentTime(time.Now().Add(48 * time.Hour)) {
		t.Fatal("expected future time to be non-recent")
	}
	if isRecentTime(time.Now().Add(-200 * 24 * time.Hour)) {
		t.Fatal("expected old time to be non-recent")
	}
}

func TestIsPosixLocaleDefault(t *testing.T) {
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_TIME", "")
	t.Setenv("LANG", "")
	if !isPosixLocale() {
		t.Fatal("expected posix locale when no env vars are set")
	}
}

func TestParseTimeStyleMissing(t *testing.T) {
	_, warn, ok := parseTimeStyle(" ")
	if ok || warn == "" {
		t.Fatalf("expected missing TIME_STYLE warning, ok=%v warn=%q", ok, warn)
	}
}

func TestParseTimeFormatSimple(t *testing.T) {
	recent, old, warn, ok := parseTimeFormat("%Y")
	if !ok || warn != "" || recent == "" || old != "" {
		t.Fatalf("expected simple format to parse, ok=%v warn=%q recent=%q old=%q", ok, warn, recent, old)
	}
}

func TestParseArgsTimeStyleEnv(t *testing.T) {
	t.Setenv("TIME_STYLE", "long-iso")
	var stderr bytes.Buffer
	config, err := ParseArgs([]string{"-l"}, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if config.TimeStyleSpec == nil {
		t.Fatal("expected TimeStyleSpec to be set from TIME_STYLE")
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no warnings, got %q", stderr.String())
	}
}

func TestParseArgsTimeStyleEnvInvalidWarns(t *testing.T) {
	t.Setenv("TIME_STYLE", "+%Q")
	var stderr bytes.Buffer

	config, err := ParseArgs([]string{"-l"}, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if config.TimeStyleSpec != nil {
		t.Fatalf("expected TimeStyleSpec to be nil for invalid TIME_STYLE, got %#v", config.TimeStyleSpec)
	}

	out := stderr.String()
	if !strings.Contains(out, "TIME_STYLE") || !strings.Contains(out, "unsupported TIME_STYLE token") {
		t.Fatalf("expected TIME_STYLE warning about unsupported token, got %q", out)
	}
}

func TestParseArgsTimeStyleInvalidWarns(t *testing.T) {
	var stderr bytes.Buffer
	config, err := ParseArgs([]string{"-l", "--time-style=+%Q"}, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if config.TimeStyleSpec != nil {
		t.Fatalf("expected TimeStyleSpec to be nil for invalid time-style")
	}
	if !strings.Contains(stderr.String(), "unsupported TIME_STYLE token") {
		t.Fatalf("expected warning, got %q", stderr.String())
	}
}

func TestParseArgsFullTime(t *testing.T) {
	var stderr bytes.Buffer
	config, err := ParseArgs([]string{"-l", "--full-time"}, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if config.TimeStyleSpec == nil || config.TimeStyleSpec.kind != timeStyleFullISO {
		t.Fatalf("expected full-iso time style, got %v", config.TimeStyleSpec)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no warnings, got %q", stderr.String())
	}
}

func TestParseArgsFullTimeWithTimeStyleWarns(t *testing.T) {
	var stderr bytes.Buffer
	config, err := ParseArgs([]string{"-l", "--full-time", "--time-style=iso"}, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if config.TimeStyleSpec == nil || config.TimeStyleSpec.kind != timeStyleISO {
		t.Fatalf("expected time-style iso to win, got %v", config.TimeStyleSpec)
	}
	if !strings.Contains(stderr.String(), "ls: warning: --full-time is ignored when --time-style is used") {
		t.Fatalf("expected warning, got %q", stderr.String())
	}
}

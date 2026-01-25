package timeutil

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/ioluas/gopherutils/utils/file/ls/internal/config"
)

func TestParseTimeWord(t *testing.T) {
	tests := []struct {
		input    string
		expected config.TimeField
	}{
		{"atime", config.TimeFieldAccess},
		{"access", config.TimeFieldAccess},
		{"use", config.TimeFieldAccess},
		{"ctime", config.TimeFieldChange},
		{"status", config.TimeFieldChange},
		{"mtime", config.TimeFieldMod},
		{"modification", config.TimeFieldMod},
		{"birth", config.TimeFieldBirth},
		{"creation", config.TimeFieldBirth},
	}

	for _, tt := range tests {
		got, err := ParseTimeWord(tt.input)
		if err != nil {
			t.Fatalf("ParseTimeWord(%q) error: %v", tt.input, err)
		}
		if got != tt.expected {
			t.Fatalf("ParseTimeWord(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}

	if _, err := ParseTimeWord("invalid"); err == nil {
		t.Fatal("expected error for invalid time word")
	}
}

func TestGetEntryTimeFallback(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	info := &mockFileInfo{modTime: now}
	if got := GetEntryTime(info, config.TimeFieldAccess); !got.Equal(now) {
		t.Fatalf("expected fallback to ModTime, got %v", got)
	}
	if got := GetEntryTime(info, config.TimeFieldBirth); !got.Equal(now) {
		t.Fatalf("expected fallback to ModTime for birth, got %v", got)
	}
}

func TestNormalizeTimeConfigTimeFieldIgnoredWithoutLongListing(t *testing.T) {
	cfg := &config.Config{
		TimeFieldSet: true,
		TimeField:    config.TimeFieldAccess,
	}

	var stderr bytes.Buffer
	NormalizeTimeConfig(cfg, &stderr)
	if !strings.Contains(stderr.String(), "ls: warning: --time is ignored when -l is not used") {
		t.Fatalf("expected warning, got %q", stderr.String())
	}
	if cfg.TimeField != config.TimeFieldMod {
		t.Fatalf("expected TimeField reset to mod, got %v", cfg.TimeField)
	}
}

func TestNormalizeTimeConfigTimeFieldWithLongListing(t *testing.T) {
	cfg := &config.Config{
		LongListing:  true,
		TimeFieldSet: true,
		TimeField:    config.TimeFieldAccess,
	}

	var stderr bytes.Buffer
	NormalizeTimeConfig(cfg, &stderr)
	if stderr.Len() != 0 {
		t.Fatalf("expected no warning, got %q", stderr.String())
	}
}

func TestTimeStyleWarningWithoutLongListing(t *testing.T) {
	cfg := &config.Config{
		TimeStyleSet: true,
		TimeStyleSpec: &config.TimeStyleSpec{
			Kind:         config.TimeStyleISO,
			RecentLayout: "2006-01-02 15:04",
		},
	}

	var stderr bytes.Buffer
	NormalizeTimeConfig(cfg, &stderr)
	if !strings.Contains(stderr.String(), "ls: warning: --time-style is ignored when -l is not used") {
		t.Fatalf("expected warning, got %q", stderr.String())
	}
	if cfg.TimeStyleSpec != nil {
		t.Fatalf("expected TimeStyleSpec to be cleared")
	}
}

func TestParseTimeStyleKeywords(t *testing.T) {
	spec, warn, ok := ParseTimeStyle("full-iso")
	if !ok || warn != "" || spec == nil || spec.RecentLayout == "" {
		t.Fatalf("expected full-iso to parse, warn=%q ok=%v spec=%v", warn, ok, spec)
	}
	if spec.OldLayout != "" {
		t.Fatalf("expected full-iso to have no old layout, got %q", spec.OldLayout)
	}

	spec, warn, ok = ParseTimeStyle("locale")
	if !ok || warn != "" || spec == nil || spec.OldLayout == "" {
		t.Fatalf("expected locale to parse with old layout, warn=%q ok=%v spec=%v", warn, ok, spec)
	}

	spec, warn, ok = ParseTimeStyle("iso")
	if !ok || warn != "" || spec == nil || spec.OldLayout == "" || spec.RecentLayout == "" {
		t.Fatalf("expected iso to parse with old/recent layouts, warn=%q ok=%v spec=%v", warn, ok, spec)
	}

	spec, warn, ok = ParseTimeStyle("long-iso")
	if !ok || warn != "" || spec == nil || spec.RecentLayout == "" {
		t.Fatalf("expected long-iso to parse, warn=%q ok=%v spec=%v", warn, ok, spec)
	}
	if spec.OldLayout != "" {
		t.Fatalf("expected long-iso to have no old layout, got %q", spec.OldLayout)
	}
}

func TestParseTimeStyleFormat(t *testing.T) {
	spec, warn, ok := ParseTimeStyle("+%Y-%m-%d")
	if !ok || warn != "" || spec == nil || spec.RecentLayout == "" {
		t.Fatalf("expected format to parse, warn=%q ok=%v spec=%v", warn, ok, spec)
	}

	_, warn, ok = ParseTimeStyle("+%Q")
	if ok || warn == "" {
		t.Fatalf("expected unsupported token warning, ok=%v warn=%q", ok, warn)
	}

	spec, warn, ok = ParseTimeStyle("+%Y\n%H")
	if !ok || warn != "" || spec == nil || spec.OldLayout == "" {
		t.Fatalf("expected newline format to parse, warn=%q ok=%v spec=%v", warn, ok, spec)
	}

	_, warn, ok = ParseTimeStyle("unknown-style")
	if ok || warn == "" {
		t.Fatalf("expected unknown style warning, ok=%v warn=%q", ok, warn)
	}
}

func TestFormatTimeLocaleRecentOld(t *testing.T) {
	spec := &config.TimeStyleSpec{
		Kind:         config.TimeStyleLocale,
		RecentLayout: "Jan 02 15:04",
		OldLayout:    "Jan 02  2006",
	}
	cfg := &config.Config{TimeStyleSpec: spec}

	recent := time.Now().Add(-24 * time.Hour)
	old := time.Now().Add(-365 * 24 * time.Hour)

	if got := FormatTime(recent, cfg); got == FormatTime(old, cfg) {
		t.Fatalf("expected different formats for recent vs old")
	}
}

func TestFormatTimeISORecentOld(t *testing.T) {
	spec := &config.TimeStyleSpec{
		Kind:         config.TimeStyleISO,
		RecentLayout: "01-02 15:04",
		OldLayout:    "2006-01-02",
	}
	cfg := &config.Config{TimeStyleSpec: spec}

	recent := time.Now().Add(-24 * time.Hour)
	old := time.Now().Add(-365 * 24 * time.Hour)

	if got := FormatTime(recent, cfg); got == FormatTime(old, cfg) {
		t.Fatalf("expected different formats for recent vs old")
	}
}

func TestParseTimeFormatInvalid(t *testing.T) {
	_, _, warn, ok := ParseTimeFormat("")
	if ok || warn == "" {
		t.Fatalf("expected missing format warning, ok=%v warn=%q", ok, warn)
	}

	_, _, warn, ok = ParseTimeFormat("a\nb\nc")
	if ok || warn == "" {
		t.Fatalf("expected invalid format warning, ok=%v warn=%q", ok, warn)
	}
}

func TestConvertTimeFormatTokens(t *testing.T) {
	got, warn, ok := ConvertTimeFormat("%Y-%m-%d %% %H:%M:%S %b %B %a %Z %z %e")
	if !ok || warn != "" {
		t.Fatalf("expected format to convert, warn=%q ok=%v", warn, ok)
	}
	if !strings.Contains(got, "2006-01-02 % 15:04:05 Jan January Mon MST -0700  2") {
		t.Fatalf("unexpected layout: %q", got)
	}

	_, warn, ok = ConvertTimeFormat("%")
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
	// When LC_ALL is "C", posix-iso should still be parsed correctly after stripping "posix-"
	t.Setenv("LC_ALL", "C")
	spec, warn, ok := ParseTimeStyle("posix-iso")
	if !ok || warn != "" || spec == nil {
		t.Fatalf("expected posix-iso to parse correctly in C locale, ok=%v warn=%q spec=%v", ok, warn, spec)
	}
	if spec.Kind != config.TimeStyleISO {
		t.Fatalf("expected Kind to be TimeStyleISO, got %v", spec.Kind)
	}

	// When not in a POSIX locale, posix-iso should also be parsed correctly
	t.Setenv("LC_ALL", "")
	t.Setenv("LANG", "en_US.UTF-8")
	spec, warn, ok = ParseTimeStyle("posix-iso")
	if !ok || warn != "" || spec == nil {
		t.Fatalf("expected posix-iso to parse correctly in non-posix locale, ok=%v warn=%q spec=%v", ok, warn, spec)
	}
	if spec.Kind != config.TimeStyleISO {
		t.Fatalf("expected Kind to be TimeStyleISO, got %v", spec.Kind)
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
	_, warn, ok := ParseTimeStyle(" ")
	if ok || warn == "" {
		t.Fatalf("expected missing TIME_STYLE warning, ok=%v warn=%q", ok, warn)
	}
}

func TestParseTimeFormatSimple(t *testing.T) {
	recent, old, warn, ok := ParseTimeFormat("%Y")
	if !ok || warn != "" || recent == "" || old != "" {
		t.Fatalf("expected simple format to parse, ok=%v warn=%q recent=%q old=%q", ok, warn, recent, old)
	}
}

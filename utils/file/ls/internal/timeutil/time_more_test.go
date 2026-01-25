package timeutil

import (
	"bytes"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/ioluas/gopherutils/utils/file/ls/internal/config"
)

func TestNormalizeTimeConfigFullTimeIgnoredWithoutLongListing(t *testing.T) {
	cfg := &config.Config{
		FullTime: true,
	}

	var stderr bytes.Buffer
	NormalizeTimeConfig(cfg, &stderr)
	if !strings.Contains(stderr.String(), "ls: warning: --full-time is ignored when -l is not used") {
		t.Fatalf("expected warning, got %q", stderr.String())
	}
	if cfg.TimeStyleSpec != nil {
		t.Fatalf("expected TimeStyleSpec to be cleared")
	}
}

func TestGetEntryTimeWithMock(t *testing.T) {
	now := time.Now()
	atime := now.Add(-time.Hour)
	ctime := now.Add(-2 * time.Hour)
	birthtime := now.Add(-3 * time.Hour)

	stat := &syscall.Stat_t{
		Atim: timespecToStat(atime),
		Ctim: timespecToStat(ctime),
		Mtim: timespecToStat(now),
	}

	// Mock stat with birthtime if supported
	if _, ok := statBirthtime(stat); ok {
		setBirthtime(stat, birthtime)
	}

	info := &mockFileInfo{
		modTime: now,
		sys:     stat,
	}

	if got := GetEntryTime(info, config.TimeFieldMod); !got.Equal(now) {
		t.Errorf("Expected mod time %v, got %v", now, got)
	}
	if got := GetEntryTime(info, config.TimeFieldAccess); !got.Equal(atime) {
		t.Errorf("Expected access time %v, got %v", atime, got)
	}
	if got := GetEntryTime(info, config.TimeFieldChange); !got.Equal(ctime) {
		t.Errorf("Expected change time %v, got %v", ctime, got)
	}
	if _, ok := statBirthtime(stat); ok {
		if got := GetEntryTime(info, config.TimeFieldBirth); !got.Equal(birthtime) {
			t.Errorf("Expected birth time %v, got %v", birthtime, got)
		}
	}
}

func TestFormatTimeNilStyle(t *testing.T) {
	cfg := &config.Config{}
	now := time.Now()
	formattedTime := FormatTime(now, cfg)
	expected := now.Format("Jan 02 15:04")
	if formattedTime != expected {
		t.Fatalf("Expected format %q, got %q", expected, formattedTime)
	}
}

func TestParseTimeFormatErrors(t *testing.T) {
	// Test case where the first line of a two-line format is invalid
	_, _, warn, ok := ParseTimeFormat("%Y\n%Q")
	if ok || warn == "" {
		t.Fatalf("expected unsupported token warning, ok=%v warn=%q", ok, warn)
	}

	// Test case where a single line format is invalid
	_, _, warn, ok = ParseTimeFormat("%Q")
	if ok || warn == "" {
		t.Fatalf("expected unsupported token warning for single line, ok=%v warn=%q", ok, warn)
	}
}

func TestConvertTimeFormatUnsupportedToken(t *testing.T) {
	_, warn, ok := ConvertTimeFormat("%X")
	if ok || !strings.Contains(warn, "unsupported TIME_STYLE token") {
		t.Fatalf("expected unsupported token warning, got warn=%q ok=%v", warn, ok)
	}
}

// Helper functions for setting up mock stat times
func timespecToStat(t time.Time) syscall.Timespec {
	return syscall.NsecToTimespec(t.UnixNano())
}

func setBirthtime(stat *syscall.Stat_t, t time.Time) {
	// This is a simplified version for testing purposes and may not be
	// portable across all platforms.
	stat.Ctim = timespecToStat(t)
}

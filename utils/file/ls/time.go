package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"syscall"
	"time"
)

func parseTimeWord(word string) (timeField, error) {
	switch strings.ToLower(word) {
	case "atime", "access", "use":
		return timeFieldAccess, nil
	case "ctime", "status":
		return timeFieldChange, nil
	case "mtime", "modification":
		return timeFieldMod, nil
	case "birth", "creation":
		return timeFieldBirth, nil
	default:
		return timeFieldMod, fmt.Errorf("invalid --time value: %s", word)
	}
}

func parseTimeStyle(raw string) (*timeStyleSpec, string, bool) {
	style := strings.TrimSpace(raw)
	if style == "" {
		return nil, "missing TIME_STYLE", false
	}
	if strings.HasPrefix(style, "posix-") {
		if isPosixLocale() {
			return nil, "", false
		}
		style = strings.TrimPrefix(style, "posix-")
	}

	switch style {
	case "full-iso":
		return &timeStyleSpec{
			kind:         timeStyleFullISO,
			recentLayout: "2006-01-02 15:04:05.000000000 -0700",
		}, "", true
	case "long-iso":
		return &timeStyleSpec{
			kind:         timeStyleLongISO,
			recentLayout: "2006-01-02 15:04",
		}, "", true
	case "iso":
		return &timeStyleSpec{
			kind:         timeStyleISO,
			recentLayout: "01-02 15:04",
			oldLayout:    "2006-01-02",
		}, "", true
	case "locale":
		return &timeStyleSpec{
			kind:         timeStyleLocale,
			recentLayout: "Jan 02 15:04",
			oldLayout:    "Jan 02  2006",
		}, "", true
	default:
		if strings.HasPrefix(style, "+") {
			recent, old, warn, ok := parseTimeFormat(style[1:])
			if !ok {
				return nil, warn, false
			}
			return &timeStyleSpec{
				kind:         timeStyleCustom,
				recentLayout: recent,
				oldLayout:    old,
			}, warn, warn == ""
		}
		return nil, fmt.Sprintf("unknown TIME_STYLE %q", style), false
	}
}

func parseTimeFormat(format string) (string, string, string, bool) {
	if format == "" {
		return "", "", "missing TIME_STYLE format", false
	}
	parts := strings.Split(format, "\n")
	if len(parts) > 2 {
		return "", "", "invalid TIME_STYLE format", false
	}
	recent, warn, ok := convertTimeFormat(parts[len(parts)-1])
	if !ok {
		return "", "", warn, false
	}
	old := ""
	if len(parts) == 2 {
		old, warn, ok = convertTimeFormat(parts[0])
		if !ok {
			return "", "", warn, false
		}
	}
	return recent, old, "", true
}

func convertTimeFormat(format string) (string, string, bool) {
	var b strings.Builder
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			b.WriteByte(format[i])
			continue
		}
		if i+1 >= len(format) {
			return "", "invalid TIME_STYLE format", false
		}
		i++
		switch format[i] {
		case '%':
			b.WriteByte('%')
		case 'Y':
			b.WriteString("2006")
		case 'y':
			b.WriteString("06")
		case 'm':
			b.WriteString("01")
		case 'd':
			b.WriteString("02")
		case 'e':
			b.WriteString(" 2")
		case 'H':
			b.WriteString("15")
		case 'M':
			b.WriteString("04")
		case 'S':
			b.WriteString("05")
		case 'b':
			b.WriteString("Jan")
		case 'B':
			b.WriteString("January")
		case 'a':
			b.WriteString("Mon")
		case 'Z':
			b.WriteString("MST")
		case 'z':
			b.WriteString("-0700")
		default:
			return "", fmt.Sprintf("unsupported TIME_STYLE token %q", "%"+string(format[i])), false
		}
	}
	return b.String(), "", true
}

func isPosixLocale() bool {
	for _, key := range []string{"LC_ALL", "LC_TIME", "LANG"} {
		if value := os.Getenv(key); value != "" {
			return value == "C" || value == "POSIX"
		}
	}
	return true
}

func formatTime(t time.Time, config *Config) string {
	if config.TimeStyleSpec == nil {
		return t.Format("Jan 02 15:04")
	}
	layout := config.TimeStyleSpec.recentLayout
	if config.TimeStyleSpec.oldLayout != "" && !isRecentTime(t) {
		layout = config.TimeStyleSpec.oldLayout
	}
	return t.Format(layout)
}

func isRecentTime(t time.Time) bool {
	now := time.Now()
	if t.After(now.Add(24 * time.Hour)) {
		return false
	}
	return t.After(now.Add(-180 * 24 * time.Hour))
}

func getEntryTime(info fs.FileInfo, field timeField) time.Time {
	if field == timeFieldMod {
		return info.ModTime()
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return info.ModTime()
	}
	switch field {
	case timeFieldAccess:
		if t := statAtime(stat); !t.IsZero() {
			return t
		}
	case timeFieldChange:
		if t := statCtime(stat); !t.IsZero() {
			return t
		}
	case timeFieldBirth:
		if t, ok := statBirthtime(stat); ok && !t.IsZero() {
			return t
		}
	}
	return info.ModTime()
}

func normalizeTimeConfig(config *Config, stderr io.Writer) {
	// Warn if --time is used without -l
	if config.TimeFieldSet && !config.LongListing {
		_, _ = fmt.Fprintf(stderr, "ls: warning: --time is ignored when -l is not used\n")
		config.TimeField = timeFieldMod
	}

	// Warn if --time-style is used without -l
	if config.TimeStyleSet && !config.LongListing {
		_, _ = fmt.Fprintf(stderr, "ls: warning: --time-style is ignored when -l is not used\n")
		config.TimeStyleSpec = nil
	}

	// Warn if --full-time is used without -l
	if config.FullTime && !config.LongListing {
		_, _ = fmt.Fprintf(stderr, "ls: warning: --full-time is ignored when -l is not used\n")
		config.TimeStyleSpec = nil
	}
}

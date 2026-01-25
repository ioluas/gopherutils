package timeutil

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/ioluas/gopherutils/utils/file/ls/internal/config"
)

func ParseTimeWord(word string) (config.TimeField, error) {
	switch strings.ToLower(word) {
	case "atime", "access", "use":
		return config.TimeFieldAccess, nil
	case "ctime", "status":
		return config.TimeFieldChange, nil
	case "mtime", "modification":
		return config.TimeFieldMod, nil
	case "birth", "creation":
		return config.TimeFieldBirth, nil
	default:
		return config.TimeFieldMod, fmt.Errorf("invalid --time value: %s", word)
	}
}

func ParseTimeStyle(raw string) (*config.TimeStyleSpec, string, bool) {
	style := strings.TrimSpace(raw)
	if style == "" {
		return nil, "missing TIME_STYLE", false
	}
	style = strings.TrimPrefix(style, "posix-")

	switch style {
	case "full-iso":
		return &config.TimeStyleSpec{
			Kind:         config.TimeStyleFullISO,
			RecentLayout: "2006-01-02 15:04:05.000000000 -0700",
		}, "", true
	case "long-iso":
		return &config.TimeStyleSpec{
			Kind:         config.TimeStyleLongISO,
			RecentLayout: "2006-01-02 15:04",
		}, "", true
	case "iso":
		return &config.TimeStyleSpec{
			Kind:         config.TimeStyleISO,
			RecentLayout: "01-02 15:04",
			OldLayout:    "2006-01-02",
		}, "", true
	case "locale":
		return &config.TimeStyleSpec{
			Kind:         config.TimeStyleLocale,
			RecentLayout: "Jan 02 15:04",
			OldLayout:    "Jan 02  2006",
		}, "", true
	default:
		if strings.HasPrefix(style, "+") {
			recent, old, warn, ok := ParseTimeFormat(style[1:])
			if !ok {
				return nil, warn, false
			}
			return &config.TimeStyleSpec{
				Kind:         config.TimeStyleCustom,
				RecentLayout: recent,
				OldLayout:    old,
			}, warn, warn == ""
		}
		return nil, fmt.Sprintf("unknown TIME_STYLE %q", style), false
	}
}

func ParseTimeFormat(format string) (string, string, string, bool) {
	if format == "" {
		return "", "", "missing TIME_STYLE format", false
	}
	parts := strings.Split(format, "\n")
	if len(parts) > 2 {
		return "", "", "invalid TIME_STYLE format", false
	}
	recent, warn, ok := ConvertTimeFormat(parts[len(parts)-1])
	if !ok {
		return "", "", warn, false
	}
	old := ""
	if len(parts) == 2 {
		old, warn, ok = ConvertTimeFormat(parts[0])
		if !ok {
			return "", "", warn, false
		}
	}
	return recent, old, "", true
}

func ConvertTimeFormat(format string) (string, string, bool) {
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

func FormatTime(t time.Time, config *config.Config) string {
	if config.TimeStyleSpec == nil {
		return t.Format("Jan 02 15:04")
	}
	layout := config.TimeStyleSpec.RecentLayout
	if config.TimeStyleSpec.OldLayout != "" && !isRecentTime(t) {
		layout = config.TimeStyleSpec.OldLayout
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

func GetEntryTime(info fs.FileInfo, field config.TimeField) time.Time {
	if field == config.TimeFieldMod {
		return info.ModTime()
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return info.ModTime()
	}
	switch field {
	case config.TimeFieldAccess:
		if t := statAtime(stat); !t.IsZero() {
			return t
		}
	case config.TimeFieldChange:
		if t := statCtime(stat); !t.IsZero() {
			return t
		}
	case config.TimeFieldBirth:
		if t, ok := statBirthtime(stat); ok && !t.IsZero() {
			return t
		}
	}
	return info.ModTime()
}

func NormalizeTimeConfig(cfg *config.Config, stderr io.Writer) {
	// Warn if --time is used without -l or -t
	if cfg.TimeFieldSet && !cfg.LongListing && !cfg.SortTime {
		_, _ = fmt.Fprintf(stderr, "ls: warning: --time is ignored when -l is not used\n")
		cfg.TimeField = config.TimeFieldMod
	}

	// Warn if --time-style is used without -l
	if cfg.TimeStyleSet && !cfg.LongListing {
		_, _ = fmt.Fprintf(stderr, "ls: warning: --time-style is ignored when -l is not used\n")
		cfg.TimeStyleSpec = nil
	}

	// Warn if --full-time is used without -l
	if cfg.FullTime && !cfg.LongListing {
		_, _ = fmt.Fprintf(stderr, "ls: warning: --full-time is ignored when -l is not used\n")
		cfg.TimeStyleSpec = nil
	}
}

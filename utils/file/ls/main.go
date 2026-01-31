package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"unicode"

	"github.com/ioluas/gopherutils/utils/file/ls/internal/config"
	"github.com/ioluas/gopherutils/utils/file/ls/internal/display"
	"github.com/ioluas/gopherutils/utils/file/ls/internal/entry"
	"github.com/ioluas/gopherutils/utils/file/ls/internal/parse"
	"github.com/ioluas/gopherutils/utils/file/ls/internal/timeutil"

	"github.com/spf13/pflag"
)

func main() {
	os.Exit(Execute(os.Args[1:], os.Stdout, os.Stderr))
}

// Execute is the entry point for the ls utility, extracted from main for testing.
func Execute(args []string, stdout, stderr io.Writer) int {
	cfg, err := parse.ParseArgs(args, stderr)
	if err != nil {
		if errors.Is(err, pflag.ErrHelp) {
			return 0
		}
		_, _ = fmt.Fprintf(stderr, "ls: %v\n", err)
		return 1
	}

	exitCode := 0
	for i, dir := range cfg.Directories {
		// Print directory name if multiple directories are listed
		if len(cfg.Directories) > 1 {
			if i > 0 {
				_, _ = fmt.Fprintln(stdout) // Blank line between directory listings
			}
			_, _ = fmt.Fprintf(stdout, "%s:\n", dir)
		}

		currentExitCode := run(dir, cfg, stdout, stderr)
		if currentExitCode != 0 {
			exitCode = currentExitCode
		}
	}
	return exitCode
}

// run executes the ls logic for a given configuration
func run(path string, cfg *config.Config, stdout, stderr io.Writer) int {
	timeutil.NormalizeTimeConfig(cfg, stderr)
	if cfg.ListDirectory {
		// List the directory itself, not its contents
		// Check if the path exists first
		info, err := os.Lstat(path)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "ls: cannot access '%s': %v\n", path, err)
			return 2
		}

		entry := entry.NewCachedDirEntry(entry.NewDirEntryWrapper(path, path, true, info, nil), info)
		entries := []os.DirEntry{entry}

		if cfg.LongListing {
			if display.PrintLongList(stdout, stderr, entries, cfg, false) {
				return 2
			}
		} else {
			name := entry.Name()
			style := cfg.QuotingStyle
			if style == config.QuotingStyleLiteral && cfg.Escape {
				style = config.QuotingStyleEscape
			}
			name = display.Quote(name, style)
			display.PrintEntries(stdout, []string{name}, cfg)
		}
		return 0
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ls: cannot access '%s': %v\n", path, err)
		return 2
	}

	var filtered []os.DirEntry

	// If ShowAll is true, explicitly add "." and ".."
	if cfg.ShowAll {
		filtered = append(filtered, &entry.CachedDirEntry{DirEntry: entry.NewDirEntryWrapper(".", path, false, nil, nil)})
		filtered = append(filtered, &entry.CachedDirEntry{DirEntry: entry.NewDirEntryWrapper("..", path, false, nil, nil)})
	}

	for _, dirEntry := range entries {
		name := dirEntry.Name()

		shouldInclude := true

		// If it's a dotfile (excluding . and .. now that we handle them separately)
		if strings.HasPrefix(name, ".") {
			if cfg.ShowAll {
				shouldInclude = true
			} else if cfg.ShowAlmostAll {
				// -A is active, include dotfiles but not '.' or '..'
				// We only reach here for dotfiles other than '.' or '..', so include them.
				shouldInclude = true
			} else {
				// No -a or -A, hide dotfiles
				shouldInclude = false
			}
		}
		// If it's not a dotfile, shouldInclude remains true
		if shouldInclude && cfg.IgnoreBackups && strings.HasSuffix(name, "~") {
			shouldInclude = false
		}

		if shouldInclude {
			filtered = append(filtered, &entry.CachedDirEntry{DirEntry: dirEntry})
		}
	}

	hadError := false
	if cfg.SortTime {
		for i, e := range filtered {
			ce, ok := e.(*entry.CachedDirEntry)
			if !ok {
				ce = &entry.CachedDirEntry{DirEntry: e}
				filtered[i] = ce
			}
			info, err := ce.Info()
			if err != nil {
				hadError = true
				if !cfg.LongListing {
					_, _ = fmt.Fprintf(stderr, "ls: cannot access '%s': %v\n", ce.Name(), err)
				}
				continue
			}
			t := timeutil.GetEntryTime(info, cfg.TimeField)
			ce.Time = &t
		}

		sort.Slice(filtered, func(i, j int) bool {
			return entry.LessByTime(filtered[i], filtered[j])
		})
	} else {
		// Sort filtered entries by name for consistent output
		sort.Slice(filtered, func(i, j int) bool {
			n1 := filtered[i].Name()
			n2 := filtered[j].Name()
			norm1 := normalizeName(n1)
			norm2 := normalizeName(n2)
			if norm1 != norm2 {
				return norm1 < norm2
			}
			return n1 < n2
		})
	}

	// Warn if -h is used without -l
	if (cfg.HumanReadable || cfg.SI) && !cfg.LongListing {
		if cfg.HumanReadable && cfg.SI {
			_, _ = fmt.Fprintf(stderr, "ls: warning: options -h and --si are ignored when -l is not used\n")
		} else {
			flag := "-h"
			if cfg.SI {
				flag = "--si"
			}
			_, _ = fmt.Fprintf(stderr, "ls: warning: option %s is ignored when -l is not used\n", flag)
		}
	}

	// Warn if --block-size is used without -l
	if cfg.BlockSize != nil && !cfg.LongListing {
		_, _ = fmt.Fprintf(stderr, "ls: warning: option --block-size is ignored when -l is not used\n")
	}

	// Warn if --no-group is used without -l
	if cfg.NoGroup && !cfg.LongListing {
		_, _ = fmt.Fprintf(stderr, "ls: warning: --no-group is ignored when -l is not used\n")
	}

	// Warn if --author is used without -l
	if cfg.ShowAuthor && !cfg.LongListing {
		_, _ = fmt.Fprintf(stderr, "ls: warning: --author is ignored when -l is not used\n")
	}

	if cfg.LongListing {
		printTotal := cfg.Dired
		if display.PrintLongList(stdout, stderr, filtered, cfg, printTotal) {
			hadError = true
		}
	} else {
		names := make([]string, len(filtered))
		for i, e := range filtered {
			name := e.Name()
			style := cfg.QuotingStyle
			if style == config.QuotingStyleLiteral && cfg.Escape {
				style = config.QuotingStyleEscape
			}
			name = display.Quote(name, style)
			names[i] = name
		}
		display.PrintEntries(stdout, names, cfg)
	}

	if hadError {
		return 2
	}
	return 0
}

// normalizeName prepares a filename for sorting by removing punctuation
// and converting to lowercase, supporting Unicode characters (letters, digits, marks).
// This mimics GNU ls's locale-aware sorting behavior where punctuation is ignored.
func normalizeName(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		// Keep letters, digits, and combining marks (e.g., accents)
		// This supports international characters like é, Å, ß, CJK, etc.
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r) {
			b.WriteRune(r)
		}
	}
	return strings.ToLower(b.String())
}

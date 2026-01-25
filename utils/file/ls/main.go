package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/spf13/pflag"
	"golang.org/x/term"
)

func main() {
	os.Exit(Execute(os.Args[1:], os.Stdout, os.Stderr))
}

// Execute is the entry point for the ls utility, extracted from main for testing.
func Execute(args []string, stdout, stderr io.Writer) int {
	config, err := ParseArgs(args, stderr)
	if err != nil {
		if errors.Is(err, pflag.ErrHelp) {
			return 0
		}
		_, _ = fmt.Fprintf(stderr, "ls: %v\n", err)
		return 1
	}

	exitCode := 0
	for i, dir := range config.Directories {
		// Print directory name if multiple directories are listed
		if len(config.Directories) > 1 {
			if i > 0 {
				_, _ = fmt.Fprintln(stdout) // Blank line between directory listings
			}
			_, _ = fmt.Fprintf(stdout, "%s:\n", dir)
		}

		currentExitCode := run(dir, config, stdout, stderr)
		if currentExitCode != 0 {
			exitCode = currentExitCode
		}
	}
	return exitCode
}

// run executes the ls logic for a given configuration
func run(path string, config *Config, stdout, stderr io.Writer) int {
	normalizeTimeConfig(config, stderr)
	if config.ListDirectory {
		// List the directory itself, not its contents
		// Check if the path exists first
		info, err := os.Lstat(path)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "ls: cannot access '%s': %v\n", path, err)
			return 2
		}

		entry := &cachedDirEntry{
			DirEntry: &dirEntryWrapper{name: path, dirPath: path, isRoot: true},
			info:     info,
		}
		entries := []os.DirEntry{entry}

		if config.LongListing {
			printLongList(stdout, entries, config)
		} else {
			name := entry.Name()
			if config.Escape {
				name = quoteName(name)
			}
			printEntries(stdout, []string{name})
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
	if config.ShowAll {
		filtered = append(filtered, &cachedDirEntry{DirEntry: &dirEntryWrapper{name: ".", dirPath: path}})
		filtered = append(filtered, &cachedDirEntry{DirEntry: &dirEntryWrapper{name: "..", dirPath: path}})
	}

	for _, entry := range entries {
		name := entry.Name()

		shouldInclude := true

		// If it's a dotfile (excluding . and .. now that we handle them separately)
		if strings.HasPrefix(name, ".") {
			if config.ShowAll {
				shouldInclude = true
			} else if config.ShowAlmostAll {
				// -A is active, include dotfiles but not '.' or '..'
				// We only reach here for dotfiles other than '.' or '..', so include them.
				shouldInclude = true
			} else {
				// No -a or -A, hide dotfiles
				shouldInclude = false
			}
		}
		// If it's not a dotfile, shouldInclude remains true
		if shouldInclude && config.IgnoreBackups && strings.HasSuffix(name, "~") {
			shouldInclude = false
		}

		if shouldInclude {
			filtered = append(filtered, &cachedDirEntry{DirEntry: entry})
		}
	}

	if config.SortTime {
		for _, e := range filtered {
			ce := e.(*cachedDirEntry)
			info, err := ce.Info()
			if err != nil {
				continue
			}
			ce.info = info
			t := getEntryTime(info, config.TimeField)
			ce.time = &t
		}

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
	} else {
		// Sort filtered entries by name for consistent output
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].Name() < filtered[j].Name()
		})
	}

	// Warn if -h is used without -l
	if (config.HumanReadable || config.SI) && !config.LongListing {
		if config.HumanReadable && config.SI {
			_, _ = fmt.Fprintf(stderr, "ls: warning: options -h and --si are ignored when -l is not used\n")
		} else {
			flag := "-h"
			if config.SI {
				flag = "--si"
			}
			_, _ = fmt.Fprintf(stderr, "ls: warning: option %s is ignored when -l is not used\n", flag)
		}
	}

	// Warn if --block-size is used without -l
	if config.BlockSize != nil && !config.LongListing {
		_, _ = fmt.Fprintf(stderr, "ls: warning: option --block-size is ignored when -l is not used\n")
	}

	// Warn if --no-group is used without -l
	if config.NoGroup && !config.LongListing {
		_, _ = fmt.Fprintf(stderr, "ls: warning: --no-group is ignored when -l is not used\n")
	}

	// Warn if --author is used without -l
	if config.ShowAuthor && !config.LongListing {
		_, _ = fmt.Fprintf(stderr, "ls: warning: --author is ignored when -l is not used\n")
	}

	if config.LongListing {
		printLongList(stdout, filtered, config)
	} else {
		names := make([]string, len(filtered))
		for i, e := range filtered {
			name := e.Name()
			if config.Escape {
				name = quoteName(name)
			}
			names[i] = name
		}
		printEntries(stdout, names)
	}
	return 0
}

// For testing purposes
var isTerminalFunc = func(fd int) bool {
	return term.IsTerminal(fd)
}

var getTermSizeFunc = func(fd int) (width, height int, err error) {
	return term.GetSize(fd)
}

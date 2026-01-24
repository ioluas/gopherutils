package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/ioluas/gopherutils/internal/fsutil"
	"golang.org/x/term"

	"github.com/spf13/pflag"
)

// dirEntryWrapper implements os.DirEntry for "." and "..", or for -d flag
type dirEntryWrapper struct {
	name    string
	dirPath string // The path of the directory whose entries are being listed, or the path to the entry itself
	isRoot  bool   // If true, Info() should stat dirPath directly
}

func (d *dirEntryWrapper) Name() string {
	return d.name
}

func (d *dirEntryWrapper) IsDir() bool {
	info, err := d.Info()
	if err != nil {
		return false
	}
	return info.IsDir()
}

func (d *dirEntryWrapper) Type() fs.FileMode {
	info, err := d.Info()
	if err != nil {
		return 0
	}
	return info.Mode().Type()
}

func (d *dirEntryWrapper) Info() (fs.FileInfo, error) {
	if d.isRoot {
		return os.Lstat(d.dirPath)
	}
	targetPath := d.dirPath
	if d.name == ".." {
		targetPath = filepath.Clean(filepath.Join(d.dirPath, ".."))
	}
	return os.Stat(targetPath)
}

type Config struct {
	Directories   []string
	ShowAll       bool // -a: do not ignore entries starting with .
	ShowAlmostAll bool // -A: do not list implied . and ..
	LongListing   bool // -l: use a long listing format
	HumanReadable bool // -h: with -l, print sizes in human readable format
	SI            bool // --si: with -l, print sizes in powers of 1000 not 1024
	ShowAuthor    bool // --author: with -l, print the author of each file
	Escape        bool // -b, --escape: print C-style escapes for nongraphic characters
	IgnoreBackups bool // -B, --ignore-backups: do not list implied entries ending with ~
	ListDirectory bool // -d, --directory: list directories themselves, not their contents
}

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

// ParseArgs parses command-line arguments using pflag and returns a Config.
func ParseArgs(args []string, stderr io.Writer) (*Config, error) {
	config := &Config{}
	var showHelp bool

	// Create a new FlagSet for ls
	flagSet := pflag.NewFlagSet("ls", pflag.ContinueOnError)

	// Suppress default error output, we'll handle errors ourselves
	flagSet.SetOutput(io.Discard)

	// Define flags with both short and long forms
	flagSet.BoolVarP(&config.ShowAll, "all", "a", false, "do not ignore entries starting with .")
	flagSet.BoolVarP(&config.ShowAlmostAll, "almost-all", "A", false, "do not list implied . and ..")
	flagSet.BoolVarP(&config.LongListing, "long", "l", false, "use a long listing format")
	flagSet.BoolVarP(&config.HumanReadable, "human-readable", "h", false, "with -l, print sizes in human readable format (e.g., 1K 234M 2G)")
	flagSet.BoolVar(&config.SI, "si", false, "with -l, print sizes in powers of 1000 not 1024")
	flagSet.BoolVar(&config.ShowAuthor, "author", false, "with -l, print the author of each file. Note: Due to OS/filesystem limitations, the author is typically the same as the owner.")
	flagSet.BoolVarP(&config.Escape, "escape", "b", false, "print C-style escapes for nongraphic characters")
	flagSet.BoolVarP(&config.IgnoreBackups, "ignore-backups", "B", false, "do not list implied entries ending with ~")
	flagSet.BoolVarP(&config.ListDirectory, "directory", "d", false, "list directories themselves, not their contents")
	flagSet.BoolVarP(&showHelp, "help", "?", false, "display this help and exit")

	// Parse the arguments
	err := flagSet.Parse(args)
	if showHelp {
		err = pflag.ErrHelp
	}

	if err != nil {
		if errors.Is(err, pflag.ErrHelp) {
			flagSet.SetOutput(stderr)
			_, _ = fmt.Fprintf(stderr, "Usage: ls [OPTION]... [FILE]...\n")
			_, _ = fmt.Fprintf(stderr, "List information about the FILEs (the current directory by default).\n\n")
			_, _ = fmt.Fprintf(stderr, "Options:\n")
			flagSet.PrintDefaults()
			return nil, pflag.ErrHelp
		}
		return nil, err
	}

	// Get remaining arguments after flags (the directory path)
	remainingArgs := flagSet.Args()

	if len(remainingArgs) == 0 {
		// No directory specified, use the current directory
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("error getting current directory: %v", err)
		}
		config.Directories = []string{cwd}
	} else {
		// Use all remaining arguments as directories
		config.Directories = remainingArgs
	}

	return config, nil
}

func quoteName(name string) string {
	var b strings.Builder
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch c {
		case '\a':
			b.WriteString(`\a`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '\v':
			b.WriteString(`\v`)
		case '\\':
			b.WriteString(`\\`)
		default:
			if c < 32 || c >= 127 {
				_, _ = fmt.Fprintf(&b, "\\%03o", c)
			} else {
				b.WriteByte(c)
			}
		}
	}
	return b.String()
}

// run executes the ls logic for a given configuration
func run(path string, config *Config, stdout, stderr io.Writer) int {
	if config.ListDirectory {
		// List the directory itself, not its contents
		// Check if the path exists first
		if _, err := os.Lstat(path); err != nil {
			_, _ = fmt.Fprintf(stderr, "ls: cannot access '%s': %v\n", path, err)
			return 2
		}

		entry := &dirEntryWrapper{name: path, dirPath: path, isRoot: true}
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
		filtered = append(filtered, &dirEntryWrapper{name: ".", dirPath: path})
		filtered = append(filtered, &dirEntryWrapper{name: "..", dirPath: path})
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
			filtered = append(filtered, entry)
		}
	}

	// Sort filtered entries by name for consistent output
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Name() < filtered[j].Name()
	})

	// Warn if -h is used without -l
	if (config.HumanReadable || config.SI) && !config.LongListing {
		// "HumanReadable should only work with long and size enabled otherwise ignored."
		flag := "-h"
		if config.SI {
			flag = "--si"
		}
		_, _ = fmt.Fprintf(stderr, "ls: warning: option %s is ignored when -l is not used\n", flag)
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

func printLongList(w io.Writer, entries []os.DirEntry, config *Config) {
	if len(entries) == 0 {
		return
	}

	// Gather all file info
	type fileDetails struct {
		mode    string
		nlink   uint64
		owner   string
		group   string
		sizeStr string
		modTime time.Time
		name    string
	}

	details := make([]fileDetails, 0, len(entries))
	var maxLinkLen, maxOwnerLen, maxGroupLen, maxSizeLen int
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			// Skip entries we can't get info for (or print error?)
			// Standard ls might print error.
			continue
		}

		var nlink uint64 = 1
		var owner, group string
		size := info.Size()

		sysStat, ok := info.Sys().(*syscall.Stat_t)
		if ok {
			nlink = sysStat.Nlink
			if config.ShowAuthor {
				owner, group = fsutil.GetOwnerGroup(sysStat)
			}
		} else {
			// Non-Unix fallback
			if config.ShowAuthor {
				owner = "-" // Placeholder for owner
				group = "-" // Placeholder for group
			}
		}

		var sizeStr string
		if config.SI {
			sizeStr = formatSize(size, 1000)
		} else if config.HumanReadable {
			sizeStr = formatSize(size, 1024)
		} else {
			sizeStr = fmt.Sprintf("%d", size)
		}

		name := entry.Name()
		if config.Escape {
			name = quoteName(name)
		}

		d := fileDetails{
			mode:    info.Mode().String(),
			nlink:   nlink,
			owner:   owner,
			group:   group,
			sizeStr: sizeStr,
			modTime: info.ModTime(),
			name:    name,
		}
		details = append(details, d)

		// Calculate max widths
		if l := len(fmt.Sprint(d.nlink)); l > maxLinkLen {
			maxLinkLen = l
		}
		if config.ShowAuthor {
			if l := len(d.owner); l > maxOwnerLen {
				maxOwnerLen = l
			}
			if l := len(d.group); l > maxGroupLen {
				maxGroupLen = l
			}
		}
		if l := len(d.sizeStr); l > maxSizeLen {
			maxSizeLen = l
		}
	}
	// Ensure minimum width for nlink, as ls often does
	if maxLinkLen < 2 {
		maxLinkLen = 2
	}

	// Print
	// Format: mode nlink owner group size date time name
	// e.g. -rw-r--r-- 1 user group 123 Jan 01 12:00 file.txt
	for _, d := range details {
		if config.ShowAuthor {
			_, _ = fmt.Fprintf(w, "%s %*d %-*s %-*s %*s %s %s\n",
				d.mode,
				maxLinkLen, d.nlink,
				maxOwnerLen, d.owner,
				maxGroupLen, d.group,
				maxSizeLen, d.sizeStr,
				d.modTime.Format("Jan 02 15:04"),
				d.name,
			)
		} else {
			// Print without owner/group if --author is not specified
			_, _ = fmt.Fprintf(w, "%s %*d %*s %s %s\n",
				d.mode,
				maxLinkLen, d.nlink,
				maxSizeLen, d.sizeStr,
				d.modTime.Format("Jan 02 15:04"),
				d.name,
			)
		}
	}
}

func formatSize(size int64, unit int64) string {
	if size < unit {
		return fmt.Sprintf("%d", size)
	}
	const suffixes = "KMGTPE"
	const maxExp = len(suffixes) - 1
	div, exp := unit, 0
	for n := size / unit; n >= unit && exp < maxExp; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%c", float64(size)/float64(div), suffixes[exp])
}

// printEntries prints entry names to the output writer.
// It detects if the writer is a terminal to format output in columns.
func printEntries(w io.Writer, names []string) {
	if len(names) == 0 {
		return
	}
	// Check if output is a terminal
	var termWidth int
	isTerminal := false
	if f, ok := w.(interface{ Fd() uintptr }); ok {
		fd := int(f.Fd())
		if isTerminalFunc(fd) {
			width, _, err := getTermSizeFunc(fd)
			if err == nil {
				termWidth = width
				isTerminal = true
			}
		}
	}

	if isTerminal {
		printGrid(w, names, termWidth)
	} else {
		// Not a terminal or failed to get size: one entry per line
		for _, name := range names {
			_, _ = fmt.Fprintln(w, name)
		}
	}
}

// For testing purposes
var isTerminalFunc = func(fd int) bool {
	return term.IsTerminal(fd)
}

var getTermSizeFunc = func(fd int) (width, height int, err error) {
	return term.GetSize(fd)
}

// printGrid formats names into a grid that fits within width.
// Users column-major order (standard ls behavior).
func printGrid(w io.Writer, names []string, width int) {
	if len(names) == 0 {
		return
	}
	// 1. Determine maximum name length (add 2 spaces for padding)
	maxLen := 0
	for _, name := range names {
		if len(name) > maxLen {
			maxLen = len(name)
		}
	}
	colWidth := maxLen + 2 // 2 spaces padding

	// 2. Determine number of columns
	// Avoid division by zero
	if colWidth > width {
		colWidth = width
	}
	if colWidth == 0 {
		colWidth = 1
	}

	numCols := width / colWidth
	if numCols == 0 {
		numCols = 1
	}
	if numCols > len(names) {
		numCols = len(names)
	}

	// 3. Determine number of rows
	// ceil(len(names) / numCols)
	numRows := (len(names) + numCols - 1) / numCols

	// 4. Print in column-major order
	// row 0:  idx 0,                idx 0+rows,            idx 0+2*rows...
	// row r:  idx r,                idx r+rows,            ...
	for r := 0; r < numRows; r++ {
		for c := 0; c < numCols; c++ {
			idx := c*numRows + r
			if idx < len(names) {
				// Calculate padding for alignment
				format := fmt.Sprintf("%%-%ds", colWidth)
				_, _ = fmt.Fprintf(w, format, names[idx])
			}
		}
		_, _ = fmt.Fprintln(w)
	}
}

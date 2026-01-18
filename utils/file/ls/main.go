package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	pflag "github.com/spf13/pflag"
)

// Config holds the configuration for ls command
type Config struct {
	Directory string
	ShowAll   bool // -a: do not ignore entries starting with .
}

func main() {
	// Parse command-line arguments
	config, err := ParseArgs(os.Args[1:])
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "ls: %v\n", err)
		os.Exit(1)
	}

	exitCode := run(config, os.Stdout, os.Stderr)
	os.Exit(exitCode)
}

// ParseArgs parses command-line arguments using pflag and returns a Config.
func ParseArgs(args []string) (*Config, error) {
	config := &Config{}

	// Create a new FlagSet for ls
	fs := pflag.NewFlagSet("ls", pflag.ContinueOnError)

	// Suppress default error output, we'll handle errors ourselves
	fs.SetOutput(io.Discard)

	// Define flags with both short and long forms
	fs.BoolVarP(&config.ShowAll, "all", "a", false, "do not ignore entries starting with .")

	// Parse the arguments
	err := fs.Parse(args)
	if err != nil {
		if errors.Is(err, pflag.ErrHelp) {
			fs.SetOutput(os.Stderr)
			_, _ = fmt.Fprintf(os.Stderr, "Usage: ls [OPTION]... [FILE]...\n")
			_, _ = fmt.Fprintf(os.Stderr, "List information about the FILEs (the current directory by default).\n\n")
			_, _ = fmt.Fprintf(os.Stderr, "Options:\n")
			fs.PrintDefaults()
			os.Exit(0)
		}
		return nil, err
	}

	// Get remaining arguments after flags (the directory path)
	remainingArgs := fs.Args()

	if len(remainingArgs) == 0 {
		// No directory specified, use the current directory
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("error getting current directory: %v", err)
		}
		config.Directory = cwd
	} else {
		// Use the first argument as the directory
		config.Directory = remainingArgs[0]
	}

	return config, nil
}

// run executes the ls logic for a given configuration
func run(config *Config, stdout, stderr io.Writer) int {
	// Read directory entries
	entries, err := os.ReadDir(config.Directory)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ls: cannot access '%s': %v\n", config.Directory, err)
		return 2
	}

	// List entries based on configuration
	names := listEntries(entries, config.ShowAll)
	printEntries(stdout, names)
	return 0
}

// listEntries extracts names from directory entries.
// If showAll is false, entries starting with '.' are filtered out.
func listEntries(entries []os.DirEntry, showAll bool) []string {
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		// Skip hidden files unless showAll is true
		if !showAll && strings.HasPrefix(name, ".") {
			continue
		}
		names = append(names, name)
	}
	return names
}

// printEntries prints entry names to the output writer
func printEntries(w io.Writer, names []string) {
	for _, name := range names {
		_, _ = fmt.Fprint(w, name+" ")
	}
	_, _ = fmt.Fprintln(w)
}

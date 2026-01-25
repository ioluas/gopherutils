package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/pflag"
)

// ParseArgs parses command-line arguments using pflag and returns a Config.
func ParseArgs(args []string, stderr io.Writer) (*Config, error) {
	config := &Config{}
	var showHelp bool
	var blockSizeRaw string
	var timeWord string
	var timeStyleRaw string

	// Create a new FlagSet for ls
	flagSet := pflag.NewFlagSet("ls", pflag.ContinueOnError)

	// Suppress default error output, we'll handle errors ourselves
	flagSet.SetOutput(io.Discard)

	// Define flags with both short and long forms
	flagSet.BoolVarP(&config.ShowAll, "all", "a", false, "do not ignore entries starting with .")
	flagSet.BoolVarP(&config.ShowAlmostAll, "almost-all", "A", false, "do not list implied . and ..")
	flagSet.BoolVarP(&config.LongListing, "long", "l", false, "use a long listing format")
	flagSet.BoolVarP(&config.SortTime, "sort-time", "t", false, "sort by time, newest first; see --time")
	flagSet.StringVar(&timeWord, "time", "", "select which timestamp used to display or sort; access time (-u): atime, access, use; metadata change time (-c): ctime, status; modified time (default): mtime, modification; birth time: birth, creation")
	flagSet.StringVar(&timeStyleRaw, "time-style", "", "time/date format with -l; see TIME_STYLE")
	flagSet.BoolVar(&config.FullTime, "full-time", false, "like -l --time-style=full-iso")
	flagSet.BoolVarP(&config.HumanReadable, "human-readable", "h", false, "with -l, print sizes in human readable format (e.g., 1K 234M 2G)")
	flagSet.BoolVar(&config.SI, "si", false, "with -l, print sizes in powers of 1000 not 1024")
	flagSet.BoolVar(&config.ShowAuthor, "author", false, "with -l, print the author of each file. Note: Due to OS/filesystem limitations, the author is typically the same as the owner.")
	flagSet.BoolVarP(&config.NoGroup, "no-group", "G", false, "in a long listing, don't print group names")
	flagSet.BoolVarP(&config.Escape, "escape", "b", false, "print C-style escapes for nongraphic characters")
	flagSet.BoolVarP(&config.IgnoreBackups, "ignore-backups", "B", false, "do not list implied entries ending with ~")
	flagSet.BoolVarP(&config.ListDirectory, "directory", "d", false, "list directories themselves, not their contents")
	flagSet.StringVar(&blockSizeRaw, "block-size", "", "with -l, scale sizes by SIZE when printing them; e.g., '--block-size=M'")
	flagSet.BoolVarP(&showHelp, "help", "?", false, "display this help and exit")

	_ = flagSet.MarkHidden("sort-time")

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

	config.TimeField = timeFieldMod
	if timeWord != "" {
		field, err := parseTimeWord(timeWord)
		if err != nil {
			return nil, err
		}
		config.TimeField = field
		config.TimeFieldSet = true
	}

	if timeStyleRaw == "" {
		timeStyleRaw = os.Getenv("TIME_STYLE")
	} else {
		config.TimeStyleSet = true
	}
	if config.FullTime {
		if config.TimeStyleSet {
			_, _ = fmt.Fprintf(stderr, "ls: warning: --full-time is ignored when --time-style is used\n")
		} else {
			config.TimeStyleSet = true
			timeStyleRaw = "full-iso"
		}
	}
	if timeStyleRaw != "" {
		spec, warnMsg, ok := parseTimeStyle(timeStyleRaw)
		if warnMsg != "" {
			prefix := "TIME_STYLE"
			if config.TimeStyleSet {
				prefix = "--time-style"
			}
			_, _ = fmt.Fprintf(stderr, "ls: warning: %s: %s\n", prefix, warnMsg)
		}
		if ok {
			config.TimeStyleSpec = spec
		}
	}

	if blockSizeRaw != "" {
		spec, warnMsg, ok := parseBlockSize(blockSizeRaw)
		if warnMsg != "" {
			_, _ = fmt.Fprintf(stderr, "ls: warning: --block-size: %s\n", warnMsg)
		}
		if ok {
			config.BlockSize = &spec
		}
	}

	return config, nil
}

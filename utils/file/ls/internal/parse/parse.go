package parse

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/pflag"

	"github.com/ioluas/gopherutils/utils/file/ls/internal/config"
	"github.com/ioluas/gopherutils/utils/file/ls/internal/size"
	"github.com/ioluas/gopherutils/utils/file/ls/internal/timeutil"
)

// ParseArgs parses command-line arguments using pflag and returns a Config.
func ParseArgs(args []string, stderr io.Writer) (*config.Config, error) {
	cfg := &config.Config{}
	var showHelp bool
	var blockSizeRaw string
	var timeWord string
	var timeStyleRaw string

	// Create a new FlagSet for ls
	flagSet := pflag.NewFlagSet("ls", pflag.ContinueOnError)

	// Suppress default error output, we'll handle errors ourselves
	flagSet.SetOutput(io.Discard)

	// Define flags with both short and long forms
	flagSet.BoolVarP(&cfg.ShowAll, "all", "a", false, "do not ignore entries starting with .")
	flagSet.BoolVarP(&cfg.ShowAlmostAll, "almost-all", "A", false, "do not list implied . and ..")
	flagSet.BoolVarP(&cfg.LongListing, "long", "l", false, "use a long listing format")
	flagSet.BoolVarP(&cfg.SortTime, "sort-time", "t", false, "sort by time, newest first; see --time")
	flagSet.StringVar(&timeWord, "time", "", "select which timestamp used to display or sort; access time (-u): atime, access, use; metadata change time (-c): ctime, status; modified time (default): mtime, modification; birth time: birth, creation")
	flagSet.StringVar(&timeStyleRaw, "time-style", "", "time/date format with -l; see TIME_STYLE")
	flagSet.BoolVar(&cfg.FullTime, "full-time", false, "like -l --time-style=full-iso")
	flagSet.BoolVarP(&cfg.HumanReadable, "human-readable", "h", false, "with -l, print sizes in human readable format (e.g., 1K 234M 2G)")
	flagSet.BoolVar(&cfg.SI, "si", false, "with -l, print sizes in powers of 1000 not 1024")
	flagSet.BoolVar(&cfg.ShowAuthor, "author", false, "with -l, print the author of each file. Note: Due to OS/filesystem limitations, the author is typically the same as the owner.")
	flagSet.BoolVarP(&cfg.NoGroup, "no-group", "G", false, "in a long listing, don't print group names")
	flagSet.BoolVarP(&cfg.Escape, "escape", "b", false, "print C-style escapes for nongraphic characters")
	flagSet.BoolVarP(&cfg.IgnoreBackups, "ignore-backups", "B", false, "do not list implied entries ending with ~")
	flagSet.BoolVarP(&cfg.Columnate, "C", "C", false, "list entries by columns")
	flagSet.BoolVarP(&cfg.OnePerLine, "one-file-per-line", "1", false, "list one file per line")
	flagSet.BoolVarP(&cfg.ListDirectory, "directory", "d", false, "list directories themselves, not their contents")
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
		cfg.Directories = []string{"."}
	} else {
		// Use all remaining arguments as directories
		cfg.Directories = remainingArgs
	}

	cfg.TimeField = config.TimeFieldMod
	if timeWord != "" {
		field, err := timeutil.ParseTimeWord(timeWord)
		if err != nil {
			return nil, err
		}
		cfg.TimeField = field
		cfg.TimeFieldSet = true
	}

	if timeStyleRaw == "" {
		timeStyleRaw = os.Getenv("TIME_STYLE")
	} else {
		cfg.TimeStyleSet = true
	}
	if cfg.FullTime {
		if cfg.TimeStyleSet {
			_, _ = fmt.Fprintf(stderr, "ls: warning: --full-time is ignored when --time-style is used\n")
		} else {
			cfg.TimeStyleSet = true
			timeStyleRaw = "full-iso"
		}
	}
	if timeStyleRaw != "" {
		spec, warnMsg, ok := timeutil.ParseTimeStyle(timeStyleRaw)
		if warnMsg != "" {
			prefix := "TIME_STYLE"
			if cfg.TimeStyleSet {
				prefix = "--time-style"
			}
			_, _ = fmt.Fprintf(stderr, "ls: warning: %s: %s\n", prefix, warnMsg)
		}
		if ok {
			cfg.TimeStyleSpec = spec
		}
	}

	if blockSizeRaw != "" {
		spec, warnMsg, ok := size.ParseBlockSize(blockSizeRaw)
		if warnMsg != "" {
			_, _ = fmt.Fprintf(stderr, "ls: warning: --block-size: %s\n", warnMsg)
		}
		if ok {
			cfg.BlockSize = &spec
		}
	}

	return cfg, nil
}

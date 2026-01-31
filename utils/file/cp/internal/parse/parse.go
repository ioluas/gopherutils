package parse

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ioluas/gopherutils/utils/file/cp/internal/config"
	"github.com/spf13/pflag"
)

// Args parses command-line arguments using pflag and returns a Config.
func Args(args []string, stderr io.Writer) (*config.Config, error) {
	cfg := &config.Config{}
	var showHelp bool
	var backupControl string
	var suffix string
	var bFlag bool
	var updateControl string

	flagSet := pflag.NewFlagSet("cp", pflag.ContinueOnError)
	flagSet.SetOutput(io.Discard)

	flagSet.BoolVarP(&cfg.Verbose, "verbose", "v", false, "explain what is being done")
	flagSet.BoolVarP(&showHelp, "help", "?", false, "display this help and exit")

	// -b: like --backup but does not accept an argument
	flagSet.BoolVarP(&bFlag, "backup-short", "b", false, "like --backup but does not accept an argument")

	// --backup[=CONTROL]
	flagSet.StringVar(&backupControl, "backup", "", "make a backup of each existing destination file")

	// --update[=UPDATE]
	flagSet.StringVarP(&updateControl, "update", "u", "", "control which existing files are updated; UPDATE={all,none,none-fail,older(default)}. -u is equivalent to --update[=older]")

	flagSet.StringVarP(&suffix, "suffix", "S", "", "override the usual backup suffix")

	// Set NoOptDefVal for --backup
	if f := flagSet.Lookup("backup"); f != nil {
		f.NoOptDefVal = "existing"
	}

	// Set NoOptDefVal for --update
	if f := flagSet.Lookup("update"); f != nil {
		f.NoOptDefVal = "older"
	}

	err := flagSet.Parse(args)
	if showHelp {
		err = pflag.ErrHelp
	}

	if err != nil {
		if errors.Is(err, pflag.ErrHelp) {
			flagSet.SetOutput(stderr)
			_, _ = fmt.Fprintf(stderr, "Usage: cp [OPTION]... SOURCE DEST\n")
			_, _ = fmt.Fprintf(stderr, "Copy SOURCE to DEST, or multiple SOURCE(s) to DIRECTORY.\n\n")
			_, _ = fmt.Fprintf(stderr, "Options:\n")
			flagSet.PrintDefaults()
			return nil, pflag.ErrHelp
		}
		return nil, err
	}

	positional := flagSet.Args()
	if len(positional) < 2 {
		return nil, fmt.Errorf("missing file operand")
	}

	cfg.Dest = positional[len(positional)-1]
	cfg.Sources = positional[:len(positional)-1]

	// Determine Backup Configuration
	backupRequested := bFlag || flagSet.Changed("backup")

	if backupRequested {
		cfg.Backup = true
		var methodStr string
		if flagSet.Changed("backup") && backupControl != "" {
			methodStr = backupControl
		} else {
			methodStr = os.Getenv("VERSION_CONTROL")
		}

		if methodStr == "" {
			methodStr = "existing"
		}

		bm, err := parseBackupMethod(methodStr)
		if err != nil {
			if flagSet.Changed("backup") {
				return nil, fmt.Errorf("invalid argument '%s' for '--backup'", methodStr)
			}
			cfg.BackupMethod = config.BackupExisting
		} else {
			cfg.BackupMethod = bm
		}
	} else {
		cfg.Backup = false
		cfg.BackupMethod = config.BackupNone
	}

	// Determine Suffix
	if suffix != "" {
		cfg.Suffix = suffix
	} else {
		cfg.Suffix = os.Getenv("SIMPLE_BACKUP_SUFFIX")
	}

	if cfg.Suffix == "" {
		cfg.Suffix = "~"
	}

	// Determine Update Mode
	if flagSet.Changed("update") {
		// If --update is present, default is older if no arg.
		// NoOptDefVal handles the "no arg" case, so if it was used without arg, updateControl is "older".
		// If it was used with arg, updateControl is that arg.
		// If it was NOT used, updateControl is empty (and Changed("update") is false).
		mode, err := parseUpdateMode(updateControl)
		if err != nil {
			return nil, err
		}
		cfg.UpdateMode = mode
	} else {
		cfg.UpdateMode = config.UpdateReplaceAll // Default behavior
	}

	return cfg, nil
}

func parseBackupMethod(s string) (config.BackupMethod, error) {
	switch strings.ToLower(s) {
	case "none", "off":
		return config.BackupNone, nil
	case "numbered", "t":
		return config.BackupNumbered, nil
	case "existing", "nil":
		return config.BackupExisting, nil
	case "simple", "never":
		return config.BackupSimple, nil
	default:
		return config.BackupExisting, fmt.Errorf("invalid argument '%s' for '--backup'", s)
	}
}

func parseUpdateMode(s string) (config.UpdateMode, error) {
	switch strings.ToLower(s) {
	case "all":
		return config.UpdateReplaceAll, nil
	case "none":
		return config.UpdateNone, nil
	case "none-fail":
		return config.UpdateNoneFail, nil
	case "older":
		return config.UpdateReplaceOlder, nil
	default:
		return config.UpdateReplaceAll, fmt.Errorf("invalid argument '%s' for '--update'", s)
	}
}

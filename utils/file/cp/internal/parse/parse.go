package parse

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
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

	flagSet := pflag.NewFlagSet("cp", pflag.ContinueOnError)
	flagSet.SetOutput(io.Discard)

	flagSet.BoolVarP(&cfg.Verbose, "verbose", "v", false, "explain what is being done")
	flagSet.BoolVar(&cfg.AttributesOnly, "attributes-only", false, "don't copy the file data, just the attributes")
	flagSet.BoolVarP(&showHelp, "help", "?", false, "display this help and exit")

	// -b: like --backup but does not accept an argument
	flagSet.BoolVarP(&bFlag, "backup-short", "b", false, "like --backup but does not accept an argument")

	// --backup[=CONTROL]
	flagSet.StringVar(&backupControl, "backup", "", "make a backup of each existing destination file")

	// Update Mode Flags handling
	// We use custom flags for --update and --no-clobber to ensure "last one wins" precedence
	// by modifying the shared cfg.UpdateMode directly during parsing.
	
	// Default behavior
	cfg.UpdateMode = config.UpdateReplaceAll

	uFlag := &updateFlag{mode: &cfg.UpdateMode}
	flagSet.VarP(uFlag, "update", "u", "control which existing files are updated; UPDATE={all,none,none-fail,older(default)}. -u is equivalent to --update[=older]")
	flagSet.Lookup("update").NoOptDefVal = "older"

	nFlag := &noClobberFlag{mode: &cfg.UpdateMode}
	flagSet.VarP(nFlag, "no-clobber", "n", "do not overwrite an existing file (overrides a previous -u)")
	flagSet.Lookup("no-clobber").NoOptDefVal = "true"

	// Preservation flags
	var preserveActions []preserveAction

	// -p: preserve default attributes
	// We use BoolVarP for -p because it's a bool flag, but we also want to hook into the actions list.
	// Since BoolVarP doesn't support custom Value, we can use VarP with a custom BoolValue-like implementation
	// OR we can just rely on pflag to parse -p as bool, AND --preserve as Var, but then we lose order between -p and --preserve.
	// To preserve order, -p MUST be a custom Var that takes "true" (implicit).
	// pflag's NoOptDefVal for bool flags is "true".

	pFlag := &preserveFlag{actions: &preserveActions, enable: true, isP: true}
	flagSet.VarP(pFlag, "preserve-short", "p", "same as --preserve=mode,ownership,timestamps")
	// Make it behave like a bool flag (no argument)
	flagSet.Lookup("preserve-short").NoOptDefVal = "true"

	preserveFlagObj := &preserveFlag{actions: &preserveActions, enable: true, isP: false}
	flagSet.Var(preserveFlagObj, "preserve", "preserve the specified attributes (default: mode,ownership,timestamps), if possible additional attributes: context, links, xattr, all")
	flagSet.Lookup("preserve").NoOptDefVal = "mode,ownership,timestamps"

	noPreserveFlagObj := &preserveFlag{actions: &preserveActions, enable: false, isP: false}
	flagSet.Var(noPreserveFlagObj, "no-preserve", "don't preserve the specified attributes")

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

	

		// Calculate Preserve Options

		cfg.Preserve = resolvePreserveOptions(preserveActions)

	

		return cfg, nil

	}

	

	type updateFlag struct {

		mode *config.UpdateMode

	}

	

	func (f *updateFlag) String() string {

		// We don't really strictly implement String() reflecting current state for usage,

		// just return something or empty.

		return ""

	}

	

	func (f *updateFlag) Set(val string) error {

		m, err := parseUpdateMode(val)

		if err != nil {

			return err

		}

		*f.mode = m

		return nil

	}

	

	func (f *updateFlag) Type() string {

		return "string"

	}

	

	type noClobberFlag struct {

		mode *config.UpdateMode

	}

	

	func (f *noClobberFlag) String() string {

		return "false"

	}

	

	func (f *noClobberFlag) Set(val string) error {

	

		b, err := strconv.ParseBool(val)

	

		if err != nil {

	

			return fmt.Errorf("invalid boolean value '%s' for -n/--no-clobber", val)

	

		}

	

		if b {

	

			*f.mode = config.UpdateNone

	

		} else {

	

			// If --no-clobber=false is passed, it implies clobbering is allowed.

	

			// We reset to default (ReplaceAll).

	

			*f.mode = config.UpdateReplaceAll

	

		}

	

		return nil

	

	}

	

	func (f *noClobberFlag) Type() string {

		return "bool"

	}

	

	type preserveAction struct {

		enable bool

		attrs  []string // "mode", "ownership", ... or "defaults" or "all"

	}

// preserveFlag collects actions in order
type preserveFlag struct {
	actions *[]preserveAction
	enable  bool
	isP     bool // true if this is -p (defaults)
}

func (f *preserveFlag) String() string {
	return ""
}

func (f *preserveFlag) Set(val string) error {
	var attrs []string
	if f.isP {
		// -p implies defaults. val is "true" (from bool flag)
		if val == "true" {
			attrs = []string{"mode", "ownership", "timestamps"}
		} else {
			return nil
		}
	} else {
		if val == "" {
			// Should be handled by NoOptDefVal, but just in case
			if f.enable {
				attrs = []string{"mode", "ownership", "timestamps"}
			}
		} else {
			parts := strings.Split(val, ",")
			for _, p := range parts {
				attrs = append(attrs, strings.TrimSpace(p))
			}
		}
	}
	*f.actions = append(*f.actions, preserveAction{enable: f.enable, attrs: attrs})
	return nil
}

func (f *preserveFlag) Type() string {
	return "string"
}

func resolvePreserveOptions(actions []preserveAction) config.PreserveOptions {
	// Start with all false
	opts := config.PreserveOptions{}

	for _, action := range actions {
		for _, attr := range action.attrs {
			val := action.enable
			switch strings.ToLower(attr) {
			case "mode":
				opts.Mode = val
			case "ownership":
				opts.Ownership = val
			case "timestamps":
				opts.Timestamps = val
			case "links":
				opts.Links = val
			case "context":
				opts.Context = val
			case "xattr":
				opts.Xattr = val
			case "defaults":
				opts.Mode = val
				opts.Ownership = val
				opts.Timestamps = val
				opts.Links = val
			case "all":
				opts.Mode = val
				opts.Ownership = val
				opts.Timestamps = val
				opts.Links = val
				opts.Context = val
				opts.Xattr = val
			}
		}
	}
	return opts
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

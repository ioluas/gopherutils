package parse

import (
	"errors"
	"fmt"
	"io"

	"github.com/ioluas/gopherutils/utils/file/cp/internal/config"
	"github.com/spf13/pflag"
)

// Args parses command-line arguments using pflag and returns a Config.
func Args(args []string, stderr io.Writer) (*config.Config, error) {
	cfg := &config.Config{}
	var showHelp bool

	flagSet := pflag.NewFlagSet("cp", pflag.ContinueOnError)
	flagSet.SetOutput(io.Discard)

	flagSet.BoolVarP(&cfg.Verbose, "verbose", "v", false, "explain what is being done")
	flagSet.BoolVarP(&showHelp, "help", "?", false, "display this help and exit")

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

	return cfg, nil
}

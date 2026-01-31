package config

// Config holds the configuration for the cp utility.
type Config struct {
	Sources []string
	Dest    string
	Verbose bool // -v, --verbose: explain what is being done
}

package config

// Config holds the configuration for the ls utility.
type Config struct {
	Directories   []string
	ShowAll       bool // -a: do not ignore entries starting with .
	ShowAlmostAll bool // -A: do not list implied . and ..
	LongListing   bool // -l: use a long listing format
	SortTime      bool // -t: sort by time, newest first
	TimeField     TimeField
	TimeFieldSet  bool
	TimeStyleSet  bool
	TimeStyleSpec *TimeStyleSpec
	FullTime      bool // --full-time: like --time-style=full-iso
	HumanReadable bool // -h: with -l, print sizes in human readable format
	SI            bool // --si: with -l, print sizes in powers of 1000 not 1024
	ShowAuthor    bool // --author: with -l, print the author of each file
	NoGroup       bool // -G, --no-group: in a long listing, don't print group names
	Escape        bool // -b, --escape: print C-style escapes for nongraphic characters
	IgnoreBackups bool // -B, --ignore-backups: do not list implied entries ending with ~
	Columnate     bool // -C: list entries by columns
	OnePerLine    bool // -1: list one file per line
	ListDirectory bool // -d, --directory: list directories themselves, not their contents
	Dired         bool // -D, --dired: generate output designed for Emacs' dired mode
	BlockSize     *BlockSizeSpec
	FormatMode    FormatMode   // Tracks which format flag was specified last (-C, -1, or default)
	QuotingStyle  QuotingStyle // --quoting-style=WORD
}

// FormatMode represents the output format mode
type FormatMode int

const (
	FormatDefault    FormatMode = iota // No explicit format flag
	FormatColumnate                    // -C was last
	FormatOnePerLine                   // -1 was last
)

type QuotingStyle int

const (
	QuotingStyleLiteral QuotingStyle = iota
	QuotingStyleLocale
	QuotingStyleShell
	QuotingStyleShellAlways
	QuotingStyleShellEscape
	QuotingStyleShellEscapeAlways
	QuotingStyleC
	QuotingStyleEscape
)

func (q QuotingStyle) String() string {
	switch q {
	case QuotingStyleLiteral:
		return "literal"
	case QuotingStyleLocale:
		return "locale"
	case QuotingStyleShell:
		return "shell"
	case QuotingStyleShellAlways:
		return "shell-always"
	case QuotingStyleShellEscape:
		return "shell-escape"
	case QuotingStyleShellEscapeAlways:
		return "shell-escape-always"
	case QuotingStyleC:
		return "c"
	case QuotingStyleEscape:
		return "escape"
	default:
		return "literal"
	}
}

type BlockSizeMode int

const (
	BlockSizeModeBytes BlockSizeMode = iota
	BlockSizeModeHuman
	BlockSizeModeSI
)

type BlockSizeSpec struct {
	Mode           BlockSizeMode
	SizeBytes      int64
	Suffix         string
	ShowSuffix     bool
	GroupThousands bool
}

type TimeField int

const (
	TimeFieldMod TimeField = iota
	TimeFieldAccess
	TimeFieldChange
	TimeFieldBirth
)

type TimeStyleKind int

const (
	TimeStyleLocale TimeStyleKind = iota
	TimeStyleFullISO
	TimeStyleLongISO
	TimeStyleISO
	TimeStyleCustom
)

type TimeStyleSpec struct {
	Kind         TimeStyleKind
	RecentLayout string
	OldLayout    string
}

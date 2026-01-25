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
	ListDirectory bool // -d, --directory: list directories themselves, not their contents
	BlockSize     *BlockSizeSpec
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

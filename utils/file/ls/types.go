package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// Config holds the configuration for the ls utility.
type Config struct {
	Directories   []string
	ShowAll       bool // -a: do not ignore entries starting with .
	ShowAlmostAll bool // -A: do not list implied . and ..
	LongListing   bool // -l: use a long listing format
	SortTime      bool // -t: sort by time, newest first
	TimeField     timeField
	TimeFieldSet  bool
	TimeStyleSet  bool
	TimeStyleSpec *timeStyleSpec
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

type cachedDirEntry struct {
	os.DirEntry
	info fs.FileInfo
	time *time.Time
}

func (e *cachedDirEntry) Info() (fs.FileInfo, error) {
	if e.info != nil {
		return e.info, nil
	}
	return e.DirEntry.Info()
}

type fileDetails struct {
	mode    string
	nlink   uint64
	owner   string
	author  string
	group   string
	sizeStr string
	modTime time.Time
	name    string
}

type longListWidths struct {
	link   int
	owner  int
	author int
	group  int
	size   int
}

type blockSizeMode int

const (
	blockSizeModeBytes blockSizeMode = iota
	blockSizeModeHuman
	blockSizeModeSI
)

type BlockSizeSpec struct {
	mode           blockSizeMode
	sizeBytes      int64
	suffix         string
	showSuffix     bool
	groupThousands bool
}

type timeField int

const (
	timeFieldMod timeField = iota
	timeFieldAccess
	timeFieldChange
	timeFieldBirth
)

type timeStyleKind int

const (
	timeStyleLocale timeStyleKind = iota
	timeStyleFullISO
	timeStyleLongISO
	timeStyleISO
	timeStyleCustom
)

type timeStyleSpec struct {
	kind         timeStyleKind
	recentLayout string
	oldLayout    string
}

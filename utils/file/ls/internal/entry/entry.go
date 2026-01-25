package entry

import (
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// DirEntryWrapper implements os.DirEntry for "." and "..", or for -d flag.
type DirEntryWrapper struct {
	EntryName string
	DirPath   string // The path of the directory whose entries are being listed, or the path to the entry itself.
	IsRoot    bool   // If true, Info() should stat DirPath directly.
}

func (d *DirEntryWrapper) Name() string {
	return d.EntryName
}

func (d *DirEntryWrapper) IsDir() bool {
	info, err := d.Info()
	if err != nil {
		return false
	}
	return info.IsDir()
}

func (d *DirEntryWrapper) Type() fs.FileMode {
	info, err := d.Info()
	if err != nil {
		return 0
	}
	return info.Mode().Type()
}

func (d *DirEntryWrapper) Info() (fs.FileInfo, error) {
	if d.IsRoot {
		return os.Lstat(d.DirPath)
	}
	targetPath := d.DirPath
	if d.EntryName == ".." {
		targetPath = filepath.Clean(filepath.Join(d.DirPath, ".."))
	}
	return os.Stat(targetPath)
}

type CachedDirEntry struct {
	os.DirEntry
	info fs.FileInfo
	Time *time.Time
	Err  error
}

func NewCachedDirEntry(dirEntry os.DirEntry, info fs.FileInfo) *CachedDirEntry {
	return &CachedDirEntry{DirEntry: dirEntry, info: info}
}

func (e *CachedDirEntry) Info() (fs.FileInfo, error) {
	if e.info != nil {
		return e.info, nil
	}
	if e.Err != nil {
		return nil, e.Err
	}
	info, err := e.DirEntry.Info()
	if err != nil {
		e.Err = err
		return nil, err
	}
	e.info = info
	return info, nil
}

func LessByTime(a, b os.DirEntry) bool {
	ca, aok := a.(*CachedDirEntry)
	cb, bok := b.(*CachedDirEntry)
	if !aok || !bok {
		return a.Name() < b.Name()
	}

	ti := ca.Time
	tj := cb.Time

	if ti != nil && tj != nil {
		if ti.Equal(*tj) {
			return a.Name() < b.Name()
		}
		return ti.After(*tj)
	}
	if ti != nil {
		return true
	}
	if tj != nil {
		return false
	}
	return a.Name() < b.Name()
}

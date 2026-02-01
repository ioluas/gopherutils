package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/ioluas/gopherutils/utils/file/cp/internal/backup"
	"github.com/ioluas/gopherutils/utils/file/cp/internal/config"
	"github.com/ioluas/gopherutils/utils/file/cp/internal/parse"
	"github.com/spf13/pflag"
)

type devIno struct {
	dev uint64
	ino uint64
}

func main() {
	os.Exit(Execute(os.Args[1:], os.Stdout, os.Stderr))
}

func Execute(args []string, stdout, stderr io.Writer) int {
	cfg, err := parse.Args(args, stderr)
	if err != nil {
		if errors.Is(err, pflag.ErrHelp) {
			return 0
		}
		_, _ = fmt.Fprintf(stderr, "cp: %v\n", err)
		return 1
	}

	linkMap := make(map[devIno]string)

	// Handle multiple sources -> Directory
	if len(cfg.Sources) > 1 {
		destInfo, err := os.Stat(cfg.Dest)
		if err != nil {
			if os.IsNotExist(err) {
				_, _ = fmt.Fprintf(stderr, "cp: target '%s' is not a directory\n", cfg.Dest)
			} else {
				_, _ = fmt.Fprintf(stderr, "cp: cannot stat '%s': %v\n", cfg.Dest, err)
			}
			return 1
		}
		if !destInfo.IsDir() {
			_, _ = fmt.Fprintf(stderr, "cp: target '%s' is not a directory\n", cfg.Dest)
			return 1
		}

		for _, src := range cfg.Sources {
			dst := filepath.Join(cfg.Dest, filepath.Base(src))
			if err := copyFile(src, dst, cfg, linkMap, stdout); err != nil {
				_, _ = fmt.Fprintf(stderr, "cp: cannot copy '%s' to '%s': %v\n", src, dst, err)
				return 1
			}
		}
		return 0
	}

	// Single source -> Destination (File or Dir)
	src := cfg.Sources[0]
	dst := cfg.Dest
	if err := copyFile(src, dst, cfg, linkMap, stdout); err != nil {
		_, _ = fmt.Fprintf(stderr, "cp: cannot copy '%s' to '%s': %v\n", src, dst, err)
		return 1
	}

	return 0
}

func sameDevIno(sourceInfo, destInfo os.FileInfo) bool {
	s1, ok1 := sourceInfo.Sys().(*syscall.Stat_t)
	if !ok1 {
		return false
	}
	s2, ok2 := destInfo.Sys().(*syscall.Stat_t)
	if !ok2 {
		return false
	}
	return s1.Dev == s2.Dev && s1.Ino == s2.Ino
}

func getDevIno(fi os.FileInfo) (devIno, bool) {
	s, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return devIno{}, false
	}
	return devIno{dev: uint64(s.Dev), ino: uint64(s.Ino)}, true
}

func copyFile(src, dst string, cfg *config.Config, linkMap map[devIno]string, stdout io.Writer) error {
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	if sourceInfo.IsDir() {
		return fmt.Errorf("omitting directory '%s'", src)
	}

	// Hard Link Preservation
	if cfg.Preserve.Links {
		if di, ok := getDevIno(sourceInfo); ok {
			if existingDst, found := linkMap[di]; found {
				// Create hard link
				// Remove dst if exists?
				if _, err := os.Stat(dst); err == nil {
					// Check if same file?
					// If we are here, we are linking to existingDst.
					// We should probably remove dst before linking?
					// Standard cp behavior: remove destination before linking.
					if err := os.Remove(dst); err != nil {
						return err
					}
				}
				if err := os.Link(existingDst, dst); err != nil {
					return err
				}
				if cfg.Verbose {
					_, _ = fmt.Fprintf(stdout, "'%s' -> '%s' (hard link to '%s')\n", src, dst, existingDst)
				}
				return nil
			}
			// Will add to map after successful copy
		}
	}

	// Check if dst is a directory
	destInfo, err := os.Stat(dst)
	if err == nil && destInfo.IsDir() {
		dst = filepath.Join(dst, filepath.Base(src))
	}

	// Check for existing destination file and Update logic
	dstFi, dstErr := os.Stat(dst)
	if dstErr == nil {
		// File exists
		// Check for same file
		if sameDevIno(sourceInfo, dstFi) {
			return fmt.Errorf("'%s' and '%s' are the same file", src, dst)
		}

		// Update logic
		switch cfg.UpdateMode {
		case config.UpdateNone:
			// none: do not replace, no error
			return nil
		case config.UpdateNoneFail:
			// none-fail: do not replace, error
			return fmt.Errorf("'%s': file exists", dst)
		case config.UpdateReplaceOlder:
			// older: replace if source is newer
			if !sourceInfo.ModTime().After(dstFi.ModTime()) {
				// Source is not newer (older or same), skip
				return nil
			}
		}
		// UpdateReplaceAll falls through
	}

	backupName := ""
	if cfg.Backup {
		var backupErr error
		backupName, backupErr = backup.MakeBackup(dst, cfg)
		if backupErr != nil {
			return backupErr
		}
	}

	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func(sourceFile *os.File) {
		_ = sourceFile.Close()
	}(sourceFile)

	// Create creates or truncates the named file. If the file already exists,
	// it is truncated. If the file does not exist, it is created with mode 0666
	// (before umask).
	openFlags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	if cfg.AttributesOnly {
		openFlags = os.O_WRONLY | os.O_CREATE
	}

	// Use source mode if preserving mode, else default (handled by Create with 0666 usually)
	// But OpenFile takes a perm.
	// If preserving mode, we might want to set it exactly later?
	// Usually cp creates with 0666&~umask, then chmod if -p is set.
	// If we use sourceInfo.Mode().Perm() here, it applies it immediately (masked by umask).
	// But if -p is set, we want the EXACT mode.
	// So we create, then Chmod.

	destFile, err := os.OpenFile(dst, openFlags, sourceInfo.Mode().Perm())
	if err != nil {
		return err
	}
	// We defer Close, but we also sync.

	if !cfg.AttributesOnly {
		_, err = io.Copy(destFile, sourceFile)
		if err != nil {
			_ = destFile.Close()
			return err
		}
	}

	// Flush to disk
	if err := destFile.Sync(); err != nil {
		_ = destFile.Close()
		return err
	}
	if err := destFile.Close(); err != nil {
		return err
	}

	// Preserve Attributes
	if err := preserveAttributes(dst, sourceInfo, cfg.Preserve); err != nil {
		// Usually cp warns but returns 0? Or 1?
		// "cp: preserving permissions for '...': Operation not permitted"
		// We'll return error for now.
		return err
	}

	// Register in linkMap if preserving links
	if cfg.Preserve.Links {
		if di, ok := getDevIno(sourceInfo); ok {
			linkMap[di] = dst
		}
	}

	if cfg.Verbose {
		if backupName != "" {
			_, _ = fmt.Fprintf(stdout, "'%s' -> '%s' (backup: '%s')\n", src, dst, backupName)
		} else {
			_, _ = fmt.Fprintf(stdout, "'%s' -> '%s'\n", src, dst)
		}
	}
	return nil
}

func preserveAttributes(dst string, srcInfo os.FileInfo, opts config.PreserveOptions) error {
	if opts.Mode {
		if err := os.Chmod(dst, srcInfo.Mode()); err != nil {
			return fmt.Errorf("preserving permissions for '%s': %w", dst, err)
		}
	}
	if opts.Ownership {
		if stat, ok := srcInfo.Sys().(*syscall.Stat_t); ok {
			if err := os.Chown(dst, int(stat.Uid), int(stat.Gid)); err != nil {
				return fmt.Errorf("preserving ownership for '%s': %w", dst, err)
			}
		}
	}
	if opts.Timestamps {
		// Use ModTime for both atime and mtime if we can't get atime?
		// Go's os.Chtimes takes (atime, mtime).
		// We can get atime from Sys().
		mtime := srcInfo.ModTime()
		atime := mtime // Fallback
		if stat, ok := srcInfo.Sys().(*syscall.Stat_t); ok {
			// Linux/Unix specific
			atime = time.Unix(int64(stat.Atim.Sec), int64(stat.Atim.Nsec))
		}
		if err := os.Chtimes(dst, atime, mtime); err != nil {
			return fmt.Errorf("preserving times for '%s': %w", dst, err)
		}
	}
	// Context and Xattr are not implemented yet.
	return nil
}

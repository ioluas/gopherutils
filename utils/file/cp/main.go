package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"github.com/ioluas/gopherutils/utils/file/cp/internal/backup"
	"github.com/ioluas/gopherutils/utils/file/cp/internal/config"
	"github.com/ioluas/gopherutils/utils/file/cp/internal/parse"
	"github.com/spf13/pflag"
)

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
			if err := copyFile(src, dst, cfg, stdout); err != nil {
				_, _ = fmt.Fprintf(stderr, "cp: cannot copy '%s' to '%s': %v\n", src, dst, err)
				return 1
			}
		}
		return 0
	}

	// Single source -> Destination (File or Dir)
	src := cfg.Sources[0]
	dst := cfg.Dest
	if err := copyFile(src, dst, cfg, stdout); err != nil {
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

func copyFile(src, dst string, cfg *config.Config, stdout io.Writer) error {
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	if sourceInfo.IsDir() {
		return fmt.Errorf("omitting directory '%s'", src)
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

	if cfg.Backup {
		backupName, err := backup.MakeBackup(dst, cfg)
		if err != nil {
			return err
		}
		if cfg.Verbose && backupName != "" {
			_, _ = fmt.Fprintf(stdout, "'%s' -> '%s' (backup: '%s')\n", src, dst, backupName)
		} else if cfg.Verbose {
			_, _ = fmt.Fprintf(stdout, "'%s' -> '%s'\n", src, dst)
		}
	} else {
		if cfg.Verbose {
			_, _ = fmt.Fprintf(stdout, "'%s' -> '%s'\n", src, dst)
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
	destFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, sourceInfo.Mode().Perm())
	if err != nil {
		return err
	}
	defer func(destFile *os.File) {
		_ = destFile.Close()
	}(destFile)

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Flush to disk
	return destFile.Sync()
}

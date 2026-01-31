package backup

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ioluas/gopherutils/utils/file/cp/internal/config"
)

// MakeBackup performs the backup operation for the given path based on configuration.
// It returns the name of the backup file created, or empty string if no backup was made.
func MakeBackup(path string, cfg *config.Config) (string, error) {
	if !cfg.Backup || cfg.BackupMethod == config.BackupNone {
		return "", nil
	}

	// Check if file exists
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	backupName, err := determineBackupName(path, cfg)
	if err != nil {
		return "", err
	}

	// Perform the rename
	err = os.Rename(path, backupName)
	if err != nil {
		return "", fmt.Errorf("cannot backup %s to %s: %w", path, backupName, err)
	}

	return backupName, nil
}

func determineBackupName(path string, cfg *config.Config) (string, error) {
	method := cfg.BackupMethod

	// Resolve BackupExisting: use Numbered if path.~1~ exists, else Simple.
	if method == config.BackupExisting {
		firstNumbered := path + ".~1~"
		if _, err := os.Stat(firstNumbered); err == nil {
			method = config.BackupNumbered
		} else if os.IsNotExist(err) {
			method = config.BackupSimple
		} else {
			return "", err
		}
	}

	switch method {
	case config.BackupSimple:
		return path + cfg.Suffix, nil
	case config.BackupNumbered:
		return findNextNumberedName(path)
	default:
		return "", fmt.Errorf("unsupported backup method: %v", method)
	}
}

func findNextNumberedName(path string) (string, error) {
	// We need to find the next available N.
	// We can search the directory for files matching path.~N~
	// Optimization: start from 1 and go up? Or listing directory?
	// Listing directory is better for large N.

	dir := filepath.Dir(path)
	base := filepath.Base(path)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	maxN := 0
	prefix := base + ".~"
	suffix := "~"

	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, suffix) {
			// Extract N
			mid := name[len(prefix) : len(name)-len(suffix)]
			if n, err := strconv.Atoi(mid); err == nil {
				if n > maxN {
					maxN = n
				}
			}
		}
	}

	return fmt.Sprintf("%s.~%d~", path, maxN+1), nil
}

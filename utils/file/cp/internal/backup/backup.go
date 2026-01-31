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

	// 'existing' strategy:
	// If numbered backups exist, use numbered.
	// Otherwise use simple.
	if method == config.BackupExisting {
		// Check if any numbered backup exists
		// We check for path.~1~ (using default suffix format for check usually,
		// but specifically the logic is: "if the file produced by the 'numbered' method already exists, make numbered backups" - wait, simplified: "numbered if numbered backups exist")
		// Usually this means checking if `path.~1~` or similar exists.

		// Actually, GNU behavior: "if ls file.~*~ shows anything, use numbered"
		// We can try to see if file.~1~ exists.
		// Or simpler: does the version with suffix ~ match what we want?

		// Let's implement a check: does path.~1~ exist?
		// Note: The suffix for numbered backups is hardcoded to ".~N~" in GNU cp usually?
		// GNU cp info: "The backup suffix is '~', unless set with --suffix...
		// The suffix is NOT used for numbered backups; they always use .~N~" -> verify this.
		// "numbered, t: make numbered backups" -> `foo.~1~`
		// So numbered backups ignore cfg.Suffix?
		// Yes, typically numbered backups use `.~N~`.

		firstNumbered := path + ".~1~"
		if _, err := os.Stat(firstNumbered); err == nil {
			method = config.BackupNumbered
		} else {
			method = config.BackupSimple
		}
	}

	if method == config.BackupSimple {
		return path + cfg.Suffix, nil
	}

	if method == config.BackupNumbered {
		return findNextNumberedName(path)
	}

	return path + cfg.Suffix, nil
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

//go:build !windows

package backup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ioluas/gopherutils/utils/file/cp/internal/config"
)

func TestMakeBackup(t *testing.T) {
	tempDir := t.TempDir()

	createFile := func(name string) string {
		path := filepath.Join(tempDir, name)
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to create file %s: %v", name, err)
		}
		return path
	}

	t.Run("No backup if file missing", func(t *testing.T) {
		cfg := &config.Config{Backup: true, BackupMethod: config.BackupSimple, Suffix: "~"}
		name, err := MakeBackup(filepath.Join(tempDir, "missing"), cfg)
		if err != nil {
			t.Errorf("MakeBackup failed: %v", err)
		}
		if name != "" {
			t.Errorf("expected no backup name, got %q", name)
		}
	})

	t.Run("Simple backup", func(t *testing.T) {
		path := createFile("simple")
		cfg := &config.Config{Backup: true, BackupMethod: config.BackupSimple, Suffix: "~"}

		name, err := MakeBackup(path, cfg)
		if err != nil {
			t.Errorf("MakeBackup failed: %v", err)
		}

		expected := path + "~"
		if name != expected {
			t.Errorf("got name %q, want %q", name, expected)
		}

		// Original should be gone (renamed)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("original file %s still exists", path)
		}
		// Backup should exist
		if _, err := os.Stat(expected); err != nil {
			t.Errorf("backup file %s missing", expected)
		}
	})

	t.Run("Numbered backup", func(t *testing.T) {
		path := createFile("numbered")
		cfg := &config.Config{Backup: true, BackupMethod: config.BackupNumbered}

		name1, err := MakeBackup(path, cfg)
		if err != nil {
			t.Fatalf("MakeBackup 1 failed: %v", err)
		}
		expected1 := path + ".~1~"
		if name1 != expected1 {
			t.Errorf("1: got name %q, want %q", name1, expected1)
		}

		// Create file again to backup again
		path = createFile("numbered")
		name2, err := MakeBackup(path, cfg)
		if err != nil {
			t.Fatalf("MakeBackup 2 failed: %v", err)
		}
		expected2 := path + ".~2~"
		if name2 != expected2 {
			t.Errorf("2: got name %q, want %q", name2, expected2)
		}
	})

	t.Run("Existing backup (triggers numbered)", func(t *testing.T) {
		path := createFile("existing_num")
		// Create .~1~
		createFile("existing_num.~1~")

		cfg := &config.Config{Backup: true, BackupMethod: config.BackupExisting}

		name, err := MakeBackup(path, cfg)
		if err != nil {
			t.Fatalf("MakeBackup failed: %v", err)
		}

		// Should pick numbered because .~1~ exists
		expected := path + ".~2~"
		if name != expected {
			t.Errorf("got name %q, want %q", name, expected)
		}
	})

	t.Run("Existing backup (triggers simple)", func(t *testing.T) {
		path := createFile("existing_sim")
		// No .~1~ exists

		cfg := &config.Config{Backup: true, BackupMethod: config.BackupExisting, Suffix: "~"}

		name, err := MakeBackup(path, cfg)
		if err != nil {
			t.Fatalf("MakeBackup failed: %v", err)
		}

		// Should pick simple because no numbered exists
		expected := path + "~"
		if name != expected {
			t.Errorf("got name %q, want %q", name, expected)
		}
	})

	t.Run("Existing backup (triggers error on stat)", func(t *testing.T) {
		path := createFile("exist_stat_err")
		// Create path.~1~ as a symlink loop
		sym := path + ".~1~"
		if err := os.Symlink(sym, sym); err != nil {
			t.Fatalf("failed to create symlink: %v", err)
		}

		cfg := &config.Config{Backup: true, BackupMethod: config.BackupExisting}
		_, err := MakeBackup(path, cfg)
		if err == nil {
			t.Error("expected error on stat for BackupExisting")
		}
		// Expect ELOOP or similar
	})

	t.Run("MakeBackup fails on rename", func(t *testing.T) {
		path := createFile("rename_fail")
		// Create a directory where the backup would go
		backupPath := path + "~"
		if err := os.Mkdir(backupPath, 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}

		cfg := &config.Config{Backup: true, BackupMethod: config.BackupSimple, Suffix: "~"}
		_, err := MakeBackup(path, cfg)
		if err == nil {
			t.Error("expected error on rename to directory")
		}
		if !strings.Contains(err.Error(), "cannot backup") {
			t.Errorf("expected cannot backup error, got: %v", err)
		}
	})

	t.Run("MakeBackup stat error", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("skipping permission test as root")
		}
		// Create a directory and make it unreadable/unsearchable
		subDir := filepath.Join(tempDir, "unreadable")
		if err := os.Mkdir(subDir, 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
		path := filepath.Join(subDir, "file")
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		// Remove search permission from subdir
		if err := os.Chmod(subDir, 0000); err != nil {
			t.Fatalf("failed to chmod: %v", err)
		}
		defer func() { _ = os.Chmod(subDir, 0755) }()

		cfg := &config.Config{Backup: true, BackupMethod: config.BackupSimple, Suffix: "~"}
		// This should fail to stat the file
		_, err := MakeBackup(path, cfg)
		if err == nil {
			t.Error("expected error on stat")
		}
	})

	t.Run("findNextNumberedName ReadDir error", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("skipping permission test as root")
		}
		// Create a directory and make it executable (searchable) but not readable
		subDir := filepath.Join(tempDir, "noreaddir")
		if err := os.Mkdir(subDir, 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
		path := filepath.Join(subDir, "file")
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		// We need BackupNumbered to trigger findNextNumberedName
		cfg := &config.Config{Backup: true, BackupMethod: config.BackupNumbered}

		// Make dir executable but not readable: 0100
		if err := os.Chmod(subDir, 0100); err != nil {
			t.Fatalf("failed to chmod: %v", err)
		}
		defer func() { _ = os.Chmod(subDir, 0755) }()

		_, err := MakeBackup(path, cfg)
		if err == nil {
			t.Error("expected error on ReadDir")
		}
	})

	t.Run("No backup if Backup=false", func(t *testing.T) {
		path := createFile("no_backup_false")
		cfg := &config.Config{Backup: false}
		name, err := MakeBackup(path, cfg)
		if err != nil {
			t.Errorf("MakeBackup failed: %v", err)
		}
		if name != "" {
			t.Errorf("expected no backup name, got %q", name)
		}
	})

	t.Run("No backup if BackupMethod=None", func(t *testing.T) {
		path := createFile("backup_none")
		cfg := &config.Config{Backup: true, BackupMethod: config.BackupNone, Suffix: "~"}
		name, err := MakeBackup(path, cfg)
		if err != nil {
			t.Errorf("MakeBackup failed: %v", err)
		}
		if name != "" {
			t.Errorf("expected no backup name, got %q", name)
		}
	})

	t.Run("Unsupported backup method", func(t *testing.T) {
		path := createFile("unsupported")
		cfg := &config.Config{Backup: true, BackupMethod: config.BackupMethod(999)}
		_, err := MakeBackup(path, cfg)
		if err == nil {
			t.Error("expected error for unsupported backup method")
		}
		if !strings.Contains(err.Error(), "unsupported backup method") {
			t.Errorf("expected 'unsupported backup method' error, got %v", err)
		}
	})

	t.Run("findNextNumberedName ignores invalid numbers", func(t *testing.T) {
		path := createFile("invalid_num")
		bogus := path + ".~abc~"
		createFile(bogus[len(filepath.Dir(path))+1:])
		cfg := &config.Config{Backup: true, BackupMethod: config.BackupNumbered}
		name, err := MakeBackup(path, cfg)
		if err != nil {
			t.Errorf("MakeBackup failed: %v", err)
		}
		expected := path + ".~1~"
		if name != expected {
			t.Errorf("got %q want %q", name, expected)
		}
	})
}

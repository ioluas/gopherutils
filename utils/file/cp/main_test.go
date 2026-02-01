package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ioluas/gopherutils/utils/file/cp/internal/config"
)

func TestExecute(t *testing.T) {
	tempDir := t.TempDir()

	// Helper to create a file with content
	createFile := func(name, content string) string {
		path := filepath.Join(tempDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create file %s: %v", name, err)
		}
		return path
	}

	// Helper to create a file with content and mod time
	createFileWithTime := func(name, content string, modTime time.Time) string {
		path := createFile(name, content)
		if err := os.Chtimes(path, modTime, modTime); err != nil {
			t.Fatalf("failed to chtimes %s: %v", name, err)
		}
		return path
	}

	// Helper to check file content
	checkContent := func(path, expected string) {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read file %s: %v", path, err)
		}
		if string(content) != expected {
			t.Errorf("file %s content mismatch: got %q, want %q", path, string(content), expected)
		}
	}

	t.Run("Copy file to new file", func(t *testing.T) {
		src := createFile("src1.txt", "hello")
		dst := filepath.Join(tempDir, "dst1.txt")

		out, err := runCp(src, dst)
		if err != nil {
			t.Errorf("cp failed: %v", err)
		}
		if out != "" {
			t.Errorf("unexpected output: %s", out)
		}

		checkContent(dst, "hello")
	})

	t.Run("Copy file to existing file (overwrite)", func(t *testing.T) {
		src := createFile("src2.txt", "world")
		dst := createFile("dst2.txt", "old")

		out, err := runCp(src, dst)
		if err != nil {
			t.Errorf("cp failed: %v", err)
		}
		if out != "" {
			t.Errorf("unexpected output: %s", out)
		}

		checkContent(dst, "world")
	})

	t.Run("Copy file to directory", func(t *testing.T) {
		src := createFile("src3.txt", "test")
		dstDir := filepath.Join(tempDir, "dir1")
		if err := os.Mkdir(dstDir, 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}

		out, err := runCp(src, dstDir)
		if err != nil {
			t.Errorf("cp failed: %v", err)
		}
		if out != "" {
			t.Errorf("unexpected output: %s", out)
		}

		checkContent(filepath.Join(dstDir, "src3.txt"), "test")
	})

	t.Run("Copy multiple files to directory", func(t *testing.T) {
		src1 := createFile("multi1.txt", "one")
		src2 := createFile("multi2.txt", "two")
		dstDir := filepath.Join(tempDir, "dir2")
		if err := os.Mkdir(dstDir, 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}

		out, err := runCp(src1, src2, dstDir)
		if err != nil {
			t.Errorf("cp failed: %v", err)
		}
		if out != "" {
			t.Errorf("unexpected output: %s", out)
		}

		checkContent(filepath.Join(dstDir, "multi1.txt"), "one")
		checkContent(filepath.Join(dstDir, "multi2.txt"), "two")
	})

	t.Run("Verbose output", func(t *testing.T) {
		src := createFile("verbose_src.txt", "content")
		dst := filepath.Join(tempDir, "verbose_dst.txt")

		out, err := runCp("-v", src, dst)
		if err != nil {
			t.Errorf("cp failed: %v", err)
		}

		expectedOut := fmt.Sprintf("'%s' -> '%s'\n", src, dst)
		if out != expectedOut {
			t.Errorf("unexpected output: got %q, want %q", out, expectedOut)
		}

		checkContent(dst, "content")
	})

	t.Run("Fail if multiple files and dest is not dir", func(t *testing.T) {
		src1 := createFile("fail1.txt", "one")
		src2 := createFile("fail2.txt", "two")
		dstFile := createFile("fail_dst.txt", "no")

		out, err := runCp(src1, src2, dstFile)
		if err == nil {
			t.Error("cp should have failed")
		}
		if !strings.Contains(out, "not a directory") {
			t.Errorf("unexpected error output: %s", out)
		}
	})

	t.Run("Fail source not found", func(t *testing.T) {
		dst := filepath.Join(tempDir, "dst_no.txt")
		out, err := runCp("nonexistent", dst)
		if err == nil {
			t.Error("cp should have failed")
		}
		if !strings.Contains(out, "no such file or directory") {
			t.Errorf("unexpected error output: %s", out)
		}
	})

	t.Run("Omitting directory", func(t *testing.T) {
		srcDir := filepath.Join(tempDir, "src_dir")
		if err := os.Mkdir(srcDir, 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
		dst := filepath.Join(tempDir, "dst_from_dir")

		out, err := runCp(srcDir, dst)
		if err == nil {
			t.Error("cp should have failed")
		}
		if !strings.Contains(out, "omitting directory") {
			t.Errorf("unexpected error output: %s", out)
		}
	})

	t.Run("Help flag", func(t *testing.T) {
		out, err := runCp("--help")
		if err != nil {
			t.Errorf("help failed: %v", err)
		}
		if !strings.Contains(out, "Usage: cp") {
			t.Errorf("expected usage in output: got %q", out)
		}
	})

	t.Run("Short help", func(t *testing.T) {
		out, err := runCp("-?")
		if err != nil {
			t.Errorf("short help failed: %v", err)
		}
		if !strings.Contains(out, "Usage: cp") {
			t.Errorf("expected usage in output: got %q", out)
		}
	})

	t.Run("Missing operand", func(t *testing.T) {
		out, err := runCp()
		if err == nil {
			t.Error("should fail on missing operands")
		}
		if !strings.Contains(out, "missing file operand") {
			t.Errorf("expected missing operand error: got %q", out)
		}
	})

	t.Run("Invalid flag", func(t *testing.T) {
		out, err := runCp("--invalid")
		if err == nil {
			t.Error("should fail on invalid flag")
		}
		if !strings.Contains(out, "unknown flag") {
			t.Errorf("expected unknown flag error: got %q", out)
		}
	})

	t.Run("Multiple sources to non-existing dir", func(t *testing.T) {
		src1 := createFile("multi_non1.txt", "one")
		src2 := createFile("multi_non2.txt", "two")
		dst := filepath.Join(tempDir, "nonexist")

		out, err := runCp(src1, src2, dst)
		if err == nil {
			t.Error("should fail")
		}
		if !strings.Contains(out, "nonexist' is not a directory") {
			t.Errorf("expected 'not a directory' error: got %q", out)
		}
	})

	t.Run("Fail permission denied on source", func(t *testing.T) {
		src := createFile("perm_src.txt", "content")
		if err := os.Chmod(src, 0000); err != nil {
			t.Fatalf("failed to chmod: %v", err)
		}
		// Restore permissions so we can clean up
		defer func() { _ = os.Chmod(src, 0644) }()

		dst := filepath.Join(tempDir, "perm_dst.txt")
		out, err := runCp(src, dst)
		if err == nil {
			t.Error("should fail")
		}
		if !strings.Contains(out, "permission denied") {
			t.Errorf("expected permission denied error: got %q", out)
		}
	})

	t.Run("Fail permission denied on dest directory", func(t *testing.T) {
		src := createFile("perm_dest_src.txt", "content")
		dstDir := filepath.Join(tempDir, "readonly_dir")
		if err := os.Mkdir(dstDir, 0555); err != nil { // Read/Exec but no Write
			t.Fatalf("failed to create dir: %v", err)
		}
		// chmod to writable for cleanup
		defer func() { _ = os.Chmod(dstDir, 0755) }()

		dst := filepath.Join(dstDir, "file.txt")
		out, err := runCp(src, dst)
		if err == nil {
			t.Error("should fail")
		}
		if !strings.Contains(out, "permission denied") {
			t.Errorf("expected permission denied error: got %q", out)
		}
	})

	t.Run("Fail permission denied on dest dir for multiple sources", func(t *testing.T) {
		src1 := createFile("mperm_src1.txt", "content1")
		src2 := createFile("mperm_src2.txt", "content2")
		dstDir := filepath.Join(tempDir, "mreadonly_dir")
		if err := os.Mkdir(dstDir, 0555); err != nil { // read-only dir (no write)
			t.Fatalf("failed to create dir: %v", err)
		}
		defer func() { _ = os.Chmod(dstDir, 0755) }()

		out, err := runCp(src1, src2, dstDir)
		if err == nil {
			t.Error("should fail")
		}
		if !strings.Contains(out, "cannot copy") || !strings.Contains(out, "permission denied") {
			t.Errorf("expected 'cannot copy' and 'permission denied': got %q", out)
		}
		// Verify no files created
		if _, err := os.Stat(filepath.Join(dstDir, "mperm_src1.txt")); err == nil {
			t.Error("unexpected file created")
		}
	})

	t.Run("Copy file to itself", func(t *testing.T) {
		src := createFile("self.txt", "content")
		out, err := runCp(src, src)
		if err == nil {
			t.Error("cp should fail when source and destination are the same file")
		}
		if !strings.Contains(out, "are the same file") {
			t.Errorf("expected 'are the same file' error: got %q", out)
		}
		checkContent(src, "content")
	})

	t.Run("Copy file to its hardlink", func(t *testing.T) {
		if runtime.GOOS != "linux" {
			t.Skip("Hardlinks are Linux-specific")
		}
		src := createFile("hard_src.txt", "hard content")
		linkPath := filepath.Join(tempDir, "hard_link.txt")
		if err := os.Link(src, linkPath); err != nil {
			t.Fatalf("failed to create hardlink: %v", err)
		}
		out, err := runCp(src, linkPath)
		if err == nil {
			t.Error("cp should fail when copying to hardlink")
		}
		if !strings.Contains(out, "are the same file") {
			t.Errorf("expected 'are the same file' error: got %q", out)
		}
		checkContent(src, "hard content")
	})

	t.Run("Verbose backup output", func(t *testing.T) {
		src := createFile("vbackup_src.txt", "content")
		dst := createFile("vbackup_dst.txt", "old")

		// -v -b
		out, err := runCp("-v", "-b", src, dst)
		if err != nil {
			t.Errorf("cp failed: %v", err)
		}

		// Expect: 'src' -> 'dst' (backup: 'dst~')
		expectedPart := fmt.Sprintf("'%s' -> '%s' (backup: '%s~')", src, dst, dst)
		if !strings.Contains(out, expectedPart) {
			t.Errorf("expected verbose backup output containing %q, got %q", expectedPart, out)
		}
	})

	t.Run("Preserve file permissions", func(t *testing.T) {
		src := filepath.Join(tempDir, "perm_check_src.txt")
		if err := os.WriteFile(src, []byte("content"), 0777); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
		// os.WriteFile uses 0666 before umask, so let's explicit chmod
		if err := os.Chmod(src, 0755); err != nil {
			t.Fatalf("failed to chmod src: %v", err)
		}

		dst := filepath.Join(tempDir, "perm_check_dst.txt")
		_, err := runCp(src, dst)
		if err != nil {
			t.Errorf("cp failed: %v", err)
		}

		fi, err := os.Stat(dst)
		if err != nil {
			t.Fatalf("failed to stat dst: %v", err)
		}
		gotPerm := fi.Mode().Perm()
		wantPerm := os.FileMode(0755)
		if gotPerm != wantPerm {
			t.Errorf("permissions not preserved: want 0%o, got 0%o", wantPerm, gotPerm)
		}
	})

	t.Run("Multiple sources to non-directory dest", func(t *testing.T) {
		src1 := createFile("multi1.txt", "1")
		src2 := createFile("multi2.txt", "2")
		dst := createFile("not_dir.txt", "old")
		out, err := runCp(src1, src2, dst)
		if err == nil || !strings.Contains(out, "not a directory") {
			t.Errorf("expected not a directory error")
		}
	})

	t.Run("Multiple sources to non-existing dest", func(t *testing.T) {
		src1 := createFile("multi3.txt", "3")
		src2 := createFile("multi4.txt", "4")
		dst := filepath.Join(tempDir, "noexist_dir")
		out, err := runCp(src1, src2, dst)
		if err == nil || !strings.Contains(out, "not a directory") {
			t.Errorf("expected not a directory for nonexist")
		}
	})

	t.Run("Multiple sources to directory dest", func(t *testing.T) {
		src1 := createFile("multi_src1.txt", "a")
		src2 := createFile("multi_src2.txt", "b")
		dstDir := filepath.Join(tempDir, "multi_dst_dir")
		if err := os.Mkdir(dstDir, 0755); err != nil {
			t.Fatalf("mkdir failed: %v", err)
		}
		_, err := runCp(src1, src2, dstDir)
		if err != nil {
			t.Errorf("cp failed: %v", err)
		}
		checkContent(filepath.Join(dstDir, filepath.Base(src1)), "a")
		checkContent(filepath.Join(dstDir, filepath.Base(src2)), "b")
	})

	t.Run("Update mode none", func(t *testing.T) {
		src := createFile("up_none_src.txt", "new")
		dst := createFile("up_none_dst.txt", "old")

		_, err := runCp("--update=none", src, dst)
		if err != nil {
			t.Errorf("cp failed: %v", err)
		}
		// Content should be OLD
		checkContent(dst, "old")
	})

	t.Run("Update mode none-fail", func(t *testing.T) {
		src := createFile("up_fail_src.txt", "new")
		dst := createFile("up_fail_dst.txt", "old")

		out, err := runCp("--update=none-fail", src, dst)
		if err == nil {
			t.Error("expected error")
		}
		if !strings.Contains(out, "file exists") {
			t.Errorf("expected file exists error, got %q", out)
		}
		checkContent(dst, "old")
	})

	t.Run("Update mode older (skip)", func(t *testing.T) {
		now := time.Now()
		oldTime := now.Add(-1 * time.Hour)
		newTime := now

		src := createFileWithTime("up_older_src_old.txt", "src_content", oldTime)
		dst := createFileWithTime("up_older_dst_new.txt", "dst_content", newTime)

		// src is older than dst, so default update (older) should SKIP
		_, err := runCp("--update", src, dst) // default is older
		if err != nil {
			t.Errorf("cp failed: %v", err)
		}
		checkContent(dst, "dst_content")
	})

	t.Run("Update mode older (overwrite)", func(t *testing.T) {
		now := time.Now()
		oldTime := now.Add(-1 * time.Hour)
		newTime := now

		src := createFileWithTime("up_older_src_new.txt", "src_content", newTime)
		dst := createFileWithTime("up_older_dst_old.txt", "dst_content", oldTime)

		// src is newer than dst, so update should OVERWRITE
		_, err := runCp("--update", src, dst)
		if err != nil {
			t.Errorf("cp failed: %v", err)
		}
		checkContent(dst, "src_content")
	})

	t.Run("Update mode all (overwrite)", func(t *testing.T) {
		src := createFile("up_all_src.txt", "new")
		dst := createFile("up_all_dst.txt", "old")

		_, err := runCp("--update=all", src, dst)
		if err != nil {
			t.Errorf("cp failed: %v", err)
		}
		checkContent(dst, "new")
	})

	t.Run("Attributes only (new file)", func(t *testing.T) {
		src := createFile("attr_only_src.txt", "content")
		dst := filepath.Join(tempDir, "attr_only_dst.txt")

		_, err := runCp("--attributes-only", src, dst)
		if err != nil {
			t.Errorf("cp failed: %v", err)
		}

		// File should exist but be empty
		checkContent(dst, "")
	})

	t.Run("Attributes only (existing file)", func(t *testing.T) {
		src := createFile("attr_only_ex_src.txt", "new_content")
		dst := createFile("attr_only_ex_dst.txt", "old_content")

		_, err := runCp("--attributes-only", src, dst)
		if err != nil {
			t.Errorf("cp failed: %v", err)
		}

		// File should still have OLD content
		checkContent(dst, "old_content")
	})

	t.Run("Attributes only with preserve (-p)", func(t *testing.T) {
		src := createFile("attr_pres_src.txt", "content")
		// Set weird mode and old time
		oldTime := time.Now().Add(-24 * time.Hour)
		if err := os.Chtimes(src, oldTime, oldTime); err != nil {
			t.Fatalf("chtimes failed: %v", err)
		}
		if err := os.Chmod(src, 0700); err != nil {
			t.Fatalf("chmod failed: %v", err)
		}

		dst := filepath.Join(tempDir, "attr_pres_dst.txt")
		_, err := runCp("--attributes-only", "-p", src, dst)
		if err != nil {
			t.Errorf("cp failed: %v", err)
		}

		// File should be empty
		checkContent(dst, "")

		fiDst, err := os.Stat(dst)
		if err != nil {
			t.Fatalf("stat dst failed: %v", err)
		}

		// Check Mode
		if fiDst.Mode().Perm() != 0700 {
			t.Errorf("expected mode 0700, got %v", fiDst.Mode().Perm())
		}
		// Check Time
		if fiDst.ModTime().Sub(oldTime).Abs() > time.Second {
			t.Errorf("expected time %v, got %v", oldTime, fiDst.ModTime())
		}
	})

	t.Run("Numbered backup", func(t *testing.T) {
		src := createFile("num_src.txt", "new")
		dst := createFile("num_dst.txt", "old")
		// Create an existing numbered backup to trigger BackupExisting -> Numbered
		createFile("num_dst.txt.~1~", "bak1")

		_, err := runCp("--backup", src, dst)
		if err != nil {
			t.Errorf("cp failed: %v", err)
		}

		checkContent(dst, "new")
		checkContent(dst+".~2~", "old")
	})

	t.Run("Simple backup with custom suffix", func(t *testing.T) {
		src := createFile("suffix_src.txt", "new")
		dst := createFile("suffix_dst.txt", "old")

		_, err := runCp("--backup", "--suffix=.bak", src, dst)
		if err != nil {
			t.Errorf("cp failed: %v", err)
		}

		checkContent(dst, "new")
		checkContent(dst+".bak", "old")
	})

	t.Run("Multiple sources to non-existent target (not dir)", func(t *testing.T) {
		src1 := createFile("m_err_1.txt", "1")
		src2 := createFile("m_err_2.txt", "2")
		dst := filepath.Join(tempDir, "not_a_dir_yet")

		out, err := runCp(src1, src2, dst)
		if err == nil {
			t.Error("should fail")
		}
		if !strings.Contains(out, "is not a directory") {
			t.Errorf("expected 'is not a directory' error, got %q", out)
		}
	})

	t.Run("Multiple sources stat error", func(t *testing.T) {
		// This is hard to trigger without specific OS errors,
		// but we can try a path that might be invalid or inaccessible.
		src1 := createFile("m_stat_1.txt", "1")
		src2 := createFile("m_stat_2.txt", "2")

		// Use a path that is a file where we expect a directory
		dst := createFile("m_stat_dst", "content")

		out, err := runCp(src1, src2, dst)
		if err == nil {
			t.Error("should fail")
		}
		if !strings.Contains(out, "is not a directory") {
			t.Errorf("expected 'is not a directory' error, got %q", out)
		}
	})

	t.Run("Backup with VERSION_CONTROL env", func(t *testing.T) {
		src := createFile("vc_src.txt", "new")
		dst := createFile("vc_dst.txt", "old")

		if err := os.Setenv("VERSION_CONTROL", "numbered"); err != nil {
			t.Fatalf("failed to set env: %v", err)
		}
		defer func() {
			if err := os.Unsetenv("VERSION_CONTROL"); err != nil {
				t.Errorf("failed to unset env: %v", err)
			}
		}()

		_, err := runCp("-b", src, dst)
		if err != nil {
			t.Errorf("cp failed: %v", err)
		}

		checkContent(dst, "new")
		checkContent(dst+".~1~", "old")
	})

	t.Run("Backup with invalid argument", func(t *testing.T) {
		src := createFile("inv_bak_src.txt", "new")
		dst := createFile("inv_bak_dst.txt", "old")

		out, err := runCp("--backup=invalid", src, dst)
		if err == nil {
			t.Error("should fail on invalid backup method")
		}
		if !strings.Contains(out, "invalid argument 'invalid' for '--backup'") {
			t.Errorf("expected invalid argument error, got %q", out)
		}
	})

	t.Run("Backup with SIMPLE_BACKUP_SUFFIX env", func(t *testing.T) {
		src := createFile("env_suffix_src.txt", "new")
		dst := createFile("env_suffix_dst.txt", "old")

		if err := os.Setenv("SIMPLE_BACKUP_SUFFIX", ".bak_env"); err != nil {
			t.Fatalf("failed to set env: %v", err)
		}
		defer func() {
			if err := os.Unsetenv("SIMPLE_BACKUP_SUFFIX"); err != nil {
				t.Errorf("failed to unset env: %v", err)
			}
		}()

		_, err := runCp("-b", src, dst)
		if err != nil {
			t.Errorf("cp failed: %v", err)
		}

		checkContent(dst, "new")
		checkContent(dst+".bak_env", "old")
	})

	t.Run("Update mode with invalid argument", func(t *testing.T) {
		src := createFile("inv_up_src.txt", "new")
		dst := createFile("inv_up_dst.txt", "old")

		out, err := runCp("--update=invalid", src, dst)
		if err == nil {
			t.Error("should fail on invalid update mode")
		}
		if !strings.Contains(out, "invalid argument 'invalid' for '--update'") {
			t.Errorf("expected invalid argument error, got %q", out)
		}
	})

	t.Run("Multiple sources stat permission error", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("skipping permission test as root")
		}
		src1 := createFile("msp_1.txt", "1")
		src2 := createFile("msp_2.txt", "2")

		badDir := filepath.Join(tempDir, "no_access")
		if err := os.Mkdir(badDir, 0000); err != nil {
			t.Fatalf("mkdir failed: %v", err)
		}
		defer func() { _ = os.Chmod(badDir, 0755) }()

		// dst inside badDir. os.Stat(dst) should fail with permission denied (not IsNotExist)
		dst := filepath.Join(badDir, "target")

		out, err := runCp(src1, src2, dst)
		if err == nil {
			t.Error("should fail")
		}
		// Expect "cannot stat ... permission denied"
		if !strings.Contains(out, "cannot stat") {
			t.Errorf("expected 'cannot stat', got %q", out)
		}
	})

	t.Run("Backup failure execution", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("skipping permission test as root")
		}

		dir := filepath.Join(tempDir, "readonly_backup")
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatalf("mkdir failed: %v", err)
		}

		src := filepath.Join(dir, "src")
		if err := os.WriteFile(src, []byte("new"), 0644); err != nil {
			t.Fatalf("write src failed: %v", err)
		}

		dst := filepath.Join(dir, "dst")
		if err := os.WriteFile(dst, []byte("old"), 0644); err != nil {
			t.Fatalf("write dst failed: %v", err)
		}

		// Make directory read-only (no write) so rename fails
		if err := os.Chmod(dir, 0555); err != nil {
			t.Fatalf("chmod failed: %v", err)
		}
		defer func() { _ = os.Chmod(dir, 0755) }()

		// cp -b src dst
		// Should try to rename dst to dst~, which fails because dir is RO.
		_, err := runCp("-b", src, dst)
		if err == nil {
			t.Error("should fail due to backup failure")
		}
	})

	t.Run("No clobber (-n)", func(t *testing.T) {
		src := createFile("n_src.txt", "new")
		dst := createFile("n_dst.txt", "old")

		_, err := runCp("-n", src, dst)
		if err != nil {
			t.Errorf("cp failed: %v", err)
		}

		// Should NOT overwrite
		checkContent(dst, "old")
	})

	t.Run("No clobber (long flag)", func(t *testing.T) {
		src := createFile("n_long_src.txt", "new")
		dst := createFile("n_long_dst.txt", "old")

		_, err := runCp("--no-clobber", src, dst)
		if err != nil {
			t.Errorf("cp failed: %v", err)
		}

		checkContent(dst, "old")
	})

	t.Run("No clobber overridden by update", func(t *testing.T) {
		src := createFile("n_over_src.txt", "new")
		dst := createFile("n_over_dst.txt", "old")

		// -n then --update=all -> update wins (overwrite)
		_, err := runCp("-n", "--update=all", src, dst)
		if err != nil {
			t.Errorf("cp failed: %v", err)
		}

		checkContent(dst, "new")
	})

	t.Run("Update overridden by no clobber", func(t *testing.T) {
		src := createFile("u_over_src.txt", "new")
		dst := createFile("u_over_dst.txt", "old")

		// --update=all then -n -> no clobber wins (skip)
		_, err := runCp("--update=all", "-n", src, dst)
		if err != nil {
			t.Errorf("cp failed: %v", err)
		}

		checkContent(dst, "old")
	})

	t.Run("Preserve mode and timestamps (-p)", func(t *testing.T) {
		src := createFile("pres_src.txt", "content")
		// Set weird mode and old time
		oldTime := time.Now().Add(-24 * time.Hour)
		if err := os.Chtimes(src, oldTime, oldTime); err != nil {
			t.Fatalf("chtimes failed: %v", err)
		}
		if err := os.Chmod(src, 0700); err != nil {
			t.Fatalf("chmod failed: %v", err)
		}

		dst := filepath.Join(tempDir, "pres_dst.txt")
		_, err := runCp("-p", src, dst)
		if err != nil {
			t.Errorf("cp failed: %v", err)
		}

		fiDst, err := os.Stat(dst)
		if err != nil {
			t.Fatalf("stat dst failed: %v", err)
		}

		// Check Mode
		if fiDst.Mode().Perm() != 0700 {
			t.Errorf("expected mode 0700, got %v", fiDst.Mode().Perm())
		}
		// Check Time (allow small delta)
		if fiDst.ModTime().Sub(oldTime).Abs() > time.Second {
			t.Errorf("expected time %v, got %v", oldTime, fiDst.ModTime())
		}
	})

	t.Run("Preserve links", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("skipping hard link test on windows")
		}
		// Create src1
		src1 := createFile("link_src1.txt", "content")
		// Create src2 hardlinked to src1
		src2 := filepath.Join(tempDir, "link_src2.txt")
		if err := os.Link(src1, src2); err != nil {
			t.Fatalf("link failed: %v", err)
		}

		dstDir := filepath.Join(tempDir, "link_dst_dir")
		if err := os.Mkdir(dstDir, 0755); err != nil {
			t.Fatalf("mkdir failed: %v", err)
		}

		// cp --preserve=links src1 src2 dstDir/
		_, err := runCp("--preserve=links", src1, src2, dstDir)
		if err != nil {
			t.Errorf("cp failed: %v", err)
		}

		// Check that dst1 and dst2 are hardlinked
		dst1 := filepath.Join(dstDir, "link_src1.txt")
		dst2 := filepath.Join(dstDir, "link_src2.txt")

		fi1, err := os.Stat(dst1)
		if err != nil {
			t.Fatalf("stat dst1 failed: %v", err)
		}
		fi2, err := os.Stat(dst2)
		if err != nil {
			t.Fatalf("stat dst2 failed: %v", err)
		}

		if !sameDevIno(fi1, fi2) {
			t.Errorf("dst1 and dst2 are not hardlinked (inodes differ)")
		}
	})

	t.Run("Preserve links respects no-clobber", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("skipping hard link test on windows")
		}
		// Create src1 and its hardlink src2
		src1 := createFile("nlink_src1.txt", "content")
		src2 := filepath.Join(tempDir, "nlink_src2.txt")
		if err := os.Link(src1, src2); err != nil {
			t.Fatalf("link failed: %v", err)
		}

		// Create dst1 and dst2. dst2 already exists with DIFFERENT content.
		dstDir := filepath.Join(tempDir, "nlink_dst_dir")
		if err := os.Mkdir(dstDir, 0755); err != nil {
			t.Fatalf("mkdir failed: %v", err)
		}
		dst1 := filepath.Join(dstDir, "nlink_src1.txt") // will be created
		dst2 := filepath.Join(dstDir, "nlink_src2.txt")
		if err := os.WriteFile(dst2, []byte("pre-existing"), 0644); err != nil {
			t.Fatalf("write failed: %v", err)
		}

		// cp -n --preserve=links src1 src2 dstDir/
		// src1 -> dst1 (created)
		// src2 -> dst2 (skipped because exists and -n)
		_, err := runCp("-n", "--preserve=links", src1, src2, dstDir)
		if err != nil {
			t.Errorf("cp failed: %v", err)
		}

		// Verify dst1 has "content"
		checkContent(dst1, "content")
		// Verify dst2 still has "pre-existing"
		checkContent(dst2, "pre-existing")

		// Verify they are NOT hardlinked
		fi1, _ := os.Stat(dst1)
		fi2, _ := os.Stat(dst2)
		if sameDevIno(fi1, fi2) {
			t.Errorf("dst1 and dst2 should NOT be hardlinked")
		}
	})

	t.Run("Preserve links with backup", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("skipping hard link test on windows")
		}
		// Create src1 and its hardlink src2
		src1 := createFile("blink_src1.txt", "content")
		src2 := filepath.Join(tempDir, "blink_src2.txt")
		if err := os.Link(src1, src2); err != nil {
			t.Fatalf("link failed: %v", err)
		}

		dstDir := filepath.Join(tempDir, "blink_dst_dir")
		if err := os.Mkdir(dstDir, 0755); err != nil {
			t.Fatalf("mkdir failed: %v", err)
		}

		// Create existing destination files to trigger backup
		dst1 := filepath.Join(dstDir, "blink_src1.txt")
		if err := os.WriteFile(dst1, []byte("old1"), 0644); err != nil {
			t.Fatalf("write dst1 failed: %v", err)
		}
		dst2 := filepath.Join(dstDir, "blink_src2.txt")
		if err := os.WriteFile(dst2, []byte("old2"), 0644); err != nil {
			t.Fatalf("write dst2 failed: %v", err)
		}

		// cp --preserve=links --backup src1 src2 dstDir/
		_, err := runCp("--preserve=links", "--backup", src1, src2, dstDir)
		if err != nil {
			t.Errorf("cp failed: %v", err)
		}

		// Verify content
		checkContent(dst1, "content")
		checkContent(dst2, "content")

		// Verify hard link
		fi1, _ := os.Stat(dst1)
		fi2, _ := os.Stat(dst2)
		if !sameDevIno(fi1, fi2) {
			t.Errorf("dst1 and dst2 should be hardlinked")
		}

		// Verify backups exist
		// Default backup is simple (~ suffix)
		checkContent(dst1+"~", "old1")
		checkContent(dst2+"~", "old2")
	})
}

type mockFileInfo struct {
	os.FileInfo
	sys interface{}
}

func (m mockFileInfo) Sys() interface{} { return m.sys }

func TestSameDevIno(t *testing.T) {
	tempDir := t.TempDir()
	f := filepath.Join(tempDir, "f")
	if err := os.WriteFile(f, nil, 0666); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	fi, err := os.Stat(f)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}

	// Mock that returns wrong Sys type
	badFi := &mockFileInfo{FileInfo: fi, sys: "not stat_t"}

	// Test s1 fail
	if sameDevIno(badFi, fi) {
		t.Error("expected false when s1 has invalid Sys")
	}
	// Test s2 fail
	if sameDevIno(fi, badFi) {
		t.Error("expected false when s2 has invalid Sys")
	}
}

func TestGetDevIno(t *testing.T) {
	tempDir := t.TempDir()
	f := filepath.Join(tempDir, "f_devino")
	if err := os.WriteFile(f, nil, 0666); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	fi, err := os.Stat(f)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}

	// Good case
	_, ok := getDevIno(fi)
	if !ok {
		t.Error("expected ok for valid file info")
	}

	// Bad case
	badFi := &mockFileInfo{FileInfo: fi, sys: "not stat_t"}
	_, ok = getDevIno(badFi)
	if ok {
		t.Error("expected not ok for invalid sys")
	}
}

func TestPreserveAttributes_Errors(t *testing.T) {
	tempDir := t.TempDir()
	f := filepath.Join(tempDir, "f_attr")
	if err := os.WriteFile(f, nil, 0666); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	fi, err := os.Stat(f)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}

	// Test Chmod error (non-existent file)
	cfgMode := config.PreserveOptions{Mode: true}
	if err := preserveAttributes(filepath.Join(tempDir, "non_existent"), fi, cfgMode); err == nil {
		t.Error("expected error for non-existent file on chmod")
	}

	// Test Chtimes error
	cfgTime := config.PreserveOptions{Timestamps: true}
	if err := preserveAttributes(filepath.Join(tempDir, "non_existent"), fi, cfgTime); err == nil {
		t.Error("expected error for non-existent file on chtimes")
	}

	// Test Chown error
	// We need a Sys() that returns valid stat_t but we can force chown failure?
	// Or just use non-existent file again, Chown also fails on non-existent.
	cfgOwn := config.PreserveOptions{Ownership: true}
	if err := preserveAttributes(filepath.Join(tempDir, "non_existent"), fi, cfgOwn); err == nil {
		t.Error("expected error for non-existent file on chown")
	}
}

func TestCopyFile_FailOpen(t *testing.T) {
	// Trigger error in os.Open(src)
	src := "non_existent_src"
	dst := "dst"
	err := copyFile(src, dst, &config.Config{}, make(map[devIno]string), nil)
	if err == nil {
		t.Error("expected error opening non-existent source")
	}
}

func runCp(args ...string) (string, error) {
	var stdout, stderr bytes.Buffer
	code := Execute(args, &stdout, &stderr)
	if code != 0 {
		return stderr.String(), fmt.Errorf("exit code %d", code)
	}
	// Return combined output for checking help messages (which print to stderr but exit 0)
	// and regular output (stdout)
	return stdout.String() + stderr.String(), nil
}

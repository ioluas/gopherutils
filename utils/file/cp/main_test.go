package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
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

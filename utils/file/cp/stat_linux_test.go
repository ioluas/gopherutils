//go:build linux

package main

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestGetAccessTime_Linux(t *testing.T) {
	tempDir := t.TempDir()
	f := filepath.Join(tempDir, "test_atime")
	if err := os.WriteFile(f, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	fi, err := os.Stat(f)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}

	atime := getAccessTime(fi)
	if atime.IsZero() {
		t.Error("expected non-zero access time")
	}

	// Verify against raw stat to ensure we are pulling Atim
	stat, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		t.Fatal("failed to cast sys to syscall.Stat_t")
	}
	expected := time.Unix(int64(stat.Atim.Sec), int64(stat.Atim.Nsec))
	if !atime.Equal(expected) {
		t.Errorf("got %v, want %v", atime, expected)
	}
}

type mockFileInfoNoSys struct {
	os.FileInfo
}

func (m mockFileInfoNoSys) Sys() interface{} { return nil }

func TestGetAccessTime_Fallback(t *testing.T) {
	tempDir := t.TempDir()
	f := filepath.Join(tempDir, "test_fallback")
	if err := os.WriteFile(f, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	fi, err := os.Stat(f)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}

	mockFi := mockFileInfoNoSys{FileInfo: fi}
	atime := getAccessTime(mockFi)

	if !atime.Equal(fi.ModTime()) {
		t.Errorf("expected ModTime %v, got %v", fi.ModTime(), atime)
	}
}

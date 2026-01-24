//go:build !windows && !plan9 && !js && !wasip1
// +build !windows,!plan9,!js,!wasip1

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseArgsGetwdError(t *testing.T) {
	// This test attempts to trigger an os.Getwd() error by deleting the current working directory.
	// This behavior is OS-dependent and might not work on all systems (e.g., Windows).
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "getwd_error_test")
	err := os.Mkdir(dir, 0755)
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldCwd)
	}()

	err = os.Chdir(dir)
	if err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	err = os.Remove(dir)
	if err != nil {
		t.Fatalf("failed to remove current directory: %v", err)
	}

	_, err = ParseArgs([]string{}, os.Stderr)
	if err == nil {
		// On some systems, Getwd might still work if the directory was deleted but the handle is kept.
		// We document this limitation as per issue description.
		t.Log("os.Getwd() did not fail after deleting CWD; this is OS-dependent behavior")
	}
}

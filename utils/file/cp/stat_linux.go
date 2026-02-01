//go:build linux

package main

import (
	"os"
	"syscall"
	"time"
)

func getAccessTime(fi os.FileInfo) time.Time {
	if stat, ok := fi.Sys().(*syscall.Stat_t); ok {
		return time.Unix(int64(stat.Atim.Sec), int64(stat.Atim.Nsec))
	}
	return fi.ModTime()
}

//go:build darwin || freebsd || openbsd || netbsd

package main

import (
	"os"
	"syscall"
	"time"
)

func getAccessTime(fi os.FileInfo) time.Time {
	if stat, ok := fi.Sys().(*syscall.Stat_t); ok {
		return time.Unix(int64(stat.Atimespec.Sec), int64(stat.Atimespec.Nsec))
	}
	return fi.ModTime()
}

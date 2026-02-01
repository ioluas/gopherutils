//go:build !linux && !darwin && !freebsd && !openbsd && !netbsd

package main

import (
	"os"
	"time"
)

func getAccessTime(fi os.FileInfo) time.Time {
	return fi.ModTime()
}

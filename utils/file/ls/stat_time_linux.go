//go:build linux

package main

import (
	"syscall"
	"time"
)

func statAtime(stat *syscall.Stat_t) time.Time {
	return time.Unix(int64(stat.Atim.Sec), int64(stat.Atim.Nsec))
}

func statCtime(stat *syscall.Stat_t) time.Time {
	return time.Unix(int64(stat.Ctim.Sec), int64(stat.Ctim.Nsec))
}

func statBirthtime(stat *syscall.Stat_t) (time.Time, bool) {
	return time.Time{}, false
}

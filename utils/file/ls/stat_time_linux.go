//go:build linux

package main

import (
	"syscall"
	"time"
)

func statAtime(stat *syscall.Stat_t) time.Time {
	return time.Unix(stat.Atim.Sec, stat.Atim.Nsec)
}

func statCtime(stat *syscall.Stat_t) time.Time {
	return time.Unix(stat.Ctim.Sec, stat.Ctim.Nsec)
}

func statBirthtime(stat *syscall.Stat_t) (time.Time, bool) {
	return time.Time{}, false
}

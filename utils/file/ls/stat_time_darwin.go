//go:build darwin

package main

import (
	"syscall"
	"time"
)

func statAtime(stat *syscall.Stat_t) time.Time {
	return time.Unix(int64(stat.Atimespec.Sec), int64(stat.Atimespec.Nsec))
}

func statCtime(stat *syscall.Stat_t) time.Time {
	return time.Unix(int64(stat.Ctimespec.Sec), int64(stat.Ctimespec.Nsec))
}

func statBirthtime(stat *syscall.Stat_t) (time.Time, bool) {
	return time.Unix(int64(stat.Birthtimespec.Sec), int64(stat.Birthtimespec.Nsec)), true
}

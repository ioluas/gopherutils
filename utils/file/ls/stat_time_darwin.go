//go:build darwin

package main

import (
	"syscall"
	"time"
)

func statAtime(stat *syscall.Stat_t) time.Time {
	return time.Unix(stat.Atimespec.Sec, stat.Atimespec.Nsec)
}

func statCtime(stat *syscall.Stat_t) time.Time {
	return time.Unix(stat.Ctimespec.Sec, stat.Ctimespec.Nsec)
}

func statBirthtime(stat *syscall.Stat_t) (time.Time, bool) {
	return time.Unix(stat.Birthtimespec.Sec, stat.Birthtimespec.Nsec), true
}

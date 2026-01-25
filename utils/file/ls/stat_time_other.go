//go:build !linux && !darwin

package main

import (
	"syscall"
	"time"
)

func statAtime(stat *syscall.Stat_t) time.Time {
	return time.Time{}
}

func statCtime(stat *syscall.Stat_t) time.Time {
	return time.Time{}
}

func statBirthtime(stat *syscall.Stat_t) (time.Time, bool) {
	return time.Time{}, false
}

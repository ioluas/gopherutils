//go:build darwin

package timeutil

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
	if stat.Birthtimespec.Sec == 0 && stat.Birthtimespec.Nsec == 0 {
		return time.Time{}, false
	}
	return time.Unix(int64(stat.Birthtimespec.Sec), int64(stat.Birthtimespec.Nsec)), true
}

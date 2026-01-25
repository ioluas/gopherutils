//go:build linux

package timeutil

import (
	"syscall"
	"testing"
	"time"
)

func TestStatTimeLinux(t *testing.T) {
	atime := time.Unix(1, 2)
	ctime := time.Unix(3, 4)
	stat := &syscall.Stat_t{
		Atim: syscall.Timespec{Sec: int64(atime.Unix()), Nsec: int64(atime.Nanosecond())},
		Ctim: syscall.Timespec{Sec: int64(ctime.Unix()), Nsec: int64(ctime.Nanosecond())},
	}

	if got := statAtime(stat); !got.Equal(atime) {
		t.Fatalf("statAtime = %v, want %v", got, atime)
	}
	if got := statCtime(stat); !got.Equal(ctime) {
		t.Fatalf("statCtime = %v, want %v", got, ctime)
	}
	if got, ok := statBirthtime(stat); ok || !got.IsZero() {
		t.Fatalf("expected no birthtime on linux, got %v ok=%v", got, ok)
	}
}

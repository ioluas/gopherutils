//go:build linux

package main

import (
	"syscall"
	"testing"
	"time"
)

func TestGetEntryTimeLinuxAccessChange(t *testing.T) {
	atime := time.Unix(10, 20)
	ctime := time.Unix(30, 40)
	info := &mockFileInfo{
		modTime: time.Unix(50, 60),
		sys: &syscall.Stat_t{
			Atim: syscall.Timespec{Sec: atime.Unix(), Nsec: int64(atime.Nanosecond())},
			Ctim: syscall.Timespec{Sec: ctime.Unix(), Nsec: int64(ctime.Nanosecond())},
		},
	}

	if got := getEntryTime(info, timeFieldAccess); !got.Equal(atime) {
		t.Fatalf("getEntryTime access = %v, want %v", got, atime)
	}
	if got := getEntryTime(info, timeFieldChange); !got.Equal(ctime) {
		t.Fatalf("getEntryTime change = %v, want %v", got, ctime)
	}
	if got := getEntryTime(info, timeFieldBirth); !got.Equal(info.modTime) {
		t.Fatalf("getEntryTime birth = %v, want fallback %v", got, info.modTime)
	}
}

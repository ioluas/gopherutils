//go:build darwin

package timeutil

import (
	"syscall"
	"testing"
	"time"
)

func TestStatTimeDarwin(t *testing.T) {
	atime := time.Unix(1, 2)
	ctime := time.Unix(3, 4)
	btime := time.Unix(5, 6)
	stat := &syscall.Stat_t{
		Atimespec:     syscall.Timespec{Sec: int64(atime.Unix()), Nsec: int32(int64(atime.Nanosecond()))},
		Ctimespec:     syscall.Timespec{Sec: int64(ctime.Unix()), Nsec: int32(int64(ctime.Nanosecond()))},
		Birthtimespec: syscall.Timespec{Sec: int64(btime.Unix()), Nsec: int32(int64(btime.Nanosecond()))},
	}

	if got := statAtime(stat); !got.Equal(atime) {
		t.Fatalf("statAtime = %v, want %v", got, atime)
	}
	if got := statCtime(stat); !got.Equal(ctime) {
		t.Fatalf("statCtime = %v, want %v", got, ctime)
	}
	if got, ok := statBirthtime(stat); !ok || !got.Equal(btime) {
		t.Fatalf("statBirthtime = %v ok=%v, want %v ok=true", got, ok, btime)
	}

	t.Run("zero_birthtime", func(t *testing.T) {
		statZero := &syscall.Stat_t{
			Atimespec:     syscall.Timespec{Sec: int64(atime.Unix()), Nsec: int32(atime.Nanosecond())},
			Ctimespec:     syscall.Timespec{Sec: int64(ctime.Unix()), Nsec: int32(ctime.Nanosecond())},
			Birthtimespec: syscall.Timespec{Sec: 0, Nsec: 0},
		}
		got, ok := statBirthtime(statZero)
		if ok || !got.IsZero() {
			t.Fatalf("statBirthtime zero: got %v ok=%v", got, ok)
		}
	})
}

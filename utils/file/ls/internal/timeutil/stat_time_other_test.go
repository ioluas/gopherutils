//go:build !linux && !darwin

package timeutil

import (
	"syscall"
	"testing"
)

func TestStatTimeOther(t *testing.T) {
	stat := &syscall.Stat_t{}
	if got := statAtime(stat); !got.IsZero() {
		t.Fatalf("statAtime expected zero, got %v", got)
	}
	if got := statCtime(stat); !got.IsZero() {
		t.Fatalf("statCtime expected zero, got %v", got)
	}
	if got, ok := statBirthtime(stat); ok || !got.IsZero() {
		t.Fatalf("statBirthtime expected zero/false, got %v ok=%v", got, ok)
	}
}

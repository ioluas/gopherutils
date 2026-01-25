package size

import (
	"fmt"
	"github.com/ioluas/gopherutils/utils/file/ls/internal/config"
	"strings"
	"testing"
)

func TestFormatSize(t *testing.T) {
	if got := FormatSize(100, 1024); got != "100" {
		t.Fatalf("Expected 100, got %s", got)
	}
}

func TestParseBlockSizeEdgeCases(t *testing.T) {
	// Test case where nonDigitIdx is 0
	spec, warn, ok := ParseBlockSize("K")
	if !ok || warn != "" {
		t.Fatalf("Expected 'K' to be a valid block size")
	}
	if spec.SizeBytes != 1024 {
		t.Fatalf("Expected size to be 1024, got %d", spec.SizeBytes)
	}

	// Test case where numStr is empty
	spec, warn, ok = ParseBlockSize("K")
	if !ok || warn != "" {
		t.Fatalf("Expected 'K' to be a valid block size")
	}
	if spec.SizeBytes != 1024 {
		t.Fatalf("Expected size to be 1024, got %d", spec.SizeBytes)
	}
	if !spec.ShowSuffix {
		t.Fatalf("Expected ShowSuffix to be true")
	}

	// Test case where num is 0
	_, warn, ok = ParseBlockSize("0K")
	if ok || !strings.Contains(warn, "invalid SIZE number") {
		t.Fatalf("Expected error for 0K, got warn: %s", warn)
	}
}

func TestFormatSizeWithBlockSpecZeroSize(t *testing.T) {
	spec := config.BlockSizeSpec{Mode: config.BlockSizeModeBytes, SizeBytes: 1024}
	if got := FormatSizeWithBlockSpec(0, spec); got != "0" {
		t.Fatalf("got %q, want %q", got, "0")
	}
}
func TestParseBlockSize_InvalidSuffix(t *testing.T) {
	_, warning, ok := ParseBlockSize("1foo")
	if ok {
		t.Error("Expected parsing to fail for invalid suffix 'foo'")
	}
	if warning != "unknown SIZE suffix" {
		t.Errorf("Expected warning 'unknown SIZE suffix', got '%s'", warning)
	}
}
func TestFormatSize_LargeSize(t *testing.T) {
	// 1EB in bytes
	largeSize := int64(1152921504606846976)
	expected := "1.0E"
	result := FormatSize(largeSize, 1024)
	if result != expected {
		t.Errorf("Expected %s, but got %s", expected, result)
	}
}
func TestParseBlockSize_MultiplierOverflow(t *testing.T) {
	// This input should cause an overflow when calculating the total size.
	overflowNum := "9223372036854775807" // max int64
	_, warning, ok := ParseBlockSize(fmt.Sprintf("%sPB", overflowNum))
	if ok {
		t.Error("Expected parsing to fail due to multiplier overflow")
	}
	if warning != "SIZE too large" {
		t.Errorf("Expected warning 'SIZE too large', got '%s'", warning)
	}
}

func TestParseBlockSize_TotalOverflow(t *testing.T) {
	// This input should cause an overflow when calculating the total size.
	overflowNum := "9223372036854775808"
	_, warning, ok := ParseBlockSize(overflowNum)
	if ok {
		t.Error("Expected parsing to fail due to total overflow")
	}
	if warning != "SIZE too large" {
		t.Errorf("Expected warning 'SIZE too large', got '%s'", warning)
	}
}

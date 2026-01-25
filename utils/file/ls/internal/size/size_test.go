package size

import (
	"strings"
	"testing"

	"github.com/ioluas/gopherutils/utils/file/ls/internal/config"
)

func TestParseBlockSize(t *testing.T) {
	tests := []struct {
		name           string
		raw            string
		expectOK       bool
		expectMode     config.BlockSizeMode
		expectSize     int64
		expectSuffix   string
		expectShowSuf  bool
		expectGrouping bool
		expectWarn     string
	}{
		{
			name:       "empty",
			raw:        "",
			expectOK:   false,
			expectWarn: "missing SIZE",
		},
		{
			name:       "human readable",
			raw:        "human-readable",
			expectOK:   true,
			expectMode: config.BlockSizeModeHuman,
		},
		{
			name:       "si",
			raw:        "si",
			expectOK:   true,
			expectMode: config.BlockSizeModeSI,
		},
		{
			name:           "grouping with suffix",
			raw:            "'1kB",
			expectOK:       true,
			expectMode:     config.BlockSizeModeBytes,
			expectSize:     1000,
			expectSuffix:   "kB",
			expectShowSuf:  false,
			expectGrouping: true,
		},
		{
			name:          "numeric suffix",
			raw:           "1K",
			expectOK:      true,
			expectMode:    config.BlockSizeModeBytes,
			expectSize:    1024,
			expectSuffix:  "K",
			expectShowSuf: false,
		},
		{
			name:           "suffix only implies one",
			raw:            "kB",
			expectOK:       true,
			expectMode:     config.BlockSizeModeBytes,
			expectSize:     1000,
			expectSuffix:   "kB",
			expectShowSuf:  true,
			expectGrouping: false,
		},
		{
			name:       "unknown suffix",
			raw:        "1X",
			expectOK:   false,
			expectWarn: "unknown SIZE suffix",
		},
		{
			name:       "zero invalid",
			raw:        "0",
			expectOK:   false,
			expectWarn: "invalid SIZE number",
		},
		{
			name:       "overflow uint64",
			raw:        "18446744073709551616",
			expectOK:   false,
			expectWarn: "invalid SIZE number",
		},
		{
			name:       "overflow int64",
			raw:        "9223372036854775808",
			expectOK:   false,
			expectWarn: "SIZE too large",
		},
		{
			name:       "missing after grouping prefix",
			raw:        "'",
			expectOK:   false,
			expectWarn: "missing SIZE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, warn, ok := ParseBlockSize(tt.raw)
			if ok != tt.expectOK {
				t.Fatalf("ok=%v, want %v (warn=%q)", ok, tt.expectOK, warn)
			}
			if tt.expectWarn != "" && !strings.Contains(warn, tt.expectWarn) {
				t.Fatalf("warn=%q, want to contain %q", warn, tt.expectWarn)
			}
			if !tt.expectOK {
				return
			}
			if got.Mode != tt.expectMode {
				t.Errorf("mode=%v, want %v", got.Mode, tt.expectMode)
			}
			if got.SizeBytes != tt.expectSize {
				t.Errorf("sizeBytes=%d, want %d", got.SizeBytes, tt.expectSize)
			}
			if got.Suffix != tt.expectSuffix {
				t.Errorf("suffix=%q, want %q", got.Suffix, tt.expectSuffix)
			}
			if got.ShowSuffix != tt.expectShowSuf {
				t.Errorf("showSuffix=%v, want %v", got.ShowSuffix, tt.expectShowSuf)
			}
			if got.GroupThousands != tt.expectGrouping {
				t.Errorf("groupThousands=%v, want %v", got.GroupThousands, tt.expectGrouping)
			}
		})
	}
}

func TestParseUintStrict(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectOK  bool
		expectVal uint64
	}{
		{name: "valid", input: "123", expectOK: true, expectVal: 123},
		{name: "empty", input: "", expectOK: false},
		{name: "invalid", input: "12a", expectOK: false},
		{name: "overflow", input: "18446744073709551616", expectOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseUintStrict(tt.input)
			if tt.expectOK {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got != tt.expectVal {
					t.Fatalf("got %d, want %d", got, tt.expectVal)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error, got %d", got)
			}
		})
	}
}

func TestBlockSizeMultiplier(t *testing.T) {
	tests := []struct {
		suffix    string
		expectOK  bool
		expectMul uint64
	}{
		{suffix: "", expectOK: true, expectMul: 1},
		{suffix: "KiB", expectOK: true, expectMul: 1 << 10},
		{suffix: "MB", expectOK: true, expectMul: 1_000_000},
		{suffix: "nope", expectOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.suffix, func(t *testing.T) {
			got, ok := blockSizeMultiplier(tt.suffix)
			if ok != tt.expectOK {
				t.Fatalf("ok=%v, want %v", ok, tt.expectOK)
			}
			if ok && got != tt.expectMul {
				t.Fatalf("multiplier=%d, want %d", got, tt.expectMul)
			}
		})
	}
}

func TestFormatSizeWithBlockSpec(t *testing.T) {
	t.Run("human", func(t *testing.T) {
		spec := config.BlockSizeSpec{Mode: config.BlockSizeModeHuman}
		if got := FormatSizeWithBlockSpec(2048, spec); got != "2.0K" {
			t.Fatalf("got %q, want %q", got, "2.0K")
		}
	})

	t.Run("si", func(t *testing.T) {
		spec := config.BlockSizeSpec{Mode: config.BlockSizeModeSI}
		if got := FormatSizeWithBlockSpec(2000, spec); got != "2.0K" {
			t.Fatalf("got %q, want %q", got, "2.0K")
		}
	})

	t.Run("bytes rounding", func(t *testing.T) {
		spec := config.BlockSizeSpec{Mode: config.BlockSizeModeBytes, SizeBytes: 1024}
		if got := FormatSizeWithBlockSpec(1500, spec); got != "2" {
			t.Fatalf("got %q, want %q", got, "2")
		}
	})

	t.Run("sizeBytes zero", func(t *testing.T) {
		spec := config.BlockSizeSpec{Mode: config.BlockSizeModeBytes, SizeBytes: 0}
		if got := FormatSizeWithBlockSpec(1500, spec); got != "1500" {
			t.Fatalf("got %q, want %q", got, "1500")
		}
	})

	t.Run("grouping", func(t *testing.T) {
		t.Setenv("LC_ALL", "")
		t.Setenv("LANG", "")
		t.Setenv("LC_NUMERIC", "en_US.UTF-8")
		spec := config.BlockSizeSpec{Mode: config.BlockSizeModeBytes, SizeBytes: 1000, GroupThousands: true}
		if got := FormatSizeWithBlockSpec(1234000, spec); got != "1,234" {
			t.Fatalf("got %q, want %q", got, "1,234")
		}
	})

	t.Run("show suffix", func(t *testing.T) {
		spec := config.BlockSizeSpec{
			Mode:       config.BlockSizeModeBytes,
			SizeBytes:  1000,
			Suffix:     "kB",
			ShowSuffix: true,
		}
		if got := FormatSizeWithBlockSpec(3000, spec); got != "3kB" {
			t.Fatalf("got %q, want %q", got, "3kB")
		}
	})
}

func TestShouldGroupThousands(t *testing.T) {
	t.Setenv("LC_ALL", "")
	t.Setenv("LANG", "")
	t.Setenv("LC_NUMERIC", "")
	if shouldGroupThousands() {
		t.Fatal("expected false when LC_NUMERIC is empty")
	}

	t.Setenv("LC_NUMERIC", "C")
	if shouldGroupThousands() {
		t.Fatal("expected false when LC_NUMERIC is C")
	}

	t.Setenv("LC_NUMERIC", "POSIX")
	if shouldGroupThousands() {
		t.Fatal("expected false when LC_NUMERIC is POSIX")
	}

	t.Setenv("LC_NUMERIC", "en_US.UTF-8")
	if !shouldGroupThousands() {
		t.Fatal("expected true when LC_NUMERIC is non-C")
	}
}

func TestGroupThousands(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{input: "1", expected: "1"},
		{input: "123456", expected: "123,456"},
		{input: "1234", expected: "1,234"},
		{input: "1234567", expected: "1,234,567"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := groupThousands(tt.input); got != tt.expected {
				t.Fatalf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

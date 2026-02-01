package size

import "testing"

func TestFormatSize(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{100, "100"},
		{1023, "1023"},
		{1024, "1.0K"},
		{1500, "1.5K"},
		{1536, "1.5K"},
		{2048, "2.0K"},
		{5120, "5.0K"},
		{1024 * 1024, "1.0M"},
		{1024*1024*2 + 500*1024, "2.5M"},
		{1024 * 1024 * 1024, "1.0G"},
		{1024 * 1024 * 1024 * 1024, "1.0T"},
		{1024 * 1024 * 1024 * 1024 * 1024, "1.0P"},
		{1024 * 1024 * 1024 * 1024 * 1024 * 1024, "1.0E"},
		{9223372036854775807, "8.0E"}, // max int64
	}

	for _, tt := range tests {
		got := FormatSize(tt.input, 1024)
		if got != tt.expected {
			t.Errorf("FormatSize(%d, 1024) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestFormatSizeSI(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0"},
		{100, "100"},
		{999, "999"},
		{1000, "1.0K"},
		{1500, "1.5K"},
		{1000 * 1000, "1.0M"},
		{1000*1000*2 + 500*1000, "2.5M"},
		{1000 * 1000 * 1000, "1.0G"},
	}

	for _, tt := range tests {
		got := FormatSize(tt.input, 1000)
		if got != tt.expected {
			t.Errorf("FormatSize(%d, 1000) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

package main

import (
	"testing"
)

func TestNormalizeNameUnicode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "ASCII only",
			input:    "Hello123",
			expected: "hello123",
		},
		{
			name:     "ASCII with punctuation",
			input:    "Hello-World_123.txt",
			expected: "helloworld123txt",
		},
		{
			name:     "French accents",
			input:    "Café",
			expected: "café",
		},
		{
			name:     "German umlaut and eszett",
			input:    "Müller-Straße",
			expected: "müllerstraße",
		},
		{
			name:     "Scandinavian characters",
			input:    "Åse_Øst",
			expected: "åseøst",
		},
		{
			name:     "Spanish",
			input:    "Año-Niño",
			expected: "añoniño",
		},
		{
			name:     "Greek",
			input:    "Αλφα-Βητα",
			expected: "αλφαβητα",
		},
		{
			name:     "Cyrillic",
			input:    "Москва_2024",
			expected: "москва2024",
		},
		{
			name:     "CJK characters",
			input:    "文件-名字.txt",
			expected: "文件名字txt",
		},
		{
			name:     "Mixed Unicode and ASCII",
			input:    "Test-Café_123.txt",
			expected: "testcafé123txt",
		},
		{
			name:     "Only punctuation",
			input:    "---...",
			expected: "",
		},
		{
			name:     "Combining marks (accents)",
			input:    "e\u0301", // é as e + combining acute accent
			expected: "e\u0301", // Combining marks are preserved
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeName(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizeNameSorting(t *testing.T) {
	// Test that normalization produces correct sort order
	tests := []struct {
		name     string
		file1    string
		file2    string
		expected bool // true if file1 should come before file2
	}{
		{
			name:     "Punctuation ignored",
			file1:    "a-file",
			file2:    "afile",
			expected: false, // normalized to same, fallback to original comparison
		},
		{
			name:     "Case insensitive",
			file1:    "Apple",
			file2:    "banana",
			expected: true,
		},
		{
			name:     "Unicode characters preserved",
			file1:    "Café",
			file2:    "Zebra",
			expected: true, // café < zebra
		},
		{
			name:     "CJK sorting",
			file1:    "文件A",
			file2:    "文件B",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			norm1 := normalizeName(tt.file1)
			norm2 := normalizeName(tt.file2)
			result := norm1 < norm2
			if result != tt.expected {
				t.Errorf("normalizeName(%q) < normalizeName(%q) = %v, want %v (norm1=%q, norm2=%q)",
					tt.file1, tt.file2, result, tt.expected, norm1, norm2)
			}
		})
	}
}

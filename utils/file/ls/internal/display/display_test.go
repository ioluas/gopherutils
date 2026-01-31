package display

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ioluas/gopherutils/utils/file/ls/internal/config"
)

func TestQuoteName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no special chars",
			input:    "file.txt",
			expected: "file.txt",
		},
		{
			name:     "newline",
			input:    "file\nname",
			expected: `file\nname`,
		},
		{
			name:     "tab",
			input:    "file\tname",
			expected: `file\tname`,
		},
		{
			name:     "backspace",
			input:    "file\bname",
			expected: `file\bname`,
		},
		{
			name:     "return",
			input:    "file\rname",
			expected: `file\rname`,
		},
		{
			name:     "alert",
			input:    "file\aname",
			expected: `file\aname`,
		},
		{
			name:     "form feed",
			input:    "file\fname",
			expected: `file\fname`,
		},
		{
			name:     "vertical tab",
			input:    "file\vname",
			expected: `file\vname`,
		},
		{
			name:     "backslash",
			input:    `file\name`,
			expected: `file\\name`,
		},
		{
			name:     "octal char (low ASCII)",
			input:    "file\x01name",
			expected: `file\001name`,
		},
		{
			name:     "octal char (high ASCII)",
			input:    "file\x80name",
			expected: `file\200name`,
		},
		{
			name:     "mix of special and normal",
			input:    "dir/\tfile\n.txt",
			expected: `dir/\tfile\n.txt`,
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := QuoteName(tt.input)
			if actual != tt.expected {
				t.Errorf("QuoteName(%q) = %q; want %q", tt.input, actual, tt.expected)
			}
		})
	}
}

func TestLongListFormatArgsCombinations(t *testing.T) {
	d := fileDetails{
		mode:    "-rw-r--r--",
		nlink:   1,
		owner:   "user",
		group:   "group",
		author:  "author",
		sizeStr: "123",
		name:    "file",
		modTime: time.Now(),
	}
	widths := longListWidths{
		link:   1,
		owner:  4,
		group:  5,
		author: 6,
		size:   3,
	}

	// Case 1: ShowAuthor=true, NoGroup=true
	cfg := &config.Config{ShowAuthor: true, NoGroup: true}
	format, args := longListFormatArgs(d, widths, cfg)
	if !strings.Contains(format, "%-*s %-*s") || len(args) != 11 {
		t.Errorf("Unexpected format or args for Author+NoGroup: %q, %d", format, len(args))
	}

	// Case 2: ShowAuthor=true, NoGroup=false
	cfg = &config.Config{ShowAuthor: true, NoGroup: false}
	format, args = longListFormatArgs(d, widths, cfg)
	if !strings.Contains(format, "%-*s %-*s %-*s") || len(args) != 13 {
		t.Errorf("Unexpected format or args for Author: %q, %d", format, len(args))
	}

	// Case 3: ShowAuthor=false, NoGroup=true
	cfg = &config.Config{ShowAuthor: false, NoGroup: true}
	format, args = longListFormatArgs(d, widths, cfg)
	if strings.Contains(format, "%-*s %-*s %-*s") || len(args) != 9 {
		t.Errorf("Unexpected format or args for NoGroup: %q, %d", format, len(args))
	}

	// Case 4: Default (both false)
	cfg = &config.Config{ShowAuthor: false, NoGroup: false}
	_, args = longListFormatArgs(d, widths, cfg)
	if len(args) != 11 {
		t.Errorf("Unexpected args for default: %d", len(args))
	}
}

// Mocking the terminal functions for testing
var (
	mockIsTerminal bool
	mockTermWidth  int
	mockTermHeight int
	mockTermErr    error
)

func mockIsTerminalFunc(fd int) bool {
	return mockIsTerminal
}

func mockGetTermSizeFunc(fd int) (width, height int, err error) {
	return mockTermWidth, mockTermHeight, mockTermErr
}

// mockFile implements io.Writer and Fd() uintptr for terminal detection in tests
type mockFile struct {
	bytes.Buffer
	fd uintptr
}

func (m *mockFile) Fd() uintptr {
	return m.fd
}

// Helper function to generate expected grid output, simulating PrintGrid's logic
func generateExpectedGridOutput(names []string, termWidth int) string {
	if len(names) == 0 {
		return ""
	}

	n := len(names)
	padding := 2

	// Try with increasing number of rows
	for rows := 1; rows <= n; rows++ {
		cols := (n + rows - 1) / rows

		// Calculate width of each column
		colWidths := make([]int, cols)
		for c := 0; c < cols; c++ {
			for r := 0; r < rows; r++ {
				idx := c*rows + r
				if idx < n {
					l := len(names[idx])
					if l > colWidths[c] {
						colWidths[c] = l
					}
				}
			}
		}

		// Calculate total width required
		totalWidth := 0
		for _, w := range colWidths {
			totalWidth += w
		}
		totalWidth += (cols - 1) * padding

		// If it fits, or if we are forced to 1 column (rows==n), print it
		if totalWidth <= termWidth || rows == n {
			var buf bytes.Buffer
			for r := 0; r < rows; r++ {
				for c := 0; c < cols; c++ {
					idx := c*rows + r
					if idx < n {
						if c < cols-1 {
							format := fmt.Sprintf("%%-%ds", colWidths[c]+padding)
							fmt.Fprintf(&buf, format, names[idx])
						} else {
							fmt.Fprint(&buf, names[idx])
						}
					}
				}
				fmt.Fprintln(&buf)
			}
			return buf.String()
		}
	}
	return ""
}

func TestPrintEntries(t *testing.T) {
	// Save original functions and restore them after test
	originalIsTerminalFunc := IsTerminalFunc
	originalGetTermSizeFunc := GetTermSizeFunc
	defer func() {
		IsTerminalFunc = originalIsTerminalFunc
		GetTermSizeFunc = originalGetTermSizeFunc
	}()

	tests := []struct {
		name           string
		names          []string
		cfg            *config.Config
		isTerminal     bool
		termWidth      int
		expected       string
		getTermSizeErr bool
	}{
		{
			name:       "empty list",
			names:      []string{},
			cfg:        &config.Config{},
			isTerminal: true,
			termWidth:  80,
			expected:   "",
		},
		{
			name:       "not a terminal, no columnate",
			names:      []string{"file1", "file2", "file3"},
			cfg:        &config.Config{Columnate: false},
			isTerminal: false,
			termWidth:  80,
			expected:   "file1\nfile2\nfile3\n",
		},
		{
			name:       "terminal, no explicit columnate (defaults to grid)",
			names:      []string{"file1", "file2", "file3"},
			cfg:        &config.Config{Columnate: false},
			isTerminal: true,
			termWidth:  80,
			expected:   generateExpectedGridOutput([]string{"file1", "file2", "file3"}, 80),
		},
		{
			name:       "terminal, force one per line",
			names:      []string{"file1", "file2", "file3"},
			cfg:        &config.Config{OnePerLine: true},
			isTerminal: true,
			termWidth:  80,
			expected:   "file1\nfile2\nfile3\n",
		},
		{
			name:       "terminal, columnate, single column (narrow width)",
			names:      []string{"longfilename", "short"},
			cfg:        &config.Config{Columnate: true},
			isTerminal: true,
			termWidth:  10,
			expected:   generateExpectedGridOutput([]string{"longfilename", "short"}, 10),
		},
		{
			name:       "terminal, columnate, multiple columns",
			names:      []string{"a", "b", "c", "d", "e"},
			cfg:        &config.Config{Columnate: true},
			isTerminal: true,
			termWidth:  20,
			expected:   generateExpectedGridOutput([]string{"a", "b", "c", "d", "e"}, 20),
		},
		{
			name:       "terminal, columnate, multiple columns, longer names",
			names:      []string{"apple", "banana", "cherry", "date", "elderberry"},
			cfg:        &config.Config{Columnate: true},
			isTerminal: true,
			termWidth:  30,
			expected:   generateExpectedGridOutput([]string{"apple", "banana", "cherry", "date", "elderberry"}, 30),
		},
		{
			name:           "GetTermSize error, columnate true, default to 80 chars",
			names:          []string{"file1", "file2"},
			cfg:            &config.Config{Columnate: true},
			isTerminal:     true,
			termWidth:      80, // Terminal width doesn't matter if error occurs
			getTermSizeErr: true,
			expected:       generateExpectedGridOutput([]string{"file1", "file2"}, 80),
		},
		{
			name:           "GetTermSize error, columnate false, consistent non-terminal style",
			names:          []string{"file1", "file2"},
			cfg:            &config.Config{Columnate: false},
			isTerminal:     true,
			termWidth:      80, // Terminal width doesn't matter if error occurs
			getTermSizeErr: true,
			expected:       "file1\nfile2\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockIsTerminal = tt.isTerminal
			mockTermWidth = tt.termWidth
			mockTermHeight = 24 // irrelevant for this test
			if tt.getTermSizeErr {
				mockTermErr = fmt.Errorf("mock error")
			} else {
				mockTermErr = nil
			}

			// Assign mock functions
			IsTerminalFunc = mockIsTerminalFunc
			GetTermSizeFunc = mockGetTermSizeFunc

			var buf mockFile
			PrintEntries(&buf, tt.names, tt.cfg)

			if actual := buf.String(); actual != tt.expected {
				t.Errorf("PrintEntries() got:\n%q\nwant:\n%q", actual, tt.expected)
			}
		})
	}
}

func TestPrintGrid(t *testing.T) {
	tests := []struct {
		name     string
		names    []string
		width    int
		expected string
	}{
		{
			name:     "empty list",
			names:    []string{},
			width:    80,
			expected: "",
		},
		{
			name:     "single item, fits",
			names:    []string{"file.txt"},
			width:    10,
			expected: generateExpectedGridOutput([]string{"file.txt"}, 10),
		},
		{
			name:     "multiple items, single column (width too small)",
			names:    []string{"longname1", "longname2"},
			width:    5,
			expected: generateExpectedGridOutput([]string{"longname1", "longname2"}, 5),
		},
		{
			name:     "multiple items, multiple columns",
			names:    []string{"a", "bb", "ccc", "dddd", "e", "ffffff"},
			width:    20,
			expected: generateExpectedGridOutput([]string{"a", "bb", "ccc", "dddd", "e", "ffffff"}, 20),
		},
		{
			name:     "multiple items, exact fit",
			names:    []string{"one", "two", "three"},
			width:    18,
			expected: generateExpectedGridOutput([]string{"one", "two", "three"}, 18),
		},
		{
			name:     "single column due to wide names",
			names:    []string{"verylongfilenameindeed", "anotherverylongfilename"},
			width:    80,
			expected: generateExpectedGridOutput([]string{"verylongfilenameindeed", "anotherverylongfilename"}, 80),
		},
		{
			name:     "single column when only one name",
			names:    []string{"onlyone"},
			width:    10,
			expected: generateExpectedGridOutput([]string{"onlyone"}, 10),
		},
		{
			name:     "names that perfectly fill columns",
			names:    []string{"a", "b", "c", "d"},
			width:    12,
			expected: generateExpectedGridOutput([]string{"a", "b", "c", "d"}, 12),
		},
		{
			name:     "column width equals terminal width, maxLen is large",
			names:    []string{"thisisverylongname", "short"},
			width:    10,
			expected: generateExpectedGridOutput([]string{"thisisverylongname", "short"}, 10),
		},
		{
			name:     "single column forced by narrow width",
			names:    []string{"file"},
			width:    1,
			expected: generateExpectedGridOutput([]string{"file"}, 1),
		},
		{
			name:     "numCols limited by num names",
			names:    []string{"a", "b"},
			width:    80,
			expected: generateExpectedGridOutput([]string{"a", "b"}, 80),
		},
		{
			name:     "zero width",
			names:    []string{"file"},
			width:    0, // Will result in colWidth = 1, numCols = 0, then numCols = 1
			expected: generateExpectedGridOutput([]string{"file"}, 0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			PrintGrid(&buf, tt.names, tt.width)

			if actual := buf.String(); actual != tt.expected {
				t.Errorf("PrintGrid(%v, %d) got:\n%q\nwant:\n%q", tt.names, tt.width, actual, tt.expected)
			}
		})
	}
}

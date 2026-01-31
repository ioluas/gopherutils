//go:build !windows

package display

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/ioluas/gopherutils/internal/fsutil"
	"github.com/ioluas/gopherutils/utils/file/ls/internal/config"
	"github.com/ioluas/gopherutils/utils/file/ls/internal/entry"
	"github.com/ioluas/gopherutils/utils/file/ls/internal/timeutil"
)

// Mock implementation of os.DirEntry
type mockDirEntry struct {
	name    string
	fileMod fs.FileMode
	fileSiz int64
	modTime time.Time
	sysStat *syscall.Stat_t // For owner/group testing
	infoErr error           // For error testing
}

func (m mockDirEntry) Name() string      { return m.name }
func (m mockDirEntry) IsDir() bool       { return m.fileMod.IsDir() }
func (m mockDirEntry) Type() fs.FileMode { return m.fileMod.Type() }
func (m mockDirEntry) Info() (fs.FileInfo, error) {
	if m.infoErr != nil {
		return nil, m.infoErr
	}
	return mockFileInfo{
		name:    m.name,
		mode:    m.fileMod,
		size:    m.fileSiz,
		modTime: m.modTime,
		sys:     m.sysStat,
	}, nil
}

// Mock implementation of fs.FileInfo
type mockFileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	sys     interface{}
}

func (m mockFileInfo) Name() string       { return m.name }
func (m mockFileInfo) Size() int64        { return m.size }
func (m mockFileInfo) Mode() fs.FileMode  { return m.mode }
func (m mockFileInfo) ModTime() time.Time { return m.modTime }
func (m mockFileInfo) IsDir() bool        { return m.mode.IsDir() }
func (m mockFileInfo) Sys() interface{}   { return m.sys }

func TestPrintLongList(t *testing.T) {
	originalGetOwnerGroupImpl := fsutil.GetOwnerGroupImpl
	fsutil.GetOwnerGroupImpl = func(stat *syscall.Stat_t) (string, string) {
		if stat == nil {
			return "-", "-"
		}
		if stat.Uid == 1000 {
			return "user", "group"
		}
		return "other", "other"
	}
	defer func() { fsutil.GetOwnerGroupImpl = originalGetOwnerGroupImpl }()

	fixedTime := time.Date(2023, 4, 1, 10, 30, 0, 0, time.UTC)
	originalTimeNow := timeutil.NowFunc
	timeutil.NowFunc = func() time.Time { return fixedTime }
	defer func() { timeutil.NowFunc = originalTimeNow }()
	yesterday := fixedTime.Add(-24 * time.Hour)
	lastYear := fixedTime.Add(-365 * 24 * time.Hour)

	testCases := []struct {
		name           string
		entries        []os.DirEntry
		config         *config.Config
		expectedStdout string
		expectedStderr string
		expectedError  bool
	}{
		{
			name:           "empty entries",
			entries:        []os.DirEntry{},
			config:         &config.Config{},
			expectedStdout: "",
			expectedError:  false,
		},
		{
			name: "basic file long list",
			entries: []os.DirEntry{
				mockDirEntry{
					name:    "file1.txt",
					fileMod: 0644,
					fileSiz: 12345,
					modTime: fixedTime,
					sysStat: &syscall.Stat_t{Nlink: 1, Uid: 1000, Gid: 1000},
				},
			},
			config: &config.Config{
				TimeStyleSpec: &config.TimeStyleSpec{Kind: config.TimeStyleFullISO, RecentLayout: "2006-01-02 15:04:05.000000000 -0700"},
			},
			expectedStdout: "-rw-r--r--  1 user group 12345 2023-04-01 10:30:00.000000000 +0000 file1.txt\n",
		},
		{
			name: "multiple files human readable",
			entries: []os.DirEntry{
				mockDirEntry{
					name:    "small.txt",
					fileMod: 0644,
					fileSiz: 100,
					modTime: yesterday,
					sysStat: &syscall.Stat_t{Nlink: 1, Uid: 1000, Gid: 1000},
				},
				mockDirEntry{
					name:    "medium.txt",
					fileMod: 0644,
					fileSiz: 102400, // 100K
					modTime: fixedTime,
					sysStat: &syscall.Stat_t{Nlink: 2, Uid: 1001, Gid: 1001},
				},
			},
			config: &config.Config{
				HumanReadable: true,
				TimeStyleSpec: &config.TimeStyleSpec{Kind: config.TimeStyleISO, RecentLayout: "01-02 15:04", OldLayout: "2006-01-02"},
			},
			expectedStdout: "-rw-r--r--  1 user  group    100 03-31 10:30 small.txt\n-rw-r--r--  2 other other 100.0K 04-01 10:30 medium.txt\n",
		},
		{
			name: "directory and file, show author, no group",
			entries: []os.DirEntry{
				mockDirEntry{
					name:    "mydir",
					fileMod: os.ModeDir | 0755,
					fileSiz: 4096,
					modTime: fixedTime,
					sysStat: &syscall.Stat_t{Nlink: 2, Uid: 1000, Gid: 1000},
				},
				mockDirEntry{
					name:    "myfile.txt",
					fileMod: 0600,
					fileSiz: 500,
					modTime: lastYear,
					sysStat: &syscall.Stat_t{Nlink: 1, Uid: 1001, Gid: 1001},
				},
			},
			config: &config.Config{
				ShowAuthor:    true,
				NoGroup:       true,
				TimeStyleSpec: &config.TimeStyleSpec{Kind: config.TimeStyleLongISO, RecentLayout: "2006-01-02 15:04"},
			},
			expectedStdout: "drwxr-xr-x  2 user  user  4096 2023-04-01 10:30 mydir\n-rw-------  1 other other  500 2022-04-01 10:30 myfile.txt\n",
		},
		{
			name: "file with special chars, escaped, SI units",
			entries: []os.DirEntry{
				mockDirEntry{
					name:    "file with\nspaces.txt",
					fileMod: 0644,
					fileSiz: 2000, // 2K in SI
					modTime: fixedTime,
					sysStat: &syscall.Stat_t{Nlink: 1, Uid: 1000, Gid: 1000},
				},
			},
			config: &config.Config{
				Escape:        true,
				SI:            true,
				TimeStyleSpec: &config.TimeStyleSpec{Kind: config.TimeStyleLongISO, RecentLayout: "2006-01-02 15:04"},
			},
			expectedStdout: "-rw-r--r--  1 user group 2.0K 2023-04-01 10:30 file with\\nspaces.txt\n",
		},
		{
			name: "non-unix fallback for owner/group",
			entries: []os.DirEntry{
				mockDirEntry{
					name:    "windows.txt",
					fileMod: 0644,
					fileSiz: 1000,
					modTime: fixedTime,
					sysStat: nil, // Simulate non-Unix system
				},
			},
			config: &config.Config{
				TimeStyleSpec: &config.TimeStyleSpec{Kind: config.TimeStyleFullISO, RecentLayout: "2006-01-02 15:04:05.000000000 -0700"},
			},
			expectedStdout: "-rw-r--r--  1 - - 1000 2023-04-01 10:30:00.000000000 +0000 windows.txt\n",
		},
		{
			name: "error getting file info",
			entries: []os.DirEntry{
				mockDirEntry{
					name:    "bad-file",
					fileMod: 0644,
					infoErr: errors.New("simulated info error"),
				},
			},
			config:         &config.Config{},
			expectedStdout: "",
			expectedStderr: "ls: cannot access 'bad-file': simulated info error\n", // PrintLongList prints error to stderr
			expectedError:  true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			stdout := new(bytes.Buffer)
			stderr := new(bytes.Buffer)

			// The `hadError` check remains from the previous refactoring
			hadError := PrintLongList(stdout, stderr, tt.entries, tt.config)
			if hadError != tt.expectedError {
				t.Errorf("Test %s: Expected error=%v, got %v", tt.name, tt.expectedError, hadError)
			}

			if actualStdout := stdout.String(); actualStdout != tt.expectedStdout {
				t.Errorf("Test %s: \nExpected stdout:\n%q\nActual stdout:\n%q", tt.name, tt.expectedStdout, actualStdout)
			}

			if actualStderr := stderr.String(); actualStderr != tt.expectedStderr {
				t.Errorf("Test %s: \nExpected stderr:\n%q\nActual stderr:\n%q", tt.name, tt.expectedStderr, actualStderr)
			}
		})
	}
}

func TestPrintLongListWithCachedDirEntryError(t *testing.T) {
	testErr := errors.New("cached error")
	entries := []os.DirEntry{
		&entry.CachedDirEntry{
			DirEntry: mockDirEntry{
				name:    "cached-error-file",
				fileMod: 0644,
			},
			Err: testErr,
		},
	}

	lsConfig := &config.Config{}
	var stdout, stderr bytes.Buffer
	hadError := PrintLongList(&stdout, &stderr, entries, lsConfig)

	if !hadError {
		t.Error("expected error to be reported")
	}
	if !strings.Contains(stderr.String(), "cached-error-file") {
		t.Errorf("expected error message to contain filename, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "cached error") {
		t.Errorf("expected error message to contain error text, got %q", stderr.String())
	}
	if stdout.String() != "" {
		t.Errorf("expected no stdout output, got %q", stdout.String())
	}
}

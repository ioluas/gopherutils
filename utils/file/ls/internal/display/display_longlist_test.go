package display

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/ioluas/gopherutils/internal/fsutil"
	"github.com/ioluas/gopherutils/utils/file/ls/internal/config"
	"github.com/ioluas/gopherutils/utils/file/ls/internal/entry"
	"github.com/ioluas/gopherutils/utils/file/ls/internal/size"
	"github.com/ioluas/gopherutils/utils/file/ls/internal/timeutil"
)

// Mock implementation of os.DirEntry
type mockDirEntry struct {
	name    string
	fileMod fs.FileMode
	fileSiz int64
	modTime time.Time
	sysStat *syscall.Stat_t // For owner/group testing
}

func (m mockDirEntry) Name() string      { return m.name }
func (m mockDirEntry) IsDir() bool       { return m.fileMod.IsDir() }
func (m mockDirEntry) Type() fs.FileMode { return m.fileMod.Type() }
func (m mockDirEntry) Info() (fs.FileInfo, error) {
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

	now := time.Now()

	yesterday := now.Add(-24 * time.Hour)

	lastYear := now.Add(-365 * 24 * time.Hour)

	// Mock fsutil.GetOwnerGroup to return predictable values for testing

	originalGetOwnerGroupImpl := fsutil.GetOwnerGroupImpl

	fsutil.GetOwnerGroupImpl = func(stat *syscall.Stat_t) (string, string) {

		if stat == nil {

			return "-", "-"

		}

		switch stat.Uid {

		case 1000:

			return "user", "group"

		case 1001:

			return "user", "group"

		default:

			return fmt.Sprint(stat.Uid), fmt.Sprint(stat.Gid)

		}

	}

	defer func() {

		fsutil.GetOwnerGroupImpl = originalGetOwnerGroupImpl

	}()

	rawTestCases := []struct {
		name string

		entries []os.DirEntry

		config *config.Config
	}{

		{

			name: "empty entries",

			entries: []os.DirEntry{},

			config: &config.Config{},
		},

		{

			name: "basic file long list",

			entries: []os.DirEntry{

				mockDirEntry{

					name: "file1.txt",

					fileMod: 0644,

					fileSiz: 12345,

					modTime: now,

					sysStat: &syscall.Stat_t{Nlink: 1, Uid: 1000, Gid: 1000},
				},
			},

			config: &config.Config{

				TimeStyleSpec: &config.TimeStyleSpec{Kind: config.TimeStyleFullISO},
			},
		},

		{

			name: "multiple files human readable",

			entries: []os.DirEntry{

				mockDirEntry{

					name: "small.txt",

					fileMod: 0644,

					fileSiz: 100,

					modTime: yesterday,

					sysStat: &syscall.Stat_t{Nlink: 1, Uid: 1000, Gid: 1000},
				},

				mockDirEntry{

					name: "medium.txt",

					fileMod: 0644,

					fileSiz: 102400, // 100K

					modTime: now,

					sysStat: &syscall.Stat_t{Nlink: 2, Uid: 1001, Gid: 1001},
				},
			},

			config: &config.Config{

				HumanReadable: true,

				TimeStyleSpec: &config.TimeStyleSpec{Kind: config.TimeStyleISO},
			},
		},

		{

			name: "directory and file, show author, no group",

			entries: []os.DirEntry{

				mockDirEntry{

					name: "mydir",

					fileMod: os.ModeDir | 0755,

					fileSiz: 4096,

					modTime: now,

					sysStat: &syscall.Stat_t{Nlink: 2, Uid: 1000, Gid: 1000},
				},

				mockDirEntry{

					name: "myfile.txt",

					fileMod: 0600,

					fileSiz: 500,

					modTime: lastYear,

					sysStat: &syscall.Stat_t{Nlink: 1, Uid: 1001, Gid: 1001},
				},
			},

			config: &config.Config{

				ShowAuthor: true,

				NoGroup: true,

				TimeStyleSpec: &config.TimeStyleSpec{Kind: config.TimeStyleLongISO},
			},
		},

		{

			name: "file with special chars, escaped, SI units",

			entries: []os.DirEntry{

				mockDirEntry{

					name: "file with\nspaces.txt",

					fileMod: 0644,

					fileSiz: 2000, // 2K in SI

					modTime: now,

					sysStat: &syscall.Stat_t{Nlink: 1, Uid: 1000, Gid: 1000},
				},
			},

			config: &config.Config{

				Escape: true,

				SI: true,

				TimeStyleSpec: &config.TimeStyleSpec{Kind: config.TimeStyleLongISO},
			},
		},

		{

			name: "non-unix fallback for owner/group",

			entries: []os.DirEntry{

				mockDirEntry{

					name: "windows.txt",

					fileMod: 0644,

					fileSiz: 1000,

					modTime: now,

					sysStat: nil, // Simulate non-Unix system

				},
			},

			config: &config.Config{

				TimeStyleSpec: &config.TimeStyleSpec{Kind: config.TimeStyleFullISO},
			},
		},

		{

			name: "cached entry time",

			entries: []os.DirEntry{

				&entry.CachedDirEntry{

					DirEntry: mockDirEntry{

						name: "cached.txt",

						fileMod: 0644,

						fileSiz: 500,

						sysStat: &syscall.Stat_t{Nlink: 1, Uid: 1000, Gid: 1000},
					},

					Time: &yesterday, // Explicitly set a cached time

				},
			},

			config: &config.Config{

				TimeStyleSpec: &config.TimeStyleSpec{Kind: config.TimeStyleISO},
			},
		},

		{

			name: "non-unix fallback with author",

			entries: []os.DirEntry{

				mockDirEntry{

					name: "file.txt",

					fileMod: 0644,

					fileSiz: 100,

					modTime: now,

					sysStat: nil,
				},
			},

			config: &config.Config{

				ShowAuthor: true,

				TimeStyleSpec: &config.TimeStyleSpec{Kind: config.TimeStyleFullISO},
			},
		},

		{

			name: "file with block size",

			entries: []os.DirEntry{

				mockDirEntry{

					name: "blockfile.txt",

					fileMod: 0644,

					fileSiz: 5123,

					modTime: now,

					sysStat: &syscall.Stat_t{Nlink: 1, Uid: 1000, Gid: 1000},
				},
			},

			config: &config.Config{

				BlockSize: &config.BlockSizeSpec{Mode: config.BlockSizeModeBytes, ShowSuffix: true, SizeBytes: 1024},

				TimeStyleSpec: &config.TimeStyleSpec{Kind: config.TimeStyleFullISO},
			},
		},
	}

	type testCase struct {
		name string

		entries []os.DirEntry

		config *config.Config

		expectedStdout string

		expectedStderr string
	}

	var testCases []testCase

	for _, rt := range rawTestCases {

		if len(rt.entries) == 0 {

			testCases = append(testCases, testCase{

				name: rt.name,

				entries: rt.entries,

				config: rt.config,

				expectedStdout: "",

				expectedStderr: "",
			})

			continue

		}

		details := make([]fileDetails, 0, len(rt.entries))

		var maxLinkLen, maxOwnerLen, maxAuthorLen, maxGroupLen, maxSizeLen int

		for _, dirEntry := range rt.entries {

			info, err := dirEntry.Info()

			if err != nil {

				t.Fatalf("failed to get file info for %s: %v", dirEntry.Name(), err)

			}

			var nlink uint64 = 1

			var owner, group, author string

			fileSize := info.Size()

			sysData := info.Sys()

			sysStat, ok := sysData.(*syscall.Stat_t)

			if ok && sysStat != nil {

				nlink = uint64(sysStat.Nlink)

				owner, group = fsutil.GetOwnerGroup(sysStat)

				if rt.config.ShowAuthor {

					author = owner

				}

			} else {

				owner = "-"

				group = "-"

				if rt.config.ShowAuthor {

					author = "-"

				}

			}

			var sizeStr string

			if rt.config.BlockSize != nil {

				sizeStr = size.FormatSizeWithBlockSpec(fileSize, *rt.config.BlockSize)

			} else if rt.config.SI {

				sizeStr = size.FormatSize(fileSize, 1000)

			} else if rt.config.HumanReadable {

				sizeStr = size.FormatSize(fileSize, 1024)

			} else {

				sizeStr = fmt.Sprintf("%d", fileSize)

			}

			name := dirEntry.Name()

			if rt.config.Escape {

				name = QuoteName(name)

			}

			var entryTime time.Time

			if ce, ok := dirEntry.(*entry.CachedDirEntry); ok && ce.Time != nil {

				entryTime = *ce.Time

			} else {

				entryTime = timeutil.GetEntryTime(info, rt.config.TimeField)

			}

			d := fileDetails{

				mode: info.Mode().String(),

				nlink: nlink,

				owner: owner,

				author: author,

				group: group,

				sizeStr: sizeStr,

				modTime: entryTime,

				name: name,
			}

			details = append(details, d)

			// Calculate max widths

			if l := len(fmt.Sprint(d.nlink)); l > maxLinkLen {

				maxLinkLen = l

			}

			if l := len(d.owner); l > maxOwnerLen {

				maxOwnerLen = l

			}

			if rt.config.ShowAuthor {

				if l := len(d.author); l > maxAuthorLen {

					maxAuthorLen = l

				}

			}

			if !rt.config.NoGroup {

				if l := len(d.group); l > maxGroupLen {

					maxGroupLen = l

				}

			}

			if l := len(d.sizeStr); l > maxSizeLen {

				maxSizeLen = l

			}

		}

		if maxLinkLen < 2 {

			maxLinkLen = 2

		}

		widths := longListWidths{

			link: maxLinkLen,

			owner: maxOwnerLen,

			author: maxAuthorLen,

			group: maxGroupLen,

			size: maxSizeLen,
		}

		var expectedStdoutBuf bytes.Buffer

		for _, d := range details {

			format, args := longListFormatArgs(d, widths, rt.config)

			fmt.Fprintf(&expectedStdoutBuf, format, args...)

		}

		testCases = append(testCases, testCase{

			name: rt.name,

			entries: rt.entries,

			config: rt.config,

			expectedStdout: expectedStdoutBuf.String(),

			expectedStderr: "", // Assuming no stderr for successful paths

		})

	}

	for _, tt := range testCases {

		t.Run(tt.name, func(t *testing.T) {

			stdout := new(bytes.Buffer)

			stderr := new(bytes.Buffer)

			PrintLongList(stdout, stderr, tt.entries, tt.config)

			if actualStdout := stdout.String(); actualStdout != tt.expectedStdout {

				t.Errorf("Test %s: \nExpected stdout:\n%q\nActual stdout:\n%q", tt.name, tt.expectedStdout, actualStdout)

			}

			if actualStderr := stderr.String(); actualStderr != tt.expectedStderr {

				t.Errorf("Test %s: \nExpected stderr:\n%q\nActual stderr:\n%q", tt.name, tt.expectedStderr, actualStderr)

			}

		})

	}

}

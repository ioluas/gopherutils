package timeutil

import (
	"io/fs"
	"time"
)

// mockFileInfo is a minimal fs.FileInfo for time-related tests.
type mockFileInfo struct {
	modTime time.Time
	sys     interface{}
}

func (m *mockFileInfo) Name() string       { return "" }
func (m *mockFileInfo) Size() int64        { return 0 }
func (m *mockFileInfo) Mode() fs.FileMode  { return 0 }
func (m *mockFileInfo) ModTime() time.Time { return m.modTime }
func (m *mockFileInfo) IsDir() bool        { return false }
func (m *mockFileInfo) Sys() interface{}   { return m.sys }

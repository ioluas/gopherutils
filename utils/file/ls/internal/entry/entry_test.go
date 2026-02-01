package entry

import (
	"errors"
	"io/fs"
	"path/filepath"
	"testing"
	"time"
)

type mockDirEntry struct {
	name string
	info fs.FileInfo
	err  error
}

func (m *mockDirEntry) Name() string               { return m.name }
func (m *mockDirEntry) IsDir() bool                { return false }
func (m *mockDirEntry) Type() fs.FileMode          { return 0 }
func (m *mockDirEntry) Info() (fs.FileInfo, error) { return m.info, m.err }

func TestDirEntryWrapperInfo(t *testing.T) {
	tmp := t.TempDir()
	w := &DirEntryWrapper{EntryName: ".", DirPath: tmp}
	info, err := w.Info()
	if err != nil {
		t.Fatalf("Info error: %v", err)
	}
	if info.Name() != filepath.Base(tmp) {
		t.Fatalf("expected name %q, got %q", filepath.Base(tmp), info.Name())
	}

	root := &DirEntryWrapper{EntryName: tmp, DirPath: tmp, IsRoot: true}
	rootInfo, err := root.Info()
	if err != nil {
		t.Fatalf("root Info error: %v", err)
	}
	if rootInfo.Name() != filepath.Base(tmp) {
		t.Fatalf("expected root name %q, got %q", filepath.Base(tmp), rootInfo.Name())
	}
}

func TestDirEntryWrapper(t *testing.T) {
	tmp := t.TempDir()
	parent := filepath.Dir(tmp)

	// Test ".."
	w := &DirEntryWrapper{EntryName: "..", DirPath: tmp}
	if w.Name() != ".." {
		t.Fatalf("expected name '..', got %q", w.Name())
	}
	info, err := w.Info()
	if err != nil {
		t.Fatalf("Info for '..' error: %v", err)
	}
	if info.Name() != filepath.Base(parent) {
		t.Fatalf("expected name %q, got %q", filepath.Base(parent), info.Name())
	}
	if !w.IsDir() {
		t.Fatal("expected '..' to be a directory")
	}
	if w.Type()&fs.ModeDir == 0 {
		t.Fatal("expected '..' to have type ModeDir")
	}

	// Test IsDir() with error
	wError := &DirEntryWrapper{EntryName: "nonexistent", DirPath: "/nonexistent"}
	if wError.IsDir() {
		t.Fatal("expected IsDir to be false on error")
	}

	// Test Type() with error
	if wError.Type() != 0 {
		t.Fatal("expected Type to be 0 on error")
	}
}

func TestNewCachedDirEntry(t *testing.T) {
	info := &fileInfoStub{name: "a"}
	de := &mockDirEntry{name: "a", info: info}
	cached := NewCachedDirEntry(de, info)
	if cached.DirEntry != de {
		t.Fatal("DirEntry not set correctly")
	}
	if cached.info != info {
		t.Fatal("info not set correctly")
	}
}

func TestCachedDirEntryInfoCaches(t *testing.T) {
	info := &fileInfoStub{name: "a"}
	de := &CachedDirEntry{DirEntry: &mockDirEntry{name: "a", info: info}}
	first, err := de.Info()
	if err != nil {
		t.Fatalf("first Info error: %v", err)
	}
	de.DirEntry = &mockDirEntry{name: "a", err: errors.New("should not be called")}
	second, err := de.Info()
	if err != nil {
		t.Fatalf("second Info error: %v", err)
	}
	if first != second {
		t.Fatal("expected cached info on second call")
	}
}

func TestCachedDirEntryInfoError(t *testing.T) {
	expectedErr := errors.New("info error")
	de := &CachedDirEntry{DirEntry: &mockDirEntry{name: "a", err: expectedErr}}
	_, err := de.Info()
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestLessByTime(t *testing.T) {
	now := time.Now()
	later := now.Add(1 * time.Hour)
	a := &CachedDirEntry{DirEntry: &mockDirEntry{name: "a"}, Time: &later}
	b := &CachedDirEntry{DirEntry: &mockDirEntry{name: "b"}, Time: &now}
	if !LessByTime(a, b) {
		t.Fatal("expected later time to sort first")
	}

	an := &CachedDirEntry{DirEntry: &mockDirEntry{name: "a"}, Time: nil}
	bn := &CachedDirEntry{DirEntry: &mockDirEntry{name: "b"}, Time: nil}
	if !LessByTime(an, bn) {
		t.Fatal("expected name tiebreak when times are nil")
	}

	// a > b
	if LessByTime(b, a) {
		t.Fatal("expected earlier time to not sort first")
	}

	// a == b, tiebreak by name
	sameTimeA := &CachedDirEntry{DirEntry: &mockDirEntry{name: "a"}, Time: &now}
	sameTimeB := &CachedDirEntry{DirEntry: &mockDirEntry{name: "b"}, Time: &now}
	if !LessByTime(sameTimeA, sameTimeB) {
		t.Fatal("expected name tiebreak when times are equal")
	}

	// one time is nil
	aWithTime := &CachedDirEntry{DirEntry: &mockDirEntry{name: "a"}, Time: &now}
	bWithoutTime := &CachedDirEntry{DirEntry: &mockDirEntry{name: "b"}, Time: nil}
	if !LessByTime(aWithTime, bWithoutTime) {
		t.Fatal("expected entry with time to sort before entry without time")
	}
	if LessByTime(bWithoutTime, aWithTime) {
		t.Fatal("expected entry without time to not sort before entry with time")
	}
}

type fileInfoStub struct {
	name string
}

func (f *fileInfoStub) Name() string       { return f.name }
func (f *fileInfoStub) Size() int64        { return 0 }
func (f *fileInfoStub) Mode() fs.FileMode  { return 0 }
func (f *fileInfoStub) ModTime() time.Time { return time.Time{} }
func (f *fileInfoStub) IsDir() bool        { return false }
func (f *fileInfoStub) Sys() interface{}   { return nil }

func TestLessByTime_NotCachedDirEntry(t *testing.T) {
	a := &mockDirEntry{name: "a"}
	b := &mockDirEntry{name: "b"}
	if !LessByTime(a, b) {
		t.Fatal("expected name tiebreak when not CachedDirEntry")
	}
	if LessByTime(b, a) {
		t.Fatal("expected name tiebreak when not CachedDirEntry")
	}
}

func TestNewDirEntryWrapper(t *testing.T) {
	info := &fileInfoStub{name: "test"}
	testErr := errors.New("test error")

	wrapper := NewDirEntryWrapper("test", "/path", true, info, testErr)

	if wrapper.EntryName != "test" {
		t.Errorf("expected EntryName 'test', got %q", wrapper.EntryName)
	}
	if wrapper.DirPath != "/path" {
		t.Errorf("expected DirPath '/path', got %q", wrapper.DirPath)
	}
	if !wrapper.IsRoot {
		t.Error("expected IsRoot to be true")
	}
	if wrapper.info != info {
		t.Error("expected info to be set")
	}
	if wrapper.Err != testErr {
		t.Error("expected Err to be set")
	}
}

func TestDirEntryWrapperWithPrecomputedError(t *testing.T) {
	testErr := errors.New("precomputed error")
	wrapper := &DirEntryWrapper{
		EntryName: "test",
		DirPath:   "/path",
		Err:       testErr,
	}

	_, err := wrapper.Info()
	if err != testErr {
		t.Errorf("expected error %v, got %v", testErr, err)
	}

	// Call again to ensure error is cached
	_, err2 := wrapper.Info()
	if err2 != testErr {
		t.Errorf("expected cached error %v, got %v", testErr, err2)
	}
}

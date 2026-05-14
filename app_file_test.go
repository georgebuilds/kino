package main

import (
	"io"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestCopyFileContents_HappyPath(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")

	if err := os.WriteFile(src, []byte("hello"), 0600); err != nil {
		t.Fatalf("write src: %v", err)
	}

	if err := copyFileContents(src, dst); err != nil {
		t.Fatalf("copyFileContents: %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(got) != "hello" {
		t.Errorf("dst content = %q, want %q", string(got), "hello")
	}

	// Calling again should succeed because dst is overwritten (O_TRUNC)
	if err := copyFileContents(src, dst); err != nil {
		t.Errorf("second copyFileContents to same dst should succeed: %v", err)
	}
}

func TestCopyFileContents_DestExists_Overwrites(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")

	if err := os.WriteFile(src, []byte("new content"), 0600); err != nil {
		t.Fatalf("write src: %v", err)
	}
	// Pre-create dst with different content
	if err := os.WriteFile(dst, []byte("old content"), 0600); err != nil {
		t.Fatalf("write dst: %v", err)
	}

	if err := copyFileContents(src, dst); err != nil {
		t.Fatalf("copyFileContents should overwrite existing dst: %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(got) != "new content" {
		t.Errorf("dst content = %q, want %q", string(got), "new content")
	}
}

func TestCopyFileContents_MissingSource_Errors(t *testing.T) {
	dir := t.TempDir()
	err := copyFileContents(filepath.Join(dir, "nonexistent.txt"), filepath.Join(dir, "dst.txt"))
	if err == nil {
		t.Error("expected error for missing source")
	}
}

func TestIsCrossDeviceErr_DirectEXDEV(t *testing.T) {
	if !isCrossDeviceErr(syscall.EXDEV) {
		t.Error("isCrossDeviceErr(syscall.EXDEV) should be true")
	}
}

func TestIsCrossDeviceErr_WrappedInLinkError(t *testing.T) {
	linkErr := &os.LinkError{Op: "rename", Old: "a", New: "b", Err: syscall.EXDEV}
	if !isCrossDeviceErr(linkErr) {
		t.Error("isCrossDeviceErr(&LinkError{EXDEV}) should be true")
	}
}

func TestIsCrossDeviceErr_OtherError_False(t *testing.T) {
	if isCrossDeviceErr(io.EOF) {
		t.Error("isCrossDeviceErr(io.EOF) should be false")
	}
	if isCrossDeviceErr(nil) {
		t.Error("isCrossDeviceErr(nil) should be false")
	}
}

func TestSaveAndLoadLastFilePath(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	const want = "/some/path/test.kino"
	if err := saveLastFilePath(want); err != nil {
		t.Fatalf("saveLastFilePath: %v", err)
	}
	got, err := lastFilePath()
	if err != nil {
		t.Fatalf("lastFilePath: %v", err)
	}
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestOpenPath_NewFile_ReturnsOpenState(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // keep saveLastFilePath side-effect isolated
	a := &App{}
	p := filepath.Join(t.TempDir(), "new.kino")
	state, err := a.openPath(p, true)
	if err != nil {
		t.Fatalf("openPath: %v", err)
	}
	if !state.IsOpen {
		t.Error("IsOpen should be true")
	}
	if !state.IsNew {
		t.Error("IsNew should be true")
	}
	if state.Path != p {
		t.Errorf("Path = %q, want %q", state.Path, p)
	}
	if a.db == nil {
		t.Error("App.db should be set after openPath")
	}
	_ = a.db.Close()
}

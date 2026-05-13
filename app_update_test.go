package main

import (
	"archive/zip"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// ── GetAppVersion ─────────────────────────────────────────────────────────────

func TestGetAppVersion_ReturnsVersionVar(t *testing.T) {
	a := &App{}
	if got := a.GetAppVersion(); got != version {
		t.Errorf("GetAppVersion() = %q, want %q (version var)", got, version)
	}
}

// ── downloadFile ──────────────────────────────────────────────────────────────

func TestDownloadFile_WritesBodyToDisk(t *testing.T) {
	want := []byte("binary payload")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write(want)
	}))
	defer srv.Close()

	dst := filepath.Join(t.TempDir(), "out")
	if err := downloadFile(context.Background(), srv.URL, dst, nil); err != nil {
		t.Fatalf("downloadFile: %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("content = %q, want %q", got, want)
	}
}

func TestDownloadFile_Non200_Errors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	err := downloadFile(context.Background(), srv.URL, filepath.Join(t.TempDir(), "out"), nil)
	if err == nil {
		t.Error("expected error for 404 response")
	}
}

func TestDownloadFile_ConnectionRefused_Errors(t *testing.T) {
	// Start a server then immediately close it so the URL is valid but refuses connections.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	url := srv.URL
	srv.Close()

	err := downloadFile(context.Background(), url, filepath.Join(t.TempDir(), "out"), nil)
	if err == nil {
		t.Error("expected error for refused connection")
	}
}

// ── extractZip ────────────────────────────────────────────────────────────────

// makeZip writes a zip archive at path containing the given name→content entries.
func makeZip(t *testing.T, path string, entries map[string][]byte) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("makeZip create: %v", err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	for name, content := range entries {
		hdr := &zip.FileHeader{Name: name, Method: zip.Deflate}
		hdr.SetMode(0755)
		entry, err := w.CreateHeader(hdr)
		if err != nil {
			t.Fatalf("makeZip entry %q: %v", name, err)
		}
		if _, err := entry.Write(content); err != nil {
			t.Fatalf("makeZip write %q: %v", name, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("makeZip close: %v", err)
	}
}

func TestExtractZip_ExtractsFilesWithContent(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "update.zip")
	makeZip(t, zipPath, map[string][]byte{
		"kino.app/Contents/MacOS/kino": []byte("binary"),
		"kino.app/Contents/Info.plist": []byte("<plist/>"),
	})

	dst := filepath.Join(dir, "extracted")
	if err := extractZip(zipPath, dst); err != nil {
		t.Fatalf("extractZip: %v", err)
	}

	for path, want := range map[string]string{
		"kino.app/Contents/MacOS/kino": "binary",
		"kino.app/Contents/Info.plist": "<plist/>",
	} {
		got, err := os.ReadFile(filepath.Join(dst, path))
		if err != nil {
			t.Errorf("read %s: %v", path, err)
			continue
		}
		if string(got) != want {
			t.Errorf("%s: content = %q, want %q", path, got, want)
		}
	}
}

func TestExtractZip_PreservesExecutableBit(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "update.zip")
	makeZip(t, zipPath, map[string][]byte{
		"kino.app/Contents/MacOS/kino": []byte("binary"),
	})

	dst := filepath.Join(dir, "extracted")
	if err := extractZip(zipPath, dst); err != nil {
		t.Fatalf("extractZip: %v", err)
	}

	info, err := os.Stat(filepath.Join(dst, "kino.app/Contents/MacOS/kino"))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode()&0111 == 0 {
		t.Errorf("executable bit not set: mode = %v", info.Mode())
	}
}

func TestExtractZip_PathTraversal_Rejected(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "evil.zip")

	// Craft a zip with a path-traversal entry by hand.
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	e, _ := w.Create("../../../etc/passwd")
	e.Write([]byte("rooted"))
	w.Close()
	if err := os.WriteFile(zipPath, buf.Bytes(), 0600); err != nil {
		t.Fatalf("write zip: %v", err)
	}

	err := extractZip(zipPath, filepath.Join(dir, "extracted"))
	if err == nil {
		t.Error("expected error for path-traversal entry in archive")
	}
}

func TestExtractZip_CorruptArchive_Errors(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "corrupt.zip")
	if err := os.WriteFile(zipPath, []byte("this is not a zip file"), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := extractZip(zipPath, filepath.Join(dir, "out")); err == nil {
		t.Error("expected error for corrupt archive")
	}
}

func TestExtractZip_EmptyArchive_NoError(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "empty.zip")
	makeZip(t, zipPath, nil)
	if err := extractZip(zipPath, filepath.Join(dir, "out")); err != nil {
		t.Errorf("empty archive: unexpected error: %v", err)
	}
}

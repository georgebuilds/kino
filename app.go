package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"kino/internal/db"
)

// App is the Wails application struct. All exported methods become callable
// from the Vue frontend via the generated bindings.
type App struct {
	ctx context.Context
	db  *db.DB
}

func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Re-open the last used file automatically if one is recorded.
	last, _ := lastFilePath()
	if last != "" {
		if opened, err := db.Open(last); err == nil {
			a.db = opened
		}
	}
}

func (a *App) shutdown(_ context.Context) {
	if a.db != nil {
		_ = a.db.Close()
	}
}

// ─── File management ──────────────────────────────────────────────────────────

// FileState reports what the frontend needs to know about the current file.
type FileState struct {
	Path    string `json:"path"`
	IsOpen  bool   `json:"isOpen"`
	IsNew   bool   `json:"isNew"`
}

// GetFileState returns the current .kino file state.
func (a *App) GetFileState() FileState {
	if a.db == nil {
		return FileState{}
	}
	return FileState{Path: a.db.Path, IsOpen: true}
}

// CreateFile opens a save-file dialog so the user can choose where to create
// a new .kino database. Returns the chosen path on success.
func (a *App) CreateFile() (FileState, error) {
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Create Kino file",
		DefaultFilename: "My Finances.kino",
		Filters: []runtime.FileFilter{
			{DisplayName: "Kino files (*.kino)", Pattern: "*.kino"},
		},
	})
	if err != nil || path == "" {
		return FileState{}, err
	}

	return a.openPath(path, true)
}

// OpenFile opens a file-picker dialog so the user can open an existing .kino file.
func (a *App) OpenFile() (FileState, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Open Kino file",
		Filters: []runtime.FileFilter{
			{DisplayName: "Kino files (*.kino)", Pattern: "*.kino"},
		},
	})
	if err != nil || path == "" {
		return FileState{}, err
	}

	return a.openPath(path, false)
}

// MoveFile lets the user relocate the current .kino file (e.g. into iCloud).
func (a *App) MoveFile() (FileState, error) {
	if a.db == nil {
		return FileState{}, fmt.Errorf("no file is open")
	}

	newPath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Move Kino file",
		DefaultFilename: filepath.Base(a.db.Path),
		Filters: []runtime.FileFilter{
			{DisplayName: "Kino files (*.kino)", Pattern: "*.kino"},
		},
	})
	if err != nil || newPath == "" {
		return FileState{}, err
	}

	oldPath := a.db.Path
	_ = a.db.Close()
	a.db = nil

	if err := os.Rename(oldPath, newPath); err != nil {
		// Re-open the original if move failed.
		if reopened, rerr := db.Open(oldPath); rerr == nil {
			a.db = reopened
		}
		return FileState{}, fmt.Errorf("move failed: %w", err)
	}

	return a.openPath(newPath, false)
}

// CloudFolderSuggestions returns paths to common cloud-synced folders that
// exist on this machine, so the UI can recommend them.
func (a *App) CloudFolderSuggestions() []CloudFolder {
	home, _ := os.UserHomeDir()

	candidates := []CloudFolder{
		{Name: "iCloud Drive", Path: filepath.Join(home, "Library/Mobile Documents/com~apple~CloudDocs")},
		{Name: "Dropbox",      Path: filepath.Join(home, "Dropbox")},
		{Name: "Google Drive", Path: filepath.Join(home, "Google Drive")},
		{Name: "Google Drive", Path: filepath.Join(home, "My Drive")},
		{Name: "OneDrive",     Path: filepath.Join(home, "OneDrive")},
		// Windows path for OneDrive
		{Name: "OneDrive", Path: filepath.Join(home, "OneDrive - Personal")},
	}

	var found []CloudFolder
	seen := map[string]bool{}

	// Proton Drive mounts via macOS File Provider at ~/Library/CloudStorage/ProtonDrive-<email>
	// and on Windows at ~/Proton Drive. Handle both.
	if matches, _ := filepath.Glob(filepath.Join(home, "Library/CloudStorage/ProtonDrive-*")); len(matches) > 0 {
		if !seen["Proton Drive"] {
			found = append(found, CloudFolder{Name: "Proton Drive", Path: matches[0]})
			seen["Proton Drive"] = true
		}
	}
	if p := filepath.Join(home, "Proton Drive"); !seen["Proton Drive"] {
		if _, err := os.Stat(p); err == nil {
			found = append(found, CloudFolder{Name: "Proton Drive", Path: p})
			seen["Proton Drive"] = true
		}
	}

	for _, c := range candidates {
		if seen[c.Name] {
			continue
		}
		if _, err := os.Stat(c.Path); err == nil {
			found = append(found, c)
			seen[c.Name] = true
		}
	}
	return found
}

type CloudFolder struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// ─── Internal helpers ─────────────────────────────────────────────────────────

func (a *App) requireDB() error {
	if a.db == nil {
		return fmt.Errorf("no file open — create or open a .kino file first")
	}
	return nil
}

func (a *App) openPath(path string, isNew bool) (FileState, error) {
	if a.db != nil {
		_ = a.db.Close()
	}

	opened, err := db.Open(path)
	if err != nil {
		return FileState{}, err
	}

	a.db = opened
	_ = saveLastFilePath(path)

	return FileState{Path: path, IsOpen: true, IsNew: isNew}, nil
}

// lastFilePath persists the most-recently-used .kino path in the user config dir.
func lastFilePath() (string, error) {
	p, err := mruPath()
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(p)
	return string(b), err
}

func saveLastFilePath(path string) error {
	p, err := mruPath()
	if err != nil {
		return err
	}
	return os.WriteFile(p, []byte(path), 0600)
}

func mruPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	d := filepath.Join(dir, "kino")
	if err := os.MkdirAll(d, 0700); err != nil {
		return "", err
	}
	return filepath.Join(d, "last_file"), nil
}

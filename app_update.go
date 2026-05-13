package main

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"time"

	selfupdate "github.com/creativeprojects/go-selfupdate"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// version is injected at build time via -ldflags "-X main.version=x.y.z"
var version = "0.0.0-dev"

const repoSlug = "georgebuilds/kino"

// UpdateInfo is returned to the frontend.
type UpdateInfo struct {
	Available    bool   `json:"available"`
	Version      string `json:"version"`
	URL          string `json:"url"`
	ReleaseNotes string `json:"releaseNotes"`
}

// GetAppVersion returns the version baked in at build time.
func (a *App) GetAppVersion() string { return version }

// CheckForUpdate queries GitHub Releases and reports any newer version.
// The release is cached on the App so ApplyUpdate can use it directly.
func (a *App) CheckForUpdate() (UpdateInfo, error) {
	updater, err := selfupdate.NewUpdater(selfupdate.Config{})
	if err != nil {
		return UpdateInfo{}, err
	}

	release, found, err := updater.DetectLatest(context.Background(), selfupdate.ParseSlug(repoSlug))
	if err != nil {
		return UpdateInfo{}, fmt.Errorf("checking for updates: %w", err)
	}
	if !found || release.LessOrEqual(version) {
		a.pendingRelease = nil
		return UpdateInfo{Available: false}, nil
	}

	a.pendingRelease = release
	return UpdateInfo{
		Available:    true,
		Version:      release.Version(),
		URL:          release.URL,
		ReleaseNotes: release.ReleaseNotes,
	}, nil
}

// ApplyUpdate downloads and installs the latest release, then quits (or
// relaunches on macOS).
//
// Uses the release cached by CheckForUpdate when available so the version the
// user saw in the UI is exactly the one installed (no second DetectLatest call
// and no race with a release shipping between check and apply).
func (a *App) ApplyUpdate() error {
	updater, err := selfupdate.NewUpdater(selfupdate.Config{})
	if err != nil {
		return err
	}

	release := a.pendingRelease
	if release == nil {
		// Fallback path: called without a prior CheckForUpdate (shouldn't
		// happen via the UI, but handle it gracefully).
		var found bool
		release, found, err = updater.DetectLatest(context.Background(), selfupdate.ParseSlug(repoSlug))
		if err != nil {
			return fmt.Errorf("detecting release: %w", err)
		}
		if !found {
			return fmt.Errorf("no update found")
		}
	}
	a.pendingRelease = nil // consume

	if goruntime.GOOS == "darwin" {
		return a.applyMacOSUpdate(release)
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding executable: %w", err)
	}
	if err := updater.UpdateTo(context.Background(), release, exe); err != nil {
		return fmt.Errorf("applying update: %w", err)
	}
	wailsruntime.Quit(a.ctx)
	return nil
}

// applyMacOSUpdate downloads the release zip, extracts the full app bundle,
// replaces the installed bundle atomically, and relaunches.
func (a *App) applyMacOSUpdate(release *selfupdate.Release) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	// os.Executable() inside the bundle is e.g.
	// /Applications/kino.app/Contents/MacOS/kino — go up three levels.
	appBundle := filepath.Clean(filepath.Join(exe, "../../.."))
	if !strings.HasSuffix(appBundle, ".app") {
		// Dev mode (not inside a bundle) — open the releases page instead.
		wailsruntime.BrowserOpenURL(a.ctx, release.URL)
		return nil
	}

	tmpDir, err := os.MkdirTemp("", "kino-update-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	zipPath := filepath.Join(tmpDir, "update.zip")
	onProgress := func(downloaded, total int64) {
		pct := 0.0
		if total > 0 {
			pct = float64(downloaded) / float64(total) * 100
		}
		wailsruntime.EventsEmit(a.ctx, "update:progress", map[string]any{
			"downloaded": downloaded,
			"total":      total,
			"percent":    pct,
		})
	}
	if err := downloadFile(a.ctx, release.AssetURL, zipPath, onProgress); err != nil {
		return fmt.Errorf("downloading update: %w", err)
	}

	extractDir := filepath.Join(tmpDir, "extracted")
	if err := extractZip(zipPath, extractDir); err != nil {
		return fmt.Errorf("extracting update: %w", err)
	}

	newApp := filepath.Join(extractDir, "kino.app")
	if _, err := os.Stat(newApp); err != nil {
		return fmt.Errorf("kino.app not found in update archive")
	}

	// Stage the new bundle next to the installed one so the final rename is
	// on the same filesystem (atomic).
	parent := filepath.Dir(appBundle)
	staged := filepath.Join(parent, "kino-new.app")
	os.RemoveAll(staged)
	// ditto preserves extended attributes and code signatures.
	if err := exec.Command("ditto", newApp, staged).Run(); err != nil {
		return fmt.Errorf("staging new bundle: %w", err)
	}

	backup := appBundle + ".bak"
	os.RemoveAll(backup)
	if err := os.Rename(appBundle, backup); err != nil {
		os.RemoveAll(staged)
		return fmt.Errorf("moving current app aside: %w", err)
	}
	if err := os.Rename(staged, appBundle); err != nil {
		os.Rename(backup, appBundle) // best-effort restore
		return fmt.Errorf("installing update: %w", err)
	}
	os.RemoveAll(backup)

	_ = exec.Command("open", "-a", appBundle).Start()
	wailsruntime.Quit(a.ctx)
	return nil
}

// progressReader wraps an io.Reader and calls onProgress after each read,
// throttled to once per integer-percent change.
type progressReader struct {
	r          io.Reader
	total      int64
	downloaded int64
	lastPct    int
	onProgress func(downloaded, total int64)
}

func (pr *progressReader) Read(b []byte) (int, error) {
	n, err := pr.r.Read(b)
	if n > 0 && pr.total > 0 {
		pr.downloaded += int64(n)
		newPct := int(float64(pr.downloaded) / float64(pr.total) * 100)
		if newPct != pr.lastPct {
			pr.lastPct = newPct
			pr.onProgress(pr.downloaded, pr.total)
		}
	}
	return n, err
}

func downloadFile(ctx context.Context, url, dst string, onProgress func(downloaded, total int64)) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d downloading asset", resp.StatusCode)
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()

	var src io.Reader = resp.Body
	if onProgress != nil {
		src = &progressReader{r: resp.Body, total: resp.ContentLength, onProgress: onProgress}
	}
	_, err = io.Copy(f, src)
	return err
}

func extractZip(src, dst string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	dstClean := filepath.Clean(dst) + string(os.PathSeparator)

	for _, f := range r.File {
		target := filepath.Join(dst, f.Name)
		// Reject path traversal.
		if !strings.HasPrefix(filepath.Clean(target)+string(os.PathSeparator), dstClean) {
			return fmt.Errorf("unsafe path in archive: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, f.Mode()); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}
		_, copyErr := io.Copy(out, rc)
		rc.Close()
		out.Close()
		if copyErr != nil {
			return copyErr
		}
	}
	return nil
}

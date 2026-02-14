package updater

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"cs2admin/internal/pkg/logger"
)

// Release represents a GitHub release.
type Release struct {
	Version     string  `json:"tag_name"`
	Name        string  `json:"name"`
	Body        string  `json:"body"`
	PublishedAt string  `json:"published_at"`
	HTMLURL     string  `json:"html_url"`
	Assets      []Asset `json:"assets"`
}

// Asset represents a release asset.
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
	ContentType        string `json:"content_type"`
}

// UpdateInfo holds update check results.
type UpdateInfo struct {
	Available      bool   `json:"available"`
	Version        string `json:"version"`
	CurrentVersion string `json:"current_version"`
	ReleaseNotes   string `json:"release_notes"`
	DownloadURL    string `json:"download_url"`
	Size           int64  `json:"size"`
}

// CheckForUpdate queries the GitHub Releases API for the latest version and returns
// UpdateInfo with Available=true if a newer version exists.
func CheckForUpdate(currentVersion string, owner string, repo string) (*UpdateInfo, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api status %d", resp.StatusCode)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decode release: %w", err)
	}

	latest := normalizeVersion(release.Version)
	current := normalizeVersion(currentVersion)

	if !versionGreater(latest, current) {
		return &UpdateInfo{
			Available:      false,
			Version:        release.Version,
			CurrentVersion: currentVersion,
			ReleaseNotes:   release.Body,
			DownloadURL:    "",
			Size:           0,
		}, nil
	}

	// Pick first .exe asset on Windows
	var downloadURL string
	var size int64
	for _, a := range release.Assets {
		if runtime.GOOS == "windows" && strings.HasSuffix(strings.ToLower(a.Name), ".exe") {
			downloadURL = a.BrowserDownloadURL
			size = a.Size
			break
		}
	}
	if downloadURL == "" && len(release.Assets) > 0 {
		downloadURL = release.Assets[0].BrowserDownloadURL
		size = release.Assets[0].Size
	}

	return &UpdateInfo{
		Available:      true,
		Version:        release.Version,
		CurrentVersion: currentVersion,
		ReleaseNotes:   release.Body,
		DownloadURL:    downloadURL,
		Size:           size,
	}, nil
}

// DownloadUpdate downloads the file to destPath, logging progress.
func DownloadUpdate(downloadURL string, destPath string) error {
	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download status %d", resp.StatusCode)
	}

	total := resp.ContentLength
	if total < 0 {
		total = 0
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer out.Close()

	buf := make([]byte, 32*1024)
	var written int64
	lastPercent := -1.0

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			wn, wErr := out.Write(buf[:n])
			written += int64(wn)
			if wErr != nil {
				return fmt.Errorf("write: %w", wErr)
			}
			if wn != n {
				return fmt.Errorf("short write")
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}

		if total > 0 {
			pct := float64(written) / float64(total) * 100
			if pct-lastPercent >= 10 || lastPercent < 0 {
				logger.Log.Debug().Float64("percent", pct).Int64("downloaded", written).Int64("total", total).Msg("Update download progress")
				lastPercent = pct
			}
		}
	}

	logger.Log.Info().Int64("bytes", written).Str("dest", destPath).Msg("Download complete")
	return nil
}

// VerifyChecksum verifies the file's SHA-256 against the expected hex string.
func VerifyChecksum(filePath string, expectedSHA256 string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("hash: %w", err)
	}

	actual := hex.EncodeToString(h.Sum(nil))
	expected := strings.ToLower(strings.TrimSpace(expectedSHA256))
	if actual != expected {
		return fmt.Errorf("checksum mismatch: got %s, want %s", actual, expected)
	}
	return nil
}

// ApplyUpdate replaces the current executable with the new one.
// On Windows: renames current exe to .old, moves new exe to current location,
// and schedules cleanup of .old on next run.
func ApplyUpdate(newBinaryPath string) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("ApplyUpdate only supported on Windows")
	}

	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}

	// Resolve symlinks to get real path
	currentExe, err = filepath.EvalSymlinks(currentExe)
	if err != nil {
		return fmt.Errorf("resolve exe path: %w", err)
	}

	oldPath := currentExe + ".old"

	// Remove existing .old from previous failed update
	if _, err := os.Stat(oldPath); err == nil {
		os.Remove(oldPath)
	}

	// Rename current exe to .old
	if err := os.Rename(currentExe, oldPath); err != nil {
		return fmt.Errorf("rename current to .old: %w", err)
	}

	// Move new binary to current location
	if err := os.Rename(newBinaryPath, currentExe); err != nil {
		// Restore on failure
		os.Rename(oldPath, currentExe)
		return fmt.Errorf("move new binary: %w", err)
	}

	logger.Log.Info().Str("path", currentExe).Msg("Update applied; .old will be cleaned on next run")
	return nil
}

// CleanupOldBinary removes the .old executable from a previous update.
// Call this at application startup.
func CleanupOldBinary() {
	if runtime.GOOS != "windows" {
		return
	}
	currentExe, err := os.Executable()
	if err != nil {
		return
	}
	currentExe, _ = filepath.EvalSymlinks(currentExe)
	oldPath := currentExe + ".old"
	if _, err := os.Stat(oldPath); err == nil {
		if removeErr := os.Remove(oldPath); removeErr != nil {
			logger.Log.Warn().Err(removeErr).Str("path", oldPath).Msg("Failed to remove old binary")
		} else {
			logger.Log.Info().Str("path", oldPath).Msg("Removed old binary")
		}
	}
}

// normalizeVersion strips a leading 'v' for comparison.
func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	if strings.HasPrefix(strings.ToLower(v), "v") {
		return v[1:]
	}
	return v
}

// versionGreater returns true if a > b (semver comparison).
// Handles versions like 0.1.0, 1.2.3, etc.
func versionGreater(a, b string) bool {
	aParts := strings.Split(normalizeVersion(a), ".")
	bParts := strings.Split(normalizeVersion(b), ".")
	maxLen := len(aParts)
	if len(bParts) > maxLen {
		maxLen = len(bParts)
	}
	for i := 0; i < maxLen; i++ {
		var aVal, bVal int
		if i < len(aParts) {
			aVal, _ = strconv.Atoi(strings.TrimSpace(aParts[i]))
		}
		if i < len(bParts) {
			bVal, _ = strconv.Atoi(strings.TrimSpace(bParts[i]))
		}
		if aVal > bVal {
			return true
		}
		if aVal < bVal {
			return false
		}
	}
	return false
}

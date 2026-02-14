package instance

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"cs2admin/internal/pkg/logger"
)

// PluginInfo describes an installed or available plugin.
type PluginInfo struct {
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
	Version   string `json:"version"`
	Path      string `json:"path"`
	Enabled   bool   `json:"enabled"`
}

// Known plugin release URLs (placeholders - resolved at runtime via GitHub API or static URLs).
var (
	// MetamodSourceURL - Metamod:Source for CS2 (Windows). AlliedModders/sourcemm.net.
	MetamodSourceURL = "https://mms.alliedmods.net/mms-drop/2.0/mmsource-2.0.0-git1384-windows.zip"
	// CounterStrikeSharpURL - CounterStrikeSharp with runtime.
	CounterStrikeSharpURL = "https://github.com/roflmuffin/CounterStrikeSharp/releases/download/v1.0.362/counterstrikesharp-with-runtime-windows-1.0.362.zip"
	// WeaponPaintsURL - WeaponPaints plugin for CSS.
	WeaponPaintsURL = "https://github.com/Nereziel/cs2-WeaponPaints/releases/download/build-411/WeaponPaints.zip"
)

// GetInstalledPlugins scans the addons directory and returns plugin info.
func GetInstalledPlugins(installPath string) ([]PluginInfo, error) {
	addonsPath := filepath.Join(installPath, "game", "csgo", "addons")
	pluginsPath := filepath.Join(addonsPath, "plugins")

	if _, err := os.Stat(addonsPath); os.IsNotExist(err) {
		return nil, nil
	}

	var result []PluginInfo

	metamodPath := filepath.Join(addonsPath, "metamod")
	if fi, err := os.Stat(metamodPath); err == nil && fi.IsDir() {
		result = append(result, PluginInfo{
			Name:      "metamod",
			Installed: true,
			Version:   "2.0",
			Path:      metamodPath,
			Enabled:   true,
		})
	}

	cssPath := filepath.Join(addonsPath, "counterstrikesharp")
	if fi, err := os.Stat(cssPath); err == nil && fi.IsDir() {
		result = append(result, PluginInfo{
			Name:      "counterstrikesharp",
			Installed: true,
			Version:   "1.0",
			Path:      cssPath,
			Enabled:   true,
		})
	}

	wpPath := filepath.Join(pluginsPath, "WeaponPaints")
	if fi, err := os.Stat(wpPath); err == nil && fi.IsDir() {
		result = append(result, PluginInfo{
			Name:      "WeaponPaints",
			Installed: true,
			Version:   "",
			Path:      wpPath,
			Enabled:   true,
		})
	}

	return result, nil
}

// InstallMetamod downloads and extracts Metamod:Source to the CS2 addons directory.
func InstallMetamod(installPath string) error {
	url := MetamodSourceURL
	if runtime.GOOS == "linux" {
		url = "https://mms.alliedmods.net/mms-drop/2.0/mmsource-2.0.0-git1384-linux.zip"
	}
	return downloadAndExtractZip(installPath, url, "game/csgo", "metamod", "Metamod:Source")
}

// InstallCounterStrikeSharp downloads and extracts CounterStrikeSharp to the CS2 addons directory.
func InstallCounterStrikeSharp(installPath string) error {
	url := CounterStrikeSharpURL
	if runtime.GOOS == "linux" {
		url = "https://github.com/roflmuffin/CounterStrikeSharp/releases/download/v1.0.362/counterstrikesharp-with-runtime-linux-1.0.362.zip"
	}
	return downloadAndExtractZip(installPath, url, "game/csgo", "counterstrikesharp", "CounterStrikeSharp")
}

// InstallWeaponPaints downloads and extracts WeaponPaints plugin to addons/counterstrikesharp/plugins/WeaponPaints.
func InstallWeaponPaints(installPath string) error {
	url := WeaponPaintsURL
	basePath := filepath.Join(installPath, "game", "csgo", "addons", "counterstrikesharp", "plugins")
	return downloadAndExtractZipTo(installPath, url, basePath, "WeaponPaints", "WeaponPaints")
}

func downloadAndExtractZip(installPath, url, subDir, expectDir, label string) error {
	basePath := filepath.Join(installPath, subDir)
	return downloadAndExtractZipTo(installPath, url, basePath, expectDir, label)
}

func downloadAndExtractZipTo(installPath, url, basePath, expectDir, label string) error {
	logger.Log.Info().Str("plugin", label).Str("url", url).Msg("downloading plugin")

	client := &http.Client{Timeout: 120 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "CS2Admin/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download %s: %w", label, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: HTTP %d", label, resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp("", "cs2admin-plugin-*.zip")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)
	defer tmpFile.Close()

	n, err := io.Copy(tmpFile, resp.Body)
	if err != nil {
		return fmt.Errorf("save %s zip: %w", label, err)
	}
	logger.Log.Info().Str("plugin", label).Int64("bytes", n).Msg("downloaded")

	if err = tmpFile.Sync(); err != nil {
		return fmt.Errorf("sync temp file: %w", err)
	}
	if _, err = tmpFile.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("seek temp file: %w", err)
	}

	rd, err := zip.NewReader(tmpFile, n)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}

	if err := os.MkdirAll(basePath, 0755); err != nil {
		return fmt.Errorf("create directory %s: %w", basePath, err)
	}

	prefix := findZipRootPrefix(rd, expectDir)

	extracted := 0
	for _, f := range rd.File {
		name := f.Name
		if prefix != "" && strings.HasPrefix(name, prefix) {
			name = name[len(prefix):]
		}
		if name == "" || name == "." {
			continue
		}
		target := filepath.Join(basePath, filepath.FromSlash(name))
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("mkdir %s: %w", target, err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return fmt.Errorf("mkdir parent: %w", err)
		}
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("open zip entry %s: %w", f.Name, err)
		}
		dst, err := os.Create(target)
		if err != nil {
			rc.Close()
			return fmt.Errorf("create %s: %w", target, err)
		}
		_, err = io.Copy(dst, rc)
		rc.Close()
		dst.Close()
		if err != nil {
			return fmt.Errorf("extract %s: %w", f.Name, err)
		}
		extracted++
	}

	logger.Log.Info().Str("plugin", label).Int("files", extracted).Str("path", basePath).Msg("plugin installed")
	return nil
}

// findZipRootPrefix finds a single top-level folder to strip so "addons/" or plugin dir lands in basePath.
// If expectDir matches the zip root, we keep it (no strip).
func findZipRootPrefix(rd *zip.Reader, expectDir string) string {
	var root string
	for _, f := range rd.File {
		parts := strings.Split(strings.ReplaceAll(f.Name, "\\", "/"), "/")
		if len(parts) >= 1 && parts[0] != "" {
			if root == "" {
				root = parts[0]
			} else if parts[0] != root {
				return ""
			}
		}
	}
	if root == "" || root == "addons" || root == "game" || root == expectDir {
		return ""
	}
	return root + "/"
}

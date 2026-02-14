package steam

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"cs2admin/internal/pkg/logger"
)

const steamcmdURL = "https://steamcdn-a.akamaihd.net/client/installer/steamcmd.zip"

// SteamCMD wraps the SteamCMD CLI for installing and updating CS2.
type SteamCMD struct {
	path string // directory where steamcmd.exe lives
	mu   sync.Mutex
}

// New creates a new SteamCMD wrapper. basePath is the directory where SteamCMD will be installed.
func New(basePath string) *SteamCMD {
	return &SteamCMD{path: basePath}
}

// ExePath returns the full path to steamcmd.exe.
func (s *SteamCMD) ExePath() string {
	exe := "steamcmd"
	if runtime.GOOS == "windows" {
		exe = "steamcmd.exe"
	}
	return filepath.Join(s.path, exe)
}

// EnsureInstalled checks if steamcmd.exe exists; if not, downloads and extracts it.
func (s *SteamCMD) EnsureInstalled() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	exePath := s.ExePath()
	if _, err := os.Stat(exePath); err == nil {
		logger.Log.Debug().Str("path", exePath).Msg("SteamCMD already installed")
		return nil
	}

	if err := os.MkdirAll(s.path, 0755); err != nil {
		return fmt.Errorf("create steamcmd dir: %w", err)
	}

	logger.Log.Info().Str("url", steamcmdURL).Msg("Downloading SteamCMD")
	resp, err := http.Get(steamcmdURL)
	if err != nil {
		return fmt.Errorf("download steamcmd: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download steamcmd: status %d", resp.StatusCode)
	}

	zipPath := filepath.Join(s.path, "steamcmd.zip")
	out, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("create zip file: %w", err)
	}

	written, err := io.Copy(out, resp.Body)
	out.Close()
	if err != nil {
		os.Remove(zipPath)
		return fmt.Errorf("write steamcmd zip: %w", err)
	}
	logger.Log.Debug().Int64("bytes", written).Msg("SteamCMD zip downloaded")

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		os.Remove(zipPath)
		return fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		dest := filepath.Join(s.path, f.Name)
		if err := extractFile(f, dest); err != nil {
			os.Remove(zipPath)
			return fmt.Errorf("extract %s: %w", f.Name, err)
		}
	}

	if err := os.Remove(zipPath); err != nil {
		logger.Log.Warn().Err(err).Msg("Failed to remove steamcmd.zip after extract")
	}

	if _, err := os.Stat(exePath); err != nil {
		return fmt.Errorf("steamcmd.exe not found after extract: %w", err)
	}
	logger.Log.Info().Str("path", exePath).Msg("SteamCMD installed")
	return nil
}

func extractFile(f *zip.File, dest string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	_, err = io.Copy(out, rc)
	out.Close()
	return err
}

// Run executes SteamCMD with the given args, parses stdout for progress, and sends
// Progress structs to progressCh. Closes progressCh when done. progressCh may be nil.
func (s *SteamCMD) Run(args []string, progressCh chan<- Progress) error {
	if err := s.EnsureInstalled(); err != nil {
		return err
	}

	cmd := exec.Command(s.ExePath(), args...)
	hideWindow(cmd) // prevent visible console window on Windows
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start steamcmd: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		logger.Log.Debug().Str("steamcmd", line).Msg("")
		if p := ParseProgressLine(line); p != nil && progressCh != nil {
			select {
			case progressCh <- *p:
			default:
				// Channel full or closed; don't block
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		if progressCh != nil {
			close(progressCh)
		}
		return fmt.Errorf("steamcmd exit: %w", err)
	}

	if progressCh != nil {
		close(progressCh)
	}
	return nil
}

// InstallCS2 installs CS2 to installDir with validation.
func (s *SteamCMD) InstallCS2(installDir string, progressCh chan<- Progress) error {
	absDir, err := filepath.Abs(installDir)
	if err != nil {
		return fmt.Errorf("resolve install dir: %w", err)
	}
	args := []string{
		"+login", "anonymous",
		"+force_install_dir", absDir,
		"+app_update", "730", "validate",
		"+quit",
	}
	return s.Run(args, progressCh)
}

// UpdateCS2 updates CS2 at installDir without validation.
func (s *SteamCMD) UpdateCS2(installDir string, progressCh chan<- Progress) error {
	absDir, err := filepath.Abs(installDir)
	if err != nil {
		return fmt.Errorf("resolve install dir: %w", err)
	}
	args := []string{
		"+login", "anonymous",
		"+force_install_dir", absDir,
		"+app_update", "730",
		"+quit",
	}
	return s.Run(args, progressCh)
}

// ValidateCS2 validates CS2 files at installDir.
func (s *SteamCMD) ValidateCS2(installDir string, progressCh chan<- Progress) error {
	absDir, err := filepath.Abs(installDir)
	if err != nil {
		return fmt.Errorf("resolve install dir: %w", err)
	}
	args := []string{
		"+login", "anonymous",
		"+force_install_dir", absDir,
		"+app_update", "730", "validate",
		"+quit",
	}
	return s.Run(args, progressCh)
}

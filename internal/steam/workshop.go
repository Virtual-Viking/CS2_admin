package steam

import (
	"fmt"
	"path/filepath"
	"strconv"
)

// DownloadWorkshopItem downloads a Steam Workshop item for CS2 (App ID 730).
func (s *SteamCMD) DownloadWorkshopItem(installDir string, workshopID int64, progressCh chan<- Progress) error {
	absDir, err := filepath.Abs(installDir)
	if err != nil {
		return fmt.Errorf("resolve install dir: %w", err)
	}
	args := []string{
		"+login", "anonymous",
		"+force_install_dir", absDir,
		"+workshop_download_item", "730", strconv.FormatInt(workshopID, 10),
		"+quit",
	}
	return s.Run(args, progressCh)
}

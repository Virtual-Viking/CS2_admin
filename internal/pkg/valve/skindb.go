package valve

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"cs2admin/internal/models"
	"cs2admin/internal/pkg/logger"
	"gorm.io/gorm"
)

const itemsGameURL = "https://raw.githubusercontent.com/SteamDatabase/GameTracking-CS2/master/game/csgo/pak01_dir/scripts/items/items_game.txt"

// FetchItemsGame downloads items_game.txt from SteamDatabase GameTracking.
func FetchItemsGame() ([]byte, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest(http.MethodGet, itemsGameURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "CS2Admin/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch items_game: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch items_game: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read items_game: %w", err)
	}
	return data, nil
}

// SeedSkinDatabase downloads items_game.txt, parses it, and populates the skins table.
func SeedSkinDatabase(db *gorm.DB) error {
	logger.Log.Info().Msg("fetching items_game.txt for skin database")
	data, err := FetchItemsGame()
	if err != nil {
		return err
	}

	root, err := ParseVDF(data)
	if err != nil {
		return fmt.Errorf("parse VDF: %w", err)
	}

	skins, err := ExtractSkins(root)
	if err != nil {
		return fmt.Errorf("extract skins: %w", err)
	}

	logger.Log.Info().Int("count", len(skins)).Msg("extracted skins from items_game")

	for _, s := range skins {
		skin := &models.Skin{
			PaintKitID:  s.PaintKitID,
			Name:        s.Name,
			WeaponType:  s.WeaponType,
			Rarity:      s.Rarity,
			RarityColor: s.RarityColor,
			MinFloat:    s.MinFloat,
			MaxFloat:    s.MaxFloat,
			Category:    s.Category,
			Collection:  s.Collection,
		}
		if err := db.Where("paint_kit_id = ?", s.PaintKitID).FirstOrCreate(skin).Error; err != nil {
			logger.Log.Warn().Err(err).Int("paint_kit_id", s.PaintKitID).Str("name", s.Name).Msg("skip error upserting skin")
			continue
		}
	}

	logger.Log.Info().Int("inserted", len(skins)).Msg("skin database seeded")
	return nil
}

// UpdateSkinDatabase clears existing skin data and re-seeds from items_game.txt.
func UpdateSkinDatabase(db *gorm.DB) error {
	logger.Log.Info().Msg("updating skin database")
	if err := db.Where("1 = 1").Delete(&models.Skin{}).Error; err != nil {
		return fmt.Errorf("clear skins: %w", err)
	}
	return SeedSkinDatabase(db)
}

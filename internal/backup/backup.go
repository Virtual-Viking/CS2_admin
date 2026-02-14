package backup

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"cs2admin/internal/models"
	"cs2admin/internal/pkg/logger"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BackupType defines the scope of a backup.
type BackupType string

const (
	BackupFull        BackupType = "full"
	BackupConfigOnly BackupType = "config"
	BackupMapsOnly   BackupType = "maps"
	BackupPluginsOnly BackupType = "plugins"
)

// Create creates a zip backup based on type and saves the Backup record to DB.
func Create(db *gorm.DB, instanceID string, installPath string, backupDir string, bType BackupType) (*models.Backup, error) {
	// Ensure backup dir exists
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, fmt.Errorf("create backup dir: %w", err)
	}

	shortID := instanceID
	if len(instanceID) > 8 {
		shortID = instanceID[:8]
	}

	ts := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("cs2admin_%s_%s_%s.zip", shortID, string(bType), ts)
	zipPath := filepath.Join(backupDir, filename)

	// Determine source paths to backup
	var sources []string
	switch bType {
	case BackupFull:
		sources = []string{installPath}
	case BackupConfigOnly:
		sources = []string{filepath.Join(installPath, "game", "csgo", "cfg")}
	case BackupMapsOnly:
		sources = []string{filepath.Join(installPath, "game", "csgo", "maps")}
	case BackupPluginsOnly:
		sources = []string{filepath.Join(installPath, "game", "csgo", "addons")}
	default:
		return nil, fmt.Errorf("invalid backup type: %s", bType)
	}

	// Create zip
	if err := createZip(zipPath, installPath, sources); err != nil {
		return nil, fmt.Errorf("create zip: %w", err)
	}

	// Get file size
	info, err := os.Stat(zipPath)
	if err != nil {
		os.Remove(zipPath)
		return nil, fmt.Errorf("stat backup file: %w", err)
	}

	// Save DB record
	instUUID, err := uuid.Parse(instanceID)
	if err != nil {
		os.Remove(zipPath)
		return nil, fmt.Errorf("invalid instance id: %w", err)
	}

	b := &models.Backup{
		InstanceID: instUUID,
		Path:       zipPath,
		SizeBytes:  info.Size(),
		BackupType: string(bType),
	}

	if err := db.Create(b).Error; err != nil {
		os.Remove(zipPath)
		return nil, fmt.Errorf("save backup record: %w", err)
	}

	logger.Log.Info().
		Str("instance", instanceID).
		Str("path", zipPath).
		Int64("size", info.Size()).
		Str("type", string(bType)).
		Msg("backup: created")

	return b, nil
}

func createZip(zipPath, rootPath string, sourceDirs []string) error {
	f, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	for _, dir := range sourceDirs {
		if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return nil // skip missing dirs
				}
				return err
			}

			rel, err := filepath.Rel(rootPath, path)
			if err != nil {
				rel = path
			}
			// Ensure forward slashes in zip
			rel = filepath.ToSlash(rel)

			if info.IsDir() {
				rel += "/"
				_, err := w.Create(rel)
				return err
			}

			zf, err := w.Create(rel)
			if err != nil {
				return err
			}

			in, err := os.Open(path)
			if err != nil {
				return err
			}
			_, err = io.Copy(zf, in)
			in.Close()
			return err
		}); err != nil {
			return err
		}
	}

	return nil
}

// Restore extracts the backup zip to installPath.
func Restore(db *gorm.DB, backupID string, installPath string) error {
	var b models.Backup
	if err := db.First(&b, "id = ?", backupID).Error; err != nil {
		return fmt.Errorf("backup not found: %w", err)
	}

	if _, err := os.Stat(b.Path); err != nil {
		return fmt.Errorf("backup file missing: %w", err)
	}

	r, err := zip.OpenReader(b.Path)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		dest := filepath.Join(installPath, filepath.FromSlash(f.Name))

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(dest, 0755); err != nil {
				return fmt.Errorf("mkdir %s: %w", dest, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return fmt.Errorf("mkdir parent %s: %w", dest, err)
		}

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("open zip entry %s: %w", f.Name, err)
		}

		out, err := os.Create(dest)
		if err != nil {
			rc.Close()
			return fmt.Errorf("create %s: %w", dest, err)
		}

		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return fmt.Errorf("write %s: %w", dest, err)
		}
	}

	logger.Log.Info().
		Str("backup_id", backupID).
		Str("install_path", installPath).
		Msg("backup: restored")

	return nil
}

// List returns backups for the given instance.
func List(db *gorm.DB, instanceID string) ([]models.Backup, error) {
	var backups []models.Backup
	err := db.Where("instance_id = ?", instanceID).Order("created_at DESC").Find(&backups).Error
	return backups, err
}

// Delete removes the backup file and DB record.
func Delete(db *gorm.DB, backupID string) error {
	var b models.Backup
	if err := db.First(&b, "id = ?", backupID).Error; err != nil {
		return fmt.Errorf("backup not found: %w", err)
	}

	if err := os.Remove(b.Path); err != nil && !os.IsNotExist(err) {
		logger.Log.Warn().Err(err).Str("path", b.Path).Msg("backup: delete file failed")
	}

	if err := db.Delete(&b).Error; err != nil {
		return fmt.Errorf("delete backup record: %w", err)
	}

	logger.Log.Info().Str("backup_id", backupID).Msg("backup: deleted")
	return nil
}

package config

import (
	"encoding/json"
	"errors"

	"cs2admin/internal/models"
	"cs2admin/internal/pkg/logger"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SaveProfile creates or updates a config profile for the given instance.
// If a profile with the same instanceID and name exists, it is updated; otherwise a new one is created.
func SaveProfile(db *gorm.DB, instanceID, name string, cvars map[string]string) error {
	instanceUUID, err := uuid.Parse(instanceID)
	if err != nil {
		return errors.New("invalid instance ID")
	}

	data, err := json.Marshal(cvars)
	if err != nil {
		return err
	}

	var existing models.ConfigProfile
	result := db.Where("instance_id = ? AND name = ?", instanceUUID, name).First(&existing)
	if result.Error == nil {
		existing.Data = string(data)
		if err := db.Save(&existing).Error; err != nil {
			logger.Log.Error().Err(err).Str("profile", existing.ID.String()).Msg("failed to update config profile")
			return err
		}
		return nil
	}

	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return result.Error
	}

	profile := models.ConfigProfile{
		InstanceID: instanceUUID,
		Name:       name,
		Data:       string(data),
	}
	if err := db.Create(&profile).Error; err != nil {
		logger.Log.Error().Err(err).Str("instance", instanceID).Str("name", name).Msg("failed to create config profile")
		return err
	}
	return nil
}

// LoadProfile loads a config profile by ID and returns its cvars as a map.
func LoadProfile(db *gorm.DB, profileID string) (map[string]string, error) {
	profileUUID, err := uuid.Parse(profileID)
	if err != nil {
		return nil, errors.New("invalid profile ID")
	}

	var profile models.ConfigProfile
	if err := db.First(&profile, "id = ?", profileUUID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("profile not found")
		}
		return nil, err
	}

	var cvars map[string]string
	if profile.Data != "" {
		if err := json.Unmarshal([]byte(profile.Data), &cvars); err != nil {
			return nil, err
		}
	} else {
		cvars = make(map[string]string)
	}
	return cvars, nil
}

// ListProfiles returns all config profiles for the given instance.
func ListProfiles(db *gorm.DB, instanceID string) ([]models.ConfigProfile, error) {
	instanceUUID, err := uuid.Parse(instanceID)
	if err != nil {
		return nil, errors.New("invalid instance ID")
	}

	var profiles []models.ConfigProfile
	if err := db.Where("instance_id = ?", instanceUUID).Order("name ASC").Find(&profiles).Error; err != nil {
		return nil, err
	}
	return profiles, nil
}

// DeleteProfile deletes a config profile by ID.
func DeleteProfile(db *gorm.DB, profileID string) error {
	profileUUID, err := uuid.Parse(profileID)
	if err != nil {
		return errors.New("invalid profile ID")
	}

	result := db.Delete(&models.ConfigProfile{}, "id = ?", profileUUID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("profile not found")
	}
	return nil
}

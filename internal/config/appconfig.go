package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// AppConfig holds the application configuration.
type AppConfig struct {
	AppDataDir        string `mapstructure:"app_data_dir" json:"app_data_dir"`
	LogDir            string `mapstructure:"log_dir" json:"log_dir"`
	LogLevel          string `mapstructure:"log_level" json:"log_level"`
	SteamCMDPath      string `mapstructure:"steamcmd_path" json:"steamcmd_path"`
	DefaultInstallDir string `mapstructure:"default_install_dir" json:"default_install_dir"`
	BackupDir         string `mapstructure:"backup_dir" json:"backup_dir"`
	Theme             string `mapstructure:"theme" json:"theme"`
	MinimizeToTray    bool   `mapstructure:"minimize_to_tray" json:"minimize_to_tray"`
	StartWithWindows  bool   `mapstructure:"start_with_windows" json:"start_with_windows"`
	AutoUpdate        bool   `mapstructure:"auto_update" json:"auto_update"`
	DiscordWebhook    string `mapstructure:"discord_webhook" json:"discord_webhook"`
}

// Load loads the configuration from %APPDATA%\CS2Admin\config.yaml.
// Creates the config file with defaults if it doesn't exist.
// Creates all directories referenced in the config.
func Load() (*AppConfig, error) {
	appDataBase := os.Getenv("APPDATA")
	if appDataBase == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		appDataBase = home
	}

	defaultAppDataDir := filepath.Join(appDataBase, "CS2Admin")
	defaultLogDir := filepath.Join(defaultAppDataDir, "logs")
	defaultSteamCMDPath := filepath.Join(defaultAppDataDir, "steamcmd")
	defaultInstallDir := filepath.Join(defaultAppDataDir, "servers")
	defaultBackupDir := filepath.Join(defaultAppDataDir, "backups")

	// Build defaults config (used if file read fails or doesn't exist)
	cfg := &AppConfig{
		AppDataDir:        defaultAppDataDir,
		LogDir:            defaultLogDir,
		LogLevel:          "info",
		SteamCMDPath:      defaultSteamCMDPath,
		DefaultInstallDir: defaultInstallDir,
		BackupDir:         defaultBackupDir,
		Theme:             "system",
		MinimizeToTray:    true,
		StartWithWindows:  false,
		AutoUpdate:        true,
		DiscordWebhook:    "",
	}

	// Create app data dir
	if err := os.MkdirAll(defaultAppDataDir, 0755); err != nil {
		// If we can't create the dir, return defaults — don't crash
		return cfg, nil
	}

	configPath := filepath.Join(defaultAppDataDir, "config.yaml")

	v := viper.New()
	v.SetConfigType("yaml")
	v.SetConfigFile(configPath)

	// Set Viper defaults
	v.SetDefault("app_data_dir", defaultAppDataDir)
	v.SetDefault("log_dir", defaultLogDir)
	v.SetDefault("log_level", "info")
	v.SetDefault("steamcmd_path", defaultSteamCMDPath)
	v.SetDefault("default_install_dir", defaultInstallDir)
	v.SetDefault("backup_dir", defaultBackupDir)
	v.SetDefault("theme", "system")
	v.SetDefault("minimize_to_tray", true)
	v.SetDefault("start_with_windows", false)
	v.SetDefault("auto_update", true)
	v.SetDefault("discord_webhook", "")

	// Try to read existing config
	if err := v.ReadInConfig(); err != nil {
		// Config doesn't exist yet — write defaults
		_ = v.WriteConfigAs(configPath)
	}

	// Unmarshal into struct (ignore error, keep defaults)
	_ = v.Unmarshal(cfg)

	// Create all directories (best-effort)
	for _, d := range []string{cfg.AppDataDir, cfg.LogDir, cfg.SteamCMDPath, cfg.DefaultInstallDir, cfg.BackupDir} {
		if d != "" {
			_ = os.MkdirAll(d, 0755)
		}
	}

	return cfg, nil
}

// Save writes the current configuration back to the YAML file.
func (c *AppConfig) Save() error {
	if c.AppDataDir == "" {
		return errors.New("app_data_dir is empty")
	}

	configPath := filepath.Join(c.AppDataDir, "config.yaml")
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetConfigFile(configPath)

	v.Set("app_data_dir", c.AppDataDir)
	v.Set("log_dir", c.LogDir)
	v.Set("log_level", c.LogLevel)
	v.Set("steamcmd_path", c.SteamCMDPath)
	v.Set("default_install_dir", c.DefaultInstallDir)
	v.Set("backup_dir", c.BackupDir)
	v.Set("theme", c.Theme)
	v.Set("minimize_to_tray", c.MinimizeToTray)
	v.Set("start_with_windows", c.StartWithWindows)
	v.Set("auto_update", c.AutoUpdate)
	v.Set("discord_webhook", c.DiscordWebhook)

	return v.WriteConfig()
}

// GetDBPath returns the path to the cs2admin.db database file.
func (c *AppConfig) GetDBPath() string {
	return filepath.Join(c.AppDataDir, "cs2admin.db")
}

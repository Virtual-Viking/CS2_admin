package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"cs2admin/internal/backup"
	"cs2admin/internal/benchmark"
	"cs2admin/internal/config"
	"cs2admin/internal/filemanager"
	"cs2admin/internal/instance"
	"cs2admin/internal/models"
	"cs2admin/internal/monitor"
	"cs2admin/internal/notify"
	"cs2admin/internal/pkg/crypto"
	"cs2admin/internal/pkg/logger"
	"cs2admin/internal/pkg/valve"
	"cs2admin/internal/rcon"
	"cs2admin/internal/scheduler"
	"cs2admin/internal/steam"
	"cs2admin/internal/updater"

	"github.com/google/uuid"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm"
)

var appVersion = "0.1.0"

// App struct holds the application state and is bound to the frontend via Wails.
type App struct {
	ctx         context.Context
	cfg         *config.AppConfig
	db          *gorm.DB
	encKey      []byte
	instanceMgr *instance.Manager
	rconPool    *rcon.Pool
	steamCmd    *steam.SteamCMD
	monitors    map[string]*monitor.Collector
	sched       *scheduler.Scheduler
}

// NewApp creates a new App application struct.
func NewApp(cfg *config.AppConfig, db *gorm.DB, encKey []byte) *App {
	return &App{
		cfg:       cfg,
		db:        db,
		encKey:    encKey,
		rconPool:  rcon.NewPool(),
		steamCmd:  steam.New(cfg.SteamCMDPath),
	}
}

// startup is called when the Wails app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Clean up .old binary from a previous update
	updater.CleanupOldBinary()

	// Initialize instance manager with event callbacks
	a.instanceMgr = instance.NewManager(a.db)
	a.instanceMgr.SetDecryptFn(func(encrypted string) (string, error) {
		return crypto.Decrypt(encrypted, a.encKey)
	})
	a.instanceMgr.SetOnOutput(func(instanceID, line string) {
		wailsruntime.EventsEmit(a.ctx, "console:"+instanceID, line)
	})
	a.instanceMgr.SetOnStatus(func(instanceID, status string) {
		wailsruntime.EventsEmit(a.ctx, "status:"+instanceID, status)
	})

	// Auto-start instances that have auto_start enabled
	if err := a.instanceMgr.AutoStartAll(); err != nil {
		logger.Log.Error().Err(err).Msg("Failed to auto-start instances")
	}

	// Initialize monitors map and scheduler
	a.monitors = make(map[string]*monitor.Collector)
	a.sched = scheduler.New(a.db)
	a.sched.SetOnAction(func(instanceID string, action scheduler.TaskAction, payload string) {
		switch action {
		case scheduler.ActionRestart:
			a.RestartInstance(instanceID)
		case scheduler.ActionRCON:
			a.SendRCON(instanceID, payload)
		}
	})
	if err := a.sched.Start(); err != nil {
		logger.Log.Error().Err(err).Msg("Failed to start scheduler")
	}

	logger.Log.Info().Str("version", appVersion).Msg("CS2 Admin started")
}

// domReady is called after the frontend DOM is ready.
func (a *App) domReady(ctx context.Context) {
	logger.Log.Debug().Msg("Frontend DOM ready")
}

// beforeClose is called when the user attempts to close the window.
func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	if a.cfg.MinimizeToTray {
		wailsruntime.WindowHide(ctx)
		return true
	}
	return false
}

// shutdown is called when the app is closing.
func (a *App) shutdown(ctx context.Context) {
	logger.Log.Info().Msg("CS2 Admin shutting down")
	a.sched.Stop()
	a.instanceMgr.StopAll()
	a.rconPool.DisconnectAll()
}

// ── General Bindings ──────────────────────────────────────────────────

// CheckForUpdate checks GitHub for a newer version and returns UpdateInfo.
func (a *App) CheckForUpdate() (*updater.UpdateInfo, error) {
	return updater.CheckForUpdate(appVersion, "CS2Admin", "CS2Admin")
}

// DownloadAndApplyUpdate downloads the update from downloadURL and applies it,
// then restarts the application.
func (a *App) DownloadAndApplyUpdate(downloadURL string) error {
	tmpDir := filepath.Join(os.TempDir(), "cs2admin-update")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	destPath := filepath.Join(tmpDir, "cs2admin.exe")
	if err := updater.DownloadUpdate(downloadURL, destPath); err != nil {
		return err
	}
	if err := updater.ApplyUpdate(destPath); err != nil {
		return err
	}
	// Restart: exec the new binary (now at original exe path) and exit
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable: %w", err)
	}
	return restartSelf(exe, os.Args[1:])
}

// restartSelf executes the current binary and exits.
func restartSelf(exe string, args []string) error {
	cmd := exec.Command(exe, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("restart: %w", err)
	}
	os.Exit(0)
	return nil
}

// GetVersion returns the current application version.
func (a *App) GetVersion() string {
	return appVersion
}

// GetAppConfig returns the current application configuration.
func (a *App) GetAppConfig() *config.AppConfig {
	return a.cfg
}

// UpdateAppConfig updates and saves the application configuration.
func (a *App) UpdateAppConfig(cfg config.AppConfig) error {
	a.cfg.Theme = cfg.Theme
	a.cfg.MinimizeToTray = cfg.MinimizeToTray
	a.cfg.StartWithWindows = cfg.StartWithWindows
	a.cfg.AutoUpdate = cfg.AutoUpdate
	a.cfg.DiscordWebhook = cfg.DiscordWebhook
	a.cfg.SteamCMDPath = cfg.SteamCMDPath
	a.cfg.DefaultInstallDir = cfg.DefaultInstallDir
	a.cfg.BackupDir = cfg.BackupDir
	if err := a.cfg.Save(); err != nil {
		logger.Log.Error().Err(err).Msg("Failed to save config")
		return err
	}
	return nil
}

// GetLogLines returns recent log lines from the in-memory ring buffer.
func (a *App) GetLogLines() []string {
	return logger.LogRing.Lines()
}

// ── Instance Management ───────────────────────────────────────────────

// InstanceConfig is the input structure for creating/updating instances.
type InstanceConfig struct {
	Name         string `json:"name"`
	InstallPath  string `json:"install_path"`
	Port         int    `json:"port"`
	MaxPlayers   int    `json:"max_players"`
	GameMode     string `json:"game_mode"`
	Map          string `json:"map"`
	RconPassword string `json:"rcon_password"`
	LaunchArgs   string `json:"launch_args"`
	AutoRestart  bool   `json:"auto_restart"`
	AutoStart    bool   `json:"auto_start"`
}

// GetInstances returns all server instances.
func (a *App) GetInstances() ([]models.ServerInstance, error) {
	var instances []models.ServerInstance
	if err := a.db.Order("created_at asc").Find(&instances).Error; err != nil {
		return nil, err
	}
	return instances, nil
}

// GetInstance returns a single server instance by ID.
func (a *App) GetInstance(id string) (*models.ServerInstance, error) {
	var inst models.ServerInstance
	if err := a.db.Where("id = ?", id).First(&inst).Error; err != nil {
		return nil, err
	}
	return &inst, nil
}

// CreateInstance creates a new server instance.
func (a *App) CreateInstance(cfg InstanceConfig) (*models.ServerInstance, error) {
	// Auto-generate RCON password if not provided — must match what buildLaunchArgs uses
	rconPass := cfg.RconPassword
	if rconPass == "" {
		rconPass = "cs2admin"
	}
	encPass, err := crypto.Encrypt(rconPass, a.encKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt rcon password: %w", err)
	}

	inst := models.ServerInstance{
		Name:         cfg.Name,
		InstallPath:  cfg.InstallPath,
		Port:         cfg.Port,
		RconPort:     cfg.Port, // same as game port for RCON
		MaxPlayers:   cfg.MaxPlayers,
		GameMode:     cfg.GameMode,
		CurrentMap:   cfg.Map,
		RconPassword: encPass,
		LaunchArgs:   cfg.LaunchArgs,
		AutoRestart:  cfg.AutoRestart,
		AutoStart:    cfg.AutoStart,
		Status:       "stopped",
	}

	if inst.Port == 0 {
		inst.Port = a.nextAvailablePort()
		inst.RconPort = inst.Port
	}
	if inst.MaxPlayers == 0 {
		inst.MaxPlayers = 10
	}
	if inst.GameMode == "" {
		inst.GameMode = "competitive"
	}
	if inst.CurrentMap == "" {
		inst.CurrentMap = "de_dust2"
	}

	if err := a.db.Create(&inst).Error; err != nil {
		return nil, err
	}
	logger.Log.Info().Str("id", inst.ID.String()).Str("name", inst.Name).Msg("Instance created")
	return &inst, nil
}

// UpdateInstance updates an existing server instance.
func (a *App) UpdateInstance(id string, cfg InstanceConfig) error {
	updates := map[string]interface{}{
		"name":         cfg.Name,
		"port":         cfg.Port,
		"rcon_port":    cfg.Port,
		"max_players":  cfg.MaxPlayers,
		"game_mode":    cfg.GameMode,
		"current_map":  cfg.Map,
		"launch_args":  cfg.LaunchArgs,
		"auto_restart": cfg.AutoRestart,
		"auto_start":   cfg.AutoStart,
	}
	if cfg.InstallPath != "" {
		updates["install_path"] = cfg.InstallPath
	}
	if cfg.RconPassword != "" {
		encPass, err := crypto.Encrypt(cfg.RconPassword, a.encKey)
		if err != nil {
			return err
		}
		updates["rcon_password"] = encPass
	}
	return a.db.Model(&models.ServerInstance{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteInstance deletes a server instance (must be stopped first).
func (a *App) DeleteInstance(id string) error {
	status := a.instanceMgr.GetStatus(id)
	if status == "running" || status == "starting" {
		return fmt.Errorf("cannot delete a running instance; stop it first")
	}
	return a.db.Where("id = ?", id).Delete(&models.ServerInstance{}).Error
}

// BrowseDirectory opens a native directory picker dialog and returns the selected path.
func (a *App) BrowseDirectory() (string, error) {
	return wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select Directory",
	})
}

// StartInstance starts a server instance and auto-starts monitoring.
func (a *App) StartInstance(id string) error {
	// Ensure the instance has an encrypted RCON password (fix for legacy empty passwords)
	var inst models.ServerInstance
	if err := a.db.Where("id = ?", id).First(&inst).Error; err == nil {
		if inst.RconPassword == "" {
			if encPass, err := crypto.Encrypt("cs2admin", a.encKey); err == nil {
				a.db.Model(&models.ServerInstance{}).Where("id = ?", id).Update("rcon_password", encPass)
			}
		}
	}

	if err := a.instanceMgr.Start(id); err != nil {
		return err
	}
	// Auto-start metrics collector
	go func() {
		// Brief delay to allow the server to start listening
		time.Sleep(3 * time.Second)
		if err := a.StartMetrics(id); err != nil {
			logger.Log.Warn().Err(err).Str("instance", id).Msg("auto-start metrics failed (server may not have RCON ready yet)")
		}
	}()
	return nil
}

// StopInstance stops a server instance.
func (a *App) StopInstance(id string) error {
	if err := a.instanceMgr.Stop(id); err != nil {
		return err
	}
	a.rconPool.Disconnect(id)
	return nil
}

// RestartInstance restarts a server instance.
func (a *App) RestartInstance(id string) error {
	a.rconPool.Disconnect(id)
	return a.instanceMgr.Restart(id)
}

// ── RCON ──────────────────────────────────────────────────────────────

// SendRCON sends an RCON command to an instance and returns the response.
func (a *App) SendRCON(instanceID string, command string) (string, error) {
	// Ensure RCON connection exists
	if _, ok := a.rconPool.Get(instanceID); !ok {
		inst, err := a.GetInstance(instanceID)
		if err != nil {
			return "", fmt.Errorf("instance not found: %w", err)
		}
		password := a.getRconPassword(inst)
		addr := fmt.Sprintf("127.0.0.1:%d", inst.Port)
		if err := a.rconPool.Connect(instanceID, addr, password); err != nil {
			return "", fmt.Errorf("rcon connect: %w", err)
		}
	}
	return a.rconPool.Execute(instanceID, command)
}

// getRconPassword decrypts the RCON password from the instance, falling back to the default.
func (a *App) getRconPassword(inst *models.ServerInstance) string {
	if inst.RconPassword == "" {
		return "cs2admin"
	}
	password, err := crypto.Decrypt(inst.RconPassword, a.encKey)
	if err != nil {
		logger.Log.Warn().Err(err).Msg("Failed to decrypt RCON password, using default")
		return "cs2admin"
	}
	return password
}

// GetCommandHistory returns the RCON command history for an instance.
func (a *App) GetCommandHistory(instanceID string) ([]string, error) {
	var macros []models.CommandMacro
	if err := a.db.Order("created_at desc").Limit(100).Find(&macros).Error; err != nil {
		return nil, err
	}
	var cmds []string
	for _, m := range macros {
		cmds = append(cmds, m.Name)
	}
	return cmds, nil
}

// ── Configuration ─────────────────────────────────────────────────────

// GetServerConfig reads the server.cfg for an instance.
func (a *App) GetServerConfig(instanceID string) (map[string]string, error) {
	inst, err := a.GetInstance(instanceID)
	if err != nil {
		return nil, err
	}
	cfgPath := filepath.Join(inst.InstallPath, "game", "csgo", "cfg", "server.cfg")
	return config.ReadCfgFile(cfgPath)
}

// UpdateServerConfig writes the server.cfg for an instance.
func (a *App) UpdateServerConfig(instanceID string, cvars map[string]string) error {
	inst, err := a.GetInstance(instanceID)
	if err != nil {
		return err
	}
	cfgPath := filepath.Join(inst.InstallPath, "game", "csgo", "cfg", "server.cfg")
	return config.WriteCfgFile(cfgPath, cvars)
}

// GetCvarDatabase returns all known CS2 cvars.
func (a *App) GetCvarDatabase() []config.CvarDef {
	return config.CvarDatabase
}

// SearchCvars searches the cvar database.
func (a *App) SearchCvars(query string) []config.CvarDef {
	return config.SearchCvars(query)
}

// GetGameModePresets returns all available game mode presets.
func (a *App) GetGameModePresets() []config.GameModePreset {
	return config.Presets
}

// GetLANOptimizedCvars returns LAN optimization cvars.
func (a *App) GetLANOptimizedCvars() map[string]string {
	return config.LANOptimizedCvars
}

// ApplyGameModePreset applies a game mode preset to an instance.
func (a *App) ApplyGameModePreset(instanceID string, presetName string) error {
	preset := config.GetPresetByName(presetName)
	if preset == nil {
		return fmt.Errorf("preset not found: %s", presetName)
	}
	return a.UpdateServerConfig(instanceID, preset.Cvars)
}

// GetConfigProfiles lists config profiles for an instance.
func (a *App) GetConfigProfiles(instanceID string) ([]models.ConfigProfile, error) {
	return config.ListProfiles(a.db, instanceID)
}

// SaveConfigProfile saves a config profile.
func (a *App) SaveConfigProfile(instanceID string, name string, cvars map[string]string) error {
	return config.SaveProfile(a.db, instanceID, name, cvars)
}

// LoadConfigProfile loads a config profile.
func (a *App) LoadConfigProfile(profileID string) (map[string]string, error) {
	return config.LoadProfile(a.db, profileID)
}

// ── Maps ──────────────────────────────────────────────────────────────

// MapInfo represents an installed map.
type MapInfo struct {
	Name     string `json:"name"`
	FileName string `json:"file_name"`
	SizeBytes int64 `json:"size_bytes"`
}

// GetInstalledMaps returns maps installed for an instance.
func (a *App) GetInstalledMaps(instanceID string) ([]MapInfo, error) {
	inst, err := a.GetInstance(instanceID)
	if err != nil {
		return nil, err
	}
	mapsDir := filepath.Join(inst.InstallPath, "game", "csgo", "maps")
	return listMapsInDir(mapsDir)
}

// GetMapRotation returns the mapcycle for an instance.
func (a *App) GetMapRotation(instanceID string) ([]string, error) {
	inst, err := a.GetInstance(instanceID)
	if err != nil {
		return nil, err
	}
	mcPath := filepath.Join(inst.InstallPath, "game", "csgo", "mapcycle.txt")
	return config.ReadMapcycle(mcPath)
}

// SetMapRotation updates the mapcycle for an instance.
func (a *App) SetMapRotation(instanceID string, maps []string) error {
	inst, err := a.GetInstance(instanceID)
	if err != nil {
		return err
	}
	mcPath := filepath.Join(inst.InstallPath, "game", "csgo", "mapcycle.txt")
	return config.WriteMapcycle(mcPath, maps)
}

// ChangeMap changes the current map on a running instance.
func (a *App) ChangeMap(instanceID string, mapName string) error {
	_, err := a.SendRCON(instanceID, "changelevel "+mapName)
	return err
}

// DownloadWorkshopMap downloads a workshop map for an instance.
func (a *App) DownloadWorkshopMap(instanceID string, workshopID int64) error {
	inst, err := a.GetInstance(instanceID)
	if err != nil {
		return err
	}
	progressCh := make(chan steam.Progress, 100)
	go func() {
		for p := range progressCh {
			wailsruntime.EventsEmit(a.ctx, "progress:"+instanceID, p)
		}
	}()
	return a.steamCmd.DownloadWorkshopItem(inst.InstallPath, workshopID, progressCh)
}

// ── Players ───────────────────────────────────────────────────────────

// Player represents a connected player.
type Player struct {
	Name    string `json:"name"`
	SteamID string `json:"steam_id"`
	Ping    int    `json:"ping"`
	Score   int    `json:"score"`
	Team    string `json:"team"`
	IP      string `json:"ip"`
}

// GetPlayers returns the current player list from a running instance.
func (a *App) GetPlayers(instanceID string) ([]Player, error) {
	resp, err := a.SendRCON(instanceID, "status")
	if err != nil {
		return nil, err
	}
	return parseStatusPlayers(resp), nil
}

// KickPlayer kicks a player from an instance.
func (a *App) KickPlayer(instanceID string, steamID string, reason string) error {
	cmd := fmt.Sprintf("kickid %s %s", steamID, reason)
	_, err := a.SendRCON(instanceID, cmd)
	return err
}

// BanPlayer bans a player on an instance.
func (a *App) BanPlayer(instanceID string, steamID string, duration int, reason string) error {
	cmd := fmt.Sprintf("banid %d %s", duration, steamID)
	if _, err := a.SendRCON(instanceID, cmd); err != nil {
		return err
	}
	// Also save to DB
	instUUID, _ := uuid.Parse(instanceID)
	ban := models.BanEntry{
		InstanceID:  instUUID,
		SteamID:     steamID,
		Reason:      reason,
		IsPermanent: duration == 0,
	}
	return a.db.Create(&ban).Error
}

// MutePlayer mutes a player on an instance.
func (a *App) MutePlayer(instanceID string, steamID string) error {
	_, err := a.SendRCON(instanceID, "sm_mute #"+steamID)
	return err
}

// GetBanList returns the ban list for an instance.
func (a *App) GetBanList(instanceID string) ([]models.BanEntry, error) {
	var bans []models.BanEntry
	if err := a.db.Where("instance_id = ?", instanceID).Order("created_at desc").Find(&bans).Error; err != nil {
		return nil, err
	}
	return bans, nil
}

// RemoveBan removes a ban entry.
func (a *App) RemoveBan(banID string) error {
	return a.db.Where("id = ?", banID).Delete(&models.BanEntry{}).Error
}

// ── Bots ──────────────────────────────────────────────────────────────

// BotConfig represents bot configuration.
type BotConfig struct {
	Quota      int    `json:"quota"`
	QuotaMode  string `json:"quota_mode"` // "fill", "match", "normal"
	Difficulty int    `json:"difficulty"`  // 0-3
}

// GetBotConfig returns the current bot config from RCON.
func (a *App) GetBotConfig(instanceID string) (*BotConfig, error) {
	// Query individual cvars
	quota, _ := a.SendRCON(instanceID, "bot_quota")
	mode, _ := a.SendRCON(instanceID, "bot_quota_mode")
	diff, _ := a.SendRCON(instanceID, "bot_difficulty")
	return &BotConfig{
		Quota:      parseIntFromCvarResponse(quota),
		QuotaMode:  parseStringFromCvarResponse(mode),
		Difficulty: parseIntFromCvarResponse(diff),
	}, nil
}

// UpdateBotConfig updates bot configuration on a running instance.
func (a *App) UpdateBotConfig(instanceID string, cfg BotConfig) error {
	cmds := []string{
		fmt.Sprintf("bot_quota %d", cfg.Quota),
		fmt.Sprintf("bot_quota_mode %s", cfg.QuotaMode),
		fmt.Sprintf("bot_difficulty %d", cfg.Difficulty),
	}
	for _, cmd := range cmds {
		if _, err := a.SendRCON(instanceID, cmd); err != nil {
			return err
		}
	}
	return nil
}

// ── SteamCMD ──────────────────────────────────────────────────────────

// InstallCS2Server installs CS2 server files for an instance.
func (a *App) InstallCS2Server(instanceID string) error {
	inst, err := a.GetInstance(instanceID)
	if err != nil {
		return err
	}

	// Ensure SteamCMD is installed
	wailsruntime.EventsEmit(a.ctx, "install-line:"+instanceID, "Checking SteamCMD installation...")
	if err := a.steamCmd.EnsureInstalled(); err != nil {
		return fmt.Errorf("steamcmd setup: %w", err)
	}
	wailsruntime.EventsEmit(a.ctx, "install-line:"+instanceID, "SteamCMD ready.")

	// Update status
	a.db.Model(&models.ServerInstance{}).Where("id = ?", instanceID).Update("status", "installing")
	wailsruntime.EventsEmit(a.ctx, "status:"+instanceID, "installing")

	progressCh := make(chan steam.Progress, 100)
	go func() {
		for p := range progressCh {
			wailsruntime.EventsEmit(a.ctx, "progress:"+instanceID, p)
		}
	}()

	// Line callback sends raw SteamCMD output to frontend mini-terminal
	lineFn := func(line string) {
		wailsruntime.EventsEmit(a.ctx, "install-line:"+instanceID, line)
	}

	wailsruntime.EventsEmit(a.ctx, "install-line:"+instanceID, fmt.Sprintf("Installing CS2 to %s ...", inst.InstallPath))

	if err := a.steamCmd.InstallCS2(inst.InstallPath, progressCh, lineFn); err != nil {
		a.db.Model(&models.ServerInstance{}).Where("id = ?", instanceID).Update("status", "stopped")
		wailsruntime.EventsEmit(a.ctx, "status:"+instanceID, "stopped")
		wailsruntime.EventsEmit(a.ctx, "install-line:"+instanceID, "ERROR: "+err.Error())
		return err
	}

	a.db.Model(&models.ServerInstance{}).Where("id = ?", instanceID).Update("status", "stopped")
	wailsruntime.EventsEmit(a.ctx, "status:"+instanceID, "stopped")
	wailsruntime.EventsEmit(a.ctx, "install-line:"+instanceID, "Installation complete!")
	logger.Log.Info().Str("id", instanceID).Msg("CS2 server installed")
	return nil
}

// UpdateCS2Server updates CS2 server files for an instance.
func (a *App) UpdateCS2Server(instanceID string) error {
	inst, err := a.GetInstance(instanceID)
	if err != nil {
		return err
	}
	a.db.Model(&models.ServerInstance{}).Where("id = ?", instanceID).Update("status", "updating")
	wailsruntime.EventsEmit(a.ctx, "status:"+instanceID, "updating")

	progressCh := make(chan steam.Progress, 100)
	go func() {
		for p := range progressCh {
			wailsruntime.EventsEmit(a.ctx, "progress:"+instanceID, p)
		}
	}()

	if err := a.steamCmd.UpdateCS2(inst.InstallPath, progressCh); err != nil {
		a.db.Model(&models.ServerInstance{}).Where("id = ?", instanceID).Update("status", "stopped")
		return err
	}

	a.db.Model(&models.ServerInstance{}).Where("id = ?", instanceID).Update("status", "stopped")
	wailsruntime.EventsEmit(a.ctx, "status:"+instanceID, "stopped")
	return nil
}

// ── Skins ─────────────────────────────────────────────────────────────

// GetSkins returns skins filtered by rarity, weaponType, and search (LIKE on name).
func (a *App) GetSkins(rarity string, weaponType string, search string) ([]models.Skin, error) {
	var skins []models.Skin
	q := a.db.Model(&models.Skin{})
	if rarity != "" {
		q = q.Where("rarity = ?", rarity)
	}
	if weaponType != "" {
		q = q.Where("weapon_type = ?", weaponType)
	}
	if search != "" {
		q = q.Where("name LIKE ?", "%"+strings.TrimSpace(search)+"%")
	}
	if err := q.Order("rarity, name").Find(&skins).Error; err != nil {
		logger.Log.Error().Err(err).Msg("GetSkins: query failed")
		return nil, err
	}
	return skins, nil
}

// UpdateSkinDatabase fetches and updates the skin database from Valve items_game.
func (a *App) UpdateSkinDatabase() error {
	if err := valve.UpdateSkinDatabase(a.db); err != nil {
		logger.Log.Error().Err(err).Msg("UpdateSkinDatabase failed")
		return err
	}
	// Record last update time
	now := time.Now().UTC().Format(time.RFC3339)
	var s models.AppSetting
	if a.db.Where("key = ?", "skin_db_updated_at").First(&s).Error != nil {
		a.db.Create(&models.AppSetting{Key: "skin_db_updated_at", Value: now})
	} else {
		a.db.Model(&s).Update("value", now)
	}
	return nil
}

// GetSkinDatabaseLastUpdated returns when the skin database was last updated, or empty string.
func (a *App) GetSkinDatabaseLastUpdated() string {
	var s models.AppSetting
	if err := a.db.Where("key = ?", "skin_db_updated_at").First(&s).Error; err != nil {
		return ""
	}
	return s.Value
}

// TestDiscordWebhook sends a test message to the configured Discord webhook URL.
func (a *App) TestDiscordWebhook() error {
	n := notify.New()
	n.SetDiscordURL(a.cfg.DiscordWebhook)
	return n.SendDiscord("CS2 Admin Test", "This is a test notification from CS2 Admin.", 0x00FF00)
}

// ExportSkinDatabaseJSON exports the skin database as a JSON file to the given instance's
// plugin directory so the CS2AdminSkins plugin can load it.
func (a *App) ExportSkinDatabaseJSON(instanceID string) error {
	inst, err := a.GetInstance(instanceID)
	if err != nil {
		return err
	}

	var skins []models.Skin
	if err := a.db.Order("rarity, name").Find(&skins).Error; err != nil {
		return fmt.Errorf("query skins: %w", err)
	}

	type SkinEntry struct {
		PaintKitID int     `json:"PaintKitId"`
		Name       string  `json:"Name"`
		WeaponType string  `json:"WeaponType"`
		Rarity     string  `json:"Rarity"`
		Category   string  `json:"Category"`
		MinFloat   float64 `json:"MinFloat"`
		MaxFloat   float64 `json:"MaxFloat"`
	}

	rarityToCategory := map[string]string{
		"Mil-Spec Grade": "blue",
		"Restricted":     "purple",
		"Classified":     "pink",
		"Covert":         "red",
		"Extraordinary":  "gold",
		"Contraband":     "gold",
	}

	var entries []SkinEntry
	for _, s := range skins {
		cat := rarityToCategory[s.Rarity]
		if cat == "" {
			cat = "blue"
		}
		entries = append(entries, SkinEntry{
			PaintKitID: s.PaintKitID,
			Name:       s.Name,
			WeaponType: s.WeaponType,
			Rarity:     s.Rarity,
			Category:   cat,
			MinFloat:   s.MinFloat,
			MaxFloat:   s.MaxFloat,
		})
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal skins: %w", err)
	}

	pluginDir := filepath.Join(inst.InstallPath, "game", "csgo", "addons", "counterstrikesharp", "plugins", "CS2AdminSkins")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return fmt.Errorf("create plugin dir: %w", err)
	}

	jsonPath := filepath.Join(pluginDir, "skins.json")
	if err := os.WriteFile(jsonPath, data, 0644); err != nil {
		return fmt.Errorf("write skins.json: %w", err)
	}

	logger.Log.Info().Str("path", jsonPath).Int("count", len(entries)).Msg("Exported skin database to plugin")
	return nil
}

// GetSkinRarities returns distinct rarity names from the skins table.
func (a *App) GetSkinRarities() []string {
	var rarities []string
	a.db.Model(&models.Skin{}).Distinct("rarity").Where("rarity != ''").Pluck("rarity", &rarities)
	return rarities
}

// ── Plugins ───────────────────────────────────────────────────────────

// GetPlugins returns installed plugins for an instance.
func (a *App) GetPlugins(instanceID string) ([]instance.PluginInfo, error) {
	inst, err := a.GetInstance(instanceID)
	if err != nil {
		return nil, err
	}
	plugins, err := instance.GetInstalledPlugins(inst.InstallPath)
	if err != nil {
		logger.Log.Error().Err(err).Str("instance", instanceID).Msg("GetPlugins failed")
		return nil, err
	}
	return plugins, nil
}

// InstallPlugin installs Metamod, CounterStrikeSharp, or WeaponPaints by name.
func (a *App) InstallPlugin(instanceID string, pluginName string) error {
	inst, err := a.GetInstance(instanceID)
	if err != nil {
		return err
	}
	pluginName = strings.ToLower(strings.TrimSpace(pluginName))
	switch pluginName {
	case "metamod", "metamod:source":
		if err := instance.InstallMetamod(inst.InstallPath); err != nil {
			logger.Log.Error().Err(err).Str("plugin", "Metamod").Msg("InstallPlugin failed")
			return err
		}
	case "counterstrikesharp", "cssharp", "css":
		if err := instance.InstallCounterStrikeSharp(inst.InstallPath); err != nil {
			logger.Log.Error().Err(err).Str("plugin", "CounterStrikeSharp").Msg("InstallPlugin failed")
			return err
		}
	case "weaponpaints":
		if err := instance.InstallWeaponPaints(inst.InstallPath); err != nil {
			logger.Log.Error().Err(err).Str("plugin", "WeaponPaints").Msg("InstallPlugin failed")
			return err
		}
	default:
		return fmt.Errorf("unknown plugin: %s (use: Metamod, CounterStrikeSharp, WeaponPaints)", pluginName)
	}
	logger.Log.Info().Str("instance", instanceID).Str("plugin", pluginName).Msg("plugin installed")
	return nil
}

// ── Match Stats ────────────────────────────────────────────────────────

// GetMatches returns matches for an instance.
func (a *App) GetMatches(instanceID string) ([]models.Match, error) {
	var matches []models.Match
	if err := a.db.Where("instance_id = ?", instanceID).Order("started_at DESC").Find(&matches).Error; err != nil {
		logger.Log.Error().Err(err).Str("instance", instanceID).Msg("GetMatches failed")
		return nil, err
	}
	return matches, nil
}

// GetMatchDetail returns a single match by ID.
func (a *App) GetMatchDetail(matchID string) (*models.Match, error) {
	var m models.Match
	if err := a.db.Where("id = ?", matchID).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// GetMatchPlayers returns players for a match.
func (a *App) GetMatchPlayers(matchID string) ([]models.MatchPlayer, error) {
	var players []models.MatchPlayer
	if err := a.db.Where("match_id = ?", matchID).Find(&players).Error; err != nil {
		logger.Log.Error().Err(err).Str("match", matchID).Msg("GetMatchPlayers failed")
		return nil, err
	}
	return players, nil
}

// GetMatchDamage returns damage events for a match.
func (a *App) GetMatchDamage(matchID string) ([]models.MatchDamage, error) {
	var damage []models.MatchDamage
	if err := a.db.Where("match_id = ?", matchID).Order("round_number, id").Find(&damage).Error; err != nil {
		logger.Log.Error().Err(err).Str("match", matchID).Msg("GetMatchDamage failed")
		return nil, err
	}
	return damage, nil
}

// ── Monitoring ────────────────────────────────────────────────────────

// GetMetricsHistory returns metric snapshots for an instance from the last N minutes.
func (a *App) GetMetricsHistory(instanceID string, minutes int) ([]models.MetricSnapshot, error) {
	instUUID, err := uuid.Parse(instanceID)
	if err != nil {
		return nil, fmt.Errorf("invalid instance id: %w", err)
	}
	since := time.Now().Add(-time.Duration(minutes) * time.Minute)
	var snapshots []models.MetricSnapshot
	if err := a.db.Where("instance_id = ? AND timestamp >= ?", instUUID, since).
		Order("timestamp ASC").Find(&snapshots).Error; err != nil {
		logger.Log.Error().Err(err).Str("instance", instanceID).Msg("GetMetricsHistory failed")
		return nil, err
	}
	return snapshots, nil
}

// StartMetrics creates and starts a metrics collector for the instance.
func (a *App) StartMetrics(instanceID string) error {
	inst, err := a.GetInstance(instanceID)
	if err != nil {
		return err
	}
	password := a.getRconPassword(inst)
	addr := fmt.Sprintf("127.0.0.1:%d", inst.Port)
	c := monitor.NewCollector(instanceID, addr, password, a.db)
	c.SetRconPool(a.rconPool)
	c.SetOnMetrics(func(id string, m monitor.Metrics) {
		wailsruntime.EventsEmit(a.ctx, "metrics:"+id, m)
	})
	c.Start()
	a.monitors[instanceID] = c
	logger.Log.Info().Str("instance", instanceID).Msg("metrics collector started")
	return nil
}

// StopMetrics stops the metrics collector for the instance.
func (a *App) StopMetrics(instanceID string) {
	if c, ok := a.monitors[instanceID]; ok {
		c.Stop()
		delete(a.monitors, instanceID)
		logger.Log.Info().Str("instance", instanceID).Msg("metrics collector stopped")
	}
}

// ── Benchmarks ────────────────────────────────────────────────────────

// RunBenchmark runs a benchmark in a goroutine, emitting progress via Wails events.
func (a *App) RunBenchmark(instanceID string, maxBots int, stepSize int, stepDurationSec int) error {
	if _, err := a.GetInstance(instanceID); err != nil {
		return err
	}
	if maxBots <= 0 || stepSize <= 0 {
		return fmt.Errorf("invalid benchmark params: maxBots=%d stepSize=%d", maxBots, stepSize)
	}
	dur := time.Duration(stepDurationSec) * time.Second
	if dur < time.Second {
		dur = 5 * time.Second
	}
	cfg := benchmark.BenchmarkConfig{
		InstanceID:   instanceID,
		MaxBots:      maxBots,
		StepSize:     stepSize,
		StepDuration: dur,
	}
	runner := benchmark.NewRunner(cfg, a.db)
	runner.SetRconPool(a.rconPool)
	runner.SetOnProgress(func(step, totalSteps int, m benchmark.Metrics) {
		wailsruntime.EventsEmit(a.ctx, "benchmark:"+instanceID, map[string]interface{}{
			"step":       step,
			"total":      totalSteps,
			"bot_count":  m.BotCount,
			"tick_rate":  m.AvgTickRate,
			"cpu_usage":  m.CPUUsage,
			"ram_usage":  m.RAMUsage,
		})
	})
	go func() {
		if _, err := runner.Run(); err != nil {
			logger.Log.Error().Err(err).Str("instance", instanceID).Msg("benchmark failed")
			wailsruntime.EventsEmit(a.ctx, "benchmark:"+instanceID+":error", err.Error())
		} else {
			wailsruntime.EventsEmit(a.ctx, "benchmark:"+instanceID+":complete", nil)
		}
	}()
	return nil
}

// GetBenchmarkResults returns benchmark results for an instance.
func (a *App) GetBenchmarkResults(instanceID string) ([]models.BenchmarkResult, error) {
	return benchmark.ListResults(a.db, instanceID)
}

// ── Scheduler ──────────────────────────────────────────────────────────

// GetScheduledTasks returns scheduled tasks for an instance.
func (a *App) GetScheduledTasks(instanceID string) ([]models.ScheduledTask, error) {
	return a.sched.ListTasks(instanceID)
}

// CreateScheduledTask adds a new scheduled task.
func (a *App) CreateScheduledTask(task models.ScheduledTask) error {
	if err := a.sched.AddTask(task); err != nil {
		logger.Log.Error().Err(err).Msg("CreateScheduledTask failed")
		return err
	}
	return nil
}

// DeleteScheduledTask removes a scheduled task by ID.
func (a *App) DeleteScheduledTask(taskID string) error {
	if err := a.sched.RemoveTask(taskID); err != nil {
		logger.Log.Error().Err(err).Str("task", taskID).Msg("DeleteScheduledTask failed")
		return err
	}
	return nil
}

// ── Backups ────────────────────────────────────────────────────────────

// CreateBackup creates a backup for an instance.
func (a *App) CreateBackup(instanceID string, backupType string) (*models.Backup, error) {
	inst, err := a.GetInstance(instanceID)
	if err != nil {
		return nil, err
	}
	backupDir := a.cfg.BackupDir
	if backupDir == "" && a.cfg.AppDataDir != "" {
		backupDir = filepath.Join(a.cfg.AppDataDir, "backups")
	}
	bType := backup.BackupType(backupType)
	if bType != backup.BackupFull && bType != backup.BackupConfigOnly && bType != backup.BackupMapsOnly && bType != backup.BackupPluginsOnly {
		bType = backup.BackupFull
	}
	b, err := backup.Create(a.db, instanceID, inst.InstallPath, backupDir, bType)
	if err != nil {
		logger.Log.Error().Err(err).Str("instance", instanceID).Msg("CreateBackup failed")
		return nil, err
	}
	return b, nil
}

// RestoreBackup restores a backup by ID.
func (a *App) RestoreBackup(backupID string) error {
	var b models.Backup
	if err := a.db.First(&b, "id = ?", backupID).Error; err != nil {
		return fmt.Errorf("backup not found: %w", err)
	}
	inst, err := a.GetInstance(b.InstanceID.String())
	if err != nil {
		return fmt.Errorf("instance not found: %w", err)
	}
	if err := backup.Restore(a.db, backupID, inst.InstallPath); err != nil {
		logger.Log.Error().Err(err).Str("backup", backupID).Msg("RestoreBackup failed")
		return err
	}
	return nil
}

// GetBackups returns backups for an instance.
func (a *App) GetBackups(instanceID string) ([]models.Backup, error) {
	return backup.List(a.db, instanceID)
}

// DeleteBackup deletes a backup by ID.
func (a *App) DeleteBackup(backupID string) error {
	if err := backup.Delete(a.db, backupID); err != nil {
		logger.Log.Error().Err(err).Str("backup", backupID).Msg("DeleteBackup failed")
		return err
	}
	return nil
}

// ── File Manager ───────────────────────────────────────────────────────

// ListFiles lists files in the instance's install directory at the given relative path.
func (a *App) ListFiles(instanceID string, relativePath string) ([]filemanager.FileEntry, error) {
	inst, err := a.GetInstance(instanceID)
	if err != nil {
		return nil, err
	}
	rootPath := filepath.Join(inst.InstallPath, "game", "csgo")
	entries, err := filemanager.ListDirectory(rootPath, relativePath)
	if err != nil {
		logger.Log.Error().Err(err).Str("instance", instanceID).Str("path", relativePath).Msg("ListFiles failed")
		return nil, err
	}
	return entries, nil
}

// ReadServerFile reads a file from the instance's CS2 directory.
func (a *App) ReadServerFile(instanceID string, relativePath string) (string, error) {
	inst, err := a.GetInstance(instanceID)
	if err != nil {
		return "", err
	}
	rootPath := filepath.Join(inst.InstallPath, "game", "csgo")
	content, err := filemanager.ReadFile(rootPath, relativePath)
	if err != nil {
		logger.Log.Error().Err(err).Str("instance", instanceID).Str("path", relativePath).Msg("ReadServerFile failed")
		return "", err
	}
	return content, nil
}

// WriteServerFile writes content to a file in the instance's CS2 directory.
func (a *App) WriteServerFile(instanceID string, relativePath string, content string) error {
	inst, err := a.GetInstance(instanceID)
	if err != nil {
		return err
	}
	rootPath := filepath.Join(inst.InstallPath, "game", "csgo")
	if err := filemanager.WriteFile(rootPath, relativePath, content); err != nil {
		logger.Log.Error().Err(err).Str("instance", instanceID).Str("path", relativePath).Msg("WriteServerFile failed")
		return err
	}
	return nil
}

// ── Notifications (Audit) ──────────────────────────────────────────────

// GetAuditLog returns recent audit log entries.
func (a *App) GetAuditLog(limit int) ([]models.AuditLog, error) {
	if limit <= 0 {
		limit = 100
	}
	var entries []models.AuditLog
	if err := a.db.Order("created_at DESC").Limit(limit).Find(&entries).Error; err != nil {
		logger.Log.Error().Err(err).Msg("GetAuditLog failed")
		return nil, err
	}
	return entries, nil
}

// LogAudit creates an audit log entry.
func (a *App) LogAudit(action string, target string, details string) {
	entry := models.AuditLog{
		Action:  action,
		Target:  target,
		Details: details,
	}
	if err := a.db.Create(&entry).Error; err != nil {
		logger.Log.Error().Err(err).Msg("LogAudit failed")
	}
}

// ── Internal Helpers ──────────────────────────────────────────────────

func (a *App) nextAvailablePort() int {
	var inst models.ServerInstance
	if err := a.db.Order("port desc").First(&inst).Error; err != nil {
		return 27015 // first instance
	}
	return inst.Port + 10
}

func listMapsInDir(dir string) ([]MapInfo, error) {
	var maps []MapInfo
	entries, err := filepath.Glob(filepath.Join(dir, "*.vpk"))
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		name := filepath.Base(e)
		maps = append(maps, MapInfo{
			Name:     name[:len(name)-4], // strip .vpk
			FileName: name,
		})
	}
	return maps, nil
}

// parseStatusPlayers parses the CS2 RCON "status" output into Player structs.
// CS2 status output format:
// # userid name uniqueid connected ping loss state rate adr
// #  2 "PlayerName" STEAM_1:0:12345678 01:23 45 0 active 786432 192.168.1.2:27005
// #  3 "BotName" BOT 01:23 0 0 active 0
func parseStatusPlayers(status string) []Player {
	var players []Player
	lines := strings.Split(status, "\n")

	// Regex to match player lines:
	// # <userid> <name> <steamid> <connected> <ping> <loss> <state> <rate> [<adr>]
	playerRe := regexp.MustCompile(`#\s+(\d+)\s+"([^"]+)"\s+(\S+)\s+\S+\s+(\d+)\s+\d+\s+(\w+)\s+\d+(?:\s+(\S+))?`)

	// Also parse the "players" summary line for team info
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "#") || strings.HasPrefix(line, "# userid") {
			continue
		}

		m := playerRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}

		steamID := m[3]
		if steamID == "BOT" {
			continue // Skip bots in player list
		}

		ping := 0
		if v, err := strconv.Atoi(m[4]); err == nil {
			ping = v
		}

		ip := ""
		if len(m) >= 7 {
			ip = m[6]
			// Strip port from IP
			if idx := strings.LastIndex(ip, ":"); idx > 0 {
				ip = ip[:idx]
			}
		}

		players = append(players, Player{
			Name:    m[2],
			SteamID: steamID,
			Ping:    ping,
			Score:   0, // Score not in status output, would need separate query
			Team:    "", // Team not in status output
			IP:      ip,
		})
	}

	return players
}

// parseIntFromCvarResponse parses an RCON cvar query response.
// Typical format: "bot_quota" = "10" ( def. "10" ) min. 0.000000 max. 64.000000
// or: "bot_quota" is "10"
func parseIntFromCvarResponse(resp string) int {
	// Try: "cvar" = "value"
	re := regexp.MustCompile(`"[^"]*"\s*(?:=|is)\s*"(\d+)"`)
	m := re.FindStringSubmatch(resp)
	if m != nil {
		v, _ := strconv.Atoi(m[1])
		return v
	}
	// Fallback: just find any bare number in the response
	numRe := regexp.MustCompile(`\b(\d+)\b`)
	all := numRe.FindAllStringSubmatch(resp, -1)
	for _, a := range all {
		v, _ := strconv.Atoi(a[1])
		if v > 0 {
			return v
		}
	}
	return 0
}

// parseStringFromCvarResponse parses a string value from an RCON cvar query response.
func parseStringFromCvarResponse(resp string) string {
	// Try: "cvar" = "value"
	re := regexp.MustCompile(`"[^"]*"\s*(?:=|is)\s*"([^"]*)"`)
	m := re.FindStringSubmatch(resp)
	if m != nil {
		return m[1]
	}
	return ""
}

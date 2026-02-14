package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ServerInstance represents a CS2 server instance
type ServerInstance struct {
	ID            uuid.UUID      `gorm:"primaryKey;type:varchar(36)" json:"id"`
	Name          string         `gorm:"not null" json:"name"`
	Port          int            `gorm:"not null" json:"port"`
	RconPort      int            `gorm:"column:rcon_port;not null" json:"rcon_port"`
	Status        string         `gorm:"default:stopped" json:"status"`
	GameMode      string         `gorm:"column:game_mode" json:"game_mode"`
	MaxPlayers    int            `gorm:"column:max_players;default:10" json:"max_players"`
	CurrentMap    string         `gorm:"column:current_map" json:"current_map"`
	InstallPath   string         `gorm:"column:install_path" json:"install_path"`
	LaunchArgs    string         `gorm:"column:launch_args" json:"launch_args"`
	RconPassword  string         `gorm:"column:rcon_password" json:"-"` // encrypted
	GsltToken     string         `gorm:"column:gslt_token" json:"-"`    // encrypted
	AutoRestart   bool      `gorm:"column:auto_restart;default:true" json:"auto_restart"`
	AutoStart     bool      `gorm:"column:auto_start" json:"auto_start"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// BeforeCreate generates UUID for ServerInstance
func (s *ServerInstance) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

// ConfigProfile holds configuration profile for an instance
type ConfigProfile struct {
	ID         uuid.UUID `gorm:"primaryKey;type:varchar(36)" json:"id"`
	InstanceID uuid.UUID `gorm:"type:varchar(36);not null;index" json:"instance_id"`
	Name       string    `gorm:"not null" json:"name"`
	Data       string    `gorm:"type:text" json:"data"` // JSON blob
	IsActive   bool      `gorm:"column:is_active" json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
}

// BeforeCreate generates UUID for ConfigProfile
func (c *ConfigProfile) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

// BanEntry represents a ban on a server instance
type BanEntry struct {
	ID          uuid.UUID  `gorm:"primaryKey;type:varchar(36)" json:"id"`
	InstanceID  uuid.UUID  `gorm:"type:varchar(36);not null;index" json:"instance_id"`
	SteamID     string     `gorm:"column:steam_id;not null;index" json:"steam_id"`
	IPAddress   string     `gorm:"column:ip_address;index" json:"ip_address"`
	Reason      string     `json:"reason"`
	ExpiresAt   *time.Time `json:"expires_at"` // nullable for permanent
	IsPermanent bool       `gorm:"column:is_permanent" json:"is_permanent"`
	CreatedAt   time.Time  `json:"created_at"`
}

// BeforeCreate generates UUID for BanEntry
func (b *BanEntry) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// WorkshopItem represents a Steam Workshop item for an instance
type WorkshopItem struct {
	ID          uuid.UUID `gorm:"primaryKey;type:varchar(36)" json:"id"`
	InstanceID  uuid.UUID `gorm:"type:varchar(36);not null;index" json:"instance_id"`
	WorkshopID  int64     `gorm:"column:workshop_id;not null;index" json:"workshop_id"`
	Title       string    `json:"title"`
	ItemType    string    `gorm:"column:item_type;index" json:"item_type"`
	FileSize    int64     `gorm:"column:file_size" json:"file_size"`
	Installed   bool      `gorm:"default:false" json:"installed"`
	CreatedAt   time.Time `json:"created_at"`
}

// BeforeCreate generates UUID for WorkshopItem
func (w *WorkshopItem) BeforeCreate(tx *gorm.DB) error {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	return nil
}

// Backup represents a server backup
type Backup struct {
	ID         uuid.UUID `gorm:"primaryKey;type:varchar(36)" json:"id"`
	InstanceID uuid.UUID `gorm:"type:varchar(36);not null;index" json:"instance_id"`
	Path       string    `gorm:"not null" json:"path"`
	SizeBytes  int64     `gorm:"column:size_bytes" json:"size_bytes"`
	BackupType string    `gorm:"column:backup_type;index" json:"backup_type"`
	CreatedAt  time.Time `json:"created_at"`
}

// BeforeCreate generates UUID for Backup
func (b *Backup) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// ScheduledTask represents a cron-scheduled task
type ScheduledTask struct {
	ID         uuid.UUID  `gorm:"primaryKey;type:varchar(36)" json:"id"`
	InstanceID uuid.UUID  `gorm:"type:varchar(36);not null;index" json:"instance_id"`
	CronExpr   string     `gorm:"column:cron_expr;not null" json:"cron_expr"`
	Action     string     `gorm:"index" json:"action"`
	Payload    string     `gorm:"type:text" json:"payload"` // JSON
	Enabled    bool       `gorm:"default:true" json:"enabled"`
	LastRun    *time.Time `gorm:"column:last_run" json:"last_run"`
	NextRun    *time.Time `gorm:"column:next_run" json:"next_run"`
	CreatedAt  time.Time  `json:"created_at"`
}

// BeforeCreate generates UUID for ScheduledTask
func (s *ScheduledTask) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

// BenchmarkResult stores benchmark metrics for an instance
type BenchmarkResult struct {
	ID           uuid.UUID `gorm:"primaryKey;type:varchar(36)" json:"id"`
	InstanceID   uuid.UUID `gorm:"type:varchar(36);not null;index" json:"instance_id"`
	BotCount     int       `gorm:"column:bot_count" json:"bot_count"`
	AvgTickrate  float64   `gorm:"column:avg_tickrate" json:"avg_tickrate"`
	MinTickrate  float64   `gorm:"column:min_tickrate" json:"min_tickrate"`
	AvgFrametime float64   `gorm:"column:avg_frametime" json:"avg_frametime"`
	CPUUsage     float64   `gorm:"column:cpu_usage" json:"cpu_usage"`
	RAMUsage     float64   `gorm:"column:ram_usage" json:"ram_usage"`
	DurationSec  int       `gorm:"column:duration_sec" json:"duration_sec"`
	CreatedAt    time.Time `json:"created_at"`
}

// BeforeCreate generates UUID for BenchmarkResult
func (b *BenchmarkResult) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// MetricSnapshot stores periodic metrics for an instance
type MetricSnapshot struct {
	ID         int       `gorm:"primaryKey;autoIncrement" json:"id"`
	InstanceID uuid.UUID `gorm:"type:varchar(36);not null;index" json:"instance_id"`
	CPUPct     float64   `gorm:"column:cpu_pct" json:"cpu_pct"`
	RAMMb      float64   `gorm:"column:ram_mb" json:"ram_mb"`
	TickRate   float64   `gorm:"column:tick_rate" json:"tick_rate"`
	Players    int       `json:"players"`
	NetInKbps  float64   `gorm:"column:net_in_kbps" json:"net_in_kbps"`
	NetOutKbps float64   `gorm:"column:net_out_kbps" json:"net_out_kbps"`
	Timestamp  time.Time `gorm:"index" json:"timestamp"`
}

// AuditLog records administrative actions
type AuditLog struct {
	ID        uuid.UUID `gorm:"primaryKey;type:varchar(36)" json:"id"`
	Action    string    `gorm:"index" json:"action"`
	Target    string    `gorm:"index" json:"target"`
	Details   string    `gorm:"type:text" json:"details"` // JSON
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

// BeforeCreate generates UUID for AuditLog
func (a *AuditLog) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// CommandMacro stores a macro of RCON commands
type CommandMacro struct {
	ID        uuid.UUID `gorm:"primaryKey;type:varchar(36)" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	Commands  string    `gorm:"type:text" json:"commands"` // JSON array
	Hotkey    string    `json:"hotkey"`
	CreatedAt time.Time `json:"created_at"`
}

// BeforeCreate generates UUID for CommandMacro
func (c *CommandMacro) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

// AppSetting stores key-value application settings
type AppSetting struct {
	ID        int       `gorm:"primaryKey" json:"id"`
	Key       string    `gorm:"uniqueIndex;not null" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

// Skin represents a CS2 weapon skin
type Skin struct {
	ID           int     `gorm:"primaryKey;autoIncrement" json:"id"`
	PaintKitID   int     `gorm:"column:paint_kit_id;index" json:"paint_kit_id"`
	Name         string  `json:"name"`
	WeaponType   string  `gorm:"column:weapon_type;index" json:"weapon_type"`
	Rarity       string  `gorm:"index" json:"rarity"`
	RarityColor  string  `gorm:"column:rarity_color" json:"rarity_color"`
	MinFloat     float64 `gorm:"column:min_float" json:"min_float"`
	MaxFloat     float64 `gorm:"column:max_float" json:"max_float"`
	ImageURL     string  `gorm:"column:image_url" json:"image_url"`
	Category     string  `gorm:"index" json:"category"`
	Collection   string  `gorm:"index" json:"collection"`
}

// Match represents a completed or in-progress match
type Match struct {
	ID              uuid.UUID `gorm:"primaryKey;type:varchar(36)" json:"id"`
	InstanceID      uuid.UUID `gorm:"type:varchar(36);not null;index" json:"instance_id"`
	MapName         string    `gorm:"column:map_name;index" json:"map_name"`
	GameMode        string    `gorm:"column:game_mode;index" json:"game_mode"`
	Team1Score      int       `gorm:"column:team1_score" json:"team1_score"`
	Team2Score      int       `gorm:"column:team2_score" json:"team2_score"`
	DurationSec     int       `gorm:"column:duration_sec" json:"duration_sec"`
	RoundsPlayed    int       `gorm:"column:rounds_played" json:"rounds_played"`
	BombPlants      int       `gorm:"column:bomb_plants" json:"bomb_plants"`
	BombDefuses     int       `gorm:"column:bomb_defuses" json:"bomb_defuses"`
	BombExplosions  int       `gorm:"column:bomb_explosions" json:"bomb_explosions"`
	StartedAt       time.Time `gorm:"column:started_at;index" json:"started_at"`
	EndedAt         time.Time `gorm:"column:ended_at" json:"ended_at"`
}

// BeforeCreate generates UUID for Match
func (m *Match) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

// MatchPlayer stores per-player stats for a match
type MatchPlayer struct {
	ID             uuid.UUID `gorm:"primaryKey;type:varchar(36)" json:"id"`
	MatchID        uuid.UUID `gorm:"type:varchar(36);not null;index" json:"match_id"`
	SteamID        string    `gorm:"column:steam_id;index" json:"steam_id"`
	PlayerName     string    `gorm:"column:player_name" json:"player_name"`
	Team           string    `gorm:"index" json:"team"`
	Kills          int       `json:"kills"`
	Deaths         int       `json:"deaths"`
	Assists        int       `json:"assists"`
	Headshots      int       `json:"headshots"`
	MVPs           int       `gorm:"column:mvps" json:"mvps"`
	TotalDamage    int       `gorm:"column:total_damage" json:"total_damage"`
	UtilityDamage  int       `gorm:"column:utility_damage" json:"utility_damage"`
	EnemiesFlashed int       `gorm:"column:enemies_flashed" json:"enemies_flashed"`
	Enemy2Ks       int       `gorm:"column:enemy_2ks" json:"enemy_2ks"`
	Enemy3Ks       int       `gorm:"column:enemy_3ks" json:"enemy_3ks"`
	Enemy4Ks       int       `gorm:"column:enemy_4ks" json:"enemy_4ks"`
	Enemy5Ks       int       `gorm:"column:enemy_5ks" json:"enemy_5ks"`
	ADR            float64   `gorm:"column:adr" json:"adr"`
	HSP            float64   `gorm:"column:hsp" json:"hsp"`
	Score          int       `json:"score"`
}

// BeforeCreate generates UUID for MatchPlayer
func (m *MatchPlayer) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

// MatchDamage stores damage events within a match
type MatchDamage struct {
	ID             int       `gorm:"primaryKey;autoIncrement" json:"id"`
	MatchID        uuid.UUID `gorm:"type:varchar(36);not null;index" json:"match_id"`
	RoundNumber    int       `gorm:"column:round_number;index" json:"round_number"`
	AttackerSteam  string    `gorm:"column:attacker_steam;index" json:"attacker_steam"`
	VictimSteam    string    `gorm:"column:victim_steam;index" json:"victim_steam"`
	Damage         int       `json:"damage"`
	Hits           int       `json:"hits"`
	Headshots      int       `json:"headshots"`
	Weapon         string    `gorm:"index" json:"weapon"`
	Killed         bool      `json:"killed"`
}

// MatchRound stores round-by-round match data
type MatchRound struct {
	ID          int       `gorm:"primaryKey;autoIncrement" json:"id"`
	MatchID     uuid.UUID `gorm:"type:varchar(36);not null;index" json:"match_id"`
	RoundNumber int       `gorm:"column:round_number;index" json:"round_number"`
	Winner      string    `gorm:"index" json:"winner"`
	WinReason   string    `gorm:"column:win_reason" json:"win_reason"`
	DurationSec int       `gorm:"column:duration_sec" json:"duration_sec"`
}

// AutoMigrate runs GORM AutoMigrate on all models
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&ServerInstance{},
		&ConfigProfile{},
		&BanEntry{},
		&WorkshopItem{},
		&Backup{},
		&ScheduledTask{},
		&BenchmarkResult{},
		&MetricSnapshot{},
		&AuditLog{},
		&CommandMacro{},
		&AppSetting{},
		&Skin{},
		&Match{},
		&MatchPlayer{},
		&MatchDamage{},
		&MatchRound{},
	)
}

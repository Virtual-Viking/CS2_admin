package config

// GameModePreset defines a preset configuration for a game mode.
type GameModePreset struct {
	Name        string            `json:"name"`
	GameType    int               `json:"game_type"`
	GameMode    int               `json:"game_mode"`
	Description string            `json:"description"`
	Cvars       map[string]string `json:"cvars"`
	DefaultMap  string            `json:"default_map"`
}

// Presets holds the built-in game mode presets.
var Presets = []GameModePreset{
	{
		Name:        "Competitive",
		GameType:    0,
		GameMode:    1,
		Description: "Matchmaking-style competitive with MR12, overtime, halftime",
		DefaultMap:  "de_dust2",
		Cvars: map[string]string{
			"game_type":           "0",
			"game_mode":           "1",
			"map":                 "de_dust2",
			"mp_maxrounds":        "24",
			"mp_overtime_enable":  "1",
			"mp_freezetime":       "15",
			"mp_roundtime":        "1.92",
			"mp_buytime":          "20",
			"mp_halftime":         "1",
			"mp_match_can_clinch": "1",
			"mp_startmoney":       "800",
		},
	},
	{
		Name:        "Casual",
		GameType:    0,
		GameMode:    0,
		Description: "Casual 10v10 with armor, longer rounds, and friendly fire off",
		DefaultMap:  "de_dust2",
		Cvars: map[string]string{
			"game_type":      "0",
			"game_mode":      "0",
			"map":            "de_dust2",
			"mp_maxrounds":   "15",
			"mp_freezetime":  "6",
			"mp_roundtime":   "2.25",
			"mp_buytime":     "90",
			"mp_friendlyfire": "0",
			"mp_free_armor":  "1",
		},
	},
	{
		Name:        "Deathmatch",
		GameType:    1,
		GameMode:    2,
		Description: "Free-for-all deathmatch with respawn",
		DefaultMap:  "de_dust2",
		Cvars: map[string]string{
			"game_type":      "1",
			"game_mode":      "2",
			"map":            "de_dust2",
			"mp_roundtime":   "10",
			"mp_warmup_time": "0",
			"mp_free_armor":  "1",
		},
	},
	{
		Name:        "Wingman",
		GameType:    0,
		GameMode:    2,
		Description: "2v2 competitive wingman mode",
		DefaultMap:  "de_inferno",
		Cvars: map[string]string{
			"game_type":     "0",
			"game_mode":     "2",
			"map":           "de_inferno",
			"mp_maxrounds":  "16",
		},
	},
	{
		Name:        "Retake",
		GameType:    0,
		GameMode:    0,
		Description: "Retake practice - attackers retake bombsite from defenders",
		DefaultMap:  "de_dust2",
		Cvars: map[string]string{
			"game_type":      "0",
			"game_mode":      "0",
			"map":            "de_dust2",
			"mp_freezetime":  "3",
			"mp_roundtime":   "0.75",
			"mp_startmoney":  "4000",
			"mp_friendlyfire": "0",
		},
	},
	{
		Name:        "Surf",
		GameType:    3,
		GameMode:    0,
		Description: "Surf maps with custom movement",
		DefaultMap:  "surf_beginner",
		Cvars: map[string]string{
			"game_type":  "3",
			"game_mode":  "0",
			"map":        "surf_beginner",
			"sv_airaccelerate": "150",
			"sv_accelerate":   "10",
			"sv_gravity":      "800",
			"sv_friction":     "4",
			"mp_roundtime":   "60",
			"mp_freezetime":  "0",
		},
	},
	{
		Name:        "KZ (Climb)",
		GameType:    3,
		GameMode:    0,
		Description: "KZ/Climb maps with bunnyhop and strafe mechanics",
		DefaultMap:  "kz_beginner",
		Cvars: map[string]string{
			"game_type":       "3",
			"game_mode":       "0",
			"map":             "kz_beginner",
			"sv_airaccelerate": "1000",
			"sv_accelerate":   "10",
			"sv_gravity":      "800",
			"sv_staminalandcost": "0",
			"sv_staminajumpcost": "0",
			"mp_roundtime":    "60",
			"mp_freezetime":   "0",
		},
	},
}

// LANOptimizedCvars contains network and performance settings for LAN play.
var LANOptimizedCvars = map[string]string{
	"sv_maxrate":              "786432",
	"sv_minrate":              "786432",
	"sv_maxupdaterate":        "128",
	"sv_minupdaterate":        "128",
	"sv_maxcmdrate":          "128",
	"sv_mincmdrate":          "128",
	"net_maxroutable":         "1200",
	"sv_maxunlag":             "0.5",
	"fps_max":                 "512",
	"sv_parallel_sendsnapshot": "1",
	"sv_clockcorrection_msecs": "15",
}

// GetPresetByName returns the preset with the given name, or nil if not found.
func GetPresetByName(name string) *GameModePreset {
	for i := range Presets {
		if Presets[i].Name == name {
			return &Presets[i]
		}
	}
	return nil
}

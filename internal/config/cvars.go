package config

import "strings"

// CvarDef represents a CS2 console variable definition.
type CvarDef struct {
	Name        string `json:"name"`
	Type        string `json:"type"`        // "string", "int", "float", "bool"
	Default     string `json:"default"`
	Description string `json:"description"`
	Category    string `json:"category"`    // "server", "gameplay", "network", "bots", "performance"
	Min         string `json:"min,omitempty"`
	Max         string `json:"max,omitempty"`
}

// CvarDatabase contains definitions for commonly used CS2 cvars.
var CvarDatabase = []CvarDef{
	// ── Server ────────────────────────────────────────────────────────
	{Name: "hostname", Type: "string", Default: "CS2 Server", Description: "Server name displayed in browser", Category: "server"},
	{Name: "sv_password", Type: "string", Default: "", Description: "Server password (empty = no password)", Category: "server"},
	{Name: "rcon_password", Type: "string", Default: "", Description: "RCON remote console password", Category: "server"},
	{Name: "sv_cheats", Type: "bool", Default: "0", Description: "Enable cheats on the server", Category: "server", Min: "0", Max: "1"},
	{Name: "sv_lan", Type: "bool", Default: "0", Description: "LAN mode (no Steam authentication)", Category: "server", Min: "0", Max: "1"},
	{Name: "sv_pure", Type: "int", Default: "1", Description: "File consistency mode (0=off, 1=warn, 2=enforce)", Category: "server", Min: "0", Max: "2"},
	{Name: "sv_visiblemaxplayers", Type: "int", Default: "-1", Description: "Max players shown in server browser (-1 = use maxplayers)", Category: "server"},
	{Name: "sv_setsteamaccount", Type: "string", Default: "", Description: "Game Server Login Token (GSLT)", Category: "server"},
	{Name: "tv_enable", Type: "bool", Default: "0", Description: "Enable GOTV", Category: "server", Min: "0", Max: "1"},

	// ── Gameplay ──────────────────────────────────────────────────────
	{Name: "mp_roundtime", Type: "float", Default: "1.92", Description: "Round time in minutes", Category: "gameplay", Min: "0.5", Max: "60"},
	{Name: "mp_roundtime_defuse", Type: "float", Default: "1.92", Description: "Defuse round time in minutes", Category: "gameplay", Min: "0.5", Max: "60"},
	{Name: "mp_freezetime", Type: "int", Default: "15", Description: "Freeze time at round start (seconds)", Category: "gameplay", Min: "0", Max: "60"},
	{Name: "mp_buytime", Type: "int", Default: "20", Description: "Buy time in seconds", Category: "gameplay", Min: "0", Max: "999"},
	{Name: "mp_maxrounds", Type: "int", Default: "24", Description: "Max rounds before map change (MR)", Category: "gameplay", Min: "0", Max: "999"},
	{Name: "mp_overtime_enable", Type: "bool", Default: "0", Description: "Enable overtime on tied matches", Category: "gameplay", Min: "0", Max: "1"},
	{Name: "mp_overtime_maxrounds", Type: "int", Default: "6", Description: "Overtime max rounds", Category: "gameplay", Min: "1", Max: "24"},
	{Name: "mp_warmup_time", Type: "int", Default: "60", Description: "Warmup duration in seconds", Category: "gameplay", Min: "0", Max: "600"},
	{Name: "mp_warmuptime", Type: "int", Default: "60", Description: "Warmup duration (alias)", Category: "gameplay", Min: "0", Max: "600"},
	{Name: "mp_friendlyfire", Type: "bool", Default: "1", Description: "Enable friendly fire", Category: "gameplay", Min: "0", Max: "1"},
	{Name: "mp_halftime", Type: "bool", Default: "1", Description: "Enable halftime team swap", Category: "gameplay", Min: "0", Max: "1"},
	{Name: "mp_match_can_clinch", Type: "bool", Default: "1", Description: "End match early when clinched", Category: "gameplay", Min: "0", Max: "1"},
	{Name: "mp_startmoney", Type: "int", Default: "800", Description: "Starting money", Category: "gameplay", Min: "0", Max: "65535"},
	{Name: "mp_free_armor", Type: "int", Default: "0", Description: "Free armor (0=none, 1=kevlar, 2=kevlar+helmet)", Category: "gameplay", Min: "0", Max: "2"},
	{Name: "mp_autoteambalance", Type: "bool", Default: "1", Description: "Auto-balance teams", Category: "gameplay", Min: "0", Max: "1"},
	{Name: "mp_limitteams", Type: "int", Default: "2", Description: "Max team size difference", Category: "gameplay", Min: "0", Max: "30"},
	{Name: "mp_c4timer", Type: "int", Default: "40", Description: "C4 bomb timer in seconds", Category: "gameplay", Min: "10", Max: "90"},
	{Name: "mp_death_drop_gun", Type: "bool", Default: "1", Description: "Drop weapon on death", Category: "gameplay", Min: "0", Max: "1"},
	{Name: "mp_round_restart_delay", Type: "int", Default: "7", Description: "Delay between rounds (seconds)", Category: "gameplay", Min: "0", Max: "30"},
	{Name: "mp_win_panel_display_time", Type: "int", Default: "3", Description: "Win panel display time (seconds)", Category: "gameplay", Min: "0", Max: "30"},

	// ── Network ───────────────────────────────────────────────────────
	{Name: "sv_maxrate", Type: "int", Default: "0", Description: "Max bandwidth rate (bytes/sec, 0=unlimited)", Category: "network", Min: "0", Max: "786432"},
	{Name: "sv_minrate", Type: "int", Default: "128000", Description: "Min bandwidth rate (bytes/sec)", Category: "network", Min: "0", Max: "786432"},
	{Name: "sv_maxupdaterate", Type: "int", Default: "64", Description: "Max server→client update rate (Hz)", Category: "network", Min: "10", Max: "128"},
	{Name: "sv_minupdaterate", Type: "int", Default: "64", Description: "Min server→client update rate (Hz)", Category: "network", Min: "10", Max: "128"},
	{Name: "sv_maxcmdrate", Type: "int", Default: "64", Description: "Max client→server command rate (Hz)", Category: "network", Min: "10", Max: "128"},
	{Name: "sv_mincmdrate", Type: "int", Default: "64", Description: "Min client→server command rate (Hz)", Category: "network", Min: "10", Max: "128"},
	{Name: "net_maxroutable", Type: "int", Default: "1200", Description: "Max routable packet payload size", Category: "network", Min: "576", Max: "1200"},
	{Name: "sv_maxunlag", Type: "float", Default: "1.0", Description: "Max lag compensation (seconds)", Category: "network", Min: "0", Max: "2"},

	// ── Bots ──────────────────────────────────────────────────────────
	{Name: "bot_quota", Type: "int", Default: "0", Description: "Number of bots", Category: "bots", Min: "0", Max: "64"},
	{Name: "bot_quota_mode", Type: "string", Default: "normal", Description: "Bot quota mode (fill, match, normal)", Category: "bots"},
	{Name: "bot_difficulty", Type: "int", Default: "1", Description: "Bot difficulty (0=easy, 1=normal, 2=hard, 3=expert)", Category: "bots", Min: "0", Max: "3"},
	{Name: "bot_knives_only", Type: "bool", Default: "0", Description: "Bots only use knives", Category: "bots", Min: "0", Max: "1"},
	{Name: "bot_allow_rogues", Type: "bool", Default: "1", Description: "Bots can go rogue", Category: "bots", Min: "0", Max: "1"},
	{Name: "bot_join_after_player", Type: "bool", Default: "1", Description: "Bots only join after a player", Category: "bots", Min: "0", Max: "1"},
	{Name: "bot_chatter", Type: "string", Default: "normal", Description: "Bot chatter level (off, radio, minimal, normal)", Category: "bots"},

	// ── Performance ───────────────────────────────────────────────────
	{Name: "fps_max", Type: "int", Default: "300", Description: "Max server framerate", Category: "performance", Min: "30", Max: "1000"},
	{Name: "sv_parallel_sendsnapshot", Type: "bool", Default: "1", Description: "Parallel snapshot sending", Category: "performance", Min: "0", Max: "1"},
	{Name: "sv_clockcorrection_msecs", Type: "int", Default: "60", Description: "Clock correction max (ms)", Category: "performance", Min: "0", Max: "200"},
}

// GetCvarByName returns the cvar definition for the given name (case-insensitive).
func GetCvarByName(name string) *CvarDef {
	lower := strings.ToLower(name)
	for i := range CvarDatabase {
		if strings.ToLower(CvarDatabase[i].Name) == lower {
			return &CvarDatabase[i]
		}
	}
	return nil
}

// GetCvarsByCategory returns all cvars in the given category.
func GetCvarsByCategory(category string) []CvarDef {
	lower := strings.ToLower(category)
	var result []CvarDef
	for _, c := range CvarDatabase {
		if strings.ToLower(c.Category) == lower {
			result = append(result, c)
		}
	}
	return result
}

// SearchCvars performs a case-insensitive substring search on name and description.
func SearchCvars(query string) []CvarDef {
	lower := strings.ToLower(query)
	var result []CvarDef
	for _, c := range CvarDatabase {
		if strings.Contains(strings.ToLower(c.Name), lower) || strings.Contains(strings.ToLower(c.Description), lower) {
			result = append(result, c)
		}
	}
	return result
}

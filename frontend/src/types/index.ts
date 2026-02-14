export interface ServerInstance {
  id: string;
  name: string;
  port: number;
  rcon_port: number;
  status:
    | "stopped"
    | "running"
    | "starting"
    | "stopping"
    | "installing"
    | "updating"
    | "crashed";
  game_mode: string;
  max_players: number;
  current_map: string;
  install_path: string;
  launch_args: string;
  rcon_password: string;
  gslt_token: string;
  auto_restart: boolean;
  auto_start: boolean;
  created_at: string;
  updated_at: string;
}

export interface InstanceConfig {
  name: string;
  install_path: string;
  port: number;
  max_players: number;
  game_mode: string;
  map: string;
  rcon_password: string;
  launch_args: string;
  auto_restart: boolean;
  auto_start: boolean;
}

export interface AppConfig {
  app_data_dir: string;
  log_dir: string;
  log_level: string;
  steamcmd_path: string;
  default_install_dir: string;
  backup_dir: string;
  theme: string;
  minimize_to_tray: boolean;
  start_with_windows: boolean;
  auto_update: boolean;
  discord_webhook: string;
}

export interface CvarDef {
  name: string;
  type: string;
  default: string;
  description: string;
  category: string;
  min?: string;
  max?: string;
}

export interface GameModePreset {
  name: string;
  game_type: number;
  game_mode: number;
  description: string;
  cvars: Record<string, string>;
  default_map: string;
}

export interface MapInfo {
  name: string;
  file_name: string;
  size_bytes: number;
}

export interface Player {
  name: string;
  steam_id: string;
  ping: number;
  score: number;
  team: string;
  ip: string;
}

export interface BotConfig {
  quota: number;
  quota_mode: string;
  difficulty: number;
}

export interface BanEntry {
  id: string;
  instance_id: string;
  steam_id: string;
  ip_address: string;
  reason: string;
  expires_at: string | null;
  is_permanent: boolean;
  created_at: string;
}

export interface ConfigProfile {
  id: string;
  instance_id: string;
  name: string;
  data: string;
  is_active: boolean;
  created_at: string;
}

export interface Progress {
  stage: string;
  percent: number;
  message: string;
}

export interface BenchmarkResult {
  id: string;
  instance_id: string;
  bot_count: number;
  avg_tickrate: number;
  min_tickrate: number;
  avg_frametime: number;
  cpu_usage: number;
  ram_usage: number;
  duration_sec: number;
  created_at: string;
}

export interface MetricSnapshot {
  cpu_pct: number;
  ram_mb: number;
  tick_rate: number;
  players: number;
  net_in_kbps: number;
  net_out_kbps: number;
  timestamp: string;
}

export interface Match {
  id: string;
  instance_id: string;
  map_name: string;
  game_mode: string;
  team1_score: number;
  team2_score: number;
  duration_sec: number;
  rounds_played: number;
  started_at: string;
  ended_at: string;
}

export interface MatchPlayer {
  id: string;
  match_id: string;
  steam_id: string;
  player_name: string;
  team: string;
  kills: number;
  deaths: number;
  assists: number;
  headshots: number;
  mvps: number;
  total_damage: number;
  utility_damage: number;
  enemies_flashed: number;
  adr: number;
  hsp: number;
  score: number;
}

export interface Skin {
  id: number;
  paint_kit_id: number;
  name: string;
  weapon_type: string;
  rarity: string;
  rarity_color: string;
  min_float: number;
  max_float: number;
  image_url: string;
  category: string;
  collection: string;
}

export type StatusColor = "green" | "red" | "yellow" | "blue" | "gray";

export interface Backup {
  id: string;
  instance_id: string;
  path: string;
  size_bytes: number;
  backup_type: string;
  created_at: string;
}

export interface ScheduledTask {
  id: string;
  instance_id: string;
  cron_expr: string;
  action: string;
  payload: string;
  enabled: boolean;
  last_run: string | null;
  next_run: string | null;
  created_at: string;
}

export interface PluginInfo {
  name: string;
  installed: boolean;
  version: string;
  path: string;
  enabled: boolean;
}

export interface FileEntry {
  name: string;
  path: string;
  is_dir: boolean;
  size: number;
  modified: string;
}

export interface AuditEntry {
  id: string;
  action: string;
  target: string;
  details: string;
  created_at: string;
}

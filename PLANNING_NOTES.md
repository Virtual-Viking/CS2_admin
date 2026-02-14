# CS2 Admin Panel — Pre-Implementation Planning & Feasibility

> Investigation results for all critical questions before writing code.

---

## Table of Contents

1. [Anonymous Server Creation & Launch](#1-anonymous-server-creation--launch)
2. [Purchased Skins Visibility](#2-purchased-skins-visibility)
3. [Equipping Unpurchased Skins (Dragon Lore, Gungnir, Knives, Gloves)](#3-equipping-unpurchased-skins)
4. [Complete Skin Database (Purple, Red, Gold Tiers)](#4-complete-skin-database)
5. [Lowest Latency & Best Gameplay Feel](#5-lowest-latency--best-gameplay-feel)
6. [CS2 Sub-Tick System vs CSGO 128 Tick — LAN Optimization](#6-cs2-sub-tick-system-vs-csgo-128-tick)
7. [End-of-Match Statistics & Damage Report](#7-end-of-match-statistics--damage-report)

---

## 1. Anonymous Server Creation & Launch

### Question
> Can we start a CS2 dedicated server without it showing "Playing Counter-Strike 2" on the host's Steam profile, so the same user can join the server with their own account?

### Findings

**YES — this is fully supported.** The CS2 dedicated server is a separate process from the Steam client. Here's how it works:

```
┌──────────────────────────────────────────────────────────────────┐
│                    How CS2 Server Stays Anonymous                  │
│                                                                  │
│  Steam Client (User's account)      CS2 Dedicated Server         │
│  ────────────────────────────        ────────────────────         │
│  Logged in as: "Player1"            Runs as: cs2.exe -dedicated  │
│  Status: Online / In Menu            Status: Separate process     │
│  Can launch CS2 and JOIN server      No Steam profile link        │
│                                                                  │
│  These are COMPLETELY INDEPENDENT processes.                      │
│  The server does NOT use the Steam client at all.                │
└──────────────────────────────────────────────────────────────────┘
```

### Implementation Plan

**Step 1: Install server files via SteamCMD with anonymous login**

```powershell
# SteamCMD anonymous login — no Steam account needed for download
steamcmd.exe +login anonymous +force_install_dir "C:\CS2Servers\Instance1" +app_update 730 validate +quit
```

> **Note**: Earlier sources claimed CS2 required authenticated login for SteamCMD. As of late 2024, CS2 server files (App ID 730) CAN be downloaded with `+login anonymous`. This was confirmed by LinuxGSM issue #4364 and multiple community guides.

**Step 2: Launch as LAN server without GSLT**

```powershell
# No GSLT token = no link to any Steam account
# sv_lan 1 = LAN-only mode (no public server list)
cs2.exe -dedicated -port 27015 +map de_dust2 +sv_lan 1 +game_mode 1 +game_type 0
```

**Step 3: Players connect via console or LAN browser**

```
# In CS2 client console:
connect 192.168.1.100:27015

# Or use the in-game Community Server Browser → LAN tab
```

### Key Architecture Decisions

| Decision | Choice | Rationale |
|---|---|---|
| SteamCMD login | `anonymous` | No account credentials needed, no profile link |
| GSLT token | **Not used** for LAN | Eliminates all GSLT ban risk, no profile link |
| `sv_lan` | `1` (LAN only) | Server stays off public lists, local network only |
| Server process | Child process of CS2Admin.exe | Completely separate from Steam client |
| Network | LAN IP (192.168.x.x) | Players connect via `connect <IP>:<port>` |

### Risk Assessment

| Risk | Severity | Mitigation |
|---|---|---|
| SteamCMD changes to require login for App 730 | Low | Fall back to authenticated login with a secondary Steam account |
| User tries to run server + client on same machine | None | Works fine — they are separate processes, separate ports |
| Server not visible in public browser | Intended | LAN-only by design; provide direct `connect` command in UI |

### Verdict: FULLY FEASIBLE — No issues.

---

## 2. Purchased Skins Visibility

### Question
> Will players see their own purchased/inventory skins when playing on our LAN dedicated server?

### Findings

**YES — purchased skins appear automatically.** When a player connects to ANY CS2 server (official, community, or LAN), their equipped inventory skins are loaded from Steam's item server and displayed in-game. This is a client-side Steam feature that works regardless of server type.

```
┌────────────────────────────────────────────────────────┐
│              How Purchased Skins Work                   │
│                                                        │
│  Player joins server                                   │
│       │                                                │
│       ▼                                                │
│  CS2 client contacts Steam Item Server                 │
│       │                                                │
│       ▼                                                │
│  Steam returns player's equipped loadout               │
│  (skins, knives, gloves, agents, music kits)           │
│       │                                                │
│       ▼                                                │
│  Client renders skins locally                          │
│  Other players also see these skins                    │
│                                                        │
│  ⚡ This happens automatically on ALL server types     │
│     including sv_lan 1 servers                          │
│                                                        │
│  ⚠️  Requires: Internet connection for initial         │
│     Steam inventory fetch (cached afterward)            │
└────────────────────────────────────────────────────────┘
```

### Requirements for Our App

1. **Internet connection required** (at least briefly) for Steam to authenticate players and load inventories
2. Even on `sv_lan 1` servers, players connect with their Steam accounts — skins load automatically
3. No plugin or configuration needed — this is built into CS2

### Edge Case: Fully Offline LAN

If the network has **zero internet access**, Steam cannot authenticate players or load inventories. In this case:
- Players would need to be connected to Steam in offline mode
- Inventory skins **may not load** without Steam item server access
- Workaround: Ensure the LAN has internet, even if limited, for Steam auth

### Verdict: WORKS OUT OF THE BOX — No action needed.

---

## 3. Equipping Unpurchased Skins

### Question
> Can players equip skins they don't own (Dragon Lore, Gungnir, knives, gloves) via in-game commands on our server?

### Findings

**YES — using the WeaponPaints server-side plugin.** This is a well-established CounterStrikeSharp plugin that overrides the player's equipped skins on the server side.

### The Plugin: WeaponPaints (Nereziel/cs2-WeaponPaints)

```
Repository:  https://github.com/Nereziel/cs2-WeaponPaints
Stars:       338+ on GitHub
Framework:   CounterStrikeSharp (C# plugin for CS2)
License:     Open source
Status:      Actively maintained (2025)
```

**In-Game Commands:**

| Command | Function |
|---|---|
| `!skins` or `!ws` | Open weapon skin selection menu |
| `!knife` | Open knife skin selection menu (Karambit, Butterfly, Bayonet, etc.) |
| `!gloves` | Open glove skin selection menu |
| `!agents` | Open agent/player model selection |
| `!pins` | Open collectible pins menu |
| `!music` | Open music kit selection |
| `!wp` | Refresh/reload skins |

**How It Works:**

```
┌─────────────────────────────────────────────────────────────┐
│            WeaponPaints Plugin Architecture                   │
│                                                             │
│  Player types !knife                                        │
│       │                                                     │
│       ▼                                                     │
│  Plugin shows in-game menu of all knives                    │
│  (Karambit, Butterfly, M9 Bayonet, Skeleton, etc.)          │
│       │                                                     │
│       ▼                                                     │
│  Player selects "Karambit | Fade"                           │
│       │                                                     │
│       ▼                                                     │
│  Plugin stores choice in database (SQLite/MySQL)            │
│       │                                                     │
│       ▼                                                     │
│  Plugin overrides the player's weapon model server-side     │
│  using CounterStrikeSharp's API                             │
│       │                                                     │
│       ▼                                                     │
│  ALL players on the server see the custom skin              │
│  (not just the player who equipped it)                      │
│                                                             │
│  ✅ Server-side only — no client modification               │
│  ✅ No VAC risk — plugin runs on the server, not client     │
│  ✅ Persists across reconnects (stored in DB)               │
└─────────────────────────────────────────────────────────────┘
```

### GSLT Ban Risk Analysis

| Server Type | GSLT Used? | Skin Plugin Risk | Our Case |
|---|---|---|---|
| Public internet server | Yes (required) | **HIGH** — Valve can ban GSLT | ❌ Not us |
| Private internet server | Yes (recommended) | **MEDIUM** — still tracked by Valve | ❌ Not us |
| **LAN server (sv_lan 1)** | **No** | **NONE** — no GSLT to ban | ✅ **This is us** |

**Since we run LAN-only servers without GSLT, there is ZERO risk of GSLT ban.** Valve's enforcement mechanism (GSLT banning) simply doesn't apply to servers that don't use GSLT tokens.

### Dependencies for Our App

```
Required plugins stack:
├── Metamod:Source          ← Plugin loader for Source 2 engine
│   └── Install to: game/csgo/addons/metamod/
├── CounterStrikeSharp      ← C# plugin framework for CS2
│   └── Install to: game/csgo/addons/counterstrikesharp/
└── WeaponPaints            ← The actual skin plugin
    └── Install to: game/csgo/addons/counterstrikesharp/plugins/WeaponPaints/
```

### Integration Plan for CS2 Admin

Our app will:

1. **Auto-install the plugin stack** (Metamod → CounterStrikeSharp → WeaponPaints) as a one-click setup in the Plugins tab
2. **Bundle a pre-built skin database** with all paint kit IDs, knife IDs, glove IDs (parsed from `items_game.txt`)
3. **Provide a visual skin browser** in the app UI — player can browse skins with preview images, and the app writes to the WeaponPaints database
4. **Optional**: Build our own lightweight skin plugin using CounterStrikeSharp if WeaponPaints doesn't meet our UX needs

### Verdict: FULLY FEASIBLE — Zero ban risk on LAN.

---

## 4. Complete Skin Database (Purple, Red, Gold Tiers)

### Question
> We need a complete list of all Purple (Classified), Red (Covert), and Gold (Rare Special / knives / gloves) skins in CS2.

### Findings

CS2's skin rarity system uses color-coded tiers:

```
┌──────────────────────────────────────────────────────────────┐
│                CS2 Skin Rarity Tiers                          │
│                                                              │
│  Color        │  Rarity Name    │  Internal Value  │  Items  │
│  ─────────────┼─────────────────┼──────────────────┼──────── │
│  ⬜ White      │  Consumer Grade  │  common (1)      │  ~200   │
│  🔵 Light Blue │  Industrial Grade│  uncommon (2)    │  ~200   │
│  🔵 Blue       │  Mil-Spec       │  rare (3)        │  ~400   │
│  🟣 Purple     │  Restricted     │  mythical (4)    │  ~250   │
│  💜 Pink       │  Classified     │  legendary (5)   │  ~150   │
│  🔴 Red        │  Covert         │  ancient (6)     │  ~100   │
│  🟡 Gold       │  Rare Special   │  immortal (7)    │  ~450   │
│               │  (Knives/Gloves)│                  │  (knife │
│               │                 │                  │  +glove │
│               │                 │                  │  skins) │
│  🟠 Orange     │  Contraband     │  (8)             │  1      │
│               │  (M4A4 Howl)    │                  │         │
│                                                              │
│  Total cataloged skins: ~1,400+ weapon skins                 │
│                          ~400+ knife variants                 │
│                          ~50+ glove variants                  │
└──────────────────────────────────────────────────────────────┘
```

### Data Sources

**Primary Source — `items_game.txt` (Valve's official item definition file)**

```
URL:  https://raw.githubusercontent.com/SteamDatabase/GameTracking-CS2/master/
      game/csgo/pak01_dir/scripts/items/items_game.txt

Maintained by: SteamDatabase (auto-updated on every CS2 patch)
Format:        Valve KeyValues (VDF format)

Contains:
├── paint_kits         ← Every skin's paint ID, name, rarity, wear range
├── items              ← Weapon definitions (weapon_ak47, weapon_m4a1, etc.)
├── rarities           ← Rarity tier definitions
├── paint_kits_rarity  ← Mapping of paint_kit → rarity tier
├── sticker_kits       ← Sticker definitions
├── music_definitions  ← Music kit definitions
└── prefabs            ← Knife and glove base definitions
```

**Secondary Sources (pre-parsed, structured):**

| Source | URL | Format | Coverage |
|---|---|---|---|
| CS2Data.gg | https://cs2data.gg | Web (scrapable) | All skins, cases, collections with images |
| CSGODatabase.com | https://www.csgodatabase.com | Web | 1,401 skins, 404 knives, full rarity data |
| CSFloat DB | https://csfloat.com/db | Web + API | 1.2B+ tracked skins with float data |
| SteamDatabase GameTracking | GitHub repo | VDF file | Authoritative, auto-updated |

### Implementation Plan

We will build an **embedded skin database** in the app:

```
Step 1: Parse items_game.txt at build time
        ├── Extract all paint_kits with IDs and names
        ├── Extract rarity mappings
        ├── Extract weapon definitions (to map paint → valid weapons)
        ├── Extract knife and glove definitions
        └── Output: JSON database file

Step 2: Fetch preview images
        ├── Source: Steam CDN or CS2Data.gg
        ├── Cache locally in %APPDATA%\CS2Admin\skin_images\
        └── Show thumbnails in the in-app skin browser

Step 3: Bundle database in the app
        ├── Embed JSON skin DB in Go binary (go:embed)
        ├── Auto-update from GitHub GameTracking on app update
        └── User can manually refresh after CS2 patches

Step 4: Integrate with WeaponPaints plugin
        ├── Our app writes player skin choices to WeaponPaints SQLite DB
        ├── Or: build a custom lightweight plugin that reads from our DB
        └── Player browses skins in CS2 Admin UI → auto-applied on server
```

### Notable Skins to Highlight (User-Requested Examples)

**Red (Covert) Tier:**

| Weapon | Skin Name | Paint Kit ID |
|---|---|---|
| AWP | Dragon Lore | 344 |
| AWP | Gungnir | 756 |
| AWP | The Prince | 803 |
| AK-47 | Wild Lotus | 770 |
| AK-47 | Fire Serpent | 180 |
| AK-47 | Gold Arabesque | 811 |
| M4A4 | Howl | 309 (Contraband) |
| M4A4 | The Emperor | 735 |
| Desert Eagle | Blaze | 37 |

**Gold (Knives):**

| Knife Type | Example Skins |
|---|---|
| Karambit | Fade, Doppler, Gamma Doppler, Tiger Tooth, Marble Fade, Crimson Web |
| Butterfly | Fade, Doppler, Marble Fade, Slaughter, Crimson Web |
| M9 Bayonet | Doppler, Fade, Marble Fade, Tiger Tooth |
| Skeleton Knife | Fade, Crimson Web, Slaughter |
| Sport Gloves | Pandora's Box, Vice, Superconductor |
| Specialist Gloves | Crimson Kimono, Fade, Emerald Web |

### Skin Database Schema for Our App

```sql
CREATE TABLE skins (
    id              INTEGER PRIMARY KEY,
    paint_kit_id    INTEGER NOT NULL,        -- Valve's paint kit index
    name            TEXT NOT NULL,            -- "Dragon Lore"
    weapon_type     TEXT NOT NULL,            -- "weapon_awp", "weapon_knife_karambit"
    rarity          TEXT NOT NULL,            -- "covert", "classified", "rare_special"
    rarity_color    TEXT NOT NULL,            -- "#eb4b4b" (red), "#d32ce6" (pink), "#ffd700" (gold)
    min_float       REAL DEFAULT 0.0,        -- Minimum wear float
    max_float       REAL DEFAULT 1.0,        -- Maximum wear float
    image_url       TEXT,                     -- Steam CDN image URL
    image_cached    BOOLEAN DEFAULT FALSE,   -- Local cache status
    category        TEXT,                     -- "rifle", "pistol", "knife", "glove", "smg", etc.
    collection      TEXT,                     -- "Cobblestone Collection", "Fever Case", etc.
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Index for fast browsing by rarity tier
CREATE INDEX idx_skins_rarity ON skins(rarity);
CREATE INDEX idx_skins_weapon ON skins(weapon_type);
CREATE INDEX idx_skins_category ON skins(category);
```

### Verdict: FULLY FEASIBLE — Data sources are reliable and auto-updating.

---

## 5. Lowest Latency & Best Gameplay Feel

### Question
> Server must provide the lowest possible latency. Gameplay feel should be as good as 128-tick or better, especially on LAN.

### Findings

On a LAN setup, we have a **massive advantage**: network latency is essentially **0 ms** (sub-1 ms). The main optimizations are about maximizing the data exchange rate between client and server and minimizing interpolation delay.

### Optimal LAN Server Configuration

```cfg
// ── server.cfg — Optimized for LAN (Lowest Latency) ──

// Network rates — maximize for LAN bandwidth
sv_maxrate              786432      // Max bandwidth: 768 KB/s per player (LAN can handle it)
sv_minrate              786432      // Force max rate on LAN
sv_maxupdaterate        128         // Server sends updates 128 times/sec to each client
sv_minupdaterate        128         // Force 128 updates/sec
sv_maxcmdrate           128         // Accept 128 commands/sec from each client
sv_mincmdrate           128         // Force 128 commands/sec

// Interpolation — minimize delay
sv_clockcorrection_msecs 15         // Clock correction window
sv_maxunlag             0.5         // Max lag compensation (seconds)

// Sub-tick optimization
sv_cheats               0           // Keep clean
net_maxroutable          1200       // Max packet size (bytes)

// Anti-cheat & fairness
sv_pure                 1           // Enforce file consistency
sv_allow_lobby_connect_only 0       // Allow direct connect

// Performance
sv_parallel_sendsnapshot 1          // Parallel snapshot sending
fps_max                  512        // Uncap server FPS (let it run as fast as possible)

// LAN-specific
sv_lan                  1           // LAN mode
sv_region               255         // Not applicable for LAN
```

### Client-Side Recommended Settings (Autoexec)

Our app will generate a recommended `autoexec.cfg` that players can use:

```cfg
// ── Client autoexec.cfg — Optimized for LAN ──

rate                    786432      // Match server's max rate
cl_updaterate           128         // Receive 128 updates/sec
cl_cmdrate              128         // Send 128 commands/sec
cl_interp               0           // Let engine calculate minimum interp
cl_interp_ratio         1           // Minimum interpolation ratio (1 tick buffer)

// The above gives effective interpolation of ~7.8ms (1/128)
// vs default ~15.6ms (1/64) — HALF the input lag
```

### Latency Breakdown on LAN

```
┌──────────────────────────────────────────────────────────────────┐
│          Latency Budget: LAN Server (Our Setup)                   │
│                                                                  │
│  Component                        │  Latency                     │
│  ─────────────────────────────────┼──────────────────────────     │
│  Network (LAN switch)             │  < 0.5 ms                    │
│  Server tick processing           │  ~7.8 ms (1/128 updates)     │
│  Client interpolation (optimized) │  ~7.8 ms (cl_interp_ratio 1) │
│  Client rendering (240fps)        │  ~4.2 ms (1/240)             │
│  Monitor refresh (144Hz)          │  ~6.9 ms (1/144)             │
│  ─────────────────────────────────┼──────────────────────────     │
│  TOTAL INPUT-TO-SCREEN            │  ~27 ms                      │
│                                                                  │
│  vs. Official Matchmaking (64 tick, 50ms ping):                  │
│  Network                          │  ~50 ms                      │
│  Server tick processing           │  ~15.6 ms (1/64)             │
│  Client interpolation             │  ~15.6 ms                    │
│  Client rendering + display       │  ~11 ms                      │
│  TOTAL                            │  ~92 ms                      │
│                                                                  │
│  ⚡ Our LAN setup is ~3.4x faster than official matchmaking      │
└──────────────────────────────────────────────────────────────────┘
```

### Verdict: ACHIEVABLE — LAN inherently beats internet play by 3x+.

---

## 6. CS2 Sub-Tick System vs CSGO 128 Tick

### Question
> CS2 doesn't have 128-tick servers like CSGO did. How does CS2 refresh data? How do we optimize gameplay response for LAN?

### Findings

### How CSGO Tick Rate Worked (Old System)

```
CSGO 64-tick server:
  tick 1 ──── tick 2 ──── tick 3 ──── tick 4 ────
  15.6ms       15.6ms       15.6ms       15.6ms

  Player shoots at 12ms → registered at tick 2 (15.6ms)
  Error: up to 15.6ms of "missed" timing

CSGO 128-tick server:
  t1 ─ t2 ─ t3 ─ t4 ─ t5 ─ t6 ─ t7 ─ t8 ─
  7.8ms 7.8ms 7.8ms 7.8ms 7.8ms 7.8ms 7.8ms

  Player shoots at 12ms → registered at tick 2 (15.6ms)
  Error: up to 7.8ms of "missed" timing
  Result: Tighter hit registration, smoother movement
```

### How CS2 Sub-Tick Works (New System)

```
CS2 64 Hz server with sub-tick:
  tick 1 ──────────── tick 2 ──────────── tick 3 ────
  0ms                 15.6ms               31.2ms

  Player shoots at 12ms
  → Client records EXACT timestamp: 12.0ms
  → Sends to server: "shot fired at t=12.0ms"
  → Server applies shot at 12.0ms (NOT rounded to tick boundary)
  → Error: ~0ms for the shot itself

  BUT: The server still only BROADCASTS game state 64 times/sec
  → Other players see the result with up to 15.6ms delay
  → Movement still updates at 64 Hz boundaries
```

### Key Insight for Our LAN Setup

CS2's sub-tick fixes **hit registration accuracy** (shots register at exact timestamps), but the **game state broadcast rate** is still 64 Hz on official servers. However, third-party dedicated servers CAN set higher update rates.

**What we can optimize on our LAN server:**

```
┌──────────────────────────────────────────────────────────────────┐
│              Our Optimization Strategy                             │
│                                                                  │
│  Layer 1: Sub-tick (built-in)                                    │
│  ├── Hit registration: sub-millisecond accuracy ✅ (automatic)    │
│  ├── Jump throws: precise timing ✅ (automatic)                   │
│  └── Nothing to configure — it's engine-level                    │
│                                                                  │
│  Layer 2: Network update rates (our optimization)                │
│  ├── sv_maxupdaterate 128 → server sends state 128x/sec         │
│  ├── sv_maxcmdrate 128 → client sends input 128x/sec            │
│  ├── cl_updaterate 128 → client requests 128 updates/sec        │
│  ├── cl_interp_ratio 1 → minimum interpolation buffer            │
│  └── rate 786432 → max bandwidth (trivial on LAN)               │
│                                                                  │
│  Layer 3: Server performance (our optimization)                  │
│  ├── fps_max 512 → uncap server framerate                        │
│  ├── CPU affinity → pin CS2 server to dedicated cores            │
│  ├── Process priority → Above Normal                             │
│  └── Minimal plugins → reduce per-tick overhead                  │
│                                                                  │
│  Combined Result:                                                │
│  ├── Sub-tick precision for hit registration (~0ms error)        │
│  ├── 128 updates/sec game state broadcast                        │
│  ├── ~7.8ms effective tick interval (matches old 128-tick feel)  │
│  └── <1ms network latency on LAN                                │
│                                                                  │
│  ⚡ BETTER than CSGO 128-tick + sub-tick precision on top         │
└──────────────────────────────────────────────────────────────────┘
```

### Important Caveat

> Valve forced all servers (including FACEIT) to 64 Hz in official matchmaking, but **private/LAN dedicated servers CAN use higher update rates** via the `sv_maxupdaterate` and `sv_maxcmdrate` convars. The sub-tick system works ALONGSIDE these higher rates.

### Server Performance Tuning (Our App's Responsibility)

| Setting | Value | Effect |
|---|---|---|
| `fps_max` | `512` | Let server run as fast as hardware allows |
| CPU affinity | Pin to cores 2–3 | Dedicate cores to CS2, keep CS2Admin on core 0 |
| Process priority | `Above Normal` | Prioritize game processing |
| `sv_parallel_sendsnapshot` | `1` | Parallelize network sends |
| Minimize plugins | Only essential (skins, stats) | Reduce per-tick computation |
| SSD for maps | Required | Fastest map loading |

### LAN Preset (Built Into Our App)

Our app will include a **"LAN Tournament" preset** that auto-applies all of these settings:

```
1. Server config: optimized cvars (as above)
2. Client config: generates recommended autoexec.cfg for players
3. CPU affinity: auto-pins CS2 server to dedicated cores
4. Process priority: auto-elevates
5. Performance monitoring: live tick rate graph to verify optimization
6. Pre-match benchmark: optional bot stress test to validate performance
```

### Verdict: BETTER THAN 128-TICK — Sub-tick + 128 update rate + LAN = best possible experience.

---

## 7. End-of-Match Statistics & Damage Report

### Question
> At the end of a match, players should see full stats: damage given to each player, kills, MVPs, bomb plants/defuses, etc.

### Findings

CounterStrikeSharp provides **full access** to match statistics and damage records through its API.

### Available Data Points

**CSMatchStats_t (per-player match statistics):**

```csharp
// Available properties from CounterStrikeSharp API
class CSMatchStats_t {
    int Kills;
    int Deaths;
    int Assists;
    int Damage;               // Total damage dealt
    int HeadShotKills;
    int UtilityDamage;        // Grenade damage
    int EnemiesFlashed;
    int Enemy2Ks;             // Double kills
    int Enemy3Ks;             // Triple kills
    int Enemy4Ks;             // Quad kills
    int Enemy5Ks;             // Aces
    int EquipmentValue;
    int MoneySaved;
    int KillReward;
    int LiveTime;             // Time alive
    int MVPs;
}
```

**CDamageRecord (per-hit damage details):**

```csharp
class CDamageRecord {
    int   Damage;                  // Damage dealt
    int   ActualHealthRemoved;     // Actual HP removed
    int   NumHits;                 // Number of hits
    int   BulletsDamage;           // Bullet-specific damage
    ulong DamagerXuid;             // Attacker's Steam ID
    ulong RecipientXuid;           // Victim's Steam ID
    CCSPlayerController PlayerControllerDamager;   // Attacker reference
    CCSPlayerController PlayerControllerRecipient;  // Victim reference
    int   KillType;                // Type of kill
}
```

**Events we can hook into:**

| Event | When | Data |
|---|---|---|
| `EventRoundEnd` | End of each round | Winner, reason, round stats |
| `EventPlayerDeath` | Each kill | Killer, victim, weapon, headshot, penetrated, etc. |
| `EventBombPlanted` | Bomb plant | Planter, site |
| `EventBombDefused` | Bomb defuse | Defuser, site |
| `EventBombExploded` | Bomb explosion | Site |
| `EventPlayerHurt` | Each hit | Attacker, victim, damage, hitgroup, weapon |
| `EventCsWinPanelMatch` | Match end panel | Final scores |
| `EventBulletDamage` | Each bullet | Precise damage info |

### Implementation Plan

We will build a **custom CounterStrikeSharp plugin** (`CS2AdminStats`) that:

```
┌──────────────────────────────────────────────────────────────────┐
│                 Match Stats System Architecture                   │
│                                                                  │
│  During Match:                                                   │
│  ─────────────                                                   │
│  EventPlayerHurt → accumulate per-player damage matrix           │
│  EventPlayerDeath → track kills, weapons, headshots              │
│  EventBombPlanted/Defused/Exploded → track bomb events           │
│  EventRoundEnd → snapshot round stats                            │
│                                                                  │
│  Data Structure (in-memory during match):                        │
│  ┌─────────────────────────────────────────────────┐             │
│  │  DamageMatrix[attacker_steam_id][victim_steam_id]│             │
│  │  ├── damage_dealt: 187                          │             │
│  │  ├── hits: 4                                    │             │
│  │  ├── headshots: 1                               │             │
│  │  ├── weapon: "AK-47"                            │             │
│  │  └── kills: 1                                   │             │
│  └─────────────────────────────────────────────────┘             │
│                                                                  │
│  At Match End (EventCsWinPanelMatch):                            │
│  ────────────────────────────────────                            │
│  1. Collect all CSMatchStats_t for each player                   │
│  2. Build final damage matrix                                    │
│  3. Write to SQLite via RCON or shared file                      │
│  4. Emit event to CS2 Admin panel app                            │
│  5. App displays comprehensive stats screen                      │
│                                                                  │
│  Display in App (Post-Match Stats Screen):                       │
│  ──────────────────────────────────────────                      │
│  ┌──────────────────────────────────────────────────────────┐    │
│  │  MATCH RESULTS — de_dust2 — Team A [16] vs [14] Team B  │    │
│  │                                                          │    │
│  │  MVP: Player3 (6 MVPs)                                   │    │
│  │  Rounds: CT 9:6 (first half) → T 7:8 (second half)      │    │
│  │  Bomb Plants: 12 | Defuses: 4 | Explosions: 8           │    │
│  │                                                          │    │
│  │  ┌─ Scoreboard ──────────────────────────────────────┐   │    │
│  │  │ Player  │ K  │ D  │ A  │ HS%  │ DMG  │ MVP │ ADR │   │    │
│  │  ├─────────┼────┼────┼────┼──────┼──────┼─────┼─────┤   │    │
│  │  │ Player1 │ 24 │ 18 │ 5  │ 52%  │ 2847 │ 4   │ 94.9│   │    │
│  │  │ Player2 │ 21 │ 16 │ 8  │ 38%  │ 2654 │ 3   │ 88.5│   │    │
│  │  │ ...     │    │    │    │      │      │     │     │   │    │
│  │  └─────────┴────┴────┴────┴──────┴──────┴─────┴─────┘   │    │
│  │                                                          │    │
│  │  ┌─ Damage Given (Player1's view) ──────────────────┐   │    │
│  │  │ Enemy     │ DMG Given │ Hits │ HS │ DMG Taken    │   │    │
│  │  ├───────────┼───────────┼──────┼────┼──────────────┤   │    │
│  │  │ Player6   │ 187       │ 4    │ 1  │ 92           │   │    │
│  │  │ Player7   │ 143       │ 3    │ 1  │ 100 (killed) │   │    │
│  │  │ Player8   │ 87        │ 2    │ 0  │ 26           │   │    │
│  │  │ ...       │           │      │    │              │   │    │
│  │  └───────────┴───────────┴──────┴────┴──────────────┘   │    │
│  │                                                          │    │
│  │  ┌─ Round Timeline ──────────────────────────────────┐   │    │
│  │  │ R1: CT Win (elimination) | R2: T Win (bomb) | ... │   │    │
│  │  └───────────────────────────────────────────────────┘   │    │
│  └──────────────────────────────────────────────────────────┘    │
└──────────────────────────────────────────────────────────────────┘
```

### Stats Database Schema

```sql
CREATE TABLE matches (
    id              TEXT PRIMARY KEY,        -- UUID
    instance_id     TEXT NOT NULL,
    map             TEXT NOT NULL,
    game_mode       TEXT NOT NULL,
    team1_score     INTEGER,
    team2_score     INTEGER,
    duration_sec    INTEGER,
    rounds_played   INTEGER,
    bomb_plants     INTEGER,
    bomb_defuses    INTEGER,
    bomb_explosions INTEGER,
    started_at      DATETIME,
    ended_at        DATETIME
);

CREATE TABLE match_players (
    id              TEXT PRIMARY KEY,
    match_id        TEXT NOT NULL REFERENCES matches(id),
    steam_id        TEXT NOT NULL,
    player_name     TEXT NOT NULL,
    team            TEXT NOT NULL,           -- "CT" or "T"
    kills           INTEGER DEFAULT 0,
    deaths          INTEGER DEFAULT 0,
    assists         INTEGER DEFAULT 0,
    headshots       INTEGER DEFAULT 0,
    mvps            INTEGER DEFAULT 0,
    total_damage    INTEGER DEFAULT 0,
    utility_damage  INTEGER DEFAULT 0,
    enemies_flashed INTEGER DEFAULT 0,
    enemy_2ks       INTEGER DEFAULT 0,
    enemy_3ks       INTEGER DEFAULT 0,
    enemy_4ks       INTEGER DEFAULT 0,
    enemy_5ks       INTEGER DEFAULT 0,
    adr             REAL DEFAULT 0.0,        -- Average Damage per Round
    hsp             REAL DEFAULT 0.0,        -- Headshot Percentage
    score           INTEGER DEFAULT 0
);

CREATE TABLE match_damage (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    match_id        TEXT NOT NULL REFERENCES matches(id),
    round_number    INTEGER NOT NULL,
    attacker_steam  TEXT NOT NULL,
    victim_steam    TEXT NOT NULL,
    damage          INTEGER NOT NULL,
    hits            INTEGER NOT NULL,
    headshots       INTEGER DEFAULT 0,
    weapon          TEXT,
    killed          BOOLEAN DEFAULT FALSE
);

CREATE TABLE match_rounds (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    match_id        TEXT NOT NULL REFERENCES matches(id),
    round_number    INTEGER NOT NULL,
    winner          TEXT NOT NULL,            -- "CT" or "T"
    win_reason      TEXT NOT NULL,            -- "elimination", "bomb_exploded", "bomb_defused", "time"
    duration_sec    INTEGER
);
```

### Data Flow: CS2 Server → App

```
Option A: Shared SQLite file (simplest)
  CS2 plugin writes stats → SQLite file in instance directory
  CS2Admin reads SQLite on match end event
  Pro: Simple, no network
  Con: File locking concerns

Option B: RCON-based (recommended)
  CS2 plugin stores stats in memory
  On match end, plugin writes JSON to a file or makes it RCON-queryable
  CS2Admin polls via RCON: "cs2admin_stats_get"
  Plugin responds with JSON stats
  Pro: Clean separation, no file locking
  Con: Slightly more complex

Option C: HTTP callback (most robust)
  CS2 plugin sends HTTP POST to CS2Admin's internal port on match end
  CS2Admin receives full match data as JSON
  Pro: Real-time, clean, works with headless mode too
  Con: Requires internal HTTP endpoint

★ Recommended: Option C (HTTP callback) for clean architecture
```

### Verdict: FULLY FEASIBLE — Rich stats API available in CounterStrikeSharp.

---

## Summary: All 7 Points — Feasibility Matrix

| # | Question | Feasible? | Risk | Approach |
|---|---|---|---|---|
| 1 | Anonymous server (no Steam profile link) | ✅ YES | None | SteamCMD anonymous + no GSLT + sv_lan 1 |
| 2 | Purchased skins visible | ✅ YES | None | Built-in CS2 feature, works on all servers |
| 3 | Equip unpurchased skins | ✅ YES | None (LAN) | WeaponPaints plugin + CounterStrikeSharp |
| 4 | Complete skin database | ✅ YES | None | Parse items_game.txt from SteamDatabase |
| 5 | Lowest latency | ✅ YES | None | LAN + optimized cvars = 3x faster than matchmaking |
| 6 | Better than 128-tick | ✅ YES | None | Sub-tick + 128 update rate + LAN = best possible |
| 7 | End-of-match stats | ✅ YES | None | CounterStrikeSharp events + custom stats plugin |

### Dependencies to Build/Integrate

```
Custom code we need to write:
├── items_game.txt parser (Go) → skin database builder
├── CS2AdminStats plugin (C#/CounterStrikeSharp) → match stats collector
├── LAN preset config generator (Go) → optimal server.cfg + autoexec.cfg
└── Skin browser UI (React) → visual skin selection with WeaponPaints integration

Third-party dependencies:
├── Metamod:Source      → Plugin loader (auto-install via our app)
├── CounterStrikeSharp  → C# plugin framework (auto-install via our app)
├── WeaponPaints        → Skin changer plugin (auto-install via our app)
└── SteamCMD            → Server installation (auto-download by our app)
```

---

*Last updated: 2026-02-13*

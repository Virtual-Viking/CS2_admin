# CS2 Admin Panel — Technical Requirements & Tech Stack

> A native Windows desktop application for managing dedicated Counter-Strike 2 servers.
> Inspired by **WindowsGSM**, CubeCoders **AMP Instance Manager**, and Hostinger **Game Panel**.

---

## Table of Contents

1. [Project Vision](#1-project-vision)
2. [Architecture Overview](#2-architecture-overview)
3. [Tech Stack](#3-tech-stack)
4. [Desktop Application Framework](#4-desktop-application-framework)
5. [Feature Requirements](#5-feature-requirements)
6. [CS2 Server Integration](#6-cs2-server-integration)
7. [Data Models](#7-data-models)
8. [Real-Time Data Pipeline](#8-real-time-data-pipeline)
9. [Security](#9-security)
10. [Distribution & Installation](#10-distribution--installation)
11. [Performance Targets](#11-performance-targets)
12. [Project Structure](#12-project-structure)
13. [Development Roadmap](#13-development-roadmap)

---

## 1. Project Vision

Build a **native Windows desktop application** (`CS2Admin.exe`) that gives server operators full control over every aspect of their CS2 dedicated server instances — from process lifecycle and RCON commands to skin workshop items, map rotations, bot AI, player management, and live performance benchmarking — all from a polished desktop GUI, with an optional headless/remote mode for Linux servers.

### How It Compares

| | WindowsGSM | AMP | Hostinger Panel | **CS2 Admin (Ours)** |
|---|---|---|---|---|
| Platform | Windows (WPF/.NET) | Web (Linux/Windows) | Web (hosted) | **Windows native + Linux headless** |
| UI Framework | WPF | Web (Less/CSS) | Web | **Wails (Go + WebView2)** |
| Game Focus | Multi-game | Multi-game | Multi-game | **CS2-specialized** |
| Depth of Control | Basic start/stop | Moderate | Basic | **Full (every cvar, RCON, skins, benchmarks)** |
| Skin Management | None | None | None | **Built-in** |
| Benchmarking | None | None | None | **Built-in** |
| Offline/LAN | Yes | Partial | No | **Yes** |
| Open Source | Yes | No | No | **Yes** |

### Design Principles

| Principle | Implementation |
|---|---|
| **Native Desktop** | Runs as a `.exe` with system tray, native notifications, and auto-start with Windows |
| **Least Latency** | Go backend with direct in-process bindings to UI — zero network hop for local use |
| **Max Compatibility** | Primary: Windows 10/11. Secondary: Linux headless mode (web remote access) |
| **Max Control** | Every CS2 cvar exposed, raw RCON terminal, file editor, full internal API |
| **Zero External Dependencies** | Single `.exe` ships everything — no Node.js, no Python, no separate server |
| **Offline-First** | Works on LAN with no internet (except Steam/Workshop downloads) |
| **Lightweight** | < 30 MB installer, < 80 MB RAM idle (vs ~150 MB+ for Electron apps) |

---

## 2. Architecture Overview

### 2.1 Desktop Mode (Primary — Windows)

The app runs as a single native `.exe`. The Go backend and the UI frontend live **in the same process**, communicating through Wails' direct function binding — no HTTP, no WebSocket, no network overhead for the local UI.

```
┌──────────────────────────────────────────────────────────────────────────┐
│                          CS2Admin.exe                                     │
│                                                                          │
│  ┌───────────────────────────┐    Wails Bindings     ┌────────────────┐ │
│  │     Go Backend (Core)     │◄────(direct call)────►│  React UI      │ │
│  │                           │    (no HTTP/WS)        │  (WebView2)    │ │
│  │  ┌─────────────────────┐  │                        │                │ │
│  │  │ Instance Manager    │──┼───► cs2.exe processes  │  TailwindCSS   │ │
│  │  │ (start/stop/watch)  │  │                        │  shadcn/ui     │ │
│  │  ├─────────────────────┤  │                        │  Recharts      │ │
│  │  │ RCON Client Pool    │──┼───► CS2 :27015 (TCP)   │  xterm.js      │ │
│  │  │ (persistent TCP)    │  │                        │  Monaco Editor │ │
│  │  ├─────────────────────┤  │                        └────────────────┘ │
│  │  │ SteamCMD Wrapper    │──┼───► steamcmd.exe                          │
│  │  ├─────────────────────┤  │                                           │
│  │  │ Config Engine       │──┼───► cfg/, mapcycle.txt                    │
│  │  ├─────────────────────┤  │                                           │
│  │  │ Monitor Collector   │──┼───► OS metrics (gopsutil)                 │
│  │  ├─────────────────────┤  │                                           │
│  │  │ File Manager        │──┼───► CS2 game directory                    │
│  │  ├─────────────────────┤  │                                           │
│  │  │ Benchmark Engine    │  │                                           │
│  │  ├─────────────────────┤  │                                           │
│  │  │ Scheduler           │  │                                           │
│  │  ├─────────────────────┤  │                                           │
│  │  │ Notification Svc    │──┼───► Windows Toast + Discord + Webhook     │
│  │  ├─────────────────────┤  │                                           │
│  │  │ SQLite (embedded)   │  │                                           │
│  │  └─────────────────────┘  │                                           │
│  └───────────────────────────┘                                           │
│                                                                          │
│  System Tray Icon ─── [Show/Hide] [Start All] [Stop All] [Exit]         │
└──────────────────────────────────────────────────────────────────────────┘
```

### 2.2 Headless Mode (Secondary — Linux / Remote Access)

For Linux servers or remote management, the same binary runs headless, exposing a web server that serves the identical React UI over HTTPS. This makes the app usable from any browser remotely.

```
┌────────────────────────────────┐          ┌────────────────────┐
│  CS2Admin (headless)           │  HTTPS   │  Browser (remote)  │
│  └── Go Backend + Web Server  │◄────────►│  └── Same React UI │
│      └── REST API + WebSocket │          └────────────────────┘
│      └── Embedded static files│
└────────────────────────────────┘
```

**Launch modes:**

```powershell
# Desktop mode (default on Windows) — opens native window
CS2Admin.exe

# Headless mode — web server only, no GUI window
CS2Admin.exe --headless --port 8443

# Headless with auto-start on boot (Linux systemd / Windows service)
CS2Admin.exe --headless --port 8443 --service
```

### 2.3 Service Layers

| Layer | Responsibility |
|---|---|
| **Wails App Shell** | Window lifecycle, system tray, native dialogs, menu bar, auto-updater |
| **Binding Layer** | Exposes Go methods to JavaScript — auto-generated TypeScript types |
| **Instance Manager** | CS2 process lifecycle (start / stop / restart / crash recovery) |
| **RCON Bridge** | Persistent TCP connections to each CS2 instance via Source RCON Protocol |
| **Config Engine** | Read/write/validate CS2 config files (`server.cfg`, `gamemode_*.cfg`, `mapcycle.txt`) |
| **Workshop Manager** | SteamCMD integration for server install, updates, workshop map/skin downloads |
| **Monitor Collector** | OS-level metrics (CPU, RAM, disk, network) + CS2-level metrics (tick rate, player count, var) |
| **Scheduler** | Cron-like tasks — auto-restart, map rotation, scheduled updates, backups |
| **File Manager** | Secure sandboxed file browser for CS2 game directories |
| **Benchmark Engine** | Automated stress tests with bots to measure server tick rate and frame time under load |
| **Headless Web Server** | Optional REST API + WebSocket + static file server for remote access |
| **Notification Service** | Windows toast notifications, Discord webhooks, generic webhooks |

---

## 3. Tech Stack

### 3.1 Backend — Go 1.23+

**Why Go over Rust, C# (.NET / WPF), Node.js, or Java:**

| Criteria | Go | Rust | C# (.NET 8) | Node.js (Electron) |
|---|---|---|---|---|
| Latency (p99) | ~0.5 ms | ~0.3 ms | ~2 ms | ~5 ms |
| Memory footprint | ~20 MB | ~10 MB | ~60 MB | ~150 MB |
| Cross-compile | ✅ one command | ✅ complex | Windows only (WPF) | ❌ needs runtime |
| Development speed | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ |
| System-level ops | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐ |
| Concurrency model | Goroutines (M:N) | async + threads | Task-based | Event loop |
| Single `.exe` deploy | ✅ (Wails) | ✅ (Tauri) | ✅ (WPF) | ❌ (~150 MB) |
| Desktop framework | **Wails v2** | Tauri v2 | WPF / MAUI | Electron |
| Installer size | **~15–25 MB** | ~8–15 MB | ~40 MB | ~150 MB |

**Go + Wails delivers the optimal balance**: near-Rust performance, drastically simpler development than Rust/Tauri, true single-binary `.exe` deployment via WebView2, and first-class concurrency via goroutines — critical for managing multiple RCON connections, process monitoring, and real-time UI updates simultaneously. Unlike C#/WPF (WindowsGSM's stack), Go cross-compiles to Linux for headless mode with zero code changes.

#### Core Go Libraries

| Purpose | Library | Why |
|---|---|---|
| Desktop framework | [**wailsapp/wails** v2](https://wails.io/) | Go + WebView2, native window, system tray, single .exe |
| Database ORM | [`go-gorm/gorm`](https://gorm.io/) | Auto-migration, multi-driver (SQLite/PG/MySQL) |
| SQLite driver | [`glebarez/sqlite`](https://github.com/glebarez/sqlite) | Pure Go, no CGO — true cross-compile, embedded in .exe |
| Config | [`spf13/viper`](https://github.com/spf13/viper) | YAML/TOML/ENV, hot-reload, defaults |
| Logging | [`rs/zerolog`](https://github.com/rs/zerolog) | Zero-allocation structured JSON logging |
| Process mgmt | `os/exec` (stdlib) | Native process spawning, stdin/stdout/stderr pipes |
| OS metrics | [`shirou/gopsutil` v4](https://github.com/shirou/gopsutil) | Cross-platform CPU/RAM/Disk/Net metrics |
| Scheduler | [`go-co-op/gocron` v2](https://github.com/go-co-op/gocron) | Cron expressions, timezone-aware |
| Validation | [`go-playground/validator` v10](https://github.com/go-playground/validator) | Struct tag validation |
| HTTP server (headless) | [`gofiber/fiber` v3](https://github.com/gofiber/fiber) | Fasthttp-based — only used in headless/remote mode |
| WebSocket (headless) | [`gorilla/websocket`](https://github.com/gorilla/websocket) | Only used in headless/remote mode |
| JWT auth (headless) | [`golang-jwt/jwt` v5](https://github.com/golang-jwt/jwt) | Only used in headless/remote mode |
| Embed assets | `embed` (stdlib) | Embed frontend build into Go binary |
| Windows service | [`kardianos/service`](https://github.com/kardianos/service) | Run as Windows Service or Linux systemd unit |
| Auto-updater | [`rhysd/go-github-selfupdate`](https://github.com/rhysd/go-github-selfupdate) | GitHub Releases-based self-update |
| Toast notifications | [`go-toast/toast`](https://github.com/go-toast/toast) | Windows 10/11 native toast notifications |
| Archive | `archive/tar`, `archive/zip` (stdlib) | Backup compression |

### 3.2 Frontend — React 19 + Vite (in WebView2)

The UI is a React SPA bundled with Vite (not Next.js — no server-side rendering needed inside a desktop app). Wails embeds it into the `.exe` and renders it via Microsoft's WebView2 runtime (Chromium-based, already installed on Windows 10/11).

| Purpose | Technology | Why |
|---|---|---|
| Framework | **React 19** | Concurrent rendering, Suspense, huge ecosystem |
| Bundler | **Vite 6** | Sub-second HMR, tiny production bundles, Wails native support |
| Language | **TypeScript 5.5+** | Full type safety; Wails auto-generates TS types from Go structs |
| Styling | **TailwindCSS 4** | Utility-first, zero runtime CSS, dark mode built-in |
| Component lib | **shadcn/ui** | Accessible, copy-paste components, customizable themes |
| Charts | **Recharts** | Real-time performance graphs with smooth animations |
| State mgmt | **Zustand** | Minimal boilerplate, excellent for real-time state from Go events |
| Data fetching | **TanStack Query v5** | Cache, auto-refetch — wraps Wails Go bindings instead of HTTP |
| Router | **React Router v7** | Client-side routing within the desktop app |
| Forms | **React Hook Form + Zod** | Type-safe validation, shared schemas |
| Terminal emulator | **xterm.js** | Full RCON terminal emulation |
| Code/config editor | **Monaco Editor** (lightweight) | Syntax highlighting for CS2 cfg/vdf files |
| Icons | **Lucide React** | Consistent, tree-shakable icon set |
| Toast notifications | **Sonner** | In-app toast notifications for server events |
| Drag and drop | **dnd-kit** | Map rotation drag-and-drop reordering |
| Tables | **TanStack Table v8** | Sortable, filterable player lists and ban tables |

### 3.3 Go ↔ JavaScript Communication (Wails Bindings)

This is the **key architectural difference** from a web app. Instead of REST APIs + WebSocket, the frontend calls Go functions **directly** through Wails' binding layer. Wails auto-generates TypeScript wrappers for every exposed Go method.

```
┌──────────────────────────────────────────────────────────┐
│               Wails Binding Architecture                  │
│                                                          │
│   Go (backend)                   TypeScript (frontend)   │
│   ────────────                   ─────────────────────   │
│                                                          │
│   // Go struct exposed to JS     // Auto-generated TS    │
│   type App struct {}             // wailsjs/go/App.ts    │
│                                                          │
│   func (a *App) GetInstances()   App.GetInstances()      │
│     → []Instance                   → Promise<Instance[]> │
│                                                          │
│   func (a *App) StartServer(id)  App.StartServer(id)     │
│     → error                        → Promise<void>       │
│                                                          │
│   func (a *App) SendRCON(        App.SendRCON(           │
│     id, cmd string) → string       id, cmd) → Promise    │
│                                                          │
│   // Go emits events to JS       // JS listens           │
│   runtime.EventsEmit(ctx,        EventsOn("metrics",     │
│     "metrics", data)               (data) => update())   │
│                                                          │
│   // JS emits events to Go       // Go listens           │
│   EventsEmit("user:action")      runtime.EventsOn(ctx,   │
│                                    "user:action", fn)     │
│                                                          │
│   ⚡ Direct IPC — no HTTP,        ⚡ Auto-generated TS    │
│      no JSON marshaling overhead,    types from Go structs│
│      no network stack                                    │
└──────────────────────────────────────────────────────────┘
```

**Latency comparison:**

| Communication Method | Round-trip Latency | Used When |
|---|---|---|
| Wails binding (direct IPC) | **< 0.1 ms** | Desktop mode (local UI) |
| REST over localhost | ~1–5 ms | Headless mode (remote browser) |
| WebSocket over localhost | ~0.5–2 ms | Headless mode (real-time) |
| WebSocket over network | ~10–100 ms | Remote access |

### 3.4 Database Layer

```
┌─────────────────────────────────────────┐
│           Data Storage Strategy         │
├─────────────────────────────────────────┤
│                                         │
│  SQLite (embedded, default)             │
│  ├─ Server instance configs             │
│  ├─ Audit logs                          │
│  ├─ Ban lists                           │
│  ├─ Scheduled tasks                     │
│  ├─ Benchmark history                   │
│  ├─ Workshop items                      │
│  ├─ Config profiles                     │
│  └─ App settings & preferences          │
│                                         │
│  Location:                              │
│  %APPDATA%\CS2Admin\cs2admin.db         │
│                                         │
│  Filesystem                             │
│  ├─ CS2 game files (maps, configs)      │
│  ├─ Backup archives (.zip)              │
│  ├─ SteamCMD cache                      │
│  └─ Application logs                    │
│                                         │
│  Location:                              │
│  %APPDATA%\CS2Admin\logs\               │
│  %APPDATA%\CS2Admin\backups\            │
│                                         │
└─────────────────────────────────────────┘
```

**SQLite only** — no PostgreSQL, no Redis. A desktop app managing game servers on the local machine doesn't need multi-node databases. SQLite handles thousands of writes/sec — more than enough for audit logs, config changes, and metric snapshots. This keeps the app truly zero-dependency.

### 3.5 Communication Protocols

| Channel | Protocol | Format | Latency |
|---|---|---|---|
| UI ↔ Backend (desktop) | **Wails IPC** (in-process) | Go structs ↔ TS types | **< 0.1 ms** |
| UI ↔ Backend (headless) | HTTP REST + WebSocket | JSON | ~1–5 ms (local), ~10–100 ms (remote) |
| Backend ↔ CS2 server | **Source RCON** (TCP) | Binary (int32 LE packets) | < 20 ms |
| Backend ↔ SteamCMD | stdin/stdout pipe | Text stream | N/A (batch) |
| Backend ↔ OS metrics | syscall / WMI | Native | < 5 ms sampling |
| Backend → Windows | Toast notification API | XML | Instant |
| Backend → Discord | HTTPS webhook | JSON | ~200 ms |

---

## 4. Desktop Application Framework

### 4.1 Why Wails v2

| Feature | Wails v2 (Go) | Tauri v2 (Rust) | Electron (Node) | WPF (C#) |
|---|---|---|---|---|
| Backend language | **Go** | Rust | Node.js | C# |
| Renderer | WebView2 (Chromium) | WebView2/WebKitGTK | Bundled Chromium | DirectX/XAML |
| Installer size | **~15–25 MB** | ~8–15 MB | ~150 MB+ | ~40 MB |
| RAM usage (idle) | **~50–80 MB** | ~30–60 MB | ~150–300 MB | ~80–120 MB |
| Go↔JS IPC speed | **< 0.1 ms** | ~0.1 ms (Rust↔JS) | ~1 ms | N/A (native) |
| System tray | ✅ | ✅ | ✅ | ✅ |
| Native dialogs | ✅ | ✅ | ✅ | ✅ |
| Auto-updater | ✅ (via lib) | ✅ (built-in) | ✅ (built-in) | Manual |
| Windows Service mode | ✅ (Go lib) | ✅ (Rust lib) | ❌ | ✅ |
| Linux headless mode | **✅ (same binary)** | ✅ | ❌ | ❌ |
| WebView2 requirement | Preinstalled Win10/11 | Same | None (bundles) | None (native) |
| TS type generation | **✅ (auto from Go)** | ✅ (auto from Rust) | Manual | N/A |
| Development speed | **⭐⭐⭐⭐⭐** | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ |

**Wails wins for this project because:**
1. Go is the ideal backend language for game server management (see Section 3.1)
2. Wails compiles to a single `.exe` with frontend embedded — same experience as WindowsGSM
3. The same Go binary runs headless on Linux with zero code changes
4. Auto-generated TypeScript types from Go structs eliminate API contract drift
5. WebView2 is preinstalled on Windows 10/11 — no extra download needed
6. Sub-0.1ms IPC means the UI feels as responsive as a native app

### 4.2 Native Desktop Features

| Feature | Implementation |
|---|---|
| **System tray** | App minimizes to tray; right-click menu: Show, Start All, Stop All, Exit |
| **Windows toast notifications** | Server crash, player join/leave, SteamCMD update available, benchmark complete |
| **Startup with Windows** | Optional: registry entry to launch CS2Admin on login |
| **Native file dialogs** | Open/Save dialogs for backup import/export, config import |
| **Window state persistence** | Remember size, position, maximized state between sessions |
| **Dark / Light theme** | Follows Windows system theme by default, manual override available |
| **Drag and drop** | Drop `.cfg` files onto the window to import configs |
| **Single instance lock** | Prevent multiple copies of CS2Admin from running simultaneously |
| **Context menu** | Right-click on instances for quick actions |
| **Keyboard shortcuts** | Ctrl+1–9 switch instances, Ctrl+R RCON focus, Ctrl+S save config, etc. |
| **Taskbar progress** | Show SteamCMD download/update progress on the Windows taskbar icon |

### 4.3 Window Layout (Conceptual)

```
┌──────────────────────────────────────────────────────────────────────┐
│  CS2 Admin Panel                                         ─ □ ✕      │
├────────────┬─────────────────────────────────────────────────────────┤
│            │  Dashboard  │  Console  │  Config  │  Maps  │  Players │
│  Instances ├─────────────────────────────────────────────────────────┤
│            │                                                         │
│  ► Server1 │  ┌─────────────────────┐  ┌──────────────────────────┐ │
│    Running │  │  CPU  ████░░ 62%    │  │  Server Info             │ │
│            │  │  RAM  ███░░░ 48%    │  │  Name: My CS2 Server     │ │
│  ► Server2 │  │  Tick ██████ 128/s  │  │  Map:  de_dust2          │ │
│    Stopped │  │  Net  ██░░░░ 24Mbps │  │  Players: 8/10           │ │
│            │  └─────────────────────┘  │  Uptime: 3h 24m          │ │
│  ► Server3 │                            └──────────────────────────┘ │
│    Updating│  ┌─────────────────────────────────────────────────────┐│
│            │  │  Players Online                                     ││
│  + New     │  │  ┌──────┬───────────┬──────┬───────┬─────────────┐ ││
│            │  │  │ Name │ SteamID   │ Ping │ Score │ Actions     │ ││
│ ────────── │  │  ├──────┼───────────┼──────┼───────┼─────────────┤ ││
│ Settings   │  │  │ p1   │ STEAM_0:1 │ 24ms │  18   │ Kick Ban Mute││
│ About      │  │  │ p2   │ STEAM_0:0 │ 31ms │  12   │ Kick Ban Mute││
│            │  │  └──────┴───────────┴──────┴───────┴─────────────┘ ││
│            │  └─────────────────────────────────────────────────────┘│
│            │                                                         │
│            │  [ Start ] [ Stop ] [ Restart ] [ Update ] [ Backup ]  │
└────────────┴─────────────────────────────────────────────────────────┘
```

---

## 5. Feature Requirements

### 5.1 Instance Management

| Feature | Description | Priority |
|---|---|---|
| Multi-instance | Run multiple CS2 servers on different ports simultaneously | P0 |
| One-click install | SteamCMD-based CS2 server download and setup wizard | P0 |
| Start / Stop / Restart | Process lifecycle with graceful shutdown (SIGTERM → SIGKILL timeout) | P0 |
| Auto-restart on crash | Watchdog monitors process, auto-restarts with configurable backoff | P0 |
| Console output streaming | Real-time stdout/stderr of CS2 process to embedded terminal | P0 |
| Scheduled restarts | Cron-based restart with player warning messages via RCON `say` | P1 |
| Update management | One-click SteamCMD update with taskbar progress bar | P0 |
| Startup parameters | Full control over CS2 launch arguments (`-dedicated`, `-port`, `-maxplayers`, etc.) | P0 |
| Instance cloning | Duplicate a configured instance with all settings | P1 |
| Instance templates | Pre-configured templates (Competitive, Casual, DM, Retake, Surf, KZ) | P1 |
| Start with Windows | Auto-start selected instances on Windows login | P1 |
| Minimize to tray | Keep running in background when window is closed | P0 |

### 5.2 RCON & Console

| Feature | Description | Priority |
|---|---|---|
| Embedded RCON terminal | xterm.js-based interactive RCON console within the app | P0 |
| Command history | Persistent per-instance command history with up/down arrow navigation | P0 |
| Command autocomplete | Suggest CS2 cvars/commands as you type (Tab completion) | P1 |
| Quick actions | Toolbar buttons for common commands (changelevel, kick, ban, restart round) | P0 |
| Command macros | User-defined command sequences (e.g., "Warm-up config" = 5 commands) | P1 |
| Multi-instance broadcast | Send RCON command to all running instances simultaneously | P1 |
| Console log viewer | Filterable, searchable server console log with severity highlighting | P0 |
| Copy to clipboard | Right-click → Copy on console output | P0 |

### 5.3 Map Management

| Feature | Description | Priority |
|---|---|---|
| Map browser | Visual grid/list of all installed maps with thumbnails | P0 |
| Workshop integration | Search, download, and manage Steam Workshop maps via SteamCMD | P0 |
| Workshop collections | Import entire workshop collections by ID | P1 |
| Map rotation editor | Drag-and-drop `mapcycle.txt` editor using dnd-kit | P0 |
| Map voting config | Configure in-game map vote settings | P1 |
| Quick map change | One-click changelevel from map browser | P0 |
| Map groups | Organize maps into custom groups (Competitive, Casual, Surf, etc.) | P1 |
| Auto-download missing maps | Detect and auto-download maps players request | P2 |

### 5.4 Skin & Workshop Item Management

| Feature | Description | Priority |
|---|---|---|
| Workshop skin browser | Browse and install weapon skins from Workshop | P1 |
| Custom skin packs | Upload and manage custom skin bundles via native file dialog | P2 |
| Skin plugin integration | Auto-configure popular skin plugins (e.g., `wp_skin_changer`) | P1 |
| Knife/glove models | Manage custom knife and glove model replacements | P2 |
| Agent skins | Configure custom player model/agent options | P2 |
| Sound packs | Manage custom sound replacements (MVP, kills) | P2 |

### 5.5 Player Management

| Feature | Description | Priority |
|---|---|---|
| Live player list | Real-time sortable table: name, SteamID, ping, score, team, IP | P0 |
| Kick / Ban / Mute | One-click player moderation with reason dialog | P0 |
| Ban list management | Persistent ban database with expiry, SteamID, IP bans | P0 |
| VIP/Admin system | In-game privilege tiers (admin, mod, VIP) per SteamID | P1 |
| Player statistics | Per-player stats history (kills, deaths, playtime, headshot %) | P1 |
| Whitelist mode | Restrict server to whitelisted SteamIDs only | P1 |
| Player notes | Attach admin notes to player profiles | P2 |
| GeoIP display | Show player country/region flag based on IP (embedded GeoLite2 DB) | P2 |

### 5.6 Bot Management

| Feature | Description | Priority |
|---|---|---|
| Bot count control | Set min/max bots, auto-fill rules | P0 |
| Bot difficulty | Per-bot or global difficulty (Easy → Expert) | P0 |
| Bot quota mode | `fill`, `match`, `normal` quota modes | P0 |
| Bot weapon preferences | Restrict bot weapon loadouts | P1 |
| Named bot profiles | Custom bot names and skill configurations | P1 |
| Bot practice modes | Pre-configured bot scenarios for aim/spray practice | P2 |
| Bot navigation meshes | Upload/manage custom nav meshes for workshop maps | P2 |

### 5.7 Server Configuration

| Feature | Description | Priority |
|---|---|---|
| Visual config editor | GUI form for all major cvars with descriptions, tooltips, and validation | P0 |
| Raw config editor | Monaco Editor for `server.cfg`, `gamemode_*.cfg` with CS2 syntax highlighting | P0 |
| Config profiles | Save/load named configuration profiles (Competitive, Casual, Practice) | P0 |
| Game mode presets | One-click switch between game modes with correct cvars | P0 |
| Cvar search | Searchable database of all CS2 cvars with descriptions and defaults | P1 |
| Config diff viewer | Compare two configs side-by-side | P1 |
| Config import/export | JSON export of full server configuration for sharing (via Save dialog) | P1 |
| Rate settings | Advanced network rate configuration (tickrate, cmdrate, updaterate) | P0 |
| Plugin manager | Install/enable/disable CounterStrikeSharp and Metamod plugins | P1 |

### 5.8 Performance & Benchmarking

| Feature | Description | Priority |
|---|---|---|
| Live dashboard | Real-time animated graphs: CPU, RAM, Network I/O, Disk I/O | P0 |
| CS2 metrics | Server tick rate (actual vs target), `sv` and `var` values, entity count | P0 |
| Player load graph | Concurrent player count over time | P0 |
| Network quality | Bandwidth per player, packet loss, choke | P1 |
| Benchmark suite | Automated stress test: spawn N bots, measure tick rate degradation | P1 |
| Benchmark history | Store and compare benchmark results over time (graphs) | P1 |
| Resource alerts | Configurable thresholds → Windows toast notification when exceeded | P1 |
| Performance advisor | Auto-detect bottlenecks and suggest cvar/hardware changes | P2 |
| Historical analytics | Long-term trends: peak hours, average load, uptime percentage | P2 |
| Taskbar metrics | Optional: show tick rate / player count on hover over tray icon | P2 |

### 5.9 Application Settings

| Feature | Description | Priority |
|---|---|---|
| Theme | Dark / Light / Follow System | P0 |
| Language | English (default), extensible i18n | P2 |
| Start with Windows | Toggle auto-launch on login | P1 |
| Minimize to tray | Toggle close-to-tray behavior | P0 |
| Default Steam path | Configure SteamCMD and Steam installation paths | P0 |
| GSLT management | Store and assign Game Server Login Tokens | P0 |
| Notification preferences | Toggle Windows toast / Discord / webhook per event type | P1 |
| Backup storage path | Configure where backups are stored | P1 |
| Auto-update | Toggle automatic app updates from GitHub Releases | P1 |
| Keyboard shortcuts | Customizable keybindings | P2 |

### 5.10 Backup & Recovery

| Feature | Description | Priority |
|---|---|---|
| Manual backup | One-click full instance backup (configs + maps + plugins) → .zip | P0 |
| Scheduled backups | Cron-based auto-backups with retention policies | P1 |
| Backup restore | One-click restore from backup archive | P0 |
| Selective restore | Restore only configs, only maps, or only plugins | P1 |
| Remote backup | Upload backups to S3-compatible storage | P2 |
| Backup to custom path | Choose folder via native file dialog | P0 |

### 5.11 Notifications & Integrations

| Feature | Description | Priority |
|---|---|---|
| Windows toast notifications | Native Windows 10/11 notifications for server events | P0 |
| Discord webhooks | Server events → Discord channel (start, stop, crash, player join/leave) | P1 |
| Generic REST webhook | Webhook for any event (custom integrations) | P1 |
| Telegram bot | Optional Telegram notifications | P2 |
| Email notifications | Alert emails for critical events (crash, threshold breach) | P2 |
| Gametracker/HLSW | Server query protocol support for server listing sites | P2 |

### 5.12 Remote Access (Headless Mode)

| Feature | Description | Priority |
|---|---|---|
| Headless web server | Same React UI served over HTTPS for remote browsers | P1 |
| User authentication | Email/password + optional 2FA (TOTP) for remote access | P1 |
| Role-Based Access Control | Roles: Super Admin, Admin, Moderator, Viewer | P1 |
| API keys | Scoped API keys for external integrations / scripts | P2 |
| Audit log | Every action logged: who, what, when, IP address | P1 |
| REST API | Full REST API for automation (same endpoints as headless web server) | P1 |

---

## 6. CS2 Server Integration

### 6.1 Source RCON Protocol

The backend maintains **persistent TCP connections** to each CS2 instance using the Source RCON Protocol.

**Packet Structure (Little-Endian):**

```
┌────────────────────────────────────────┐
│  Field        │  Type    │  Bytes      │
├────────────────────────────────────────┤
│  Size         │  int32   │  4          │  ← Size of remaining packet
│  Request ID   │  int32   │  4          │  ← Client-set correlation ID
│  Type         │  int32   │  4          │  ← 3=Auth, 2=ExecCmd, 0=Response
│  Body         │  string  │  variable   │  ← Null-terminated ASCII
│  Padding      │  byte    │  1          │  ← Empty null terminator
└────────────────────────────────────────┘
```

**Connection Lifecycle:**

```
1. TCP Connect → 127.0.0.1:RCON_PORT (localhost — same machine)
2. Send Auth Packet (type=3, body=rcon_password)
3. Receive Auth Response (type=2, request_id matches = success, -1 = fail)
4. Send Command Packets (type=2, body="status", "changelevel de_dust2", etc.)
5. Receive Response Packets (type=0, body=command output)
6. Keep-alive pings every 30s to prevent timeout
7. Auto-reconnect on TCP disconnect with exponential backoff
```

**Implementation Requirements:**
- Connection pool: one persistent TCP connection per CS2 instance
- Request-response correlation via `request_id`
- Multi-packet response handling (for responses > 4096 bytes)
- Thread-safe command queue per connection (goroutine + channel)
- Configurable timeout per command (default 5s)
- All connections are localhost — sub-1ms network latency

### 6.2 SteamCMD Integration

```
SteamCMD Wrapper
├── Auto-install SteamCMD
│   └── Download + extract to %APPDATA%\CS2Admin\steamcmd\
├── Install CS2 Server
│   └── steamcmd +force_install_dir <path> +login <user> +app_update 730 +quit
├── Update CS2 Server
│   └── steamcmd +force_install_dir <path> +login <user> +app_update 730 +quit
├── Download Workshop Map
│   └── steamcmd +login <user> +workshop_download_item 730 <map_id> +quit
├── Validate Files
│   └── steamcmd +force_install_dir <path> +login <user> +app_update 730 validate +quit
└── Progress Tracking
    ├── Parse stdout for download percentage, ETA
    └── Push to UI via Wails event → frontend progress bar + taskbar progress
```

**Requirements:**
- SteamCMD auto-download and self-update on first run
- Steam Guard 2FA handling (prompt user via native dialog in app)
- Download progress parsing → real-time progress bar in UI + Windows taskbar
- Concurrent downloads with queue management
- Login session caching (Steam token reuse)

### 6.3 CS2 Server File Layout

```
<instance_root>/
├── game/
│   └── csgo/
│       ├── cfg/
│       │   ├── server.cfg                    ← Main server config
│       │   ├── gamemode_competitive_server.cfg
│       │   ├── gamemode_casual_server.cfg
│       │   ├── gamemode_deathmatch_server.cfg
│       │   ├── gamemode_wingman_server.cfg
│       │   ├── gamemode_custom_server.cfg
│       │   └── autoexec.cfg                  ← Auto-executed on start
│       ├── maps/                             ← BSP map files
│       ├── addons/
│       │   ├── metamod/                      ← Metamod:Source
│       │   └── counterstrikesharp/           ← CSS plugins
│       │       ├── plugins/
│       │       └── configs/
│       ├── mapcycle.txt                      ← Map rotation
│       └── gamemodes_server.txt              ← Workshop map groups
├── cs2.exe                                   ← Server binary (Windows)
└── steam_appid.txt
```

### 6.4 CS2 Launch Parameters

```powershell
cs2.exe -dedicated                    # Run as dedicated server
        -port 27015                   # Game port
        -ip 0.0.0.0                   # Bind IP
        -maxplayers 10                # Max player slots
        +game_type 0                  # 0=Classic, 1=GunGame, etc.
        +game_mode 1                  # 0=Casual, 1=Competitive, 2=Wingman
        +map de_dust2                 # Starting map
        +mapgroup mg_active           # Map group
        +sv_setsteamaccount <GSLT>    # Game Server Login Token
        -tickrate 128                 # Server tick rate (if supported)
        +rcon_password <pass>         # RCON password
        +sv_password <pass>           # Server join password
        -console                      # Enable console
        -usercon                      # Enable RCON
        +sv_lan 0                     # 0=Internet, 1=LAN only
        -authkey <API_KEY>            # Steam Web API key for workshop
```

---

## 7. Data Models

### Core Entities (ERD)

```
┌─────────────────┐     ┌───────────────────┐     ┌──────────────────┐
│  AppSettings     │     │  ServerInstance    │     │  Backup          │
├─────────────────┤     ├───────────────────┤     ├──────────────────┤
│  id         int  │     │  id          UUID │────►│  id         UUID │
│  key      string │     │  name       string│     │  instance_id UUID│
│  value      text │     │  port         int │     │  path      string│
│  updated_at time │     │  rcon_port    int │     │  size_bytes  int │
└─────────────────┘     │  status      enum │     │  type       enum │
                         │  game_mode  string│     │  created_at time │
                         │  max_players  int │     └──────────────────┘
                         │  map        string│
┌─────────────────┐     │  install_path str │     ┌──────────────────┐
│  AuditLog        │     │  launch_args  text│     │  BenchmarkResult │
├─────────────────┤     │  rcon_pass   (enc)│     ├──────────────────┤
│  id         UUID │     │  gslt_token  (enc)│     │  id         UUID │
│  action   string │     │  auto_restart bool│     │  instance_id UUID│
│  target   string │     │  auto_start  bool │     │  bot_count    int│
│  details    JSON │     │  created_at  time │     │  avg_tickrate flt│
│  created_at time │     │  updated_at  time │     │  min_tickrate flt│
└─────────────────┘     └───────────────────┘     │  avg_frametime flt│
                                                    │  cpu_usage    flt│
┌─────────────────┐     ┌───────────────────┐     │  ram_usage    flt│
│  BanEntry        │     │  ConfigProfile    │     │  duration_sec int│
├─────────────────┤     ├───────────────────┤     │  created_at  time│
│  id         UUID │     │  id          UUID │     └──────────────────┘
│  instance_id UUID│     │  instance_id UUID │
│  steam_id  string│     │  name       string│     ┌──────────────────┐
│  ip_address str  │     │  data         JSON│     │  ScheduledTask   │
│  reason    string│     │  is_active   bool │     ├──────────────────┤
│  expires_at time │     │  created_at  time │     │  id         UUID │
│  is_permanent bool│     └───────────────────┘     │  instance_id UUID│
│  created_at time │                               │  cron_expr string│
└─────────────────┘     ┌───────────────────┐     │  action     enum │
                         │  WorkshopItem     │     │  payload     JSON│
┌─────────────────┐     ├───────────────────┤     │  enabled     bool│
│  CommandMacro    │     │  id          UUID │     │  last_run   time │
├─────────────────┤     │  instance_id UUID │     │  next_run   time │
│  id         UUID │     │  workshop_id  int │     └──────────────────┘
│  name      string│     │  title      string│
│  commands    JSON│     │  type        enum │     ┌──────────────────┐
│  hotkey    string│     │  file_size    int │     │  MetricSnapshot  │
│  created_at time │     │  installed   bool │     ├──────────────────┤
└─────────────────┘     │  created_at  time │     │  id         int  │
                         └───────────────────┘     │  instance_id UUID│
                                                    │  cpu_pct     flt │
                                                    │  ram_mb      flt │
                                                    │  tick_rate   flt │
                                                    │  players      int│
                                                    │  net_in_kbps flt │
                                                    │  net_out_kbps flt│
                                                    │  timestamp   time│
                                                    └──────────────────┘
```

### Enums

```
ServerStatus:   stopped | starting | running | stopping | crashed | updating | installing
GameMode:       competitive | casual | deathmatch | wingman | custom | retake | surf | kz
WorkshopType:   map | skin | model | sound | plugin
BackupType:     full | config_only | maps_only | plugins_only
TaskAction:     restart | update | backup | rcon_command | map_change
```

---

## 8. Real-Time Data Pipeline

### 8.1 Desktop Mode — Wails Events

In desktop mode, real-time data flows through **Wails event emission** — direct in-process pub/sub with < 0.1 ms latency.

```
┌──────────────────────────────────────────────────────────────────┐
│                    Wails Event Bus (in-process)                   │
│                                                                  │
│   Go goroutine: Monitor             →  runtime.EventsEmit(ctx,  │
│     (1s tick: CPU, RAM, tick rate)       "metrics:{instanceID}", │
│                                          metricsData)            │
│                                                                  │
│   Go goroutine: RCON stdout pipe    →  runtime.EventsEmit(ctx,  │
│     (real-time CS2 console output)      "console:{instanceID}", │
│                                          logLine)                │
│                                                                  │
│   Go goroutine: Player poller       →  runtime.EventsEmit(ctx,  │
│     (3s tick: RCON "status")            "players:{instanceID}", │
│                                          playerList)             │
│                                                                  │
│   Go goroutine: Process watchdog    →  runtime.EventsEmit(ctx,  │
│     (crash/stop detection)              "status:{instanceID}",  │
│                                          newStatus)              │
│                                                                  │
│   Go goroutine: SteamCMD            →  runtime.EventsEmit(ctx,  │
│     (download progress)                 "progress:{instanceID}",│
│                                          percent)                │
│                                                                  │
│   ──────────────────────────────────────────────────────────     │
│                                                                  │
│   React UI (TypeScript):                                         │
│     EventsOn("metrics:abc-123", (data) => setMetrics(data))     │
│     EventsOn("console:abc-123", (line) => appendLine(line))     │
│     EventsOn("players:abc-123", (list) => setPlayers(list))     │
│     EventsOn("status:abc-123", (s) => setStatus(s))             │
│     EventsOn("progress:abc-123", (p) => setProgress(p))         │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

### 8.2 Headless Mode — WebSocket

In headless mode (remote access), the same events are bridged to WebSocket channels.

```
┌────────────────────────────────────────┐
│     WebSocket Hub (Go Fiber)           │
│                                        │
│  Channel: instance:{id}:metrics  (1s)  │◄── Monitor goroutine
│  Channel: instance:{id}:console  (RT)  │◄── RCON stdout pipe
│  Channel: instance:{id}:players  (3s)  │◄── RCON "status" poll
│  Channel: instance:{id}:status   (RT)  │◄── Process watchdog
│  Channel: global:notifications   (RT)  │◄── System events
│                                        │
└────────────────────────────────────────┘
```

### 8.3 Event Data Format

```typescript
// Metrics event payload (emitted every 1s per instance)
interface MetricsEvent {
  instance_id: string;
  timestamp: number;
  cpu_percent: number;
  ram_mb: number;
  tick_rate: number;
  sv: number;
  var: number;
  players: number;
  max_players: number;
  map: string;
  uptime_seconds: number;
  net_in_kbps: number;
  net_out_kbps: number;
}

// Console event payload (emitted per line from CS2 stdout)
interface ConsoleEvent {
  instance_id: string;
  timestamp: number;
  line: string;
  level: "info" | "warning" | "error";
}

// Status change event
interface StatusEvent {
  instance_id: string;
  old_status: ServerStatus;
  new_status: ServerStatus;
  message?: string;
}
```

---

## 9. Security

### 9.1 Local Desktop Mode

Since the app runs locally on the same machine as the CS2 servers, the security model is simpler than a web app:

| Concern | Approach |
|---|---|
| No network auth needed | The `.exe` runs locally — whoever has access to the Windows user account has access |
| RCON passwords | Encrypted at rest in SQLite using AES-256-GCM with machine-derived key |
| GSLT tokens | Same encryption as RCON passwords |
| Steam credentials | Stored as encrypted SteamCMD login tokens (Steam's own mechanism) |
| File access | Sandboxed to CS2 instance directories — no arbitrary path traversal |
| Single instance lock | Named mutex prevents running multiple copies |
| Audit log | All actions logged locally for accountability |

### 9.2 Headless / Remote Access Mode

When remote access is enabled, full web security applies:

| Layer | Implementation |
|---|---|
| HTTPS | TLS 1.3 enforced; auto-cert via Let's Encrypt or self-signed |
| Password hashing | **Argon2id** (memory-hard, GPU-resistant) |
| Session tokens | **JWT** (RS256) — 15 min access + 7 day refresh |
| 2FA | TOTP (RFC 6238) — Google Authenticator / Authy compatible |
| RBAC | Roles: Super Admin, Admin, Moderator, Viewer |
| Rate limiting | Per-IP token bucket (configurable) |
| WebSocket auth | JWT validated on handshake upgrade |
| CORS | Strict origin whitelist |
| CSRF | SameSite cookie + custom header |
| Brute-force protection | Account lockout after N failed attempts |
| API keys | SHA-256 hashed, scoped to specific endpoints/instances |

### 9.3 Process Isolation

```
Default mode (Windows):
  └── Each CS2 instance runs as a child process of CS2Admin.exe
      ├── Process priority management (Normal / Below Normal / Low)
      ├── CPU affinity pinning (optional — assign specific cores)
      └── Monitored via WMI for resource usage

Docker mode (optional — Linux headless):
  └── Full container isolation per instance
      ├── CPU/RAM limits per container
      ├── Network namespace isolation
      └── Managed via Docker Engine API
```

---

## 10. Distribution & Installation

### 10.1 Windows (Primary)

```
Distribution formats:
├── CS2Admin-Setup-x64.exe          ← NSIS/Inno Setup installer (~20 MB)
│   ├── Installs to C:\Program Files\CS2Admin\
│   ├── Creates Start Menu shortcut
│   ├── Creates Desktop shortcut (optional)
│   ├── Registers uninstaller in Add/Remove Programs
│   └── Optionally installs WebView2 runtime (if not present)
│
├── CS2Admin-Portable-x64.zip       ← Portable version (~15 MB)
│   └── Extract anywhere and run CS2Admin.exe
│
└── winget install cs2admin          ← WinGet package (future)
```

### 10.2 Linux (Headless Mode)

```
Distribution formats:
├── cs2admin-linux-amd64.tar.gz      ← Portable binary
│   └── ./cs2admin --headless --port 8443
│
├── cs2admin_amd64.deb               ← Debian/Ubuntu package
│   └── systemctl enable --now cs2admin
│
└── cs2admin_amd64.rpm               ← RHEL/Fedora package
    └── systemctl enable --now cs2admin
```

### 10.3 Auto-Updater

```
Update flow:
1. On startup (or periodically), query GitHub Releases API
2. Compare current version vs latest release tag
3. If update available → show notification in app
4. User clicks "Update" → download new binary in background
5. Verify SHA-256 checksum
6. On next restart, replace binary and relaunch
   (Windows: use rename-on-reboot for locked .exe)
```

### 10.4 System Requirements

| Component | Minimum | Recommended |
|---|---|---|
| OS | Windows 10 (v1809+) or Windows 11 | Windows 11 |
| WebView2 Runtime | Required (preinstalled on Win10 21H2+ and Win11) | Auto-installed |
| CPU | 2 cores (panel only) | 4+ cores (panel + CS2 instances) |
| RAM | 512 MB (panel) + 2 GB per CS2 instance | 1 GB (panel) + 4 GB per CS2 instance |
| Disk | 50 GB (1 CS2 instance) | SSD, 100 GB+ per instance |
| Network | 5 Mbps per instance | 10+ Mbps per instance |
| SteamCMD | Auto-downloaded by CS2Admin | — |

### 10.5 Port Allocation

```
CS2Admin App:
  No ports needed in desktop mode (all in-process)
  8443/tcp when headless/remote mode is enabled

Per CS2 Instance (auto-allocated):
  27015/tcp ← RCON (localhost only by default)
  27015/udp ← Game traffic
  27020/udp ← SourceTV (GOTV)
  27005/udp ← Client port

Auto-allocation scheme:
  Instance 1: 27015-27020
  Instance 2: 27025-27030
  Instance 3: 27035-27040
  ...
```

---

## 11. Performance Targets

| Metric | Target | How |
|---|---|---|
| App startup | **< 1.5 s** to interactive UI | Wails lazy init, embedded SQLite, pre-compiled frontend |
| Go↔UI round-trip | **< 0.1 ms** | Wails direct IPC binding (no HTTP) |
| RCON round-trip | **< 5 ms** (localhost) | Persistent TCP to 127.0.0.1 |
| Dashboard refresh | **1 Hz** (metrics), **real-time** (console) | Wails EventsEmit push from Go goroutines |
| UI frame rate | **60 fps** | React 19 concurrent rendering, CSS animations |
| App memory (idle) | **< 80 MB** (no instances) | Go ~20 MB + WebView2 ~60 MB |
| App memory (per instance) | **+ ~5 MB** per managed instance | Goroutine pool, RCON buffer pool |
| CS2 process memory | **~2–4 GB** per CS2 server (not our app) | Monitored, not controlled |
| App CPU (idle) | **< 1%** | Event-driven, no polling in desktop mode |
| App CPU (10 instances) | **< 5%** | Efficient goroutine multiplexing |
| Installer size | **< 25 MB** | Go binary (~15 MB) + embedded frontend (~5 MB) + metadata |
| Instance limit | **20+ CS2 servers** | Limited by hardware, not app |
| SteamCMD download speed | **Wire speed** | Direct SteamCMD, progress forwarded to UI |

---

## 12. Project Structure

```
CS2_admin/
├── TECHNICAL_REQUIREMENTS.md         ← This document
├── README.md                         ← User-facing documentation
├── LICENSE
├── wails.json                        ← Wails project configuration
├── go.mod                            ← Go module definition
├── go.sum
├── main.go                           ← Wails app entry point
├── app.go                            ← Main App struct (Wails bindings)
├── Makefile                          ← Build commands
│
├── internal/                         ← Go backend (internal packages)
│   ├── instance/                     ← CS2 process management
│   │   ├── manager.go                ← Instance CRUD, lifecycle orchestration
│   │   ├── process.go                ← OS process spawn, stdin/stdout piping
│   │   ├── watchdog.go               ← Crash detection, auto-restart
│   │   └── templates.go              ← Pre-configured game mode templates
│   │
│   ├── rcon/                         ← Source RCON protocol client
│   │   ├── client.go                 ← TCP connection, auth, send/receive
│   │   ├── packet.go                 ← Packet encode/decode (int32 LE)
│   │   └── pool.go                   ← Connection pool (one per instance)
│   │
│   ├── steam/                        ← SteamCMD wrapper
│   │   ├── steamcmd.go               ← Download, install, update CS2
│   │   ├── workshop.go               ← Workshop item download
│   │   └── progress.go               ← stdout progress parser
│   │
│   ├── config/                       ← CS2 config file engine
│   │   ├── parser.go                 ← Read/write server.cfg, gamemode_*.cfg
│   │   ├── validator.go              ← Cvar validation rules
│   │   ├── cvars.go                  ← CS2 cvar database (name, type, default, desc)
│   │   └── profiles.go               ← Config profile save/load
│   │
│   ├── monitor/                      ← System & CS2 metrics collection
│   │   ├── collector.go              ← Aggregator, starts/stops per-instance monitors
│   │   ├── system.go                 ← CPU, RAM, Disk, Net via gopsutil
│   │   ├── cs2metrics.go             ← Tick rate, sv, var via RCON queries
│   │   └── history.go                ← Persist metric snapshots to SQLite
│   │
│   ├── benchmark/                    ← Stress test engine
│   │   ├── runner.go                 ← Spawn bots, collect metrics, generate report
│   │   └── report.go                 ← Benchmark result analysis
│   │
│   ├── backup/                       ← Backup/restore logic
│   │   ├── backup.go                 ← Create .zip archives
│   │   └── restore.go                ← Extract and validate
│   │
│   ├── filemanager/                  ← Sandboxed file operations
│   │   └── manager.go                ← List, read, write, delete (scoped to instance dir)
│   │
│   ├── notify/                       ← Notification dispatch
│   │   ├── toast.go                  ← Windows toast notifications
│   │   ├── discord.go                ← Discord webhook
│   │   └── webhook.go                ← Generic REST webhook
│   │
│   ├── scheduler/                    ← Cron task engine
│   │   └── scheduler.go              ← gocron wrapper, task CRUD
│   │
│   ├── headless/                     ← Headless web server (optional)
│   │   ├── server.go                 ← Fiber HTTP server + static files
│   │   ├── handlers/                 ← REST API handlers
│   │   │   ├── auth.go
│   │   │   ├── instance.go
│   │   │   ├── rcon.go
│   │   │   ├── config.go
│   │   │   ├── maps.go
│   │   │   ├── players.go
│   │   │   ├── benchmark.go
│   │   │   └── files.go
│   │   ├── middleware/
│   │   │   ├── auth.go
│   │   │   ├── ratelimit.go
│   │   │   └── cors.go
│   │   └── ws/                       ← WebSocket hub (headless only)
│   │       ├── hub.go
│   │       └── client.go
│   │
│   ├── auth/                         ← Authentication (headless mode)
│   │   ├── jwt.go
│   │   ├── password.go               ← Argon2id
│   │   └── totp.go                   ← 2FA
│   │
│   ├── models/                       ← GORM models + migrations
│   │   ├── instance.go
│   │   ├── ban.go
│   │   ├── workshop.go
│   │   ├── audit.go
│   │   ├── benchmark.go
│   │   ├── config_profile.go
│   │   ├── scheduled_task.go
│   │   ├── metric_snapshot.go
│   │   ├── settings.go
│   │   └── migrate.go                ← Auto-migration on startup
│   │
│   └── pkg/                          ← Shared utilities
│       ├── logger/                   ← zerolog wrapper
│       ├── crypto/                   ← AES-256-GCM encrypt/decrypt
│       └── valve/                    ← Valve file format parsers (VDF, KeyValues)
│
├── frontend/                         ← React + Vite frontend
│   ├── src/
│   │   ├── main.tsx                  ← React entry point
│   │   ├── App.tsx                   ← Root component + router
│   │   ├── pages/
│   │   │   ├── Dashboard.tsx         ← Home: instance overview cards
│   │   │   ├── InstanceView.tsx      ← Tabbed instance detail view
│   │   │   ├── Console.tsx           ← xterm.js RCON terminal
│   │   │   ├── Config.tsx            ← Visual + raw config editor
│   │   │   ├── Maps.tsx              ← Map browser + rotation editor
│   │   │   ├── Players.tsx           ← Player list + moderation
│   │   │   ├── Bots.tsx              ← Bot configuration
│   │   │   ├── Skins.tsx             ← Skin/workshop item manager
│   │   │   ├── Benchmark.tsx         ← Benchmark runner + history
│   │   │   ├── Files.tsx             ← File browser + Monaco editor
│   │   │   ├── Backups.tsx           ← Backup list + restore
│   │   │   └── Settings.tsx          ← App preferences
│   │   ├── components/
│   │   │   ├── ui/                   ← shadcn/ui components
│   │   │   ├── layout/
│   │   │   │   ├── AppShell.tsx      ← Sidebar + header + content area
│   │   │   │   ├── Sidebar.tsx       ← Instance list + navigation
│   │   │   │   └── TitleBar.tsx      ← Custom window title bar (optional)
│   │   │   ├── instance/
│   │   │   │   ├── InstanceCard.tsx  ← Instance overview card
│   │   │   │   ├── StatusBadge.tsx   ← Running/Stopped/Crashed badge
│   │   │   │   └── QuickActions.tsx  ← Start/Stop/Restart buttons
│   │   │   ├── console/
│   │   │   │   └── RconTerminal.tsx  ← xterm.js wrapper
│   │   │   ├── charts/
│   │   │   │   ├── MetricsChart.tsx  ← CPU/RAM/Tick live graphs
│   │   │   │   └── PlayerChart.tsx   ← Player count over time
│   │   │   ├── maps/
│   │   │   │   ├── MapBrowser.tsx    ← Grid view of installed maps
│   │   │   │   ├── MapRotation.tsx   ← Drag-and-drop rotation editor
│   │   │   │   └── WorkshopSearch.tsx← Workshop map search + download
│   │   │   ├── players/
│   │   │   │   ├── PlayerTable.tsx   ← TanStack Table player list
│   │   │   │   └── BanDialog.tsx     ← Ban reason + duration dialog
│   │   │   └── config/
│   │   │       ├── ConfigForm.tsx    ← Visual cvar editor
│   │   │       └── ConfigEditor.tsx  ← Monaco raw config editor
│   │   ├── hooks/
│   │   │   ├── useWailsEvent.ts      ← Listen to Go event emissions
│   │   │   ├── useInstance.ts        ← Instance state + actions
│   │   │   ├── useMetrics.ts         ← Real-time metrics subscription
│   │   │   └── useConsole.ts         ← Console log buffer
│   │   ├── lib/
│   │   │   ├── bindings.ts           ← Re-exports from wailsjs/go/
│   │   │   ├── utils.ts              ← Shared helpers
│   │   │   └── types.ts              ← Shared TypeScript types
│   │   ├── stores/
│   │   │   ├── app-store.ts          ← Global app state (theme, settings)
│   │   │   └── instance-store.ts     ← Instance list + active instance
│   │   └── styles/
│   │       └── globals.css           ← TailwindCSS base + custom styles
│   │
│   ├── wailsjs/                      ← Auto-generated by Wails
│   │   ├── go/                       ← TypeScript wrappers for Go methods
│   │   │   └── main/
│   │   │       ├── App.d.ts
│   │   │       └── App.js
│   │   └── runtime/                  ← Wails runtime (events, window, etc.)
│   │       └── runtime.d.ts
│   │
│   ├── index.html
│   ├── vite.config.ts
│   ├── tailwind.config.ts
│   ├── tsconfig.json
│   └── package.json
│
├── build/                            ← Wails build configuration
│   ├── windows/
│   │   ├── icon.ico                  ← App icon
│   │   ├── info.json                 ← Version info for .exe metadata
│   │   ├── installer/
│   │   │   └── project.nsi           ← NSIS installer script
│   │   └── wails.exe.manifest        ← Windows manifest (DPI, admin, etc.)
│   └── linux/
│       └── icon.png
│
├── scripts/
│   ├── build-all.ps1                 ← Cross-compile Windows + Linux
│   └── release.ps1                   ← Build + package + checksums
│
└── .github/
    └── workflows/
        ├── ci.yml                    ← Lint + Test + Build (Windows + Linux)
        └── release.yml               ← Cross-compile + NSIS installer + GitHub Release
```

---

## 13. Development Roadmap

### Phase 1 — Foundation (Weeks 1–3)

```
[ ] Wails v2 project scaffold (Go + React + Vite + TypeScript)
[ ] App shell: sidebar, routing, window chrome, system tray
[ ] SQLite database + GORM models + auto-migration
[ ] Config file loading (Viper — config.yaml in %APPDATA%)
[ ] Structured logging (zerolog → file + in-app log viewer)
[ ] Source RCON protocol client (TCP, auth, command, multi-packet)
[ ] CS2 process manager (start, stop, restart, crash watchdog)
[ ] SteamCMD wrapper (auto-download, install CS2, update)
[ ] Wails bindings: instance CRUD, start/stop, RCON send
[ ] Basic dashboard: instance cards with status
[ ] Theme: dark/light mode following system
```

### Phase 2 — Core Panel (Weeks 4–6)

```
[ ] Embedded RCON terminal (xterm.js + Wails events)
[ ] Console output streaming (CS2 stdout → Wails event → xterm.js)
[ ] Server config visual editor (form-based, shadcn/ui components)
[ ] Raw config editor (Monaco Editor with CS2 cfg highlighting)
[ ] Config profiles (save/load)
[ ] Game mode presets (Competitive, Casual, DM, Wingman, Custom)
[ ] Map browser (grid view with thumbnails)
[ ] Map rotation drag-and-drop editor
[ ] Workshop map download (SteamCMD + progress bar)
[ ] Player list (real-time via RCON "status" polling)
[ ] Kick / Ban / Mute actions with dialogs
[ ] Bot management controls (count, difficulty, quota mode)
```

### Phase 3 — Monitoring & Performance (Weeks 7–8)

```
[ ] System metrics collector (CPU, RAM, Disk, Net via gopsutil)
[ ] CS2 metrics collector (tick rate, sv, var via RCON queries)
[ ] Real-time dashboard with animated Recharts graphs
[ ] Wails event-driven data push (1s metrics, RT console)
[ ] Metric history persistence (SQLite, configurable retention)
[ ] Performance alert system → Windows toast notifications
[ ] Benchmark engine (automated bot stress test)
[ ] Benchmark history + comparison view
```

### Phase 4 — Advanced Features (Weeks 9–11)

```
[ ] Skin/workshop item management
[ ] Plugin manager (CounterStrikeSharp / Metamod install/toggle)
[ ] File manager (sandboxed browser + Monaco editor)
[ ] Backup & restore system (.zip archives)
[ ] Scheduled tasks (cron-based — restart, update, backup, RCON)
[ ] Ban list management (permanent, timed, IP, SteamID)
[ ] Player statistics history
[ ] Instance templates & cloning
[ ] Multi-instance RCON broadcast
[ ] Command macros (custom command sequences)
```

### Phase 5 — Polish & Distribution (Weeks 12–14)

```
[ ] Discord webhook notifications
[ ] Generic webhook system
[ ] Windows toast notification preferences
[ ] Start with Windows (registry entry)
[ ] Minimize to tray behavior
[ ] NSIS installer (Windows setup .exe)
[ ] Portable .zip distribution
[ ] Auto-updater (GitHub Releases)
[ ] Keyboard shortcuts (Ctrl+1–9 instances, Ctrl+R RCON, etc.)
[ ] Config diff viewer
[ ] About page (version, credits, links)
[ ] CI/CD: GitHub Actions (test + build + release)
[ ] User guide / documentation
```

### Phase 6 — Headless & Remote Access (Weeks 15–16)

```
[ ] Headless mode (--headless flag, no GUI window)
[ ] Embedded web server (Fiber + static React build)
[ ] REST API (mirrors Wails bindings)
[ ] WebSocket hub (mirrors Wails events)
[ ] JWT authentication + Argon2id passwords
[ ] RBAC (Super Admin, Admin, Moderator, Viewer)
[ ] HTTPS (TLS 1.3, self-signed + optional Let's Encrypt)
[ ] Linux systemd service support
[ ] Linux .deb and .rpm packages
```

### Phase 7 — Future Enhancements

```
🔮 Steam/Discord OAuth for remote login
🔮 Performance advisor (AI-powered suggestions)
🔮 GeoIP player flag display
🔮 Custom nav mesh editor
🔮 SourceTV / GOTV controls
🔮 Match system (PUG/scrim organizer)
🔮 Map testing sandbox
🔮 Remote backup to S3
🔮 WinGet / Chocolatey / Scoop package
🔮 Localization (multi-language)
🔮 Plugin marketplace
🔮 Custom dashboard widgets
```

---

## Appendix A — Key CS2 Console Variables (Cvars)

Commonly managed through the panel:

| Category | Cvar | Description |
|---|---|---|
| **Server** | `hostname` | Server name in browser |
| | `sv_password` | Join password |
| | `rcon_password` | RCON password |
| | `sv_cheats` | Enable cheat commands |
| | `sv_lan` | LAN-only mode |
| **Gameplay** | `mp_roundtime` | Round duration (minutes) |
| | `mp_freezetime` | Freeze time (seconds) |
| | `mp_buytime` | Buy time (seconds) |
| | `mp_maxrounds` | Max rounds per half |
| | `mp_overtime_enable` | Enable overtime |
| | `mp_warmup_time` | Warmup duration |
| | `mp_friendlyfire` | Friendly fire toggle |
| **Bots** | `bot_quota` | Number of bots |
| | `bot_quota_mode` | fill / match / normal |
| | `bot_difficulty` | 0-3 (easy to expert) |
| | `bot_knives_only` | Bots use knives only |
| **Network** | `sv_minrate` | Min data rate |
| | `sv_maxrate` | Max data rate |
| | `sv_mincmdrate` | Min command rate |
| | `sv_maxcmdrate` | Max command rate |
| | `sv_minupdaterate` | Min update rate |
| | `sv_maxupdaterate` | Max update rate |
| **Performance** | `sv_maxunlag` | Max lag compensation |
| | `net_maxroutable` | Max packet size |

---

## Appendix B — Wails Go Binding Surface (API)

All Go methods exposed to the frontend via Wails bindings. In headless mode, these same methods are exposed as REST endpoints.

```go
// App is the main Wails application struct.
// Every public method is auto-exposed to JavaScript with TypeScript types.

// ── Instance Management ──────────────────────────────────────────────
func (a *App) GetInstances() ([]Instance, error)
func (a *App) GetInstance(id string) (*Instance, error)
func (a *App) CreateInstance(cfg InstanceConfig) (*Instance, error)
func (a *App) UpdateInstance(id string, cfg InstanceConfig) error
func (a *App) DeleteInstance(id string) error
func (a *App) StartInstance(id string) error
func (a *App) StopInstance(id string) error
func (a *App) RestartInstance(id string) error
func (a *App) GetInstanceLogs(id string, lines int) ([]string, error)

// ── RCON ─────────────────────────────────────────────────────────────
func (a *App) SendRCON(instanceID string, command string) (string, error)
func (a *App) GetCommandHistory(instanceID string) ([]string, error)

// ── Configuration ────────────────────────────────────────────────────
func (a *App) GetConfig(instanceID string) (*ServerConfig, error)
func (a *App) UpdateConfig(instanceID string, cfg ServerConfig) error
func (a *App) GetConfigProfiles(instanceID string) ([]ConfigProfile, error)
func (a *App) SaveConfigProfile(instanceID string, profile ConfigProfile) error
func (a *App) LoadConfigProfile(instanceID string, profileID string) error
func (a *App) ApplyGameModePreset(instanceID string, mode string) error

// ── Maps ─────────────────────────────────────────────────────────────
func (a *App) GetInstalledMaps(instanceID string) ([]MapInfo, error)
func (a *App) GetMapRotation(instanceID string) ([]string, error)
func (a *App) SetMapRotation(instanceID string, maps []string) error
func (a *App) ChangeMap(instanceID string, mapName string) error
func (a *App) DownloadWorkshopMap(instanceID string, workshopID int64) error
func (a *App) SearchWorkshop(query string) ([]WorkshopItem, error)

// ── Players ──────────────────────────────────────────────────────────
func (a *App) GetPlayers(instanceID string) ([]Player, error)
func (a *App) KickPlayer(instanceID string, steamID string, reason string) error
func (a *App) BanPlayer(instanceID string, ban BanRequest) error
func (a *App) MutePlayer(instanceID string, steamID string) error
func (a *App) GetBanList(instanceID string) ([]BanEntry, error)
func (a *App) RemoveBan(instanceID string, banID string) error

// ── Bots ─────────────────────────────────────────────────────────────
func (a *App) GetBotConfig(instanceID string) (*BotConfig, error)
func (a *App) UpdateBotConfig(instanceID string, cfg BotConfig) error

// ── Monitoring ───────────────────────────────────────────────────────
func (a *App) GetCurrentMetrics(instanceID string) (*Metrics, error)
func (a *App) GetMetricsHistory(instanceID string, from, to int64) ([]MetricSnapshot, error)

// ── Benchmark ────────────────────────────────────────────────────────
func (a *App) RunBenchmark(instanceID string, opts BenchmarkOptions) error
func (a *App) GetBenchmarkResults(instanceID string) ([]BenchmarkResult, error)

// ── Files ────────────────────────────────────────────────────────────
func (a *App) ListFiles(instanceID string, path string) ([]FileEntry, error)
func (a *App) ReadFile(instanceID string, path string) (string, error)
func (a *App) WriteFile(instanceID string, path string, content string) error
func (a *App) DeleteFile(instanceID string, path string) error

// ── Backups ──────────────────────────────────────────────────────────
func (a *App) CreateBackup(instanceID string, backupType string) error
func (a *App) GetBackups(instanceID string) ([]Backup, error)
func (a *App) RestoreBackup(instanceID string, backupID string) error
func (a *App) DeleteBackup(instanceID string, backupID string) error

// ── SteamCMD ─────────────────────────────────────────────────────────
func (a *App) InstallCS2Server(instanceID string) error
func (a *App) UpdateCS2Server(instanceID string) error
func (a *App) ValidateCS2Server(instanceID string) error

// ── Scheduled Tasks ──────────────────────────────────────────────────
func (a *App) GetScheduledTasks(instanceID string) ([]ScheduledTask, error)
func (a *App) CreateScheduledTask(instanceID string, task ScheduledTask) error
func (a *App) UpdateScheduledTask(taskID string, task ScheduledTask) error
func (a *App) DeleteScheduledTask(taskID string) error

// ── Settings ─────────────────────────────────────────────────────────
func (a *App) GetSettings() (*AppSettings, error)
func (a *App) UpdateSettings(settings AppSettings) error
func (a *App) GetAuditLog(limit int, offset int) ([]AuditEntry, error)
```

---

## Appendix C — Headless REST API Endpoints

When running in `--headless` mode, these REST endpoints mirror the Wails bindings:

```
Auth (headless only):
  POST   /api/v1/auth/login
  POST   /api/v1/auth/refresh
  POST   /api/v1/auth/logout
  POST   /api/v1/auth/2fa/setup
  POST   /api/v1/auth/2fa/verify

Instances:
  GET    /api/v1/instances
  POST   /api/v1/instances
  GET    /api/v1/instances/:id
  PUT    /api/v1/instances/:id
  DELETE /api/v1/instances/:id
  POST   /api/v1/instances/:id/start
  POST   /api/v1/instances/:id/stop
  POST   /api/v1/instances/:id/restart
  POST   /api/v1/instances/:id/update
  GET    /api/v1/instances/:id/logs
  POST   /api/v1/instances/:id/rcon

Config:
  GET    /api/v1/instances/:id/config
  PUT    /api/v1/instances/:id/config
  GET    /api/v1/instances/:id/config/profiles
  POST   /api/v1/instances/:id/config/profiles
  PUT    /api/v1/instances/:id/config/profiles/:pid
  POST   /api/v1/instances/:id/config/preset/:mode

Maps:
  GET    /api/v1/instances/:id/maps
  POST   /api/v1/instances/:id/maps/workshop
  DELETE /api/v1/instances/:id/maps/:mapId
  GET    /api/v1/instances/:id/maps/rotation
  PUT    /api/v1/instances/:id/maps/rotation
  POST   /api/v1/instances/:id/maps/change

Players:
  GET    /api/v1/instances/:id/players
  POST   /api/v1/instances/:id/players/:steamId/kick
  POST   /api/v1/instances/:id/players/:steamId/ban
  POST   /api/v1/instances/:id/players/:steamId/mute
  GET    /api/v1/instances/:id/bans
  DELETE /api/v1/instances/:id/bans/:banId

Bots:
  GET    /api/v1/instances/:id/bots
  PUT    /api/v1/instances/:id/bots

Monitoring:
  GET    /api/v1/instances/:id/metrics
  GET    /api/v1/instances/:id/metrics/history

Benchmark:
  POST   /api/v1/instances/:id/benchmark/run
  GET    /api/v1/instances/:id/benchmark/results

Files:
  GET    /api/v1/instances/:id/files?path=
  GET    /api/v1/instances/:id/files/content?path=
  PUT    /api/v1/instances/:id/files/content?path=
  DELETE /api/v1/instances/:id/files?path=

Backups:
  GET    /api/v1/instances/:id/backups
  POST   /api/v1/instances/:id/backups
  POST   /api/v1/instances/:id/backups/:bid/restore
  DELETE /api/v1/instances/:id/backups/:bid

Tasks:
  GET    /api/v1/instances/:id/tasks
  POST   /api/v1/instances/:id/tasks
  PUT    /api/v1/instances/:id/tasks/:tid
  DELETE /api/v1/instances/:id/tasks/:tid

Settings:
  GET    /api/v1/settings
  PUT    /api/v1/settings
  GET    /api/v1/audit

WebSocket (headless only):
  GET    /ws    ← Upgrade to WebSocket (JWT in query param)
```

---

*Last updated: 2026-02-13*
*Version: 0.2.0-draft*

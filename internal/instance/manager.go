package instance

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"cs2admin/internal/models"
	"cs2admin/internal/pkg/logger"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	StatusStopped   = "stopped"
	StatusStarting  = "starting"
	StatusRunning   = "running"
	StatusStopping  = "stopping"
	StatusCrashed   = "crashed"
	StatusUpdating  = "updating"
	StatusInstalling = "installing"
)

// Manager manages multiple CS2 instances.
type Manager struct {
	db         *gorm.DB
	processes  map[string]*Process
	watchdogs  map[string]*Watchdog
	mu         sync.RWMutex
	onOutput   func(instanceID, line string)
	onStatus   func(instanceID, status string)
}

// NewManager creates a new Manager with the given database.
func NewManager(db *gorm.DB) *Manager {
	return &Manager{
		db:        db,
		processes:  make(map[string]*Process),
		watchdogs:  make(map[string]*Watchdog),
		onOutput:  func(_, _ string) {},
		onStatus:  func(_, _ string) {},
	}
}

// SetOnOutput sets the callback invoked for each stdout line (instanceID, line).
func (m *Manager) SetOnOutput(fn func(instanceID, line string)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if fn != nil {
		m.onOutput = fn
	} else {
		m.onOutput = func(_, _ string) {}
	}
}

// SetOnStatus sets the callback invoked when instance status changes.
func (m *Manager) SetOnStatus(fn func(instanceID, status string)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if fn != nil {
		m.onStatus = fn
	} else {
		m.onStatus = func(_, _ string) {}
	}
}

// Start loads the instance from DB, builds the launch command, creates and starts
// the Process, updates DB status to "running", and creates a Watchdog if auto_restart is enabled.
func (m *Manager) Start(instanceID string) error {
	m.mu.Lock()
	if _, exists := m.processes[instanceID]; exists {
		m.mu.Unlock()
		return fmt.Errorf("instance %s already has a running process", instanceID)
	}
	m.mu.Unlock()

	var inst models.ServerInstance
	id, err := uuid.Parse(instanceID)
	if err != nil {
		return fmt.Errorf("invalid instance ID: %w", err)
	}

	if err := m.db.First(&inst, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("instance %s not found", instanceID)
		}
		return fmt.Errorf("load instance: %w", err)
	}

	m.updateStatus(instanceID, StatusStarting)
	logger.Log.Info().Str("instance", instanceID).Str("name", inst.Name).Msg("starting instance")

	exePath := m.getCS2ExePath(&inst)
	args := m.buildLaunchArgs(&inst)

	proc := NewProcess(exePath, args)
	proc.SetOnOutput(func(line string) {
		m.mu.RLock()
		fn := m.onOutput
		m.mu.RUnlock()
		fn(instanceID, line)
	})

	m.mu.Lock()
	w := m.watchdogs[instanceID]
	if inst.AutoRestart && w == nil {
		w = NewWatchdog(instanceID, m)
		m.watchdogs[instanceID] = w
	}
	m.mu.Unlock()

	proc.SetOnExit(func(code int) {
		m.mu.Lock()
		delete(m.processes, instanceID)
		w := m.watchdogs[instanceID]
		m.mu.Unlock()
		// Only set "crashed" if this was an unexpected exit (not a graceful Stop)
		var inst models.ServerInstance
		if err := m.db.Select("status").First(&inst, "id = ?", id).Error; err == nil && inst.Status == StatusRunning {
			m.updateStatus(instanceID, StatusCrashed)
		}
		if w != nil {
			w.NotifyExit(code)
		}
	})

	if err := proc.Start(); err != nil {
		m.mu.Lock()
		delete(m.watchdogs, instanceID)
		m.mu.Unlock()
		m.updateStatus(instanceID, StatusStopped)
		logger.Log.Error().Err(err).Str("instance", instanceID).Msg("failed to start process")
		return fmt.Errorf("start process: %w", err)
	}

	m.mu.Lock()
	m.processes[instanceID] = proc
	m.mu.Unlock()

	if w != nil {
		w.SetLastStartAt()
		w.Start()
	}

	m.updateStatus(instanceID, StatusRunning)
	logger.Log.Info().Str("instance", instanceID).Int("pid", proc.PID()).Msg("instance started")

	return nil
}

// Stop stops the Process, stops the Watchdog, and updates DB status to "stopped".
func (m *Manager) Stop(instanceID string) error {
	m.mu.Lock()
	proc := m.processes[instanceID]
	w := m.watchdogs[instanceID]
	delete(m.processes, instanceID)
	delete(m.watchdogs, instanceID)
	m.mu.Unlock()

	if w != nil {
		w.Stop()
	}

	if proc == nil {
		m.updateStatus(instanceID, StatusStopped)
		return nil
	}

	m.updateStatus(instanceID, StatusStopping)
	logger.Log.Info().Str("instance", instanceID).Msg("stopping instance")

	if err := proc.Stop(); err != nil {
		logger.Log.Warn().Err(err).Str("instance", instanceID).Msg("error during stop")
	}
	m.updateStatus(instanceID, StatusStopped)
	logger.Log.Info().Str("instance", instanceID).Msg("instance stopped")
	return nil
}

// Restart stops then starts the instance.
func (m *Manager) Restart(instanceID string) error {
	if err := m.Stop(instanceID); err != nil {
		return err
	}
	return m.Start(instanceID)
}

// GetStatus returns the current status for the instance.
func (m *Manager) GetStatus(instanceID string) string {
	m.mu.RLock()
	proc := m.processes[instanceID]
	m.mu.RUnlock()

	if proc != nil && proc.IsRunning() {
		return StatusRunning
	}

	var inst models.ServerInstance
	id, err := uuid.Parse(instanceID)
	if err != nil {
		return StatusStopped
	}
	if err := m.db.First(&inst, "id = ?", id).Error; err != nil {
		return StatusStopped
	}
	return inst.Status
}

// StopAll stops all running instances.
func (m *Manager) StopAll() {
	m.mu.Lock()
	ids := make([]string, 0, len(m.processes))
	for id := range m.processes {
		ids = append(ids, id)
	}
	m.mu.Unlock()

	for _, id := range ids {
		_ = m.Stop(id)
	}
}

// AutoStartAll starts all instances with auto_start=true.
func (m *Manager) AutoStartAll() error {
	var instances []models.ServerInstance
	if err := m.db.Where("auto_start = ?", true).Find(&instances).Error; err != nil {
		return fmt.Errorf("load auto-start instances: %w", err)
	}

	for i := range instances {
		id := instances[i].ID.String()
		if err := m.Start(id); err != nil {
			logger.Log.Error().Err(err).Str("instance", id).Msg("auto-start failed")
			// Continue with other instances
		}
	}

	return nil
}

// buildLaunchArgs builds cs2.exe launch args: -dedicated -port <port> +sv_lan 1
// +game_mode <x> +game_type <y> +map <map> -maxplayers <n> +rcon_password <pw>
// -console -usercon plus any custom launch_args (space-separated).
func (m *Manager) buildLaunchArgs(inst *models.ServerInstance) []string {
	gameMode, gameType := gameModeToValues(inst.GameMode)
	mapName := inst.CurrentMap
	if mapName == "" {
		mapName = "de_dust2"
	}
	maxPlayers := inst.MaxPlayers
	if maxPlayers <= 0 {
		maxPlayers = 10
	}
	rconPass := inst.RconPassword
	if rconPass == "" {
		rconPass = "changeme"
	}

	args := []string{
		"-dedicated",
		"-port", fmt.Sprintf("%d", inst.Port),
		"+sv_lan", "1",
		"+game_mode", fmt.Sprintf("%d", gameMode),
		"+game_type", fmt.Sprintf("%d", gameType),
		"+map", mapName,
		"-maxplayers", fmt.Sprintf("%d", maxPlayers),
		"+rcon_password", rconPass,
		"-console",
		"-usercon",
	}

	if strings.TrimSpace(inst.LaunchArgs) != "" {
		extra := strings.Fields(inst.LaunchArgs)
		args = append(args, extra...)
	}

	return args
}

// gameModeToValues maps GameMode string to (game_mode, game_type) for CS2.
func gameModeToValues(mode string) (int, int) {
	switch strings.ToLower(mode) {
	case "competitive":
		return 1, 0
	case "casual":
		return 0, 0
	case "wingman":
		return 2, 0
	case "deathmatch", "dm":
		return 2, 1
	case "custom":
		return 3, 0
	default:
		return 1, 0
	}
}

// getCS2ExePath returns <install_path>/game/bin/win64/cs2.exe.
func (m *Manager) getCS2ExePath(inst *models.ServerInstance) string {
	return filepath.Join(inst.InstallPath, "game", "bin", "win64", "cs2.exe")
}

// hasProcess returns true if the instance has a running process.
func (m *Manager) hasProcess(instanceID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.processes[instanceID]
	return ok
}

// hasWatchdog returns true if the instance has an active watchdog.
func (m *Manager) hasWatchdog(instanceID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.watchdogs[instanceID]
	return ok
}

// updateStatus updates the DB and calls the onStatus callback.
func (m *Manager) updateStatus(instanceID, status string) {
	id, err := uuid.Parse(instanceID)
	if err != nil {
		return
	}

	if err := m.db.Model(&models.ServerInstance{}).Where("id = ?", id).Update("status", status).Error; err != nil {
		logger.Log.Error().Err(err).Str("instance", instanceID).Str("status", status).Msg("failed to update status in DB")
	}

	m.mu.RLock()
	fn := m.onStatus
	m.mu.RUnlock()
	fn(instanceID, status)
}

package scheduler

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"cs2admin/internal/models"
	"cs2admin/internal/pkg/logger"

	"gorm.io/gorm"
)

// TaskAction defines the type of scheduled action.
type TaskAction string

const (
	ActionRestart   TaskAction = "restart"
	ActionUpdate   TaskAction = "update"
	ActionBackup   TaskAction = "backup"
	ActionRCON     TaskAction = "rcon"
	ActionMapChange TaskAction = "map_change"
)

// ScheduledEntry holds a task and its next run time.
type ScheduledEntry struct {
	Task    models.ScheduledTask
	NextRun time.Time
}

// Scheduler manages cron-like scheduled tasks.
type Scheduler struct {
	db       *gorm.DB
	tasks    map[string]*ScheduledEntry
	mu       sync.RWMutex
	stopCh   chan struct{}
	onAction func(instanceID string, action TaskAction, payload string)
}

// New creates a new scheduler.
func New(db *gorm.DB) *Scheduler {
	return &Scheduler{
		db:    db,
		tasks: make(map[string]*ScheduledEntry),
	}
}

// SetOnAction sets the callback invoked when a task is due.
func (s *Scheduler) SetOnAction(fn func(instanceID string, action TaskAction, payload string)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onAction = fn
}

// Start loads tasks from DB and starts the ticker (checks every 30s).
func (s *Scheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopCh != nil {
		return nil
	}

	// Load tasks from DB
	var dbTasks []models.ScheduledTask
	if err := s.db.Where("enabled = ?", true).Find(&dbTasks).Error; err != nil {
		return fmt.Errorf("load tasks: %w", err)
	}

	s.stopCh = make(chan struct{})

	for i := range dbTasks {
		t := &dbTasks[i]
		nextRun, err := ParseSimpleCron(t.CronExpr)
		if err != nil {
			logger.Log.Warn().Err(err).Str("task_id", t.ID.String()).Str("cron", t.CronExpr).Msg("scheduler: skip task, invalid cron")
			continue
		}
		s.tasks[t.ID.String()] = &ScheduledEntry{Task: *t, NextRun: nextRun}
	}

	go s.run()
	logger.Log.Info().Int("tasks", len(s.tasks)).Msg("scheduler: started")
	return nil
}

// Stop stops the scheduler.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopCh == nil {
		return
	}
	close(s.stopCh)
	s.stopCh = nil
	logger.Log.Info().Msg("scheduler: stopped")
}

func (s *Scheduler) run() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	s.mu.Lock()
	stopCh := s.stopCh
	s.mu.Unlock()

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			s.checkAndRun()
		}
	}
}

func (s *Scheduler) checkAndRun() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for _, entry := range s.tasks {
		if !entry.Task.Enabled || entry.NextRun.After(now) {
			continue
		}

		// Task is due
		instanceID := entry.Task.InstanceID.String()
		action := TaskAction(entry.Task.Action)
		payload := entry.Task.Payload

		// Update last_run, compute next_run
		lastRun := now
		nextRun, err := ParseSimpleCron(entry.Task.CronExpr)
		if err != nil {
			nextRun = now.Add(24 * time.Hour)
		}

		s.db.Model(&entry.Task).Updates(map[string]interface{}{
			"last_run": lastRun,
			"next_run": nextRun,
		})

		entry.NextRun = nextRun

		// Invoke callback
		fn := s.onAction
		s.mu.Unlock()
		if fn != nil {
			fn(instanceID, action, payload)
		}
		s.mu.Lock()
	}
}

// AddTask adds a scheduled task.
func (s *Scheduler) AddTask(task models.ScheduledTask) error {
	nextRun, err := ParseSimpleCron(task.CronExpr)
	if err != nil {
		return fmt.Errorf("invalid cron: %w", err)
	}

	task.Enabled = true
	task.NextRun = &nextRun
	if err := s.db.Create(&task).Error; err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[task.ID.String()] = &ScheduledEntry{Task: task, NextRun: nextRun}
	return nil
}

// RemoveTask removes a task by ID.
func (s *Scheduler) RemoveTask(taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tasks, taskID)
	return s.db.Delete(&models.ScheduledTask{}, "id = ?", taskID).Error
}

// ListTasks returns tasks for the given instance.
func (s *Scheduler) ListTasks(instanceID string) ([]models.ScheduledTask, error) {
	var tasks []models.ScheduledTask
	err := s.db.Where("instance_id = ?", instanceID).Order("created_at DESC").Find(&tasks).Error
	return tasks, err
}

// ParseSimpleCron parses a simple cron expression and returns the next run time.
// Supports: "min hour * * *" with basic step/range.
// Examples:
//   - "0 */6 * * *" = every 6 hours at minute 0
//   - "0 3 * * *" = daily at 03:00
//   - "30 2 * * *" = daily at 02:30
//   - "* * * * *" = every minute
func ParseSimpleCron(expr string) (time.Time, error) {
	parts := strings.Fields(strings.TrimSpace(expr))
	if len(parts) != 5 {
		return time.Time{}, fmt.Errorf("cron must have 5 fields: min hour day month dow")
	}

	now := time.Now()

	min, err := parseCronField(parts[0], 0, 59, now.Minute())
	if err != nil {
		return time.Time{}, fmt.Errorf("minute: %w", err)
	}
	hour, err := parseCronField(parts[1], 0, 23, now.Hour())
	if err != nil {
		return time.Time{}, fmt.Errorf("hour: %w", err)
	}

	// Simple handling: if the next min/hour is in the past today, advance to next day
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, min, 0, 0, now.Location())
	if next.Before(now) || next.Equal(now) {
		next = next.Add(24 * time.Hour)
	}
	return next, nil
}

func parseCronField(f string, lo, hi, current int) (int, error) {
	if f == "*" {
		return current, nil
	}

	// Step: "*/6" means every 6
	if strings.HasPrefix(f, "*/") {
		step, err := strconv.Atoi(f[2:])
		if err != nil || step <= 0 {
			return 0, fmt.Errorf("invalid step: %s", f)
		}
		// For hour: "*/6" → 0, 6, 12, 18
		// Next value >= current that matches
		v := ((current / step) + 1) * step
		if v > hi {
			v = 0
		}
		return v, nil
	}

	v, err := strconv.Atoi(f)
	if err != nil {
		return 0, fmt.Errorf("invalid number: %s", f)
	}
	if v < lo || v > hi {
		return 0, fmt.Errorf("out of range [%d,%d]: %d", lo, hi, v)
	}
	return v, nil
}

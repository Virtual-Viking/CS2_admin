package instance

import (
	"sync"
	"time"

	"cs2admin/internal/pkg/logger"
)

// Watchdog provides auto-restart with exponential backoff for a CS2 instance.
type Watchdog struct {
	instanceID  string
	manager     *Manager
	running     bool
	stopCh      chan struct{}
	exitCh      chan int
	mu          sync.Mutex
	backoff     time.Duration
	maxBackoff  time.Duration
	lastStartAt time.Time
}

// NewWatchdog creates a new Watchdog for the given instance.
func NewWatchdog(instanceID string, manager *Manager) *Watchdog {
	return &Watchdog{
		instanceID: instanceID,
		manager:    manager,
		stopCh:     make(chan struct{}),
		exitCh:     make(chan int, 1),
		backoff:    1 * time.Second,
		maxBackoff: 30 * time.Second,
	}
}

// SetLastStartAt records the current time as the last successful start.
// Called by Manager after a successful process start.
func (w *Watchdog) SetLastStartAt() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.lastStartAt = time.Now()
}

// NotifyExit is called when the managed process exits. It sends the exit code
// to the Watchdog goroutine (non-blocking).
func (w *Watchdog) NotifyExit(exitCode int) {
	select {
	case w.exitCh <- exitCode:
	default:
	}
}

// Start begins the watchdog goroutine that waits for process exit, then
// restarts with exponential backoff. Backoff resets when a start lasts > 60s.
func (w *Watchdog) Start() {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return
	}
	w.running = true
	w.stopCh = make(chan struct{})
	w.mu.Unlock()

	logger.Log.Debug().Str("instance", w.instanceID).Msg("watchdog started")

	go func() {
		for {
			select {
			case <-w.stopCh:
				w.mu.Lock()
				w.running = false
				w.mu.Unlock()
				logger.Log.Debug().Str("instance", w.instanceID).Msg("watchdog stopped")
				return

			case exitCode := <-w.exitCh:
				w.mu.Lock()
				ranLongEnough := !w.lastStartAt.IsZero() && time.Since(w.lastStartAt) > 60*time.Second
				if ranLongEnough {
					w.backoff = 1 * time.Second
				}
				backoff := w.backoff
				w.backoff *= 2
				if w.backoff > w.maxBackoff {
					w.backoff = w.maxBackoff
				}
				w.mu.Unlock()

				logger.Log.Info().
					Str("instance", w.instanceID).
					Int("exitCode", exitCode).
					Dur("backoff", backoff).
					Bool("resetBackoff", ranLongEnough).
					Msg("process exited, scheduling restart")

				select {
				case <-w.stopCh:
					w.mu.Lock()
					w.running = false
					w.mu.Unlock()
					return
				case <-time.After(backoff):
					// proceed to restart
				}

				if !w.manager.hasWatchdog(w.instanceID) {
					return
				}
				if w.manager.hasProcess(w.instanceID) {
					continue
				}

				w.lastStartAt = time.Now()
				if err := w.manager.Start(w.instanceID); err != nil {
					logger.Log.Error().Err(err).Str("instance", w.instanceID).Msg("watchdog restart failed")
				}
			}
		}
	}()
}

// Stop signals the watchdog to stop.
func (w *Watchdog) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.running {
		return
	}

	close(w.stopCh)
	w.running = false
}

// IsRunning returns true if the watchdog is running.
func (w *Watchdog) IsRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.running
}

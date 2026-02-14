package benchmark

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cs2admin/internal/models"
	"cs2admin/internal/pkg/logger"
	"cs2admin/internal/rcon"

	"github.com/google/uuid"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"gorm.io/gorm"
)

// BenchmarkConfig configures a benchmark run.
type BenchmarkConfig struct {
	InstanceID   string
	MaxBots      int
	StepSize     int
	StepDuration time.Duration
}

// Metrics holds benchmark step metrics.
type Metrics struct {
	BotCount    int     `json:"bot_count"`
	AvgTickRate float64 `json:"avg_tickrate"`
	MinTickRate float64 `json:"min_tickrate"`
	CPUUsage    float64 `json:"cpu_usage"`
	RAMUsage    float64 `json:"ram_usage"`
}

// BenchmarkRunner executes performance benchmarks.
type BenchmarkRunner struct {
	config     BenchmarkConfig
	db         *gorm.DB
	rconPool   *rcon.Pool
	stopCh     chan struct{}
	onProgress func(step int, totalSteps int, metrics Metrics)
	mu         sync.Mutex
}

// NewRunner creates a new benchmark runner.
func NewRunner(cfg BenchmarkConfig, db *gorm.DB) *BenchmarkRunner {
	return &BenchmarkRunner{
		config:   cfg,
		db:       db,
		stopCh:   make(chan struct{}),
	}
}

// SetRconPool sets the RCON pool for sending bot commands.
func (r *BenchmarkRunner) SetRconPool(pool *rcon.Pool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rconPool = pool
}

// SetOnProgress sets the callback invoked at each benchmark step.
func (r *BenchmarkRunner) SetOnProgress(fn func(step int, totalSteps int, metrics Metrics)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onProgress = fn
}

// Run executes the benchmark: adds bots incrementally, collects metrics at each step.
func (r *BenchmarkRunner) Run() (*models.BenchmarkResult, error) {
	if r.config.MaxBots <= 0 || r.config.StepSize <= 0 {
		return nil, fmt.Errorf("invalid config: MaxBots=%d StepSize=%d", r.config.MaxBots, r.config.StepSize)
	}

	totalSteps := (r.config.MaxBots + r.config.StepSize - 1) / r.config.StepSize
	if totalSteps == 0 {
		totalSteps = 1
	}

	r.mu.Lock()
	r.stopCh = make(chan struct{})
	r.mu.Unlock()

	var tickRates []float64
	var cpuUsages []float64
	var ramUsages []float64
	stepDuration := r.config.StepDuration
	if stepDuration < time.Second {
		stepDuration = 5 * time.Second
	}

	instUUID, err := uuid.Parse(r.config.InstanceID)
	if err != nil {
		return nil, fmt.Errorf("invalid instance id: %w", err)
	}

	for step := 1; step <= totalSteps; step++ {
		select {
		case <-r.stopCh:
			return nil, fmt.Errorf("benchmark stopped")
		default:
		}

		botCount := step * r.config.StepSize
		if botCount > r.config.MaxBots {
			botCount = r.config.MaxBots
		}

		// Set bot count via RCON
		if r.rconPool != nil {
			if _, ok := r.rconPool.Get(r.config.InstanceID); ok {
				cmd := fmt.Sprintf("bot_quota %d", botCount)
				r.rconPool.Execute(r.config.InstanceID, cmd)
				time.Sleep(2 * time.Second) // Allow bots to spawn
			}
		}

		// Collect metrics for StepDuration
		var ticks []float64
		var cpus []float64
		var rams []float64
		deadline := time.Now().Add(stepDuration)
		interval := 500 * time.Millisecond

		for time.Now().Before(deadline) {
			select {
			case <-r.stopCh:
				return nil, fmt.Errorf("benchmark stopped")
			default:
			}

			// System metrics
			if pcts, err := cpu.PercentWithContext(context.Background(), interval, false); err == nil && len(pcts) > 0 {
				var sum float64
				for _, p := range pcts {
					sum += p
				}
				cpus = append(cpus, sum/float64(len(pcts)))
			}
			if vm, err := mem.VirtualMemory(); err == nil {
				rams = append(rams, float64(vm.Used)/(1024*1024))
			}

			// Tick rate from RCON status
			if r.rconPool != nil {
				if out, err := r.rconPool.Execute(r.config.InstanceID, "status"); err == nil && out != "" {
					tickRate := parseTickFromStatus(out)
					ticks = append(ticks, tickRate)
				}
			}

			time.Sleep(interval)
		}

		// Aggregate step metrics
		avgTick, minTick := avgAndMin(ticks)
		avgCPU := avg(cpus)
		avgRAM := avg(rams)

		tickRates = append(tickRates, avgTick)
		cpuUsages = append(cpuUsages, avgCPU)
		ramUsages = append(ramUsages, avgRAM)

		m := Metrics{
			BotCount:    botCount,
			AvgTickRate: avgTick,
			MinTickRate: minTick,
			CPUUsage:    avgCPU,
			RAMUsage:    avgRAM,
		}

		r.mu.Lock()
		fn := r.onProgress
		r.mu.Unlock()
		if fn != nil {
			fn(step, totalSteps, m)
		}
	}

	// Reset bots
	if r.rconPool != nil {
		if _, ok := r.rconPool.Get(r.config.InstanceID); ok {
			r.rconPool.Execute(r.config.InstanceID, "bot_quota 0")
		}
	}

	// Final result (use last step or aggregate)
	avgTick := avg(tickRates)
	minTick := min(tickRates)
	avgCPU := avg(cpuUsages)
	avgRAM := avg(ramUsages)

	result := &models.BenchmarkResult{
		InstanceID:   instUUID,
		BotCount:     r.config.MaxBots,
		AvgTickrate:  avgTick,
		MinTickrate:  minTick,
		AvgFrametime: 0, // Not parsed from status
		CPUUsage:     avgCPU,
		RAMUsage:     avgRAM,
		DurationSec:  int(stepDuration.Seconds()) * totalSteps,
	}

	if err := r.db.Create(result).Error; err != nil {
		return nil, fmt.Errorf("save result: %w", err)
	}

	logger.Log.Info().
		Str("instance", r.config.InstanceID).
		Int("bots", r.config.MaxBots).
		Float64("avg_tick", avgTick).
		Msg("benchmark: completed")

	return result, nil
}

func parseTickFromStatus(out string) float64 {
	// Look for tick in status output
	for _, line := range splitLines(out) {
		if len(line) > 4 && (line[:4] == "tick" || line[:8] == "tickrate") {
			var v float64
			fmt.Sscanf(line, "tick : %f", &v)
			if v == 0 {
				fmt.Sscanf(line, "tickrate : %f", &v)
			}
			return v
		}
	}
	return 0
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == '\n' {
			if i > start {
				lines = append(lines, s[start:i])
			}
			start = i + 1
		}
	}
	return lines
}

func avg(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	var sum float64
	for _, x := range xs {
		sum += x
	}
	return sum / float64(len(xs))
}

func avgAndMin(xs []float64) (avg, min float64) {
	if len(xs) == 0 {
		return 0, 0
	}
	sum := xs[0]
	minVal := xs[0]
	for i := 1; i < len(xs); i++ {
		sum += xs[i]
		if xs[i] < minVal {
			minVal = xs[i]
		}
	}
	return sum / float64(len(xs)), minVal
}

func min(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	m := xs[0]
	for _, x := range xs[1:] {
		if x < m {
			m = x
		}
	}
	return m
}

// Stop stops an in-progress benchmark.
func (r *BenchmarkRunner) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.stopCh != nil {
		close(r.stopCh)
		r.stopCh = nil
	}
}

// ListResults returns benchmark results for the instance.
func ListResults(db *gorm.DB, instanceID string) ([]models.BenchmarkResult, error) {
	var results []models.BenchmarkResult
	err := db.Where("instance_id = ?", instanceID).Order("created_at DESC").Find(&results).Error
	return results, err
}

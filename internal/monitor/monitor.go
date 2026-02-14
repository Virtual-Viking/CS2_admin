package monitor

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"sync"
	"time"

	"cs2admin/internal/models"
	"cs2admin/internal/pkg/logger"
	"cs2admin/internal/rcon"

	"github.com/google/uuid"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
	"gorm.io/gorm"
)

// Metrics holds system and CS2 metrics.
type Metrics struct {
	CPUPercent float64 `json:"cpu_pct"`
	RAMMb      float64 `json:"ram_mb"`
	TickRate   float64 `json:"tick_rate"`
	Players    int     `json:"players"`
	NetInKbps  float64 `json:"net_in_kbps"`
	NetOutKbps float64 `json:"net_out_kbps"`
}

// Collector collects system and CS2 metrics periodically.
type Collector struct {
	instanceID string
	rconAddr  string
	rconPass  string
	db        *gorm.DB
	rconPool  *rcon.Pool
	stopCh    chan struct{}
	onMetrics func(instanceID string, m Metrics)
	running   bool
	mu        sync.Mutex

	// For network rate calculation
	prevNetRecv uint64
	prevNetSent uint64
	prevNetTime time.Time
}

// NewCollector creates a new metrics collector.
func NewCollector(instanceID, rconAddr, rconPass string, db *gorm.DB) *Collector {
	return &Collector{
		instanceID:  instanceID,
		rconAddr:    rconAddr,
		rconPass:    rconPass,
		db:          db,
		prevNetTime: time.Now(),
	}
}

// SetRconPool sets the RCON pool for querying CS2 metrics.
func (c *Collector) SetRconPool(pool *rcon.Pool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rconPool = pool
}

// SetOnMetrics sets the callback invoked when new metrics are collected.
func (c *Collector) SetOnMetrics(fn func(instanceID string, m Metrics)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onMetrics = fn
}

// Start begins the metrics collection goroutine (1s interval).
func (c *Collector) Start() {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return
	}
	c.stopCh = make(chan struct{})
	c.running = true
	c.mu.Unlock()

	// Ensure RCON is connected for this instance
	if c.rconPool != nil {
		if _, ok := c.rconPool.Get(c.instanceID); !ok {
			if err := c.rconPool.Connect(c.instanceID, c.rconAddr, c.rconPass); err != nil {
				logger.Log.Warn().Err(err).Str("instance", c.instanceID).Msg("monitor: rcon not connected, cs2 metrics will be 0")
			}
		}
	}

	go c.run()
	logger.Log.Info().Str("instance", c.instanceID).Msg("monitor: collector started")
}

// Stop stops the metrics collection goroutine.
func (c *Collector) Stop() {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return
	}
	c.running = false
	ch := c.stopCh
	c.stopCh = nil
	c.mu.Unlock()
	close(ch)
	logger.Log.Info().Str("instance", c.instanceID).Msg("monitor: collector stopped")
}

func (c *Collector) run() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	c.mu.Lock()
	stopCh := c.stopCh
	c.mu.Unlock()

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			c.collectOnce()
		}
	}
}

func (c *Collector) collectOnce() {
	m := c.collectMetrics()

	// Store MetricSnapshot in DB
	instUUID, err := uuid.Parse(c.instanceID)
	if err == nil {
		snap := models.MetricSnapshot{
			InstanceID: instUUID,
			CPUPct:     m.CPUPercent,
			RAMMb:      m.RAMMb,
			TickRate:   m.TickRate,
			Players:    m.Players,
			NetInKbps:  m.NetInKbps,
			NetOutKbps: m.NetOutKbps,
			Timestamp:  time.Now(),
		}
		if err := c.db.Create(&snap).Error; err != nil {
			logger.Log.Debug().Err(err).Str("instance", c.instanceID).Msg("monitor: failed to store snapshot")
		}
	}

	// Callback
	c.mu.Lock()
	fn := c.onMetrics
	c.mu.Unlock()
	if fn != nil {
		fn(c.instanceID, m)
	}
}

func (c *Collector) collectMetrics() Metrics {
	m := Metrics{}

	// System metrics (gopsutil)
	if pcts, err := cpu.PercentWithContext(context.Background(), 500*time.Millisecond, false); err == nil && len(pcts) > 0 {
		// Average across all cores
		var sum float64
		for _, p := range pcts {
			sum += p
		}
		m.CPUPercent = sum / float64(len(pcts))
	}

	if vm, err := mem.VirtualMemory(); err == nil {
		m.RAMMb = float64(vm.Used) / (1024 * 1024)
	}

	// Network: compute kbps from IOCounters delta
	if counters, err := net.IOCounters(false); err == nil && len(counters) > 0 {
		totalRecv := counters[0].BytesRecv
		totalSent := counters[0].BytesSent
		now := time.Now()
		elapsed := now.Sub(c.prevNetTime).Seconds()
		if elapsed > 0 {
			m.NetInKbps = (float64(totalRecv-c.prevNetRecv) / 1024) / elapsed
			m.NetOutKbps = (float64(totalSent-c.prevNetSent) / 1024) / elapsed
		}
		c.prevNetRecv = totalRecv
		c.prevNetSent = totalSent
		c.prevNetTime = now
	}

	// CS2 metrics via RCON status
	if c.rconPool != nil {
		if client, ok := c.rconPool.Get(c.instanceID); ok && client != nil {
			out, err := c.rconPool.Execute(c.instanceID, "status")
			if err == nil && out != "" {
				m.TickRate, m.Players = parseStatusOutput(out)
			}
		}
	}

	return m
}

// parseStatusOutput extracts tick rate and player count from RCON "status" output.
var (
	playersRe = regexp.MustCompile(`players\s*:\s*(\d+)\s+human(?:s)?(?:,\s*(\d+)\s+bot(?:s)?)?\s*\((\d+)\s+max\)`)
	tickRe    = regexp.MustCompile(`(?:tick|tickrate)\s*[:=]?\s*(\d+)`)
)

func parseStatusOutput(out string) (tickRate float64, players int) {
	// Players: "players : 3 humans, 2 bots (10 max)" or "players : 5 humans (10 max)"
	if m := playersRe.FindStringSubmatch(out); len(m) >= 2 {
		humans, _ := strconv.Atoi(m[1])
		bots := 0
		if len(m) >= 3 && m[2] != "" {
			bots, _ = strconv.Atoi(m[2])
		}
		players = humans + bots
	}

	// Tick rate: look for "tick: 128" or similar
	if m := tickRe.FindStringSubmatch(out); len(m) >= 2 {
		tickRate, _ = strconv.ParseFloat(m[1], 64)
	}

	return tickRate, players
}

// GetHistory returns recent metric snapshots for the instance.
func (c *Collector) GetHistory(duration time.Duration) ([]models.MetricSnapshot, error) {
	instUUID, err := uuid.Parse(c.instanceID)
	if err != nil {
		return nil, fmt.Errorf("invalid instance id: %w", err)
	}

	since := time.Now().Add(-duration)
	var snapshots []models.MetricSnapshot
	err = c.db.Where("instance_id = ? AND timestamp >= ?", instUUID, since).
		Order("timestamp ASC").
		Find(&snapshots).Error
	if err != nil {
		return nil, fmt.Errorf("query history: %w", err)
	}
	return snapshots, nil
}

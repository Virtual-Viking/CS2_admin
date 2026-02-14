package notify

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"cs2admin/internal/pkg/logger"
)

// AlertThresholds defines performance thresholds for alerting.
type AlertThresholds struct {
	CPUPercent  float64
	RAMPercent  float64
	TickRateMin float64
}

// AlertManager checks metrics against thresholds and sends notifications when breached.
type AlertManager struct {
	thresholds  map[string]AlertThresholds
	notifier    *Notifier
	mu          sync.RWMutex
	lastAlert   map[string]time.Time
	alertCooldown time.Duration
}

// NewAlertManager creates a new AlertManager.
func NewAlertManager(notifier *Notifier) *AlertManager {
	return &AlertManager{
		thresholds:    make(map[string]AlertThresholds),
		notifier:      notifier,
		lastAlert:     make(map[string]time.Time),
		alertCooldown: 5 * time.Minute, // avoid spamming
	}
}

// SetThresholds sets the alert thresholds for an instance.
func (am *AlertManager) SetThresholds(instanceID string, t AlertThresholds) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.thresholds[instanceID] = t
}

// Check checks metrics against thresholds and sends notification if breached.
func (am *AlertManager) Check(instanceID string, cpuPct, ramPct, tickRate float64) {
	am.mu.RLock()
	t, ok := am.thresholds[instanceID]
	am.mu.RUnlock()

	if !ok {
		return
	}

	var breaches []string
	if t.CPUPercent > 0 && cpuPct >= t.CPUPercent {
		breaches = append(breaches, "CPU")
	}
	if t.RAMPercent > 0 && ramPct >= t.RAMPercent {
		breaches = append(breaches, "RAM")
	}
	if t.TickRateMin > 0 && tickRate > 0 && tickRate < t.TickRateMin {
		breaches = append(breaches, "TickRate")
	}

	if len(breaches) == 0 {
		return
	}

	// Cooldown: don't alert repeatedly
	key := instanceID + ":" + stringList(breaches)
	am.mu.Lock()
	if last, ok := am.lastAlert[key]; ok && time.Since(last) < am.alertCooldown {
		am.mu.Unlock()
		return
	}
	am.lastAlert[key] = time.Now()
	am.mu.Unlock()

	title := "CS2 Admin: Performance Alert"
	msg := "Instance " + instanceID + " breached thresholds: " + stringList(breaches) +
		" (CPU: " + formatPct(cpuPct) + "%, RAM: " + formatPct(ramPct) + "%, Tick: " + formatFloat(tickRate) + ")"

	if err := am.notifier.SendToast(title, msg); err != nil {
		logger.Log.Error().Err(err).Msg("SendToast from AlertManager failed")
	}
	if am.notifier.discordURL != "" {
		payload, _ := json.Marshal(map[string]interface{}{
			"instance_id": instanceID,
			"breaches":    breaches,
			"cpu_pct":     cpuPct,
			"ram_pct":     ramPct,
			"tick_rate":   tickRate,
		})
		_ = am.notifier.SendDiscord(title, msg, 0xFF0000) // red
		_ = am.notifier.SendWebhook("performance_alert", string(payload))
	}
}

func stringList(s []string) string {
	if len(s) == 0 {
		return ""
	}
	out := s[0]
	for i := 1; i < len(s); i++ {
		out += ", " + s[i]
	}
	return out
}

func formatPct(v float64) string {
	return fmt.Sprintf("%.1f", v)
}

func formatFloat(v float64) string {
	return fmt.Sprintf("%.1f", v)
}

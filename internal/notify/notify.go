package notify

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"cs2admin/internal/pkg/logger"
)

// NotifyType represents the type of notification.
type NotifyType string

const (
	NotifyToast   NotifyType = "toast"
	NotifyDiscord NotifyType = "discord"
	NotifyWebhook NotifyType = "webhook"
)

// Notifier sends notifications via various channels.
type Notifier struct {
	discordURL string
	webhookURL string
	client     *http.Client
}

// New creates a new Notifier.
func New() *Notifier {
	return &Notifier{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SetDiscordURL sets the Discord webhook URL.
func (n *Notifier) SetDiscordURL(url string) {
	n.discordURL = url
}

// SetWebhookURL sets the generic webhook URL.
func (n *Notifier) SetWebhookURL(url string) {
	n.webhookURL = url
}

// SendToast sends a Windows toast notification. On Windows, uses PowerShell with
// New-BurntToastNotification (if module installed). Fallback: log only.
func (n *Notifier) SendToast(title, message string) error {
	if runtime.GOOS == "windows" {
		// Try BurntToast: New-BurntToastNotification -Text "Title","Message"
		script := `$ErrorActionPreference='Stop'; try { Import-Module BurntToast -ErrorAction Stop; New-BurntToastNotification -Text "` + escapePS(title) + `","` + escapePS(message) + `" } catch { Write-Error $_ }`
		cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
		if err := cmd.Run(); err != nil {
			logger.Log.Warn().Err(err).Str("title", title).Str("message", message).Msg("toast notification failed, falling back to log")
			logger.Log.Info().Str("title", title).Str("message", message).Msg("Toast (fallback):")
			return nil
		}
		return nil
	}
	logger.Log.Info().Str("title", title).Str("message", message).Msg("Toast (non-Windows, logged):")
	return nil
}

// escapePS escapes double quotes (using PowerShell backtick) and newlines.
func escapePS(s string) string {
	result := ""
	for _, c := range s {
		if c == '"' {
			result += "`\""
		} else if c == '\r' || c == '\n' {
			result += " "
		} else {
			result += string(c)
		}
	}
	return result
}

// discordPayload is the JSON structure for Discord webhook.
type discordPayload struct {
	Embeds []discordEmbed `json:"embeds"`
}

type discordEmbed struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Color       int    `json:"color"`
}

// SendDiscord sends a Discord webhook embed.
func (n *Notifier) SendDiscord(title, message string, color int) error {
	if n.discordURL == "" {
		logger.Log.Debug().Msg("Discord URL not set, skipping")
		return nil
	}
	payload := discordPayload{
		Embeds: []discordEmbed{
			{
				Title:       title,
				Description: message,
				Color:       color,
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := n.client.Post(n.discordURL, "application/json", bytes.NewReader(body))
	if err != nil {
		logger.Log.Error().Err(err).Str("title", title).Msg("Discord webhook POST failed")
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logger.Log.Error().Int("status", resp.StatusCode).Str("title", title).Msg("Discord webhook returned non-2xx")
		return nil
	}
	return nil
}

// webhookPayload is the JSON structure for generic webhook.
type webhookPayload struct {
	Event   string `json:"event"`
	Payload string `json:"payload"`
}

// SendWebhook sends a generic JSON webhook POST.
func (n *Notifier) SendWebhook(event, payload string) error {
	if n.webhookURL == "" {
		logger.Log.Debug().Msg("Webhook URL not set, skipping")
		return nil
	}
	body, err := json.Marshal(webhookPayload{Event: event, Payload: payload})
	if err != nil {
		return err
	}
	resp, err := n.client.Post(n.webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		logger.Log.Error().Err(err).Str("event", event).Msg("Webhook POST failed")
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logger.Log.Error().Int("status", resp.StatusCode).Str("event", event).Msg("Webhook returned non-2xx")
		return nil
	}
	return nil
}

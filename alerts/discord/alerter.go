package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ahmadsaubani/go-logging-lib/alerts"
)

type Config struct {
	Enabled    bool   `yaml:"enabled"`
	WebhookURL string `yaml:"webhook_url"`
	Username   string `yaml:"username"`
	AvatarURL  string `yaml:"avatar_url"`
}

type Alerter struct {
	config *Config
	client *http.Client
}

/**
 * New creates a new Discord alerter instance.
 * Uses Discord webhooks to send rich embed messages with error details.
 *
 * @param config Discord webhook configuration including URL and display settings
 * @return *Alerter Ready-to-use Discord alerter
 */
func New(config *Config) *Alerter {
	return &Alerter{
		config: config,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (a *Alerter) Name() string {
	return "Discord"
}

/**
 * Send dispatches an alert to Discord via webhook.
 * Creates a rich embed message with color-coded severity and detailed fields.
 *
 * @param payload Alert data containing error details and request metadata
 * @return error Returns nil on success, or error if webhook fails
 */
func (a *Alerter) Send(payload alerts.Payload) error {
	if a.config.WebhookURL == "" {
		return fmt.Errorf("discord webhook URL is empty")
	}

	message := a.buildMessage(payload)

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal discord message: %w", err)
	}

	resp, err := a.client.Post(a.config.WebhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send discord webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("discord webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func (a *Alerter) buildMessage(payload alerts.Payload) map[string]interface{} {
	color := a.getLevelColor(payload.Level)

	embed := map[string]interface{}{
		"title":       fmt.Sprintf("🚨 %s Alert", payload.Level),
		"description": payload.Error,
		"color":       color,
		"timestamp":   payload.Timestamp.Format(time.RFC3339),
		"fields": []map[string]interface{}{
			{"name": "Service", "value": payload.ServiceName, "inline": true},
			{"name": "Level", "value": payload.Level, "inline": true},
			{"name": "Method", "value": payload.Method, "inline": true},
			{"name": "Path", "value": payload.Path, "inline": false},
			{"name": "Client IP", "value": defaultIfEmpty(payload.IP, "N/A"), "inline": true},
			{"name": "Source", "value": fmt.Sprintf("%s:%d", payload.File, payload.Line), "inline": true},
			{"name": "Request ID", "value": defaultIfEmpty(payload.RequestID, "N/A"), "inline": false},
		},
		"footer": map[string]string{
			"text": "Go Logging Library",
		},
	}

	if len(payload.Stack) > 0 {
		stackStr := "```\n"
		for _, frame := range payload.Stack {
			stackStr += frame + "\n"
		}
		stackStr += "```"
		embed["fields"] = append(embed["fields"].([]map[string]interface{}),
			map[string]interface{}{"name": "Stack Trace", "value": truncate(stackStr, 1024), "inline": false},
		)
	}

	message := map[string]interface{}{
		"embeds": []map[string]interface{}{embed},
	}

	if a.config.Username != "" {
		message["username"] = a.config.Username
	}
	if a.config.AvatarURL != "" {
		message["avatar_url"] = a.config.AvatarURL
	}

	return message
}

func (a *Alerter) getLevelColor(level string) int {
	colors := map[string]int{
		"CRITICAL": 0xDC3545,
		"ERROR":    0xFD7E14,
		"WARN":     0xFFC107,
	}
	if color, ok := colors[level]; ok {
		return color
	}
	return 0x6C757D
}

func defaultIfEmpty(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

package slack

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
	Channel    string `yaml:"channel"`
	Username   string `yaml:"username"`
	IconEmoji  string `yaml:"icon_emoji"`
}

type Alerter struct {
	config *Config
	client *http.Client
}

/**
 * New creates a new Slack alerter instance.
 * Uses Slack incoming webhooks to send attachment messages with error details.
 *
 * @param config Slack webhook configuration including URL and channel settings
 * @return *Alerter Ready-to-use Slack alerter
 */
func New(config *Config) *Alerter {
	return &Alerter{
		config: config,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (a *Alerter) Name() string {
	return "Slack"
}

/**
 * Send dispatches an alert to Slack via webhook.
 * Creates an attachment message with color-coded severity and detailed fields.
 *
 * @param payload Alert data containing error details and request metadata
 * @return error Returns nil on success, or error if webhook fails
 */
func (a *Alerter) Send(payload alerts.Payload) error {
	if a.config.WebhookURL == "" {
		return fmt.Errorf("slack webhook URL is empty")
	}

	message := a.buildMessage(payload)

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal slack message: %w", err)
	}

	resp, err := a.client.Post(a.config.WebhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send slack webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func (a *Alerter) buildMessage(payload alerts.Payload) map[string]interface{} {
	color := a.getLevelColor(payload.Level)

	stackText := "No stack trace"
	if len(payload.Stack) > 0 {
		stackText = ""
		for _, frame := range payload.Stack {
			stackText += frame + "\n"
		}
	}

	attachment := map[string]interface{}{
		"color":  color,
		"title":  fmt.Sprintf("🚨 %s Alert", payload.Level),
		"text":   payload.Error,
		"footer": "Go Logging Library",
		"ts":     payload.Timestamp.Unix(),
		"fields": []map[string]interface{}{
			{"title": "Service", "value": payload.ServiceName, "short": true},
			{"title": "Level", "value": payload.Level, "short": true},
			{"title": "Method", "value": payload.Method, "short": true},
			{"title": "Path", "value": payload.Path, "short": true},
			{"title": "Client IP", "value": defaultIfEmpty(payload.IP, "N/A"), "short": true},
			{"title": "Source", "value": fmt.Sprintf("%s:%d", payload.File, payload.Line), "short": true},
			{"title": "Request ID", "value": defaultIfEmpty(payload.RequestID, "N/A"), "short": false},
			{"title": "Stack Trace", "value": "```" + truncate(stackText, 500) + "```", "short": false},
		},
	}

	message := map[string]interface{}{
		"attachments": []map[string]interface{}{attachment},
	}

	if a.config.Channel != "" {
		message["channel"] = a.config.Channel
	}
	if a.config.Username != "" {
		message["username"] = a.config.Username
	}
	if a.config.IconEmoji != "" {
		message["icon_emoji"] = a.config.IconEmoji
	}

	return message
}

func (a *Alerter) getLevelColor(level string) string {
	colors := map[string]string{
		"CRITICAL": "#dc3545",
		"ERROR":    "#fd7e14",
		"WARN":     "#ffc107",
	}
	if color, ok := colors[level]; ok {
		return color
	}
	return "#6c757d"
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

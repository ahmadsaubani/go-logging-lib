package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ahmadsaubani/go-logging-lib/alerts"
)

type Config struct {
	Enabled  bool   `yaml:"enabled"`
	BotToken string `yaml:"bot_token"`
	ChatID   string `yaml:"chat_id"`
}

type Alerter struct {
	config *Config
	client *http.Client
}

/**
 * New creates a new Telegram alerter instance.
 * Uses Telegram Bot API to send HTML-formatted messages to a chat/channel.
 *
 * @param config Telegram bot configuration including token and chat ID
 * @return *Alerter Ready-to-use Telegram alerter
 */
func New(config *Config) *Alerter {
	return &Alerter{
		config: config,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (a *Alerter) Name() string {
	return "Telegram"
}

/**
 * Send dispatches an alert to Telegram via Bot API.
 * Creates an HTML-formatted message with emoji indicators and monospace code blocks.
 *
 * @param payload Alert data containing error details and request metadata
 * @return error Returns nil on success, or error if API call fails
 */
func (a *Alerter) Send(payload alerts.Payload) error {
	if a.config.BotToken == "" || a.config.ChatID == "" {
		return fmt.Errorf("telegram bot token or chat ID is empty")
	}

	message := a.buildMessage(payload)

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", a.config.BotToken)

	body := map[string]interface{}{
		"chat_id":    a.config.ChatID,
		"text":       message,
		"parse_mode": "HTML",
	}

	jsonData, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal telegram message: %w", err)
	}

	resp, err := a.client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send telegram message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("telegram API returned status %d", resp.StatusCode)
	}

	return nil
}

func (a *Alerter) buildMessage(payload alerts.Payload) string {
	emoji := a.getLevelEmoji(payload.Level)

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s <b>%s Alert</b>\n\n", emoji, payload.Level))
	sb.WriteString(fmt.Sprintf("<b>Service:</b> %s\n", escapeHTML(payload.ServiceName)))
	sb.WriteString(fmt.Sprintf("<b>Error:</b> %s\n\n", escapeHTML(payload.Error)))
	sb.WriteString(fmt.Sprintf("<b>Method:</b> %s\n", escapeHTML(payload.Method)))
	sb.WriteString(fmt.Sprintf("<b>Path:</b> <code>%s</code>\n", escapeHTML(payload.Path)))
	sb.WriteString(fmt.Sprintf("<b>Client IP:</b> %s\n", escapeHTML(defaultIfEmpty(payload.IP, "N/A"))))
	sb.WriteString(fmt.Sprintf("<b>Source:</b> <code>%s:%d</code>\n", escapeHTML(payload.File), payload.Line))
	sb.WriteString(fmt.Sprintf("<b>Request ID:</b>\n<code>%s</code>\n\n", escapeHTML(defaultIfEmpty(payload.RequestID, "N/A"))))
	sb.WriteString(fmt.Sprintf("<b>Time:</b> %s\n", payload.Timestamp.Format("02 Jan 2006 15:04:05")))

	if len(payload.Stack) > 0 {
		sb.WriteString("\n<b>Stack Trace:</b>\n<pre>")
		for _, frame := range payload.Stack {
			sb.WriteString(escapeHTML(frame) + "\n")
		}
		sb.WriteString("</pre>")
	}

	return sb.String()
}

func (a *Alerter) getLevelEmoji(level string) string {
	emojis := map[string]string{
		"CRITICAL": "🔴",
		"ERROR":    "🟠",
		"WARN":     "🟡",
	}
	if emoji, ok := emojis[level]; ok {
		return emoji
	}
	return "⚪"
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func defaultIfEmpty(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

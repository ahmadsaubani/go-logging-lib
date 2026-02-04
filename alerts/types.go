package alerts

import "time"

type LogLevel string

const (
	LevelWarn     LogLevel = "WARN"
	LevelError    LogLevel = "ERROR"
	LevelCritical LogLevel = "CRITICAL"
)

/**
 * Alerter defines the interface that all alert providers must implement.
 * This allows for dependency injection and easy addition of new alert channels.
 *
 * Implementations:
 *   - discord.Alerter: Sends alerts via Discord webhooks
 *   - slack.Alerter: Sends alerts via Slack webhooks
 *   - telegram.Alerter: Sends alerts via Telegram Bot API
 *   - email.Alerter: Sends alerts via SMTP email
 */
type Alerter interface {
	Name() string
	Send(payload Payload) error
}

type Payload struct {
	ServiceName string
	Level       string
	Error       string
	RequestID   string
	Method      string
	Path        string
	IP          string
	UserAgent   string
	File        string
	Line        int
	Stack       []string
	Timestamp   time.Time
}

type Config struct {
	Enabled      bool
	MinLevel     LogLevel
	RateLimitSec int
}

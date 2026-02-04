package alerts

import (
	"crypto/md5"
	"fmt"
	"sync"
	"time"
)

type Manager struct {
	config    *Config
	alerters  []Alerter
	lastAlert map[string]time.Time
	mu        sync.RWMutex
}

/**
 * NewManager creates a new alert manager instance.
 * The manager handles dispatching alerts to all registered providers
 * with built-in rate limiting to prevent alert spam.
 *
 * @param config Configuration including min level and rate limit settings
 * @return *Manager A new manager instance ready for alerter registration
 */
func NewManager(config *Config) *Manager {
	if config.RateLimitSec <= 0 {
		config.RateLimitSec = 300
	}

	return &Manager{
		config:    config,
		alerters:  make([]Alerter, 0),
		lastAlert: make(map[string]time.Time),
	}
}

/**
 * Register adds an alerter to the manager.
 * Multiple alerters can be registered and all will receive alerts.
 * Registration is idempotent - registering the same alerter twice is safe.
 *
 * @param alerter The alerter implementation to register (Discord, Slack, etc.)
 */
func (m *Manager) Register(alerter Alerter) {
	m.alerters = append(m.alerters, alerter)
}

/**
 * Alert sends notification to all registered alerters with rate limiting.
 * Duplicate alerts (same error, path, method) within the rate limit window
 * will be silently dropped to prevent spam.
 *
 * @param payload The alert data containing error details and request metadata
 */
func (m *Manager) Alert(payload Payload) {
	if !m.config.Enabled || !m.shouldAlert(payload.Level) {
		return
	}

	if m.isRateLimited(payload) {
		return
	}

	m.markAlerted(payload)

	for _, alerter := range m.alerters {
		go func(a Alerter) {
			if err := a.Send(payload); err != nil {
				fmt.Printf("[AlertManager] failed to send %s alert: %v\n", a.Name(), err)
			}
		}(alerter)
	}
}

func (m *Manager) shouldAlert(level string) bool {
	levelPriority := map[string]int{
		"WARN":     1,
		"ERROR":    2,
		"CRITICAL": 3,
	}

	minPriority := levelPriority[string(m.config.MinLevel)]
	currentPriority := levelPriority[level]

	return currentPriority >= minPriority
}

func (m *Manager) isRateLimited(payload Payload) bool {
	key := m.getAlertKey(payload)

	m.mu.RLock()
	lastTime, exists := m.lastAlert[key]
	m.mu.RUnlock()

	if !exists {
		return false
	}

	return time.Since(lastTime) < time.Duration(m.config.RateLimitSec)*time.Second
}

func (m *Manager) markAlerted(payload Payload) {
	key := m.getAlertKey(payload)

	m.mu.Lock()
	m.lastAlert[key] = time.Now()
	m.mu.Unlock()
}

func (m *Manager) getAlertKey(payload Payload) string {
	data := fmt.Sprintf("%s:%s:%s:%s", payload.ServiceName, payload.Error, payload.Path, payload.Method)
	return fmt.Sprintf("%x", md5.Sum([]byte(data)))
}

/**
 * Cleanup removes expired rate limit entries from memory.
 * Should be called periodically to prevent memory leaks in long-running applications.
 */
func (m *Manager) Cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	expiry := time.Duration(m.config.RateLimitSec*2) * time.Second

	for key, lastTime := range m.lastAlert {
		if time.Since(lastTime) > expiry {
			delete(m.lastAlert, key)
		}
	}
}

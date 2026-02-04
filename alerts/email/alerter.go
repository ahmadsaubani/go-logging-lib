package email

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"net/smtp"
	"strings"

	"github.com/ahmadsaubani/go-logging-lib/alerts"
)

type Config struct {
	Enabled    bool     `yaml:"enabled"`
	SMTPHost   string   `yaml:"smtp_host"`
	SMTPPort   int      `yaml:"smtp_port"`
	Username   string   `yaml:"username"`
	Password   string   `yaml:"password"`
	From       string   `yaml:"from"`
	To         []string `yaml:"to"`
	UseTLS     bool     `yaml:"use_tls"`
	SkipVerify bool     `yaml:"skip_verify"`
}

type Alerter struct {
	config   *Config
	template *template.Template
}

/**
 * New creates a new Email alerter instance.
 * Uses SMTP to send HTML-formatted emails with professional template.
 * Supports both plain SMTP and TLS connections.
 *
 * @param config SMTP configuration including host, credentials, and recipients
 * @return *Alerter Ready-to-use Email alerter
 */
func New(config *Config) *Alerter {
	tmpl := template.Must(template.New("email").Parse(htmlTemplate))
	return &Alerter{
		config:   config,
		template: tmpl,
	}
}

func (a *Alerter) Name() string {
	return "Email"
}

/**
 * Send dispatches an alert via SMTP email.
 * Renders HTML template with error details and sends to all configured recipients.
 * Automatically handles TLS if configured.
 *
 * @param payload Alert data containing error details and request metadata
 * @return error Returns nil on success, or error if SMTP fails
 */
func (a *Alerter) Send(payload alerts.Payload) error {
	if a.config.SMTPHost == "" || len(a.config.To) == 0 {
		return fmt.Errorf("email SMTP host or recipients is empty")
	}

	subject := fmt.Sprintf("[%s] %s - %s", payload.Level, payload.ServiceName, truncate(payload.Error, 50))

	body, err := a.renderTemplate(payload)
	if err != nil {
		return fmt.Errorf("failed to render email template: %w", err)
	}

	message := a.buildMessage(subject, body)
	addr := fmt.Sprintf("%s:%d", a.config.SMTPHost, a.config.SMTPPort)
	auth := a.getAuth()

	if a.config.UseTLS {
		return a.sendWithTLS(addr, auth, message)
	}

	return smtp.SendMail(addr, auth, a.config.From, a.config.To, []byte(message))
}

func (a *Alerter) renderTemplate(payload alerts.Payload) (string, error) {
	data := templateData{
		LevelColor:  getLevelColor(payload.Level),
		MethodColor: getMethodColor(payload.Method),
		Level:       payload.Level,
		ServiceName: payload.ServiceName,
		Timestamp:   payload.Timestamp.Format("02 Jan 2006, 15:04:05"),
		Error:       payload.Error,
		Method:      payload.Method,
		Path:        payload.Path,
		IP:          defaultIfEmpty(payload.IP, "N/A"),
		Source:      fmt.Sprintf("%s:%d", payload.File, payload.Line),
		RequestID:   defaultIfEmpty(payload.RequestID, "N/A"),
		UserAgent:   defaultIfEmpty(payload.UserAgent, "N/A"),
		Stack:       payload.Stack,
		Year:        payload.Timestamp.Year(),
	}

	var buf bytes.Buffer
	if err := a.template.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (a *Alerter) buildMessage(subject, body string) string {
	var msg strings.Builder

	msg.WriteString(fmt.Sprintf("From: %s\r\n", a.config.From))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(a.config.To, ", ")))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(body)

	return msg.String()
}

func (a *Alerter) getAuth() smtp.Auth {
	if a.config.Username != "" && a.config.Password != "" {
		return smtp.PlainAuth("", a.config.Username, a.config.Password, a.config.SMTPHost)
	}
	return nil
}

func (a *Alerter) sendWithTLS(addr string, auth smtp.Auth, message string) error {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: a.config.SkipVerify,
		ServerName:         a.config.SMTPHost,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, a.config.SMTPHost)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth failed: %w", err)
		}
	}

	if err := client.Mail(a.config.From); err != nil {
		return fmt.Errorf("SMTP MAIL command failed: %w", err)
	}

	for _, to := range a.config.To {
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("SMTP RCPT command failed: %w", err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA command failed: %w", err)
	}

	if _, err = w.Write([]byte(message)); err != nil {
		return fmt.Errorf("failed to write email body: %w", err)
	}

	if err = w.Close(); err != nil {
		return fmt.Errorf("failed to close email writer: %w", err)
	}

	return client.Quit()
}

func getLevelColor(level string) string {
	colors := map[string]string{
		"CRITICAL": "#ef4444",
		"ERROR":    "#f97316",
		"WARN":     "#eab308",
	}
	if color, ok := colors[level]; ok {
		return color
	}
	return "#6b7280"
}

func getMethodColor(method string) string {
	colors := map[string]string{
		"GET":    "#22c55e",
		"POST":   "#3b82f6",
		"PUT":    "#f97316",
		"PATCH":  "#eab308",
		"DELETE": "#ef4444",
	}
	if color, ok := colors[method]; ok {
		return color
	}
	return "#6b7280"
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

package logging

import (
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DailyWriter implements daily log rotation
type DailyWriter struct {
	mu       sync.Mutex
	basePath string
	file     *os.File
	current  string
}

// NewDailyWriter creates a new daily rotating writer
func NewDailyWriter(basePath string) (*DailyWriter, error) {
	w := &DailyWriter{
		basePath: basePath,
	}
	if err := w.rotateIfNeeded(); err != nil {
		return nil, err
	}
	return w, nil
}

// Write implements the io.Writer interface
func (w *DailyWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.rotateIfNeeded(); err != nil {
		return 0, err
	}
	return w.file.Write(p)
}

// rotateIfNeeded rotates the log file if the date has changed
func (w *DailyWriter) rotateIfNeeded() error {
	today := time.Now().Format("2006-01-02")

	if w.file != nil && w.current == today {
		return nil
	}

	if w.file != nil {
		_ = w.file.Close()
	}

	dir := filepath.Dir(w.basePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	filename := w.basePath + "-" + today + ".log"
	file, err := os.OpenFile(
		filename,
		os.O_CREATE|os.O_APPEND|os.O_WRONLY,
		0644,
	)
	if err != nil {
		return err
	}

	w.file = file
	w.current = today
	return nil
}

// Close closes the current file
func (w *DailyWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}
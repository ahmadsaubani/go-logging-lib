package logging

import (
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DailyWriter implements daily log rotation
type DailyWriter struct {
	mu             sync.Mutex
	basePath       string
	file           *os.File
	current        string
	enableRotation bool
}

// NewDailyWriter creates a new daily rotating writer
func NewDailyWriter(basePath string, enableRotation bool) (*DailyWriter, error) {
	w := &DailyWriter{
		basePath:       basePath,
		enableRotation: enableRotation,
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

// rotateIfNeeded rotates the log file if the date has changed (when rotation is enabled)
func (w *DailyWriter) rotateIfNeeded() error {
	today := time.Now().Format("2006-01-02")

	// If rotation is disabled, only open file once
	if !w.enableRotation {
		if w.file != nil {
			return nil
		}
		return w.openFile(w.basePath + ".log")
	}

	// Daily rotation logic
	if w.file != nil && w.current == today {
		return nil
	}

	if w.file != nil {
		_ = w.file.Close()
	}

	filename := w.basePath + "-" + today + ".log"
	if err := w.openFile(filename); err != nil {
		return err
	}

	w.current = today
	return nil
}

// openFile opens the specified file
func (w *DailyWriter) openFile(filename string) error {
	dir := filepath.Dir(w.basePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	file, err := os.OpenFile(
		filename,
		os.O_CREATE|os.O_APPEND|os.O_WRONLY,
		0644,
	)
	if err != nil {
		return err
	}

	w.file = file
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
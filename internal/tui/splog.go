// Package tui provides terminal user interface components and utilities.
package tui

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
)

// simpleHandler is a custom slog handler that writes messages without timestamps or level prefixes
type simpleHandler struct {
	writer    io.Writer
	debugMode bool
}

func (h *simpleHandler) Enabled(_ context.Context, level slog.Level) bool {
	// Debug messages only enabled in debug mode
	if level == slog.LevelDebug {
		return h.debugMode
	}
	// Info, Warn, and Error are always enabled
	return true
}

func (h *simpleHandler) Handle(_ context.Context, record slog.Record) error {
	_, err := fmt.Fprintln(h.writer, record.Message)
	return err
}

func (h *simpleHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	return h
}

func (h *simpleHandler) WithGroup(_ string) slog.Handler {
	return h
}

// Splog provides structured logging and output
type Splog struct {
	logger *slog.Logger
	writer *os.File
}

// NewSplog creates a new splog instance
// Debug messages are enabled when the DEBUG environment variable is set
func NewSplog() *Splog {
	writer := os.Stdout
	debugMode := os.Getenv("DEBUG") != ""
	handler := &simpleHandler{
		writer:    writer,
		debugMode: debugMode,
	}
	logger := slog.New(handler)
	return &Splog{
		logger: logger,
		writer: writer,
	}
}

// logMessage is a helper to log a message using slog without format string validation
func (s *Splog) logMessage(level slog.Level, msg string) {
	s.logger.Log(context.Background(), level, msg)
}

// Info writes an info message
// The format parameter may be a variable string, which is safe as we use fmt.Sprintf internally
// nolint // format string validation is handled internally via fmt.Sprintf
func (s *Splog) Info(format string, args ...interface{}) {
	var msg string
	if len(args) == 0 {
		msg = format
	} else {
		msg = fmt.Sprintf(format, args...)
	}
	s.logMessage(slog.LevelInfo, msg)
}

// Page writes output that should be paged (for now, just print)
func (s *Splog) Page(content string) {
	_, _ = fmt.Fprint(s.writer, content)
}

// Newline writes a newline
func (s *Splog) Newline() {
	_, _ = fmt.Fprintln(s.writer)
}

// Warn writes a warning message
// The format parameter may be a variable string, which is safe as we use fmt.Sprintf internally
// nolint // format string validation is handled internally via fmt.Sprintf
func (s *Splog) Warn(format string, args ...interface{}) {
	var msg string
	if len(args) == 0 {
		msg = "⚠️  " + format
	} else {
		msg = fmt.Sprintf("⚠️  "+format, args...)
	}
	s.logMessage(slog.LevelWarn, msg)
}

// Debug writes a debug message
// The format parameter may be a variable string, which is safe as we use fmt.Sprintf internally
// nolint // format string validation is handled internally via fmt.Sprintf
func (s *Splog) Debug(format string, args ...interface{}) {
	var msg string
	if len(args) == 0 {
		msg = format
	} else {
		msg = fmt.Sprintf(format, args...)
	}
	s.logMessage(slog.LevelDebug, msg)
}

// Tip writes a tip message
// The format parameter may be a variable string, which is safe as we use fmt.Sprintf internally
// nolint // format string validation is handled internally via fmt.Sprintf
func (s *Splog) Tip(format string, args ...interface{}) {
	var msg string
	if len(args) == 0 {
		msg = "💡 " + format
	} else {
		msg = fmt.Sprintf("💡 "+format, args...)
	}
	s.logMessage(slog.LevelInfo, msg)
}

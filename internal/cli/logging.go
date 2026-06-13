package cli

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"
)

// Port of the Rust `logging.rs` (was log4rs). Mirrors `--log-level` /
// `--log-target console|file`, the level mapping, and the log line format
// `{d(%Y-%m-%d %H:%M:%S)} - {l} - [{M}] - {m}`.

const (
	// fileAppenderName mirrors FILE_APPENDER_NAME.
	fileAppenderName = "file"
	// consoleAppenderName mirrors CONSOLE_APPENDER_NAME.
	consoleAppenderName = "console"
)

// getLogFilePath mirrors the Rust `get_log_file_path` ("sm.log").
func getLogFilePath() string {
	return "sm.log"
}

// LevelOff is a sentinel slog level mapping LevelFilter::Off (disables logging).
// log4rs Off is "no records"; slog has no Off, so we use a very high level.
const LevelOff = slog.Level(1 << 30)

// getLoggingLevelFromString mirrors the Rust `get_logging_level_from_string`.
// Unknown values default to Info (LevelFilter default arm).
func getLoggingLevelFromString(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "error":
		return slog.LevelError
	case "warn":
		return slog.LevelWarn
	case "trace":
		// slog has no Trace; map below Debug like log4rs Trace < Debug.
		return slog.LevelDebug - 4
	case "off":
		return LevelOff
	default:
		return slog.LevelInfo
	}
}

// GetLogger builds a configured *slog.Logger for the requested level and target.
// It is the slog port of `get_logging_config` + appender construction.
func GetLogger(loggingLevel, logTarget string) *slog.Logger {
	level := getLoggingLevelFromString(loggingLevel)

	var w io.Writer
	switch logTarget {
	case fileAppenderName:
		logPath := getLogFilePath()
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			panic(fmt.Sprintf("unable to create log file '%s'", logPath))
		}
		w = f
	default:
		_ = consoleAppenderName
		w = os.Stdout
	}

	handler := &patternHandler{w: w, level: level}
	return slog.New(handler)
}

// InitLogging builds the logger for the requested level/target and installs it
// as the slog default. Port of `log4rs::init_config(get_logging_config(...))`.
func InitLogging(loggingLevel, logTarget string) {
	slog.SetDefault(GetLogger(loggingLevel, logTarget))
}

// patternHandler is a slog.Handler that renders records in the log4rs pattern
// `{d(%Y-%m-%d %H:%M:%S)} - {l} - [{M}] - {m}` and applies a threshold filter
// (records below the configured level are dropped).
type patternHandler struct {
	w     io.Writer
	level slog.Level
	attrs []slog.Attr
	group string
}

// Enabled mirrors the ThresholdFilter: a record is emitted only when its level
// is at or above the configured threshold.
func (h *patternHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

// levelString maps an slog level to the log4rs level token `{l}`.
func levelString(level slog.Level) string {
	switch {
	case level >= slog.LevelError:
		return "ERROR"
	case level >= slog.LevelWarn:
		return "WARN"
	case level >= slog.LevelInfo:
		return "INFO"
	case level >= slog.LevelDebug:
		return "DEBUG"
	default:
		return "TRACE"
	}
}

// Handle renders a record following the log4rs pattern encoder.
func (h *patternHandler) Handle(_ context.Context, r slog.Record) error {
	ts := r.Time
	if ts.IsZero() {
		ts = time.Now()
	}

	var sb strings.Builder
	sb.WriteString(ts.Format("2006-01-02 15:04:05"))
	sb.WriteString(" - ")
	sb.WriteString(levelString(r.Level))
	sb.WriteString(" - [")
	// {M} is the module/target; slog has no module so use the group if set.
	sb.WriteString(h.group)
	sb.WriteString("] - ")
	sb.WriteString(r.Message)

	r.Attrs(func(a slog.Attr) bool {
		sb.WriteString(" ")
		sb.WriteString(a.Key)
		sb.WriteString("=")
		sb.WriteString(a.Value.String())
		return true
	})
	sb.WriteString("\n")

	_, err := io.WriteString(h.w, sb.String())
	return err
}

// WithAttrs returns a handler with the supplied attributes appended.
func (h *patternHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	nh := *h
	nh.attrs = append(append([]slog.Attr(nil), h.attrs...), attrs...)
	return &nh
}

// WithGroup returns a handler with the group name recorded for the `{M}` token.
func (h *patternHandler) WithGroup(name string) slog.Handler {
	nh := *h
	nh.group = name
	return &nh
}

package zlog

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

// logLevelMap maps string log level names to zerolog levels.
var logLevelMap = map[string]zerolog.Level{
	"debug": zerolog.DebugLevel,
	"info":  zerolog.InfoLevel,
	"warn":  zerolog.WarnLevel,
	"error": zerolog.ErrorLevel,
	"fatal": zerolog.FatalLevel,
}

// Logger wraps zerolog.Logger with convenience methods.
type Logger struct {
	zerolog.Logger
}

// LogConfig contains configuration for the logger.
type LogConfig struct {
	LogLevel    string
	LogDir      string
	LogFileName string
	MaxSize     int // Maximum file size in MB
	MaxBackups  int // Maximum number of old log files to retain
	MaxAge      int // Maximum number of days to retain old log files
}

// NewLogger creates a new logger that outputs to both console and file.
func NewLogger(config LogConfig) *Logger {
	level := parseLogLevel(config.LogLevel)
	zerolog.SetGlobalLevel(level)

	ensureLogDir(config.LogDir)

	writers := buildWriters(config)
	multiWriter := io.MultiWriter(writers...)

	logger := zerolog.New(multiWriter).
		With().
		Timestamp().
		Caller().
		Logger()

	return &Logger{logger}
}

// parseLogLevel converts a string log level to zerolog.Level.
func parseLogLevel(levelStr string) zerolog.Level {
	if level, ok := logLevelMap[levelStr]; ok {
		return level
	}
	return zerolog.InfoLevel
}

// ensureLogDir creates the log directory if it doesn't exist.
func ensureLogDir(logDir string) {
	if logDir == "" {
		return
	}
	if err := os.MkdirAll(logDir, os.ModePerm); err != nil {
		log.Error().Err(err).Msg("Failed to create log directory")
	}
}

// buildWriters creates the console and file writers for the logger.
func buildWriters(config LogConfig) []io.Writer {
	writers := []io.Writer{
		zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		},
	}

	if config.LogFileName != "" {
		fileWriter := createFileWriter(config)
		writers = append(writers, fileWriter)
	}

	return writers
}

// createFileWriter creates a rotating file writer.
func createFileWriter(config LogConfig) io.Writer {
	logPath := filepath.Join(config.LogDir, config.LogFileName)
	return &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    config.MaxSize,
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge,
		Compress:   true,
	}
}

// Debug returns a debug-level event.
func (l *Logger) Debug() *zerolog.Event {
	return l.Logger.Debug()
}

// Info returns an info-level event.
func (l *Logger) Info() *zerolog.Event {
	return l.Logger.Info()
}

// Warn returns a warn-level event.
func (l *Logger) Warn() *zerolog.Event {
	return l.Logger.Warn()
}

// Error returns an error-level event.
func (l *Logger) Error() *zerolog.Event {
	return l.Logger.Error()
}

// Fatal returns a fatal-level event.
func (l *Logger) Fatal() *zerolog.Event {
	return l.Logger.Fatal()
}

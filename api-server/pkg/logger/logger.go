package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// L is the global logger instance
var L *zap.SugaredLogger

// Init initializes the logger with the given debug mode
func Init(debug bool) {
	var config zap.Config

	if debug {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		config = zap.NewProductionConfig()
	}

	config.EncoderConfig.TimeKey = "time"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.CallerKey = "caller"
	config.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	logger, err := config.Build(zap.AddCallerSkip(1))
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}

	L = logger.Sugar()
}

// Sync flushes any buffered log entries
func Sync() {
	if L != nil {
		_ = L.Sync()
	}
}

// Debug logs a debug message with key-value pairs
func Debug(msg string, keysAndValues ...interface{}) {
	L.Debugw(msg, keysAndValues...)
}

// Info logs an info message with key-value pairs
func Info(msg string, keysAndValues ...interface{}) {
	L.Infow(msg, keysAndValues...)
}

// Warn logs a warning message with key-value pairs
func Warn(msg string, keysAndValues ...interface{}) {
	L.Warnw(msg, keysAndValues...)
}

// Error logs an error message with key-value pairs
func Error(msg string, keysAndValues ...interface{}) {
	L.Errorw(msg, keysAndValues...)
}

// Fatal logs a fatal message and exits
func Fatal(msg string, keysAndValues ...interface{}) {
	L.Fatalw(msg, keysAndValues...)
	os.Exit(1)
}

// WithFields returns a logger with the given fields
func WithFields(keysAndValues ...interface{}) *zap.SugaredLogger {
	return L.With(keysAndValues...)
}


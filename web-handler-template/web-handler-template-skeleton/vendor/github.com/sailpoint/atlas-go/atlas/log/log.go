// Copyright (c) 2020-2022. SailPoint Technologies, Inc. All rights reserved.

package log

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// globalLogger is the default logger used when the context doesn't contain
// a logger.
var globalLogger *zap.Logger

var _level zap.AtomicLevel

type contextKey int

const (
	loggerKey contextKey = iota
)

func init() {
	globalLogger, _ = zap.NewDevelopment()
	_level = zap.NewAtomicLevel()
	zap.RedirectStdLog(globalLogger)
}

// ConfigureJSON sets up logging to match our expected production output when running
// in AWS.
func ConfigureJSON(stack string) {
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.MessageKey = "message"
	encoderCfg.TimeKey = "@timestamp"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderCfg.EncodeLevel = zapcore.CapitalLevelEncoder

	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig = encoderCfg
	cfg.OutputPaths = []string{"stdout"}
	cfg.ErrorOutputPaths = []string{"stdout"}

	if stack != "" {
		cfg.InitialFields = map[string]interface{}{
			"stack": stack,
		}
	}
	cfg.Level = _level

	var err error
	globalLogger, err = cfg.Build()
	if err != nil {
		panic(err)
	}
	zap.RedirectStdLog(globalLogger)
}

// SetLevel sets log level, use zap.<Level>level for value, e.g. zap.DebugLevel
func SetLevel(level zapcore.Level) {
	_level.SetLevel(level)
}

// Global returns the global logger instance.
func Global() *zap.Logger {
	return globalLogger
}

// With derives a new context from ctx that contains the specified logger.
func With(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// WithFields derives a new context ctx that contains a logger with the additional derived fields.
func WithFields(ctx context.Context, fields ...zap.Field) context.Context {
	return With(ctx, Get(ctx).With(fields...))
}

// Get loads a logger out of the specified context.
// Returns the global logger if the context doesn't have an associated logger.
func Get(ctx context.Context) *zap.Logger {
	if ctx == nil {
		return globalLogger
	}

	logger := ctx.Value(loggerKey)

	if logger == nil {
		return globalLogger
	}

	return logger.(*zap.Logger)
}

// GetSugar gets a sugared logger out of the specified context.
func GetSugar(ctx context.Context) *zap.SugaredLogger {
	return Get(ctx).Sugar()
}

// Debug writes a debug-level log message to the log associated with ctx, using the globalLogger
// no log is associated.
func Debug(ctx context.Context, args ...interface{}) {
	GetSugar(ctx).Debug(args...)
}

// Debugf writes a formatted debug-level log message to the log associated with ctx, using the globalLogger
// no log is associated.
func Debugf(ctx context.Context, format string, args ...interface{}) {
	GetSugar(ctx).Debugf(format, args...)
}

// Info writes an info-level log message to the log associated with ctx, using the globalLogger
// no log is associated.
func Info(ctx context.Context, args ...interface{}) {
	GetSugar(ctx).Info(args...)
}

// Infof writes a formatted info-level log message to the log associated with ctx, using the globalLogger
// no log is associated.
func Infof(ctx context.Context, format string, args ...interface{}) {
	GetSugar(ctx).Infof(format, args...)
}

// Warn writes a warn-level log message to the log associated with ctx, using the globalLogger
// no log is associated.
func Warn(ctx context.Context, args ...interface{}) {
	GetSugar(ctx).Warn(args...)
}

// Warnf writes a formatted warn-level log message to the log associated with ctx, using the globalLogger
// no log is associated.
func Warnf(ctx context.Context, format string, args ...interface{}) {
	GetSugar(ctx).Warnf(format, args...)
}

// Error writes an error-level log message to the log associated with ctx, using the globalLogger
// no log is associated.
func Error(ctx context.Context, args ...interface{}) {
	GetSugar(ctx).Error(args...)
}

// Errorf writes a formatted error-level log message to the log associated with ctx, using the globalLogger
// no log is associated.
func Errorf(ctx context.Context, format string, args ...interface{}) {
	GetSugar(ctx).Errorf(format, args...)
}

// Fatal writes a fatal-level log message to the log associated with ctx, using the globalLogger
// no log is associated. The program then terminates with a non-zero error code.
func Fatal(ctx context.Context, args ...interface{}) {
	GetSugar(ctx).Fatal(args...)
}

// Fatalf writes a formatted fatal-level log message to the log associated with ctx, using the globalLogger
// no log is associated. The program then terminates with a non-zero error code.
func Fatalf(ctx context.Context, format string, args ...interface{}) {
	GetSugar(ctx).Fatalf(format, args...)
}

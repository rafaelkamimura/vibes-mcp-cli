package telemetry

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// TelemetryLogger wraps a zap logger and sends logs to telemetry
type TelemetryLogger struct {
	client    Client
	zapLogger *zap.Logger
	component string
	ctx       context.Context
}

// NewTelemetryLogger creates a new telemetry-enabled logger
func NewTelemetryLogger(client Client, zapLogger *zap.Logger, component string) *TelemetryLogger {
	if zapLogger == nil {
		zapLogger = zap.NewNop()
	}
	
	return &TelemetryLogger{
		client:    client,
		zapLogger: zapLogger,
		component: component,
		ctx:       context.Background(),
	}
}

// WithContext returns a copy of the logger with the given context
func (l *TelemetryLogger) WithContext(ctx context.Context) *TelemetryLogger {
	return &TelemetryLogger{
		client:    l.client,
		zapLogger: l.zapLogger,
		component: l.component,
		ctx:       ctx,
	}
}

// WithComponent returns a copy of the logger with the given component name
func (l *TelemetryLogger) WithComponent(component string) *TelemetryLogger {
	return &TelemetryLogger{
		client:    l.client,
		zapLogger: l.zapLogger,
		component: component,
		ctx:       l.ctx,
	}
}

// Debug logs a debug message
func (l *TelemetryLogger) Debug(msg string, fields ...zap.Field) {
	l.zapLogger.Debug(msg, fields...)
	l.sendTelemetry(LogLevelDebug, msg, fields, nil)
}

// Info logs an info message
func (l *TelemetryLogger) Info(msg string, fields ...zap.Field) {
	l.zapLogger.Info(msg, fields...)
	l.sendTelemetry(LogLevelInfo, msg, fields, nil)
}

// Warn logs a warning message
func (l *TelemetryLogger) Warn(msg string, fields ...zap.Field) {
	l.zapLogger.Warn(msg, fields...)
	l.sendTelemetry(LogLevelWarn, msg, fields, nil)
}

// Error logs an error message
func (l *TelemetryLogger) Error(msg string, fields ...zap.Field) {
	l.zapLogger.Error(msg, fields...)
	l.sendTelemetry(LogLevelError, msg, fields, nil)
}

// ErrorWithContext logs an error with additional context
func (l *TelemetryLogger) ErrorWithContext(msg string, err error, fields ...zap.Field) {
	l.zapLogger.Error(msg, append(fields, zap.Error(err))...)
	l.sendTelemetry(LogLevelError, msg, fields, err)
}

// Fatal logs a fatal message and exits
func (l *TelemetryLogger) Fatal(msg string, fields ...zap.Field) {
	l.zapLogger.Fatal(msg, fields...)
	l.sendTelemetry(LogLevelError, msg, fields, nil)
}

// Panic logs a panic message and panics
func (l *TelemetryLogger) Panic(msg string, fields ...zap.Field) {
	l.zapLogger.Panic(msg, fields...)
	l.sendTelemetry(LogLevelError, msg, fields, nil)
}

// Sync flushes any buffered log entries
func (l *TelemetryLogger) Sync() error {
	if err := l.zapLogger.Sync(); err != nil {
		return err
	}
	
	if l.client != nil {
		return l.client.Flush(l.ctx)
	}
	
	return nil
}

// With adds fields to the logger
func (l *TelemetryLogger) With(fields ...zap.Field) *TelemetryLogger {
	return &TelemetryLogger{
		client:    l.client,
		zapLogger: l.zapLogger.With(fields...),
		component: l.component,
		ctx:       l.ctx,
	}
}

// Named adds a name to the logger
func (l *TelemetryLogger) Named(name string) *TelemetryLogger {
	component := l.component
	if component != "" {
		component = component + "." + name
	} else {
		component = name
	}
	
	return &TelemetryLogger{
		client:    l.client,
		zapLogger: l.zapLogger.Named(name),
		component: component,
		ctx:       l.ctx,
	}
}

// sendTelemetry sends a log entry to the telemetry system
func (l *TelemetryLogger) sendTelemetry(level LogLevel, msg string, fields []zap.Field, err error) {
	if l.client == nil || !l.client.IsEnabled() {
		return
	}
	
	entry := LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     level,
		Message:   msg,
		Component: l.component,
		Metadata:  l.fieldsToMetadata(fields),
	}
	
	// Add error context if present
	if err != nil {
		entry.Error = &ErrorContext{
			Type:       fmt.Sprintf("%T", err),
			StackTrace: l.getStackTrace(),
		}
	}
	
	// Send asynchronously - don't block on telemetry failures
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		if err := l.client.Log(ctx, entry); err != nil {
			// Only log telemetry failures to zap, not back to telemetry
			l.zapLogger.Debug("Failed to send telemetry", zap.Error(err))
		}
	}()
}

// fieldsToMetadata converts zap fields to metadata map
func (l *TelemetryLogger) fieldsToMetadata(fields []zap.Field) map[string]interface{} {
	if len(fields) == 0 {
		return nil
	}
	
	metadata := make(map[string]interface{})
	
	for _, field := range fields {
		switch field.Type {
		case zapcore.StringType:
			metadata[field.Key] = field.String
		case zapcore.Int64Type, zapcore.Int32Type, zapcore.Int16Type, zapcore.Int8Type:
			metadata[field.Key] = field.Integer
		case zapcore.Uint64Type, zapcore.Uint32Type, zapcore.Uint16Type, zapcore.Uint8Type:
			metadata[field.Key] = field.Integer
		case zapcore.Float64Type, zapcore.Float32Type:
			metadata[field.Key] = field.Integer // zap stores floats as int64 in memory
		case zapcore.BoolType:
			metadata[field.Key] = field.Integer == 1
		case zapcore.TimeType:
			if field.Interface != nil {
				metadata[field.Key] = field.Interface
			}
		case zapcore.DurationType:
			metadata[field.Key] = time.Duration(field.Integer).String()
		case zapcore.ErrorType:
			if field.Interface != nil {
				metadata[field.Key] = field.Interface.(error).Error()
			}
		case zapcore.ReflectType:
			metadata[field.Key] = field.Interface
		default:
			// For unknown types, try to convert to string
			metadata[field.Key] = fmt.Sprintf("%v", field.Interface)
		}
	}
	
	return metadata
}

// getStackTrace captures the current stack trace
func (l *TelemetryLogger) getStackTrace() string {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(3, pcs[:]) // Skip getStackTrace, sendTelemetry, and the logging method
	
	if n == 0 {
		return ""
	}
	
	frames := runtime.CallersFrames(pcs[:n])
	var lines []string
	
	for {
		frame, more := frames.Next()
		if !more {
			break
		}
		
		lines = append(lines, fmt.Sprintf("%s:%d in %s", frame.File, frame.Line, frame.Function))
	}
	
	return strings.Join(lines, "\n")
}

// TelemetryCore is a zapcore.Core that sends logs to telemetry
type TelemetryCore struct {
	client    Client
	component string
	level     zapcore.Level
}

// NewTelemetryCore creates a new zapcore.Core that sends logs to telemetry
func NewTelemetryCore(client Client, component string, level zapcore.Level) zapcore.Core {
	return &TelemetryCore{
		client:    client,
		component: component,
		level:     level,
	}
}

// Enabled returns whether the given level is enabled
func (c *TelemetryCore) Enabled(level zapcore.Level) bool {
	return c.client.IsEnabled() && level >= c.level
}

// With adds fields to the core
func (c *TelemetryCore) With(fields []zapcore.Field) zapcore.Core {
	// For simplicity, return the same core since we handle fields in Check
	return c
}

// Check checks if an entry should be logged
func (c *TelemetryCore) Check(entry zapcore.Entry, checkedEntry *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return checkedEntry.AddCore(entry, c)
	}
	return checkedEntry
}

// Write writes a log entry
func (c *TelemetryCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	if !c.client.IsEnabled() {
		return nil
	}
	
	level := c.zapLevelToTelemetryLevel(entry.Level)
	
	telemetryEntry := LogEntry{
		Timestamp: entry.Time,
		Level:     level,
		Message:   entry.Message,
		Component: c.component,
		Metadata:  c.fieldsToMetadata(fields),
	}
	
	// Add error context if it's an error level log
	if entry.Level >= zapcore.ErrorLevel {
		telemetryEntry.Error = &ErrorContext{
			Type: "log_error",
		}
	}
	
	// Send asynchronously
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		c.client.Log(ctx, telemetryEntry)
	}()
	
	return nil
}

// Sync flushes buffered logs
func (c *TelemetryCore) Sync() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	return c.client.Flush(ctx)
}

// zapLevelToTelemetryLevel converts zap log level to telemetry log level
func (c *TelemetryCore) zapLevelToTelemetryLevel(level zapcore.Level) LogLevel {
	switch level {
	case zapcore.DebugLevel:
		return LogLevelDebug
	case zapcore.InfoLevel:
		return LogLevelInfo
	case zapcore.WarnLevel:
		return LogLevelWarn
	case zapcore.ErrorLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		return LogLevelError
	default:
		return LogLevelInfo
	}
}

// fieldsToMetadata converts zapcore fields to metadata map
func (c *TelemetryCore) fieldsToMetadata(fields []zapcore.Field) map[string]interface{} {
	if len(fields) == 0 {
		return nil
	}
	
	metadata := make(map[string]interface{})
	
	for _, field := range fields {
		switch field.Type {
		case zapcore.StringType:
			metadata[field.Key] = field.String
		case zapcore.Int64Type, zapcore.Int32Type, zapcore.Int16Type, zapcore.Int8Type:
			metadata[field.Key] = field.Integer
		case zapcore.Uint64Type, zapcore.Uint32Type, zapcore.Uint16Type, zapcore.Uint8Type:
			metadata[field.Key] = field.Integer
		case zapcore.Float64Type, zapcore.Float32Type:
			metadata[field.Key] = field.Integer
		case zapcore.BoolType:
			metadata[field.Key] = field.Integer == 1
		case zapcore.TimeType:
			if field.Interface != nil {
				metadata[field.Key] = field.Interface
			}
		case zapcore.DurationType:
			metadata[field.Key] = time.Duration(field.Integer).String()
		case zapcore.ErrorType:
			if field.Interface != nil {
				metadata[field.Key] = field.Interface.(error).Error()
			}
		case zapcore.ReflectType:
			metadata[field.Key] = field.Interface
		default:
			metadata[field.Key] = fmt.Sprintf("%v", field.Interface)
		}
	}
	
	return metadata
}
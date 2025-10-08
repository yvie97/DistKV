// Package logging provides a centralized, structured logging system for DistKV.
// It supports multiple log levels, structured fields, and consistent formatting.
package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Logger provides structured logging capabilities
type Logger struct {
	level      LogLevel
	component  string
	fields     map[string]interface{}
	output     io.Writer
	mutex      sync.Mutex
	timeFormat string
}

// LogConfig holds logger configuration
type LogConfig struct {
	Level      LogLevel
	Component  string
	Output     io.Writer
	TimeFormat string
}

// DefaultLogConfig returns default logging configuration
func DefaultLogConfig() *LogConfig {
	return &LogConfig{
		Level:      INFO,
		Component:  "distkv",
		Output:     os.Stdout,
		TimeFormat: "2006-01-02 15:04:05.000",
	}
}

var (
	globalLogger *Logger
	once         sync.Once
)

// InitGlobalLogger initializes the global logger
func InitGlobalLogger(config *LogConfig) {
	once.Do(func() {
		if config == nil {
			config = DefaultLogConfig()
		}
		globalLogger = NewLogger(config)
	})
}

// GetGlobalLogger returns the global logger instance
func GetGlobalLogger() *Logger {
	if globalLogger == nil {
		InitGlobalLogger(nil)
	}
	return globalLogger
}

// NewLogger creates a new logger instance
func NewLogger(config *LogConfig) *Logger {
	if config == nil {
		config = DefaultLogConfig()
	}

	return &Logger{
		level:      config.Level,
		component:  config.Component,
		fields:     make(map[string]interface{}),
		output:     config.Output,
		timeFormat: config.TimeFormat,
	}
}

// WithComponent creates a new logger with a specific component name
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		level:      l.level,
		component:  component,
		fields:     copyFields(l.fields),
		output:     l.output,
		timeFormat: l.timeFormat,
	}
}

// WithField adds a field to the logger context
func (l *Logger) WithField(key string, value interface{}) *Logger {
	newFields := copyFields(l.fields)
	newFields[key] = value

	return &Logger{
		level:      l.level,
		component:  l.component,
		fields:     newFields,
		output:     l.output,
		timeFormat: l.timeFormat,
	}
}

// WithFields adds multiple fields to the logger context
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	newFields := copyFields(l.fields)
	for k, v := range fields {
		newFields[k] = v
	}

	return &Logger{
		level:      l.level,
		component:  l.component,
		fields:     newFields,
		output:     l.output,
		timeFormat: l.timeFormat,
	}
}

// WithError adds an error field to the logger
func (l *Logger) WithError(err error) *Logger {
	if err == nil {
		return l
	}
	return l.WithField("error", err.Error())
}

// SetLevel sets the logging level
func (l *Logger) SetLevel(level LogLevel) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.level = level
}

// Debug logs a debug message
func (l *Logger) Debug(message string) {
	l.log(DEBUG, message)
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log(DEBUG, fmt.Sprintf(format, args...))
}

// Info logs an info message
func (l *Logger) Info(message string) {
	l.log(INFO, message)
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(INFO, fmt.Sprintf(format, args...))
}

// Warn logs a warning message
func (l *Logger) Warn(message string) {
	l.log(WARN, message)
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log(WARN, fmt.Sprintf(format, args...))
}

// Error logs an error message
func (l *Logger) Error(message string) {
	l.log(ERROR, message)
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(ERROR, fmt.Sprintf(format, args...))
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(message string) {
	l.log(FATAL, message)
	os.Exit(1)
}

// Fatalf logs a formatted fatal message and exits
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.log(FATAL, fmt.Sprintf(format, args...))
	os.Exit(1)
}

// log is the internal logging method
func (l *Logger) log(level LogLevel, message string) {
	// Check if we should log at this level
	if level < l.level {
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	// Build log entry
	timestamp := time.Now().Format(l.timeFormat)

	// Get caller information
	var caller string
	if pc, file, line, ok := runtime.Caller(2); ok {
		funcName := runtime.FuncForPC(pc).Name()
		// Extract just the function name
		parts := strings.Split(funcName, "/")
		funcName = parts[len(parts)-1]
		// Extract just the file name
		fileParts := strings.Split(file, "/")
		file = fileParts[len(fileParts)-1]
		caller = fmt.Sprintf("%s:%d %s", file, line, funcName)
	}

	// Build fields string
	var fieldsStr string
	if len(l.fields) > 0 {
		var parts []string
		for k, v := range l.fields {
			parts = append(parts, fmt.Sprintf("%s=%v", k, v))
		}
		fieldsStr = " [" + strings.Join(parts, ", ") + "]"
	}

	// Format: [TIMESTAMP] [LEVEL] [COMPONENT] message [fields] (caller)
	logLine := fmt.Sprintf("[%s] [%s] [%s] %s%s (%s)\n",
		timestamp,
		level.String(),
		l.component,
		message,
		fieldsStr,
		caller,
	)

	// Write to output
	fmt.Fprint(l.output, logLine)
}

// copyFields creates a copy of the fields map
func copyFields(fields map[string]interface{}) map[string]interface{} {
	copy := make(map[string]interface{}, len(fields))
	for k, v := range fields {
		copy[k] = v
	}
	return copy
}

// Package-level convenience functions using the global logger

// Debug logs a debug message using the global logger
func Debug(message string) {
	GetGlobalLogger().Debug(message)
}

// Debugf logs a formatted debug message using the global logger
func Debugf(format string, args ...interface{}) {
	GetGlobalLogger().Debugf(format, args...)
}

// Info logs an info message using the global logger
func Info(message string) {
	GetGlobalLogger().Info(message)
}

// Infof logs a formatted info message using the global logger
func Infof(format string, args ...interface{}) {
	GetGlobalLogger().Infof(format, args...)
}

// Warn logs a warning message using the global logger
func Warn(message string) {
	GetGlobalLogger().Warn(message)
}

// Warnf logs a formatted warning message using the global logger
func Warnf(format string, args ...interface{}) {
	GetGlobalLogger().Warnf(format, args...)
}

// Error logs an error message using the global logger
func Error(message string) {
	GetGlobalLogger().Error(message)
}

// Errorf logs a formatted error message using the global logger
func Errorf(format string, args ...interface{}) {
	GetGlobalLogger().Errorf(format, args...)
}

// Fatal logs a fatal message using the global logger and exits
func Fatal(message string) {
	GetGlobalLogger().Fatal(message)
}

// Fatalf logs a formatted fatal message using the global logger and exits
func Fatalf(format string, args ...interface{}) {
	GetGlobalLogger().Fatalf(format, args...)
}

// WithComponent creates a logger with a component name
func WithComponent(component string) *Logger {
	return GetGlobalLogger().WithComponent(component)
}

// WithField creates a logger with an additional field
func WithField(key string, value interface{}) *Logger {
	return GetGlobalLogger().WithField(key, value)
}

// WithFields creates a logger with additional fields
func WithFields(fields map[string]interface{}) *Logger {
	return GetGlobalLogger().WithFields(fields)
}

// WithError creates a logger with an error field
func WithError(err error) *Logger {
	return GetGlobalLogger().WithError(err)
}

// SetGlobalLevel sets the global logger level
func SetGlobalLevel(level LogLevel) {
	GetGlobalLogger().SetLevel(level)
}

// LegacyAdapter provides compatibility with standard log package
type LegacyAdapter struct {
	logger *Logger
}

// NewLegacyAdapter creates an adapter for the standard log package
func NewLegacyAdapter(logger *Logger) *LegacyAdapter {
	return &LegacyAdapter{logger: logger}
}

// Write implements io.Writer for compatibility with standard log
func (a *LegacyAdapter) Write(p []byte) (n int, err error) {
	message := strings.TrimSuffix(string(p), "\n")
	a.logger.Info(message)
	return len(p), nil
}

// SetStandardLogger replaces the standard library logger with our structured logger
func SetStandardLogger(logger *Logger) {
	adapter := NewLegacyAdapter(logger)
	log.SetOutput(adapter)
	log.SetFlags(0) // Disable default formatting
}

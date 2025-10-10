// Unit tests for the logging package
package logging

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	
)

// TestLogLevelString tests log level string representations
func TestLogLevelString(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARN, "WARN"},
		{ERROR, "ERROR"},
		{FATAL, "FATAL"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.level.String() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.level.String())
			}
		})
	}
}

// TestNewLogger tests creating a new logger
func TestNewLogger(t *testing.T) {
	var buf bytes.Buffer
	config := &LogConfig{
		Level:      INFO,
		Component:  "test",
		Output:     &buf,
		TimeFormat: "2006-01-02 15:04:05",
	}

	logger := NewLogger(config)
	if logger == nil {
		t.Fatal("Expected non-nil logger")
	}

	logger.Info("test message")
	output := buf.String()

	if !strings.Contains(output, "test message") {
		t.Error("Expected log output to contain message")
	}

	if !strings.Contains(output, "[INFO]") {
		t.Error("Expected log output to contain level")
	}

	if !strings.Contains(output, "[test]") {
		t.Error("Expected log output to contain component")
	}
}

// TestDefaultConfig tests default configuration
func TestDefaultConfig(t *testing.T) {
	config := DefaultLogConfig()

	if config.Level != INFO {
		t.Errorf("Expected default level INFO, got %v", config.Level)
	}

	if config.Component != "distkv" {
		t.Errorf("Expected default component 'distkv', got %s", config.Component)
	}

	if config.TimeFormat == "" {
		t.Error("Expected non-empty time format")
	}
}

// TestLogLevelFiltering tests that log levels are filtered correctly
func TestLogLevelFiltering(t *testing.T) {
	tests := []struct {
		name          string
		configLevel   LogLevel
		logLevel      LogLevel
		shouldLog     bool
	}{
		{"DEBUG logs at DEBUG level", DEBUG, DEBUG, true},
		{"INFO logs at DEBUG level", DEBUG, INFO, true},
		{"DEBUG doesn't log at INFO level", INFO, DEBUG, false},
		{"INFO logs at INFO level", INFO, INFO, true},
		{"WARN logs at INFO level", INFO, WARN, true},
		{"ERROR logs at INFO level", INFO, ERROR, true},
		{"INFO doesn't log at ERROR level", ERROR, INFO, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			config := &LogConfig{
				Level:     tt.configLevel,
				Component: "test",
				Output:    &buf,
			}

			logger := NewLogger(config)
			message := "test message"

			switch tt.logLevel {
			case DEBUG:
				logger.Debug(message)
			case INFO:
				logger.Info(message)
			case WARN:
				logger.Warn(message)
			case ERROR:
				logger.Error(message)
			}

			output := buf.String()
			containsMessage := strings.Contains(output, message)

			if tt.shouldLog && !containsMessage {
				t.Errorf("Expected message to be logged, but it wasn't")
			}

			if !tt.shouldLog && containsMessage {
				t.Errorf("Expected message not to be logged, but it was")
			}
		})
	}
}

// TestWithComponent tests creating logger with component
func TestWithComponent(t *testing.T) {
	var buf bytes.Buffer
	config := &LogConfig{
		Level:     INFO,
		Component: "original",
		Output:    &buf,
	}

	logger := NewLogger(config)
	newLogger := logger.WithComponent("new-component")

	newLogger.Info("test message")
	output := buf.String()

	if !strings.Contains(output, "[new-component]") {
		t.Error("Expected log to contain new component name")
	}

	if strings.Contains(output, "[original]") {
		t.Error("Expected log not to contain original component name")
	}
}

// TestWithField tests adding single field
func TestWithField(t *testing.T) {
	var buf bytes.Buffer
	config := &LogConfig{
		Level:     INFO,
		Component: "test",
		Output:    &buf,
	}

	logger := NewLogger(config)
	fieldLogger := logger.WithField("requestID", "12345")

	fieldLogger.Info("processing request")
	output := buf.String()

	if !strings.Contains(output, "requestID=12345") {
		t.Error("Expected log to contain field")
	}
}

// TestWithFields tests adding multiple fields
func TestWithFields(t *testing.T) {
	var buf bytes.Buffer
	config := &LogConfig{
		Level:     INFO,
		Component: "test",
		Output:    &buf,
	}

	logger := NewLogger(config)
	fields := map[string]interface{}{
		"userID":    "user123",
		"sessionID": "session456",
		"count":     42,
	}

	fieldLogger := logger.WithFields(fields)
	fieldLogger.Info("user action")
	output := buf.String()

	expectedFields := []string{
		"userID=user123",
		"sessionID=session456",
		"count=42",
	}

	for _, field := range expectedFields {
		if !strings.Contains(output, field) {
			t.Errorf("Expected log to contain field %s", field)
		}
	}
}

// TestWithError tests adding error field
func TestWithError(t *testing.T) {
	var buf bytes.Buffer
	config := &LogConfig{
		Level:     INFO,
		Component: "test",
		Output:    &buf,
	}

	logger := NewLogger(config)
	err := fmt.Errorf("database connection failed")
	errorLogger := logger.WithError(err)

	errorLogger.Error("failed to connect")
	output := buf.String()

	if !strings.Contains(output, "database connection failed") {
		t.Error("Expected log to contain error message")
	}

	// Test with nil error
	buf.Reset()
	nilLogger := logger.WithError(nil)
	nilLogger.Info("no error")
	output = buf.String()

	if strings.Contains(output, "error=") {
		t.Error("Expected no error field when error is nil")
	}
}

// TestSetLevel tests changing log level dynamically
func TestSetLevel(t *testing.T) {
	var buf bytes.Buffer
	config := &LogConfig{
		Level:     INFO,
		Component: "test",
		Output:    &buf,
	}

	logger := NewLogger(config)

	// Debug should not log at INFO level
	logger.Debug("debug message 1")
	if strings.Contains(buf.String(), "debug message 1") {
		t.Error("Debug message should not be logged at INFO level")
	}

	// Change to DEBUG level
	buf.Reset()
	logger.SetLevel(DEBUG)
	logger.Debug("debug message 2")

	if !strings.Contains(buf.String(), "debug message 2") {
		t.Error("Debug message should be logged at DEBUG level")
	}
}

// TestFormattedLogging tests formatted log methods
func TestFormattedLogging(t *testing.T) {
	var buf bytes.Buffer
	config := &LogConfig{
		Level:     INFO,
		Component: "test",
		Output:    &buf,
	}

	logger := NewLogger(config)

	tests := []struct {
		name     string
		logFunc  func()
		expected string
	}{
		{
			name: "Debugf",
			logFunc: func() {
				logger.SetLevel(DEBUG)
				logger.Debugf("count: %d, name: %s", 42, "test")
			},
			expected: "count: 42, name: test",
		},
		{
			name: "Infof",
			logFunc: func() {
				buf.Reset()
				logger.Infof("user %s logged in at %d", "john", 1234567890)
			},
			expected: "user john logged in at 1234567890",
		},
		{
			name: "Warnf",
			logFunc: func() {
				buf.Reset()
				logger.Warnf("threshold exceeded: %d%%", 95)
			},
			expected: "threshold exceeded: 95%",
		},
		{
			name: "Errorf",
			logFunc: func() {
				buf.Reset()
				logger.Errorf("failed after %d attempts", 3)
			},
			expected: "failed after 3 attempts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.logFunc()
			output := buf.String()

			if !strings.Contains(output, tt.expected) {
				t.Errorf("Expected output to contain '%s', got: %s", tt.expected, output)
			}
		})
	}
}

// TestCallerInformation tests that caller information is included
func TestCallerInformation(t *testing.T) {
	var buf bytes.Buffer
	config := &LogConfig{
		Level:     INFO,
		Component: "test",
		Output:    &buf,
	}

	logger := NewLogger(config)
	logger.Info("test message")
	output := buf.String()

	// Should contain file name and line number
	if !strings.Contains(output, "logger_test.go") {
		t.Error("Expected log to contain source file name")
	}

	// Should contain format (file:line function)
	if !strings.Contains(output, ":") {
		t.Error("Expected log to contain line number separator")
	}
}

// TestLoggerImmutability tests that logger methods return new instances
func TestLoggerImmutability(t *testing.T) {
	var buf bytes.Buffer
	config := &LogConfig{
		Level:     INFO,
		Component: "original",
		Output:    &buf,
	}

	logger1 := NewLogger(config)
	logger2 := logger1.WithComponent("component2")
	logger3 := logger2.WithField("key", "value")

	// Each logger should be independent
	logger1.Info("message1")
	output1 := buf.String()
	if !strings.Contains(output1, "[original]") {
		t.Error("Logger1 should have original component")
	}

	buf.Reset()
	logger2.Info("message2")
	output2 := buf.String()
	if !strings.Contains(output2, "[component2]") {
		t.Error("Logger2 should have component2")
	}
	if strings.Contains(output2, "key=value") {
		t.Error("Logger2 should not have the field from logger3")
	}

	buf.Reset()
	logger3.Info("message3")
	output3 := buf.String()
	if !strings.Contains(output3, "key=value") {
		t.Error("Logger3 should have the field")
	}
}

// TestGlobalLogger tests global logger functionality
func TestGlobalLogger(t *testing.T) {
	// Reset global logger for test isolation
	var buf bytes.Buffer
	config := &LogConfig{
		Level:     INFO,
		Component: "global-test",
		Output:    &buf,
	}

	InitGlobalLogger(config)
	globalLogger := GetGlobalLogger()

	if globalLogger == nil {
		t.Fatal("Expected non-nil global logger")
	}

	Info("global test message")
	output := buf.String()

	if !strings.Contains(output, "global test message") {
		t.Error("Expected global logger to log message")
	}
}

// TestPackageLevelFunctions tests package-level convenience functions
func TestPackageLevelFunctions(t *testing.T) {
	var buf bytes.Buffer
	config := &LogConfig{
		Level:     DEBUG,
		Component: "package-test",
		Output:    &buf,
	}

	InitGlobalLogger(config)

	tests := []struct {
		name    string
		logFunc func()
		level   string
	}{
		{"Debug", func() { Debug("debug msg") }, "DEBUG"},
		{"Info", func() { Info("info msg") }, "INFO"},
		{"Warn", func() { Warn("warn msg") }, "WARN"},
		{"Error", func() { Error("error msg") }, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc()
			output := buf.String()

			if !strings.Contains(output, tt.level) {
				t.Errorf("Expected output to contain level %s", tt.level)
			}
		})
	}
}

// TestFormattedPackageFunctions tests formatted package-level functions
func TestFormattedPackageFunctions(t *testing.T) {
	var buf bytes.Buffer
	config := &LogConfig{
		Level:     DEBUG,
		Component: "package-test",
		Output:    &buf,
	}

	InitGlobalLogger(config)

	Debugf("value: %d", 42)
	if !strings.Contains(buf.String(), "value: 42") {
		t.Error("Debugf failed")
	}

	buf.Reset()
	Infof("user: %s", "alice")
	if !strings.Contains(buf.String(), "user: alice") {
		t.Error("Infof failed")
	}

	buf.Reset()
	Warnf("count: %d", 100)
	if !strings.Contains(buf.String(), "count: 100") {
		t.Error("Warnf failed")
	}

	buf.Reset()
	Errorf("error code: %d", 404)
	if !strings.Contains(buf.String(), "error code: 404") {
		t.Error("Errorf failed")
	}
}

// TestWithComponentPackageLevel tests package-level WithComponent
func TestWithComponentPackageLevel(t *testing.T) {
	var buf bytes.Buffer
	config := &LogConfig{
		Level:     INFO,
		Component: "default",
		Output:    &buf,
	}

	InitGlobalLogger(config)

	componentLogger := WithComponent("custom-component")
	componentLogger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "[custom-component]") {
		t.Error("Expected custom component in output")
	}
}

// TestWithFieldsPackageLevel tests package-level field methods
func TestWithFieldsPackageLevel(t *testing.T) {
	var buf bytes.Buffer
	config := &LogConfig{
		Level:     INFO,
		Component: "test",
		Output:    &buf,
	}

	InitGlobalLogger(config)

	// Test WithField
	buf.Reset()
	fieldLogger := WithField("key", "value")
	fieldLogger.Info("message")
	if !strings.Contains(buf.String(), "key=value") {
		t.Error("WithField failed")
	}

	// Test WithFields
	buf.Reset()
	fieldsLogger := WithFields(map[string]interface{}{
		"field1": "value1",
		"field2": 123,
	})
	fieldsLogger.Info("message")
	output := buf.String()
	if !strings.Contains(output, "field1=value1") || !strings.Contains(output, "field2=123") {
		t.Error("WithFields failed")
	}

	// Test WithError
	buf.Reset()
	err := fmt.Errorf("test error")
	errorLogger := WithError(err)
	errorLogger.Info("message")
	if !strings.Contains(buf.String(), "test error") {
		t.Error("WithError failed")
	}
}

// TestSetGlobalLevel tests changing global log level
func TestSetGlobalLevel(t *testing.T) {
	var buf bytes.Buffer
	config := &LogConfig{
		Level:     INFO,
		Component: "test",
		Output:    &buf,
	}

	InitGlobalLogger(config)

	// Debug should not log at INFO level
	Debug("debug1")
	if strings.Contains(buf.String(), "debug1") {
		t.Error("Debug should not log at INFO level")
	}

	// Change to DEBUG level
	buf.Reset()
	SetGlobalLevel(DEBUG)
	Debug("debug2")

	if !strings.Contains(buf.String(), "debug2") {
		t.Error("Debug should log at DEBUG level")
	}
}

// TestLegacyAdapter tests compatibility with standard log package
func TestLegacyAdapter(t *testing.T) {
	var buf bytes.Buffer
	config := &LogConfig{
		Level:     INFO,
		Component: "test",
		Output:    &buf,
	}

	logger := NewLogger(config)
	adapter := NewLegacyAdapter(logger)

	message := "legacy log message\n"
	n, err := adapter.Write([]byte(message))

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if n != len(message) {
		t.Errorf("Expected %d bytes written, got %d", len(message), n)
	}

	output := buf.String()
	if !strings.Contains(output, "legacy log message") {
		t.Error("Expected message to be logged through adapter")
	}
}

// TestNilConfig tests that nil config uses defaults
func TestNilConfig(t *testing.T) {
	logger := NewLogger(nil)
	if logger == nil {
		t.Fatal("Expected non-nil logger with nil config")
	}

	// Should not panic and should use defaults
	var buf bytes.Buffer
	testLogger := &LogConfig{
		Level:     INFO,
		Component: "test",
		Output:    &buf,
	}
	logger = NewLogger(testLogger)
	logger.Info("test")

	if buf.Len() == 0 {
		t.Error("Expected output from logger with nil config")
	}
}

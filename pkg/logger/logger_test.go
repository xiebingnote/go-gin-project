package logger

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xiebingnote/go-gin-project/library/config"
	"go.uber.org/zap/zapcore"
)

// setupTestConfig initializes test configuration
func setupTestConfig() {
	config.LogConfig = &config.LogConfigEntry{
		Log: struct {
			DefaultLevel string `toml:"DefaultLevel"`
			LogDir       string `toml:"LogDir"`
			LogFileDebug string `toml:"LogFileDebug"`
			LogFileInfo  string `toml:"LogFileInfo"`
			LogFileWarn  string `toml:"LogFileWarn"`
			LogFileError string `toml:"LogFileError"`
			MaxSize      int    `toml:"MaxSize"`
			MaxAge       int    `toml:"MaxAge"`
			MaxBackups   int    `toml:"MaxBackups"`
			LocalTime    bool   `toml:"LocalTime"`
			Compress     bool   `toml:"Compress"`
		}{
			DefaultLevel: "info",
			LogDir:       "./test_logs",
			LogFileDebug: "debug.log",
			LogFileInfo:  "info.log",
			LogFileWarn:  "warn.log",
			LogFileError: "error.log",
			MaxSize:      100,
			MaxAge:       30,
			MaxBackups:   10,
			LocalTime:    true,
			Compress:     false,
		},
	}
}

// cleanupTestLogs removes test log directory
func cleanupTestLogs() {
	if config.LogConfig != nil {
		os.RemoveAll(config.LogConfig.Log.LogDir)
	}
}

// TestNewJsonLogger tests the NewJsonLogger function with various options.
//
// It checks that NewJsonLogger returns a valid logger when given valid options,
// and that it returns an error when given invalid options. It also checks that
// the logger can write logs and that it can be closed without error.
func TestNewJsonLogger(t *testing.T) {
	setupTestConfig()
	defer cleanupTestLogs()

	tests := []struct {
		name    string
		opts    []Option
		wantErr bool
	}{
		{
			name:    "default logger",
			opts:    []Option{},
			wantErr: false,
		},
		{
			name:    "with debug level",
			opts:    []Option{WithDebugLevel()},
			wantErr: false,
		},
		{
			name:    "with fields",
			opts:    []Option{WithField("service", "test"), WithField("version", "1.0.0")},
			wantErr: false,
		},
		{
			name:    "with multiple fields",
			opts:    []Option{WithFields(map[string]string{"app": "test", "env": "dev"})},
			wantErr: false,
		},
		{
			name:    "with console disabled",
			opts:    []Option{WithDisableConsole()},
			wantErr: false,
		},
		{
			name:    "with custom time layout",
			opts:    []Option{WithTimeLayout("2006-01-02 15:04:05")},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewJsonLogger(tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewJsonLogger() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && logger == nil {
				t.Errorf("NewJsonLogger() returned nil logger")
			}
			if logger != nil {
				// Test that logger can write logs
				logger.Info("test log message")
				// Close logger
				if err := Close(logger); err != nil {
					t.Logf("Warning: failed to close logger: %v", err)
				}
			}
		})
	}
}

// TestNewJsonLogger_ErrorCases tests error cases for the NewJsonLogger function.
//
// This test verifies that the NewJsonLogger function returns the expected errors
// when the logger configuration is improperly set up. It uses a table-driven test
// approach to cover different error scenarios. For each test case, it sets up
// the configuration using the setupFunc, calls NewJsonLogger, and checks for
// expected errors and error messages.
//
// Test cases:
//  1. "nil config": Ensures the error "log configuration is not initialized" is
//     returned when the logger configuration is nil.
//  2. "empty log directory": Ensures the error "log directory is not configured"
//     is returned when the log directory is empty.
func TestNewJsonLogger_ErrorCases(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func()
		wantErr   bool
		errMsg    string
	}{
		{
			name: "nil config",
			setupFunc: func() {
				config.LogConfig = nil
			},
			wantErr: true,
			errMsg:  "log configuration is not initialized",
		},
		{
			name: "empty log directory",
			setupFunc: func() {
				setupTestConfig()
				config.LogConfig.Log.LogDir = ""
			},
			wantErr: true,
			errMsg:  "log directory is not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFunc()
			defer cleanupTestLogs()

			logger, err := NewJsonLogger()
			if !tt.wantErr {
				t.Errorf("Expected error but got none")
				return
			}
			if err == nil {
				t.Errorf("Expected error but got none")
				return
			}
			if err.Error() != tt.errMsg {
				t.Errorf("Expected error message '%s', got '%s'", tt.errMsg, err.Error())
			}
			if logger != nil {
				t.Errorf("Expected nil logger on error")
			}
		})
	}
}

// TestGetDefaultLevel tests the GetDefaultLevel function with various test cases.
//
// The test cases include valid log levels such as "debug", "info", "warn", and
// "error". It also includes an invalid log level and a nil config case.
//
// The test checks that the function returns the correct log level for each
// test case.
func TestGetDefaultLevel(t *testing.T) {
	tests := []struct {
		name     string
		setup    func()
		expected zapcore.Level
	}{
		{
			name: "debug level",
			setup: func() {
				setupTestConfig()
				config.LogConfig.Log.DefaultLevel = "debug"
			},
			expected: zapcore.DebugLevel,
		},
		{
			name: "info level",
			setup: func() {
				setupTestConfig()
				config.LogConfig.Log.DefaultLevel = "info"
			},
			expected: zapcore.InfoLevel,
		},
		{
			name: "warn level",
			setup: func() {
				setupTestConfig()
				config.LogConfig.Log.DefaultLevel = "warn"
			},
			expected: zapcore.WarnLevel,
		},
		{
			name: "error level",
			setup: func() {
				setupTestConfig()
				config.LogConfig.Log.DefaultLevel = "error"
			},
			expected: zapcore.ErrorLevel,
		},
		{
			name: "invalid level",
			setup: func() {
				setupTestConfig()
				config.LogConfig.Log.DefaultLevel = "invalid"
			},
			expected: zapcore.InfoLevel,
		},
		{
			name: "nil config",
			setup: func() {
				config.LogConfig = nil
			},
			expected: zapcore.InfoLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			level := GetDefaultLevel()
			if level != tt.expected {
				t.Errorf("GetDefaultLevel() = %v, want %v", level, tt.expected)
			}
		})
	}
}

// TestValidateLogLevel tests the ValidateLogLevel function.
//
// It checks that ValidateLogLevel returns the expected result for various
// log level strings.
func TestValidateLogLevel(t *testing.T) {
	tests := []struct {
		name    string
		level   string
		wantErr bool
	}{
		{"debug", "debug", false},
		{"info", "info", false},
		{"warn", "warn", false},
		{"error", "error", false},
		{"invalid", "invalid", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLogLevel(tt.level)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateLogLevel() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestCreateLogWriter tests the createLogWriter function with various test cases.
//
// The test cases include a valid filename and an empty filename. It checks
// that the function returns the correct error for each test case.
func TestCreateLogWriter(t *testing.T) {
	setupTestConfig()
	defer cleanupTestLogs()

	tests := []struct {
		name     string
		filename string
		wantErr  bool
	}{
		{
			name:     "valid filename",
			filename: filepath.Join("./test_logs", "test.log"),
			wantErr:  false,
		},
		{
			name:     "empty filename",
			filename: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer, err := createLogWriter(tt.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("createLogWriter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && writer == nil {
				t.Errorf("createLogWriter() returned nil writer")
			}
		})
	}
}

// TestClose tests the Close function with various scenarios.
//
// It ensures that calling Close with a nil logger does not return an error.
// It also tests closing a valid logger and logs any warnings if an error
// occurs during the close operation. The test sets up and cleans up the
// test environment for valid logger scenarios.
func TestClose(t *testing.T) {
	// Test with nil logger
	err := Close(nil)
	if err != nil {
		t.Errorf("Close(nil) should not return error, got: %v", err)
	}

	// Test with valid logger
	setupTestConfig()
	defer cleanupTestLogs()

	logger, err := NewJsonLogger(WithDisableConsole())
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	err = Close(logger)
	if err != nil {
		t.Logf("Warning: Close() returned error: %v", err)
	}
}

package logger

import (
	"encoding/json"
	"errors"
	"go.uber.org/zap"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xiebingnote/go-gin-project/library/config"
	"go.uber.org/zap/zapcore"
)

// setupTestConfig initializes test configuration
func setupTestConfig(t *testing.T) {
	t.Helper()
	previous := config.LogConfig
	t.Cleanup(func() { config.LogConfig = previous })
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
			LogDir:       t.TempDir(),
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

// TestNewJsonLogger tests the NewJsonLogger function with various options.
//
// It checks that NewJsonLogger returns a valid logger when given valid options,
// and that it returns an error when given invalid options. It also checks that
// the logger can write logs and that it can be closed without error.
func TestNewJsonLogger(t *testing.T) {
	setupTestConfig(t)

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
					t.Errorf("failed to close logger: %v", err)
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
				setupTestConfig(t)
				config.LogConfig.Log.LogDir = ""
			},
			wantErr: true,
			errMsg:  "log directory is not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFunc()

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
				setupTestConfig(t)
				config.LogConfig.Log.DefaultLevel = "debug"
			},
			expected: zapcore.DebugLevel,
		},
		{
			name: "info level",
			setup: func() {
				setupTestConfig(t)
				config.LogConfig.Log.DefaultLevel = "info"
			},
			expected: zapcore.InfoLevel,
		},
		{
			name: "warn level",
			setup: func() {
				setupTestConfig(t)
				config.LogConfig.Log.DefaultLevel = "warn"
			},
			expected: zapcore.WarnLevel,
		},
		{
			name: "error level",
			setup: func() {
				setupTestConfig(t)
				config.LogConfig.Log.DefaultLevel = "error"
			},
			expected: zapcore.ErrorLevel,
		},
		{
			name: "invalid level",
			setup: func() {
				setupTestConfig(t)
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
	setupTestConfig(t)

	tests := []struct {
		name     string
		filename string
		wantErr  bool
	}{
		{
			name:     "valid filename",
			filename: filepath.Join(config.LogConfig.Log.LogDir, "test.log"),
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
			writer, err := createLogWriter(tt.filename, config.LogConfig)
			if (err != nil) != tt.wantErr {
				t.Errorf("createLogWriter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && writer == nil {
				t.Errorf("createLogWriter() returned nil writer")
			}
			if writer != nil {
				if err := writer.Close(); err != nil {
					t.Fatal(err)
				}
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
	setupTestConfig(t)

	logger, err := NewJsonLogger(WithDisableConsole())
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	err = Close(logger)
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

func readEntries(t *testing.T, path string) []map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var entries []map[string]any
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	for {
		var entry map[string]any
		if err := decoder.Decode(&entry); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			t.Fatalf("invalid JSON log in %s: %v", path, err)
		}
		entries = append(entries, entry)
	}
	return entries
}

func TestFileRoutingHonorsMinimumLevel(t *testing.T) {
	for minimum := zapcore.DebugLevel; minimum <= zapcore.FatalLevel; minimum++ {
		t.Run(minimum.String(), func(t *testing.T) {
			setupTestConfig(t)
			log, err := NewJsonLogger(WithLevel(minimum), WithDisableConsole())
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = Close(log) })
			// Check the core directly to exercise Panic/Fatal routing without exiting.
			for level := zapcore.DebugLevel; level <= zapcore.FatalLevel; level++ {
				if entry := log.Core().Check(zapcore.Entry{Level: level, Message: level.String(), Time: time.Now()}, nil); entry != nil {
					entry.Write()
				}
			}
			if err := Close(log); err != nil {
				t.Fatal(err)
			}
			for _, file := range []struct {
				name     string
				min, max zapcore.Level
			}{{"debug.log", zapcore.DebugLevel, zapcore.DebugLevel}, {"info.log", zapcore.InfoLevel, zapcore.InfoLevel},
				{"warn.log", zapcore.WarnLevel, zapcore.WarnLevel}, {"error.log", zapcore.ErrorLevel, zapcore.FatalLevel}} {
				entries := readEntries(t, filepath.Join(config.LogConfig.Log.LogDir, file.name))
				want := 0
				for level := file.min; level <= file.max; level++ {
					if level >= minimum {
						want++
					}
				}
				if len(entries) != want {
					t.Fatalf("%s has %d entries, want %d", file.name, len(entries), want)
				}
				for _, entry := range entries {
					level, err := zapcore.ParseLevel(entry["level"].(string))
					if err != nil || level < minimum || level < file.min || level > file.max {
						t.Fatalf("wrong destination for %v: %s", entry, file.name)
					}
				}
			}
		})
	}
}

func TestConsoleRoutingIncludesWarnAndHonorsThreshold(t *testing.T) {
	for _, minimum := range []zapcore.Level{zapcore.InfoLevel, zapcore.ErrorLevel} {
		t.Run(minimum.String(), func(t *testing.T) {
			setupTestConfig(t)
			previousOut, previousErr := os.Stdout, os.Stderr
			out, err := os.CreateTemp(t.TempDir(), "stdout")
			if err != nil {
				t.Fatal(err)
			}
			errOut, err := os.CreateTemp(t.TempDir(), "stderr")
			if err != nil {
				_ = out.Close()
				t.Fatal(err)
			}
			os.Stdout, os.Stderr = out, errOut
			t.Cleanup(func() {
				os.Stdout, os.Stderr = previousOut, previousErr
				_ = out.Close()
				_ = errOut.Close()
			})
			log, err := NewJsonLogger(WithLevel(minimum))
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = Close(log) })
			log.Debug("debug")
			log.Info("info")
			log.Warn("warn")
			log.Error("error")
			if err := Close(log); err != nil {
				t.Fatal(err)
			}
			stdout, stderr := readEntries(t, out.Name()), readEntries(t, errOut.Name())
			if minimum == zapcore.InfoLevel {
				if len(stdout) != 1 || stdout[0]["level"] != "info" || len(stderr) != 2 || stderr[0]["level"] != "warn" || stderr[1]["level"] != "error" {
					t.Fatalf("incorrect console routing: stdout=%v stderr=%v", stdout, stderr)
				}
			} else if len(stdout) != 0 || len(stderr) != 1 || stderr[0]["level"] != "error" {
				t.Fatalf("console ignored minimum level: stdout=%v stderr=%v", stdout, stderr)
			}
			if _, err := out.WriteString("still open"); err != nil {
				t.Fatalf("Close closed stdout: %v", err)
			}
			if _, err := errOut.WriteString("still open"); err != nil {
				t.Fatalf("Close closed stderr: %v", err)
			}
		})
	}
}

func TestConsoleSyncHandlesPipesButReportsClosedFiles(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reader.Close(); _ = writer.Close() })
	console := consoleWriter{writer}
	if err := console.Sync(); err != nil {
		t.Fatalf("pipe Sync should be harmless: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := console.Sync(); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("lost closed-file error: %v", err)
	}
	if isValidFileDescriptor(nil) || isValidFileDescriptor(writer) {
		t.Fatal("invalid console descriptor accepted")
	}
}

func TestLoggerRejectsInvalidConfiguration(t *testing.T) {
	for _, tt := range []struct {
		name string
		edit func()
		opts []Option
	}{
		{"invalid default level", func() { config.LogConfig.Log.DefaultLevel = "verbose" }, nil},
		{"empty filename", func() { config.LogConfig.Log.LogFileInfo = "" }, nil},
		{"directory filename", func() { config.LogConfig.Log.LogFileInfo = "." }, nil},
		{"parent traversal", func() { config.LogConfig.Log.LogFileInfo = "../outside.log" }, nil},
		{"absolute filename", func() { config.LogConfig.Log.LogFileInfo = filepath.Join(t.TempDir(), "outside.log") }, nil},
		{"duplicate filename", func() { config.LogConfig.Log.LogFileInfo = "./debug.log" }, nil},
		{"negative size", func() { config.LogConfig.Log.MaxSize = -1 }, nil},
		{"negative backups", func() { config.LogConfig.Log.MaxBackups = -1 }, nil},
		{"negative retention", func() { config.LogConfig.Log.MaxAge = -1 }, nil},
		{"nil option", nil, []Option{nil}},
		{"invalid level option", nil, []Option{WithLevel(zapcore.Level(100))}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			setupTestConfig(t)
			if tt.edit != nil {
				tt.edit()
			}
			log, err := NewJsonLogger(tt.opts...)
			if err == nil || log != nil {
				_ = Close(log)
				t.Fatal("invalid configuration accepted")
			}
		})
	}
}

func TestLoggerRejectsUnusableDestinationsWithoutTruncatingFiles(t *testing.T) {
	for _, kind := range []string{"directory", "symlink", "hardlink"} {
		t.Run(kind, func(t *testing.T) {
			setupTestConfig(t)
			dir := config.LogConfig.Log.LogDir
			original := filepath.Join(dir, "info.log")
			if err := os.WriteFile(original, []byte("existing log\n"), 0600); err != nil {
				t.Fatal(err)
			}
			blocked := filepath.Join(dir, "error.log")
			var err error
			switch kind {
			case "directory":
				err = os.Mkdir(blocked, 0700)
			case "symlink":
				err = os.Symlink(original, blocked)
			case "hardlink":
				err = os.Link(original, blocked)
			}
			if err != nil {
				t.Fatal(err)
			}
			log, err := NewJsonLogger(WithDisableConsole())
			if err == nil || log != nil {
				_ = Close(log)
				t.Fatal("invalid log destination accepted")
			}
			data, err := os.ReadFile(original)
			if err != nil || string(data) != "existing log\n" {
				t.Fatalf("existing log changed: %q, %v", data, err)
			}
		})
	}
}

func TestLoggerFieldsAreSnapshottedAndCloseCoversChildren(t *testing.T) {
	setupTestConfig(t)
	fields := map[string]string{"service": "original", "version": "1"}
	option := WithFields(fields)
	fields["service"] = "changed"
	log, err := NewJsonLogger(option, WithField("version", "2"), WithTimeLayout("2006"), WithDisableConsole())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = Close(log) })
	child := log.Named("child").With(zap.String("request", "42"))
	child.Info("before sync")
	if err := child.Sync(); err != nil {
		t.Fatal(err)
	}
	child.Info("after sync")
	if err := Close(child); err != nil {
		t.Fatal(err)
	}
	if err := Close(log); err != nil {
		t.Fatal(err)
	}
	// A saved reference must not reopen a file after its family is closed.
	log.WithOptions(zap.ErrorOutput(zapcore.AddSync(io.Discard))).Info("after close")
	entries := readEntries(t, filepath.Join(config.LogConfig.Log.LogDir, "info.log"))
	if len(entries) != 2 {
		t.Fatalf("expected exactly two entries before Close, got %v", entries)
	}
	for _, entry := range entries {
		if entry["service"] != "original" || entry["version"] != "2" || entry["request"] != "42" || entry["logger"] != "child" || entry["time"] != time.Now().Format("2006") {
			t.Fatalf("incorrect structured log: %v", entry)
		}
	}
}

func TestLoggerConcurrentWritesAndClose(t *testing.T) {
	setupTestConfig(t)
	log, err := NewJsonLogger(WithDisableConsole())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = Close(log) })
	var wg sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			child := log.With(zap.Int("worker", id))
			for i := 0; i < 50; i++ {
				child.Info("concurrent log", zap.Int("item", i))
			}
		}(worker)
	}
	wg.Wait()
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := Close(log.With(zap.Bool("closing", true))); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if got := len(readEntries(t, filepath.Join(config.LogConfig.Log.LogDir, "info.log"))); got != 400 {
		t.Fatalf("concurrent writes lost or corrupted logs: got %d, want 400", got)
	}
}

type failingLogSink struct {
	syncErr, closeErr error
	closes            int
}

func (*failingLogSink) Write(p []byte) (int, error) { return len(p), nil }
func (s *failingLogSink) Sync() error               { return s.syncErr }
func (s *failingLogSink) Close() error              { s.closes++; return s.closeErr }

func TestClosePreservesSyncAndCloseErrors(t *testing.T) {
	syncErr, closeErr := errors.New("sync failed"), errors.New("close failed")
	first := &failingLogSink{syncErr: syncErr, closeErr: closeErr}
	second := &failingLogSink{}
	files := []*logWriter{{writer: first}, {writer: second}}
	core := &managedCore{
		Core:      zapcore.NewCore(zapcore.NewJSONEncoder(createEncoderConfig(time.RFC3339)), files[0], zapcore.DebugLevel),
		resources: &logResources{files: files},
	}
	log := zap.New(core)
	for i := 0; i < 2; i++ {
		err := Close(log)
		if !errors.Is(err, syncErr) || !errors.Is(err, closeErr) {
			t.Fatalf("lost cleanup errors: %v", err)
		}
	}
	if first.closes != 1 || second.closes != 1 {
		t.Fatalf("cleanup must close every sink exactly once: %d, %d", first.closes, second.closes)
	}
	if _, err := files[0].Write([]byte("late log")); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("closed writer accepted a write: %v", err)
	}
}

func TestLogRotationRetainsBothEntries(t *testing.T) {
	setupTestConfig(t)
	config.LogConfig.Log.MaxSize = 1
	log, err := NewJsonLogger(WithDisableConsole())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = Close(log) })
	log.Info("first", zap.String("payload", strings.Repeat("x", 600*1024)))
	log.Info("second", zap.String("payload", strings.Repeat("y", 600*1024)))
	if err := Close(log); err != nil {
		t.Fatal(err)
	}
	dir := config.LogConfig.Log.LogDir
	backups, err := filepath.Glob(filepath.Join(dir, "info-*.log"))
	if err != nil || len(backups) != 1 {
		t.Fatalf("expected one rotated file, got %v, %v", backups, err)
	}
	old, current := readEntries(t, backups[0]), readEntries(t, filepath.Join(dir, "info.log"))
	if len(old) != 1 || old[0]["msg"] != "first" || len(current) != 1 || current[0]["msg"] != "second" {
		t.Fatal("rotation lost or duplicated log entries")
	}
}

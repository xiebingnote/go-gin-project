package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/xiebingnote/go-gin-project/library/common"
	"github.com/xiebingnote/go-gin-project/library/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Option defines a function type for configuring logger options
type Option func(o *options)

// options contains the configuration settings for the logger
type options struct {
	level          zapcore.Level
	fields         map[string]string
	timeLayout     string
	disableConsole bool
	logDir         string
}

// WithLevel returns an Option that sets the logging level
//
// The logging level is an inclusive threshold. Messages with a level below
// the specified level will be discarded. Valid levels are Debug, Info, Warn,
// Error, and DPanic.
//
// This option is mutually exclusive with WithDebugLevel, WithInfoLevel,
// WithWarnLevel, and WithErrorLevel.
func WithLevel(level zapcore.Level) Option {
	return func(opt *options) {
		opt.level = level
	}
}

// WithDebugLevel sets the logging level to Debug
//
// This option sets the logging level to Debug, which will log all messages with
// level Debug and above. Only logs with level Debug and above will be printed.
func WithDebugLevel() Option {
	return WithLevel(zapcore.DebugLevel)
}

// WithInfoLevel sets the logging level to Info
//
// This option sets the logging level to Info, which will log all messages with
// level Info and above. Only logs with level Info and above will be printed.
func WithInfoLevel() Option {
	return WithLevel(zapcore.InfoLevel)
}

// WithWarnLevel sets the logging level to Warn
//
// This option sets the logging level to Warn, which will log all messages with
// level Warn and above. Only logs with level Warn and above will be printed.
func WithWarnLevel() Option {
	return WithLevel(zapcore.WarnLevel)
}

// WithErrorLevel sets the logging level to Error
//
// This option sets the logging level to Error, which is the highest level of
// logging. Only logs with level Error and above will be printed.
func WithErrorLevel() Option {
	return WithLevel(zapcore.ErrorLevel)
}

// WithField returns an Option that adds a single key-value pair as a field to the logger.
//
// Parameters:
//   - key: The field key to be added to the logger.
//   - val: The field value to be associated with the key.
//
// The function ensures that the fields map is initialized if it is nil
// and then adds the provided key-value pair to the map.
func WithField(key, val string) Option {
	return func(opt *options) {
		if opt.fields == nil {
			opt.fields = make(map[string]string)
		}
		opt.fields[key] = val
	}
}

// WithFields returns an Option that adds a map of key-value pairs as fields to the logger.
//
// Parameters:
//   - fields: A map of key-value pairs to be added as fields to the logger.
//
// The function ensures that the fields map is initialized if it is nil
// and then adds the provided key-value pairs to the map.
func WithFields(fields map[string]string) Option {
	return func(opt *options) {
		if opt.fields == nil {
			opt.fields = make(map[string]string)
		}
		for k, v := range fields {
			opt.fields[k] = v
		}
	}
}

// WithTimeLayout returns an Option that configures the time layout used by the logger.
//
// The function accepts a string parameter that is used to set the time layout
// used by the logger. The time layout is used to format the time in log messages.
//
// The default time layout is "2006-01-02 15:04:05.000000 -0700 MST".
func WithTimeLayout(timeLayout string) Option {
	return func(opt *options) {
		opt.timeLayout = timeLayout
	}
}

// WithDisableConsole returns an Option that disables the console writer.
//
// When this option is used, the logger will not write logs to the console.
// This is useful when you want to log to a file only.
func WithDisableConsole() Option {
	return func(opt *options) {
		opt.disableConsole = true
	}
}

// WithLogDir returns an Option that sets the log directory for the logger.
//
// The log directory is where the logger will write log files. The default log
// directory is the current working directory.
//
// Parameters:
//   - logDir: The log directory to be used by the logger.
func WithLogDir(logDir string) Option {
	return func(opt *options) {
		opt.logDir = logDir
	}
}

// NewJsonLogger creates a new JSON formatted logger with the given options.
//
// This function validates the log configuration and initializes default options
// for the logger, such as the logging level, log directory, and time layout.
// It applies user-provided options to override these defaults. The logger writes
// logs to files in the specified log directory and can log to the console if enabled.
//
// It creates log cores for different log levels, ensuring the log directory exists,
// and combines them into a single core. The logger includes caller information and
// stack traces for errors. Additional fields configured through options are added
// to the logger.
//
// Parameters:
//   - opts: A variadic list of Option functions to configure the logger.
//
// Returns:
//   - *zap.Logger: A configured zap.Logger instance.
//   - error: An error if the logger cannot be created, such as when the log
//     configuration is not initialized or the log directory cannot be
//     created.
func NewJsonLogger(opts ...Option) (*zap.Logger, error) {
	// Validate configuration
	if config.LogConfig == nil {
		return nil, fmt.Errorf("log configuration is not initialized")
	}

	// Initialize default options
	opt := &options{
		level:  GetDefaultLevel(),
		fields: make(map[string]string),
		logDir: config.LogConfig.Log.LogDir,
	}

	// Apply provided options
	for _, f := range opts {
		f(opt)
	}

	// Validate log directory
	if opt.logDir == "" {
		return nil, fmt.Errorf("log directory is not configured")
	}

	// Set time layout
	timeLayout := common.DefaultTimeLayout
	if opt.timeLayout != "" {
		timeLayout = opt.timeLayout
	}

	// Create encoder configuration
	encoderConfig := createEncoderConfig(timeLayout)
	jsonEncoder := zapcore.NewJSONEncoder(encoderConfig)

	// Ensure log directory exists
	if err := os.MkdirAll(opt.logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create log writers and cores
	cores, errorWriter, err := createLogCores(jsonEncoder, opt)
	if err != nil {
		return nil, fmt.Errorf("failed to create log cores: %w", err)
	}

	// Combine all cores
	core := zapcore.NewTee(cores...)

	// Add console logging if enabled
	if !opt.disableConsole {
		consoleCores := createConsoleCores(jsonEncoder, opt.level)
		core = zapcore.NewTee(append(cores, consoleCores...)...)
	}

	// Create logger with options
	loggerOptions := []zap.Option{
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	}

	if errorWriter != nil {
		loggerOptions = append(loggerOptions, zap.ErrorOutput(zapcore.AddSync(errorWriter)))
	}

	zapLogger := zap.New(core, loggerOptions...)

	// Add configured fields
	if len(opt.fields) > 0 {
		fields := make([]zapcore.Field, 0, len(opt.fields))
		for k, v := range opt.fields {
			fields = append(fields, zap.String(k, v))
		}
		zapLogger = zapLogger.With(fields...)
	}

	return zapLogger, nil
}

// createEncoderConfig returns a zapcore.EncoderConfig configured with the specified
// time layout for encoding time fields. It defines the keys for different log
// components such as time, level, logger name, caller, message, and stack trace.
// The function also sets the encoding formats for log levels, time, duration, and
// caller information. The time is encoded using the provided timeLayout, allowing
// customization of the timestamp format in the logs.
func createEncoderConfig(timeLayout string) zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		TimeKey:       "time",
		LevelKey:      "level",
		NameKey:       "logger",
		CallerKey:     "caller",
		MessageKey:    "msg",
		StacktraceKey: "stacktrace",
		LineEnding:    zapcore.DefaultLineEnding,
		EncodeLevel:   zapcore.LowercaseLevelEncoder,
		EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format(timeLayout))
		},
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}

// createLogCores creates log cores for different log levels and returns a slice
// of zapcore.Core and an error writer.
//
// Parameters:
//   - encoder: The zapcore.Encoder used to encode log messages.
//   - opt: A pointer to an options struct containing configuration options,
//     including the log directory and file names.
//
// Returns:
//   - []zapcore.Core: A slice of zapcore.Core, each responsible for logging
//     at a specific log level.
//   - io.Writer: An error writer for handling errors during logging.
//   - error: An error if there is a failure in creating log writers or cores.
//
// This function creates log writers for debug, info, warn, and error levels
// by calling createLogWriter with paths constructed from opt.logDir and
// configuration file names. It then initializes zapcore.Core for each log level
// with corresponding level enablers.
func createLogCores(encoder zapcore.Encoder, opt *options) ([]zapcore.Core, io.Writer, error) {
	cfg := &config.LogConfig.Log

	// Create writers for each log level
	debugWriter, err := createLogWriter(filepath.Join(opt.logDir, cfg.LogFileDebug))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create debug writer: %w", err)
	}

	infoWriter, err := createLogWriter(filepath.Join(opt.logDir, cfg.LogFileInfo))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create info writer: %w", err)
	}

	warnWriter, err := createLogWriter(filepath.Join(opt.logDir, cfg.LogFileWarn))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create warn writer: %w", err)
	}

	errorWriter, err := createLogWriter(filepath.Join(opt.logDir, cfg.LogFileError))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create error writer: %w", err)
	}

	// Create cores for each level
	cores := []zapcore.Core{
		zapcore.NewCore(encoder, zapcore.AddSync(debugWriter), createLevelEnabler(zapcore.DebugLevel, zapcore.DebugLevel)),
		zapcore.NewCore(encoder, zapcore.AddSync(infoWriter), createLevelEnabler(zapcore.InfoLevel, zapcore.InfoLevel)),
		zapcore.NewCore(encoder, zapcore.AddSync(warnWriter), createLevelEnabler(zapcore.WarnLevel, zapcore.WarnLevel)),
		zapcore.NewCore(encoder, zapcore.AddSync(errorWriter), createLevelEnabler(zapcore.ErrorLevel, zapcore.FatalLevel)),
	}

	return cores, errorWriter, nil
}

// createConsoleCores returns zapcore.Cores that write logs to the console.
//
// This function creates two zapcore.Cores: one for standard output (stdout)
// and another for standard error (stderr). It uses the provided encoder to
// format log entries. The stdout core logs messages with levels from minLevel
// up to, but not including, the warn level. The stderr core logs messages
// with levels from error to fatal.
//
// Parameters:
//   - encoder: zapcore.Encoder used to format the log entries.
//   - minLevel: zapcore.Level specifying the minimum log level for stdout.
//
// Returns:
//   - []zapcore.Core: A slice containing the stdout and stderr cores.
func createConsoleCores(encoder zapcore.Encoder, minLevel zapcore.Level) []zapcore.Core {
	stdout := zapcore.Lock(os.Stdout)
	stderr := zapcore.Lock(os.Stderr)

	return []zapcore.Core{
		zapcore.NewCore(encoder, stdout, createLevelEnabler(minLevel, zapcore.WarnLevel-1)),
		zapcore.NewCore(encoder, stderr, createLevelEnabler(zapcore.ErrorLevel, zapcore.FatalLevel)),
	}
}

// createLevelEnabler returns a zap.LevelEnablerFunc that enables logging for
// levels in the range [min, max] (inclusive).
//
// Parameters:
//   - min: The minimum zapcore.Level for which logging is enabled.
//   - max: The maximum zapcore.Level for which logging is enabled.
//
// Returns:
//   - zap.LevelEnablerFunc: A function that takes a zapcore.Level and returns a
//     boolean indicating whether logging is enabled for that level.
func createLevelEnabler(min, max zapcore.Level) zap.LevelEnablerFunc {
	return func(lvl zapcore.Level) bool {
		return lvl >= min && lvl <= max
	}
}

// createLogWriter creates and returns a log writer for the specified filename.
//
// This function checks if the provided filename is not empty and ensures that
// the directory for the file exists, creating it if necessary. It then creates
// a new lumberjack.Logger configured with the log settings, which include the
// maximum size, number of backups, age, and compression options.
//
// Parameters:
//   - filename: A string representing the path to the log file.
//
// Returns:
//   - io.Writer: A writer interface for the created log file.
//   - error: An error if the filename is empty or the directory cannot be created.
func createLogWriter(filename string) (io.Writer, error) {
	if filename == "" {
		return nil, fmt.Errorf("filename cannot be empty")
	}

	// Ensure the directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	cfg := &config.LogConfig.Log
	return &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		LocalTime:  cfg.LocalTime,
		Compress:   cfg.Compress,
	}, nil
}

// Close synchronizes the logger, flushing any buffered log entries to their
// respective writers. This should be called when the program is exiting.
//
// Parameters:
//   - zapLogger: A pointer to the zap.Logger to be closed. If nil, this function
//     returns immediately without error.
//
// Returns:
//   - error: An error if the logger cannot be synchronized.
func Close(zapLogger *zap.Logger) error {
	if zapLogger == nil {
		return nil
	}
	return zapLogger.Sync()
}

// GetDefaultLevel returns the default logging level defined in the configuration.
//
// If the log configuration is not initialized, it defaults to the Info level.
// The function checks the DefaultLevel field of the log configuration and
// maps it to the corresponding zapcore.Level. Supported log levels are
// Debug, Info, Warn, and Error. If the DefaultLevel is not recognized,
// it defaults to the Info level.
//
// Returns:
//   - zapcore.Level: The logging level based on the configuration or
//     the default Info level if the configuration is not available.
func GetDefaultLevel() zapcore.Level {
	if config.LogConfig == nil {
		return zapcore.InfoLevel
	}

	switch config.LogConfig.Log.DefaultLevel {
	case common.LogLevelDebug:
		return zapcore.DebugLevel
	case common.LogLevelInfo:
		return zapcore.InfoLevel
	case common.LogLevelWarn:
		return zapcore.WarnLevel
	case common.LogLevelError:
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// ValidateLogLevel checks if the provided log level is valid and returns an
// error if not. Valid log levels are debug, info, warn, and error.
//
// Parameters:
//   - level: A string representing the log level to be validated.
//
// Returns:
//   - error: An error if the log level is invalid, otherwise nil.
func ValidateLogLevel(level string) error {
	switch level {
	case common.LogLevelDebug, common.LogLevelInfo, common.LogLevelWarn, common.LogLevelError:
		return nil
	default:
		return fmt.Errorf("invalid log level: %s, must be one of: debug, info, warn, error", level)
	}
}

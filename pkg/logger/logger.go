package logger

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"go-gin-project/library/common"
	"go-gin-project/library/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Option func(o *option)

// option is the configuration options for the logger.
type option struct {
	level          zapcore.Level
	fields         map[string]string
	timeLayout     string
	disableConsole bool
	logDir         string
}

// WithDebugLevel returns an Option that sets the logging level to
// zapcore.DebugLevel.
//
// DebugLevel is used to log detailed information, typically of interest
// only when diagnosing problems.
//
// Example:
//
//	 logger, err := logger.NewJsonLogger(
//			logger.WithDebugLevel(),
//		)
//
// This will log all messages with a level of Debug and above.
func WithDebugLevel() Option {
	return func(opt *option) {
		opt.level = zapcore.DebugLevel
	}
}

// WithWarnLevel returns an Option that sets the logging level to
// zapcore.WarnLevel.
//
// WarnLevel is a level that is used when an event has occurred that may
// potentially cause a problem.
//
// Example:
//
//	 logger, err := logger.NewJsonLogger(
//			logger.WithWarnLevel(),
//		)
func WithWarnLevel() Option {
	return func(opt *option) {
		opt.level = zapcore.WarnLevel
	}
}

// WithErrorLevel returns an Option that sets the logging level to
// zapcore.ErrorLevel.
//
// ErrorLevel is a level used when an error has occurred.
//
// Example:
//
//	 logger, err := logger.NewJsonLogger(
//			logger.WithErrorLevel(),
//		)
func WithErrorLevel() Option {
	return func(opt *option) {
		opt.level = zapcore.ErrorLevel
	}
}

// WithField returns an Option that sets a field to the logger.
//
// The key argument specifies the key of the field, and the val argument
// specifies the value of the field.
//
// If the key is already present in the logger, the value will be
// overwritten.
//
// Example:
//
//	 logger, err := logger.NewJsonLogger(
//			logger.WithField("service", "myapp"),
//		)
func WithField(key, val string) Option {
	return func(opt *option) {
		if opt.fields == nil {
			opt.fields = make(map[string]string)
		}
		opt.fields[key] = val
	}
}

// WithTimeLayout returns an Option that sets the time layout for the logger.
//
// The time layout is used to format the time field in the log messages.
// The time layout must be a valid time layout string as specified in the
// time package.
//
// Example:
//
//	 logger, err := logger.NewJsonLogger(
//			logger.WithTimeLayout("2006-01-02 15:04:05"),
//		)
func WithTimeLayout(timeLayout string) Option {
	return func(opt *option) {
		opt.timeLayout = timeLayout
	}
}

// WithDisableConsole returns an Option that disables console logging.
//
// When this option is applied, log messages will not be printed to the console
// but will still be written to the configured log files.
//
// Example:
//
//	 logger, err := logger.NewJsonLogger(
//			logger.WithDisableConsole(),
//		)
func WithDisableConsole() Option {
	return func(opt *option) {
		// Set the disableConsole flag to true to disable console logging.
		opt.disableConsole = true
	}
}

// WithLogDir returns an Option that sets the log directory for the logger.
//
// The logDir argument specifies the log directory. If the logDir is empty,
// the logger will not write log messages to files.
//
// Example:
//
//	 logger, err := logger.NewJsonLogger(
//			logger.WithLogDir("/var/log"),
//		)
func WithLogDir(logDir string) Option {
	return func(opt *option) {
		opt.logDir = logDir
	}
}

// NewJsonLogger creates a new JSON logger with the given options.
//
// The option argument is a variadic list of Option functions that customize
// the logger. The available options are:
//
//   - WithLevel: sets the log level to the specified level.
//   - WithFields: sets the log fields to the specified values.
//   - WithTimeLayout: sets the time layout to the specified format.
//   - WithDisableConsole: disables console logging.
//   - WithLogDir: sets the log directory to the specified path.
func NewJsonLogger(opts ...Option) (*zap.Logger, error) {
	// Initialize the logger configuration with default values
	opt := &option{
		level:  GetDefaultLevel(),
		fields: make(map[string]string),
		logDir: config.LogConfig.Log.LogDir,
	}

	// Apply the provided options to the logger configuration
	for _, f := range opts {
		f(opt)
	}

	// Set the time layout for the logger
	timeLayout := common.DefaultTimeLayout
	if opt.timeLayout != "" {
		timeLayout = opt.timeLayout
	}

	// Configure the encoder for the logger
	encoderConfig := zapcore.EncoderConfig{
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

	// Create a JSON encoder using the configured encoder settings
	jsonEncoder := zapcore.NewJSONEncoder(encoderConfig)

	// Ensure the log directory exists, creating it if necessary
	if err := os.MkdirAll(opt.logDir, 0766); err != nil {
		return nil, err
	}

	// Create log writers for different log levels
	debugWriter := newLogWriter(filepath.Join(opt.logDir, config.LogConfig.Log.LogFileDebug))
	infoWriter := newLogWriter(filepath.Join(opt.logDir, config.LogConfig.Log.LogFileInfo))
	warnWriter := newLogWriter(filepath.Join(opt.logDir, config.LogConfig.Log.LogFileWarn))
	errorWriter := newLogWriter(filepath.Join(opt.logDir, config.LogConfig.Log.LogFileError))

	// Create cores for each log level, specifying the encoder and writer
	debugCore := zapcore.NewCore(
		jsonEncoder,
		zapcore.AddSync(debugWriter),
		zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl == zapcore.DebugLevel
		}),
	)

	infoCore := zapcore.NewCore(
		jsonEncoder,
		zapcore.AddSync(infoWriter),
		zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl == zapcore.InfoLevel
		}),
	)

	warnCore := zapcore.NewCore(
		jsonEncoder,
		zapcore.AddSync(warnWriter),
		zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl == zapcore.WarnLevel
		}),
	)

	errorCore := zapcore.NewCore(
		jsonEncoder,
		zapcore.AddSync(errorWriter),
		zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl >= zapcore.ErrorLevel
		}),
	)

	// Combine the cores into a single tee core
	core := zapcore.NewTee(debugCore, infoCore, warnCore, errorCore)

	// Optionally add console logging to the core
	if !opt.disableConsole {
		stdout := zapcore.Lock(os.Stdout)
		stderr := zapcore.Lock(os.Stderr)

		core = zapcore.NewTee(core,
			zapcore.NewCore(jsonEncoder, zapcore.NewMultiWriteSyncer(stdout), zapcore.InfoLevel),
			zapcore.NewCore(jsonEncoder, zapcore.NewMultiWriteSyncer(stderr), zapcore.ErrorLevel),
		)
	}

	// Create the final logger with the combined core and caller information
	logger := zap.New(core, zap.AddCaller(), zap.ErrorOutput(zapcore.AddSync(errorWriter)))

	// Add any configured fields to the logger
	for k, v := range opt.fields {
		logger = logger.WithOptions(zap.Fields(zapcore.Field{Key: k, Type: zapcore.StringType, String: v}))
	}

	return logger, nil
}

// newLogWriter creates a new log writer that writes to a file.
// It returns a lumberjack.Logger that implements the io.Writer interface.
// The logger will write to the file specified by the file parameter and
// will rotate the file when it reaches the maximum size specified by the
// MaxSize parameter. The logger will also keep a maximum of 300 backups
// and will delete any backups older than 30 days.
//
// The logger will also use the local time zone when writing log entries.
// If the Compress option is true, the logger will compress the log
// entries using gzip.
//
// Parameters:
//   - file: The name of the file to write to.
//
// Returns:
//   - A lumberjack.Logger that implements the io.Writer interface.
func newLogWriter(file string) io.Writer {
	// Create a new lumberjack.Logger that implements the io.Writer interface
	// The logger will write to the file specified by the file parameter and
	// will rotate the file when it reaches the maximum size specified by the
	// MaxSize parameter. The logger will also keep a maximum of 300 backups
	// and will delete any backups older than 30 days.
	// The logger will also use the local time zone when writing log entries.
	// If the Compress option is true, the logger will compress the log
	// entries using gzip.
	return &lumberjack.Logger{
		Filename:   file,
		MaxSize:    config.LogConfig.Log.MaxSize,
		MaxBackups: config.LogConfig.Log.MaxBackups,
		MaxAge:     config.LogConfig.Log.MaxAge,
		LocalTime:  config.LogConfig.Log.LocalTime,
		Compress:   config.LogConfig.Log.Compress,
	}
}

// Close gracefully closes the provided zap.Logger.
// It ensures that all buffered log entries are written out before returning.
// If the logger is nil, it returns immediately without an error.
// The function is safe to call from multiple goroutines.
func Close(logger *zap.Logger) error {
	if logger == nil {
		// No logger to close, return immediately.
		return nil
	}
	// Sync the logger to flush all buffered log entries.
	// This ensures that all log entries are written out before returning.
	// The Sync function is safe to call from multiple goroutines.
	return logger.Sync()
}

// GetDefaultLevel returns the default log level for the logger.
//
// It checks the DefaultLevel field in the LogConfig struct and returns
// the corresponding zapcore.Level value.
// If the value in the configuration is invalid, it returns zapcore.InfoLevel.
func GetDefaultLevel() zapcore.Level {
	// Check the value in the configuration and return the corresponding zapcore.Level
	// The default log level is determined by the value in the DefaultLevel field
	// of the LogConfig struct. The field can be one of the following values:
	//  - "debug"
	//  - "info"
	//  - "warn"
	//  - "error"
	// If the value in the configuration is invalid, it returns zapcore.InfoLevel
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
		// If the value in the configuration is invalid, return zapcore.InfoLevel
		return zapcore.InfoLevel
	}
}

package logger

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	DefaultLevel      = zapcore.InfoLevel
	DefaultTimeLayout = time.RFC3339
)

type Option func(o *option)

type option struct {
	level          zapcore.Level
	fields         map[string]string
	file           io.Writer
	timeLayout     string
	disableConsole bool
}

// WithDebugLevel returns an Option that sets the logger level to zapcore.DebugLevel
func WithDebugLevel() Option {
	return func(opt *option) {
		opt.level = zapcore.DebugLevel
	}
}

// WithWarnLevel returns an Option that sets the logger level to zapcore.WarnLevel.
// Logging events at or above this level will be emitted.
func WithWarnLevel() Option {
	return func(opt *option) {
		opt.level = zapcore.WarnLevel
	}
}

// WithErrorLevel returns an Option that sets the logger level to zapcore.ErrorLevel.
// Logging events at or above this level will be emitted.
func WithErrorLevel() Option {
	return func(opt *option) {
		opt.level = zapcore.ErrorLevel
	}
}

// WithFiled returns an Option that sets the logger fields to the given key-value pair.
// The field key will be used as the key in the zapcore.Field, and the field val will be used as the value.
// If the key is empty, the Option is a no-op.
func WithFiled(key, val string) Option {
	return func(opt *option) {
		opt.fields[key] = val
	}
}

// WithFileP returns an Option that sets the logger output to a specified file.
// It ensures that the directory structure for the file path exists, creating directories as needed.
// The file is opened with append and read/write permissions. If any error occurs during directory creation
// or file opening, the function will panic.
//
// The file path can be either an absolute path or a relative path. If the path is relative, it is
// resolved relative to the current working directory.
func WithFileP(file string) Option {
	// Ensure the directory structure exists
	dir := filepath.Dir(file)
	if err := os.MkdirAll(dir, 0766); err != nil {
		panic(err)
	}

	// Open the file with append and read/write permissions
	f, err := os.OpenFile(file, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0766)
	if err != nil {
		panic(err)
	}

	// Return an Option that sets the logger output to the file
	return func(opt *option) {
		opt.file = f
	}
}

// WithFileRelationP returns an Option that sets the logger output to a lumberjack.Logger.
//
// The lumberjack.Logger is a rolling logger that rotates files based on size and age.
// The file path can be either an absolute path or a relative path. If the path is relative, it is
// resolved relative to the current working directory.
//
// The MaxSize, MaxBackups, and MaxAge fields of the lumberjack.Logger are set to 128M, 300, and 30 days,
// respectively. LocalTime is set to true, and Compress is set to true.
//
// If any error occurs during directory creation or file opening, the function will panic.
func WithFileRelationP(file string) Option {
	// Ensure the directory structure exists
	dir := filepath.Dir(file)
	if err := os.MkdirAll(dir, 0766); err != nil {
		panic(err)
	}

	// Return an Option that sets the logger output to a lumberjack.Logger
	return func(opt *option) {
		opt.file = &lumberjack.Logger{
			Filename:   file, // 文件路径
			MaxSize:    128,  // 单个文件最大尺寸，默认单位 M
			MaxBackups: 300,  // 最多保留 300 个备份
			MaxAge:     30,   // 最大时间，默认单位 day
			LocalTime:  true, // 使用本地时间
			Compress:   true, // 是否压缩 disabled by default
		}
	}
}

// WithTimeLayout returns an Option that sets the time layout for the logger.
// The timeLayout parameter specifies the format in which timestamps will be
// logged. It should be a valid Go time format string. If the provided timeLayout
// is empty or invalid, the default time layout will be used.
func WithTimeLayout(timeLayout string) Option {
	return func(opt *option) {
		opt.timeLayout = timeLayout
	}
}

// WithDisableConsole returns an Option that sets the logger to disable console output.
// It is used to disable console output when the logger is used in a server environment.
func WithDisableConsole() Option {
	return func(opt *option) {
		opt.disableConsole = true
	}
}

// NewJsonLogger creates a new JSON-formatted zap.Logger with the provided options.
// It configures the logger's level, fields, output file, time layout, and console output
// settings based on the given Option functions. By default, it logs to both stdout and stderr,
// but console output can be disabled using the WithDisableConsole option.
// The logger supports optional fields, which can be set with WithFiled.
// If a file is provided with WithFileP or WithFileRelationP, logs are additionally written to that file.
// The function returns the configured zap.Logger or an error if the configuration fails.
func NewJsonLogger(opts ...Option) (*zap.Logger, error) {
	opt := &option{
		level:  DefaultLevel,
		fields: make(map[string]string),
	}

	for _, f := range opts {
		f(opt)
	}
	timeLayout := DefaultTimeLayout
	if opt.timeLayout != "" {
		timeLayout = opt.timeLayout
	}

	// Configure the JSON encoder
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:       "time",
		LevelKey:      "level",
		NameKey:       "logger", // used by logger.Named(key); optional; useless
		CallerKey:     "caller",
		MessageKey:    "msg",
		StacktraceKey: "stacktrace", // use by zap.AddStacktrace; optional; useless
		LineEnding:    zapcore.DefaultLineEnding,
		EncodeLevel:   zapcore.LowercaseLevelEncoder, // 小写编码器
		EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format(timeLayout))
		},
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder, // 全路径编码器
	}

	jsonEncoder := zapcore.NewJSONEncoder(encoderConfig)

	// Configure the low-priority and high-priority writers
	lowPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= opt.level && lvl < zapcore.ErrorLevel
	})

	highPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= opt.level && lvl >= zapcore.ErrorLevel
	})

	stdout := zapcore.Lock(os.Stdout)
	stderr := zapcore.Lock(os.Stderr)
	core := zapcore.NewTee()

	if !opt.disableConsole {
		// Add the console output streams
		core = zapcore.NewTee(
			zapcore.NewCore(jsonEncoder, zapcore.NewMultiWriteSyncer(stdout), lowPriority),
			zapcore.NewCore(jsonEncoder, zapcore.NewMultiWriteSyncer(stderr), highPriority),
		)
	}
	if opt.file != nil {
		// Add the file output stream
		core = zapcore.NewTee(core,
			zapcore.NewCore(jsonEncoder, zapcore.AddSync(opt.file), zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
				return lvl >= opt.level
			}),
			),
		)
	}

	logger := zap.New(core, zap.AddCaller(), zap.ErrorOutput(stderr))

	// Add the optional fields
	for k, v := range opt.fields {
		logger = logger.WithOptions(zap.Fields(zapcore.Field{Key: k, Type: zapcore.StringType, String: v}))
	}

	return logger, nil
}

package logger

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/xiebingnote/go-gin-project/library/common"
	"github.com/xiebingnote/go-gin-project/library/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Option configures a logger. When options overlap, the last one wins.
type Option func(o *options)

type options struct {
	level          zapcore.Level
	fields         map[string]string
	timeLayout     string
	disableConsole bool
	logDir         string
}

// WithLevel sets the inclusive minimum level for both files and the console.
// Levels from Debug through Fatal are supported.
func WithLevel(level zapcore.Level) Option {
	return func(opt *options) { opt.level = level }
}

func WithDebugLevel() Option { return WithLevel(zapcore.DebugLevel) }
func WithInfoLevel() Option  { return WithLevel(zapcore.InfoLevel) }
func WithWarnLevel() Option  { return WithLevel(zapcore.WarnLevel) }
func WithErrorLevel() Option { return WithLevel(zapcore.ErrorLevel) }

// WithField adds a string field to every log entry.
func WithField(key, val string) Option {
	return func(opt *options) {
		if opt.fields == nil {
			opt.fields = make(map[string]string)
		}
		opt.fields[key] = val
	}
}

// WithFields snapshots fields so callers can reuse the option after changing
// their original map. Later options can override individual fields.
func WithFields(fields map[string]string) Option {
	copyFields := make(map[string]string, len(fields))
	for key, value := range fields {
		copyFields[key] = value
	}
	return func(opt *options) {
		if opt.fields == nil {
			opt.fields = make(map[string]string)
		}
		for key, value := range copyFields {
			opt.fields[key] = value
		}
	}
}

// WithTimeLayout overrides common.DefaultTimeLayout. An empty layout uses the default.
func WithTimeLayout(layout string) Option {
	return func(opt *options) { opt.timeLayout = layout }
}

// WithDisableConsole disables application logs on stdout/stderr. Internal zap
// write failures still use stderr so a failing log file cannot hide its own error.
func WithDisableConsole() Option {
	return func(opt *options) { opt.disableConsole = true }
}

// WithLogDir overrides the log directory from config.LogConfig.
func WithLogDir(dir string) Option {
	return func(opt *options) { opt.logDir = dir }
}

// NewJsonLogger creates a logger with separate debug, info, warn and error files.
// Error and higher levels share the error file. Configuration is copied at
// construction time; log filenames must be distinct relative paths.
// Call Close after writers stop, including when the logger was derived with With.
func NewJsonLogger(opts ...Option) (*zap.Logger, error) {
	if config.LogConfig == nil {
		return nil, fmt.Errorf("log configuration is not initialized")
	}
	cfg := *config.LogConfig
	if err := ValidateLogLevel(cfg.Log.DefaultLevel); err != nil {
		return nil, err
	}
	level, err := zapcore.ParseLevel(cfg.Log.DefaultLevel)
	if err != nil {
		return nil, err
	}
	opt := &options{level: level, fields: make(map[string]string), logDir: cfg.Log.LogDir}
	for _, apply := range opts {
		if apply == nil {
			return nil, fmt.Errorf("logger option must not be nil")
		}
		apply(opt)
	}
	if opt.logDir == "" {
		return nil, fmt.Errorf("log directory is not configured")
	}
	if opt.level < zapcore.DebugLevel || opt.level > zapcore.FatalLevel {
		return nil, fmt.Errorf("invalid minimum log level: %d", opt.level)
	}
	if cfg.Log.MaxSize < 0 || cfg.Log.MaxAge < 0 || cfg.Log.MaxBackups < 0 {
		return nil, fmt.Errorf("log rotation settings must not be negative")
	}
	if int64(cfg.Log.MaxSize) > (1<<63-1)/(1024*1024) || int64(cfg.Log.MaxAge) > (1<<63-1)/int64(24*time.Hour) {
		return nil, fmt.Errorf("log rotation settings exceed supported limits")
	}
	layout := opt.timeLayout
	if layout == "" {
		layout = common.DefaultTimeLayout
	}
	encoder := zapcore.NewJSONEncoder(createEncoderConfig(layout))
	cores, files, err := createLogCores(encoder, opt, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create log cores: %w", err)
	}
	if !opt.disableConsole {
		cores = append(cores, createConsoleCores(encoder, opt.level)...)
	}
	core := &managedCore{Core: zapcore.NewTee(cores...), resources: &logResources{files: files}}
	loggerOptions := []zap.Option{zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel)}
	if isValidFileDescriptor(os.Stderr) {
		loggerOptions = append(loggerOptions, zap.ErrorOutput(zapcore.Lock(consoleWriter{os.Stderr})))
	}
	log := zap.New(core, loggerOptions...)
	keys := make([]string, 0, len(opt.fields))
	for key := range opt.fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	fields := make([]zap.Field, 0, len(keys))
	for _, key := range keys {
		fields = append(fields, zap.String(key, opt.fields[key]))
	}
	return log.With(fields...), nil
}

func createEncoderConfig(layout string) zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		TimeKey: "time", LevelKey: "level", NameKey: "logger", CallerKey: "caller",
		MessageKey: "msg", StacktraceKey: "stacktrace", LineEnding: zapcore.DefaultLineEnding,
		EncodeLevel: zapcore.LowercaseLevelEncoder,
		EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format(layout))
		},
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}

func createLogCores(encoder zapcore.Encoder, opt *options, cfg *config.LogConfigEntry) ([]zapcore.Core, []*logWriter, error) {
	specs := []struct {
		name     string
		min, max zapcore.Level
	}{
		{cfg.Log.LogFileDebug, zapcore.DebugLevel, zapcore.DebugLevel},
		{cfg.Log.LogFileInfo, zapcore.InfoLevel, zapcore.InfoLevel},
		{cfg.Log.LogFileWarn, zapcore.WarnLevel, zapcore.WarnLevel},
		{cfg.Log.LogFileError, zapcore.ErrorLevel, zapcore.FatalLevel},
	}
	// Validate all names before creating files; independent rotating writers
	// must never own the same path.
	seen := make(map[string]bool, len(specs))
	for _, spec := range specs {
		name := filepath.Clean(spec.name)
		if !filepath.IsLocal(spec.name) || name == "." {
			return nil, nil, fmt.Errorf("invalid log filename: %q", spec.name)
		}
		if seen[name] {
			return nil, nil, fmt.Errorf("duplicate log filename: %q", spec.name)
		}
		seen[name] = true
	}
	var cores []zapcore.Core
	var files []*logWriter
	var identities []os.FileInfo
	complete := false
	defer func() {
		if !complete {
			for _, file := range files {
				_ = file.Close()
			}
		}
	}()
	for _, spec := range specs {
		path := filepath.Join(opt.logDir, spec.name)
		writer, err := createLogWriter(path, cfg)
		if err != nil {
			return nil, nil, err
		}
		files = append(files, writer)
		info, err := os.Stat(path)
		if err != nil {
			return nil, nil, err
		}
		for _, previous := range identities {
			if os.SameFile(previous, info) {
				return nil, nil, fmt.Errorf("log files refer to the same file: %s", path)
			}
		}
		identities = append(identities, info)
		cores = append(cores, zapcore.NewCore(encoder.Clone(), writer, createLevelEnabler(max(opt.level, spec.min), spec.max)))
	}
	complete = true
	return cores, files, nil
}

// Info and lower go to stdout; Warn and higher go to stderr. Both obey minLevel.
func createConsoleCores(encoder zapcore.Encoder, minLevel zapcore.Level) []zapcore.Core {
	var cores []zapcore.Core
	if isValidFileDescriptor(os.Stdout) {
		cores = append(cores, zapcore.NewCore(encoder.Clone(), zapcore.Lock(consoleWriter{os.Stdout}), createLevelEnabler(minLevel, zapcore.InfoLevel)))
	}
	if isValidFileDescriptor(os.Stderr) {
		cores = append(cores, zapcore.NewCore(encoder.Clone(), zapcore.Lock(consoleWriter{os.Stderr}), createLevelEnabler(max(minLevel, zapcore.WarnLevel), zapcore.FatalLevel)))
	}
	return cores
}

func createLevelEnabler(min, max zapcore.Level) zap.LevelEnablerFunc {
	return func(level zapcore.Level) bool { return level >= min && level <= max }
}

// consoleWriter syncs redirected regular files, but not pipes or terminals.
// Stat still detects closed descriptors; regular-file I/O failures propagate.
type consoleWriter struct{ *os.File }

func (w consoleWriter) Sync() error {
	info, err := w.File.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return nil
	}
	return w.File.Sync()
}

// logWriter prevents lumberjack from reopening a file after Close. Its lock
// also makes closing safe against a write already in progress.
type logWriter struct {
	mu       sync.Mutex
	writer   io.WriteCloser
	closed   bool
	closeErr error
}

func (w *logWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return 0, os.ErrClosed
	}
	return w.writer.Write(p)
}

func (w *logWriter) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return nil
	}
	if syncer, ok := w.writer.(interface{ Sync() error }); ok {
		return syncer.Sync()
	}
	// Lumberjack writes directly to os.File, with no user-space buffer. Its API
	// does not expose fsync; Sync must not close/reopen the rolling file.
	return nil
}

func (w *logWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.closed {
		w.closed = true
		w.closeErr = w.writer.Close()
	}
	return w.closeErr
}

func createLogWriter(filename string, cfg *config.LogConfigEntry) (*logWriter, error) {
	if filename == "" {
		return nil, fmt.Errorf("filename cannot be empty")
	}
	if cfg == nil {
		return nil, fmt.Errorf("log configuration is not initialized")
	}
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}
	if info, err := os.Lstat(filename); err == nil && !info.Mode().IsRegular() {
		return nil, fmt.Errorf("log path must be a regular file: %s", filename)
	} else if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("check log file: %w", err)
	}
	// Lumberjack opens lazily. Probe append access now without truncating existing
	// logs or starting its rotation worker, so startup cannot silently succeed
	// with an unwritable file. Its own files use mode 0600 as well.
	probe, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}
	if err := probe.Close(); err != nil {
		return nil, fmt.Errorf("close log file probe: %w", err)
	}
	return &logWriter{writer: &lumberjack.Logger{
		Filename: filename, MaxSize: cfg.Log.MaxSize, MaxBackups: cfg.Log.MaxBackups,
		MaxAge: cfg.Log.MaxAge, LocalTime: cfg.Log.LocalTime, Compress: cfg.Log.Compress,
	}}, nil
}

type logResources struct {
	files []*logWriter
	once  sync.Once
	err   error
}

type managedCore struct {
	zapcore.Core
	resources *logResources
}

func (c *managedCore) With(fields []zapcore.Field) zapcore.Core {
	return &managedCore{Core: c.Core.With(fields), resources: c.resources}
}

func (c *managedCore) Close() error {
	c.resources.once.Do(func() {
		errs := []error{c.Core.Sync()}
		for _, file := range c.resources.files {
			errs = append(errs, file.Close())
		}
		c.resources.err = errors.Join(errs...)
	})
	return c.resources.err
}

// Close flushes and closes files owned by NewJsonLogger, once for the entire
// logger family (including With/Named children). It does not close stdout/stderr.
// Other cores use their Close method if available, or fall back to Sync.
func Close(log *zap.Logger) error {
	if log == nil {
		return nil
	}
	if closer, ok := log.Core().(io.Closer); ok {
		return closer.Close()
	}
	return log.Sync()
}

// GetDefaultLevel returns Info if the configuration is missing or invalid.
func GetDefaultLevel() zapcore.Level {
	if config.LogConfig == nil || ValidateLogLevel(config.LogConfig.Log.DefaultLevel) != nil {
		return zapcore.InfoLevel
	}
	level, _ := zapcore.ParseLevel(config.LogConfig.Log.DefaultLevel)
	return level
}

// ValidateLogLevel checks the four supported configuration levels.
func ValidateLogLevel(level string) error {
	switch level {
	case common.LogLevelDebug, common.LogLevelInfo, common.LogLevelWarn, common.LogLevelError:
		return nil
	default:
		return fmt.Errorf("invalid log level: %s, must be one of: debug, info, warn, error", level)
	}
}

func isValidFileDescriptor(file *os.File) bool {
	if file == nil {
		return false
	}
	_, err := file.Stat()
	return err == nil
}

package logger

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/term"
)

// Config drives how the zap logger is built.
type Config struct {
	Development bool
	Level       string
	Encoding    string
}

var (
	mu     sync.RWMutex
	global *zap.Logger
	colors = shouldColorize()
)

// Init builds a zap logger with the provided config and stores it globally.
func Init(cfg Config) (*zap.Logger, error) {
	l, err := New(cfg)
	if err != nil {
		return nil, err
	}

	mu.Lock()
	defer mu.Unlock()

	if global != nil {
		_ = global.Sync()
	}

	global = l
	return global, nil
}

// MustInit panics if the logger cannot be built.
func MustInit(cfg Config) *zap.Logger {
	l, err := Init(cfg)
	if err != nil {
		panic(err)
	}
	return l
}

// L returns the global logger, creating a development logger if Init was never called.
func L() *zap.Logger {
	mu.RLock()
	l := global
	mu.RUnlock()
	if l != nil {
		return l
	}

	mu.Lock()
	defer mu.Unlock()
	if global == nil {
		dev, err := zap.NewDevelopment()
		if err != nil {
			global = zap.NewNop()
		} else {
			global = dev
		}
	}
	return global
}

// Sync flushes any buffered log entries on the global logger.
func Sync() error {
	mu.RLock()
	l := global
	mu.RUnlock()

	if l == nil {
		return nil
	}

	if err := l.Sync(); err != nil {
		if errors.Is(err, syscall.ENOTTY) || errors.Is(err, os.ErrInvalid) {
			return nil
		}
		return err
	}
	return nil
}

// New returns a zap.Logger configured according to cfg.
func New(cfg Config) (*zap.Logger, error) {
	var zapCfg zap.Config
	if cfg.Development {
		zapCfg = zap.NewDevelopmentConfig()
	} else {
		zapCfg = zap.NewProductionConfig()
	}

	if cfg.Encoding != "" {
		zapCfg.Encoding = cfg.Encoding
	}

	zapCfg.EncoderConfig = buildEncoderConfig(zapCfg.Encoding)

	if cfg.Level != "" {
		level := zapcore.InfoLevel
		if err := level.UnmarshalText([]byte(strings.ToLower(cfg.Level))); err != nil {
			return nil, fmt.Errorf("logger: invalid level %q: %w", cfg.Level, err)
		}
		zapCfg.Level = zap.NewAtomicLevelAt(level)
	}

	return zapCfg.Build(zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
}

func buildEncoderConfig(encoding string) zapcore.EncoderConfig {
	cfg := zapcore.EncoderConfig{
		TimeKey:          "time",
		LevelKey:         "level",
		NameKey:          "logger",
		CallerKey:        "caller",
		MessageKey:       "msg",
		StacktraceKey:    "stack",
		LineEnding:       zapcore.DefaultLineEnding,
		EncodeDuration:   zapcore.StringDurationEncoder,
		EncodeCaller:     zapcore.ShortCallerEncoder,
		EncodeName:       zapcore.FullNameEncoder,
		ConsoleSeparator: " | ",
	}

	if encoding == "console" {
		cfg.EncodeLevel = prettyLevelEncoder
		cfg.EncodeTime = prettyTimeEncoder
	} else {
		cfg.ConsoleSeparator = " "
		cfg.EncodeLevel = zapcore.LowercaseLevelEncoder
		cfg.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	return cfg
}

func prettyTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
}

func prettyLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	label := fmt.Sprintf("%-5s", strings.ToUpper(level.String()))
	if colors {
		enc.AppendString(levelColor(level) + label + colorReset)
		return
	}
	enc.AppendString(label)
}

func shouldColorize() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	file := os.Stdout
	if file == nil {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}

const (
	colorReset   = "\x1b[0m"
	colorGreen   = "\x1b[32m"
	colorCyan    = "\x1b[36m"
	colorYellow  = "\x1b[33m"
	colorRed     = "\x1b[31m"
	colorMagenta = "\x1b[35m"
)

func levelColor(level zapcore.Level) string {
	switch level {
	case zapcore.DebugLevel:
		return colorCyan
	case zapcore.InfoLevel:
		return colorGreen
	case zapcore.WarnLevel:
		return colorYellow
	case zapcore.ErrorLevel:
		return colorRed
	case zapcore.DPanicLevel, zapcore.PanicLevel:
		return colorMagenta
	case zapcore.FatalLevel:
		return colorRed
	default:
		return colorGreen
	}
}

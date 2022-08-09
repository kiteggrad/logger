package logger

import (
	"fmt"
	"os"
	"syscall"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// zap doesn't have predefined const to do nothing on fatal. So, define it ourself
const doNothingOnFatal zapcore.CheckWriteAction = 100

// Logger is a wrapper for *zap.SugaredLogger compatible with logrus.FieldLogger
type Logger struct {
	zap   *zap.SugaredLogger
	level zap.AtomicLevel
}

type Config struct {
	// DisableStdOut disables loggig to stdout
	DisableStdOut bool
	// DisableColor disables colored output
	DisableColor bool
	// Files is a list of file paths to write logging output to
	Files []string
}

// New creates a new logger
func New(cfg Config) (logger *Logger, err error) {
	level := zap.NewAtomicLevelAt(zap.DebugLevel)

	var outputPaths []string
	if !cfg.DisableStdOut {
		outputPaths = append(outputPaths, "stdout")
	}
	if cfg.Files != nil {
		outputPaths = append(outputPaths, cfg.Files...)
	}

	levelEncoder := zapcore.CapitalColorLevelEncoder
	if cfg.DisableColor {
		levelEncoder = zapcore.CapitalLevelEncoder
	}

	zapCfg := zap.Config{
		Level:             level,
		Development:       true,
		DisableStacktrace: true,
		Encoding:          "console",
		OutputPaths:       outputPaths,
		ErrorOutputPaths:  []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "T",
			LevelKey:       "L",
			NameKey:        "N",
			CallerKey:      "C",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "M",
			StacktraceKey:  "S",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    levelEncoder,
			EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05"),
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
	}

	z, err := zapCfg.Build()
	if err != nil {
		return nil, errors.Wrap(err, "failed to zapCfg.Build")
	}

	// Send SIGINT on fatal calls
	z = z.WithOptions(
		zap.OnFatal(doNothingOnFatal),
		zap.Hooks(func(e zapcore.Entry) error {
			if e.Level == zap.FatalLevel {
				err := syscall.Kill(syscall.Getpid(), syscall.SIGINT)
				if err != nil {
					z.With(zap.Any("error", err)).Sugar().Error("failed to syscall.SIGINT on FATAL()")
					z.Sync()
					os.Exit(1)
				}
			}
			return nil
		}),
	)

	z = z.WithOptions(zap.AddCallerSkip(1))

	return &Logger{
		zap:   z.Sugar(),
		level: level,
	}, nil
}

// NewNoop returns a noop logger
func NewNoop() *Logger {
	return &Logger{
		zap:   zap.NewNop().Sugar(),
		level: zap.NewAtomicLevel(),
	}
}

// NewWith returns a logger based on the passed zap logger
func NewWith(log *zap.Logger, currentLvl zapcore.Level) *Logger {
	return &Logger{
		zap:   log.Sugar(),
		level: zap.NewAtomicLevelAt(currentLvl),
	}
}

// Zap returns the underlying *zap.SugaredLogger
func (l *Logger) Zap() *zap.SugaredLogger {
	return l.zap
}

func (l *Logger) SetLevel(lvl string) {
	if lvl == "trace" || lvl == "TRACE" {
		// zap doesn't have a trace level. See TODO for more info
		lvl = "debug"
	}

	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(lvl)); err == nil {
		l.level.SetLevel(zapLevel)
	}
}

// WithCallerSkip returns a cloned logger with increased number of skipped callers.
// Skip can be negative
func (l *Logger) WithCallerSkip(skip int) *Logger {
	clone := l.clone()
	clone.zap = clone.zap.Desugar().WithOptions(zap.AddCallerSkip(skip)).Sugar()
	return clone
}

// WithField returns a cloned logger with a new field
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return l.withFields(zap.Any(key, value))
}

// WithError is a shorthand for Logger.WithField("error", err)
func (l *Logger) WithError(err error) *Logger {
	return l.WithField("error", err)
}

// WithField returns a cloned logger with new fields
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	zapFields := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}
	return l.withFields(zapFields...)
}

func (l *Logger) withFields(fields ...zap.Field) *Logger {
	clone := l.clone()
	clone.zap = clone.zap.Desugar().With(fields...).Sugar()
	return clone
}

func (l *Logger) clone() *Logger {
	return &Logger{
		zap:   l.zap,
		level: l.level,
	}
}

// TODO: zap doesn't have a trace level (it can be added in v2). So, use debug level instead.
// See https://github.com/uber-go/zap/issues/680 for more info.

// // Trace is an alias for Debug
// func (l *Logger) Trace(args ...interface{}) { l.zap.Debug(args...) }

// // Tracef is an alias for Debugf
// func (l *Logger) Tracef(format string, args ...interface{}) { l.zap.Debugf(format, args...) }

// // Traceln is an alias for Debugln
// func (l *Logger) Traceln(args ...interface{}) { l.zap.Debug(sprintln(args...)) }

func (l *Logger) Debug(args ...interface{})                 { l.zap.Debug(args...) }
func (l *Logger) Debugf(format string, args ...interface{}) { l.zap.Debugf(format, args...) }
func (l *Logger) Debugln(args ...interface{})               { l.zap.Debug(sprintln(args...)) }

func (l *Logger) Info(args ...interface{})                 { l.zap.Info(args...) }
func (l *Logger) Infof(format string, args ...interface{}) { l.zap.Infof(format, args...) }
func (l *Logger) Infoln(args ...interface{})               { l.zap.Info(sprintln(args...)) }

func (l *Logger) Warn(args ...interface{})                 { l.zap.Warn(args...) }
func (l *Logger) Warnf(format string, args ...interface{}) { l.zap.Warnf(format, args...) }
func (l *Logger) Warnln(args ...interface{})               { l.zap.Warn(sprintln(args...)) }

func (l *Logger) Warning(args ...interface{})                 { l.zap.Warn(args...) }
func (l *Logger) Warningf(format string, args ...interface{}) { l.zap.Warnf(format, args...) }
func (l *Logger) Warningln(args ...interface{})               { l.zap.Warn(sprintln(args...)) }

func (l *Logger) Error(args ...interface{})                 { l.zap.Error(args...) }
func (l *Logger) Errorf(format string, args ...interface{}) { l.zap.Errorf(format, args...) }
func (l *Logger) Errorln(args ...interface{})               { l.zap.Error(sprintln(args...)) }

func (l *Logger) Fatal(args ...interface{})                 { l.zap.Fatal(args...) }
func (l *Logger) Fatalf(format string, args ...interface{}) { l.zap.Fatalf(format, args...) }
func (l *Logger) Fatalln(args ...interface{})               { l.zap.Fatal(sprintln(args...)) }

func (l *Logger) Panic(args ...interface{})                 { l.zap.Panic(args...) }
func (l *Logger) Panicf(format string, args ...interface{}) { l.zap.Panicf(format, args...) }
func (l *Logger) Panicln(args ...interface{})               { l.zap.Panic(sprintln(args...)) }

func (l *Logger) Print(args ...interface{})                 { l.zap.Info(args...) }
func (l *Logger) Printf(format string, args ...interface{}) { l.zap.Infof(format, args...) }
func (l *Logger) Println(args ...interface{})               { l.zap.Info(sprintln(args...)) }

// Sync flushes any buffered log entries
func (l *Logger) Sync() error { return l.zap.Sync() }

// sprintln returns the result of fmt.Sprintln without the trailing \n
func sprintln(args ...interface{}) string {
	msg := fmt.Sprintln(args...)
	return msg[:len(msg)-1]
}

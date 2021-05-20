package log

import (
	"io"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/term"
)

// Level is logger level
type Level zapcore.Level

const (
	// DebugLevel logs are typically voluminous, and are usually disabled in
	// production.
	DebugLevel Level = iota - 1
	// InfoLevel is the default logging priority.
	InfoLevel
	// WarnLevel logs are more important than Info, but don't need individual
	// human review.
	WarnLevel
	// ErrorLevel logs are high-priority. If an application is running smoothly,
	// it shouldn't generate any error-level logs.
	ErrorLevel
	// DPanicLevel logs are particularly important errors. In development the
	// logger panics after writing the message.
	DPanicLevel
	// PanicLevel logs a message, then panics.
	PanicLevel
	// FatalLevel logs a message, then calls os.Exit(1).
	FatalLevel
)

const (
	// JSON format prints logs as JSON
	JSON Format = iota - 1
	// TEXT format prints logs as human readable text
	TEXT
	// AUTO determines if log should be in TEXT or JSON format based on terminal type
	AUTO
)

type (
	// Format indicates log message output
	Format int8

	// An Option configures a Logger.
	Option interface {
		apply(*logger)
	}
	optionFunc func(*logger)

	// Logger for the projec
	Logger interface {
		NamedLogger(name string) Logger
		Info(args ...interface{})
		Warn(args ...interface{})
		Debug(args ...interface{})
		Error(args ...interface{})
		Fatal(args ...interface{})
		Panic(args ...interface{})

		Infof(template string, args ...interface{})
		Warnf(template string, args ...interface{})
		Debugf(template string, args ...interface{})
		Errorf(template string, args ...interface{})
		Fatalf(template string, args ...interface{})
		Panicf(template string, args ...interface{})

		Infow(msg string, keysAndValues ...interface{})
		Warnw(msg string, keysAndValues ...interface{})
		Debugw(msg string, keysAndValues ...interface{})
		Errorw(msg string, keysAndValues ...interface{})
		Fatalw(msg string, keysAndValues ...interface{})
		Panicw(msg string, keysAndValues ...interface{})

		Flush()
	}

	logger struct {
		wrappedLogger *zap.SugaredLogger
		level         Level
		name          string
		tags          map[string]string
		format        Format
		out           io.Writer

		rollbarToken    string
		rollbarMinLevel Level
	}
)

func (f optionFunc) apply(r *logger) {
	f(r)
}

// NewNop returns a no-op Logger.
func NewNop() Logger {
	return &logger{wrappedLogger: zap.NewNop().Sugar()}
}

// New returns a new Logger
func New(options ...Option) Logger {

	logger := &logger{
		format:          AUTO,
		level:           InfoLevel,
		out:             os.Stdout,
		rollbarMinLevel: ErrorLevel,
	}

	for _, opt := range options {
		opt.apply(logger)
	}
	if logger.name == "" {
		logger.name = "root"
	}
	logger.initWrappedLogger()

	return logger
}

func (l *logger) initWrappedLogger() {
	atom := zap.NewAtomicLevel()
	atom.SetLevel(zapcore.Level(l.level))
	logOut := zapcore.Lock(os.Stdout) // could be a file or a remote sync

	zcores := []zapcore.Core{
		zapcore.NewCore(
			l.getEncoder(),
			logOut,
			atom,
		),
	}

	if l.rollbarToken != "" {
		// Tee off logs to rollbar
		zcores = append(zcores, newRollbarCore(l.rollbarToken, l.getEvironment(), l.getVersion(), l.rollbarMinLevel))
	}
	wl := zap.New(zapcore.NewTee(zcores...), zap.AddCaller(), zap.AddCallerSkip(1), zap.AddStacktrace(zap.ErrorLevel))
	l.wrappedLogger = wl.Named(l.name).Sugar()
}

func (l *logger) getEncoder() (enc zapcore.Encoder) {

	encoderCfg := zap.NewProductionEncoderConfig()
	switch l.format {
	case AUTO:
		if l.isTerminal() {
			encoderCfg.TimeKey = ""
			enc = zapcore.NewConsoleEncoder(encoderCfg)
		} else {
			enc = zapcore.NewJSONEncoder(encoderCfg)
		}
	case TEXT:
		encoderCfg.TimeKey = ""
		enc = zapcore.NewConsoleEncoder(encoderCfg)
	default:
		enc = zapcore.NewJSONEncoder(encoderCfg)
	}

	return
}

func (l *logger) isTerminal() bool {
	switch v := l.out.(type) {
	case *os.File:
		return term.IsTerminal(int(v.Fd()))
	default:
		return false
	}
}

// NamedLogger returns a named sub logger
func (l *logger) NamedLogger(name string) Logger {
	return &logger{name: name, wrappedLogger: l.wrappedLogger.Named(name)}
}

//Info - wrapper to underlying logger
func (l *logger) Info(args ...interface{}) {
	l.wrappedLogger.Info(args...)
}

//Warn - wrapper to underlying logger
func (l *logger) Warn(args ...interface{}) {
	l.wrappedLogger.Warn(args...)
}

//Debug - wrapper to underlying logger
func (l *logger) Debug(args ...interface{}) {
	l.wrappedLogger.Debug(args...)
}

//Error - wrapper to underlying logger
func (l *logger) Error(args ...interface{}) {
	l.wrappedLogger.Error(args...)
}

//Fatal - wrapper to underlying logger
func (l *logger) Fatal(args ...interface{}) {
	l.wrappedLogger.Fatal(args...)
}

// Panic - log info message with template
func (l *logger) Panic(args ...interface{}) {
	format, args := args[0], args[1:]
	l.wrappedLogger.Panicf(format.(string), args...)
}

// Infof - log info message with template
func (l *logger) Infof(template string, args ...interface{}) {
	l.wrappedLogger.Infof(template, args...)
}

// Warnf - log info message with template
func (l *logger) Warnf(template string, args ...interface{}) {
	l.wrappedLogger.Warnf(template, args...)
}

// Debugf - log info message with template
func (l *logger) Debugf(template string, args ...interface{}) {
	l.wrappedLogger.Debugf(template, args...)
}

// Errorf - log info message with template
func (l *logger) Errorf(template string, args ...interface{}) {
	l.wrappedLogger.Errorf(template, args...)
}

// Fatalf - log info message with template
func (l *logger) Fatalf(template string, args ...interface{}) {
	l.wrappedLogger.Fatalf(template, args...)
}

// Panicf - log info message with template
func (l *logger) Panicf(template string, args ...interface{}) {
	l.wrappedLogger.Panicf(template, args...)
}

// Infow logs a message with some additional context
func (l *logger) Infow(msg string, keysAndValues ...interface{}) {
	l.wrappedLogger.Infow(msg, keysAndValues...)
}

// Warnw logs a message with some additional context
func (l *logger) Warnw(msg string, keysAndValues ...interface{}) {
	l.wrappedLogger.Warnw(msg, keysAndValues...)
}

// Debugw logs a message with some additional context
func (l *logger) Debugw(msg string, keysAndValues ...interface{}) {
	l.wrappedLogger.Debugw(msg, keysAndValues...)
}

// Errorw logs a message with some additional context
func (l *logger) Errorw(msg string, keysAndValues ...interface{}) {
	l.wrappedLogger.Errorw(msg, keysAndValues...)
}

// Fatalw logs a message with some additional context
func (l *logger) Fatalw(msg string, keysAndValues ...interface{}) {
	l.wrappedLogger.Fatalw(msg, keysAndValues...)
}

// Panicw logs a message with some additional context
func (l *logger) Panicw(msg string, keysAndValues ...interface{}) {
	l.wrappedLogger.Panicw(msg, keysAndValues...)
}

// Flush any buffered log entries.
func (l *logger) Flush() {
	l.wrappedLogger.Sync()
}

func (l *logger) getVersion() string {
	return l.tags["version"]
}

func (l *logger) getEvironment() string {
	return l.tags["environment"]
}

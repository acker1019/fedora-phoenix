package logging

import (
	"io"

	"github.com/sirupsen/logrus"
)

// Log defines the interface compatible with logrus.FieldLogger
// This allows for dependency injection and easier testing
type Log interface {
	// Standard logging methods
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Debugln(args ...interface{})

	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Infoln(args ...interface{})

	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	Warnln(args ...interface{})

	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Errorln(args ...interface{})

	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Fatalln(args ...interface{})

	Panic(args ...interface{})
	Panicf(format string, args ...interface{})
	Panicln(args ...interface{})

	// Field methods
	WithField(key string, value interface{}) *logrus.Entry
	WithFields(fields logrus.Fields) *logrus.Entry
	WithError(err error) *logrus.Entry

	// Writer method
	Writer() *io.PipeWriter
}

// Global logger instance
var log Log

// init initializes the global logger with a logrus instance
func init() {
	logger := logrus.New()

	// Configure logrus (can be customized later)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	logger.SetLevel(logrus.InfoLevel)

	// Inject the logrus instance into our global logger
	log = logger
}

// Package-level public functions wrapping the global logger instance

// Debug logs a message at level Debug
func Debug(args ...interface{}) {
	log.Debug(args...)
}

// Debugf logs a formatted message at level Debug
func Debugf(format string, args ...interface{}) {
	log.Debugf(format, args...)
}

// Debugln logs a message at level Debug with a newline
func Debugln(args ...interface{}) {
	log.Debugln(args...)
}

// Info logs a message at level Info
func Info(args ...interface{}) {
	log.Info(args...)
}

// Infof logs a formatted message at level Info
func Infof(format string, args ...interface{}) {
	log.Infof(format, args...)
}

// Infoln logs a message at level Info with a newline
func Infoln(args ...interface{}) {
	log.Infoln(args...)
}

// Warn logs a message at level Warn
func Warn(args ...interface{}) {
	log.Warn(args...)
}

// Warnf logs a formatted message at level Warn
func Warnf(format string, args ...interface{}) {
	log.Warnf(format, args...)
}

// Warnln logs a message at level Warn with a newline
func Warnln(args ...interface{}) {
	log.Warnln(args...)
}

// Error logs a message at level Error
func Error(args ...interface{}) {
	log.Error(args...)
}

// Errorf logs a formatted message at level Error
func Errorf(format string, args ...interface{}) {
	log.Errorf(format, args...)
}

// Errorln logs a message at level Error with a newline
func Errorln(args ...interface{}) {
	log.Errorln(args...)
}

// Fatal logs a message at level Fatal then calls os.Exit(1)
func Fatal(args ...interface{}) {
	log.Fatal(args...)
}

// Fatalf logs a formatted message at level Fatal then calls os.Exit(1)
func Fatalf(format string, args ...interface{}) {
	log.Fatalf(format, args...)
}

// Fatalln logs a message at level Fatal with a newline then calls os.Exit(1)
func Fatalln(args ...interface{}) {
	log.Fatalln(args...)
}

// Panic logs a message at level Panic then panics
func Panic(args ...interface{}) {
	log.Panic(args...)
}

// Panicf logs a formatted message at level Panic then panics
func Panicf(format string, args ...interface{}) {
	log.Panicf(format, args...)
}

// Panicln logs a message at level Panic with a newline then panics
func Panicln(args ...interface{}) {
	log.Panicln(args...)
}

// WithSource adds a "source" field to the log entry
func WithSource(unit string) *logrus.Entry {
	return log.WithField("source", unit)
}

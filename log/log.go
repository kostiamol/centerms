package log

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type (
	// Logger is a contract for the logger.
	Logger interface {
		Debugf(format string, args ...interface{})
		Infof(format string, args ...interface{})
		Info(msg string)
		Errorf(format string, args ...interface{})
		Error(msg string)
		Fatalf(format string, args ...interface{})
		Fatal(msg string)
		With(args ...interface{}) Logger
		Flush() error
	}

	Field = zapcore.Field

	zapLogger struct {
		sugared *zap.SugaredLogger
		fast    *zap.Logger
	}
)

// Logging can be improved by introducing Field type.

// InitLog initializes and returns a new instance of a logger.
func New(appID, logLevel string) *zapLogger { //nolint
	atom := zap.NewAtomicLevel()

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	log := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.Lock(os.Stdout),
		atom,
	))

	atom.SetLevel(zap.InfoLevel)
	if logLevel != "" {
		err := atom.UnmarshalText([]byte(strings.ToLower(logLevel)))
		if err != nil {
			log.Error("invalid log level")
		}
	}

	l := log.With(zap.String("svc", appID))

	return &zapLogger{
		fast:    l,
		sugared: l.Sugar(),
	}
}

// With adds a variadic number of fields to the logging context. Use the func as it is now only with
// sugared loger funcs in order to keep the fields otherwise fast loger with neglect them.
func (l *zapLogger) With(args ...interface{}) Logger {
	return &zapLogger{
		fast:    l.fast,
		sugared: l.sugared.With(args...),
	}
}

// Flush .
func (l *zapLogger) Flush() error {
	if err := l.fast.Sync(); err != nil {
		return err
	}
	if err := l.sugared.Sync(); err != nil {
		return err
	}
	return nil
}

// Debugf .
func (l *zapLogger) Debugf(format string, args ...interface{}) {
	l.sugared.Debugf(format, args...)
}

// Infof .
func (l *zapLogger) Infof(format string, args ...interface{}) {
	l.sugared.Infof(format, args...)
}

// Info .
func (l *zapLogger) Info(msg string) {
	l.fast.Info(msg)
}

// Errorf .
func (l *zapLogger) Errorf(format string, args ...interface{}) {
	l.sugared.Errorf(format, args...)
}

// Error .
func (l *zapLogger) Error(msg string) {
	l.fast.Error(msg)
}

// Fatalf .
func (l *zapLogger) Fatalf(format string, args ...interface{}) {
	l.sugared.Fatalf(format, args...)
}

// Fatal .
func (l *zapLogger) Fatal(msg string) {
	l.fast.Fatal(msg)
}

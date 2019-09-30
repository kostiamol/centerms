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
		Info(args ...interface{})
		Errorf(format string, args ...interface{})
		Error(args ...interface{})
		Fatalf(format string, args ...interface{})
		Fatal(args ...interface{})
		With(args ...interface{}) Logger
		Flush() error
	}

	zapLogger struct {
		log *zap.SugaredLogger
	}
)

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

	return &zapLogger{log: log.Sugar().With("svc", appID)}
}

// Debugf .
func (l *zapLogger) Debugf(format string, args ...interface{}) {
	l.log.Debugf(format, args...)
}

// Infof .
func (l *zapLogger) Infof(format string, args ...interface{}) {
	l.log.Infof(format, args...)
}

// Info .
func (l *zapLogger) Info(args ...interface{}) {
	l.log.Info(args...)
}

// Errorf .
func (l *zapLogger) Errorf(format string, args ...interface{}) {
	l.log.Errorf(format, args...)
}

// Error .
func (l *zapLogger) Error(args ...interface{}) {
	l.log.Error(args...)
}

// Fatalf .
func (l *zapLogger) Fatalf(format string, args ...interface{}) {
	l.log.Fatalf(format, args...)
}

// Fatal .
func (l *zapLogger) Fatal(args ...interface{}) {
	l.log.Fatal(args...)
}

// With .
func (l *zapLogger) With(args ...interface{}) Logger {
	return &zapLogger{l.log.With(args...)}
}

// Flush .
func (l *zapLogger) Flush() error {
	return l.log.Sync()
}

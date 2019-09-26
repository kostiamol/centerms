package log

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// InitLog initializes and returns a new instance of a logger.
func NewLog(appID, logLevel string) *zap.SugaredLogger {
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

	return log.Sugar().With("svc", appID)
}

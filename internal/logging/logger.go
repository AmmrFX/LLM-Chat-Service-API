package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is the global logger instance
var Logger *zap.Logger

// ------------------------------------------------------------------------------------------------------
func Init() error {
	config := zap.NewProductionConfig()
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.MessageKey = "message"
	config.EncoderConfig.LevelKey = "level"
	config.EncoderConfig.CallerKey = "caller"

	var err error
	Logger, err = config.Build()
	if err != nil {
		return err
	}

	return nil
}

// ------------------------------------------------------------------------------------------------------
func Sync() {
	if Logger != nil {
		_ = Logger.Sync()
	}
}

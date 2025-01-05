package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	Development bool
	Level       string // debug, info, warn, error
}

// NewLogger returns a SugaredLogger instead of the standard Logger
func NewLogger(cfg Config) (*zap.SugaredLogger, error) {
	var zapCfg zap.Config

	if cfg.Development {
		zapCfg = zap.NewDevelopmentConfig()
		zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		zapCfg = zap.NewProductionConfig()
		zapCfg.EncoderConfig.TimeKey = "timestamp"
		zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	zapCfg.EncoderConfig.CallerKey = ""
	// zapCfg.EncoderConfig.MessageKey = "msg" // Change "msg" to "message" (or "" to remove it)
	// zapCfg.EncoderConfig.NameKey = "n"      // Remove the "logger" field

	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}
	zapCfg.Level = zap.NewAtomicLevelAt(level)

	logger, err := zapCfg.Build()
	if err != nil {
		return nil, err
	}

	return logger.Sugar(), nil
}

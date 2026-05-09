package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var global *zap.Logger

func Init(level, format string) error {
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
		zapLevel = zapcore.InfoLevel
	}

	var cfg zap.Config
	if format == "json" {
		cfg = zap.NewProductionConfig()
	} else {
		cfg = zap.NewDevelopmentConfig()
	}
	cfg.Level = zap.NewAtomicLevelAt(zapLevel)

	l, err := cfg.Build(zap.AddCallerSkip(0))
	if err != nil {
		return err
	}

	global = l
	return nil
}

func Get() *zap.Logger {
	if global == nil {
		global, _ = zap.NewProduction()
	}
	return global
}

func Sync() {
	if global != nil {
		_ = global.Sync()
	}
}

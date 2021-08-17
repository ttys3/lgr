package lgr

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func getZapLevel(level string) zapcore.Level {
	zapLevel := zap.InfoLevel
	switch level {
	case "debug":
		zapLevel = zap.DebugLevel
	case "info":
		zapLevel = zap.InfoLevel
	case "warn":
		zapLevel = zap.WarnLevel
	case "error":
		zapLevel = zap.ErrorLevel
	case "fatal":
		zapLevel = zap.FatalLevel
	}
	return zapLevel
}

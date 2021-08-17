package lgr

import (
	"time"

	"go.uber.org/zap/zapcore"
)

const DefaultTimeLayout = "2006-01-02T15:04:05.999Z07:00"

func ZapTimeEncoder(layout string) func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	// RFC3339     = "2006-01-02T15:04:05Z07:00"
	// RFC3339Nano = "2006-01-02T15:04:05.999999999Z07:00"
	return func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format(layout))
	}
}

func DefaultTimeEncoder() func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	return ZapTimeEncoder(DefaultTimeLayout)
}

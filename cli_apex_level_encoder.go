package lgr

import (
	"github.com/fatih/color"
	"go.uber.org/zap/zapcore"
)

// LevelEncoder

var bold = color.New(color.Bold)

// Colors mapping.
var Colors = [...]*color.Color{
	zapcore.DebugLevel + 1: color.New(color.FgWhite),
	zapcore.InfoLevel + 1:  color.New(color.FgBlue),
	zapcore.WarnLevel + 1:  color.New(color.FgYellow),
	zapcore.ErrorLevel + 1: color.New(color.FgRed),
	zapcore.FatalLevel + 1: color.New(color.FgRed),
}

// Strings mapping.
var Strings = [...]string{
	zapcore.DebugLevel + 1: "•",
	zapcore.InfoLevel + 1:  "•",
	zapcore.WarnLevel + 1:  "•",
	zapcore.ErrorLevel + 1: "⨯",
	zapcore.FatalLevel + 1: "⨯",
}

func CliLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	padding := 3
	color := Colors[level+1]
	lvlIcon := Strings[level+1]
	enc.AppendString(color.Sprintf("%s", bold.Sprintf("%*s", padding+1, lvlIcon)))
}

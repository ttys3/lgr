package lgr

import "io"

type Option func(l *LogImpl)

func WithEncoding(encoding string) Option {
	return func(l *LogImpl) { l.Encoding = encoding }
}

func WithLevel(level string) Option {
	return func(l *LogImpl) { l.Level = level }
}

func WithColorLevel(enable bool) Option {
	return func(l *LogImpl) { l.ColorLevel = enable }
}

func WithTimeKey(tk string) Option {
	return func(l *LogImpl) { l.TimeKey = tk }
}

func WithDatetimeLayout(layout string) Option {
	return func(l *LogImpl) { l.DatetimeLayout = layout }
}

func WithDisableStacktrace(disableStacktrace bool) Option {
	return func(l *LogImpl) { l.DisableStacktrace = disableStacktrace }
}

func WithName(loggerName string) Option {
	return func(l *LogImpl) { l.Name = loggerName }
}

func WithInitialFields(kv ...string) Option {
	if len(kv)%2 != 0 {
		panic("ContextFields must in key, value pairs")
	}
	return func(l *LogImpl) {
		l.InitialFields = kv
	}
}

func WithOutputPaths(outputPaths ...string) Option {
	return func(l *LogImpl) {
		l.OutputPaths = outputPaths
	}
}

func WithErrorOutputPaths(errOutputPaths ...string) Option {
	return func(l *LogImpl) {
		l.ErrorOutputPaths = errOutputPaths
	}
}

func WithCustomSink(writer io.Writer) Option {
	return func(l *LogImpl) { l.CustomSink = writer }
}

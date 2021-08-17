package lgr

import (
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	once           sync.Once
	_globalLog     *LogImpl
	_globalLogLock sync.RWMutex
)

// type assertion ensure implementation
// do not return interface, this is bad for caller
// see https://github.com/golang/go/wiki/CodeReviewComments#interfaces
var _ Logger = (*LogImpl)(nil)

// global logger, call via S()
func S() *LogImpl {
	once.Do(func() {
		_globalLogLock.RLock()
		oldLogger := _globalLog
		_globalLogLock.RUnlock()

		// only initialize default logger if it is not set
		if oldLogger == nil {
			l := NewDefault()
			ReplaceGlobal(l)
		}
	})
	return _globalLog
}

type LogImpl struct {
	s *zap.SugaredLogger
	Config
}

type Config struct {
	DisableStacktrace bool
	Name              string // Named adds a sub-scope to the logger's name. See Logger.Named for details.
	Encoding          string
	Level             string
	DatetimeLayout    string
	ContextFields     []string // key, value  adds structured context
	OutputPaths       []string
	ErrorOutputPaths  []string // for zap logging self error
}

func NewDefault() *LogImpl {
	l := defaultCfg()
	return l.build()
}

func NewLogger(options ...Option) *LogImpl {
	l := defaultCfg()
	// apply otions

	for _, option := range options {
		option(l)
	}
	return l.build()
}

type Logger interface {
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
	Fatal(msg string, keysAndValues ...interface{})
	Sync() error
	Named(name string) *LogImpl
	With(keysAndValues ...interface{}) *LogImpl
}

type Option func(l *LogImpl)

func WithEncoding(encoding string) Option {
	return func(l *LogImpl) { l.Encoding = encoding }
}

func WithLevel(level string) Option {
	return func(l *LogImpl) { l.Level = level }
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

func WithContextFields(kv ...string) Option {
	if len(kv)%2 != 0 {
		panic("ContextFields must in key, value pairs")
	}
	return func(l *LogImpl) {
		l.ContextFields = kv
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

func defaultCfg() *LogImpl {
	c := Config{
		Encoding:          "json",
		Level:             "info",
		DatetimeLayout:    DefaultTimeLayout,
		DisableStacktrace: true,
		Name:              "",
		ContextFields:     []string{},
		// zap.newFileSink has special handle for `stdout` and `stderr`
		// if you want a dummy sink, use `/dev/null`
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}

	l := &LogImpl{Config: c}
	return l
}

func (l *LogImpl) clone() *LogImpl {
	cloned := &LogImpl{
		s:      l.s,
		Config: l.Config,
	}
	return cloned
}

func ReplaceGlobal(newlgr *LogImpl) *LogImpl {
	_globalLogLock.Lock()
	_globalLog = newlgr
	_globalLogLock.Unlock()

	return _globalLog
}

func (l *LogImpl) build() *LogImpl {
	zapcfg := zap.NewProductionConfig()

	zapcfg.DisableStacktrace = l.DisableStacktrace
	zapcfg.EncoderConfig.EncodeTime = ZapTimeEncoder(l.DatetimeLayout)

	if l.Encoding != "" {
		zapcfg.Encoding = l.Encoding
	}
	if zapcfg.Encoding == "console" {
		zapcfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	if l.Level != "" {
		zapcfg.Level = zap.NewAtomicLevelAt(getZapLevel(l.Level))
	}

	if len(l.OutputPaths) > 0 {
		zapcfg.OutputPaths = l.OutputPaths
	}

	if len(l.ErrorOutputPaths) > 0 {
		zapcfg.ErrorOutputPaths = l.ErrorOutputPaths
	}

	zaplgr, err := zapcfg.Build()
	if err != nil {
		panic(err)
	}

	// skip ourself from caller stack
	zaplgr = zaplgr.WithOptions(zap.AddCallerSkip(1))

	if l.Name != "" {
		zaplgr = zaplgr.Named(l.Name)
	}

	if len(l.ContextFields)%2 == 0 && len(l.ContextFields) > 0 {
		fields := make([]zap.Field, 0, len(l.ContextFields)/2)
		for i := 0; i < len(l.ContextFields); i += 2 {
			fields = append(fields, zap.String(l.ContextFields[i], l.ContextFields[i+1]))
		}
		zaplgr = zaplgr.With(fields...)
	}

	zapsugar := zaplgr.Sugar()
	l.s = zapsugar
	return l
}

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

func (l *LogImpl) Debug(msg string, keysAndValues ...interface{}) {
	l.s.Debugw(msg, keysAndValues...)
}

func (l *LogImpl) Info(msg string, keysAndValues ...interface{}) {
	l.s.Infow(msg, keysAndValues...)
}

func (l *LogImpl) Warn(msg string, keysAndValues ...interface{}) {
	l.s.Warnw(msg, keysAndValues...)
}

func (l *LogImpl) Error(msg string, keysAndValues ...interface{}) {
	l.s.Errorw(msg, keysAndValues...)
}

func (l *LogImpl) Fatal(msg string, keysAndValues ...interface{}) {
	l.s.Fatalw(msg, keysAndValues...)
}

func (l *LogImpl) Sync() error {
	return l.s.Sync()
}

func (l *LogImpl) Named(name string) *LogImpl {
	newLgr := l.clone()
	newLgr.s = l.s.Named(name)
	return newLgr
}

func (l *LogImpl) With(keysAndValues ...interface{}) *LogImpl {
	newLgr := l.clone()
	newLgr.s = l.s.With(keysAndValues...)
	return newLgr
}

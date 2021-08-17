package lgr

import (
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	once       sync.Once
	_globalLog *logImpl
)

// type assertion ensure implementation
// do not return interface, this is bad for caller
// see https://github.com/golang/go/wiki/CodeReviewComments#interfaces
var _ Logger = (*logImpl)(nil)

// global logger, call via S()
func S() *logImpl {
	once.Do(func() {
		l := NewDefault()
		ReplaceGlobal(l)
	})
	return _globalLog
}

type logImpl struct {
	s                 *zap.SugaredLogger
	DisableStacktrace bool
	Name              string // Named adds a sub-scope to the logger's name. See Logger.Named for details.
	Encoding          string
	Level             string
	DatetimeLayout    string
	ContextFields     []string // key, value  adds structured context
}

type Logger interface {
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
	Fatal(msg string, keysAndValues ...interface{})
	Sync() error
	Named(name string) *logImpl
	With(keysAndValues ...interface{}) *logImpl
}

type Option func(l *logImpl)

func WithEncoding(encoding string) Option {
	return func(l *logImpl) { l.Encoding = encoding }
}

func WithLevel(level string) Option {
	return func(l *logImpl) { l.Level = level }
}

func WithDatetimeLayout(layout string) Option {
	return func(l *logImpl) { l.DatetimeLayout = layout }
}

func WithDisableStacktrace(disableStacktrace bool) Option {
	return func(l *logImpl) { l.DisableStacktrace = disableStacktrace }
}

func WithName(loggerName string) Option {
	return func(l *logImpl) { l.Name = loggerName }
}

func WithContextFields(kv ...string) Option {
	if len(kv)%2 != 0 {
		panic("ContextFields must in key, value pairs")
	}
	return func(l *logImpl) {
		l.ContextFields = kv
	}
}

func defaultCfg() *logImpl {
	l := &logImpl{
		Encoding:          "json",
		Level:             "info",
		DatetimeLayout:    DefaultTimeLayout,
		DisableStacktrace: true,
		Name:              "",
		ContextFields:     []string{},
	}
	return l
}

func NewDefault() *logImpl {
	l := defaultCfg()
	return l.build()
}

func ReplaceGlobal(newlgr *logImpl) *logImpl {
	_globalLog = newlgr
	return _globalLog
}

func NewLogger(options ...Option) *logImpl {
	l := defaultCfg()
	// apply otions

	for _, option := range options {
		option(l)
	}
	return l.build()
}

func (l *logImpl) build() *logImpl {
	zapcfg := zap.NewProductionConfig()
	zapcfg.Encoding = l.Encoding
	zapcfg.Level = zap.NewAtomicLevelAt(getZapLevel(l.Level))
	zapcfg.DisableStacktrace = l.DisableStacktrace
	zapcfg.EncoderConfig.EncodeTime = ZapTimeEncoder(l.DatetimeLayout)

	if zapcfg.Encoding == "console" {
		zapcfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	zaplgr, err := zapcfg.Build()
	if err != nil {
		panic(err)
	}
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

	_globalLog = &logImpl{s: zapsugar}
	return _globalLog
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

func (l *logImpl) Debug(msg string, keysAndValues ...interface{}) {
	l.s.Debugw(msg, keysAndValues...)
}

func (l *logImpl) Info(msg string, keysAndValues ...interface{}) {
	l.s.Infow(msg, keysAndValues...)
}

func (l *logImpl) Warn(msg string, keysAndValues ...interface{}) {
	l.s.Warnw(msg, keysAndValues...)
}

func (l *logImpl) Error(msg string, keysAndValues ...interface{}) {
	l.s.Errorw(msg, keysAndValues...)
}

func (l *logImpl) Fatal(msg string, keysAndValues ...interface{}) {
	l.s.Fatalw(msg, keysAndValues...)
}

func (l *logImpl) Sync() error {
	return l.s.Sync()
}

func (l *logImpl) Named(name string) *logImpl {
	return &logImpl{s: l.s.Named(name)}
}

func (l *logImpl) With(keysAndValues ...interface{}) *logImpl {
	return &logImpl{s: l.s.With(keysAndValues...)}
}

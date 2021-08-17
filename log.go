package lgr

import (
	"io"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	EncodingConsole = "console"
	EncodingJSON    = "json"
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

type LogImpl struct {
	s *zap.SugaredLogger
	Config
}

type Config struct {
	DisableStacktrace bool
	DisableCaller     bool
	Development       bool
	Sampling          bool
	ColorLevel        bool   // this is only for console encoding
	Name              string // Named adds a sub-scope to the logger's name. See Logger.Named for details.
	Encoding          string
	Level             string
	TimeKey           string
	DatetimeLayout    string
	ContextFields     []string // key, value paris to add structured context
	OutputPaths       []string
	ErrorOutputPaths  []string  // for zap logging self error
	CustomSink        io.Writer // this will override OutputPaths config
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

func WithCustomSink(writer io.Writer) Option {
	return func(l *LogImpl) { l.CustomSink = writer }
}

func defaultCfg() *LogImpl {
	c := Config{
		DisableStacktrace: true,
		DisableCaller:     false,
		Development:       false,
		Sampling:          false,
		ColorLevel:        true,
		Name:              "",
		Encoding:          EncodingJSON,
		Level:             "info",
		TimeKey:           "ts",
		DatetimeLayout:    DefaultTimeLayout,
		ContextFields:     []string{},
		OutputPaths:       []string{"stderr"},
		ErrorOutputPaths:  []string{"stderr"},
		CustomSink:        nil,
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

func (cfg *Config) openSinks() (zapcore.WriteSyncer, zapcore.WriteSyncer, error) {
	sink, closeOut, err := zap.Open(cfg.OutputPaths...)
	if err != nil {
		return nil, nil, err
	}
	errSink, _, err := zap.Open(cfg.ErrorOutputPaths...)
	if err != nil {
		closeOut()
		return nil, nil, err
	}
	return sink, errSink, nil
}

func (cfg *Config) buildOptions(errSink zapcore.WriteSyncer, scfg *zap.SamplingConfig) []zap.Option {
	opts := []zap.Option{zap.ErrorOutput(errSink)}

	if cfg.Development {
		opts = append(opts, zap.Development())
	}

	if !cfg.DisableCaller {
		opts = append(opts, zap.AddCaller())
	}

	stackLevel := zap.ErrorLevel
	if cfg.Development {
		stackLevel = zap.WarnLevel
	}
	if !cfg.DisableStacktrace {
		opts = append(opts, zap.AddStacktrace(stackLevel))
	}

	if scfg != nil {
		opts = append(opts, zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return zapcore.NewSamplerWithOptions(
				core,
				time.Second,
				scfg.Initial,
				scfg.Thereafter,
			)
		}))
	}

	return opts
}

func (l *LogImpl) build() *LogImpl {
	if l.Encoding != EncodingConsole && l.Encoding != EncodingJSON {
		panic("invalid encoding config")
	}

	level := zap.NewAtomicLevelAt(zap.InfoLevel)
	if l.Level != "" {
		level = zap.NewAtomicLevelAt(getZapLevel(l.Level))
	}

	// https://github.com/uber-go/zap/blob/master/FAQ.md#why-sample-application-logs
	// https://github.com/uber-go/zap/blob/master/FAQ.md#why-are-some-of-my-logs-missing
	sampling := &zap.SamplingConfig{
		Initial:    100,
		Thereafter: 100,
	}

	encoderConfig := zap.NewProductionEncoderConfig()

	// if cfg.Level == (zap.AtomicLevel{}) {
	// panic("missing Level")
	// }

	encoderConfig.EncodeTime = ZapTimeEncoder(l.DatetimeLayout)
	if l.Encoding == "console" {
		if l.ColorLevel {
			encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		}
	}

	// allow set to empty to disable default ts field
	encoderConfig.TimeKey = l.TimeKey

	// begin build
	// zaplgr, err := cfg.Build()
	// if err != nil {
	// 	panic(err)
	// }

	// custom build
	var enc zapcore.Encoder
	switch l.Encoding {
	case EncodingConsole:
		enc = zapcore.NewConsoleEncoder(encoderConfig)
	case EncodingJSON:
		enc = zapcore.NewJSONEncoder(encoderConfig)
	}

	var sink zapcore.WriteSyncer
	var errSink zapcore.WriteSyncer

	// using custom sink if specific
	if l.CustomSink != nil {
		sink = zapcore.AddSync(l.CustomSink)
		errSink = zapcore.AddSync(os.Stderr)
	} else {
		// otherwise using the config paths
		var err error
		sink, errSink, err = l.Config.openSinks()
		if err != nil {
			panic(err)
		}
	}

	zaplgr := zap.New(
		zapcore.NewCore(enc, sink, level),
		l.Config.buildOptions(errSink, sampling)...,
	)
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

// With return a new LogImpl with context fields
func (l *LogImpl) With(keysAndValues ...interface{}) *LogImpl {
	newLgr := l.clone()
	newLgr.s = l.s.With(keysAndValues...)
	return newLgr
}

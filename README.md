# lgr -- simple structured logging wrapper based on zap


## 1. Usage

## 1.1 Supported Methods

```golang
Debug(msg string, keysAndValues ...interface{})
Info(msg string, keysAndValues ...interface{})
Warn(msg string, keysAndValues ...interface{})
Error(msg string, keysAndValues ...interface{})
Fatal(msg string, keysAndValues ...interface{})
Sync() error
Named(name string) *LogImpl
With(keysAndValues ...interface{}) *LogImpl
```

### 1.2 using default config

```golang
// with default config: info level, encode to json
log := lgr.NewDefault()
log.Info("this is a info message", "uid", 7, "name", "user001")
log.Warn("this is a warning message", "uid", 8, "name", "Tom")
```

### 1.3 With Options

```golang
// log to console, set log level to debug
log := lgr.NewLogger(lgr.WithEncoding("console"), lgr.WithLevel("debug"))

// with context fields: key,value ...
log = log.With(
"app_name", "hellow-world",
"version", "v1.0.0")

log.Info("this is a info message", "uid", 7, "name", "user001")
```

### 1.4 Using the Package Level Global Logger

```golang
lgr.S().Info("this is a info message", "uid", 7, "name", "user001")
lgr.S().Warn("this is a warning message", "uid", 8, "name", "Tom")
```

### 1.5 Replace the Package Level Global Logger

```golang
// log to console, set log level to debug
log := lgr.NewLogger(lgr.WithEncoding("console"), lgr.WithLevel("debug"))

lgr.ReplaceGlobal(log)

lgr.S().Info("this is a info message", "uid", 7, "name", "user001")
lgr.S().Warn("this is a warning message", "uid", 8, "name", "Tom")
```

## 2. Construct Options

```golang
WithEncoding(encoding string)

WithLevel(level string)

WithColorLevel(enable bool)

WithTimeKey(tk string)

WithDatetimeLayout(layout string)

WithDisableStacktrace(disableStacktrace bool)

WithName(loggerName string)

WithInitialFields(kv ...string)

WithOutputPaths(outputPaths ...string)

WithErrorOutputPaths(errOutputPaths ...string)

WithCustomSink(writer io.Writer)
```


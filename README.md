# lgr -- simple structured logging wrapper based on zap


## usage

### simple

```golang
// with default config: info level, encode to json
log := lgr.NewDefault()
log.Info("this is a info message", "uid", 7, "name", "user001")
log.Warn("this is a warning message", "uid", 8, "name", "Tom")
```

### with options

```golang
// log to console, set log level to debug
log := lgr.NewLogger(lgr.WithEncoding("console"), lgr.WithLevel("debug"))

// with context fields: key,value ...
log = log.With(
"app_name", "hellow-world",
"version", "v1.0.0")

log.Info("this is a info message", "uid", 7, "name", "user001")
```

### using the package level global logger

```golang
lgr.S().Info("this is a info message", "uid", 7, "name", "user001")
lgr.S().Warn("this is a warning message", "uid", 8, "name", "Tom")
```

### replace the package level global logger

```golang
// log to console, set log level to debug
log := lgr.NewLogger(lgr.WithEncoding("console"), lgr.WithLevel("debug"))

lgr.ReplaceGlobal(log)

lgr.S().Info("this is a info message", "uid", 7, "name", "user001")
lgr.S().Warn("this is a warning message", "uid", 8, "name", "Tom")
```

## construct Options

```golang
WithEncoding(encoding string)

WithLevel(level string)

WithDatetimeLayout(layout string)

WithDisableStacktrace(disableStacktrace bool)

WithName(loggerName string)

WithContextFields(kv ...string)
```




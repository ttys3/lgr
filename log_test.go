package lgr

import (
	"bytes"
	"testing"
)

func TestLgr(t *testing.T) {
	t.Logf("-----------------------------------------------------------------")
	log := NewDefault()
	log = log.With("foo", "bar")
	log.Info("this is a info message", "uid", 7, "name", "user001")
	log.Warn("danger, be aware!", "uid", 8, "name", "user001")
}

func TestNewWithCustomSink(t *testing.T) {
	t.Logf("----------------------------------------------------------------- no time key")
	buf := bytes.NewBuffer([]byte(""))
	log := NewLogger(WithName("log001"), WithEncoding("console"), WithCustomSink(buf), WithTimeKey(""), WithColorLevel(false))
	log = log.With("foo", "bar")
	log.Info("this is a info message", "uid", 7, "name", "user001")
	log.Warn("danger, be aware!", "uid", 8, "name", "user001")
	t.Logf("buf=\n%s", buf)
	expect := `info	log001	lgr/log_test.go:21	this is a info message	{"foo": "bar", "uid": 7, "name": "user001"}
warn	log001	lgr/log_test.go:22	danger, be aware!	{"foo": "bar", "uid": 8, "name": "user001"}
`

	if buf.String() != expect {
		t.Fail()
	}
}

func TestNewLevelInfo(t *testing.T) {
	t.Logf("-----------------------------------------------------------------")
	log := NewLogger(WithName("log001"), WithEncoding("console"), WithLevel("info"))
	log = log.With("foo", "bar")

	log.Debug("this is some debug log", "uid", 7, "name", "user001")
	log.Info("this is a info message", "uid", 7, "name", "user001")
	log.Warn("danger, be aware!", "uid", 8, "name", "user001")
	log.Error("something bad happend!", "uid", 1024, "name", "user001")
	log.Error("something really bad happend!", "uid", 1024, "name", "user001")
}

func TestNewLevelDebug(t *testing.T) {
	t.Logf("-----------------------------------------------------------------")
	log := NewLogger(WithName("log001"), WithEncoding("console"), WithLevel("debug"))
	log = log.With("foo", "bar")

	log.Debug("this is some debug log", "uid", 7, "name", "user001")
	log.Info("this is a info message", "uid", 7, "name", "user001")
	log.Warn("danger, be aware!", "uid", 8, "name", "user001")
	log.Error("something bad happend!", "uid", 1024, "name", "user001")
	log.Error("something really bad happend!", "uid", 1024, "name", "user001")
}

func TestNewLogWriteToFile(t *testing.T) {
	t.Logf("-----------------------------------------------------------------")
	log := NewLogger(WithName("log001"), WithEncoding("json"), WithLevel("debug"), WithOutputPaths("/tmp/lgr.log"), WithErrorOutputPaths("/tmp/lgr.err.log"))
	log = log.With("foo", "bar")

	log.Debug("this is some debug log", "uid", 7, "name", "user001")
	log.Info("this is a info message", "uid", 7, "name", "user001")
	log.Warn("danger, be aware!", "uid", 8, "name", "user001")
	log.Error("something bad happend!", "uid", 1024, "name", "user001")
	log.Error("something really bad happend!", "uid", 1024, "name", "user001")
}

func TestNewInvalidEncodingMustPanic(t *testing.T) {
	t.Logf("-----------------------------------------------------------------")
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("oh, no! The code did not panic")
		} else {
			t.Logf("yes! the code paniced as expected, panic=%v", r)
		}
	}()

	log := NewLogger(WithName("log001"), WithEncoding("consolexxx"))
	log = log.With("foo", "bar")
	log.Info("this is a info message", "uid", 7, "name", "user001")
	log.Warn("danger, be aware!", "uid", 8, "name", "user001")
}

func TestNewInvalidKeyValPairsMustPanic(t *testing.T) {
	t.Logf("-----------------------------------------------------------------")
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("oh, no! The code did not panic")
		} else {
			t.Logf("yes! the code paniced as expected, panic=%v", r)
		}
	}()

	NewLogger(WithName("log001"), WithEncoding("console"), WithContextFields("only_key_no_value"))
}

func TestNewInvalidKeyValPairsSugarLoggerWillNotPanic(t *testing.T) {
	t.Logf("-----------------------------------------------------------------")
	log := NewLogger(WithName("log001"), WithEncoding("console"), WithContextFields("version", "1.0.0"))
	// will add an extra error log message: Ignored key without a value
	log.With("only_key").Info("hello world")
}

func TestRegplaceGlobal(t *testing.T) {
	t.Logf("----------------------------------------------------------------- console, level debug")
	log := NewLogger(WithName("log001"), WithEncoding("console"), WithLevel("debug"))
	log = log.With("foo", "bar")

	log.Debug("this is some debug log", "uid", 7, "name", "user001")
	log.Info("this is a info message", "uid", 7, "name", "user001")
	log.Warn("danger, be aware!", "uid", 8, "name", "user001")
	log.Error("something bad happend!", "uid", 1024, "name", "user001")

	t.Logf("----------------------------------------------------------------- json, level info")
	S().Debug("this is some debug log", "uid", 7, "name", "user001")
	S().Info("this is a info message", "uid", 7, "name", "user001")
	S().Warn("danger, be aware!", "uid", 8, "name", "user001")
	S().Error("something bad happend!", "uid", 1024, "name", "user001")

	t.Logf("----------------------------------------------------------------- console, level warn")
	log2 := NewLogger(WithName("log001"), WithEncoding("console"), WithLevel("warn"))
	ReplaceGlobal(log2)
	S().Debug("this is some debug log", "uid", 7, "name", "user001")
	S().Info("this is a info message", "uid", 7, "name", "user001")
	S().Warn("danger, be aware!", "uid", 8, "name", "user001")
	S().Error("something bad happend!", "uid", 1024, "name", "user001")

	t.Logf("----------------------------------------------------------------- console, level warn")
	// should extend warn level
	log3 := S().With("key001", "val001")
	ReplaceGlobal(log3)
	S().Debug("this is some debug log", "uid", 7, "name", "user001")
	S().Info("this is a info message", "uid", 7, "name", "user001")
	S().Warn("danger, be aware!", "uid", 8, "name", "user001")
	S().Error("something bad happend!", "uid", 1024, "name", "user001")
}

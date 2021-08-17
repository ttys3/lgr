package lgr

import (
	"testing"
)

func TestLgr(t *testing.T) {
	t.Logf("-----------------------------------------------------------------")
	log := NewDefault()
	log = log.With("foo", "bar")
	log.Info("this is a info message", "uid", 7, "name", "user001")
	log.Warn("danger, be aware!", "uid", 8, "name", "user001")
}

func TestNew(t *testing.T) {
	t.Logf("-----------------------------------------------------------------")
	log := NewLogger(WithName("log001"), WithEncoding("console"))
	log = log.With("foo", "bar")
	log.Info("this is a info message", "uid", 7, "name", "user001")
	log.Warn("danger, be aware!", "uid", 8, "name", "user001")
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

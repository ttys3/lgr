package lgr

import (
	"encoding/base64"
	"fmt"
	"github.com/ttys3/lgr/internal"
	"github.com/ttys3/lgr/internal/bufferpool"
	"go.uber.org/zap/zapcore"
	"math"
	"sync"
	"time"
	"unicode/utf8"

	"go.uber.org/zap/buffer"
)

var _sliceEncoderPool = sync.Pool{
	New: func() interface{} {
		return &internal.SliceArrayEncoder{Elems: make([]interface{}, 0, 2)}
	},
}

func getSliceEncoder() *internal.SliceArrayEncoder {
	return _sliceEncoderPool.Get().(*internal.SliceArrayEncoder)
}

func putSliceEncoder(e *internal.SliceArrayEncoder) {
	e.Elems = e.Elems[:0]
	_sliceEncoderPool.Put(e)
}

type cliEncoder struct {
	*zapcore.EncoderConfig
	buf            *buffer.Buffer
	colored        bool // colored context key
	openNamespaces int

	// for encoding generic values by reflection
	reflectBuf *buffer.Buffer
	reflectEnc zapcore.ReflectedEncoder
}

var _cliPool = sync.Pool{New: func() interface{} {
	return &cliEncoder{}
}}

func getcliEncoder() *cliEncoder {
	return _cliPool.Get().(*cliEncoder)
}

func putcliEncoder(enc *cliEncoder) {
	if enc.reflectBuf != nil {
		enc.reflectBuf.Free()
	}
	enc.EncoderConfig = nil
	enc.buf = nil
	enc.colored = false
	enc.openNamespaces = 0
	enc.reflectBuf = nil
	enc.reflectEnc = nil
	_cliPool.Put(enc)
}

// NewCliEncoder creates an encoder whose output is designed for human -
// rather than machine - consumption. It serializes the core log entry data
// (message, level, timestamp, etc.) in a plain-text format and leaves the
// structured context as JSON.
//
// Note that although the console encoder doesn't use the keys specified in the
// encoder configuration, it will omit any element whose key is set to the empty
// string.
func NewCliEncoder(cfg zapcore.EncoderConfig) zapcore.Encoder {
	if cfg.ConsoleSeparator == "" {
		// Use a default delimiter of '\t' for backwards compatibility
		cfg.ConsoleSeparator = " "
	}
	if cfg.SkipLineEnding {
		cfg.LineEnding = ""
	} else if cfg.LineEnding == "" {
		cfg.LineEnding = zapcore.DefaultLineEnding
	}

	// If no EncoderConfig.NewReflectedEncoder is provided by the user, then use default
	if cfg.NewReflectedEncoder == nil {
		cfg.NewReflectedEncoder = internal.DefaultReflectedEncoder
	}

	forceEnableColor()
	cfg.EncodeLevel = CliLevelEncoder
	cfg.TimeKey = ""

	return &cliEncoder{
		EncoderConfig: &cfg,
		buf:           bufferpool.Get(),
		colored:       true,
	}
}

func (enc *cliEncoder) Clone() zapcore.Encoder {
	clone := enc.clone()
	clone.buf.Write(enc.buf.Bytes())
	return clone
}

func (enc *cliEncoder) clone() *cliEncoder {
	clone := getcliEncoder()
	clone.EncoderConfig = enc.EncoderConfig
	clone.colored = enc.colored
	clone.openNamespaces = enc.openNamespaces
	clone.buf = bufferpool.Get()
	return clone
}

func (enc cliEncoder) EncodeEntry(ent zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	line := bufferpool.Get()

	// We don't want the entry's metadata to be quoted and escaped (if it's
	// encoded as strings), which means that we can't use the JSON encoder. The
	// simplest option is to use the memory encoder and fmt.Fprint.
	//
	// If this ever becomes a performance bottleneck, we can implement
	// ArrayEncoder for our plain-text format.
	arr := getSliceEncoder()
	if enc.TimeKey != "" && enc.EncodeTime != nil {
		enc.EncodeTime(ent.Time, arr)
	}
	if enc.LevelKey != "" && enc.EncodeLevel != nil {
		enc.EncodeLevel(ent.Level, arr)
	}
	if ent.LoggerName != "" && enc.NameKey != "" {
		nameEncoder := enc.EncodeName

		if nameEncoder == nil {
			// Fall back to FullNameEncoder for backward compatibility.
			nameEncoder = zapcore.FullNameEncoder
		}

		nameEncoder(ent.LoggerName, arr)
	}
	if ent.Caller.Defined {
		if enc.CallerKey != "" && enc.EncodeCaller != nil {
			enc.EncodeCaller(ent.Caller, arr)
		}
		if enc.FunctionKey != "" {
			arr.AppendString(ent.Caller.Function)
		}
	}
	for i := range arr.Elems {
		if i > 0 {
			line.AppendString(enc.ConsoleSeparator)
		}
		fmt.Fprint(line, arr.Elems[i])
	}
	putSliceEncoder(arr)

	// Add the message itself.
	if enc.MessageKey != "" {
		enc.addSeparatorIfNecessary(line)
		line.AppendString(ent.Message)
	}

	// Add any structured context.
	enc.writeContext(ent.Level, line, fields)

	// If there's no stacktrace key, honor that; this allows users to force
	// single-line output.
	if ent.Stack != "" && enc.StacktraceKey != "" {
		line.AppendByte('\n')
		line.AppendString(ent.Stack)
	}

	if enc.LineEnding != "" {
		line.AppendString(enc.LineEnding)
	} else {
		line.AppendString(zapcore.DefaultLineEnding)
	}
	return line, nil
}

func (enc cliEncoder) writeContext(level zapcore.Level, line *buffer.Buffer, extra []zapcore.Field) {
	context := enc.Clone().(*cliEncoder)
	defer func() {
		// putcliEncoder assumes the buffer is still used, but we write out the buffer so
		// we can free it.
		context.buf.Free()
		putcliEncoder(context)
	}()

	addFields(context, level, extra)
	context.closeOpenNamespaces()
	if context.buf.Len() == 0 {
		return
	}

	enc.addSeparatorIfNecessary(line)
	line.AppendByte(' ')
	line.Write(context.buf.Bytes())
	line.AppendByte(' ')
}

func (enc cliEncoder) addSeparatorIfNecessary(line *buffer.Buffer) {
	if line.Len() > 0 {
		line.AppendString(enc.ConsoleSeparator)
	}
}

func (enc *cliEncoder) AddArray(key string, arr zapcore.ArrayMarshaler) error {
	enc.addKey(key)
	return enc.AppendArray(arr)
}

func (enc *cliEncoder) AddObject(key string, obj zapcore.ObjectMarshaler) error {
	enc.addKey(key)
	return enc.AppendObject(obj)
}

func (enc *cliEncoder) AddBinary(key string, val []byte) {
	enc.AddString(key, base64.StdEncoding.EncodeToString(val))
}

func (enc *cliEncoder) AddByteString(key string, val []byte) {
	enc.addKey(key)
	enc.AppendByteString(val)
}

func (enc *cliEncoder) AddBool(key string, val bool) {
	enc.addKey(key)
	enc.AppendBool(val)
}

func (enc *cliEncoder) AddComplex128(key string, val complex128) {
	enc.addKey(key)
	enc.AppendComplex128(val)
}

func (enc *cliEncoder) AddComplex64(key string, val complex64) {
	enc.addKey(key)
	enc.AppendComplex64(val)
}

func (enc *cliEncoder) AddDuration(key string, val time.Duration) {
	enc.addKey(key)
	enc.AppendDuration(val)
}

func (enc *cliEncoder) AddFloat64(key string, val float64) {
	enc.addKey(key)
	enc.AppendFloat64(val)
}

func (enc *cliEncoder) AddFloat32(key string, val float32) {
	enc.addKey(key)
	enc.AppendFloat32(val)
}

func (enc *cliEncoder) AddInt64(key string, val int64) {
	enc.addKey(key)
	enc.AppendInt64(val)
}

var nullLiteralBytes = []byte("null")

func (enc *cliEncoder) resetReflectBuf() {
	if enc.reflectBuf == nil {
		enc.reflectBuf = bufferpool.Get()
		enc.reflectEnc = enc.NewReflectedEncoder(enc.reflectBuf)
	} else {
		enc.reflectBuf.Reset()
	}
}

// Only invoke the standard JSON encoder if there is actually something to
// encode; otherwise write JSON null literal directly.
func (enc *cliEncoder) encodeReflected(obj interface{}) ([]byte, error) {
	if obj == nil {
		return nullLiteralBytes, nil
	}
	enc.resetReflectBuf()
	if err := enc.reflectEnc.Encode(obj); err != nil {
		return nil, err
	}
	enc.reflectBuf.TrimNewline()
	return enc.reflectBuf.Bytes(), nil
}

func (enc *cliEncoder) AddReflected(key string, obj interface{}) error {
	valueBytes, err := enc.encodeReflected(obj)
	if err != nil {
		return err
	}
	enc.addKey(key)
	_, err = enc.buf.Write(valueBytes)
	return err
}

func (enc *cliEncoder) OpenNamespace(key string) {
	enc.addKey(key)
	enc.buf.AppendByte('{')
	enc.openNamespaces++
}

func (enc *cliEncoder) AddString(key, val string) {
	enc.addKey(key)
	enc.AppendString(val)
}

func (enc *cliEncoder) AddTime(key string, val time.Time) {
	enc.addKey(key)
	enc.AppendTime(val)
}

func (enc *cliEncoder) AddUint64(key string, val uint64) {
	enc.addKey(key)
	enc.AppendUint64(val)
}

func (enc *cliEncoder) AppendArray(arr zapcore.ArrayMarshaler) error {
	// enc.addElementSeparator()
	enc.buf.AppendByte('[')
	err := arr.MarshalLogArray(enc)
	enc.buf.AppendByte(']')
	return err
}

func (enc *cliEncoder) AppendObject(obj zapcore.ObjectMarshaler) error {
	// Close ONLY new openNamespaces that are created during
	// AppendObject().
	old := enc.openNamespaces
	enc.openNamespaces = 0
	// enc.addElementSeparator()
	enc.buf.AppendByte('{')
	err := obj.MarshalLogObject(enc)
	enc.buf.AppendByte('}')
	enc.closeOpenNamespaces()
	enc.openNamespaces = old
	return err
}

func (enc *cliEncoder) closeOpenNamespaces() {
	for i := 0; i < enc.openNamespaces; i++ {
		enc.buf.AppendByte('}')
	}
	enc.openNamespaces = 0
}

func (enc *cliEncoder) AppendBool(val bool) {
	// enc.addElementSeparator()
	enc.buf.AppendBool(val)
}

func (enc *cliEncoder) AppendByteString(val []byte) {
	// enc.addElementSeparator()

	enc.safeAddByteString(val)

}

// appendComplex appends the encoded form of the provided complex128 value.
// precision specifies the encoding precision for the real and imaginary
// components of the complex number.
func (enc *cliEncoder) appendComplex(val complex128, precision int) {
	// enc.addElementSeparator()
	// Cast to a platform-independent, fixed-size type.
	r, i := float64(real(val)), float64(imag(val))
	// enc.buf.AppendByte('"')
	// Because we're always in a quoted string, we can use strconv without
	// special-casing NaN and +/-Inf.
	enc.buf.AppendFloat(r, precision)
	// If imaginary part is less than 0, minus (-) sign is added by default
	// by AppendFloat.
	if i >= 0 {
		enc.buf.AppendByte('+')
	}
	enc.buf.AppendFloat(i, precision)
	enc.buf.AppendByte('i')
	// enc.buf.AppendByte('"')
}

func (enc *cliEncoder) AppendDuration(val time.Duration) {
	cur := enc.buf.Len()
	if e := enc.EncodeDuration; e != nil {
		e(val, enc)
	}
	if cur == enc.buf.Len() {
		// User-supplied EncodeDuration is a no-op. Fall back to nanoseconds to keep
		// JSON valid.
		enc.AppendInt64(int64(val))
	}
}

func (enc *cliEncoder) AppendInt64(val int64) {
	// enc.addElementSeparator()
	enc.buf.AppendInt(val)
}

func (enc *cliEncoder) AppendReflected(val interface{}) error {
	valueBytes, err := enc.encodeReflected(val)
	if err != nil {
		return err
	}
	// enc.addElementSeparator()
	_, err = enc.buf.Write(valueBytes)
	return err
}

func (enc *cliEncoder) AppendString(val string) {
	// enc.addElementSeparator()
	enc.safeAddString(val)
}

func (enc *cliEncoder) AppendTimeLayout(time time.Time, layout string) {
	// enc.addElementSeparator()
	enc.buf.AppendTime(time, layout)
}

func (enc *cliEncoder) AppendTime(val time.Time) {
	cur := enc.buf.Len()
	if e := enc.EncodeTime; e != nil {
		e(val, enc)
	}
	if cur == enc.buf.Len() {
		// User-supplied EncodeTime is a no-op. Fall back to nanos since epoch to keep
		// output JSON valid.
		enc.AppendInt64(val.UnixNano())
	}
}

func (enc *cliEncoder) AppendUint64(val uint64) {
	// enc.addElementSeparator()
	enc.buf.AppendUint(val)
}

func (enc *cliEncoder) appendFloat(val float64, bitSize int) {
	// enc.addElementSeparator()
	switch {
	case math.IsNaN(val):
		enc.buf.AppendString("NaN")
	case math.IsInf(val, 1):
		enc.buf.AppendString("+Inf")
	case math.IsInf(val, -1):
		enc.buf.AppendString("-Inf")
	default:
		enc.buf.AppendFloat(val, bitSize)
	}
}

func (enc *cliEncoder) AddInt(k string, v int)         { enc.AddInt64(k, int64(v)) }
func (enc *cliEncoder) AddInt32(k string, v int32)     { enc.AddInt64(k, int64(v)) }
func (enc *cliEncoder) AddInt16(k string, v int16)     { enc.AddInt64(k, int64(v)) }
func (enc *cliEncoder) AddInt8(k string, v int8)       { enc.AddInt64(k, int64(v)) }
func (enc *cliEncoder) AddUint(k string, v uint)       { enc.AddUint64(k, uint64(v)) }
func (enc *cliEncoder) AddUint32(k string, v uint32)   { enc.AddUint64(k, uint64(v)) }
func (enc *cliEncoder) AddUint16(k string, v uint16)   { enc.AddUint64(k, uint64(v)) }
func (enc *cliEncoder) AddUint8(k string, v uint8)     { enc.AddUint64(k, uint64(v)) }
func (enc *cliEncoder) AddUintptr(k string, v uintptr) { enc.AddUint64(k, uint64(v)) }
func (enc *cliEncoder) AppendComplex64(v complex64)    { enc.appendComplex(complex128(v), 32) }
func (enc *cliEncoder) AppendComplex128(v complex128)  { enc.appendComplex(complex128(v), 64) }
func (enc *cliEncoder) AppendFloat64(v float64)        { enc.appendFloat(v, 64) }
func (enc *cliEncoder) AppendFloat32(v float32)        { enc.appendFloat(float64(v), 32) }
func (enc *cliEncoder) AppendInt(v int)                { enc.AppendInt64(int64(v)) }
func (enc *cliEncoder) AppendInt32(v int32)            { enc.AppendInt64(int64(v)) }
func (enc *cliEncoder) AppendInt16(v int16)            { enc.AppendInt64(int64(v)) }
func (enc *cliEncoder) AppendInt8(v int8)              { enc.AppendInt64(int64(v)) }
func (enc *cliEncoder) AppendUint(v uint)              { enc.AppendUint64(uint64(v)) }
func (enc *cliEncoder) AppendUint32(v uint32)          { enc.AppendUint64(uint64(v)) }
func (enc *cliEncoder) AppendUint16(v uint16)          { enc.AppendUint64(uint64(v)) }
func (enc *cliEncoder) AppendUint8(v uint8)            { enc.AppendUint64(uint64(v)) }
func (enc *cliEncoder) AppendUintptr(v uintptr)        { enc.AppendUint64(uint64(v)) }

// safeAddString JSON-escapes a string and appends it to the internal buffer.
// Unlike the standard library's encoder, it doesn't attempt to protect the
// user from browser vulnerabilities or JSONP-related problems.
func (enc *cliEncoder) safeAddString(s string) {
	for i := 0; i < len(s); {
		if enc.tryAddRuneSelf(s[i]) {
			i++
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if enc.tryAddRuneError(r, size) {
			i++
			continue
		}
		enc.buf.AppendString(s[i : i+size])
		i += size
	}
}

// safeAddByteString is no-alloc equivalent of safeAddString(string(s)) for s []byte.
func (enc *cliEncoder) safeAddByteString(s []byte) {
	for i := 0; i < len(s); {
		if enc.tryAddRuneSelf(s[i]) {
			i++
			continue
		}
		r, size := utf8.DecodeRune(s[i:])
		if enc.tryAddRuneError(r, size) {
			i++
			continue
		}
		enc.buf.Write(s[i : i+size])
		i += size
	}
}

// For JSON-escaping; see cliEncoder.safeAddString below.
const _hex = "0123456789abcdef"

// tryAddRuneSelf appends b if it is valid UTF-8 character represented in a single byte.
func (enc *cliEncoder) tryAddRuneSelf(b byte) bool {
	enc.buf.AppendByte(b)
	return true
}

func (enc *cliEncoder) tryAddRuneError(r rune, size int) bool {
	if r == utf8.RuneError && size == 1 {
		enc.buf.AppendString(`\ufffd`)
		return true
	}
	return false
}

func (enc *cliEncoder) addKey(key string) {
	enc.addElementSeparator()

	enc.safeAddString(key)

	enc.buf.AppendByte('=')
}

func (enc *cliEncoder) addElementSeparator() {
	enc.buf.AppendByte(' ')
}

func addFields(enc zapcore.ObjectEncoder, level zapcore.Level, fields []zapcore.Field) {
	color := Colors[level+1]

	for i := range fields {
		fields[i].Key = color.Sprintf("%s", fields[i].Key)
		fields[i].AddTo(enc)
	}
}

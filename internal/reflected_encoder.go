package internal

import (
	"encoding/json"
	"go.uber.org/zap/zapcore"
	"io"
)

func DefaultReflectedEncoder(w io.Writer) zapcore.ReflectedEncoder {
	enc := json.NewEncoder(w)
	// For consistency with our custom JSON encoder.
	enc.SetEscapeHTML(false)
	return enc
}

func CliReflectedEncoder(w io.Writer) zapcore.ReflectedEncoder {
	return &CliJSONEncoder{w: w}
}

type CliJSONEncoder struct {
	w io.Writer
}

func (c *CliJSONEncoder) Encode(v interface{}) error {
	out, err := Marshal(v)
	if err != nil {
		return err
	}
	_, err = c.w.Write(out)
	return err
}

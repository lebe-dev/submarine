package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

// marshalCompact serializes v the way serde_json::to_string does: compact and
// WITHOUT HTML-escaping '<', '>', '&' (Go's encoding/json escapes these by
// default; serde_json does not). The trailing newline added by json.Encoder is
// trimmed so the caller's Println supplies exactly one (matching Rust println!).
func marshalCompact(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

// successEnvelope is a port of the Rust `struct SuccessEnvelope`.
type successEnvelope struct {
	Ok   bool `json:"ok"`
	Data any  `json:"data"`
}

// errorEnvelope is a port of the Rust `struct ErrorEnvelope`.
type errorEnvelope struct {
	Ok    bool        `json:"ok"`
	Error errorDetail `json:"error"`
}

// errorDetail is a port of the Rust `struct ErrorDetail`.
type errorDetail struct {
	Code    string  `json:"code"`
	Message string  `json:"message"`
	Hint    *string `json:"hint,omitempty"`
}

// OutputSuccess outputs a successful result in the requested format.
//
// In text mode, textFn is called to produce human-readable output. In JSON
// mode, data is serialized into a `{"ok": true, "data": ...}` envelope.
// Port of the Rust `output_success`.
func OutputSuccess(format OutputFormat, data any, textFn func()) {
	switch format {
	case OutputFormatText:
		textFn()
	case OutputFormatJson:
		envelope := successEnvelope{Ok: true, Data: data}
		b, err := marshalCompact(envelope)
		if err != nil {
			panic("failed to serialize JSON")
		}
		fmt.Println(string(b))
	}
}

// OutputError outputs an error in the requested format.
//
// In text mode, the error goes to stderr. In JSON mode, the error goes to
// stdout as `{"ok": false, "error": {"code": ..., "message": ..., "hint": ...}}`.
// Port of the Rust `output_error`.
func OutputError(format OutputFormat, code, message string, hint *string) {
	switch format {
	case OutputFormatText:
		fmt.Fprintf(os.Stderr, "error: %s\n", message)
		if hint != nil {
			fmt.Fprintf(os.Stderr, "hint: %s\n", *hint)
		}
	case OutputFormatJson:
		envelope := errorEnvelope{
			Ok: false,
			Error: errorDetail{
				Code:    code,
				Message: message,
				Hint:    hint,
			},
		}
		b, err := marshalCompact(envelope)
		if err != nil {
			panic("failed to serialize JSON")
		}
		fmt.Println(string(b))
	}
}

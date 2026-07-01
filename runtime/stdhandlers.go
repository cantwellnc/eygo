package runtime

import (
	"fmt"
	"io"
	"os"

	"github.com/cantwellnc/eygo/eval"
)

// StdHandlers returns the standard set of CLI-compatible effect handlers.
// Pass a subset if you want to restrict what scripts can do.
func StdHandlers() []Handler {
	return []Handler{
		PrintHandler(os.Stdout),
		EnvHandler(),
	}
}

// PrintHandler handles the "Print" effect: Print(string) -> {}
func PrintHandler(w io.Writer) Handler {
	return NewHandler("Print", func(v eval.Value) (eval.Value, error) {
		s, ok := v.(eval.VString)
		if !ok {
			return nil, fmt.Errorf("Print: expected string, got %T", v)
		}
		fmt.Fprintln(w, s.V)
		return eval.VRecord{}, nil
	})
}

// EnvHandler handles the "Env" effect: Env(key) -> Ok(value) | Error({})
func EnvHandler() Handler {
	return NewHandler("Env", func(v eval.Value) (eval.Value, error) {
		key, ok := v.(eval.VString)
		if !ok {
			return nil, fmt.Errorf("Env: expected string key, got %T", v)
		}
		val, exists := os.LookupEnv(key.V)
		if !exists {
			return eval.VTagged{Label: "Error", Value: eval.VRecord{}}, nil
		}
		return eval.VTagged{Label: "Ok", Value: eval.VString{V: val}}, nil
	})
}

// ReadFileHandler handles "ReadFile": ReadFile(path) -> Ok(binary) | Error(string)
func ReadFileHandler() Handler {
	return NewHandler("ReadFile", func(v eval.Value) (eval.Value, error) {
		path, ok := v.(eval.VString)
		if !ok {
			return nil, fmt.Errorf("ReadFile: expected string path, got %T", v)
		}
		data, err := os.ReadFile(path.V)
		if err != nil {
			return eval.VTagged{Label: "Error", Value: eval.VString{V: err.Error()}}, nil
		}
		return eval.VTagged{Label: "Ok", Value: eval.VBinary{V: data}}, nil
	})
}

// WriteFileHandler handles "WriteFile": WriteFile({path, content}) -> Ok({}) | Error(string)
func WriteFileHandler() Handler {
	return NewHandler("WriteFile", func(v eval.Value) (eval.Value, error) {
		rec, ok := v.(eval.VRecord)
		if !ok {
			return nil, fmt.Errorf("WriteFile: expected record {path, content}, got %T", v)
		}
		pathVal := rec.Get("path")
		contentVal := rec.Get("content")
		if pathVal == nil || contentVal == nil {
			return nil, fmt.Errorf("WriteFile: record must have path and content fields")
		}
		path, ok := pathVal.(eval.VString)
		if !ok {
			return nil, fmt.Errorf("WriteFile: path must be string")
		}
		content, ok := contentVal.(eval.VBinary)
		if !ok {
			return nil, fmt.Errorf("WriteFile: content must be binary")
		}
		if err := os.WriteFile(path.V, content.V, 0o644); err != nil {
			return eval.VTagged{Label: "Error", Value: eval.VString{V: err.Error()}}, nil
		}
		return eval.VTagged{Label: "Ok", Value: eval.VRecord{}}, nil
	})
}

// DenyHandler returns a handler that rejects a specific effect with an error.
// Useful for explicitly blocking effects in sandboxed execution.
func DenyHandler(label string) Handler {
	return NewHandler(label, func(v eval.Value) (eval.Value, error) {
		return nil, fmt.Errorf("effect %q is not permitted in this context", label)
	})
}

// LoggingHandler wraps another handler, logging each invocation to w.
func LoggingHandler(h Handler, w io.Writer) Handler {
	return NewHandler(h.Label(), func(v eval.Value) (eval.Value, error) {
		fmt.Fprintf(w, "[eygo] effect %s(%s)\n", h.Label(), v)
		reply, err := h.Handle(v)
		if err != nil {
			fmt.Fprintf(w, "[eygo] effect %s -> error: %v\n", h.Label(), err)
		} else {
			fmt.Fprintf(w, "[eygo] effect %s -> %s\n", h.Label(), reply)
		}
		return reply, err
	})
}

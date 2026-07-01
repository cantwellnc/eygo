package eval

import "fmt"

// Env is an immutable linked-list environment mapping labels to values.
type Env struct {
	label string
	value Value
	outer *Env
}

var emptyEnv = &Env{}

func NewEnv() *Env { return emptyEnv }

// Extend returns a new Env with label bound to value.
func (e *Env) Extend(label string, value Value) *Env {
	return &Env{label: label, value: value, outer: e}
}

// Lookup finds a binding by label.
func (e *Env) Lookup(label string) (Value, error) {
	for cur := e; cur != nil && cur.label != ""; cur = cur.outer {
		if cur.label == label {
			return cur.value, nil
		}
	}
	return nil, fmt.Errorf("eval: unbound variable %q", label)
}

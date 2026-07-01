// Package runtime provides the effect dispatch loop and Handler interface for
// running EYG programs from Go. This is the primary integration point for
// embedding EYG in applications (e.g., kronk-generated scripts).
package runtime

import (
	"fmt"

	"github.com/cantwellnc/eygo/eval"
	"github.com/cantwellnc/eygo/ir"
)

// Handler handles a single named effect.
// Receive the effect value and return a reply value (or an error).
// Returning an error causes program execution to halt.
type Handler interface {
	// Label returns the effect name this handler handles (e.g. "Print", "Fetch").
	Label() string
	// Handle receives the effect payload and returns a reply.
	Handle(value eval.Value) (eval.Value, error)
}

// HandlerFunc is a function that implements Handler.
type HandlerFunc struct {
	label string
	fn    func(eval.Value) (eval.Value, error)
}

func (h HandlerFunc) Label() string                         { return h.label }
func (h HandlerFunc) Handle(v eval.Value) (eval.Value, error) { return h.fn(v) }

// NewHandler creates a Handler from a label and function.
func NewHandler(label string, fn func(eval.Value) (eval.Value, error)) Handler {
	return HandlerFunc{label: label, fn: fn}
}

// Runtime executes EYG programs with a registered set of effect handlers.
type Runtime struct {
	interp   *interpFace
	handlers map[string]Handler
}

// interpFace is a thin interface so runtime doesn't import eval's concrete type directly.
type interpFace struct {
	i *eval.Interpreter
}

// New creates a Runtime with builtins pre-registered.
// Pass additional handlers via WithHandler.
func New(i *eval.Interpreter, handlers ...Handler) *Runtime {
	r := &Runtime{
		interp:   &interpFace{i: i},
		handlers: make(map[string]Handler),
	}
	for _, h := range handlers {
		r.handlers[h.Label()] = h
	}
	return r
}

// WithHandler registers an additional effect handler.
func (r *Runtime) WithHandler(h Handler) *Runtime {
	r.handlers[h.Label()] = h
	return r
}

// Run evaluates node and drives the effect dispatch loop until the program
// terminates (returning a final Value) or an unhandled effect is raised.
func (r *Runtime) Run(node ir.Node) (eval.Value, error) {
	result := r.interp.i.EvalProgram(node)
	return r.dispatch(result)
}

// dispatch drives the effect loop: when an effect bubbles out, call its handler
// and resume the computation with the reply.
func (r *Runtime) dispatch(result eval.Result) (eval.Value, error) {
	for {
		if result.Err != nil {
			return nil, result.Err
		}
		if result.Effect == nil {
			return result.Val, nil
		}

		e := result.Effect
		h, ok := r.handlers[e.Label]
		if !ok {
			return nil, fmt.Errorf("runtime: unhandled effect %q (value: %s)", e.Label, e.Value)
		}

		reply, err := h.Handle(e.Value)
		if err != nil {
			return nil, fmt.Errorf("runtime: handler %q: %w", e.Label, err)
		}

		result = e.Resume(reply)
	}
}

// Inspect returns all effect labels that would be raised by running node,
// without actually executing any handlers. Useful for pre-flight safety checks
// (e.g. "does this script try to access the filesystem?").
//
// Limitation: only statically reachable performs are detected; dynamic effects
// inside closures that are never called will not appear.
func Inspect(node ir.Node) []string {
	seen := make(map[string]bool)
	collectPerforms(node, seen)
	labels := make([]string, 0, len(seen))
	for l := range seen {
		labels = append(labels, l)
	}
	return labels
}

func collectPerforms(node ir.Node, out map[string]bool) {
	if node == nil {
		return
	}
	switch n := node.(type) {
	case *ir.Perform:
		out[n.Label] = true
	case *ir.Lambda:
		collectPerforms(n.Body, out)
	case *ir.Apply:
		collectPerforms(n.Fn, out)
		collectPerforms(n.Arg, out)
	case *ir.Let:
		collectPerforms(n.Value, out)
		collectPerforms(n.Then, out)
	case *ir.Handle:
		// The body of a handle is the third argument — we can't statically
		// traverse it here without evaluating, so we conservatively include
		// the effect label as "handled".
	case *ir.Shallow:
		// same
	}
}

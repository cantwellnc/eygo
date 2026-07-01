package eval

import (
	"fmt"

	"github.com/cantwellnc/eygo/ir"
)

// Interpreter is a tree-walking EYG evaluator.
type Interpreter struct {
	builtins map[string]func(Value) Result
}

// New returns an Interpreter with no builtins registered.
func New() *Interpreter {
	return &Interpreter{builtins: make(map[string]func(Value) Result)}
}

// RegisterBuiltin registers a named primitive (curried: receives first arg,
// returns Result which may be another VPartial for multi-arg builtins).
func (interp *Interpreter) RegisterBuiltin(name string, fn func(Value) Result) {
	interp.builtins[name] = fn
}

// Eval evaluates node in env. Effects bubble out as Result.Effect with a
// non-nil Resume that continues the computation when called with a reply.
func (interp *Interpreter) Eval(node ir.Node, env *Env) Result {
	return interp.eval(node, env)
}

func (interp *Interpreter) eval(node ir.Node, env *Env) Result {
	switch n := node.(type) {

	// ── Primitives ────────────────────────────────────────────────────────────

	case *ir.Var:
		v, err := env.Lookup(n.Label)
		if err != nil {
			return Err(err)
		}
		return Ok(v)

	case *ir.Int:
		return Ok(VInt{V: n.Value})

	case *ir.String:
		return Ok(VString{V: n.Value})

	case *ir.Binary:
		return Ok(VBinary{V: n.Value})

	// ── Functions ─────────────────────────────────────────────────────────────

	case *ir.Lambda:
		return Ok(VClosure{Label: n.Label, Body: n.Body, Env: env})

	case *ir.Apply:
		fnRes := interp.eval(n.Fn, env)
		if fnRes.Err != nil {
			return fnRes
		}
		if fnRes.Effect != nil {
			// Effect came from evaluating Fn — after resume, re-enter Apply
			// with the resumed fn value, still needing to eval Arg and apply.
			argNode, env2 := n.Arg, env
			return propagate(fnRes.Effect, func(resumedFn Value) Result {
				argRes := interp.eval(argNode, env2)
				if argRes.Err != nil || argRes.Effect != nil {
					return liftEffect(argRes, func(argVal Value) Result {
						return interp.apply(resumedFn, argVal)
					})
				}
				return interp.apply(resumedFn, argRes.Val)
			})
		}

		argRes := interp.eval(n.Arg, env)
		if argRes.Err != nil {
			return argRes
		}
		if argRes.Effect != nil {
			fn := fnRes.Val
			return propagate(argRes.Effect, func(argVal Value) Result {
				return interp.apply(fn, argVal)
			})
		}

		return interp.apply(fnRes.Val, argRes.Val)

	case *ir.Let:
		// Desugar: let x = v in t  →  eval v, bind x, eval t
		valRes := interp.eval(n.Value, env)
		if valRes.Err != nil {
			return valRes
		}
		if valRes.Effect != nil {
			then, label, env2 := n.Then, n.Label, env
			return propagate(valRes.Effect, func(val Value) Result {
				return interp.eval(then, env2.Extend(label, val))
			})
		}
		return interp.eval(n.Then, env.Extend(n.Label, valRes.Val))

	// ── Lists ─────────────────────────────────────────────────────────────────

	case *ir.Tail:
		return Ok(VList{})

	case *ir.Cons:
		return Ok(VPartial{Name: "cons", Fn: func(head Value) Result {
			return Ok(VPartial{Name: "cons/1", Fn: func(tail Value) Result {
				t, ok := tail.(VList)
				if !ok {
					return Errf("cons: tail must be a list, got %T", tail)
				}
				return Ok(VList{Head: head, Tail: &t})
			}})
		}})

	// ── Records ───────────────────────────────────────────────────────────────

	case *ir.Empty:
		return Ok(VRecord{})

	case *ir.Extend:
		label := n.Label
		return Ok(VPartial{Name: "extend:" + label, Fn: func(val Value) Result {
			return Ok(VPartial{Name: "extend:" + label + "/1", Fn: func(rec Value) Result {
				r, ok := rec.(VRecord)
				if !ok {
					return Errf("extend: expected record, got %T", rec)
				}
				return Ok(RecordExtend(label, val, r))
			}})
		}})

	case *ir.Overwrite:
		label := n.Label
		return Ok(VPartial{Name: "overwrite:" + label, Fn: func(val Value) Result {
			return Ok(VPartial{Name: "overwrite:" + label + "/1", Fn: func(rec Value) Result {
				r, ok := rec.(VRecord)
				if !ok {
					return Errf("overwrite: expected record, got %T", rec)
				}
				return Ok(RecordOverwrite(label, val, r))
			}})
		}})

	case *ir.Select:
		label := n.Label
		return Ok(VPartial{Name: "select:" + label, Fn: func(v Value) Result {
			rec, ok := v.(VRecord)
			if !ok {
				return Errf("select: expected record, got %T", v)
			}
			field := rec.Get(label)
			if field == nil {
				return Errf("select: no field %q in %s", label, rec)
			}
			return Ok(field)
		}})

	// ── Variants ──────────────────────────────────────────────────────────────

	case *ir.Tag:
		label := n.Label
		return Ok(VPartial{Name: "tag:" + label, Fn: func(v Value) Result {
			return Ok(VTagged{Label: label, Value: v})
		}})

	case *ir.Case:
		label := n.Label
		return Ok(VPartial{Name: "case:" + label, Fn: func(handler Value) Result {
			return Ok(VPartial{Name: "case:" + label + "/otherwise", Fn: func(otherwise Value) Result {
				return Ok(VPartial{Name: "case:" + label + "/scrutinee", Fn: func(scrutinee Value) Result {
					tagged, ok := scrutinee.(VTagged)
					if !ok {
						return Errf("case: expected tagged value, got %T", scrutinee)
					}
					if tagged.Label == label {
						return interp.apply(handler, tagged.Value)
					}
					return interp.apply(otherwise, scrutinee)
				}})
			}})
		}})

	case *ir.NoCases:
		return Ok(VPartial{Name: "nocases", Fn: func(v Value) Result {
			return Errf("no matching case for %s", v)
		}})

	// ── Effects ───────────────────────────────────────────────────────────────

	case *ir.Perform:
		label := n.Label
		return Ok(VPartial{Name: "perform:" + label, Fn: func(v Value) Result {
			// Resume is identity: handler reply flows straight through.
			return Eff(&Effect{
				Label:  label,
				Value:  v,
				Resume: Ok,
			})
		}})

	case *ir.Handle:
		// handle(label)(handler)(body_thunk)
		// Deep: reinstalls handler after each resume so subsequent effects are caught.
		label := n.Label
		return Ok(VPartial{Name: "handle:" + label, Fn: func(handler Value) Result {
			return Ok(VPartial{Name: "handle:" + label + "/body", Fn: func(body Value) Result {
				res := interp.apply(body, VRecord{})
				return interp.dispatchDeep(label, handler, res)
			}})
		}})

	case *ir.Shallow:
		// Shallow: handles one occurrence and does not reinstall.
		label := n.Label
		return Ok(VPartial{Name: "shallow:" + label, Fn: func(handler Value) Result {
			return Ok(VPartial{Name: "shallow:" + label + "/body", Fn: func(body Value) Result {
				res := interp.apply(body, VRecord{})
				return interp.dispatchShallow(label, handler, res)
			}})
		}})

	// ── Builtins ──────────────────────────────────────────────────────────────

	case *ir.Builtin:
		fn, ok := interp.builtins[n.Label]
		if !ok {
			return Errf("unknown builtin %q", n.Label)
		}
		return Ok(VPartial{Name: n.Label, Fn: fn})

	// ── References ────────────────────────────────────────────────────────────

	case *ir.Reference:
		return Errf("CID references not yet supported: %s", n.CID)

	case *ir.Release:
		return Errf("release references not yet supported: %s@%d", n.Package, n.Release)

	case *ir.Vacant:
		return Errf("vacant expression (hole): %q", n.Comment)

	default:
		return Errf("unhandled node type %T", node)
	}
}

// apply calls fn with arg, propagating effects.
func (interp *Interpreter) apply(fn, arg Value) Result {
	switch f := fn.(type) {
	case VClosure:
		node, ok := f.Body.(ir.Node)
		if !ok {
			return Errf("closure body is not an ir.Node")
		}
		return interp.eval(node, f.Env.Extend(f.Label, arg))
	case VPartial:
		return f.Fn(arg)
	default:
		return Errf("cannot apply non-function %T (%s)", fn, fn)
	}
}

// Apply is exported for use by the runtime and RegisterFix.
func (interp *Interpreter) Apply(fn, arg Value) Result {
	return interp.apply(fn, arg)
}

// EvalProgram evaluates node with an empty environment.
func (interp *Interpreter) EvalProgram(node ir.Node) Result {
	return interp.eval(node, NewEnv())
}

// dispatchDeep handles effects for a deep Handle node.
// Reinstalls the handler after each resume so subsequent effects of the same
// label within the same computation are also caught.
func (interp *Interpreter) dispatchDeep(label string, handler Value, res Result) Result {
	for {
		if res.Err != nil || res.Effect == nil {
			return res
		}
		e := res.Effect
		if e.Label != label {
			// Different effect — let it propagate, but wrap its resume so we
			// reinstall this handler when the outer effect is eventually handled.
			h, lbl := handler, label
			return propagate(e, func(v Value) Result {
				return interp.dispatchDeep(lbl, h, Ok(v))
			})
		}
		// Matching effect: call handler(value)(resumeFn)
		capturedE := e
		resumeFn := VPartial{
			Name: "resume:" + label,
			Fn: func(resumeVal Value) Result {
				inner := capturedE.Resume(resumeVal)
				return interp.dispatchDeep(label, handler, inner)
			},
		}
		r1 := interp.apply(handler, e.Value)
		if r1.Err != nil || r1.Effect != nil {
			res = r1
			continue
		}
		res = interp.apply(r1.Val, resumeFn)
	}
}

// dispatchShallow handles one occurrence of the labeled effect, then stops.
func (interp *Interpreter) dispatchShallow(label string, handler Value, res Result) Result {
	if res.Err != nil || res.Effect == nil {
		return res
	}
	e := res.Effect
	if e.Label != label {
		return res
	}
	resumeFn := VPartial{
		Name: "resume:" + label,
		Fn:   e.Resume, // does NOT reinstall the handler
	}
	r1 := interp.apply(handler, e.Value)
	if r1.Err != nil || r1.Effect != nil {
		return r1
	}
	return interp.apply(r1.Val, resumeFn)
}

// propagate rewraps an effect's Resume with a continuation k.
// When the effect is eventually handled and Resume is called, the reply flows
// through the original resume chain then into k.
func propagate(e *Effect, k func(Value) Result) Result {
	orig := e.Resume
	wrapped := &Effect{
		Label: e.Label,
		Value: e.Value,
		Resume: func(reply Value) Result {
			r := orig(reply)
			return liftEffect(r, k)
		},
	}
	return Eff(wrapped)
}

// liftEffect applies k to a Result's value, threading any nested effects through.
func liftEffect(r Result, k func(Value) Result) Result {
	if r.Err != nil {
		return r
	}
	if r.Effect != nil {
		return propagate(r.Effect, k)
	}
	return k(r.Val)
}

// Sprint returns a human-readable representation of an EYG value.
func Sprint(v Value) string {
	if v == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%s", v)
}

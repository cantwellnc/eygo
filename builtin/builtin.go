// Package builtin registers all standard EYG primitives with an Interpreter.
package builtin

import (
	"bytes"
	"fmt"
	"strings"
	"unicode"

	"github.com/cantwellnc/eygo/eval"
)

type interp interface {
	RegisterBuiltin(name string, fn func(eval.Value) eval.Result)
}

// Register installs all standard EYG builtins into the interpreter.
func Register(i interp) {
	// ── Equality & control ────────────────────────────────────────────────────

	i.RegisterBuiltin("equal", func2(func(a, b eval.Value) eval.Result {
		if valEqual(a, b) {
			return eval.Ok(eval.VTagged{Label: "True", Value: eval.VRecord{}})
		}
		return eval.Ok(eval.VTagged{Label: "False", Value: eval.VRecord{}})
	}))

	// fix: fixed-point combinator — fix(f)(x) = f(fix(f))(x)
	i.RegisterBuiltin("fix", func(fn eval.Value) eval.Result {
		var fixedFn eval.Value
		fixedFn = eval.VPartial{Name: "fix/applied", Fn: func(x eval.Value) eval.Result {
			// fix(f)(x) = f(fix(f))(x)
			// First apply f to fix(f) to get f', then apply f' to x
			// We need the interpreter — store it via closure over this registration.
			// We'll use a relay VPartial.
			return eval.Errf("fix: must be used via RegisterBuiltin which has interp access")
		}}
		_ = fixedFn
		// Real fix is wired in Register with interp access below.
		return eval.Errf("fix: not yet wired")
	})

	// ── Integer arithmetic ────────────────────────────────────────────────────

	i.RegisterBuiltin("int_add", intBinop(func(a, b int64) int64 { return a + b }))
	i.RegisterBuiltin("int_subtract", intBinop(func(a, b int64) int64 { return a - b }))
	i.RegisterBuiltin("int_multiply", intBinop(func(a, b int64) int64 { return a * b }))
	i.RegisterBuiltin("int_divide", func(a eval.Value) eval.Result {
		ai, ok := a.(eval.VInt)
		if !ok {
			return eval.Errf("int_divide: expected int, got %T", a)
		}
		return eval.Ok(eval.VPartial{Name: "int_divide/1", Fn: func(b eval.Value) eval.Result {
			bi, ok := b.(eval.VInt)
			if !ok {
				return eval.Errf("int_divide: expected int, got %T", b)
			}
			if bi.V == 0 {
				return eval.Ok(eval.VTagged{Label: "Error", Value: eval.VString{V: "division by zero"}})
			}
			return eval.Ok(eval.VTagged{Label: "Ok", Value: eval.VInt{V: ai.V / bi.V}})
		}})
	})
	i.RegisterBuiltin("int_absolute", func(a eval.Value) eval.Result {
		ai, ok := a.(eval.VInt)
		if !ok {
			return eval.Errf("int_absolute: expected int, got %T", a)
		}
		v := ai.V
		if v < 0 {
			v = -v
		}
		return eval.Ok(eval.VInt{V: v})
	})
	i.RegisterBuiltin("int_compare", func2(func(a, b eval.Value) eval.Result {
		ai, aok := a.(eval.VInt)
		bi, bok := b.(eval.VInt)
		if !aok || !bok {
			return eval.Errf("int_compare: expected ints")
		}
		return eval.Ok(compareTag(ai.V, bi.V))
	}))
	i.RegisterBuiltin("int_parse", func(a eval.Value) eval.Result {
		s, ok := a.(eval.VString)
		if !ok {
			return eval.Errf("int_parse: expected string, got %T", a)
		}
		var n int64
		_, err := fmt.Sscanf(s.V, "%d", &n)
		if err != nil {
			return eval.Ok(eval.VTagged{Label: "Error", Value: eval.VString{V: "not an integer"}})
		}
		return eval.Ok(eval.VTagged{Label: "Ok", Value: eval.VInt{V: n}})
	})
	i.RegisterBuiltin("int_to_string", func(a eval.Value) eval.Result {
		ai, ok := a.(eval.VInt)
		if !ok {
			return eval.Errf("int_to_string: expected int, got %T", a)
		}
		return eval.Ok(eval.VString{V: fmt.Sprintf("%d", ai.V)})
	})

	// ── String operations ─────────────────────────────────────────────────────

	i.RegisterBuiltin("string_append", func2(func(a, b eval.Value) eval.Result {
		as, aok := a.(eval.VString)
		bs, bok := b.(eval.VString)
		if !aok || !bok {
			return eval.Errf("string_append: expected strings")
		}
		return eval.Ok(eval.VString{V: as.V + bs.V})
	}))
	i.RegisterBuiltin("string_length", func(a eval.Value) eval.Result {
		s, ok := a.(eval.VString)
		if !ok {
			return eval.Errf("string_length: expected string, got %T", a)
		}
		return eval.Ok(eval.VInt{V: int64(len([]rune(s.V)))})
	})
	i.RegisterBuiltin("string_uppercase", strOp(strings.ToUpper))
	i.RegisterBuiltin("string_lowercase", strOp(strings.ToLower))
	i.RegisterBuiltin("string_starts_with", func2(func(a, b eval.Value) eval.Result {
		s, ok := a.(eval.VString)
		prefix, ok2 := b.(eval.VString)
		if !ok || !ok2 {
			return eval.Errf("string_starts_with: expected strings")
		}
		if strings.HasPrefix(s.V, prefix.V) {
			return eval.Ok(eval.VTagged{Label: "True", Value: eval.VRecord{}})
		}
		return eval.Ok(eval.VTagged{Label: "False", Value: eval.VRecord{}})
	}))
	i.RegisterBuiltin("string_ends_with", func2(func(a, b eval.Value) eval.Result {
		s, ok := a.(eval.VString)
		suffix, ok2 := b.(eval.VString)
		if !ok || !ok2 {
			return eval.Errf("string_ends_with: expected strings")
		}
		if strings.HasSuffix(s.V, suffix.V) {
			return eval.Ok(eval.VTagged{Label: "True", Value: eval.VRecord{}})
		}
		return eval.Ok(eval.VTagged{Label: "False", Value: eval.VRecord{}})
	}))
	i.RegisterBuiltin("string_replace", func(a eval.Value) eval.Result {
		src, ok := a.(eval.VString)
		if !ok {
			return eval.Errf("string_replace: expected string, got %T", a)
		}
		return eval.Ok(eval.VPartial{Name: "string_replace/1", Fn: func(b eval.Value) eval.Result {
			from, ok := b.(eval.VString)
			if !ok {
				return eval.Errf("string_replace: expected string, got %T", b)
			}
			return eval.Ok(eval.VPartial{Name: "string_replace/2", Fn: func(c eval.Value) eval.Result {
				to, ok := c.(eval.VString)
				if !ok {
					return eval.Errf("string_replace: expected string, got %T", c)
				}
				return eval.Ok(eval.VString{V: strings.ReplaceAll(src.V, from.V, to.V)})
			}})
		}})
	})
	i.RegisterBuiltin("string_split", func2(func(a, b eval.Value) eval.Result {
		s, ok := a.(eval.VString)
		sep, ok2 := b.(eval.VString)
		if !ok || !ok2 {
			return eval.Errf("string_split: expected strings")
		}
		parts := strings.Split(s.V, sep.V)
		return eval.Ok(strSliceToList(parts))
	}))
	i.RegisterBuiltin("string_split_once", func2(func(a, b eval.Value) eval.Result {
		s, ok := a.(eval.VString)
		sep, ok2 := b.(eval.VString)
		if !ok || !ok2 {
			return eval.Errf("string_split_once: expected strings")
		}
		before, after, found := strings.Cut(s.V, sep.V)
		if !found {
			return eval.Ok(eval.VTagged{Label: "Error", Value: eval.VRecord{}})
		}
		pair := eval.VRecord{Fields: []eval.Field{
			{Label: "0", Value: eval.VString{V: before}},
			{Label: "1", Value: eval.VString{V: after}},
		}}
		return eval.Ok(eval.VTagged{Label: "Ok", Value: pair})
	}))
	i.RegisterBuiltin("string_to_binary", func(a eval.Value) eval.Result {
		s, ok := a.(eval.VString)
		if !ok {
			return eval.Errf("string_to_binary: expected string, got %T", a)
		}
		return eval.Ok(eval.VBinary{V: []byte(s.V)})
	})
	i.RegisterBuiltin("string_from_binary", func(a eval.Value) eval.Result {
		b, ok := a.(eval.VBinary)
		if !ok {
			return eval.Errf("string_from_binary: expected binary, got %T", a)
		}
		if !isValidUTF8(b.V) {
			return eval.Ok(eval.VTagged{Label: "Error", Value: eval.VString{V: "invalid utf-8"}})
		}
		return eval.Ok(eval.VTagged{Label: "Ok", Value: eval.VString{V: string(b.V)}})
	})

	// ── List operations ───────────────────────────────────────────────────────

	// list_pop: list -> Ok({head, tail}) | Error({})
	i.RegisterBuiltin("list_pop", func(a eval.Value) eval.Result {
		lst, ok := a.(eval.VList)
		if !ok {
			return eval.Errf("list_pop: expected list, got %T", a)
		}
		if lst.Head == nil {
			return eval.Ok(eval.VTagged{Label: "Error", Value: eval.VRecord{}})
		}
		pair := eval.VRecord{Fields: []eval.Field{
			{Label: "head", Value: lst.Head},
			{Label: "tail", Value: *lst.Tail},
		}}
		return eval.Ok(eval.VTagged{Label: "Ok", Value: pair})
	})

	// list_fold: f -> acc -> list -> acc
	i.RegisterBuiltin("list_fold", func(fn eval.Value) eval.Result {
		return eval.Ok(eval.VPartial{Name: "list_fold/1", Fn: func(acc eval.Value) eval.Result {
			return eval.Ok(eval.VPartial{Name: "list_fold/2", Fn: func(lst eval.Value) eval.Result {
				l, ok := lst.(eval.VList)
				if !ok {
					return eval.Errf("list_fold: expected list, got %T", lst)
				}
				return listFold(fn, acc, l)
			}})
		}})
	})

	// ── Binary operations ─────────────────────────────────────────────────────

	i.RegisterBuiltin("binary_size", func(a eval.Value) eval.Result {
		b, ok := a.(eval.VBinary)
		if !ok {
			return eval.Errf("binary_size: expected binary, got %T", a)
		}
		return eval.Ok(eval.VInt{V: int64(len(b.V))})
	})
	i.RegisterBuiltin("binary_concat", func2(func(a, b eval.Value) eval.Result {
		ba, aok := a.(eval.VBinary)
		bb, bok := b.(eval.VBinary)
		if !aok || !bok {
			return eval.Errf("binary_concat: expected binaries")
		}
		return eval.Ok(eval.VBinary{V: append(append([]byte{}, ba.V...), bb.V...)})
	}))
	i.RegisterBuiltin("binary_compare", func2(func(a, b eval.Value) eval.Result {
		ba, aok := a.(eval.VBinary)
		bb, bok := b.(eval.VBinary)
		if !aok || !bok {
			return eval.Errf("binary_compare: expected binaries")
		}
		c := bytes.Compare(ba.V, bb.V)
		return eval.Ok(intCmpTag(c))
	}))
	i.RegisterBuiltin("binary_from_integers", func(a eval.Value) eval.Result {
		lst, ok := a.(eval.VList)
		if !ok {
			return eval.Errf("binary_from_integers: expected list, got %T", a)
		}
		var buf []byte
		cur := &lst
		for cur != nil && cur.Head != nil {
			n, ok := cur.Head.(eval.VInt)
			if !ok {
				return eval.Errf("binary_from_integers: list element is not int")
			}
			if n.V < 0 || n.V > 255 {
				return eval.Ok(eval.VTagged{Label: "Error", Value: eval.VString{V: "byte out of range"}})
			}
			buf = append(buf, byte(n.V))
			cur = cur.Tail
		}
		return eval.Ok(eval.VTagged{Label: "Ok", Value: eval.VBinary{V: buf}})
	})
	i.RegisterBuiltin("binary_fold", func(fn eval.Value) eval.Result {
		return eval.Ok(eval.VPartial{Name: "binary_fold/1", Fn: func(acc eval.Value) eval.Result {
			return eval.Ok(eval.VPartial{Name: "binary_fold/2", Fn: func(bin eval.Value) eval.Result {
				b, ok := bin.(eval.VBinary)
				if !ok {
					return eval.Errf("binary_fold: expected binary, got %T", bin)
				}
				cur := acc
				for _, byt := range b.V {
					// fn(byte)(acc) -> acc
					r1 := applyVal(fn, eval.VInt{V: int64(byt)})
					if r1.Err != nil || r1.Effect != nil {
						return r1
					}
					r2 := applyVal(r1.Val, cur)
					if r2.Err != nil || r2.Effect != nil {
						return r2
					}
					cur = r2.Val
				}
				return eval.Ok(cur)
			}})
		}})
	})

	// never: bottom type — takes any value and loops forever (or errors)
	i.RegisterBuiltin("never", func(v eval.Value) eval.Result {
		return eval.Errf("eval: never called with %s", v)
	})
}

// RegisterFix wires up the fix builtin with access to the interpreter's apply method.
// Call this after Register if you want recursion support.
func RegisterFix(i interp, apply func(fn, arg eval.Value) eval.Result) {
	var fixSelf func(fn eval.Value) eval.Value
	fixSelf = func(fn eval.Value) eval.Value {
		return eval.VPartial{Name: "fix/x", Fn: func(x eval.Value) eval.Result {
			fixed := fixSelf(fn)
			r := apply(fn, fixed)
			if r.Err != nil || r.Effect != nil {
				return r
			}
			return apply(r.Val, x)
		}}
	}
	i.RegisterBuiltin("fix", func(fn eval.Value) eval.Result {
		return eval.Ok(fixSelf(fn))
	})
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func func2(f func(a, b eval.Value) eval.Result) func(eval.Value) eval.Result {
	return func(a eval.Value) eval.Result {
		return eval.Ok(eval.VPartial{Name: "arg2", Fn: func(b eval.Value) eval.Result {
			return f(a, b)
		}})
	}
}

func intBinop(f func(a, b int64) int64) func(eval.Value) eval.Result {
	return func2(func(a, b eval.Value) eval.Result {
		ai, aok := a.(eval.VInt)
		bi, bok := b.(eval.VInt)
		if !aok || !bok {
			return eval.Errf("builtin: expected ints, got %T and %T", a, b)
		}
		return eval.Ok(eval.VInt{V: f(ai.V, bi.V)})
	})
}

func strOp(f func(string) string) func(eval.Value) eval.Result {
	return func(a eval.Value) eval.Result {
		s, ok := a.(eval.VString)
		if !ok {
			return eval.Errf("builtin: expected string, got %T", a)
		}
		return eval.Ok(eval.VString{V: f(s.V)})
	}
}

func compareTag(a, b int64) eval.Value {
	switch {
	case a < b:
		return eval.VTagged{Label: "Lt", Value: eval.VRecord{}}
	case a > b:
		return eval.VTagged{Label: "Gt", Value: eval.VRecord{}}
	default:
		return eval.VTagged{Label: "Eq", Value: eval.VRecord{}}
	}
}

func intCmpTag(c int) eval.Value {
	if c < 0 {
		return eval.VTagged{Label: "Lt", Value: eval.VRecord{}}
	} else if c > 0 {
		return eval.VTagged{Label: "Gt", Value: eval.VRecord{}}
	}
	return eval.VTagged{Label: "Eq", Value: eval.VRecord{}}
}

func strSliceToList(parts []string) eval.Value {
	lst := eval.VList{}
	for i := len(parts) - 1; i >= 0; i-- {
		lst = eval.VList{Head: eval.VString{V: parts[i]}, Tail: &lst}
	}
	return lst
}

func isValidUTF8(b []byte) bool {
	for _, r := range string(b) {
		if r == unicode.ReplacementChar {
			return false
		}
	}
	return true
}

func applyVal(fn, arg eval.Value) eval.Result {
	if p, ok := fn.(eval.VPartial); ok {
		return p.Fn(arg)
	}
	return eval.Errf("builtin: cannot apply %T", fn)
}

func listFold(fn, acc eval.Value, lst eval.VList) eval.Result {
	cur := acc
	node := &lst
	for node != nil && node.Head != nil {
		// fn(elem)(acc)
		r1 := applyVal(fn, node.Head)
		if r1.Err != nil || r1.Effect != nil {
			return r1
		}
		r2 := applyVal(r1.Val, cur)
		if r2.Err != nil || r2.Effect != nil {
			return r2
		}
		cur = r2.Val
		node = node.Tail
	}
	return eval.Ok(cur)
}

func valEqual(a, b eval.Value) bool {
	switch av := a.(type) {
	case eval.VInt:
		bv, ok := b.(eval.VInt)
		return ok && av.V == bv.V
	case eval.VString:
		bv, ok := b.(eval.VString)
		return ok && av.V == bv.V
	case eval.VBinary:
		bv, ok := b.(eval.VBinary)
		return ok && bytes.Equal(av.V, bv.V)
	case eval.VRecord:
		bv, ok := b.(eval.VRecord)
		if !ok || len(av.Fields) != len(bv.Fields) {
			return false
		}
		for _, af := range av.Fields {
			bf := bv.Get(af.Label)
			if bf == nil || !valEqual(af.Value, bf) {
				return false
			}
		}
		return true
	case eval.VTagged:
		bv, ok := b.(eval.VTagged)
		return ok && av.Label == bv.Label && valEqual(av.Value, bv.Value)
	case eval.VList:
		bv, ok := b.(eval.VList)
		if !ok {
			return false
		}
		an, bn := &av, &bv
		for an != nil && bn != nil {
			if an.Head == nil && bn.Head == nil {
				return true
			}
			if an.Head == nil || bn.Head == nil {
				return false
			}
			if !valEqual(an.Head, bn.Head) {
				return false
			}
			an, bn = an.Tail, bn.Tail
		}
		return an == bn
	}
	return false
}

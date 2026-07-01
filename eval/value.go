// Package eval implements a tree-walking interpreter for EYG IR.
package eval

import "fmt"

// Value is the result of evaluating an EYG expression.
type Value interface {
	eygValue()
	String() string
}

type VInt struct{ V int64 }
type VString struct{ V string }
type VBinary struct{ V []byte }

// VList is a linked list: either Cons(head, tail) or Nil.
type VList struct {
	Head Value // nil when empty
	Tail *VList
}

// VRecord is a row-extensible record. Fields ordered by insertion (most-recent first).
type VRecord struct{ Fields []Field }

type Field struct {
	Label string
	Value Value
}

// VTagged is a variant value: Label(Value).
type VTagged struct {
	Label string
	Value Value
}

// VClosure is a lambda capturing its environment.
type VClosure struct {
	Label string
	Body  interface{} // ir.Node — avoids import cycle
	Env   *Env
}

// VPartial is a curried function returning a Result (so effects can propagate).
type VPartial struct {
	Name string
	Fn   func(Value) Result
}

func (VInt) eygValue()     {}
func (VString) eygValue()  {}
func (VBinary) eygValue()  {}
func (VList) eygValue()    {}
func (VRecord) eygValue()  {}
func (VTagged) eygValue()  {}
func (VClosure) eygValue() {}
func (VPartial) eygValue() {}

func (v VInt) String() string    { return fmt.Sprintf("%d", v.V) }
func (v VString) String() string { return fmt.Sprintf("%q", v.V) }
func (v VBinary) String() string { return fmt.Sprintf("<%d bytes>", len(v.V)) }
func (v VTagged) String() string { return fmt.Sprintf("%s(%s)", v.Label, v.Value) }
func (v VClosure) String() string {
	return fmt.Sprintf("<fn %s>", v.Label)
}
func (v VPartial) String() string { return fmt.Sprintf("<builtin %s>", v.Name) }

func (v VRecord) String() string {
	if len(v.Fields) == 0 {
		return "{}"
	}
	s := "{"
	for i, f := range v.Fields {
		if i > 0 {
			s += ", "
		}
		s += f.Label + ": " + f.Value.String()
	}
	return s + "}"
}

func (v VList) String() string {
	if v.Head == nil {
		return "[]"
	}
	s := "[" + v.Head.String()
	cur := v.Tail
	for cur != nil && cur.Head != nil {
		s += ", " + cur.Head.String()
		cur = cur.Tail
	}
	return s + "]"
}

// Get retrieves a field by label from a record.
func (v VRecord) Get(label string) Value {
	for _, f := range v.Fields {
		if f.Label == label {
			return f.Value
		}
	}
	return nil
}

// RecordExtend prepends label=val, shadowing any existing field with that label.
func RecordExtend(label string, val Value, rec VRecord) VRecord {
	fields := make([]Field, 0, len(rec.Fields)+1)
	fields = append(fields, Field{Label: label, Value: val})
	for _, f := range rec.Fields {
		if f.Label != label {
			fields = append(fields, f)
		}
	}
	return VRecord{Fields: fields}
}

// RecordOverwrite replaces an existing field (same semantics as extend for our purposes).
func RecordOverwrite(label string, val Value, rec VRecord) VRecord {
	return RecordExtend(label, val, rec)
}

// Package ir defines the EYG intermediate representation.
// Each node type corresponds to a dag-json object with a "0" discriminant.
// Spec: https://github.com/crowdhailer/eyg-lang/tree/main/spec
package ir

// Node is any expression in the EYG IR.
type Node interface {
	irNode()
}

// Var references a variable. {"0":"v","l":"name"}
type Var struct{ Label string }

// Lambda introduces a binding. {"0":"f","l":"param","b":<body>}
type Lambda struct {
	Label string
	Body  Node
}

// Apply applies Fn to Arg. {"0":"a","f":<fn>,"a":<arg>}
type Apply struct {
	Fn  Node
	Arg Node
}

// Let binds a value. {"0":"l","l":"x","v":<value>,"t":<then>}
// Desugars to Apply(Lambda(l, t), v).
type Let struct {
	Label string
	Value Node
	Then  Node
}

// Int is an integer literal. {"0":"i","v":42}
type Int struct{ Value int64 }

// String is a string literal. {"0":"s","v":"hello"}
type String struct{ Value string }

// Binary is a raw byte sequence. {"0":"x","v":<dag-json bytes>}
type Binary struct{ Value []byte }

// Tail is the empty list (nil). {"0":"ta"}
type Tail struct{}

// Cons prepends an element to a list (curried). {"0":"c"}
type Cons struct{}

// Empty is the empty record {}. {"0":"u"}
type Empty struct{}

// Extend extends a record with a labeled field (curried). {"0":"e","l":"field"}
type Extend struct{ Label string }

// Overwrite replaces a field in a record (curried). {"0":"o","l":"field"}
type Overwrite struct{ Label string }

// Select extracts a field from a record (curried). {"0":"g","l":"field"}
type Select struct{ Label string }

// Tag wraps a value with a variant label (curried). {"0":"t","l":"Some"}
type Tag struct{ Label string }

// Case matches one variant arm. {"0":"m","l":"Ok"}
// Semantics: case(label)(handler)(otherwise)(scrutinee)
//   - if scrutinee.label == label → handler(scrutinee.value)
//   - else → otherwise(scrutinee)
type Case struct{ Label string }

// NoCases is the exhausted match arm — raises an error if reached. {"0":"n"}
type NoCases struct{}

// Perform raises an effect (curried: value → effect). {"0":"p","l":"EffectName"}
type Perform struct{ Label string }

// Handle installs a deep effect handler (reinstalls after each resume). {"0":"h","l":"EffectName"}
type Handle struct{ Label string }

// Shallow installs a one-shot effect handler (does not reinstall). {"0":"hs","l":"EffectName"}
type Shallow struct{ Label string }

// Builtin references a named primitive. {"0":"b","l":"int_add"}
type Builtin struct{ Label string }

// Reference is a content-addressed node (CID link). {"0":"#","l":{"/":"<cid>"}}
type Reference struct{ CID string }

// Release is a versioned package reference. {"0":"@","p":"pkg","r":1,"l":{"/":"<cid>"}}
type Release struct {
	Package string
	Release int64
	CID     string
}

// Vacant is a hole / TODO placeholder. {"0":"z","c":"comment"}
type Vacant struct{ Comment string }

func (Var) irNode()       {}
func (Lambda) irNode()    {}
func (Apply) irNode()     {}
func (Let) irNode()       {}
func (Int) irNode()       {}
func (String) irNode()    {}
func (Binary) irNode()    {}
func (Tail) irNode()      {}
func (Cons) irNode()      {}
func (Empty) irNode()     {}
func (Extend) irNode()    {}
func (Overwrite) irNode() {}
func (Select) irNode()    {}
func (Tag) irNode()       {}
func (Case) irNode()      {}
func (NoCases) irNode()   {}
func (Perform) irNode()   {}
func (Handle) irNode()    {}
func (Shallow) irNode()   {}
func (Builtin) irNode()   {}
func (Reference) irNode() {}
func (Release) irNode()   {}
func (Vacant) irNode()    {}

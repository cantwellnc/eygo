package eval

import "fmt"

// Result is the outcome of evaluating an EYG expression.
// Exactly one of Val, Effect, or Err is set.
type Result struct {
	Val    Value
	Effect *Effect
	Err    error
}

// Effect is an unhandled side-effect suspended mid-computation.
type Effect struct {
	Label  string
	Value  Value
	Resume func(Value) Result // continue the computation with a reply value
}

func Ok(v Value) Result       { return Result{Val: v} }
func Eff(e *Effect) Result    { return Result{Effect: e} }
func Err(err error) Result    { return Result{Err: err} }
func Errf(f string, a ...any) Result { return Result{Err: fmt.Errorf(f, a...)} }

package syntax

import (
	"bytes"
	"fmt"
	"strconv"
)

// Term is implemented by Atom, Integer, Float, String and Rule.
type Term interface {
	// Equals determines if two terms unify.
	Unify(Term) bool
	// A Term can be callable if it's bounded to a callable term.
	// While some examples are simple, such as Atoms or Facts, Variables
	// may sometimes be callable. For example:
	//
	//		likes(eric, beer).
	//		f1(likes(eric, beer)).
	//
	//		:- f1(X), X.
	//		X = likes(eric, beer).
	//
	Callable() *Compound
}

type Clause interface {
	// Call calls the given type with a set of arguments to match. Rules will
	// return a Compound type associated the body of that rule. If body is nil
	// and matches is true, the callable term has been matched and no further
	// action is needed.
	//
	// The caller owns the returned compound and may alter variables
	// however it chooses. Call should therefore create a copy of variables
	// before returning a match.
	Call(args []Term) (body *Goal, matches bool)

	// Signature returns the callable signature of the underlying type.
	// For example 'write/2'
	Signature() (functor Atom, nArgs int)
}

var (
	AnonVariable Term = &anonVariable{}
	Cut          Term = &cut{}
	EmptyList    Term = Atom("[]")
)

type anonVariable struct{}

func (*anonVariable) Unify(t2 Term) bool  { return true }
func (*anonVariable) Callable() *Compound { return nil }
func (*anonVariable) String() string      { return "_" }

type cut struct{}

func (*cut) Unify(t2 Term) bool  { return true }
func (*cut) Callable() *Compound { return nil }
func (*cut) String() string      { return "!" }

// Atom is a general-purpose name with no inherent meaning.
type Atom string

func (a Atom) Callable() *Compound { return nil }

func (a Atom) Unify(t Term) bool {
	switch t := t.(type) {
	case *Variable:
		return t.Unify(a)
	case Atom:
		return t == a
	}
	return false
}

func (a Atom) String() string { return string(a) }

// Integer aliases an interger type. It can be unified with other numeric types.
type Integer int

func (i Integer) Callable() *Compound { return nil }

func (i Integer) Unify(t Term) bool {
	switch t := t.(type) {
	case *Variable:
		return t.Unify(i)
	case Integer:
		return t == i
	case Float64:
		return Float64(i) == t
	}
	return false
}

func (i Integer) String() string { return strconv.Itoa(int(i)) }

// Float64 aliases a float64 type. It can be unified with other numeric types.
type Float64 float64

func (f Float64) Callable() *Compound { return nil }

func (f Float64) Unify(t Term) bool {
	switch t := t.(type) {
	case Integer:
		return f == Float64(t)
	case Float64:
		return t == f
	case *Variable:
		return t.Unify(f)
	}
	return false
}

func (f Float64) String() string {
	return strconv.FormatFloat(float64(f), 'G', -1, 64)
}

// Variable is a special term which can be set to other terms.
// It is bounded to terms through unification.
//
// Variables are not identified by address rather than name.
type Variable struct {
	name  string // only for debugging.
	value Term   // if nil, unset
}

func NewVariable(name string) *Variable { return &Variable{name, nil} }

func (v *Variable) String() string {
	return v.name
}

// Value returns the underlying bounded term, returning nil if not bounded.
// If the variable is bounded to another variable, it recursively returns
// the type of the bound to variable.
func (v *Variable) Value() Term {
	t := v.value
	for {
		v, ok := t.(*Variable)
		if !ok {
			return t
		}
		t = v.value
	}
}

func (v *Variable) Unify(t Term) (rv bool) {
	v2, isVar := t.(*Variable)
	if !isVar {
		if v.value == nil {
			v.value = t
			return true
		}
		return v.value.Unify(t)
	}

	// TODO: review this logic, prevent infinate loops
	if v == v2 {
		return true
	}
	if v.value == nil {
		v.value = v2
		return true
	}
	if v2.value == nil {
		v2.value = v
		return true
	}
	return v.value.Unify(v2.value)
}

func (v *Variable) Callable() *Compound {
	if v.value == nil {
		return nil
	}
	return v.value.Callable()
}

// Compound represents any term that is a functor with additional arguments.
type Compound struct {
	functor Atom
	args    []Term
}

func NewCompound(functor Atom, args ...Term) *Compound {
	return &Compound{functor, args}
}

func (c *Compound) Unify(t Term) bool {
	switch t := t.(type) {
	case *Variable:
		return t.Unify(c)
	case *Compound:
		if c.functor != t.functor || len(c.args) != len(t.args) {
			return false
		}
		for i, arg := range t.args {
			if !c.args[i].Unify(arg) {
				return false
			}
		}
		return true
	}
	return false
}

func (c *Compound) Callable() *Compound {
	return c
}

func (c *Compound) Signature() (functor Atom, nArgs int) {
	return c.functor, len(c.args)
}

func (c *Compound) Call(args []Term) (results *Goal, matches bool) {
	if len(c.args) != len(args) {
		return
	}

	for i, arg := range args {
		if !arg.Unify(c.args[i]) {
			return
		}
	}
	return nil, true
}

func (c *Compound) String() string {
	var b bytes.Buffer
	b.WriteString(string(c.functor))
	b.WriteString("(")
	for i, arg := range c.args {
		if i != 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%s", arg)
	}
	b.WriteString(")")
	return b.String()
}

// Goal is a comma separated list of terms.
//
// Goal does not implement Term or Callable.
type Goal struct {
	head Term // should never be nil
	tail *Goal
}

func NewGoal(head Term, tail ...Term) *Goal {
	comp := &Goal{head: head}
	c := comp
	for _, t := range tail {
		c.tail = &Goal{head: t}
		c = c.tail
	}
	return comp
}

func (g *Goal) String() string {
	var b bytes.Buffer
	goal := g
	for i := 0; goal != nil; i++ {
		if i != 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%s", goal.head)
		goal = goal.tail
	}
	b.WriteString(".")
	return b.String()
}

type Rule struct {
	functor Atom
	args    []Term
	body    *Goal
}

func NewRule(functor Atom, args []Term, body *Goal) *Rule {
	return &Rule{functor, args, body}
}

// cp creates a copy of a Rule, recursively replacing all Variables with
// unset ones.
func (r *Rule) cp() *Rule {
	vars := map[*Variable]*Variable{}

	// declare anonymous functions for recursive use.
	var cpTerm func(t Term) Term
	var cpTerms func(terms []Term) []Term

	cpTerm = func(t Term) Term {
		switch t := t.(type) {
		case *Variable:
			newval, ok := vars[t]
			if !ok {
				newval = &Variable{name: t.name}
				vars[t] = newval
			}
			return newval
		case *Compound:
			return &Compound{
				functor: t.functor,
				args:    cpTerms(t.args),
			}
		}
		return t
	}

	cpTerms = func(terms []Term) []Term {
		newTerms := make([]Term, len(terms))
		for i, t := range terms {
			newTerms[i] = cpTerm(t)
		}
		return newTerms
	}

	cp := Rule{functor: r.functor, args: cpTerms(r.args)}
	if r.body == nil {
		return &cp
	}
	cp.body = &Goal{cpTerm(r.body.head), nil}
	last := cp.body
	next := r.body.tail

	for next != nil {
		last.tail = &Goal{cpTerm(next.head), nil}
		last, next = next, next.tail
	}
	return &cp
}

func (r *Rule) Call(args []Term) (results *Goal, matches bool) {
	if len(args) != len(r.args) {
		return
	}
	// Because the r might hold Variables that are already bound, must
	// use cp to create an unset version of the rule.
	ruleCP := r.cp()
	for i, arg := range args {
		if !arg.Unify(ruleCP.args[i]) {
			return
		}
	}
	return ruleCP.body, true
}

func (r *Rule) Signature() (Atom, int) { return r.functor, len(r.args) }

func (r *Rule) String() string {
	var b bytes.Buffer
	b.WriteString(string(r.functor))
	b.WriteString("(")
	for i, arg := range r.args {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprint(&b, arg)
	}
	b.WriteString(")")
	b.WriteString(" :- ")
	fmt.Fprint(&b, r.body)
	return b.String()
}

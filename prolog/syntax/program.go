package syntax

import (
	"errors"
	"fmt"
)

type TypeErr struct {
	Exp  string
	Term Term
}

func (err *TypeErr) Error() string {
	return fmt.Sprintf("Type error: `%s` expected got `%s`", err.Exp, err.Term)
}

type sig struct {
	functor Atom
	nArgs   int
}

// Prog represents a Prolog program, a list of clauses
type Prog struct {
	running bool

	clauses map[sig][]Clause
}

func NewProg(caluses ...Clause) *Prog {
	prog := Prog{
		clauses: make(map[sig][]Clause),
	}
	for _, caluse := range caluses {
		prog.Add(caluse)
	}
	return &prog
}

// Add adds a clause to the list of clauses held by the program.
// It should not be called during the evaluation of a query.
func (p *Prog) Add(clause Clause) {
	if clause == nil {
		panic("syntax: clause cannot be nil")
	}
	functor, nArgs := clause.Signature()
	s := sig{functor, nArgs}
	p.clauses[s] = append(p.clauses[s], clause)
}

// match returns an ordered list of all clauses with signatures that match c.
// Clause is read only. The caller should not alter the values of the slice.
func (p *Prog) match(c Clause) []Clause {
	functor, nArgs := c.Signature()
	s := sig{functor, nArgs}
	if clauses := p.clauses[s]; clauses != nil {
		return clauses[:]
	}
	return []Clause{}
}

type Results struct {
	p   *Prog
	cp  *choicepoint
	err error // sticky error
}

// Close attempts to help the garbage collector by relinquish pointers to
// choicepoints.
func (r *Results) Close() {
	r.p = nil
	r.cp = nil
	if r.err == nil {
		r.err = errors.New("results closed")
	}
}

// Next advances the state of the evaluated query until either a match is found
// no more matches are possible, or an error was encountered during evaluation.
func (r *Results) Next() bool {
	if r.err != nil {
		return false
	}

	for r.cp != nil {

		// advance the choicepoint
		compound, match := r.cp.next()
		if !match {
			// if a match is not found, backtrack
			r.cp = r.cp.backtrack
			continue
		}

		// evaluate cuts and nil the backtracks
		for compound != nil && compound.head == Cut {
			r.cp.backtrack = nil
			compound = compound.tail
		}

		if compound == nil {
			// there are no more terms to evaluate, a match has been found
			return true
		}

		// construct a new choicepoint with the remaining compound to evaluate
		r.cp, r.err = r.p.choicepoint(compound, r.cp)
		if r.err != nil {
			return false
		}
	}
	return false
}

// Err returns the results stick error.
func (r *Results) Err() error { return r.err }

func (p *Prog) Query(c *Goal) *Results {
	choicepoint, err := p.choicepoint(c, nil)
	if err != nil {
		return &Results{err: err}
	}
	return &Results{
		p:  p,
		cp: choicepoint,
	}
}

// choicepoint returns a new choicepoint pointing to the list of rules.
func (p *Prog) choicepoint(c *Goal, backtrack *choicepoint) (*choicepoint, error) {

	if c == nil || c.head == nil {
		panic("syntax: Compound cannot be nil")
	}

	// evaluate the head of the compound for a callable term
	fact := c.head.Callable()
	if fact == nil {
		return nil, &TypeErr{"callable", c.head}
	}

	state := map[*Variable]Term{}
	visitVars(c, func(v *Variable) { state[v] = v.value })

	return &choicepoint{
		backtrack: backtrack,
		fact:      fact,
		remaining: c.tail,
		clauses:   p.match(fact),
		state:     state,
	}, nil
}

// choicepoint
type choicepoint struct {
	backtrack *choicepoint       // the choicepoint to backtrack to
	fact      *Compound          // fact to match
	remaining *Goal              // the remaining
	clauses   []Clause           // the set of matching clauses
	state     map[*Variable]Term // the beginning state of all variables
}

func (cp *choicepoint) pop() Clause {
	if len(cp.clauses) == 0 {
		return nil
	}
	var clause Clause
	clause, cp.clauses = cp.clauses[0], cp.clauses[1:]
	return clause
}

// next returns if a match has been made and if so the list of compounds
// remaining to evaluate.
// In the event of a rule match, the body is prepended to the choicepoints
// existing remaining compound.
func (cp *choicepoint) next() (c *Goal, match bool) {

	for clause := cp.pop(); clause != nil; clause = cp.pop() {
		cp.resetVars()

		result, matches := clause.Call(cp.fact.args)
		if !matches {
			continue
		}
		if result == nil {
			return cp.remaining, true
		}

		// append remaining to result
		tail := result
		for tail.tail != nil {
			tail = tail.tail
		}
		tail.tail = cp.remaining

		return result, true
	}
	return nil, false
}

func (cp *choicepoint) resetVars() {
	reset := func(v *Variable) { v.value = cp.state[v] }
	visitVarsTerm(cp.fact, reset)
	visitVars(cp.remaining, reset)
}

func visitVars(c *Goal, fn func(v *Variable)) {
	if c == nil {
		return
	}
	visitVarsTerm(c.head, fn)
	visitVars(c.tail, fn)
}

func visitVarsTerm(t Term, fn func(v *Variable)) {
	switch t := t.(type) {
	case *Variable:
		fn(t)
	case *Compound:
		for _, arg := range t.args {
			visitVarsTerm(arg, fn)
		}
	}
}

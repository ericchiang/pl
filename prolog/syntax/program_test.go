package syntax

import "testing"

func TestNewProgram(t *testing.T) {
	_ = NewProg()
}

func TestSimpleFact(t *testing.T) {
	p := NewProg()

	f := NewCompound(Atom("likes"), Atom("bob"), Atom("pizza"))
	p.Add(f)
	c := NewGoal(f)
	r := p.Query(c)
	nMatches := 0
	for r.Next() {
		nMatches++
	}
	if nMatches != 1 {
		t.Errorf("expected one match, got %d", nMatches)
	}
	if err := r.Err(); err != nil {
		t.Errorf("error during search: %v", err)
	}
}

func TestSimpleVariable(t *testing.T) {
	p := NewProg()

	x := NewVariable("X")

	f := NewCompound("likes", Atom("bob"), Atom("pizza"))
	p.Add(f)
	r := p.Query(NewGoal(NewCompound("likes", Atom("bob"), x)))
	nMatches := 0
	for r.Next() {
		nMatches++
	}
	if nMatches != 1 {
		t.Errorf("expected one match, got %d", nMatches)
	}
	if err := r.Err(); err != nil {
		t.Errorf("error during search: %v", err)
	}
	val := x.Value()
	if val == nil {
		t.Fatalf("expected x to be binded to a value")
	}
	a := Atom("pizza")
	if val != a {
		t.Fatalf("expected val to be %s got %s", a, val)
	}
}

type varExp map[*Variable]Term

func TestMultiMatch(t *testing.T) {
	clauses := []Clause{
		NewCompound("likes", Atom("eric"), Atom("shoes")),
		NewCompound("likes", Atom("bob"), Atom("pizza")),
		NewCompound("likes", Atom("eric"), Atom("bubblegum")),
		NewCompound("likes", Atom("bob"), Atom("beer")),
	}
	x := NewVariable("X")
	exp := []varExp{
		{x: Atom("pizza")},
		{x: Atom("beer")},
	}
	q := NewGoal(NewCompound("likes", Atom("bob"), x))
	testQuery(t, clauses, q, exp)
}

func TestSimpleRule(t *testing.T) {
	p1 := NewVariable("Person1")
	p2 := NewVariable("Person2")
	thing := NewVariable("Thing")
	clauses := []Clause{
		NewCompound("likes", Atom("eric"), Atom("pizza")),
		NewCompound("likes", Atom("bob"), Atom("pizza")),
		NewRule("friends", []Term{p1, p2},
			NewGoal(
				NewCompound("likes", p1, thing),
				NewCompound("likes", p2, thing),
			),
		),
	}
	x := NewVariable("X")
	y := NewVariable("Y")
	exp := []varExp{
		{x: Atom("eric"), y: Atom("eric")},
		{x: Atom("eric"), y: Atom("bob")},
		{x: Atom("bob"), y: Atom("eric")},
		{x: Atom("bob"), y: Atom("bob")},
	}
	q := NewGoal(NewCompound("friends", x, y))
	testQuery(t, clauses, q, exp)
}

func testQuery(t *testing.T, clauses []Clause, query *Goal, exp []varExp) {
	p := NewProg()
	for _, clause := range clauses {
		p.Add(clause)
	}

	nExp := len(exp)
	nResults := 0

	r := p.Query(query)
	for r.Next() {
		nResults++
		if len(exp) == 0 {
			t.Errorf("%s: expected %d results, got %d", query, nExp, nResults)
			return
		}

		var expR map[*Variable]Term
		expR, exp = exp[0], exp[1:]
		for v, expT := range expR {
			n := v.Value()
			if n == nil || !n.Unify(expT) {
				t.Errorf("%s result %d, expected %s to be %s got %s", query, nResults, v, expT, n)
			}
		}
	}
	if len(exp) != 0 {
		t.Errorf("%s: expected %d results, got %d", query, nExp, nResults)
	}
	if err := r.Err(); err != nil {
		t.Errorf("%s: failed %s", query, err)
	}
}

func TestChoicepoint(t *testing.T) {

	p1 := NewVariable("Person1")
	p2 := NewVariable("Person2")
	thing := NewVariable("Thing")

	f := NewRule("friends", []Term{p1, p2},
		NewGoal(
			NewCompound("likes", p1, thing),
			NewCompound("likes", p2, thing),
		),
	)
	clauses := []Clause{
		NewCompound("likes", Atom("eric"), Atom("pizza")),
		NewCompound("likes", Atom("bob"), Atom("pizza")),
		f,
	}
	x := NewVariable("X")
	y := NewVariable("Y")
	body, matches := f.Call([]Term{x, y})
	if !matches {
		t.Fatalf("expected to match")
		return
	}
	p := NewProg(clauses...)
	cp, err := p.choicepoint(body, nil)
	if err != nil {
		t.Fatal(err)
	}
	nMatches := 0
	for {
		comp, match := cp.next()
		if !match {
			break
		}
		cp, err := p.choicepoint(comp, cp)
		if err != nil {
			t.Fatal(err)
		}
		if len(cp.clauses) != 2 {
			t.Errorf("expected like to match 2 clauses, got %s", cp.clauses)
		}
		subMatches := 0
		for {
			if _, match := cp.next(); !match {
				break
			}
			subMatches++
		}
		if subMatches != 2 {
			t.Errorf("%s expected 2 matches got %d", comp, subMatches)
		}

		nMatches++
	}
	if nMatches != 2 {
		t.Fatalf("expected 2 matches got %d", nMatches)
	}
}

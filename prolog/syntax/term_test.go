package syntax

import "testing"

func testUnify(t1, t2 Term, should bool, t *testing.T) {
	if t1.Unify(t2) != should {
		if should {
			t.Errorf("%T(%s) does not unify with %T(%s)", t1, t1, t2, t2)
		} else {
			t.Errorf("did not expect %T(%s) to unify with %T(%s)", t1, t1, t2, t2)
		}
	}
}

func TestAtomUnify(t *testing.T) {
	testUnify(Atom("a"), Atom("a"), true, t)
	testUnify(Atom("a"), Atom("b"), false, t)
	testUnify(Atom("a"), Integer(1), false, t)
	testUnify(Atom("a"), Float64(1.), false, t)

	testUnify(Atom("foobar"), Atom("foobar"), true, t)
}

func TestNumberUnify(t *testing.T) {
	testUnify(Float64(1.), Float64(1.), true, t)
	testUnify(Float64(1.), Integer(1), true, t)
	testUnify(Float64(1.), Float64(1.0001), false, t)
}

func TestVariableUnify(t *testing.T) {
	x := NewVariable("X")
	testUnify(x, Integer(1), true, t)
	testUnify(x, Integer(2), false, t)
	testUnify(x, Integer(1), true, t)
	testUnify(Integer(1), x, true, t)

	y := NewVariable("Y")
	testUnify(y, Atom("foo"), true, t)
	testUnify(y, Atom("foo"), true, t)
}

func TestFactUnify(t *testing.T) {
	t1 := &Compound{
		functor: "f1",
		args:    []Term{Atom("foo"), Atom("bar")},
	}
	x := NewVariable("X")
	testUnify(x, Atom("bar"), true, t)

	t2 := &Compound{
		functor: "f1",
		args:    []Term{Atom("foo"), x},
	}
	testUnify(t1, t2, true, t)
}

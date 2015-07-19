package builtin

import "github.com/ericchiang/pl/prolog/syntax"

// Type checking for variables, see http://www.swi-prolog.org/pldoc/man?section=typetest

var Var1 syntax.Clause = &builtin{
	name:  "var",
	nArgs: 1,
	call: func(args []syntax.Term) (*syntax.Goal, bool) {
		matches := false
		if len(args) == 1 {
			_, matches = args[1].(*syntax.Variable)
		}
		return nil, matches
	},
}

var Nonvar1 syntax.Clause = &builtin{
	name:  "nonvar",
	nArgs: 1,
	call: func(args []syntax.Term) (*syntax.Goal, bool) {
		matches := false
		if len(args) == 1 {
			_, matches = args[1].(*syntax.Variable)
			matches = !matches
		}
		return nil, matches
	},
}

var Integer1 syntax.Clause = &builtin{
	name:  "integer",
	nArgs: 1,
	call: func(args []syntax.Term) (*syntax.Goal, bool) {
		matches := false
		if len(args) == 1 {
			_, matches = args[1].(syntax.Integer)
		}
		return nil, matches
	},
}

var Float1 syntax.Clause = &builtin{
	name:  "float",
	nArgs: 1,
	call: func(args []syntax.Term) (*syntax.Goal, bool) {
		matches := false
		if len(args) == 1 {
			_, matches = args[1].(syntax.Float64)
		}
		return nil, matches
	},
}

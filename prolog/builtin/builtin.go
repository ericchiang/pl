package builtin

import (
	"fmt"

	"github.com/ericchiang/pl/prolog/syntax"
)

type builtin struct {
	name  string
	nArgs int
	call  func(arg []syntax.Term) (*syntax.Goal, bool)
}

func (b *builtin) Signature() (syntax.Atom, int) {
	return syntax.Atom(b.name), b.nArgs
}

func (b *builtin) Call(args []syntax.Term) (*syntax.Goal, bool) {
	return b.call(args)
}

func (b *builtin) String() string {
	return fmt.Sprintf("%s/%d", b.name, b.nArgs)
}

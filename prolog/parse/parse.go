package parse

type Op struct {
	Prec int // Operator precidence
}

type OpPattern string

const (
	OpInLeftAssoc  OpPattern = "yfx"
	OpInRightAssoc OpPattern = "xfy"
	OpInNonAssoc   OpPattern = "xfx" // 'is', '<'
	OpPreAsso      OpPattern = "fy"  // - (i.e., - - 5 allowed)
	OpPreNonAssoc  OpPattern = "fx"  // :- (i.e., :- :- goal not allowed)
	OpPostAssoc    OpPattern = "yf"
)

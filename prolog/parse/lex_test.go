package parse

import "testing"

func TestErrors(t *testing.T) {
	// these are all strings which should return lex errors
	tests := []string{
		"foobar",
		"foobar())",
		"'foobar",
	}
	for _, test := range tests {
	}
}

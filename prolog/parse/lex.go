package parse

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type itemType int

const (
	itemAtom       itemType = iota
	itemComma               // ','
	itemCut                 // '!'
	itemDot                 // '.'
	itemNumber              // '6', '2.1'
	itemLeftBrace           // '['
	itemLeftParen           // '('
	itemPipe                // '|'
	itemQuoted              // a quoted atom
	itemRightBrace          // ']'
	itemRightParen          // ')'
	itemString
	itemVariable
	itemEOF
	itemError
)

type item struct {
	typ itemType
	pos int
	val string
}

const eof rune = -1

type stateFn func(*lexer) stateFn

// lexer holds the state of the scanner.
type lexer struct {
	name       string    // the name of the input; used only for error reports
	input      string    // the string being scanned
	state      stateFn   // the next lexing function to enter
	pos        int       // current position in the input
	start      int       // start position of this item
	width      int       // width of last rune read from input
	lastPos    int       // position of most recent item returned by nextItem
	items      chan item // channel of scanned items
	parenDepth int       // nesting depth of ( ) exprs
	braceDepth int       // nesting depth of [ ] exprs
}

// nextItem returns the next item from the input.
func (l *lexer) nextItem() item {
	item := <-l.items
	l.lastPos = item.pos
	return item
}

// next returns the next rune in the input.
func (l *lexer) next() rune {
	if int(l.pos) >= len(l.input) {
		l.width = 0
		return eof
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.width = w
	l.pos += l.width
	return r
}

// peek returns but does not consume the next rune in the input.
func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

// backup steps back one rune. Can only be called once per call of next.
func (l *lexer) backup() {
	l.pos -= l.width
}

// emit passes an item back to the client.
func (l *lexer) emit(t itemType) {
	l.items <- item{t, l.start, l.input[l.start:l.pos]}
	l.start = l.pos
}

// ignore skips over the pending input before this point.
func (l *lexer) ignore() {
	l.start = l.pos
}

// accept consumes the next rune if it's from the valid set.
func (l *lexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.next()) >= 0 {
		return true
	}
	l.backup()
	return false
}

// acceptRun consumes a run of runes from the valid set.
func (l *lexer) acceptRun(valid string) {
	for strings.IndexRune(valid, l.next()) >= 0 {
	}
	l.backup()
}

// lineNumber reports which line we're on, based on the position of
// the previous item returned by nextItem. Doing it this way
// means we don't have to worry about peek double counting.
func (l *lexer) lineNumber() int {
	return 1 + strings.Count(l.input[:l.lastPos], "\n")
}

// errorf returns an error token and terminates the scan by passing
// back a nil pointer that will be the next state, terminating l.nextItem.
func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.items <- item{itemError, l.start, fmt.Sprintf(format, args...)}
	return nil
}

// isSpace reports whether r is a space character.
func isSpace(r rune) bool {
	return r == ' ' || r == '\t'
}

// isEndOfLine reports whether r is an end-of-line character.
func isEndOfLine(r rune) bool {
	return r == '\r' || r == '\n'
}

// isAlphaNumeric reports whether r
func isAlphaNumeric(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func isSpecial(r rune) bool {
	return strings.IndexRune(`\+-*=<>:&`, r) > -1
}

// lexNext lexes the item immediately following an identifier
func lexNext(l *lexer) stateFn {
	r := l.next()
	switch {
	case r == eof:
		return l.errorf("statement unterminated by '.'")
	case r == '.':
		if l.peek() == '(' {
			l.emit(itemAtom)
		} else {
			l.emit(itemDot)
			return lexSpace
		}
	case r == '|':
		l.emit(itemPipe)
	case r == '!':
		l.emit(itemCut)
	case r == ',':
		l.emit(itemComma)
	case r == '(':
		l.emit(itemLeftParen)
		l.parenDepth++
	case r == ')':
		l.emit(itemRightParen)
		l.parenDepth--
		if l.parenDepth < 0 {
			return l.errorf("unexpected right paren %#U", r)
		}
	case r == '[':
		l.emit(itemLeftBrace)
		l.braceDepth++
	case r == ']':
		l.emit(itemRightParen)
		l.parenDepth--
		if l.parenDepth < 0 {
			return l.errorf("unexpected right paren %#U", r)
		}
	case unicode.IsDigit(r):
		return lexNumber
	case unicode.IsUpper(r) || r == '_':
		return lexVariable
	case unicode.IsLower(r):
		return lexAtom
	case r == '\'' || r == '"':
		l.backup()
		return lexQuoted
	case unicode.IsDigit(r):
		l.backup()
		return lexNumber
	default:
		l.errorf("unexpected character %#U", r)
	}
	return lexNext
}

// lexAtom lexes an atom which consists of alphanumeric characters
// It assumes the first character has already been seen
func lexAtom(l *lexer) stateFn {
	for {
		r := l.peek()
		if !isAlphaNumeric(r) && r != '_' {
			l.emit(itemAtom)
			return lexNext
		}
	}
}

func lexVariable(l *lexer) stateFn {
	for {
		r := l.peek()
		if !isAlphaNumeric(r) && r != '_' {
			l.emit(itemVariable)
			return lexNext
		}
	}
}

// lexAtomSpecial lexes and atom which consists of special characters.
// It assumes the first character has already been seen
func lexAtomSpecial(l *lexer) stateFn {
	for {
		if !isSpecial(l.peek()) {
			l.emit(itemAtom)
			return lexNext
		}
	}
	return nil
}

// lexNumber lexes a number with an optional single dot.
// It assumes the first digit has already been seen.
func lexNumber(l *lexer) stateFn {
	seenDot := false

loop:
	for {
		r := l.peek()
		switch {
		case r == '.':
			if seenDot {
				break loop
			}
			seenDot = true
		case unicode.IsDigit(r):
		default:
			break loop
		}
		l.next()
	}
	l.emit(itemNumber)
	return nil
}

func lexQuoted(l *lexer) stateFn {
	quoteChar := l.next()
	if quoteChar != '\'' && quoteChar != '"' {
		l.errorf("unexpected quote char %#U", quoteChar)
	}
	for {
		r := l.next()
		switch r {
		case eof:
			l.errorf("unterminated quote %#U", quoteChar)
		case '\\':
			// handling of the unquote error whill be done elsewhere
			l.next()
		case quoteChar:
			l.emit(itemQuoted)
			return lexNext
		}
	}
	return nil
}

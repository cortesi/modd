package conf

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const spaces = " \t\n"
const quotes = `'"`

// Characters we don't allow in bare strings
const bareStringDisallowed = "{}#\n" + spaces + quotes + ":"

// itemType identifies the type of lex items.
type itemType int

const (
	itemBareString itemType = iota
	itemColon
	itemCommand
	itemComment
	itemDaemon
	itemError // error occurred; value is text of error
	itemEOF
	itemExclude
	itemLeftParen
	itemQuotedString
	itemPrep
	itemRightParen
	itemSpace
)

func (i itemType) String() string {
	switch i {
	case itemBareString:
		return "barestring"
	case itemColon:
		return "colon"
	case itemCommand:
		return "command"
	case itemComment:
		return "comment"
	case itemDaemon:
		return "daemon"
	case itemError:
		return "error"
	case itemEOF:
		return "eof"
	case itemExclude:
		return "exclude"
	case itemLeftParen:
		return "lparen"
	case itemPrep:
		return "prep"
	case itemQuotedString:
		return "quotedstring"
	case itemRightParen:
		return "rparen"
	case itemSpace:
		return "space"
	default:
		panic("unreachable")
	}
}

// Pos represents a byte position in the original input text from which
// this template was parsed.
type Pos int

// stateFn represents the state of the scanner as a function that returns the next state.
type stateFn func(*lexer) stateFn

const eof = -1

// item represents a token or text string returned from the scanner.
type item struct {
	typ itemType // The type of this item.
	pos Pos      // The starting position, in bytes, of this item in the input string.
	val string   // The value of this item.
}

// lexer holds the state of the scanner.
type lexer struct {
	name    string    // the name of the input; used only for error reports
	input   string    // the string being scanned
	state   stateFn   // the next lexing function to enter
	pos     Pos       // current position in the input
	start   Pos       // start position of this item
	width   Pos       // width of last rune read from input
	lastPos Pos       // position of most recent item returned by nextItem
	items   chan item // channel of scanned items
}

func (l *lexer) current() string {
	return l.input[l.start:l.pos]
}

// next returns the next rune in the input.
func (l *lexer) next() rune {
	if int(l.pos) >= len(l.input) {
		l.width = 0
		return eof
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.width = Pos(w)
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
	l.items <- item{t, l.start, l.current()}
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

// acceptFunc accepts a run of characters based on a match function
func (l *lexer) acceptFunc(match func(rune) bool) {
	for match(l.peek()) {
		l.next()
	}
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

// nextItem returns the next item from the input.
func (l *lexer) nextItem() item {
	item := <-l.items
	l.lastPos = item.pos
	return item
}

// nextSignificantItem returns the next significant item from the input,
// ignoring comments and spaces.
func (l *lexer) nextSignificantItem() item {
	for {
		item := l.nextItem()
		switch item.typ {
		case itemSpace:
			continue
		case itemComment:
			continue
		default:
			return item
		}
	}
}

// lex creates a new scanner for the input string.
func lex(name, input string) *lexer {
	l := &lexer{
		name:  name,
		input: input,
		items: make(chan item),
	}
	go l.run()
	return l
}

// run runs the state machine for the lexer.
func (l *lexer) run() {
	for l.state = lexTop; l.state != nil; {
		l.state = l.state(l)
	}
}

// acceptBareString accepts a bare, unquoted string
func (l *lexer) acceptBareString() {
	l.acceptFunc(
		func(r rune) bool {
			return !any(r, bareStringDisallowed) && r != eof
		},
	)
}

// acceptLine accepts the remainder of a line
func (l *lexer) acceptLine() {
	l.acceptFunc(
		func(r rune) bool {
			return r != '\n' && r != eof
		},
	)
	l.accept("\n")
}

// acceptQuotedString accepts a quoted string
func (l *lexer) acceptQuotedString(quote rune) error {
Loop:
	for {
		switch l.next() {
		case '\\':
			if r := l.next(); r != eof {
				break
			}
			fallthrough
		case eof:
			return fmt.Errorf("unterminated quoted string")
		case quote:
			break Loop
		}
	}
	return nil
}

func any(r rune, s string) bool {
	return strings.IndexRune(s, r) >= 0
}

// stateFns
func lexTop(l *lexer) stateFn {
	for {
		n := l.next()
		if n == '#' {
			l.acceptLine()
			l.emit(itemComment)
		} else if any(n, quotes) {
			err := l.acceptQuotedString(n)
			if err != nil {
				l.errorf("%s", err)
				return nil
			}
			l.emit(itemQuotedString)
		} else if n == '{' {
			l.emit(itemLeftParen)
			return lexInside
		} else if n == eof {
			l.emit(itemEOF)
			return nil
		} else if any(n, spaces) {
			l.acceptRun(spaces)
			l.emit(itemSpace)
		} else if !any(n, bareStringDisallowed) {
			l.acceptBareString()
			l.emit(itemBareString)
		} else {
			return l.errorf("invalid input")
		}
	}
}

func lexInside(l *lexer) stateFn {
	for {
		n := l.next()
		if n == '#' {
			l.acceptLine()
			l.emit(itemComment)
		} else if any(n, quotes) {
			err := l.acceptQuotedString(n)
			if err != nil {
				l.errorf("%s", err)
				return nil
			}
			l.emit(itemQuotedString)
		} else if n == '}' {
			l.emit(itemRightParen)
			return lexTop
		} else if n == '{' {
			return l.errorf("unterminated block")
		} else if n == eof {
			return l.errorf("unterminated block")
		} else if any(n, spaces) {
			l.acceptRun(spaces)
			l.emit(itemSpace)
		} else if !any(n, bareStringDisallowed) {
			l.acceptBareString()
			switch l.current() {
			case "exclude":
				l.emit(itemExclude)
				return lexCommand
			case "daemon":
				l.emit(itemDaemon)
				return lexCommand
			case "prep":
				l.emit(itemPrep)
				return lexCommand
			default:
				l.errorf("Unexpected directive in block: %s", l.current())
				return nil
			}
		} else {
			return l.errorf("invalid input")
		}
	}
}

// lexCommand lexes a single command. Commands can either be unquoted and on a
// single line, or quoted and span multiple lines.
func lexCommand(l *lexer) stateFn {
	colonseen := false
	for {
		n := l.next()
		if n == ':' {
			colonseen = true
			l.emit(itemColon)
		} else if any(n, spaces) {
			l.acceptRun(spaces)
			l.emit(itemSpace)
		} else {
			if colonseen {
				if any(n, quotes) {
					err := l.acceptQuotedString(n)
					if err != nil {
						l.errorf("%s", err)
						return nil
					}
					l.emit(itemCommand)
				} else {
					l.acceptLine()
					l.emit(itemCommand)

				}
				return lexInside
			}
			l.errorf("unexpected character")
			return nil
		}
	}
}

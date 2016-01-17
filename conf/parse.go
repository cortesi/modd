package conf

import (
	"fmt"
	"runtime"
	"strings"
)

type parser struct {
	name   string
	text   string
	lex    *lexer
	config *Config

	peekItem *item
}

// next returns the next token.
func (p *parser) next() item {
	if p.peekItem != nil {
		itm := *p.peekItem
		p.peekItem = nil
		return itm
	}
	nxt := p.lex.nextSignificantItem()
	if nxt.typ == itemError {
		p.errorf("%s", nxt.val)
	}
	return nxt
}

func anyType(t itemType, allowed []itemType) bool {
	for _, i := range allowed {
		if t == i {
			return true
		}
	}
	return false
}

func (p *parser) mustNext(allowed ...itemType) item {
	nxt := p.next()
	if !anyType(nxt.typ, allowed) {
		p.errorf("invalid syntax")
	}
	return nxt
}

// peek returns but does not consume the next token.
func (p *parser) peek() item {
	if p.peekItem == nil {
		itm := p.lex.nextSignificantItem()
		p.peekItem = &itm
	}
	return *p.peekItem
}

func (p *parser) collect(types ...itemType) []item {
	itms := []item{}
	for {
		nxt := p.peek()
		if anyType(nxt.typ, types) {
			itms = append(itms, p.next())
		} else {
			break
		}
	}
	return itms
}

func (p *parser) collectValues(types ...itemType) []string {
	items := p.collect(types...)
	ret := make([]string, len(items))
	for i, v := range items {
		ret[i] = v.val
	}
	return ret
}

// Collects an arbitrary number of patterns, and returns a (watch, exclude,
// NoCommonFilter) tuple.
func (p *parser) collectPatterns() ([]string, []string, bool) {
	noCommonFilter := false
	watch := []string{}
	exclude := []string{}

	vals := p.collect(itemBareString, itemQuotedString)
	for _, v := range vals {
		switch v.typ {
		case itemBareString:
			if v.val[0] == '!' {
				exclude = append(exclude, v.val[1:])
			} else {
				if v.val == "+common" {
					noCommonFilter = true
				} else {
					watch = append(watch, v.val)
				}
			}
		case itemQuotedString:
			if v.val[0] == '!' {
				exclude = append(exclude, v.val[2:len(v.val)-1])
			} else {
				watch = append(watch, v.val[1:len(v.val)-1])
			}
		}
	}
	if len(watch) == 0 {
		watch = nil
	}
	if len(exclude) == 0 {
		exclude = nil
	}
	return watch, exclude, noCommonFilter
}

// errorf formats the error and terminates processing.
func (p *parser) errorf(format string, args ...interface{}) {
	p.config = nil
	format = fmt.Sprintf("%s:%d: %s", p.name, p.lex.lineNumber(), format)
	panic(fmt.Errorf(format, args...))
}

func (p *parser) stopParse() {
	p.lex = nil
}

// recover is the handler that turns panics into returns from the top level of
// Parse.
func (p *parser) recover(errp *error) {
	e := recover()
	if e != nil {
		if _, ok := e.(runtime.Error); ok {
			panic(e)
		}
		if p != nil {
			p.stopParse()
		}
		*errp = e.(error)
	}
	return
}

func (p *parser) parse() (err error) {
	defer p.recover(&err)
	p.lex = lex(p.name, p.text)
	p.config = &Config{}
	for {
		if p.peek().typ == itemEOF {
			break
		}
		p.config.addBlock(*p.parseBlock())
	}
	return err
}

func prepCommand(itm item) string {
	val := itm.val
	if itm.typ == itemQuotedString {
		val = val[1 : len(val)-1]
		val = strings.Replace(val, "\n", " ", -1)
	}
	return strings.TrimSpace(val)
}

func (p *parser) parseBlock() *Block {
	block := &Block{}
	block.Include, block.Exclude, block.NoCommonFilter = p.collectPatterns()
	nxt := p.next()
	if nxt.typ != itemLeftParen {
		p.errorf("expected block open parentheses, got %q", nxt.val)
	}
Loop:
	for {
		nxt = p.next()
		switch nxt.typ {
		case itemDaemon:
			options := p.collectValues(itemBareString)
			p.mustNext(itemColon)
			err := block.addDaemon(
				prepCommand(p.mustNext(itemBareString, itemQuotedString)),
				options,
			)
			if err != nil {
				p.errorf("%s", err)
			}
		case itemPrep:
			options := p.collectValues(itemBareString)
			p.mustNext(itemColon)
			err := block.addPrep(
				prepCommand(p.mustNext(itemBareString, itemQuotedString)),
				options,
			)
			if err != nil {
				p.errorf("%s", err)
			}
		case itemRightParen:
			break Loop
		default:
			p.errorf("unexpected input: %s", nxt.val)
		}
	}
	return block
}

// Parse parses a string, and returns a completed Config
func Parse(name string, text string) (*Config, error) {
	p := &parser{name: name, text: text}
	err := p.parse()
	if err != nil {
		return nil, err
	}
	return p.config, nil
}

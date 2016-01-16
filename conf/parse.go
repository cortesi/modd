package conf

import (
	"fmt"
	"runtime"
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
		panic("invalid token type")
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

// Collects an arbitrary number of patterns, and returns a []Pattern,
// NoCommonFilter tuple.
func (p *parser) collectPatterns() ([]Pattern, bool) {
	noCommonFilter := false
	vals := p.collect(itemBareString, itemQuotedString)
	ret := []Pattern{}
	for _, v := range vals {
		var val *Pattern
		switch v.typ {
		case itemBareString:
			if v.val[0] == '!' {
				val = &Pattern{
					Spec:   v.val[1:],
					Filter: true,
				}
			} else {
				if v.val == "+common" {
					noCommonFilter = true
				} else {
					val = &Pattern{Spec: v.val}
				}
			}
		case itemQuotedString:
			if v.val[0] == '!' {
				val = &Pattern{
					Spec:   v.val[2 : len(v.val)-1],
					Filter: true,
				}
			} else {
				val = &Pattern{Spec: v.val[1 : len(v.val)-1]}
			}
		}
		if val != nil {
			ret = append(ret, *val)
		}
	}
	if len(ret) > 0 {
		return ret, noCommonFilter
	}
	return nil, noCommonFilter
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

func (p *parser) parseBlock() *Block {
	block := &Block{}
	block.Patterns, block.NoCommonFilter = p.collectPatterns()
	nxt := p.next()
	if nxt.typ != itemLeftParen {
		p.errorf("expected block open parentheses, got %q", nxt.val)
	}
Loop:
	for {
		nxt = p.next()
		switch nxt.typ {
		case itemDaemon:
			block.addDaemon(p.mustNext(itemBareString, itemQuotedString).val)
		case itemPrep:
			block.addPrep(p.mustNext(itemBareString, itemQuotedString).val)
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

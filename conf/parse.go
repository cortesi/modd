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

// peek returns but does not consume the next token.
func (p *parser) peek() item {
	if p.peekItem == nil {
		itm := p.lex.nextSignificantItem()
		p.peekItem = &itm
	}
	return *p.peekItem
}

func anyType(t itemType, allowed []itemType) bool {
	for _, i := range allowed {
		if t == i {
			return true
		}
	}
	return false
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

func (p *parser) collectStrings() []string {
	vals := p.collect(itemBareString, itemQuotedString)
	ret := make([]string, len(vals))
	for i, v := range vals {
		switch v.typ {
		case itemBareString:
			ret[i] = v.val
		case itemQuotedString:
			ret[i] = v.val[1 : len(v.val)-1]
		}
	}
	if len(ret) > 0 {
		return ret
	}
	return nil
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
	blocks := []Block{}
	for {
		if p.peek().typ == itemEOF {
			break
		}
		blocks = append(blocks, *p.parseBlock())
	}
	if len(blocks) > 0 {
		p.config.Blocks = blocks
	}
	return err
}

func (p *parser) parseBlock() *Block {
	block := &Block{
		Patterns: p.collectStrings(),
	}

	nxt := p.next()
	if nxt.typ != itemLeftParen {
		p.errorf("expected block open parentheses")
	}

	nxt = p.next()
	switch nxt.typ {
	case itemRightParen:
		break
	default:
		p.errorf("unexpected input: %s", nxt.val)
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

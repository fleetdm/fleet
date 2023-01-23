//go:build darwin
// +build darwin

package remotectl

import (
	"bufio"
	"io"
)

type parser struct {
	scanner      *bufio.Scanner
	lastReadLine string
}

var Parser = New()

func New() *parser {
	return &parser{}
}

func (p *parser) Parse(reader io.Reader) (any, error) {
	return p.parseDumpstate(reader)
}

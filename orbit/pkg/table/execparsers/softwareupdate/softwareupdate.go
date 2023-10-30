//go:build darwin
// +build darwin

package softwareupdate

import (
	"bufio"
	"io"
)

type parser struct {
	scanner *bufio.Scanner
}

var Parser = New()

func New() *parser {
	return &parser{}
}

func (p *parser) Parse(reader io.Reader) (any, error) {
	return p.parseSoftwareupdate(reader)
}

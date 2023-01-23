package dsregcmd

import (
	"io"
)

type parser struct{}

// Parser is a parser for dsregcmd output
var Parser = New()

func New() parser {
	return parser{}
}

func (p parser) Parse(reader io.Reader) (any, error) {
	return parseDsreg(reader)
}

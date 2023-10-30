package pacman_info

import (
	"io"
)

type parser struct{}

var Parser = New()

func New() parser {
	return parser{}
}

func (p parser) Parse(reader io.Reader) (any, error) {
	return pacmanParse(reader)
}

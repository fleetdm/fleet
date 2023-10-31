// based on github.com/kolide/launcher/pkg/osquery/tables
package dnf

import (
	"io"
)

type parser struct{}

var Parser = New()

func New() parser {
	return parser{}
}

func (p parser) Parse(reader io.Reader) (any, error) {
	return dnfParse(reader)
}

// based on github.com/kolide/launcher/pkg/osquery/tables
package apt

import (
	"io"
)

type parser struct{}

var Parser = New()

func New() parser {
	return parser{}
}

func (p parser) Parse(reader io.Reader) (any, error) {
	return aptParse(reader)
}

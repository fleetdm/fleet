// based on github.com/kolide/launcher/pkg/osquery/tables
package repcli

import (
	"io"
)

type parser struct{}

func (p parser) Parse(reader io.Reader) (any, error) {
	return repcliParse(reader)
}

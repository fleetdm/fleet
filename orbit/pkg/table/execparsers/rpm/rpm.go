// based on github.com/kolide/launcher/pkg/osquery/tables
package rpm

import (
	"io"
)

type parser struct{}

func (p parser) Parse(reader io.Reader) (any, error) {
	return rpmParse(reader)
}

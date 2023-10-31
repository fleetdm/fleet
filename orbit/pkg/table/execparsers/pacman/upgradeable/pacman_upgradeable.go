// based on github.com/kolide/launcher/pkg/osquery/tables
package pacman_upgradeable

import (
	"io"
)

type parser struct{}

func (p parser) Parse(reader io.Reader) (any, error) {
	return pacmanParse(reader)
}

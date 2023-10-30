package dataflatten

import (
	"strings"
)

// Row is the record type we return.
type Row struct {
	Path  []string
	Value string
}

// NewRow does a copy of the path elements, and returns a row. We do
// this copy to correct for some odd pointer passing bugs
func NewRow(path []string, value string) Row {
	copiedPath := make([]string, len(path))
	copy(copiedPath, path)
	return Row{
		Path:  copiedPath,
		Value: value,
	}
}

func (r Row) StringPath(sep string) string {
	return strings.Join(r.Path, sep)
}

func (r Row) ParentKey(sep string) (string, string) {
	switch len(r.Path) {
	case 0:
		return "", ""
	case 1:
		return "", r.Path[0]
	}

	parent := strings.Join(r.Path[:len(r.Path)-1], sep)
	key := r.Path[len(r.Path)-1]

	return parent, key
}

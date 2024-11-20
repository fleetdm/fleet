package test

import (
	"bytes"
	"io"

	"github.com/groob/plist"
)

// PlistReader encodes v to XML Plist.
func PlistReader(v interface{}) (io.Reader, error) {
	buf := new(bytes.Buffer)
	enc := plist.NewEncoder(buf)
	enc.Indent("\t")
	return buf, enc.Encode(v)
}

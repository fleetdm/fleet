package log

import (
	"io"
	"os"
)

var output = io.Writer(os.Stderr)

func SetOutput(w io.Writer) {
	output = w
}

package log

import (
	"bufio"
	"io"
	"os"
)

var output = bufio.NewWriter(os.Stderr)

func SetOutput(w io.Writer) {
	output = bufio.NewWriter(w)
}

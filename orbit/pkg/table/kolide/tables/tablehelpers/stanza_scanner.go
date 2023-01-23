package tablehelpers

import (
	"regexp"
)

var (
	twoEOLs = regexp.MustCompile(`(\r?\n){2}`)
)

// StanzaSplitter implements the bufio.SplitFunc type, when used as the Split
// function for a bufio.Scanner, it will return chunks of bytes separated by an
// empty newline, e.g. \n\n or \r\n\r\n.
func StanzaSplitter(data []byte, atEOF bool) (int, []byte, error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if loc := twoEOLs.FindIndex(data); loc != nil && loc[0] >= 0 {
		s := data[0:loc[0]]
		return loc[1], s, nil
	}

	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}

	// Request more data.
	return 0, nil, nil
}

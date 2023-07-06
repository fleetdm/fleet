package file

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckPDF(t *testing.T) {
	testCases := []struct {
		in     []byte
		outErr string
	}{
		{[]byte{}, ErrInvalidType.Error()},
		{[]byte("--"), ErrInvalidType.Error()},
		{[]byte("invalid"), ErrInvalidType.Error()},
		{[]byte("%"), ErrInvalidType.Error()},
		{[]byte("%P"), ErrInvalidType.Error()},
		{[]byte("%PD"), ErrInvalidType.Error()},
		{[]byte("%PDF"), ""},
		{[]byte("%PDF-"), ""},
		{[]byte("%PDF-1"), ""},
		{[]byte("%PDF-2"), ""},
	}

	for _, c := range testCases {
		r := bytes.NewReader(c.in)
		err := CheckPDF(r)
		if c.outErr != "" {
			require.ErrorContains(t, err, c.outErr)
		} else {
			require.NoError(t, err)
		}
	}
}

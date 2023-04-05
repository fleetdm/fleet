package file

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckPKGSignature(t *testing.T) {
	read := func(name string) []byte {
		b, err := os.ReadFile(name)
		require.NoError(t, err)
		return b
	}
	testCases := []struct {
		in  []byte
		out error
	}{
		{in: []byte{}, out: io.EOF},
		{
			in:  read("./testdata/invalid.tar.gz"),
			out: ErrInvalidType,
		},
		{
			in:  read("./testdata/unsigned.pkg"),
			out: ErrNotSigned,
		},
		{
			in:  read("./testdata/signed.pkg"),
			out: nil,
		},
		{
			out: errors.New("decompressing TOC: unexpected EOF"),
			in:  read("./testdata/wrong-toc.pkg"),
		},
	}

	for _, c := range testCases {
		r := bytes.NewReader(c.in)
		err := CheckPKGSignature(r)
		if c.out != nil {
			require.ErrorContains(t, err, c.out.Error())
		} else {
			require.NoError(t, err)
		}
	}
}

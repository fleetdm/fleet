package appmanifest

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/micromdm/micromdm/mdm/appmanifest"
	"github.com/stretchr/testify/require"
)

var errTest = errors.New("test error")

type alwaysFailReader struct{}

func (alwaysFailReader) Read(p []byte) (n int, err error) {
	return 0, errTest
}

func TestCreate(t *testing.T) {
	url := "https://test.example.com"

	cases := []struct {
		in  io.Reader
		out *appmanifest.Manifest
		err error
	}{
		{
			in: strings.NewReader("foo"),
			out: &appmanifest.Manifest{
				ManifestItems: []appmanifest.Item{{Assets: []appmanifest.Asset{{
					Kind:       "software-package",
					SHA256Size: 32,
					SHA256s:    []string{"2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae"},
					URL:        "https://test.example.com",
				}}}}},
			err: nil,
		},
		{
			in:  alwaysFailReader{},
			out: nil,
			err: errTest,
		},
	}

	for _, c := range cases {
		m, err := Create(c.in, url)
		require.Equal(t, c.out, m)
		require.Equal(t, c.err, err)
	}
}

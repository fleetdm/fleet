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

func TestNew(t *testing.T) {
	url := "https://test.example.com"

	cases := []struct {
		in  io.Reader
		out *Manifest
		err error
	}{
		{
			in: strings.NewReader("foo"),
			out: &Manifest{
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
		m, err := New(c.in, url)
		require.Equal(t, c.out, m)
		require.Equal(t, c.err, err)
	}
}

func TestNewPlist(t *testing.T) {
	url := "https://test.example.com"
	cases := []struct {
		in  io.Reader
		out string
		err error
	}{
		{
			in: strings.NewReader("foo"),
			out: `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>items</key>
  <array>
    <dict>
      <key>assets</key>
      <array>
        <dict>
          <key>kind</key>
          <string>software-package</string>
          <key>sha256-size</key>
          <integer>32</integer>
          <key>sha256s</key>
          <array>
            <string>2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae</string>
          </array>
          <key>url</key>
          <string>https://test.example.com</string>
        </dict>
      </array>
    </dict>
  </array>
</dict>
</plist>
`,
			err: nil,
		},
		{
			in:  alwaysFailReader{},
			out: "",
			err: errTest,
		},
	}

	for _, c := range cases {
		m, err := NewPlist(c.in, url)
		require.Equal(t, c.out, string(m))
		require.Equal(t, c.err, err)
	}
}

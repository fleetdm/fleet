package dataflatten

import (
	"path/filepath"
	"testing"
)

// TestPlist is testing a very simple plist case. Most of the more complex testing is in the spec files.
func TestPlist(t *testing.T) {
	t.Parallel()

	var tests = []flattenTestCase{
		{
			in: `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><array><string>a</string><string>b</string></array></plist>`,
			out: []Row{
				{Path: []string{"0"}, Value: "a"},
				{Path: []string{"1"}, Value: "b"},
			},
		},
		{
			in:  `<?xml version="1.0" encoding="UTF-8"?>`,
			err: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.comment, func(t *testing.T) {
			t.Parallel()

			actual, err := Plist([]byte(tt.in))
			testFlattenCase(t, tt, actual, err)
		})
	}
}

func TestNestedPlists(t *testing.T) {
	t.Parallel()

	var tests = []flattenTestCase{
		{
			options: []FlattenOpts{WithNestedPlist()},
			comment: "expand nested",
			out: []Row{
				{Path: []string{"inbinary", "astring"}, Value: "hello"},
				{Path: []string{"inbinary", "arr", "0"}, Value: "one"},
				{Path: []string{"inbinary", "arr", "1"}, Value: "two"},
				{Path: []string{"inxml", "arr", "0"}, Value: "one"},
				{Path: []string{"inxml", "arr", "1"}, Value: "two"},
				{Path: []string{"inxml", "astring"}, Value: "hello"},
			},
		},
		{
			comment: "nested and queried",
			options: []FlattenOpts{WithNestedPlist(), WithQuery([]string{"*", "arr", "0"})},
			out: []Row{
				{Path: []string{"inbinary", "arr", "0"}, Value: "one"},
				{Path: []string{"inxml", "arr", "0"}, Value: "one"},
			},
		},
		{
			comment: "not expanded",
			out: []Row{
				{Path: []string{"inbinary"}, Value: "YnBsaXN0MDDSAQIDBlNhcnJXYXN0cmluZ6IEBVNvbmVTdHdvVWhlbGxvCA0RGRwgJAAAAAAAAAEBAAAAAAAAAAcAAAAAAAAAAAAAAAAAAAAq"},
				{Path: []string{"inxml"}, Value: "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<!DOCTYPE plist PUBLIC \"-//Apple//DTD PLIST 1.0//EN\" \"http://www.apple.com/DTDs/PropertyList-1.0.dtd\">\n<plist version=\"1.0\">\n<dict>\n\t<key>arr</key>\n\t<array>\n\t\t<string>one</string>\n\t\t<string>two</string>\n\t</array>\n\t<key>astring</key>\n\t<string>hello</string>\n</dict>\n</plist>"},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.comment, func(t *testing.T) {
			t.Parallel()

			for _, f := range []string{"nested.xml", "nested.plist"} {
				actual, err := PlistFile(filepath.Join("testdata", "nested", f), tt.options...)
				testFlattenCase(t, tt, actual, err)
			}
		})
	}
}

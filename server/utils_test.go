package server

import (
	"encoding/base64"
	"errors"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMaskSecretURLParams(t *testing.T) {
	secretKeywords := []string{"secret", "token", "key", "password"}
	mask := "MASKED"

	type testCase struct {
		name     string
		rawURL   string
		expected string
	}

	cases := []testCase{
		{
			name:     "no params",
			rawURL:   "https://example.com",
			expected: "https://example.com",
		},
		{
			name:     "user info redacted",
			rawURL:   "https://user:P@$$w0rD@example.com/foo/bar?baz=qux&secret_key=baz",
			expected: "https://user:xxxxx@example.com/foo/bar?baz=qux&secret_key=" + mask,
		},
	}
	for i, kw := range secretKeywords {
		cases = append(cases, testCase{
			name:     "single " + kw,
			rawURL:   "https://example.com?" + kw + "=foo",
			expected: "https://example.com?" + kw + "=" + mask,
		})
		cases = append(cases, testCase{
			name:     "multiple " + kw,
			rawURL:   "https://example.com?" + kw + "=foo" + "&bar_" + kw + "=bar",
			expected: "https://example.com?" + kw + "=" + mask + "&bar_" + kw + "=" + mask,
		})
		cases = append(cases, testCase{
			name:     "multiple " + kw + " with other params",
			rawURL:   "https://example.com?foo=bar&" + kw + "=foo" + "&bar_" + kw + "=bar",
			expected: "https://example.com?foo=bar&" + kw + "=" + mask + "&bar_" + kw + "=" + mask,
		})
		cases = append(cases, testCase{
			name:     "multiple " + kw + " with other params and fragment",
			rawURL:   "https://example.com?foo=bar&" + kw + "=foo" + "&bar_" + kw + "=bar#fragment",
			expected: "https://example.com?foo=bar&" + kw + "=" + mask + "&bar_" + kw + "=" + mask + "#fragment",
		})
		kw2 := secretKeywords[(i+1)%len(secretKeywords)]
		cases = append(cases, testCase{
			name:     "combined " + kw + " and " + kw2,
			rawURL:   "https://example.com?foo=bar&" + kw + "=foo" + "&bar_" + kw2 + "=bar",
			expected: "https://example.com?foo=bar&" + kw + "=" + mask + "&bar_" + kw2 + "=" + mask,
		})
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			masked := MaskSecretURLParams(c.rawURL)
			got, err := url.Parse(masked)
			require.NoError(t, err)
			want, err := url.Parse(c.expected)
			require.NoError(t, err)
			require.EqualValues(t, got.Query(), want.Query())
			require.Equal(t, got.Fragment, want.Fragment)
			require.Equal(t, got.Host, want.Host)
			require.Equal(t, got.Path, want.Path)
			require.Equal(t, got.Scheme, want.Scheme)
		})
	}
}

func TestMaskURLError(t *testing.T) {
	t.Run("not url error", func(t *testing.T) {
		e := errors.New("not url.Error")
		errStr := e.Error()
		masked := MaskURLError(e)
		require.Equal(t, e, masked)
		require.EqualError(t, masked, errStr)
	})

	t.Run("no secret in URL", func(t *testing.T) {
		e := &url.Error{Op: "GET", URL: "https://example.com?foo=bar", Err: errors.New("not found")}
		errStr := e.Error()
		masked := MaskURLError(e)
		require.Equal(t, e, masked)
		require.EqualError(t, masked, errStr)
		require.Contains(t, masked.Error(), "?foo=bar")
	})

	t.Run("masked secret in URL", func(t *testing.T) {
		e := &url.Error{Op: "GET", URL: "https://example.com?the_secret=42", Err: errors.New("not found")}
		masked := MaskURLError(e)
		require.Equal(t, e, masked)
		require.EqualError(t, masked, "GET \"https://example.com?the_secret=MASKED\": not found")
		require.NotContains(t, masked.Error(), "42")
	})
}

func TestBase64DecodePaddingAgnostic(t *testing.T) {
	cases := []struct {
		in   string
		want []byte
		err  error
	}{
		{"", []byte{}, nil},
		{"==", []byte{}, nil},
		{"==", []byte{}, nil},
		{"dGVzdA==", []byte("test"), nil},
		{"dGVzdA", []byte("test"), nil},
		{"dGVzdA==ABC", []byte("tes"), base64.CorruptInputError(6)},
	}

	for _, c := range cases {
		got, err := Base64DecodePaddingAgnostic(c.in)
		require.Equal(t, c.err, err)
		require.Equal(t, got, c.want)
	}
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		// Zero and negative values
		{"zero bytes", 0, "0 bytes"},
		{"negative bytes", -100, "0 bytes"},

		// Raw bytes (not divisible by 100 or 128)
		{"raw bytes small", 1, "1 bytes"},
		{"raw bytes 99", 99, "99 bytes"},
		{"raw bytes 127", 127, "127 bytes"},
		{"raw bytes 129", 129, "129 bytes"},
		{"raw bytes 1001", 1001, "1001 bytes"},

		// SI units (divisible by 100)
		{"SI 100 bytes", 100, "100 bytes"},
		{"SI 1000 bytes (1 KB)", 1000, "1 KB"},
		{"SI 1500 bytes", 1500, "1.5 KB"},
		{"SI 1000000 bytes (1 MB)", 1000000, "1 MB"},
		{"SI 1500000 bytes", 1500000, "1.5 MB"},
		{"SI 1000000000 bytes (1 GB)", 1000000000, "1 GB"},
		{"SI 1500000000 bytes", 1500000000, "1.5 GB"},
		{"SI 1000000000000 bytes (1 TB)", 1000000000000, "1 TB"},
		{"SI 2500000000000 bytes", 2500000000000, "2.5 TB"},

		// Binary units (divisible by 128)
		{"binary 128 bytes", 128, "128 bytes"},
		{"binary 1024 bytes (1 KiB)", 1024, "1 KiB"},
		{"binary 1536 bytes (1.5 KiB)", 1536, "1.5 KiB"},
		{"binary 1048576 bytes (1 MiB)", 1048576, "1 MiB"},
		{"binary 1572864 bytes (1.5 MiB)", 1572864, "1.5 MiB"},
		{"binary 1073741824 bytes (1 GiB)", 1073741824, "1 GiB"},
		{"binary 1610612736 bytes (1.5 GiB)", 1610612736, "1.5 GiB"},
		{"binary 1099511627776 bytes (1 TiB)", 1099511627776, "1 TiB"},

		// Edge cases - values divisible by both 100 and 128
		// 6400 is divisible by both 100 and 128, so SI takes precedence
		{"divisible by both 100 and 128", 6400, "6.4 KB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatFileSize(tt.bytes)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestRemoveDuplicatesFromSlice(t *testing.T) {
	tests := map[string]struct {
		input  []interface{}
		output []interface{}
	}{
		"no duplicates": {
			input:  []interface{}{34, 56, 1},
			output: []interface{}{34, 56, 1},
		},
		"1 duplicate": {
			input:  []interface{}{"a", "d", "a"},
			output: []interface{}{"a", "d"},
		},
		"all duplicates": {
			input:  []interface{}{true, true, true},
			output: []interface{}{true},
		},
	}
	for name, test := range tests {
		t.Run(
			name, func(t *testing.T) {
				require.Equal(t, test.output, RemoveDuplicatesFromSlice(test.input))
			},
		)
	}
}

package server

import (
	"encoding/base64"
	"errors"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMaskSecretURLParams(t *testing.T) {
	secretKeywords := []string{"secret", "token", "key", "password", "signature", "credential"}
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

	t.Run("presigned URL credential and signature", func(t *testing.T) {
		rawURL := "https://storage.googleapis.com/bucket/dev-prefix/uploads/0b8f1f0e-6c1d-4a39-9d57-3f1f2a9c7e11" +
			"?X-Amz-Algorithm=AWS4-HMAC-SHA256" +
			"&X-Amz-Credential=GOOG1EEXAMPLEACCESSKEY%2F20261026%2Fauto%2Fs3%2Faws4_request" +
			"&X-Amz-Date=20261026T120000Z&X-Amz-Expires=21600" +
			"&X-Amz-SignedHeaders=content-length%3Bhost&x-id=PutObject" +
			"&X-Amz-Signature=3f5a9c0d1e2b4a6c8d7e9f0a1b2c3d4e5f60718293a4b5c6d7e8f9a0b1c2d3e4"
		e := &url.Error{Op: "Put", URL: rawURL, Err: errors.New("connection reset")}
		masked := MaskURLError(e)
		require.NotContains(t, masked.Error(), "GOOG1EEXAMPLEACCESSKEY")
		require.NotContains(t, masked.Error(), "3f5a9c0d1e2b4a6c")
		require.Contains(t, masked.Error(), "X-Amz-Expires=21600")
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

func TestGenerateRandomEmail(t *testing.T) {
	cases := []struct {
		name      string
		base      string
		wantParts func(t *testing.T, result string)
	}{
		{
			name: "standard email",
			base: "user@example.com",
			wantParts: func(t *testing.T, result string) {
				require.Contains(t, result, "@example.com")
				require.Contains(t, result, "user+")
			},
		},
		{
			name: "no @ in base",
			base: "useronly",
			wantParts: func(t *testing.T, result string) {
				require.True(t, strings.HasPrefix(result, "useronly+"))
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result, err := GenerateRandomEmail(c.base)
			require.NoError(t, err)
			require.NotEmpty(t, result)
			c.wantParts(t, result)

			// Each call must produce a different value.
			result2, err := GenerateRandomEmail(c.base)
			require.NoError(t, err)
			require.NotEqual(t, result, result2)
		})
	}
}

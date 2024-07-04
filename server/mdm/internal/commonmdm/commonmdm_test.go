package commonmdm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveURL(t *testing.T) {
	type testCase struct {
		serverURL  string
		relPath    string
		cleanQuery bool
		expected   string
		expectErr  bool
	}

	testCases := []testCase{
		{
			serverURL:  "http://example.com",
			relPath:    "path/to/resource",
			cleanQuery: false,
			expected:   "http://example.com/path/to/resource",
			expectErr:  false,
		},
		{
			serverURL:  "http://example.com?query=string",
			relPath:    "path",
			cleanQuery: true,
			expected:   "http://example.com/path",
			expectErr:  false,
		},
		{
			serverURL:  "http://example.com/base/",
			relPath:    "/path",
			cleanQuery: false,
			expected:   "http://example.com/base/path",
			expectErr:  false,
		},
		{
			serverURL:  "http://example.com",
			relPath:    "path/to/resource",
			cleanQuery: true,
			expected:   "http://example.com/path/to/resource",
			expectErr:  false,
		},
		{
			serverURL:  ":invalidurl",
			relPath:    "path",
			cleanQuery: false,
			expected:   "",
			expectErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.serverURL+"_"+tc.relPath, func(t *testing.T) {
			result, err := ResolveURL(tc.serverURL, tc.relPath, tc.cleanQuery)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, result)
			}
		})
	}
}

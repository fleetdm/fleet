package oval_parsed

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEval(t *testing.T) {
	utest := UnixUnameTest{
		States: []ObjectStateString{
			NewObjectStateString("less than", "0:5.15.0-1004"),
			NewObjectStateString("pattern match", `5.15.0-\d+(-generic|-generic-64k|-generic-lpae|-lowlatency|-lowlatency-64k)`),
		},
	}

	testCases := []struct {
		Name     string
		Input    string
		Expected bool
	}{
		{Name: "less than", Input: "5.15.0-1003-generic", Expected: true},
		{Name: "greater than", Input: "5.15.0-1005-generic", Expected: false},
		{Name: "equal", Input: "5.15.0-1004-generic", Expected: false},
		{Name: "alt pattern match", Input: "5.15.0-1003-lowlatency", Expected: true},
		{Name: "suffix doesn't match", Input: "5.15.0-1004-foo", Expected: false},
		{Name: "lower version fails pattern match", Input: "4.0.0-10-generic", Expected: false},
		{Name: "higher version fails pattern match", Input: "6.0.0-10-generic", Expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			matches, err := utest.Eval(tc.Input)
			require.NoError(t, err)
			require.Equal(t, tc.Expected, matches)
		})
	}
}

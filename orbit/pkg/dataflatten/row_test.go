package dataflatten

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRowParentFunctions(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		in     Row
		parent string
		key    string
	}{
		{
			in: Row{},
		},

		{
			in: Row{Path: []string{}},
		},
		{
			in:     Row{Path: []string{"a"}},
			parent: "",
			key:    "a",
		},
		{
			in:     Row{Path: []string{"a", "b"}},
			parent: "a",
			key:    "b",
		},
		{
			in:     Row{Path: []string{"a", "b", "c"}},
			parent: "a/b",
			key:    "c",
		},
	}

	for _, tt := range tests {
		parent, key := tt.in.ParentKey("/")
		require.Equal(t, tt.parent, parent)
		require.Equal(t, tt.key, key)
	}
}

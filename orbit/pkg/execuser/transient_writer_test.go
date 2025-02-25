package execuser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransientWriter(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		p    [][]byte
		want string
	}{
		{
			name: "empty",
			p:    nil,
			want: "",
		},
		{
			name: "small",
			p:    [][]byte{[]byte("abc")},
			want: "abc",
		},
		{
			name: "small_2",
			p:    [][]byte{[]byte("abc"), []byte("def")},
			want: "abcdef",
		},
		{
			name: "large",
			p:    [][]byte{[]byte(strings.Repeat("a", bufSize))},
			want: strings.Repeat("a", bufSize),
		},
		{
			name: "large_2",
			p:    [][]byte{[]byte(strings.Repeat("a", bufSize-1)), []byte(strings.Repeat("b", bufSize-1))},
			want: "a" + strings.Repeat("b", bufSize-1),
		},
		{
			name: "large_3",
			p:    [][]byte{[]byte(strings.Repeat("a", bufSize) + "b")},
			want: strings.Repeat("a", bufSize-1) + "b",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tw := &TransientWriter{}
			for _, p := range tt.p {
				n, err := tw.Write(p)
				require.NoError(t, err)
				assert.Equal(t, len(p), n)
			}
			assert.Equal(t, tt.want, tw.String())
		})
	}
}

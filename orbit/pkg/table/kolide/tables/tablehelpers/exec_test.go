//go:build !windows
// +build !windows

package tablehelpers

import (
	"context"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
)

func TestExec(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		name    string
		timeout int
		bins    []string
		args    []string
		err     bool
		output  string
	}{
		{
			name:    "timeout",
			timeout: 1,
			bins:    []string{"/bin/sleep", "/usr/bin/sleep"},
			args:    []string{"30"},
			err:     true,
		},
		{
			name: "no binaries",
			bins: []string{"/hello/world", "/hello/friends"},
			err:  true,
		},
		{
			name: "false",
			bins: []string{"/bin/false", "/usr/bin/false"},
			err:  true,
		},
		{
			name: "eventually finds binary",
			bins: []string{"/hello/world", "/bin/true", "/usr/bin/true"},
		},
		{
			name:   "output",
			bins:   []string{"/bin/echo"},
			args:   []string{"hello"},
			output: "hello\n",
		},
	}

	ctx := context.Background()
	logger := log.NewNopLogger()

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.timeout == 0 {
				tt.timeout = 30
			}
			output, err := Exec(ctx, logger, tt.timeout, tt.bins, tt.args)
			if tt.err {
				assert.Error(t, err)
				assert.Empty(t, output)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, []byte(tt.output), output)
			}

		})
	}
}

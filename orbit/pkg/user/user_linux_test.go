//go:build linux

package user

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLoginctlUsersOutput(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []User
		wantErr string
	}{
		{
			name:  "single user",
			input: " 1000 alice\n",
			want:  []User{{Name: "alice", ID: 1000}},
		},
		{
			name:  "multiple users",
			input: " 1000 alice\n 1001 bob\n",
			want: []User{
				{Name: "alice", ID: 1000},
				{Name: "bob", ID: 1001},
			},
		},
		{
			name:  "includes system user",
			input: "    0 root\n 1000 alice\n  120 gdm\n",
			want: []User{
				{Name: "root", ID: 0},
				{Name: "alice", ID: 1000},
				{Name: "gdm", ID: 120},
			},
		},
		{
			name:  "no leading whitespace",
			input: "1000 alice\n1001 bob\n",
			want: []User{
				{Name: "alice", ID: 1000},
				{Name: "bob", ID: 1001},
			},
		},
		{
			name:  "extra columns (some systemd versions)",
			input: "1000 alice extra-stuff\n",
			want:  []User{{Name: "alice", ID: 1000}},
		},
		{
			name:    "empty output",
			input:   "",
			wantErr: "no user session found",
		},
		{
			name:    "whitespace only",
			input:   "   \n  \n",
			wantErr: "no user session found",
		},
		{
			name:  "skips malformed lines",
			input: "not-a-uid alice\n1000 bob\n",
			want:  []User{{Name: "bob", ID: 1000}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseLoginctlUsersOutput(tt.input)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

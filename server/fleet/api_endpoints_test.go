package fleet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizePathPlaceholders(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{
			path: "/api/_version_/fleet/trigger",
			want: "/api/_version_/fleet/trigger",
		},
		{
			path: "/software/titles/:title_id/icon/:id",
			want: "/software/titles/:placeholder_1/icon/:placeholder_2",
		},
		{
			path: "/api/_version_/fleet/hosts/:id",
			want: "/api/_version_/fleet/hosts/:placeholder_1",
		},
		{
			path: "/a/:b/:c/:d",
			want: "/a/:placeholder_1/:placeholder_2/:placeholder_3",
		},
		{
			path: "/no/placeholders/here",
			want: "/no/placeholders/here",
		},
		{
			path: "/:single",
			want: "/:placeholder_1",
		},
		{
			path: "/UPPER/CASE",
			want: "/UPPER/CASE",
		},
		// gorilla/mux brace-style placeholders
		{
			path: "/api/_version_/fleet/hosts/{id:[0-9]+}",
			want: "/api/_version_/fleet/hosts/:placeholder_1",
		},
		{
			path: "/api/_version_/fleet/hosts/{id:[0-9]+}/reports/{report_id:[0-9]+}",
			want: "/api/_version_/fleet/hosts/:placeholder_1/reports/:placeholder_2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			require.Equal(t, tt.want, normalizePathPlaceholders(tt.path))
		})
	}
}

func TestAPIEndpointValidate(t *testing.T) {
	base := APIEndpoint{Method: "GET", Path: "/api/_version_/fleet/foo", DisplayName: "foo"}

	tests := []struct {
		name    string
		modify  func(APIEndpoint) APIEndpoint
		wantErr string
	}{
		{
			name:   "valid endpoint",
			modify: func(e APIEndpoint) APIEndpoint { return e },
		},
		{
			name:    "missing display_name",
			modify:  func(e APIEndpoint) APIEndpoint { e.DisplayName = ""; return e },
			wantErr: "display_name is required",
		},
		{
			name:    "whitespace display_name",
			modify:  func(e APIEndpoint) APIEndpoint { e.DisplayName = "   "; return e },
			wantErr: "display_name is required",
		},
		{
			name:    "invalid method",
			modify:  func(e APIEndpoint) APIEndpoint { e.Method = "GTE"; return e },
			wantErr: "invalid HTTP method",
		},
		{
			name:    "empty path",
			modify:  func(e APIEndpoint) APIEndpoint { e.Path = " "; return e },
			wantErr: "path is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.modify(base).validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

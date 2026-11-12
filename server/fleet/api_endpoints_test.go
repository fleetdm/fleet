package fleet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIdIsConstructedCorrectly(t *testing.T) {
	tests := []struct {
		method string
		path   string
		want   string
	}{
		{
			method: "GET",
			path:   "/api/v1/fleet/trigger",
			want:   "|GET|/api/v1/fleet/trigger|",
		},
		{
			method: "GET",
			path:   "/software/titles/:title_id/icon/:id",
			want:   "|GET|/software/titles/:placeholder_1/icon/:placeholder_2|",
		},
		{
			method: "GET",
			path:   "/api/v1/fleet/hosts/:id",
			want:   "|GET|/api/v1/fleet/hosts/:placeholder_1|",
		},
		{
			method: "post",
			path:   "/a/:b/:c/:d",
			want:   "|POST|/a/:placeholder_1/:placeholder_2/:placeholder_3|",
		},
		{
			method: "pAtCh",
			path:   "/no/placeholders/here",
			want:   "|PATCH|/no/placeholders/here|",
		},
		{
			method: "GET",
			path:   "/:single",
			want:   "|GET|/:placeholder_1|",
		},
		{
			method: "GET",
			path:   "/UPPER/CASE",
			want:   "|GET|/upper/case|",
		},
		// gorilla/mux brace-style placeholders
		{
			method: "post",
			path:   "/api/v1/fleet/hosts/{id:[0-9]+}",
			want:   "|POST|/api/v1/fleet/hosts/:placeholder_1|",
		},
		{
			method: "get",
			path:   "/api/v1/fleet/hosts/{id:[0-9]+}/reports/{report_id:[0-9]+}",
			want:   "|GET|/api/v1/fleet/hosts/:placeholder_1/reports/:placeholder_2|",
		},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			sut := NewAPIEndpointFromTpl(tt.method, tt.path)
			require.Equal(t, tt.want, sut.Fingerprint())
		})
	}
}

func TestAPIEndpointValidate(t *testing.T) {
	base := APIEndpoint{Method: "GET", Path: "/api/v1/fleet/foo", DisplayName: "foo"}

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

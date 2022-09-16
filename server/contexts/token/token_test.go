package token

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestFromHTTPRequest(t *testing.T) {
	tests := []struct {
		name string
		r    *http.Request
		want Token
	}{
		{
			name: "no auth",
			want: "",
			r:    &http.Request{},
		}, {
			name: "empty auth",
			r: &http.Request{
				Header: map[string][]string{
					"Authorization": {""},
				},
			},
			want: "",
		}, {
			name: "BEARER no data",
			r: &http.Request{
				Header: map[string][]string{
					"Authorization": {"BEARER"},
					"Content-Type":  {"application/x-www-form-urlencoded"},
				},
				Method: http.MethodPost,
				Body:   io.NopCloser(strings.NewReader("token=bar")),
			},
			want: "",
		}, {
			name: "BEARER foobar",
			r: &http.Request{
				Header: map[string][]string{
					"Authorization": {"BEARER foobar"},
					"Content-Type":  {"application/x-www-form-urlencoded"},
				},
				Method: http.MethodPost,
				Body:   io.NopCloser(strings.NewReader("token=bar")),
			},
			want: "foobar",
		}, {
			name: "from body",
			r: &http.Request{
				Header: map[string][]string{
					"Authorization": {"FOOBAR foobar"},
					"Content-Type":  {"application/x-www-form-urlencoded"},
				},
				Method: http.MethodPost,
				Body:   io.NopCloser(strings.NewReader("token=bar")),
			},
			want: "bar",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FromHTTPRequest(tt.r); got != tt.want {
				t.Errorf("FromHTTPRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

package main

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/mdm/profilevalidation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ddfPage mirrors the Learn page: the current drop first, then older drops and pre-v2 downloads.
const ddfPage = `<html><body>
<p>DDF v2 XML files for Windows 10 and Windows 11 are combined, and provided in a single download:</p>
<ul><li><a href="https://download.microsoft.com/download/015bd9f5-9cca-4821-8a85-a4c5f9a5d0f2/DDFv2Feb2026.zip" data-linktype="external">DDF v2 Files, February 2026</a></li></ul>
<h2>Older DDF files</h2>
<ul>
<li><a href="https://download.microsoft.com/download/2ff2c8b9-2e3f-47af-89e6-11c19b7f0c2a/DDFv2Sept25.zip">September 2025</a></li>
<li><a href="https://download.microsoft.com/download/a/a/a/aaadc008-67d4-4dcd-b864-70c479baf7d6/DDFv2September24.zip">September 2024</a></li>
<li><a href="https://download.microsoft.com/download/6/F/0/6F019079-6EB0-41B5-88E8-D1CE77DBA27B/Windows10_1903_DDF_download.zip">1903</a></li>
</ul></body></html>`

func TestResolveDDF(t *testing.T) {
	cases := []struct {
		name    string
		status  int
		page    string
		want    profilevalidation.WindowsVersion
		wantErr string
	}{
		{
			name: "first link is current",
			page: ddfPage,
			want: profilevalidation.WindowsVersion{
				Release: "DDFv2Feb2026",
				URL:     "https://download.microsoft.com/download/015bd9f5-9cca-4821-8a85-a4c5f9a5d0f2/DDFv2Feb2026.zip",
			},
		},
		{
			name: "other hosts and pre-v2 downloads ignored",
			page: `<a href="https://example.com/download/x/DDFv2Fake.zip">x</a>
				<a href="https://download.microsoft.com/download/6/F/0/6F019079/Windows10_1903_DDF_download.zip">1903</a>
				<a href="https://download.microsoft.com/download/a/b/DDFv2May24.zip">May 2024</a>`,
			want: profilevalidation.WindowsVersion{Release: "DDFv2May24", URL: "https://download.microsoft.com/download/a/b/DDFv2May24.zip"},
		},
		{
			name:    "no links",
			page:    `<html><body>Moved to a new page.</body></html>`,
			wantErr: "no DDFv2 zip linked from",
		},
		{
			name:    "not found",
			status:  http.StatusNotFound,
			wantErr: "unexpected status 404 Not Found",
		},
		{
			name:    "server error",
			status:  http.StatusInternalServerError,
			page:    ddfPage,
			wantErr: "unexpected status 500 Internal Server Error",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if c.status != 0 {
					w.WriteHeader(c.status)
				}
				_, _ = w.Write([]byte(c.page))
			}))
			defer srv.Close()

			got, err := resolveDDF(t.Context(), slog.New(slog.DiscardHandler), fleethttp.NewClient(), srv.URL)
			if c.wantErr != "" {
				require.ErrorContains(t, err, c.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, c.want, got)
		})
	}
}

package homebrew

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

func TestIngestValidations(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var cask brewCask

		appToken := strings.TrimSuffix(path.Base(r.URL.Path), ".json")
		switch appToken {
		case "fail":
			w.WriteHeader(http.StatusInternalServerError)
			return

		case "notfound":
			w.WriteHeader(http.StatusNotFound)
			return

		case "noname":
			cask = brewCask{
				Token:   appToken,
				Name:    nil,
				URL:     "https://example.com",
				Version: "1.0",
			}

		case "emptyname":
			cask = brewCask{
				Token:   appToken,
				Name:    []string{""},
				URL:     "https://example.com",
				Version: "1.0",
			}

		case "notoken":
			cask = brewCask{
				Token:   "",
				Name:    []string{appToken},
				URL:     "https://example.com",
				Version: "1.0",
			}

		case "noversion":
			cask = brewCask{
				Token:   appToken,
				Name:    []string{appToken},
				URL:     "https://example.com",
				Version: "",
			}

		case "nourl":
			cask = brewCask{
				Token:   appToken,
				Name:    []string{appToken},
				URL:     "",
				Version: "1.0",
			}

		case "invalidurl":
			cask = brewCask{
				Token:   appToken,
				Name:    []string{appToken},
				URL:     "https://\x00\x01\x02",
				Version: "1.0",
			}

		case "ok":
			cask = brewCask{
				Token:   appToken,
				Name:    []string{appToken},
				URL:     "https://example.com",
				Version: "1.0",
			}

		default:
			w.WriteHeader(http.StatusBadRequest)
			t.Fatalf("unexpected app token %s", appToken)
		}

		err := json.NewEncoder(w).Encode(cask)
		require.NoError(t, err)
	}))
	t.Cleanup(srv.Close)

	ctx := context.Background()

	cases := []struct {
		appToken     string
		wantErr      string
		upsertCalled bool
	}{
		{"fail", "brew API returned status 500", false},
		{"notfound", "app not found in brew API", false},
		{"noname", "missing name for cask noname", false},
		{"emptyname", "missing name for cask emptyname", false},
		{"notoken", "missing token for cask notoken", false},
		{"noversion", "missing version for cask noversion", false},
		{"nourl", "missing URL for cask nourl", false},
		{"invalidurl", "parse URL for cask invalidurl", false},
		{"ok", "", true},
	}
	for _, c := range cases {
		t.Run(c.appToken, func(t *testing.T) {
			i := &brewIngester{
				logger:  log.NewNopLogger(),
				client:  fleethttp.NewClient(fleethttp.WithTimeout(10 * time.Second)),
				baseURL: srv.URL + "/",
			}

			inputApp := inputApp{Token: c.appToken, UniqueIdentifier: "abc", InstallerFormat: "pkg"}

			_, err := i.ingestOne(ctx, inputApp)
			if c.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, c.wantErr)
			}
		})
	}
}

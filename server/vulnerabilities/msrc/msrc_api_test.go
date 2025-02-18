package msrc

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMSRCClient(t *testing.T) {
	t.Run("#feedName", func(t *testing.T) {
		date := time.Date(2010, 10, 10, 0, 0, 0, 0, time.UTC)
		require.Equal(t, "2010-Oct", feedName(date))
	})

	t.Run("#GetFeed", func(t *testing.T) {
		t.Run("with invalid args", func(t *testing.T) {
			sut := NewMSRCClient(nil, "", "")
			now := time.Now()

			t.Run("year is below min allowed", func(t *testing.T) {
				_, err := sut.GetFeed(time.January, MSRCMinYear-1)
				require.Error(t, err)
			})

			t.Run("year is above current year", func(t *testing.T) {
				_, err := sut.GetFeed(time.January, now.Year()+1)
				require.Error(t, err)
			})

			t.Run("provided arguments are in the future", func(t *testing.T) {
				cases := []time.Time{
					now.AddDate(0, 1, 0),
					now.AddDate(1, 0, 0),
					now.AddDate(1, 1, 0),
				}

				for _, c := range cases {
					_, err := sut.GetFeed(c.Month(), c.Year())
					require.Error(t, err)
				}
			})
		})

		t.Run("it downloads the feed file in the provided path", func(t *testing.T) {
			date := time.Date(2021, 10, 10, 0, 0, 0, 0, time.UTC)
			dir := t.TempDir()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/cvrf/v3.0/document/2021-Oct" {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("some payload")) //nolint:errcheck
				}
			}))
			t.Cleanup(server.Close)

			sut := NewMSRCClient(server.Client(), dir, server.URL)
			result, err := sut.GetFeed(date.Month(), date.Year())
			require.NoError(t, err)

			contents, err := os.ReadFile(result)
			require.NoError(t, err)
			require.Equal(t, []byte("some payload"), contents)
		})
	})
}

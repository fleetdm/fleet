package msrc_io

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMSRCClient(t *testing.T) {
	t.Run("#urlSuffix", func(t *testing.T) {
		date := time.Date(2010, 10, 10, 0, 0, 0, 0, time.UTC)
		actual := urlSuffix(date)
		require.Equal(t, "2010-Oct", actual)
	})

	t.Run("#GetFeed", func(t *testing.T) {
		t.Run("with invalid args", func(t *testing.T) {
			sut := NewMSRCClient(nil, "", nil)
			now := time.Now()

			t.Run("year is below min allowed", func(t *testing.T) {
				_, err := sut.GetFeed(time.January, minFeedYear-1)
				require.Error(t, err)
			})

			t.Run("year is above current year", func(t *testing.T) {
				_, err := sut.GetFeed(time.January, now.Year()+1)
				require.Error(t, err)
			})

			t.Run("provided month and year is in the future", func(t *testing.T) {
				_, err := sut.GetFeed((now.AddDate(0, 1, 0)).Month(), now.Year())
				require.Error(t, err)
			})
		})

		t.Run("it downloads the feed file in the provided path", func(t *testing.T) {
			date := time.Date(2021, 10, 10, 0, 0, 0, 0, time.UTC)
			dir := t.TempDir()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Println(r.URL.Path)
				if r.URL.Path == "/cvrf/v2.0/document/2021-Oct" {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("some payload"))
				}
			}))
			t.Cleanup(server.Close)

			sut := NewMSRCClient(server.Client(), dir, &server.URL)
			result, err := sut.GetFeed(date.Month(), date.Year())
			require.NoError(t, err)
			require.FileExists(t, result)
		})
	})
}

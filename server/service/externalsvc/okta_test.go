package externalsvc

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOktaROPFlow(t *testing.T) {
	mockOkta := RunMockOktaServer(t)
	t.Run("valid request", func(t *testing.T) {
		okta := Okta{
			BaseURL:      mockOkta.Srv.URL,
			ClientID:     mockOkta.ClientID,
			ClientSecret: mockOkta.ClientSecret(),
		}
		err := okta.ROPLogin(context.Background(), mockOkta.Username, mockOkta.UserPassword)
		require.NoError(t, err)
	})

	t.Run("bad user credentials", func(t *testing.T) {
		okta := Okta{
			BaseURL:      mockOkta.Srv.URL,
			ClientID:     mockOkta.ClientID,
			ClientSecret: mockOkta.ClientSecret(),
		}
		err := okta.ROPLogin(context.Background(), mockOkta.Username, "invalid")
		require.ErrorIs(t, err, ErrInvalidGrant)

		err = okta.ROPLogin(context.Background(), "invalid", mockOkta.UserPassword)
		require.ErrorIs(t, err, ErrInvalidGrant)
	})

	t.Run("bad client credentials", func(t *testing.T) {
		okta := Okta{
			BaseURL:      mockOkta.Srv.URL,
			ClientID:     "invalid",
			ClientSecret: mockOkta.ClientSecret(),
		}
		err := okta.ROPLogin(context.Background(), mockOkta.Username, mockOkta.UserPassword)
		require.ErrorContains(t, err, "invalid_client")

		okta = Okta{
			BaseURL:      mockOkta.Srv.URL,
			ClientID:     mockOkta.ClientID,
			ClientSecret: "invalid",
		}
		err = okta.ROPLogin(context.Background(), mockOkta.Username, mockOkta.UserPassword)
		require.ErrorContains(t, err, "invalid_client")
	})

	t.Run("bad server responses", func(t *testing.T) {
		t.Cleanup(func() { mockOkta.SetCustomResp(nil) })
		okta := Okta{
			BaseURL:      mockOkta.Srv.URL,
			ClientID:     mockOkta.ClientID,
			ClientSecret: mockOkta.ClientSecret(),
		}

		mockOkta.SetCustomResp(func(w http.ResponseWriter) {
			w.WriteHeader(http.StatusInternalServerError)
			_, err := w.Write([]byte(`{invalid}`))
			require.NoError(t, err)
		})
		err := okta.ROPLogin(context.Background(), mockOkta.Username, mockOkta.UserPassword)
		require.Error(t, err)
	})
}

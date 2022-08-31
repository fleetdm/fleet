package service

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUrlGeneration(t *testing.T) {
	t.Run("without prefix", func(t *testing.T) {
		bc, err := newBaseClient("https://test.com", true, "", "")
		require.NoError(t, err)
		require.Equal(t, "https://test.com/test/path", bc.url("test/path", "").String())
		require.Equal(t, "https://test.com/test/path?raw=query", bc.url("test/path", "raw=query").String())
	})

	t.Run("with prefix", func(t *testing.T) {
		bc, err := newBaseClient("https://test.com", true, "", "prefix/")
		require.NoError(t, err)
		require.Equal(t, "https://test.com/prefix/test/path", bc.url("test/path", "").String())
		require.Equal(t, "https://test.com/prefix/test/path?raw=query", bc.url("test/path", "raw=query").String())
	})
}

func TestParseResponseKnownErrors(t *testing.T) {
	cases := []struct {
		message string
		code    int
		out     error
	}{
		{"not found errors", http.StatusNotFound, notFoundErr{}},
		{"unauthenticated errors", http.StatusUnauthorized, ErrUnauthenticated},
		{"license errors", http.StatusPaymentRequired, ErrMissingLicense},
	}

	for _, c := range cases {
		t.Run(c.message, func(t *testing.T) {
			bc, err := newBaseClient("https://test.com", true, "", "")
			require.NoError(t, err)
			response := &http.Response{
				StatusCode: c.code,
				Body:       ioutil.NopCloser(bytes.NewBufferString(`{"test": "ok"}`)),
			}
			err = bc.parseResponse("GET", "", response, &struct{}{})
			require.ErrorIs(t, err, c.out)
		})
	}
}

func TestParseResponseOK(t *testing.T) {
	bc, err := newBaseClient("https://test.com", true, "", "")
	require.NoError(t, err)
	response := &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(bytes.NewBufferString(`{"test": "ok"}`)),
	}

	var resDest struct{ Test string }
	err = bc.parseResponse("", "", response, &resDest)
	require.NoError(t, err)
	require.Equal(t, "ok", resDest.Test)
}

func TestParseResponseGeneralErrors(t *testing.T) {
	t.Run("general HTTP errors", func(t *testing.T) {
		bc, err := newBaseClient("https://test.com", true, "", "")
		require.NoError(t, err)
		response := &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       ioutil.NopCloser(bytes.NewBufferString(`{"test": "ok"}`)),
		}
		err = bc.parseResponse("GET", "", response, &struct{}{})
		require.Error(t, err)
	})

	t.Run("parse errors", func(t *testing.T) {
		bc, err := newBaseClient("https://test.com", true, "", "")
		require.NoError(t, err)
		response := &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       ioutil.NopCloser(bytes.NewBufferString(`invalid json`)),
		}
		err = bc.parseResponse("GET", "", response, &struct{}{})
		require.Error(t, err)
	})
}

func TestNewBaseClient(t *testing.T) {
	t.Run("invalid addresses are an error", func(t *testing.T) {
		_, err := newBaseClient("invalid", true, "", "")
		require.Error(t, err)
	})

	t.Run("http is only valid in development", func(t *testing.T) {
		_, err := newBaseClient("http://test.com", true, "", "")
		require.Error(t, err)

		_, err = newBaseClient("http://localhost:8080", true, "", "")
		require.NoError(t, err)

		_, err = newBaseClient("http://127.0.0.1:8080", true, "", "")
		require.NoError(t, err)
	})
}

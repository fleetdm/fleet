package update

import (
	"bytes"
	"crypto/sha512"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloadWithSHA512HashInvalidURL(t *testing.T) {
	t.Parallel()

	err := DownloadWithSHA512Hash("localhost:12345569900", ioutil.Discard, 55, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "make get request")
}

func TestDownloadWithSHA512HashErrorResponse(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "error", http.StatusInternalServerError)
	}))
	defer ts.Close()

	err := DownloadWithSHA512Hash(ts.URL, ioutil.Discard, 55, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected HTTP status")
}

func TestDownloadWithSHA512Hash(t *testing.T) {
	t.Parallel()

	expectedData := []byte("abc")
	expectedHash, expectedLen := sha512Hash(expectedData), int64(len(expectedData))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, string(expectedData))
	}))
	defer ts.Close()

	var b bytes.Buffer
	err := DownloadWithSHA512Hash(ts.URL, &b, expectedLen, expectedHash)
	require.NoError(t, err)
	assert.Equal(t, expectedData, b.Bytes())
}

func TestDownloadWithSHA512HashTooSmall(t *testing.T) {
	t.Parallel()

	expectedData := []byte("abc")
	expectedHash, expectedLen := sha512Hash(expectedData), int64(len(expectedData))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Don't write all of data
		fmt.Fprint(w, string(expectedData[:2]))
	}))
	defer ts.Close()

	err := DownloadWithSHA512Hash(ts.URL, ioutil.Discard, expectedLen, expectedHash)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "small")
}

func TestDownloadWithSHA512HashTooLarge(t *testing.T) {
	t.Parallel()

	expectedData := []byte("abc")
	expectedHash, expectedLen := sha512Hash(expectedData), int64(len(expectedData))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Write additional data
		fmt.Fprintf(w, string(expectedData)+"foobar")
	}))
	defer ts.Close()

	err := DownloadWithSHA512Hash(ts.URL, ioutil.Discard, expectedLen, expectedHash)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "large")
}

func TestDownloadWithSHA512HashMismatch(t *testing.T) {
	t.Parallel()

	expectedData := []byte("abc")
	expectedHash, expectedLen := sha512Hash(expectedData), int64(len(expectedData))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Write non-matching data
		fmt.Fprint(w, string("def"))
	}))
	defer ts.Close()

	err := DownloadWithSHA512Hash(ts.URL, ioutil.Discard, expectedLen, expectedHash)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not match")
}

func sha512Hash(data []byte) []byte {
	hash := sha512.New()
	if _, err := hash.Write(data); err != nil {
		panic(err)
	}
	return hash.Sum(nil)
}

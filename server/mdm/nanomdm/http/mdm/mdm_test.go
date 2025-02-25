package mdm

import (
	"bytes"
	"context"
	"crypto/x509"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/micromdm/nanolib/log"
	"github.com/stretchr/testify/require"
)

const (
	testHash = "ZZZYYYXXX"
	testID   = "AAABBBCCC"
)

func testHashCert(_ *x509.Certificate) string {
	return testHash
}

type testCertAuthRetriever struct{}

func (c *testCertAuthRetriever) EnrollmentFromHash(ctx context.Context, hash string) (string, error) {
	if hash != testHash {
		return "", errors.New("invalid test hash")
	}
	return testID, nil
}
func TestCertWithEnrollmentIDMiddleware(t *testing.T) {
	response := []byte("mock response")
	// mock handler
	var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write(response)
		require.NoError(t, err)
	})
	handler = CertWithEnrollmentIDMiddleware(handler, testHashCert, &testCertAuthRetriever{}, true, log.NopLogger)
	req, err := http.NewRequest("GET", "/test", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	// we requested enforcement, and did not include a cert, so make sure we get a BadResponse
	if have, want := rr.Code, http.StatusBadRequest; have != want {
		t.Errorf("have: %d, want: %d", have, want)
	}
	req, err = http.NewRequest("GET", "/test", nil)
	if err != nil {
		t.Fatal(err)
	}
	// mock "cert"
	req = req.WithContext(context.WithValue(req.Context(), contextKeyCert{}, &x509.Certificate{}))
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	// now that we have a "cert" included, we should get an OK
	if have, want := rr.Code, http.StatusOK; have != want {
		t.Errorf("have: %d, want: %d", have, want)
	}
	// verify the actual body, too
	if !bytes.Equal(rr.Body.Bytes(), response) {
		t.Error("body not equal")
	}
}

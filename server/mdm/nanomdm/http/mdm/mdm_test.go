package mdm

import (
	"bytes"
	"context"
	"crypto/x509"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service/certauth"
	"github.com/micromdm/nanolib/log"
	"github.com/micromdm/plist"
	"github.com/stretchr/testify/assert"
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

// mockCertAuthService simulates certificate auth errors
type mockCertAuthService struct {
	authenticateErr error
	tokenUpdateErr  error
	commandErr      error
}

func (m *mockCertAuthService) Authenticate(r *mdm.Request, msg *mdm.Authenticate) error {
	return m.authenticateErr
}

func (m *mockCertAuthService) TokenUpdate(r *mdm.Request, msg *mdm.TokenUpdate) error {
	return m.tokenUpdateErr
}

func (m *mockCertAuthService) CheckOut(r *mdm.Request, msg *mdm.CheckOut) error {
	return nil
}

func (m *mockCertAuthService) UserAuthenticate(r *mdm.Request, msg *mdm.UserAuthenticate) ([]byte, error) {
	return nil, nil
}

func (m *mockCertAuthService) SetBootstrapToken(r *mdm.Request, msg *mdm.SetBootstrapToken) error {
	return nil
}

func (m *mockCertAuthService) GetBootstrapToken(r *mdm.Request, msg *mdm.GetBootstrapToken) (*mdm.BootstrapToken, error) {
	return nil, nil
}

func (m *mockCertAuthService) DeclarativeManagement(r *mdm.Request, msg *mdm.DeclarativeManagement) ([]byte, error) {
	return nil, nil
}

func (m *mockCertAuthService) GetToken(r *mdm.Request, msg *mdm.GetToken) (*mdm.GetTokenResponse, error) {
	return nil, nil
}

func (m *mockCertAuthService) CommandAndReportResults(r *mdm.Request, results *mdm.CommandResults) (*mdm.Command, error) {
	return nil, m.commandErr
}

// TestCheckinAndCommandHandler_ErrorHandling verifies handlers return HTTP status codes for errors
func TestCheckinAndCommandHandler_ErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		wrapError      bool // if true, wrap with HTTPStatusError
		serviceError   error
		expectedStatus int
	}{
		{
			name:           "Unwrapped_CertAuth_Error_Returns_500",
			wrapError:      false,
			serviceError:   certauth.ErrNoCertAssoc,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "Wrapped_CertAuth_Error_Returns_403",
			wrapError:      true,
			serviceError:   certauth.ErrNoCertAssoc,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Wrapped_Missing_Cert_Returns_400",
			wrapError:      true,
			serviceError:   certauth.ErrMissingCert,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, isCheckin := range []bool{true, false} {
				subtest := "Command"
				if isCheckin {
					subtest = "Checkin"
				}
				t.Run(subtest, func(t *testing.T) {
					// Setup error
					err := tt.serviceError
					if tt.wrapError {
						if errors.Is(tt.serviceError, certauth.ErrNoCertAssoc) || errors.Is(tt.serviceError, certauth.ErrNoCertReuse) {
							err = service.NewHTTPStatusError(http.StatusForbidden, tt.serviceError)
						} else if errors.Is(tt.serviceError, certauth.ErrMissingCert) {
							err = service.NewHTTPStatusError(http.StatusBadRequest, tt.serviceError)
						}
					}

					mockSvc := &mockCertAuthService{
						tokenUpdateErr: err,
						commandErr:     err,
					}

					// Create handler and request
					var handler http.HandlerFunc
					var body []byte
					var contentType string

					if isCheckin {
						handler = CheckinHandler(mockSvc, log.NopLogger)
						tokenUpdate := &mdm.TokenUpdate{
							Enrollment:  mdm.Enrollment{UDID: "test-udid"},
							MessageType: mdm.MessageType{MessageType: "TokenUpdate"},
						}
						body, err = plist.Marshal(tokenUpdate)
						require.NoError(t, err)
						contentType = "application/x-apple-aspen-mdm-checkin"
					} else {
						handler = CommandAndReportResultsHandler(mockSvc, log.NopLogger)
						cmdResults := &mdm.CommandResults{
							Enrollment:  mdm.Enrollment{UDID: "test-udid"},
							CommandUUID: "test-cmd-uuid",
							Status:      "Acknowledged",
						}
						body, err = plist.Marshal(cmdResults)
						require.NoError(t, err)
						contentType = "application/x-apple-aspen-mdm"
					}

					req := httptest.NewRequest(http.MethodPost, "/mdm", bytes.NewReader(body))
					req.Header.Set("Content-Type", contentType)

					rr := httptest.NewRecorder()
					handler.ServeHTTP(rr, req)

					assert.Equal(t, tt.expectedStatus, rr.Code,
						"Expected status %d, got %d", tt.expectedStatus, rr.Code)
				})
			}
		})
	}
}

// TestErrorResponseBody verifies error response bodies are correct
func TestErrorResponseBody(t *testing.T) {
	mockSvc := &mockCertAuthService{
		tokenUpdateErr: service.NewHTTPStatusError(http.StatusForbidden, certauth.ErrNoCertAssoc),
	}

	handler := CheckinHandler(mockSvc, log.NopLogger)
	tokenUpdate := &mdm.TokenUpdate{
		Enrollment:  mdm.Enrollment{UDID: "test-udid"},
		MessageType: mdm.MessageType{MessageType: "TokenUpdate"},
	}
	body, err := plist.Marshal(tokenUpdate)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/mdm", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-apple-aspen-mdm-checkin")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
	assert.Equal(t, "Forbidden\n", rr.Body.String())
}

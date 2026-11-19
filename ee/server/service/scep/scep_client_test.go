package scep

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/ee/server/service/scep/sceptest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/enrollment"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/x509util"
	smallstepscep "github.com/smallstep/scep"
	"github.com/stretchr/testify/require"
)

func TestEnrollmentClientGetCertificate(t *testing.T) {
	csrKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	csrDER, err := x509util.CreateCertificateRequest(rand.Reader, &x509util.CertificateRequest{
		Subject:           pkix.Name{CommonName: "test-host managementAttestation"},
		ChallengePassword: "challenge",
	}, csrKey)
	require.NoError(t, err)
	csr, err := x509.ParseCertificateRequest(csrDER)
	require.NoError(t, err)

	newClient := func() *EnrollmentClient {
		return NewEnrollmentClient(slog.New(slog.DiscardHandler))
	}

	t.Run("request failures that clear on their own are CA transient, others are not", func(t *testing.T) {
		caCert, _ := sceptest.NewSelfSignedCert(t, "CA", x509.KeyUsageCertSign|x509.KeyUsageKeyEncipherment)
		// failing answers op with status and serves the CA certificate to everything else.
		failing := func(op string, status int) string {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("operation") != op {
					w.Header().Set("Content-Type", "application/x-x509-ca-cert")
					_, _ = w.Write(caCert.Raw)
					return
				}
				w.WriteHeader(status)
			}))
			t.Cleanup(srv.Close)
			return srv.URL + "/scep"
		}
		closed := httptest.NewServer(http.NotFoundHandler())
		closed.Close()

		const getCACert, pkiOperation = "getting CA certificates from SCEP URL: ", "sending SCEP PKCSReq: "
		for name, tc := range map[string]struct {
			url           string
			wantMessage   string
			wantTransient bool
		}{
			"GetCACert no response": {url: closed.URL + "/scep", wantMessage: getCACert, wantTransient: true},
			"GetCACert HTTP 503":    {url: failing("GetCACert", http.StatusServiceUnavailable), wantMessage: getCACert, wantTransient: true},
			"GetCACert HTTP 429":    {url: failing("GetCACert", http.StatusTooManyRequests), wantMessage: getCACert, wantTransient: true},
			"GetCACert HTTP 404":    {url: failing("GetCACert", http.StatusNotFound), wantMessage: getCACert, wantTransient: false},
			"GetCACert HTTP 401":    {url: failing("GetCACert", http.StatusUnauthorized), wantMessage: getCACert, wantTransient: false},
			"PKIOperation HTTP 500": {url: failing("PKIOperation", http.StatusInternalServerError), wantMessage: pkiOperation, wantTransient: true},
			"PKIOperation HTTP 408": {url: failing("PKIOperation", http.StatusRequestTimeout), wantMessage: pkiOperation, wantTransient: true},
			"PKIOperation HTTP 403": {url: failing("PKIOperation", http.StatusForbidden), wantMessage: pkiOperation, wantTransient: false},
		} {
			t.Run(name, func(t *testing.T) {
				_, err := newClient().GetCertificate(t.Context(), tc.url, csr)
				require.ErrorContains(t, err, tc.wantMessage)
				transient, ok := errors.AsType[fleet.CertificateAuthorityTransientError](err)
				require.Equal(t, tc.wantTransient, ok)
				if ok {
					require.Equal(t, enrollmentRetryAfterSeconds, transient.RetryAfterSeconds)
				}
			})
		}
	})

	t.Run("a FAILURE carries the challenge hint", func(t *testing.T) {
		srv := sceptest.NewCraftedServer(t, func(s *sceptest.CraftedServer, req *smallstepscep.PKIMessage) ([]byte, error) {
			return s.Fail(req, smallstepscep.BadRequest)
		})
		_, err := newClient().GetCertificate(t.Context(), srv.URL, csr)
		require.EqualError(t, err, "SCEP server rejected the request: status FAILURE with fail info badRequest (2); "+
			"if this certificate authority requires a challenge, include it as the CSR's challengePassword attribute")
		_, ok := errors.AsType[enrollment.RejectedError](err)
		require.True(t, ok)
	})

	t.Run("a PENDING has no challenge hint", func(t *testing.T) {
		srv := sceptest.NewCraftedServer(t, func(s *sceptest.CraftedServer, req *smallstepscep.PKIMessage) ([]byte, error) {
			return s.CertRep(t, req, smallstepscep.PENDING, nil), nil
		})
		_, err := newClient().GetCertificate(t.Context(), srv.URL, csr)
		require.EqualError(t, err, "SCEP server rejected the request: status PENDING; requests that need manual approval are not supported")
	})

	// countingCACert serves the CA certificate and counts GetCACert requests.
	countingCACert := func(count *atomic.Int64) sceptest.CraftedOption {
		return sceptest.WithCACertResponse(func(s *sceptest.CraftedServer) ([]byte, int) {
			count.Add(1)
			return s.CACert.Raw, 1
		})
	}

	t.Run("CA certificates are fetched once per SCEP URL until the cache expires", func(t *testing.T) {
		var fetchesA, fetchesB atomic.Int64
		srvA := sceptest.NewCraftedServer(t, nil, countingCACert(&fetchesA))
		srvB := sceptest.NewCraftedServer(t, nil, countingCACert(&fetchesB))
		c := newClient()
		mockClock := clock.NewMockClock()
		c.clock = mockClock

		for range 3 {
			_, err := c.GetCertificate(t.Context(), srvA.URL, csr)
			require.NoError(t, err)
		}
		require.EqualValues(t, 1, fetchesA.Load())

		_, err := c.GetCertificate(t.Context(), srvB.URL, csr)
		require.NoError(t, err)
		require.EqualValues(t, 1, fetchesB.Load())

		mockClock.AddTime(caCertsCacheTTL + time.Second)
		_, err = c.GetCertificate(t.Context(), srvA.URL, csr)
		require.NoError(t, err)
		require.EqualValues(t, 2, fetchesA.Load())
	})

	t.Run("a failed enrollment drops the cached CA certificates", func(t *testing.T) {
		var fetches atomic.Int64
		var reject atomic.Bool
		reject.Store(true)
		srv := sceptest.NewCraftedServer(t, func(s *sceptest.CraftedServer, req *smallstepscep.PKIMessage) ([]byte, error) {
			if reject.Load() {
				return s.Fail(req, smallstepscep.BadRequest)
			}
			return s.Succeed(t, req)
		}, countingCACert(&fetches))
		c := newClient()

		_, err := c.GetCertificate(t.Context(), srv.URL, csr)
		require.Error(t, err)
		reject.Store(false)
		_, err = c.GetCertificate(t.Context(), srv.URL, csr)
		require.NoError(t, err)
		require.EqualValues(t, 2, fetches.Load())
	})

	t.Run("a failed GetCACert is not cached", func(t *testing.T) {
		var fetches atomic.Int64
		var broken atomic.Bool
		broken.Store(true)
		srv := sceptest.NewCraftedServer(t, nil, sceptest.WithCACertResponse(func(s *sceptest.CraftedServer) ([]byte, int) {
			fetches.Add(1)
			if broken.Load() {
				return []byte("not a certificate"), 1
			}
			return s.CACert.Raw, 1
		}))
		c := newClient()

		_, err := c.GetCertificate(t.Context(), srv.URL, csr)
		require.ErrorContains(t, err, "parsing CA certificates from SCEP URL")
		broken.Store(false)
		_, err = c.GetCertificate(t.Context(), srv.URL, csr)
		require.NoError(t, err)
		require.EqualValues(t, 2, fetches.Load())
	})
}

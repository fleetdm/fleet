package mdm

import (
	"context"
	"crypto/x509"
	"errors"
	"net/http"
	"net/url"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/cryptoutil"
	mdmhttp "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/http"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/log"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/log/ctxlog"
)

type contextKeyCert struct{}

// CertExtractPEMHeaderMiddleware extracts the MDM enrollment identity
// certificate from the request into the HTTP request context. It looks
// at the request header which should be a URL-encoded PEM certificate.
//
// This is ostensibly to support Nginx' $ssl_client_escaped_cert in a
// proxy_set_header directive. Though any reverse proxy setting a
// similar header could be used, of course.
func CertExtractPEMHeaderMiddleware(next http.Handler, header string, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		escapedCert := r.Header.Get(header)
		if escapedCert == "" {
			logger.Debug("msg", "empty header", "header", header)
			next.ServeHTTP(w, r)
			return
		}
		pemCert, err := url.QueryUnescape(escapedCert)
		if err != nil {
			logger.Info("msg", "unescaping header", "header", header, "err", err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		cert, err := cryptoutil.DecodePEMCertificate([]byte(pemCert))
		if err != nil {
			logger.Info("msg", "decoding cert", "header", header, "err", err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		ctx := context.WithValue(r.Context(), contextKeyCert{}, cert)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// CertExtractTLSMiddleware extracts the MDM enrollment identity
// certificate from the request into the HTTP request context. It looks
// at the TLS peer certificate in the request.
func CertExtractTLSMiddleware(next http.Handler, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil || len(r.TLS.PeerCertificates) < 1 {
			ctxlog.Logger(r.Context(), logger).Debug(
				"msg", "no TLS peer certificate",
			)
			next.ServeHTTP(w, r)
			return
		}
		cert := r.TLS.PeerCertificates[0]
		ctx := context.WithValue(r.Context(), contextKeyCert{}, cert)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// CertExtractMdmSignatureMiddleware extracts the MDM enrollment
// identity certificate from the request into the HTTP request context.
// It tries to verify the Mdm-Signature header on the request.
//
// This middleware does not error if a certificate is not found. It
// will, however, error with an HTTP 400 status if the signature
// verification fails.
func CertExtractMdmSignatureMiddleware(next http.Handler, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		mdmSig := r.Header.Get("Mdm-Signature")
		if mdmSig == "" {
			logger.Debug("msg", "empty Mdm-Signature header")
			next.ServeHTTP(w, r)
			return
		}
		b, err := mdmhttp.ReadAllAndReplaceBody(r)
		if err != nil {
			logger.Info("msg", "reading body", "err", err)
			var toErr interface{ Timeout() bool }
			if errors.As(err, &toErr) && toErr.Timeout() {
				http.Error(w, http.StatusText(http.StatusRequestTimeout), http.StatusRequestTimeout)
				return
			}
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		cert, err := cryptoutil.VerifyMdmSignature(mdmSig, b)
		if err != nil {
			logger.Info("msg", "verifying Mdm-Signature header", "err", err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		ctx := context.WithValue(r.Context(), contextKeyCert{}, cert)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// GetCert retrieves the MDM enrollment identity certificate
// from the HTTP request context.
func GetCert(ctx context.Context) *x509.Certificate {
	cert, _ := ctx.Value(contextKeyCert{}).(*x509.Certificate)
	return cert
}

// CertVerifier is a simple interface for verifying a certificate.
type CertVerifier interface {
	Verify(*x509.Certificate) error
}

// CertVerifyMiddleware checks the MDM certificate against verifier and
// returns an error if it fails.
//
// We deliberately do not reply with 401 as this may cause unintentional
// MDM unenrollments in the case of bugs or something going wrong.
func CertVerifyMiddleware(next http.Handler, verifier CertVerifier, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := verifier.Verify(GetCert(r.Context())); err != nil {
			ctxlog.Logger(r.Context(), logger).Info(
				"msg", "error verifying MDM certificate",
				"err", err,
			)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r)
	}
}

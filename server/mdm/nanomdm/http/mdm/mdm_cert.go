package mdm

import (
	"context"
	"crypto/x509"
	"errors"
	"net/http"
	"net/url"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/cryptoutil"
	mdmhttp "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/http"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage"

	"github.com/micromdm/nanolib/log"
	"github.com/micromdm/nanolib/log/ctxlog"
)

type contextKeyCert struct{}

type contextEnrollmentID struct{}

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

// sigLogConfig is a configuration struct for CertExtractMdmSignatureMiddleware.
type sigLogConfig struct {
	logger log.Logger
	always bool
	errors bool
}

// SigLogOption sets configurations.
type SigLogOption func(*sigLogConfig)

// SigLogWithLogger sets the logger to use when logging with the MDM signature header.
func SigLogWithLogger(logger log.Logger) SigLogOption {
	return func(c *sigLogConfig) {
		c.logger = logger
	}
}

// SigLogWithLogAlways always logs the raw Mdm-Signature header.
func SigLogWithLogAlways(always bool) SigLogOption {
	return func(c *sigLogConfig) {
		c.always = always
	}
}

// SigLogWithLogErrors logs the raw Mdm-Signature header when errors occur.
func SigLogWithLogErrors(errors bool) SigLogOption {
	return func(c *sigLogConfig) {
		c.errors = errors
	}
}

// MdmSignatureVerifier verifies Apple Mdm-Signature headers and extracts certificates.
type MdmSignatureVerifier interface {
	// VerifyMdmSignature verifies an Apple MDM "Mdm-Signature" header and returns the signing certificate.
	// See https://developer.apple.com/documentation/devicemanagement/implementing_device_management/managing_certificates_for_mdm_servers_and_devices
	// section "Pass an Identity Certificate Through a Proxy."
	VerifyMdmSignature(header string, body []byte) (*x509.Certificate, error)
}

// MdmSignatureVerifierFunc is an adapter for verifying Apple MDM "Mdm-Signature" headers.
type MdmSignatureVerifierFunc func(header string, body []byte) (*x509.Certificate, error)

// VerifyMdmSignature calls v with header and body.
func (v MdmSignatureVerifierFunc) VerifyMdmSignature(header string, body []byte) (*x509.Certificate, error) {
	return v(header, body)
}

// CertExtractMdmSignatureMiddleware extracts the MDM enrollment
// identity certificate from the request into the HTTP request context.
// It tries to verify the Mdm-Signature header on the request.
//
// This middleware does not error if a certificate is not found. It
// will, however, error with an HTTP 400 status if the signature
// verification fails.
func CertExtractMdmSignatureMiddleware(next http.Handler, verifier MdmSignatureVerifier, opts ...SigLogOption) http.HandlerFunc {
	if verifier == nil {
		panic("nil verifier")
	}
	config := &sigLogConfig{logger: log.NopLogger}
	for _, opt := range opts {
		opt(config)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), config.logger)
		mdmSig := r.Header.Get("Mdm-Signature")
		if mdmSig == "" {
			logger.Debug("msg", "empty Mdm-Signature header")
			next.ServeHTTP(w, r)
			return
		}
		if config.errors || config.always {
			logger = logger.With("mdm-signature", mdmSig)
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
		cert, err := verifier.VerifyMdmSignature(mdmSig, b)
		if err != nil {
			logger.Info("msg", "verifying Mdm-Signature header", "err", err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		} else if config.always {
			logger.Debug("msg", "verifying Mdm-Signature header")
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
	Verify(context.Context, *x509.Certificate) error
}

// CertVerifyMiddleware checks the MDM certificate against verifier and
// returns an error if it fails.
//
// We deliberately do not reply with 401 as this may cause unintentional
// MDM unenrollments in the case of bugs or something going wrong.
func CertVerifyMiddleware(next http.Handler, verifier CertVerifier, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := verifier.Verify(r.Context(), GetCert(r.Context())); err != nil {
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

// GetEnrollmentID retrieves the MDM enrollment ID from ctx.
func GetEnrollmentID(ctx context.Context) string {
	id, _ := ctx.Value(contextEnrollmentID{}).(string)
	return id
}

type HashFn func(*x509.Certificate) string

// CertWithEnrollmentIDMiddleware tries to associate the enrollment ID to the request context.
// It does this by looking up the certificate on the context, hashing it with
// hasher, looking up the hash in storage, and setting the ID on the context.
//
// The next handler will be called even if cert or ID is not found unless
// enforce is true. This way next is able to use the existence of the ID on
// the context to make its own decisions.
func CertWithEnrollmentIDMiddleware(next http.Handler, hasher HashFn, store storage.CertAuthRetriever, enforce bool,
	logger log.Logger) http.HandlerFunc {
	if store == nil || hasher == nil {
		panic("store and hasher must not be nil")
	}
	return func(w http.ResponseWriter, r *http.Request) {
		cert := GetCert(r.Context())
		if cert == nil {
			if enforce {
				ctxlog.Logger(r.Context(), logger).Info(
					"err", "missing certificate",
				)
				// we cannot send a 401 to the client as it has MDM protocol semantics
				// i.e. the device may unenroll
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusBadRequest)
				return
			}
			ctxlog.Logger(r.Context(), logger).Debug(
				"msg", "missing certificate",
			)
			next.ServeHTTP(w, r)
			return
		}
		id, err := store.EnrollmentFromHash(r.Context(), hasher(cert))
		if err != nil {
			ctxlog.Logger(r.Context(), logger).Info(
				"msg", "retreiving enrollment from hash",
				"err", err,
			)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if id == "" {
			if enforce {
				ctxlog.Logger(r.Context(), logger).Info(
					"err", "missing enrollment id",
				)
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusBadRequest)
				return
			}
			ctxlog.Logger(r.Context(), logger).Debug(
				"msg", "missing enrollment id",
			)
			next.ServeHTTP(w, r)
			return
		}
		ctx := context.WithValue(r.Context(), contextEnrollmentID{}, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

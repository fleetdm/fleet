package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"net/url"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/cryptoutil"
)

// HTTP paths for the Apple Platform SSO endpoints. All but the AASA path live
// under /api/mdm/apple/psso and are registered on the unauthenticated
// endpointer (see handler.go); auth is protocol-level (signed JWTs verified
// against registered device keys). The AASA document must be served at the
// /.well-known path(Apple requirement), so it stays on the root *http.ServeMux
// (see registerPSSO).
const (
	pssoNoncePath        = "/api/mdm/apple/psso/nonce"
	pssoRegistrationPath = "/api/mdm/apple/psso/registration"
	pssoTokenPath        = "/api/mdm/apple/psso/token" //nolint:gosec // G101 false positive, this is a URL path
	pssoJWKSPath         = "/api/mdm/apple/psso/jwks"
	pssoAASAPath         = "/.well-known/apple-app-site-association"
)

// pssoContentTypeLoginResponse is the Content-Type Apple's PSSO framework
// expects on token endpoint responses.
const pssoContentTypeLoginResponse = "application/platformsso-login-response+jwt"

////////////////////////////////////////////////////////////////////////////////
// POST /api/mdm/apple/psso/nonce
////////////////////////////////////////////////////////////////////////////////

type pssoNonceRequest struct{}

// DecodeBody drains and discards the request body. Apple's AppSSOAgent POSTs a
// urlencoded grant_type=srv_challenge form to the nonce endpoint, but Fleet
// needs nothing from it — it just mints a nonce. This method must exist so the
// endpoint framework routes the form body here instead of trying to decode as JSON.
func (pssoNonceRequest) DecodeBody(_ context.Context, r io.Reader, _ url.Values, _ []*x509.Certificate) error {
	_, _ = io.Copy(io.Discard, r)
	return nil
}

type pssoNonceResponse struct {
	// Nonce is PascalCase on the wire: Apple's AppSSOAgent consumes this
	// response directly and expects the capitalized key.
	Nonce string `json:"Nonce"`
	Err   error  `json:"error,omitempty"`
}

func (r pssoNonceResponse) Error() error { return r.Err }

func pssoNonceEndpoint(ctx context.Context, _ any, svc fleet.Service) (fleet.Errorer, error) {
	nonce, err := svc.PSSONonce(ctx)
	if err != nil {
		return pssoNonceResponse{Err: err}, nil
	}
	return pssoNonceResponse{Nonce: nonce}, nil
}

////////////////////////////////////////////////////////////////////////////////
// POST /api/mdm/apple/psso/registration
////////////////////////////////////////////////////////////////////////////////

type pssoRegistrationRequest struct {
	fleet.PSSODeviceRegistrationRequest
}

// DecodeBody parses the urlencoded form the extension POSTs. The reader is
// already capped by the endpointer's request body size limit.
func (req *pssoRegistrationRequest) DecodeBody(ctx context.Context, r io.Reader, _ url.Values, _ []*x509.Certificate) error {
	form, err := parseURLEncodedForm(ctx, r)
	if err != nil {
		return err
	}
	req.DeviceUUID = form.Get("device_uuid")
	req.DeviceSigningKey = form.Get("device_signing_key")
	req.DeviceEncryptionKey = form.Get("device_encryption_key")
	req.SigningKeyID = form.Get("signing_key_id")
	req.EncryptionKeyID = form.Get("encryption_key_id")
	req.RegistrationToken = form.Get("registration_token")
	return nil
}

type pssoRegistrationResponse struct {
	Err error `json:"error,omitempty"`
}

func (r pssoRegistrationResponse) Error() error { return r.Err }

func (r pssoRegistrationResponse) Status() int { return http.StatusNoContent }

func pssoRegistrationEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*pssoRegistrationRequest)
	if err := svc.PSSORegisterDevice(ctx, req.PSSODeviceRegistrationRequest); err != nil {
		return pssoRegistrationResponse{Err: err}, nil
	}
	return pssoRegistrationResponse{}, nil
}

////////////////////////////////////////////////////////////////////////////////
// POST /api/mdm/apple/psso/token
////////////////////////////////////////////////////////////////////////////////

type pssoTokenRequest struct {
	Assertion string
}

// DecodeBody parses the OAuth jwt-bearer-style urlencoded form whose
// `assertion` field holds the compact JWS signed by the device. The JWT must
// be extracted from the form, not read from the raw body.
func (req *pssoTokenRequest) DecodeBody(ctx context.Context, r io.Reader, _ url.Values, _ []*x509.Certificate) error {
	form, err := parseURLEncodedForm(ctx, r)
	if err != nil {
		return err
	}
	req.Assertion = form.Get("assertion")
	if req.Assertion == "" {
		return &fleet.BadRequestError{Message: "psso token: missing assertion"}
	}
	return nil
}

type pssoTokenResponse struct {
	Err error `json:"error,omitempty"`
	jwe []byte
}

func (r pssoTokenResponse) Error() error { return r.Err }

func (r pssoTokenResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Type", pssoContentTypeLoginResponse)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if n, err := w.Write(r.jwe); err != nil {
		logging.WithExtras(ctx, "err", err, "written", n)
	}
}

func pssoTokenEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*pssoTokenRequest)
	out, err := svc.PSSOToken(ctx, []byte(req.Assertion))
	if err != nil {
		return pssoTokenResponse{Err: err}, nil
	}
	return pssoTokenResponse{jwe: out}, nil
}

////////////////////////////////////////////////////////////////////////////////
// GET /api/mdm/apple/psso/jwks
////////////////////////////////////////////////////////////////////////////////

type pssoJWKSRequest struct{}

type pssoJWKSResponse struct {
	Err  error `json:"error,omitempty"`
	body []byte
}

func (r pssoJWKSResponse) Error() error { return r.Err }

func (r pssoJWKSResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/jwk-set+json")
	if n, err := w.Write(r.body); err != nil {
		logging.WithExtras(ctx, "err", err, "written", n)
	}
}

func pssoJWKSEndpoint(ctx context.Context, _ any, svc fleet.Service) (fleet.Errorer, error) {
	body, err := svc.PSSOJWKS(ctx)
	if err != nil {
		return pssoJWKSResponse{Err: err}, nil
	}
	return pssoJWKSResponse{body: body}, nil
}

////////////////////////////////////////////////////////////////////////////////
// GET /.well-known/apple-app-site-association
////////////////////////////////////////////////////////////////////////////////

// pssoAASAHandler serves the Apple App Site Association JSON Apple's CDN
// fetches to validate the extension's `authsrv:` entitlement against this
// hostname. It stays a raw root-mux handler because the path is fixed by
// Apple's spec and can't live under /api.
func pssoAASAHandler(svc fleet.Service, _ *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, err := svc.PSSOAASA(ctx)
		if err != nil {
			encodeError(ctx, err, w)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodHead {
			return
		}
		_, _ = w.Write(body)
	})
}

// parseURLEncodedForm reads an x-www-form-urlencoded body from an
// already-size-limited reader.
func parseURLEncodedForm(ctx context.Context, r io.Reader) (url.Values, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "read psso form body")
	}
	form, err := url.ParseQuery(string(raw))
	if err != nil {
		return nil, &fleet.BadRequestError{Message: "invalid urlencoded form body", InternalErr: err}
	}
	return form, nil
}

// ----- core-side service-method stubs --------------------------------------
//
// All PSSO business logic lives in ee/server/service. The core stubs below
// return fleet.ErrMissingLicense so unlicensed Fleet deployments respond with
// a well-formed error instead of 404. The ee implementation overrides these
// methods on the embedded core Service.

func (svc *Service) PSSONonce(ctx context.Context) (string, error) {
	// skipauth: Implementation returns only the license error; nothing to authorize.
	svc.authz.SkipAuthorization(ctx)
	return "", fleet.ErrMissingLicense
}

func (svc *Service) PSSORegisterDevice(ctx context.Context, _ fleet.PSSODeviceRegistrationRequest) error {
	// skipauth: Implementation returns only the license error; nothing to authorize.
	svc.authz.SkipAuthorization(ctx)
	return fleet.ErrMissingLicense
}

func (svc *Service) PSSOToken(ctx context.Context, _ []byte) ([]byte, error) {
	// skipauth: Implementation returns only the license error; nothing to authorize.
	svc.authz.SkipAuthorization(ctx)
	return nil, fleet.ErrMissingLicense
}

func (svc *Service) PSSOJWKS(ctx context.Context) ([]byte, error) {
	// skipauth: Implementation returns only the license error; nothing to authorize.
	svc.authz.SkipAuthorization(ctx)
	return nil, fleet.ErrMissingLicense
}

func (svc *Service) PSSOAASA(ctx context.Context) ([]byte, error) {
	// skipauth: Implementation returns only the license error; nothing to authorize.
	svc.authz.SkipAuthorization(ctx)
	return nil, fleet.ErrMissingLicense
}

// ----- PSSO asset bootstrap -------------------------------------------------
//
// The signing key and CA are pure crypto + datastore work, so they live here in
// core (callable from ModifyAppConfig) rather than in ee/. The ee service only
// loads them back, using the standard PEM encodings written below.

// pssoCAValidYears is the lifetime of the self-signed Platform SSO CA, matching
// other CAs in fleet and minted once, when the feature is first configured.
const pssoCAValidYears = 10

// bootstrapPSSOAssets ensures the Platform SSO signing key and its CA certificate
// (which is signed by the signing key) exist in mdm_config_assets. It runs when the
// feature is configured and is idempotent: existing assets are never regenerated, so
// the signing key (published via JWKS) and the CA remain stable.
func bootstrapPSSOAssets(ctx context.Context, ds fleet.Datastore) error {
	assets, err := ds.GetAllMDMConfigAssetsByName(ctx,
		[]fleet.MDMAssetName{fleet.MDMAssetPSSOSigningKey, fleet.MDMAssetPSSOCACert},
		nil,
	)
	// A partial result (one asset present, the other missing) returns an error
	// alongside the assets it did find; only a hard error with nothing usable is fatal.
	if err != nil && !fleet.IsNotFound(err) && len(assets) == 0 {
		return ctxerr.Wrap(ctx, err, "load psso assets")
	}

	haveKey := false
	haveCA := false
	if assets != nil {
		_, haveKey = assets[fleet.MDMAssetPSSOSigningKey]
		_, haveCA = assets[fleet.MDMAssetPSSOCACert]
	}
	if haveKey && haveCA {
		return nil
	}

	// Throw an error because this is an inconsistent state - the CA was created apparently with a different signing key?
	if haveCA && !haveKey {
		return ctxerr.New(ctx, "psso ca certificate exists but signing key is missing")
	}

	signingKey, err := pssoSigningKeyFromAssets(assets)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "parse existing psso signing key")
	}

	var toInsert []fleet.MDMConfigAsset
	if signingKey == nil {
		signingKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "generate psso signing key")
		}
		der, err := x509.MarshalECPrivateKey(signingKey)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "marshal psso signing key")
		}
		toInsert = append(toInsert, fleet.MDMConfigAsset{
			Name:  fleet.MDMAssetPSSOSigningKey,
			Value: pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der}),
		})
	}
	if !haveCA {
		caDER, err := selfSignPSSOCACert(signingKey)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "create psso ca certificate")
		}
		toInsert = append(toInsert, fleet.MDMConfigAsset{
			Name:  fleet.MDMAssetPSSOCACert,
			Value: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER}),
		})
	}

	if err := ds.InsertMDMConfigAssets(ctx, toInsert, nil); err != nil {
		return ctxerr.Wrap(ctx, err, "insert psso assets")
	}
	return nil
}

// pssoSigningKeyFromAssets parses the PSSO signing key out of a loaded asset map,
// returning (nil, nil) when it isn't present so the caller can mint a fresh one.
func pssoSigningKeyFromAssets(assets map[fleet.MDMAssetName]fleet.MDMConfigAsset) (*ecdsa.PrivateKey, error) {
	asset, ok := assets[fleet.MDMAssetPSSOSigningKey]
	if !ok || len(asset.Value) == 0 {
		return nil, nil
	}
	block, _ := pem.Decode(asset.Value)
	if block == nil {
		return nil, errors.New("psso signing key: pem decode returned nil block")
	}
	return x509.ParseECPrivateKey(block.Bytes)
}

// selfSignPSSOCACert self-signs a Platform SSO CA certificate over signingKey.
// Serial 1 matches Fleet's other self-signed CA roots (server/mdm/scep/depot):
// the CA is the only self-signed certificate this key ever produces, so the
// serial is unique by construction.
func selfSignPSSOCACert(signingKey *ecdsa.PrivateKey) ([]byte, error) {
	subjectKeyID, err := cryptoutil.GenerateSubjectKeyID(&signingKey.PublicKey)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Fleet PSSO CA"},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.AddDate(pssoCAValidYears, 0, 0),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		MaxPathLen:            0,
		MaxPathLenZero:        true,
		SubjectKeyId:          subjectKeyID,
	}
	return x509.CreateCertificate(rand.Reader, tmpl, tmpl, &signingKey.PublicKey, signingKey)
}

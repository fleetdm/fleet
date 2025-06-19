package httpsig

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/log"
	"github.com/remitly-oss/httpsig-go"
)

type HTTPSig struct {
	ds     fleet.Datastore
	logger log.Logger
}

func NewHTTPSig(ds fleet.Datastore, logger log.Logger) *HTTPSig {
	return &HTTPSig{
		ds:     ds,
		logger: logger,
	}
}

func (h *HTTPSig) Verifier() (*httpsig.Verifier, error) {
	return httpsig.NewVerifier(h, httpsig.VerifyProfile{
		SignatureLabel:     httpsig.DefaultSignatureLabel,
		AllowedAlgorithms:  []httpsig.Algorithm{httpsig.Algo_ECDSA_P256_SHA256, httpsig.Algo_ECDSA_P384_SHA384},
		RequiredFields:     httpsig.Fields("@method", "@target-uri", "content-digest"),
		RequiredMetadata:   []httpsig.Metadata{httpsig.MetaCreated, httpsig.MetaKeyID, httpsig.MetaNonce},
		DisallowedMetadata: []httpsig.Metadata{httpsig.MetaAlgorithm}, // The algorithm should be looked up from the keyid not an explicit setting.
	})
}

func (h *HTTPSig) FetchByKeyID(ctx context.Context, _ http.Header, keyID string) (httpsig.KeySpecer, error) {
	keyIDInt, err := strconv.ParseUint(keyID, 16, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid hex key ID: %w", err)
	}
	pemData, err := h.ds.CertBySerialNumber(ctx, keyIDInt)
	if err != nil {
		return nil, fmt.Errorf("loading certificate: %w", err)
	}

	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing certificate: %w", err)
	}

	var algo httpsig.Algorithm
	switch pub := cert.PublicKey.(type) {
	case *ecdsa.PublicKey:
		switch pub.Curve {
		case elliptic.P256():
			algo = httpsig.Algo_ECDSA_P256_SHA256
		case elliptic.P384():
			algo = httpsig.Algo_ECDSA_P384_SHA384
		default:
			return nil, fmt.Errorf("unsupported elliptic curve: %s", pub.Curve.Params().Name)
		}
	default:
		return nil, fmt.Errorf("unsupported public key type: %T", cert.PublicKey)
	}

	return &httpsig.KeySpec{
		KeyID:  keyID,
		Algo:   algo,
		PubKey: cert.PublicKey,
	}, nil
}

func (h *HTTPSig) Fetch(_ context.Context, _ http.Header, _ httpsig.MetadataProvider) (httpsig.KeySpecer, error) {
	return nil, errors.New("not implemented")
}

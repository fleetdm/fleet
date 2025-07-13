package httpsig

import (
	"context"
	"crypto/elliptic"
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
		SignatureLabel:    httpsig.DefaultSignatureLabel,
		AllowedAlgorithms: []httpsig.Algorithm{httpsig.Algo_ECDSA_P256_SHA256, httpsig.Algo_ECDSA_P384_SHA384},
		// We are not using @target-uri in the signature so that we don't run into issues with HTTPS forwarding and proxies (http vs https).
		RequiredFields:     httpsig.Fields("@method", "@authority", "@path", "@query", "content-digest"),
		RequiredMetadata:   []httpsig.Metadata{httpsig.MetaKeyID, httpsig.MetaCreated, httpsig.MetaNonce},
		DisallowedMetadata: []httpsig.Metadata{httpsig.MetaAlgorithm}, // The algorithm should be looked up from the keyid not an explicit setting.
	})
}

func (h *HTTPSig) FetchByKeyID(ctx context.Context, _ http.Header, keyID string) (httpsig.KeySpecer, error) {
	keyIDInt, err := strconv.ParseUint(keyID, 16, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid hex key ID: %w", err)
	}
	identityCert, err := h.ds.GetHostIdentityCertBySerialNumber(ctx, keyIDInt)
	if err != nil {
		return nil, fmt.Errorf("loading certificate: %w", err)
	}
	publicKey, err := identityCert.UnmarshalPublicKey()
	if err != nil {
		return nil, fmt.Errorf("unmarshaling public key: %w", err)
	}

	var algo httpsig.Algorithm
	switch publicKey.Curve {
	case elliptic.P256():
		algo = httpsig.Algo_ECDSA_P256_SHA256
	case elliptic.P384():
		algo = httpsig.Algo_ECDSA_P384_SHA384
	default:
		return nil, fmt.Errorf("unsupported elliptic curve: %s", publicKey.Curve.Params().Name)
	}

	return &httpsig.KeySpec{
		KeyID:  keyID,
		Algo:   algo,
		PubKey: publicKey,
	}, nil
}

func (h *HTTPSig) Fetch(_ context.Context, _ http.Header, _ httpsig.MetadataProvider) (httpsig.KeySpecer, error) {
	return nil, errors.New("not implemented")
}

package httpsig

import (
	"context"
	"crypto/elliptic"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/fleetdm/fleet/v4/ee/server/service/hostidentity/types"
	"github.com/fleetdm/fleet/v4/pkg/fleethttpsig"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/log"
	"github.com/remitly-oss/httpsig-go"
)

type HTTPSig struct {
	ds     fleet.Datastore
	logger log.Logger
}

type KeySpecer struct {
	hostIdentityCert types.HostIdentityCertificate
	keySpec          httpsig.KeySpec
}

func (k KeySpecer) KeySpec() (httpsig.KeySpec, error) {
	return k.keySpec, nil
}

// _ ensures that KeySpecer implements the httpsig.KeySpecer interface.
var _ httpsig.KeySpecer = KeySpecer{}

func NewHTTPSig(ds fleet.Datastore, logger log.Logger) *HTTPSig {
	return &HTTPSig{
		ds:     ds,
		logger: logger,
	}
}

// _ ensures that HTTPSig implements the httpsig.KeyFetcher interface.
var _ httpsig.KeyFetcher = (*HTTPSig)(nil)

func (h *HTTPSig) Verifier() (*httpsig.Verifier, error) {
	return fleethttpsig.Verifier(h)
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

	return &KeySpecer{
		hostIdentityCert: *identityCert,
		keySpec: httpsig.KeySpec{
			KeyID:  keyID,
			Algo:   algo,
			PubKey: publicKey,
		},
	}, nil
}

func (h *HTTPSig) Fetch(_ context.Context, _ http.Header, _ httpsig.MetadataProvider) (httpsig.KeySpecer, error) {
	return nil, errors.New("not implemented")
}

// VerifyHostIdentity checks that host identity certificate matches the node key and host ID.
// Host identity cert is used for TPM-backed HTTP message signatures.
// If the host has one, then all agent traffic should have HTTP message signatures unless specified otherwise.
// The host identity certificate must match the host's node key.
func VerifyHostIdentity(ctx context.Context, ds fleet.Datastore, host *fleet.Host) error {
	hostIdentityCert, ok := FromContext(ctx)
	if !ok {
		return errors.New("authentication error: missing host identity certificate")
	}
	if host.OsqueryHostID == nil || *host.OsqueryHostID != hostIdentityCert.CommonName {
		return errors.New("authentication error: http message signature does not match node key")
	}
	if hostIdentityCert.HostID == nil {
		return fmt.Errorf("authentication error: found host identity certificate without host ID. "+
			"This should not happen since host ID for a certificate should be set at enrollment. identifier/CN: %s host ID: %d",
			hostIdentityCert.CommonName, host.ID)
	}
	if *hostIdentityCert.HostID != host.ID {
		return errors.New("authentication error: http message signature does not match host ID")
	}
	return nil
}

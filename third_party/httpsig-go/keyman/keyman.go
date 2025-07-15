// keyman provides key management functionality
package keyman

import (
	"context"
	"fmt"
	"net/http"

	"github.com/remitly-oss/httpsig-go"
)

// KeyFetchInMemory implements KeyFetcher for public keys stored in memory.
type KeyFetchInMemory struct {
	pubkeys map[string]httpsig.KeySpec
}

func NewKeyFetchInMemory(pubkeys map[string]httpsig.KeySpec) *KeyFetchInMemory {
	if pubkeys == nil {
		pubkeys = map[string]httpsig.KeySpec{}
	}
	return &KeyFetchInMemory{pubkeys}
}

func (kf *KeyFetchInMemory) FetchByKeyID(ctx context.Context, rh http.Header, keyID string) (httpsig.KeySpecer, error) {
	ks, found := kf.pubkeys[keyID]
	if !found {
		return nil, fmt.Errorf("Key for keyid '%s' not found", keyID)
	}
	return ks, nil
}

func (kf *KeyFetchInMemory) Fetch(context.Context, http.Header, httpsig.MetadataProvider) (httpsig.KeySpecer, error) {
	return nil, fmt.Errorf("Fetch without keyid not supported")
}

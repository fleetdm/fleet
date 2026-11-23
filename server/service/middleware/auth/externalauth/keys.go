package externalauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/MicahParks/jwkset"
	"github.com/fleetdm/fleet/v4/server/config"
)

const (
	maxJWKSResponseBytes = 1 << 20
	maxRetryInterval     = 30 * time.Second
	maxKeyFetchRedirects = 5
)

// issuer holds one configured issuer and its cached signing keys. Keys are only ever
// replaced by the timer-driven refresh, never because a request asked for it, so request
// input cannot make Fleet contact the issuer.
type issuer struct {
	cfg    config.JWTIssuerConfig
	algs   map[string]struct{}
	client *http.Client

	mu      sync.RWMutex
	keys    map[string]jwkset.JWK
	jwksURL string
}

func newIssuer(cfg config.JWTIssuerConfig, client *http.Client) *issuer {
	algs := make(map[string]struct{}, len(cfg.AllowedAlgorithms))
	for _, a := range cfg.AllowedAlgorithms {
		algs[a] = struct{}{}
	}
	return &issuer{cfg: cfg, algs: algs, client: client, jwksURL: cfg.JWKSURL}
}

func (i *issuer) key(kid string) (jwkset.JWK, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	k, ok := i.keys[kid]
	return k, ok
}

func (i *issuer) hasKeys() bool {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.keys != nil
}

// refresh fetches the issuer's JWKS. On failure the previously cached keys stay in use.
func (i *issuer) refresh(ctx context.Context) error {
	i.mu.RLock()
	jwksURL := i.jwksURL
	i.mu.RUnlock()
	if jwksURL == "" {
		var err error
		if jwksURL, err = i.discover(ctx); err != nil {
			return err
		}
	}

	var set jwkset.JWKSMarshal
	if err := i.getJSON(ctx, jwksURL, &set); err != nil {
		return err
	}
	keys := make(map[string]jwkset.JWK, len(set.Keys))
	for _, m := range set.Keys {
		if m.KID == "" {
			continue
		}
		// Without JWKMarshalOptions.Private, symmetric ("oct") keys fail to parse and are skipped.
		k, err := jwkset.NewJWKFromMarshal(m, jwkset.JWKMarshalOptions{}, jwkset.JWKValidateOptions{})
		if err != nil {
			continue
		}
		keys[m.KID] = k
	}
	if len(keys) == 0 {
		// Treat this as an outage rather than a key rotation, so cached keys keep serving.
		return fmt.Errorf("jwks at %s has no usable keys", jwksURL)
	}

	i.mu.Lock()
	i.keys, i.jwksURL = keys, jwksURL
	i.mu.Unlock()
	return nil
}

func (i *issuer) discover(ctx context.Context) (string, error) {
	var doc struct {
		Issuer  string `json:"issuer"`
		JWKSURI string `json:"jwks_uri"`
	}
	u := strings.TrimSuffix(i.cfg.Issuer, "/") + "/.well-known/openid-configuration"
	if err := i.getJSON(ctx, u, &doc); err != nil {
		return "", err
	}
	if doc.Issuer != i.cfg.Issuer {
		return "", fmt.Errorf("discovery document issuer %q does not match configured issuer %q", doc.Issuer, i.cfg.Issuer)
	}
	if doc.JWKSURI == "" {
		return "", fmt.Errorf("discovery document at %s has no jwks_uri", u)
	}
	return doc.JWKSURI, nil
}

func (i *issuer) getJSON(ctx context.Context, u string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return fmt.Errorf("build request for %s: %w", u, err)
	}
	resp, err := i.client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch %s: %w", u, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch %s: status %d", u, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxJWKSResponseBytes+1))
	if err != nil {
		return fmt.Errorf("read %s: %w", u, err)
	}
	if len(body) > maxJWKSResponseBytes {
		return errors.New("response from " + u + " exceeds size limit")
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("decode %s: %w", u, err)
	}
	return nil
}

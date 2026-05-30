// Package wns implements a client for the Windows Push Notification Service (WNS).
//
// Fleet uses raw WNS notifications to wake a Windows MDM device so it starts an OMA-DM session
// immediately, instead of waiting for its next scheduled poll. The device must have been provisioned
// with the matching Package Family Name (PFN) via the DMClient CSP, and must have reported a ChannelURI
// back to the server. See ai/push for the design and the validated protocol notes.
package wns

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
)

const (
	// defaultTokenURL is the WNS OAuth2 token endpoint (legacy UWP/Microsoft Store flow).
	defaultTokenURL = "https://login.live.com/accesstoken.srf" //nolint:gosec // G101: public WNS OAuth endpoint URL, not a credential

	// scope is the OAuth2 scope required to send WNS notifications.
	scope = "notify.windows.com"

	// msAppScheme is the prefix the Package SID must be wrapped in to be accepted as the WNS client_id.
	msAppScheme = "ms-app://"

	// tokenExpiryMargin is subtracted from the reported token lifetime so we refresh before WNS would
	// reject an about-to-expire token.
	tokenExpiryMargin = 5 * time.Minute

	// rawPayload is the body of the raw notification. For MDM push the device ignores the payload, so the
	// content is irrelevant; a small non-empty body keeps the request well-formed.
	rawPayload = "fleet"

	httpTimeout = 30 * time.Second
)

// ErrChannelExpired is returned by SendRaw when WNS reports the ChannelURI is gone (HTTP 410). Callers
// should clear the stored ChannelURI; the device renews it (~every 15 days) and reports a new one.
var ErrChannelExpired = errors.New("wns: channel URI expired or invalid (410)")

// IsValidChannelURI reports whether uri is a well-formed HTTPS WNS channel on the notify.windows.com domain.
// The channel URI is supplied by the device, so it must be validated before being stored or used as a push
// target: a compromised enrollment could otherwise redirect the WNS bearer token to an attacker-controlled host
// (SSRF / token exfiltration). Microsoft guarantees channel URIs are HTTPS on the notify.windows.com domain.
func IsValidChannelURI(uri string) bool {
	u, err := url.Parse(uri)
	if err != nil || u.Scheme != "https" {
		return false
	}
	host := u.Hostname()
	return host == "notify.windows.com" || strings.HasSuffix(host, ".notify.windows.com")
}

// Client sends raw WNS push notifications. It is safe for concurrent use; the cached access token is
// guarded by a mutex.
type Client struct {
	sid          string
	clientSecret string
	tokenURL     string
	httpClient   *http.Client

	mu          sync.Mutex
	token       string
	tokenExpiry time.Time

	// nowFunc returns the current time; overridable in tests.
	nowFunc func() time.Time
}

// NewClient returns a WNS client. sid is the Package SID as shown in Partner Center (the bare S-1-15-2-...
// form); the required ms-app:// scheme is added internally.
func NewClient(sid, clientSecret string) *Client {
	return &Client{
		sid:          sid,
		clientSecret: clientSecret,
		tokenURL:     defaultTokenURL,
		httpClient:   fleethttp.NewClient(fleethttp.WithTimeout(httpTimeout)),
		nowFunc:      time.Now,
	}
}

// SendRaw sends a raw WNS push notification to channelURI to wake the device for an OMA-DM session. On
// HTTP 401 it refreshes the access token once and retries. It returns ErrChannelExpired on HTTP 410.
func (c *Client) SendRaw(ctx context.Context, channelURI string) error {
	// The channel URI is supplied by the device and is the target we post a bearer token to. Refuse
	// anything but HTTPS so a malformed or malicious URI cannot leak the token over plaintext or trigger
	// an SSRF-style request. WNS channel URIs are always HTTPS on the notify.windows.com domain.
	parsed, err := url.Parse(channelURI)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "parse WNS channel URI")
	}
	if parsed.Scheme != "https" {
		return ctxerr.Errorf(ctx, "wns: refusing to send to non-HTTPS channel URI (scheme %q)", parsed.Scheme)
	}

	token, err := c.accessToken(ctx)
	if err != nil {
		return err
	}

	status, err := c.postRaw(ctx, channelURI, token)
	if err != nil {
		return err
	}

	if status == http.StatusUnauthorized {
		// The token may have been revoked or expired ahead of schedule; force a refresh and retry once.
		token, err = c.refreshToken(ctx, token)
		if err != nil {
			return err
		}
		status, err = c.postRaw(ctx, channelURI, token)
		if err != nil {
			return err
		}
	}

	switch status {
	case http.StatusOK:
		return nil
	case http.StatusGone:
		return ErrChannelExpired
	default:
		return ctxerr.Errorf(ctx, "wns: push to channel failed with status %d", status)
	}
}

// accessToken returns a cached token if still valid, otherwise fetches a new one.
func (c *Client) accessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && c.nowFunc().Before(c.tokenExpiry) {
		return c.token, nil
	}
	return c.fetchTokenLocked(ctx)
}

// refreshToken fetches a fresh token after the one in failedToken was rejected. If another goroutine
// already replaced that token while this one waited for the lock, the newer token is reused so
// concurrent pushes that all hit a 401 do not each trigger a redundant token request.
func (c *Client) refreshToken(ctx context.Context, failedToken string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && c.token != failedToken {
		return c.token, nil
	}
	c.token = ""
	c.tokenExpiry = time.Time{}
	return c.fetchTokenLocked(ctx)
}

// fetchTokenLocked requests a new access token from WNS. The caller must hold c.mu.
func (c *Client) fetchTokenLocked(ctx context.Context) (string, error) {
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	// client_id must be the Package SID wrapped in the ms-app:// scheme. Partner Center shows the bare
	// SID, so we add the prefix here (and tolerate a caller that already included it).
	form.Set("client_id", msAppScheme+strings.TrimPrefix(c.sid, msAppScheme))
	form.Set("client_secret", c.clientSecret)
	form.Set("scope", scope)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "create WNS token request")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "request WNS token")
	}
	defer resp.Body.Close()

	// Bound the response read; a well-behaved token response is tiny.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "read WNS token response")
	}
	if resp.StatusCode != http.StatusOK {
		return "", ctxerr.Errorf(ctx, "wns: token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tr struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", ctxerr.Wrap(ctx, err, "decode WNS token response")
	}
	if tr.AccessToken == "" {
		return "", ctxerr.New(ctx, "wns: token response missing access_token")
	}
	// Guard against a missing/absurd lifetime so we don't compute an already-expired token and thrash.
	if tr.ExpiresIn <= 0 {
		tr.ExpiresIn = 86400
	}

	lifetime := time.Duration(tr.ExpiresIn) * time.Second
	// Refresh slightly before the real expiry, but only when the lifetime is long enough that the margin
	// would not push the cached expiry into the past (which would cause continuous refetching).
	if lifetime > tokenExpiryMargin {
		lifetime -= tokenExpiryMargin
	}
	c.token = tr.AccessToken
	c.tokenExpiry = c.nowFunc().Add(lifetime)
	return c.token, nil
}

// postRaw posts a single raw notification and returns the HTTP status code.
func (c *Client) postRaw(ctx context.Context, channelURI, token string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, channelURI, strings.NewReader(rawPayload))
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "create WNS push request")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("X-WNS-Type", "wns/raw")
	// Cache the notification so a device that is briefly offline is still woken when it reconnects.
	req.Header.Set("X-WNS-Cache-Policy", "cache")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "send WNS push")
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode, nil
}

package installers

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/fleetdm/fleet/tools/hangar/internal/fleetctl"
	"github.com/fleetdm/fleet/tools/hangar/internal/paths"
)

// Team is one enrollment target found by a scan. ID nil is Unassigned (the
// global enroll secret). A scan result carries no secrets: the tab asks for
// the one it needs once a fleet is picked (SecretFor).
type Team struct {
	ID        *uint  `json:"id"`
	Name      string `json:"name"`
	HasSecret bool   `json:"has_secret"`
	// SecretHidden means the server returned the secret masked: the
	// context's user isn't allowed to read enroll secrets.
	SecretHidden bool `json:"secret_hidden"`
	secret       string
}

// maskedSecret is what Fleet returns in place of a secret the user can't read
// (fleet.MaskedPassword).
const maskedSecret = "********"

// ScanResult is what a scan of one server found.
type ScanResult struct {
	Context   string `json:"context"`
	Address   string `json:"address"`
	ServerURL string `json:"server_url"`
	Premium   bool   `json:"premium"`
	Teams     []Team `json:"teams"`
}

// UnassignedName is the label for the global enroll secret.
const UnassignedName = "Unassigned"

// Client calls one Fleet server's API with a fleetctl context's credentials.
type Client struct {
	base    string
	token   string
	headers map[string]string
	http    *http.Client
}

// NewClient builds a client from fleetctl context credentials, honoring the
// context's rootca / tls-skip-verify like fleetctl does.
func NewClient(c fleetctl.Credentials) (*Client, error) {
	if strings.TrimSpace(c.Address) == "" {
		return nil, errors.New("fleetctl context has no address")
	}
	if strings.TrimSpace(c.Token) == "" {
		return nil, errors.New("fleetctl context has no token — run fleetctl login for it")
	}
	tlsCfg := &tls.Config{InsecureSkipVerify: c.TLSSkipVerify} //nolint:gosec // mirrors the context's own tls-skip-verify
	if c.RootCA != "" {
		pem, err := os.ReadFile(paths.Expand(c.RootCA))
		if err != nil {
			return nil, fmt.Errorf("read rootca: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("no certificates in rootca %s", c.RootCA)
		}
		tlsCfg.RootCAs = pool
	}
	// fleetctl joins the address, url-prefix and request path the same way.
	base := strings.TrimRight(c.Address, "/")
	if prefix := strings.Trim(c.URLPrefix, "/"); prefix != "" {
		base += "/" + prefix
	}
	return &Client{
		base:    base,
		token:   c.Token,
		headers: c.CustomHeaders,
		// Clients are short-lived (one per scan), so don't leave idle connections behind.
		http: &http.Client{Timeout: 15 * time.Second, Transport: &http.Transport{TLSClientConfig: tlsCfg, DisableKeepAlives: true}},
	}, nil
}

type statusError struct {
	code int
	body string
}

func (e *statusError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.code, strings.TrimSpace(e.body))
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path, nil)
	if err != nil {
		return err
	}
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		if len(body) > 300 {
			body = body[:300]
		}
		return &statusError{code: resp.StatusCode, body: string(body)}
	}
	return json.Unmarshal(body, out)
}

// describe turns a failed first request into what to fix: the server being
// down, the context's token or TLS settings, or the address itself.
func (c *Client) describe(err error) error {
	var dnsErr *net.DNSError
	var netErr net.Error
	var certErr *tls.CertificateVerificationError
	switch {
	case errors.Is(err, syscall.ECONNREFUSED):
		return fmt.Errorf("no Fleet server is running at %s", c.base)
	case hasStatus(err, http.StatusUnauthorized):
		return errors.New("the server rejected the context's API token (expired or logged out) — run fleetctl login for this context")
	case hasStatus(err, http.StatusForbidden):
		return errors.New("the context's user isn't allowed to read the server's settings")
	case errors.As(err, &certErr):
		return fmt.Errorf("%s's TLS certificate isn't trusted — set tls-skip-verify or rootca on the context", c.base)
	case errors.As(err, &dnsErr):
		return fmt.Errorf("can't resolve %s", dnsErr.Name)
	case errors.As(err, &netErr) && netErr.Timeout():
		return fmt.Errorf("%s didn't answer in time", c.base)
	}
	return fmt.Errorf("read %s/api/v1/fleet/config: %w", c.base, err)
}

func hasStatus(err error, codes ...int) bool {
	var se *statusError
	if !errors.As(err, &se) {
		return false
	}
	for _, c := range codes {
		if se.code == c {
			return true
		}
	}
	return false
}

type apiSecret struct {
	Secret string `json:"secret"`
}

type apiTeam struct {
	ID      uint        `json:"id"`
	Name    string      `json:"name"`
	Secrets []apiSecret `json:"secrets"`
}

// Scan reads the server's public URL, license tier, the global enroll secret
// and (on Premium) every fleet/team with its enroll secret.
func (c *Client) Scan(ctx context.Context) (ScanResult, error) {
	var cfg struct {
		ServerSettings struct {
			ServerURL string `json:"server_url"`
		} `json:"server_settings"`
		License struct {
			Tier string `json:"tier"`
		} `json:"license"`
	}
	if err := c.get(ctx, "/api/v1/fleet/config", &cfg); err != nil {
		return ScanResult{}, c.describe(err)
	}
	res := ScanResult{Address: c.base, ServerURL: cfg.ServerSettings.ServerURL, Premium: cfg.License.Tier == "premium"}

	var spec struct {
		Spec struct {
			Secrets []apiSecret `json:"secrets"`
		} `json:"spec"`
	}
	switch err := c.get(ctx, "/api/v1/fleet/spec/enroll_secret", &spec); {
	case hasStatus(err, http.StatusForbidden):
		// Only global admins and maintainers may read the global secret; a
		// fleet-level user can still build for the fleets they manage.
		res.Teams = append(res.Teams, Team{Name: UnassignedName, SecretHidden: true})
	case err != nil:
		return ScanResult{}, fmt.Errorf("read global enroll secret: %w", err)
	default:
		res.Teams = append(res.Teams, teamFrom(nil, UnassignedName, spec.Spec.Secrets))
	}

	if !res.Premium {
		return res, nil
	}
	teams, err := c.listTeams(ctx)
	if err != nil {
		return ScanResult{}, err
	}
	for _, t := range teams {
		id := t.ID
		res.Teams = append(res.Teams, teamFrom(&id, t.Name, t.Secrets))
	}
	return res, nil
}

// listTeams lists fleets on current servers (/fleets) and falls back to the
// pre-rename /teams route for older releases. The response carries the list
// under "fleets", "teams", or both. Only the first 1000 are read, far more
// than a test server has.
func (c *Client) listTeams(ctx context.Context) ([]apiTeam, error) {
	var resp struct {
		Fleets []apiTeam `json:"fleets"`
		Teams  []apiTeam `json:"teams"`
	}
	err := c.get(ctx, "/api/v1/fleet/fleets?per_page=1000", &resp)
	if hasStatus(err, http.StatusNotFound, http.StatusMethodNotAllowed) {
		err = c.get(ctx, "/api/v1/fleet/teams?per_page=1000", &resp)
	}
	if err != nil {
		return nil, fmt.Errorf("list fleets: %w", err)
	}
	if len(resp.Fleets) > 0 {
		return resp.Fleets, nil
	}
	return resp.Teams, nil
}

func teamFrom(id *uint, name string, secrets []apiSecret) Team {
	t := Team{ID: id, Name: name}
	for _, s := range secrets {
		switch s.Secret {
		case "":
		case maskedSecret:
			t.SecretHidden = true
		default:
			t.secret = s.Secret
			t.HasSecret = true
			t.SecretHidden = false
			return t
		}
	}
	return t
}

// SecretFor returns the enroll secret of the team with id (nil = Unassigned)
// from a scan, or an error naming what's missing.
func SecretFor(res ScanResult, id *uint) (secret, name string, err error) {
	for _, t := range res.Teams {
		if (t.ID == nil) != (id == nil) || (t.ID != nil && *t.ID != *id) {
			continue
		}
		if !t.HasSecret {
			if t.SecretHidden {
				return "", t.Name, fmt.Errorf("the fleetctl context's user can't read %s's enroll secret — use an admin or maintainer context", t.Name)
			}
			return "", t.Name, fmt.Errorf("%s has no enroll secret — add one in Fleet first", t.Name)
		}
		return t.secret, t.Name, nil
	}
	if id == nil {
		return "", "", errors.New("server returned no global enroll secret")
	}
	return "", "", fmt.Errorf("fleet %d not found on the server — scan again", *id)
}

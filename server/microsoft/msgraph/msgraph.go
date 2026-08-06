// Package msgraph is Fleet's client for Microsoft Graph. It authenticates as an Entra app registration using the
// OAuth2 client-credentials grant and reads Windows Autopilot device identities, which Fleet surfaces as pending hosts.
//
// Note this is Microsoft Graph, the unified API over Entra, Intune and the rest of Microsoft 365; the Autopilot
// collection specifically is Intune data. The credential is an Entra app registration, which is why the config calls it
// an Entra credential while the API is Graph.
package msgraph

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

const (
	// defaultLoginHost is Entra's token endpoint host. Overridable in tests.
	defaultLoginHost = "https://login.microsoftonline.com"
	// defaultGraphHost is the Microsoft Graph host. Overridable in tests.
	defaultGraphHost = "https://graph.microsoft.com"

	// graphScope requests every application permission already consented for the app registration, which is the
	// standard scope for the client-credentials flow.
	graphScope = "https://graph.microsoft.com/.default"

	autopilotDevicesPath = "/v1.0/deviceManagement/windowsAutopilotDeviceIdentities"

	// maxPages bounds the pagination walk. Graph paginates this collection on serial number with an inclusive cursor,
	// and can return an @odata.nextLink identical to the request URL, so an unguarded walk does not terminate. The
	// per-page and identical-link guards below catch that first; this is the last-resort backstop.
	maxPages = 1000

	// requestTimeout bounds a single Graph call.
	requestTimeout = 60 * time.Second
)

// Client reads Windows Autopilot data from Microsoft Graph.
type Client interface {
	// VerifyCredential mints a token and lists a single page to confirm the credential works. Used to reject a bad
	// credential at config-write time rather than silently failing on the next sync.
	VerifyCredential(ctx context.Context) error
	// ListWindowsAutopilotDevices returns all Autopilot device identities for the credential's tenant, deduplicated by
	// device ID.
	ListWindowsAutopilotDevices(ctx context.Context) ([]fleet.WindowsAutopilotDevice, error)
}

// ClientFactory builds a Client for a credential. It is injected so callers (notably the sync cron) do not import the
// concrete client and tests can supply a fake, mirroring GoogleWorkspaceDirectoryFactory.
type ClientFactory func(cred *fleet.MicrosoftGraphCredential) (Client, error)

type client struct {
	httpClient *http.Client
	graphHost  string
}

// NewClient builds a Graph client for the given credential. The returned client refreshes its own access token: tokens
// live about an hour, which is why Fleet stores an app-registration credential rather than a pasted token.
func NewClient(cred *fleet.MicrosoftGraphCredential) (Client, error) {
	return newClientWithHosts(cred, defaultLoginHost, defaultGraphHost)
}

func newClientWithHosts(cred *fleet.MicrosoftGraphCredential, loginHost, graphHost string) (Client, error) {
	if cred == nil || !cred.Configured() {
		return nil, fmt.Errorf("microsoft graph credential is not fully configured")
	}

	cfg := &clientcredentials.Config{
		ClientID:     cred.ClientID,
		ClientSecret: cred.ClientSecret,
		TokenURL:     fmt.Sprintf("%s/%s/oauth2/v2.0/token", strings.TrimSuffix(loginHost, "/"), url.PathEscape(cred.TenantID)),
		Scopes:       []string{graphScope},
		// Send the credential as form parameters, which is the shape Microsoft documents for this flow. The library's
		// default is auto-detect, which tries HTTP Basic first and silently retries on failure; pinning the style keeps
		// the request deterministic and avoids a wasted probe round-trip.
		AuthStyle: oauth2.AuthStyleInParams,
	}

	// Drive the oauth2 transport from fleethttp rather than a bare http.Client so Fleet's shared transport settings
	// apply to both the token request and the Graph calls.
	base := fleethttp.NewClient(fleethttp.WithTimeout(requestTimeout))
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, base)

	return &client{
		httpClient: cfg.Client(ctx),
		graphHost:  strings.TrimSuffix(graphHost, "/"),
	}, nil
}

func (c *client) VerifyCredential(ctx context.Context) error {
	// One page is enough: it proves the secret mints a token and that the app has the Autopilot read permission with
	// admin consent, which are the two things that actually go wrong.
	if _, _, err := c.getPage(ctx, c.graphHost+autopilotDevicesPath); err != nil {
		return ctxerr.Wrap(ctx, err, "verify microsoft graph credential")
	}
	return nil
}

func (c *client) ListWindowsAutopilotDevices(ctx context.Context) ([]fleet.WindowsAutopilotDevice, error) {
	var (
		devices  []fleet.WindowsAutopilotDevice
		seen     = make(map[string]struct{})
		nextURL  = c.graphHost + autopilotDevicesPath
		pageNum  int
		prevPage string
	)

	for nextURL != "" {
		pageNum++
		if pageNum > maxPages {
			return nil, ctxerr.Errorf(ctx, "microsoft graph autopilot listing exceeded %d pages, aborting", maxPages)
		}

		// Graph's cursor for this collection is $skiptoken=LastSerialNumber='<serial>', and it is inclusive. When a
		// page's cursor does not advance past its own last row, the service echoes back the URL just requested, and a
		// naive "follow nextLink until absent" loop spins forever hammering Graph. Verified against a live tenant at
		// $top=1.
		if nextURL == prevPage {
			return nil, ctxerr.Errorf(ctx,
				"microsoft graph returned a nextLink identical to the request URL at page %d, aborting to avoid an infinite loop", pageNum)
		}
		prevPage = nextURL

		page, link, err := c.getPage(ctx, nextURL)
		if err != nil {
			return nil, ctxerr.Wrapf(ctx, err, "list windows autopilot devices page %d", pageNum)
		}

		// The same inclusive cursor repeats the boundary device on the next page, so dedupe by the Autopilot device
		// ID. Serial numbers cannot be used for this: they are not unique (placeholder serials such as "Default
		// string" ship on real hardware) and are exactly what the cursor is keyed on.
		var newOnPage int
		for _, d := range page {
			if d.ID == "" {
				continue
			}
			if _, ok := seen[d.ID]; ok {
				continue
			}
			seen[d.ID] = struct{}{}
			devices = append(devices, d)
			newOnPage++
		}

		// A page that returned rows but contributed nothing new means the cursor did not advance, which happens when a
		// run of devices shares one serial number (the cursor is keyed on serial). This is an error rather than a
		// graceful stop on purpose: returning the devices gathered so far would be a silently truncated list, and the
		// sync deletes hosts that are absent from what it is given. Failing leaves the tenant's pending hosts intact.
		if link != "" && newOnPage == 0 && len(page) > 0 {
			return nil, ctxerr.Errorf(ctx,
				"microsoft graph pagination stopped advancing at page %d (%d rows, none new), aborting rather than returning a partial list",
				pageNum, len(page))
		}

		// The next link is a URL chosen by the remote service, and the oauth2 transport attaches the access token to
		// whatever we request. Requiring it to stay on the Graph origin means a malformed or hostile link cannot make
		// Fleet hand its token to another host.
		if link != "" {
			if err := c.assertGraphOrigin(link); err != nil {
				return nil, ctxerr.Wrap(ctx, err, "validate microsoft graph next link")
			}
		}

		nextURL = link
	}

	return devices, nil
}

// assertGraphOrigin rejects a next link that is relative or points anywhere other than the Graph host.
func (c *client) assertGraphOrigin(link string) error {
	next, err := url.Parse(link)
	if err != nil {
		return fmt.Errorf("parse next link: %w", err)
	}
	graph, err := url.Parse(c.graphHost)
	if err != nil {
		return fmt.Errorf("parse graph host: %w", err)
	}
	if !strings.EqualFold(next.Scheme, graph.Scheme) || !strings.EqualFold(next.Host, graph.Host) {
		return fmt.Errorf("nextLink points at unexpected origin %q://%q, expected %q://%q",
			next.Scheme, next.Host, graph.Scheme, graph.Host)
	}
	return nil
}

type autopilotDevicesResponse struct {
	Value    []fleet.WindowsAutopilotDevice `json:"value"`
	NextLink string                         `json:"@odata.nextLink"`
}

// getPage performs one Graph GET and returns the devices plus the next link, if any.
func (c *client) getPage(ctx context.Context, requestURL string) ([]fleet.WindowsAutopilotDevice, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, "", ctxerr.Wrap(ctx, err, "build microsoft graph request")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// The oauth2 transport fetches the access token lazily on the first request, so a rejected client secret
		// surfaces here as a token-endpoint failure rather than as a Graph response. Convert it so callers see an
		// auth error instead of a generic connection failure.
		if retrieveErr, ok := errors.AsType[*oauth2.RetrieveError](err); ok {
			return nil, "", ctxerr.Wrap(ctx, newTokenError(retrieveErr), "acquire microsoft graph token")
		}
		return nil, "", ctxerr.Wrap(ctx, err, "call microsoft graph")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", ctxerr.Wrap(ctx, err, "read microsoft graph response")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, "", ctxerr.Wrap(ctx, newGraphError(resp, body), "microsoft graph request failed")
	}

	var parsed autopilotDevicesResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, "", ctxerr.Wrap(ctx, err, "decode microsoft graph response")
	}
	return parsed.Value, parsed.NextLink, nil
}

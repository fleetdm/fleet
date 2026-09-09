// Package msgraph is Fleet's client for Microsoft Graph. It authenticates as an Entra app registration using the
// OAuth2 client-credentials grant and reads Windows Autopilot device identities, which Fleet surfaces as pending hosts.
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

	// graphScope is the only scope value Entra accepts for the client-credentials flow. This is not a broad grant.
	// App-only permissions are assigned to the app registration and admin-consented up front, so there is no incremental
	// consent to negotiate at token time; .default means "the permissions this app has already been granted for this
	// resource", and the token carries nothing more.
	graphScope = "https://graph.microsoft.com/.default"

	// autopilotDevicesPath uses v1.0, which is the current GA endpoint as of 2026/08/07
	autopilotDevicesPath = "/v1.0/deviceManagement/windowsAutopilotDeviceIdentities"

	// verifyPageSize is what VerifyCredential asks for.
	verifyPageSize = 1

	// pageSize is sent as $top so the number of round trips is a property of our code rather than of an undocumented
	// service default that Microsoft can change. A large tenant can register 100k+ Autopilot devices; at 1000 per page
	// that is ~100 requests
	pageSize = 1000

	// maxPages bounds the pagination walk as a last-resort backstop against a non-advancing cursor.
	maxPages = 1000

	// requestTimeout bounds a single Graph call.
	requestTimeout = 60 * time.Second
)

// WindowsAutopilotDevice is a Windows Autopilot device identity as returned by Microsoft Graph from
// /deviceManagement/windowsAutopilotDeviceIdentities. ID is the Graph resource id of the Autopilot registration.
type WindowsAutopilotDevice struct {
	ID           string `json:"id"`
	SerialNumber string `json:"serialNumber"`
	GroupTag     string `json:"groupTag"`
	// Model and Manufacturer are what Autopilot recorded for the hardware.
	Model        string `json:"model"`
	Manufacturer string `json:"manufacturer"`
	// The JSON tag is Graph's and must stay as-is: Microsoft rebranded Azure AD to Entra ID but never renamed this API field.
	EntraDeviceID string `json:"azureActiveDirectoryDeviceId"`
}

// Client reads Windows Autopilot data from Microsoft Graph.
type Client interface {
	// VerifyCredential mints a token and lists a single page to confirm the credential works.
	VerifyCredential(ctx context.Context) error
	// ListWindowsAutopilotDevices returns all Autopilot device identities for the credential's tenant, deduplicated by
	// device ID.
	ListWindowsAutopilotDevices(ctx context.Context) ([]WindowsAutopilotDevice, error)
}

// ClientFactory builds a Client for a credential. It is injected so callers (notably the sync cron) do not import the
// concrete client and tests can supply a fake, mirroring GoogleWorkspaceDirectoryFactory.
type ClientFactory func(cred *fleet.MicrosoftGraphCredential) (Client, error)

type client struct {
	cfg *clientcredentials.Config
	// baseClient carries Fleet's shared transport settings and is handed to the oauth2 machinery per call.
	baseClient *http.Client
	graphHost  string
}

// NewClient builds a Graph client for the given credential. The returned client refreshes its own access token.
func NewClient(cred *fleet.MicrosoftGraphCredential) (Client, error) {
	return newClientWithHosts(cred, defaultLoginHost, defaultGraphHost)
}

func newClientWithHosts(cred *fleet.MicrosoftGraphCredential, loginHost, graphHost string) (Client, error) {
	if cred == nil || !cred.Configured() {
		return nil, errors.New("microsoft graph credential is not fully configured")
	}

	cfg := &clientcredentials.Config{
		ClientID:     cred.ClientID,
		ClientSecret: cred.ClientSecret,
		TokenURL:     fmt.Sprintf("%s/%s/oauth2/v2.0/token", strings.TrimSuffix(loginHost, "/"), url.PathEscape(cred.TenantID)),
		Scopes:       []string{graphScope},
		// Send the credential as form parameters, which is the shape Microsoft documents for this flow.
		AuthStyle: oauth2.AuthStyleInParams,
	}

	return &client{
		cfg:        cfg,
		baseClient: fleethttp.NewClient(fleethttp.WithTimeout(requestTimeout)),
		graphHost:  strings.TrimSuffix(graphHost, "/"),
	}, nil
}

// httpClientFor builds an HTTP client whose token acquisition inherits ctx.
func (c *client) httpClientFor(ctx context.Context) *http.Client {
	// Configure clientcredentials package. It is using context for configuration.
	httpClient := c.cfg.Client(context.WithValue(ctx, oauth2.HTTPClient, c.baseClient))
	return httpClient
}

func (c *client) VerifyCredential(ctx context.Context) error {
	verifyURL := fmt.Sprintf("%s%s?$top=%d", c.graphHost, autopilotDevicesPath, verifyPageSize)
	if _, _, err := c.getPage(ctx, c.httpClientFor(ctx), verifyURL); err != nil {
		return ctxerr.Wrap(ctx, err, "verify microsoft graph credential")
	}
	return nil
}

func (c *client) ListWindowsAutopilotDevices(ctx context.Context) ([]WindowsAutopilotDevice, error) {
	// One client for the whole walk, so every page shares a single access token.
	httpClient := c.httpClientFor(ctx)

	var (
		devices  []WindowsAutopilotDevice
		seen     = make(map[string]struct{})
		nextURL  = fmt.Sprintf("%s%s?$top=%d", c.graphHost, autopilotDevicesPath, pageSize)
		pageNum  int
		prevPage string
	)

	for nextURL != "" {
		pageNum++
		if pageNum > maxPages {
			return nil, ctxerr.Errorf(ctx, "microsoft graph autopilot listing exceeded %d pages, aborting", maxPages)
		}

		// Graph's cursor for this collection is $skiptoken=LastSerialNumber='<serial>', and it is inclusive.
		if nextURL == prevPage {
			return nil, ctxerr.Errorf(ctx,
				"microsoft graph returned a nextLink identical to the request URL at page %d, aborting to avoid an infinite loop", pageNum)
		}
		prevPage = nextURL

		page, link, err := c.getPage(ctx, httpClient, nextURL)
		if err != nil {
			return nil, ctxerr.Wrapf(ctx, err, "list windows autopilot devices page %d", pageNum)
		}

		// The same inclusive cursor repeats the boundary device on the next page, so dedupe by the Autopilot device ID.
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

		// A page that advertises more results but contributes no new devices means the cursor is not advancing, so
		// continuing cannot make progress. Two shapes hit this: a run of devices sharing one serial number (the cursor
		// is keyed on serial), and empty pages returned with a non-empty nextLink. This is an error rather than a
		// graceful stop on purpose: returning the devices gathered so far would be a silently truncated list, and the
		// sync deletes hosts that are absent from what it is given. Failing leaves the tenant's pending hosts intact.
		if link != "" && newOnPage == 0 {
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
	Value    []WindowsAutopilotDevice `json:"value"`
	NextLink string                   `json:"@odata.nextLink"`
}

// getPage performs one Graph GET and returns the devices plus the next link, if any.
func (c *client) getPage(ctx context.Context, httpClient *http.Client, requestURL string) ([]WindowsAutopilotDevice, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, "", ctxerr.Wrap(ctx, err, "build microsoft graph request")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		// The oauth2 transport fetches the access token lazily on the first request, so a rejected client secret
		// surfaces here as a token-endpoint failure rather than as a Graph response.
		if retrieveErr, ok := errors.AsType[*oauth2.RetrieveError](err); ok {
			return nil, "", ctxerr.Wrap(ctx, newTokenError(retrieveErr), "acquire microsoft graph token")
		}
		return nil, "", ctxerr.Wrap(ctx, err, "call microsoft graph")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Bound the read rather than truncating afterwards: an edge proxy can answer a 5xx with a very large HTML page,
		// and reading it in full to then keep 512 bytes would allocate the whole thing. One byte over the limit is read
		// so truncateBody can still mark the message as truncated.
		body, err := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes+1))
		if err != nil {
			return nil, "", ctxerr.Wrap(ctx, err, "read microsoft graph error response")
		}
		return nil, "", ctxerr.Wrap(ctx, newGraphError(resp, body), "microsoft graph request failed")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", ctxerr.Wrap(ctx, err, "read microsoft graph response")
	}

	var parsed autopilotDevicesResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, "", ctxerr.Wrap(ctx, err, "decode microsoft graph response")
	}
	return parsed.Value, parsed.NextLink, nil
}

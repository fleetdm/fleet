package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/Azure/go-ntlmssp"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	scepclient "github.com/fleetdm/fleet/v4/server/mdm/scep/client"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
	"github.com/go-kit/log"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

var _ scepserver.Service = (*scepProxyService)(nil)
var challengeRegex = regexp.MustCompile(`(?i)The enrollment challenge password is: <B> (?P<password>\S*)`)

const fullPasswordCache = "The password cache is full."

type scepProxyService struct {
	// info logging is implemented in the service middleware layer.
	debugLogger log.Logger
}

func (svc *scepProxyService) GetCACaps(_ context.Context) ([]byte, error) {
	return nil, errors.New("not implemented")
}

func (svc *scepProxyService) GetCACert(_ context.Context, _ string) ([]byte, int, error) {
	return nil, 0, errors.New("not implemented")
}

func (svc *scepProxyService) PKIOperation(_ context.Context, data []byte) ([]byte, error) {
	return nil, errors.New("not implemented")
}

func (svc *scepProxyService) GetNextCACert(ctx context.Context) ([]byte, error) {
	return nil, errors.New("not implemented")
}

// NewSCEPProxyService creates a new scep proxy service
func NewSCEPProxyService(logger log.Logger) scepserver.Service {
	return &scepProxyService{
		debugLogger: logger,
	}
}

func ValidateNDESSCEPAdminURL(ctx context.Context, proxy fleet.NDESSCEPProxyIntegration) error {
	adminURL, username, password := proxy.AdminURL, proxy.Username, proxy.Password
	// Get the challenge from NDES
	client := fleethttp.NewClient()
	client.Transport = ntlmssp.Negotiator{
		RoundTripper: fleethttp.NewTransport(),
	}
	req, err := http.NewRequest(http.MethodGet, adminURL, http.NoBody)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "creating request")
	}
	req.SetBasicAuth(username, password)
	resp, err := client.Do(req)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "sending request")
	}
	if resp.StatusCode != http.StatusOK {
		return ctxerr.New(ctx, fmt.Sprintf("unexpected status code: %d", resp.StatusCode))
	}
	// Make a transformer that converts MS-Win default to UTF8:
	win16be := unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)
	// Make a transformer that is like win16be, but abides by BOM:
	utf16bom := unicode.BOMOverride(win16be.NewDecoder())

	// Make a Reader that uses utf16bom:
	unicodeReader := transform.NewReader(resp.Body, utf16bom)
	bodyText, err := io.ReadAll(unicodeReader)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "reading response body")
	}
	htmlString := string(bodyText)

	matches := challengeRegex.FindStringSubmatch(htmlString)
	challenge := ""
	if matches != nil {
		challenge = matches[challengeRegex.SubexpIndex("password")]
	}
	if challenge == "" {
		if strings.Contains(htmlString, fullPasswordCache) {
			return ctxerr.New(ctx,
				"the password cache is full; please increase the number of cached passwords in NDES; by default, NDES caches 5 passwords and they expire 60 minutes after they are created")
		}
		return ctxerr.New(ctx, "could not retrieve the enrollment challenge password")
	}
	return nil
}

func ValidateNDESSCEPURL(ctx context.Context, proxy fleet.NDESSCEPProxyIntegration, logger log.Logger) error {
	client, err := scepclient.New(proxy.URL, logger)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "creating SCEP client")
	}

	certs, _, err := client.GetCACert(ctx, "")
	if err != nil {
		return ctxerr.Wrap(ctx, err, "could not retrieve CA certificate from SCEP URL")
	}
	if len(certs) == 0 {
		return ctxerr.New(ctx, "SCEP URL did not return a CA certificate")
	}
	return nil
}

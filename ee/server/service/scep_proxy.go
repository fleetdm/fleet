package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Azure/go-ntlmssp"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	scepclient "github.com/fleetdm/fleet/v4/server/mdm/scep/client"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/log"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

var _ scepserver.ServiceWithIdentifier = (*scepProxyService)(nil)
var challengeRegex = regexp.MustCompile(`(?i)The enrollment challenge password is: <B> (?P<password>\S*)`)

const (
	fullPasswordCache             = "The password cache is full."
	ndesInsufficientPermissions   = "You do not have sufficient permission to enroll with SCEP."
	MessageSCEPProxyNotConfigured = "SCEP proxy is not configured"
	NDESChallengeInvalidAfter     = 57 * time.Minute
)

// NDESTimeout is the timeout for NDES requests. It is exportable for testing.
var NDESTimeout = ptr.Duration(30 * time.Second)

type scepProxyService struct {
	ds fleet.Datastore
	// info logging is implemented in the service middleware layer.
	debugLogger log.Logger
}

// GetCACaps returns a list of SCEP options which are supported by the server.
// It is a pass-through call to the SCEP server.
func (svc *scepProxyService) GetCACaps(ctx context.Context) ([]byte, error) {
	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting app config")
	}
	if !appConfig.Integrations.NDESSCEPProxy.Valid {
		// Return error that implements kithttp.StatusCoder interface
		return nil, &scepserver.BadRequestError{Message: MessageSCEPProxyNotConfigured}
	}
	client, err := scepclient.New(appConfig.Integrations.NDESSCEPProxy.Value.URL, svc.debugLogger, NDESTimeout)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating SCEP client")
	}
	res, err := client.GetCACaps(ctx)
	if err != nil {
		return res, ctxerr.Wrapf(ctx, err, "Could not GetCACaps from SCEP server %s", appConfig.Integrations.NDESSCEPProxy.Value.URL)
	}
	return res, nil
}

// GetCACert returns the CA certificate(s) from SCEP server.
// It is a pass-through call to the SCEP server.
func (svc *scepProxyService) GetCACert(ctx context.Context, message string) ([]byte, int, error) {
	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, 0, ctxerr.Wrap(ctx, err, "getting app config")
	}
	if !appConfig.Integrations.NDESSCEPProxy.Valid {
		// Return error that implements kithttp.StatusCoder interface
		return nil, 0, &scepserver.BadRequestError{Message: MessageSCEPProxyNotConfigured}
	}
	client, err := scepclient.New(appConfig.Integrations.NDESSCEPProxy.Value.URL, svc.debugLogger, NDESTimeout)
	if err != nil {
		return nil, 0, ctxerr.Wrap(ctx, err, "creating SCEP client")
	}
	res, num, err := client.GetCACert(ctx, message)
	if err != nil {
		return res, num, ctxerr.Wrapf(ctx, err, "Could not GetCACert from SCEP server %s", appConfig.Integrations.NDESSCEPProxy.Value.URL)
	}
	return res, num, nil
}

func (svc *scepProxyService) PKIOperation(ctx context.Context, data []byte, identifier string) ([]byte, error) {
	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting app config")
	}
	if !appConfig.Integrations.NDESSCEPProxy.Valid {
		// Return error that implements kithttp.StatusCoder interface
		return nil, &scepserver.BadRequestError{Message: MessageSCEPProxyNotConfigured}
	}

	// Validate the identifier and challenge password expiration.
	parsedID, err := url.PathUnescape(identifier)
	if err != nil {
		// Should never happen since the identifier comes in as a path variable
		return nil, ctxerr.Wrap(ctx, err, "unescaping identifier in URL path")
	}
	parsedIDs := strings.Split(parsedID, ",")
	if len(parsedIDs) != 2 || parsedIDs[0] == "" || parsedIDs[1] == "" {
		// Return error that implements kithttp.StatusCoder interface
		return nil, &scepserver.BadRequestError{Message: "invalid identifier in URL path"}
	}
	hostUUID := parsedIDs[0]
	profileUUID := parsedIDs[1]
	if !strings.HasPrefix(profileUUID, fleet.MDMAppleProfileUUIDPrefix) {
		return nil, &scepserver.BadRequestError{Message: fmt.Sprintf("invalid profile UUID (only Apple config profiles are supported): %s",
			profileUUID)}
	}
	profile, err := svc.ds.GetHostMDMCertificateProfile(ctx, hostUUID, profileUUID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting host MDM profile")
	}
	if profile == nil {
		// Return error that implements kithttp.StatusCoder interface
		return nil, &scepserver.BadRequestError{Message: "unknown identifier in URL path"}
	}
	if profile.Status == nil || *profile.Status != fleet.MDMDeliveryPending {
		// This could happen if Fleet DB was updated before the profile was updated on the host.
		// We expect another certificate request from the host once the profile is updated.
		status := "null"
		if profile.Status != nil {
			status = string(*profile.Status)
		}
		return nil, &scepserver.BadRequestError{Message: fmt.Sprintf("profile status (%s) is not 'pending' for host:%s profile:%s", status,
			hostUUID, profileUUID)}
	}
	if profile.ChallengeRetrievedAt != nil && profile.ChallengeRetrievedAt.Add(NDESChallengeInvalidAfter).Before(time.Now()) {
		// The challenge password was retrieved for this profile, and is now invalid.
		// We need to resend the profile with a new challenge password.
		// Note: we don't actually know if it is invalid, and we can't get that exact feedback from SCEP server.
		if err = svc.ds.ResendHostMDMProfile(ctx, hostUUID, profileUUID); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "resending host mdm profile")
		}
		return nil, &scepserver.BadRequestError{Message: "challenge password has expired"}
	}

	client, err := scepclient.New(appConfig.Integrations.NDESSCEPProxy.Value.URL, svc.debugLogger, NDESTimeout)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating SCEP client")
	}
	res, err := client.PKIOperation(ctx, data)
	if err != nil {
		return res, ctxerr.Wrapf(ctx, err,
			"Could not do PKIOperation on SCEP server %s", appConfig.Integrations.NDESSCEPProxy.Value.URL)
	}
	return res, nil
}

func (svc *scepProxyService) GetNextCACert(ctx context.Context) ([]byte, error) {
	// NDES on Windows Server 2022 does not support this, as advertised via GetCACaps
	return nil, errors.New("GetNextCACert is not implemented for SCEP proxy")
}

// NewSCEPProxyService creates a new scep proxy service
func NewSCEPProxyService(ds fleet.Datastore, logger log.Logger) scepserver.ServiceWithIdentifier {
	return &scepProxyService{
		ds:          ds,
		debugLogger: logger,
	}
}

func ValidateNDESSCEPAdminURL(ctx context.Context, proxy fleet.NDESSCEPProxyIntegration) error {
	_, err := GetNDESSCEPChallenge(ctx, proxy)
	return err
}

func GetNDESSCEPChallenge(ctx context.Context, proxy fleet.NDESSCEPProxyIntegration) (string, error) {
	adminURL, username, password := proxy.AdminURL, proxy.Username, proxy.Password
	// Get the challenge from NDES
	client := fleethttp.NewClient(fleethttp.WithTimeout(*NDESTimeout))
	client.Transport = ntlmssp.Negotiator{
		RoundTripper: fleethttp.NewTransport(),
	}
	req, err := http.NewRequest(http.MethodGet, adminURL, http.NoBody)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "creating request")
	}
	req.SetBasicAuth(username, password)
	resp, err := client.Do(req)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "sending request")
	}
	if resp.StatusCode != http.StatusOK {
		return "", ctxerr.Wrap(ctx, NDESInvalidError{msg: fmt.Sprintf(
			"unexpected status code: %d; could not retrieve the enrollment challenge password; invalid admin URL or credentials; please correct and try again",
			resp.StatusCode)})
	}
	// Make a transformer that converts MS-Win default to UTF8:
	win16be := unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)
	// Make a transformer that is like win16be, but abides by BOM:
	utf16bom := unicode.BOMOverride(win16be.NewDecoder())

	// Make a Reader that uses utf16bom:
	unicodeReader := transform.NewReader(resp.Body, utf16bom)
	bodyText, err := io.ReadAll(unicodeReader)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "reading response body")
	}
	htmlString := string(bodyText)

	matches := challengeRegex.FindStringSubmatch(htmlString)
	challenge := ""
	if matches != nil {
		challenge = matches[challengeRegex.SubexpIndex("password")]
	}
	if challenge == "" {
		switch {
		case strings.Contains(htmlString, fullPasswordCache):
			return "", ctxerr.Wrap(ctx,
				NewNDESPasswordCacheFullError("the password cache is full; please increase the number of cached passwords in NDES; by default, NDES caches 5 passwords and they expire 60 minutes after they are created"))
		case strings.Contains(htmlString, ndesInsufficientPermissions):
			return "", ctxerr.Wrap(ctx,
				NewNDESInsufficientPermissionsError("this account does not have sufficient permissions to enroll with SCEP. Please use a different account with NDES SCEP enroll permissions."))
		}
		return "", ctxerr.Wrap(ctx,
			NewNDESInvalidError("could not retrieve the enrollment challenge password; invalid admin URL or credentials; please correct and try again"))
	}
	return challenge, nil
}

func ValidateNDESSCEPURL(ctx context.Context, proxy fleet.NDESSCEPProxyIntegration, logger log.Logger) error {
	client, err := scepclient.New(proxy.URL, logger, NDESTimeout)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "creating SCEP client; invalid SCEP URL; please correct and try again")
	}

	certs, _, err := client.GetCACert(ctx, "")
	if err != nil {
		return ctxerr.Wrap(ctx, err, "could not retrieve CA certificate from SCEP URL; invalid SCEP URL; please correct and try again")
	}
	if len(certs) == 0 {
		return ctxerr.New(ctx, "SCEP URL did not return a CA certificate")
	}
	return nil
}

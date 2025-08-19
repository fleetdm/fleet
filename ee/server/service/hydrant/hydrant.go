package hydrant

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/log"
)

// defaultTimeout is the timeout for requests.
const defaultTimeout = 20 * time.Second

type Service struct {
	logger  kitlog.Logger
	timeout time.Duration
	client  *http.Client
}

// Compile-time check for HydrantService interface
var _ fleet.HydrantService = (*Service)(nil)

func NewService(opts ...Opt) fleet.HydrantService {
	s := &Service{}
	s.populateOpts(opts)
	s.client = fleethttp.NewClient(fleethttp.WithTimeout(s.timeout))
	return s
}

// Opt is the type for Hydrant integration options.
type Opt func(*Service)

// WithTimeout sets the timeout to use for the HTTP client.
func WithTimeout(t time.Duration) Opt {
	return func(s *Service) {
		s.timeout = t
	}
}

// WithLogger sets the logger to use for the service.
func WithLogger(logger kitlog.Logger) Opt {
	return func(s *Service) {
		s.logger = logger
	}
}

func (s *Service) populateOpts(opts []Opt) {
	for _, opt := range opts {
		opt(s)
	}
	if s.timeout <= 0 {
		s.timeout = defaultTimeout
	}
	if s.logger == nil {
		s.logger = kitlog.NewLogfmtLogger(kitlog.NewSyncWriter(os.Stdout))
	}
}

func (s *Service) ValidateHydrantURL(ctx context.Context, hydrantCA fleet.HydrantCA) error {
	reqURL := hydrantCA.URL + "/cacerts"
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "creating Hydrant CA request")
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "sending Hydrant CA request")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ctxerr.Errorf(ctx, "unexpected Hydrant CA status code: %d", resp.StatusCode)
	}
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/pkcs7-mime") {
		return ctxerr.Errorf(ctx, "unexpected Hydrant CA content type: %s", contentType)
	}
	// For now we are just verifying that there is a body of the reportedly correct format. We could
	// possibly do more. A better implementation would be similar to Digicert's which validates the
	// credentials in addition to the URL but I don't see a way to do that with Hydrant's API.
	caCerts, err := io.ReadAll(resp.Body)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "reading Hydrant CA response body")
	}
	if len(caCerts) == 0 {
		return ctxerr.Errorf(ctx, "no CA certificates found in Hydrant CA /cacerts response. URL may be incorrect")
	}
	return nil
}

func (s *Service) GetCertificate(ctx context.Context, hydrantCA fleet.HydrantCA, csr string) (*fleet.HydrantCertificate, error) {
	reqURL, err := url.Parse(hydrantCA.URL + "/simpleenroll")
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "parsing Hydrant CA URL")
	}
	apiCredential := hydrantCA.ClientID + ":" + hydrantCA.ClientSecret
	encodedCredential := base64.StdEncoding.EncodeToString([]byte(apiCredential))

	hydrantRequest, err := http.NewRequestWithContext(ctx, "POST", reqURL.String(), strings.NewReader(csr))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating Hydrant CA request")
	}
	hydrantRequest.Header.Set("Content-Type", "application/pkcs10")
	hydrantRequest.Header.Set("Accept", "application/pkcs7-mime")
	hydrantRequest.Header.Set("Authorization", "Basic "+encodedCredential)
	resp, err := s.client.Do(hydrantRequest)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "sending Hydrant CA request")
	}
	defer resp.Body.Close()
	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "reading Hydrant CA response body")
	}
	if resp.StatusCode != http.StatusOK {
		bytesToLog := bytes
		// Limit logged data in case we get a huge response(a certificate perhaps?)
		if len(bytes) > 1000 {
			bytesToLog = bytes[:1000]
		}
		s.logger.Log("msg", "unexpected Hydrant CA status code", "status_code", resp.StatusCode, "response_body", string(bytesToLog))
		return nil, ctxerr.Errorf(ctx, "unexpected Hydrant CA status code: %d", resp.StatusCode)
	}

	return &fleet.HydrantCertificate{
		Certificate: bytes,
	}, nil
}

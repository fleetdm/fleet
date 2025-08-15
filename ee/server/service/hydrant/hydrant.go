package hydrant

import (
	"context"
	"encoding/base64"
	"fmt"
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

// REST client for https://one.digicert.com/mpki/docs/swagger-ui/index.html

// defaultTimeout is the timeout for requests.
const defaultTimeout = 20 * time.Second

const (
	errMessageInvalidAPIToken = "The API token configured in %s certificate authority is invalid. " + // nolint:gosec // ignore G101
		"Status code for POST request: %d"
	errMessageInvalidProfile = "The \"profile_id\" configured in %s certificate authority doesn't exist. Status code for POST request: %d"
)

type Service struct {
	logger  kitlog.Logger
	timeout time.Duration
}

// Compile-time check for DigiCertService interface
var _ fleet.HydrantService = (*Service)(nil)

func NewService(opts ...Opt) fleet.HydrantService {
	s := &Service{}
	s.populateOpts(opts)
	return s
}

// Opt is the type for DigiCert integration options.
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
	client := fleethttp.NewClient(fleethttp.WithTimeout(s.timeout))
	reqURL := hydrantCA.URL + "/cacerts"
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "creating Hydrant CA request")
	}
	resp, err := client.Do(req)
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
	// possibly do more.
	// TODO: A better implementation would be similar to Digicert's which validates the credentials in
	// addition to the URL.
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
	client := fleethttp.NewClient(fleethttp.WithTimeout(s.timeout))

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
	hydrantRequest.Header.Set("Accept", "application/pkcs12")
	hydrantRequest.Header.Set("Authorization", "Basic "+encodedCredential)
	resp, err := client.Do(hydrantRequest)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "sending Hydrant CA request")
	}
	defer resp.Body.Close()
	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "reading Hydrant CA response body")
	}
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Response body: %s\n", string(bytes))
		return nil, ctxerr.Errorf(ctx, "unexpected Hydrant CA status code: %d", resp.StatusCode)
	}

	return &fleet.HydrantCertificate{
		Certificate: bytes,
	}, nil
}

/*
REF
TODO HCA REMOVE THIS
module.exports = {


  friendlyName: 'Get a device certificate via EST protocol',

  description: 'Take a certificate signing request and authentication token, then use the EST protocol to request a certificate to be issued.',

  extendedDescription: 'This action is the result of a customer hackathon working on issuing device certificates to Linux devices.',

  moreInfoUrl: 'https://github.com/fleetdm/confidential/issues/8785',


  inputs: {
    csrData: {
      required: true,
      type: 'string',
      description: 'Certificate Signing Request (CSR) data'
    },
    authToken: {
      required: true,
      type: 'string',
      description: 'Authorization token provided by IdP',
    },
    introspectEndpoint: {
      required: true,
      type: 'string',
      description: 'IdP introspect endpoint URL'
    },
    idpClientId: {
      required: true,
      type: 'string',
      description: 'IdP client ID'
    },
    estEndpoint: {
      required: true,
      type: 'string',
      description: 'EST protocol endpoint URL'
    },
    estClientId: {
      required: true,
      type: 'string',
      description: 'EST client ID'
    },
    estClientKey: {
      required: true,
      type: 'string',
      description: 'EST client key'
    }
  },


  exits: {

    success: {
      description: 'Successfully generated certificate',
      extendedDescription: 'This action is the result of a hackathon where we were relying on this datashape.',
      outputType: {certificate:'string'},
      outputFriendlyName: 'Certificate',
    },

    invalidToken: {
      description: 'The IdP auth token is invalid',
      statusCode: 403,
    },

    invalidCsr: {
      description: 'The provided CSR data was invalid.',
      responseType: 'badRequest'
    }

  },

  fn: async function ({ csrData, authToken, introspectEndpoint, idpClientId, estEndpoint, estClientId, estClientKey }) {

    // Ask the IdP to introspect the auth token (ensure it's valid and extract the values).
    const introspectResponse = await sails.helpers.http.sendHttpRequest.with({
      url: introspectEndpoint,
      method: 'POST',
      enctype: 'application/x-www-form-urlencoded',
      body: {
        'client_id': idpClientId,
        'token': authToken,
      },
    });

    if (!introspectResponse.body) {
      throw 'invalidToken';
    }

    const introspectBody = JSON.parse(introspectResponse.body);
    if (!introspectBody.active) {
      throw 'invalidToken';
    }
    const introspectUsername = introspectBody.username;

    // Extract the email and username from the CSR. Ensure they match.
    let jsrsasign = require('jsrsasign');
    const csrUtil = jsrsasign.asn1.csr.CSRUtil;
    const csrObj = csrUtil.getParam(csrData);
    let csrEmail = '';
    let csrUsername = '';
    for (const extension of csrObj.extreq) {
      if (extension.extname === 'subjectAltName') {
        for (const extentry of extension.array) {
          if ('rfc822' in extentry) {
            csrEmail = extentry.rfc822;
          }
          if ('other' in extentry) {
            if ('oid' in extentry.other && extentry.other.oid === '1.3.6.1.4.1.311.20.2.3') {
              csrUsername = extentry.other.value.utf8str.str;
            }
          }
        }
      }
    }
    if (csrEmail === '') {
      throw 'invalidCsr';
    }
    if (!csrEmail.startsWith(csrUsername)) {
      throw 'invalidCsr';
    }

    // Ensure username from IdP auth matches username in CSR. If they don't match, perhaps the user
    // is trying to get a certificate with another user's name?
    if (csrEmail !== introspectUsername) {
      throw 'invalidToken';
    }

    // Ask the PKI provider for a certificate
    const request = require('@sailshq/request');
    const estResponse = await new Promise((resolve, reject) => {
      request({
        url: estEndpoint,
        method: 'POST',
        body: csrData.replace(/(-----(BEGIN|END) CERTIFICATE REQUEST-----|\n)/g, ''),
        headers: {
          'Content-Type': 'application/pkcs10',
          'Authorization': `Basic ${Buffer.from(`${estClientId}:${estClientKey}`).toString('base64')}`,
        },
      }, (err, response)=>{
        if (err) {
          reject(err);
        } else {
          response.body = '-----BEGIN CERTIFICATE-----\n' + response.body + '\n-----END CERTIFICATE-----';
          resolve(response);
        }
      });
    });

    return estResponse.body;
  }

};
*/

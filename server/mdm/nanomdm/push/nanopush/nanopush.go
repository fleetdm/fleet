// Pacakge nanopush implements an Apple APNs HTTP/2 service for MDM.
// It implements the PushProvider and PushProviderFactory interfaces.
package nanopush

import (
	"crypto/tls"
	"errors"
	"net/http"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push"
	"golang.org/x/net/http2"
)

// NewClient describes a callback for setting up an HTTP client for Push notifications.
type NewClient func(*tls.Certificate) (*http.Client, error)

// ClientWithCert configures an mTLS client cert on the HTTP client.
func ClientWithCert(client *http.Client, cert *tls.Certificate) (*http.Client, error) {
	if cert == nil {
		return client, errors.New("no cert provided")
	}
	if client == nil {
		clone := *http.DefaultClient
		client = &clone
	}
	config := &tls.Config{
		Certificates: []tls.Certificate{*cert},
		MinVersion:   tls.VersionTLS12,
	}
	if client.Transport == nil {
		client.Transport = &http.Transport{} // nolint: gocritic // allow not using fleethttp.NewClient
	}
	transport := client.Transport.(*http.Transport)
	transport.TLSClientConfig = config
	// force HTTP/2
	err := http2.ConfigureTransport(transport)
	return client, err
}

func defaultNewClient(cert *tls.Certificate) (*http.Client, error) {
	return ClientWithCert(nil, cert)
}

// Factory instantiates new PushProviders.
type Factory struct {
	newClient  NewClient
	expiration time.Duration
	workers    int
}

type Option func(*Factory)

// WithNewClient sets a callback to setup an HTTP client for each
// new Push provider.
func WithNewClient(newClient NewClient) Option {
	return func(f *Factory) {
		f.newClient = newClient
	}
}

// WithExpiration sets the APNs expiration time for the push notifications.
func WithExpiration(expiration time.Duration) Option {
	return func(f *Factory) {
		f.expiration = expiration
	}
}

// WithWorkers sets how many worker goroutines to use when sending pushes.
func WithWorkers(workers int) Option {
	return func(f *Factory) {
		f.workers = workers
	}
}

// NewFactory creates a new Factory.
func NewFactory(opts ...Option) *Factory {
	f := &Factory{
		newClient: defaultNewClient,
		workers:   5,
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

// NewPushProvider generates a new PushProvider given a tls keypair.
func (f *Factory) NewPushProvider(cert *tls.Certificate) (push.PushProvider, error) {
	p := &Provider{
		expiration: f.expiration,
		workers:    f.workers,
		baseURL:    Production,
	}
	var err error
	p.client, err = f.newClient(cert)
	return p, err
}

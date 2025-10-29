package scepclient

import (
	"time"

	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

// Client is a SCEP Client
type Client interface {
	scepserver.Service
	Supports(capacity string) bool
}

type clientOpts struct {
	timeout  *time.Duration
	rootCA   string
	insecure bool
}

// Option is a functional option for configuring a SCEP Client
type Option func(*clientOpts)

// WithRootCA sets the root CA file to use when connecting to the SCEP server.
func WithRootCA(rootCA string) Option {
	return func(c *clientOpts) {
		c.rootCA = rootCA
	}
}

// Insecure configures the client to not verify server certificates.
// Only used for tests.
func Insecure() Option {
	return func(c *clientOpts) {
		c.insecure = true
	}
}

// WithTimeout configures the timeout for SCEP client requests.
func WithTimeout(timeout *time.Duration) Option {
	return func(c *clientOpts) {
		c.timeout = timeout
	}
}

// New creates a SCEP Client.
func New(
	serverURL string,
	logger log.Logger,
	opts ...Option,
) (Client, error) {
	var co clientOpts
	for _, fn := range opts {
		fn(&co)
	}
	clientOpts := []scepserver.ClientOption{
		scepserver.WithClientTimeout(co.timeout),
		scepserver.WithClientRootCA(co.rootCA),
	}
	if co.insecure {
		clientOpts = append(clientOpts, scepserver.ClientInsecure())
	}
	endpoints, err := scepserver.MakeClientEndpoints(serverURL, clientOpts...)
	if err != nil {
		return nil, err
	}
	logger = level.Info(logger)
	endpoints.GetEndpoint = scepserver.EndpointLoggingMiddleware(logger)(endpoints.GetEndpoint)
	endpoints.PostEndpoint = scepserver.EndpointLoggingMiddleware(logger)(endpoints.PostEndpoint)
	return endpoints, nil
}

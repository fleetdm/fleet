package scepclient

import (
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

// Client is a SCEP Client
type Client interface {
	scepserver.Service
	Supports(cap string) bool
}

// New creates a SCEP Client.
func New(
	serverURL string,
	logger log.Logger,
) (Client, error) {
	endpoints, err := scepserver.MakeClientEndpoints(serverURL)
	if err != nil {
		return nil, err
	}
	logger = level.Info(logger)
	endpoints.GetEndpoint = scepserver.EndpointLoggingMiddleware(logger)(endpoints.GetEndpoint)
	endpoints.PostEndpoint = scepserver.EndpointLoggingMiddleware(logger)(endpoints.PostEndpoint)
	return endpoints, nil
}

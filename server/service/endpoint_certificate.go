package service

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/kolide/kolide/server/kolide"
)

type certificateResponse struct {
	CertificateChain []byte `json:"certificate_chain"`
	Err              error  `json:"error,omitempty"`
}

func (r certificateResponse) error() error { return r.Err }

func makeCertificateEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		chain, err := svc.CertificateChain(ctx)
		if err != nil {
			return certificateResponse{Err: err}, nil
		}
		return certificateResponse{CertificateChain: chain}, nil
	}
}

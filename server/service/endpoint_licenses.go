package service

import (
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/kolide/kolide/server/kolide"
	"golang.org/x/net/context"
)

type licenseRequest struct {
	License string `json:"license"`
}

type license struct {
	Token        string    `json:"token"`
	Expiry       time.Time `json:"expiry"`
	AllowedHosts int       `json:"allowed_hosts"`
	Hosts        uint      `json:"hosts"`
	Evaluation   bool      `json:"evaluation"`
}

type licenseResponse struct {
	License license `json:"license,omitempty"`
	Err     error   `json:"error,omitempty"`
}

func (lr licenseResponse) error() error { return lr.Err }

func makeUpdateLicenseEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		lr := request.(licenseRequest)
		updated, err := svc.SaveLicense(ctx, lr.License)
		if err != nil {
			return licenseResponse{Err: err}, nil
		}
		claims, err := updated.Claims()
		if err != nil {
			return licenseResponse{Err: err}, nil
		}
		response := &licenseResponse{
			License: license{
				Token:        *updated.Token,
				Expiry:       claims.ExpiresAt,
				AllowedHosts: claims.HostLimit,
				Hosts:        updated.HostCount,
			},
		}
		return response, nil
	}
}

func makeGetLicenseEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return nil, nil
	}
}

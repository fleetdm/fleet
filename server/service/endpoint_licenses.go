package service

import (
	"context"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/kolide/fleet/server/kolide"
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
	Revoked      bool      `json:"revoked"`
	Organization string    `json:"organization"`
}

type licenseResponse struct {
	License license `json:"license,omitempty"`
	Err     error   `json:"error,omitempty"`
}

func (lr licenseResponse) error() error { return lr.Err }

// makeUpdateLicenseEndpoint is used by admins to replace or update
// licenses.
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
				Revoked:      updated.Revoked,
				Organization: claims.OrganizationName,
			},
		}
		return response, nil
	}
}

func makeGetLicenseEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		lic, err := svc.License(ctx)
		if err != nil {
			return licenseResponse{Err: err}, nil
		}
		claims, err := lic.Claims()
		if err != nil {
			return licenseResponse{Err: err}, nil
		}
		response := licenseResponse{
			License: license{
				Expiry:       claims.ExpiresAt,
				AllowedHosts: claims.HostLimit,
				Hosts:        lic.HostCount,
				Token:        *lic.Token,
				Revoked:      lic.Revoked,
				Organization: claims.OrganizationName,
			},
		}
		return response, nil
	}
}

// makeSetupLicenseEndpoint is only to be used once, if a license is successfully
// installed we return an error if it is called again.  Note that this endpoint
// requires no authentication.  Once a license is installed the user will be
// redirected to login or setup, depending on whether setup is complete or not.
func makeSetupLicenseEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(licenseRequest)
		saved, err := svc.SaveLicense(ctx, req.License)
		if err != nil {
			return licenseResponse{Err: err}, nil
		}
		claims, err := saved.Claims()
		if err != nil {
			return licenseResponse{Err: licensingError{err.Error()}}, nil
		}
		response := licenseResponse{
			License: license{
				Token:        *saved.Token,
				Revoked:      saved.Revoked,
				Expiry:       claims.ExpiresAt,
				AllowedHosts: claims.HostLimit,
				Hosts:        saved.HostCount,
				Organization: claims.OrganizationName,
			},
		}
		return response, nil
	}
}

package service

import (
	"context"

	"github.com/kolide/fleet/server/kolide"
)

func (svc service) License(ctx context.Context) (*kolide.License, error) {
	license, err := svc.ds.License()
	if err != nil {
		return nil, err
	}
	return license, nil
}

func (svc service) SaveLicense(ctx context.Context, jwtToken string) (*kolide.License, error) {
	publicKey, err := svc.ds.LicensePublicKey(jwtToken)
	if err != nil {
		return nil, licensingError{err.Error()}
	}
	updated, err := svc.ds.SaveLicense(jwtToken, publicKey)
	if err != nil {
		return nil, err
	}
	// schedule a checkin with the license server.
	go func() { svc.licenseChecker.RunLicenseCheck(ctx) }()
	return updated, nil
}

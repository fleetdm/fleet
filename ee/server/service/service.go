package service

import (
	"context"
	"fmt"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/kit/log"
)

type Service struct {
	fleet.Service

	ds      fleet.Datastore
	logger  kitlog.Logger
	config  config.FleetConfig
	clock   clock.Clock
	authz   *authz.Authorizer
	license *fleet.LicenseInfo
}

func (s *Service) ExampleMethod(ctx context.Context) error {
	fmt.Println("premium example method!")
	return nil
}

func NewService(
	svc fleet.Service,
	ds fleet.Datastore,
	logger kitlog.Logger,
	config config.FleetConfig,
	mailService fleet.MailService,
	c clock.Clock,
	license *fleet.LicenseInfo,
) (*Service, error) {

	authorizer, err := authz.NewAuthorizer()
	if err != nil {
		return nil, fmt.Errorf("new authorizer: %w", err)
	}

	eesvc := &Service{
		Service: svc,
		ds:      ds,
		logger:  logger,
		config:  config,
		clock:   c,
		authz:   authorizer,
		license: license,
	}
	svc.HijackWith(eesvc)
	return eesvc, nil
}

package service

import (
	"fmt"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/kit/log"
	"github.com/micromdm/nanodep/storage"
)

// Service wraps a free Service and implements additional premium functionality on top of it.
type Service struct {
	fleet.Service

	ds                fleet.Datastore
	logger            kitlog.Logger
	config            config.FleetConfig
	clock             clock.Clock
	authz             *authz.Authorizer
	depStorage        storage.AllStorage
	mdmAppleCommander fleet.MDMAppleCommandIssuer
	mdmPushCertTopic  string
}

func NewService(
	svc fleet.Service,
	ds fleet.Datastore,
	logger kitlog.Logger,
	config config.FleetConfig,
	mailService fleet.MailService,
	c clock.Clock,
	depStorage storage.AllStorage,
	mdmAppleCommander fleet.MDMAppleCommandIssuer,
	mdmPushCertTopic string,
) (*Service, error) {
	authorizer, err := authz.NewAuthorizer()
	if err != nil {
		return nil, fmt.Errorf("new authorizer: %w", err)
	}

	eeservice := &Service{
		Service:           svc,
		ds:                ds,
		logger:            logger,
		config:            config,
		clock:             c,
		authz:             authorizer,
		depStorage:        depStorage,
		mdmAppleCommander: mdmAppleCommander,
		mdmPushCertTopic:  mdmPushCertTopic,
	}

	// Override methods that can't be easily overriden via
	// embedding.
	svc.SetEnterpriseOverrides(fleet.EnterpriseOverrides{
		HostFeatures:               eeservice.HostFeatures,
		TeamByIDOrName:             eeservice.teamByIDOrName,
		UpdateTeamMDMAppleSettings: eeservice.updateTeamMDMAppleSettings,
	})

	return eeservice, nil
}

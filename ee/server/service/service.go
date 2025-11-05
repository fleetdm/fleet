package service

import (
	"fmt"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/storage"
	"github.com/fleetdm/fleet/v4/server/sso"
	kitlog "github.com/go-kit/log"
)

// Service wraps a free Service and implements additional premium functionality on top of it.
type Service struct {
	fleet.Service

	ds                     fleet.Datastore
	logger                 kitlog.Logger
	config                 config.FleetConfig
	clock                  clock.Clock
	authz                  *authz.Authorizer
	depStorage             storage.AllDEPStorage
	mdmAppleCommander      fleet.MDMAppleCommandIssuer
	ssoSessionStore        sso.SessionStore
	depService             *apple_mdm.DEPService
	profileMatcher         fleet.ProfileMatcher
	softwareInstallStore   fleet.SoftwareInstallerStore
	bootstrapPackageStore  fleet.MDMBootstrapPackageStore
	softwareTitleIconStore fleet.SoftwareTitleIconStore
	distributedLock        fleet.Lock
	keyValueStore          fleet.KeyValueStore
	scepConfigService      fleet.SCEPConfigService
	digiCertService        fleet.DigiCertService
	androidModule          android.Service
	estService             fleet.ESTService
}

func NewService(
	svc fleet.Service,
	ds fleet.Datastore,
	logger kitlog.Logger,
	config config.FleetConfig,
	mailService fleet.MailService,
	c clock.Clock,
	depStorage storage.AllDEPStorage,
	mdmAppleCommander fleet.MDMAppleCommandIssuer,
	sso sso.SessionStore,
	profileMatcher fleet.ProfileMatcher,
	softwareInstallStore fleet.SoftwareInstallerStore,
	bootstrapPackageStore fleet.MDMBootstrapPackageStore,
	softwareTitleIconStore fleet.SoftwareTitleIconStore,
	distributedLock fleet.Lock,
	keyValueStore fleet.KeyValueStore,
	scepConfigService fleet.SCEPConfigService,
	digiCertService fleet.DigiCertService,
	androidService android.Service,
	estService fleet.ESTService,
) (*Service, error) {
	authorizer, err := authz.NewAuthorizer()
	if err != nil {
		return nil, fmt.Errorf("new authorizer: %w", err)
	}

	eeservice := &Service{
		Service:                svc,
		ds:                     ds,
		logger:                 logger,
		config:                 config,
		clock:                  c,
		authz:                  authorizer,
		depStorage:             depStorage,
		mdmAppleCommander:      mdmAppleCommander,
		ssoSessionStore:        sso,
		depService:             apple_mdm.NewDEPService(ds, depStorage, logger),
		profileMatcher:         profileMatcher,
		softwareInstallStore:   softwareInstallStore,
		bootstrapPackageStore:  bootstrapPackageStore,
		softwareTitleIconStore: softwareTitleIconStore,
		distributedLock:        distributedLock,
		keyValueStore:          keyValueStore,
		scepConfigService:      scepConfigService,
		digiCertService:        digiCertService,
		androidModule:          androidService,
		estService:             estService,
	}

	// Override methods that can't be easily overriden via
	// embedding.
	svc.SetEnterpriseOverrides(fleet.EnterpriseOverrides{
		HostFeatures:                      eeservice.HostFeatures,
		TeamByIDOrName:                    eeservice.teamByIDOrName,
		UpdateTeamMDMDiskEncryption:       eeservice.updateTeamMDMDiskEncryption,
		MDMAppleEnableFileVaultAndEscrow:  eeservice.MDMAppleEnableFileVaultAndEscrow,
		MDMAppleDisableFileVaultAndEscrow: eeservice.MDMAppleDisableFileVaultAndEscrow,
		DeleteMDMAppleSetupAssistant:      eeservice.DeleteMDMAppleSetupAssistant,
		MDMAppleSyncDEPProfiles:           eeservice.mdmAppleSyncDEPProfiles,
		DeleteMDMAppleBootstrapPackage:    eeservice.DeleteMDMAppleBootstrapPackage,
		MDMWindowsEnableOSUpdates:         eeservice.mdmWindowsEnableOSUpdates,
		MDMWindowsDisableOSUpdates:        eeservice.mdmWindowsDisableOSUpdates,
		MDMAppleEditedAppleOSUpdates:      eeservice.mdmAppleEditedAppleOSUpdates,
		SetupExperienceNextStep:           eeservice.SetupExperienceNextStep,
		GetVPPTokenIfCanInstallVPPApps:    eeservice.GetVPPTokenIfCanInstallVPPApps,
		InstallVPPAppPostValidation:       eeservice.InstallVPPAppPostValidation,
	})

	return eeservice, nil
}

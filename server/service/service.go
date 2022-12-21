// Package service holds the implementation of the fleet interface and HTTP
// endpoints for the API
package service

import (
	"context"
	"fmt"
	"html/template"
	"sync"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/logging"
	"github.com/fleetdm/fleet/v4/server/service/async"
	"github.com/fleetdm/fleet/v4/server/sso"
	kitlog "github.com/go-kit/kit/log"
	nanodep_storage "github.com/micromdm/nanodep/storage"
	nanomdm_push "github.com/micromdm/nanomdm/push"
	nanomdm_storage "github.com/micromdm/nanomdm/storage"
)

var _ fleet.Service = (*Service)(nil)

// Service is the struct implementing fleet.Service. Create a new one with NewService.
type Service struct {
	ds             fleet.Datastore
	task           *async.Task
	carveStore     fleet.CarveStore
	installerStore fleet.InstallerStore
	resultStore    fleet.QueryResultStore
	liveQueryStore fleet.LiveQueryStore
	logger         kitlog.Logger
	config         config.FleetConfig
	clock          clock.Clock

	osqueryLogWriter *logging.OsqueryLogger

	mailService     fleet.MailService
	ssoSessionStore sso.SessionStore

	failingPolicySet  fleet.FailingPolicySet
	enrollHostLimiter fleet.EnrollHostLimiter

	authz *authz.Authorizer

	jitterMu *sync.Mutex
	jitterH  map[time.Duration]*jitterHashTable

	geoIP fleet.GeoIP

	*fleet.EnterpriseOverrides

	depStorage       nanodep_storage.AllStorage
	mdmStorage       nanomdm_storage.AllStorage
	mdmPushService   nanomdm_push.Pusher
	mdmPushCertTopic string

	cronSchedulesService fleet.CronSchedulesService
}

func (s *Service) LookupGeoIP(ctx context.Context, ip string) *fleet.GeoLocation {
	return s.geoIP.Lookup(ctx, ip)
}

func (s *Service) SetEnterpriseOverrides(overrides fleet.EnterpriseOverrides) {
	s.EnterpriseOverrides = &overrides
}

// NewService creates a new service from the config struct
func NewService(
	ctx context.Context,
	ds fleet.Datastore,
	task *async.Task,
	resultStore fleet.QueryResultStore,
	logger kitlog.Logger,
	osqueryLogger *logging.OsqueryLogger,
	config config.FleetConfig,
	mailService fleet.MailService,
	c clock.Clock,
	sso sso.SessionStore,
	lq fleet.LiveQueryStore,
	carveStore fleet.CarveStore,
	installerStore fleet.InstallerStore,
	failingPolicySet fleet.FailingPolicySet,
	geoIP fleet.GeoIP,
	enrollHostLimiter fleet.EnrollHostLimiter,
	depStorage nanodep_storage.AllStorage,
	mdmStorage nanomdm_storage.AllStorage,
	mdmPushService nanomdm_push.Pusher,
	mdmPushCertTopic string,
	cronSchedulesService fleet.CronSchedulesService,
) (fleet.Service, error) {
	authorizer, err := authz.NewAuthorizer()
	if err != nil {
		return nil, fmt.Errorf("new authorizer: %w", err)
	}

	svc := &Service{
		ds:                   ds,
		task:                 task,
		carveStore:           carveStore,
		installerStore:       installerStore,
		resultStore:          resultStore,
		liveQueryStore:       lq,
		logger:               logger,
		config:               config,
		clock:                c,
		osqueryLogWriter:     osqueryLogger,
		mailService:          mailService,
		ssoSessionStore:      sso,
		failingPolicySet:     failingPolicySet,
		authz:                authorizer,
		jitterH:              make(map[time.Duration]*jitterHashTable),
		jitterMu:             new(sync.Mutex),
		geoIP:                geoIP,
		enrollHostLimiter:    enrollHostLimiter,
		depStorage:           depStorage,
		mdmStorage:           mdmStorage,
		mdmPushService:       mdmPushService,
		mdmPushCertTopic:     mdmPushCertTopic,
		cronSchedulesService: cronSchedulesService,
	}
	return validationMiddleware{svc, ds, sso}, nil
}

func (s *Service) SendEmail(mail fleet.Email) error {
	return s.mailService.SendEmail(mail)
}

// logRoleChangeActivities stores the activities for role changes, globally and in teams.
func (svc *Service) logRoleChangeActivities(ctx context.Context, adminUser *fleet.User, oldRole *string, oldTeams []fleet.UserTeam, user *fleet.User) error {
	if user.GlobalRole != nil && (oldRole == nil || *oldRole != *user.GlobalRole) {
		if err := svc.ds.NewActivity(
			ctx,
			adminUser,
			fleet.ActivityTypeChangedUserGlobalRole,
			&map[string]interface{}{"user_name": user.Name, "user_id": user.ID, "user_email": user.Email, "role": *user.GlobalRole},
		); err != nil {
			return err
		}
	}
	if user.GlobalRole == nil && oldRole != nil {
		if err := svc.ds.NewActivity(
			ctx,
			adminUser,
			fleet.ActivityTypeDeletedUserGlobalRole,
			&map[string]interface{}{"user_name": user.Name, "user_id": user.ID, "user_email": user.Email, "role": *oldRole},
		); err != nil {
			return err
		}
	}
	oldTeamsLookup := make(map[uint]fleet.UserTeam, len(oldTeams))
	for _, t := range oldTeams {
		oldTeamsLookup[t.ID] = t
	}

	newTeamLookup := make(map[uint]struct{}, len(user.Teams))
	for _, t := range user.Teams {
		newTeamLookup[t.ID] = struct{}{}
		o, ok := oldTeamsLookup[t.ID]
		if ok && o.Role == t.Role {
			continue
		}
		if err := svc.ds.NewActivity(
			ctx,
			adminUser,
			fleet.ActivityTypeChangedUserTeamRole,
			&map[string]interface{}{"user_name": user.Name, "user_id": user.ID, "user_email": user.Email, "team_name": t.Name, "team_id": t.ID, "role": t.Role},
		); err != nil {
			return err
		}
	}
	for _, o := range oldTeams {
		if _, ok := newTeamLookup[o.ID]; ok {
			continue
		}
		if err := svc.ds.NewActivity(
			ctx,
			adminUser,
			fleet.ActivityTypeDeletedUserTeamRole,
			&map[string]interface{}{"user_name": user.Name, "user_id": user.ID, "user_email": user.Email, "team_name": o.Name, "team_id": o.ID, "role": o.Role},
		); err != nil {
			return err
		}
	}
	return nil
}

type validationMiddleware struct {
	fleet.Service
	ds              fleet.Datastore
	ssoSessionStore sso.SessionStore
}

// getAssetURL simply returns the base url used for retrieving image assets from fleetdm.com.
func getAssetURL() template.URL {
	return template.URL("https://fleetdm.com/images/permanent")
}

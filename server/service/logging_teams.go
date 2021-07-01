package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (mw loggingMiddleware) NewTeam(ctx context.Context, p fleet.TeamPayload) (*fleet.Team, error) {
	var (
		team         *fleet.Team
		loggedInUser = "unauthenticated"
		err          error
	)

	if vc, ok := viewer.FromContext(ctx); ok {

		loggedInUser = vc.Email()
	}

	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "NewTeam",
			"err", err,
			"user", loggedInUser,
			"took", time.Since(begin),
		)
	}(time.Now())

	team, err = mw.Service.NewTeam(ctx, p)
	return team, err
}

func (mw loggingMiddleware) ModifyTeam(ctx context.Context, id uint, p fleet.TeamPayload) (*fleet.Team, error) {
	var (
		team         *fleet.Team
		loggedInUser = "unauthenticated"
		err          error
	)

	if vc, ok := viewer.FromContext(ctx); ok {

		loggedInUser = vc.Email()
	}

	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "ModifyTeam",
			"err", err,
			"user", loggedInUser,
			"took", time.Since(begin),
		)
	}(time.Now())

	team, err = mw.Service.ModifyTeam(ctx, id, p)
	return team, err
}

func (mw loggingMiddleware) ModifyTeamAgentOptions(ctx context.Context, id uint, options json.RawMessage) (*fleet.Team, error) {
	var (
		team         *fleet.Team
		loggedInUser = "unauthenticated"
		err          error
	)

	if vc, ok := viewer.FromContext(ctx); ok {

		loggedInUser = vc.Email()
	}

	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "ModifyTeamAgentOptions",
			"err", err,
			"user", loggedInUser,
			"took", time.Since(begin),
		)
	}(time.Now())

	team, err = mw.Service.ModifyTeamAgentOptions(ctx, id, options)
	return team, err
}

func (mw loggingMiddleware) AddTeamUsers(ctx context.Context, id uint, users []fleet.TeamUser) (*fleet.Team, error) {
	var (
		team         *fleet.Team
		loggedInUser = "unauthenticated"
		err          error
	)

	if vc, ok := viewer.FromContext(ctx); ok {

		loggedInUser = vc.Email()
	}

	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "AddTeamUsers",
			"err", err,
			"user", loggedInUser,
			"took", time.Since(begin),
			"users", users,
		)
	}(time.Now())

	team, err = mw.Service.AddTeamUsers(ctx, id, users)
	return team, err
}

func (mw loggingMiddleware) DeleteTeamUsers(ctx context.Context, id uint, users []fleet.TeamUser) (*fleet.Team, error) {
	var (
		team         *fleet.Team
		loggedInUser = "unauthenticated"
		err          error
	)

	if vc, ok := viewer.FromContext(ctx); ok {

		loggedInUser = vc.Email()
	}

	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "DeleteTeamUsers",
			"err", err,
			"user", loggedInUser,
			"took", time.Since(begin),
			"users", users,
		)
	}(time.Now())

	team, err = mw.Service.DeleteTeamUsers(ctx, id, users)
	return team, err
}

func (mw loggingMiddleware) DeleteTeam(ctx context.Context, id uint) error {
	var (
		loggedInUser = "unauthenticated"
		err          error
	)

	if vc, ok := viewer.FromContext(ctx); ok {

		loggedInUser = vc.Email()
	}

	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "DeleteTeam",
			"err", err,
			"user", loggedInUser,
			"took", time.Since(begin),
			"id", id,
		)
	}(time.Now())
	err = mw.Service.DeleteTeam(ctx, id)
	return err
}

func (mw loggingMiddleware) ListTeams(ctx context.Context, opt fleet.ListOptions) ([]*fleet.Team, error) {
	var (
		teams []*fleet.Team
		err   error
	)

	defer func(begin time.Time) {
		_ = mw.loggerDebug(err).Log(
			"method", "ListTeams",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	teams, err = mw.Service.ListTeams(ctx, opt)
	return teams, err
}

func (mw loggingMiddleware) ListTeamUsers(ctx context.Context, teamID uint, opt fleet.ListOptions) ([]*fleet.User, error) {
	var (
		users []*fleet.User
		err   error
	)

	defer func(begin time.Time) {
		_ = mw.loggerDebug(err).Log(
			"method", "ListTeamUsers",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	users, err = mw.Service.ListTeamUsers(ctx, teamID, opt)
	return users, err
}

func (mw loggingMiddleware) TeamEnrollSecrets(ctx context.Context, teamID uint) ([]*fleet.EnrollSecret, error) {
	var (
		secrets []*fleet.EnrollSecret
		err     error
	)

	defer func(begin time.Time) {
		_ = mw.loggerDebug(err).Log(
			"method", "TeamEnrollSecrets",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	secrets, err = mw.Service.TeamEnrollSecrets(ctx, teamID)
	return secrets, err
}

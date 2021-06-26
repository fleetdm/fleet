package service

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (mw loggingMiddleware) NewPack(ctx context.Context, p fleet.PackPayload) (*fleet.Pack, error) {
	var (
		pack         *fleet.Pack
		loggedInUser = "unauthenticated"
		err          error
	)

	if vc, ok := viewer.FromContext(ctx); ok {

		loggedInUser = vc.Email()
	}

	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "NewPack",
			"err", err,
			"user", loggedInUser,
			"took", time.Since(begin),
		)
	}(time.Now())

	pack, err = mw.Service.NewPack(ctx, p)
	return pack, err
}

func (mw loggingMiddleware) ModifyPack(ctx context.Context, id uint, p fleet.PackPayload) (*fleet.Pack, error) {
	var (
		pack         *fleet.Pack
		loggedInUser = "unauthenticated"
		err          error
	)

	if vc, ok := viewer.FromContext(ctx); ok {

		loggedInUser = vc.Email()
	}

	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "ModifyPack",
			"err", err,
			"user", loggedInUser,
			"took", time.Since(begin),
		)
	}(time.Now())

	pack, err = mw.Service.ModifyPack(ctx, id, p)
	return pack, err
}

func (mw loggingMiddleware) ListPacks(ctx context.Context, opt fleet.ListOptions) ([]*fleet.Pack, error) {
	var (
		packs []*fleet.Pack
		err   error
	)

	defer func(begin time.Time) {
		_ = mw.loggerDebug(err).Log(
			"method", "ListPacks",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	packs, err = mw.Service.ListPacks(ctx, opt)
	return packs, err
}

func (mw loggingMiddleware) GetPack(ctx context.Context, id uint) (*fleet.Pack, error) {
	var (
		pack *fleet.Pack
		err  error
	)

	defer func(begin time.Time) {
		_ = mw.loggerDebug(err).Log(
			"method", "GetPack",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	pack, err = mw.Service.GetPack(ctx, id)
	return pack, err
}

func (mw loggingMiddleware) DeletePack(ctx context.Context, name string) error {
	var (
		loggedInUser = "unauthenticated"
		err          error
	)

	if vc, ok := viewer.FromContext(ctx); ok {

		loggedInUser = vc.Email()
	}

	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "DeletePack",
			"err", err,
			"user", loggedInUser,
			"took", time.Since(begin),
		)
	}(time.Now())
	err = mw.Service.DeletePack(ctx, name)
	return err
}

func (mw loggingMiddleware) ListPacksForHost(ctx context.Context, hid uint) ([]*fleet.Pack, error) {
	var (
		packs []*fleet.Pack
		err   error
	)

	defer func(begin time.Time) {
		_ = mw.loggerDebug(err).Log(
			"method", "ListPacksForHost",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	packs, err = mw.Service.ListPacksForHost(ctx, hid)
	return packs, err
}

func (mw loggingMiddleware) GetPackSpec(ctx context.Context, name string) (spec *fleet.PackSpec, err error) {
	defer func(begin time.Time) {
		_ = mw.loggerDebug(err).Log(
			"method", "GetPackSpec",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	spec, err = mw.Service.GetPackSpec(ctx, name)
	return spec, err
}

func (mw loggingMiddleware) GetPackSpecs(ctx context.Context) (specs []*fleet.PackSpec, err error) {
	defer func(begin time.Time) {
		_ = mw.loggerDebug(err).Log(
			"method", "GetPackSpecs",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	specs, err = mw.Service.GetPackSpecs(ctx)
	return specs, err
}

func (mw loggingMiddleware) ApplyPackSpecs(ctx context.Context, specs []*fleet.PackSpec) (err error) {
	var (
		loggedInUser = "unauthenticated"
	)

	if vc, ok := viewer.FromContext(ctx); ok {

		loggedInUser = vc.Email()
	}

	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "ApplyPackSpecs",
			"err", err,
			"user", loggedInUser,
			"took", time.Since(begin),
		)
	}(time.Now())
	err = mw.Service.ApplyPackSpecs(ctx, specs)
	return err
}

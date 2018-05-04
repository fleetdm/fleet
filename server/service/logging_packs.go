package service

import (
	"context"
	"time"

	"github.com/kolide/fleet/server/kolide"
)

func (mw loggingMiddleware) ListPacks(ctx context.Context, opt kolide.ListOptions) ([]*kolide.Pack, error) {
	var (
		packs []*kolide.Pack
		err   error
	)

	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "ListPacks",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	packs, err = mw.Service.ListPacks(ctx, opt)
	return packs, err
}

func (mw loggingMiddleware) GetPack(ctx context.Context, id uint) (*kolide.Pack, error) {
	var (
		pack *kolide.Pack
		err  error
	)

	defer func(begin time.Time) {
		_ = mw.logger.Log(
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
		err error
	)

	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "DeletePack",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	err = mw.Service.DeletePack(ctx, name)
	return err
}

func (mw loggingMiddleware) ListLabelsForPack(ctx context.Context, pid uint) ([]*kolide.Label, error) {
	var (
		labels []*kolide.Label
		err    error
	)

	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "ListLabelsForPack",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	labels, err = mw.Service.ListLabelsForPack(ctx, pid)
	return labels, err
}

func (mw loggingMiddleware) ListPacksForHost(ctx context.Context, hid uint) ([]*kolide.Pack, error) {
	var (
		packs []*kolide.Pack
		err   error
	)

	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "ListPacksForHost",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	packs, err = mw.Service.ListPacksForHost(ctx, hid)
	return packs, err
}

func (mw loggingMiddleware) ListHostsInPack(ctx context.Context, pid uint, opt kolide.ListOptions) ([]uint, error) {
	var (
		hosts []uint
		err   error
	)

	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "ListHostsInPack",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	hosts, err = mw.Service.ListHostsInPack(ctx, pid, opt)
	return hosts, err
}

package service

import (
	"time"

	"github.com/kolide/kolide-ose/server/kolide"
	"golang.org/x/net/context"
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

func (mw loggingMiddleware) NewPack(ctx context.Context, p kolide.PackPayload) (*kolide.Pack, error) {
	var (
		pack *kolide.Pack
		err  error
	)

	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "NewPack",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	pack, err = mw.Service.NewPack(ctx, p)
	return pack, err
}

func (mw loggingMiddleware) ModifyPack(ctx context.Context, id uint, p kolide.PackPayload) (*kolide.Pack, error) {
	var (
		pack *kolide.Pack
		err  error
	)

	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "ModifyPack",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	pack, err = mw.Service.ModifyPack(ctx, id, p)
	return pack, err
}

func (mw loggingMiddleware) DeletePack(ctx context.Context, id uint) error {
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

	err = mw.Service.DeletePack(ctx, id)
	return err
}

func (mw loggingMiddleware) AddLabelToPack(ctx context.Context, lid uint, pid uint) error {
	var (
		err error
	)

	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "AddLabelToPack",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	err = mw.Service.AddLabelToPack(ctx, lid, pid)
	return err
}

func (mw loggingMiddleware) RemoveLabelFromPack(ctx context.Context, lid uint, pid uint) error {
	var (
		err error
	)

	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "RemoveLabelFromPack",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	err = mw.Service.RemoveLabelFromPack(ctx, lid, pid)
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

func (mw loggingMiddleware) AddHostToPack(ctx context.Context, hid uint, pid uint) error {
	var (
		err error
	)

	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "AddHostToPack",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	err = mw.Service.AddHostToPack(ctx, hid, pid)
	return err
}

func (mw loggingMiddleware) RemoveHostFromPack(ctx context.Context, hid uint, pid uint) error {
	var (
		err error
	)

	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "RemoveHostFromPack",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	err = mw.Service.RemoveHostFromPack(ctx, hid, pid)
	return err
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

func (mw loggingMiddleware) ListHostsInPack(ctx context.Context, pid uint, opt kolide.ListOptions) ([]*kolide.Host, error) {
	var (
		hosts []*kolide.Host
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

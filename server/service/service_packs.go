package service

import (
	"context"

	"github.com/kolide/fleet/server/kolide"
)

func (svc service) ApplyPackSpecs(ctx context.Context, specs []*kolide.PackSpec) error {
	return svc.ds.ApplyPackSpecs(specs)
}

func (svc service) GetPackSpecs(ctx context.Context) ([]*kolide.PackSpec, error) {
	return svc.ds.GetPackSpecs()
}

func (svc service) ListPacks(ctx context.Context, opt kolide.ListOptions) ([]*kolide.Pack, error) {
	return svc.ds.ListPacks(opt)
}

func (svc service) GetPack(ctx context.Context, id uint) (*kolide.Pack, error) {
	return svc.ds.Pack(id)
}

func (svc service) DeletePack(ctx context.Context, id uint) error {
	return svc.ds.DeletePack(id)
}

func (svc service) ListLabelsForPack(ctx context.Context, pid uint) ([]*kolide.Label, error) {
	return svc.ds.ListLabelsForPack(pid)
}

func (svc service) ListHostsInPack(ctx context.Context, pid uint, opt kolide.ListOptions) ([]uint, error) {
	return svc.ds.ListHostsInPack(pid, opt)
}

func (svc service) ListExplicitHostsInPack(ctx context.Context, pid uint, opt kolide.ListOptions) ([]uint, error) {
	return svc.ds.ListExplicitHostsInPack(pid, opt)
}

func (svc service) ListPacksForHost(ctx context.Context, hid uint) ([]*kolide.Pack, error) {
	return svc.ds.ListPacksForHost(hid)
}

package service

import (
	"github.com/kolide/kolide-ose/server/kolide"
	"golang.org/x/net/context"
)

func (svc service) GetAllPacks(ctx context.Context) ([]*kolide.Pack, error) {
	return svc.ds.Packs()
}

func (svc service) GetPack(ctx context.Context, id uint) (*kolide.Pack, error) {
	return svc.ds.Pack(id)
}

func (svc service) NewPack(ctx context.Context, p kolide.PackPayload) (*kolide.Pack, error) {
	var pack kolide.Pack

	if p.Name != nil {
		pack.Name = *p.Name
	}

	if p.Platform != nil {
		pack.Platform = *p.Platform
	}

	err := svc.ds.NewPack(&pack)
	if err != nil {
		return nil, err
	}
	return &pack, nil
}

func (svc service) ModifyPack(ctx context.Context, id uint, p kolide.PackPayload) (*kolide.Pack, error) {
	pack, err := svc.ds.Pack(id)
	if err != nil {
		return nil, err
	}

	if p.Name != nil {
		pack.Name = *p.Name
	}

	if p.Platform != nil {
		pack.Platform = *p.Platform
	}

	err = svc.ds.SavePack(pack)
	if err != nil {
		return nil, err
	}

	return pack, err
}

func (svc service) DeletePack(ctx context.Context, id uint) error {
	return svc.ds.DeletePack(id)
}

func (svc service) AddQueryToPack(ctx context.Context, qid, pid uint) error {
	return svc.ds.AddQueryToPack(qid, pid)
}

func (svc service) GetQueriesInPack(ctx context.Context, id uint) ([]*kolide.Query, error) {
	pack, err := svc.ds.Pack(id)
	if err != nil {
		return nil, err
	}

	queries, err := svc.ds.GetQueriesInPack(pack)
	if err != nil {
		return nil, err
	}

	return queries, nil
}

func (svc service) RemoveQueryFromPack(ctx context.Context, qid, pid uint) error {
	pack, err := svc.ds.Pack(pid)
	if err != nil {
		return err
	}

	query, err := svc.ds.Query(qid)
	if err != nil {
		return err
	}

	err = svc.ds.RemoveQueryFromPack(query, pack)
	if err != nil {
		return err
	}

	return nil
}
func (svc service) AddLabelToPack(ctx context.Context, lid, pid uint) error {
	return svc.ds.AddLabelToPack(lid, pid)
}

func (svc service) GetLabelsForPack(ctx context.Context, pid uint) ([]*kolide.Label, error) {
	pack, err := svc.ds.Pack(pid)
	if err != nil {
		return nil, err
	}

	labels, err := svc.ds.GetLabelsForPack(pack)
	if err != nil {
		return nil, err
	}

	return labels, nil
}

func (svc service) RemoveLabelFromPack(ctx context.Context, lid, pid uint) error {
	pack, err := svc.ds.Pack(pid)
	if err != nil {
		return err
	}

	label, err := svc.ds.Label(lid)
	if err != nil {
		return err
	}

	err = svc.ds.RemoveLabelFromPack(label, pack)
	if err != nil {
		return err
	}

	return nil
}

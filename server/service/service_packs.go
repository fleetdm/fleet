package service

import (
	"github.com/kolide/kolide-ose/server/kolide"
	"golang.org/x/net/context"
)

func (svc service) ListPacks(ctx context.Context, opt kolide.ListOptions) ([]*kolide.Pack, error) {
	return svc.ds.ListPacks(opt)
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

	_, err := svc.ds.NewPack(&pack)
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

func (svc service) ListQueriesInPack(ctx context.Context, id uint) ([]*kolide.Query, error) {
	pack, err := svc.ds.Pack(id)
	if err != nil {
		return nil, err
	}

	queries, err := svc.ds.ListQueriesInPack(pack)
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

func (svc service) ListLabelsForPack(ctx context.Context, pid uint) ([]*kolide.Label, error) {
	pack, err := svc.ds.Pack(pid)
	if err != nil {
		return nil, err
	}

	labels, err := svc.ds.ListLabelsForPack(pack)
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

func (svc service) ListPacksForHost(ctx context.Context, hid uint) ([]*kolide.Pack, error) {
	packs := []*kolide.Pack{}

	// we will need to give some subset of packs to this host based on the
	// labels which this host is known to belong to
	allPacks, err := svc.ds.ListPacks(kolide.ListOptions{})
	if err != nil {
		return nil, err
	}

	// pull the labels that this host belongs to
	labels, err := svc.ds.ListLabelsForHost(hid)
	if err != nil {
		return nil, err
	}

	// in order to use o(1) array indexing in an o(n) loop vs a o(n^2) double
	// for loop iteration, we must create the array which may be indexed below
	labelIDs := map[uint]bool{}
	for _, label := range labels {
		labelIDs[label.ID] = true
	}

	for _, pack := range allPacks {
		// for each pack, we must know what labels have been assigned to that
		// pack
		labelsForPack, err := svc.ds.ListLabelsForPack(pack)
		if err != nil {
			return nil, err
		}

		// o(n) iteration to determine whether or not a pack is enabled
		// in this case, n is len(labelsForPack)
		for _, label := range labelsForPack {
			if labelIDs[label.ID] {
				packs = append(packs, pack)
				break
			}
		}
	}

	return packs, nil
}

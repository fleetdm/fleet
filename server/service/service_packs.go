package service

import (
	"context"

	"github.com/fleetdm/fleet/server/kolide"
)

func (svc *Service) ApplyPackSpecs(ctx context.Context, specs []*kolide.PackSpec) error {
	if err := svc.authz.Authorize(ctx, &kolide.Pack{}, "write"); err != nil {
		return err
	}

	return svc.ds.ApplyPackSpecs(specs)
}

func (svc *Service) GetPackSpecs(ctx context.Context) ([]*kolide.PackSpec, error) {
	if err := svc.authz.Authorize(ctx, &kolide.Pack{}, "read"); err != nil {
		return nil, err
	}

	return svc.ds.GetPackSpecs()
}

func (svc *Service) GetPackSpec(ctx context.Context, name string) (*kolide.PackSpec, error) {
	if err := svc.authz.Authorize(ctx, &kolide.Pack{}, "read"); err != nil {
		return nil, err
	}

	return svc.ds.GetPackSpec(name)
}

func (svc *Service) ListPacks(ctx context.Context, opt kolide.ListOptions) ([]*kolide.Pack, error) {
	if err := svc.authz.Authorize(ctx, &kolide.Pack{}, "read"); err != nil {
		return nil, err
	}

	return svc.ds.ListPacks(opt)
}

func (svc *Service) GetPack(ctx context.Context, id uint) (*kolide.Pack, error) {
	if err := svc.authz.Authorize(ctx, &kolide.Pack{}, "read"); err != nil {
		return nil, err
	}

	return svc.ds.Pack(id)
}

func (svc *Service) NewPack(ctx context.Context, p kolide.PackPayload) (*kolide.Pack, error) {
	if err := svc.authz.Authorize(ctx, &kolide.Pack{}, "write"); err != nil {
		return nil, err
	}

	var pack kolide.Pack

	if p.Name != nil {
		pack.Name = *p.Name
	}

	if p.Description != nil {
		pack.Description = *p.Description
	}

	if p.Platform != nil {
		pack.Platform = *p.Platform
	}

	if p.Disabled != nil {
		pack.Disabled = *p.Disabled
	}

	_, err := svc.ds.NewPack(&pack)
	if err != nil {
		return nil, err
	}

	if p.HostIDs != nil {
		for _, hostID := range *p.HostIDs {
			err = svc.AddHostToPack(ctx, hostID, pack.ID)
			if err != nil {
				return nil, err
			}
		}
	}

	if p.LabelIDs != nil {
		for _, labelID := range *p.LabelIDs {
			err = svc.AddLabelToPack(ctx, labelID, pack.ID)
			if err != nil {
				return nil, err
			}
		}
	}

	return &pack, nil
}

func (svc *Service) ModifyPack(ctx context.Context, id uint, p kolide.PackPayload) (*kolide.Pack, error) {
	if err := svc.authz.Authorize(ctx, &kolide.Pack{}, "write"); err != nil {
		return nil, err
	}

	pack, err := svc.ds.Pack(id)
	if err != nil {
		return nil, err
	}

	if p.Name != nil {
		pack.Name = *p.Name
	}

	if p.Description != nil {
		pack.Description = *p.Description
	}

	if p.Platform != nil {
		pack.Platform = *p.Platform
	}

	if p.Disabled != nil {
		pack.Disabled = *p.Disabled
	}

	err = svc.ds.SavePack(pack)
	if err != nil {
		return nil, err
	}

	// we must determine what hosts are attached to this pack. then, given
	// our new set of host_ids, we will mutate the database to reflect the
	// desired state.
	if p.HostIDs != nil {

		// first, let's retrieve the total set of hosts
		hosts, err := svc.ListHostsInPack(ctx, pack.ID, kolide.ListOptions{})
		if err != nil {
			return nil, err
		}

		// it will be efficient to create a data structure with constant time
		// lookups to determine whether or not a host is already added
		existingHosts := map[uint]bool{}
		for _, host := range hosts {
			existingHosts[host] = true
		}

		// we will also make a constant time lookup map for the desired set of
		// hosts as well.
		desiredHosts := map[uint]bool{}
		for _, hostID := range *p.HostIDs {
			desiredHosts[hostID] = true
		}

		// if the request declares a host ID but the host is not already
		// associated with the pack, we add it
		for _, hostID := range *p.HostIDs {
			if !existingHosts[hostID] {
				err = svc.AddHostToPack(ctx, hostID, pack.ID)
				if err != nil {
					return nil, err
				}
			}
		}

		// if the request does not declare the ID of a host which currently
		// exists, we delete the existing relationship
		for hostID := range existingHosts {
			if !desiredHosts[hostID] {
				err = svc.RemoveHostFromPack(ctx, hostID, pack.ID)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	// we must determine what labels are attached to this pack. then, given
	// our new set of label_ids, we will mutate the database to reflect the
	// desired state.
	if p.LabelIDs != nil {

		// first, let's retrieve the total set of labels
		labels, err := svc.ListLabelsForPack(ctx, pack.ID)
		if err != nil {
			return nil, err
		}

		// it will be efficient to create a data structure with constant time
		// lookups to determine whether or not a label is already added
		existingLabels := map[uint]bool{}
		for _, label := range labels {
			existingLabels[label.ID] = true
		}

		// we will also make a constant time lookup map for the desired set of
		// labels as well.
		desiredLabels := map[uint]bool{}
		for _, labelID := range *p.LabelIDs {
			desiredLabels[labelID] = true
		}

		// if the request declares a label ID but the label is not already
		// associated with the pack, we add it
		for _, labelID := range *p.LabelIDs {
			if !existingLabels[labelID] {
				err = svc.AddLabelToPack(ctx, labelID, pack.ID)
				if err != nil {
					return nil, err
				}
			}
		}

		// if the request does not declare the ID of a label which currently
		// exists, we delete the existing relationship
		for labelID := range existingLabels {
			if !desiredLabels[labelID] {
				err = svc.RemoveLabelFromPack(ctx, labelID, pack.ID)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return pack, err
}

func (svc *Service) DeletePack(ctx context.Context, name string) error {
	if err := svc.authz.Authorize(ctx, &kolide.Pack{}, "write"); err != nil {
		return err
	}

	return svc.ds.DeletePack(name)
}

func (svc *Service) DeletePackByID(ctx context.Context, id uint) error {
	if err := svc.authz.Authorize(ctx, &kolide.Pack{}, "write"); err != nil {
		return err
	}

	pack, err := svc.ds.Pack(id)
	if err != nil {
		return err
	}
	return svc.ds.DeletePack(pack.Name)
}

func (svc *Service) AddLabelToPack(ctx context.Context, lid, pid uint) error {
	if err := svc.authz.Authorize(ctx, &kolide.Pack{}, "write"); err != nil {
		return err
	}

	return svc.ds.AddLabelToPack(lid, pid)
}

func (svc *Service) RemoveLabelFromPack(ctx context.Context, lid, pid uint) error {
	if err := svc.authz.Authorize(ctx, &kolide.Pack{}, "write"); err != nil {
		return err
	}

	return svc.ds.RemoveLabelFromPack(lid, pid)
}

func (svc *Service) AddHostToPack(ctx context.Context, hid, pid uint) error {
	if err := svc.authz.Authorize(ctx, &kolide.Pack{}, "write"); err != nil {
		return err
	}

	return svc.ds.AddHostToPack(hid, pid)
}

func (svc *Service) RemoveHostFromPack(ctx context.Context, hid, pid uint) error {
	if err := svc.authz.Authorize(ctx, &kolide.Pack{}, "write"); err != nil {
		return err
	}

	return svc.ds.RemoveHostFromPack(hid, pid)
}

func (svc *Service) ListLabelsForPack(ctx context.Context, pid uint) ([]*kolide.Label, error) {
	if err := svc.authz.Authorize(ctx, &kolide.Pack{}, "read"); err != nil {
		return nil, err
	}

	return svc.ds.ListLabelsForPack(pid)
}

func (svc *Service) ListHostsInPack(ctx context.Context, pid uint, opt kolide.ListOptions) ([]uint, error) {
	if err := svc.authz.Authorize(ctx, &kolide.Pack{}, "read"); err != nil {
		return nil, err
	}

	return svc.ds.ListHostsInPack(pid, opt)
}

func (svc *Service) ListExplicitHostsInPack(ctx context.Context, pid uint, opt kolide.ListOptions) ([]uint, error) {
	if err := svc.authz.Authorize(ctx, &kolide.Pack{}, "read"); err != nil {
		return nil, err
	}

	return svc.ds.ListExplicitHostsInPack(pid, opt)
}

func (svc *Service) ListPacksForHost(ctx context.Context, hid uint) ([]*kolide.Pack, error) {
	if err := svc.authz.Authorize(ctx, &kolide.Pack{}, "read"); err != nil {
		return nil, err
	}

	return svc.ds.ListPacksForHost(hid)
}

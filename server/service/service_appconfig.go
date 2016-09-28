package service

import (
	"github.com/kolide/kolide-ose/server/kolide"
	"golang.org/x/net/context"
)

func (svc service) NewOrgInfo(ctx context.Context, p kolide.OrgInfoPayload) (*kolide.OrgInfo, error) {
	info := &kolide.OrgInfo{}
	if p.OrgName != nil {
		info.OrgName = *p.OrgName
	}
	if p.OrgLogoURL != nil {
		info.OrgLogoURL = *p.OrgLogoURL
	}
	info, err := svc.ds.NewOrgInfo(info)
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (svc service) OrgInfo(ctx context.Context) (*kolide.OrgInfo, error) {
	return svc.ds.OrgInfo()
}

func (svc service) ModifyOrgInfo(ctx context.Context, p kolide.OrgInfoPayload) (*kolide.OrgInfo, error) {
	info, err := svc.ds.OrgInfo()
	if err != nil {
		return nil, err
	}

	if p.OrgName != nil {
		info.OrgName = *p.OrgName
	}
	if p.OrgLogoURL != nil {
		info.OrgLogoURL = *p.OrgLogoURL
	}

	err = svc.ds.SaveOrgInfo(info)
	if err != nil {
		return nil, err
	}
	return info, nil
}

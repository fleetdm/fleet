//go:build integration
// +build integration

package mysql

import (
	"context"
	"errors"
	"io/ioutil"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage"
)

type DeviceInterfaces interface {
	storage.CheckinStore
}

type Device struct {
	UDID string
}

func (d *Device) EnrollID() *mdm.EnrollID {
	return &mdm.EnrollID{Type: mdm.Device, ID: d.UDID}
}

func loadAuthMsg() (*mdm.Authenticate, Device, error) {
	var d Device
	b, err := ioutil.ReadFile("../../mdm/testdata/Authenticate.2.plist")
	if err != nil {
		return nil, d, err
	}
	r, err := mdm.DecodeCheckin(b)
	if err != nil {
		return nil, d, err
	}
	a, ok := r.(*mdm.Authenticate)
	if !ok {
		return nil, d, errors.New("not an Authenticate message")
	}
	d = Device{UDID: a.UDID}
	return a, d, nil
}

func loadTokenMsg() (*mdm.TokenUpdate, error) {
	b, err := ioutil.ReadFile("../../mdm/testdata/TokenUpdate.2.plist")
	if err != nil {
		return nil, err
	}
	r, err := mdm.DecodeCheckin(b)
	if err != nil {
		return nil, err
	}
	a, ok := r.(*mdm.TokenUpdate)
	if !ok {
		return nil, errors.New("not a TokenUpdate message")
	}
	return a, nil
}

func (d *Device) newMdmReq() *mdm.Request {
	return &mdm.Request{
		Context: context.Background(),
		EnrollID: &mdm.EnrollID{
			Type: mdm.Device,
			ID:   d.UDID,
		},
	}
}

func enrollTestDevice(storage DeviceInterfaces) (Device, error) {
	authMsg, d, err := loadAuthMsg()
	if err != nil {
		return d, err
	}
	err = storage.StoreAuthenticate(d.newMdmReq(), authMsg)
	if err != nil {
		return d, err
	}
	tokenMsg, err := loadTokenMsg()
	if err != nil {
		return d, err
	}
	err = storage.StoreTokenUpdate(d.newMdmReq(), tokenMsg)
	if err != nil {
		return d, err
	}
	return d, nil
}

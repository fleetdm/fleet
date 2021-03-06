// Automatically generated by mockimpl. DO NOT EDIT!

package mock

import "github.com/fleetdm/fleet/v4/server/fleet"

var _ fleet.SoftwareStore = (*SoftwareStore)(nil)

type SaveHostSoftwareFunc func(host *fleet.Host) error

type LoadHostSoftwareFunc func(host *fleet.Host) error

type SoftwareStore struct {
	SaveHostSoftwareFunc        SaveHostSoftwareFunc
	SaveHostSoftwareFuncInvoked bool

	LoadHostSoftwareFunc        LoadHostSoftwareFunc
	LoadHostSoftwareFuncInvoked bool
}

func (s *SoftwareStore) SaveHostSoftware(host *fleet.Host) error {
	s.SaveHostSoftwareFuncInvoked = true
	return s.SaveHostSoftwareFunc(host)
}

func (s *SoftwareStore) LoadHostSoftware(host *fleet.Host) error {
	s.LoadHostSoftwareFuncInvoked = true
	return s.LoadHostSoftwareFunc(host)
}

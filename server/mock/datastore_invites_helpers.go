package mock

import "github.com/fleetdm/fleet/server/kolide"

func ReturnNewInivite(fake *kolide.Invite) NewInviteFunc {
	return func(i *kolide.Invite) (*kolide.Invite, error) {
		return fake, nil
	}
}

func ReturnFakeInviteByEmail(fake *kolide.Invite) InviteByEmailFunc {
	return func(string) (*kolide.Invite, error) {
		return fake, nil
	}
}

func ReturnFakeInviteByToken(fake *kolide.Invite) InviteByTokenFunc {
	return func(string) (*kolide.Invite, error) {
		return fake, nil
	}
}

func ReturnInviteFuncNotFound() InviteFunc {
	return func(id uint) (*kolide.Invite, error) {
		return nil, &Error{"not found"}
	}
}

func ReturnFakeInviteByID(fake *kolide.Invite) InviteFunc {
	return func(id uint) (*kolide.Invite, error) {
		return fake, nil
	}
}

package inmem

import (
	"github.com/kolide/kolide/server/kolide"
)

func (d *Datastore) NewIdentityProvider(idp kolide.IdentityProvider) (*kolide.IdentityProvider, error) {
	panic("inmem is being deprecated")
}

func (d *Datastore) SaveIdentityProvider(idb kolide.IdentityProvider) error {
	panic("inmem is being deprecated")
}

func (d *Datastore) IdentityProvider(id uint) (*kolide.IdentityProvider, error) {
	panic("inmem is being deprecated")
}

func (d *Datastore) DeleteIdentityProvider(id uint) error {
	panic("inmem is being deprecated")
}

func (d *Datastore) ListIdentityProviders() ([]kolide.IdentityProvider, error) {
	panic("inmem is being deprecated")
}

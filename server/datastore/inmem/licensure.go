package inmem

import "github.com/kolide/kolide/server/kolide"

func (ds *Datastore) SaveLicense(token, key string) (*kolide.License, error) {
	panic("inmem is being deprecated")
}

func (ds *Datastore) License() (*kolide.License, error) {
	panic("inmem is being deprecated")
}

func (ds *Datastore) LicensePublicKey(string) (string, error) {
	panic("inmem is being deprecated")
}

func (ds *Datastore) RevokeLicense(revoked bool) error {
	panic("inmem is being deprecated")
}

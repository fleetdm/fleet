package allmulti

import (
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage"
)

func (ms *MultiAllStorage) HasCertHash(r *mdm.Request, hash string) (bool, error) {
	val, err := ms.execStores(r.Context, func(s storage.AllStorage) (interface{}, error) {
		return s.HasCertHash(r, hash)
	})
	return val.(bool), err
}

func (ms *MultiAllStorage) EnrollmentHasCertHash(r *mdm.Request, hash string) (bool, error) {
	val, err := ms.execStores(r.Context, func(s storage.AllStorage) (interface{}, error) {
		return s.EnrollmentHasCertHash(r, hash)
	})
	return val.(bool), err
}

func (ms *MultiAllStorage) IsCertHashAssociated(r *mdm.Request, hash string) (bool, error) {
	val, err := ms.execStores(r.Context, func(s storage.AllStorage) (interface{}, error) {
		return s.IsCertHashAssociated(r, hash)
	})
	return val.(bool), err
}

func (ms *MultiAllStorage) AssociateCertHash(r *mdm.Request, hash string, certNotValidAfter time.Time) error {
	_, err := ms.execStores(r.Context, func(s storage.AllStorage) (interface{}, error) {
		return nil, s.AssociateCertHash(r, hash, certNotValidAfter)
	})
	return err
}

package allmulti

import (
	"context"
	"crypto/tls"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage"
)

func (ms *MultiAllStorage) IsPushCertStale(ctx context.Context, topic string, staleToken string) (bool, error) {
	val, err := ms.execStores(ctx, func(s storage.AllStorage) (interface{}, error) {
		return s.IsPushCertStale(ctx, topic, staleToken)
	})
	return val.(bool), err
}

type retrievePushCertReturns struct {
	cert       *tls.Certificate
	staleToken string
}

func (ms *MultiAllStorage) RetrievePushCert(ctx context.Context, topic string) (cert *tls.Certificate, staleToken string, err error) {
	val, err := ms.execStores(ctx, func(s storage.AllStorage) (interface{}, error) {
		rets := new(retrievePushCertReturns)
		var err error
		rets.cert, rets.staleToken, err = s.RetrievePushCert(ctx, topic)
		return rets, err
	})
	rets := val.(*retrievePushCertReturns)
	return rets.cert, rets.staleToken, err
}

func (ms *MultiAllStorage) StorePushCert(ctx context.Context, pemCert, pemKey []byte) error {
	_, err := ms.execStores(ctx, func(s storage.AllStorage) (interface{}, error) {
		return nil, s.StorePushCert(ctx, pemCert, pemKey)
	})
	return err
}

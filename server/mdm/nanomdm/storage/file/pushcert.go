package file

import (
	"context"
	"crypto/tls"
	"errors"
	"io/ioutil"
	"os"
	"path"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/cryptoutil"
)

// RetrievePushCert is passed through to a new PushCertFileStorage
func (s *FileStorage) RetrievePushCert(ctx context.Context, topic string) (*tls.Certificate, string, error) {
	ps := &PushCertFileStorage{
		certFilepath: path.Join(s.path, topic+".pem"),
		keyFilepath:  path.Join(s.path, topic+".key"),
	}
	return ps.RetrievePushCert(ctx, topic)
}

// IsPushCertStale is passed through to a new PushCertFileStorage
func (s *FileStorage) IsPushCertStale(ctx context.Context, topic, providedStaleToken string) (bool, error) {
	ps := &PushCertFileStorage{
		certFilepath: path.Join(s.path, topic+".pem"),
	}
	return ps.IsPushCertStale(ctx, topic, providedStaleToken)
}

// StorePushCert is passed through to a new PushCertFileStorage
func (s *FileStorage) StorePushCert(ctx context.Context, pemCert, pemKey []byte) error {
	topic, err := cryptoutil.TopicFromPEMCert(pemCert)
	if err != nil {
		return err
	}
	ps := &PushCertFileStorage{
		certFilepath: path.Join(s.path, topic+".pem"),
		keyFilepath:  path.Join(s.path, topic+".key"),
		allowStore:   true,
	}
	return ps.StorePushCert(ctx, pemCert, pemKey)
}

// PushCertFileStorage is a filesystem-based PushCertStore
type PushCertFileStorage struct {
	certFilepath string
	keyFilepath  string
	allowStore   bool
}

func NewPushCertFileStorage(certPath, keyPath string) *PushCertFileStorage {
	return &PushCertFileStorage{certFilepath: certPath, keyFilepath: keyPath}
}

func (s *PushCertFileStorage) getPushCertStaleToken(filename string) (string, error) {
	info, err := os.Stat(filename)
	if err != nil {
		return "", err
	}
	return info.ModTime().String(), nil
}

// RetrievePushCert reads the Push Certificate from disk
func (s *PushCertFileStorage) RetrievePushCert(_ context.Context, topic string) (*tls.Certificate, string, error) {
	pemCert, err := ioutil.ReadFile(s.certFilepath)
	if err != nil {
		return nil, "", err
	}
	certTopic, err := cryptoutil.TopicFromPEMCert(pemCert)
	if err != nil {
		return nil, "", err
	}
	if certTopic != topic {
		return nil, "", errors.New("certificate topic mismatch")
	}
	pemKey, err := ioutil.ReadFile(s.keyFilepath)
	if err != nil {
		return nil, "", err
	}
	cert, err := tls.X509KeyPair(pemCert, pemKey)
	if err != nil {
		return nil, "", err
	}
	staleToken, err := s.getPushCertStaleToken(s.certFilepath)
	return &cert, staleToken, err
}

// IsPushCertStale inspects staleToken to tell if our push certs are stale
func (s *PushCertFileStorage) IsPushCertStale(_ context.Context, topic, providedStaleToken string) (bool, error) {
	staleToken, err := s.getPushCertStaleToken(s.certFilepath)
	if err != nil {
		return true, err
	}
	return providedStaleToken != staleToken, nil
}

// StorePushCert writes the push cert to disk
func (s *PushCertFileStorage) StorePushCert(_ context.Context, pemCert, pemKey []byte) error {
	if !s.allowStore {
		return errors.New("store push cert: not permitted")
	}
	err := ioutil.WriteFile(s.certFilepath, pemCert, 0644)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(s.keyFilepath, pemKey, 0600)
}

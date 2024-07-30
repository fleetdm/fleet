package file

import (
	"errors"
	"os"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
)

func (s *FileStorage) StoreBootstrapToken(r *mdm.Request, msg *mdm.SetBootstrapToken) error {
	e := s.newEnrollment(r.ID)
	if len(msg.BootstrapToken.BootstrapToken) > 0 {
		return e.writeFile(BootstrapTokenFile, msg.BootstrapToken.BootstrapToken)
	}
	if err := os.Remove(e.dirPrefix(BootstrapTokenFile)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (s *FileStorage) RetrieveBootstrapToken(r *mdm.Request, _ *mdm.GetBootstrapToken) (*mdm.BootstrapToken, error) {
	e := s.newEnrollment(r.ID)
	bsTokenRaw, err := e.readFile(BootstrapTokenFile)
	if err != nil {
		return nil, err
	}
	bsToken := &mdm.BootstrapToken{
		BootstrapToken: bsTokenRaw,
	}
	return bsToken, nil
}

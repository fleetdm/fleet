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

// RetrieveBootstrapToken reads the BootstrapToken from disk and returns it.
// If no token yet exists a nil token and no error are returned.
func (s *FileStorage) RetrieveBootstrapToken(r *mdm.Request, _ *mdm.GetBootstrapToken) (*mdm.BootstrapToken, error) {
	e := s.newEnrollment(r.ID)
	bsTokenRaw, err := e.readFile(BootstrapTokenFile)
	if errors.Is(err, os.ErrNotExist) {
		// mute the error if we haven't escrowed a token yet.
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	bsToken := &mdm.BootstrapToken{
		BootstrapToken: bsTokenRaw,
	}
	return bsToken, nil
}

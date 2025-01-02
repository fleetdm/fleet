// Package file implements filesystem-based storage for MDM services
package file

import (
	"context"
	"errors"
	"io/ioutil"
	"os"
	"path"
	"strconv"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/cryptoutil"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
)

const (
	AuthenticateFilename = "Authenticate.plist"
	TokenUpdateFilename  = "TokenUpdate.plist"
	UnlockTokenFilename  = "UnlockToken.dat"
	SerialNumberFilename = "SerialNumber.txt"
	IdentityCertFilename = "Identity.pem"
	DisabledFilename     = "Disabled"
	BootstrapTokenFile   = "BootstrapToken.dat"

	TokenUpdateTallyFilename = "TokenUpdate.tally.txt"

	UserAuthFilename       = "UserAuthenticate.plist"
	UserAuthDigestFilename = "UserAuthenticate.Digest.plist"

	CertAuthFilename             = "CertAuth.sha256.txt"
	CertAuthAssociationsFilename = "CertAuth.txt"

	// The associations for "sub"-enrollments (that is: user-channel
	// enrollments to device-channel enrollments) are stored in this
	// directory under the device's directory.
	SubEnrollmentPathname = "SubEnrollments"
)

// FileStorage implements filesystem-based storage for MDM services
type FileStorage struct {
	path string
}

// New creates a new FileStorage backend
func New(path string) (*FileStorage, error) {
	err := os.Mkdir(path, 0755)
	if err != nil && !errors.Is(err, os.ErrExist) {
		return nil, err
	}
	return &FileStorage{path: path}, nil
}

type enrollment struct {
	id string
	fs *FileStorage
}

func (s *FileStorage) newEnrollment(id string) *enrollment {
	return &enrollment{fs: s, id: id}
}

func (e *enrollment) dir() string {
	return path.Join(e.fs.path, e.id)
}

func (e *enrollment) mkdir() error {
	return os.MkdirAll(e.dir(), 0755)
}

func (e *enrollment) dirPrefix(name string) string {
	return path.Join(e.dir(), name)
}

func (e *enrollment) writeFile(name string, bytes []byte) error {
	if name == "" {
		return errors.New("write: empty name")
	}
	if err := e.mkdir(); err != nil {
		return err
	}
	return ioutil.WriteFile(e.dirPrefix(name), bytes, 0644)
}

func (e *enrollment) readFile(name string) ([]byte, error) {
	if name == "" {
		return nil, errors.New("write: empty name")
	}
	return ioutil.ReadFile(e.dirPrefix(name))
}

func (e *enrollment) fileExists(name string) (bool, error) {
	if _, err := os.Stat(e.dirPrefix(name)); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (e *enrollment) bumpNumericFile(name string) error {
	ctr, err := e.readNumericFile(name)
	if err != nil {
		return err
	}
	ctr += 1
	return e.writeFile(name, []byte(strconv.Itoa(ctr)))
}

func (e *enrollment) resetNumericFile(name string) error {
	return e.writeFile(name, []byte{48})
}

func (e *enrollment) readNumericFile(name string) (int, error) {
	val, err := e.readFile(name)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return 0, err
	}
	ctr, _ := strconv.Atoi(string(val))
	return ctr, nil
}

// assocSubEnrollment writes an empty file of the sub (user) enrollment for tracking.
func (e *enrollment) assocSubEnrollment(id string) error {
	subPath := e.dirPrefix(SubEnrollmentPathname)
	if err := os.MkdirAll(subPath, 0755); err != nil {
		return err
	}
	f, err := os.Create(path.Join(subPath, id))
	if err != nil {
		return err
	}
	return f.Close()
}

// listSubEnrollments returns an array of the sub-enrollment IDs
func (e *enrollment) listSubEnrollments() (ids []string) {
	entries, err := os.ReadDir(e.dirPrefix(SubEnrollmentPathname))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil
	}
	for _, entry := range entries {
		if len(entry.Name()) > 10 {
			ids = append(ids, entry.Name())
		}
	}
	return
}

func (e *enrollment) removeSubEnrollments() error {
	return os.RemoveAll(e.dirPrefix(SubEnrollmentPathname))
}

// StoreAuthenticate stores the Authenticate message
func (s *FileStorage) StoreAuthenticate(r *mdm.Request, msg *mdm.Authenticate) error {
	e := s.newEnrollment(r.ID)
	if r.Certificate != nil {
		if err := e.writeFile(IdentityCertFilename, cryptoutil.PEMCertificate(r.Certificate.Raw)); err != nil {
			return err
		}
	}
	// A nice-to-have even though it's duplicated in msg
	if msg.SerialNumber != "" {
		err := e.writeFile(SerialNumberFilename, []byte(msg.SerialNumber))
		if err != nil {
			return err
		}
	}
	if err := e.resetNumericFile(TokenUpdateTallyFilename); err != nil {
		return err
	}
	// remove the BootstrapToken when we receive an Authenticate message
	// BS tokens are only valid when a new one is escrowed after enrollment.
	if err := os.Remove(e.dirPrefix(BootstrapTokenFile)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return e.writeFile(AuthenticateFilename, msg.Raw)
}

// StoreTokenUpdate stores the TokenUpdate message
func (s *FileStorage) StoreTokenUpdate(r *mdm.Request, msg *mdm.TokenUpdate) error {
	e := s.newEnrollment(r.ID)
	// the UnlockToken should be saved separately in case future
	// TokenUpdates do not contain it and it gets overwritten
	if len(msg.UnlockToken) > 0 {
		if err := e.writeFile(UnlockTokenFilename, msg.UnlockToken); err != nil {
			return err
		}
	}
	if r.ParentID != "" {
		parentEnrollment := s.newEnrollment(r.ParentID)
		if err := parentEnrollment.assocSubEnrollment(r.ID); err != nil {
			return err
		}
	}
	if err := e.writeFile(TokenUpdateFilename, msg.Raw); err != nil {
		return err
	}
	if err := e.bumpNumericFile(TokenUpdateTallyFilename); err != nil {
		return err
	}
	// delete the disabled flag to let signify this enrollment is enabled
	if err := os.Remove(e.dirPrefix(DisabledFilename)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (s *FileStorage) RetrieveTokenUpdateTally(_ context.Context, id string) (int, error) {
	e := s.newEnrollment(id)
	return e.readNumericFile(TokenUpdateTallyFilename)
}

func (s *FileStorage) StoreUserAuthenticate(r *mdm.Request, msg *mdm.UserAuthenticate) error {
	e := s.newEnrollment(r.ID)
	filename := UserAuthFilename
	// if the DigestResponse is empty then this is the first (of two)
	// UserAuthenticate messages depending on our response
	if msg.DigestResponse != "" {
		filename = UserAuthDigestFilename
	}
	return e.writeFile(filename, msg.Raw)
}

func (s *FileStorage) Disable(r *mdm.Request) error {
	if r.ParentID != "" {
		return errors.New("can only disable a device channel")
	}
	// assemble list of IDs for which to disable
	e := s.newEnrollment(r.ID)
	disableIDs := e.listSubEnrollments()
	disableIDs = append(disableIDs, r.ID)
	// disable all of the ids
	for _, id := range disableIDs {
		e := s.newEnrollment(id)
		// write zero-byte disabled marker
		if err := e.writeFile(DisabledFilename, nil); err != nil {
			return err
		}
	}
	return e.removeSubEnrollments()
}

func (s *FileStorage) ExpandEmbeddedSecrets(_ context.Context, document string) (string, error) {
	// NOT IMPLEMENTED
	return document, nil
}

func (m *FileStorage) BulkDeleteHostUserCommandsWithoutResults(_ context.Context, _ map[string][]string) error {
	// NOT IMPLEMENTED
	return nil
}

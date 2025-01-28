package file

import (
	"bufio"
	"context"
	"errors"
	"os"
	"path"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
)

func (s *FileStorage) EnrollmentHasCertHash(r *mdm.Request, _ string) (bool, error) {
	e := s.newEnrollment(r.ID)
	_, err := e.readFile(CertAuthFilename)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, err
}

func (s *FileStorage) HasCertHash(r *mdm.Request, hash string) (bool, error) {
	f, err := os.Open(path.Join(s.path, CertAuthAssociationsFilename))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), hash) {
			return true, nil
		}
	}
	return false, scanner.Err()
}

func (s *FileStorage) IsCertHashAssociated(r *mdm.Request, hash string) (bool, error) {
	e := s.newEnrollment(r.ID)
	b, err := e.readFile(CertAuthFilename)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return strings.EqualFold(string(b), hash), nil
}

func (s *FileStorage) AssociateCertHash(r *mdm.Request, hash string, _ time.Time) error {
	f, err := os.OpenFile(
		path.Join(s.path, CertAuthAssociationsFilename),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString(r.ID + "," + hash + "\n"); err != nil {
		return err
	}
	e := s.newEnrollment(r.ID)
	return e.writeFile(CertAuthFilename, []byte(hash))
}

func (s *FileStorage) EnrollmentFromHash(_ context.Context, hash string) (string, error) {
	f, err := os.Open(path.Join(s.path, CertAuthAssociationsFilename))
	if err != nil {
		return "", err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		text := scanner.Text()
		if strings.Contains(text, hash) {
			split := strings.Split(text, ",")
			if len(split) < 2 {
				return "", errors.New("hash and enrollment id not present on line")
			}
			return split[0], nil
		}
	}
	return "", nil
}

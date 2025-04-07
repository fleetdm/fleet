package certauth

import (
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
)

func (s *CertAuth) Authenticate(r *mdm.Request, m *mdm.Authenticate) error {
	if err := s.associateForNewEnrollment(r, &m.Enrollment); err != nil {
		return err
	}
	return s.next.Authenticate(r, m)
}

func (s *CertAuth) TokenUpdate(r *mdm.Request, m *mdm.TokenUpdate) error {
	if err := s.validateOrAssociateForExistingEnrollment(r, &m.Enrollment); err != nil {
		return err
	}
	return s.next.TokenUpdate(r, m)
}

func (s *CertAuth) CheckOut(r *mdm.Request, m *mdm.CheckOut) error {
	if err := s.validateOrAssociateForExistingEnrollment(r, &m.Enrollment); err != nil {
		return err
	}
	return s.next.CheckOut(r, m)
}

func (s *CertAuth) UserAuthenticate(r *mdm.Request, m *mdm.UserAuthenticate) ([]byte, error) {
	if err := s.validateOrAssociateForExistingEnrollment(r, &m.Enrollment); err != nil {
		return nil, err
	}
	return s.next.UserAuthenticate(r, m)
}

func (s *CertAuth) SetBootstrapToken(r *mdm.Request, m *mdm.SetBootstrapToken) error {
	if err := s.validateOrAssociateForExistingEnrollment(r, &m.Enrollment); err != nil {
		return err
	}
	return s.next.SetBootstrapToken(r, m)
}

func (s *CertAuth) GetBootstrapToken(r *mdm.Request, m *mdm.GetBootstrapToken) (*mdm.BootstrapToken, error) {
	if err := s.validateOrAssociateForExistingEnrollment(r, &m.Enrollment); err != nil {
		return nil, err
	}
	return s.next.GetBootstrapToken(r, m)
}

func (s *CertAuth) DeclarativeManagement(r *mdm.Request, m *mdm.DeclarativeManagement) ([]byte, error) {
	if err := s.validateOrAssociateForExistingEnrollment(r, &m.Enrollment); err != nil {
		return nil, err
	}
	return s.next.DeclarativeManagement(r, m)
}

func (s *CertAuth) GetToken(r *mdm.Request, m *mdm.GetToken) (*mdm.GetTokenResponse, error) {
	if err := s.validateOrAssociateForExistingEnrollment(r, &m.Enrollment); err != nil {
		return nil, err
	}
	return s.next.GetToken(r, m)
}

func (s *CertAuth) CommandAndReportResults(r *mdm.Request, results *mdm.CommandResults) (*mdm.Command, error) {
	if err := s.validateOrAssociateForExistingEnrollment(r, &results.Enrollment); err != nil {
		return nil, err
	}
	return s.next.CommandAndReportResults(r, results)
}

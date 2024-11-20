// Package certauth
package certauth

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage"

	"github.com/micromdm/nanolib/log"
	"github.com/micromdm/nanolib/log/ctxlog"
)

var (
	ErrNoCertReuse = errors.New("cert re-use not permitted")
	ErrNoCertAssoc = errors.New("enrollment not associated with cert")
	ErrMissingCert = errors.New("missing MDM certificate")
)

// normalize pulls out only the "device" ID (i.e. the "parent" of the)
// MDM relationship regardless of enrollment type.
func normalize(e *mdm.Enrollment) *mdm.EnrollID {
	r := e.Resolved()
	if r == nil {
		return nil
	}
	return &mdm.EnrollID{
		ID:   r.DeviceChannelID,
		Type: r.Type,
	}
}

type CertAuth struct {
	next       service.CheckinAndCommandService
	logger     log.Logger
	normalizer func(e *mdm.Enrollment) *mdm.EnrollID
	storage    storage.CertAuthStore

	// allowDup potentially allows duplicate certificates to be used
	// for more than one enrollment. This may be permissible if, say,
	// a shared embedded identity is used in the enrollment profile.
	// otherwise, for SCEP, this should never happen: every enrollment
	// should be a uniquely issued certificate.
	allowDup bool

	// allowRetroactive allows cert-hash associations to happen in
	// requests other than the Authenticate Check-in message. If this is
	// true then you can effectively add cert-hash associations to
	// existing enrollments and not just new enrollments. However,
	// if the enrollment has an existing association we explicitly
	// disallow re-association.
	allowRetroactive bool

	// warnOnly won't return an error when we find a cert auth problem
	// and we won't save associations for existing enrollments.
	// This can be used to troubleshoot or remediate enrollments that
	// are having problems with associations by e.g. sending them a
	// enrollment profile via the MDM channel.
	//
	// WARNING: This allows MDM clients to spoof other MDM clients.
	warnOnly bool
}

type Option func(*CertAuth)

func WithLogger(logger log.Logger) Option {
	return func(certAuth *CertAuth) {
		certAuth.logger = logger
	}
}

func WithAllowRetroactive() Option {
	return func(certAuth *CertAuth) {
		certAuth.allowRetroactive = true
	}
}

// New creates a new certificate authorization middleware service. It
// will forward requests to next or return errors for failing authentication.
func New(next service.CheckinAndCommandService, storage storage.CertAuthStore, opts ...Option) *CertAuth {
	certAuth := &CertAuth{
		next:       next,
		logger:     log.NopLogger,
		normalizer: normalize,
		storage:    storage,
	}
	for _, opt := range opts {
		opt(certAuth)
	}
	if certAuth.allowRetroactive {
		certAuth.logger.Info("msg", "allowing retroactive associations")
	}
	return certAuth
}

// HashCert returns the string representation
func HashCert(cert *x509.Certificate) string {
	hashed := sha256.Sum256(cert.Raw)
	b := make([]byte, len(hashed))
	copy(b, hashed[:])
	return hex.EncodeToString(b)
}

func (s *CertAuth) associateNewEnrollment(r *mdm.Request) error {
	if r.Certificate == nil {
		return ErrMissingCert
	}
	if err := r.EnrollID.Validate(); err != nil {
		return err
	}
	logger := ctxlog.Logger(r.Context, s.logger)
	hash := HashCert(r.Certificate)
	if hasHash, err := s.storage.HasCertHash(r, hash); err != nil {
		return err
	} else if hasHash {
		if !s.allowDup {
			// test to see if we're using the same cert for an
			// enrollment. the only way this should happen is if
			// the cert is embedded in the profile and they're re-using
			// the cert. permit this one case.
			if isAssoc, err := s.storage.IsCertHashAssociated(r, hash); err != nil {
				return err
			} else if isAssoc {
				return nil
			}
			logger.Info(
				"msg", "cert hash exists",
				"enrollment", "new",
				"id", r.ID,
				"hash", hash,
			)
			if !s.warnOnly {
				return ErrNoCertReuse
			}
		}
	}
	if err := s.storage.AssociateCertHash(r, hash, r.Certificate.NotAfter); err != nil {
		return err
	}
	logger.Info(
		"msg", "cert associated",
		"enrollment", "new",
		"id", r.ID,
		"hash", hash,
	)
	return nil
}

func (s *CertAuth) validateAssociateExistingEnrollment(r *mdm.Request) error {
	if r.Certificate == nil {
		return ErrMissingCert
	}
	if err := r.EnrollID.Validate(); err != nil {
		return err
	}
	logger := ctxlog.Logger(r.Context, s.logger)
	hash := HashCert(r.Certificate)
	if isAssoc, err := s.storage.IsCertHashAssociated(r, hash); err != nil {
		return err
	} else if isAssoc {
		return nil
	}
	if !s.allowRetroactive {
		logger.Info(
			"msg", "no cert association",
			"enrollment", "existing",
			"id", r.ID,
			"hash", hash,
		)
		if !s.warnOnly {
			return ErrNoCertAssoc
		}
	}
	// even if allowRetroactive is true we don't want to allow arbitrary
	// existing enrollments to use a different association. you must
	// MDM re-Authenticate first. so we check that this enrollment
	// has no association yet.
	if hasHash, err := s.storage.EnrollmentHasCertHash(r, hash); err != nil {
		return err
	} else if hasHash {
		logger.Info(
			"msg", "enrollment cannot have associated cert hash",
			"enrollment", "existing",
			"id", r.ID,
		)
		if !s.warnOnly {
			return ErrNoCertReuse
		}
	}
	// even if allowDup were true we don't want to allow arbitrary
	// existing enrollments to use another association. you must
	// MDM re-Authenticate first. so we check that this cert hasn't
	// been seen before to prevent any possible exfiltrated cert
	// connections.
	if hasHash, err := s.storage.HasCertHash(r, hash); err != nil {
		return err
	} else if hasHash {
		logger.Info(
			"msg", "cert hash exists",
			"enrollment", "existing",
			"id", r.ID,
			"hash", hash,
		)
		if !s.warnOnly {
			return ErrNoCertReuse
		}
	}
	if s.warnOnly {
		return nil
	}
	if err := s.storage.AssociateCertHash(r, hash, r.Certificate.NotAfter); err != nil {
		return err
	}
	logger.Info(
		"msg", "cert associated",
		"enrollment", "existing",
		"id", r.ID,
		"hash", hash,
	)
	return nil
}

func (s *CertAuth) associateForNewEnrollment(r *mdm.Request, e *mdm.Enrollment) error {
	req := r.Clone()
	req.EnrollID = s.normalizer(e)
	if err := s.associateNewEnrollment(req); err != nil {
		return fmt.Errorf("cert auth: new enrollment: %w", err)
	}
	return nil
}

func (s *CertAuth) validateOrAssociateForExistingEnrollment(r *mdm.Request, e *mdm.Enrollment) error {
	req := r.Clone()
	req.EnrollID = s.normalizer(e)
	if err := s.validateAssociateExistingEnrollment(req); err != nil {
		return fmt.Errorf("cert auth: existing enrollment: %w", err)
	}
	return nil
}

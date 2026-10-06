package challenge

import (
	"context"
	"crypto/subtle"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"log/slog"
	"slices"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
	"github.com/smallstep/scep"
)

// AppleMDMSCEPStore is an interface for managing SCEP challenges in the Apple MDM context.
// Supports Static challenge retrieval and dynamic challenge consumption
type AppleMDMSCEPStore interface {
	ConsumeAppleSCEPChallenge(ctx context.Context, challenge string) (*fleet.AppleSCEPChallengeInfo, error)
	SetAppleSCEPChallengeIssuedCert(ctx context.Context, challenge string, certSerial int64) error
	fleet.MDMAssetRetriever
}

type appleMDMChallengeSigner interface {
	SignX509CSRWithCallback(csr *x509.CertificateRequest, subject pkix.Name, callback func(tmpl *x509.Certificate)) (*x509.Certificate, error)
}

// AppleMDMChallengeMiddleware is a middleware that takes a custom signer that allows setting extensions.
// It supports both static and dynamic challenges. Static challenges are retrieved from MDM assets, while dynamic challenges are consumed from the store.
func AppleMDMChallengeMiddleware(logger *slog.Logger, store AppleMDMSCEPStore, staticChallengeEnabled bool, next appleMDMChallengeSigner) scepserver.CSRSignerContextFunc {
	return func(ctx context.Context, m *scep.CSRReqMessage) (*x509.Certificate, error) {
		// Check the OU for the newEnrollment identifier
		hasNewEnrollmentOU := slices.Contains(m.CSR.Subject.OrganizationalUnit, apple_mdm.FleetEnrollmentSubjectOU)

		if staticChallengeEnabled {
			assets, err := store.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetSCEPChallenge}, nil)
			if err == nil {
				// no error, proceed with static challenge check
				staticSCEPChallenge := string(assets[fleet.MDMAssetSCEPChallenge].Value)
				if subtle.ConstantTimeCompare([]byte(m.ChallengePassword), []byte(staticSCEPChallenge)) == 1 {
					// pass on match, on failure fall through to dynamic challenge check
					// do not sign certificate with information since we don't have it for static challenge
					return next.SignX509CSRWithCallback(m.CSR, apple_mdm.AppleMDMSCEPCertificateSubject(hasNewEnrollmentOU), nil)
				}
			} else {
				logger.ErrorContext(ctx, "failed to get SCEP challenge assets", "error", err)
			}

			// falling through to always do dynamic check if datastore errors
		}

		// attempt the challenge as a dynamic challenge
		dynamicChallengeInfo, err := store.ConsumeAppleSCEPChallenge(ctx, m.ChallengePassword)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "failed to consume dynamic SCEP challenge")
		}

		// if we fail after consuming the challenge we are okay with that,
		// it will require restarting the enrollment process

		// We build the case here, so each caller can do it their way.
		extension := apple_mdm.AppleMDMCertificateBindingExtension{Purpose: dynamicChallengeInfo.Purpose}
		switch dynamicChallengeInfo.Purpose {
		case fleet.AppleMDMCertPurposeADE, fleet.AppleMDMCertPurposeOTAPhaseOne, fleet.AppleMDMCertPurposeOTAPhaseTwo:
			extension.UDID = dynamicChallengeInfo.UUID
			extension.Serial = dynamicChallengeInfo.Serial
		case fleet.AppleMDMCertPurposeADUE:
			extension.IDPAccountUUID = dynamicChallengeInfo.IDPAccountUUID
		case fleet.AppleMDMCertPurposeSCEPRenewal:
			extension.EnrollmentID = dynamicChallengeInfo.UUID
		default:
			return nil, ctxerr.New(ctx, fmt.Sprintf("unsupported SCEP challenge purpose: %s", dynamicChallengeInfo.Purpose))
		}

		ext, err := apple_mdm.BuildAppleMDMCertificateBindingExtension(extension)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "failed to build Apple MDM certificate binding extension")
		}

		cert, err := next.SignX509CSRWithCallback(m.CSR, apple_mdm.AppleMDMSCEPCertificateSubject(hasNewEnrollmentOU), func(tmpl *x509.Certificate) {
			tmpl.ExtraExtensions = []pkix.Extension{ext}
		})
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "failed to sign CSR with extensions")
		}

		if err := store.SetAppleSCEPChallengeIssuedCert(ctx, m.ChallengePassword, cert.SerialNumber.Int64()); err != nil {
			// This is used primarily for auditing purposes, so we log the error but do not fail the request.
			logger.WarnContext(ctx, "failed to set issued cert for SCEP challenge", "err", err)
		}

		return cert, nil
	}
}

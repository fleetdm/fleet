package service

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
	"github.com/fxamacker/cbor/v2"
)

const appleEnterpriseAttestationRootCA = `-----BEGIN CERTIFICATE-----
MIICJDCCAamgAwIBAgIUQsDCuyxyfFxeq/bxpm8frF15hzcwCgYIKoZIzj0EAwMw
UTEtMCsGA1UEAwwkQXBwbGUgRW50ZXJwcmlzZSBBdHRlc3RhdGlvbiBSb290IENB
MRMwEQYDVQQKDApBcHBsZSBJbmMuMQswCQYDVQQGEwJVUzAeFw0yMjAyMTYxOTAx
MjRaFw00NzAyMjAwMDAwMDBaMFExLTArBgNVBAMMJEFwcGxlIEVudGVycHJpc2Ug
QXR0ZXN0YXRpb24gUm9vdCBDQTETMBEGA1UECgwKQXBwbGUgSW5jLjELMAkGA1UE
BhMCVVMwdjAQBgcqhkjOPQIBBgUrgQQAIgNiAAT6Jigq+Ps9Q4CoT8t8q+UnOe2p
oT9nRaUfGhBTbgvqSGXPjVkbYlIWYO+1zPk2Sz9hQ5ozzmLrPmTBgEWRcHjA2/y7
7GEicps9wn2tj+G89l3INNDKETdxSPPIZpPj8VmjQjBAMA8GA1UdEwEB/wQFMAMB
Af8wHQYDVR0OBBYEFPNqTQGd8muBpV5du+UIbVbi+d66MA4GA1UdDwEB/wQEAwIB
BjAKBggqhkjOPQQDAwNpADBmAjEA1xpWmTLSpr1VH4f8Ypk8f3jMUKYz4QPG8mL5
8m9sX/b2+eXpTv2pH4RZgJjucnbcAjEA4ZSB6S45FlPuS/u4pTnzoz632rA+xW/T
ZwFEh9bhKjJ+5VQ9/Do1os0u3LEkgN/r
-----END CERTIFICATE-----`

var (
	OIDAppleSerialNumber = asn1.ObjectIdentifier{1, 2, 840, 113635, 100, 8, 9, 1}
	OIDAppleNonce        = asn1.ObjectIdentifier{1, 2, 840, 113635, 100, 8, 11, 1}
)

func (s *Service) ValidateChallenge(ctx context.Context, enrollment *types.Enrollment, account *types.Account, challengeID uint, payload string) (*types.ChallengeResponse, error) {
	ctx, span := tracer.Start(ctx, "acme.service.ValidateChallenge")
	defer span.End()

	challenge, err := s.store.GetChallengeByID(ctx, account.ID, challengeID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting challenge by ID")
	}

	if challenge.Status != types.ChallengeStatusPending {
		return nil, types.InvalidChallengeStatusError(fmt.Sprintf("Challenge with ID %d is not pending and can not be validated", challengeID))
	}

	var validationErr error
	var updatedChallenge *types.Challenge
	switch challenge.ChallengeType {
	case types.DeviceAttestationChallengeType:
		updatedChallenge, validationErr = s.validateDeviceAttestationChallenge(ctx, enrollment, challenge, payload)
	default:
		return nil, types.InternalServerError(fmt.Sprintf("unsupported challenge type %s", challenge.ChallengeType))
	}

	if validationErr != nil && updatedChallenge == nil {
		return nil, validationErr
	}

	// We always call UpdateChallenge here since we update it's validity by reference
	// this internally updates challenge status, authorization status and order status, which we can
	// confidently do with only one authorization and one challenge per order, if we add more we need to re-work this.
	if updatedChallenge, err = s.store.UpdateChallenge(ctx, updatedChallenge); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "updating challenge status")
	}

	if validationErr != nil {
		return nil, validationErr
	}

	challengeResponse := &types.ChallengeResponse{
		ChallengeType: updatedChallenge.ChallengeType,
		Status:        updatedChallenge.Status,
		Token:         updatedChallenge.Token,
		Validated:     updatedChallenge.ValidatedAt(),
	}

	baseURL, err := s.getACMEBaseURL(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting base URL")
	}

	challengeURL, err := s.getACMEURLWithBaseURL(ctx, baseURL, enrollment.PathIdentifier, "challenges", fmt.Sprint(updatedChallenge.ID))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "constructing challenge URL")
	}
	challengeResponse.URL = challengeURL
	challengeResponse.Location = challengeURL

	return challengeResponse, nil
}

func (s *Service) validateDeviceAttestationChallenge(ctx context.Context, enrollment *types.Enrollment, challenge *types.Challenge, payload string) (*types.Challenge, error) {
	base64Decoded, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return nil, types.MalformedError(fmt.Sprintf("Failed to base64 decode payload: %v", err))
	}

	// Verify the CBOR structure
	if err := cbor.Wellformed(base64Decoded); err != nil {
		return nil, types.BadAttestationStatementError(fmt.Sprintf("Device attestation statement is not correctly CBOR formatted: %s", err.Error()))
	}

	var attestationObject types.AttestationObject
	if err := cbor.Unmarshal(base64Decoded, &attestationObject); err != nil {
		return nil, types.BadAttestationStatementError(fmt.Sprintf("Failed to unmarshal CBOR payload into attestation object: %s", err.Error()))
	}

	switch attestationObject.Format {
	case "apple":
		var appleStmt types.AppleDeviceAttestationStatement
		if err := cbor.Unmarshal(attestationObject.AttestationStatement, &appleStmt); err != nil {
			return nil, types.BadAttestationStatementError(fmt.Sprintf("Failed to unmarshal CBOR attestation statement into Apple format: %s", err.Error()))
		}

		return challenge, s.validateAppleDeviceAttestationStatement(ctx, enrollment, challenge, appleStmt)
	default:
		return nil, types.BadAttestationStatementError(fmt.Sprintf("Unsupported device attestation format: %s", attestationObject.Format))
	}
}

// Challenge status is updated by reference
func (s *Service) validateAppleDeviceAttestationStatement(ctx context.Context, enrollment *types.Enrollment, challenge *types.Challenge, attStmt types.AppleDeviceAttestationStatement) error {
	roots := s.TestAppleRootCAs
	if roots == nil {
		roots = x509.NewCertPool()
		rootCABlock, _ := pem.Decode([]byte(appleEnterpriseAttestationRootCA))
		if rootCABlock == nil {
			return types.BadAttestationStatementError("Failed to parse Apple Enterprise Attestation Root CA certificate")
		}
		rootCA, err := x509.ParseCertificate(rootCABlock.Bytes)
		if err != nil {
			return types.BadAttestationStatementError(fmt.Sprintf("Failed to parse Apple Enterprise Attestation Root CA certificate: %s", err.Error()))
		}
		roots.AddCert(rootCA)
	}

	if len(attStmt.X5C) < 1 {
		return types.BadAttestationStatementError("Apple device attestation statement must contain at least one certificate in x5c field")
	}

	leaf, err := x509.ParseCertificate(attStmt.X5C[0])
	if err != nil {
		return types.BadAttestationStatementError(fmt.Sprintf("Failed to parse leaf certificate in Apple device attestation statement: %s", err.Error()))
	}

	intermediates := x509.NewCertPool()
	for _, certBytes := range attStmt.X5C[1:] {
		cert, err := x509.ParseCertificate(certBytes)
		if err != nil {
			return types.BadAttestationStatementError(fmt.Sprintf("Failed to parse intermediate certificate in Apple device attestation statement: %s", err.Error()))
		}
		intermediates.AddCert(cert)
	}

	if _, err := leaf.Verify(x509.VerifyOptions{
		Roots:         roots,
		Intermediates: intermediates,
		CurrentTime:   time.Now().Truncate(time.Second),
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}); err != nil {
		return types.BadAttestationStatementError(fmt.Sprintf("Failed to verify Apple Root CA is part of certificate chain: %s", err.Error()))
	}

	// TODO: Should we do any validation on leaf.PublicKey? Apple docs on validation calls out "Retain the public key in the attestation leaf certificate for a later validation."
	// So unsure if we should persist it, or what the later validation might be.
	appleData := struct {
		SerialNumber string
		Nonce        []byte
	}{}

	for _, ext := range leaf.Extensions {
		if ext.Id.Equal(OIDAppleSerialNumber) {
			appleData.SerialNumber = string(ext.Value)
		} else if ext.Id.Equal(OIDAppleNonce) {
			appleData.Nonce = ext.Value
		}
	}

	sha256Token := sha256.Sum256([]byte(challenge.Token))
	if subtle.ConstantTimeCompare(appleData.Nonce, sha256Token[:]) != 1 {
		challenge.MarkInvalid()
		return types.BadAttestationStatementError("Apple freshness nonce does not match challenge token")
	}

	if appleData.SerialNumber != enrollment.HostIdentifier {
		challenge.MarkInvalid()
		return types.BadAttestationStatementError("Serial number in certificate does not match enrollment's host identifier")
	}

	enrolled, err := s.providers.IsDEPEnrolled(ctx, appleData.SerialNumber)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "checking DEP enrollment for serial number in attestation certificate")
	}

	if !enrolled {
		challenge.MarkInvalid()
		return types.BadAttestationStatementError("No DEP assignments found for serial number in certificate")
	}

	challenge.MarkValid()
	return nil
}

package service

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
	"go.step.sm/crypto/jose"
)

func (s *Service) CreateAccount(ctx context.Context, pathIdentifier string, enrollmentID uint, jwk jose.JSONWebKey, onlyReturnExisting bool) (*types.AccountResponse, error) {
	// authorization is checked in the endpoint implementation for JWS-protected endpoints

	account := &types.Account{
		ACMEEnrollmentID: enrollmentID,
		JSONWebKey:       jwk,
	}
	account, didCreate, err := s.store.CreateAccount(ctx, account, onlyReturnExisting)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating account in datastore")
	}

	baseURL, err := s.getACMEBaseURL(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting base URL")
	}

	ordersURL, err := s.getACMEURLWithBaseURL(ctx, baseURL, pathIdentifier, "accounts", fmt.Sprint(account.ID), "orders")
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "constructing orders URL for account")
	}
	acctURL, err := s.getACMEURLWithBaseURL(ctx, baseURL, pathIdentifier, "accounts", fmt.Sprint(account.ID))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "constructing account URL for account")
	}

	return &types.AccountResponse{
		CreatedAccount: account,
		DidCreate:      didCreate,
		Status:         "valid", // for now, in our implementation, always valid
		Orders:         ordersURL,
		Location:       acctURL,
	}, nil
}

func (s *Service) CreateOrder(ctx context.Context, enrollment *types.Enrollment, account *types.Account, partialOrder *types.Order) (*types.OrderResponse, error) {
	// authorization is checked in the endpoint implementation for JWS-protected endpoints

	// The "identifiers" passed as part of the newOrder request must be an array with a
	// single member of type "permanent-identifier" matching the serial specified in the
	// acme_enrollment that this enrollment was created for.
	if len(partialOrder.Identifiers) != 1 || partialOrder.Identifiers[0].Type != types.IdentifierTypePermanentIdentifier {
		return nil, types.UnsupportedIdentifierError("A single identifier of type permanent-identifier must be provided in the order request")
	}
	if partialOrder.Identifiers[0].Value != enrollment.HostIdentifier {
		return nil, types.RejectedIdentifierError("The identifier value does not match the host identifier for this enrollment")
	}

	// notBefore and notAfter, which are optional, must not be set because fleet is going
	// to control these and the Apple payload doesn't allow specification of them.
	if partialOrder.NotBefore != nil || partialOrder.NotAfter != nil {
		return nil, types.MalformedError("notBefore and notAfter must not be set in the order request")
	}

	identifiers := []types.Identifier{
		{Type: partialOrder.Identifiers[0].Type, Value: partialOrder.Identifiers[0].Value},
	}
	order := &types.Order{
		ACMEAccountID: account.ID,
		Finalized:     false,
		Identifiers:   identifiers,
		Status:        "pending", // always pending at creation
	}
	authz := &types.Authorization{
		Identifier: identifiers[0],
		Status:     "pending", // always pending at creation
	}
	challenge := &types.Challenge{
		ChallengeType: types.DeviceAttestationChallengeType, // only supported challenge for now
		Token:         types.CreateNonceEncodedForHeader(),
		Status:        "pending", // always pending at creation
	}
	order, err := s.store.CreateOrder(ctx, order, authz, challenge)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating order in datastore")
	}

	baseURL, err := s.getACMEBaseURL(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting base URL")
	}

	orderURL, err := s.getACMEURLWithBaseURL(ctx, baseURL, enrollment.PathIdentifier, "orders", fmt.Sprint(order.ID))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "constructing order URL for account")
	}
	finalizeURL, err := s.getACMEURLWithBaseURL(ctx, baseURL, enrollment.PathIdentifier, "orders", fmt.Sprint(order.ID), "finalize")
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "constructing finalize URL for account")
	}
	authzURL, err := s.getACMEURLWithBaseURL(ctx, baseURL, enrollment.PathIdentifier, "authorizations", fmt.Sprint(authz.ID))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "constructing authorization URL for account")
	}

	return &types.OrderResponse{
		ID:             order.ID,
		Status:         order.Status,
		Expires:        enrollment.NotValidAfter,
		Identifiers:    order.Identifiers,
		Authorizations: []string{authzURL},
		Finalize:       finalizeURL,
		Location:       orderURL,
	}, nil
}

func (s *Service) FinalizeOrder(ctx context.Context, enrollment *types.Enrollment, orderID uint, csr string) (*types.OrderResponse, error) {
	order, err := s.store.GetOrder(ctx, enrollment.ID, orderID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting order from datastore")
	}
	if order.Status != types.OrderStatusReady || order.Finalized {
		extra := ""
		if order.Finalized {
			extra = " and order has already been finalized"
		}
		return nil, types.OrderNotReadyError(fmt.Sprintf("Order is in status %s%s.", order.Status, extra))
	}
	authorizations, err := s.store.GetAuthorizationsByOrderID(ctx, orderID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting authorizations for order from datastore")
	}

	baseURL, err := s.getACMEBaseURL(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting base URL")
	}
	authzURLs := make([]string, 0, len(authorizations))
	for _, authz := range authorizations {
		authzURL, err := s.getACMEURLWithBaseURL(ctx, baseURL, enrollment.PathIdentifier, "authorizations", fmt.Sprint(authz.ID))
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "constructing authorization URL for account")
		}
		authzURLs = append(authzURLs, authzURL)
		if authz.Status != types.AuthorizationStatusValid {
			return nil, types.OrderNotReadyError(fmt.Sprintf("Order has correct status but authorization %s has status %s.", authzURL, authz.Status))
		}
		challenges, err := s.store.GetChallengesByAuthorizationID(ctx, authz.ID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "getting challenges for authorization from datastore")
		}
		hasAValidChallenge := false
		for _, chlg := range challenges {
			if chlg.Status == types.ChallengeStatusValid {
				hasAValidChallenge = true
				break
			}
		}
		if !hasAValidChallenge {
			return nil, types.OrderNotReadyError(fmt.Sprintf("Order has correct status but no valid challenges for authorization %s.", authzURL))
		}
	}

	parsedCSR, err := parsePEMCSR(csr)
	if err != nil {
		return nil, types.BadCSRError(fmt.Sprintf("Error parsing PEM CSR: %w", err))
	}
	if parsedCSR.Subject.CommonName != order.Identifiers[0].Value {
		return nil, types.BadCSRError("CSR common name does not match identifier value")
	}
	// We only support ecdsa CSRs for now since that's what the Apple MDM protocol supports, so if it's not an ECDSA CSR we
	// return a bad CSR error. We can always add support for more types later if needed. This mirrors the logic we use with
	// the JWK
	if parsedCSR.PublicKeyAlgorithm != x509.ECDSA {
		return nil, types.BadCSRError("Public key is not an Elliptic Curve key as expected")
	}
	err = parsedCSR.CheckSignature()
	if err != nil {
		return nil, types.BadCSRError("CSR signature is invalid")
	}
	// Update the CSR common name and OU to match Fleet-issued SCEP certs
	parsedCSR.Subject.CommonName = "Fleet Identity"
	parsedCSR.Subject.OrganizationalUnit = []string{"fleet"}

	signer, err := s.providers.CSRSigner(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting CSR signer")
	}
	cert, err := signer.SignCSR(ctx, parsedCSR)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "signing CSR")
	}

	orderURL, err := s.getACMEURLWithBaseURL(ctx, baseURL, enrollment.PathIdentifier, "orders", fmt.Sprint(order.ID))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "constructing order URL for account")
	}
	s.store.FinalizeOrder(ctx, orderID, csr, cert.SerialNumber.Int64())
	finalizeURL, err := s.getACMEURLWithBaseURL(ctx, baseURL, enrollment.PathIdentifier, "orders", fmt.Sprint(order.ID), "finalize")
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "constructing finalize URL for account")
	}
	certificateURL, err := s.getACMEURLWithBaseURL(ctx, baseURL, enrollment.PathIdentifier, "orders", fmt.Sprint(order.ID), "certificate")
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "constructing certificate URL for account")
	}

	return &types.OrderResponse{
		ID:             order.ID,
		Status:         types.OrderStatusValid,
		Expires:        enrollment.NotValidAfter,
		Identifiers:    order.Identifiers,
		Authorizations: authzURLs,
		Finalize:       finalizeURL,
		Certificate:    certificateURL,
		Location:       orderURL,
	}, nil
}

func parsePEMCSR(pemCSR string) (*x509.CertificateRequest, error) {
	block, _ := pem.Decode([]byte(pemCSR))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	if block.Type != "CERTIFICATE REQUEST" {
		// TODO Bad CSR error
	}

	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		// TODO Bad CSR error
	}

	return csr, nil
}

func certificateToPEM(cert *x509.Certificate) string {
	pemBlock := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	}
	return string(pem.EncodeToMemory(pemBlock))
}

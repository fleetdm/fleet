package service

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"strings"

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
	ctx, span := tracer.Start(ctx, "acme.service.CreateOrder")
	defer span.End()

	// authorization is checked in the endpoint implementation for JWS-protected endpoints

	if err := partialOrder.ValidateOrderCreation(enrollment); err != nil {
		return nil, err
	}

	identifiers := []types.Identifier{
		{Type: partialOrder.Identifiers[0].Type, Value: partialOrder.Identifiers[0].Value},
	}
	order := &types.Order{
		ACMEAccountID: account.ID,
		Finalized:     false,
		Identifiers:   identifiers,
		Status:        types.OrderStatusPending, // always pending at creation
	}
	authz := &types.Authorization{
		Identifier: identifiers[0],
		Status:     types.AuthorizationStatusPending, // always pending at creation
	}
	challenge := &types.Challenge{
		ChallengeType: types.DeviceAttestationChallengeType, // only supported challenge for now
		Token:         types.CreateNonceEncodedForHeader(),
		Status:        types.ChallengeStatusPending, // always pending at creation
	}
	order, err := s.store.CreateOrder(ctx, order, authz, challenge)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating order in datastore")
	}
	return s.createOrderResponse(ctx, enrollment, order, []*types.Authorization{authz})
}

func (s *Service) createOrderResponse(
	ctx context.Context,
	enrollment *types.Enrollment,
	order *types.Order,
	authorizations []*types.Authorization,
) (*types.OrderResponse, error) {
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

	var authzURL string
	if len(authorizations) == 1 {
		authzURL, err = s.getACMEURLWithBaseURL(ctx, baseURL, enrollment.PathIdentifier, "authorizations", fmt.Sprint(authorizations[0].ID))
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "constructing authorization URL for account")
		}
	}

	var certURL string
	if err := order.IsCertificateReady(); err == nil {
		certURL, err = s.getACMEURLWithBaseURL(ctx, baseURL, enrollment.PathIdentifier, "orders", fmt.Sprint(order.ID), "certificate")
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "constructing certificate URL for account")
		}
	}

	return &types.OrderResponse{
		ID:             order.ID,
		Status:         order.Status,
		Expires:        enrollment.NotValidAfter,
		Identifiers:    order.Identifiers,
		Authorizations: []string{authzURL},
		Finalize:       finalizeURL,
		Certificate:    certURL,
		Location:       orderURL,
	}, nil
}

func (s *Service) FinalizeOrder(ctx context.Context, enrollment *types.Enrollment, account *types.Account, orderID uint, csr string) (*types.OrderResponse, error) {
	ctx, span := tracer.Start(ctx, "acme.service.FinalizeOrder")
	defer span.End()

	order, authorizations, err := s.store.GetOrderByID(ctx, account.ID, orderID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting order from datastore")
	}

	if err := order.IsReadyToFinalize(); err != nil {
		return nil, err
	}

	baseURL, err := s.getACMEBaseURL(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting base URL")
	}

	for _, authz := range authorizations {
		authzURL, err := s.getACMEURLWithBaseURL(ctx, baseURL, enrollment.PathIdentifier, "authorizations", fmt.Sprint(authz.ID))
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "constructing authorization URL for account")
		}

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

	// The RFC 7.4 calls out that for the CSR it sends a base64url-encoded DER (so not a full PEM block)
	parsedCSR, err := parseDERCSR(csr)
	if err != nil {
		return nil, types.BadCSRError(fmt.Sprintf("Error parsing DER CSR: %s", err))
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

	err = s.store.FinalizeOrder(ctx, orderID, csr, cert.SerialNumber.Int64())
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "finalizing order")
	}
	order.Status = types.OrderStatusValid
	order.Finalized = true

	return s.createOrderResponse(ctx, enrollment, order, authorizations)
}

func parseDERCSR(csr string) (*x509.CertificateRequest, error) {
	// The CSR is base64 url encoded
	base64DecodedCSR, err := base64.RawURLEncoding.DecodeString(csr)
	if err != nil {
		return nil, types.BadCSRError(fmt.Sprintf("Error decoding base64 CSR: %s", err))
	}

	parsedCSR, err := x509.ParseCertificateRequest(base64DecodedCSR)
	if err != nil {
		return nil, fmt.Errorf("error parsing certificate request: %w", err)
	}

	return parsedCSR, nil
}

func (s *Service) GetOrder(ctx context.Context, enrollment *types.Enrollment, account *types.Account, orderID uint) (*types.OrderResponse, error) {
	// authorization is checked in the endpoint implementation for JWS-protected endpoints

	order, authorizations, err := s.store.GetOrderByID(ctx, account.ID, orderID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get order from datastore")
	}
	return s.createOrderResponse(ctx, enrollment, order, authorizations)
}

func (s *Service) ListAccountOrders(ctx context.Context, pathIdentifier string, account *types.Account) ([]string, error) {
	// authorization is checked in the endpoint implementation for JWS-protected endpoints

	orderIDs, err := s.store.ListAccountOrderIDs(ctx, account.ID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing account order IDs from datastore")
	}

	var orderURLs []string
	if len(orderIDs) > 0 {
		baseURL, err := s.getACMEBaseURL(ctx)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "getting base URL")
		}

		orderURLs = make([]string, len(orderIDs))
		for i, orderID := range orderIDs {
			orderURL, err := s.getACMEURLWithBaseURL(ctx, baseURL, pathIdentifier, "orders", fmt.Sprint(orderID))
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "constructing order URL for account")
			}
			orderURLs[i] = orderURL
		}
	}
	return orderURLs, nil
}

func (s *Service) GetCertificate(ctx context.Context, accountID, orderID uint) (string, error) {
	// authorization is checked in the endpoint implementation for JWS-protected endpoints

	order, _, err := s.store.GetOrderByID(ctx, accountID, orderID)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "get order from datastore")
	}

	if err := order.IsCertificateReady(); err != nil {
		return "", err
	}

	certPEM, err := s.store.GetCertificatePEMByOrderID(ctx, accountID, orderID)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "get certificate from datastore")
	}
	if !strings.HasSuffix(certPEM, "\n") {
		certPEM += "\n"
	}

	// retrieve the root certificate
	rootPEMBytes, err := s.providers.GetCACertificatePEM(ctx)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "getting Apple SCEP/ACME root certificate")
	}
	block, _ := pem.Decode(rootPEMBytes)
	if block == nil || block.Type != "CERTIFICATE" {
		return "", ctxerr.New(ctx, "failed to parse PEM block from root SCEP/ACME certificate")
	}
	rootPEM := string(pem.EncodeToMemory(block))

	return certPEM + rootPEM, nil
}

package service

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
)

////////////////////////////////////////////////////////////////////////////////
// GET /mdm/apple
////////////////////////////////////////////////////////////////////////////////

type getAppleMDMResponse struct {
	*fleet.AppleMDM
	Err error `json:"error,omitempty"`
}

func (r getAppleMDMResponse) error() error { return r.Err }

func getAppleMDMEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	appleMDM, err := svc.GetAppleMDM(ctx)
	if err != nil {
		return getAppleMDMResponse{Err: err}, nil
	}

	return getAppleMDMResponse{AppleMDM: appleMDM}, nil
}

func (svc *Service) GetAppleMDM(ctx context.Context) (*fleet.AppleMDM, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AppleMDM{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	// if there is no apple mdm config, fail with a 404
	if !svc.config.MDM.IsAppleAPNsSet() {
		return nil, notFoundError{}
	}

	apns, _, _, err := svc.config.MDM.AppleAPNs()
	if err != nil {
		return nil, err
	}

	appleMDM := &fleet.AppleMDM{
		CommonName: apns.Leaf.Subject.CommonName,
		Issuer:     apns.Leaf.Issuer.CommonName,
		RenewDate:  apns.Leaf.NotAfter,
	}
	if apns.Leaf.SerialNumber != nil {
		appleMDM.SerialNumber = apns.Leaf.SerialNumber.String()
	}

	return appleMDM, nil
}

////////////////////////////////////////////////////////////////////////////////
// GET /mdm/apple_bm
////////////////////////////////////////////////////////////////////////////////

type getAppleBMResponse struct {
	*fleet.AppleBM
	Err error `json:"error,omitempty"`
}

func (r getAppleBMResponse) error() error { return r.Err }

func getAppleBMEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	appleBM, err := svc.GetAppleBM(ctx)
	if err != nil {
		return getAppleBMResponse{Err: err}, nil
	}

	return getAppleBMResponse{AppleBM: appleBM}, nil
}

func (svc *Service) GetAppleBM(ctx context.Context) (*fleet.AppleBM, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// GET /mdm/apple/request_csr
////////////////////////////////////////////////////////////////////////////////

type requestMDMAppleCSRRequest struct {
	EmailAddress string `json:"email_address"`
	Organization string `json:"organization"`
}

type requestMDMAppleCSRResponse struct {
	*fleet.AppleCSR
	Err error `json:"error,omitempty"`
}

func (r requestMDMAppleCSRResponse) error() error { return r.Err }

func requestMDMAppleCSREndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*requestMDMAppleCSRRequest)

	csr, err := svc.RequestMDMAppleCSR(ctx, req.EmailAddress, req.Organization)
	if err != nil {
		return requestMDMAppleCSRResponse{Err: err}, nil
	}
	return requestMDMAppleCSRResponse{
		AppleCSR: csr,
	}, nil
}

func (svc *Service) RequestMDMAppleCSR(ctx context.Context, email, org string) (*fleet.AppleCSR, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AppleCSR{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	if err := fleet.ValidateEmail(email); err != nil {
		if strings.TrimSpace(email) == "" {
			return nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("email_address", "missing email address"))
		}
		return nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("email_address", fmt.Sprintf("invalid email address: %v", err)))
	}
	if strings.TrimSpace(org) == "" {
		return nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("organization", "missing organization"))
	}

	// create the raw SCEP CA cert and key (creating before the CSR signing
	// request so that nothing can fail after the request is made, except for the
	// network during the response of course)
	scepCACert, scepCAKey, err := apple_mdm.NewSCEPCACertKey()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "generate SCEP CA cert and key")
	}

	// create the APNs CSR
	apnsCSR, apnsKey, err := apple_mdm.GenerateAPNSCSRKey(email, org)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "generate APNs CSR")
	}

	// request the signed APNs CSR from fleetdm.com
	client := fleethttp.NewClient(fleethttp.WithTimeout(10 * time.Second))
	if err := apple_mdm.GetSignedAPNSCSR(client, apnsCSR); err != nil {
		if ferr, ok := err.(apple_mdm.FleetWebsiteError); ok {
			status := http.StatusBadGateway
			if ferr.Status >= 400 && ferr.Status <= 499 {
				// TODO: fleetdm.com returns a genereric "Bad
				// Request" message, we should coordinate and
				// stablish a response schema from which we can get
				// the invalid field and use
				// fleet.NewInvalidArgumentError instead
				//
				// For now, since we have already validated
				// everything else, we assume that a 4xx
				// response is an email with an invalid domain
				return nil, ctxerr.Wrap(
					ctx,
					fleet.NewInvalidArgumentError(
						"email_address",
						fmt.Sprintf("this email address is not valid: %v", err),
					),
				)
			}
			return nil, ctxerr.Wrap(
				ctx,
				fleet.NewUserMessageError(
					fmt.Errorf("FleetDM CSR request failed: %w", err),
					status,
				),
			)
		}

		return nil, ctxerr.Wrap(ctx, err, "get signed CSR")
	}

	// PEM-encode the cert and keys
	scepCACertPEM := apple_mdm.EncodeCertPEM(scepCACert)
	scepCAKeyPEM := apple_mdm.EncodePrivateKeyPEM(scepCAKey)
	apnsKeyPEM := apple_mdm.EncodePrivateKeyPEM(apnsKey)

	return &fleet.AppleCSR{
		APNsKey:  apnsKeyPEM,
		SCEPCert: scepCACertPEM,
		SCEPKey:  scepCAKeyPEM,
	}, nil
}

////////////////////////////////////////////////////////////////////////////////
// POST /mdm/apple/dep_login
////////////////////////////////////////////////////////////////////////////////

type mdmAppleDEPLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type mdmAppleDEPLoginResponse struct {
	// profile is used in hijackRender for the response.
	profile []byte
	Err     error `json:"error,omitempty"`
}

func (r mdmAppleDEPLoginResponse) hijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(r.profile)), 10))
	w.Header().Set("Content-Type", "application/x-apple-aspen-config")

	// OK to just log the error here as writing anything on
	// `http.ResponseWriter` sets the status code to 200 (and it can't be
	// changed.) Clients should rely on matching content-length with the
	// header provided.
	if n, err := w.Write(r.profile); err != nil {
		logging.WithExtras(ctx, "err", err, "written", n)
	}
}

func (r mdmAppleDEPLoginResponse) error() error { return r.Err }

func mdmAppleDEPLoginEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*mdmAppleDEPLoginRequest)
	enrollProfile, err := svc.MDMAppleOktaLogin(ctx, req.Username, req.Password)
	if err != nil {
		return mdmAppleDEPLoginResponse{Err: err}, nil
	}

	return mdmAppleDEPLoginResponse{profile: enrollProfile}, nil
}

func (svc *Service) MDMAppleOktaLogin(ctx context.Context, username, password string) ([]byte, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

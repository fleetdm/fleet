package service

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"

	mdm_types "github.com/fleetdm/fleet/v4/server/fleet"
	mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/google/uuid"
)

//////////////////////////////////////////////////////////////////////////////////////////////////////////
// MS-MDE2 XML types used by the SOAP protocol
// MS-MDE2 is a client-to-server protocol that consists of a SOAP-based Web service.
// SOAP is a lightweight and XML based protocol that consists of three parts:
// - An envelope that defines a framework for describing what is in a message and how to process it
// - A set of encoding rules for expressing instances of application-defined datatypes
// - And a convention for representing remote procedure calls and responses.

// SoapResponse is the Soap Envelope Response type for MS-MDE2 responses from the server
// This envelope XML message is composed by a mandatory SOAP envelope, a SOAP header, and a SOAP body
type SoapResponse struct {
	XMLName xml.Name                 `xml:"s:Envelope"`
	XmlNSS  string                   `xml:"xmlns:s,attr"`
	XmlNSA  string                   `xml:"xmlns:a,attr"`
	XmlNSU  *string                  `xml:"xmlns:u,attr,omitempty"`
	Header  mdm_types.ResponseHeader `xml:"s:Header"`
	Body    mdm_types.BodyResponse   `xml:"s:Body"`
}

// error returns soap fault error if present
func (msg SoapResponse) error() error {
	if msg.Body.SoapFault != nil {
		return fmt.Errorf("soap fault: %s", msg.Body.SoapFault.Reason.Text.Content)
	}
	return nil
}

// isValidHeader checks for required fields in the header
func (req *SoapRequest) isValidHeader() error {
	// Check for required fields

	if len(req.XmlNSS) == 0 {
		return errors.New("invalid req header: XmlNSS")
	}

	if len(req.XmlNSA) == 0 {
		return errors.New("XmlNSA")
	}

	if len(req.Header.MessageID) == 0 {
		return errors.New("Header.MessageID")
	}

	if len(req.Header.Action.Content) == 0 {
		return errors.New("Header.Action")
	}

	if len(req.Header.ReplyTo.Address) == 0 {
		return errors.New("Header.ReplyTo")
	}

	if len(req.Header.To.Content) == 0 {
		return errors.New("Header.To")
	}

	return nil
}

// IsValidDiscoveryMsg checks for required fields in the Discover message
func (req *SoapRequest) IsValidDiscoveryMsg() error {
	if err := req.isValidHeader(); err != nil {
		return fmt.Errorf("invalid header: %s", err)
	}

	if req.Body.Discover == nil {
		return errors.New("invalid body: Discover message not present")
	}

	if len(req.Body.Discover.XmlNS) == 0 {
		return errors.New("invalid discover message: XmlNS")
	}

	// TODO: add check for valid email address
	if len(req.Body.Discover.Request.EmailAddress) == 0 {
		return errors.New("invalid discover message: Request.EmailAddress")
	}

	// Ensure that only valid versions are supported
	if req.Body.Discover.Request.RequestVersion != mdm.MinEnrollmentVersion &&
		req.Body.Discover.Request.RequestVersion != mdm.MaxEnrollmentVersion {
		return errors.New("invalid discover message: Request.RequestVersion")
	}

	// Traverse the AuthPolicies slice and check for valid values
	isInvalidAuth := true
	for _, authPolicy := range req.Body.Discover.Request.AuthPolicies.AuthPolicy {
		if authPolicy == mdm.AuthOnPremise {
			isInvalidAuth = false
			break
		}
	}

	if isInvalidAuth {
		return errors.New("invalid discover message: Request.AuthPolicies")
	}

	return nil
}

// IsValidGetPolicyMsg checks for required fields in the GetPolicies message
func (req *SoapRequest) IsValidGetPolicyMsg() error {
	if err := req.isValidHeader(); err != nil {
		return fmt.Errorf("invalid header: %s", err)
	}

	if req.Body.GetPolicies == nil {
		return errors.New("invalid body: GetPolicies message not present")
	}

	if len(req.Body.GetPolicies.XmlNS) == 0 {
		return errors.New("invalid getpolicies message: XmlNS")
	}

	return nil
}

// IsValidRequestSecurityTokenMsg checks for required fields in the RequestSecurityToken message
func (req *SoapRequest) IsValidRequestSecurityTokenMsg() error {
	if err := req.isValidHeader(); err != nil {
		return fmt.Errorf("invalid header: %s", err)
	}

	if req.Body.RequestSecurityToken == nil {
		return errors.New("invalid body: RequestSecurityToken message not present")
	}

	if len(req.Body.RequestSecurityToken.TokenType) == 0 {
		return errors.New("invalid requestsecuritytoken message: TokenType")
	}

	if len(req.Body.RequestSecurityToken.RequestType) == 0 {
		return errors.New("invalid requestsecuritytoken message: RequestType")
	}

	if len(req.Body.RequestSecurityToken.BinarySecurityToken.ValueType) == 0 {
		return errors.New("invalid requestsecuritytoken message: BinarySecurityToken.ValueType")
	}

	if len(req.Body.RequestSecurityToken.BinarySecurityToken.EncodingType) == 0 {
		return errors.New("invalid requestsecuritytoken message: BinarySecurityToken.EncodingType")
	}

	if len(req.Body.RequestSecurityToken.BinarySecurityToken.Content) == 0 {
		return errors.New("invalid requestsecuritytoken message: BinarySecurityToken.Content")
	}

	return nil
}

// SoapRequest is the Soap Envelope Request type for MS-MDE2 responses to the server
// This envelope XML message is composed by a mandatory SOAP envelope, a SOAP header, and a SOAP body
type SoapRequest struct {
	XMLName   xml.Name                `xml:"Envelope"`
	XmlNSS    string                  `xml:"s,attr"`
	XmlNSA    string                  `xml:"a,attr"`
	XmlNSU    *string                 `xml:"u,attr,omitempty"`
	XmlNSWsse *string                 `xml:"wsse,attr,omitempty"`
	XmlNSWST  *string                 `xml:"wst,attr,omitempty"`
	XmlNSAC   *string                 `xml:"ac,attr,omitempty"`
	Header    mdm_types.RequestHeader `xml:"Header"`
	Body      mdm_types.BodyRequest   `xml:"Body"`
}

// getUtcTime returns the current timestamp plus the specified number of minutes,
// formatted as "2006-01-02T15:04:05.000Z".
func getUtcTime(minutes int) string {
	// Get the current time and then add the specified number of minutes
	now := time.Now()
	future := now.Add(time.Duration(minutes) * time.Minute)

	// Format and return the future time as a string
	return future.UTC().Format("2006-01-02T15:04:05.000Z")
}

// NewDiscoverResponse creates a new DiscoverResponse struct based on the auth policy, policy url, and enrollment url
func NewDiscoverResponse(authPolicy string, policyUrl string, enrollmentUrl string) (mdm_types.DiscoverResponse, error) {
	if (len(authPolicy) == 0) || (len(policyUrl) == 0) || (len(enrollmentUrl) == 0) {
		return mdm_types.DiscoverResponse{}, errors.New("invalid parameters")
	}

	return mdm_types.DiscoverResponse{
		XmlNS: mdm.DiscoverNS,
		DiscoverResult: mdm_types.DiscoverResult{
			AuthPolicy:                 authPolicy,
			EnrollmentVersion:          mdm.MinEnrollmentVersion,
			EnrollmentPolicyServiceUrl: policyUrl,
			EnrollmentServiceUrl:       enrollmentUrl,
		},
	}, nil
}

// NewGetPoliciesResponse creates a new GetPoliciesResponse struct based on the minimal key length, certificate validity period, and renewal period
func NewGetPoliciesResponse(minimalKeyLength string, certificateValidityPeriodSeconds string, renewalPeriodSeconds string) (mdm_types.GetPoliciesResponse, error) {
	if (len(minimalKeyLength) == 0) || (len(certificateValidityPeriodSeconds) == 0) || (len(renewalPeriodSeconds) == 0) {
		return mdm_types.GetPoliciesResponse{}, errors.New("invalid parameters")
	}

	return mdm_types.GetPoliciesResponse{
		XmlNS: mdm.PolicyNS,
		Response: mdm_types.Response{
			PolicyFriendlyName: mdm_types.ContentAttr{
				Xsi:   mdm.DefaultStateXSI,
				XmlNS: mdm.EnrollXSI,
			},
			NextUpdateHours: mdm_types.ContentAttr{
				Xsi:   mdm.DefaultStateXSI,
				XmlNS: mdm.EnrollXSI,
			},
			PoliciesNotChanged: mdm_types.ContentAttr{
				Xsi:   mdm.DefaultStateXSI,
				XmlNS: mdm.EnrollXSI,
			},
			Policies: mdm_types.Policies{
				Policy: mdm_types.GPPolicy{
					PolicyOIDReference: "0",
					CAs: mdm_types.GenericAttr{
						Xsi: mdm.DefaultStateXSI,
					},
					Attributes: mdm_types.Attributes{
						CommonName:                "FleetDMAttributes",
						PolicySchema:              "3",
						HashAlgorithmOIDReference: "0",
						Revision: mdm_types.Revision{
							MajorRevision: "101",
							MinorRevision: "0",
						},
						CertificateValidity: mdm_types.CertificateValidity{
							ValidityPeriodSeconds: certificateValidityPeriodSeconds,
							RenewalPeriodSeconds:  renewalPeriodSeconds,
						},
						Permission: mdm_types.Permission{
							Enroll:     "true",
							AutoEnroll: "false",
						},
						SupersededPolicies: mdm_types.GenericAttr{
							Xsi: mdm.DefaultStateXSI,
						},
						PrivateKeyFlags: mdm_types.GenericAttr{
							Xsi: mdm.DefaultStateXSI,
						},
						SubjectNameFlags: mdm_types.GenericAttr{
							Xsi: mdm.DefaultStateXSI,
						},
						EnrollmentFlags: mdm_types.GenericAttr{
							Xsi: mdm.DefaultStateXSI,
						},
						GeneralFlags: mdm_types.GenericAttr{
							Xsi: mdm.DefaultStateXSI,
						},
						RARequirements: mdm_types.GenericAttr{
							Xsi: mdm.DefaultStateXSI,
						},
						KeyArchivalAttributes: mdm_types.GenericAttr{
							Xsi: mdm.DefaultStateXSI,
						},
						Extensions: mdm_types.GenericAttr{
							Xsi: mdm.DefaultStateXSI,
						},
						PrivateKeyAttributes: mdm_types.PrivateKeyAttributes{
							MinimalKeyLength: minimalKeyLength,
							KeySpec: mdm_types.GenericAttr{
								Xsi: mdm.DefaultStateXSI,
							},
							KeyUsageProperty: mdm_types.GenericAttr{
								Xsi: mdm.DefaultStateXSI,
							},
							Permissions: mdm_types.GenericAttr{
								Xsi: mdm.DefaultStateXSI,
							},
							AlgorithmOIDReference: mdm_types.GenericAttr{
								Xsi: mdm.DefaultStateXSI,
							},
							CryptoProviders: mdm_types.GenericAttr{
								Xsi: mdm.DefaultStateXSI,
							},
						},
					},
				},
			},
		},
		OIDs: mdm_types.OIDs{
			// SHA1WithRSA encryption OID
			// https://oidref.com/1.3.14.3.2.29
			OID: mdm_types.OID{
				Value:          "1.3.14.3.2.29",
				Group:          "1",
				OIDReferenceID: "0",
				DefaultName:    "szOID_NIST_sha256",
			},
		},
	}, nil
}

// NewRequestSecurityTokenResponseCollection creates a new RequestSecurityTokenResponseCollection struct based on the provisioned token
func NewRequestSecurityTokenResponseCollection(provisionedToken string) (mdm_types.RequestSecurityTokenResponseCollection, error) {
	if len(provisionedToken) == 0 {
		return mdm_types.RequestSecurityTokenResponseCollection{}, errors.New("invalid parameters")
	}

	enrollSecExtVal := mdm.EnrollSecExt
	return mdm_types.RequestSecurityTokenResponseCollection{
		XmlNS: mdm.EnrollWSTrust,
		RequestSecurityTokenResponse: mdm_types.RequestSecurityTokenResponse{
			TokenType: mdm.EnrollTokenType,
			DispositionMessage: mdm_types.SecAttr{
				Content: "",
				XmlNS:   mdm.EnrollReq,
			},
			RequestID: mdm_types.SecAttr{
				Content: "0",
				XmlNS:   mdm.EnrollReq,
			},
			RequestedSecurityToken: mdm_types.RequestedSecurityToken{
				BinarySecurityToken: mdm_types.BinarySecurityToken{
					Content:      provisionedToken,
					XmlNS:        &enrollSecExtVal,
					ValueType:    mdm.EnrollPDoc,
					EncodingType: mdm.EnrollEncode,
				},
			},
		},
	}, nil
}

// NewSoapFault creates a new SoapFault struct based on the error type, original message type, and error message
func NewSoapFault(errorType mdm.SoapError, origMessage mdm_types.MDEMessageType, errorMessage error) mdm_types.SoapFault {
	return mdm_types.SoapFault{
		OriginalMessageType: origMessage,
		Code: mdm_types.Code{
			Value: mdm.SoapFaultRecv,
			Subcode: mdm_types.Subcode{
				Value: string(errorType),
			},
		},
		Reason: mdm_types.Reason{
			Text: mdm_types.ReasonText{
				Content: errorMessage.Error(),
				Lang:    mdm.SoapFaultLocale,
			},
		},
	}
}

// getSoapResponseFault Returns a SoapResponse with a SoapFault on its body
func getSoapResponseFault(relatesTo string, errorType mdm.SoapError, origMessage mdm_types.MDEMessageType, errorMessage error) SoapResponse {
	soapFault := NewSoapFault(mdm.SoapErrorMessageFormat, mdm_types.MDEDiscovery, errorMessage)
	soapResponse, _ := NewSoapResponse(soapFault, relatesTo)
	return soapResponse
}

// NewSoapResponse creates a new SoapRequest struct based on the message type and the message content
func NewSoapResponse(payload interface{}, relatesTo string) (SoapResponse, error) {
	// Sanity check
	if len(relatesTo) == 0 {
		return SoapResponse{}, errors.New("relatesTo is invalid")
	}

	// Useful constants
	// Some of these are string urls to be assigned to pointers - they need to have a type and cannot be const literals
	var (
		urlNSS                = mdm.EnrollNSS
		urlNSA                = mdm.EnrollNSA
		urlXSI                = mdm.EnrollXSI
		urlXSD                = mdm.EnrollXSD
		urlXSU                = mdm.EnrollXSU
		urlDiag               = mdm.ActionNsDiag
		urlDiscovery          = mdm.ActionNsDiscovery
		urlPolicy             = mdm.ActionNsPolicy
		urlEnroll             = mdm.ActionNsEnroll
		urlSecExt             = mdm.EnrollSecExt
		MUValue               = "1"
		timestampID           = "_0"
		secWindowStartTimeMin = -5
		secWindowEndTimeMin   = 5
	)

	// string pointers - they need to be pointers to not be marshalled into the XML when nil
	var (
		headerXsu  *string
		action     string
		activityID *mdm_types.ActivityId
		security   *mdm_types.WsSecurity
	)

	// Build the response body
	var body mdm_types.BodyResponse

	// Set the message specific fields based on the message type
	switch msg := payload.(type) {

	case mdm_types.DiscoverResponse:
		action = urlDiscovery
		uuid := uuid.New().String()
		activityID = &mdm_types.ActivityId{
			Content:       uuid,
			CorrelationId: uuid,
			XmlNS:         urlDiag,
		}
		body.DiscoverResponse = &msg

	case mdm_types.GetPoliciesResponse:
		action = urlPolicy
		headerXsu = &urlXSU
		body.Xsi = &urlXSI
		body.Xsd = &urlXSD
		body.GetPoliciesResponse = &msg

	case mdm_types.RequestSecurityTokenResponseCollection:
		action = urlEnroll
		headerXsu = &urlXSU
		security = &mdm_types.WsSecurity{
			MustUnderstand: MUValue,
			XmlNS:          urlSecExt,
			Timestamp: mdm_types.Timestamp{
				ID:      timestampID,
				Created: getUtcTime(secWindowStartTimeMin), // minutes ago
				Expires: getUtcTime(secWindowEndTimeMin),   // minutes from now
			},
		}
		body.RequestSecurityTokenResponseCollection = &msg

	case mdm_types.SoapFault:
		// Setting the target action
		if msg.OriginalMessageType == mdm_types.MDEDiscovery {
			action = urlDiscovery
		} else if msg.OriginalMessageType == mdm_types.MDEPolicy {
			action = urlPolicy
		} else if msg.OriginalMessageType == mdm_types.MDEEnrollment {
			action = urlEnroll
		} else {
			action = urlDiag
		}
		uuid := uuid.New().String()
		activityID = &mdm_types.ActivityId{
			Content:       uuid,
			CorrelationId: uuid,
			XmlNS:         urlDiag,
		}
		body.SoapFault = &msg

	default:
		return SoapResponse{}, errors.New("mdm response message not supported")
	}

	// Return the SoapRequest type with the appropriate fields set
	return SoapResponse{
		XmlNSS: urlNSS,
		XmlNSA: urlNSA,
		XmlNSU: headerXsu,
		Header: mdm_types.ResponseHeader{
			Action: mdm_types.Action{
				Content:        action,
				MustUnderstand: MUValue,
			},
			RelatesTo:  relatesTo,
			ActivityId: activityID,
			Security:   security,
		},
		Body: body,
	}, nil
}

// mdmMicrosoftDiscoveryEndpoint is the response struct for the GetDiscovery endpoint
func mdmMicrosoftDiscoveryEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*SoapRequest)

	// Checking first if Discovery message is valid and returning error if this is not the case
	if err := req.IsValidDiscoveryMsg(); err != nil {
		return getSoapResponseFault(req.Header.MessageID, mdm.SoapErrorMessageFormat, mdm_types.MDEDiscovery, err), nil
	}

	// Getting the DiscoveryResponse message
	discoveryMessage, err := svc.GetMDMMicrosoftDiscoveryResponse(ctx)
	if err != nil {
		return getSoapResponseFault(req.Header.MessageID, mdm.SoapErrorMessageFormat, mdm_types.MDEDiscovery, err), nil
	}

	// Embedding the DiscoveryResponse message inside of a SoapResponse
	response, err := NewSoapResponse(discoveryMessage, req.Header.MessageID)
	if err != nil {
		return getSoapResponseFault(req.Header.MessageID, mdm.SoapErrorMessageFormat, mdm_types.MDEDiscovery, err), nil
	}

	return response, nil
}

// GetMDMMicrosoftDiscoveryResponse returns a valid DiscoveryResponse message
func (svc *Service) GetMDMMicrosoftDiscoveryResponse(ctx context.Context) (*fleet.DiscoverResponse, error) {
	// skipauth: This endpoint does not use authentication
	svc.authz.SkipAuthorization(ctx)

	// Getting the app config
	appCfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	// We can now craft the crafting DiscoveryResponse message
	urlMDMServerDomain := appCfg.ServerSettings.ServerURL
	urlDiscoveryEndpoint := urlMDMServerDomain + mdm.MDE2DiscoveryPath
	urlPolicyEndpoint := urlMDMServerDomain + mdm.MDE2PolicyPath
	urlEnrollEndpoint := urlMDMServerDomain + mdm.MDE2EnrollPath

	discoveryMsg, err := NewDiscoverResponse(urlDiscoveryEndpoint, urlPolicyEndpoint, urlEnrollEndpoint)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	return &discoveryMsg, nil
}

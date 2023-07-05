package service

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"

	mdm_types "github.com/fleetdm/fleet/v4/server/fleet"
	mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/google/uuid"
)

type SoapRequestContainer struct {
	Data *fleet.SoapRequest
	Err  error
}

// MDM SOAP request decoder
func (req *SoapRequestContainer) DecodeBody(ctx context.Context, r io.Reader) error {
	// Reading the request bytes
	reqBytes, err := io.ReadAll(r)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "reading soap mdm request")
	}

	// Unmarshal the XML data from the request into the SoapRequest struct
	err = xml.Unmarshal(reqBytes, &req.Data)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshalling soap mdm request")
	}

	return nil
}

type SoapResponseContainer struct {
	Data *fleet.SoapResponse
	Err  error
}

func (r SoapResponseContainer) error() error { return r.Err }

// hijackRender writes the response header and the RAW XML output
func (r SoapResponseContainer) hijackRender(ctx context.Context, w http.ResponseWriter) {
	xmlRes, err := xml.MarshalIndent(r.Data, "", "\t")
	if err != nil {
		logging.WithExtras(ctx, "Windows MDM SoapResponseContainer", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	xmlRes = append(xmlRes, '\n')

	w.Header().Set("Content-Type", mdm.SoapContentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(xmlRes)))
	w.WriteHeader(http.StatusOK)
	if n, err := w.Write(xmlRes); err != nil {
		logging.WithExtras(ctx, "err", err, "written", n)
	}
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
		XMLNS: mdm.DiscoverNS,
		DiscoverResult: mdm_types.DiscoverResult{
			AuthPolicy:                 authPolicy,
			EnrollmentVersion:          mdm.EnrollmentVersionV4,
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
		XMLNS: mdm.PolicyNS,
		Response: mdm_types.Response{
			PolicyFriendlyName: mdm_types.ContentAttr{
				Xsi:   mdm.DefaultStateXSI,
				XMLNS: mdm.EnrollXSI,
			},
			NextUpdateHours: mdm_types.ContentAttr{
				Xsi:   mdm.DefaultStateXSI,
				XMLNS: mdm.EnrollXSI,
			},
			PoliciesNotChanged: mdm_types.ContentAttr{
				Xsi:   mdm.DefaultStateXSI,
				XMLNS: mdm.EnrollXSI,
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
							CryptoProviders: []mdm_types.ProviderAttr{
								{Content: "Microsoft Platform Crypto Provider"},
								{Content: "Microsoft Software Key Storage Provider"},
							},
						},
					},
				},
			},
		},
		// These are MS-XCEP OIDs defined in section 3.1.4.1.3.16
		// https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-xcep/161aab9f-d159-4df3-85c9-f732ed2a8445
		OIDs: mdm_types.OIDs{
			OID: []mdm_types.OID{
				{
					// SHA256WithRSA OID
					// https://oidref.com/2.16.840.1.101.3.4.2.1
					Value:          "2.16.840.1.101.3.4.2.1",
					Group:          "4",
					OIDReferenceID: "0",
					DefaultName:    "szOID_NIST_sha256",
				},
				{
					// RSA OID
					// https://oidref.com/1.2.840.113549.1.1.1
					Value:          "1.2.840.113549.1.1.1",
					Group:          "3",
					OIDReferenceID: "1",
					DefaultName:    "szOID_RSA_RSA",
				},
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
		XMLNS: mdm.EnrollWSTrust,
		RequestSecurityTokenResponse: mdm_types.RequestSecurityTokenResponse{
			TokenType: mdm.EnrollTType,
			DispositionMessage: mdm_types.SecAttr{
				Content: "",
				XMLNS:   mdm.EnrollReq,
			},
			RequestID: mdm_types.SecAttr{
				Content: "0",
				XMLNS:   mdm.EnrollReq,
			},
			RequestedSecurityToken: mdm_types.RequestedSecurityToken{
				BinarySecurityToken: mdm_types.BinarySecurityToken{
					Content:      provisionedToken,
					XMLNS:        &enrollSecExtVal,
					ValueType:    mdm.EnrollPDoc,
					EncodingType: mdm.EnrollEncode,
				},
			},
		},
	}, nil
}

// NewSoapFault creates a new SoapFault struct based on the error type, original message type, and error message
func NewSoapFault(errorType string, origMessage int, errorMessage error) mdm_types.SoapFault {
	return mdm_types.SoapFault{
		OriginalMessageType: origMessage,
		Code: mdm_types.Code{
			Value: mdm.SoapFaultRecv,
			Subcode: mdm_types.Subcode{
				Value: errorType,
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
func getSoapResponseFault(relatesTo string, soapFault *mdm_types.SoapFault) errorer {
	if len(relatesTo) == 0 {
		relatesTo = "invalid_message_id"
	}

	response, _ := NewSoapResponse(soapFault, relatesTo)
	return SoapResponseContainer{
		Data: &response,
		Err:  nil,
	}
}

// NewSoapResponse creates a new SoapRequest struct based on the message type and the message content
func NewSoapResponse(payload interface{}, relatesTo string) (fleet.SoapResponse, error) {
	// Sanity check
	if len(relatesTo) == 0 {
		return fleet.SoapResponse{}, errors.New("relatesTo is invalid")
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

	case *mdm_types.DiscoverResponse:
		action = urlDiscovery
		uuid := uuid.New().String()
		activityID = &mdm_types.ActivityId{
			Content:       uuid,
			CorrelationId: uuid,
			XMLNS:         urlDiag,
		}
		body.DiscoverResponse = msg

	case *mdm_types.GetPoliciesResponse:
		action = urlPolicy
		headerXsu = &urlXSU
		body.Xsi = &urlXSI
		body.Xsd = &urlXSD
		body.GetPoliciesResponse = msg

	case *mdm_types.RequestSecurityTokenResponseCollection:
		action = urlEnroll
		headerXsu = &urlXSU
		security = &mdm_types.WsSecurity{
			MustUnderstand: MUValue,
			XMLNS:          urlSecExt,
			Timestamp: mdm_types.Timestamp{
				ID:      timestampID,
				Created: getUtcTime(secWindowStartTimeMin), // minutes ago
				Expires: getUtcTime(secWindowEndTimeMin),   // minutes from now
			},
		}
		body.RequestSecurityTokenResponseCollection = msg

		// Setting the target action
	case *mdm_types.SoapFault:
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
			XMLNS:         urlDiag,
		}
		body.SoapFault = msg

	default:
		return fleet.SoapResponse{}, errors.New("mdm response message not supported")
	}

	// Return the SoapRequest type with the appropriate fields set
	return fleet.SoapResponse{
		XMLNSS: urlNSS,
		XMLNSA: urlNSA,
		XMLNSU: headerXsu,
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

// NewBinarySecurityTokenPayload returns the BinarySecurityTokenPayload type
func NewBinarySecurityTokenPayload(encodedToken string) (fleet.WindowsMDMAccessTokenPayload, error) {
	if len(encodedToken) == 0 {
		return fleet.WindowsMDMAccessTokenPayload{}, errors.New("binary security token: token is empty")
	}

	rawBytes, err := base64.StdEncoding.DecodeString(encodedToken)
	if err != nil {
		return fleet.WindowsMDMAccessTokenPayload{}, fmt.Errorf("binary security token: %v", err)
	}

	var tokenPayload fleet.WindowsMDMAccessTokenPayload
	err = json.Unmarshal(rawBytes, &tokenPayload)
	if err != nil {
		return fleet.WindowsMDMAccessTokenPayload{}, fmt.Errorf("binary security token: %v", err)
	}

	return tokenPayload, nil
}

// GetEncodedBinarySecurityToken returns the base64 form of a BinarySecurityTokenPayload
func GetEncodedBinarySecurityToken(typeID fleet.WindowsMDMEnrollmentType, hostUUID string) (string, error) {
	var pld fleet.WindowsMDMAccessTokenPayload
	pld.Type = typeID
	pld.Payload.HostUUID = hostUUID
	rawBytes, err := json.Marshal(pld)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(rawBytes), nil
}

// newParm returns a new ProvisioningDoc Parameter
func newParm(name, value, datatype string) mdm_types.Param {
	return mdm_types.Param{
		Name:     name,
		Value:    value,
		Datatype: datatype,
	}
}

// newCharacteristic returns a new ProvisioningDoc Characteristic
func newCharacteristic(typ string, parms []mdm_types.Param, characteristics []mdm_types.Characteristic) mdm_types.Characteristic {
	return mdm_types.Characteristic{
		Type:            typ,
		Params:          parms,
		Characteristics: characteristics,
	}
}

// NewProvisioningDoc returns a new ProvisioningDoc container

// NewCertStoreProvisioningData returns a new CertStoreProvisioningData Characteristic
// The enrollment client installs the client certificate, as well as the trusted root certificate and intermediate certificates.
// The provisioning information in NewCertStoreProvisioningData includes various properties that the device management client uses to communicate with the MDM Server.
// identityFingerprint is the fingerprint of the identity certificate
// identityCert is the identity certificate bytes
// signedClientFingerprint is the fingerprint of the signed client certificate
// signedClientCert is the signed client certificate bytes
func NewCertStoreProvisioningData(enrollmentType string, identityFingerprint string, identityCert []byte, signedClientFingerprint string, signedClientCert []byte) mdm_types.Characteristic {
	// Target Cert Store selection based on Enrollment type
	targetCertStore := "User"
	if enrollmentType == "Device" {
		targetCertStore = "System"
	}

	root := newCharacteristic("Root", nil, []mdm_types.Characteristic{
		newCharacteristic("System", nil, []mdm_types.Characteristic{
			newCharacteristic(identityFingerprint, []mdm_types.Param{
				newParm("EncodedCertificate", base64.StdEncoding.EncodeToString(identityCert), ""),
			}, nil),
		}),
	})

	my := newCharacteristic("My", nil, []mdm_types.Characteristic{
		newCharacteristic(targetCertStore, nil, []mdm_types.Characteristic{
			newCharacteristic(signedClientFingerprint, []mdm_types.Param{
				newParm("EncodedCertificate", base64.StdEncoding.EncodeToString(signedClientCert), ""),
			}, nil),
			newCharacteristic("PrivateKeyContainer", nil, nil),
		}),
		newCharacteristic("WSTEP", nil, []mdm_types.Characteristic{
			newCharacteristic("Renew", []mdm_types.Param{
				newParm("ROBOSupport", mdm.WstepROBOSupport, "boolean"),
				newParm("RenewPeriod", mdm.WstepCertRenewalPeriodInDays, "integer"),
				newParm("RetryInterval", mdm.WstepRenewRetryInterval, "integer"),
			}, nil),
		}),
	})

	certStore := newCharacteristic("CertificateStore", nil, []mdm_types.Characteristic{root, my})
	return certStore
}

// NewApplicationProvisioningData returns a new ApplicationProvisioningData Characteristic
// The Application Provisioning configuration is used for bootstrapping a device with an OMA DM account
// The paramenters here maps to the W7 application CSP
// https://learn.microsoft.com/en-us/windows/client-management/mdm/w7-application-csp
func NewApplicationProvisioningData(mdmEndpoint string) mdm_types.Characteristic {
	provDoc := newCharacteristic("APPLICATION", []mdm_types.Param{
		// The PROVIDER-ID parameter specifies the server identifier for a management server used in the current management session
		newParm("PROVIDER-ID", mdm.DocProvisioningAppProviderID, ""),

		// The APPID parameter is used to differentiate the types of available application services and protocols.
		newParm("APPID", "w7", ""),

		// The NAME parameter is used in the APPLICATION characteristic to specify a user readable application identity.
		newParm("NAME", mdm.DocProvisioningAppName, ""),

		// The ADDR parameter is used in the APPADDR param to get or set the address of the OMA DM server.
		newParm("ADDR", mdmEndpoint, ""),

		// The ROLE parameter is used in the APPLICATION characteristic to specify the security application chamber that the DM session should run with when communicating with the DM server.

		// The BACKCOMPATRETRYFREQ parameter is used  to specify how many retries the DM client performs when there are Connection Manager-level or WinInet-level errors
		newParm("CONNRETRYFREQ", mdm.DocProvisioningAppConnRetryFreq, ""),

		// The INITIALBACKOFFTIME parameter is used to specify the initial wait time in milliseconds when the DM client retries for the first time
		newParm("INITIALBACKOFFTIME", mdm.DocProvisioningAppInitialBackoffTime, ""),

		// The MAXBACKOFFTIME parameter is used to specify the maximum number of milliseconds to sleep after package-sending failure
		newParm("MAXBACKOFFTIME", mdm.DocProvisioningAppMaxBackoffTime, ""),

		// The DEFAULTENCODING parameter is used to specify whether the DM client should use WBXML or XML for the DM package when communicating with the server.
		newParm("DEFAULTENCODING", "application/vnd.syncml.dm+xml", ""),

		// The BACKCOMPATRETRYDISABLED parameter is used to specify whether to retry resending a package with an older protocol version
		newParm("BACKCOMPATRETRYDISABLED", "", ""),
	}, []mdm_types.Characteristic{
		// CLIENT specifies that the server authenticates itself to the OMA DM Client at the DM protocol level.
		newCharacteristic("APPAUTH", []mdm_types.Param{
			newParm("AAUTHLEVEL", "CLIENT", ""),
			// DIGEST - Specifies that the SyncML DM 'syncml:auth-md5' authentication type.
			newParm("AAUTHTYPE", "DIGEST", ""),
			newParm("AAUTHSECRET", "dummy", ""),
			newParm("AAUTHDATA", "nonce", ""),
		}, nil),
		// APPSRV specifies that the client authenticates itself to the OMA DM Server at the DM protocol level.
		newCharacteristic("APPAUTH", []mdm_types.Param{
			newParm("AAUTHLEVEL", "APPSRV", ""),
			// DIGEST - Specifies that the SyncML DM 'syncml:auth-md5' authentication type.
			newParm("AAUTHTYPE", "DIGEST", ""),
			newParm("AAUTHNAME", "dummy", ""),
			newParm("AAUTHSECRET", "dummy", ""),
			newParm("AAUTHDATA", "nonce", ""),
		}, nil),
	})

	return provDoc
}

// NewDMClientProvisioningData returns a new DMClient Characteristic
// These settings can be used to define different aspects of the DM client behavior
// The provisioning information in NewCertStoreProvisioningData includes various properties that the device management client uses to communicate with the MDM Server.
// c2DeviceName is the device name used by the IT admin console
// listOfMSIAppToInstall contains a list of LocURIs that expected to be provision via EnterpriseDesktopAppManagement CSP
func NewDMClientProvisioningData() mdm_types.Characteristic {
	dmClient := newCharacteristic("DMClient", nil, []mdm_types.Characteristic{
		newCharacteristic("Provider", nil, []mdm_types.Characteristic{
			newCharacteristic(mdm.DocProvisioningAppProviderID,
				[]mdm_types.Param{}, []mdm_types.Characteristic{
					newCharacteristic("Poll", []mdm_types.Param{
						newParm("NumberOfFirstRetries", mdm.DmClientCSPNumberOfFirstRetries, mdm.DmClientIntType),
						newParm("IntervalForFirstSetOfRetries", mdm.DmClientCSPIntervalForFirstSetOfRetries, mdm.DmClientIntType),
						newParm("NumberOfSecondRetries", mdm.DmClientCSPNumberOfSecondRetries, mdm.DmClientIntType),
						newParm("IntervalForSecondSetOfRetries", mdm.DmClientCSPIntervalForSecondSetOfRetries, mdm.DmClientIntType),
						newParm("NumberOfRemainingScheduledRetries", mdm.DmClientCSPNumberOfRemainingScheduledRetries, mdm.DmClientIntType),
						newParm("IntervalForRemainingScheduledRetries", mdm.DmClientCSPIntervalForRemainingScheduledRetries, mdm.DmClientIntType),
						newParm("PollOnLogin", mdm.DmClientCSPPollOnLogin, mdm.DmClientBoolType),
						newParm("AllUsersPollOnFirstLogin", mdm.DmClientCSPPollOnLogin, mdm.DmClientBoolType),
					}, nil),
				}),
		}),
	})

	return dmClient
}

// NewProvisioningDoc returns a new ProvisioningDoc container
func NewProvisioningDoc(certStoreData mdm_types.Characteristic, applicationData mdm_types.Characteristic, dmClientData mdm_types.Characteristic) mdm_types.WapProvisioningDoc {
	return mdm_types.WapProvisioningDoc{
		Version: mdm.DocProvisioningVersion,
		Characteristics: []mdm_types.Characteristic{
			certStoreData,
			applicationData,
			dmClientData,
		},
	}
}

// mdmMicrosoftDiscoveryEndpoint handles the Discovery message and returns a valid DiscoveryResponse message
// DiscoverResponse message contains the Uniform Resource Locators (URLs) of service endpoints required for the following enrollment steps
func mdmMicrosoftDiscoveryEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*SoapRequestContainer).Data

	// Checking first if Discovery message is valid and returning error if this is not the case
	if err := req.IsValidDiscoveryMsg(); err != nil {
		soapFault := svc.GetAuthorizedSoapFault(ctx, mdm.SoapErrorMessageFormat, mdm_types.MDEDiscovery, err)
		return getSoapResponseFault(req.GetMessageID(), soapFault), nil
	}

	// Getting the DiscoveryResponse message
	discoveryResponseMsg, err := svc.GetMDMMicrosoftDiscoveryResponse(ctx)
	if err != nil {
		soapFault := svc.GetAuthorizedSoapFault(ctx, mdm.SoapErrorMessageFormat, mdm_types.MDEDiscovery, err)
		return getSoapResponseFault(req.GetMessageID(), soapFault), nil
	}

	// Embedding the DiscoveryResponse message inside of a SoapResponse
	response, err := NewSoapResponse(discoveryResponseMsg, req.GetMessageID())
	if err != nil {
		soapFault := svc.GetAuthorizedSoapFault(ctx, mdm.SoapErrorMessageFormat, mdm_types.MDEDiscovery, err)
		return getSoapResponseFault(req.GetMessageID(), soapFault), nil
	}

	return SoapResponseContainer{
		Data: &response,
		Err:  nil,
	}, nil
}

// mdmMicrosoftPolicyEndpoint handles the GetPolicies message and returns a valid GetPoliciesResponse message
// GetPoliciesResponse message contains the certificate policies required for the next enrollment step. For more information about these messages, see [MS-XCEP] sections 3.1.4.1.1.1 and 3.1.4.1.1.2.
func mdmMicrosoftPolicyEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*SoapRequestContainer).Data

	// Checking first if GetPolicies message is valid and returning error if this is not the case
	if err := req.IsValidGetPolicyMsg(); err != nil {
		soapFault := svc.GetAuthorizedSoapFault(ctx, mdm.SoapErrorMessageFormat, mdm_types.MDEPolicy, err)
		return getSoapResponseFault(req.GetMessageID(), soapFault), nil
	}

	// Binary security token should be extracted to ensure this is a valid call
	binSecTokenData, err := req.GetBinarySecurityToken()
	if err != nil {
		soapFault := svc.GetAuthorizedSoapFault(ctx, mdm.SoapErrorMessageFormat, mdm_types.MDEPolicy, err)
		return getSoapResponseFault(req.GetMessageID(), soapFault), nil
	}

	// Getting the GetPoliciesResponse message
	policyResponseMsg, err := svc.GetMDMWindowsPolicyResponse(ctx, binSecTokenData)
	if err != nil {
		soapFault := svc.GetAuthorizedSoapFault(ctx, mdm.SoapErrorMessageFormat, mdm_types.MDEPolicy, err)
		return getSoapResponseFault(req.GetMessageID(), soapFault), nil
	}

	// Embedding the DiscoveryResponse message inside of a SoapResponse
	response, err := NewSoapResponse(policyResponseMsg, req.GetMessageID())
	if err != nil {
		soapFault := svc.GetAuthorizedSoapFault(ctx, mdm.SoapErrorMessageFormat, mdm_types.MDEPolicy, err)
		return getSoapResponseFault(req.GetMessageID(), soapFault), nil
	}

	return SoapResponseContainer{
		Data: &response,
		Err:  nil,
	}, nil
}

// mdmMicrosoftEnrollEndpoint handles the RequestSecurityToken message and returns a valid RequestSecurityTokenResponseCollection message
// RequestSecurityTokenResponseCollection message contains the identity and provisioning information for the device management client.
func mdmMicrosoftEnrollEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*SoapRequestContainer).Data

	// Checking first if RequestSecurityToken message is valid and returning error if this is not the case
	if err := req.IsValidRequestSecurityTokenMsg(); err != nil {
		soapFault := svc.GetAuthorizedSoapFault(ctx, mdm.SoapErrorMessageFormat, mdm_types.MDEEnrollment, err)
		return getSoapResponseFault(req.GetMessageID(), soapFault), nil
	}

	// Getting the RequestSecurityToken message from the SOAP request
	reqSecurityTokenMsg, err := req.GetRequestSecurityTokenMessage()
	if err != nil {
		soapFault := svc.GetAuthorizedSoapFault(ctx, mdm.SoapErrorMessageFormat, mdm_types.MDEEnrollment, err)
		return getSoapResponseFault(req.GetMessageID(), soapFault), nil
	}

	// Binary security token should be extracted to ensure this is a valid call
	binSecTokenData, err := req.GetBinarySecurityToken()
	if err != nil {
		soapFault := svc.GetAuthorizedSoapFault(ctx, mdm.SoapErrorMessageFormat, mdm_types.MDEEnrollment, err)
		return getSoapResponseFault(req.GetMessageID(), soapFault), nil
	}

	// Getting the RequestSecurityTokenResponseCollection message
	enrollResponseMsg, err := svc.GetMDMWindowsEnrollResponse(ctx, reqSecurityTokenMsg, binSecTokenData)
	if err != nil {
		soapFault := svc.GetAuthorizedSoapFault(ctx, mdm.SoapErrorMessageFormat, mdm_types.MDEEnrollment, err)
		return getSoapResponseFault(req.GetMessageID(), soapFault), nil
	}

	// Embedding the DiscoveryResponse message inside of a SoapResponse
	response, err := NewSoapResponse(enrollResponseMsg, req.GetMessageID())
	if err != nil {
		soapFault := svc.GetAuthorizedSoapFault(ctx, mdm.SoapErrorMessageFormat, mdm_types.MDEEnrollment, err)
		return getSoapResponseFault(req.GetMessageID(), soapFault), nil
	}

	return SoapResponseContainer{
		Data: &response,
		Err:  nil,
	}, nil
}

// validateBinarySecurityToken checks if the provided token is valid
func (svc *Service) validateBinarySecurityToken(ctx context.Context, encodedBinarySecToken string) error {
	if len(encodedBinarySecToken) == 0 {
		return errors.New("binarySecurityTokenValidation: encoded token is invalid")
	}

	// Getting the Binary Security Token Payload
	binSecToken, err := NewBinarySecurityTokenPayload(encodedBinarySecToken)
	if err != nil {
		return fmt.Errorf("binarySecurityTokenValidation: token creation error %v", err)
	}

	// Validating the Binary Security Token Payload
	err = binSecToken.IsValidToken()
	if err != nil {
		return fmt.Errorf("binarySecurityTokenValidation: invalid token data %v", err)
	}

	// Validating the Binary Security Token Type used on Programmatic Enrollments
	if binSecToken.Type == mdm_types.WindowsMDMProgrammaticEnrollmentType {
		host, err := svc.ds.HostByIdentifier(ctx, binSecToken.Payload.HostUUID)
		if err != nil {
			return fmt.Errorf("binarySecurityTokenValidation: host data cannot be found %v", err)
		}

		// This ensures that only hosts that are eligible for Windows enrollment can be enrolled
		if !host.IsEligibleForWindowsMDMEnrollment() {
			return errors.New("binarySecurityTokenValidation: host is not elegible for Windows MDM enrollment")
		}

		// No errors, token is authorized
		return nil
	}

	return errors.New("binarySecurityTokenValidation: token is not authorized")
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

	// Getting the DiscoveryResponse message content
	urlPolicyEndpoint, err := mdm.ResolveWindowsMDMPolicy(appCfg.ServerSettings.ServerURL)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "resolve policy endpoint")
	}

	urlEnrollEndpoint, err := mdm.ResolveWindowsMDMEnroll(appCfg.ServerSettings.ServerURL)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "resolve enroll endpoint")
	}

	discoveryMsg, err := NewDiscoverResponse(mdm.AuthOnPremise, urlPolicyEndpoint, urlEnrollEndpoint)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creation of DiscoverResponse message")
	}

	return &discoveryMsg, nil
}

// GetMDMWindowsPolicyResponse returns a valid GetPoliciesResponse message
func (svc *Service) GetMDMWindowsPolicyResponse(ctx context.Context, authToken string) (*fleet.GetPoliciesResponse, error) {
	if len(authToken) == 0 {
		return nil, fleet.NewInvalidArgumentError("policy response", "authToken is empty")
	}

	// Validate the binary security token
	err := svc.validateBinarySecurityToken(ctx, authToken)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "validate binary security token")
	}

	// Token is authorized
	svc.authz.SkipAuthorization(ctx)

	// Getting the GetPoliciesResponse message content
	policyMsg, err := NewGetPoliciesResponse(mdm.PolicyMinKeyLength, mdm.PolicyCertValidityPeriodInSecs, mdm.PolicyCertRenewalPeriodInSecs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creation of GetPoliciesResponse message")
	}

	return &policyMsg, nil
}

// GetMDMWindowsEnrollResponse returns a valid RequestSecurityTokenResponseCollection message
// secTokenMsg is the RequestSecurityToken message
// authToken is the base64 encoded binary security token
func (svc *Service) GetMDMWindowsEnrollResponse(ctx context.Context, secTokenMsg *fleet.RequestSecurityToken, authToken string) (*fleet.RequestSecurityTokenResponseCollection, error) {
	if len(authToken) == 0 {
		return nil, fleet.NewInvalidArgumentError("enroll response", "authToken is empty")
	}

	// Validate the binary security token
	err := svc.validateBinarySecurityToken(ctx, authToken)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "validate binary security token")
	}

	// Removing the device if already MDM enrolled
	err = svc.removeWindowsDeviceIfAlreadyMDMEnrolled(ctx, secTokenMsg)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "device enroll check")
	}

	// Getting the the device provisioning information in the form of a WapProvisioningDoc
	deviceProvisioning, err := svc.getDeviceProvisioningInformation(ctx, secTokenMsg)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "device provisioning information")
	}

	// Token is authorized
	svc.authz.SkipAuthorization(ctx)

	// Getting the RequestSecurityTokenResponseCollection message content
	secTokenResponseCollectionMsg, err := NewRequestSecurityTokenResponseCollection(deviceProvisioning)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creation of RequestSecurityTokenResponseCollection message")
	}

	// RequestSecurityTokenResponseCollection message is ready The identity
	// and provisioning information will be sent to the Windows MDM
	// Enrollment Client

	// But before doing that, let's save the device information to the list
	// of MDM enrolled MDM devices
	//
	// This method also creates the relevant enrollment activity as it has
	// access to the device information.
	err = svc.storeWindowsMDMEnrolledDevice(ctx, secTokenMsg)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "enrolled device information cannot be stored")
	}

	return &secTokenResponseCollectionMsg, nil
}

// removeWindowsDeviceIfAlreadyMDMEnrolled removes the device if already MDM enrolled
// HW DeviceID is used to check the list of enrolled devices
func (svc *Service) removeWindowsDeviceIfAlreadyMDMEnrolled(ctx context.Context, secTokenMsg *fleet.RequestSecurityToken) error {
	// Getting the HW DeviceID from the RequestSecurityToken msg
	reqHWDeviceID, err := GetContextItem(secTokenMsg, mdm.ReqSecTokenContextItemHWDevID)
	if err != nil {
		return err
	}

	// Checking the storage to see if the device is already enrolled
	device, err := svc.ds.MDMWindowsGetEnrolledDevice(ctx, reqHWDeviceID)
	if err != nil {
		// Device is not present
		if fleet.IsNotFound(err) {
			return nil
		}

		return err
	}

	// Device is already enrolled, let's remove it
	err = svc.ds.MDMWindowsDeleteEnrolledDevice(ctx, device.MDMHardwareID)
	if err != nil {
		return err
	}

	return nil
}

// getDeviceProvisioningInformation returns a valid WapProvisioningDoc
// This is the provisioning information that will be sent to the Windows MDM Enrollment Client
// This information is used to configure the device management client
// See section 2.2.9.1 for more details on the XML provision schema used here
// https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-mde2/35e1aca6-1b8a-48ba-bbc0-23af5d46907a
func (svc *Service) getDeviceProvisioningInformation(ctx context.Context, secTokenMsg *fleet.RequestSecurityToken) (string, error) {
	// Getting the HW DeviceID from the RequestSecurityToken msg
	reqHWDeviceID, err := GetContextItem(secTokenMsg, mdm.ReqSecTokenContextItemHWDevID)
	if err != nil {
		return "", err
	}

	// Getting the EnrollmentType information from the RequestSecurityToken msg
	reqEnrollType, err := GetContextItem(secTokenMsg, mdm.ReqSecTokenContextItemEnrollmentType)
	if err != nil {
		return "", err
	}

	// Getting the BinarySecurityToken from the RequestSecurityToken msg
	binSecurityTokenData, err := secTokenMsg.GetBinarySecurityTokenData()
	if err != nil {
		return "", err
	}

	// Getting the BinarySecurityToken type from the RequestSecurityToken msg
	binSecurityTokenType, err := secTokenMsg.GetBinarySecurityTokenType()
	if err != nil {
		return "", err
	}

	// Getting the client CSR request from the device
	clientCSR, err := mdm.GetClientCSR(binSecurityTokenData, binSecurityTokenType)
	if err != nil {
		return "", err
	}

	// Getting the signed, DER-encoded certificate bytes and its uppercased, hex-endcoded SHA1 fingerprint
	rawSignedCertDER, rawSignedCertFingerprint, err := svc.SignMDMMicrosoftClientCSR(ctx, reqHWDeviceID, clientCSR)
	if err != nil {
		return "", err
	}

	// Preparing client certificate and identity certificate information to be sent to the Windows MDM Enrollment Client
	certStoreProvisioningData := NewCertStoreProvisioningData(
		reqEnrollType,
		svc.wstepCertManager.IdentityFingerprint(),
		svc.wstepCertManager.IdentityCert().Raw,
		rawSignedCertFingerprint,
		rawSignedCertDER)

	// Preparing the provisioning information that includes the location of the Device Management Service (DMS)
	appCfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return "", err
	}

	// Getting the MS-MDM management URL to provision the device
	urlManagementEndpoint, err := mdm.ResolveWindowsMDMManagement(appCfg.ServerSettings.ServerURL)
	if err != nil {
		return "", err
	}

	// Preparing the Application Provisioning information
	appConfigProvisioningData := NewApplicationProvisioningData(urlManagementEndpoint)

	// Preparing the DM Client Provisioning information
	appDMClientProvisioningData := NewDMClientProvisioningData()

	// And finally returning the Base64 encoded representation of the Provisioning Doc XML
	provDoc := NewProvisioningDoc(certStoreProvisioningData, appConfigProvisioningData, appDMClientProvisioningData)
	encodedProvDoc, err := provDoc.GetEncodedB64Representation()
	if err != nil {
		return "", err
	}

	return encodedProvDoc, nil
}

// storeWindowsMDMEnrolledDevice stores the device information to the list of MDM enrolled devices
func (svc *Service) storeWindowsMDMEnrolledDevice(ctx context.Context, secTokenMsg *fleet.RequestSecurityToken) error {
	const (
		error_tag = "windows MDM enrolled storage: "
	)

	// Getting the DeviceID context information from the RequestSecurityToken msg
	reqDeviceID, err := GetContextItem(secTokenMsg, mdm.ReqSecTokenContextItemDeviceID)
	if err != nil {
		return fmt.Errorf("%s %v", error_tag, err)
	}

	// Getting the HWDevID context information from the RequestSecurityToken msg
	reqHWDevID, err := GetContextItem(secTokenMsg, mdm.ReqSecTokenContextItemHWDevID)
	if err != nil {
		return fmt.Errorf("%s %v", error_tag, err)
	}

	// Getting the Enroll DeviceType context information from the RequestSecurityToken msg
	reqDeviceType, err := GetContextItem(secTokenMsg, mdm.ReqSecTokenContextItemDeviceType)
	if err != nil {
		return fmt.Errorf("%s %v", error_tag, err)
	}

	// Getting the Enroll DeviceName context information from the RequestSecurityToken msg
	reqDeviceName, err := GetContextItem(secTokenMsg, mdm.ReqSecTokenContextItemDeviceName)
	if err != nil {
		return fmt.Errorf("%s %v", error_tag, err)
	}

	// Getting the Enroll RequestVersion context information from the RequestSecurityToken msg
	reqEnrollVersion, err := GetContextItem(secTokenMsg, mdm.ReqSecTokenContextItemRequestVersion)
	if err != nil {
		return fmt.Errorf("%s %v", error_tag, err)
	}

	// Getting the RequestVersion context information from the RequestSecurityToken msg
	reqAppVersion, err := GetContextItem(secTokenMsg, mdm.ReqSecTokenContextItemApplicationVersion)
	if err != nil {
		return fmt.Errorf("%s %v", error_tag, err)
	}

	// Getting the EnrollmentType information from the RequestSecurityToken msg
	reqEnrollType, err := GetContextItem(secTokenMsg, mdm.ReqSecTokenContextItemEnrollmentType)
	if err != nil {
		return fmt.Errorf("%s %v", error_tag, err)
	}

	// Getting the Windows Enrolled Device Information
	enrolledDevice := &fleet.MDMWindowsEnrolledDevice{
		MDMDeviceID:            reqDeviceID,
		MDMHardwareID:          reqHWDevID,
		MDMDeviceState:         mdm.MDMDeviceStateEnrolled,
		MDMDeviceType:          reqDeviceType,
		MDMDeviceName:          reqDeviceName,
		MDMEnrollType:          reqEnrollType,
		MDMEnrollUserID:        "", // No user information is available at this point
		MDMEnrollProtoVersion:  reqEnrollVersion,
		MDMEnrollClientVersion: reqAppVersion,
		MDMNotInOOBE:           false,
	}

	if err := svc.ds.MDMWindowsInsertEnrolledDevice(ctx, enrolledDevice); err != nil {
		return err
	}

	// note: computer name is coming provisorily in place of `host_serial`
	// until we get other bits of the MDM protocol in place. This will be
	// updated to actually use the serial number in the future.
	err = svc.ds.NewActivity(ctx, nil, &fleet.ActivityTypeMDMEnrolled{
		HostSerial:  reqDeviceName,
		MDMPlatform: fleet.MDMPlatformMicrosoft,
	})
	if err != nil {
		// only logging, the device is enrolled at this point, and we
		// wouldn't want to fail the request because there was a problem
		// creating an activity feed item.
		logging.WithExtras(logging.WithNoUser(ctx),
			"msg", "failed to generate windows MDM enrolled activity",
		)
	}

	return nil
}

// GetContextItem returns the context item from the RequestSecurityToken message
func GetContextItem(secTokenMsg *fleet.RequestSecurityToken, contextItem string) (string, error) {
	reqHWDeviceID, err := secTokenMsg.GetContextItem(contextItem)
	if err != nil {
		return "", fmt.Errorf("%s token context information is not present: %v", contextItem, err)
	}

	return reqHWDeviceID, nil
}

// GetAuthorizedSoapFault authorize the request so SoapFault message can be returned
func (svc *Service) GetAuthorizedSoapFault(ctx context.Context, eType string, origMsg int, errorMsg error) *fleet.SoapFault {
	svc.authz.SkipAuthorization(ctx)

	soapFault := NewSoapFault(eType, origMsg, errorMsg)

	return &soapFault
}

func (svc *Service) SignMDMMicrosoftClientCSR(ctx context.Context, subject string, csr *x509.CertificateRequest) ([]byte, string, error) {
	cert, fpHex, err := svc.wstepCertManager.SignClientCSR(ctx, subject, csr)
	if err != nil {
		return nil, "signing wstep client csr", ctxerr.Wrap(ctx, err)
	}

	// TODO: if desired, the signature of this method can be modified to accept a device UUID so
	// that we can associate the certificate with the host here by calling
	// svc.wstepCertManager.AssociateCertHash

	return cert, fpHex, nil
}

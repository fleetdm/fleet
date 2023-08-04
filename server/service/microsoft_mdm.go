package service

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"

	mdm_types "github.com/fleetdm/fleet/v4/server/fleet"
	mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/google/uuid"
)

type SoapRequestContainer struct {
	Data   *fleet.SoapRequest
	Params url.Values
	Err    error
}

// MDM SOAP request decoder
func (req *SoapRequestContainer) DecodeBody(ctx context.Context, r io.Reader, u url.Values) error {
	// Reading the request bytes
	reqBytes, err := io.ReadAll(r)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "reading soap mdm request")
	}

	// Set the request parameters
	req.Params = u

	// Handle empty body scenario
	req.Data = &fleet.SoapRequest{}

	if len(reqBytes) != 0 {
		// Unmarshal the XML data from the request into the SoapRequest struct
		err = xml.Unmarshal(reqBytes, &req.Data)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "unmarshalling soap mdm request")
		}
	}

	return nil
}

type SoapResponseContainer struct {
	Data *fleet.SoapResponse
	Err  error
}

func (r SoapResponseContainer) error() error { return r.Err }

// hijackRender writes the response header and the RAW HTML output
func (r SoapResponseContainer) hijackRender(ctx context.Context, w http.ResponseWriter) {
	xmlRes, err := xml.MarshalIndent(r.Data, "", "\t")
	if err != nil {
		logging.WithExtras(ctx, "error with SoapResponseContainer", err)
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

type SyncMLReqMsgContainer struct {
	Data   *fleet.SyncMLMessage
	Params url.Values
	Err    error
}

// MDM SOAP request decoder
func (req *SyncMLReqMsgContainer) DecodeBody(ctx context.Context, r io.Reader, u url.Values) error {
	// Reading the request bytes
	reqBytes, err := io.ReadAll(r)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "reading SyncML message request")
	}

	// Set the request parameters
	req.Params = u

	// Handle empty body scenario
	req.Data = &fleet.SyncMLMessage{}

	if len(reqBytes) != 0 {
		// Unmarshal the XML data from the request into the SoapRequest struct
		err = xml.Unmarshal(reqBytes, &req.Data)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "unmarshalling SyncML message request")
		}
	}

	return nil
}

type SyncMLResponseMsgContainer struct {
	Data *string
	Err  error
}

func (r SyncMLResponseMsgContainer) error() error { return r.Err }

// hijackRender writes the response header and the RAW HTML output
func (r SyncMLResponseMsgContainer) hijackRender(ctx context.Context, w http.ResponseWriter) {
	resData := []byte(*r.Data + "\n")

	w.Header().Set("Content-Type", mdm.SyncMLContentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(resData)))
	w.WriteHeader(http.StatusOK)
	if n, err := w.Write(resData); err != nil {
		logging.WithExtras(ctx, "err", err, "written", n)
	}
}

type MDMWebContainer struct {
	Data   *string
	Params url.Values
	Err    error
}

// MDM SOAP request decoder
func (req *MDMWebContainer) DecodeBody(ctx context.Context, r io.Reader, u url.Values) error {
	reqBytes, err := io.ReadAll(r)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "reading Webcontainer HTML message request")
	}

	// Set the request parameters
	req.Params = u

	// Get req data
	content := string(reqBytes)
	req.Data = &content

	return nil
}

func (req MDMWebContainer) error() error { return req.Err }

// hijackRender writes the response header and the RAW HTML output
func (req MDMWebContainer) hijackRender(ctx context.Context, w http.ResponseWriter) {
	resData := []byte(*req.Data + "\n")

	w.Header().Set("Content-Type", mdm.WebContainerContentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(resData)))
	w.WriteHeader(http.StatusOK)
	if n, err := w.Write(resData); err != nil {
		logging.WithExtras(ctx, "err", err, "written", n)
	}
}

type MDMAuthContainer struct {
	Data *string
	Err  error
}

func (r MDMAuthContainer) error() error { return r.Err }

// hijackRender writes the response header and the RAW XML output
func (r MDMAuthContainer) hijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(*r.Data)))
	w.WriteHeader(http.StatusOK)
	if n, err := w.Write([]byte(*r.Data)); err != nil {
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

// getSTSAuthContent Retuns STS auth content
func getSTSAuthContent(data string) errorer {
	return MDMAuthContainer{
		Data: &data,
		Err:  nil,
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

// GetEncodedBinarySecurityToken returns the base64 form of a input payload
func GetEncodedBinarySecurityToken(typeID fleet.WindowsMDMEnrollmentType, payload string) (string, error) {
	var pld fleet.WindowsMDMAccessTokenPayload
	pld.Type = typeID

	if typeID == fleet.WindowsMDMProgrammaticEnrollmentType {
		pld.Payload.HostUUID = payload
	} else if typeID == fleet.WindowsMDMAutomaticEnrollmentType {
		pld.Payload.AuthToken = payload
	} else {
		return "", fmt.Errorf("invalid enrollment type: %v", typeID)
	}

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
	discoveryResponseMsg, err := svc.GetMDMMicrosoftDiscoveryResponse(ctx, req.Body.Discover.Request.EmailAddress)
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

// mdmMicrosoftAuthEndpoint handles the Security Token Service (STS) implementation
func mdmMicrosoftAuthEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	params := request.(*SoapRequestContainer).Params

	// Sanity check on the expected query params
	if !params.Has(mdm.STSAuthAppRu) || !params.Has(mdm.STSLoginHint) {
		return getSTSAuthContent(""), errors.New("expected STS params are not present")
	}

	appru := params.Get(mdm.STSAuthAppRu)
	loginHint := params.Get(mdm.STSLoginHint)

	if (len(appru) == 0) || (len(loginHint) == 0) {
		return getSTSAuthContent(""), errors.New("expected STS params are empty")
	}

	// Getting the STS endpoint HTML content
	stsAuthContent, err := svc.GetMDMMicrosoftSTSAuthResponse(ctx, appru, loginHint)
	if err != nil {
		return getSTSAuthContent(""), errors.New("error generating STS content")
	}

	return getSTSAuthContent(stsAuthContent), nil
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
	hdrSecToken, err := req.GetHeaderBinarySecurityToken()
	if err != nil {
		soapFault := svc.GetAuthorizedSoapFault(ctx, mdm.SoapErrorMessageFormat, mdm_types.MDEPolicy, err)
		return getSoapResponseFault(req.GetMessageID(), soapFault), nil
	}

	// Getting the GetPoliciesResponse message
	policyResponseMsg, err := svc.GetMDMWindowsPolicyResponse(ctx, hdrSecToken)
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
	hdrBinarySecToken, err := req.GetHeaderBinarySecurityToken()
	if err != nil {
		soapFault := svc.GetAuthorizedSoapFault(ctx, mdm.SoapErrorMessageFormat, mdm_types.MDEEnrollment, err)
		return getSoapResponseFault(req.GetMessageID(), soapFault), nil
	}

	// Getting the RequestSecurityTokenResponseCollection message
	enrollResponseMsg, err := svc.GetMDMWindowsEnrollResponse(ctx, reqSecurityTokenMsg, hdrBinarySecToken)
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

// mdmMicrosoftManagementEndpoint handles the OMA DM management sessions
// It receives a SyncML message with protocol commands, it process the commands and responds with a
// SyncML message with protocol commands results and more protocol commands for the calling host
// Note: This logic needs to be improved with better SyncML message parsing, better message tracking
// and better security authentication (done through TLS and in-message hash)
func mdmMicrosoftManagementEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	reqSyncML := request.(*SyncMLReqMsgContainer).Data

	// Checking first if incoming SyncML message is valid and returning error if this is not the case
	if err := reqSyncML.IsValidSyncMLMsg(); err != nil {
		soapFault := svc.GetAuthorizedSoapFault(ctx, mdm.SoapErrorMessageFormat, mdm_types.MDEFault, err)
		return getSoapResponseFault(strconv.Itoa(reqSyncML.Header.MsgID), soapFault), nil
	}

	// Getting the RequestSecurityTokenResponseCollection message
	resSyncML, err := svc.GetMDMWindowsManagementResponse(ctx, reqSyncML)
	if err != nil {
		soapFault := svc.GetAuthorizedSoapFault(ctx, mdm.SoapErrorMessageFormat, mdm_types.MDEEnrollment, err)
		return getSoapResponseFault(strconv.Itoa(reqSyncML.Header.MsgID), soapFault), nil
	}

	return SyncMLResponseMsgContainer{
		Data: resSyncML,
		Err:  nil,
	}, nil
}

// mdmMicrosoftTOSEndpoint handles the TOS content for the incoming MDM enrollment request
func mdmMicrosoftTOSEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	params := request.(*MDMWebContainer).Params

	// Sanity check on the expected query params
	if !params.Has(mdm.TOCRedirectURI) || !params.Has(mdm.TOCReqID) {
		soapFault := svc.GetAuthorizedSoapFault(ctx, mdm.SoapErrorMessageFormat, mdm_types.MDEEnrollment, errors.New("invalid params"))
		return getSoapResponseFault(mdm.SoapErrorInternalServiceFault, soapFault), nil
	}

	redirectURI := params.Get(mdm.TOCRedirectURI)
	reqID := params.Get(mdm.TOCReqID)

	// Getting the TOS content message
	resTOCData, err := svc.GetMDMWindowsTOSContent(ctx, redirectURI, reqID)
	if err != nil {
		soapFault := svc.GetAuthorizedSoapFault(ctx, mdm.SoapErrorMessageFormat, mdm_types.MDEEnrollment, err)
		return getSoapResponseFault(mdm.SoapErrorInternalServiceFault, soapFault), nil
	}

	return MDMWebContainer{
		Data: &resTOCData,
		Err:  nil,
	}, nil
}

// authBinarySecurityToken checks if the provided token is valid
func (svc *Service) authBinarySecurityToken(ctx context.Context, authToken *fleet.HeaderBinarySecurityToken) (string, error) {
	if authToken == nil {
		return "", errors.New("authToken is empty")
	}

	err := authToken.IsValidToken()
	if err != nil {
		return "", errors.New("authToken is not valid")
	}

	// Tokens that were generated by enrollment client
	if authToken.IsDeviceToken() {

		// Getting the Binary Security Token Payload
		binSecToken, err := NewBinarySecurityTokenPayload(authToken.Content)
		if err != nil {
			return "", fmt.Errorf("token creation error %v", err)
		}

		// Validating the Binary Security Token Payload
		err = binSecToken.IsValidToken()
		if err != nil {
			return "", fmt.Errorf("invalid token data %v", err)
		}

		// Validating the Binary Security Token Type used on Programmatic Enrollments
		if binSecToken.Type == mdm_types.WindowsMDMProgrammaticEnrollmentType {
			host, err := svc.ds.HostByIdentifier(ctx, binSecToken.Payload.HostUUID)
			if err != nil {
				return "", fmt.Errorf("host data cannot be found %v", err)
			}

			// This ensures that only hosts that are eligible for Windows enrollment can be enrolled
			if !host.IsEligibleForWindowsMDMEnrollment() {
				return "", errors.New("host is not elegible for Windows MDM enrollment")
			}

			// No errors, token is authorized
			return binSecToken.Payload.HostUUID, nil
		}

		// Validating the Binary Security Token Type used on Automatic Enrollments (returned by STS Auth Endpoint)
		if binSecToken.Type == mdm_types.WindowsMDMAutomaticEnrollmentType {

			upnToken, err := svc.wstepCertManager.GetSTSAuthTokenUPNClaim(binSecToken.Payload.AuthToken)
			if err != nil {
				return "", ctxerr.Wrap(ctx, err, "issue retrieving UPN from Auth token")
			}

			// No errors, token is authorized
			return upnToken, nil
		}
	}

	// Validating the Binary Security Token Type used on Automatic Enrollments
	if authToken.IsAzureJWTToken() {

		// Validate the JWT Auth token by retreving its claims
		tokenData, err := mdm.GetAzureAuthTokenClaims(authToken.Content)
		if err != nil {
			return "", fmt.Errorf("binary security token claim failed: %v", err)
		}

		// No errors, token is authorized
		return tokenData.UPN, nil
	}

	return "", errors.New("token is not authorized")
}

// GetMDMMicrosoftDiscoveryResponse returns a valid DiscoveryResponse message
func (svc *Service) GetMDMMicrosoftDiscoveryResponse(ctx context.Context, upnEmail string) (*fleet.DiscoverResponse, error) {
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

// GetMDMMicrosoftSTSAuthResponse returns a valid Security Token Service (STS) page content
func (svc *Service) GetMDMMicrosoftSTSAuthResponse(ctx context.Context, appru string, loginHint string) (string, error) {
	// skipauth: This endpoint does not use authentication
	svc.authz.SkipAuthorization(ctx)

	// Dummy data will be returned as part of the token as user-driven enrollment is not supported yet
	// In the future, the following calls would have to be made to support user-driven enrollment
	// encodedBST will carry the token to return
	// authToken, err := svc.wstepCertManager.NewSTSAuthToken(loginHint)
	// encodedBST, err := GetEncodedBinarySecurityToken(fleet.WindowsMDMAutomaticEnrollmentType, authToken)
	encodedBST := "user_driven_enrollment_not_implemented"

	// STS Auth Endpoint returns HTML content that gets render in a webview container
	// The webview container expect a POST request to the appru URL with the wresult parameter set to the auth token
	// The security token in wresult is later passed back in <wsse:BinarySecurityToken>
	// This string is opaque to the enrollment client; the client does not interpret the string.
	// The returned HTML content contains a JS script that will perform a POST request to the appru URL automatically
	// This will set the wresult parameter to the value of auth token
	tmpl, err := template.New("").Parse(`
				<script>
				function performPost() {
				  // Dinamically create a form element to submit the request
				  var form = document.createElement('form');
				  form.method = 'POST';
				  form.action = "{{.ActionURL}}"

				  var inputToken = document.createElement('input');
				  inputToken.type = 'hidden';
				  inputToken.name = 'wresult';
				  inputToken.value = '{{.Token}}';
				  form.appendChild(inputToken);

				  // Submit the form
				  document.body.appendChild(form);
				  form.submit();
				}

				// Call performPost() when the script is executed
				performPost();
				</script>
				`)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "STS content template")
	}

	var htmlBuf bytes.Buffer
	err = tmpl.Execute(&htmlBuf, map[string]string{"ActionURL": appru, "Token": encodedBST})
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "creation of STS content")
	}

	return htmlBuf.String(), nil
}

// GetMDMWindowsPolicyResponse returns a valid GetPoliciesResponse message
func (svc *Service) GetMDMWindowsPolicyResponse(ctx context.Context, authToken *fleet.HeaderBinarySecurityToken) (*fleet.GetPoliciesResponse, error) {
	if authToken == nil {
		return nil, fleet.NewInvalidArgumentError("policy response", "authToken is invalid")
	}

	// Validate the binary security token
	_, err := svc.authBinarySecurityToken(ctx, authToken)
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
func (svc *Service) GetMDMWindowsEnrollResponse(ctx context.Context, secTokenMsg *fleet.RequestSecurityToken, authToken *fleet.HeaderBinarySecurityToken) (*fleet.RequestSecurityTokenResponseCollection, error) {
	if authToken == nil {
		return nil, fleet.NewInvalidArgumentError("enroll response", "authToken is not present")
	}

	// Auth the binary security token
	userID, err := svc.authBinarySecurityToken(ctx, authToken)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "validate binary security token")
	}

	// Removing the device if already MDM enrolled
	err = svc.removeWindowsDeviceIfAlreadyMDMEnrolled(ctx, secTokenMsg)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "device enroll check")
	}

	// Getting the device provisioning information in the form of a WapProvisioningDoc
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

	// RequestSecurityTokenResponseCollection message is ready. The identity
	// and provisioning information will be sent to the Windows MDM
	// Enrollment Client

	// But before doing that, let's save the device information to the list
	// of MDM enrolled MDM devices
	//
	// This method also creates the relevant enrollment activity as it has
	// access to the device information.
	err = svc.storeWindowsMDMEnrolledDevice(ctx, userID, secTokenMsg)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "enrolled device information cannot be stored")
	}

	return &secTokenResponseCollectionMsg, nil
}

// GetMDMWindowsManagementResponse returns a valid SyncML response message
func (svc *Service) GetMDMWindowsManagementResponse(ctx context.Context, reqSyncML *fleet.SyncMLMessage) (*string, error) {
	if reqSyncML == nil {
		return nil, fleet.NewInvalidArgumentError("syncml req message", "message is not present")
	}

	// TODO: The following logic should happen here
	// - TLS based auth
	// - Device auth based on Source/LocURI DeviceID information
	//   (this should be present on Enrollment DB)
	// - Processing of incoming protocol commands (Alerts mostly
	// - MS-MDM session management
	// - Inclusion of queued protocol commands should be performed here
	// - Tracking of message acknowledgements through Message queue

	// Getting the management response message
	resSyncMLmsg, err := svc.getManagementResponse(ctx, reqSyncML)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "device provisioning information")
	}

	// Token is authorized
	svc.authz.SkipAuthorization(ctx)

	return resSyncMLmsg, nil
}

// GetMDMWindowsTOSContent returns valid TOC content
func (svc *Service) GetMDMWindowsTOSContent(ctx context.Context, redirectUri string, reqID string) (string, error) {
	tmpl, err := template.New("").Parse(`
	<html>
	<head>
	<style>
	  button {
		background-color: #008CBA;
		color: white;
		padding: 10px 60px;
		border: none;
		cursor: pointer;
	  }
	</style>
	</head>
	<body>
	  <center>
		<img src="https://fleetdm.com/images/logo-blue-162x92@2x.png">
		<br>
		<h2>Terms and conditions</h2>
		<br> Terms and Conditions PDF content should go here <center>
		  <br>
		  <button type="button" onClick="acceptBtn()">Accept</button>
		  <script>
			function acceptBtn() {
			  window.location = "{{.RedirectURL}}" + "?IsAccepted=true&OpaqueBlob={{.ClientData}}";
			}
		  </script>
	</body>
	</html>`)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "issue generating TOS content")
	}

	var htmlBuf bytes.Buffer
	err = tmpl.Execute(&htmlBuf, map[string]string{"RedirectURL": redirectUri, "ClientData": reqID})
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "executing TOS template content")
	}

	// skipauth: This endpoint does not use authentication
	svc.authz.SkipAuthorization(ctx)

	return htmlBuf.String(), nil
}

func (svc *Service) getManagementResponse(ctx context.Context, reqSyncML *fleet.SyncMLMessage) (*string, error) {
	if reqSyncML == nil {
		return nil, fleet.NewInvalidArgumentError("syncml req message", "message is not present")
	}

	// cmdID tracks the command sequence
	cmdID := 0

	// Retrieve the MessageID from the syncml req body
	deviceID := reqSyncML.Header.Source

	// Retrieve the sessionID from the syncml req body
	sessionID := reqSyncML.Header.SessionID

	// Retrieve the msgID from the syncml req body
	msgID := reqSyncML.Header.MsgID

	// Getting the management URL message content
	appCfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	urlManagementEndpoint, err := mdm.ResolveWindowsMDMManagement(appCfg.ServerSettings.ServerURL)
	if err != nil {
		return nil, err
	}

	// Checking the SyncML message types
	var response string
	if isSessionInitializationMessage(reqSyncML.Body) {
		// Create response payload - MDM SyncML configuration profiles commands will be enforced here
		response = `
			<?xml version="1.0" encoding="UTF-8"?>
			<SyncML xmlns="SYNCML:SYNCML1.2">
				<SyncHdr>
					<VerDTD>1.2</VerDTD>
					<VerProto>DM/1.2</VerProto>
					<SessionID>` + strconv.Itoa(sessionID) + `</SessionID>
					<MsgID>` + strconv.Itoa(msgID) + `</MsgID>
					<Target>
						<LocURI>` + deviceID + `</LocURI>
					</Target>
					<Source>
						<LocURI>` + urlManagementEndpoint + `</LocURI>
					</Source>
				</SyncHdr>
				<SyncBody>
					<Status>
						<CmdID>` + getNextCmdID(&cmdID) + `</CmdID>
						<MsgRef>` + strconv.Itoa(msgID) + `</MsgRef>
						<CmdRef>0</CmdRef>
						<Cmd>SyncHdr</Cmd>
						<Data>200</Data>
					</Status>
					<Status>
						<CmdID>` + getNextCmdID(&cmdID) + `</CmdID>
						<MsgRef>` + strconv.Itoa(msgID) + `</MsgRef>
						<CmdRef>2</CmdRef>
						<Cmd>Alert</Cmd>
						<Data>200</Data>
					</Status>
					<Status>
						<CmdID>` + getNextCmdID(&cmdID) + `</CmdID>
						<MsgRef>` + strconv.Itoa(msgID) + `</MsgRef>
						<CmdRef>3</CmdRef>
						<Cmd>Alert</Cmd>
						<Data>200</Data>
					</Status>
					<Status>
						<CmdID>` + getNextCmdID(&cmdID) + `</CmdID>
						<MsgRef>` + strconv.Itoa(msgID) + `</MsgRef>
						<CmdRef>4</CmdRef>
						<Cmd>Replace</Cmd>
						<Data>200</Data>
					</Status>
					` + svc.getConfigProfilesToEnforce(ctx, &cmdID) + `
					<Final />
				</SyncBody>
			</SyncML>`
	} else {
		// Acknowledge SyncML messages sent by host
		response = `
			<?xml version="1.0" encoding="UTF-8"?>
			<SyncML xmlns="SYNCML:SYNCML1.2">
				<SyncHdr>
					<VerDTD>1.2</VerDTD>
					<VerProto>DM/1.2</VerProto>
					<SessionID>` + strconv.Itoa(sessionID) + `</SessionID>
					<MsgID>` + strconv.Itoa(msgID) + `</MsgID>
					<Target>
						<LocURI>` + deviceID + `</LocURI>
					</Target>
					<Source>
						<LocURI>` + urlManagementEndpoint + `</LocURI>
					</Source>
				</SyncHdr>
				<SyncBody>
					<Status>
						<CmdID>` + getNextCmdID(&cmdID) + `</CmdID>
						<MsgRef>` + strconv.Itoa(msgID) + `</MsgRef>
						<CmdRef>0</CmdRef>
						<Cmd>SyncHdr</Cmd>
						<Data>200</Data>
					</Status>
					<Final />
				</SyncBody>
			</SyncML>`
	}

	// Create a replacer to replace both "\n" and "\t"
	replacer := strings.NewReplacer("\n", "", "\t", "")

	// Use the replacer on the string representation of xmlContent
	responseRaw := replacer.Replace(response)

	return &responseRaw, nil
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
func (svc *Service) storeWindowsMDMEnrolledDevice(ctx context.Context, userID string, secTokenMsg *fleet.RequestSecurityToken) error {
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
		reqEnrollVersion = "request_version_not_present"
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
		MDMEnrollUserID:        userID, // This could be Host UUID or UPN email
		MDMEnrollProtoVersion:  reqEnrollVersion,
		MDMEnrollClientVersion: reqAppVersion,
		MDMNotInOOBE:           false,
	}

	if err := svc.ds.MDMWindowsInsertEnrolledDevice(ctx, enrolledDevice); err != nil {
		return err
	}

	err = svc.ds.NewActivity(ctx, nil, &fleet.ActivityTypeMDMEnrolled{
		HostDisplayName: reqDeviceName,
		MDMPlatform:     fleet.MDMPlatformMicrosoft,
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
	if svc.wstepCertManager == nil {
		return nil, "", errors.New("windows mdm identity keypair was not configured")
	}

	cert, fpHex, err := svc.wstepCertManager.SignClientCSR(ctx, subject, csr)
	if err != nil {
		return nil, "signing wstep client csr", ctxerr.Wrap(ctx, err)
	}

	// TODO: if desired, the signature of this method can be modified to accept a device UUID so
	// that we can associate the certificate with the host here by calling
	// svc.wstepCertManager.AssociateCertHash

	return cert, fpHex, nil
}

func (svc *Service) getConfigProfilesToEnforce(ctx context.Context, commandID *int) string {
	// Getting the management URL
	appCfg, _ := svc.ds.AppConfig(ctx)
	fleetEnrollUrl := appCfg.ServerSettings.ServerURL

	// Getting the global enrollment secret
	var globalEnrollSecret string
	secrets, err := svc.ds.GetEnrollSecrets(ctx, nil)
	if err != nil {
		return ""
	}

	for _, secret := range secrets {
		if secret.TeamID == nil {
			globalEnrollSecret = secret.Secret
			break
		}
	}

	// keeping the same GUID will prevent the MSI to be installed multiple times - it will be
	// installed only the first time the message is issued.
	// FleetURL and FleetSecret properties are passed to the Fleet MSI
	// See here for more information: https://learn.microsoft.com/en-us/windows/win32/msi/command-line-options
	installCommandPayload := `<MsiInstallJob id="{A427C0AA-E2D5-40DF-ACE8-0D726A6BE096}">
					<Product Version="1.0.0.0">
						<Download>
							<ContentURLList>
								<ContentURL>https://download.fleetdm.com/fleetd-base.msi</ContentURL>
							</ContentURLList>
						</Download>
						<Validation>
							<FileHash>9F89C57D1B34800480B38BD96186106EB6418A82B137A0D56694BF6FFA4DDF1A</FileHash>
						</Validation>
						<Enforcement>
							<CommandLine>/quiet FLEET_URL="` + fleetEnrollUrl + `" FLEET_SECRET="` + globalEnrollSecret + `"</CommandLine>
							<TimeOut>10</TimeOut>
							<RetryCount>1</RetryCount>
							<RetryInterval>5</RetryInterval>
						</Enforcement>
					</Product>
				</MsiInstallJob>`

	newCmds := `<Add>
				<CmdID>` + getNextCmdID(commandID) + `</CmdID>
				<Item>
					<Target>
					<LocURI>./Device/Vendor/MSFT/EnterpriseDesktopAppManagement/MSI/%7BA427C0AA-E2D5-40DF-ACE8-0D726A6BE096%7D/DownloadInstall</LocURI>
					</Target>
				</Item>
				</Add>
				<Exec>
				<CmdID>` + getNextCmdID(commandID) + `</CmdID>
				<Item>
					<Target>
					<LocURI>./Device/Vendor/MSFT/EnterpriseDesktopAppManagement/MSI/%7BA427C0AA-E2D5-40DF-ACE8-0D726A6BE096%7D/DownloadInstall</LocURI>
					</Target>
					<Data>` + html.EscapeString(installCommandPayload) + `</Data>
					<Meta>
					<Type xmlns="syncml:metinf">text/plain</Type>
					<Format xmlns="syncml:metinf">xml</Format>
					</Meta>
				</Item>
				</Exec>`

	return newCmds
}

// getNextCmdID returns the next command ID
func getNextCmdID(i *int) string {
	*i++
	return strconv.Itoa(*i)
}

// Checks if body contains a DM device unrollment SyncML message
func isDeviceUnenrollmentMessage(body fleet.SyncMLBody) bool {
	for _, element := range body.Item {
		if element.Data == mdm.DeviceUnenrollmentID {
			return true
		}
	}

	return false
}

// Checks if body contains a DM session initialization SyncML message sent by device
func isSessionInitializationMessage(body fleet.SyncMLBody) bool {
	isUnenrollMessage := isDeviceUnenrollmentMessage(body)

	for _, element := range body.Item {
		if element.Data == mdm.HostInitMessageID && !isUnenrollMessage {
			return true
		}
	}

	return false
}

package service

import (
	"bytes"
	"context"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleetdbase"
	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"
	mdmlifecycle "github.com/fleetdm/fleet/v4/server/mdm/lifecycle"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/fleetdm/fleet/v4/server/mdm/microsoft/syncml"
	"github.com/fleetdm/fleet/v4/server/ptr"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"

	mdm_types "github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
)

type SoapRequestContainer struct {
	Data   *fleet.SoapRequest
	Params url.Values
	Err    error
}

// MDM SOAP request decoder
func (req *SoapRequestContainer) DecodeBody(ctx context.Context, r io.Reader, u url.Values, c []*x509.Certificate) error {
	// Reading the request bytes
	reqBytes, err := io.ReadAll(r)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "reading soap mdm request")
	}

	// Set the request parameters
	req.Params = u

	// Handle empty body scenario
	req.Data = &fleet.SoapRequest{Raw: reqBytes}

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

	w.Header().Set("Content-Type", syncml.SoapContentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(xmlRes)))
	w.WriteHeader(http.StatusOK)
	if n, err := w.Write(xmlRes); err != nil {
		logging.WithExtras(ctx, "err", err, "written", n)
	}
}

type SyncMLReqMsgContainer struct {
	Data   *fleet.SyncML
	Params url.Values
	Certs  []*x509.Certificate
	Err    error
}

// MDM SOAP request decoder
func (req *SyncMLReqMsgContainer) DecodeBody(ctx context.Context, r io.Reader, u url.Values, c []*x509.Certificate) error {
	// Reading the request bytes
	reqBytes, err := io.ReadAll(r)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "reading SyncML message request")
	}

	// Set the request parameters
	req.Params = u

	// Set the request certs
	req.Certs = c

	// Handle empty body scenario
	req.Data = &fleet.SyncML{Raw: reqBytes}

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
	Data *fleet.SyncML
	Err  error
}

func (r SyncMLResponseMsgContainer) error() error { return r.Err }

// hijackRender writes the response header and the RAW HTML output
func (r SyncMLResponseMsgContainer) hijackRender(ctx context.Context, w http.ResponseWriter) {
	xmlRes, err := xml.MarshalIndent(r.Data, "", "\t")
	if err != nil {
		logging.WithExtras(ctx, "error with SyncMLResponseMsgContainer", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	xmlRes = append(xmlRes, '\n')

	w.Header().Set("Content-Type", syncml.SyncMLContentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(xmlRes)))
	w.WriteHeader(http.StatusOK)
	if n, err := w.Write(xmlRes); err != nil {
		logging.WithExtras(ctx, "err", err, "written", n)
	}
}

type MDMWebContainer struct {
	Data   *string
	Params url.Values
	Err    error
}

// MDM SOAP request decoder
func (req *MDMWebContainer) DecodeBody(ctx context.Context, r io.Reader, u url.Values, c []*x509.Certificate) error {
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

	w.Header().Set("Content-Type", syncml.WebContainerContentType)
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
		XMLNS: syncml.DiscoverNS,
		DiscoverResult: mdm_types.DiscoverResult{
			AuthPolicy:                 authPolicy,
			EnrollmentVersion:          syncml.EnrollmentVersionV4,
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
		XMLNS: syncml.PolicyNS,
		Response: mdm_types.Response{
			PolicyFriendlyName: mdm_types.ContentAttr{
				Xsi:   syncml.DefaultStateXSI,
				XMLNS: syncml.EnrollXSI,
			},
			NextUpdateHours: mdm_types.ContentAttr{
				Xsi:   syncml.DefaultStateXSI,
				XMLNS: syncml.EnrollXSI,
			},
			PoliciesNotChanged: mdm_types.ContentAttr{
				Xsi:   syncml.DefaultStateXSI,
				XMLNS: syncml.EnrollXSI,
			},
			Policies: mdm_types.Policies{
				Policy: mdm_types.GPPolicy{
					PolicyOIDReference: "0",
					CAs: mdm_types.GenericAttr{
						Xsi: syncml.DefaultStateXSI,
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
							Xsi: syncml.DefaultStateXSI,
						},
						PrivateKeyFlags: mdm_types.GenericAttr{
							Xsi: syncml.DefaultStateXSI,
						},
						SubjectNameFlags: mdm_types.GenericAttr{
							Xsi: syncml.DefaultStateXSI,
						},
						EnrollmentFlags: mdm_types.GenericAttr{
							Xsi: syncml.DefaultStateXSI,
						},
						GeneralFlags: mdm_types.GenericAttr{
							Xsi: syncml.DefaultStateXSI,
						},
						RARequirements: mdm_types.GenericAttr{
							Xsi: syncml.DefaultStateXSI,
						},
						KeyArchivalAttributes: mdm_types.GenericAttr{
							Xsi: syncml.DefaultStateXSI,
						},
						Extensions: mdm_types.GenericAttr{
							Xsi: syncml.DefaultStateXSI,
						},
						PrivateKeyAttributes: mdm_types.PrivateKeyAttributes{
							MinimalKeyLength: minimalKeyLength,
							KeySpec: mdm_types.GenericAttr{
								Xsi: syncml.DefaultStateXSI,
							},
							KeyUsageProperty: mdm_types.GenericAttr{
								Xsi: syncml.DefaultStateXSI,
							},
							Permissions: mdm_types.GenericAttr{
								Xsi: syncml.DefaultStateXSI,
							},
							AlgorithmOIDReference: mdm_types.GenericAttr{
								Xsi: syncml.DefaultStateXSI,
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

	enrollSecExtVal := syncml.EnrollSecExt
	return mdm_types.RequestSecurityTokenResponseCollection{
		XMLNS: syncml.EnrollWSTrust,
		RequestSecurityTokenResponse: mdm_types.RequestSecurityTokenResponse{
			TokenType: syncml.EnrollTType,
			DispositionMessage: mdm_types.SecAttr{
				Content: "",
				XMLNS:   syncml.EnrollReq,
			},
			RequestID: mdm_types.SecAttr{
				Content: "0",
				XMLNS:   syncml.EnrollReq,
			},
			RequestedSecurityToken: mdm_types.RequestedSecurityToken{
				BinarySecurityToken: mdm_types.BinarySecurityToken{
					Content:      provisionedToken,
					XMLNS:        &enrollSecExtVal,
					ValueType:    syncml.EnrollPDoc,
					EncodingType: syncml.EnrollEncode,
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
			Value: syncml.SoapFaultRecv,
			Subcode: mdm_types.Subcode{
				Value: errorType,
			},
		},
		Reason: mdm_types.Reason{
			Text: mdm_types.ReasonText{
				Content: errorMessage.Error(),
				Lang:    syncml.SoapFaultLocale,
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
		urlNSS                = syncml.EnrollNSS
		urlNSA                = syncml.EnrollNSA
		urlXSI                = syncml.EnrollXSI
		urlXSD                = syncml.EnrollXSD
		urlXSU                = syncml.EnrollXSU
		urlDiag               = syncml.ActionNsDiag
		urlDiscovery          = syncml.ActionNsDiscovery
		urlPolicy             = syncml.ActionNsPolicy
		urlEnroll             = syncml.ActionNsEnroll
		urlSecExt             = syncml.EnrollSecExt
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
		if msg.OriginalMessageType == mdm_types.MDEDiscovery { //nolint:gocritic // ignore ifElseChain
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
				newParm("ROBOSupport", syncml.WstepROBOSupport, "boolean"),
				newParm("RenewPeriod", syncml.WstepCertRenewalPeriodInDays, "integer"),
				newParm("RetryInterval", syncml.WstepRenewRetryInterval, "integer"),
			}, nil),
		}),
	})

	certStore := newCharacteristic("CertificateStore", nil, []mdm_types.Characteristic{root, my})
	return certStore
}

// isEligibleForWindowsMDMEnrollment returns true if the host can be enrolled
// in Fleet's Windows MDM (if it was enabled).
func isEligibleForWindowsMDMEnrollment(host *fleet.Host, mdmInfo *fleet.HostMDM) bool {
	return host.FleetPlatform() == "windows" &&
		host.IsOsqueryEnrolled() &&
		(mdmInfo == nil || (!mdmInfo.IsServer && !mdmInfo.Enrolled))
}

// isEligibleForWindowsMDMMigration returns true if the host can be migrated to
// Fleet's Windows MDM (if it was enabled).
func isEligibleForWindowsMDMMigration(host *fleet.Host, mdmInfo *fleet.HostMDM) bool {
	return host.FleetPlatform() == "windows" &&
		host.IsOsqueryEnrolled() &&
		(mdmInfo != nil && !mdmInfo.IsServer && mdmInfo.Enrolled && mdmInfo.Name != fleet.WellKnownMDMFleet)
}

// NewApplicationProvisioningData returns a new ApplicationProvisioningData Characteristic
// The Application Provisioning configuration is used for bootstrapping a device with an OMA DM account
// The paramenters here maps to the W7 application CSP
// https://learn.microsoft.com/en-us/windows/client-management/mdm/w7-application-csp
func NewApplicationProvisioningData(mdmEndpoint string) mdm_types.Characteristic {
	provDoc := newCharacteristic("APPLICATION", []mdm_types.Param{
		// The PROVIDER-ID parameter specifies the server identifier for a management server used in the current management session
		newParm("PROVIDER-ID", syncml.DocProvisioningAppProviderID, ""),

		// The APPID parameter is used to differentiate the types of available application services and protocols.
		newParm("APPID", "w7", ""),

		// The NAME parameter is used in the APPLICATION characteristic to specify a user readable application identity.
		newParm("NAME", syncml.DocProvisioningAppName, ""),

		// The ADDR parameter is used in the APPADDR param to get or set the address of the OMA DM server.
		newParm("ADDR", mdmEndpoint, ""),

		// The ROLE parameter is used in the APPLICATION characteristic to specify the security application chamber that the DM session should run with when communicating with the DM server.

		// The BACKCOMPATRETRYFREQ parameter is used  to specify how many retries the DM client performs when there are Connection Manager-level or WinInet-level errors
		newParm("CONNRETRYFREQ", syncml.DocProvisioningAppConnRetryFreq, ""),

		// The INITIALBACKOFFTIME parameter is used to specify the initial wait time in milliseconds when the DM client retries for the first time
		newParm("INITIALBACKOFFTIME", syncml.DocProvisioningAppInitialBackoffTime, ""),

		// The MAXBACKOFFTIME parameter is used to specify the maximum number of milliseconds to sleep after package-sending failure
		newParm("MAXBACKOFFTIME", syncml.DocProvisioningAppMaxBackoffTime, ""),

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
			newCharacteristic(syncml.DocProvisioningAppProviderID,
				[]mdm_types.Param{}, []mdm_types.Characteristic{
					newCharacteristic("Poll", []mdm_types.Param{
						// AllUsersPollOnFirstLogin - enabled
						// https://learn.microsoft.com/en-us/windows/client-management/mdm/dmclient-csp#deviceproviderprovideridpollalluserspollonfirstlogin
						newParm("AllUsersPollOnFirstLogin", "true", syncml.DmClientBoolType),

						// PollOnLogin - enabled
						// https://learn.microsoft.com/en-us/windows/client-management/mdm/dmclient-csp#deviceproviderprovideridpollpollonlogin
						newParm("PollOnLogin", "true", syncml.DmClientBoolType),

						// NumberOfFirstRetries - 0 (meaning repeat infinitely, Second and Remaining retries will not be used)
						// https://learn.microsoft.com/en-us/windows/client-management/mdm/dmclient-csp#deviceproviderprovideridpollnumberoffirstretries
						//
						// Note that the docs do mention:
						//
						//   The total time for first set of retries shouldn't be more than
						//   a few hours. The server shouldn't set NumberOfFirstRetries to
						//   be 0. RemainingScheduledRetries is used for the long run
						//   device polling schedule.
						//
						// but we really want to keep polling regularly at short intervals
						// and it seems like the way to do it (and they do support infinite
						// retries, so...).
						newParm("NumberOfFirstRetries", "0", syncml.DmClientIntType),
						// IntervalForFirstSetOfRetries - 1 minute (we can't go lower than that)
						// https://learn.microsoft.com/en-us/windows/client-management/mdm/dmclient-csp#deviceproviderprovideridpollintervalforfirstsetofretries
						newParm("IntervalForFirstSetOfRetries", "1", syncml.DmClientIntType),

						// Second and Remaining retries are disabled (0).
						newParm("NumberOfSecondRetries", "0", syncml.DmClientIntType),
						newParm("IntervalForSecondSetOfRetries", "0", syncml.DmClientIntType),
						newParm("NumberOfRemainingScheduledRetries", "0", syncml.DmClientIntType),
						newParm("IntervalForRemainingScheduledRetries", "0", syncml.DmClientIntType),
					}, nil),
				}),
		}),
	})

	return dmClient
}

// NewProvisioningDoc returns a new ProvisioningDoc container
func NewProvisioningDoc(certStoreData mdm_types.Characteristic, applicationData mdm_types.Characteristic, dmClientData mdm_types.Characteristic) mdm_types.WapProvisioningDoc {
	return mdm_types.WapProvisioningDoc{
		Version: syncml.DocProvisioningVersion,
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
		soapFault := svc.GetAuthorizedSoapFault(ctx, syncml.SoapErrorMessageFormat, mdm_types.MDEDiscovery, err)
		return getSoapResponseFault(req.GetMessageID(), soapFault), nil
	}

	// Getting the DiscoveryResponse message
	discoveryResponseMsg, err := svc.GetMDMMicrosoftDiscoveryResponse(ctx, req.Body.Discover.Request.EmailAddress)
	if err != nil {
		soapFault := svc.GetAuthorizedSoapFault(ctx, syncml.SoapErrorMessageFormat, mdm_types.MDEDiscovery, err)
		return getSoapResponseFault(req.GetMessageID(), soapFault), nil
	}

	// Embedding the DiscoveryResponse message inside of a SoapResponse
	response, err := NewSoapResponse(discoveryResponseMsg, req.GetMessageID())
	if err != nil {
		soapFault := svc.GetAuthorizedSoapFault(ctx, syncml.SoapErrorMessageFormat, mdm_types.MDEDiscovery, err)
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
	if !params.Has(syncml.STSAuthAppRu) || !params.Has(syncml.STSLoginHint) {
		return getSTSAuthContent(""), errors.New("expected STS params are not present")
	}

	appru := params.Get(syncml.STSAuthAppRu)
	loginHint := params.Get(syncml.STSLoginHint)

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
		soapFault := svc.GetAuthorizedSoapFault(ctx, syncml.SoapErrorMessageFormat, mdm_types.MDEPolicy, err)
		return getSoapResponseFault(req.GetMessageID(), soapFault), nil
	}

	// Binary security token should be extracted to ensure this is a valid call
	hdrSecToken, err := req.GetHeaderBinarySecurityToken()
	if err != nil {
		soapFault := svc.GetAuthorizedSoapFault(ctx, syncml.SoapErrorMessageFormat, mdm_types.MDEPolicy, err)
		return getSoapResponseFault(req.GetMessageID(), soapFault), nil
	}

	// Getting the GetPoliciesResponse message
	policyResponseMsg, err := svc.GetMDMWindowsPolicyResponse(ctx, hdrSecToken)
	if err != nil {
		soapFault := svc.GetAuthorizedSoapFault(ctx, syncml.SoapErrorMessageFormat, mdm_types.MDEPolicy, err)
		return getSoapResponseFault(req.GetMessageID(), soapFault), nil
	}

	// Embedding the DiscoveryResponse message inside of a SoapResponse
	response, err := NewSoapResponse(policyResponseMsg, req.GetMessageID())
	if err != nil {
		soapFault := svc.GetAuthorizedSoapFault(ctx, syncml.SoapErrorMessageFormat, mdm_types.MDEPolicy, err)
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
		soapFault := svc.GetAuthorizedSoapFault(ctx, syncml.SoapErrorMessageFormat, mdm_types.MDEEnrollment, err)
		return getSoapResponseFault(req.GetMessageID(), soapFault), nil
	}

	// Getting the RequestSecurityToken message from the SOAP request
	reqSecurityTokenMsg, err := req.GetRequestSecurityTokenMessage()
	if err != nil {
		soapFault := svc.GetAuthorizedSoapFault(ctx, syncml.SoapErrorMessageFormat, mdm_types.MDEEnrollment, err)
		return getSoapResponseFault(req.GetMessageID(), soapFault), nil
	}

	// Binary security token should be extracted to ensure this is a valid call
	hdrBinarySecToken, err := req.GetHeaderBinarySecurityToken()
	if err != nil {
		soapFault := svc.GetAuthorizedSoapFault(ctx, syncml.SoapErrorMessageFormat, mdm_types.MDEEnrollment, err)
		return getSoapResponseFault(req.GetMessageID(), soapFault), nil
	}

	// Getting the RequestSecurityTokenResponseCollection message
	enrollResponseMsg, err := svc.GetMDMWindowsEnrollResponse(ctx, reqSecurityTokenMsg, hdrBinarySecToken)
	if err != nil {
		soapFault := svc.GetAuthorizedSoapFault(ctx, syncml.SoapErrorMessageFormat, mdm_types.MDEEnrollment, err)
		return getSoapResponseFault(req.GetMessageID(), soapFault), nil
	}

	// Embedding the DiscoveryResponse message inside of a SoapResponse
	response, err := NewSoapResponse(enrollResponseMsg, req.GetMessageID())
	if err != nil {
		soapFault := svc.GetAuthorizedSoapFault(ctx, syncml.SoapErrorMessageFormat, mdm_types.MDEEnrollment, err)
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
	reqCerts := request.(*SyncMLReqMsgContainer).Certs

	// Checking first if incoming SyncML message is valid and returning error if this is not the case
	if err := reqSyncML.IsValidMsg(); err != nil {
		soapFault := svc.GetAuthorizedSoapFault(ctx, syncml.SoapErrorMessageFormat, mdm_types.MSMDM, err)
		return getSoapResponseFault(reqSyncML.SyncHdr.MsgID, soapFault), nil
	}

	// Getting the MS-MDM response message
	resSyncML, err := svc.GetMDMWindowsManagementResponse(ctx, reqSyncML, reqCerts)
	if err != nil {
		soapFault := svc.GetAuthorizedSoapFault(ctx, syncml.SoapErrorMessageFormat, mdm_types.MSMDM, err)
		return getSoapResponseFault(reqSyncML.SyncHdr.MsgID, soapFault), nil
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
	if !params.Has(syncml.TOCRedirectURI) || !params.Has(syncml.TOCReqID) {
		soapFault := svc.GetAuthorizedSoapFault(ctx, syncml.SoapErrorMessageFormat, mdm_types.MDEEnrollment, errors.New("invalid params"))
		return getSoapResponseFault(syncml.SoapErrorInternalServiceFault, soapFault), nil
	}

	redirectURI := params.Get(syncml.TOCRedirectURI)
	reqID := params.Get(syncml.TOCReqID)

	// Getting the TOS content message
	resTOCData, err := svc.GetMDMWindowsTOSContent(ctx, redirectURI, reqID)
	if err != nil {
		soapFault := svc.GetAuthorizedSoapFault(ctx, syncml.SoapErrorMessageFormat, mdm_types.MDEEnrollment, err)
		return getSoapResponseFault(syncml.SoapErrorInternalServiceFault, soapFault), nil
	}

	return MDMWebContainer{
		Data: &resTOCData,
		Err:  nil,
	}, nil
}

// authBinarySecurityToken checks if the provided token is valid. For programmatic enrollment, it
// returns the orbit node key and host uuid. For automatic enrollment, it returns only the UPN (the
// host uuid will be an empty string).
func (svc *Service) authBinarySecurityToken(ctx context.Context, authToken *fleet.HeaderBinarySecurityToken) (claim string, hostUUID string, err error) {
	if authToken == nil {
		return "", "", errors.New("authToken is empty")
	}

	err = authToken.IsValidToken()
	if err != nil {
		return "", "", errors.New("authToken is not valid")
	}

	// Tokens that were generated by enrollment client
	if authToken.IsDeviceToken() {

		// Getting the Binary Security Token Payload
		binSecToken, err := NewBinarySecurityTokenPayload(authToken.Content)
		if err != nil {
			return "", "", fmt.Errorf("token creation error %v", err)
		}

		// Validating the Binary Security Token Payload
		err = binSecToken.IsValidToken()
		if err != nil {
			return "", "", fmt.Errorf("invalid token data %v", err)
		}

		// Validating the Binary Security Token Type used on Programmatic Enrollments
		if binSecToken.Type == mdm_types.WindowsMDMProgrammaticEnrollmentType {
			host, err := svc.ds.LoadHostByOrbitNodeKey(ctx, binSecToken.Payload.OrbitNodeKey)
			if err != nil {
				return "", "", fmt.Errorf("host data cannot be found %v", err)
			}

			mdmInfo, err := svc.ds.GetHostMDM(ctx, host.ID)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return "", "", errors.New("unable to retrieve host mdm info")
			}

			// This ensures that only hosts that are eligible for Windows enrollment can be enrolled
			if !isEligibleForWindowsMDMEnrollment(host, mdmInfo) {
				return "", "", errors.New("host is not elegible for Windows MDM enrollment")
			}

			// No errors, token is authorized
			return binSecToken.Payload.OrbitNodeKey, host.UUID, nil
		}

		// Validating the Binary Security Token Type used on Automatic Enrollments (returned by STS Auth Endpoint)
		if binSecToken.Type == mdm_types.WindowsMDMAutomaticEnrollmentType {

			upnToken, err := svc.wstepCertManager.GetSTSAuthTokenUPNClaim(binSecToken.Payload.AuthToken)
			if err != nil {
				return "", "", ctxerr.Wrap(ctx, err, "issue retrieving UPN from Auth token")
			}

			// No errors, token is authorized
			return upnToken, "", nil
		}
	}

	// Validating the Binary Security Token Type used on Automatic Enrollments
	if authToken.IsAzureJWTToken() {

		// Validate the JWT Auth token by retreving its claims
		tokenData, err := microsoft_mdm.GetAzureAuthTokenClaims(authToken.Content)
		if err != nil {
			return "", "", fmt.Errorf("binary security token claim failed: %v", err)
		}

		// No errors, token is authorized
		return tokenData.UPN, "", nil
	}

	return "", "", errors.New("token is not authorized")
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
	urlPolicyEndpoint, err := microsoft_mdm.ResolveWindowsMDMPolicy(appCfg.ServerSettings.ServerURL)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "resolve policy endpoint")
	}

	urlEnrollEndpoint, err := microsoft_mdm.ResolveWindowsMDMEnroll(appCfg.ServerSettings.ServerURL)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "resolve enroll endpoint")
	}

	discoveryMsg, err := NewDiscoverResponse(syncml.AuthOnPremise, urlPolicyEndpoint, urlEnrollEndpoint)
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
	_, _, err := svc.authBinarySecurityToken(ctx, authToken)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "validate binary security token")
	}

	// Token is authorized
	svc.authz.SkipAuthorization(ctx)

	// Getting the GetPoliciesResponse message content
	policyMsg, err := NewGetPoliciesResponse(syncml.PolicyMinKeyLength, syncml.PolicyCertValidityPeriodInSecs, syncml.PolicyCertRenewalPeriodInSecs)
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
	userID, hostUUID, err := svc.authBinarySecurityToken(ctx, authToken)
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
	err = svc.storeWindowsMDMEnrolledDevice(ctx, userID, hostUUID, secTokenMsg)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "enrolled device information cannot be stored")
	}

	return &secTokenResponseCollectionMsg, nil
}

// GetMDMWindowsManagementResponse returns a valid SyncML response message
func (svc *Service) GetMDMWindowsManagementResponse(ctx context.Context, reqSyncML *fleet.SyncML, reqCerts []*x509.Certificate) (*fleet.SyncML, error) {
	if reqSyncML == nil {
		return nil, fleet.NewInvalidArgumentError("syncml req message", "message is not present")
	}

	// Checking if the incoming request is trusted
	err := svc.isTrustedRequest(ctx, reqSyncML, reqCerts)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "management request is not trusted")
	}

	// Getting the management response message
	resSyncMLmsg, err := svc.getManagementResponse(ctx, reqSyncML)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "management response message")
	}

	// Token is authorized
	svc.authz.SkipAuthorization(ctx)

	return resSyncMLmsg, nil
}

// GetMDMWindowsTOSContent returns valid TOC content
func (svc *Service) GetMDMWindowsTOSContent(ctx context.Context, redirectUri string, reqID string) (string, error) {
	tmpl, err := server.GetTemplate("frontend/templates/windowsTOS.html", "windows-tos")
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

// isValidUPN checks if the provided user ID is a valid UPN
func isValidUPN(userID string) bool {
	return upnRegex.MatchString(userID)
}

// isTrustedRequest checks if the incoming request was sent from MDM enrolled device
func (svc *Service) isTrustedRequest(ctx context.Context, reqSyncML *fleet.SyncML, reqCerts []*x509.Certificate) error {
	if reqSyncML == nil {
		return fleet.NewInvalidArgumentError("syncml req message", "message is not present")
	}

	// Checking if calling request is coming from an already MDM enrolled device
	deviceID, err := reqSyncML.GetSource()
	if err != nil || deviceID == "" {
		return fmt.Errorf("invalid SyncML message %w", err)
	}

	enrolledDevice, err := svc.ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, deviceID)
	if err != nil || enrolledDevice == nil {
		return errors.New("device was not MDM enrolled")
	}

	// Check if TLS certs contains device ID on its common name
	if len(reqCerts) > 0 {
		for _, reqCert := range reqCerts {
			if strings.Contains(reqCert.Subject.CommonName, deviceID) {
				return nil
			}
		}
	}

	// TODO: Latest version of the MDM client stack don't populate TLS.PeerCertificates array
	// This is a temporary workaround to allow the management request to proceed
	// Transport-level security should be replaced for Application-level security
	// Transport-level security is defined in the MS-MDM spec in section 1.3.1
	// On the other hand, Application-level security is defined here
	// https://www.openmobilealliance.org/release/DM/V1_2_1-20080617-A/OMA-TS-DM_Security-V1_2_1-20080617-A.pdf
	// The initial values for Application-level security configuration are defined in the
	// WAP Profile blob that is sent to the device during the enrollment process. Example below
	//	<characteristic type="APPAUTH">
	//		<parm name="AAUTHLEVEL" value="CLIENT"/>
	//		<parm name="AAUTHTYPE" value="DIGEST"/>
	//		<parm name="AAUTHSECRET" value="2jsidqgffx"/>
	//		<parm name="AAUTHDATA" value="aGVsbG8gd29ybGQ="/>
	//	</characteristic>
	//	<characteristic type="APPAUTH">
	//		<parm name="AAUTHLEVEL" value="APPSRV"/>
	//		<parm name="AAUTHTYPE" value="DIGEST"/>
	//		<parm name="AAUTHNAME" value="43f8bf591b8557346021"/>
	//		<parm name="AAUTHSECRET" value="crbr3w2cab"/>
	//		<parm name="AAUTHDATA" value="aGVsbG8gd29ybGQ="/>
	//	</characteristic>

	if len(reqCerts) == 0 {
		return nil
	}

	return errors.New("calling device is not trusted")
}

// regex to validate UPN
var upnRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// isFleetdPresentOnDevice checks if the device requires Fleetd to be deployed
func (svc *Service) isFleetdPresentOnDevice(ctx context.Context, deviceID string) (bool, error) {
	// checking first if the device was enrolled through programmatic flow
	enrolledDevice, err := svc.ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, deviceID)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "get windows enrolled device")
	}

	// If user identity is a MS-MDM UPN it means that the device was enrolled through user-driven flow
	// This means that fleetd might not be installed
	if isValidUPN(enrolledDevice.MDMEnrollUserID) {
		var isPresent bool
		if enrolledDevice.HostUUID != "" {
			host, err := svc.ds.HostLiteByIdentifier(ctx, enrolledDevice.HostUUID)
			if err != nil && !fleet.IsNotFound(err) {
				return false, ctxerr.Wrap(ctx, err, "get host lite by identifier")
			}
			if host != nil {
				orbitInfo, err := svc.ds.GetHostOrbitInfo(ctx, host.ID)
				if err != nil && !fleet.IsNotFound(err) {
					return false, ctxerr.Wrap(ctx, err, "get host orbit info")
				}
				if orbitInfo != nil {
					isPresent = orbitInfo.Version != ""
				}
			}
		}
		return isPresent, nil
	}

	// TODO: Add check here to determine if MDM DeviceID is connected with Smbios UUID present on
	// host table. This new check should look into command results table and extract the value of
	// ./DevDetail/Ext/Microsoft/SMBIOSSerialNumber for the given DeviceID and use that for hosts
	// table lookup
	return true, nil
}

func (svc *Service) enqueueInstallFleetdCommand(ctx context.Context, deviceID string) error {
	secrets, err := svc.ds.GetEnrollSecrets(ctx, nil)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting enroll secrets")
	}

	if len(secrets) == 0 {
		level.Warn(svc.logger).Log("msg", "unable to find a global enroll secret to install fleetd")
		return nil
	}

	// it's okay to skip the installation if we're not able to retrieve the
	// metadata, we don't want to completely error the SyncML transaction
	// and we'll try again the next time the host checks in
	fleetdMetadata, err := fleetdbase.GetMetadata()
	if err != nil {
		level.Warn(svc.logger).Log("msg", "unable to get fleetd-base metadata")
		return nil
	}

	appCfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting app config")
	}
	fleetURL := appCfg.ServerSettings.ServerURL
	globalEnrollSecret := secrets[0].Secret
	addCommandUUID := uuid.NewString()
	execCommandUUID := uuid.NewString()

	rawAddCmd := []byte(`
<Add>
	<CmdID>` + addCommandUUID + `</CmdID>
	<Item>
		<Target>
			<LocURI>` + syncml.FleetdWindowsInstallerGUID + `</LocURI>
		</Target>
	</Item>
</Add>`)

	// keeping the same GUID will prevent the MSI to be installed multiple times - it will be
	// installed only the first time the message is issued.
	// FleetURL and FleetSecret properties are passed to the Fleet MSI
	rawExecCmd := []byte(`
<Exec>
	<CmdID>` + execCommandUUID + `</CmdID>
	<Item>
		<Target>
			<LocURI>` + syncml.FleetdWindowsInstallerGUID + `</LocURI>
		</Target>
		<Data>
			<MsiInstallJob id="{A427C0AA-E2D5-40DF-ACE8-0D726A6BE096}">
			<Product Version="1.0.0.0">
				<Download>
					<ContentURLList>
						<ContentURL>` + fleetdMetadata.MSIURL + `</ContentURL>
					</ContentURLList>
				</Download>
				<Validation>
					<FileHash>` + fleetdMetadata.MSISha256 + `</FileHash>
				</Validation>
				<Enforcement>
					<CommandLine>/quiet FLEET_URL="` + fleetURL + `" FLEET_SECRET="` + globalEnrollSecret + `" ENABLE_SCRIPTS="True"</CommandLine>
					<TimeOut>10</TimeOut>
					<RetryCount>1</RetryCount>
					<RetryInterval>5</RetryInterval>
				</Enforcement>
			</Product>
			</MsiInstallJob>
		</Data>
		<Meta>
			<Type xmlns="syncml:metinf">text/plain</Type>
			<Format xmlns="syncml:metinf">xml</Format>
		</Meta>
	</Item>
</Exec>
`)

	// TODO: add ability to batch-enqueue multiple commands at the same time
	addFleetdCmd := &fleet.MDMWindowsCommand{
		CommandUUID:  addCommandUUID,
		RawCommand:   rawAddCmd,
		TargetLocURI: syncml.FleetdWindowsInstallerGUID,
	}
	if err := svc.ds.MDMWindowsInsertCommandForHosts(ctx, []string{deviceID}, addFleetdCmd); err != nil {
		return ctxerr.Wrap(ctx, err, "insert add command to install fleetd")
	}

	execFleetCmd := &fleet.MDMWindowsCommand{
		CommandUUID:  execCommandUUID,
		RawCommand:   rawExecCmd,
		TargetLocURI: syncml.FleetdWindowsInstallerGUID,
	}
	if err := svc.ds.MDMWindowsInsertCommandForHosts(ctx, []string{deviceID}, execFleetCmd); err != nil {
		return ctxerr.Wrap(ctx, err, "insert exec command to install fleetd")
	}

	return nil
}

// Alerts Handlers

// New session Alert Handler
// This handler will return an protocol command to install an MSI on a new session from unenrolled device
func (svc *Service) processNewSessionAlert(ctx context.Context, messageID string, deviceID string, cmd mdm_types.ProtoCmdOperation) error {
	// Checking if fleetd is present on the device
	fleetdPresent, err := svc.isFleetdPresentOnDevice(ctx, deviceID)
	if err != nil {
		return err
	}

	if !fleetdPresent {
		return svc.enqueueInstallFleetdCommand(ctx, deviceID)
	}

	return nil
}

// Generic Alert Handlers
// This handler will check for generic alerts. Device unenrollment is handled here
func (svc *Service) processGenericAlert(ctx context.Context, messageID string, deviceID string, cmd mdm_types.ProtoCmdOperation) error {
	// Checking user-initiated unenrollment request
	if len(cmd.Cmd.Items) > 0 {
		for _, item := range cmd.Cmd.Items {

			if item.Meta == nil || item.Meta.Type == nil || item.Meta.Type.Content == nil {
				continue
			}

			// Checking if user-initiated unenrollment request is present
			if *item.Meta.Type.Content == syncml.AlertUserUnenrollmentRequest {

				// Deleting the device from the list of enrolled device
				err := svc.ds.MDMWindowsDeleteEnrolledDeviceWithDeviceID(ctx, deviceID)
				if err != nil {
					return fmt.Errorf("unenrolling windows device: %w", err)
				}
			}
		}
	}

	return nil
}

// processIncomingAlertsCommands will process the incoming Alerts commands.
// These commands don't require an status response.
func (svc *Service) processIncomingAlertsCommands(ctx context.Context, messageID string, deviceID string, cmd mdm_types.ProtoCmdOperation) error {
	if cmd.Cmd.Data == nil {
		return errors.New("invalid alert command")
	}

	// gathering the incoming Alert ID
	alertID := *cmd.Cmd.Data

	switch alertID {
	case syncml.CmdAlertClientInitiatedManagement:
		return svc.processNewSessionAlert(ctx, messageID, deviceID, cmd)
	case syncml.CmdAlertServerInitiatedManagement:
		return svc.processNewSessionAlert(ctx, messageID, deviceID, cmd)
	case syncml.CmdAlertGeneric:
		return svc.processGenericAlert(ctx, messageID, deviceID, cmd)
	}

	return nil
}

// processIncomingMDMCmds process the incoming message from the device
// It will return the list of operations that need to be sent to the device
func (svc *Service) processIncomingMDMCmds(ctx context.Context, deviceID string, reqMsg *fleet.SyncML) ([]*fleet.SyncMLCmd, error) {
	var responseCmds []*fleet.SyncMLCmd

	// Get the incoming MessageID
	reqMessageID, err := reqMsg.GetMessageID()
	if err != nil {
		return nil, fmt.Errorf("get incoming msg: %w", err)
	}

	// Acknowledge the message header
	// msgref is always 0 for the header
	if err = reqMsg.IsValidHeader(); err == nil {
		ackMsg := NewSyncMLCmdStatus(reqMessageID, "0", syncml.SyncMLHdrName, syncml.CmdStatusOK)
		responseCmds = append(responseCmds, ackMsg)
	}

	if err := svc.ds.MDMWindowsSaveResponse(ctx, deviceID, reqMsg); err != nil {
		return nil, fmt.Errorf("store incoming msgs: %w", err)
	}

	// Iterate over the operations and process them
	for _, protoCMD := range reqMsg.GetOrderedCmds() {
		// Alerts, Results and Status don't require a status response
		switch protoCMD.Verb {
		case mdm_types.CmdAlert:
			err := svc.processIncomingAlertsCommands(ctx, reqMessageID, deviceID, protoCMD)
			if err != nil {
				return nil, fmt.Errorf("process incoming command: %w", err)
			}
			continue
		case mdm_types.CmdStatus, mdm_types.CmdResults:
			continue
		}

		// CmdStatusOK is returned for the rest of the operations
		responseCmds = append(responseCmds, NewSyncMLCmdStatus(reqMessageID, protoCMD.Cmd.CmdID.Value, protoCMD.Verb, syncml.CmdStatusOK))
	}

	return responseCmds, nil
}

// getPendingMDMCmds returns the list of pending MDM commands for the device
func (svc *Service) getPendingMDMCmds(ctx context.Context, deviceID string) ([]*mdm_types.SyncMLCmd, error) {
	pendingCmds, err := svc.ds.MDMWindowsGetPendingCommands(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("getting incoming cmds %w", err)
	}

	// Converting the pending commands to its target SyncML types
	var cmds []*mdm_types.SyncMLCmd
	for _, pendingCmd := range pendingCmds {
		// The raw MDM command may contain a $FLEET_SECRET_XXX, the value of which should never be exposed or stored unencrypted.
		rawCommandWithSecret, err := svc.ds.ExpandEmbeddedSecrets(ctx, string(pendingCmd.RawCommand))
		if err != nil {
			// This error should never happen since we validate the presence of needed secrets on profile upload.
			return nil, ctxerr.Wrap(ctx, err, "expanding embedded secrets for Windows pending commands")
		}
		cmd := new(mdm_types.SyncMLCmd)
		if err := xml.Unmarshal([]byte(rawCommandWithSecret), cmd); err != nil {
			logging.WithErr(ctx, ctxerr.Wrap(ctx, err, "getPendingMDMCmds syncML cmd creation"))
			continue
		}
		cmds = append(cmds, cmd)
	}

	return cmds, nil
}

// createResponseSyncML returns a valid SyncML message
func (svc *Service) createResponseSyncML(ctx context.Context, req *fleet.SyncML, responseOps []*mdm_types.SyncMLCmd) (*fleet.SyncML, error) {
	// Get the DeviceID
	deviceID, err := req.GetSource()
	if err != nil || deviceID == "" {
		return nil, fmt.Errorf("invalid SyncML message %w", err)
	}

	// Get SessionID
	sessionID, err := req.GetSessionID()
	if err != nil {
		return nil, fmt.Errorf("session ID processing error %w", err)
	}

	// Get MessageID
	messageID, err := req.GetMessageID()
	if err != nil {
		return nil, fmt.Errorf("message ID processing error %w", err)
	}

	// Getting the Management endpoint URL
	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("appconfig was not available %w", err)
	}

	urlManagementEndpoint, err := microsoft_mdm.ResolveWindowsMDMManagement(appConfig.ServerSettings.ServerURL)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "resolve management endpoint")
	}

	// Create the SyncML message with the response operations
	msg, err := createSyncMLMessage(sessionID, messageID, deviceID, urlManagementEndpoint, responseOps)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creation of SyncML message")
	}

	return msg, nil
}

// getManagementResponse returns a valid SyncML response message
func (svc *Service) getManagementResponse(ctx context.Context, reqMsg *fleet.SyncML) (*mdm_types.SyncML, error) {
	if reqMsg == nil {
		return nil, fleet.NewInvalidArgumentError("syncml req message", "message is not present")
	}

	// Get the DeviceID
	deviceID, err := reqMsg.GetSource()
	if err != nil || deviceID == "" {
		return nil, fmt.Errorf("invalid SyncML message %w", err)
	}

	// Process the incoming MDM protocol commands and get the response MDM protocol commands
	resIncomingCmds, err := svc.processIncomingMDMCmds(ctx, deviceID, reqMsg)
	if err != nil {
		return nil, fmt.Errorf("message processing error %w", err)
	}

	// Process the pending operations and get the MDM response protocol commands
	resPendingCmds, err := svc.getPendingMDMCmds(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("message processing error %w", err)
	}

	// Combined cmd responses
	resCmds := resIncomingCmds
	resCmds = append(resCmds, resPendingCmds...)

	// Create the response SyncML message
	msg, err := svc.createResponseSyncML(ctx, reqMsg, resCmds)
	if err != nil {
		return nil, fmt.Errorf("message syncML creation error %w", err)
	}

	return msg, nil
}

// removeWindowsDeviceIfAlreadyMDMEnrolled removes the device if already MDM enrolled
// HW DeviceID is used to check the list of enrolled devices
func (svc *Service) removeWindowsDeviceIfAlreadyMDMEnrolled(ctx context.Context, secTokenMsg *fleet.RequestSecurityToken) error {
	// Getting the HW DeviceID from the RequestSecurityToken msg
	reqHWDeviceID, err := GetContextItem(secTokenMsg, syncml.ReqSecTokenContextItemHWDevID)
	if err != nil {
		return err
	}

	// Device is already enrolled, let's remove it
	err = svc.ds.MDMWindowsDeleteEnrolledDevice(ctx, reqHWDeviceID)
	if err != nil {
		if fleet.IsNotFound(err) {
			return nil
		}
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
	reqHWDeviceID, err := GetContextItem(secTokenMsg, syncml.ReqSecTokenContextItemHWDevID)
	if err != nil {
		return "", err
	}

	// Getting the EnrollmentType information from the RequestSecurityToken msg
	reqEnrollType, err := GetContextItem(secTokenMsg, syncml.ReqSecTokenContextItemEnrollmentType)
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
	clientCSR, err := microsoft_mdm.GetClientCSR(binSecurityTokenData, binSecurityTokenType)
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
	urlManagementEndpoint, err := microsoft_mdm.ResolveWindowsMDMManagement(appCfg.ServerSettings.ServerURL)
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
func (svc *Service) storeWindowsMDMEnrolledDevice(ctx context.Context, userID string, hostUUID string, secTokenMsg *fleet.RequestSecurityToken) error {
	const (
		error_tag = "windows MDM enrolled storage: "
	)

	// Getting the DeviceID context information from the RequestSecurityToken msg
	reqDeviceID, err := GetContextItem(secTokenMsg, syncml.ReqSecTokenContextItemDeviceID)
	if err != nil {
		return fmt.Errorf("%s %v", error_tag, err)
	}

	// Getting the HWDevID context information from the RequestSecurityToken msg
	reqHWDevID, err := GetContextItem(secTokenMsg, syncml.ReqSecTokenContextItemHWDevID)
	if err != nil {
		return fmt.Errorf("%s %v", error_tag, err)
	}

	// Getting the Enroll DeviceType context information from the RequestSecurityToken msg
	reqDeviceType, err := GetContextItem(secTokenMsg, syncml.ReqSecTokenContextItemDeviceType)
	if err != nil {
		return fmt.Errorf("%s %v", error_tag, err)
	}

	// Getting the Enroll DeviceName context information from the RequestSecurityToken msg
	reqDeviceName, err := GetContextItem(secTokenMsg, syncml.ReqSecTokenContextItemDeviceName)
	if err != nil {
		return fmt.Errorf("%s %v", error_tag, err)
	}

	// Getting the Enroll RequestVersion context information from the RequestSecurityToken msg
	reqEnrollVersion, err := GetContextItem(secTokenMsg, syncml.ReqSecTokenContextItemRequestVersion)
	if err != nil {
		reqEnrollVersion = "request_version_not_present"
	}

	// Getting the RequestVersion context information from the RequestSecurityToken msg
	reqAppVersion, err := GetContextItem(secTokenMsg, syncml.ReqSecTokenContextItemApplicationVersion)
	if err != nil {
		return fmt.Errorf("%s %v", error_tag, err)
	}

	// Getting the EnrollmentType information from the RequestSecurityToken msg
	reqEnrollType, err := GetContextItem(secTokenMsg, syncml.ReqSecTokenContextItemEnrollmentType)
	if err != nil {
		return fmt.Errorf("%s %v", error_tag, err)
	}

	// Getting the Windows Enrolled Device Information
	enrolledDevice := &fleet.MDMWindowsEnrolledDevice{
		MDMDeviceID:            reqDeviceID,
		MDMHardwareID:          reqHWDevID,
		MDMDeviceState:         microsoft_mdm.MDMDeviceStateEnrolled,
		MDMDeviceType:          reqDeviceType,
		MDMDeviceName:          reqDeviceName,
		MDMEnrollType:          reqEnrollType,
		MDMEnrollUserID:        userID, // This could be Host UUID or UPN email
		MDMEnrollProtoVersion:  reqEnrollVersion,
		MDMEnrollClientVersion: reqAppVersion,
		MDMNotInOOBE:           false,
		HostUUID:               hostUUID,
	}

	if err := svc.ds.MDMWindowsInsertEnrolledDevice(ctx, enrolledDevice); err != nil {
		return err
	}

	// TODO: azure enrollments come with an empty uuid, I haven't figured
	// out a good way to identify the device.
	displayName := reqDeviceName
	var serial string
	if hostUUID != "" {
		mdmLifecycle := mdmlifecycle.New(svc.ds, svc.logger)
		err = mdmLifecycle.Do(ctx, mdmlifecycle.HostOptions{
			Action:   mdmlifecycle.HostActionTurnOn,
			Platform: "windows",
			UUID:     hostUUID,
		})
		if err != nil {
			return err
		}

		// Get the host in order to get the correct display name and serial number for the activity
		adminTeamFilter := fleet.TeamFilter{
			User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
		}

		hosts, err := svc.ds.ListHostsLiteByUUIDs(ctx, adminTeamFilter, []string{hostUUID})
		if err != nil {
			// Do not abort; this call was only made to get better data for the activity, so shouldn't
			// fail the request. We fall back to `reqDeviceName` for the display name in this case.
			logging.WithExtras(logging.WithNoUser(ctx),
				"msg", "failed to get host data for windows MDM enrollment activity",
			)
		}

		if len(hosts) == 1 {
			// then we found the host, so use the data from there for the activity
			displayName = hosts[0].DisplayName()
			serial = hosts[0].HardwareSerial
		}

	}

	err = svc.NewActivity(
		ctx, nil, &fleet.ActivityTypeMDMEnrolled{
			HostDisplayName: displayName,
			MDMPlatform:     fleet.MDMPlatformMicrosoft,
			HostSerial:      serial,
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
	logging.WithErr(ctx, ctxerr.Wrap(ctx, errorMsg, "soap fault"))
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

// MS-MDM Commands helpers
// createSyncMLMessage takes input data and returns a SyncML struct
func createSyncMLMessage(sessionID string, msgID string, deviceID string, source string, protoCommands []*mdm_types.SyncMLCmd) (*mdm_types.SyncML, error) {
	// Sanity check on input
	if len(sessionID) == 0 || len(msgID) == 0 || len(deviceID) == 0 || len(source) == 0 {
		return nil, errors.New("invalid parameters")
	}

	if sessionID == "0" {
		return nil, errors.New("invalid session ID")
	}

	if msgID == "0" {
		return nil, errors.New("invalid msg ID")
	}

	if len(protoCommands) == 0 {
		return nil, errors.New("invalid operations")
	}

	// Setting source LocURI
	var sourceLocURI *mdm_types.LocURI

	if len(source) > 0 {
		sourceLocURI = &mdm_types.LocURI{
			LocURI: &source,
		}
	}

	// setting up things on the SyncML message
	var msg mdm_types.SyncML
	msg.Xmlns = syncml.SyncCmdNamespace
	msg.SyncHdr = mdm_types.SyncHdr{
		VerDTD:    syncml.SyncMLSupportedVersion,
		VerProto:  syncml.SyncMLVerProto,
		SessionID: sessionID,
		MsgID:     msgID,
		Target:    &mdm_types.LocURI{LocURI: &deviceID},
		Source:    sourceLocURI,
	}

	// iterate over operations and append them to the SyncML message
	for _, protoCmd := range protoCommands {
		msg.AppendCommand(fleet.MDMRaw, *protoCmd)
	}

	// If there was no error, return the SyncML and a nil error
	return &msg, nil
}

// newSyncMLCmdWithNoItem creates a new SyncML command
func newSyncMLCmdWithNoItem(cmdVerb *string, cmdData *string) *mdm_types.SyncMLCmd {
	return &mdm_types.SyncMLCmd{
		XMLName: xml.Name{Local: *cmdVerb},
		Data:    cmdData,
		Items:   nil,
	}
}

// newSyncMLCmdWithItem creates a new SyncML command
func newSyncMLCmdWithItem(cmdVerb *string, cmdData *string, cmdItem *mdm_types.CmdItem) *mdm_types.SyncMLCmd {
	return &mdm_types.SyncMLCmd{
		XMLName: xml.Name{Local: *cmdVerb},
		Data:    cmdData,
		Items:   []mdm_types.CmdItem{*cmdItem},
	}
}

// newSyncMLItem creates a new SyncML command
func newSyncMLItem(cmdSource *string, cmdTarget *string, cmdDataType *string, cmdDataFormat *string, cmdDataValue *string) *mdm_types.CmdItem {
	var metaFormat *mdm_types.MetaAttr
	var metaType *mdm_types.MetaAttr
	var meta *mdm_types.Meta
	var data *mdm_types.RawXmlData

	if cmdDataFormat != nil && len(*cmdDataFormat) > 0 {
		metaFormat = &mdm_types.MetaAttr{
			XMLNS:   "syncml:metinf",
			Content: cmdDataFormat,
		}
	}

	if cmdDataType != nil && len(*cmdDataType) > 0 {
		metaType = &mdm_types.MetaAttr{
			XMLNS:   "syncml:metinf",
			Content: cmdDataType,
		}
	}

	if metaFormat != nil || metaType != nil {
		meta = &mdm_types.Meta{
			Format: metaFormat,
			Type:   metaType,
		}
	}

	if cmdDataValue != nil {
		data = &mdm_types.RawXmlData{
			Content: *cmdDataValue,
		}
	}

	return &mdm_types.CmdItem{
		Meta:   meta,
		Data:   data,
		Target: cmdTarget,
		Source: cmdSource,
	}
}

// NewSyncMLCmd creates a new SyncML command
func NewSyncMLCmd(cmdVerb string, cmdSource string, cmdTarget string, cmdDataType string, cmdDataFormat string, cmdDataValue string) *mdm_types.SyncMLCmd {
	var workCmdVerb *string
	var workCmdSource *string
	var workCmdTarget *string
	var workCmdDataType *string
	var workCmdDataFormat *string
	var workCmdDataValue *string

	if len(cmdVerb) > 0 {
		workCmdVerb = &cmdVerb
	}

	if len(cmdSource) > 0 {
		workCmdSource = &cmdSource
	}

	if len(cmdTarget) > 0 {
		workCmdTarget = &cmdTarget
	}

	if len(cmdDataType) > 0 {
		workCmdDataType = &cmdDataType
	}

	if len(cmdDataFormat) > 0 {
		workCmdDataFormat = &cmdDataFormat
	}

	if len(cmdDataValue) > 0 {
		workCmdDataValue = &cmdDataValue
	}

	item := newSyncMLItem(workCmdSource, workCmdTarget, workCmdDataType, workCmdDataFormat, workCmdDataValue)
	return newSyncMLCmdWithItem(workCmdVerb, nil, item)
}

func NewTypedSyncMLCmd(dataType mdm_types.SyncMLDataType, cmdVerb string, cmdTarget string, cmdData string) (*mdm_types.SyncMLCmd, error) {
	errInvalidParameters := errors.New("invalid parameters")

	// Checking if command verb is present
	if cmdVerb == "" {
		return nil, errInvalidParameters
	}

	// Returning command based on input command data type
	switch dataType {
	case mdm_types.SFEmpty:
		if len(cmdData) > 0 {
			rawCmd := newSyncMLNoItem(cmdVerb, cmdData)
			return rawCmd, nil
		}

		return nil, errInvalidParameters

	case mdm_types.SFNoFormat:
		if len(cmdData) > 0 && len(cmdTarget) > 0 {
			rawCmd := newSyncMLNoFormat(cmdVerb, cmdTarget)
			return rawCmd, nil
		}

		return nil, errInvalidParameters

	case mdm_types.SFText:
		if len(cmdData) > 0 && len(cmdTarget) > 0 && len(cmdData) > 0 {
			rawCmd := newSyncMLCmdText(cmdVerb, cmdTarget, cmdData)
			return rawCmd, nil
		}

		return nil, errInvalidParameters

	case mdm_types.SFXml:
		if len(cmdData) > 0 && len(cmdTarget) > 0 && len(cmdData) > 0 {
			rawCmd := newSyncMLCmdXml(cmdVerb, cmdTarget, cmdData)
			return rawCmd, nil
		}

		return nil, errInvalidParameters

	case mdm_types.SFInteger:
		if len(cmdData) > 0 && len(cmdTarget) > 0 && len(cmdData) > 0 {
			rawCmd := newSyncMLCmdInt(cmdVerb, cmdTarget, cmdData)
			return rawCmd, nil
		}

		return nil, errInvalidParameters

	case mdm_types.SFBase64:
		if len(cmdData) > 0 && len(cmdTarget) > 0 && len(cmdData) > 0 {
			rawCmd := newSyncMLCmdBase64(cmdVerb, cmdTarget, cmdData)
			return rawCmd, nil
		}

		return nil, errInvalidParameters

	case mdm_types.SFBoolean:
		if len(cmdData) > 0 && len(cmdTarget) > 0 && len(cmdData) > 0 {
			rawCmd := newSyncMLCmdBool(cmdVerb, cmdTarget, cmdData)
			return rawCmd, nil
		}

		return nil, errInvalidParameters
	}

	return nil, errInvalidParameters
}

// newSyncMLNoItem creates a new SyncML command with no item
// This is used for commands that do not have any items such as Alerts
func newSyncMLNoItem(cmdVerb string, cmdData string) *mdm_types.SyncMLCmd {
	return newSyncMLCmdWithNoItem(&cmdVerb, &cmdData)
}

// newSyncMLNoFormat creates a new SyncML command with no format
// This is used for commands that do not have any data such as Get
func newSyncMLNoFormat(cmdVerb string, cmdTarget string) *mdm_types.SyncMLCmd {
	item := newSyncMLItem(nil, &cmdTarget, nil, nil, nil)
	return newSyncMLCmdWithItem(&cmdVerb, nil, item)
}

// newSyncMLCmdText creates a new SyncML command with text data
func newSyncMLCmdText(cmdVerb string, cmdTarget string, cmdDataValue string) *mdm_types.SyncMLCmd {
	cmdType := "text/plain"
	cmdFormat := "chr"
	item := newSyncMLItem(nil, &cmdTarget, &cmdType, &cmdFormat, &cmdDataValue)
	return newSyncMLCmdWithItem(&cmdVerb, nil, item)
}

// newSyncMLCmdXml creates a new SyncML command with XML data
func newSyncMLCmdXml(cmdVerb string, cmdTarget string, cmdDataValue string) *mdm_types.SyncMLCmd {
	cmdType := "text/plain"
	cmdFormat := "xml"
	escapedXML := html.EscapeString(cmdDataValue)
	item := newSyncMLItem(nil, &cmdTarget, &cmdType, &cmdFormat, &escapedXML)
	return newSyncMLCmdWithItem(&cmdVerb, nil, item)
}

// newSyncMLCmdBase64 creates a new SyncML command with Base64 encoded data
func newSyncMLCmdBase64(cmdVerb string, cmdTarget string, cmdDataValue string) *mdm_types.SyncMLCmd {
	cmdFormat := "b64"
	escapedXML := html.EscapeString(cmdDataValue)
	item := newSyncMLItem(nil, &cmdTarget, nil, &cmdFormat, &escapedXML)
	return newSyncMLCmdWithItem(&cmdVerb, nil, item)
}

// newSyncMLCmdInt creates a new SyncML command with text data
func newSyncMLCmdInt(cmdVerb string, cmdTarget string, cmdDataValue string) *mdm_types.SyncMLCmd {
	cmdType := "text/plain"
	cmdFormat := "int"
	item := newSyncMLItem(nil, &cmdTarget, &cmdType, &cmdFormat, &cmdDataValue)
	return newSyncMLCmdWithItem(&cmdVerb, nil, item)
}

// newSyncMLCmdBool creates a new SyncML command with text data
func newSyncMLCmdBool(cmdVerb string, cmdTarget string, cmdDataValue string) *mdm_types.SyncMLCmd {
	cmdType := "text/plain"
	cmdFormat := "bool"
	item := newSyncMLItem(nil, &cmdTarget, &cmdType, &cmdFormat, &cmdDataValue)
	return newSyncMLCmdWithItem(&cmdVerb, nil, item)
}

// NewSyncMLCmdStatus creates a new SyncML command with text data
func NewSyncMLCmdStatus(msgRef string, cmdRef string, cmdOrig string, statusCode string) *mdm_types.SyncMLCmd {
	return &mdm_types.SyncMLCmd{
		XMLName: xml.Name{Local: mdm_types.CmdStatus},
		MsgRef:  &msgRef,
		CmdRef:  &cmdRef,
		Cmd:     &cmdOrig,
		Data:    &statusCode,
		Items:   nil,
		CmdID: mdm_types.CmdID{
			Value: uuid.NewString(),
		},
	}
}

func (svc *Service) GetMDMWindowsProfilesSummary(ctx context.Context, teamID *uint) (*fleet.MDMProfilesSummary, error) {
	if err := svc.authz.Authorize(ctx, fleet.MDMConfigProfileAuthz{TeamID: teamID}, fleet.ActionRead); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	if err := svc.VerifyMDMWindowsConfigured(ctx); err != nil {
		return &fleet.MDMProfilesSummary{}, nil
	}

	ps, err := svc.ds.GetMDMWindowsProfilesSummary(ctx, teamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	return ps, nil
}

func ReconcileWindowsProfiles(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger) error {
	appConfig, err := ds.AppConfig(ctx)
	if err != nil {
		return fmt.Errorf("reading app config: %w", err)
	}
	if !appConfig.MDM.WindowsEnabledAndConfigured {
		return nil
	}

	// retrieve the profiles to install/remove.
	toInstall, err := ds.ListMDMWindowsProfilesToInstall(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting profiles to install")
	}
	toRemove, err := ds.ListMDMWindowsProfilesToRemove(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting profiles to remove")
	}

	// toGetContents contains the IDs of all the profiles from which we
	// need to retrieve contents. Since the previous query returns one row
	// per host, it would be too expensive to retrieve the profile contents
	// there, so we make another request. Using a map to deduplicate.
	toGetContents := make(map[string]bool)

	// hostProfiles tracks each host_mdm_windows_profile we need to upsert
	// with the new status, operation_type, etc.
	hostProfiles := make([]*fleet.MDMWindowsBulkUpsertHostProfilePayload, 0, len(toInstall))

	// install are maps from profileUUID -> command uuid and host
	// UUIDs as the underlying MDM services are optimized to send one command to
	// multiple hosts at the same time. Note that the same command uuid is used
	// for all hosts in a given install/remove target operation.
	type cmdTarget struct {
		cmdUUID   string
		profID    string
		hostUUIDs []string
	}
	installTargets := make(map[string]*cmdTarget)

	for _, p := range toInstall {
		toGetContents[p.ProfileUUID] = true
		target := installTargets[p.ProfileUUID]
		if target == nil {
			target = &cmdTarget{
				cmdUUID: uuid.New().String(),
				profID:  p.ProfileUUID,
			}
			installTargets[p.ProfileUUID] = target
		}
		target.hostUUIDs = append(target.hostUUIDs, p.HostUUID)

		hostProfiles = append(hostProfiles, &fleet.MDMWindowsBulkUpsertHostProfilePayload{
			ProfileUUID:   p.ProfileUUID,
			HostUUID:      p.HostUUID,
			ProfileName:   p.ProfileName,
			CommandUUID:   target.cmdUUID,
			OperationType: fleet.MDMOperationTypeInstall,
			Status:        &fleet.MDMDeliveryPending,
		})
		level.Debug(logger).Log("msg", "installing profile", "profile_uuid", p.ProfileUUID, "host_id", p.HostUUID, "name", p.ProfileName)
	}

	// Grab the contents of all the profiles we need to install
	profileUUIDs := make([]string, 0, len(toGetContents))
	for pid := range toGetContents {
		profileUUIDs = append(profileUUIDs, pid)
	}
	profileContents, err := ds.GetMDMWindowsProfilesContents(ctx, profileUUIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get profile contents")
	}

	for profUUID, target := range installTargets {
		p, ok := profileContents[profUUID]
		if !ok {
			// this should never happen
			return ctxerr.Wrapf(ctx, err, "missing profile content for profile %s", profUUID)
		}

		command, err := buildCommandFromProfileBytes(p, target.cmdUUID)
		if err != nil {
			level.Info(logger).Log("err", err, "profile_uuid", profUUID)
			continue
		}
		if err := ds.MDMWindowsInsertCommandForHosts(ctx, target.hostUUIDs, command); err != nil {
			return ctxerr.Wrap(ctx, err, "inserting commands for hosts")
		}
	}

	// Windows profiles are just deleted from the DB, the notion of sending
	// a command to remove a profile doesn't exist.
	if err := ds.BulkDeleteMDMWindowsHostsConfigProfiles(ctx, toRemove); err != nil {
		return ctxerr.Wrap(ctx, err, "deleting profiles that didn't change")
	}

	// Upsert the status of the host profiles we need to track.
	if err := ds.BulkUpsertMDMWindowsHostProfiles(ctx, hostProfiles); err != nil {
		return ctxerr.Wrap(ctx, err, "updating host profiles")
	}

	return nil
}

// TODO(roberto): I think this should live separately in the
// Windows equivalent of Apple's Commander struct, but I'd like
// to keep it simpler for now until we understand more.
func buildCommandFromProfileBytes(profileBytes []byte, commandUUID string) (*fleet.MDMWindowsCommand, error) {
	rawCommand := []byte(fmt.Sprintf(`<Atomic>%s</Atomic>`, profileBytes))
	cmd := new(mdm_types.SyncMLCmd)
	if err := xml.Unmarshal(rawCommand, cmd); err != nil {
		return nil, fmt.Errorf("unmarshalling profile: %w", err)
	}
	// set the CmdID for the <Atomic> command
	cmd.CmdID = mdm_types.CmdID{
		Value:               commandUUID,
		IncludeFleetComment: true,
	}
	// generate a CmdID for any nested <Replace>
	for i := range cmd.ReplaceCommands {
		cmd.ReplaceCommands[i].CmdID = mdm_types.CmdID{
			Value:               uuid.NewString(),
			IncludeFleetComment: true,
		}
	}

	// generate a CmdID for any nested <Add>
	for i := range cmd.AddCommands {
		cmd.AddCommands[i].CmdID = mdm_types.CmdID{
			Value:               uuid.NewString(),
			IncludeFleetComment: true,
		}
	}

	rawCommand, err := xml.Marshal(cmd)
	if err != nil {
		return nil, fmt.Errorf("marshalling command: %w", err)
	}

	command := &fleet.MDMWindowsCommand{
		CommandUUID: commandUUID,
		RawCommand:  rawCommand,
		// Atomic commands don't have a Target element.
		TargetLocURI: "",
	}

	return command, nil
}

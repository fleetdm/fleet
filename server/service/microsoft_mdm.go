package service

import (
	"bytes"
	"context"
	"crypto/md5" //nolint:gosec // Windows MDM Auth uses MD5
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"html/template"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/ee/server/service/scep"
	"github.com/fleetdm/fleet/v4/pkg/fleetdbase"
	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"
	mdmlifecycle "github.com/fleetdm/fleet/v4/server/mdm/lifecycle"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/fleetdm/fleet/v4/server/mdm/microsoft/syncml"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service/osquery_utils"
	"github.com/fleetdm/fleet/v4/server/variables"
	mysql_driver "github.com/go-sql-driver/mysql"

	mdm_types "github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
)

const maxRequestLogSize = 10240

// devDetailSMBIOSSerialNumberURI is the OMA-DM LocURI for the device's SMBIOS serial number.
const devDetailSMBIOSSerialNumberURI = "./DevDetail/Ext/Microsoft/SMBIOSSerialNumber"

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
			// We log the request body for debug by using an error implementing ErrWithInternal interface.
			return ctxerr.Wrap(ctx, &fleet.BadRequestError{
				Message:     "unmarshalling soap mdm request: " + err.Error(),
				InternalErr: fmt.Errorf("request: %s", truncateString(string(reqBytes), maxRequestLogSize)),
			})
		}
	}

	return nil
}

type SoapResponseContainer struct {
	Data *fleet.SoapResponse
	Err  error
}

func (r SoapResponseContainer) Error() error { return r.Err }

// HijackRender writes the response header and the RAW HTML output
func (r SoapResponseContainer) HijackRender(ctx context.Context, w http.ResponseWriter) {
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

func (r SyncMLResponseMsgContainer) Error() error { return r.Err }

// HijackRender writes the response header and the RAW HTML output
func (r SyncMLResponseMsgContainer) HijackRender(ctx context.Context, w http.ResponseWriter) {
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

func (req MDMWebContainer) Error() error { return req.Err }

// HijackRender writes the response header and the RAW HTML output
func (req MDMWebContainer) HijackRender(ctx context.Context, w http.ResponseWriter) {
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

func (r MDMAuthContainer) Error() error { return r.Err }

// HijackRender writes the response header and the RAW XML output
func (r MDMAuthContainer) HijackRender(ctx context.Context, w http.ResponseWriter) {
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

// getSTSAuthContent Returns STS auth content
func getSTSAuthContent(data string) mdm_types.Errorer {
	return MDMAuthContainer{
		Data: &data,
		Err:  nil,
	}
}

// getSoapResponseFault Returns a SoapResponse with a SoapFault on its body
func getSoapResponseFault(relatesTo string, soapFault *mdm_types.SoapFault) mdm_types.Errorer {
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
func NewApplicationProvisioningData(mdmEndpoint string, username string, secret string) mdm_types.Characteristic {
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
			newParm("AAUTHNAME", username, ""),
			newParm("AAUTHSECRET", secret, ""),
			newParm("AAUTHDATA", "nonce", ""), // We don't care about setting the first round nonce, as when the device checks in we will prompt the credentials and pass a new nonce.
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
func mdmMicrosoftDiscoveryEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (mdm_types.Errorer, error) {
	req := request.(*SoapRequestContainer).Data

	// Process the discovery request using the Service method which handles validation, logging, and response generation
	response, err := svc.ProcessMDMMicrosoftDiscovery(ctx, req)
	if err != nil {
		soapFault := svc.GetAuthorizedSoapFault(ctx, syncml.SoapErrorMessageFormat, mdm_types.MDEDiscovery, err)
		return getSoapResponseFault(req.GetMessageID(), soapFault), nil
	}

	return SoapResponseContainer{
		Data: response,
		Err:  nil,
	}, nil
}

// isValidAppru validates that appru is a valid URL with an allowed scheme.
// It returns true if appru is a valid URL with http, https, or ms-app scheme.
func isValidAppru(appru string) bool {
	parsed, err := url.Parse(appru)
	if err != nil {
		return false
	}

	return slices.Contains([]string{"http", "https", "ms-app"}, parsed.Scheme)
}

// mdmMicrosoftAuthEndpoint handles the Security Token Service (STS) implementation
func mdmMicrosoftAuthEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (mdm_types.Errorer, error) {
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

	// Validate that appru is a valid URL
	if !isValidAppru(appru) {
		return getSTSAuthContent(""), fmt.Errorf("non-URL appru parameter attempted: %q", appru)
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
func mdmMicrosoftPolicyEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (mdm_types.Errorer, error) {
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
func mdmMicrosoftEnrollEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (mdm_types.Errorer, error) {
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
func mdmMicrosoftManagementEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (mdm_types.Errorer, error) {
	reqSyncML := request.(*SyncMLReqMsgContainer).Data

	// Checking first if incoming SyncML message is valid and returning error if this is not the case
	if err := reqSyncML.IsValidMsg(); err != nil {
		soapFault := svc.GetAuthorizedSoapFault(ctx, syncml.SoapErrorMessageFormat, mdm_types.MSMDM, err)
		return getSoapResponseFault(reqSyncML.SyncHdr.MsgID, soapFault), nil
	}

	// Getting the MS-MDM response message
	resSyncML, err := svc.GetMDMWindowsManagementResponse(ctx, reqSyncML, request.(*SyncMLReqMsgContainer).Certs)
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
func mdmMicrosoftTOSEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (mdm_types.Errorer, error) {
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

// hasAuthorizedAzureAudience reports whether any audience value in an Entra-issued JWT is authorized for Windows
// automatic enrollment. An audience is authorized if either:
//   - it equals a configured Entra application client ID (v2 access tokens, whose `aud` is the app's client ID, a
//     GUID), compared case-insensitively after trimming
//   - it parses as a URL whose host (host:port) matches serverHost (v1 access tokens, whose `aud` is the Fleet server URL).
//
// The client-ID (v2) path is additive: it does not change the v1 server-URL behavior. An audience that is neither a
// known client ID nor a URL is ignored, so a token with multiple audiences is authorized if any one of them matches.
// serverHost is the expected host including any port; callers pass url.URL.Host.
func hasAuthorizedAzureAudience(audiences []string, serverHost string, clientIDs []string) bool {
	clientIDSet := make(map[string]struct{}, len(clientIDs))
	for _, id := range clientIDs {
		clientIDSet[strings.ToLower(strings.TrimSpace(id))] = struct{}{}
	}
	for _, aud := range audiences {
		// v2 token: `aud` is the application client ID (a GUID).
		if _, ok := clientIDSet[strings.ToLower(strings.TrimSpace(aud))]; ok {
			return true
		}
		// v1 token: `aud` is a URL whose host must match the Fleet server URL's. The audience may have multiple values
		// and not everything in it will be a URL, and that's OK. Compare the full host (host:port) case-insensitively:
		// per RFC 3986 the host is case-insensitive, but the port must match so a token for a different port on the same
		// hostname is not authorized.
		audURL, err := url.Parse(aud)
		if err != nil {
			continue
		}
		if audURL.Host != "" && strings.EqualFold(audURL.Host, serverHost) {
			return true
		}
	}
	return false
}

// hasAuthorizedAzureTenant reports whether the token's tenant (the `tid` claim) matches one of the configured Entra
// tenant IDs. The comparison is case-insensitive after trimming: Entra emits `tid` in lower-case, but validation
// accepts configured tenant IDs in either case (it does not normalize them), so a tenant ID pasted with upper-case
// hex digits must still authorize enrollment instead of silently failing to match the lower-cased claim.
func hasAuthorizedAzureTenant(tenantIDs []string, tokenTenant string) bool {
	tokenTenant = strings.TrimSpace(tokenTenant)
	if tokenTenant == "" {
		return false
	}
	for _, id := range tenantIDs {
		if strings.EqualFold(strings.TrimSpace(id), tokenTenant) {
			return true
		}
	}
	return false
}

// authBinarySecurityToken checks if the provided token is valid. For programmatic enrollment, it
// returns the orbit node key and host uuid. For automatic enrollment, it returns only the UPN (the
// host uuid will be an empty string).
func (svc *Service) authBinarySecurityToken(ctx context.Context, authToken *fleet.HeaderBinarySecurityToken) (claim string, hostUUID string, enrollType fleet.WindowsMDMEnrollType, err error) {
	if authToken == nil {
		return "", "", 0, errors.New("authToken is empty")
	}

	err = authToken.IsValidToken()
	if err != nil {
		return "", "", 0, errors.New("authToken is not valid")
	}

	// Tokens that were generated by enrollment client
	if authToken.IsDeviceToken() {

		// Getting the Binary Security Token Payload
		binSecToken, err := NewBinarySecurityTokenPayload(authToken.Content)
		if err != nil {
			return "", "", 0, fmt.Errorf("token creation error %v", err)
		}

		// Validating the Binary Security Token Payload
		err = binSecToken.IsValidToken()
		if err != nil {
			return "", "", 0, fmt.Errorf("invalid token data %v", err)
		}

		// Validating the Binary Security Token Type used on Programmatic Enrollments
		if binSecToken.Type == mdm_types.WindowsMDMProgrammaticEnrollmentType {
			host, err := svc.ds.LoadHostByOrbitNodeKey(ctx, binSecToken.Payload.OrbitNodeKey)
			if err != nil {
				return "", "", 0, fmt.Errorf("host data cannot be found %v", err)
			}
			if host == nil {
				return "", "", 0, errors.New("host not found for orbit node key")
			}

			mdmInfo, err := svc.ds.GetHostMDM(ctx, host.ID)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return "", "", 0, errors.New("unable to retrieve host mdm info")
			}

			// This ensures that only hosts that are eligible for Windows enrollment can be enrolled
			if !isEligibleForWindowsMDMEnrollment(host, mdmInfo) {
				return "", "", 0, errors.New("host is not elegible for Windows MDM enrollment")
			}

			// No errors, token is authorized
			return binSecToken.Payload.OrbitNodeKey, host.UUID, fleet.WindowsMDMEnrollTypeProgrammatic, nil
		}

		// Validating the Binary Security Token Type used on Automatic Enrollments (returned by STS Auth Endpoint)
		if binSecToken.Type == mdm_types.WindowsMDMAutomaticEnrollmentType {
			upnToken, err := svc.wstepCertManager.GetSTSAuthTokenUPNClaim(binSecToken.Payload.AuthToken)
			if err != nil {
				return "", "", 0, ctxerr.Wrap(ctx, err, "issue retrieving UPN from Auth token")
			}

			// No errors, token is authorized
			return upnToken, "", fleet.WindowsMDMEnrollTypeAutomatic, nil
		}
	}

	// Validating the Binary Security Token Type used on Automatic Enrollments
	if authToken.IsAzureJWTToken() {
		appConfig, err := svc.ds.AppConfig(ctx)
		if err != nil {
			return "", "", 0, ctxerr.Wrap(ctx, err, "retrieving app config for auth token validation")
		}

		entraTenantIDs := appConfig.MDM.WindowsEntraTenantIDs.Value
		if len(entraTenantIDs) == 0 {
			return "", "", 0, ctxerr.New(ctx, "no entra tenant IDs configured for automatic enrollment")
		}
		expectedURL := appConfig.ServerSettings.ServerURL
		expectedURLParsed, err := url.Parse(expectedURL)
		if err != nil {
			return "", "", 0, ctxerr.Wrap(ctx, err, "parsing server URL for auth token validation")
		}

		// Validate the JWT Auth token by retreving its claims
		tokenData, err := microsoft_mdm.GetAzureAuthTokenClaims(ctx, authToken.Content)
		if err != nil {
			return "", "", 0, fmt.Errorf("binary security token claim failed: %v", err)
		}

		if !hasAuthorizedAzureAudience(tokenData.Audience, expectedURLParsed.Host, appConfig.MDM.WindowsEntraClientIDs.Value) {
			// Log bad audiences here for debugging.
			svc.logger.ErrorContext(ctx, "unexpected token audience in AzureAD Binary Security Token",
				"expected_host", expectedURLParsed.Host,
				"configured_client_ids", strings.Join(appConfig.MDM.WindowsEntraClientIDs.Value, ","),
				"token_audiences", strings.Join(tokenData.Audience, ","),
			)
			return "", "", 0, ctxerr.Errorf(ctx, "token audience is not authorized")
		}
		if !hasAuthorizedAzureTenant(entraTenantIDs, tokenData.TenantID) {
			svc.logger.ErrorContext(ctx, "unexpected token tenant in AzureAD Binary Security Token",
				"token_tenant", tokenData.TenantID,
			)
			return "", "", 0, ctxerr.New(ctx, "token tenant is not authorized")
		}

		// No errors, token is authorized
		return tokenData.UPN, "", fleet.WindowsMDMEnrollTypeAutomatic, nil
	}

	return "", "", 0, ctxerr.New(ctx, "token is not authorized")
}

// ProcessMDMMicrosoftDiscovery handles the Discovery message validation and response
func (svc *Service) ProcessMDMMicrosoftDiscovery(ctx context.Context, req *fleet.SoapRequest) (*fleet.SoapResponse, error) {
	// Checking first if Discovery message is valid and returning error if this is not the case
	if err := req.IsValidDiscoveryMsg(); err != nil {
		// Log the raw XML request for debugging invalid messages
		svc.logger.DebugContext(ctx, "invalid discover message",
			"err", err.Error(),
			"request_xml", string(req.Raw),
		)
		return nil, err
	}

	// Getting the DiscoveryResponse message
	discoveryResponseMsg, err := svc.GetMDMMicrosoftDiscoveryResponse(ctx, req.Body.Discover.Request.EmailAddress)
	if err != nil {
		return nil, err
	}

	// Embedding the DiscoveryResponse message inside of a SoapResponse
	response, err := NewSoapResponse(discoveryResponseMsg, req.GetMessageID())
	if err != nil {
		return nil, err
	}

	return &response, nil
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
	_, _, _, err := svc.authBinarySecurityToken(ctx, authToken)
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
	userID, hostUUID, enrollType, err := svc.authBinarySecurityToken(ctx, authToken)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "validate binary security token")
	}

	// Removing the device if already MDM enrolled
	err = svc.removeWindowsDeviceIfAlreadyMDMEnrolled(ctx, secTokenMsg)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "device enroll check")
	}

	// Getting the device provisioning information in the form of a WapProvisioningDoc
	deviceProvisioning, credentialsHash, err := svc.getDeviceProvisioningInformation(ctx, secTokenMsg)
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
	err = svc.storeWindowsMDMEnrolledDevice(ctx, userID, hostUUID, enrollType, secTokenMsg, credentialsHash)
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
	enrolledDevice, requestAuthState, err := svc.isTrustedRequest(ctx, reqSyncML, reqCerts)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "management request is not trusted")
	}

	// Token is authorized
	svc.authz.SkipAuthorization(ctx)

	if requestAuthState == RequestAuthStateRekey {
		// If we signalled to rekey the device, we short-circuit into a rekey flow.
		return svc.rekeyWindowsDevice(ctx, reqSyncML)
	}

	// Getting the management response message
	resSyncMLmsg, err := svc.getManagementResponse(ctx, reqSyncML, enrolledDevice, requestAuthState)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "management response message")
	}

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

type requestAuthState int

const (
	RequestAuthStateUntrusted requestAuthState = iota
	RequestAuthStateUnauthorized
	RequestAuthStateChallenge
	RequestAuthStateRekey
	RequestAuthStateTrusted
)

// isTrustedRequest checks if the incoming request was sent from an MDM-enrolled
// device. It returns the matched enrollment (when the device was found), the
// auth state, and an error only when the request is malformed or otherwise
// cannot be processed. Expected non-trusted outcomes (for example
// RequestAuthStateChallenge or RequestAuthStateUnauthorized) are reported via
// the returned auth state and may return a nil error. The returned enrolled
// device may be nil when the state is RequestAuthStateUntrusted.
func (svc *Service) isTrustedRequest(ctx context.Context, reqSyncML *fleet.SyncML, reqCerts []*x509.Certificate) (*fleet.MDMWindowsEnrolledDevice, requestAuthState, error) {
	if reqSyncML == nil {
		return nil, RequestAuthStateUntrusted, fleet.NewInvalidArgumentError("syncml req message", "message is not present")
	}

	// Checking if calling request is coming from an already MDM enrolled device
	deviceID, err := reqSyncML.GetSource()
	if err != nil || deviceID == "" {
		return nil, RequestAuthStateUntrusted, fmt.Errorf("invalid SyncML message %w", err)
	}

	enrolledDevice, err := svc.ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, deviceID)
	if err != nil || enrolledDevice == nil {
		return nil, RequestAuthStateUntrusted, errors.New("device was not MDM enrolled")
	}

	// Check if TLS certs contains device ID on its common name
	if len(reqCerts) > 0 {
		for _, reqCert := range reqCerts {
			if strings.Contains(reqCert.Subject.CommonName, deviceID) {
				return enrolledDevice, RequestAuthStateTrusted, nil
			}
		}
	}

	if !enrolledDevice.CredentialsAcknowledged && enrolledDevice.CredentialsHash == nil {
		// Device has not gotten new credentials, rekey the device only once
		return enrolledDevice, RequestAuthStateRekey, nil
	}

	if reqSyncML.SyncHdr.Cred == nil {
		// No certs, but no credentials present - challenge the device
		return enrolledDevice, RequestAuthStateChallenge, nil
	}

	// Extract the last nonce used to generate the credentials hash
	nonce, err := svc.keyValueStore.Get(ctx, fleet.WindowsMDMAuthNoncePrefix+deviceID)
	if err != nil {
		return nil, RequestAuthStateUntrusted, ctxerr.Wrap(ctx, err, "get device nonce from kv store")
	}

	if nonce == nil || *nonce == "" {
		// Challenge the device if nonce is missing, which will send a new nonce and store it
		return enrolledDevice, RequestAuthStateChallenge, nil
	}

	// Credentials are present, validate it
	credFormat := reqSyncML.SyncHdr.Cred.Meta.Format
	credType := reqSyncML.SyncHdr.Cred.Meta.Type
	credData := reqSyncML.SyncHdr.Cred.Data

	if credFormat == nil || credType == nil || credFormat.Content == nil || credType.Content == nil {
		return nil, RequestAuthStateUntrusted, errors.New("SyncML credentials format or type is missing")
	}

	if *credFormat.Content != syncml.AuthB64Format || *credType.Content != syncml.AuthMD5 {
		return nil, RequestAuthStateUntrusted, errors.New("SyncML credentials format or type is invalid")
	}

	// MD5 auth digest, which includes (username:password):nonce
	// Where username:password is hashed and b64 encoded, and then further hased with the nonce and finally b64 encoded for transport
	// https://www.openmobilealliance.org/release/DM/V1_2_1-20080617-A/OMA-TS-DM_Security-V1_2_1-20080617-A.pdf Chaper (5.3)
	receivedDigestHash, err := base64.StdEncoding.DecodeString(credData)
	if err != nil {
		return nil, RequestAuthStateUntrusted, ctxerr.Wrap(ctx, err, "decode SyncML credentials data")
	}

	encodedCredentialsHash := base64.StdEncoding.EncodeToString(*enrolledDevice.CredentialsHash)
	expectedDigest := fmt.Sprintf("%s:%s", encodedCredentialsHash, *nonce)
	expectedDigestHash := md5.Sum([]byte(expectedDigest)) //nolint:gosec // Windows MDM Auth uses MD5

	if !bytes.Equal(receivedDigestHash, expectedDigestHash[:]) {
		// Credentials do not match what we expect
		return enrolledDevice, RequestAuthStateUnauthorized, nil
	}

	// We verified the username, password and nonce match what we expect, so we can ack the rekeyed credentials
	if !enrolledDevice.CredentialsAcknowledged {
		err = svc.ds.MDMWindowsAcknowledgeEnrolledDeviceCredentials(ctx, enrolledDevice.MDMDeviceID)
		if err != nil {
			return nil, RequestAuthStateUntrusted, ctxerr.Wrap(ctx, err, "mark device credentials as acknowledged")
		}
	}

	return enrolledDevice, RequestAuthStateTrusted, nil
}

func (svc *Service) rekeyWindowsDevice(ctx context.Context, reqSyncML *fleet.SyncML) (*fleet.SyncML, error) {
	if reqSyncML == nil {
		return nil, fleet.NewInvalidArgumentError("syncml req message", "message is not present")
	}

	// Getting the device ID from the SyncML message
	deviceID, err := reqSyncML.GetSource()
	if err != nil || deviceID == "" {
		return nil, fmt.Errorf("invalid SyncML message %w", err)
	}

	enrolledDevice, err := svc.ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, deviceID)
	if err != nil || enrolledDevice == nil {
		return nil, errors.New("device was not MDM enrolled")
	}

	username := deviceID
	password := uuid.NewString()
	credentialsHash := md5.Sum(fmt.Appendf([]byte{}, "%s:%s", username, password)) //nolint:gosec // Windows MDM Auth uses MD5

	// Store the new credentials hash and mark that the device has not acknowledged them yet
	err = svc.ds.MDMWindowsUpdateEnrolledDeviceCredentials(ctx, enrolledDevice.MDMDeviceID, credentialsHash[:])
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "update enrolled device credentials")
	}

	// Queue two replace commands to update the credentials on the device
	accountUid := "0x0000800cABF8CA7D87C0FA21BB21B896AE2C8468CA07710326415228ACF2392EA1327077"
	usernameReplace := newSyncMLCmdText("Replace", fmt.Sprintf("./SyncML/DMAcc/%s/AppAuth/CLCRED/AAuthName", accountUid), username)
	usernameReplace.CmdID = mdm_types.CmdID{
		IncludeFleetComment: true,
		Value:               "rekey-credentials-username",
	}
	passwordReplace := newSyncMLCmdText("Replace", fmt.Sprintf("./SyncML/DMAcc/%s/AppAuth/CLCRED/AAuthSecret", accountUid), password)
	passwordReplace.CmdID = mdm_types.CmdID{
		IncludeFleetComment: true,
		Value:               "rekey-credentials-password",
	}

	// Get the incoming MessageID
	reqMessageID, err := reqSyncML.GetMessageID()
	if err != nil {
		return nil, fmt.Errorf("get incoming msg: %w", err)
	}

	// We only create a response here, and does not persist it to the DB to avoid saving the secrets in plain text
	return svc.createResponseSyncML(ctx, reqSyncML, []*mdm_types.SyncMLCmd{
		NewSyncMLCmdStatus(reqMessageID, "0", syncml.SyncMLHdrName, syncml.CmdStatusOK), // We need to ack the incoming message to send rekey commands
		usernameReplace,
		passwordReplace,
	})
}

// fleetdPresenceGracePeriod is how far before the current MDM enrollment's created_at an orbit/osquery check-in still
// counts as "fleetd present". It absorbs a relaxed agent check-in interval.
const fleetdPresenceGracePeriod = 90 * time.Second

// isFleetdPresentOnDevice checks if the device requires Fleetd to be deployed.
// The enrolled device is resolved upstream (by isTrustedRequest) and threaded
// in to avoid a duplicate lookup on every session-start alert.
func (svc *Service) isFleetdPresentOnDevice(ctx context.Context, enrolledDevice *fleet.MDMWindowsEnrolledDevice) (bool, error) {
	// If user identity is a MS-MDM UPN it means that the device was enrolled through user-driven flow
	// This means that fleetd might not be installed
	if microsoft_mdm.IsValidUPN(enrolledDevice.MDMEnrollUserID) {
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
					// Require a recent orbit/osquery check-in (seen_time at/after the enrollment, minus a grace window)
					// so a host that has not checked in since re-enrollment gets the fleetd install (re)enqueued.
					// seen_time only moves forward, so once fleetd checks in after install this stays true.
					isPresent = orbitInfo.Version != "" &&
						!host.SeenTime.Before(enrolledDevice.CreatedAt.Add(-fleetdPresenceGracePeriod))
				}
			}
		}
		return isPresent, nil
	}

	return true, nil
}

// generateWindowsEUAToken returns a Fleet-signed EUA token for the given Windows
// MDM device ID if the device enrolled with a valid Azure UPN
func (svc *Service) generateWindowsEUAToken(ctx context.Context, deviceID string) string {
	if svc.wstepCertManager == nil {
		return ""
	}
	device, err := svc.ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, deviceID)
	if err != nil {
		svc.logger.ErrorContext(ctx, "unable to fetch windows mdm enrollment for EUA token generation", "err", err, "device_id", deviceID)
		return ""
	}
	if device == nil || !microsoft_mdm.IsValidUPN(device.MDMEnrollUserID) {
		return ""
	}
	token, err := svc.wstepCertManager.NewEUAToken(device.MDMEnrollUserID, deviceID)
	if err != nil {
		svc.logger.ErrorContext(ctx, "unable to generate EUA token for fleetd install", "err", err, "device_id", deviceID)
		return ""
	}
	return token
}

func (svc *Service) enqueueInstallFleetdCommand(ctx context.Context, deviceID string) error {
	secrets, err := svc.ds.GetEnrollSecrets(ctx, nil)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting enroll secrets")
	}

	if len(secrets) == 0 {
		svc.logger.WarnContext(ctx, "unable to find a global enroll secret to install fleetd")
		return nil
	}

	// it's okay to skip the installation if we're not able to retrieve the
	// metadata, we don't want to completely error the SyncML transaction
	// and we'll try again the next time the host checks in
	fleetdMetadata, err := fleetdbase.GetMetadata()
	if err != nil {
		svc.logger.WarnContext(ctx, "unable to get fleetd-base metadata")
		return nil
	}

	appCfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting app config")
	}
	fleetURL := appCfg.ServerSettings.ServerURL
	globalEnrollSecret := secrets[0].Secret
	// Fleet-internal CmdID: the Add is injected inline and is never its own tracked queue command. The Exec command is
	// the important one, and we only track that.
	addCommandUUID := fleet.FleetInternalCmdIDPrefix + "fleetd-install-add"
	execCommandUUID := uuid.NewString()

	euaTokenArg := ""
	if token := svc.generateWindowsEUAToken(ctx, deviceID); token != "" {
		euaTokenArg = ` EUA_TOKEN="` + token + `"`
	}

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
					<CommandLine>/quiet FLEET_URL="` + fleetURL + `" FLEET_SECRET="` + globalEnrollSecret + `" ENABLE_SCRIPTS="True"` + euaTokenArg + `</CommandLine>
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

	// Deliver the Add and Exec as a SINGLE command so they ride in one SyncML body with Add textually before Exec. As
	// two separate queued commands they can be reordered when they are applied in the same second, we must manually
	// guarantee the ordering.
	rawCombinedCmd := slices.Concat(rawAddCmd, rawExecCmd)
	fleetdInstallCmd := &fleet.MDMWindowsCommand{
		CommandUUID:  execCommandUUID,
		RawCommand:   rawCombinedCmd,
		TargetLocURI: syncml.FleetdWindowsInstallerGUID,
	}
	if err := svc.ds.MDMWindowsInsertCommandForHosts(ctx, []string{deviceID}, fleetdInstallCmd); err != nil {
		return ctxerr.Wrap(ctx, err, "insert command to install fleetd")
	}

	return nil
}

// Alerts Handlers

// New session Alert Handler
// This handler will return an protocol command to install an MSI on a new session from unenrolled device
func (svc *Service) processNewSessionAlert(ctx context.Context, messageID string, enrolledDevice *fleet.MDMWindowsEnrolledDevice, cmd mdm_types.ProtoCmdOperation) error {
	// Checking if fleetd is present on the device
	fleetdPresent, err := svc.isFleetdPresentOnDevice(ctx, enrolledDevice)
	if err != nil {
		return err
	}

	if !fleetdPresent {
		return svc.enqueueInstallFleetdCommand(ctx, enrolledDevice.MDMDeviceID)
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
func (svc *Service) processIncomingAlertsCommands(ctx context.Context, messageID string, enrolledDevice *fleet.MDMWindowsEnrolledDevice, cmd mdm_types.ProtoCmdOperation) error {
	if cmd.Cmd.Data == nil {
		return errors.New("invalid alert command")
	}

	// gathering the incoming Alert ID
	alertID := *cmd.Cmd.Data

	switch alertID {
	case syncml.CmdAlertClientInitiatedManagement:
		return svc.processNewSessionAlert(ctx, messageID, enrolledDevice, cmd)
	case syncml.CmdAlertServerInitiatedManagement:
		return svc.processNewSessionAlert(ctx, messageID, enrolledDevice, cmd)
	case syncml.CmdAlertGeneric:
		return svc.processGenericAlert(ctx, messageID, enrolledDevice.MDMDeviceID, cmd)
	}

	return nil
}

// tryLinkUnlinkedEnrollmentFromDevDetail scans an incoming SyncML message for a Results command answering an earlier
// Get for ./DevDetail/Ext/Microsoft/SMBIOSSerialNumber, and if found, looks up the host by hardware_serial and links
// the enrollment to it. Returns true if a linkage was established.
//
// Any error is non-fatal: this is the primary linkage path but osquery direct-ingest still runs as a backstop, and the
// Get is reinjected on every subsequent session until linkage succeeds.
func (svc *Service) tryLinkUnlinkedEnrollmentFromDevDetail(ctx context.Context, enrolledDevice *fleet.MDMWindowsEnrolledDevice, reqMsg *fleet.SyncML) bool {
	if reqMsg == nil {
		return false
	}
	// A SyncML Results command can carry multiple Items; the device may include the SMBIOSSerialNumber alongside other
	// DevDetail values in a single response. Scan every item in every Results command, not just Items[0], or a serial
	// returned in a later position is missed and the Get gets reinjected forever.
	var serial string
scan:
	for _, op := range reqMsg.GetOrderedCmds() {
		if op.Verb != mdm_types.CmdResults {
			continue
		}
		for _, item := range op.Cmd.Items {
			if item.Source == nil || *item.Source != devDetailSMBIOSSerialNumberURI {
				continue
			}
			if item.Data == nil {
				continue
			}
			candidate := strings.TrimSpace(item.Data.Content)
			// Skip empty or well-known placeholder serials (whitebox/consumer BIOS defaults, un-sysprepped VM
			// templates). Such devices fall back to the osquery directIngestMDMDeviceIDWindows backstop, which links by
			// the unique MDMDeviceID instead.
			if fleet.IsPlaceholderHardwareSerial(candidate) {
				continue
			}
			serial = candidate
			break scan
		}
	}
	if serial == "" {
		return false
	}
	// Require the primary DB: the hosts row may have been inserted seconds ago by osquery enroll, just before this
	// first SyncML management session arrived. A replica-lag read would return a false NotFound and delay linkage to a
	// later session, defeating the point of this path.
	host, err := svc.ds.WindowsHostLiteByHardwareSerial(ctxdb.RequirePrimary(ctx, true), serial)
	if err != nil {
		if !fleet.IsNotFound(err) {
			svc.logger.ErrorContext(ctx, "windows mdm: host lookup by serial failed", "err", err, "device_id", enrolledDevice.MDMDeviceID)
			ctxerr.Handle(ctx, err)
		}
		// NotFound means the host hasn't enrolled in osquery yet (hosts row not created yet); we'll retry next session.
		return false
	}
	updated, err := osquery_utils.LinkWindowsHostMDMEnrollment(ctx, svc.logger, svc.ds, host.ID, host.UUID, enrolledDevice.MDMDeviceID)
	if err != nil {
		svc.logger.ErrorContext(ctx, "windows mdm: link by DevDetail failed", "err", err, "device_id", enrolledDevice.MDMDeviceID)
		ctxerr.Handle(ctx, err)
		return false
	}
	// Always refresh in-memory HostUUID after a successful link attempt.
	enrolledDevice.HostUUID = host.UUID
	return updated
}

// processIncomingMDMCmds process the incoming message from the device
// It will return the list of operations that need to be sent to the device.
// enrolledDevice is the enrollment resolved upstream by isTrustedRequest and
// is threaded in so downstream paths (saveResponse, alert handlers) do not
// re-query mdm_windows_enrollments for the same row.
func (svc *Service) processIncomingMDMCmds(ctx context.Context, enrolledDevice *fleet.MDMWindowsEnrolledDevice, reqMsg *fleet.SyncML, requestAuthState requestAuthState) ([]*fleet.SyncMLCmd, error) {
	var responseCmds []*fleet.SyncMLCmd
	deviceID := enrolledDevice.MDMDeviceID

	saveResponse := func(topLevelExists []string) error {
		enrichedSyncML := fleet.NewEnrichedSyncML(reqMsg)
		if enrichedSyncML.HasCommands() {
			result, err := svc.ds.MDMWindowsSaveResponse(ctx, enrolledDevice, enrichedSyncML, topLevelExists)
			if err != nil {
				return fmt.Errorf("store incoming msgs: %w", err)
			}

			if result != nil && result.WipeFailed != nil {
				host, err := svc.ds.HostByIdentifier(ctx, result.WipeFailed.HostUUID)
				if err != nil {
					svc.logger.WarnContext(ctx, "failed to look up host for wipe_failed_host activity",
						"host_uuid", result.WipeFailed.HostUUID, "err", err)
				} else {
					if err := svc.NewActivity(ctx, nil, fleet.ActivityTypeWipeFailedHost{
						HostID:          host.ID,
						HostDisplayName: host.DisplayName(),
						HostPlatform:    host.Platform,
					}); err != nil {
						svc.logger.WarnContext(ctx, "failed to create wipe_failed_host activity",
							"host_id", host.ID, "err", err)
					}
				}
			}

			if result != nil && result.WipeSucceeded != nil {
				host, err := svc.ds.HostByIdentifier(ctx, result.WipeSucceeded.HostUUID)
				if err != nil {
					return ctxerr.Wrap(ctx, err, "wipe succeeded: get host by identifier")
				}
				if _, err := svc.ds.BatchCancelAllHostUpcomingActivities(ctx, host.ID); err != nil {
					return ctxerr.Wrap(ctx, err, "cancel upcoming activities after wipe")
				}
			}
		}
		return nil
	}

	// Get the incoming MessageID
	reqMessageID, err := reqMsg.GetMessageID()
	if err != nil {
		return nil, fmt.Errorf("get incoming msg: %w", err)
	}

	if requestAuthState == RequestAuthStateChallenge || requestAuthState == RequestAuthStateUnauthorized {
		nonce := uuid.NewString() // using UUID as nonce since it has 122 bits of entropy
		base64Nonce := base64.StdEncoding.EncodeToString([]byte(nonce))
		err := svc.keyValueStore.Set(ctx, fleet.WindowsMDMAuthNoncePrefix+deviceID, nonce, 5*time.Minute)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "store device nonce in kv store")
		}

		status := syncml.CmdStatusAuthenticationRequired
		if requestAuthState == RequestAuthStateUnauthorized {
			status = syncml.CmdStatusInvalidCredentials
		}

		ackMsg := NewSyncMLCmdStatus(reqMessageID, "0", syncml.SyncMLHdrName, status)
		ackMsg.Chal = &fleet.SyncMLChallenge{
			Meta: fleet.ChallengeMeta{
				NextNonce: fleet.MetaAttr{
					XMLNS:   syncml.SyncMLMetaNamespace,
					Content: &base64Nonce,
				},
				Meta: fleet.Meta{
					Type: &fleet.MetaAttr{
						XMLNS:   syncml.SyncMLMetaNamespace,
						Content: ptr.String(syncml.AuthMD5),
					},
					Format: &fleet.MetaAttr{
						XMLNS:   syncml.SyncMLMetaNamespace,
						Content: ptr.String(syncml.AuthB64Format),
					},
				},
			},
		}

		responseCmds = append(responseCmds, ackMsg)
		err = saveResponse([]string{})
		if err != nil {
			return nil, err
		}
		return responseCmds, nil
	}

	if requestAuthState != RequestAuthStateTrusted {
		return nil, errors.New("untrusted request cannot be processed")
	}

	// Acknowledge the message header
	// msgref is always 0 for the header
	if err = reqMsg.IsValidHeader(); err == nil {
		// We always return 200 here, we could also return 212 to indicate we don't need the credentials every time,
		// but our logic is built around it being present on each request
		ackMsg := NewSyncMLCmdStatus(reqMessageID, "0", syncml.SyncMLHdrName, syncml.CmdStatusOK)
		responseCmds = append(responseCmds, ackMsg)
	}

	// If this enrollment isn't linked yet, try to link it using a DevDetail/SMBIOSSerialNumber Result the device may
	// have included in this message (in response to a Get we sent during a previous session). If the link succeeds,
	// enrolledDevice.HostUUID is updated in memory so downstream callers (ESP coordination, saveResponse, etc.) in this
	// same request see the linked state instead of waiting for the next session.
	if enrolledDevice.HostUUID == "" {
		svc.tryLinkUnlinkedEnrollmentFromDevDetail(ctx, enrolledDevice, reqMsg)
	}

	// List of CmdRef that need to be re-issued as <Replace> commands
	// However it's a list of nested Command IDs, and not something we can use directly for command_uuid in windows_mdm_commands
	alreadyExistsCmdIDs := []string{}

	// Iterate over the operations and process them
	for _, protoCMD := range reqMsg.GetOrderedCmds() {
		if protoCMD.Cmd.Data != nil && *protoCMD.Cmd.Data == "418" && protoCMD.Cmd.CmdRef != nil {
			// 418 = Already exists, and indicate that an <Add> failed due to the item already existing on the device
			// We need to re-issue a <Replace> command for this item
			alreadyExistsCmdIDs = append(alreadyExistsCmdIDs, *protoCMD.Cmd.CmdRef)
		}

		// Alerts, Results and Status don't require a status response
		switch protoCMD.Verb {
		case mdm_types.CmdAlert:
			err := svc.processIncomingAlertsCommands(ctx, reqMessageID, enrolledDevice, protoCMD)
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

	// We gain an additional benefit of doing this here, which is since we processIncoming before grabbing Pending CMDs,
	// we get this new resend to send back to the Windows MDM protocol, so it gets processed almost immediately, instead
	// of having to wait for the next check-in.
	topLevelExists, err := handleResendingAlreadyExistsCommands(ctx, svc, alreadyExistsCmdIDs, deviceID)
	if err != nil {
		return nil, err
	}

	err = saveResponse(topLevelExists)
	if err != nil {
		return nil, err
	}

	// If this enrollment is still unlinked after processing the incoming message, ask the device for its SMBIOS serial
	// on the next round-trip. This is the primary linkage mechanism: it lets the server populate
	// mdm_windows_enrollments.host_uuid in one SyncML round-trip instead of waiting for osquery's distributed-read
	// cycle (~10s) to backfill via directIngestMDMDeviceIDWindows. The Get is idempotent and reinjected each session
	// until linkage succeeds; osquery direct-ingest remains as a backstop for hosts that never reply to DevDetail.
	//
	// The Get uses a stable fleet-internal CmdID instead of a fresh UUID so that MDMWindowsSaveResponse can recognize
	// and skip it when checking for "unmatched Windows MDM commands".
	if enrolledDevice.HostUUID == "" {
		get := newSyncMLCmdGet(devDetailSMBIOSSerialNumberURI)
		get.CmdID = mdm_types.CmdID{Value: fleet.FleetInternalCmdIDPrefix + "devdetail-smbios-serial"}
		responseCmds = append(responseCmds, get)
	}

	return responseCmds, nil
}

func handleResendingAlreadyExistsCommands(ctx context.Context, svc *Service, alreadyExistsCmdIDs []string, deviceID string) ([]string, error) {
	commands, err := svc.ds.GetWindowsMDMCommandsForResending(ctx, deviceID, alreadyExistsCmdIDs)
	if err != nil {
		return nil, fmt.Errorf("get commands for resending: %w", err)
	}

	// We use a new list here to track the top-level (atomic) commandID so that we can skip it being re-triggered in the saveResponse flow.
	topLevelExists := []string{}
	for _, cmd := range commands {
		if !strings.Contains(string(cmd.RawCommand), "<Add>") {
			// Only Add commands can be re-issued as Replace
			continue
		}

		// Copy value, and avoid referencing the old values
		newCmd := &fleet.MDMWindowsCommand{
			CommandUUID:  cmd.CommandUUID,
			TargetLocURI: cmd.TargetLocURI,
		}
		newCmd.RawCommand = make([]byte, len(cmd.RawCommand))
		copy(newCmd.RawCommand, cmd.RawCommand)

		newCmd.RawCommand = []byte(strings.ReplaceAll(string(newCmd.RawCommand), "<Add>", "<Replace>"))
		newCmd.RawCommand = []byte(strings.ReplaceAll(string(newCmd.RawCommand), "</Add>", "</Replace>"))

		// Generate a new top-level command UUID, so we can track it separately
		newCmd.CommandUUID = uuid.NewString()
		newCmd.RawCommand = []byte(strings.ReplaceAll(string(newCmd.RawCommand), cmd.CommandUUID, newCmd.CommandUUID))

		// We use the NEW top-level command UUID here, instead of the old to let the save response track and save the 418 response
		// however we don't want it to correlate anything in this run with this new command we are adding, so we populate the list
		// which will be used to skip it, in the save response.
		topLevelExists = append(topLevelExists, newCmd.CommandUUID)
		err = svc.ds.ResendWindowsMDMCommand(ctx, deviceID, newCmd, cmd)
		if err != nil {
			return nil, fmt.Errorf("re-insert command for resending: %w", err)
		}
	}
	return topLevelExists, nil
}

// getPendingMDMCmds returns the list of pending MDM commands for the given enrollment, plus onlyPollCmdsPending: true
// when everything still pending (if anything) is an internal poll-schedule Replace.
func (svc *Service) getPendingMDMCmds(ctx context.Context, enrollmentID uint) ([]*mdm_types.SyncMLCmd, bool, error) {
	pendingCmds, err := svc.ds.MDMWindowsGetPendingCommands(ctx, enrollmentID)
	if err != nil {
		return nil, false, fmt.Errorf("getting incoming cmds %w", err)
	}

	// Converting the pending commands to its target SyncML types
	var cmds []*mdm_types.SyncMLCmd
	onlyPollCmdsPending := true
	for _, pendingCmd := range pendingCmds {
		isPollCmd := pendingCmd.TargetLocURI == syncml.DMClientPollIntervalLocURI
		if !isPollCmd {
			onlyPollCmdsPending = false
		}
		// The raw MDM command may contain a $FLEET_SECRET_XXX, the value of which should never be exposed or stored unencrypted.
		rawCommandWithSecret, err := svc.ds.ExpandEmbeddedSecrets(ctx, string(pendingCmd.RawCommand))
		if err != nil {
			// This error should never happen since we validate the presence of needed secrets on profile upload.
			return nil, false, ctxerr.Wrap(ctx, err, "expanding embedded secrets for Windows pending commands")
		}
		parsedCmds, err := fleet.UnmarshallMultiTopLevelXMLProfile([]byte(rawCommandWithSecret))
		if err != nil {
			logging.WithErr(ctx, ctxerr.Wrap(ctx, err, "getPendingMDMCmds syncML cmd creation"))
			continue
		}
		for _, pcmd := range parsedCmds {
			cmds = append(cmds, &pcmd)
		}
	}

	return cmds, onlyPollCmdsPending, nil
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

// getManagementResponse returns a valid SyncML response message. The
// enrolledDevice is the enrollment resolved upstream by isTrustedRequest; it is
// non-nil whenever requestAuthState is RequestAuthStateTrusted.
func (svc *Service) getManagementResponse(ctx context.Context, reqMsg *fleet.SyncML, enrolledDevice *fleet.MDMWindowsEnrolledDevice, requestAuthState requestAuthState) (*mdm_types.SyncML, error) {
	if reqMsg == nil {
		return nil, fleet.NewInvalidArgumentError("syncml req message", "message is not present")
	}

	// Get the DeviceID
	deviceID, err := reqMsg.GetSource()
	if err != nil || deviceID == "" {
		return nil, fmt.Errorf("invalid SyncML message %w", err)
	}

	// Process the incoming MDM protocol commands and get the response MDM protocol commands
	resIncomingCmds, err := svc.processIncomingMDMCmds(ctx, enrolledDevice, reqMsg, requestAuthState)
	if err != nil {
		return nil, fmt.Errorf("message processing error %w", err)
	}

	var resPendingCmds, espCmds []*mdm_types.SyncMLCmd

	if requestAuthState == RequestAuthStateTrusted {
		// Outside the Autopilot ESP, reconcile the DMClient poll schedule: relax it for hosts whose fleetd can be woken on demand, so we can stop
		// the aggressive default 1-minute poll. This enqueues the Replace (if needed) BEFORE draining the command queue below, so it ships in this same
		// session.
		if enrolledDevice.AwaitingConfiguration == fleet.WindowsMDMAwaitingConfigurationNone {
			if err := svc.reconcileWindowsMDMPollSchedule(ctx, enrolledDevice); err != nil {
				return nil, fmt.Errorf("poll schedule reconcile error: %w", err)
			}
		}

		// Process the pending operations and get the MDM response protocol commands
		pendingCmds, onlyPollCmdsPending, err := svc.getPendingMDMCmds(ctx, enrolledDevice.ID)
		if err != nil {
			return nil, fmt.Errorf("message processing error %w", err)
		}
		resPendingCmds = pendingCmds

		// Per-session has_pending_commands maintenance: refresh the denormalized flag only when everything still
		// pending (if anything) is an internal poll-schedule Replace, which the flag's definition excludes. While
		// non-poll commands remain queued the flag provably stays 1 (set by the enqueue paths), so mid-session
		// messages skip the recompute entirely.
		//
		// The HasPendingCommands gate (as loaded at session start) keeps idle check-ins at zero writer-side statements:
		// when the flag was already 0 and nothing is pending, there is no 1 -> 0 transition to record. A flag stranded
		// at 1 by an aborted session still self-heals - it loads as 1 on the next session and the refresh runs. A
		// mid-session enqueue that flips 0 -> 1 after this row was loaded needs no refresh either: its commands are
		// genuinely pending, so 1 is already correct. Best-effort: a failed refresh only delays the flag flip until
		// the next session, so log and continue rather than failing the device's response.
		if onlyPollCmdsPending && enrolledDevice.HasPendingCommands {
			if err := svc.ds.MDMWindowsRefreshHasPendingCommands(ctx, enrolledDevice.ID); err != nil {
				svc.logger.ErrorContext(ctx, "refresh windows mdm has_pending_commands", "err", err,
					"enrollment_id", enrolledDevice.ID)
				ctxerr.Handle(ctx, err)
			}
		}

		// Build ESP (Enrollment Status Page) commands for Windows Autopilot devices. Only run for trusted requests
		// so we don't leak ESP state to unauthenticated devices.
		if enrolledDevice.AwaitingConfiguration != fleet.WindowsMDMAwaitingConfigurationNone {
			espCmds, err = svc.getESPCommands(ctx, enrolledDevice)
			if err != nil {
				return nil, fmt.Errorf("ESP commands error: %w", err)
			}
		}
	}

	allCmds := make([]*mdm_types.SyncMLCmd, 0, len(resIncomingCmds)+len(resPendingCmds)+len(espCmds))
	allCmds = append(allCmds, resIncomingCmds...)
	allCmds = append(allCmds, resPendingCmds...)
	allCmds = append(allCmds, espCmds...)

	// Create the response SyncML message
	msg, err := svc.createResponseSyncML(ctx, reqMsg, allCmds)
	if err != nil {
		return nil, fmt.Errorf("message syncML creation error %w", err)
	}

	return msg, nil
}

const (
	// windowsMDMFastPollIntervalMinutes mirrors NewDMClientProvisioningData's provisioned IntervalForFirstSetOfRetries: the aggressive poll
	// used until a host is known to support on-demand sync.
	windowsMDMFastPollIntervalMinutes = "1"
	// windowsMDMRelaxedPollIntervalMinutes is the steady poll for hosts woken on demand by fleetd. 480 minutes (8 hours) matches Intune's
	// default check-in cadence. Steady-state command latency comes from the on-demand wake; this interval only bounds the fallback latency
	// if a wake ever fails, so a longer value trades a larger worst-case fallback for less polling load.
	windowsMDMRelaxedPollIntervalMinutes = "480"
)

// buildPollScheduleCommand builds an MDMWindowsCommand that Replaces the DMClient Poll interval with the value for the given schedule
// (relaxed vs the aggressive default). It is enqueued like any other Windows MDM command, so the command queue handles delivery, automatic
// re-delivery until acknowledged, and ack recording.
func buildPollScheduleCommand(relaxed bool) (*fleet.MDMWindowsCommand, error) {
	interval := windowsMDMFastPollIntervalMinutes
	if relaxed {
		interval = windowsMDMRelaxedPollIntervalMinutes
	}
	cmdUUID := uuid.NewString()
	cmd := newSyncMLCmdInt(fleet.CmdReplace, syncml.DMClientPollIntervalLocURI, interval)
	cmd.CmdID = mdm_types.CmdID{Value: cmdUUID}
	rawCommand, err := xml.Marshal(cmd)
	if err != nil {
		return nil, fmt.Errorf("marshal poll schedule command: %w", err)
	}
	return &fleet.MDMWindowsCommand{
		CommandUUID:  cmdUUID,
		RawCommand:   rawCommand,
		TargetLocURI: syncml.DMClientPollIntervalLocURI,
	}, nil
}

// reconcileWindowsMDMPollSchedule relaxes (or restores) the device's DMClient poll schedule based on whether its fleetd can be woken on
// demand. Capable hosts get a relaxed poll, so steady-state command delivery is driven by the on-demand wake instead of frequent polling
// (the server-load reduction); non-capable hosts stay on the fast poll.
//
// It enqueues the poll Replace through the standard Windows MDM command queue. poll_schedule_relaxed records only the
// INTENDED schedule, so a command is enqueued exactly once per intended change (and excluded from the
// pending-command/wake signal, so tuning the poll does not itself request a wake).
func (svc *Service) reconcileWindowsMDMPollSchedule(ctx context.Context, device *fleet.MDMWindowsEnrolledDevice) error {
	// A capable fleetd (one advertising CapabilityWindowsMDMSync, persisted to fleetd_sync_capable by the orbit-config endpoint) can be woken
	// on demand, so relax its poll; otherwise keep the fast default. The flag is read straight off the already-loaded enrolled-device row, so
	// this hot management path does no extra DB lookups.
	desiredRelaxed := device.FleetdSyncCapable
	if desiredRelaxed == device.PollScheduleRelaxed {
		// Intended schedule already matches; any still-undelivered Replace is re-sent by the command queue.
		return nil
	}

	cmd, err := buildPollScheduleCommand(desiredRelaxed)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "build windows MDM poll schedule command")
	}
	// Enqueue the Replace and record the intended schedule atomically.
	if err := svc.ds.MDMWindowsEnqueuePollScheduleCommand(ctx, device.MDMDeviceID, device.ID, cmd, desiredRelaxed); err != nil {
		return ctxerr.Wrap(ctx, err, "enqueue windows MDM poll schedule command")
	}
	svc.logger.DebugContext(ctx, "reconciled Windows MDM poll schedule", "device_id", device.MDMDeviceID, "relaxed", desiredRelaxed)
	return nil
}

// getESPCommands dispatches ESP coordination for a Windows Autopilot device.
//
// For awaiting_configuration=Pending: send hold commands to block the device at the ESP during OOBE, then transition
// to Active once orbit links the host UUID.
//
// For awaiting_configuration=Active: run the wait gates (profiles + setup-experience software) and release or block
// the device when ready, including the 3-hour timeout.
func (svc *Service) getESPCommands(ctx context.Context, device *fleet.MDMWindowsEnrolledDevice) ([]*mdm_types.SyncMLCmd, error) {
	switch device.AwaitingConfiguration {
	case fleet.WindowsMDMAwaitingConfigurationPending:
		return svc.handleESPHoldOrTransition(ctx, device)
	case fleet.WindowsMDMAwaitingConfigurationActive:
		return svc.handleESPRelease(ctx, device)
	default:
		return nil, nil
	}
}

// handleESPHoldOrTransition handles awaiting_configuration=Pending.
// Before orbit links the host UUID: sends hold commands to block the device at
// the ESP. These are idempotent and sent on every management session.
// After orbit links: transitions to Active so the release check can begin.
func (svc *Service) handleESPHoldOrTransition(ctx context.Context, device *fleet.MDMWindowsEnrolledDevice) ([]*mdm_types.SyncMLCmd, error) {
	providerID := syncml.DocProvisioningAppProviderID

	if device.HostUUID == "" {
		// Orbit hasn't enrolled yet. Send DMClient FirstSyncStatus hold commands to
		// activate the ESP and block the device during OOBE. These must be sent
		// immediately -- if we wait for orbit, OOBE progresses past the ESP window.
		svc.logger.DebugContext(ctx, "ESP: sending hold commands", "device_id", device.MDMDeviceID)
		policyProviderURI := fmt.Sprintf("./Device/Vendor/MSFT/EnrollmentStatusTracking/DevicePreparation/PolicyProviders/%s", providerID)
		holdCmds := []*mdm_types.SyncMLCmd{
			// Create the user-scope DMClient Provider tree before any user-scope writes. Verified on Win11 26200
			// in the Entra-join-during-OOBE flow without an Autopilot deployment profile: the user-scope
			// Provider/Fleet node does not exist by default. The Add commands here create the user-scope Provider
			// node and its FirstSyncStatus child so the release-phase ServerHasFinishedProvisioning=true write can
			// land.
			//
			// Hold commands are sent on every management session during the Pending phase, so these Adds will
			// repeat. The device returns SyncML status 418 ("Already Exists") on the second and later calls. That
			// per-command status does not fail the SyncML session and does not affect the hold-cycle Replace
			// commands that follow it; the Pending phase only lasts a few minutes (until orbit links and we
			// transition to Active, after which hold commands stop). Net effect: a small amount of expected 418
			// noise during OOBE in exchange for not having to track per-enrollment "have I sent the Add" state.
			newSyncMLCmdNode(fleet.CmdAdd, fmt.Sprintf("./User/Vendor/MSFT/DMClient/Provider/%s", providerID)),
			newSyncMLCmdNode(fleet.CmdAdd, fmt.Sprintf("./User/Vendor/MSFT/DMClient/Provider/%s/FirstSyncStatus", providerID)),
			// SkipDeviceStatusPage and SkipUserStatusPage are both set to false so the ESP page is visible during
			// OOBE.
			newSyncMLCmdBool(fleet.CmdReplace, fmt.Sprintf("./Device/Vendor/MSFT/DMClient/Provider/%s/FirstSyncStatus/SkipDeviceStatusPage", providerID), "false"),
			newSyncMLCmdBool(fleet.CmdReplace, fmt.Sprintf("./Device/Vendor/MSFT/DMClient/Provider/%s/FirstSyncStatus/SkipUserStatusPage", providerID), "false"),
			// BlockInStatusPage=1: block user, show "Reset PC" button on failure.
			// Per DMClient CSP docs: 1=Reset PC, 2=Try Again, 4=Continue Anyway.
			// We pre-configure Reset here so it's already set when/if the failure UI renders.
			newSyncMLCmdInt(fleet.CmdReplace, fmt.Sprintf("./Device/Vendor/MSFT/DMClient/Provider/%s/FirstSyncStatus/BlockInStatusPage", providerID), "1"),
			// AllowCollectLogsButton: pre-configure Collect Logs button so it's visible on both progress and failure pages.
			newSyncMLCmdBool(fleet.CmdReplace, fmt.Sprintf("./Device/Vendor/MSFT/DMClient/Provider/%s/FirstSyncStatus/AllowCollectLogsButton", providerID), "true"),
			// CustomErrorText: pre-configure with the timeout-flavored error so that if the OS-side ESP failure UI
			// renders before Fleet's Active-phase finalize runs (e.g. when TimeOutUntilSyncFailure expires, or when the
			// device is stuck in Pending and Windows itself fails the ESP), the user sees Fleet's text instead of
			// Windows's default "We ran into a problem with one of the following setup steps" message. Active-phase
			// finalize will Replace this with ESPSoftwareFailureErrorText if a software install actually failed; in the
			// pure-timeout case the value here is already correct.
			newSyncMLCmdText(fleet.CmdReplace, fmt.Sprintf("./Device/Vendor/MSFT/DMClient/Provider/%s/FirstSyncStatus/CustomErrorText", providerID), microsoft_mdm.ESPTimeoutErrorText),
			// TimeOutUntilSyncFailure is in minutes per DMClient CSP (range 60-1440).
			newSyncMLCmdInt(fleet.CmdReplace, fmt.Sprintf("./Device/Vendor/MSFT/DMClient/Provider/%s/FirstSyncStatus/TimeOutUntilSyncFailure", providerID), fmt.Sprintf("%d", microsoft_mdm.ESPTimeoutSeconds/60)),
			// PolicyProviders/{providerID} is a dynamic node -- must be created with Add before its children can be set with Replace.
			newSyncMLCmdNode(fleet.CmdAdd, policyProviderURI),
			// DevicePreparation InstallationState=1 signals "installing" to hold ESP.
			newSyncMLCmdInt(fleet.CmdReplace, policyProviderURI+"/InstallationState", "1"),
		}
		for _, cmd := range holdCmds {
			cmd.CmdID = mdm_types.CmdID{Value: uuid.New().String()}
		}
		return holdCmds, nil
	}

	// Orbit has linked the host UUID. Transition to Active so handleESPRelease
	// can finalize and release the device.
	svc.logger.DebugContext(ctx, "ESP: orbit linked, transitioning to active", "device_id", device.MDMDeviceID, "host_uuid", device.HostUUID)
	transitioned, err := svc.ds.SetMDMWindowsAwaitingConfiguration(ctx, device.MDMDeviceID,
		fleet.WindowsMDMAwaitingConfigurationPending, fleet.WindowsMDMAwaitingConfigurationActive)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "set awaiting configuration to active")
	}
	if !transitioned {
		return nil, nil
	}

	// Mark DevicePreparation as completed to advance the ESP phase.
	dpCmd := newSyncMLCmdInt(fleet.CmdReplace,
		fmt.Sprintf("./Device/Vendor/MSFT/EnrollmentStatusTracking/DevicePreparation/PolicyProviders/%s/InstallationState", providerID), "3")
	dpCmd.CmdID = mdm_types.CmdID{Value: uuid.New().String()}
	return []*mdm_types.SyncMLCmd{dpCmd}, nil
}

// handleESPRelease handles awaiting_configuration=Active. It waits for all profiles and setup experience items to reach
// a terminal state, then finalizes the device down one of three paths:
//
//   - hard block (require_all_software_windows=true and any item failed, or the 3-hour timeout was hit): the ESP
//     failure screen is shown with only the "Reset device" recovery option, and remaining items are canceled.
//   - soft block (require_all_software_windows=false and an item failed, or the 3-hour timeout was hit): the ESP
//     failure screen is shown with a "Continue anyway" option so the user can proceed to the desktop and install the
//     missing software via self-service. It lists the failed software by name when any failed, otherwise (a timeout
//     with nothing failed) shows the timeout message. Still-pending items are cancelled only on the timeout path.
//   - release (no failure and no timeout): the device proceeds to login.
func (svc *Service) handleESPRelease(ctx context.Context, device *fleet.MDMWindowsEnrolledDevice) ([]*mdm_types.SyncMLCmd, error) {
	if device.HostUUID == "" {
		return nil, nil
	}

	// Check timeout first: if we've exceeded the 3-hour window, finalize regardless of profile/software status.
	timedOut := device.AwaitingConfigurationAt != nil && time.Since(*device.AwaitingConfigurationAt) > time.Duration(microsoft_mdm.ESPTimeoutSeconds)*time.Second
	if timedOut {
		svc.logger.WarnContext(ctx, "ESP: timeout reached", "device_id", device.MDMDeviceID)
	}

	// hasSoftwareFailure tracks setup-experience software failures only.
	var hasSoftwareFailure bool

	// failedSoftwareNames collects the names of the failed setup-experience items
	var failedSoftwareNames []string

	// recordSoftwareFailure marks a failed setup-experience item
	recordSoftwareFailure := func(r *fleet.SetupExperienceStatusResult) {
		hasSoftwareFailure = true
		name := r.Name
		if r.DisplayName != "" {
			name = r.DisplayName
		}
		if name != "" {
			failedSoftwareNames = append(failedSoftwareNames, name)
		}
	}

	// loadHost lazily fetches the host (writer-routed) and memoizes for the rest of this checkin. Writer routing guards
	// two replica-lag races: (1) spurious notFound during the brief gap between orbit's host registration and the next
	// management session, and (2) stale team_id if a host transferred teams mid-enrollment, which could let
	// require_all_software_windows be read from the wrong team and bypass the gate.
	//
	// We cache the full HostLite (not just team_id) because Stage 3 also needs OsqueryHostID: setup_experience_status_results
	// is keyed by fleet.HostUUIDForSetupExperience, which on Windows resolves to OsqueryHostID -- not the Fleet host UUID
	// stored on the MDM enrollment record.
	var (
		cachedHost *fleet.HostLite
		hostLoaded bool
	)
	loadHost := func() (*fleet.HostLite, error) {
		if hostLoaded {
			return cachedHost, nil
		}
		host, err := svc.ds.HostLiteByIdentifier(ctxdb.RequirePrimary(ctx, true), device.HostUUID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "lookup host for ESP")
		}
		cachedHost = host
		hostLoaded = true
		return host, nil
	}

	// hostTeamID resolves the host's team_id (0 for "no team / global"), used to scope setup-experience queries.
	hostTeamID := func() (uint, error) {
		host, err := loadHost()
		if err != nil {
			return 0, err
		}
		if host.TeamID != nil {
			return *host.TeamID, nil
		}
		return 0, nil
	}

	// setupExperienceHostUUID returns the identifier used as setup_experience_status_results.host_uuid for this host.
	// On Windows that's OsqueryHostID per fleet.HostUUIDForSetupExperience; if it's missing for some reason we fall back
	// to the Fleet host UUID.
	setupExperienceHostUUID := func() (string, error) {
		host, err := loadHost()
		if err != nil {
			return "", err
		}
		if host.OsqueryHostID != nil && *host.OsqueryHostID != "" {
			return *host.OsqueryHostID, nil
		}
		return device.HostUUID, nil
	}

	// listSetupExperienceResults fetches the host's setup-experience status rows. They are keyed by the Windows
	// setup-experience host UUID (OsqueryHostID per fleet.HostUUIDForSetupExperience) and team-scoped so the rows carry
	// the team's custom software display names (used in the soft-block message).
	listSetupExperienceResults := func() ([]*fleet.SetupExperienceStatusResult, error) {
		seHostUUID, err := setupExperienceHostUUID()
		if err != nil {
			return nil, err
		}
		teamID, err := hostTeamID()
		if err != nil {
			return nil, err
		}
		results, err := svc.ds.ListSetupExperienceResultsByHostUUID(ctx, seHostUUID, teamID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "list setup experience results for ESP")
		}
		return results, nil
	}

	// loadRequireAll memoizes the host -> team's require_all_software_windows lookup. It is consulted at most twice
	// per checkin: once inside Stage 3 (to decide whether to short-circuit on a software failure) and once below
	// (to drive the block/release decision).
	var (
		cachedRequireAll bool
		requireAllLoaded bool
	)
	loadRequireAll := func() (bool, error) {
		if requireAllLoaded {
			return cachedRequireAll, nil
		}
		host, err := loadHost()
		if err != nil {
			return false, err
		}
		if host.TeamID == nil {
			ac, err := svc.ds.AppConfig(ctx)
			if err != nil {
				return false, ctxerr.Wrap(ctx, err, "get app config for ESP finalization")
			}
			cachedRequireAll = ac.MDM.MacOSSetup.RequireAllSoftwareWindows
		} else {
			team, err := svc.ds.TeamLite(ctx, *host.TeamID)
			if err != nil {
				return false, ctxerr.Wrap(ctx, err, "get team for ESP finalization")
			}
			cachedRequireAll = team.Config.MDM.MacOSSetup.RequireAllSoftwareWindows
		}
		requireAllLoaded = true
		return cachedRequireAll, nil
	}

	if !timedOut {
		// Profile delivery has two stages:
		//
		// 1. Profiles configured for the host's team but not yet queued by the profile reconciler. Rather than just
		//    listing them and blocking the release, run the per-host reconciler now: it queues any missing desired
		//    profiles so stage 2 sees them as in-flight rows in the same checkin. This closes the race where a
		//    freshly-enrolled host would otherwise wait for the cron's next pass.
		// 2. Profiles queued (rows in host_mdm_windows_profiles) but not yet delivered to a terminal state
		//    (GetHostMDMWindowsProfiles).
		//
		// We never release while profiles are pending at either stage. Each management checkin re-evaluates both.

		// Stage 1: queue any desired profiles the reconciler hasn't picked up yet. An error blocks the release (we
		// can't know whether desired work is still unqueued); the next checkin retries.
		if err := ReconcileWindowsProfilesForEnrollingHost(ctx, svc.ds, svc.logger, device.HostUUID); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "reconcile profiles for ESP release check")
		}

		// Stage 2: profiles queued but still in-flight (pending/verifying). Read from the primary: it must see the
		// rows stage 1 just wrote, and even when stage 1 queued nothing, "nothing to queue" means every desired
		// profile already has a row on the PRIMARY (possibly written by the previous checkin under a second ago). A
		// lagging replica could be missing those rows entirely, and a row this read can't see is indistinguishable
		// from no work, which would release the device early.
		profiles, err := svc.ds.GetHostMDMWindowsProfiles(ctxdb.RequirePrimary(ctx, true), device.HostUUID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "get host profiles for ESP release check")
		}
		for _, p := range profiles {
			if p.OperationType != fleet.MDMOperationTypeInstall {
				continue
			}
			// Wait for terminal state (verified or failed) before proceeding. Profile failures are NOT propagated to
			// the block decision -- see hasSoftwareFailure comment above.
			if p.Status == nil || (*p.Status != fleet.MDMDeliveryVerified && *p.Status != fleet.MDMDeliveryFailed) {
				return nil, nil
			}
		}

		// Stage 3: setup experience software/scripts still running. Orbit initiates setup experience during startup when it
		// is enabled for the current OS/flags, which enqueues items into setup_experience_status_results.
		results, err := listSetupExperienceResults()
		if err != nil {
			return nil, err
		}
		svc.logger.DebugContext(ctx, "ESP: setup experience check",
			"host_uuid", device.HostUUID, "results_count", len(results))

		// Empty results is ambiguous: it can mean "no setup experience is configured for this team" (safe to release) or
		// "setup is configured but orbit hasn't called SetupExperienceInit yet" (must wait). Orbit links the host UUID to
		// the MDM enrollment independently of when it calls init, so on the first Active checkin after link we can hit
		// this race. Disambiguate by checking whether items are configured for the host's team.
		if len(results) == 0 {
			teamID, err := hostTeamID()
			if err != nil {
				return nil, err
			}
			hasItems, err := svc.ds.HasWindowsSetupExperienceItemsForTeam(ctx, teamID)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "check setup experience items configured for team")
			}
			if hasItems {
				svc.logger.DebugContext(ctx, "ESP: setup experience configured but not yet initialized; waiting",
					"host_uuid", device.HostUUID)
				return nil, nil
			}
		}

		// Single pass: collect hasSoftwareFailure and "are any rows still in flight". We deliberately do NOT bail
		// early on the first non-terminal row -- we need to know whether any failure exists in the result set
		// before deciding whether to wait, so we can short-circuit when require_all=true and we've already
		// observed a failure. IsTerminalStatus() returns true for success/failure; Cancelled is also a completed
		// outcome (must not block release).
		anyInFlight := false
		for _, r := range results {
			switch r.Status {
			case fleet.SetupExperienceStatusFailure:
				recordSoftwareFailure(r)
			case fleet.SetupExperienceStatusSuccess, fleet.SetupExperienceStatusCancelled:
				// terminal, nothing to record
			default:
				// pending / running
				anyInFlight = true
			}
		}
		if anyInFlight {
			if !hasSoftwareFailure {
				svc.logger.DebugContext(ctx, "ESP: waiting for in-flight setup experience items",
					"host_uuid", device.HostUUID, "results_count", len(results))
				return nil, nil
			}
			// A software install has already failed. If require_all=true, the device is going to block.
			requireAll, err := loadRequireAll()
			if err != nil {
				return nil, err
			}
			if !requireAll {
				svc.logger.DebugContext(ctx, "ESP: software failure observed but require_all=false; waiting for rest",
					"host_uuid", device.HostUUID, "results_count", len(results))
				return nil, nil
			}
			svc.logger.InfoContext(ctx, "ESP: software failure with require_all=true; blocking install",
				"host_uuid", device.HostUUID)
		}
	}

	// We're past the wait gate or timed out. Look up require_all_software_windows (memoized via loadRequireAll if
	// Stage 3 already consulted it) to decide between block and release. Return the error on lookup failure so the
	// device stays Active and retries on the next management session: failing open here would permanently bypass
	// the policy after the Active->None transition below.
	requireAll, err := loadRequireAll()
	if err != nil {
		return nil, err
	}

	// On a timeout with require_all=false, scan for software failures so the finalize surfaces them instead of
	// releasing silently: the timeout path skipped Stage 3, so failures observed before the timeout are otherwise
	// invisible. require_all=true keeps its existing behavior (hard block with timeout text) and is deliberately not
	// re-scanned here -- a real failure under require_all=true would already have blocked on an earlier checkin via the
	// Stage 3 in-flight short-circuit, before the 3-hour window elapsed.
	if timedOut && !requireAll {
		results, err := listSetupExperienceResults()
		if err != nil {
			return nil, err
		}
		for _, r := range results {
			if r.Status == fleet.SetupExperienceStatusFailure {
				recordSoftwareFailure(r)
			}
		}
	}

	shouldBlock := requireAll && (timedOut || hasSoftwareFailure)
	shouldWarn := !requireAll && (timedOut || hasSoftwareFailure)

	// Build commands for the response.
	provID := syncml.DocProvisioningAppProviderID
	var cmds []*mdm_types.SyncMLCmd
	switch {
	case shouldBlock:
		// Pick the user-facing error text to surface on the failure UI. Software failure takes precedence over timeout
		// because it's more actionable; pure timeout (no software failed) uses the timeout text so the user sees an
		// accurate reason.
		errorText := microsoft_mdm.ESPTimeoutErrorText
		if hasSoftwareFailure {
			errorText = microsoft_mdm.ESPSoftwareFailureErrorText
		}
		cmds = buildESPBlockCommands(provID, errorText, espBlockButtonsReset)
	case shouldWarn:
		// List the failed software when we have any; otherwise (require_all=false timeout with nothing failed) show the
		// timeout text. Both keep the "Continue anyway" option so the user is never stuck.
		errorText := microsoft_mdm.ESPTimeoutErrorText
		if hasSoftwareFailure {
			errorText = microsoft_mdm.ESPSoftwareFailureContinuableErrorText(failedSoftwareNames)
		}
		cmds = buildESPBlockCommands(provID, errorText, espBlockButtonsResetAndContinue)
	default:
		// Release path: device proceeds to login. We do not send CustomErrorText here because the failure UI never renders
		// on a release (no BlockInStatusPage, no forced timeout), so any error text would be dead state on the DMClient
		// node.
		cmds = buildESPReleaseCommands(provID)
	}

	// On timeout (regardless of require_all) and on software-failure+require_all=true, cancel any pending items. Run
	// this BEFORE the compare-and-swap (CAS) that commits awaiting_configuration=None at the bottom of this function,
	// so a transient cancel failure aborts the finalize cleanly -- otherwise we'd commit awaiting=None while leaving
	// non-terminal setup-experience rows behind, exactly the state cancellation is supposed to prevent.
	//
	// We must cancel both halves: the upcoming_activities queue AND the setup_experience_status_results status
	// table (so the UI and downstream queries see the cancelled state).
	//
	// Cancel ordering: upcoming_activities first, then status table. If we crash mid-loop, the next retry sees the same
	// status rows still pending, re-iterates, and will tolerate the now-deleted upcoming_activities row via IsNotFound.
	//
	// The canceled_setup_experience activity is emitted at two points:
	// - For software-failure with require_all=true: emitted by maybeCancelPendingSetupExperienceSteps in the
	//   software-install-result reporting path, referencing the failed software item.
	// - For pure-timeout (regardless of require_all): emitted below, referencing the first pending/running software
	//   item if one exists (the item that was in flight when the timeout fired). This deviates from macOS, which
	//   only emits on the require_all=true software-failure case, but matches the activity-feed expectation in
	//   issue #38785's test plan: "Timeout cancellation: canceled_setup_experience activity emitted on the timeout
	//   branch as well as the failure branch."
	if timedOut || (hasSoftwareFailure && requireAll) {
		seHostUUID, err := setupExperienceHostUUID()
		if err != nil {
			return nil, err
		}
		// We re-list statuses here rather than reusing Stage 3's `results`: that variable is scoped inside the
		// `if !timedOut` block and isn't visible here, AND the timeout path skipped Stage 3 entirely so there's
		// nothing to reuse on that branch. Hoisting the variable out to share it would tangle the two paths; one
		// extra DB read per finalize keeps this block self-contained.
		statuses, err := svc.ds.ListSetupExperienceResultsByHostUUID(ctx, seHostUUID, 0)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "list setup experience results for cancel")
		}
		host, err := loadHost()
		if err != nil {
			return nil, err
		}
		for _, s := range statuses {
			if s.Status != fleet.SetupExperienceStatusPending && s.Status != fleet.SetupExperienceStatusRunning {
				continue
			}
			var executionID string
			switch {
			case s.HostSoftwareInstallsExecutionID != nil:
				executionID = *s.HostSoftwareInstallsExecutionID
			case s.NanoCommandUUID != nil:
				executionID = *s.NanoCommandUUID
			case s.ScriptExecutionID != nil:
				executionID = *s.ScriptExecutionID
			default:
				continue
			}
			// Tolerate notFound: a previous attempt may have cancelled the upcoming_activities row before crashing
			// before the status table update, or another path (manual cancel, re-enrollment cleanup) may have removed
			// it concurrently. In either case the queue is already in the desired state.
			if _, err := svc.ds.CancelHostUpcomingActivity(ctx, host.ID, executionID); err != nil && !fleet.IsNotFound(err) {
				return nil, ctxerr.Wrap(ctx, err, "cancel upcoming setup experience activity")
			}
		}
		// CancelPendingSetupExperienceSteps is idempotent (status filter excludes terminal rows) so a retry on the
		// next session is safe.
		if err := svc.ds.CancelPendingSetupExperienceSteps(ctx, seHostUUID); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "cancel pending setup experience steps")
		}

		// Emit canceled_setup_experience for every timeout, which cancels whatever was still pending. It references the
		// first pending/running software item (in flight when the timeout fired), or empty fields if none was queued.
		// Gated on timedOut alone (not !hasSoftwareFailure): no duplication, since the require_all=true failure case
		// emits upstream in maybeCancelPendingSetupExperienceSteps and blocks before any timeout, while the
		// require_all=false failure case emits nowhere else.
		if timedOut {
			host, err := loadHost()
			if err != nil {
				return nil, err
			}
			var softwareTitle string
			var softwareTitleID uint
			for _, s := range statuses {
				if (s.Status == fleet.SetupExperienceStatusPending || s.Status == fleet.SetupExperienceStatusRunning) && s.IsForSoftware() {
					softwareTitle = s.Name
					if s.SoftwareTitleID != nil {
						softwareTitleID = *s.SoftwareTitleID
					}
					break
				}
			}
			if err := svc.NewActivity(ctx, nil, fleet.ActivityTypeCanceledSetupExperience{
				HostID:          host.ID,
				HostDisplayName: host.DisplayName(),
				SoftwareTitle:   softwareTitle,
				SoftwareTitleID: softwareTitleID,
			}); err != nil {
				return nil, ctxerr.Wrap(ctx, err, "creating canceled setup experience activity on timeout")
			}
		}
	}

	// Persist BEFORE the CAS. The persist is the dropped-response retry safety net; if it fails, we want to leave
	// awaiting_configuration=Active so the next management session retries the whole finalize from scratch.
	//
	// The persist is a single transactional batch (MDMWindowsInsertCommandsForHost) so a partial-fail-then-retry can't
	// leave orphan rows in the queue.
	//
	// On concurrent-CAS races (two checkins both reach this point) both callers persist with fresh UUIDs and only one
	// wins the CAS. The loser's rows are delivered later by the regular command queue, the device acks them as
	// idempotent Replaces of post-ESP-irrelevant DMClient nodes, and the queue clears -- no permanent leak, just brief
	// extra traffic.
	if err := svc.persistESPFinalCommands(ctx, device.HostUUID, cmds); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "persist ESP finalization commands")
	}

	// CAS Active -> None: only one concurrent checkin commits the finalize. Cancel and persist above ran for both
	// concurrent winners, but cancel is idempotent and persist's losers get harmlessly delivered as orphan Replaces.
	transitioned, err := svc.ds.SetMDMWindowsAwaitingConfiguration(ctx, device.MDMDeviceID,
		fleet.WindowsMDMAwaitingConfigurationActive, fleet.WindowsMDMAwaitingConfigurationNone)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "set awaiting configuration to none")
	}
	if !transitioned {
		// Another concurrent checkin already finalized.
		return nil, nil
	}

	svc.logger.InfoContext(ctx, "ESP: finalizing",
		"device_id", device.MDMDeviceID,
		"host_uuid", device.HostUUID,
		"timed_out", timedOut,
		"has_software_failure", hasSoftwareFailure,
		"require_all", requireAll,
		"blocking", shouldBlock,
		"soft_blocking", shouldWarn)

	return cmds, nil
}

// BlockInStatusPage values per Microsoft DMClient CSP docs (bit flags): 1=Reset PC, 2=Try Again, 4=Continue Anyway.
const (
	// espBlockButtonsReset shows only the "Reset PC" button: used for the hard block
	espBlockButtonsReset = "1"
	// espBlockButtonsResetAndContinue shows "Reset PC" and "Continue anyway" (1|4)
	espBlockButtonsResetAndContinue = "5"
)

// buildESPBlockCommands builds SyncML commands that put the device's ESP into a failed state showing errorText, a
// "Collect logs" button, and the recovery buttons selected by blockButtons (one of the espBlockButtons* constants).
func buildESPBlockCommands(provID, errorText, blockButtons string) []*mdm_types.SyncMLCmd {
	cmds := []*mdm_types.SyncMLCmd{
		// CustomErrorText: shown in the ESP failure UI as the failure reason.
		newSyncMLCmdText(fleet.CmdReplace,
			fmt.Sprintf("./Device/Vendor/MSFT/DMClient/Provider/%s/FirstSyncStatus/CustomErrorText", provID),
			errorText),
		// BlockInStatusPage: which recovery buttons the failure UI offers. Reset triggers an Autopilot wipe and
		// re-enrollment; Continue anyway (soft block only) lets the user proceed to the desktop.
		newSyncMLCmdInt(fleet.CmdReplace,
			fmt.Sprintf("./Device/Vendor/MSFT/DMClient/Provider/%s/FirstSyncStatus/BlockInStatusPage", provID),
			blockButtons),
		// AllowCollectLogsButton: show the "Collect logs" button so IT can gather diagnostics from the failure screen.
		newSyncMLCmdBool(fleet.CmdReplace,
			fmt.Sprintf("./Device/Vendor/MSFT/DMClient/Provider/%s/FirstSyncStatus/AllowCollectLogsButton", provID),
			"true"),
		// TimeOutUntilSyncFailure=1 (minute): force the ESP to time out and enter its failure state quickly. We
		// deliberately do NOT send ServerHasFinishedProvisioning=true here -- that would tell the ESP it succeeded
		// (Windows treats "server done + no expected items missing" as success and proceeds past the ESP). Instead we
		// rely on the timeout to trigger the failure UI, which then renders our BlockInStatusPage + CustomErrorText +
		// AllowCollectLogsButton.
		//
		// NOTE: the documented range for this node is 60-1440 minutes. Below the documented minimum, behavior is
		// technically undefined, but Windows builds we have tested honor the smaller value and time out in roughly the
		// configured number of minutes (verified empirically on Windows 11 23H2 / Autopilot). If a future Windows build
		// clamps to 60, the failure UI would take ~1 hour to appear instead of ~1 minute -- bad UX but the contract
		// (eventual failure UI) still holds.
		//
		// We tried the alternative documented in the DMClient CSP -- setting WasDeviceSuccessfullyProvisioned=0
		// followed by IsSyncDone=true, which the docs say should render the failure UI without any timeout. On
		// Win11 26200 the device acks both Replaces with status 200, but the Account-setup page stays at "Working
		// on it..." indefinitely (verified past 25 minutes). The OS-side ESP UI on this build appears to require
		// per-software-item progress state that Fleet cannot populate today; until that gap is closed (documented in
		// https://github.com/fleetdm/fleet/issues/43776), the timeout trigger above is the only mechanism that
		// reliably surfaces the failure UI for these enrollments.
		newSyncMLCmdInt(fleet.CmdReplace,
			fmt.Sprintf("./Device/Vendor/MSFT/DMClient/Provider/%s/FirstSyncStatus/TimeOutUntilSyncFailure", provID),
			"1"),
	}
	for _, cmd := range cmds {
		cmd.CmdID = mdm_types.CmdID{Value: uuid.New().String()}
	}
	return cmds
}

// buildESPReleaseCommands builds SyncML commands that release the device from the ESP. The release path advances
// DevicePreparation to "complete" and signals ServerHasFinishedProvisioning at both Device and User scopes to
// complete both the Device setup and Account setup phases of the ESP so Windows proceeds to login.
func buildESPReleaseCommands(provID string) []*mdm_types.SyncMLCmd {
	cmds := []*mdm_types.SyncMLCmd{
		newSyncMLCmdInt(fleet.CmdReplace,
			fmt.Sprintf("./Device/Vendor/MSFT/EnrollmentStatusTracking/DevicePreparation/PolicyProviders/%s/InstallationState", provID), "3"),
		newSyncMLCmdBool(fleet.CmdReplace,
			fmt.Sprintf("./Device/Vendor/MSFT/DMClient/Provider/%s/FirstSyncStatus/ServerHasFinishedProvisioning", provID), "true"),
		newSyncMLCmdBool(fleet.CmdReplace,
			fmt.Sprintf("./User/Vendor/MSFT/DMClient/Provider/%s/FirstSyncStatus/ServerHasFinishedProvisioning", provID), "true"),
	}
	for _, cmd := range cmds {
		cmd.CmdID = mdm_types.CmdID{Value: uuid.New().String()}
	}
	return cmds
}

// persistESPFinalCommands stores backup copies of every finalization command
// so the existing command-retry infrastructure can resend them if this
// response is dropped. The persisted commands intentionally reuse the same
// CmdIDs as the inline commands: when the device acks the inline send,
// MDMWindowsSaveResponse will match those CmdRefs against the persisted
// rows and clear them, so the backup is only resent if delivery actually
// failed. All commands are idempotent Replaces, so even a duplicate delivery
// is safe.
func (svc *Service) persistESPFinalCommands(ctx context.Context, hostUUID string, cmds []*mdm_types.SyncMLCmd) error {
	persistCmds := make([]*fleet.MDMWindowsCommand, 0, len(cmds))
	for _, cmd := range cmds {
		// Skip commands without a target URI -- shouldn't happen for the commands we build, but guard against nil-deref.
		targetURI := cmd.GetTargetURI()
		if targetURI == "" || len(cmd.Items) == 0 {
			continue
		}
		rawXML, err := xml.Marshal(cmd)
		if err != nil {
			// Marshal of a SyncMLCmd we just built is a deterministic code bug, not a transient failure. Returning an
			// error here would just loop forever (every retry hits the same bug); log + Handle
			wrapped := ctxerr.Wrap(ctx, err, "marshal ESP final command for persistence")
			svc.logger.ErrorContext(ctx, "ESP: failed to marshal final command for persistence",
				"err", wrapped, "target_uri", targetURI)
			ctxerr.Handle(ctx, wrapped)
			continue
		}
		persistCmds = append(persistCmds, &fleet.MDMWindowsCommand{
			CommandUUID:  cmd.CmdID.Value,
			RawCommand:   rawXML,
			TargetLocURI: targetURI,
		})
	}
	if len(persistCmds) == 0 {
		return nil
	}
	// Single transactional insert: either every backup row is committed or none.
	if err := svc.ds.MDMWindowsInsertCommandsForHost(ctx, hostUUID, persistCmds); err != nil {
		return ctxerr.Wrap(ctx, err, "persist ESP finalization commands")
	}
	return nil
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
	err = svc.ds.MDMWindowsDeleteEnrolledDeviceOnReenrollment(ctx, reqHWDeviceID)
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
func (svc *Service) getDeviceProvisioningInformation(ctx context.Context, secTokenMsg *fleet.RequestSecurityToken) (string, []byte, error) {
	reqDeviceID, err := GetContextItem(secTokenMsg, syncml.ReqSecTokenContextItemDeviceID)
	if err != nil {
		return "", nil, err
	}

	// Getting the HW DeviceID from the RequestSecurityToken msg
	reqHWDeviceID, err := GetContextItem(secTokenMsg, syncml.ReqSecTokenContextItemHWDevID)
	if err != nil {
		return "", nil, err
	}

	// Getting the EnrollmentType information from the RequestSecurityToken msg
	reqEnrollType, err := GetContextItem(secTokenMsg, syncml.ReqSecTokenContextItemEnrollmentType)
	if err != nil {
		return "", nil, err
	}

	// Getting the BinarySecurityToken from the RequestSecurityToken msg
	binSecurityTokenData, err := secTokenMsg.GetBinarySecurityTokenData()
	if err != nil {
		return "", nil, err
	}

	// Getting the BinarySecurityToken type from the RequestSecurityToken msg
	binSecurityTokenType, err := secTokenMsg.GetBinarySecurityTokenType()
	if err != nil {
		return "", nil, err
	}

	// Getting the client CSR request from the device
	clientCSR, err := microsoft_mdm.GetClientCSR(binSecurityTokenData, binSecurityTokenType)
	if err != nil {
		return "", nil, err
	}

	// Getting the signed, DER-encoded certificate bytes and its uppercased, hex-endcoded SHA1 fingerprint
	rawSignedCertDER, rawSignedCertFingerprint, err := svc.SignMDMMicrosoftClientCSR(ctx, reqHWDeviceID, clientCSR)
	if err != nil {
		return "", nil, err
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
		return "", nil, err
	}

	// Getting the MS-MDM management URL to provision the device
	urlManagementEndpoint, err := microsoft_mdm.ResolveWindowsMDMManagement(appCfg.ServerSettings.ServerURL)
	if err != nil {
		return "", nil, err
	}

	// generate username and password for device management service
	username := reqDeviceID
	password := uuid.NewString()
	credentialsHash := md5.Sum(fmt.Appendf(nil, "%s:%s", username, password)) //nolint:gosec // Windows MDM Auth uses MD5

	// Preparing the Application Provisioning information
	appConfigProvisioningData := NewApplicationProvisioningData(urlManagementEndpoint, username, password)

	// Preparing the DM Client Provisioning information
	appDMClientProvisioningData := NewDMClientProvisioningData()

	// And finally returning the Base64 encoded representation of the Provisioning Doc XML
	provDoc := NewProvisioningDoc(certStoreProvisioningData, appConfigProvisioningData, appDMClientProvisioningData)
	encodedProvDoc, err := provDoc.GetEncodedB64Representation()
	if err != nil {
		return "", nil, err
	}

	return encodedProvDoc, credentialsHash[:], nil
}

// storeWindowsMDMEnrolledDevice stores the device information to the list of MDM enrolled devices
func (svc *Service) storeWindowsMDMEnrolledDevice(ctx context.Context, userID string, hostUUID string, enrollType fleet.WindowsMDMEnrollType, secTokenMsg *fleet.RequestSecurityToken, credentialsHash []byte) error {
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

	reqNotInOOBE := false
	notInOOBEStr, err := GetContextItem(secTokenMsg, syncml.ReqSecTokenContextItemNotInOobe)
	if err != nil {
		return fmt.Errorf("%s %v", error_tag, err)
	}
	if notInOOBEStr == "true" {
		reqNotInOOBE = true
	}

	// Determine if the device is awaiting configuration. Set to Pending when the enrollment is
	// automatic (Autopilot via JWT/WSTEP, not orbit node key) AND the device is in OOBE
	// (NotInOobe is false, since the field name is inverted). Later phases transition to Active
	// (ESP commands enqueued) and back to None (setup complete/failed).
	awaitingConfiguration := fleet.WindowsMDMAwaitingConfigurationNone
	var awaitingConfigurationAt *time.Time
	isInOOBE := !reqNotInOOBE
	if enrollType == fleet.WindowsMDMEnrollTypeAutomatic && isInOOBE {
		awaitingConfiguration = fleet.WindowsMDMAwaitingConfigurationPending
		now := time.Now().UTC()
		awaitingConfigurationAt = &now
		svc.logger.InfoContext(ctx, "ESP: device enrolled in OOBE, activating setup experience", "device_id", reqDeviceID)
	}

	// Getting the Windows Enrolled Device Information
	enrolledDevice := &fleet.MDMWindowsEnrolledDevice{
		MDMDeviceID:             reqDeviceID,
		MDMHardwareID:           reqHWDevID,
		MDMDeviceState:          microsoft_mdm.MDMDeviceStateEnrolled,
		MDMDeviceType:           reqDeviceType,
		MDMDeviceName:           reqDeviceName,
		MDMEnrollType:           reqEnrollType,
		MDMEnrollUserID:         userID, // This could be Host UUID or UPN email
		MDMEnrollProtoVersion:   reqEnrollVersion,
		MDMEnrollClientVersion:  reqAppVersion,
		MDMNotInOOBE:            reqNotInOOBE,
		AwaitingConfiguration:   awaitingConfiguration,
		AwaitingConfigurationAt: awaitingConfigurationAt,
		HostUUID:                hostUUID,
		CredentialsHash:         &credentialsHash,
		CredentialsAcknowledged: true,
	}

	if err := svc.ds.MDMWindowsInsertEnrolledDevice(ctx, enrolledDevice); err != nil {
		return err
	}

	// For Azure (automatic) enrollments, hostUUID is empty here because the WSTEP RequestSecurityToken does not carry
	// any identifier that maps to hosts.uuid. The enrollment row is inserted unlinked; processIncomingMDMCmds asks the
	// device for its SMBIOS serial number on the first management session and links the row when the device replies.
	// osquery's directIngestMDMDeviceIDWindows remains as a backstop.
	displayName := reqDeviceName
	var serial string
	if hostUUID != "" {
		mdmLifecycle := mdmlifecycle.New(svc.ds, svc.logger, svc.NewActivity)
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

			// Flip host_mdm.enrolled = 1 immediately so the Windows profile reconciler selects this host. This covers
			// fresh enrollment, ESP/OOBE, and the post-disable re-enable cycle. The values written here are the same
			// shape directIngestMDMWindows writes: discovery URL as server_url, fleet.WellKnownMDMFleet as the name.
			// isServer is best-effort false (osquery corrects it later from installation_type); installed_from_dep
			// mirrors osquery's "automatic" semantics (Azure AD + OOBE = automatic). On failure, we log and continue:
			// osquery will reconcile the row on the next check-in.
			appCfg, acErr := svc.ds.AppConfig(ctx)
			if acErr != nil {
				svc.logger.WarnContext(ctx, "loading app config for host_mdm sync after Windows MDM enrollment", "err", acErr)
				ctxerr.Handle(ctx, acErr)
			} else {
				discoveryURL, dErr := microsoft_mdm.ResolveWindowsMDMDiscovery(appCfg.ServerSettings.ServerURL)
				if dErr != nil {
					svc.logger.WarnContext(ctx, "resolving Windows MDM discovery URL after enrollment", "err", dErr)
					ctxerr.Handle(ctx, dErr)
				} else {
					installedFromDep := enrollType == fleet.WindowsMDMEnrollTypeAutomatic && isInOOBE
					if err := svc.ds.SetOrUpdateMDMData(ctx, hosts[0].ID,
						false, // is_server: osquery corrects later from installation_type
						true,  // enrolled
						discoveryURL,
						installedFromDep,
						fleet.WellKnownMDMFleet,
						"",    // fleet_enrollment_ref: empty for Windows
						false, // is_personal_enrollment: always false for Windows
					); err != nil {
						svc.logger.WarnContext(ctx, "updating host_mdm.enrolled after Windows MDM enrollment", "err", err)
						ctxerr.Handle(ctx, err)
					}
				}
			}

			// Queue this host's profiles right away instead of waiting for the cron's next pass (mirrors Apple's
			// post-enrollment ReconcileProfilesForEnrollingHost). Best-effort: on failure the enrollment still
			// succeeds and the cron's walk-all pass delivers the profiles.
			if err := ReconcileWindowsProfilesForEnrollingHost(ctx, svc.ds, svc.logger, hostUUID); err != nil {
				svc.logger.WarnContext(ctx, "reconciling profiles after Windows MDM enrollment", "err", err)
				ctxerr.Handle(ctx, err)
			}
		}

	}

	err = svc.NewActivity(
		ctx, nil, &fleet.ActivityTypeMDMEnrolled{
			HostDisplayName: displayName,
			MDMPlatform:     fleet.MDMPlatformMicrosoft,
			HostSerial:      &serial,
			Platform:        "windows",
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

	// check if it's a network/database error, if so log as error, otherwise as
	// info (client validation error, e.g. empty binary security token). We
	// unfortunately use raw errors such as fmt.Errorf and errors.New a lot,
	// which does not carry much semantic information to discriminate error
	// types, so we do a best-effort check here.
	var ne net.Error
	var me *mysql_driver.MySQLError
	if errors.As(errorMsg, &ne) || errors.As(errorMsg, &me) {
		logging.WithErr(ctx, ctxerr.Wrap(ctx, errorMsg, "soap fault"))
	} else {
		logging.WithLevel(ctx, slog.LevelInfo)
		logging.WithExtras(ctx, "soap_fault", errorMsg.Error())
	}
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
		if len(cmdData) > 0 && len(cmdTarget) > 0 {
			rawCmd := newSyncMLCmdText(cmdVerb, cmdTarget, cmdData)
			return rawCmd, nil
		}

		return nil, errInvalidParameters

	case mdm_types.SFXml:
		if len(cmdData) > 0 && len(cmdTarget) > 0 {
			rawCmd := newSyncMLCmdXml(cmdVerb, cmdTarget, cmdData)
			return rawCmd, nil
		}

		return nil, errInvalidParameters

	case mdm_types.SFInteger:
		if len(cmdData) > 0 && len(cmdTarget) > 0 {
			rawCmd := newSyncMLCmdInt(cmdVerb, cmdTarget, cmdData)
			return rawCmd, nil
		}

		return nil, errInvalidParameters

	case mdm_types.SFBase64:
		if len(cmdData) > 0 && len(cmdTarget) > 0 {
			rawCmd := newSyncMLCmdBase64(cmdVerb, cmdTarget, cmdData)
			return rawCmd, nil
		}

		return nil, errInvalidParameters

	case mdm_types.SFBoolean:
		if len(cmdData) > 0 && len(cmdTarget) > 0 {
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

// newSyncMLCmdText creates a new SyncML command with text data. The value is XML-escaped (matching
// newSyncMLCmdXml/newSyncMLCmdBase64) because SyncML <Data> is serialized as innerxml (written raw); without
// escaping, a value containing &, <, or > -- e.g. a software title like "AT&T" surfaced in CustomErrorText --
// would produce malformed SyncML that the device rejects.
func newSyncMLCmdText(cmdVerb string, cmdTarget string, cmdDataValue string) *mdm_types.SyncMLCmd {
	cmdType := "text/plain"
	cmdFormat := "chr"
	escapedData := html.EscapeString(cmdDataValue)
	item := newSyncMLItem(nil, &cmdTarget, &cmdType, &cmdFormat, &escapedData)
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

// newSyncMLCmdNode creates a new SyncML command that targets a node with no data.
// Used for Add commands on dynamic OMA-DM nodes that must be created before their
// children can be set.
func newSyncMLCmdNode(cmdVerb string, cmdTarget string) *mdm_types.SyncMLCmd {
	cmdFormat := "node"
	item := newSyncMLItem(nil, &cmdTarget, nil, &cmdFormat, nil)
	return newSyncMLCmdWithItem(&cmdVerb, nil, item)
}

// newSyncMLCmdGet creates a SyncML Get command targeting the given OMA-DM LocURI. Get commands have no data, format, or
// type on the request; the device fills those in on the corresponding Results response.
func newSyncMLCmdGet(cmdTarget string) *mdm_types.SyncMLCmd {
	verb := fleet.CmdGet
	item := newSyncMLItem(nil, &cmdTarget, nil, nil, nil)
	return newSyncMLCmdWithItem(&verb, nil, item)
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

// reconcileWindowsProfilesBatchSize is the scan window: how many enrolled Windows hosts the reconciler reads per snapshot.
// Snapshot reads are cheap (indexed, no set-difference), so within a single tick the drain loop pages through many windows until
// a budget is hit.
//
// var rather than const so property-based tests can shrink the batch size.
var reconcileWindowsProfilesBatchSize = 2000

// reconcileWindowsProfilesDeliveryCap bounds how many distinct hosts the cron schedules for install/remove per tick. It governs
// the bulk case: once this many hosts have been delivered work, the tick stops even if scan budget remains, advancing the cursor
// only to the last delivered host so the remainder resumes next tick. This preserves the writer-pressure smoothing: a bulk change
// is spread across ~ceil(hosts/cap) ticks. Set <= 0 to disable the cap (drain the whole fleet, bounded only by the scan budget).
//
// var rather than const so tests can override it.
var reconcileWindowsProfilesDeliveryCap = 2000

// reconcileWindowsProfilesScanBudget is the wall-clock budget for a single tick's drain loop. It governs the sparse/idle case: a
// no-work pass over the whole fleet completes within one tick, collapsing single-change latency from ceil(hosts/batch) x interval
// to roughly the actual work time. ~24s of the 30s cron interval leaves headroom for the final batch's writes.
//
// var rather than const so tests can override it.
var reconcileWindowsProfilesScanBudget = 24 * time.Second

// ReconcileWindowsProfilesForEnrollingHost is the per-host reconciler, the Windows mirror of Apple's
// ReconcileProfilesForEnrollingHost. It is invoked right after a host turns on Windows MDM (so profiles are queued without
// waiting for the cron's next pass) and from the ESP release check (so a host awaiting setup-experience release has its
// desired profiles queued before the release decision evaluates in-flight rows). It reuses the shared snapshot loaders,
// ComputeWindowsReconcileDeltas, and the execute step, so it can't drift from the batched cron reconciler on "what should be
// installed."
//
// Returns nil (no-op) when Windows MDM is disabled or the host isn't an eligible MDM-enrolled Windows host. Errors
// are returned to the caller; the cron's walk-all pass remains the eventual-consistency backstop.
func ReconcileWindowsProfilesForEnrollingHost(ctx context.Context, ds fleet.Datastore, logger *slog.Logger, hostUUID string) error {
	appConfig, err := ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "reading app config")
	}
	if !appConfig.MDM.WindowsEnabledAndConfigured {
		return nil
	}

	// Two reads below need the primary (read-your-writes): eligibility, because host_mdm.enrolled was written moments
	// ago in the enrollment request and a lagging replica would turn the reconcile into a silent no-op; and current
	// rows, because deltas computed against replica-stale rows would re-enqueue duplicate commands for work queued by
	// a previous checkin.
	primaryCtx := ctxdb.RequirePrimary(ctx, true)

	host, err := ds.GetWindowsMDMHostForReconcile(primaryCtx, hostUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get windows mdm host for reconcile")
	}
	if host == nil {
		return nil
	}

	// Load only profiles for this host's team so the per-host call doesn't scan every profile in the system.
	teamProfiles, err := ds.ListWindowsProfilesForReconcileByTeam(ctx, host.EffectiveTeamID())
	if err != nil {
		return ctxerr.Wrap(ctx, err, "listing windows profiles for reconcile by team")
	}

	profilesByTeam := make(map[uint][]*fleet.WindowsProfileForReconcile, 1)
	profilesWithBrokenLabel := make(map[string]struct{})
	labelIDSet := make(map[uint]struct{})
	for _, p := range teamProfiles {
		profilesByTeam[p.TeamID] = append(profilesByTeam[p.TeamID], p)
		if p.HasBrokenLabel() {
			profilesWithBrokenLabel[p.ProfileUUID] = struct{}{}
		}
		for _, lr := range p.IncludeLabels {
			if lr.LabelID != nil {
				labelIDSet[*lr.LabelID] = struct{}{}
			}
		}
		for _, lr := range p.ExcludeLabels {
			if lr.LabelID != nil {
				labelIDSet[*lr.LabelID] = struct{}{}
			}
		}
	}
	labelIDs := make([]uint, 0, len(labelIDSet))
	for id := range labelIDSet {
		labelIDs = append(labelIDs, id)
	}

	hostLabels, err := ds.BulkGetHostLabelMemberships(ctx, []uint{host.HostID}, labelIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "bulk get host label memberships")
	}

	currentByHost, err := ds.BulkGetHostMDMWindowsProfilesByUUIDs(primaryCtx, []string{host.UUID})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "bulk get host mdm windows profiles")
	}

	hosts := []*fleet.WindowsHostReconcileInfo{host}
	toInstall, toRemove := microsoft_mdm.ComputeWindowsReconcileDeltas(hosts, hostLabels, currentByHost, profilesByTeam, profilesWithBrokenLabel)
	if len(toInstall) == 0 && len(toRemove) == 0 {
		return nil
	}

	desiredByHost := microsoft_mdm.DesiredWindowsProfileUUIDsByHost(hosts, hostLabels, profilesByTeam)
	return executeWindowsProfileReconcileBatch(ctx, ds, logger, appConfig, toInstall, toRemove, desiredByHost)
}

// windowsProfileNeedsPerHostProcessing reports whether a Windows profile must be
// processed per-host at delivery time — i.e. it references any FLEET_VAR_ variable
// or any $FLEET_HOST_VITAL_<id> custom host vital.
func windowsProfileNeedsPerHostProcessing(syncML []byte) bool {
	return variables.ContainsBytes(syncML) || len(fleet.ContainsCustomHostVitalIDs(string(syncML))) > 0
}

// ReconcileWindowsProfiles applies configuration profiles to Windows MDM hosts.
//
// It walks every enrolled Windows host via a host_uuid cursor (persisted in Redis through the mysqlredis wrapper), loading a
// bounded snapshot per window, computing install/remove deltas in memory (no set-difference SQL), and executing them. Within one
// tick it drains successive windows until either the delivery cap or the scan budget is hit, or the host space is exhausted
// (which resets the cursor for the next pass).
//
// Named return so the deferred SetCursor block sees the actual function-exit error: the cursor is persisted only on a clean (err
// == nil) tick, so any failure leaves the cursor untouched and the next tick re-scans from the same point. Re-scanning is cheap
// and idempotent since delivered work is now pending, so it no longer computes as work.
func ReconcileWindowsProfiles(ctx context.Context, ds fleet.Datastore, logger *slog.Logger) (err error) {
	appConfig, err := ds.AppConfig(ctx)
	if err != nil {
		return fmt.Errorf("reading app config: %w", err)
	}
	if !appConfig.MDM.WindowsEnabledAndConfigured {
		return nil
	}

	// Read the cursor; on error, treat as start-of-pass and continue. A stale or missing cursor is harmless because the in-memory
	// diff installs only what actually differs from the current state.
	entryCursor, cerr := ds.GetMDMWindowsReconcileCursor(ctx)
	if cerr != nil {
		logger.WarnContext(ctx, "failed to read windows MDM reconcile cursor; starting from beginning", "err", cerr)
		entryCursor = ""
	}

	cursor := entryCursor
	// commitCursor is the cursor value to persist at tick end. It advances only past windows that were fully delivered; the deferred
	// write fires only when err == nil, so an error leaves the cursor untouched.
	commitCursor := entryCursor

	defer func() {
		if err == nil && commitCursor != entryCursor {
			if serr := ds.SetMDMWindowsReconcileCursor(ctx, commitCursor); serr != nil {
				logger.WarnContext(ctx, "failed to advance windows MDM reconcile cursor", "err", serr)
			}
		}
	}()

	deadline := time.Now().Add(reconcileWindowsProfilesScanBudget)
	deliveredHosts := 0

	for {
		hosts, allProfiles, hostLabels, currentByHost, serr := ds.GetWindowsProfileReconcileSnapshot(ctx, cursor, reconcileWindowsProfilesBatchSize)
		if serr != nil {
			err = ctxerr.Wrap(ctx, serr, "loading windows profile reconcile snapshot")
			return err
		}

		if len(hosts) == 0 {
			// Reached the end of the host space (or empty fleet): reset the cursor so the next pass restarts from the beginning.
			commitCursor = ""
			return nil
		}

		profilesByTeam := make(map[uint][]*fleet.WindowsProfileForReconcile, 4)
		profilesWithBrokenLabel := make(map[string]struct{})
		for _, p := range allProfiles {
			profilesByTeam[p.TeamID] = append(profilesByTeam[p.TeamID], p)
			if p.HasBrokenLabel() {
				profilesWithBrokenLabel[p.ProfileUUID] = struct{}{}
			}
		}

		toInstall, toRemove := microsoft_mdm.ComputeWindowsReconcileDeltas(hosts, hostLabels, currentByHost, profilesByTeam, profilesWithBrokenLabel)
		// Per-host desired (applicable) live profiles, used by the execute step to protect LocURIs a remove target shares with a
		// profile still desired on the same host (label-aware, so a label-scoped profile only protects the hosts it applies to).
		desiredByHost := microsoft_mdm.DesiredWindowsProfileUUIDsByHost(hosts, hostLabels, profilesByTeam)

		// Apply the per-tick delivery cap at host granularity. Hosts come back ascending by uuid, so capping keeps a contiguous prefix of
		// the work-hosts and the cursor can resume at the last delivered host.
		workHosts := windowsHostsWithWork(hosts, toInstall, toRemove)
		advanceTo := hosts[len(hosts)-1].UUID
		fullBatch := len(hosts) >= reconcileWindowsProfilesBatchSize

		partial := false
		if reconcileWindowsProfilesDeliveryCap > 0 {
			// Invariant: deliveredHosts < cap here. We return below as soon as it reaches the cap. So remaining >= 1.
			remaining := reconcileWindowsProfilesDeliveryCap - deliveredHosts
			if len(workHosts) > remaining {
				allowed := make(map[string]struct{}, remaining)
				for _, h := range workHosts[:remaining] {
					allowed[h] = struct{}{}
				}
				toInstall = filterWindowsPayloadsByHost(toInstall, allowed)
				toRemove = filterWindowsPayloadsByHost(toRemove, allowed)
				advanceTo = workHosts[remaining-1] // resume after the last delivered host
				workHosts = workHosts[:remaining]
				partial = true
			}
		}

		if len(toInstall) > 0 || len(toRemove) > 0 {
			if eerr := executeWindowsProfileReconcileBatch(ctx, ds, logger, appConfig, toInstall, toRemove, desiredByHost); eerr != nil {
				err = eerr
				return err
			}
		}
		deliveredHosts += len(workHosts)

		// Advance only after a successful execute.
		commitCursor = advanceTo
		cursor = advanceTo

		switch {
		case partial:
			// Delivery cap hit mid-window; the un-delivered remainder resumes next tick from cursor = advanceTo.
			return nil
		case !fullBatch:
			// Short window => end of the host space; reset for the next pass.
			commitCursor = ""
			return nil
		case reconcileWindowsProfilesDeliveryCap > 0 && deliveredHosts >= reconcileWindowsProfilesDeliveryCap:
			// Delivery cap reached exactly at a window boundary.
			return nil
		case time.Now().After(deadline):
			// Scan budget exhausted; resume next tick from cursor = advanceTo.
			return nil
		}
		// Otherwise keep draining the next window within this tick.
	}
}

// windowsHostsWithWork returns the host UUIDs that have at least one install or remove in this window, in the order hosts are
// given (ascending by uuid). The drain loop uses this both to count delivered hosts against the cap and to pick the contiguous
// prefix to deliver when the cap is reached mid-window.
func windowsHostsWithWork(hosts []*fleet.WindowsHostReconcileInfo, toInstall, toRemove []*fleet.MDMWindowsProfilePayload) []string {
	work := make(map[string]struct{})
	for _, p := range toInstall {
		work[p.HostUUID] = struct{}{}
	}
	for _, p := range toRemove {
		work[p.HostUUID] = struct{}{}
	}
	ordered := make([]string, 0, len(work))
	for _, h := range hosts {
		if _, ok := work[h.UUID]; ok {
			ordered = append(ordered, h.UUID)
		}
	}
	return ordered
}

// filterWindowsPayloadsByHost returns only the payloads whose HostUUID is in the allowed set, preserving order. Used to trim a
// window's deltas to the hosts that fit under the per-tick delivery cap.
func filterWindowsPayloadsByHost(payloads []*fleet.MDMWindowsProfilePayload, allowed map[string]struct{}) []*fleet.MDMWindowsProfilePayload {
	out := make([]*fleet.MDMWindowsProfilePayload, 0, len(payloads))
	for _, p := range payloads {
		if _, ok := allowed[p.HostUUID]; ok {
			out = append(out, p)
		}
	}
	return out
}

// executeWindowsProfileReconcileBatch runs the post-compute reconcile pipeline against the in-memory toInstall / toRemove sets
// produced by ComputeWindowsReconcileDeltas: content fetch, deleted-profile race guard, bulk command pre-build for non-variable
// profiles, per-host variable expansion, LocURI-protected <Delete> generation, host-profile upserts, and managed-certificate
// bookkeeping. This is the legacy reconciler body verbatim, now invoked once per (capped) window by the drain loop above.
func executeWindowsProfileReconcileBatch(
	ctx context.Context,
	ds fleet.Datastore,
	logger *slog.Logger,
	appConfig *fleet.AppConfig,
	toInstall, toRemove []*fleet.MDMWindowsProfilePayload,
	desiredByHost map[string][]string,
) error {
	// toGetContents contains the IDs of all the profiles from which we
	// need to retrieve contents. Since the previous query returns one row
	// per host, it would be too expensive to retrieve the profile contents
	// there, so we make another request. Using a map to deduplicate.
	toGetContents := make(map[string]bool)

	// hostProfilesToUpdate tracks each host_mdm_windows_profile we need to upsert
	// with the new status, operation_type, etc.
	hostProfilesToUpdate := make([]*fleet.MDMWindowsBulkUpsertHostProfilePayload, 0, len(toInstall))

	// hostProfilesMap provides O(1) lookup for host profiles by host UUID and profile UUID
	// Key format: "hostUUID|profileUUID"
	hostProfilesMap := make(map[string]*fleet.MDMWindowsBulkUpsertHostProfilePayload, len(toInstall))

	// batchProfileCmdsMap maps command UUID -> host profiles, used to fetch all host profiles associated
	// with a given command UUID so all host_mdm_windows_profiles entries can be updated as the command
	// is enqueued for hosts
	batchProfileCmdsMap := make(map[string][]*fleet.MDMWindowsBulkUpsertHostProfilePayload)

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
	removeTargets := make(map[string]*cmdTarget)

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

		hp := &fleet.MDMWindowsBulkUpsertHostProfilePayload{
			ProfileUUID:   p.ProfileUUID,
			HostUUID:      p.HostUUID,
			ProfileName:   p.ProfileName,
			CommandUUID:   target.cmdUUID,
			OperationType: fleet.MDMOperationTypeInstall,
			Status:        &fleet.MDMDeliveryPending,
			Checksum:      p.Checksum,
		}
		hostProfilesToUpdate = append(hostProfilesToUpdate, hp)
		// Add to map for fast lookup
		hostProfilesMap[p.HostUUID+"|"+p.ProfileUUID] = hp
		batchProfileCmdsMap[target.cmdUUID] = append(batchProfileCmdsMap[target.cmdUUID], hp)
		logger.DebugContext(ctx, "installing profile", "profile_uuid", p.ProfileUUID, "host_id", p.HostUUID, "name", p.ProfileName)
	}

	// Build remove targets: profiles that need to be removed from hosts (e.g. host moved teams, label membership changed).
	// We only collect targets here; host_mdm_windows_profiles entries are
	// created after the delete command is successfully built and enqueued.
	removePayloadData := make(map[string][]*fleet.MDMWindowsProfilePayload) // profUUID -> payloads
	for _, p := range toRemove {
		toGetContents[p.ProfileUUID] = true
		target := removeTargets[p.ProfileUUID]
		if target == nil {
			target = &cmdTarget{
				cmdUUID: uuid.New().String(),
				profID:  p.ProfileUUID,
			}
			removeTargets[p.ProfileUUID] = target
		}
		target.hostUUIDs = append(target.hostUUIDs, p.HostUUID)
		removePayloadData[p.ProfileUUID] = append(removePayloadData[p.ProfileUUID], p)
	}

	// Also fetch the contents of profiles still desired on the hosts we are removing from: their LocURIs protect shared settings, so a
	// <Delete> for a removed profile does not revert a setting another profile still applicable to that host enforces. Walk each
	// removed-from host's desired list once, not once per removed profile on that host.
	seenRemoveHost := make(map[string]struct{}, len(toRemove))
	for _, p := range toRemove {
		if _, ok := seenRemoveHost[p.HostUUID]; ok {
			continue
		}
		seenRemoveHost[p.HostUUID] = struct{}{}
		for _, q := range desiredByHost[p.HostUUID] {
			toGetContents[q] = true
		}
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

	groupedCAs, err := ds.GetGroupedCertificateAuthorities(ctx, true)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting grouped certificate authorities")
	}

	scepConfigSvc := scep.NewSCEPConfigService(logger, nil)
	managedCertificatePayloads := &[]*fleet.MDMManagedCertificate{}
	deps := microsoft_mdm.ProfilePreprocessDependencies{
		Context:                    ctx,
		Logger:                     logger,
		DataStore:                  ds,
		HostIDForUUIDCache:         make(map[string]uint),
		AppConfig:                  appConfig,
		CustomSCEPCAs:              groupedCAs.ToCustomSCEPProxyCAMap(),
		ManagedCertificatePayloads: managedCertificatePayloads,
		NDESConfig:                 groupedCAs.NDESSCEP,
		GetNDESSCEPChallenge:       scepConfigSvc.GetNDESSCEPChallenge,
		NDESChallengeErrorToDetail: scep.NDESChallengeErrorToDetail,
	}

	// Guard against a race where an admin deletes a profile between the
	// initial snapshot/GetMDMWindowsProfilesContents
	// calls above and the per-profile upsert below. If we missed the
	// deletion, we'd create a host_mdm_windows_profiles row (and enqueue a
	// command) for a profile that no longer exists in
	// mdm_windows_configuration_profiles. The remove path couldn't clean
	// that row up later — <Delete> command generation needs the original
	// SyncML, which is gone — and the host would be stuck with an
	// un-removable install. Re-query existence right before the upsert
	// loops to shrink the race window to just the loop body.
	installProfileUUIDs := make([]string, 0, len(installTargets))
	for profUUID := range installTargets {
		installProfileUUIDs = append(installProfileUUIDs, profUUID)
	}
	stillExistingInstallProfiles, err := ds.GetExistingMDMWindowsProfileUUIDs(ctx, installProfileUUIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "checking Windows profile existence before install upsert")
	}

	// Pre-build and bulk-insert commands for non-variable install profiles.
	// Variable profiles and remove commands are not pre-inserted because
	// they require per-host or per-profile processing.
	var bulkCommands []*fleet.MDMWindowsCommand
	for profUUID, target := range installTargets {
		if _, stillExists := stillExistingInstallProfiles[profUUID]; !stillExists {
			continue
		}
		p, ok := profileContents[profUUID]
		if !ok || windowsProfileNeedsPerHostProcessing(p.SyncML) {
			continue // variable profiles get per-host commands, can't pre-build
		}
		command, err := buildCommandFromProfileBytes(p.SyncML, target.cmdUUID)
		if err != nil {
			continue // will be handled in the per-profile loop below
		}
		bulkCommands = append(bulkCommands, command)
	}
	if len(bulkCommands) > 0 {
		if err := ds.MDMWindowsBulkInsertCommands(ctx, bulkCommands); err != nil {
			return ctxerr.Wrap(ctx, err, "bulk inserting commands")
		}
	}

	for profUUID, target := range installTargets {
		if _, stillExists := stillExistingInstallProfiles[profUUID]; !stillExists {
			logger.InfoContext(ctx, "skipping Windows profile install; profile was deleted after list",
				"profile_uuid", profUUID, "host_count", len(target.hostUUIDs))
			continue
		}
		p, ok := profileContents[profUUID]
		if !ok {
			// Insert-lag race: GetMDMWindowsProfilesContents earlier in
			// this function reads from the replica, while
			// GetExistingMDMWindowsProfileUUIDs above reads from the
			// primary (it has to, to defeat the delete-race). For a
			// profile that was inserted on primary just before this tick
			// fired, the existence check sees it but profileContents
			// (replica) misses it until replication catches up.
			//
			// Skip the install for now and let a later tick pick it up
			// after replication catches up. With the host cursor
			// advancing past this batch, the affected hosts won't be
			// revisited until the cursor cycles back to the start of
			// the host space; for large host populations that can be
			// many ticks rather than the next one. The hosts stay in
			// the listing's pending universe because no
			// host_mdm_windows_profiles row was written for them, so
			// no state is lost.
			logger.InfoContext(ctx, "skipping Windows profile install; profile content not visible on replica yet (insert-lag), will retry on a later tick",
				"profile_uuid", profUUID, "host_count", len(target.hostUUIDs))
			continue
		}

		if !windowsProfileNeedsPerHostProcessing(p.SyncML) {
			// No Fleet variables, send the same command to all hosts
			payloads, ok := batchProfileCmdsMap[target.cmdUUID]
			if !ok {
				logger.ErrorContext(ctx, "no host profiles found for command UUID", "command_uuid", target.cmdUUID)
				continue
			}
			command, err := buildCommandFromProfileBytes(p.SyncML, target.cmdUUID)
			if err != nil {
				logger.InfoContext(ctx, "error building command from profile", "err", err, "profile_uuid", profUUID)
				for _, payload := range payloads {
					payload.Status = &fleet.MDMDeliveryFailed
					payload.Detail = fmt.Sprintf("Failed to build command from profile: %s", err.Error())
				}
				continue
			}

			// Since we are not using DB transactions here, there is a small chance that the profile contents don't match
			// the checksum we retrieved earlier. Update the checksums if needed.
			for _, payload := range payloads {
				payload.Checksum = p.Checksum
			}
			if err := ds.MDMWindowsEnqueueCommandAndUpsertHostProfiles(ctx, target.hostUUIDs, command, payloads); err != nil {
				return ctxerr.Wrap(ctx, err, "inserting commands for hosts")
			}
		} else {
			// Profile contains Fleet variables, process each host individually
			for _, hostUUID := range target.hostUUIDs {
				mapKey := hostUUID + "|" + profUUID
				hp := hostProfilesMap[mapKey]
				if hp == nil {
					// This should never happen, but handle gracefully
					logger.ErrorContext(ctx, "host profile not found in map", "profile_uuid", profUUID, "host_uuid", hostUUID)
					continue
				}

				// Preprocess the profile content for this specific host
				processedContent, err := microsoft_mdm.PreprocessWindowsProfileContentsForDeployment(
					deps,
					microsoft_mdm.ProfilePreprocessParams{HostUUID: hostUUID, ProfileUUID: profUUID},
					string(p.SyncML),
				)
				var profileProcessingError *microsoft_mdm.MicrosoftProfileProcessingError
				if err != nil && !errors.As(err, &profileProcessingError) {
					return ctxerr.Wrapf(ctx, err, "preprocessing profile contents for host %s and profile %s", hostUUID, profUUID)
				} else if err != nil && errors.As(err, &profileProcessingError) {
					hp.Status = &fleet.MDMDeliveryFailed
					hp.Detail = profileProcessingError.Error()
					continue
				}

				// Create a unique command UUID for this host since the content is unique
				hostCmdUUID := uuid.New().String()

				// Build the command with the processed content
				command, err := buildCommandFromProfileBytes([]byte(processedContent), hostCmdUUID)
				if err != nil {
					logger.InfoContext(ctx, "error building command from profile", "err", err, "profile_uuid", profUUID, "host_uuid", hostUUID)
					// Mark this host's profile as failed
					hp.Status = &fleet.MDMDeliveryFailed
					hp.Detail = fmt.Sprintf("Failed to build command from profile: %s", err.Error())
					continue
				}

				// Update the command UUID for this specific host
				hp.CommandUUID = hostCmdUUID
				// Since we are not using DB transactions here, there is a small chance that the profile contents don't match
				// the checksum we retrieved earlier. Update the checksum if needed.
				hp.Checksum = p.Checksum
				// Insert the command for this specific host
				if err := ds.MDMWindowsInsertCommandAndUpsertHostProfilesForHosts(ctx, []string{hostUUID}, command, []*fleet.MDMWindowsBulkUpsertHostProfilePayload{hp}); err != nil {
					logger.ErrorContext(ctx, "inserting command for host", "err", err, "host_uuid", hostUUID)
					// Mark this host's profile as failed
					hp.Status = &fleet.MDMDeliveryFailed
					hp.Detail = fmt.Sprintf("Failed to insert command for host: %s", err.Error())
					continue
				}
			}
		}
	}

	// Generate and enqueue <Delete> commands for profiles being removed from hosts. Protection is per host: a removed profile's
	// LocURI is deleted on a host only when no still-desired profile on that host enforces it. Hosts sharing the same protected
	// subset of the removed profile's LocURIs are grouped into one command; a label-scoped protector that applies to only some hosts
	// naturally splits them into separate groups. desiredByHost contains only kept (applicable) profiles, never removed ones, so it
	// needs no further exclusion.
	resolvedLocURIs := make(map[string][]string) // profileUUID -> SCEP-resolved LocURIs (cached across hosts/targets)
	locURIsFor := func(profUUID string) []string {
		if v, ok := resolvedLocURIs[profUUID]; ok {
			return v
		}
		var uris []string
		if c, ok := profileContents[profUUID]; ok {
			// Resolve the SCEP cert-ID variable to the profile's own UUID so LocURIs compare on resolved paths (a SCEP path is
			// per-profile and therefore never shared), consistent with how the delete command itself is built.
			resolved := fleet.FleetVarSCEPWindowsCertificateIDRegexp.ReplaceAll(c.SyncML, []byte(profUUID))
			uris = fleet.ExtractLocURIsFromProfileBytes(resolved)
		}
		resolvedLocURIs[profUUID] = uris
		return uris
	}

	for profUUID, target := range removeTargets {
		if _, ok := profileContents[profUUID]; !ok {
			// No retained content for this removed profile, so we can't build its <Delete> this tick. This is normally a transient
			// replica-lag miss (the pending-delete row written on the writer hasn't replicated to the reader yet) that resolves on a
			// later tick; warn and skip for now. The host's remove rows remain, so it is retried, and no state is lost.
			logger.WarnContext(ctx, "windows profile reconcile: missing content for removed profile, skipping this tick",
				"profile_uuid", profUUID, "host_count", len(target.hostUUIDs))
			continue
		}
		removedURIs := locURIsFor(profUUID)

		type removeGroup struct {
			activeLocURIs map[string]struct{}
			hostUUIDs     []string
		}
		groups := make(map[string]*removeGroup)
		for _, hostUUID := range target.hostUUIDs {
			active := make(map[string]struct{})
			for _, desiredUUID := range desiredByHost[hostUUID] {
				if desiredUUID == profUUID {
					continue
				}
				for _, uri := range locURIsFor(desiredUUID) {
					active[uri] = struct{}{}
				}
			}
			// Key on the protected subset of the removed profile's own LocURIs so hosts with identical effective protection share a
			// single command; the common (no label) case collapses to one group.
			var keyURIs []string
			for _, uri := range removedURIs {
				if _, ok := active[uri]; ok {
					keyURIs = append(keyURIs, uri)
				}
			}
			slices.Sort(keyURIs)
			key := strings.Join(keyURIs, "\n")
			g := groups[key]
			if g == nil {
				g = &removeGroup{activeLocURIs: active}
				groups[key] = g
			}
			g.hostUUIDs = append(g.hostUUIDs, hostUUID)
		}

		// Index this profile's remove payloads by host once so each group builds its command payloads with O(1) lookups instead of
		// rescanning the full per-profile payload list per group (which is O(groups x hosts) when label scoping forms many groups).
		payloadByHost := make(map[string]*fleet.MDMWindowsProfilePayload, len(removePayloadData[profUUID]))
		for _, rp := range removePayloadData[profUUID] {
			payloadByHost[rp.HostUUID] = rp
		}

		for _, g := range groups {
			cmdUUID := uuid.New().String()
			command, err := fleet.BuildDeleteCommandFromProfileBytes(profileContents[profUUID].SyncML, cmdUUID, profUUID, g.activeLocURIs)
			if err != nil {
				logger.InfoContext(ctx, "error building delete command from profile", "err", err, "profile_uuid", profUUID)
				continue
			}
			if command == nil {
				// Every LocURI of the removed profile is still enforced by another profile on these hosts; nothing to send.
				continue
			}

			removePayloadsForCommand := make([]*fleet.MDMWindowsBulkUpsertHostProfilePayload, 0, len(g.hostUUIDs))
			for _, hostUUID := range g.hostUUIDs {
				rp := payloadByHost[hostUUID]
				if rp == nil {
					continue
				}
				// Remove operations don't need a checksum; use a zero value if none exists (defensive coding).
				checksum := rp.Checksum
				if len(checksum) == 0 {
					checksum = make([]byte, 16)
				}
				removePayloadsForCommand = append(removePayloadsForCommand, &fleet.MDMWindowsBulkUpsertHostProfilePayload{
					ProfileUUID:   rp.ProfileUUID,
					HostUUID:      rp.HostUUID,
					ProfileName:   rp.ProfileName,
					CommandUUID:   cmdUUID,
					OperationType: fleet.MDMOperationTypeRemove,
					Status:        &fleet.MDMDeliveryPending,
					Checksum:      checksum,
				})
				logger.DebugContext(ctx, "removing profile", "profile.uuid", rp.ProfileUUID, "host.uuid", rp.HostUUID, "profile.name", rp.ProfileName)
			}
			if err := ds.MDMWindowsInsertCommandAndUpsertHostProfilesForHosts(ctx, g.hostUUIDs, command, removePayloadsForCommand); err != nil {
				return ctxerr.Wrap(ctx, err, "inserting remove commands for hosts")
			}
		}
	}

	// Upsert the host profiles we need to track.
	// Store list of failed profiles (profile UUID + host UUID to create uniqueness) to avoid updating other stuff for that, such as managed certs.
	failedProfileHostUUIDs := make(map[string]bool)
	// Create a final pass list of the failed profiles to update at the end. We already updated the pending ones during command processing
	hostProfilesForFinalUpdate := []*fleet.MDMWindowsBulkUpsertHostProfilePayload{}

	for _, p := range hostProfilesToUpdate {
		if p.Status != nil && *p.Status == fleet.MDMDeliveryFailed {
			failedProfileHostUUIDs[p.HostUUID+"|"+p.ProfileUUID] = true
			hostProfilesForFinalUpdate = append(hostProfilesForFinalUpdate, p)
		}
	}
	if err := ds.BulkUpsertMDMWindowsHostProfiles(ctx, hostProfilesForFinalUpdate); err != nil {
		return ctxerr.Wrap(ctx, err, "updating host profiles")
	}

	// Run through managed certs and remove all those that belong to failed profiles
	filteredManagedCerts := []*fleet.MDMManagedCertificate{}
	for _, mc := range *managedCertificatePayloads {
		if _, failed := failedProfileHostUUIDs[mc.HostUUID+"|"+mc.ProfileUUID]; !failed {
			filteredManagedCerts = append(filteredManagedCerts, mc)
		}
	}

	err = ds.BulkUpsertMDMManagedCertificates(ctx, filteredManagedCerts)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "updating managed certificates for windows")
	}

	return nil
}

// TODO(roberto): I think this should live separately in the
// Windows equivalent of Apple's Commander struct, but I'd like
// to keep it simpler for now until we understand more.
func buildCommandFromProfileBytes(profileBytes []byte, commandUUID string) (*fleet.MDMWindowsCommand, error) {
	rawCommand := profileBytes
	if strings.Contains(string(rawCommand), "/Vendor/MSFT/ClientCertificateInstall/SCEP") && !strings.Contains(string(rawCommand), "<Atomic>") {
		// It's a SCEP profile, so wrap it with <Atomic>
		rawCommand = fmt.Appendf([]byte{}, "<Atomic>%s</Atomic>", rawCommand)
	}
	cmds, err := fleet.UnmarshallMultiTopLevelXMLProfile(rawCommand)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling profile bytes: %w", err)
	}

	if len(cmds) == 0 {
		return nil, errors.New("no commands found in profile")
	}

	if len(cmds) == 1 {
		// We know it's either atomic or just a single command, so just set the commandUUID to the first top level element.
		cmd := cmds[0]
		cmd.CmdID = mdm_types.CmdID{
			Value:               commandUUID,
			IncludeFleetComment: true,
		}

		if cmd.XMLName.Local == mdm_types.CmdAtomic {
			// Iterate through all nested commands and set their CmdID as well
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

			// generate a CmdID for any nested <Exec>
			for i := range cmd.ExecCommands {
				cmd.ExecCommands[i].CmdID = mdm_types.CmdID{
					Value:               uuid.NewString(),
					IncludeFleetComment: true,
				}
			}
		}

		marshalledRawCommand, err := xml.Marshal(cmd)
		if err != nil {
			return nil, fmt.Errorf("marshalling command: %w", err)
		}
		rawCommand = marshalledRawCommand
	} else {
		// We know this is non-atomic, since we have a list of commands, which only happens with multiple top-level elements.
		for i, cmd := range cmds {

			cmd.CmdID = mdm_types.CmdID{
				Value:               uuid.NewString(),
				IncludeFleetComment: true,
			}

			if i == 0 {
				// First element in a non-atomic profile
				cmd.CmdID = mdm_types.CmdID{
					Value:               commandUUID,
					IncludeFleetComment: true,
				}
			}

			cmds[i] = cmd
		}

		marshalledRawCommand, err := xml.Marshal(cmds)
		if err != nil {
			return nil, fmt.Errorf("marshalling commands: %w", err)
		}
		rawCommand = marshalledRawCommand
	}

	command := &fleet.MDMWindowsCommand{
		CommandUUID: commandUUID,
		RawCommand:  rawCommand,
		// Atomic commands don't have a Target element.
		TargetLocURI: "",
	}

	return command, nil
}

// truncateString truncates a string to maxLen characters, adding "..." if truncated
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

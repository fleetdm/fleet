package microsoft_mdm

import (
	"github.com/fleetdm/fleet/v4/server/mdm/internal/commonmdm"
)

const (
	// MDMPath is Fleet's HTTP path for the core Windows MDM service.
	MDMPath = "/api/mdm/microsoft"

	// DiscoveryPath is the HTTP endpoint path that serves the IDiscoveryService functionality.
	// This is the endpoint that process the Discover and DiscoverResponse messages
	// See the section 3.1 on the MS-MDE2 specification for more details:
	// https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-mde2/2681fd76-1997-4557-8963-cf656ab8d887
	MDE2DiscoveryPath = MDMPath + "/discovery"

	// AuthPath is the HTTP endpoint path that delivers the Security Token Servicefunctionality.
	// The MS-MDE2 protocol is agnostic to the token format and value returned by this endpoint.
	// See the section 3.2 on the MS-MDE2 specification for more details:
	// https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-mde2/27ed8c2c-0140-41ce-b2fa-c3d1a793ab4a
	MDE2AuthPath = MDMPath + "/auth"

	// MDE2PolicyPath is the HTTP endpoint path that delivers the X.509 Certificate Enrollment Policy (MS-XCEP) functionality.
	// This is the endpoint that process the GetPolicies and GetPoliciesResponse messages
	// See the section 3.3 on the MS-MDE2 specification for more details on this endpoint requirements:
	// https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-mde2/8a5efdf8-64a9-44fd-ab63-071a26c9f2dc
	// The MS-XCEP specification is available here:
	// https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-xcep/08ec4475-32c2-457d-8c27-5a176660a210
	MDE2PolicyPath = MDMPath + "/policy"

	// MDE2EnrollPath is the HTTP endpoint path that delivers WS-Trust X.509v3 Token Enrollment (MS-WSTEP) functionality.
	// This is the endpoint that process the RequestSecurityToken and RequestSecurityTokenResponseCollection messages
	// See the section 3.4 on the MS-MDE2 specification for more details on this endpoint requirements:
	// https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-mde2/5b02c625-ced2-4a01-a8e1-da0ae84f5bb7
	// The MS-WSTEP specification is available here:
	// https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-wstep/4766a85d-0d18-4fa1-a51f-e5cb98b752ea
	MDE2EnrollPath = MDMPath + "/enroll"

	// MDE2ManagementPath is the HTTP endpoint path that delivers WS-Trust X.509v3 Token Enrollment (MS-WSTEP) functionality.
	// This is the endpoint that process the RequestSecurityToken and RequestSecurityTokenResponseCollection messages
	// See the section 3.4 on the MS-MDE2 specification for more details:
	// https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-mde2/5b02c625-ced2-4a01-a8e1-da0ae84f5bb7
	MDE2ManagementPath = MDMPath + "/management"

	// MDE2TOSPath is the HTTP endpoint path that delivers Terms of Service Content
	MDE2TOSPath = MDMPath + "/tos"

	// These are the entry points for the Microsoft Device Enrollment (MS-MDE) and Microsoft Device Enrollment v2 (MS-MDE2) protocols.
	// These are required to be implemented by the MDM server to support user-driven enrollments
	MSEnrollEntryPoint = "/EnrollmentServer/Discovery.svc"
	MSManageEntryPoint = "/ManagementServer/MDM.svc"
)

// XML Namespaces and type URLs used by the Microsoft Device Enrollment v2 protocol (MS-MDE2)
const (
	DiscoverNS                 = "http://schemas.microsoft.com/windows/management/2012/01/enrollment"
	PolicyNS                   = "http://schemas.microsoft.com/windows/pki/2009/01/enrollmentpolicy"
	EnrollWSTrust              = "http://docs.oasis-open.org/ws-sx/ws-trust/200512"
	EnrollSecExt               = "http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd"
	EnrollTType                = "http://schemas.microsoft.com/5.0.0.0/ConfigurationManager/Enrollment/DeviceEnrollmentToken"
	EnrollPDoc                 = "http://schemas.microsoft.com/5.0.0.0/ConfigurationManager/Enrollment/DeviceEnrollmentProvisionDoc"
	EnrollEncode               = "http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd#base64binary"
	EnrollReq                  = "http://schemas.microsoft.com/windows/pki/2009/01/enrollment"
	EnrollNSS                  = "http://www.w3.org/2003/05/soap-envelope"
	EnrollNSA                  = "http://www.w3.org/2005/08/addressing"
	EnrollXSI                  = "http://www.w3.org/2001/XMLSchema-instance"
	EnrollXSD                  = "http://www.w3.org/2001/XMLSchema"
	EnrollXSU                  = "http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd"
	ActionNsDiag               = "http://schemas.microsoft.com/2004/09/ServiceModel/Diagnostics"
	ActionNsDiscovery          = "http://schemas.microsoft.com/windows/management/2012/01/enrollment/IDiscoveryService/DiscoverResponse"
	ActionNsPolicy             = "http://schemas.microsoft.com/windows/pki/2009/01/enrollmentpolicy/IPolicy/GetPoliciesResponse"
	ActionNsEnroll             = EnrollReq + "/RSTRC/wstep"
	EnrollReqTypePKCS10        = EnrollReq + "#PKCS10"
	EnrollReqTypePKCS7         = EnrollReq + "#PKCS7"
	BinarySecurityDeviceEnroll = "http://schemas.microsoft.com/5.0.0.0/ConfigurationManager/Enrollment/DeviceEnrollmentUserToken"
	BinarySecurityAzureEnroll  = "urn:ietf:params:oauth:token-type:jwt"
)

// Soap Error constants
// Details here: https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-mde2/0a78f419-5fd7-4ddb-bc76-1c0f7e11da23

const (
	// Message format is bad
	SoapErrorMessageFormat = "s:messageformat"

	// User not recognized
	SoapErrorAuthentication = "s:authentication"

	// User not allowed to enroll
	SoapErrorAuthorization = "s:authorization"

	// Failed to get certificate
	SoapErrorCertificateRequest = "s:certificaterequest"

	// Generic failure from management server, such as a database access error
	SoapErrorEnrollmentServer = "s:enrollmentserver"

	// The server hit an unexpected issue
	SoapErrorInternalServiceFault = "s:internalservicefault"

	// Cannot parse the security header
	SoapErrorInvalidSecurity = "a:invalidsecurity"
)

// Device Enrolled States

const (
	// Device is not yet MDM enrolled
	MDMDeviceStateNotEnrolled = "MDMDeviceEnrolledNotEnrolled"

	// Device is MDM enrolled
	MDMDeviceStateEnrolled = "MDMDeviceEnrolledEnrolled"

	// Device is MDM enrolled and managed
	/* #nosec G101 -- this constant doesn't contain any credentials */
	MDMDeviceStateManaged = "MDMDeviceEnrolledManaged"
)

// MS-MDE2 Message constants
const (
	// Minimum supported version
	EnrollmentVersionV4 = "4.0"

	// Maximum supported version
	EnrollmentVersionV5 = "5.0"

	// xsi:nil indicates value is not present
	DefaultStateXSI = "true"

	// Supported authentication types
	AuthOnPremise = "OnPremise"

	// SOAP Fault codes
	SoapFaultRecv = "s:receiver"

	// SOAP Fault default error locale
	SoapFaultLocale = "en-us"

	// HTTP Content Type for SOAP responses
	SoapContentType = "application/soap+xml; charset=utf-8"

	// HTTP Content Type for SyncML MDM responses
	SyncMLContentType = "application/vnd.syncml.dm+xml"

	// HTTP Content Type for Webcontainer responses
	WebContainerContentType = "text/html; charset=UTF-8"

	// Minimal Key Length for SHA1WithRSA encryption
	PolicyMinKeyLength = "2048"

	// Certificate Validity Period in seconds (365 days)
	PolicyCertValidityPeriodInSecs = "31536000"

	// Certificate Renewal Period in seconds (180 days)
	PolicyCertRenewalPeriodInSecs = "15552000"

	// Supported Enroll Type
	ReqSecTokenEnrollType = "Full"

	// Provisioning Doc Certificate Renewal Period (365 days)
	WstepCertRenewalPeriodInDays = "365"

	// Provisioning Doc Server supports ROBO auto certificate renewal
	// TODO: Add renewal support
	WstepROBOSupport = "true"

	// Provisioning Doc Server retry interval
	WstepRenewRetryInterval = "4"

	// The PROVIDER-ID paramer specifies the server identifier for a management server used in the current management session
	DocProvisioningAppProviderID = "FleetDM"

	// The NAME parameter is used in the APPLICATION characteristic to specify a user readable application identity
	DocProvisioningAppName = DocProvisioningAppProviderID

	// The CONNRETRYFREQ parameter is used in the APPLICATION characteristic to specify a user readable application identity
	DocProvisioningAppConnRetryFreq = "6"

	// The INITIALBACKOFFTIME parameter is used to specify the initial wait time in milliseconds when the DM client retries for the first time
	DocProvisioningAppInitialBackoffTime = "30000"

	// The MAXBACKOFFTIME parameter is used to specify the maximum number of milliseconds to sleep after package-sending failure
	DocProvisioningAppMaxBackoffTime = "120000"

	// The DocProvisioningVersion attributes defines the version of the provisioning document format
	DocProvisioningVersion = "1.1"

	// The number of times the DM client should retry to connect to the server when the client is initially configured or enrolled to communicate with the server.
	DmClientCSPNumberOfFirstRetries = "8"

	// The waiting time (in minutes) for the initial set of retries as specified by the number of retries in NumberOfFirstRetries
	DmClientCSPIntervalForFirstSetOfRetries = "15"

	// The number of times the DM client should retry a second round of connecting to the server when the client is initially configured/enrolled to communicate with the server
	DmClientCSPNumberOfSecondRetries = "5"

	// The waiting time (in minutes) for the second set of retries as specified by the number of retries in NumberOfSecondRetries
	DmClientCSPIntervalForSecondSetOfRetries = "3"

	// The number of times the DM client should retry connecting to the server when the client is initially configured/enrolled to communicate with the server
	DmClientCSPNumberOfRemainingScheduledRetries = "0"

	// The waiting time (in minutes) for the initial set of retries as specified by the number of retries in NumberOfRemainingScheduledRetries
	DmClientCSPIntervalForRemainingScheduledRetries = "1560"

	// It allows the IT admin to require the device to start a management session on any user login, regardless of if the user has preciously logged in
	DmClientCSPPollOnLogin = "true"

	// It specifies whether the DM client should send out a request pending alert in case the device response to a DM request is too slow.
	DmClientCSPEnableOmaDmKeepAliveMessage = "true"

	// CSR issuer should be verified during enrollment
	EnrollVerifyIssue = true

	// Int type used by the DM client configuration
	DmClientIntType = "integer"

	// Bool type used by the DM client configuration
	DmClientBoolType = "boolean"

	// Additional Context items present on the RequestSecurityToken token message
	ReqSecTokenContextItemUXInitiated          = "UXInitiated"
	ReqSecTokenContextItemHWDevID              = "HWDevID"
	ReqSecTokenContextItemLocale               = "Locale"
	ReqSecTokenContextItemTargetedUserLoggedIn = "TargetedUserLoggedIn"
	ReqSecTokenContextItemOSEdition            = "OSEdition"
	ReqSecTokenContextItemDeviceName           = "DeviceName"
	ReqSecTokenContextItemDeviceID             = "DeviceID"
	ReqSecTokenContextItemEnrollmentType       = "EnrollmentType"
	ReqSecTokenContextItemDeviceType           = "DeviceType"
	ReqSecTokenContextItemOSVersion            = "OSVersion"
	ReqSecTokenContextItemApplicationVersion   = "ApplicationVersion"
	ReqSecTokenContextItemNotInOobe            = "NotInOobe"
	ReqSecTokenContextItemRequestVersion       = "RequestVersion"

	// APPRU query param expected by STS Auth endpoint
	STSAuthAppRu = "appru"

	// Login related query param expected by STS Auth endpoint
	STSLoginHint = "login_hint"

	// redirect_uri query param expected by TOS endpoint
	TOCRedirectURI = "redirect_uri"

	// client-request-id query param expected by TOS endpoint
	TOCReqID = "client-request-id"

	// Alert Command IDs
	DeviceUnenrollmentID = "1226"
	HostInitMessageID    = "1201"
)

func ResolveWindowsMDMDiscovery(serverURL string) (string, error) {
	return commonmdm.ResolveURL(serverURL, MDE2DiscoveryPath, false)
}

func ResolveWindowsMDMPolicy(serverURL string) (string, error) {
	return commonmdm.ResolveURL(serverURL, MDE2PolicyPath, false)
}

func ResolveWindowsMDMEnroll(serverURL string) (string, error) {
	return commonmdm.ResolveURL(serverURL, MDE2EnrollPath, false)
}

func ResolveWindowsMDMAuth(serverURL string) (string, error) {
	return commonmdm.ResolveURL(serverURL, MDE2AuthPath, false)
}

func ResolveWindowsMDMManagement(serverURL string) (string, error) {
	return commonmdm.ResolveURL(serverURL, MDE2ManagementPath, false)
}

package microsoft_mdm

import (
	"github.com/fleetdm/fleet/v4/server/mdm/internal/commonmdm"
)

const (
	// MDMPath is Fleet's HTTP path for the core Microsoft MDM service.
	MDMPath = "/api/mdm/microsoft"

	// DiscoveryPath is the HTTP endpoint path that serves the IDiscoveryService functionality.
	// This is the endpoint that process the Discover and DiscoverResponse messages
	// See the section 3.1 on the MS-MDE2 specification for more details:
	// https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-mde2/2681fd76-1997-4557-8963-cf656ab8d887
	MDE2DiscoveryPath = MDMPath + "/discovery"

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

	// These are the entry points for the Microsoft Device Enrollment (MS-MDE) and Microsoft Device Enrollment v2 (MS-MDE2) protocols.
	// These are required to be implemented by the MDM server to support user-driven enrollments
	MSEnrollEntryPoint = "/EnrollmentServer/Discovery.svc"
	MSManageEntryPoint = "/ManagementServer/MDM.svc"
)

// XML Namespaces used by the Microsoft Device Enrollment v2 protocol (MS-MDE2)
const (
	DiscoverNS        = "http://schemas.microsoft.com/windows/management/2012/01/enrollment"
	PolicyNS          = "http://schemas.microsoft.com/windows/pki/2009/01/enrollmentpolicy"
	EnrollWSTrust     = "http://docs.oasis-open.org/ws-sx/ws-trust/200512"
	EnrollSecExt      = "http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd"
	EnrollTType       = "http://schemas.microsoft.com/5.0.0.0/ConfigurationManager/Enrollment/DeviceEnrollmentToken"
	EnrollPDoc        = "http://schemas.microsoft.com/5.0.0.0/ConfigurationManager/Enrollment/DeviceEnrollmentProvisionDoc"
	EnrollEncode      = "http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd#base64binary"
	EnrollReq         = "http://schemas.microsoft.com/windows/pki/2009/01/enrollment"
	EnrollNSS         = "http://www.w3.org/2003/05/soap-envelope"
	EnrollNSA         = "http://www.w3.org/2005/08/addressing"
	EnrollXSI         = "http://www.w3.org/2001/XMLSchema-instance"
	EnrollXSD         = "http://www.w3.org/2001/XMLSchema"
	EnrollXSU         = "http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd"
	ActionNsDiag      = "http://schemas.microsoft.com/2004/09/ServiceModel/Diagnostics"
	ActionNsDiscovery = "http://schemas.microsoft.com/windows/management/2012/01/enrollment/IDiscoveryService/DiscoverResponse"
	ActionNsPolicy    = "http://schemas.microsoft.com/windows/pki/2009/01/enrollmentpolicy/IPolicy/GetPoliciesResponse"
	ActionNsEnroll    = "http://schemas.microsoft.com/windows/pki/2009/01/enrollment/RSTRC/wstep"
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

// MS-MDE2 Message constants
const (
	// Minimum supported version
	MinEnrollmentVersion = "4.0"

	// Maximum supported version
	MaxEnrollmentVersion = "5.0"

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

	// Minimal Key Length for SHA1WithRSA encryption
	PolicyMinKeyLength = "2048"

	// Certificate Validity Period in seconds (365 days)
	PolicyCertValidityPeriodInSecs = "31536000"

	// Certificate Renewal Period in seconds (180 days)
	PolicyCertRenewalPeriodInSecs = "15552000"
)

func ResolveMicrosoftMDMDiscovery(serverURL string) (string, error) {
	return commonmdm.ResolveURL(serverURL, MDE2DiscoveryPath, false)
}

func ResolveMicrosoftMDMPolicy(serverURL string) (string, error) {
	return commonmdm.ResolveURL(serverURL, MDE2PolicyPath, false)
}

func ResolveMicrosoftMDMEnroll(serverURL string) (string, error) {
	return commonmdm.ResolveURL(serverURL, MDE2EnrollPath, false)
}

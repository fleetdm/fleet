package microsoft_mdm

import (
	"crypto/x509"
	"encoding/base64"

	"github.com/fleetdm/fleet/v4/server/mdm/internal/commonmdm"
	"github.com/smallstep/pkcs7"
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

func ResolveWindowsMDMDiscovery(serverURL string) (string, error) {
	return commonmdm.ResolveURL(serverURL, MDE2DiscoveryPath, false)
}

func ResolveWindowsMDMPolicy(serverURL string) (string, error) {
	return commonmdm.ResolveURL(serverURL, MDE2PolicyPath, false)
}

func ResolveWindowsMDMEnroll(serverURL string) (string, error) {
	return commonmdm.ResolveURL(serverURL, MDE2EnrollPath, false)
}

func ResolveWindowsMDMManagement(serverURL string) (string, error) {
	return commonmdm.ResolveURL(serverURL, MDE2ManagementPath, false)
}

// Encrypt uses pkcs7 to encrypt a raw value using the provided certificate.
// The returned encrypted value is base64-encoded.
func Encrypt(rawValue string, cert *x509.Certificate) (string, error) {
	encrypted, err := pkcs7.Encrypt([]byte(rawValue), []*x509.Certificate{cert})
	if err != nil {
		return "", err
	}
	b64Enc := base64.StdEncoding.EncodeToString(encrypted)
	return b64Enc, nil
}

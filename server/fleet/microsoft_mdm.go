package fleet

import (
	"encoding/xml"
)

// MS-MDE2 Message request types
const (
	MDEDiscovery = iota
	MDEPolicy
	MDEEnrollment
	MDEFault
)

///////////////////////////////////////////////////////////////
/// Microsoft MS-MDE2 SOAP types

// ResponseHeader is the header for MDM responses from the server
type ResponseHeader struct {
	Action     Action      `xml:"Action"`
	RelatesTo  string      `xml:"a:RelatesTo"`
	ActivityId *ActivityId `xml:"ActivityId,omitempty"`
	Security   *WsSecurity `xml:"o:Security,omitempty"`
}

// RequestHeader is the header for MDM requests to the server
type RequestHeader struct {
	Action    Action         `xml:"Action"`
	MessageID string         `xml:"MessageID"`
	ReplyTo   ReplyTo        `xml:"ReplyTo"`
	To        To             `xml:"To"`
	Security  *TokenSecurity `xml:"Security,omitempty"`
}

// BodyReponse is the body of the MDM SOAP response message
type BodyResponse struct {
	Xsd                                    *string                                 `xml:"xmlns:xsd,attr,omitempty"`
	Xsi                                    *string                                 `xml:"xmlns:xsi,attr,omitempty"`
	DiscoverResponse                       *DiscoverResponse                       `xml:"DiscoverResponse,omitempty"`
	GetPoliciesResponse                    *GetPoliciesResponse                    `xml:"GetPoliciesResponse,omitempty"`
	RequestSecurityTokenResponseCollection *RequestSecurityTokenResponseCollection `xml:"RequestSecurityTokenResponseCollection,omitempty"`
	SoapFault                              *SoapFault                              `xml:"s:fault,omitempty"`
}

// BodyRequest is the body of the MDM SOAP request message
type BodyRequest struct {
	Xsi                  *string               `xml:"xsi,attr,omitempty"`
	Xsd                  *string               `xml:"xsd,attr,omitempty"`
	Discover             *Discover             `xml:"Discover,omitempty"`
	GetPolicies          *GetPolicies          `xml:"GetPolicies,omitempty"`
	RequestSecurityToken *RequestSecurityToken `xml:"RequestSecurityToken,omitempty"`
}

// HTTP request header field used to indicate the intent of the SOAP request, using a URI value
// See section 6.1.1 on SOAP Spec - https://www.w3.org/TR/2000/NOTE-SOAP-20000508/#_Toc478383527
type Action struct {
	Content        string `xml:",chardata"`
	MustUnderstand string `xml:"mustUnderstand,attr"`
}

// ActivityId is a unique identifier for the activity
type ActivityId struct {
	Content       string `xml:",chardata"`
	CorrelationId string `xml:"CorrelationId,attr"`
	XmlNS         string `xml:"xmlns,attr"`
}

// Timestamp for certificate authentication
type Timestamp struct {
	ID      string `xml:"u:Id,attr"`
	Created string `xml:"u:Created"`
	Expires string `xml:"u:Expires"`
}

// Security token container
type WsSecurity struct {
	XmlNS          string    `xml:"xmlns:o,attr"`
	MustUnderstand string    `xml:"s:mustUnderstand,attr"`
	Timestamp      Timestamp `xml:"u:Timestamp"`
}

// Security token container for encoded security sensitive data
type BinSecurityToken struct {
	Content  string `xml:",chardata"`
	Value    string `xml:"ValueType,attr"`
	Encoding string `xml:"EncodingType,attr"`
}

// TokenSecurity is the security token container for BinSecurityToken
type TokenSecurity struct {
	MustUnderstand string           `xml:"mustUnderstand,attr"`
	Security       BinSecurityToken `xml:"BinarySecurityToken"`
}

// To target endpoint header field
type To struct {
	Content        string `xml:",chardata"`
	MustUnderstand string `xml:"mustUnderstand,attr"`
}

// ReplyTo message correlation header field
type ReplyTo struct {
	Address string `xml:"Address"`
}

///////////////////////////////////////////////////////////////
/// Discover MS-MDE2 Message request type
/// https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-mde2/36e33def-59ab-484f-b0bc-701496346925

type Discover struct {
	XmlNS   string          `xml:"xmlns,attr"`
	Request DiscoverRequest `xml:"request"`
}

type AuthPolicies struct {
	AuthPolicy []string `xml:"AuthPolicy"`
}

type DiscoverRequest struct {
	XmlNS              string       `xml:"i,attr"`
	EmailAddress       string       `xml:"EmailAddress"`
	RequestVersion     string       `xml:"RequestVersion"`
	DeviceType         string       `xml:"DeviceType"`
	ApplicationVersion string       `xml:"ApplicationVersion"`
	OSEdition          string       `xml:"OSEdition"`
	AuthPolicies       AuthPolicies `xml:"AuthPolicies"`
}

///////////////////////////////////////////////////////////////
/// GetPolicies MS-MDE2 Message request type
/// https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-mde2/02b080e4-d1d8-4e0c-af14-b77931cec404

type GetPolicies struct {
	XmlNS         string        `xml:"xmlns,attr"`
	Client        Client        `xml:"client"`
	RequestFilter RequestFilter `xml:"requestFilter"`
}

type ClientContent struct {
	Content string `xml:",chardata"`
	Xsi     string `xml:"nil,attr"`
}

type Client struct {
	LastUpdate        ClientContent `xml:"lastUpdate"`
	PreferredLanguage ClientContent `xml:"preferredLanguage"`
}

type RequestFilter struct {
	Xsi string `xml:"nil,attr"`
}

///////////////////////////////////////////////////////////////
/// RequestSecurityToken MS-MDE2 Message request type
/// https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-mde2/6ba9c509-8bce-4899-85b2-8c3d41f8f845

type RequestSecurityToken struct {
	TokenType           string              `xml:"TokenType"`
	RequestType         string              `xml:"RequestType"`
	BinarySecurityToken BinarySecurityToken `xml:"BinarySecurityToken"`
	AdditionalContext   AdditionalContext   `xml:"AdditionalContext"`
}

type BinarySecurityToken struct {
	Content      string  `xml:",chardata"`
	XmlNS        *string `xml:"xmlns,attr"`
	ValueType    string  `xml:"ValueType,attr"`
	EncodingType string  `xml:"EncodingType,attr"`
}

type ContextItem struct {
	Name  string `xml:"Name,attr"`
	Value string `xml:"Value"`
}

type AdditionalContext struct {
	XmlNS       string        `xml:"xmlns,attr"`
	ContextItem []ContextItem `xml:"ContextItem"`
}

///////////////////////////////////////////////////////////////
/// DiscoverResponse MS-MDE2 Message response type
/// https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-mde2/aa198049-e691-41f9-a45a-b973b9089be7

type DiscoverResponse struct {
	XMLName        xml.Name       `xml:"DiscoverResponse"`
	XmlNS          string         `xml:"xmlns,attr"`
	DiscoverResult DiscoverResult `xml:"DiscoverResult"`
}

type DiscoverResult struct {
	AuthPolicy                 string `xml:"AuthPolicy"`
	EnrollmentVersion          string `xml:"EnrollmentVersion"`
	EnrollmentPolicyServiceUrl string `xml:"EnrollmentPolicyServiceUrl"`
	EnrollmentServiceUrl       string `xml:"EnrollmentServiceUrl"`
}

///////////////////////////////////////////////////////////////
/// GetPoliciesResponse MS-MDE2 Message response type
/// https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-mde2/6e74dcdb-c3d9-4044-af10-536224904e72

type GetPoliciesResponse struct {
	XMLName  xml.Name `xml:"GetPoliciesResponse"`
	XmlNS    string   `xml:"xmlns,attr"`
	Response Response `xml:"response"`
	OIDs     OIDs     `xml:"oIDs"`
}

type ContentAttr struct {
	Content string `xml:",chardata"`
	Xsi     string `xml:"xsi:nil,attr"`
	XmlNS   string `xml:"xmlns:xsi,attr"`
}

type GenericAttr struct {
	Xsi string `xml:"xsi:nil,attr"`
}

type CertificateValidity struct {
	ValidityPeriodSeconds string `xml:"validityPeriodSeconds"`
	RenewalPeriodSeconds  string `xml:"renewalPeriodSeconds"`
}

type Permission struct {
	Enroll     string `xml:"enroll"`
	AutoEnroll string `xml:"autoEnroll"`
}

type PrivateKeyAttributes struct {
	MinimalKeyLength      string      `xml:"minimalKeyLength"`
	KeySpec               GenericAttr `xml:"keySpec"`
	KeyUsageProperty      GenericAttr `xml:"keyUsageProperty"`
	Permissions           GenericAttr `xml:"permissions"`
	AlgorithmOIDReference GenericAttr `xml:"algorithmOIDReference"`
	CryptoProviders       GenericAttr `xml:"cryptoProviders"`
}
type Revision struct {
	MajorRevision string `xml:"majorRevision"`
	MinorRevision string `xml:"minorRevision"`
}

type Attributes struct {
	CommonName                string               `xml:"commonName"`
	PolicySchema              string               `xml:"policySchema"`
	CertificateValidity       CertificateValidity  `xml:"certificateValidity"`
	Permission                Permission           `xml:"permission"`
	PrivateKeyAttributes      PrivateKeyAttributes `xml:"privateKeyAttributes"`
	Revision                  Revision             `xml:"revision"`
	SupersededPolicies        GenericAttr          `xml:"supersededPolicies"`
	PrivateKeyFlags           GenericAttr          `xml:"privateKeyFlags"`
	SubjectNameFlags          GenericAttr          `xml:"subjectNameFlags"`
	EnrollmentFlags           GenericAttr          `xml:"enrollmentFlags"`
	GeneralFlags              GenericAttr          `xml:"generalFlags"`
	HashAlgorithmOIDReference string               `xml:"hashAlgorithmOIDReference"`
	RARequirements            GenericAttr          `xml:"rARequirements"`
	KeyArchivalAttributes     GenericAttr          `xml:"keyArchivalAttributes"`
	Extensions                GenericAttr          `xml:"extensions"`
}

type GPPolicy struct {
	PolicyOIDReference string      `xml:"policyOIDReference"`
	CAs                GenericAttr `xml:"cAs"`
	Attributes         Attributes  `xml:"attributes"`
}

type Policies struct {
	Policy GPPolicy `xml:"policy"`
}

type Response struct {
	PolicyID           string      `xml:"policyID"`
	PolicyFriendlyName ContentAttr `xml:"policyFriendlyName"`
	NextUpdateHours    ContentAttr `xml:"nextUpdateHours"`
	PoliciesNotChanged ContentAttr `xml:"policiesNotChanged"`
	Policies           Policies    `xml:"policies"`
}

type OID struct {
	Value          string `xml:"value"`
	Group          string `xml:"group"`
	OIDReferenceID string `xml:"oIDReferenceID"`
	DefaultName    string `xml:"defaultName"`
}

type OIDs struct {
	Content string `xml:",chardata"`
	OID     OID    `xml:"oID"`
}

///////////////////////////////////////////////////////////////
/// RequestSecurityTokenResponseCollection MS-MDE2 Message response type
/// https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-mde2/3452fe7d-2441-49f7-8801-c056b58edb6a

type RequestSecurityTokenResponseCollection struct {
	XMLName                      xml.Name                     `xml:"RequestSecurityTokenResponseCollection"`
	XmlNS                        string                       `xml:"xmlns,attr"`
	RequestSecurityTokenResponse RequestSecurityTokenResponse `xml:"RequestSecurityTokenResponse"`
}

type SecAttr struct {
	Content string `xml:",chardata"`
	XmlNS   string `xml:"xmlns,attr"`
}

type RequestedSecurityToken struct {
	BinarySecurityToken BinarySecurityToken `xml:"BinarySecurityToken"`
}

type RequestSecurityTokenResponse struct {
	TokenType              string                 `xml:"TokenType"`
	DispositionMessage     SecAttr                `xml:"DispositionMessage"`
	RequestedSecurityToken RequestedSecurityToken `xml:"RequestedSecurityToken"`
	RequestID              SecAttr                `xml:"RequestID"`
}

///////////////////////////////////////////////////////////////
/// SoapFault MS-MDE2 Message response type
/// https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-mde2/0a78f419-5fd7-4ddb-bc76-1c0f7e11da23

type Subcode struct {
	Value string `xml:"s:value"`
}

type Code struct {
	Value   string  `xml:"s:value"`
	Subcode Subcode `xml:"s:subcode"`
}

type ReasonText struct {
	Content string `xml:",chardata"`
	Lang    string `xml:"xml:lang,attr"`
}

type Reason struct {
	Text ReasonText `xml:"s:text"`
}

type SoapFault struct {
	XMLName             xml.Name `xml:"s:fault"`
	Code                Code     `xml:"s:code"`
	Reason              Reason   `xml:"s:reason"`
	OriginalMessageType int      `xml:"-"`
}

// MicrosoftMDMAccessTokenPayload is the payload that gets encoded as JSON and
// provided as opaque access token to the RegisterDeviceWithManagement API.
type MicrosoftMDMAccessTokenPayload struct {
	// Type is the enrollment type, such as "programmatic".
	Type    MicrosoftMDMEnrollmentType `json:"type"`
	Payload struct {
		HostUUID string `json:"host_uuid"`
	} `json:"payload"`
}

type MicrosoftMDMEnrollmentType int

// List of supported Microsoft MDM enrollment types.
const (
	MicrosoftMDMProgrammaticEnrollmentType MicrosoftMDMEnrollmentType = 1
)

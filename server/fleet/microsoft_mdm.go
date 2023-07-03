package fleet

import (
	"encoding/xml"
	"errors"
	"fmt"

	mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
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
	XMLName xml.Name       `xml:"s:Envelope"`
	XMLNSS  string         `xml:"xmlns:s,attr"`
	XMLNSA  string         `xml:"xmlns:a,attr"`
	XMLNSU  *string        `xml:"xmlns:u,attr,omitempty"`
	Header  ResponseHeader `xml:"s:Header"`
	Body    BodyResponse   `xml:"s:Body"`
}

// SoapRequest is the Soap Envelope Request type for MS-MDE2 responses to the server
// This envelope XML message is composed by a mandatory SOAP envelope, a SOAP header, and a SOAP body
type SoapRequest struct {
	XMLName   xml.Name      `xml:"Envelope"`
	XMLNSS    string        `xml:"s,attr"`
	XMLNSA    string        `xml:"a,attr"`
	XMLNSU    *string       `xml:"u,attr,omitempty"`
	XMLNSWsse *string       `xml:"wsse,attr,omitempty"`
	XMLNSWST  *string       `xml:"wst,attr,omitempty"`
	XMLNSAC   *string       `xml:"ac,attr,omitempty"`
	Header    RequestHeader `xml:"Header"`
	Body      BodyRequest   `xml:"Body"`
}

// GetBinarySecurityToken returns the header BinarySecurityToken if present
func (req *SoapRequest) GetBinarySecurityToken() (string, error) {
	if req.Header.Security == nil {
		return "", errors.New("header BinarySecurityToken is not present")
	}

	return req.Header.Security.Security.Content, nil
}

// GetMessageID returns the message ID from the header
func (req *SoapRequest) GetMessageID() string {
	return req.Header.MessageID
}

// isValidHeader checks for required fields in the header
func (req *SoapRequest) isValidHeader() error {
	// Check for required fields

	if len(req.XMLNSS) == 0 {
		return errors.New("invalid SOAP header: XMLNSS")
	}

	if len(req.XMLNSA) == 0 {
		return errors.New("invalid SOAP header: XMLNSA")
	}

	if len(req.Header.MessageID) == 0 {
		return errors.New("invalid SOAP header: Header.MessageID")
	}

	if len(req.Header.Action.Content) == 0 {
		return errors.New("invalid SOAP header: Header.Action")
	}

	if len(req.Header.ReplyTo.Address) == 0 {
		return errors.New("invalid SOAP header: Header.ReplyTo")
	}

	if len(req.Header.To.Content) == 0 {
		return errors.New("invalid SOAP header: Header.To")
	}

	return nil
}

// isValidBody checks for the presence of only one message
func (req *SoapRequest) isValidBody() error {
	nonNilCount := 0

	if req.Body.Discover != nil {
		nonNilCount++
	}
	if req.Body.GetPolicies != nil {
		nonNilCount++
	}
	if req.Body.RequestSecurityToken != nil {
		nonNilCount++
	}

	if nonNilCount != 1 {
		return errors.New("invalid SOAP body: Multiple messages or no message")
	}

	return nil
}

// IsValidDiscoveryMsg checks for required fields in the Discover message
func (req *SoapRequest) IsValidDiscoveryMsg() error {
	if err := req.isValidHeader(); err != nil {
		return fmt.Errorf("invalid discover message: %s", err)
	}

	if err := req.isValidBody(); err != nil {
		return fmt.Errorf("invalid discover message: %s", err)
	}

	if req.Body.Discover == nil {
		return errors.New("invalid discover message: Discover message not present")
	}

	if len(req.Body.Discover.XMLNS) == 0 {
		return errors.New("invalid discover message: XMLNS")
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
		return fmt.Errorf("invalid getpolicies message:  %s", err)
	}

	if err := req.isValidBody(); err != nil {
		return fmt.Errorf("invalid getpolicies message: %s", err)
	}

	if req.Body.GetPolicies == nil {
		return errors.New("invalid getpolicies message:  GetPolicies message not present")
	}

	if len(req.Body.GetPolicies.XMLNS) == 0 {
		return errors.New("invalid getpolicies message: XMLNS")
	}

	return nil
}

// IsValidRequestSecurityTokenMsg checks for required fields in the RequestSecurityToken message
func (req *SoapRequest) IsValidRequestSecurityTokenMsg() error {
	if err := req.isValidHeader(); err != nil {
		return fmt.Errorf("invalid requestsecuritytoken message: %s", err)
	}

	if err := req.isValidBody(); err != nil {
		return fmt.Errorf("invalid requestsecuritytoken message: %s", err)
	}

	if req.Body.RequestSecurityToken == nil {
		return errors.New("invalid requestsecuritytoken message: RequestSecurityToken message not present")
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

// MS-MDE2 Message request types
const (
	MDEDiscovery = iota
	MDEPolicy
	MDEEnrollment
	MDEFault
)

///////////////////////////////////////////////////////////////
/// Microsoft MS-MDE2 SOAP messages

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
	XMLNS         string `xml:"xmlns,attr"`
}

// Timestamp for certificate authentication
type Timestamp struct {
	ID      string `xml:"u:Id,attr"`
	Created string `xml:"u:Created"`
	Expires string `xml:"u:Expires"`
}

// Security token container
type WsSecurity struct {
	XMLNS          string    `xml:"xmlns:o,attr"`
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
	XMLNS   string          `xml:"xmlns,attr"`
	Request DiscoverRequest `xml:"request"`
}

type AuthPolicies struct {
	AuthPolicy []string `xml:"AuthPolicy"`
}

type DiscoverRequest struct {
	XMLNS              string       `xml:"i,attr"`
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
	XMLNS         string        `xml:"xmlns,attr"`
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
	XMLNS        *string `xml:"xmlns,attr"`
	ValueType    string  `xml:"ValueType,attr"`
	EncodingType string  `xml:"EncodingType,attr"`
}

type ContextItem struct {
	Name  string `xml:"Name,attr"`
	Value string `xml:"Value"`
}

type AdditionalContext struct {
	XMLNS       string        `xml:"xmlns,attr"`
	ContextItem []ContextItem `xml:"ContextItem"`
}

///////////////////////////////////////////////////////////////
/// DiscoverResponse MS-MDE2 Message response type
/// https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-mde2/aa198049-e691-41f9-a45a-b973b9089be7

type DiscoverResponse struct {
	XMLName        xml.Name       `xml:"DiscoverResponse"`
	XMLNS          string         `xml:"xmlns,attr"`
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
	XMLNS    string   `xml:"xmlns,attr"`
	Response Response `xml:"response"`
	OIDs     OIDs     `xml:"oIDs"`
}

type ContentAttr struct {
	Content string `xml:",chardata"`
	Xsi     string `xml:"xsi:nil,attr"`
	XMLNS   string `xml:"xmlns:xsi,attr"`
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
	XMLNS                        string                       `xml:"xmlns,attr"`
	RequestSecurityTokenResponse RequestSecurityTokenResponse `xml:"RequestSecurityTokenResponse"`
}

type SecAttr struct {
	Content string `xml:",chardata"`
	XMLNS   string `xml:"xmlns,attr"`
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

// ///////////////////////////////////////////////////////////////////////////
// / WindowsMDMAccessTokenPayload is the payload that gets encoded as JSON and
// / provided as opaque access token to the RegisterDeviceWithManagement API.
type WindowsMDMAccessTokenPayload struct {
	// Type is the enrollment type, such as "programmatic".
	Type    WindowsMDMEnrollmentType `json:"type"`
	Payload struct {
		HostUUID string `json:"host_uuid"`
	} `json:"payload"`
}

type WindowsMDMEnrollmentType int

// List of supported Windows MDM enrollment types.
const (
	WindowsMDMProgrammaticEnrollmentType WindowsMDMEnrollmentType = 1
)

func (t *WindowsMDMAccessTokenPayload) IsValidToken() error {
	// Only BSProgrammaticEnrollment are supported for now
	if t.Type != WindowsMDMProgrammaticEnrollmentType {
		return errors.New("invalid binary security payload type")
	}

	if len(t.Payload.HostUUID) == 0 {
		return errors.New("invalid binary security payload content")
	}

	return nil
}

func (t *WindowsMDMAccessTokenPayload) GetType() WindowsMDMEnrollmentType {
	return t.Type
}

package fleet

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm/microsoft/syncml"
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

	// Raw XML, stored alongside the decoded fields for convenience
	Raw []byte `xml:"-"`
}

// GetHeaderBinarySecurityToken returns the header BinarySecurityToken if present
func (req *SoapRequest) GetHeaderBinarySecurityToken() (*HeaderBinarySecurityToken, error) {
	if req.Header.Security == nil {
		return nil, errors.New("binarySecurityToken is not present")
	}

	if len(req.Header.Security.Security.Content) == 0 {
		return nil, errors.New("binarySecurityToken is empty")
	}

	if req.Header.Security.Security.Encoding != syncml.EnrollEncode {
		return nil, errors.New("binarySecurityToken encoding is invalid")
	}

	if req.Header.Security.Security.Value != syncml.BinarySecurityDeviceEnroll && req.Header.Security.Security.Value != syncml.BinarySecurityAzureEnroll {
		return nil, errors.New("binarySecurityToken type is invalid")
	}

	return &req.Header.Security.Security, nil
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

	// Check if the request version is one of the defined enrollment versions
	versionFound := false
	for _, v := range syncml.SupportedEnrollmentVersions {
		if req.Body.Discover.Request.RequestVersion == v {
			versionFound = true
			break
		}
	}

	if !versionFound {
		return errors.New("invalid discover message: Request.RequestVersion")
	}

	// Traverse the AuthPolicies slice and check for valid values
	isInvalidAuth := true
	for _, authPolicy := range req.Body.Discover.Request.AuthPolicies.AuthPolicy {
		if authPolicy == syncml.AuthOnPremise {
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

	if req.Body.RequestSecurityToken.BinarySecurityToken.ValueType != syncml.EnrollReqTypePKCS10 &&
		req.Body.RequestSecurityToken.BinarySecurityToken.ValueType != syncml.EnrollReqTypePKCS7 {
		return errors.New("invalid requestsecuritytoken message: BinarySecurityToken.EncodingType not supported")
	}

	if len(req.Body.RequestSecurityToken.BinarySecurityToken.Content) == 0 {
		return errors.New("invalid requestsecuritytoken message: BinarySecurityToken.Content")
	}

	if len(req.Body.RequestSecurityToken.AdditionalContext.ContextItems) == 0 {
		return errors.New("invalid requestsecuritytoken message: AdditionalContext.ContextItems missing")
	}

	reqEnrollType, err := req.Body.RequestSecurityToken.GetContextItem(syncml.ReqSecTokenContextItemEnrollmentType)
	if err != nil || (reqEnrollType != syncml.ReqSecTokenEnrollTypeDevice && reqEnrollType != syncml.ReqSecTokenEnrollTypeFull) {
		return fmt.Errorf("invalid requestsecuritytoken message %s: %s - %v", syncml.ReqSecTokenContextItemEnrollmentType, reqEnrollType, err)
	}

	reqDeviceID, err := req.Body.RequestSecurityToken.GetContextItem(syncml.ReqSecTokenContextItemDeviceID)
	if err != nil || len(reqDeviceID) == 0 {
		return fmt.Errorf("invalid requestsecuritytoken message %s: %s - %v", syncml.ReqSecTokenContextItemDeviceID, reqDeviceID, err)
	}

	reqHwDeviceID, err := req.Body.RequestSecurityToken.GetContextItem(syncml.ReqSecTokenContextItemHWDevID)
	if err != nil || len(reqHwDeviceID) == 0 {
		return fmt.Errorf("invalid requestsecuritytoken message %s: %s - %v", syncml.ReqSecTokenContextItemHWDevID, reqHwDeviceID, err)
	}

	reqOSEdition, err := req.Body.RequestSecurityToken.GetContextItem(syncml.ReqSecTokenContextItemOSEdition)
	if err != nil || len(reqOSEdition) == 0 {
		return fmt.Errorf("invalid requestsecuritytoken message %s: %s - %v", syncml.ReqSecTokenContextItemOSEdition, reqOSEdition, err)
	}

	reqOSVersion, err := req.Body.RequestSecurityToken.GetContextItem(syncml.ReqSecTokenContextItemOSVersion)
	if err != nil || len(reqOSVersion) == 0 {
		return fmt.Errorf("invalid requestsecuritytoken message %s: %s - %v", syncml.ReqSecTokenContextItemOSVersion, reqOSVersion, err)
	}

	return nil
}

// Get RequestSecurityToken MDM Message from the body
func (req *SoapRequest) GetRequestSecurityTokenMessage() (*RequestSecurityToken, error) {
	if req.Body.RequestSecurityToken == nil {
		return nil, errors.New("invalid body: RequestSecurityToken message not present")
	}

	return req.Body.RequestSecurityToken, nil
}

// MS-MDE2 and MS-MDM Message types
const (
	MDEDiscovery = iota
	MDEPolicy
	MDEEnrollment
	MDEFault
	MSMDM
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
type HeaderBinarySecurityToken struct {
	Content  string `xml:",chardata"`
	Value    string `xml:"ValueType,attr"`
	Encoding string `xml:"EncodingType,attr"`
}

// Get RequestSecurityToken MDM Message from the body
func (token *HeaderBinarySecurityToken) IsValidToken() error {
	if token == nil {
		return errors.New("binary security token is not present")
	}

	if len(token.Content) == 0 {
		return errors.New("binary security token is empty")
	}

	if token.Value != syncml.BinarySecurityDeviceEnroll && token.Value != syncml.BinarySecurityAzureEnroll {
		return errors.New("binary security token is invalid")
	}

	return nil
}

// Check if input token is a valid Azure JWT token
func (token *HeaderBinarySecurityToken) IsAzureJWTToken() bool {
	if token == nil {
		return false
	}

	if token.Value == syncml.BinarySecurityAzureEnroll {
		return true
	}

	return false
}

// Check if input token is a valid Device Enroll token
func (token *HeaderBinarySecurityToken) IsDeviceToken() bool {
	if token == nil {
		return false
	}

	if token.Value == syncml.BinarySecurityDeviceEnroll {
		return true
	}

	return false
}

// TokenSecurity is the security token container for BinSecurityToken
type TokenSecurity struct {
	MustUnderstand string                    `xml:"mustUnderstand,attr"`
	Security       HeaderBinarySecurityToken `xml:"BinarySecurityToken"`
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
	TokenType           string                 `xml:"TokenType"`
	RequestType         string                 `xml:"RequestType"`
	BinarySecurityToken BinarySecurityToken    `xml:"BinarySecurityToken"`
	AdditionalContext   AdditionalContext      `xml:"AdditionalContext"`
	MapContextItems     map[string]ContextItem `xml:"-"`
}

// BinarySecurityToken contains the base64 encoding representation of the security token
// BinarySecurityToken is the sole operation in the WSTEP protocol. It provides the mechanism for
// certificate enrollment requests, retrieval of pending certificate status, and the request of the
// server key exchange certificate.
// The token format is defined by the WS-Trust X509v3 Enrollment Extensions [MS-WSTEP] specification
// https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-mde2/5b02c625-ced2-4a01-a8e1-da0ae84f5bb7
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
	XMLNS        string        `xml:"xmlns,attr"`
	ContextItems []ContextItem `xml:"ContextItem"`
}

// Get Binary Security Token
func (msg RequestSecurityToken) GetBinarySecurityTokenData() (string, error) {
	if len(msg.BinarySecurityToken.Content) == 0 {
		return "", errors.New("BinarySecurityToken is empty")
	}

	return msg.BinarySecurityToken.Content, nil
}

// Get Binary Security Token Type
func (msg RequestSecurityToken) GetBinarySecurityTokenType() (string, error) {
	if msg.BinarySecurityToken.ValueType == syncml.EnrollReqTypePKCS10 ||
		msg.BinarySecurityToken.ValueType == syncml.EnrollReqTypePKCS7 {
		return msg.BinarySecurityToken.ValueType, nil
	}

	return "", errors.New("BinarySecurityToken is invalid")
}

// Get SecurityToken Context Item
func (msg *RequestSecurityToken) GetContextItem(item string) (string, error) {
	if len(msg.AdditionalContext.ContextItems) == 0 {
		return "", errors.New("ContextItems is empty")
	}

	// Generate map of ContextItems if not there
	if msg.MapContextItems == nil {
		contextMap := make(map[string]ContextItem)
		for _, item := range msg.AdditionalContext.ContextItems {
			contextMap[item.Name] = item
		}
		msg.MapContextItems = contextMap
	}

	itemVal, ok := (msg.MapContextItems)[item]
	if !ok {
		return "", fmt.Errorf("ContextItem item %s is not present", item)
	}

	return itemVal.Value, nil
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
	AuthPolicy                 string  `xml:"AuthPolicy"`
	EnrollmentVersion          string  `xml:"EnrollmentVersion"`
	EnrollmentPolicyServiceUrl string  `xml:"EnrollmentPolicyServiceUrl"`
	EnrollmentServiceUrl       string  `xml:"EnrollmentServiceUrl"`
	AuthServiceUrl             *string `xml:"AuthenticationServiceUrl"`
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

type ProviderAttr struct {
	Content string `xml:",chardata"`
}

type PrivateKeyAttributes struct {
	MinimalKeyLength      string         `xml:"minimalKeyLength"`
	KeySpec               GenericAttr    `xml:"keySpec"`
	KeyUsageProperty      GenericAttr    `xml:"keyUsageProperty"`
	Permissions           GenericAttr    `xml:"permissions"`
	AlgorithmOIDReference GenericAttr    `xml:"algorithmOIDReference"`
	CryptoProviders       []ProviderAttr `xml:"provider"`
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
	OID     []OID  `xml:"oID"`
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

//////////////////////////////////////////////////////////////////////////////
/// WindowsMDMAccessTokenPayload is the payload that gets encoded as JSON and
/// provided as opaque access token to the RegisterDeviceWithManagement API

type WindowsMDMAccessTokenPayload struct {
	// Type is the enrollment type, such as "programmatic".
	Type    WindowsMDMEnrollmentType `json:"type"`
	Payload struct {
		OrbitNodeKey string `json:"orbit_node_key"`
		AuthToken    string `json:"auth_token"`
	} `json:"payload"`
}

type WindowsMDMEnrollmentType int

// List of supported Windows MDM enrollment types

const (
	WindowsMDMProgrammaticEnrollmentType WindowsMDMEnrollmentType = 1
	WindowsMDMAutomaticEnrollmentType    WindowsMDMEnrollmentType = 2
)

func (t *WindowsMDMAccessTokenPayload) IsValidToken() error {
	// Only BSProgrammaticEnrollment are supported for now
	if t.Type != WindowsMDMProgrammaticEnrollmentType && t.Type != WindowsMDMAutomaticEnrollmentType {
		return errors.New("invalid binary security payload type")
	}

	if t.Type == WindowsMDMProgrammaticEnrollmentType && len(t.Payload.OrbitNodeKey) == 0 {
		return errors.New("invalid binary security payload content")
	}

	if t.Type == WindowsMDMAutomaticEnrollmentType && len(t.Payload.AuthToken) == 0 {
		return errors.New("invalid STS auth token payload content")
	}

	return nil
}

func (t *WindowsMDMAccessTokenPayload) GetType() WindowsMDMEnrollmentType {
	return t.Type
}

///////////////////////////////////////////////////////////////
/// MS-MDE2 ProvisioningDoc (XML Provisioning Schema) message type
/// Section 2.2.9.1 on the specification
/// https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-mde2/35e1aca6-1b8a-48ba-bbc0-23af5d46907a

type Param struct {
	Name     string `xml:"name,attr,omitempty"`
	Value    string `xml:"value,attr,omitempty"`
	Datatype string `xml:"datatype,attr,omitempty"`
}

type Characteristic struct {
	Type            string           `xml:"type,attr"`
	Params          []Param          `xml:"parm"`
	Characteristics []Characteristic `xml:"characteristic,omitempty"`
}

type WapProvisioningDoc struct {
	XMLName         xml.Name         `xml:"wap-provisioningdoc"`
	Version         string           `xml:"version,attr"`
	Characteristics []Characteristic `xml:"characteristic"`
}

// Add Characteristic to the Characteristic container
func (msg *Characteristic) AddCharacteristic(c Characteristic) {
	msg.Characteristics = append(msg.Characteristics, c)
}

// Add Param to the Params container
func (msg *Characteristic) AddParam(name string, value string, dataType string) {
	param := Param{Name: name, Value: value, Datatype: dataType}
	msg.Params = append(msg.Params, param)
}

// Add Characteristic to the WapProvisioningDoc
func (msg *WapProvisioningDoc) AddCharacteristic(c Characteristic) {
	msg.Characteristics = append(msg.Characteristics, c)
}

// GetEncodedB64Representation returns encoded WapProvisioningDoc representation
func (msg WapProvisioningDoc) GetEncodedB64Representation() (string, error) {
	rawXML, err := xml.MarshalIndent(msg, "", "  ")
	if err != nil {
		return "", err
	}

	// Appending the XML header beforing encoding it
	xmlContent := append([]byte(xml.Header), rawXML...)

	// Create a replacer to replace both "\n" and "\t"
	replacer := strings.NewReplacer("\n", "", "\t", "")

	// Use the replacer on the string representation of xmlContent
	xmlStripContent := []byte(replacer.Replace(string(xmlContent)))

	return base64.StdEncoding.EncodeToString(xmlStripContent), nil
}

///////////////////////////////////////////////////////////////
/// MDMWindowsEnrolledDevice type
/// Contains the information of the enrolled Windows host

type MDMWindowsEnrolledDevice struct {
	ID                     uint      `db:"id"`
	HostUUID               string    `db:"host_uuid"`
	MDMDeviceID            string    `db:"mdm_device_id"`
	MDMHardwareID          string    `db:"mdm_hardware_id"`
	MDMDeviceState         string    `db:"device_state"`
	MDMDeviceType          string    `db:"device_type"`
	MDMDeviceName          string    `db:"device_name"`
	MDMEnrollType          string    `db:"enroll_type"`
	MDMEnrollUserID        string    `db:"enroll_user_id"`
	MDMEnrollProtoVersion  string    `db:"enroll_proto_version"`
	MDMEnrollClientVersion string    `db:"enroll_client_version"`
	MDMNotInOOBE           bool      `db:"not_in_oobe"`
	CreatedAt              time.Time `db:"created_at"`
	UpdatedAt              time.Time `db:"updated_at"`
}

func (e MDMWindowsEnrolledDevice) AuthzType() string {
	return "mdm_windows"
}

///////////////////////////////////////////////////////////////
// Microsoft MS-MDM message
// MS-MDM is a client-to-server protocol that consists of a SOAP-based Web service.
// MDM is based on the OMA-DM protocol. Messages are issued by a requester and results and status are returned by a responder as a SynCML message.
// A SyncML message is a well-formed XML document that adheres to the document type definition (DTD), but which does not require validation.
// The XML document is identified by a SyncML document element type that serves as a parent container for the SyncML message.
// The SyncML message consists of a header specified by the SyncHdr  element type and a body specified by the SyncBody element type.
// The SyncML header identifies the routing and versioning information about the SyncML message.
// The SyncML body functions as a container for one or more SyncML commands.
// A SyncML command is specified by individual element types that provide specific details about the command, including any data or meta-information.
// MS-MDM uses a subset of the SyncML message definition specified in OMA-SyncMLRP spec. MDM-specific SyncML xml message format is defined in OMA-DMRP.

type SyncML struct {
	XMLName  xml.Name `xml:"SyncML"`
	Xmlns    string   `xml:"xmlns,attr"`
	SyncHdr  SyncHdr  `xml:"SyncHdr"`
	SyncBody SyncBody `xml:"SyncBody"`

	// Raw XML, stored alongside the decoded fields for convenience
	Raw []byte `xml:"-"`
}

type SyncHdr struct {
	VerDTD    string   `xml:"VerDTD"`
	VerProto  string   `xml:"VerProto"`
	SessionID string   `xml:"SessionID"`
	MsgID     string   `xml:"MsgID"`
	Target    *LocURI  `xml:"Target,omitempty"`
	Source    *LocURI  `xml:"Source,omitempty"`
	Meta      *MetaHdr `xml:"Meta,omitempty"`
}

type MetaHdr struct {
	MaxMsgSize *string `xml:"MaxMsgSize,omitempty"`
}

// ProtoCmds contains a slice of SyncML protocol commands
type ProtoCmds []SyncMLCmd

// See supported Commands in section 2.2.7.1
type SyncBody struct {
	Final *string `xml:"Final,omitempty"`

	// Request Protocol Commands
	Add     ProtoCmds `xml:"Add,omitempty"`
	Alert   ProtoCmds `xml:"Alert,omitempty"`
	Atomic  ProtoCmds `xml:"Atomic,omitempty"`
	Delete  ProtoCmds `xml:"Delete,omitempty"`
	Exec    ProtoCmds `xml:"Exec,omitempty"`
	Get     ProtoCmds `xml:"Get,omitempty"`
	Replace ProtoCmds `xml:"Replace,omitempty"`

	// Response Protocol Commands
	Results ProtoCmds `xml:"Results,omitempty"`
	Status  ProtoCmds `xml:"Status,omitempty"`

	// Raw container
	Raw ProtoCmds `xml:",omitempty"`
}

// ProtoCmdState is the state of the SyncML protocol commands
type ProtoCmdState int

const (
	Received           ProtoCmdState = iota // Protocol Command was received
	Pending                                 // Protocol Command is on the pending queue and has not been sent yet
	Sent                                    // Protocol Command has been sent
	ResponseProcessing                      // Protocol Command was acknowledged and is being processed
	ResponseAck                             // Protocol Command was acknowledged and processed
)

// Supported protocol command verbs
const (
	CmdAdd     = "Add"     // Protocol Command verb Add
	CmdAlert   = "Alert"   // Protocol Command verb Alert
	CmdAtomic  = "Atomic"  // Protocol Command verb Atomic
	CmdDelete  = "Delete"  // Protocol Command verb Delete
	CmdExec    = "Exec"    // Protocol Command verb Exec
	CmdGet     = "Get"     // Protocol Command verb Get
	CmdReplace = "Replace" // Protocol Command verb Replace
	CmdResults = "Results" // Protocol Command verb Results
	CmdStatus  = "Status"  // Protocol Command verb Status
)

// ProtoCmdOperation is the abstraction to represent a SyncML Protocol Command
type ProtoCmdOperation struct {
	Verb string    `db:"verb"`
	Cmd  SyncMLCmd `db:"cmd"`
}

// CmdID struct holds the value of a CmdID and a flag to determine whether
// to include a comment when marshaling to XML.
type CmdID struct {
	Value               string
	IncludeFleetComment bool
}

// MarshalXML implements the xml.Marshaler interface
// It optionally adds a predefined comment before the CmdID value when marshaled to XML.
func (c CmdID) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if c.IncludeFleetComment {
		if err := e.EncodeToken(xml.Comment(" CmdID generated by Fleet ")); err != nil {
			return err
		}
	}

	return e.EncodeElement(c.Value, start)
}

// UnMarshalXML implements the xml.UnMarshaler interface
// It decodes an XML element's content into the CmdID's Value field.
func (c *CmdID) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var value string
	if err := d.DecodeElement(&value, &start); err != nil {
		return err
	}
	c.Value = value
	return nil
}

// Protocol Command
type SyncMLCmd struct {
	XMLName xml.Name  `xml:",omitempty"`
	CmdID   CmdID     `xml:"CmdID"`
	MsgRef  *string   `xml:"MsgRef,omitempty"`
	CmdRef  *string   `xml:"CmdRef,omitempty"`
	Cmd     *string   `xml:"Cmd,omitempty"`
	Data    *string   `xml:"Data,omitempty"`
	Items   []CmdItem `xml:"Item,omitempty"`

	// ReplaceCommands is a catch-all for any nested <Replace> commands,
	// which can be found under <Atomic> elements.
	//
	// NOTE: in theory Atomics can have anything except Get verbs, but
	// for the moment we're not allowing anything besides Replaces and Adds
	ReplaceCommands []SyncMLCmd `xml:"Replace,omitempty"`

	// AddCommands is a catch-all for any nested <Add> commands,
	// which can be found under <Atomic> elements.
	AddCommands []SyncMLCmd `xml:"Add,omitempty"`
}

// ParseWindowsMDMCommand parses the raw XML as a single Windows MDM command.
// A single <Exec> command is accepted as input.
func ParseWindowsMDMCommand(rawXMLCmd []byte) (*SyncMLCmd, error) {
	// a command must have the <Exec> element as top-level
	var cmdMsg SyncMLCmd
	dec := xml.NewDecoder(bytes.NewReader(bytes.TrimSpace(rawXMLCmd)))
	if err := dec.Decode(&cmdMsg); err != nil {
		return nil, fmt.Errorf("The payload isn't valid XML: %w", err)
	}

	// check if there were multiple top-level elements provided
	if _, err := dec.Token(); err != io.EOF {
		return nil, errors.New("You can run only a single <Exec> command.")
	}

	if cmdMsg.XMLName.Local != CmdExec {
		return nil, errors.New("You can run only <Exec> command type.")
	}
	if len(cmdMsg.Items) != 1 {
		return nil, errors.New("You can run only a single <Exec> command.")
	}
	return &cmdMsg, nil
}

// WindowsMDMRequiresPremiumCmdMessage is the error message displayed by fleetctl mdm
// run-command when a Premium license is required. It must be kept in sync with
// the SyncMLCmd.IsPremium implementation below so that it reflects the proper
// command names.
const WindowsMDMRequiresPremiumCmdMessage = "Missing or invalid license. Wipe command is available in Fleet Premium only."

// IsPremium returns true if the command is available for Fleet premium only.
// See e.g. https://learn.microsoft.com/en-us/windows/client-management/mdm/remotewipe-csp#dowipe
// for details.
func (cmd SyncMLCmd) IsPremium() bool {
	// NOTE: if this implementation changes, make sure to also update the error
	// message above - the WindowsMDMRequiresPremiumCmdMessage constant.
	return strings.Contains(cmd.GetTargetURI(), "/Device/Vendor/MSFT/RemoteWipe/")
}

// DataType returns the SyncMLDataType corresponding to the command's format.
// This is the inverse of the NewTypedSyncMLCmd operation (regarding the
// format part of the command).
func (cmd SyncMLCmd) DataType() SyncMLDataType {
	if cmd.IsEmpty() || cmd.Items[0].Meta == nil ||
		cmd.Items[0].Meta.Format == nil || cmd.Items[0].Meta.Format.Content == nil {
		return SFNoFormat
	}

	switch strings.TrimSpace(*cmd.Items[0].Meta.Format.Content) {
	case "chr":
		return SFText
	case "xml":
		return SFXml
	case "b64":
		return SFBase64
	case "int":
		return SFInteger
	case "bool":
		return SFBoolean
	default:
		return SFNoFormat
	}
}

type CmdItem struct {
	Source *string     `xml:"Source>LocURI,omitempty"`
	Target *string     `xml:"Target>LocURI,omitempty"`
	Meta   *Meta       `xml:"Meta,omitempty"`
	Data   *RawXmlData `xml:"Data"`
}

type RawXmlData struct {
	Attrs   []xml.Attr `xml:",any,attr"`
	Content string     `xml:",innerxml"`
}

type Meta struct {
	Type   *MetaAttr `xml:"Type,omitempty"`
	Format *MetaAttr `xml:"Format,omitempty"`
}

type MetaAttr struct {
	XMLNS   string  `xml:"xmlns,attr"`
	Content *string `xml:",chardata"`
}

type LocURI struct {
	LocURI *string `xml:",omitempty"`
}

func (msg *SyncML) IsValidHeader() error {
	if strings.TrimSpace(msg.Xmlns) == "" {
		return errors.New("msg namespace")
	}

	// SyncML DTD version check
	if msg.SyncHdr.VerDTD != syncml.SyncMLSupportedVersion {
		return errors.New("unsupported DTD version")
	}

	// SyncML Proto version check
	if msg.SyncHdr.VerProto != syncml.SyncMLVerProto {
		return errors.New("unsupported proto version")
	}

	// SyncML SessionID check
	if msg.SyncHdr.SessionID == "" {
		return errors.New("sessionID")
	}

	// SyncML MsgID check
	if msg.SyncHdr.MsgID == "" {
		return errors.New("MsgID")
	}

	// Target LocURI check
	if msg.SyncHdr.Target == nil || msg.SyncHdr.Target.LocURI == nil || strings.TrimSpace(*msg.SyncHdr.Target.LocURI) == "" {
		return errors.New("Target.LocURI")
	}

	// Device ID check
	if msg.SyncHdr.Source == nil || msg.SyncHdr.Source.LocURI == nil || strings.TrimSpace(*msg.SyncHdr.Source.LocURI) == "" {
		return errors.New("Source.LocURI")
	}

	return nil
}

func (msg *SyncML) IsValidBody() error {
	nonNilCount := 0

	if len(msg.SyncBody.Add) > 0 {
		nonNilCount++
	}

	if len(msg.SyncBody.Alert) > 0 {
		nonNilCount++
	}

	if len(msg.SyncBody.Atomic) > 0 {
		nonNilCount++
	}

	if len(msg.SyncBody.Delete) > 0 {
		nonNilCount++
	}

	if len(msg.SyncBody.Exec) > 0 {
		nonNilCount++
	}

	if len(msg.SyncBody.Get) > 0 {
		nonNilCount++
	}

	if len(msg.SyncBody.Replace) > 0 {
		nonNilCount++
	}

	if len(msg.SyncBody.Status) > 0 {
		nonNilCount++
	}

	if len(msg.SyncBody.Results) > 0 {
		nonNilCount++
	}

	if len(msg.SyncBody.Raw) > 0 {
		nonNilCount++
	}

	if nonNilCount == 0 {
		return errors.New("no SyncML protocol commands")
	}

	return nil
}

// IsValidMsg checks for required fields in the SyncML message
func (msg *SyncML) IsValidMsg() error {
	if err := msg.IsValidHeader(); err != nil {
		return fmt.Errorf("invalid SyncML header: %s", err)
	}

	if err := msg.IsValidBody(); err != nil {
		return fmt.Errorf("invalid SyncML body: %s", err)
	}

	return nil
}

func (msg *SyncML) IsFinal() bool {
	if (msg.SyncBody.Final != nil) && strings.TrimSpace(*msg.SyncBody.Final) != "" {
		return true
	}

	return false
}

func (msg *SyncML) GetMessageID() (string, error) {
	if strings.TrimSpace(msg.SyncHdr.MsgID) != "" {
		return msg.SyncHdr.MsgID, nil
	}

	return "", errors.New("message id is empty")
}

func (msg *SyncML) GetSessionID() (string, error) {
	if strings.TrimSpace(msg.SyncHdr.SessionID) != "" {
		return msg.SyncHdr.SessionID, nil
	}

	return "", errors.New("session id is empty")
}

func (msg *SyncML) GetSource() (string, error) {
	if (msg.SyncHdr.Source != nil) &&
		(msg.SyncHdr.Source.LocURI != nil) &&
		strings.TrimSpace(*msg.SyncHdr.Source.LocURI) != "" {

		return *msg.SyncHdr.Source.LocURI, nil
	}

	return "", errors.New("message source is empty")
}

func (msg *SyncML) GetTarget() (string, error) {
	if (msg.SyncHdr.Target != nil) &&
		(msg.SyncHdr.Target.LocURI != nil) &&
		strings.TrimSpace(*msg.SyncHdr.Target.LocURI) != "" {

		return *msg.SyncHdr.Target.LocURI, nil
	}

	return "", errors.New("message target is empty")
}

// GetOrderedCmds returns the commands in the order they are defined in the message.
func (msg *SyncML) GetOrderedCmds() []ProtoCmdOperation {
	var cmds []ProtoCmdOperation

	// Helper function to add commands to the cmds slice
	addCmds := func(cmdsList ProtoCmds, verb string) {
		for _, cmd := range cmdsList {
			cmds = append(cmds, ProtoCmdOperation{Verb: verb, Cmd: cmd})
		}
	}

	// Process each command one by one
	if msg.SyncBody.Add != nil {
		addCmds(msg.SyncBody.Add, CmdAdd)
	}
	if msg.SyncBody.Alert != nil {
		addCmds(msg.SyncBody.Alert, CmdAlert)
	}
	if msg.SyncBody.Atomic != nil {
		addCmds(msg.SyncBody.Atomic, CmdAtomic)
	}
	if msg.SyncBody.Delete != nil {
		addCmds(msg.SyncBody.Delete, CmdDelete)
	}
	if msg.SyncBody.Exec != nil {
		addCmds(msg.SyncBody.Exec, CmdExec)
	}
	if msg.SyncBody.Get != nil {
		addCmds(msg.SyncBody.Get, CmdGet)
	}
	if msg.SyncBody.Replace != nil {
		addCmds(msg.SyncBody.Replace, CmdReplace)
	}
	if msg.SyncBody.Results != nil {
		addCmds(msg.SyncBody.Results, CmdResults)
	}
	if msg.SyncBody.Status != nil {
		addCmds(msg.SyncBody.Status, CmdStatus)
	}

	return cmds
}

type MDMCommandType int

const (
	MDMRaw MDMCommandType = iota
	MDMAdd
	MDMAlert
	MDMAtomic
	MDMDelete
	MDMExec
	MDMGet
	MDMReplace
	MDMResults
	MDMStatus
)

type SyncMLDataType uint16

const (
	SFEmpty SyncMLDataType = iota
	SFNoFormat
	SFText
	SFXml
	SFInteger
	SFBoolean
	SFBase64
)

func (msg *SyncML) AppendCommand(cmdType MDMCommandType, cmd SyncMLCmd) {
	switch cmdType {
	case MDMRaw:
		if msg.SyncBody.Raw == nil {
			msg.SyncBody.Raw = []SyncMLCmd{}
		}
		msg.SyncBody.Raw = append(msg.SyncBody.Raw, cmd)
	case MDMAdd:
		if msg.SyncBody.Add == nil {
			msg.SyncBody.Add = []SyncMLCmd{}
		}
		msg.SyncBody.Add = append(msg.SyncBody.Add, cmd)
	case MDMAlert:
		if msg.SyncBody.Alert == nil {
			msg.SyncBody.Alert = []SyncMLCmd{}
		}
		msg.SyncBody.Alert = append(msg.SyncBody.Alert, cmd)
	case MDMAtomic:
		if msg.SyncBody.Atomic == nil {
			msg.SyncBody.Atomic = []SyncMLCmd{}
		}
		msg.SyncBody.Atomic = append(msg.SyncBody.Atomic, cmd)
	case MDMDelete:
		if msg.SyncBody.Delete == nil {
			msg.SyncBody.Delete = []SyncMLCmd{}
		}
		msg.SyncBody.Delete = append(msg.SyncBody.Delete, cmd)
	case MDMExec:
		if msg.SyncBody.Exec == nil {
			msg.SyncBody.Exec = []SyncMLCmd{}
		}
		msg.SyncBody.Exec = append(msg.SyncBody.Exec, cmd)
	case MDMGet:
		if msg.SyncBody.Get == nil {
			msg.SyncBody.Get = []SyncMLCmd{}
		}
		msg.SyncBody.Get = append(msg.SyncBody.Get, cmd)
	case MDMReplace:
		if msg.SyncBody.Replace == nil {
			msg.SyncBody.Replace = []SyncMLCmd{}
		}
		msg.SyncBody.Replace = append(msg.SyncBody.Replace, cmd)
	case MDMResults:
		if msg.SyncBody.Results == nil {
			msg.SyncBody.Results = []SyncMLCmd{}
		}
		msg.SyncBody.Results = append(msg.SyncBody.Results, cmd)
	case MDMStatus:
		if msg.SyncBody.Status == nil {
			msg.SyncBody.Status = []SyncMLCmd{}
		}
		msg.SyncBody.Status = append(msg.SyncBody.Status, cmd)
	}
}

// SetID sets the MsgID field in the SyncML header
func (msg *SyncML) SetID(cmdID int) {
	msg.SyncHdr.MsgID = strconv.Itoa(cmdID)
}

// IsValid checks for required fields in the SyncML command
func (cmd *SyncMLCmd) IsValid() bool {
	if len(cmd.Items) == 0 && cmd.Data == nil {
		return false
	}

	return true
}

// IsEmpty checks if there are not items in the command
func (cmd *SyncMLCmd) IsEmpty() bool {
	return len(cmd.Items) == 0
}

// GetTargetURI returns the first protocol commands target URI from the items list
// This is OK as the protocol commands only have one item when sent from the server
func (cmd *SyncMLCmd) GetTargetURI() string {
	if cmd.IsEmpty() {
		return ""
	}

	if cmd.Items[0].Target != nil {
		return *cmd.Items[0].Target
	}

	return ""
}

// GetTargetData returns the first protocol commands target data from the items list
// This is OK as the protocol commands only have one item when sent from the server
func (cmd *SyncMLCmd) GetTargetData() string {
	if cmd.IsEmpty() {
		return ""
	}

	if cmd.Items[0].Data != nil {
		return cmd.Items[0].Data.Content
	}

	return ""
}

func (cmd *SyncMLCmd) ShouldBeTracked(cmdVerb string) bool {
	if (cmdVerb == "") || cmd.CmdRef == nil || *cmd.CmdRef == "0" {
		return false
	}

	return true
}

// /////////////////////////////////////////////////////////////
// MDMWindowsCommand type
// Represents a command in the windows_mdm_commands table
type MDMWindowsCommand struct {
	CommandUUID  string    `db:"command_uuid"`
	RawCommand   []byte    `db:"raw_command"`
	TargetLocURI string    `db:"target_loc_uri"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

// GetEncodedBinarySecurityToken returns the base64 form of a input payload
func GetEncodedBinarySecurityToken(typeID WindowsMDMEnrollmentType, payload string) (string, error) {
	var pld WindowsMDMAccessTokenPayload
	pld.Type = typeID

	switch typeID {
	case WindowsMDMProgrammaticEnrollmentType:
		pld.Payload.OrbitNodeKey = payload
	case WindowsMDMAutomaticEnrollmentType:
		pld.Payload.AuthToken = payload
	default:
		return "", fmt.Errorf("invalid enrollment type: %v", typeID)
	}

	rawBytes, err := json.Marshal(pld)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(rawBytes), nil
}

// HostMDMWindowsProfile represents the status of an MDM profile for a Windows host.
type HostMDMWindowsProfile struct {
	HostUUID      string             `db:"host_uuid" json:"host_uuid"`
	CommandUUID   string             `db:"command_uuid" json:"command_uuid"`
	ProfileUUID   string             `db:"profile_uuid" json:"profile_uuid"`
	Name          string             `db:"name" json:"name"`
	Status        *MDMDeliveryStatus `db:"status" json:"status"`
	OperationType MDMOperationType   `db:"operation_type" json:"operation_type"`
	Detail        string             `db:"detail" json:"detail"`
}

func (p HostMDMWindowsProfile) ToHostMDMProfile() HostMDMProfile {
	return HostMDMProfile{
		HostUUID:      p.HostUUID,
		ProfileUUID:   p.ProfileUUID,
		Name:          p.Name,
		Identifier:    "",
		Status:        p.Status,
		OperationType: p.OperationType,
		Detail:        p.Detail,
		Platform:      "windows",
	}
}

// BuildMDMWindowsProfilePayloadFromMDMResponse builds a
// MDMWindowsProfilePayload for a command that was used to deliver a
// configuration profile.
//
// Profiles are groups of `<Replace>` or `<Add>` commands wrapped in an `<Atomic>`, both
// the top-level atomic and each replace have different CmdID values and Status
// responses. For example a profile might look like:
//
// <Atomic>
//
//	<CmdID>foo</CmdID>
//	<Replace>
//	  <CmdID>bar</CmdID>
//	  ...
//	</Replace>
//	<Replace>
//	  <CmdID>baz</CmdID>
//	  ...
//	</Replace>
//
// </Atomic>
//
// And the response from the MDM server will be something like:
//
// <SyncBody>
// <Status>
//
//	<CmdID>foo</CmdID>
//	<Cmd>Atomic</Cmd>
//	<Data>200</Data>
//	...
//
// </Status>
// <Status>
//
//	<CmdID>bar</CmdID>
//	<Cmd>Replace</Cmd>
//	<Data>200</Data>
//	...
//
// </Status>
// <Status>
//
//	<CmdID>baz</CmdID>
//	<Cmd>Replace</Cmd>
//	<Data>200</Data>
//	...
//
// </Status>
// ...
// </SyncBody>
//
// As currently specified:
//   - The status of the resulting command should be the status of the
//     top-level `<Atomic>` operation
//   - The detail of the resulting command should be an aggregate of all the
//     status responses of every nested `Replace` operation
func BuildMDMWindowsProfilePayloadFromMDMResponse(
	// IMPORTANT: The cmdWithSecret.RawCommand may contain a Fleet secret variable value, so it should never be exposed or saved.
	cmdWithSecret MDMWindowsCommand,
	statuses map[string]SyncMLCmd,
	hostUUID string,
) (*MDMWindowsProfilePayload, error) {
	status, ok := statuses[cmdWithSecret.CommandUUID]
	if !ok {
		return nil, fmt.Errorf("missing status for root command %s", cmdWithSecret.CommandUUID)
	}
	commandStatus := WindowsResponseToDeliveryStatus(*status.Data)
	var details []string
	if status.Data != nil && commandStatus == MDMDeliveryFailed {
		syncML := new(SyncMLCmd)
		if err := xml.Unmarshal(cmdWithSecret.RawCommand, syncML); err != nil {
			return nil, err
		}
		for _, nested := range syncML.ReplaceCommands {
			if status, ok := statuses[nested.CmdID.Value]; ok && status.Data != nil {
				details = append(details, fmt.Sprintf("%s: status %s", nested.GetTargetURI(), *status.Data))
			}
		}

		for _, nested := range syncML.AddCommands {
			if status, ok := statuses[nested.CmdID.Value]; ok && status.Data != nil {
				details = append(details, fmt.Sprintf("%s: status %s", nested.GetTargetURI(), *status.Data))
			}
		}
	}
	detail := strings.Join(details, ", ")
	return &MDMWindowsProfilePayload{
		HostUUID:      hostUUID,
		Status:        &commandStatus,
		OperationType: "",
		Detail:        detail,
		CommandUUID:   cmdWithSecret.CommandUUID,
	}, nil
}

// WindowsResponseToDeliveryStatus converts a response string from Windows MDM
// into an MDMDeliveryStatus.
//
// If the response starts with "2" (any 2xx response), it returns
// MDMDeliveryVerifying, otherwise, it returns MDMDeliveryFailed.
func WindowsResponseToDeliveryStatus(resp string) MDMDeliveryStatus {
	if len(resp) == 0 {
		return MDMDeliveryPending
	}

	if strings.HasPrefix(resp, "2") {
		return MDMDeliveryVerifying
	}

	return MDMDeliveryFailed
}

package service

import (
	"encoding/xml"
	"errors"
	"fmt"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	mdm_types "github.com/fleetdm/fleet/v4/server/fleet"
	mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/stretchr/testify/require"
)

// NewSoapRequest takes a SOAP request in the form of a byte slice and tries to unmarshal it into a SoapRequest struct.
func NewSoapRequest(request []byte) (fleet.SoapRequest, error) {
	// Sanity check on input
	if len(request) == 0 {
		return fleet.SoapRequest{}, errors.New("soap request is invalid")
	}

	// Unmarshal the XML data from the request into the SoapRequest struct
	var req fleet.SoapRequest
	err := xml.Unmarshal(request, &req)
	if err != nil {
		return req, fmt.Errorf("there was a problem unmarshalling soap request: %v", err)
	}

	// If there was no error, return the SoapRequest and a nil error
	return req, nil
}

func TestValidSoapResponse(t *testing.T) {
	relatesTo := "urn:uuid:0d5a1441-5891-453b-becf-a2e5f6ea3749"
	soapFaultMsg := NewSoapFault(mdm.SoapErrorAuthentication, mdm_types.MDEDiscovery, errors.New("test"))
	sres, err := NewSoapResponse(&soapFaultMsg, relatesTo)
	require.NoError(t, err)
	outXML, err := xml.MarshalIndent(sres, "", "  ")
	require.NoError(t, err)
	require.NotEmpty(t, outXML)
	require.Contains(t, string(outXML), fmt.Sprintf("<a:RelatesTo>%s</a:RelatesTo>", relatesTo))
}

func TestInvalidSoapResponse(t *testing.T) {
	relatesTo := "urn:uuid:0d5a1441-5891-453b-becf-a2e5f6ea3749"
	_, err := NewSoapResponse(relatesTo, relatesTo)
	require.Error(t, err)
}

func TestFaultMessageSoapResponse(t *testing.T) {
	targetErrorString := "invalid input request"
	soapFaultMsg := NewSoapFault(mdm.SoapErrorAuthentication, mdm_types.MDEDiscovery, errors.New(targetErrorString))
	sres, err := NewSoapResponse(&soapFaultMsg, "urn:uuid:0d5a1441-5891-453b-becf-a2e5f6ea3749")
	require.NoError(t, err)
	outXML, err := xml.MarshalIndent(sres, "", "  ")
	require.NoError(t, err)
	require.NotEmpty(t, outXML)
	require.Contains(t, string(outXML), fmt.Sprintf("<s:text xml:lang=\"en-us\">%s</s:text>", targetErrorString))
}

// func NewRequestSecurityTokenResponseCollection(provisionedToken string) (mdm_types.RequestSecurityTokenResponseCollection, error) {
func TestRequestSecurityTokenResponseCollectionSoapResponse(t *testing.T) {
	provisionedToken := "provisionedToken"
	reqSecTokenCollectionMsg, err := NewRequestSecurityTokenResponseCollection(provisionedToken)
	require.NoError(t, err)
	sres, err := NewSoapResponse(&reqSecTokenCollectionMsg, "urn:uuid:0d5a1441-5891-453b-becf-a2e5f6ea3749")
	require.NoError(t, err)
	outXML, err := xml.MarshalIndent(sres, "", "  ")
	require.NoError(t, err)
	require.NotEmpty(t, outXML)
	require.Contains(t, string(outXML), fmt.Sprintf("base64binary\">%s</BinarySecurityToken>", provisionedToken))
}

func TestGetPoliciesResponseSoapResponse(t *testing.T) {
	minKey := "2048"
	getPoliciesMsg, err := NewGetPoliciesResponse(minKey, "10", "20")
	require.NoError(t, err)
	sres, err := NewSoapResponse(&getPoliciesMsg, "urn:uuid:0d5a1441-5891-453b-becf-a2e5f6ea3749")
	require.NoError(t, err)
	outXML, err := xml.MarshalIndent(sres, "", "  ")
	require.NoError(t, err)
	require.NotEmpty(t, outXML)
	require.Contains(t, string(outXML), fmt.Sprintf("<minimalKeyLength>%s</minimalKeyLength>", minKey))
}

func TestValidSoapRequestWithDiscoverMsg(t *testing.T) {
	requestBytes := []byte(`
	<s:Envelope xmlns:a="http://www.w3.org/2005/08/addressing" xmlns:s="http://www.w3.org/2003/05/soap-envelope">
	<s:Header>
		<a:Action s:mustUnderstand="1">http://schemas.microsoft.com/windows/management/2012/01/enrollment/IDiscoveryService/Discover</a:Action>
		<a:MessageID>urn:uuid:748132ec-a575-4329-b01b-6171a9cf8478</a:MessageID>
		<a:ReplyTo>
		<a:Address>http://www.w3.org/2005/08/addressing/anonymous</a:Address>
		</a:ReplyTo>
		<a:To s:mustUnderstand="1">https://mdmwindows.com:443/EnrollmentServer/Discovery.svc</a:To>
	</s:Header>
	<s:Body>
		<Discover xmlns="http://schemas.microsoft.com/windows/management/2012/01/enrollment">
		<request xmlns:i="http://www.w3.org/2001/XMLSchema-instance">
			<EmailAddress>demo@mdmwindows.com</EmailAddress>
			<RequestVersion>5.0</RequestVersion>
			<DeviceType>CIMClient_Windows</DeviceType>
			<ApplicationVersion>6.2.9200.2965</ApplicationVersion>
			<OSEdition>48</OSEdition>
			<AuthPolicies>
			<AuthPolicy>OnPremise</AuthPolicy>
			<AuthPolicy>Federated</AuthPolicy>
			</AuthPolicies>
		</request>
		</Discover>
	</s:Body>
	</s:Envelope>
		  `)

	req, err := NewSoapRequest(requestBytes)
	require.NoError(t, err)
	err = req.IsValidDiscoveryMsg()
	require.NoError(t, err)
}

func TestInvalidSoapRequestWithDiscoverMsg(t *testing.T) {
	requestBytes := []byte(`
	<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope" xmlns:a="http://www.w3.org/2005/08/addressing" xmlns:u="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd" xmlns:wsse="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd" xmlns:wst="http://docs.oasis-open.org/ws-sx/ws-trust/200512" xmlns:ac="http://schemas.xmlsoap.org/ws/2006/12/authorization">
	<s:Header>
		<a:Action s:mustUnderstand="1">http://schemas.microsoft.com/windows/pki/2009/01/enrollment/RST/wstep</a:Action>
		<a:MessageID>urn:uuid:0d5a1441-5891-453b-becf-a2e5f6ea3749</a:MessageID>
		<a:ReplyTo>
		<a:Address>http://www.w3.org/2005/08/addressing/anonymous</a:Address>
		</a:ReplyTo>
		<a:To s:mustUnderstand="1">https://mdmwindows.com/EnrollmentServer/Enrollment.svc</a:To>
		<wsse:Security s:mustUnderstand="1">
		<wsse:BinarySecurityToken ValueType="http://schemas.microsoft.com/5.0.0.0/ConfigurationManager/Enrollment/DeviceEnrollmentUserToken" EncodingType="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd#base64binary">aGVsbG93b3JsZA==</wsse:BinarySecurityToken>
		</wsse:Security>
	</s:Header>
	<s:Body>
		<wst:RequestSecurityToken>
		<wst:TokenType>http://schemas.microsoft.com/5.0.0.0/ConfigurationManager/Enrollment/DeviceEnrollmentToken</wst:TokenType>
		<wst:RequestType>http://docs.oasis-open.org/ws-sx/ws-trust/200512/Issue</wst:RequestType>
		<wsse:BinarySecurityToken ValueType="http://schemas.microsoft.com/windows/pki/2009/01/enrollment#PKCS10" EncodingType="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd#base64binary">MIICzjCCAboCAQAwSzFJMEcGA1UEAxNAMkI5QjUyQUMtREYzOC00MTYxLTgxNDItRjRCMUUwIURCMjU3QzNBMDg3NzhGNEZCNjFFMjc0OTA2NkMxRjI3ADCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAKogsEpbKL8fuXpTNAE5RTZim8JO5CCpxj3z+SuWabs/s9Zse6RziKr12R4BXPiYE1zb8god4kXxet8x3ilGqAOoXKkdFTdNkdVa23PEMrIZSX5MuQ7mwGtctayARxmDvsWRF/icxJbqSO+bYIKvuifesOCHW2cJ1K+JSKijTMik1N8NFbLi5fg1J+xImT9dW1z2fLhQ7SNEMLosUPHsbU9WKoDBfnPsLHzmhM2IMw+5dICZRoxHZalh70FefBk0XoT8b6w4TIvc8572TyPvvdwhc5o/dvyR3nAwTmJpjBs1YhJfSdP+EBN1IC2T/i/mLNUuzUSC2OwiHPbZ6MMr/hUCAwEAAaBCMEAGCSqGSIb3DQEJDjEzMDEwLwYKKwYBBAGCN0IBAAQhREIyNTdDM0EwODc3OEY0RkI2MUUyNzQ5MDY2QzFGMjcAMAkGBSsOAwIdBQADggEBACQtxyy74sCQjZglwdh/Ggs6ofMvnWLMq9A9rGZyxAni66XqDUoOg5PzRtSt+Gv5vdLQyjsBYVzo42W2HCXLD2sErXWwh/w0k4H7vcRKgEqv6VYzpZ/YRVaewLYPcqo4g9NoXnbW345OPLwT3wFvVR5v7HnD8LB2wHcnMu0fAQORgafCRWJL1lgw8VZRaGw9BwQXCF/OrBNJP1ivgqtRdbSoH9TD4zivlFFa+8VDz76y2mpfo0NbbD+P0mh4r0FOJan3X9bLswOLFD6oTiyXHgcVSzLN0bQ6aQo0qKp3yFZYc8W4SgGdEl07IqNquKqJ/1fvmWxnXEbl3jXwb1efhbM=</wsse:BinarySecurityToken>
		<ac:AdditionalContext xmlns="http://schemas.xmlsoap.org/ws/2006/12/authorization">
			<ac:ContextItem Name="UXInitiated">
			<ac:Value>false</ac:Value>
			</ac:ContextItem>
			<ac:ContextItem Name="HWDevID">
			<ac:Value>BF2D12A95AE42E47D58465E9A71336CAF33FCCAD3088F140F4D50B371FB2256F</ac:Value>
			</ac:ContextItem>
			<ac:ContextItem Name="Locale">
			<ac:Value>en-US</ac:Value>
			</ac:ContextItem>
			<ac:ContextItem Name="TargetedUserLoggedIn">
			<ac:Value>true</ac:Value>
			</ac:ContextItem>
			<ac:ContextItem Name="OSEdition">
			<ac:Value>48</ac:Value>
			</ac:ContextItem>
			<ac:ContextItem Name="DeviceName">
			<ac:Value>DESKTOP-0C89RC0</ac:Value>
			</ac:ContextItem>
			<ac:ContextItem Name="MAC">
			<ac:Value>00-0C-29-7B-4E-4C</ac:Value>
			</ac:ContextItem>
			<ac:ContextItem Name="MAC">
			<ac:Value>00-0C-29-7B-4E-56</ac:Value>
			</ac:ContextItem>
			<ac:ContextItem Name="DeviceID">
			<ac:Value>DB257C3A08778F4FB61E2749066C1F27</ac:Value>
			</ac:ContextItem>
			<ac:ContextItem Name="EnrollmentType">
			<ac:Value>Full</ac:Value>
			</ac:ContextItem>
			<ac:ContextItem Name="DeviceType">
			<ac:Value>CIMClient_Windows</ac:Value>
			</ac:ContextItem>
			<ac:ContextItem Name="OSVersion">
			<ac:Value>10.0.19045.2965</ac:Value>
			</ac:ContextItem>
			<ac:ContextItem Name="ApplicationVersion">
			<ac:Value>10.0.19045.2965</ac:Value>
			</ac:ContextItem>
			<ac:ContextItem Name="NotInOobe">
			<ac:Value>false</ac:Value>
			</ac:ContextItem>
			<ac:ContextItem Name="RequestVersion">
			<ac:Value>5.0</ac:Value>
			</ac:ContextItem>
		</ac:AdditionalContext>
		</wst:RequestSecurityToken>
	</s:Body>
	</s:Envelope>
		  `)

	req, err := NewSoapRequest(requestBytes)
	require.NoError(t, err)
	err = req.IsValidDiscoveryMsg()
	require.Error(t, err)
}

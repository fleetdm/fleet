package service

import (
	"context"
	"crypto/md5" //nolint:gosec // Windows MDM Auth uses MD5
	"crypto/x509"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/fleetdm/fleet/v4/server/mdm/microsoft/syncml"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/platform/logging/testutils"
	"github.com/stretchr/testify/assert"
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

func TestIsValidAppruURL(t *testing.T) {
	tests := []struct {
		name     string
		appru    string
		expected bool
	}{
		// Valid URLs
		{
			name:     "valid ms-app scheme",
			appru:    "ms-app://windows.immersivecontrolpanel",
			expected: true,
		},
		{
			name:     "valid https scheme",
			appru:    "https://example.com/callback",
			expected: true,
		},
		{
			name:     "valid http scheme",
			appru:    "http://localhost/callback",
			expected: true,
		},
		// Invalid URLs - XSS attempts
		{
			name:     "javascript injection",
			appru:    ";for (var key in localStorage){ alert(key)};//",
			expected: false,
		},
		{
			name:     "javascript protocol",
			appru:    "javascript:alert(1)",
			expected: false,
		},
		{
			name:     "data URI",
			appru:    "data:text/html,<script>alert(1)</script>",
			expected: false,
		},
		{
			name:     "empty scheme",
			appru:    "://example.com",
			expected: false,
		},
		{
			name:     "plain text",
			appru:    "not-a-url",
			expected: false,
		},
		{
			name:     "empty string",
			appru:    "",
			expected: false,
		},
		{
			name:     "file scheme",
			appru:    "file:///etc/passwd",
			expected: false,
		},
		{
			name:     "ftp scheme",
			appru:    "ftp://example.com",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isValidAppru(tc.appru)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestValidSoapResponse(t *testing.T) {
	relatesTo := "urn:uuid:0d5a1441-5891-453b-becf-a2e5f6ea3749"
	soapFaultMsg := NewSoapFault(syncml.SoapErrorAuthentication, fleet.MDEDiscovery, errors.New("test"))
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
	soapFaultMsg := NewSoapFault(syncml.SoapErrorAuthentication, fleet.MDEDiscovery, errors.New(targetErrorString))
	sres, err := NewSoapResponse(&soapFaultMsg, "urn:uuid:0d5a1441-5891-453b-becf-a2e5f6ea3749")
	require.NoError(t, err)
	outXML, err := xml.MarshalIndent(sres, "", "  ")
	require.NoError(t, err)
	require.NotEmpty(t, outXML)
	require.Contains(t, string(outXML), fmt.Sprintf("<s:text xml:lang=\"en-us\">%s</s:text>", targetErrorString))
}

// func NewRequestSecurityTokenResponseCollection(provisionedToken string) (fleet.RequestSecurityTokenResponseCollection, error) {
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

func TestProvisioningDocGeneration(t *testing.T) {
	deviceIdentityFingerprint := "031336C933CC7E228B88880D78824FB2909A0A2F"
	serverIdentityFingerprint := "F9A4F20FC50D990FDD0E3DB9AFCBF401818D5462"

	// Preparing the WAP Provisioning Doc response
	certStoreData := NewCertStoreProvisioningData(
		"full",
		deviceIdentityFingerprint,
		[]byte{0x1, 0x2, 0x3},
		serverIdentityFingerprint,
		[]byte{0x4, 0x5, 0x6})

	// Preparing the WAP Provisioning Doc response
	appConfigData := NewApplicationProvisioningData(microsoft_mdm.MDE2EnrollPath, "testuser", "testpassword")
	appDMClientData := NewDMClientProvisioningData()
	provDoc := NewProvisioningDoc(certStoreData, appConfigData, appDMClientData)

	outXML, err := xml.MarshalIndent(provDoc, "", "  ")
	require.NoError(t, err)
	require.NotEmpty(t, outXML)
	require.Contains(t, string(outXML), deviceIdentityFingerprint)
	require.Contains(t, string(outXML), serverIdentityFingerprint)
	require.Contains(t, string(outXML), microsoft_mdm.MDE2EnrollPath)
	require.Contains(t, string(outXML), "testuser")
	require.Contains(t, string(outXML), "testpassword")
}

func TestValidSyncMLCmdStatus(t *testing.T) {
	testMsgRef := "testmsgref"
	testCmdRef := "testcmdref"
	testCmdOrig := "testcmdorig"
	testStatusCode := "teststatuscode"
	cmdMsg := NewSyncMLCmdStatus(testMsgRef, testCmdRef, testCmdOrig, testStatusCode)
	outXML, err := xml.MarshalIndent(cmdMsg, "", "  ")
	require.NoError(t, err)
	require.NotEmpty(t, outXML)
	payload := string(outXML)
	err = checkWrappedSyncMLCmd(fleet.CmdStatus, payload)
	require.NoError(t, err)
	require.Contains(t, payload, fmt.Sprintf("<MsgRef>%s</MsgRef>", testMsgRef))
	require.Contains(t, payload, fmt.Sprintf("<CmdRef>%s</CmdRef>", testCmdRef))
	require.Contains(t, payload, fmt.Sprintf("<Cmd>%s</Cmd>", testCmdOrig))
	require.Contains(t, payload, fmt.Sprintf("<Data>%s</Data>", testStatusCode))
}

func TestValidNewSyncMLCmdGet(t *testing.T) {
	testOmaURI := "testuri"
	cmdMsg := newSyncMLNoFormat(fleet.CmdGet, testOmaURI)
	outXML, err := xml.MarshalIndent(cmdMsg, "", "  ")
	require.NoError(t, err)
	require.NotEmpty(t, outXML)
	payload := string(outXML)
	err = checkWrappedSyncMLCmd(fleet.CmdGet, payload)
	require.NoError(t, err)
	require.Contains(t, payload, fmt.Sprintf("<LocURI>%s</LocURI>", testOmaURI))
}

func TestValidNewSyncMLCmdBool(t *testing.T) {
	testOmaURI := "testuri"
	testData := "testdata"
	cmdMsg := newSyncMLCmdBool(fleet.CmdReplace, testOmaURI, testData)
	outXML, err := xml.MarshalIndent(cmdMsg, "", "  ")
	require.NoError(t, err)
	require.NotEmpty(t, outXML)
	payload := string(outXML)
	err = checkWrappedSyncMLCmd(fleet.CmdReplace, payload)
	require.NoError(t, err)
	require.Contains(t, payload, fmt.Sprintf("<LocURI>%s</LocURI>", testOmaURI))
	require.Contains(t, payload, fmt.Sprintf("<Data>%s</Data>", testData))
	require.Contains(t, payload, "<Type xmlns=\"syncml:metinf\">text/plain</Type>")
	require.Contains(t, payload, "<Format xmlns=\"syncml:metinf\">bool</Format>")
}

func TestValidNewSyncMLCmdInt(t *testing.T) {
	testOmaURI := "testuri"
	testData := "testdata"
	cmdMsg := newSyncMLCmdInt(fleet.CmdReplace, testOmaURI, testData)
	outXML, err := xml.MarshalIndent(cmdMsg, "", "  ")
	require.NoError(t, err)
	require.NotEmpty(t, outXML)
	payload := string(outXML)
	err = checkWrappedSyncMLCmd(fleet.CmdReplace, payload)
	require.NoError(t, err)
	require.Contains(t, payload, fmt.Sprintf("<LocURI>%s</LocURI>", testOmaURI))
	require.Contains(t, payload, fmt.Sprintf("<Data>%s</Data>", testData))
	require.Contains(t, payload, "<Type xmlns=\"syncml:metinf\">text/plain</Type>")
	require.Contains(t, payload, "<Format xmlns=\"syncml:metinf\">int</Format>")
}

func TestValidSyncMLCmdText(t *testing.T) {
	testOmaURI := "testuri"
	testData := "testdata"
	cmdMsg := newSyncMLCmdText(fleet.CmdReplace, testOmaURI, testData)
	outXML, err := xml.MarshalIndent(cmdMsg, "", "  ")
	require.NoError(t, err)
	require.NotEmpty(t, outXML)
	payload := string(outXML)
	err = checkWrappedSyncMLCmd(fleet.CmdReplace, payload)
	require.NoError(t, err)
	require.Contains(t, payload, fmt.Sprintf("<LocURI>%s</LocURI>", testOmaURI))
	require.Contains(t, payload, fmt.Sprintf("<Data>%s</Data>", testData))
	require.Contains(t, payload, "<Type xmlns=\"syncml:metinf\">text/plain</Type>")
	require.Contains(t, payload, "<Format xmlns=\"syncml:metinf\">chr</Format>")
}

func TestValidSyncMLCmdXml(t *testing.T) {
	testOmaURI := "testuri"
	testData := "testdata"
	cmdMsg := newSyncMLCmdXml(fleet.CmdReplace, testOmaURI, testData)
	outXML, err := xml.MarshalIndent(cmdMsg, "", "  ")
	require.NoError(t, err)
	require.NotEmpty(t, outXML)
	payload := string(outXML)
	err = checkWrappedSyncMLCmd(fleet.CmdReplace, payload)
	require.NoError(t, err)
	require.Contains(t, payload, fmt.Sprintf("<LocURI>%s</LocURI>", testOmaURI))
	require.Contains(t, payload, fmt.Sprintf("<Data>%s</Data>", testData))
	require.Contains(t, payload, "<Type xmlns=\"syncml:metinf\">text/plain</Type>")
	require.Contains(t, payload, "<Format xmlns=\"syncml:metinf\">xml</Format>")
}

func TestValidSyncMLCmdAlert(t *testing.T) {
	testData := "1234"
	cmdMsg := newSyncMLNoItem(fleet.CmdAlert, testData)
	outXML, err := xml.MarshalIndent(cmdMsg, "", "  ")
	require.NoError(t, err)
	require.NotEmpty(t, outXML)
	payload := string(outXML)
	err = checkWrappedSyncMLCmd(fleet.CmdAlert, payload)
	require.NoError(t, err)
	require.Contains(t, payload, fmt.Sprintf("<Data>%s</Data>", testData))
}

func TestValidSyncMLCmd(t *testing.T) {
	testCmdSource := "testcmdsource"
	testCmdTarget := "testcmdtarget"
	testCmdDataType := "testcmddatatype"
	testCmdDataFormat := "testchr"
	testCmdDataValue := "testdata"
	cmdMsg := NewSyncMLCmd(fleet.CmdReplace, testCmdSource, testCmdTarget, testCmdDataType, testCmdDataFormat, testCmdDataValue)
	outXML, err := xml.MarshalIndent(cmdMsg, "", "  ")
	require.NoError(t, err)
	require.NotEmpty(t, outXML)
	payload := string(outXML)
	err = checkWrappedSyncMLCmd(fleet.CmdReplace, payload)
	require.NoError(t, err)
	require.Contains(t, payload, fmt.Sprintf("<LocURI>%s</LocURI>", testCmdSource))
	require.Contains(t, payload, fmt.Sprintf("<LocURI>%s</LocURI>", testCmdTarget))
	require.Contains(t, payload, fmt.Sprintf("<Data>%s</Data>", testCmdDataValue))
	require.Contains(t, payload, fmt.Sprintf("<Type xmlns=\"syncml:metinf\">%s</Type>", testCmdDataType))
	require.Contains(t, payload, fmt.Sprintf("<Format xmlns=\"syncml:metinf\">%s</Format>", testCmdDataFormat))
}

// checkWrappedSyncMLCmd checks that the payload is wrapped in the given tag.
func checkWrappedSyncMLCmd(tag string, data string) error {
	trimmedData := strings.TrimSpace(data)
	openTag := fmt.Sprintf("<%s>", tag)
	closeTag := fmt.Sprintf("</%s>", tag)
	if !strings.HasPrefix(trimmedData, openTag) || !strings.HasSuffix(trimmedData, closeTag) {
		return fmt.Errorf("payload is not wrapped in %s%s", openTag, closeTag)
	}
	return nil
}

func TestBuildCommandFromProfileBytes(t *testing.T) {
	t.Run("fail unmarshalling xml", func(t *testing.T) {
		cmd, err := buildCommandFromProfileBytes([]byte("<Replace></Add>"), "")
		require.Nil(t, cmd)
		require.ErrorContains(t, err, "unmarshalling profile")
	})

	t.Run("non atomic profile", func(t *testing.T) {
		// build and generate a command
		cmd, err := buildCommandFromProfileBytes(syncMLForTest("foo/bar"), "uuid-1")
		require.Nil(t, err)
		require.Equal(t, "uuid-1", cmd.CommandUUID)
		require.Empty(t, cmd.TargetLocURI)

		cmds, err := fleet.UnmarshallMultiTopLevelXMLProfile(cmd.RawCommand)
		require.NoError(t, err)

		replaceCommandsSeen := 0
		firstReplaceCmdID := ""
		for _, cmdXML := range cmds {
			if cmdXML.XMLName.Local == fleet.CmdReplace {
				replaceCommandsSeen++
				require.NotEmpty(t, cmdXML.CmdID.Value)
				firstReplaceCmdID = cmdXML.CmdID.Value // This works because we only expect one
			}
		}
		require.EqualValues(t, 1, replaceCommandsSeen)
		// generated xml contains additional comments about CmdID
		require.Equal(
			t,
			fmt.Sprintf(`<Add><!-- CmdID generated by Fleet --><CmdID>uuid-1</CmdID><Item><Target><LocURI>foo/bar</LocURI></Target></Item></Add><Replace><!-- CmdID generated by Fleet --><CmdID>%s</CmdID><Item><Target><LocURI>foo/bar</LocURI></Target></Item></Replace>`, firstReplaceCmdID),
			string(cmd.RawCommand),
		)

		// build and generate a second command with the same syncml
		cmd, err = buildCommandFromProfileBytes(syncMLForTestWithExec("foo/bar"), "uuid-2")
		require.Nil(t, err)
		require.Equal(t, "uuid-2", cmd.CommandUUID)
		require.Empty(t, cmd.TargetLocURI)
		cmds, err = fleet.UnmarshallMultiTopLevelXMLProfile(cmd.RawCommand)
		require.NoError(t, err)

		replaceCommandsSeen = 0
		secondReplaceCmdID := ""
		secondExecCmdID := ""
		for _, cmdXML := range cmds {
			if cmdXML.XMLName.Local == fleet.CmdReplace {
				replaceCommandsSeen++
				require.NotEmpty(t, cmdXML.CmdID.Value)
				secondReplaceCmdID = cmdXML.CmdID.Value // This works because we only expect one
			} else if cmdXML.XMLName.Local == fleet.CmdExec {
				require.NotEmpty(t, cmdXML.CmdID.Value)
				secondExecCmdID = cmdXML.CmdID.Value
			}
		}
		require.EqualValues(t, 1, replaceCommandsSeen)
		require.NotEqualValues(t, "", secondReplaceCmdID)
		// generated xml contains additional comments about CmdID
		require.Equal(
			t,
			fmt.Sprintf(`<Add><!-- CmdID generated by Fleet --><CmdID>uuid-2</CmdID><Item><Target><LocURI>foo/bar</LocURI></Target></Item></Add><Replace><!-- CmdID generated by Fleet --><CmdID>%s</CmdID><Item><Target><LocURI>foo/bar</LocURI></Target></Item></Replace><Exec><!-- CmdID generated by Fleet --><CmdID>%s</CmdID><Item><Target><LocURI>foo/bar</LocURI></Target></Item></Exec>`, secondReplaceCmdID, secondExecCmdID),
			string(cmd.RawCommand),
		)

		// uuids of replaces are different
		require.NotEqual(t, firstReplaceCmdID, secondReplaceCmdID)
	})

	t.Run("atomic profile", func(t *testing.T) {
		// build and generate a command
		cmd, err := buildCommandFromProfileBytes(atomicSyncMLForTest("foo/bar"), "uuid-1")
		require.Nil(t, err)
		require.Equal(t, "uuid-1", cmd.CommandUUID)
		require.Empty(t, cmd.TargetLocURI)

		syncOne := new(fleet.SyncMLCmd)
		err = xml.Unmarshal(cmd.RawCommand, syncOne)
		require.NoError(t, err)
		require.Len(t, syncOne.ReplaceCommands, 1)
		require.NotEmpty(t, syncOne.ReplaceCommands[0].CmdID.Value)
		// generated xml contains additional comments about CmdID
		require.Equal(
			t,
			fmt.Sprintf(`<Atomic><!-- CmdID generated by Fleet --><CmdID>uuid-1</CmdID><Replace><!-- CmdID generated by Fleet --><CmdID>%s</CmdID><Item><Target><LocURI>foo/bar</LocURI></Target></Item></Replace><Add><!-- CmdID generated by Fleet --><CmdID>%s</CmdID><Item><Target><LocURI>foo/bar</LocURI></Target></Item></Add></Atomic>`, syncOne.ReplaceCommands[0].CmdID.Value, syncOne.AddCommands[0].CmdID.Value),
			string(cmd.RawCommand),
		)

		// build and generate a second command with the same syncml
		cmd, err = buildCommandFromProfileBytes(atomicSyncMLForTestWithExec("foo/bar"), "uuid-2")
		require.Nil(t, err)
		require.Equal(t, "uuid-2", cmd.CommandUUID)
		require.Empty(t, cmd.TargetLocURI)
		syncTwo := new(fleet.SyncMLCmd)
		err = xml.Unmarshal(cmd.RawCommand, syncTwo)
		require.NoError(t, err)
		require.Len(t, syncTwo.ReplaceCommands, 1)
		require.NotEmpty(t, syncTwo.ReplaceCommands[0].CmdID.Value)
		// generated xml contains additional comments about CmdID
		require.Equal(
			t,
			fmt.Sprintf(`<Atomic><!-- CmdID generated by Fleet --><CmdID>uuid-2</CmdID><Replace><!-- CmdID generated by Fleet --><CmdID>%s</CmdID><Item><Target><LocURI>foo/bar</LocURI></Target></Item></Replace><Add><!-- CmdID generated by Fleet --><CmdID>%s</CmdID><Item><Target><LocURI>foo/bar</LocURI></Target></Item></Add><Exec><!-- CmdID generated by Fleet --><CmdID>%s</CmdID><Item><Target><LocURI>foo/bar</LocURI></Target></Item></Exec></Atomic>`, syncTwo.ReplaceCommands[0].CmdID.Value, syncTwo.AddCommands[0].CmdID.Value, syncTwo.ExecCommands[0].CmdID.Value),
			string(cmd.RawCommand),
		)

		// uuids of replaces are different
		require.NotEqual(t, syncOne.ReplaceCommands[0].CmdID.Value, syncTwo.ReplaceCommands[0].CmdID.Value)
	})

	t.Run("SCEP profiles", func(t *testing.T) {
		// build and generate a command
		scepCmdWithAtomic, err := buildCommandFromProfileBytes(atomicSyncMLForTest("/Vendor/MSFT/ClientCertificateInstall/SCEP"), "uuid-1")
		require.Nil(t, err)
		require.Equal(t, "uuid-1", scepCmdWithAtomic.CommandUUID)
		require.Empty(t, scepCmdWithAtomic.TargetLocURI)
		syncTwo := new(fleet.SyncMLCmd)
		err = xml.Unmarshal(scepCmdWithAtomic.RawCommand, syncTwo)
		require.NoError(t, err)

		expectedString := "<Atomic><!-- CmdID generated by Fleet --><CmdID>uuid-1</CmdID><Replace><!-- CmdID generated by Fleet --><CmdID>%s</CmdID><Item><Target><LocURI>/Vendor/MSFT/ClientCertificateInstall/SCEP</LocURI></Target></Item></Replace><Add><!-- CmdID generated by Fleet --><CmdID>%s</CmdID><Item><Target><LocURI>/Vendor/MSFT/ClientCertificateInstall/SCEP</LocURI></Target></Item></Add></Atomic>"
		require.Equal(
			t,
			fmt.Sprintf(expectedString, syncTwo.ReplaceCommands[0].CmdID.Value, syncTwo.AddCommands[0].CmdID.Value),
			string(scepCmdWithAtomic.RawCommand),
		)

		scepCmdWithoutAtomic, err := buildCommandFromProfileBytes(syncMLForTest("/Vendor/MSFT/ClientCertificateInstall/SCEP"), "uuid-1")
		require.Nil(t, err)
		require.Equal(t, "uuid-1", scepCmdWithoutAtomic.CommandUUID)
		require.Empty(t, scepCmdWithoutAtomic.TargetLocURI)
		syncTwo = new(fleet.SyncMLCmd)
		err = xml.Unmarshal(scepCmdWithAtomic.RawCommand, syncTwo)
		require.NoError(t, err)

		require.Equal(
			t,
			fmt.Sprintf(expectedString, syncTwo.ReplaceCommands[0].CmdID.Value, syncTwo.AddCommands[0].CmdID.Value),
			string(scepCmdWithAtomic.RawCommand),
		)
	})
}

func syncMLForTest(locURI string) []byte {
	return []byte(fmt.Sprintf(`
<Add>
  <Item>
    <Target>
      <LocURI>%s</LocURI>
    </Target>
  </Item>
</Add>
<Replace>
  <Item>
    <Target>
      <LocURI>%s</LocURI>
    </Target>
  </Item>
</Replace>`, locURI, locURI))
}

func atomicSyncMLForTest(locURI string) []byte {
	data := syncMLForTest(locURI)
	return fmt.Appendf([]byte{}, `
<Atomic>%s</Atomic>`, data)
}

func syncMLForTestWithExec(locURI string) []byte {
	return []byte(fmt.Sprintf(`
<Add>
  <Item>
    <Target>
      <LocURI>%s</LocURI>
    </Target>
  </Item>
</Add>
<Replace>
  <Item>
    <Target>
      <LocURI>%s</LocURI>
    </Target>
  </Item>
</Replace>
<Exec>
  <Item>
    <Target>
	  <LocURI>%s</LocURI>
	</Target>
  </Item>
</Exec>`, locURI, locURI, locURI))
}

func atomicSyncMLForTestWithExec(locURI string) []byte {
	data := syncMLForTestWithExec(locURI)
	return fmt.Appendf([]byte{}, `
<Atomic>%s</Atomic>`, data)
}

// Setups a reconciler test run by mocking required datastore methods, for a single profile pending installation.
// Use $FLEET_VAR_HOST_UUID in the profile SyncML to simulate error in profile variable processing flow.
func setupReconcilerTest(ds *mock.Store, hostToProfile map[string]*fleet.MDMWindowsConfigProfile) (capturedUpdates *[]*fleet.MDMWindowsBulkUpsertHostProfilePayload, managedCerts *[]*fleet.MDMManagedCertificate) {
	// Cursor stubs: tests don't care about cursor state, just need the
	// reconciler not to panic on the calls.
	ds.GetMDMWindowsReconcileCursorFunc = func(ctx context.Context) (string, error) {
		return "", nil
	}
	ds.SetMDMWindowsReconcileCursorFunc = func(ctx context.Context, cursor string) error {
		return nil
	}

	// The cron's batched path picks a host window first, then calls the
	// scoped listings for that window. For the mock, return all host UUIDs
	// from hostToProfile so the rest of the reconciler runs against the
	// same set the test wants.
	ds.ListNextPendingMDMWindowsHostUUIDsFunc = func(ctx context.Context, afterHostUUID string, batchSize int) ([]string, error) {
		hostUUIDs := make([]string, 0, len(hostToProfile))
		for hostUUID := range hostToProfile {
			hostUUIDs = append(hostUUIDs, hostUUID)
		}
		return hostUUIDs, nil
	}

	listInstall := func(_ context.Context, _ ...any) ([]*fleet.MDMWindowsProfilePayload, error) {
		profilesToInstall := []*fleet.MDMWindowsProfilePayload{}
		for hostUUID, profile := range hostToProfile {
			profilesToInstall = append(profilesToInstall, &fleet.MDMWindowsProfilePayload{
				ProfileUUID:   profile.ProfileUUID,
				ProfileName:   profile.Name,
				HostUUID:      hostUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			})
		}
		return profilesToInstall, nil
	}
	// Mock both the legacy global listing (kept for tests still using it
	// directly) and the new scoped listing the cron now calls.
	ds.ListMDMWindowsProfilesToInstallFunc = func(ctx context.Context) ([]*fleet.MDMWindowsProfilePayload, error) {
		return listInstall(ctx)
	}
	ds.ListMDMWindowsProfilesToInstallForHostsFunc = func(ctx context.Context, hostUUIDs []string) ([]*fleet.MDMWindowsProfilePayload, error) {
		return listInstall(ctx)
	}

	ds.ListMDMWindowsProfilesToRemoveFunc = func(ctx context.Context) ([]*fleet.MDMWindowsProfilePayload, error) {
		return nil, nil
	}
	ds.ListMDMWindowsProfilesToRemoveForHostsFunc = func(ctx context.Context, hostUUIDs []string) ([]*fleet.MDMWindowsProfilePayload, error) {
		return nil, nil
	}

	ds.GetMDMWindowsProfilesContentsFunc = func(ctx context.Context, profileUUIDs []string) (map[string]fleet.MDMWindowsProfileContents, error) {
		profileContentsMap := make(map[string]fleet.MDMWindowsProfileContents)
		for _, profile := range hostToProfile {
			profileContentsMap[profile.ProfileUUID] = fleet.MDMWindowsProfileContents{
				SyncML:   profile.SyncML,
				Checksum: []byte("test-checksum"),
			}
		}
		return profileContentsMap, nil
	}

	// Default: every requested profile still exists. Tests that want to
	// exercise the deletion-race guard can override this with their own Func.
	ds.GetExistingMDMWindowsProfileUUIDsFunc = func(ctx context.Context, profileUUIDs []string) (map[string]struct{}, error) {
		out := make(map[string]struct{}, len(profileUUIDs))
		for _, u := range profileUUIDs {
			out[u] = struct{}{}
		}
		return out, nil
	}

	capturedUpdates = &[]*fleet.MDMWindowsBulkUpsertHostProfilePayload{}
	ds.BulkUpsertMDMWindowsHostProfilesFunc = func(ctx context.Context, payload []*fleet.MDMWindowsBulkUpsertHostProfilePayload) error {
		*capturedUpdates = append(*capturedUpdates, payload...)
		return nil
	}

	ds.GetMDMWindowsBitLockerSummaryFunc = func(ctx context.Context, teamID *uint) (*fleet.MDMWindowsBitLockerSummary, error) {
		return &fleet.MDMWindowsBitLockerSummary{}, nil
	}

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			MDM: fleet.MDM{
				WindowsEnabledAndConfigured: true,
			},
		}, nil
	}

	ds.BulkDeleteMDMWindowsHostsConfigProfilesFunc = func(ctx context.Context, payload []*fleet.MDMWindowsProfilePayload) error {
		return nil
	}

	ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
		return &fleet.GroupedCertificateAuthorities{
			CustomScepProxy: []fleet.CustomSCEPProxyCA{},
		}, nil
	}

	managedCerts = &[]*fleet.MDMManagedCertificate{}
	ds.BulkUpsertMDMManagedCertificatesFunc = func(ctx context.Context, payload []*fleet.MDMManagedCertificate) error {
		*managedCerts = payload
		return nil
	}

	return capturedUpdates, managedCerts
}

func TestReconcileWindowsProfilesWithFleetVariableError(t *testing.T) {
	ctx := context.Background()
	ds := new(mock.Store)
	logger := slog.New(slog.DiscardHandler)

	// Setup test data with a profile containing Fleet variable
	testHostUUID := "test-host-uuid"
	// Profile with Fleet variable that would cause preprocessing to succeed normally
	testProfile := &fleet.MDMWindowsConfigProfile{
		ProfileUUID: "test-profile-uuid",
		Name:        "Test Profile with Variable",
		SyncML:      []byte(`<Replace><Item><Target><LocURI>./Test</LocURI></Target><Data>Host: $FLEET_VAR_HOST_UUID</Data></Item></Replace>`),
	}

	hostToProfile := map[string]*fleet.MDMWindowsConfigProfile{
		testHostUUID: testProfile,
	}
	capturedUpdates, managedCerts := setupReconcilerTest(ds, hostToProfile)

	var receivedCommand *fleet.MDMWindowsCommand
	ds.MDMWindowsInsertCommandAndUpsertHostProfilesForHostsFunc = func(ctx context.Context, hostUUIDs []string, cmd *fleet.MDMWindowsCommand, updates []*fleet.MDMWindowsBulkUpsertHostProfilePayload) error {
		receivedCommand = cmd
		// Simulate error only for commands with substituted UUID (to test error handling)
		if strings.Contains(string(cmd.RawCommand), testHostUUID) {
			return errors.New("command insert failed after preprocessing")
		}
		*capturedUpdates = append(*capturedUpdates, updates...)
		return nil
	}

	// Run ReconcileWindowsProfiles
	err := ReconcileWindowsProfiles(ctx, ds, logger)
	require.NoError(t, err) // The function should not return an error even if insert fails

	// Verify the command was preprocessed (UUID should be substituted)
	require.NotNil(t, receivedCommand, "Command should have been created")
	require.Contains(t, string(receivedCommand.RawCommand), testHostUUID, "UUID should have been substituted in the command")
	require.NotContains(t, string(receivedCommand.RawCommand), "$FLEET_VAR_HOST_UUID", "Fleet variable should have been replaced")

	// Verify managed certs is empty as no certs were in the profile
	require.Empty(t, managedCerts, "No managed certificates should have been added")

	// Verify that the error was captured and the profile was marked as failed
	require.True(t, ds.MDMWindowsInsertCommandAndUpsertHostProfilesForHostsFuncInvoked, "MDMWindowsInsertCommandAndUpsertHostProfilesForHosts should have been called")
	require.True(t, ds.BulkUpsertMDMWindowsHostProfilesFuncInvoked, "BulkUpsertMDMWindowsHostProfiles should have been called")

	// Find the error status update
	var foundError bool
	for _, update := range *capturedUpdates {
		if update.Status != nil && *update.Status == fleet.MDMDeliveryFailed {
			foundError = true
			require.Contains(t, update.Detail, "command insert failed after preprocessing", "Error detail should contain the original error message")
			break
		}
	}
	require.True(t, foundError, "Should have found a failed status update")
}

func TestReconcileWindowsProfileWithCertificateFailureDoesNotAddManagedCertificate(t *testing.T) {
	ctx := t.Context()
	ctx = license.NewContext(ctx, &fleet.LicenseInfo{
		Tier: fleet.TierPremium,
	})
	ds := new(mock.Store)
	logger := slog.New(slog.DiscardHandler)

	// Setup test data with a profile containing a certificate that will fail processing
	testHostUUID := "test-host-uuid"
	testProfile := &fleet.MDMWindowsConfigProfile{
		ProfileUUID: "test-profile-uuid",
		Name:        "Test Profile with Cert",
		SyncML:      []byte(`<Replace><Item><Target><LocURI>./Certificate</LocURI></Target><Data>$FLEET_VAR_CUSTOM_SCEP_PROXY_URL_CA</Data></Item></Replace>`),
	}

	hostToProfile := map[string]*fleet.MDMWindowsConfigProfile{
		testHostUUID: testProfile,
	}
	capturedUpdates, managedCerts := setupReconcilerTest(ds, hostToProfile)

	// Override GetGroupedCertificateAuthorities to return a valid CA
	ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
		return &fleet.GroupedCertificateAuthorities{
			CustomScepProxy: []fleet.CustomSCEPProxyCA{
				{
					ID:        1,
					Name:      "CA",
					URL:       "https://scep.proxy.url",
					Challenge: "secret",
				},
			},
		}, nil
	}

	ds.NewChallengeFunc = func(ctx context.Context) (string, error) {
		return "secret", nil
	}

	ds.MDMWindowsInsertCommandAndUpsertHostProfilesForHostsFunc = func(ctx context.Context, hostUUIDs []string, cmd *fleet.MDMWindowsCommand, updates []*fleet.MDMWindowsBulkUpsertHostProfilePayload) error {
		return errors.New("fake error to check managed certificate")
	}

	// Run ReconcileWindowsProfiles
	err := ReconcileWindowsProfiles(ctx, ds, logger)
	require.NoError(t, err) // The function should not return an error even if cert processing fails

	// Verify no managed certificates were added due to failure
	require.Empty(t, managedCerts, "No managed certificates should have been added")

	// Verify that the error was captured and the profile was marked as failed
	require.True(t, ds.BulkUpsertMDMWindowsHostProfilesFuncInvoked, "BulkUpsertMDMWindowsHostProfiles should have been called")

	// Find the error status update
	var foundError bool
	for _, update := range *capturedUpdates {
		if update.Status != nil && *update.Status == fleet.MDMDeliveryFailed {
			foundError = true
			require.Contains(t, update.Detail, "fake error to check managed certificate", "Error detail should indicate certificate processing failure")
			break
		}
	}
	require.True(t, foundError, "Should have found a failed status update")
}

func TestReconcileWindowsProfilesWithOneHostFailingStillAddsManagedCertificate(t *testing.T) {
	ctx := t.Context()
	ctx = license.NewContext(ctx, &fleet.LicenseInfo{
		Tier: fleet.TierPremium,
	})
	ds := new(mock.Store)
	logger := slog.New(slog.DiscardHandler)

	// Setup test data with a profile containing a certificate that will fail processing
	testHostUUID := "test-host-uuid"
	testProfile := &fleet.MDMWindowsConfigProfile{
		ProfileUUID: "test-profile-uuid",
		Name:        "Test Profile with Cert",
		SyncML:      []byte(`<Replace><Item><Target><LocURI>./Certificate</LocURI></Target><Data>$FLEET_VAR_CUSTOM_SCEP_PROXY_URL_CA</Data></Item></Replace><Replace><Item><Target><LocURI>./Certificate</LocURI></Target><Data>$FLEET_VAR_HOST_END_USER_IDP_USERNAME</Data></Item></Replace>`),
	}
	testHostUUID2 := "test-host-uuid-2"
	hostToProfile := map[string]*fleet.MDMWindowsConfigProfile{
		testHostUUID:  testProfile,
		testHostUUID2: testProfile,
	}

	capturedUpdates, managedCerts := setupReconcilerTest(ds, hostToProfile)

	// Override GetGroupedCertificateAuthorities to return a valid CA
	ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
		return &fleet.GroupedCertificateAuthorities{
			CustomScepProxy: []fleet.CustomSCEPProxyCA{
				{
					ID:        1,
					Name:      "CA",
					URL:       "https://scep.proxy.url",
					Challenge: "secret",
				},
			},
		}, nil
	}

	ds.NewChallengeFunc = func(ctx context.Context) (string, error) {
		return "secret", nil
	}

	ds.MDMWindowsInsertCommandAndUpsertHostProfilesForHostsFunc = func(ctx context.Context, hostUUIDs []string, cmd *fleet.MDMWindowsCommand, updates []*fleet.MDMWindowsBulkUpsertHostProfilePayload) error {
		*capturedUpdates = append(*capturedUpdates, updates...)
		return nil
	}

	ds.HostIDsByIdentifierFunc = func(ctx context.Context, filter fleet.TeamFilter, hostnames []string) ([]uint, error) {
		if hostnames[0] == testHostUUID {
			return []uint{1}, nil
		}
		return []uint{2}, nil
	}

	ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
		if hostID == 1 {
			return &fleet.ScimUser{
				UserName: "test@example.com",
			}, nil
		}
		return nil, nil
	}
	ds.ListHostDeviceMappingFunc = func(ctx context.Context, id uint) ([]*fleet.HostDeviceMapping, error) {
		return []*fleet.HostDeviceMapping{}, nil
	}

	// Run ReconcileWindowsProfiles
	err := ReconcileWindowsProfiles(ctx, ds, logger)
	require.NoError(t, err) // The function should not return an error even if cert processing fails

	// Verify one managed certificates were added, for the successful host, but not for the failing one
	require.NotNil(t, managedCerts, "Managed certificates slice should not be nil")
	require.Len(t, *managedCerts, 1, "No managed certificates should have been added")

	// Verify that the error was captured and the profile was marked as failed
	require.True(t, ds.BulkUpsertMDMWindowsHostProfilesFuncInvoked, "BulkUpsertMDMWindowsHostProfiles should have been called")

	// Check the error and only one error
	foundErrors := 0
	for _, update := range *capturedUpdates {
		if update.Status != nil && *update.Status == fleet.MDMDeliveryFailed {
			foundErrors++
			require.Contains(t, update.Detail, "There is no IdP username for this host.", "Error detail should indicate missing IdP username")
			break
		}
	}
	require.EqualValues(t, 1, foundErrors, "Should have found one failed status update")
}

// TestReconcileWindowsProfilesSkipsDeletedProfile covers the race where an
// admin deletes a Windows profile between the cron's initial list and the
// per-profile upsert. Without the guard, the cron would insert a
// host_mdm_windows_profiles row + enqueue an install command for a profile
// that no longer exists in mdm_windows_configuration_profiles; later the
// remove path can't build a <Delete> command (SyncML is gone) and the row
// is stuck. The fix: GetExistingMDMWindowsProfileUUIDs pre-filter right
// before the upsert loop.
func TestReconcileWindowsProfilesSkipsDeletedProfile(t *testing.T) {
	ctx := context.Background()
	ds := new(mock.Store)
	logger := slog.New(slog.DiscardHandler)

	deletedProfile := &fleet.MDMWindowsConfigProfile{
		ProfileUUID: "deleted-profile-uuid",
		Name:        "Deleted Before Upsert",
		SyncML:      []byte(`<Replace><Item><Target><LocURI>./Test</LocURI></Target><Data>v</Data></Item></Replace>`),
	}
	hostToProfile := map[string]*fleet.MDMWindowsConfigProfile{
		"host-a": deletedProfile,
	}
	setupReconcilerTest(ds, hostToProfile)

	// Simulate the race: ListMDMWindowsProfilesToInstall and
	// GetMDMWindowsProfilesContents already ran (both set up by
	// setupReconcilerTest to include the profile). Between those and the
	// upsert, the admin deleted the profile, so
	// GetExistingMDMWindowsProfileUUIDs returns an empty set.
	ds.GetExistingMDMWindowsProfileUUIDsFunc = func(ctx context.Context, profileUUIDs []string) (map[string]struct{}, error) {
		return map[string]struct{}{}, nil
	}

	err := ReconcileWindowsProfiles(ctx, ds, logger)
	require.NoError(t, err)
	require.True(t, ds.GetExistingMDMWindowsProfileUUIDsFuncInvoked, "existence pre-check must run")
	require.False(t, ds.MDMWindowsInsertCommandAndUpsertHostProfilesForHostsFuncInvoked,
		"no zombie row should be written when the profile is gone")
}

// TestReconcileWindowsProfilesSkipsInsertLag covers the asymmetric race
// where a profile was just inserted on the primary but the replica
// hasn't caught up: GetMDMWindowsProfilesContents (replica) misses the
// row even though GetExistingMDMWindowsProfileUUIDs (primary) sees it.
// Without the skip-and-continue, the cron would error out with
// "missing profile content", leave the cursor unchanged, and re-fire the
// same race every 30s until the replica converges. The fix: log + skip
// + advance the cursor; the hosts stay in the listing universe and the
// next tick picks them up after replication catches up.
func TestReconcileWindowsProfilesSkipsInsertLag(t *testing.T) {
	ctx := context.Background()
	ds := new(mock.Store)
	logger := slog.New(slog.DiscardHandler)

	freshProfile := &fleet.MDMWindowsConfigProfile{
		ProfileUUID: "fresh-profile-uuid",
		Name:        "Just-Inserted-On-Primary",
		SyncML:      []byte(`<Replace><Item><Target><LocURI>./Test</LocURI></Target><Data>v</Data></Item></Replace>`),
	}
	hostToProfile := map[string]*fleet.MDMWindowsConfigProfile{
		"host-a": freshProfile,
	}
	setupReconcilerTest(ds, hostToProfile)

	// Simulate insert-lag: the existence pre-check (primary) finds the
	// profile, but the content fetch (replica) returns nothing because
	// replication hasn't caught up to the just-committed insert.
	ds.GetMDMWindowsProfilesContentsFunc = func(ctx context.Context, profileUUIDs []string) (map[string]fleet.MDMWindowsProfileContents, error) {
		return map[string]fleet.MDMWindowsProfileContents{}, nil
	}

	err := ReconcileWindowsProfiles(ctx, ds, logger)
	require.NoError(t, err, "insert-lag must not fail the tick; the cursor must advance so the next tick can retry")
	require.True(t, ds.GetExistingMDMWindowsProfileUUIDsFuncInvoked,
		"existence pre-check still runs (it confirms the profile exists on primary)")
	require.False(t, ds.MDMWindowsInsertCommandAndUpsertHostProfilesForHostsFuncInvoked,
		"no install command should be enqueued when content is not yet visible")
}

// TestReconcileWindowsProfilesEmptyPopulation covers the cron's two
// terminating branches when there is no pending Windows MDM work.
// A fresh ("") cursor stays empty and writes nothing. A non-empty cursor
// (left over from a prior partial pass) is reset to "" exactly once. In
// both cases the cron returns nil and the per-host / per-profile mocks
// are never reached, so leaving them nil on the mock store is itself an
// implicit assertion.
func TestReconcileWindowsProfilesEmptyPopulation(t *testing.T) {
	cases := []struct {
		name            string
		initialCursor   string
		wantSetCalls    int
		wantFinalCursor string
	}{
		{
			name:            "fresh cursor and no work is a no-op",
			initialCursor:   "",
			wantSetCalls:    0,
			wantFinalCursor: "",
		},
		{
			name:            "non-empty cursor is reset to empty once",
			initialCursor:   "left-over-host-uuid",
			wantSetCalls:    1,
			wantFinalCursor: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			ds := new(mock.Store)
			logger := slog.New(slog.DiscardHandler)
			cursor := tc.initialCursor
			var setCalls int

			ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
				cfg := &fleet.AppConfig{}
				cfg.MDM.WindowsEnabledAndConfigured = true
				return cfg, nil
			}
			ds.GetMDMWindowsReconcileCursorFunc = func(ctx context.Context) (string, error) {
				return cursor, nil
			}
			ds.SetMDMWindowsReconcileCursorFunc = func(ctx context.Context, c string) error {
				cursor = c
				setCalls++
				return nil
			}
			ds.ListNextPendingMDMWindowsHostUUIDsFunc = func(ctx context.Context, after string, batchSize int) ([]string, error) {
				return nil, nil
			}

			require.NoError(t, ReconcileWindowsProfiles(ctx, ds, logger))
			require.Equal(t, tc.wantSetCalls, setCalls)
			require.Equal(t, tc.wantFinalCursor, cursor)
		})
	}
}

func TestRekeyWindowsDevice(t *testing.T) {
	ds := new(mock.Store)
	kv := new(mock.KVStore)
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{
		KeyValueStore: kv,
	})

	var credsHash *[]byte
	const testEnrollmentID uint = 123
	ds.MDMWindowsGetEnrolledDeviceWithDeviceIDFunc = func(ctx context.Context, mdmDeviceID string) (*fleet.MDMWindowsEnrolledDevice, error) {
		return &fleet.MDMWindowsEnrolledDevice{
			ID:              testEnrollmentID,
			MDMDeviceID:     "device",
			HostUUID:        "host-uuid-123",
			CredentialsHash: credsHash,
		}, nil
	}

	ds.MDMWindowsUpdateEnrolledDeviceCredentialsFunc = func(ctx context.Context, deviceId string, credentialsHash []byte) error {
		require.Equal(t, "device", deviceId)
		credsHash = &credentialsHash
		return nil
	}

	ackCalled := 0
	ds.MDMWindowsAcknowledgeEnrolledDeviceCredentialsFunc = func(ctx context.Context, deviceId string) error {
		require.Equal(t, "device", deviceId)
		ackCalled++
		return nil
	}

	kv.SetFunc = func(ctx context.Context, key string, value string, expireTime time.Duration) error {
		return nil
	}

	var nonce string
	kv.GetFunc = func(ctx context.Context, key string) (*string, error) {
		return &nonce, nil
	}

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			ServerSettings: fleet.ServerSettings{
				ServerURL: "fake-mdm-server.com",
			},
		}, nil
	}

	syncml := `<SyncML xmlns="SYNCML:SYNCML1.2">
  <SyncHdr>
    <VerDTD>1.2</VerDTD>
    <VerProto>DM/1.2</VerProto>
    <SessionID>1</SessionID>
    <MsgID>1</MsgID>
    <Target>
      <LocURI>fake-mdm-server.com</LocURI>
    </Target>
    <Source>
      <LocURI>device</LocURI>
    </Source>
  </SyncHdr>
  <SyncBody>
    <Alert>
      <CmdID>2</CmdID>
      <Data>1201</Data>
    </Alert>
    <Final />
  </SyncBody>
</SyncML>`

	var req *fleet.SyncML
	err := xml.Unmarshal([]byte(syncml), &req)
	require.NoError(t, err)

	res, err := svc.GetMDMWindowsManagementResponse(ctx, req, []*x509.Certificate{})
	require.NoError(t, err)
	require.NotNil(t, res)

	seenStatuses := 0
	seenReplaces := 0
	seenOther := 0
	var username string
	var password string
	for _, cmd := range res.SyncBody.Raw {
		switch cmd.XMLName.Local {
		case fleet.CmdStatus:
			require.Equal(t, "200", *cmd.Data)
			seenStatuses++
		case fleet.CmdReplace:
			containsAuthReplace := strings.Contains(cmd.GetTargetURI(), "AAuthName") || strings.Contains(cmd.GetTargetURI(), "AAuthSecret")
			require.True(t, containsAuthReplace, "Replace command should be for AAuthName or AAuthSecret")
			seenReplaces++

			if strings.Contains(cmd.GetTargetURI(), "AAuthName") {
				username = cmd.GetTargetData()
			} else if strings.Contains(cmd.GetTargetURI(), "AAuthSecret") {
				password = cmd.GetTargetData()
			}
		default:
			seenOther++
		}
	}

	assert.Equal(t, 1, seenStatuses, "should have one Status command")
	assert.Equal(t, 2, seenReplaces, "should have two Replace commands")
	assert.Equal(t, 0, seenOther, "should not have other commands")

	// Respond with no credentials again to get a nonce
	res, err = svc.GetMDMWindowsManagementResponse(ctx, req, []*x509.Certificate{})
	require.NoError(t, err)
	require.NotNil(t, res)

	// Require Chal in header
	require.Len(t, res.SyncBody.Raw, 1, "should short circuit with challenge")
	chalFound := false
	for _, cmd := range res.SyncBody.Raw {
		if cmd.Chal != nil {
			chalFound = true
			nonce = *cmd.Chal.Meta.NextNonce.Content
			break
		}
	}
	require.True(t, chalFound, "should have challenge command")

	// Now respond with credentials to ack the rekey
	// WE only need to mock this as we short-circuit when challenging or invalid creds
	ds.MDMWindowsGetPendingCommandsFunc = func(ctx context.Context, enrollmentID uint) ([]*fleet.MDMWindowsCommand, error) {
		require.Equal(t, testEnrollmentID, enrollmentID)
		return []*fleet.MDMWindowsCommand{}, nil
	}
	ds.GetWindowsMDMCommandsForResendingFunc = func(ctx context.Context, deviceID string, failedCommandIds []string) ([]*fleet.MDMWindowsCommand, error) {
		return []*fleet.MDMWindowsCommand{}, nil
	}

	deviceCredsHash := hashMDMCredentials(username, password, nonce)
	syncmlWithCreds := fmt.Sprintf(`<SyncML xmlns="SYNCML:SYNCML1.2">
  <SyncHdr>
    <VerDTD>1.2</VerDTD>
    <VerProto>DM/1.2</VerProto>
    <SessionID>1</SessionID>
    <MsgID>1</MsgID>
    <Target>
      <LocURI>fake-mdm-server.com</LocURI>
    </Target>
    <Source>
      <LocURI>device</LocURI>
    </Source>
	<Cred>
		<Meta>
        <Format xmlns="syncml:metinf">b64</Format>
        <Type xmlns="syncml:metinf">syncml:auth-md5</Type>
      </Meta>
      <Data>%s</Data>
	</Cred>
  </SyncHdr>
  <SyncBody>
    <Alert>
      <CmdID>2</CmdID>
      <Data>1201</Data>
    </Alert>
    <Final />
  </SyncBody>
</SyncML>`, base64.StdEncoding.EncodeToString(deviceCredsHash))
	err = xml.Unmarshal([]byte(syncmlWithCreds), &req)
	require.NoError(t, err)

	res, err = svc.GetMDMWindowsManagementResponse(ctx, req, []*x509.Certificate{})
	require.NoError(t, err)
	require.NotNil(t, res)

	require.Equal(t, 1, ackCalled, "acknowledge should have been called once")
}

func hashMDMCredentials(username, password, nonce string) []byte {
	credsHash := md5.Sum([]byte(username + ":" + password)) //nolint:gosec // Windows MDM Auth uses MD5
	encodedCreds := base64.StdEncoding.EncodeToString(credsHash[:])
	nonceHash := md5.Sum([]byte(encodedCreds + ":" + nonce)) //nolint:gosec // Windows MDM Auth uses MD5
	return nonceHash[:]
}

func TestGetESPCommands(t *testing.T) {
	t.Parallel()
	const deviceID = "test-device-id"
	const hostUUID = "test-host-uuid"

	newSvc := func(t *testing.T) (*mock.Store, *Service) {
		ds := new(mock.Store)
		// Default HostLite mock: every test reaching Stage 3 calls HostLiteByIdentifier (via the cached loadHost
		// closure) to translate device.HostUUID -> setup_experience_status_results.host_uuid (= OsqueryHostID on
		// Windows). Provide a sensible default so tests that don't care about the team/setup-experience identifier
		// still work; tests that do care override this.
		osqueryHostID := "osquery-" + hostUUID
		ds.HostLiteByIdentifierFunc = func(ctx context.Context, identifier string) (*fleet.HostLite, error) {
			return &fleet.HostLite{ID: 1, UUID: identifier, OsqueryHostID: &osqueryHostID, TeamID: nil}, nil
		}
		return ds, &Service{ds: ds, logger: testutils.TestLogger(t)}
	}

	t.Run("no awaiting configuration returns nil", func(t *testing.T) {
		ds, svc := newSvc(t)
		ds.MDMWindowsGetEnrolledDeviceWithDeviceIDFunc = func(ctx context.Context, mdmDeviceID string) (*fleet.MDMWindowsEnrolledDevice, error) {
			return &fleet.MDMWindowsEnrolledDevice{
				MDMDeviceID:           deviceID,
				AwaitingConfiguration: fleet.WindowsMDMAwaitingConfigurationNone,
			}, nil
		}

		cmds, err := svc.getESPCommands(t.Context(), deviceID)
		require.NoError(t, err)
		assert.Nil(t, cmds)
	})

	t.Run("pending without host UUID sends hold commands", func(t *testing.T) {
		ds, svc := newSvc(t)
		ds.MDMWindowsGetEnrolledDeviceWithDeviceIDFunc = func(ctx context.Context, mdmDeviceID string) (*fleet.MDMWindowsEnrolledDevice, error) {
			return &fleet.MDMWindowsEnrolledDevice{
				MDMDeviceID:           deviceID,
				HostUUID:              "",
				AwaitingConfiguration: fleet.WindowsMDMAwaitingConfigurationPending,
			}, nil
		}

		cmds, err := svc.getESPCommands(t.Context(), deviceID)
		require.NoError(t, err)
		require.NotEmpty(t, cmds, "should return hold commands")
	})

	t.Run("pending with host UUID transitions to active", func(t *testing.T) {
		ds, svc := newSvc(t)
		ds.MDMWindowsGetEnrolledDeviceWithDeviceIDFunc = func(ctx context.Context, mdmDeviceID string) (*fleet.MDMWindowsEnrolledDevice, error) {
			return &fleet.MDMWindowsEnrolledDevice{
				MDMDeviceID:           deviceID,
				HostUUID:              hostUUID,
				AwaitingConfiguration: fleet.WindowsMDMAwaitingConfigurationPending,
			}, nil
		}
		transitioned := false
		ds.SetMDMWindowsAwaitingConfigurationFunc = func(ctx context.Context, mdmDeviceID string, from, to fleet.WindowsMDMAwaitingConfiguration) (bool, error) {
			transitioned = true
			return true, nil
		}

		cmds, err := svc.getESPCommands(t.Context(), deviceID)
		require.NoError(t, err)
		require.NotEmpty(t, cmds, "should return DevicePreparation completed command")
		assert.True(t, transitioned)
	})

	t.Run("active with pending profiles waits", func(t *testing.T) {
		ds, svc := newSvc(t)
		ds.MDMWindowsGetEnrolledDeviceWithDeviceIDFunc = func(ctx context.Context, mdmDeviceID string) (*fleet.MDMWindowsEnrolledDevice, error) {
			return &fleet.MDMWindowsEnrolledDevice{
				MDMDeviceID:           deviceID,
				HostUUID:              hostUUID,
				AwaitingConfiguration: fleet.WindowsMDMAwaitingConfigurationActive,
			}, nil
		}
		ds.ListMDMWindowsProfilesToInstallForHostFunc = func(ctx context.Context, hUUID string) ([]*fleet.MDMWindowsProfilePayload, error) {
			return nil, nil // reconciler already ran
		}
		ds.GetHostMDMWindowsProfilesFunc = func(ctx context.Context, hUUID string) ([]fleet.HostMDMWindowsProfile, error) {
			return []fleet.HostMDMWindowsProfile{
				{ProfileUUID: "prof-1", Name: "WiFi", Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			}, nil
		}

		cmds, err := svc.getESPCommands(t.Context(), deviceID)
		require.NoError(t, err)
		assert.Nil(t, cmds, "should wait while profiles are pending")
	})

	t.Run("active with verifying profiles waits", func(t *testing.T) {
		ds, svc := newSvc(t)
		ds.MDMWindowsGetEnrolledDeviceWithDeviceIDFunc = func(ctx context.Context, mdmDeviceID string) (*fleet.MDMWindowsEnrolledDevice, error) {
			return &fleet.MDMWindowsEnrolledDevice{
				MDMDeviceID:           deviceID,
				HostUUID:              hostUUID,
				AwaitingConfiguration: fleet.WindowsMDMAwaitingConfigurationActive,
			}, nil
		}
		ds.ListMDMWindowsProfilesToInstallForHostFunc = func(ctx context.Context, hUUID string) ([]*fleet.MDMWindowsProfilePayload, error) {
			return nil, nil // reconciler already ran
		}
		ds.GetHostMDMWindowsProfilesFunc = func(ctx context.Context, hUUID string) ([]fleet.HostMDMWindowsProfile, error) {
			return []fleet.HostMDMWindowsProfile{
				{ProfileUUID: "prof-1", Name: "WiFi", Status: &fleet.MDMDeliveryVerifying, OperationType: fleet.MDMOperationTypeInstall},
			}, nil
		}

		cmds, err := svc.getESPCommands(t.Context(), deviceID)
		require.NoError(t, err)
		assert.Nil(t, cmds, "should wait while profiles are verifying")
	})

	t.Run("active waits when profiles not yet queued by reconciler", func(t *testing.T) {
		ds, svc := newSvc(t)
		ds.MDMWindowsGetEnrolledDeviceWithDeviceIDFunc = func(ctx context.Context, mdmDeviceID string) (*fleet.MDMWindowsEnrolledDevice, error) {
			return &fleet.MDMWindowsEnrolledDevice{
				MDMDeviceID:           deviceID,
				HostUUID:              hostUUID,
				AwaitingConfiguration: fleet.WindowsMDMAwaitingConfigurationActive,
			}, nil
		}
		ds.ListMDMWindowsProfilesToInstallForHostFunc = func(ctx context.Context, hUUID string) ([]*fleet.MDMWindowsProfilePayload, error) {
			return []*fleet.MDMWindowsProfilePayload{
				{ProfileUUID: "prof-1", ProfileName: "WiFi"},
			}, nil
		}

		cmds, err := svc.getESPCommands(t.Context(), deviceID)
		require.NoError(t, err)
		assert.Nil(t, cmds, "should wait when profiles are configured but not yet queued")
		assert.False(t, ds.GetHostMDMWindowsProfilesFuncInvoked, "should not check delivery status when profiles not yet queued")
	})

	// setRequireAll wires the host -> require_all_software_windows lookup chain (HostLiteByIdentifier ->
	// AppConfig) to return the given value via the no-team / app-config path. Tests don't need to
	// exercise the team lookup separately; the team path is structurally the same.
	setRequireAll := func(ds *mock.Store, requireAll bool) {
		ds.HostLiteByIdentifierFunc = func(ctx context.Context, identifier string) (*fleet.HostLite, error) {
			return &fleet.HostLite{ID: 1, UUID: identifier, TeamID: nil}, nil
		}
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			ac := &fleet.AppConfig{}
			ac.MDM.MacOSSetup.RequireAllSoftwareWindows = requireAll
			return ac, nil
		}
	}

	// setReleaseMocks sets up mocks needed for tests that reach the release path
	// (all profiles delivered + setup experience done).
	setReleaseMocks := func(ds *mock.Store) {
		ds.ListSetupExperienceResultsByHostUUIDFunc = func(ctx context.Context, hUUID string, teamID uint) ([]*fleet.SetupExperienceStatusResult, error) {
			return nil, nil
		}
		// Default: no setup experience items configured. Tests that
		// expect waiting due to items configured override this.
		ds.HasWindowsSetupExperienceItemsForTeamFunc = func(ctx context.Context, teamID uint) (bool, error) {
			return false, nil
		}
		ds.SetMDMWindowsAwaitingConfigurationFunc = func(ctx context.Context, mdmDeviceID string, from, to fleet.WindowsMDMAwaitingConfiguration) (bool, error) {
			return true, nil
		}
		setRequireAll(ds, false)
		ds.MDMWindowsInsertCommandsForHostFunc = func(ctx context.Context, hostUUIDOrDeviceID string, cmds []*fleet.MDMWindowsCommand) error {
			return nil
		}
	}

	t.Run("active with all profiles delivered releases device", func(t *testing.T) {
		ds, svc := newSvc(t)
		ds.MDMWindowsGetEnrolledDeviceWithDeviceIDFunc = func(ctx context.Context, mdmDeviceID string) (*fleet.MDMWindowsEnrolledDevice, error) {
			return &fleet.MDMWindowsEnrolledDevice{
				MDMDeviceID:           deviceID,
				HostUUID:              hostUUID,
				AwaitingConfiguration: fleet.WindowsMDMAwaitingConfigurationActive,
			}, nil
		}
		ds.ListMDMWindowsProfilesToInstallForHostFunc = func(ctx context.Context, hUUID string) ([]*fleet.MDMWindowsProfilePayload, error) {
			return nil, nil
		}
		ds.GetHostMDMWindowsProfilesFunc = func(ctx context.Context, hUUID string) ([]fleet.HostMDMWindowsProfile, error) {
			return []fleet.HostMDMWindowsProfile{
				{ProfileUUID: "prof-1", Name: "WiFi", Status: &fleet.MDMDeliveryVerified, OperationType: fleet.MDMOperationTypeInstall},
			}, nil
		}
		setReleaseMocks(ds)
		// Capture ordering: persist must run BEFORE the CAS so a persist failure can't leave the device finalized
		// without the dropped-response retry safety net.
		persisted := false
		ds.MDMWindowsInsertCommandsForHostFunc = func(ctx context.Context, hostUUIDOrDeviceID string, cmds []*fleet.MDMWindowsCommand) error {
			persisted = true
			return nil
		}
		ds.SetMDMWindowsAwaitingConfigurationFunc = func(ctx context.Context, mdmDeviceID string, from, to fleet.WindowsMDMAwaitingConfiguration) (bool, error) {
			require.True(t, persisted, "persist must run BEFORE CAS Active->None")
			return true, nil
		}

		cmds, err := svc.getESPCommands(t.Context(), deviceID)
		require.NoError(t, err)
		require.NotEmpty(t, cmds, "should return release commands")
		assert.True(t, ds.MDMWindowsInsertCommandsForHostFuncInvoked,
			"release path must persist final commands as the dropped-response retry backup")
		assert.True(t, ds.SetMDMWindowsAwaitingConfigurationFuncInvoked,
			"should transition awaiting_configuration out of Active")
	})

	t.Run("active waits when setup experience software is pending", func(t *testing.T) {
		ds, svc := newSvc(t)
		ds.MDMWindowsGetEnrolledDeviceWithDeviceIDFunc = func(ctx context.Context, mdmDeviceID string) (*fleet.MDMWindowsEnrolledDevice, error) {
			return &fleet.MDMWindowsEnrolledDevice{
				MDMDeviceID:           deviceID,
				HostUUID:              hostUUID,
				AwaitingConfiguration: fleet.WindowsMDMAwaitingConfigurationActive,
			}, nil
		}
		ds.ListMDMWindowsProfilesToInstallForHostFunc = func(ctx context.Context, hUUID string) ([]*fleet.MDMWindowsProfilePayload, error) {
			return nil, nil
		}
		ds.GetHostMDMWindowsProfilesFunc = func(ctx context.Context, hUUID string) ([]fleet.HostMDMWindowsProfile, error) {
			return nil, nil // no profiles
		}
		ds.ListSetupExperienceResultsByHostUUIDFunc = func(ctx context.Context, hUUID string, teamID uint) ([]*fleet.SetupExperienceStatusResult, error) {
			return []*fleet.SetupExperienceStatusResult{
				{Name: "Slack", Status: fleet.SetupExperienceStatusRunning},
			}, nil
		}

		cmds, err := svc.getESPCommands(t.Context(), deviceID)
		require.NoError(t, err)
		assert.Nil(t, cmds, "should wait while setup experience software is running")
	})

	t.Run("active treats cancelled setup experience items as terminal", func(t *testing.T) {
		ds, svc := newSvc(t)
		ds.MDMWindowsGetEnrolledDeviceWithDeviceIDFunc = func(ctx context.Context, mdmDeviceID string) (*fleet.MDMWindowsEnrolledDevice, error) {
			return &fleet.MDMWindowsEnrolledDevice{
				MDMDeviceID:           deviceID,
				HostUUID:              hostUUID,
				AwaitingConfiguration: fleet.WindowsMDMAwaitingConfigurationActive,
			}, nil
		}
		ds.ListMDMWindowsProfilesToInstallForHostFunc = func(ctx context.Context, hUUID string) ([]*fleet.MDMWindowsProfilePayload, error) {
			return nil, nil
		}
		ds.GetHostMDMWindowsProfilesFunc = func(ctx context.Context, hUUID string) ([]fleet.HostMDMWindowsProfile, error) {
			return nil, nil
		}
		ds.ListSetupExperienceResultsByHostUUIDFunc = func(ctx context.Context, hUUID string, teamID uint) ([]*fleet.SetupExperienceStatusResult, error) {
			return []*fleet.SetupExperienceStatusResult{
				{Name: "Slack", Status: fleet.SetupExperienceStatusCancelled},
				{Name: "Asana", Status: fleet.SetupExperienceStatusSuccess},
			}, nil
		}
		ds.SetMDMWindowsAwaitingConfigurationFunc = func(ctx context.Context, mdmDeviceID string, from, to fleet.WindowsMDMAwaitingConfiguration) (bool, error) {
			return true, nil
		}
		setRequireAll(ds, false)
		ds.MDMWindowsInsertCommandsForHostFunc = func(ctx context.Context, hostUUIDOrDeviceID string, cmds []*fleet.MDMWindowsCommand) error {
			return nil
		}

		cmds, err := svc.getESPCommands(t.Context(), deviceID)
		require.NoError(t, err)
		require.NotEmpty(t, cmds, "should release device when all setup items are terminal (cancelled or success)")
		assert.True(t, ds.SetMDMWindowsAwaitingConfigurationFuncInvoked,
			"should transition awaiting_configuration out of Active")
	})

	t.Run("active with no profiles releases device", func(t *testing.T) {
		ds, svc := newSvc(t)
		ds.MDMWindowsGetEnrolledDeviceWithDeviceIDFunc = func(ctx context.Context, mdmDeviceID string) (*fleet.MDMWindowsEnrolledDevice, error) {
			return &fleet.MDMWindowsEnrolledDevice{
				MDMDeviceID:           deviceID,
				HostUUID:              hostUUID,
				AwaitingConfiguration: fleet.WindowsMDMAwaitingConfigurationActive,
			}, nil
		}
		ds.ListMDMWindowsProfilesToInstallForHostFunc = func(ctx context.Context, hUUID string) ([]*fleet.MDMWindowsProfilePayload, error) {
			return nil, nil
		}
		ds.GetHostMDMWindowsProfilesFunc = func(ctx context.Context, hUUID string) ([]fleet.HostMDMWindowsProfile, error) {
			return nil, nil
		}
		setReleaseMocks(ds)
		ds.MDMWindowsInsertCommandsForHostFunc = func(ctx context.Context, hostUUIDOrDeviceID string, cmds []*fleet.MDMWindowsCommand) error {
			return nil
		}

		cmds, err := svc.getESPCommands(t.Context(), deviceID)
		require.NoError(t, err)
		require.NotEmpty(t, cmds, "should return release commands when no profiles configured")
		assert.True(t, ds.SetMDMWindowsAwaitingConfigurationFuncInvoked,
			"should transition awaiting_configuration out of Active")
	})

	// findCmdByLocURI returns the first SyncMLCmd whose target LocURI contains
	// the given substring, or nil if none match.
	findCmdByLocURI := func(cmds []*fleet.SyncMLCmd, substr string) *fleet.SyncMLCmd {
		for _, c := range cmds {
			if c.GetTargetURI() != "" && strings.Contains(c.GetTargetURI(), substr) {
				return c
			}
		}
		return nil
	}

	t.Run("software failure with require_all=true sends block commands", func(t *testing.T) {
		ds, svc := newSvc(t)
		ds.MDMWindowsGetEnrolledDeviceWithDeviceIDFunc = func(ctx context.Context, mdmDeviceID string) (*fleet.MDMWindowsEnrolledDevice, error) {
			return &fleet.MDMWindowsEnrolledDevice{
				MDMDeviceID:           deviceID,
				HostUUID:              hostUUID,
				AwaitingConfiguration: fleet.WindowsMDMAwaitingConfigurationActive,
			}, nil
		}
		ds.ListMDMWindowsProfilesToInstallForHostFunc = func(ctx context.Context, hUUID string) ([]*fleet.MDMWindowsProfilePayload, error) {
			return nil, nil
		}
		ds.GetHostMDMWindowsProfilesFunc = func(ctx context.Context, hUUID string) ([]fleet.HostMDMWindowsProfile, error) {
			return nil, nil
		}
		ds.ListSetupExperienceResultsByHostUUIDFunc = func(ctx context.Context, hUUID string, teamID uint) ([]*fleet.SetupExperienceStatusResult, error) {
			titleID := uint(42)
			return []*fleet.SetupExperienceStatusResult{
				{
					Name:                "Critical App",
					Status:              fleet.SetupExperienceStatusFailure,
					SoftwareInstallerID: new(uint(7)),
					SoftwareTitleID:     &titleID,
				},
			}, nil
		}
		setRequireAll(ds, true)
		ds.SetMDMWindowsAwaitingConfigurationFunc = func(ctx context.Context, mdmDeviceID string, from, to fleet.WindowsMDMAwaitingConfiguration) (bool, error) {
			return true, nil
		}
		var cancelled bool
		ds.CancelPendingSetupExperienceStepsFunc = func(ctx context.Context, hUUID string) error {
			cancelled = true
			return nil
		}
		var persistedURIs []string
		var persistedCmdUUIDs []string
		var batchCalls int
		ds.MDMWindowsInsertCommandsForHostFunc = func(ctx context.Context, hostUUIDOrDeviceID string, cmds []*fleet.MDMWindowsCommand) error {
			batchCalls++
			for _, c := range cmds {
				persistedURIs = append(persistedURIs, c.TargetLocURI)
				persistedCmdUUIDs = append(persistedCmdUUIDs, c.CommandUUID)
			}
			return nil
		}

		cmds, err := svc.getESPCommands(t.Context(), deviceID)
		require.NoError(t, err)
		require.NotEmpty(t, cmds, "should return block commands")
		assert.True(t, cancelled, "should cancel pending setup experience steps")

		// Block path must persist all four commands in a SINGLE batch call so the retry path preserves
		// CustomErrorText / BlockInStatusPage / AllowCollectLogsButton / TimeOutUntilSyncFailure on dropped responses,
		// and so a partial DB failure can't leave orphan rows behind (the batch is one transaction).
		assert.Equal(t, 1, batchCalls, "persist must be a single batched call, not a loop of single inserts")
		joined := strings.Join(persistedURIs, ",")
		assert.Contains(t, joined, "CustomErrorText", "must persist CustomErrorText")
		assert.Contains(t, joined, "BlockInStatusPage", "must persist BlockInStatusPage")
		assert.Contains(t, joined, "AllowCollectLogsButton", "must persist AllowCollectLogsButton")
		assert.Contains(t, joined, "TimeOutUntilSyncFailure",
			"must persist TimeOutUntilSyncFailure -- this is what actually triggers the failure UI on Windows ESP")

		// Persisted CmdIDs MUST equal the inline CmdIDs so that when the
		// device acks the inline send, the persisted backup row clears via
		// the existing ack-matching path. A regression that generates new
		// UUIDs for the persisted record (e.g., uuid.New() instead of
		// cmd.CmdID.Value) would silently break this contract: the server
		// would re-send the persisted commands on every subsequent session
		// because the device's ack would never match.
		var inlineCmdUUIDs []string
		for _, c := range cmds {
			inlineCmdUUIDs = append(inlineCmdUUIDs, c.CmdID.Value)
		}
		assert.ElementsMatch(t, inlineCmdUUIDs, persistedCmdUUIDs,
			"persisted CommandUUIDs must equal inline CmdID.Value (1-to-1) so the device ack clears the backup")

		// Block path must include CustomErrorText, BlockInStatusPage=4, AllowCollectLogsButton.
		errCmd := findCmdByLocURI(cmds, "CustomErrorText")
		require.NotNil(t, errCmd, "block commands must include CustomErrorText")
		require.NotNil(t, errCmd.Items[0].Data)
		assert.Equal(t, microsoft_mdm.ESPSoftwareFailureErrorText, errCmd.Items[0].Data.Content)

		blockCmd := findCmdByLocURI(cmds, "BlockInStatusPage")
		require.NotNil(t, blockCmd, "block commands must include BlockInStatusPage")
		require.NotNil(t, blockCmd.Items[0].Data)
		assert.Equal(t, "1", blockCmd.Items[0].Data.Content,
			"BlockInStatusPage must be 1 (Reset PC) per DMClient CSP docs")

		logsCmd := findCmdByLocURI(cmds, "AllowCollectLogsButton")
		require.NotNil(t, logsCmd, "block commands must include AllowCollectLogsButton")

		// Block path forces a quick ESP timeout to trigger the failure UI. We deliberately do NOT send
		// ServerHasFinishedProvisioning here: that would tell the ESP it succeeded and proceed past the failure
		// screen entirely.
		timeoutCmd := findCmdByLocURI(cmds, "TimeOutUntilSyncFailure")
		require.NotNil(t, timeoutCmd, "block commands must include TimeOutUntilSyncFailure to force failure")
		assert.Equal(t, "1", timeoutCmd.Items[0].Data.Content,
			"TimeOutUntilSyncFailure must be 1 minute to trigger failure quickly")

		assert.Nil(t, findCmdByLocURI(cmds, "ServerHasFinishedProvisioning"),
			"block commands must NOT include ServerHasFinishedProvisioning -- it would cause ESP success")
		// VM testing confirmed setting InstallationState=4 on the parent PolicyProviders node alone does NOT
		// escalate the ESP UI without per-tracker state from #43776. The timeout-based trigger remains the
		// load-bearing mechanism until LocalMDM tracking lands.
		assert.Nil(t, findCmdByLocURI(cmds, "InstallationState"),
			"block path uses the timeout-based trigger, not InstallationState")
	})

	t.Run("software failure with require_all=false releases with error text", func(t *testing.T) {
		ds, svc := newSvc(t)
		ds.MDMWindowsGetEnrolledDeviceWithDeviceIDFunc = func(ctx context.Context, mdmDeviceID string) (*fleet.MDMWindowsEnrolledDevice, error) {
			return &fleet.MDMWindowsEnrolledDevice{
				MDMDeviceID:           deviceID,
				HostUUID:              hostUUID,
				AwaitingConfiguration: fleet.WindowsMDMAwaitingConfigurationActive,
			}, nil
		}
		ds.ListMDMWindowsProfilesToInstallForHostFunc = func(ctx context.Context, hUUID string) ([]*fleet.MDMWindowsProfilePayload, error) {
			return nil, nil
		}
		ds.GetHostMDMWindowsProfilesFunc = func(ctx context.Context, hUUID string) ([]fleet.HostMDMWindowsProfile, error) {
			return nil, nil
		}
		ds.ListSetupExperienceResultsByHostUUIDFunc = func(ctx context.Context, hUUID string, teamID uint) ([]*fleet.SetupExperienceStatusResult, error) {
			return []*fleet.SetupExperienceStatusResult{
				{Name: "Critical App", Status: fleet.SetupExperienceStatusFailure, SoftwareInstallerID: new(uint(7))},
			}, nil
		}
		setRequireAll(ds, false)
		ds.SetMDMWindowsAwaitingConfigurationFunc = func(ctx context.Context, mdmDeviceID string, from, to fleet.WindowsMDMAwaitingConfiguration) (bool, error) {
			return true, nil
		}
		ds.MDMWindowsInsertCommandsForHostFunc = func(ctx context.Context, hostUUIDOrDeviceID string, cmds []*fleet.MDMWindowsCommand) error {
			return nil
		}

		cmds, err := svc.getESPCommands(t.Context(), deviceID)
		require.NoError(t, err)
		require.NotEmpty(t, cmds, "should return release commands")

		// Release path: device proceeds to login. We do not send CustomErrorText
		// because the failure UI never renders on a release, so any error text
		// would be dead state on the DMClient node.
		assert.Nil(t, findCmdByLocURI(cmds, "CustomErrorText"),
			"release path must not include CustomErrorText (failure UI never renders)")

		assert.NotNil(t, findCmdByLocURI(cmds, "ServerHasFinishedProvisioning"),
			"release path must include ServerHasFinishedProvisioning")
		// Should NOT include BlockInStatusPage (we're not blocking).
		assert.Nil(t, findCmdByLocURI(cmds, "BlockInStatusPage"))

		// Cancel should NOT have been called: items are already terminal.
		assert.False(t, ds.CancelPendingSetupExperienceStepsFuncInvoked,
			"cancel should not be called for require_all=false failure (items already terminal)")
	})

	t.Run("profile failure alone does not block even with require_all=true", func(t *testing.T) {
		// Profile delivery failures (e.g. CSP not supported on the host's
		// edition) should not trigger the ESP block screen. The
		// require_all_software_windows setting is software-scoped (matching
		// macOS), so a failed profile with no software failure must release
		// the device normally.
		ds, svc := newSvc(t)
		ds.MDMWindowsGetEnrolledDeviceWithDeviceIDFunc = func(ctx context.Context, mdmDeviceID string) (*fleet.MDMWindowsEnrolledDevice, error) {
			return &fleet.MDMWindowsEnrolledDevice{
				MDMDeviceID:           deviceID,
				HostUUID:              hostUUID,
				AwaitingConfiguration: fleet.WindowsMDMAwaitingConfigurationActive,
			}, nil
		}
		ds.ListMDMWindowsProfilesToInstallForHostFunc = func(ctx context.Context, hUUID string) ([]*fleet.MDMWindowsProfilePayload, error) {
			return nil, nil
		}
		ds.GetHostMDMWindowsProfilesFunc = func(ctx context.Context, hUUID string) ([]fleet.HostMDMWindowsProfile, error) {
			return []fleet.HostMDMWindowsProfile{
				{ProfileUUID: "prof-1", Name: "WiFi", Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeInstall},
			}, nil
		}
		ds.ListSetupExperienceResultsByHostUUIDFunc = func(ctx context.Context, hUUID string, teamID uint) ([]*fleet.SetupExperienceStatusResult, error) {
			return nil, nil
		}
		ds.HasWindowsSetupExperienceItemsForTeamFunc = func(ctx context.Context, teamID uint) (bool, error) {
			return false, nil
		}
		setRequireAll(ds, true)
		ds.SetMDMWindowsAwaitingConfigurationFunc = func(ctx context.Context, mdmDeviceID string, from, to fleet.WindowsMDMAwaitingConfiguration) (bool, error) {
			return true, nil
		}
		ds.MDMWindowsInsertCommandsForHostFunc = func(ctx context.Context, hostUUIDOrDeviceID string, cmds []*fleet.MDMWindowsCommand) error {
			return nil
		}

		cmds, err := svc.getESPCommands(t.Context(), deviceID)
		require.NoError(t, err)
		require.NotEmpty(t, cmds, "profile failure alone should release the device")

		// Release path: ServerHasFinishedProvisioning is set, BlockInStatusPage is not.
		assert.NotNil(t, findCmdByLocURI(cmds, "ServerHasFinishedProvisioning"),
			"profile-only failure must release the device")
		assert.Nil(t, findCmdByLocURI(cmds, "BlockInStatusPage"),
			"profile-only failure must not block the device")

		// No software failure and no timeout means no error text on the release.
		assert.Nil(t, findCmdByLocURI(cmds, "CustomErrorText"),
			"profile-only failure should not surface error text")

		// Cancel should NOT be called: profile failures don't trigger cancel.
		assert.False(t, ds.CancelPendingSetupExperienceStepsFuncInvoked,
			"profile failure must not cancel pending setup experience steps")
	})

	t.Run("profile failure combined with software failure still blocks on software", func(t *testing.T) {
		// When BOTH a profile and a software install fail, the software
		// failure still triggers the block (with require_all=true) and the
		// software-specific error text wins (because it's more actionable
		// than a generic timeout/profile message).
		ds, svc := newSvc(t)
		ds.MDMWindowsGetEnrolledDeviceWithDeviceIDFunc = func(ctx context.Context, mdmDeviceID string) (*fleet.MDMWindowsEnrolledDevice, error) {
			return &fleet.MDMWindowsEnrolledDevice{
				MDMDeviceID:           deviceID,
				HostUUID:              hostUUID,
				AwaitingConfiguration: fleet.WindowsMDMAwaitingConfigurationActive,
			}, nil
		}
		ds.ListMDMWindowsProfilesToInstallForHostFunc = func(ctx context.Context, hUUID string) ([]*fleet.MDMWindowsProfilePayload, error) {
			return nil, nil
		}
		ds.GetHostMDMWindowsProfilesFunc = func(ctx context.Context, hUUID string) ([]fleet.HostMDMWindowsProfile, error) {
			return []fleet.HostMDMWindowsProfile{
				{ProfileUUID: "prof-1", Name: "WiFi", Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeInstall},
			}, nil
		}
		ds.ListSetupExperienceResultsByHostUUIDFunc = func(ctx context.Context, hUUID string, teamID uint) ([]*fleet.SetupExperienceStatusResult, error) {
			return []*fleet.SetupExperienceStatusResult{
				{Name: "Critical App", Status: fleet.SetupExperienceStatusFailure, SoftwareInstallerID: new(uint(7))},
			}, nil
		}
		setRequireAll(ds, true)
		ds.SetMDMWindowsAwaitingConfigurationFunc = func(ctx context.Context, mdmDeviceID string, from, to fleet.WindowsMDMAwaitingConfiguration) (bool, error) {
			return true, nil
		}
		ds.CancelPendingSetupExperienceStepsFunc = func(ctx context.Context, hUUID string) error {
			return nil
		}
		ds.MDMWindowsInsertCommandsForHostFunc = func(ctx context.Context, hostUUIDOrDeviceID string, cmds []*fleet.MDMWindowsCommand) error {
			return nil
		}

		cmds, err := svc.getESPCommands(t.Context(), deviceID)
		require.NoError(t, err)
		require.NotEmpty(t, cmds)

		assert.NotNil(t, findCmdByLocURI(cmds, "BlockInStatusPage"),
			"software failure with require_all=true blocks regardless of profile state")
		errCmd := findCmdByLocURI(cmds, "CustomErrorText")
		require.NotNil(t, errCmd)
		require.NotNil(t, errCmd.Items[0].Data)
		assert.Equal(t, microsoft_mdm.ESPSoftwareFailureErrorText, errCmd.Items[0].Data.Content,
			"software failure error text takes precedence over profile/timeout text")
	})

	t.Run("timeout with require_all=true sends block and cancels", func(t *testing.T) {
		ds, svc := newSvc(t)
		past := time.Now().Add(-4 * time.Hour)
		ds.MDMWindowsGetEnrolledDeviceWithDeviceIDFunc = func(ctx context.Context, mdmDeviceID string) (*fleet.MDMWindowsEnrolledDevice, error) {
			return &fleet.MDMWindowsEnrolledDevice{
				MDMDeviceID:             deviceID,
				HostUUID:                hostUUID,
				AwaitingConfiguration:   fleet.WindowsMDMAwaitingConfigurationActive,
				AwaitingConfigurationAt: &past,
			}, nil
		}
		setRequireAll(ds, true)
		ds.SetMDMWindowsAwaitingConfigurationFunc = func(ctx context.Context, mdmDeviceID string, from, to fleet.WindowsMDMAwaitingConfiguration) (bool, error) {
			return true, nil
		}
		var cancelled bool
		ds.CancelPendingSetupExperienceStepsFunc = func(ctx context.Context, hUUID string) error {
			cancelled = true
			return nil
		}
		ds.MDMWindowsInsertCommandsForHostFunc = func(ctx context.Context, hostUUIDOrDeviceID string, cmds []*fleet.MDMWindowsCommand) error {
			return nil
		}

		cmds, err := svc.getESPCommands(t.Context(), deviceID)
		require.NoError(t, err)
		require.NotEmpty(t, cmds, "timeout with require_all=true should return block commands")
		assert.True(t, cancelled, "timeout should cancel pending steps")
		assert.NotNil(t, findCmdByLocURI(cmds, "BlockInStatusPage"))

		// Pure-timeout path uses the timeout-specific error text (no software
		// failure was observed because the wait gates were skipped).
		errCmd := findCmdByLocURI(cmds, "CustomErrorText")
		require.NotNil(t, errCmd, "block path includes CustomErrorText")
		require.NotNil(t, errCmd.Items[0].Data)
		assert.Equal(t, microsoft_mdm.ESPTimeoutErrorText, errCmd.Items[0].Data.Content,
			"timeout without software failure uses timeout-specific error text")

		// Wait gates should be skipped on timeout.
		assert.False(t, ds.ListMDMWindowsProfilesToInstallForHostFuncInvoked)
		assert.False(t, ds.GetHostMDMWindowsProfilesFuncInvoked)
		assert.False(t, ds.ListSetupExperienceResultsByHostUUIDFuncInvoked)
	})

	t.Run("timeout with require_all=false releases with error text and cancels", func(t *testing.T) {
		ds, svc := newSvc(t)
		past := time.Now().Add(-4 * time.Hour)
		ds.MDMWindowsGetEnrolledDeviceWithDeviceIDFunc = func(ctx context.Context, mdmDeviceID string) (*fleet.MDMWindowsEnrolledDevice, error) {
			return &fleet.MDMWindowsEnrolledDevice{
				MDMDeviceID:             deviceID,
				HostUUID:                hostUUID,
				AwaitingConfiguration:   fleet.WindowsMDMAwaitingConfigurationActive,
				AwaitingConfigurationAt: &past,
			}, nil
		}
		setRequireAll(ds, false)
		ds.SetMDMWindowsAwaitingConfigurationFunc = func(ctx context.Context, mdmDeviceID string, from, to fleet.WindowsMDMAwaitingConfiguration) (bool, error) {
			return true, nil
		}
		var cancelled bool
		ds.CancelPendingSetupExperienceStepsFunc = func(ctx context.Context, hUUID string) error {
			cancelled = true
			return nil
		}
		ds.MDMWindowsInsertCommandsForHostFunc = func(ctx context.Context, hostUUIDOrDeviceID string, cmds []*fleet.MDMWindowsCommand) error {
			return nil
		}

		cmds, err := svc.getESPCommands(t.Context(), deviceID)
		require.NoError(t, err)
		require.NotEmpty(t, cmds)
		assert.True(t, cancelled, "timeout cancels pending steps regardless of require_all")
		assert.NotNil(t, findCmdByLocURI(cmds, "ServerHasFinishedProvisioning"),
			"timeout with require_all=false releases the device")
		// Release path does not include CustomErrorText: the failure UI never
		// renders on a release, so any error text would be dead state.
		assert.Nil(t, findCmdByLocURI(cmds, "CustomErrorText"),
			"release path must not include CustomErrorText (failure UI never renders)")
		assert.Nil(t, findCmdByLocURI(cmds, "BlockInStatusPage"))
	})

	t.Run("success path does not include error text", func(t *testing.T) {
		ds, svc := newSvc(t)
		ds.MDMWindowsGetEnrolledDeviceWithDeviceIDFunc = func(ctx context.Context, mdmDeviceID string) (*fleet.MDMWindowsEnrolledDevice, error) {
			return &fleet.MDMWindowsEnrolledDevice{
				MDMDeviceID:           deviceID,
				HostUUID:              hostUUID,
				AwaitingConfiguration: fleet.WindowsMDMAwaitingConfigurationActive,
			}, nil
		}
		ds.ListMDMWindowsProfilesToInstallForHostFunc = func(ctx context.Context, hUUID string) ([]*fleet.MDMWindowsProfilePayload, error) {
			return nil, nil
		}
		ds.GetHostMDMWindowsProfilesFunc = func(ctx context.Context, hUUID string) ([]fleet.HostMDMWindowsProfile, error) {
			return []fleet.HostMDMWindowsProfile{
				{ProfileUUID: "prof-1", Name: "WiFi", Status: &fleet.MDMDeliveryVerified, OperationType: fleet.MDMOperationTypeInstall},
			}, nil
		}
		// setReleaseMocks first, then override the setup experience stub
		// so the success-result branch is actually exercised (otherwise
		// setReleaseMocks resets it back to returning empty).
		setReleaseMocks(ds)
		ds.ListSetupExperienceResultsByHostUUIDFunc = func(ctx context.Context, hUUID string, teamID uint) ([]*fleet.SetupExperienceStatusResult, error) {
			return []*fleet.SetupExperienceStatusResult{
				{Name: "Slack", Status: fleet.SetupExperienceStatusSuccess},
			}, nil
		}

		cmds, err := svc.getESPCommands(t.Context(), deviceID)
		require.NoError(t, err)
		require.NotEmpty(t, cmds)
		assert.NotNil(t, findCmdByLocURI(cmds, "ServerHasFinishedProvisioning"))
		assert.Nil(t, findCmdByLocURI(cmds, "CustomErrorText"),
			"success path should not set error text")
		assert.Nil(t, findCmdByLocURI(cmds, "BlockInStatusPage"))
		// Verify we actually exercised the success path (a non-empty results
		// slice with Status=success) rather than falling through the empty
		// branch.
		assert.True(t, ds.ListSetupExperienceResultsByHostUUIDFuncInvoked,
			"success path must call ListSetupExperienceResultsByHostUUID")
	})

	t.Run("require_all read via team config blocks when team has require_all_software_windows=true", func(t *testing.T) {
		// Covers the team-path branch of the require_all_software_windows
		// lookup chain (HostLite returns TeamID set -> TeamLite -> team
		// config). Other tests use the no-team path via setRequireAll.
		ds, svc := newSvc(t)
		ds.MDMWindowsGetEnrolledDeviceWithDeviceIDFunc = func(ctx context.Context, mdmDeviceID string) (*fleet.MDMWindowsEnrolledDevice, error) {
			return &fleet.MDMWindowsEnrolledDevice{
				MDMDeviceID:           deviceID,
				HostUUID:              hostUUID,
				AwaitingConfiguration: fleet.WindowsMDMAwaitingConfigurationActive,
			}, nil
		}
		ds.ListMDMWindowsProfilesToInstallForHostFunc = func(ctx context.Context, hUUID string) ([]*fleet.MDMWindowsProfilePayload, error) {
			return nil, nil
		}
		ds.GetHostMDMWindowsProfilesFunc = func(ctx context.Context, hUUID string) ([]fleet.HostMDMWindowsProfile, error) {
			return nil, nil
		}
		ds.ListSetupExperienceResultsByHostUUIDFunc = func(ctx context.Context, hUUID string, teamID uint) ([]*fleet.SetupExperienceStatusResult, error) {
			return []*fleet.SetupExperienceStatusResult{
				{Name: "Critical App", Status: fleet.SetupExperienceStatusFailure, SoftwareInstallerID: new(uint(7))},
			}, nil
		}

		// Team-path mocks: HostLite returns a host with TeamID set; TeamLite
		// returns the team config with require_all_software_windows=true.
		teamID := uint(42)
		ds.HostLiteByIdentifierFunc = func(ctx context.Context, identifier string) (*fleet.HostLite, error) {
			return &fleet.HostLite{ID: 1, UUID: identifier, TeamID: &teamID}, nil
		}
		var teamLiteCalled bool
		ds.TeamLiteFunc = func(ctx context.Context, tid uint) (*fleet.TeamLite, error) {
			require.Equal(t, teamID, tid, "TeamLite must be called with the host's team_id")
			teamLiteCalled = true
			return &fleet.TeamLite{
				ID: tid,
				Config: fleet.TeamConfigLite{
					MDM: fleet.TeamMDM{
						MacOSSetup: fleet.MacOSSetup{
							RequireAllSoftwareWindows: true,
						},
					},
				},
			}, nil
		}
		// AppConfig MUST NOT be consulted on the team path.
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			t.Fatal("AppConfig must not be called when host has a team_id")
			return nil, nil
		}
		ds.SetMDMWindowsAwaitingConfigurationFunc = func(ctx context.Context, mdmDeviceID string, from, to fleet.WindowsMDMAwaitingConfiguration) (bool, error) {
			return true, nil
		}
		ds.CancelPendingSetupExperienceStepsFunc = func(ctx context.Context, hUUID string) error {
			return nil
		}
		ds.MDMWindowsInsertCommandsForHostFunc = func(ctx context.Context, hostUUIDOrDeviceID string, cmds []*fleet.MDMWindowsCommand) error {
			return nil
		}

		cmds, err := svc.getESPCommands(t.Context(), deviceID)
		require.NoError(t, err)
		require.NotEmpty(t, cmds)
		assert.True(t, teamLiteCalled, "TeamLite must be called on the team path")
		assert.NotNil(t, findCmdByLocURI(cmds, "BlockInStatusPage"),
			"team config require_all_software_windows=true must drive the block path")
	})

	t.Run("persist failure aborts finalize without committing CAS", func(t *testing.T) {
		// Safety property: if the persist (dropped-response retry safety net) fails, we must NOT commit the CAS
		// transition Active -> None. Otherwise the device would be left without an inline send AND without the retry
		// backup -- stuck on "Working on it..." forever, since awaiting_configuration=None means subsequent management
		// sessions return no ESP commands. Persist runs before the CAS for exactly this reason.
		ds, svc := newSvc(t)
		ds.MDMWindowsGetEnrolledDeviceWithDeviceIDFunc = func(ctx context.Context, mdmDeviceID string) (*fleet.MDMWindowsEnrolledDevice, error) {
			return &fleet.MDMWindowsEnrolledDevice{
				MDMDeviceID:           deviceID,
				HostUUID:              hostUUID,
				AwaitingConfiguration: fleet.WindowsMDMAwaitingConfigurationActive,
			}, nil
		}
		ds.ListMDMWindowsProfilesToInstallForHostFunc = func(ctx context.Context, hUUID string) ([]*fleet.MDMWindowsProfilePayload, error) {
			return nil, nil
		}
		ds.GetHostMDMWindowsProfilesFunc = func(ctx context.Context, hUUID string) ([]fleet.HostMDMWindowsProfile, error) {
			return nil, nil
		}
		ds.ListSetupExperienceResultsByHostUUIDFunc = func(ctx context.Context, hUUID string, teamID uint) ([]*fleet.SetupExperienceStatusResult, error) {
			return []*fleet.SetupExperienceStatusResult{
				{Name: "Critical App", Status: fleet.SetupExperienceStatusFailure, SoftwareInstallerID: new(uint(7))},
			}, nil
		}
		setRequireAll(ds, true)
		// Cancel runs BEFORE persist (so a transient cancel failure aborts cleanly without committing CAS); cancel
		// is idempotent so it's safe to run again on the next session if persist fails afterwards.
		ds.CancelPendingSetupExperienceStepsFunc = func(ctx context.Context, hUUID string) error {
			return nil
		}
		ds.MDMWindowsInsertCommandsForHostFunc = func(ctx context.Context, hostUUIDOrDeviceID string, cmds []*fleet.MDMWindowsCommand) error {
			return errors.New("transient db error")
		}
		ds.SetMDMWindowsAwaitingConfigurationFunc = func(ctx context.Context, mdmDeviceID string, from, to fleet.WindowsMDMAwaitingConfiguration) (bool, error) {
			t.Fatal("CAS Active->None must NOT run when persist fails")
			return false, nil
		}

		cmds, err := svc.getESPCommands(t.Context(), deviceID)
		require.Error(t, err, "must return error so device retries on next session")
		assert.Nil(t, cmds)
		assert.False(t, ds.SetMDMWindowsAwaitingConfigurationFuncInvoked,
			"CAS must NOT have been invoked when persist fails")
	})

	t.Run("cancel failure aborts finalize without committing CAS", func(t *testing.T) {
		// Cancel runs before persist and CAS. A transient cancel failure must abort the finalize cleanly: otherwise we
		// would commit awaiting=None while leaving non-terminal setup-experience rows behind, which is exactly the
		// state cancellation is supposed to prevent. CancelPendingSetupExperienceSteps is idempotent so a retry on the
		// next session is safe.
		ds, svc := newSvc(t)
		ds.MDMWindowsGetEnrolledDeviceWithDeviceIDFunc = func(ctx context.Context, mdmDeviceID string) (*fleet.MDMWindowsEnrolledDevice, error) {
			return &fleet.MDMWindowsEnrolledDevice{
				MDMDeviceID:           deviceID,
				HostUUID:              hostUUID,
				AwaitingConfiguration: fleet.WindowsMDMAwaitingConfigurationActive,
			}, nil
		}
		ds.ListMDMWindowsProfilesToInstallForHostFunc = func(ctx context.Context, hUUID string) ([]*fleet.MDMWindowsProfilePayload, error) {
			return nil, nil
		}
		ds.GetHostMDMWindowsProfilesFunc = func(ctx context.Context, hUUID string) ([]fleet.HostMDMWindowsProfile, error) {
			return nil, nil
		}
		ds.ListSetupExperienceResultsByHostUUIDFunc = func(ctx context.Context, hUUID string, teamID uint) ([]*fleet.SetupExperienceStatusResult, error) {
			return []*fleet.SetupExperienceStatusResult{
				{Name: "Critical App", Status: fleet.SetupExperienceStatusFailure, SoftwareInstallerID: new(uint(7))},
			}, nil
		}
		setRequireAll(ds, true)
		ds.CancelPendingSetupExperienceStepsFunc = func(ctx context.Context, hUUID string) error {
			return errors.New("transient db error")
		}
		ds.MDMWindowsInsertCommandsForHostFunc = func(ctx context.Context, hostUUIDOrDeviceID string, cmds []*fleet.MDMWindowsCommand) error {
			t.Fatal("persist must NOT run when cancel fails")
			return nil
		}
		ds.SetMDMWindowsAwaitingConfigurationFunc = func(ctx context.Context, mdmDeviceID string, from, to fleet.WindowsMDMAwaitingConfiguration) (bool, error) {
			t.Fatal("CAS Active->None must NOT run when cancel fails")
			return false, nil
		}

		cmds, err := svc.getESPCommands(t.Context(), deviceID)
		require.Error(t, err, "must return error so device retries on next session")
		assert.Nil(t, cmds)
		assert.False(t, ds.MDMWindowsInsertCommandsForHostFuncInvoked,
			"persist must NOT have been invoked when cancel fails")
		assert.False(t, ds.SetMDMWindowsAwaitingConfigurationFuncInvoked,
			"CAS must NOT have been invoked when cancel fails")
	})

	t.Run("require_all lookup error returns error and keeps device active", func(t *testing.T) {
		ds, svc := newSvc(t)
		ds.MDMWindowsGetEnrolledDeviceWithDeviceIDFunc = func(ctx context.Context, mdmDeviceID string) (*fleet.MDMWindowsEnrolledDevice, error) {
			return &fleet.MDMWindowsEnrolledDevice{
				MDMDeviceID:           deviceID,
				HostUUID:              hostUUID,
				AwaitingConfiguration: fleet.WindowsMDMAwaitingConfigurationActive,
			}, nil
		}
		ds.ListMDMWindowsProfilesToInstallForHostFunc = func(ctx context.Context, hUUID string) ([]*fleet.MDMWindowsProfilePayload, error) {
			return nil, nil
		}
		ds.GetHostMDMWindowsProfilesFunc = func(ctx context.Context, hUUID string) ([]fleet.HostMDMWindowsProfile, error) {
			return nil, nil
		}
		ds.ListSetupExperienceResultsByHostUUIDFunc = func(ctx context.Context, hUUID string, teamID uint) ([]*fleet.SetupExperienceStatusResult, error) {
			return nil, nil
		}
		ds.HasWindowsSetupExperienceItemsForTeamFunc = func(ctx context.Context, teamID uint) (bool, error) {
			return false, nil
		}
		ds.HostLiteByIdentifierFunc = func(ctx context.Context, identifier string) (*fleet.HostLite, error) {
			return nil, errors.New("transient db error")
		}

		cmds, err := svc.getESPCommands(t.Context(), deviceID)
		require.Error(t, err, "must return error so device retries on next session")
		assert.Nil(t, cmds)
		assert.False(t, ds.SetMDMWindowsAwaitingConfigurationFuncInvoked,
			"must NOT transition to None on lookup failure")
	})

	t.Run("active waits when results empty but setup experience configured", func(t *testing.T) {
		ds, svc := newSvc(t)
		ds.MDMWindowsGetEnrolledDeviceWithDeviceIDFunc = func(ctx context.Context, mdmDeviceID string) (*fleet.MDMWindowsEnrolledDevice, error) {
			return &fleet.MDMWindowsEnrolledDevice{
				MDMDeviceID:           deviceID,
				HostUUID:              hostUUID,
				AwaitingConfiguration: fleet.WindowsMDMAwaitingConfigurationActive,
			}, nil
		}
		ds.ListMDMWindowsProfilesToInstallForHostFunc = func(ctx context.Context, hUUID string) ([]*fleet.MDMWindowsProfilePayload, error) {
			return nil, nil
		}
		ds.GetHostMDMWindowsProfilesFunc = func(ctx context.Context, hUUID string) ([]fleet.HostMDMWindowsProfile, error) {
			return nil, nil
		}
		// Setup experience is configured for the team but orbit hasn't
		// called SetupExperienceInit yet, so results are empty.
		ds.ListSetupExperienceResultsByHostUUIDFunc = func(ctx context.Context, hUUID string, teamID uint) ([]*fleet.SetupExperienceStatusResult, error) {
			return nil, nil
		}
		ds.HasWindowsSetupExperienceItemsForTeamFunc = func(ctx context.Context, teamID uint) (bool, error) {
			return true, nil
		}
		// HostLite is needed by the empty-results disambiguation (loadTeamID feeds the EXISTS query) but
		// the wait outcome must NOT trigger the Active->None transition.
		ds.HostLiteByIdentifierFunc = func(ctx context.Context, identifier string) (*fleet.HostLite, error) {
			return &fleet.HostLite{ID: 1, UUID: identifier, TeamID: nil}, nil
		}

		cmds, err := svc.getESPCommands(t.Context(), deviceID)
		require.NoError(t, err)
		assert.Nil(t, cmds, "should wait for orbit to initialize setup experience")
		// Must NOT have proceeded to the Active->None transition.
		assert.False(t, ds.SetMDMWindowsAwaitingConfigurationFuncInvoked,
			"must not transition state while waiting for orbit init")
	})
}

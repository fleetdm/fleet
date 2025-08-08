package service

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	mdm_types "github.com/fleetdm/fleet/v4/server/fleet"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/fleetdm/fleet/v4/server/mdm/microsoft/syncml"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/go-kit/log"
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
	soapFaultMsg := NewSoapFault(syncml.SoapErrorAuthentication, mdm_types.MDEDiscovery, errors.New("test"))
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
	soapFaultMsg := NewSoapFault(syncml.SoapErrorAuthentication, mdm_types.MDEDiscovery, errors.New(targetErrorString))
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
	appConfigData := NewApplicationProvisioningData(microsoft_mdm.MDE2EnrollPath)
	appDMClientData := NewDMClientProvisioningData()
	provDoc := NewProvisioningDoc(certStoreData, appConfigData, appDMClientData)

	outXML, err := xml.MarshalIndent(provDoc, "", "  ")
	require.NoError(t, err)
	require.NotEmpty(t, outXML)
	require.Contains(t, string(outXML), deviceIdentityFingerprint)
	require.Contains(t, string(outXML), serverIdentityFingerprint)
	require.Contains(t, string(outXML), microsoft_mdm.MDE2EnrollPath)
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

func TestPreprocessWindowsProfileContents(t *testing.T) {
	testHostUUID := "test-host-1234-uuid"

	tests := []struct {
		name            string
		hostUUID        string
		profileContents string
		wantContents    string
	}{
		{
			name:            "no_fleet_variables",
			hostUUID:        testHostUUID,
			profileContents: `<Replace><CmdID>1</CmdID><Item><Target><LocURI>./Device/Config</LocURI></Target></Item></Replace>`,
			wantContents:    `<Replace><CmdID>1</CmdID><Item><Target><LocURI>./Device/Config</LocURI></Target></Item></Replace>`,
		},
		{
			name:            "host_uuid_without_braces",
			hostUUID:        testHostUUID,
			profileContents: `<Replace><CmdID>1</CmdID><Item><Data>Device ID: $FLEET_VAR_HOST_UUID</Data></Item></Replace>`,
			wantContents:    `<Replace><CmdID>1</CmdID><Item><Data>Device ID: test-host-1234-uuid</Data></Item></Replace>`,
		},
		{
			name:            "host_uuid_with_braces",
			hostUUID:        testHostUUID,
			profileContents: `<Replace><CmdID>1</CmdID><Item><Data>Device ID: ${FLEET_VAR_HOST_UUID}</Data></Item></Replace>`,
			wantContents:    `<Replace><CmdID>1</CmdID><Item><Data>Device ID: test-host-1234-uuid</Data></Item></Replace>`,
		},
		{
			name:            "multiple_host_uuid_occurrences",
			hostUUID:        testHostUUID,
			profileContents: `<Replace><Data>ID1: $FLEET_VAR_HOST_UUID, ID2: ${FLEET_VAR_HOST_UUID}</Data></Replace>`,
			wantContents:    `<Replace><Data>ID1: test-host-1234-uuid, ID2: test-host-1234-uuid</Data></Replace>`,
		},
		{
			name:            "host_uuid_with_xml_special_chars",
			hostUUID:        "test<>&\"'uuid",
			profileContents: `<Replace><Data>ID: $FLEET_VAR_HOST_UUID</Data></Replace>`,
			wantContents:    `<Replace><Data>ID: test&lt;&gt;&amp;&#34;&#39;uuid</Data></Replace>`,
		},
		{
			name:            "unsupported_variable_ignored",
			hostUUID:        testHostUUID,
			profileContents: `<Replace><Data>ID: $FLEET_VAR_HOST_UUID, Other: $FLEET_VAR_UNSUPPORTED</Data></Replace>`,
			wantContents:    `<Replace><Data>ID: test-host-1234-uuid, Other: $FLEET_VAR_UNSUPPORTED</Data></Replace>`,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			gotContents := preprocessWindowsProfileContents(tt.hostUUID, tt.profileContents)
			require.Equal(t, tt.wantContents, gotContents)
		})
	}
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
	cmdMsg := newSyncMLCmdBool(mdm_types.CmdReplace, testOmaURI, testData)
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
	cmdMsg := newSyncMLCmdInt(mdm_types.CmdReplace, testOmaURI, testData)
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
	cmdMsg := newSyncMLCmdText(mdm_types.CmdReplace, testOmaURI, testData)
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
	cmdMsg := newSyncMLCmdXml(mdm_types.CmdReplace, testOmaURI, testData)
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
	cmdMsg := NewSyncMLCmd(mdm_types.CmdReplace, testCmdSource, testCmdTarget, testCmdDataType, testCmdDataFormat, testCmdDataValue)
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
	cmd, err := buildCommandFromProfileBytes([]byte("<Replace></Add>"), "")
	require.Nil(t, cmd)
	require.ErrorContains(t, err, "unmarshalling profile")

	rawSyncML := syncMLForTest("foo/bar")

	// build and generate a command
	cmd, err = buildCommandFromProfileBytes(rawSyncML, "uuid-1")
	require.Nil(t, err)
	require.Equal(t, "uuid-1", cmd.CommandUUID)
	require.Empty(t, cmd.TargetLocURI)

	syncOne := new(mdm_types.SyncMLCmd)
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
	cmd, err = buildCommandFromProfileBytes(rawSyncML, "uuid-2")
	require.Nil(t, err)
	require.Equal(t, "uuid-2", cmd.CommandUUID)
	require.Empty(t, cmd.TargetLocURI)
	syncTwo := new(mdm_types.SyncMLCmd)
	err = xml.Unmarshal(cmd.RawCommand, syncTwo)
	require.NoError(t, err)
	require.Len(t, syncTwo.ReplaceCommands, 1)
	require.NotEmpty(t, syncTwo.ReplaceCommands[0].CmdID.Value)
	// generated xml contains additional comments about CmdID
	require.Equal(
		t,
		fmt.Sprintf(`<Atomic><!-- CmdID generated by Fleet --><CmdID>uuid-2</CmdID><Replace><!-- CmdID generated by Fleet --><CmdID>%s</CmdID><Item><Target><LocURI>foo/bar</LocURI></Target></Item></Replace><Add><!-- CmdID generated by Fleet --><CmdID>%s</CmdID><Item><Target><LocURI>foo/bar</LocURI></Target></Item></Add></Atomic>`, syncTwo.ReplaceCommands[0].CmdID.Value, syncTwo.AddCommands[0].CmdID.Value),
		string(cmd.RawCommand),
	)

	// uuids of replaces are different
	require.NotEqual(t, syncOne.ReplaceCommands[0].CmdID.Value, syncTwo.ReplaceCommands[0].CmdID.Value)
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

func TestReconcileWindowsProfilesWithFleetVariableError(t *testing.T) {
	ctx := context.Background()
	ds := new(mock.Store)
	logger := log.NewNopLogger()

	// Setup test data with a profile containing Fleet variable
	testHostUUID := "test-host-uuid"
	// Profile with Fleet variable that would cause preprocessing to succeed normally
	testProfile := &fleet.MDMWindowsConfigProfile{
		ProfileUUID: "test-profile-uuid",
		Name:        "Test Profile with Variable",
		SyncML:      []byte(`<Replace><Item><Target><LocURI>./Test</LocURI></Target><Data>Host: $FLEET_VAR_HOST_UUID</Data></Item></Replace>`),
	}

	// Mock ListMDMWindowsProfilesToInstall to return a profile with Fleet variable
	ds.ListMDMWindowsProfilesToInstallFunc = func(ctx context.Context) ([]*fleet.MDMWindowsProfilePayload, error) {
		return []*fleet.MDMWindowsProfilePayload{
			{
				ProfileUUID:   testProfile.ProfileUUID,
				ProfileName:   testProfile.Name,
				HostUUID:      testHostUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		}, nil
	}

	// Mock ListMDMWindowsProfilesToRemove to return no profiles
	ds.ListMDMWindowsProfilesToRemoveFunc = func(ctx context.Context) ([]*fleet.MDMWindowsProfilePayload, error) {
		return nil, nil
	}

	// Mock GetMDMWindowsProfilesContents to return the profile content
	ds.GetMDMWindowsProfilesContentsFunc = func(ctx context.Context, profileUUIDs []string) (map[string]fleet.MDMWindowsProfileContents, error) {
		return map[string]fleet.MDMWindowsProfileContents{
			testProfile.ProfileUUID: {
				SyncML:   testProfile.SyncML,
				Checksum: []byte("test-checksum"),
			},
		}, nil
	}

	// Mock MDMWindowsInsertCommandForHosts to succeed (but fail specifically for preprocessed content)
	var receivedCommand *fleet.MDMWindowsCommand
	ds.MDMWindowsInsertCommandForHostsFunc = func(ctx context.Context, hostUUIDs []string, cmd *fleet.MDMWindowsCommand) error {
		receivedCommand = cmd
		// Simulate error only for commands with substituted UUID (to test error handling)
		if strings.Contains(string(cmd.RawCommand), testHostUUID) {
			return errors.New("command insert failed after preprocessing")
		}
		return nil
	}

	// Mock BulkUpsertMDMWindowsHostProfiles to capture status updates
	var capturedUpdates []*fleet.MDMWindowsBulkUpsertHostProfilePayload
	ds.BulkUpsertMDMWindowsHostProfilesFunc = func(ctx context.Context, payload []*fleet.MDMWindowsBulkUpsertHostProfilePayload) error {
		capturedUpdates = append(capturedUpdates, payload...)
		return nil
	}

	// Mock GetMDMWindowsBitLockerSummary
	ds.GetMDMWindowsBitLockerSummaryFunc = func(ctx context.Context, teamID *uint) (*fleet.MDMWindowsBitLockerSummary, error) {
		return &fleet.MDMWindowsBitLockerSummary{}, nil
	}

	// Mock AppConfig to enable Windows MDM
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			MDM: fleet.MDM{
				WindowsEnabledAndConfigured: true,
			},
		}, nil
	}

	// Mock BulkDeleteMDMWindowsHostsConfigProfiles (for remove operations)
	ds.BulkDeleteMDMWindowsHostsConfigProfilesFunc = func(ctx context.Context, payload []*fleet.MDMWindowsProfilePayload) error {
		return nil
	}

	// Run ReconcileWindowsProfiles
	err := ReconcileWindowsProfiles(ctx, ds, logger)
	require.NoError(t, err) // The function should not return an error even if insert fails

	// Verify the command was preprocessed (UUID should be substituted)
	require.NotNil(t, receivedCommand, "Command should have been created")
	require.Contains(t, string(receivedCommand.RawCommand), testHostUUID, "UUID should have been substituted in the command")
	require.NotContains(t, string(receivedCommand.RawCommand), "$FLEET_VAR_HOST_UUID", "Fleet variable should have been replaced")

	// Verify that the error was captured and the profile was marked as failed
	require.True(t, ds.MDMWindowsInsertCommandForHostsFuncInvoked, "MDMWindowsInsertCommandForHosts should have been called")
	require.True(t, ds.BulkUpsertMDMWindowsHostProfilesFuncInvoked, "BulkUpsertMDMWindowsHostProfiles should have been called")

	// Find the error status update
	var foundError bool
	for _, update := range capturedUpdates {
		if update.Status != nil && *update.Status == fleet.MDMDeliveryFailed {
			foundError = true
			require.Contains(t, update.Detail, "command insert failed after preprocessing", "Error detail should contain the original error message")
			break
		}
	}
	require.True(t, foundError, "Should have found a failed status update")
}

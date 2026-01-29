package mdmtest

import (
	"bytes"
	"crypto/md5" //nolint:gosec // Windows MDM Auth uses MD5
	"crypto/rsa"
	"crypto/tls"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/fleet"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/fleetdm/fleet/v4/server/mdm/microsoft/syncml"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

type TestWindowsMDMClient struct {
	// DeviceID identifies a MDM enrollment, sent and managed by the device.
	DeviceID string
	// HardwareID identifies a device.
	HardwareID string
	// NotInOOBE indicates whether the enrollment is happening outside of the OOBE(Out Of Box Experience).
	// NotInOOBE true basically means the enrollment happened post-setup(i.e. settings->Work or School Account)
	// False means the enrollment is happening during OOBE/autopilot.
	notInOOBE bool
	// fleetServerURL is the URL of the Fleet server, used to ping the MDM endpoints.
	fleetServerURL string
	// debug enables debug logging of request/responses.
	debug bool
	// enrollmentType is used to simulate different Windows enrollment
	// types (programatic, automatic.)
	enrollmentType fleet.WindowsMDMEnrollmentType
	// TokenIdentifier is used for authentication during the programmatic enrollment.
	TokenIdentifier string
	// lastManagementResp tracks the last response we received from the server.
	lastManagementResp *fleet.SyncML
	// queuedCommandResponses tracks the commands that will be sent next
	// time the device responds to the server.
	queuedCommandResponses map[string]fleet.SyncMLCmd
	// jwtSigningKey is the key used to sign JWTs
	jwtSigningKey *rsa.PrivateKey
	// jwtSigningKeyID is the ID to report in the header for the signing key
	jwtSigningKeyID string

	username string
	password string
	nonce    string
}

// This is a test-only enrollment type to force erroneous behavior.
const WindowsMDMEmptyBinarySecurityTokenEnrollmentType fleet.WindowsMDMEnrollmentType = -1

// TestWindowsMDMClientOption allows configuring a
// TestWindowsMDMClient.
type TestWindowsMDMClientOption func(*TestWindowsMDMClient)

// TestWindowsMDMClientDebug configures the TestWindowsMDMClient to
// run in debug mode.
func TestWindowsMDMClientDebug() TestWindowsMDMClientOption {
	return func(c *TestWindowsMDMClient) {
		c.debug = true
	}
}

func TestWindowsMDMClientNotInOOBE() TestWindowsMDMClientOption {
	return func(c *TestWindowsMDMClient) {
		c.notInOOBE = true
	}
}

func TestWindowsMDMClientWithSigningKey(signingKey *rsa.PrivateKey, signingKeyID string) TestWindowsMDMClientOption {
	return func(c *TestWindowsMDMClient) {
		c.jwtSigningKey = signingKey
		c.jwtSigningKeyID = signingKeyID
	}
}

func NewTestMDMClientWindowsProgramatic(serverURL string, orbitNodeKey string, opts ...TestWindowsMDMClientOption) *TestWindowsMDMClient {
	return newTestMDMClient(serverURL, fleet.WindowsMDMProgrammaticEnrollmentType, orbitNodeKey, opts...)
}

func NewTestMDMClientWindowsEmptyBinarySecurityToken(serverURL string, orbitNodeKey string, opts ...TestWindowsMDMClientOption) *TestWindowsMDMClient {
	return newTestMDMClient(serverURL, WindowsMDMEmptyBinarySecurityTokenEnrollmentType, orbitNodeKey, opts...)
}

func NewTestMDMClientWindowsAutomatic(serverURL string, email string, opts ...TestWindowsMDMClientOption) *TestWindowsMDMClient {
	return newTestMDMClient(serverURL, fleet.WindowsMDMAutomaticEnrollmentType, email, opts...)
}

func newTestMDMClient(serverURL string, enrollmentType fleet.WindowsMDMEnrollmentType, tokenIdentifier string, opts ...TestWindowsMDMClientOption) *TestWindowsMDMClient {
	c := TestWindowsMDMClient{
		fleetServerURL:  serverURL,
		DeviceID:        uuid.NewString(),
		enrollmentType:  enrollmentType,
		TokenIdentifier: tokenIdentifier,
		HardwareID:      uuid.NewString(),
	}
	for _, fn := range opts {
		fn(&c)
	}
	return &c
}

func (c *TestWindowsMDMClient) StartManagementSession() (map[string]fleet.ProtoCmdOperation, error) {
	// Get SessionID
	sessionIDInt := 0
	if c.lastManagementResp != nil {
		sessionID, err := c.lastManagementResp.GetSessionID()
		if err != nil {
			return nil, fmt.Errorf("session ID processing error %w", err)
		}
		sessionIDInt, err = strconv.Atoi(sessionID)
		if err != nil {
			return nil, fmt.Errorf("converting session ID to int: %w", err)
		}
	}
	managementReq := []byte(`
<SyncML xmlns="SYNCML:SYNCML1.2">
<SyncHdr>
	<VerDTD>1.2</VerDTD>
	<VerProto>DM/1.2</VerProto>
	<SessionID>` + fmt.Sprint(sessionIDInt) + `</SessionID>
	<MsgID>1</MsgID>
	<Target>
	<LocURI>` + c.fleetServerURL + microsoft_mdm.MDE2ManagementPath + `</LocURI>
	</Target>
	<Source>
	<LocURI>` + c.DeviceID + `</LocURI>
	</Source>
</SyncHdr>
<SyncBody>
	<Alert>
	<CmdID>2</CmdID>
	<Data>1201</Data>
	</Alert>
	<Alert>
	<CmdID>3</CmdID>
	<Data>1224</Data>
	<Item>
		<Meta>
		<Type xmlns="syncml:metinf">com.microsoft/MDM/LoginStatus</Type>
		</Meta>
		<Data>user</Data>
	</Item>
	</Alert>
	<Replace>
	<CmdID>4</CmdID>
	<Item>
		<Source>
		<LocURI>./DevInfo/DevId</LocURI>
		</Source>
		<Data>` + c.DeviceID + `</Data>
	</Item>
	<Item>
		<Source>
		<LocURI>./DevInfo/Man</LocURI>
		</Source>
		<Data>VMware, Inc.</Data>
	</Item>
	<Item>
		<Source>
		<LocURI>./DevInfo/Mod</LocURI>
		</Source>
		<Data>VMware7,1</Data>
	</Item>
	<Item>
		<Source>
		<LocURI>./DevInfo/DmV</LocURI>
		</Source>
		<Data>1.3</Data>
	</Item>
	<Item>
		<Source>
		<LocURI>./DevInfo/Lang</LocURI>
		</Source>
		<Data>en-US</Data>
	</Item>
	</Replace>
	<Final/>
</SyncBody>
</SyncML>
  `)

	return c.doManagementReq(managementReq)
}

func (c *TestWindowsMDMClient) doManagementReq(rawXMLReq []byte) (map[string]fleet.ProtoCmdOperation, error) {
	if c.debug {
		fmt.Println("=============== management request ================")
		fmt.Println(string(rawXMLReq))
	}

	sendRequest := func(req []byte) (*fleet.SyncML, error) {
		managementResp, err := c.request(microsoft_mdm.MDE2ManagementPath, req)
		if err != nil {
			return nil, err
		}

		rawXMLResp, err := io.ReadAll(managementResp.Body)
		if err != nil {
			return nil, fmt.Errorf("reading response body: %w", err)
		}

		if c.debug {
			fmt.Println("=============== management response ================")
			fmt.Println(string(rawXMLResp))
		}

		var syncML fleet.SyncML
		if err := xml.Unmarshal(rawXMLResp, &syncML); err != nil {
			return nil, fmt.Errorf("unmarshalling response body: %w", err)
		}

		return &syncML, nil
	}

	syncML, err := sendRequest(rawXMLReq)
	if err != nil {
		return nil, err
	}

	if username, password := c.isRekeyRequest(syncML); username != "" && password != "" {
		c.username = username
		c.password = password

		// We rekeyed, so we need to resend the original request
		syncML, err = sendRequest(rawXMLReq)
		if err != nil {
			return nil, err
		}
	}

	if shouldAuth, nonce := c.shouldAuth(syncML); shouldAuth {
		var reqSyncML fleet.SyncML
		if err := xml.Unmarshal(rawXMLReq, &reqSyncML); err != nil {
			return nil, fmt.Errorf("unmarshalling request body for auth: %w", err)
		}

		extractedNonce, _ := base64.StdEncoding.DecodeString(*nonce)
		c.nonce = string(extractedNonce)
		reqSyncML.SyncHdr.Cred = c.getCredHDR()

		// resend the request but now with credentials
		rawXMLReq, err = xml.MarshalIndent(reqSyncML, "", "\t")
		if err != nil {
			return nil, fmt.Errorf("serializing XML req with auth: %w", err)
		}

		if c.debug {
			fmt.Println("=============== management request with auth ================")
			fmt.Println(string(rawXMLReq))
		}

		syncML, err = sendRequest(rawXMLReq)
		if err != nil {
			return nil, err
		}
	}

	c.lastManagementResp = syncML

	cmds := make(map[string]fleet.ProtoCmdOperation)
	for _, p := range c.lastManagementResp.GetOrderedCmds() {
		cmds[p.Cmd.CmdID.Value] = p
	}

	// before returning, clean up any lingering responses
	c.queuedCommandResponses = make(map[string]fleet.SyncMLCmd)
	return cmds, nil
}

func (c *TestWindowsMDMClient) isRekeyRequest(req *fleet.SyncML) (username string, password string) {
	for _, cmd := range req.GetOrderedCmds() {
		if cmd.Verb == fleet.CmdReplace && strings.Contains(cmd.Cmd.GetTargetURI(), "AAuthName") {
			username = cmd.Cmd.GetTargetData()
		} else if cmd.Verb == fleet.CmdReplace && strings.Contains(cmd.Cmd.GetTargetURI(), "AAuthSecret") {
			password = cmd.Cmd.GetTargetData()
		}
	}
	return
}

func (c *TestWindowsMDMClient) shouldAuth(req *fleet.SyncML) (bool, *string) {
	for _, cmd := range req.GetOrderedCmds() {
		if cmd.Verb == fleet.CmdStatus && cmd.Cmd.Chal != nil {
			return true, cmd.Cmd.Chal.Meta.NextNonce.Content
		}
	}
	return false, nil
}

func (c *TestWindowsMDMClient) SendResponse() (map[string]fleet.ProtoCmdOperation, error) {
	// Get SessionID
	sessionID, err := c.lastManagementResp.GetSessionID()
	if err != nil {
		return nil, fmt.Errorf("session ID processing error %w", err)
	}

	// Get MessageID
	messageID, err := c.lastManagementResp.GetMessageID()
	if err != nil {
		return nil, fmt.Errorf("message ID processing error %w", err)
	}
	messageIDInt, err := strconv.Atoi(messageID)
	if err != nil {
		return nil, fmt.Errorf("converting message ID to int: %w", err)
	}

	var msg fleet.SyncML
	msg.Xmlns = syncml.SyncCmdNamespace
	msg.SyncHdr = fleet.SyncHdr{
		VerDTD:    syncml.SyncMLSupportedVersion,
		VerProto:  syncml.SyncMLVerProto,
		SessionID: sessionID,
		MsgID:     fmt.Sprint(messageIDInt + 1),
		Source:    &fleet.LocURI{LocURI: &c.DeviceID},
		Target: &fleet.LocURI{
			LocURI: ptr.String(c.fleetServerURL + microsoft_mdm.MDE2ManagementPath),
		},
		Cred: c.getCredHDR(),
	}

	// iterate over mocked responses and append them to the SyncML message
	for _, protoCmd := range c.queuedCommandResponses {
		msg.AppendCommand(fleet.MDMRaw, protoCmd)
	}

	xmlReq, err := xml.MarshalIndent(msg, "", "\t")
	if err != nil {
		return nil, fmt.Errorf("serializing XML req: %w", err)
	}

	return c.doManagementReq(xmlReq)
}

func (c *TestWindowsMDMClient) getCredHDR() *fleet.CredHdr {
	return &fleet.CredHdr{
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
		Data: c.hashedCredentials(),
	}
}

func (c *TestWindowsMDMClient) hashedCredentials() string {
	credentials := fmt.Sprintf("%s:%s", c.username, c.password)
	credentialsHash := md5.Sum([]byte(credentials)) //nolint:gosec // Windows MDM Auth uses MD5
	credentialsWithNonce := fmt.Sprintf("%s:%s", base64.StdEncoding.EncodeToString(credentialsHash[:]), c.nonce)
	digestHash := md5.Sum([]byte(credentialsWithNonce)) //nolint:gosec // Windows MDM Auth uses MD5
	return base64.StdEncoding.EncodeToString(digestHash[:])
}

// AppendResponse sets a response for a specific command UUID.
func (c *TestWindowsMDMClient) AppendResponse(op fleet.SyncMLCmd) {
	c.queuedCommandResponses[op.CmdID.Value] = op
}

func (c *TestWindowsMDMClient) GetCurrentMsgID() (string, error) {
	msgID, err := c.lastManagementResp.GetMessageID()
	if err != nil {
		return "", fmt.Errorf("getting management response msg id: %w", err)
	}
	return msgID, nil
}

func (c *TestWindowsMDMClient) Enroll() error {
	if err := c.Discovery(); err != nil {
		return err
	}

	if err := c.Policy(); err != nil {
		return err
	}

	binarySecToken, tokenValueType, err := c.getToken()
	if err != nil {
		return fmt.Errorf("getting binary token: %w", err)
	}

	enrollReq := []byte(`
<s:Envelope
    xmlns:s="http://www.w3.org/2003/05/soap-envelope"
    xmlns:a="http://www.w3.org/2005/08/addressing"
    xmlns:u="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd"
    xmlns:wsse="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd"
    xmlns:wst="http://docs.oasis-open.org/ws-sx/ws-trust/200512"
    xmlns:ac="http://schemas.xmlsoap.org/ws/2006/12/authorization">
    <s:Header>
        <a:Action s:mustUnderstand="1">http://schemas.microsoft.com/windows/pki/2009/01/enrollment/RST/wstep</a:Action>
        <a:MessageID>urn:uuid:0d5a1441-5891-453b-becf-a2e5f6ea3749</a:MessageID>
        <a:ReplyTo>
            <a:Address>http://www.w3.org/2005/08/addressing/anonymous</a:Address>
        </a:ReplyTo>
        <a:To s:mustUnderstand="1">` + c.fleetServerURL + microsoft_mdm.MDE2EnrollPath + `</a:To>
        <wsse:Security s:mustUnderstand="1">
            <wsse:BinarySecurityToken ValueType="` + tokenValueType + `" EncodingType="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd#base64binary">` + binarySecToken + `</wsse:BinarySecurityToken>
        </wsse:Security>
    </s:Header>
    <s:Body>
        <wst:RequestSecurityToken>
            <wst:TokenType>http://schemas.microsoft.com/5.0.0.0/ConfigurationManager/Enrollment/DeviceEnrollmentToken</wst:TokenType>
            <wst:RequestType>http://docs.oasis-open.org/ws-sx/ws-trust/200512/Issue</wst:RequestType>
            <wsse:BinarySecurityToken ValueType="http://schemas.microsoft.com/windows/pki/2009/01/enrollment#PKCS10" EncodingType="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd#base64binary">MIIC5jCCAc4CAQAwSjFIMEYGA1UEAww/MEYzQjhFNkMtQTI3MS00NTU2LTlCNzIt
QTI2Q0JEITgwOTBDOEI0ODRBMEUyNEVCNUM1NkU4MDZDQjRFRTVCMIIBIjANBgkq
hkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAoLj7gBWVMPiVsbrB13jW86bB/Rz+bAOj
J9MxMwuwOtbPicESpReZ7QgjNhv5tTubLCHRlIRhcawxPOhpZCZTRolT/3q2xhYT
3WnW8uLiPLTyQpmoI66yfMAUlNfKboeFpgMB6GCM3FColmQBHzWrPulY5zUSwBFs
YwogoSKVH9ekAv5FQZpqW8zj9tTU1t7U1qMwyb03u1+7JGJ0lBBjCDkoMCB0sSVO
Fybg//zsHqdYs876jnh8qH6GG8XUVrCk4PYX/b1Fak9D4DcedCQ/sDlsxB1i4TjY
apbduFo9/wc/OL9KVBk2LWPXvwV0/EWggx4QFZpaabeJy5J0CbdpvQIDAQABoFcw
VQYJKoZIhvcNAQkOMUgwRjATBgNVHSUEDDAKBggrBgEFBQcDAjAvBgorBgEEAYI3
QgEABCE4MDkwQzhCNDg0QTBFMjRFQjVDNTZFODA2Q0I0RUU1QgAwDQYJKoZIhvcN
AQELBQADggEBAFwiNxM90FippSvLgoqMw9TpyoSTD2hftPW+bpGA1OxxBmSwCwI9
oE7/6bMLX9k9iBt6QaQomWp6Gh+Rpuz0uzHp32TLbuV87//awydG8meyU6GMVZ6R
xfIAH4rmdhJ9ccpnugSLMYr3+UKLWSOjeTB2ZKcVx7LTsHzqaDg3ghJDSNx12wSY
LmEKCHDR1FNPcXB6hfs3CfJOnJhcOX+Gg2GrqjAEA2ty2rEJ9LVZo0Q3A7pfEezs
YioVozr1IWYySwWVzMf/SUwKZkKJCAJmSVcixE+4kxPkyPGyauIrN3wWC0zb+mjF
3aJBpJrK45UhKb1LOBHOtV7BsoEkOUNmCdQ=
</wsse:BinarySecurityToken>
            <ac:AdditionalContext
                xmlns="http://schemas.xmlsoap.org/ws/2006/12/authorization">
                <ac:ContextItem Name="UXInitiated">
                    <ac:Value>false</ac:Value>
                </ac:ContextItem>
                <ac:ContextItem Name="HWDevID">
                    <ac:Value>` + c.HardwareID + `</ac:Value>
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
                    <ac:Value>DESKTOP-H1T20J1</ac:Value>
                </ac:ContextItem>
                <ac:ContextItem Name="MAC">
                    <ac:Value>A2-19-7D-41-B3-9C</ac:Value>
                </ac:ContextItem>
                <ac:ContextItem Name="DeviceID">
                    <ac:Value>` + c.DeviceID + `</ac:Value>
                </ac:ContextItem>
                <ac:ContextItem Name="EnrollmentType">
                    <ac:Value>Full</ac:Value>
                </ac:ContextItem>
                <ac:ContextItem Name="DeviceType">
                    <ac:Value>CIMClient_Windows</ac:Value>
                </ac:ContextItem>
                <ac:ContextItem Name="OSVersion">
                    <ac:Value>10.0.22598.1</ac:Value>
                </ac:ContextItem>
                <ac:ContextItem Name="ApplicationVersion">
                    <ac:Value>10.0.22598.1</ac:Value>
                </ac:ContextItem>
                <ac:ContextItem Name="NotInOobe">
                    <ac:Value>` + fmt.Sprintf("%t", c.notInOOBE) + `</ac:Value>
                </ac:ContextItem>
                <ac:ContextItem Name="RequestVersion">
                    <ac:Value>5.0</ac:Value>
                </ac:ContextItem>
            </ac:AdditionalContext>
        </wst:RequestSecurityToken>
    </s:Body>
</s:Envelope>
`)

	resp, err := c.request(microsoft_mdm.MDE2EnrollPath, enrollReq)
	if err != nil {
		return err
	}

	// Check for SOAP fault
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if strings.Contains(string(body), "s:fault") {
		return fmt.Errorf("enroll request returned SOAP fault: %s", string(body))
	}

	var soapResponse fleet.SoapResponse
	if err := xml.Unmarshal(body, &soapResponse); err != nil {
		return fmt.Errorf("unmarshalling enroll response body: %w", err)
	}

	decoded, err := base64.StdEncoding.DecodeString(soapResponse.Body.RequestSecurityTokenResponseCollection.RequestSecurityTokenResponse.RequestedSecurityToken.BinarySecurityToken.Content)
	if err != nil {
		return fmt.Errorf("decoding enroll response binary security token: %w", err)
	}

	// strip xml header
	decoded = bytes.TrimPrefix(decoded, []byte(xml.Header))
	var provDoc fleet.WapProvisioningDoc
	if err := xml.Unmarshal(decoded, &provDoc); err != nil {
		return fmt.Errorf("unmarshalling enroll response provisioning doc: %w", err)
	}

Outer:
	for _, char := range provDoc.Characteristics {
		if char.Type != "APPLICATION" {
			continue
		}

		for _, appChar := range char.Characteristics {
			if appChar.Type != "APPAUTH" {
				continue
			}
			username := ""
			password := ""
			for _, appAuthParam := range appChar.Params {
				if appAuthParam.Name == "AAUTHNAME" {
					username = appAuthParam.Value
				}
				if appAuthParam.Name == "AAUTHSECRET" {
					password = appAuthParam.Value
				}
			}

			// We can do this since only the client credentials characteristic has both username and password, the other one only has password.
			if username != "" && password != "" {
				c.username = username
				c.password = password
				break Outer
			}
		}
	}

	return nil
}

func (c *TestWindowsMDMClient) Discovery() error {
	discoveryReq := []byte(`
<s:Envelope
    xmlns:a="http://www.w3.org/2005/08/addressing"
    xmlns:s="http://www.w3.org/2003/05/soap-envelope">
    <s:Header>
        <a:Action s:mustUnderstand="1">http://schemas.microsoft.com/windows/management/2012/01/enrollment/IDiscoveryService/Discover</a:Action>
        <a:MessageID>urn:uuid:748132ec-a575-4329-b01b-6171a9cf8478</a:MessageID>
        <a:ReplyTo>
            <a:Address>http://www.w3.org/2005/08/addressing/anonymous</a:Address>
        </a:ReplyTo>
        <a:To s:mustUnderstand="1">` + c.fleetServerURL + microsoft_mdm.MDE2DiscoveryPath + `</a:To>
    </s:Header>
    <s:Body>
        <Discover
            xmlns="http://schemas.microsoft.com/windows/management/2012/01/enrollment">
            <request
                xmlns:i="http://www.w3.org/2001/XMLSchema-instance">
                <EmailAddress></EmailAddress>
                <RequestVersion>5.0</RequestVersion>
                <DeviceType>CIMClient_Windows</DeviceType>
                <ApplicationVersion>6.2.9200.1</ApplicationVersion>
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

	// TODO: parse the response and store the policy and enroll endpoints instead
	// of hardcoding them to truly test that the server is behaving as expected.
	resp, err := c.request(microsoft_mdm.MDE2DiscoveryPath, discoveryReq)
	if err != nil {
		return err
	}

	// Check for SOAP fault
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if strings.Contains(string(body), "s:fault") {
		return fmt.Errorf("discovery request returned SOAP fault: %s", string(body))
	}

	return nil
}

func (c *TestWindowsMDMClient) Policy() error {
	binarySecToken, tokenValueType, err := c.getToken()
	if err != nil {
		return fmt.Errorf("getting binary token: %w", err)
	}

	policyReq := []byte(`
<s:Envelope
    xmlns:s="http://www.w3.org/2003/05/soap-envelope"
    xmlns:a="http://www.w3.org/2005/08/addressing"
    xmlns:u="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd"
    xmlns:wsse="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd"
    xmlns:wst="http://docs.oasis-open.org/ws-sx/ws-trust/200512"
    xmlns:ac="http://schemas.xmlsoap.org/ws/2006/12/authorization">
    <s:Header>
        <a:Action s:mustUnderstand="1">http://schemas.microsoft.com/windows/pki/2009/01/enrollmentpolicy/IPolicy/GetPolicies</a:Action>
        <a:MessageID>urn:uuid:72048B64-0F19-448F-8C2E-B4C661860AA0</a:MessageID>
        <a:ReplyTo>
            <a:Address>http://www.w3.org/2005/08/addressing/anonymous</a:Address>
        </a:ReplyTo>
        <a:To s:mustUnderstand="1">` + c.fleetServerURL + microsoft_mdm.MDE2PolicyPath + `</a:To>
        <wsse:Security s:mustUnderstand="1">
            <wsse:BinarySecurityToken ValueType="` + tokenValueType + `" EncodingType="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd#base64binary">` + binarySecToken + `</wsse:BinarySecurityToken>
        </wsse:Security>
    </s:Header>
    <s:Body
        xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
        xmlns:xsd="http://www.w3.org/2001/XMLSchema">
        <GetPolicies
            xmlns="http://schemas.microsoft.com/windows/pki/2009/01/enrollmentpolicy">
            <client>
                <lastUpdate xsi:nil="true"/>
                <preferredLanguage xsi:nil="true"/>
                <TPMManufacturer>IBM</TPMManufacturer>
                <TPMFirmwareVersion>8217.4131.22.13878</TPMFirmwareVersion>
            </client>
            <requestFilter xsi:nil="true"/>
        </GetPolicies>
    </s:Body>
</s:Envelope>
  `)

	// TODO: store the policy requirements to generate a certificate and generate
	// one on the fly using them instead of using hardcoded values.
	resp, err := c.request(microsoft_mdm.MDE2PolicyPath, policyReq)
	if err != nil {
		return err
	}
	// Check for SOAP fault
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if strings.Contains(string(body), "s:fault") {
		return fmt.Errorf("policy request returned SOAP fault: %s", string(body))
	}

	return nil
}

func (c *TestWindowsMDMClient) request(path string, reqBody []byte) (*http.Response, error) {
	request, err := http.NewRequest("POST", c.fleetServerURL+path, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	cc := fleethttp.NewClient(fleethttp.WithTLSClientConfig(&tls.Config{
		// Ignoring "G402: TLS InsecureSkipVerify set true", this is only used for automated testing.
		InsecureSkipVerify: true, //nolint:gosec
	}))
	response, err := cc.Do(request)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request error: %d, %s", response.StatusCode, response.Status)
	}

	return response, nil
}

func (c *TestWindowsMDMClient) getToken() (binarySecToken string, tokenValueType string, err error) {
	switch c.enrollmentType {
	case fleet.WindowsMDMAutomaticEnrollmentType:
		claims := &jwt.MapClaims{
			"upn":         c.TokenIdentifier,
			"tid":         "tenant_id",
			"unique_name": "foo_bar",
			"scp":         "mdm_delegation",
		}
		if c.jwtSigningKey == nil || c.jwtSigningKeyID == "" {
			return "", "", errors.New("jwt signing key is not set")
		}

		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		token.Header["kid"] = c.jwtSigningKeyID
		tokenString, err := token.SignedString(c.jwtSigningKey)
		if err != nil {
			return "", "", err
		}

		tokenValueType = syncml.BinarySecurityAzureEnroll
		binarySecToken = base64.URLEncoding.EncodeToString([]byte(tokenString))

	case fleet.WindowsMDMProgrammaticEnrollmentType:
		var err error
		tokenValueType = syncml.BinarySecurityDeviceEnroll
		binarySecToken, err = fleet.GetEncodedBinarySecurityToken(c.enrollmentType, c.TokenIdentifier)
		if err != nil {
			return "", "", fmt.Errorf("generating encoded security token: %w", err)
		}

	case WindowsMDMEmptyBinarySecurityTokenEnrollmentType:
		tokenValueType = syncml.BinarySecurityAzureEnroll
		binarySecToken = ""
	}

	return binarySecToken, tokenValueType, nil
}

func (c *TestWindowsMDMClient) Unenroll() error {
	unenrollRequest := []byte(`
			 <SyncML xmlns="SYNCML:SYNCML1.2">
			<SyncHdr>
				<VerDTD>1.2</VerDTD>
				<VerProto>DM/1.2</VerProto>
				<SessionID>2</SessionID>
				<MsgID>1</MsgID>
				<Target>
				<LocURI>` + c.fleetServerURL + microsoft_mdm.MDE2ManagementPath + `</LocURI>
				</Target>
				<Source>
				<LocURI>` + c.DeviceID + `</LocURI>
				</Source>
			</SyncHdr>
			<SyncBody>
				<Alert>
				<CmdID>4</CmdID>
				<Data>1226</Data>
				<Item>
					<Meta>
					<Type xmlns="syncml:metinf">com.microsoft:mdm.unenrollment.userrequest</Type>
					<Format xmlns="syncml:metinf">int</Format>
					</Meta>
					<Data>1</Data>
				</Item>
				</Alert>
				<Final/>
			</SyncBody>
			</SyncML>`)

	_, err := c.doManagementReq(unenrollRequest)
	return err
}

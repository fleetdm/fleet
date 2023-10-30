package mdmtest

import (
	"bytes"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/fleet"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/uuid"
)

type TestWindowsMDMClient struct {
	deviceID           string
	hardwareID         string
	fleetServerURL     string
	debug              bool
	enrollmentType     fleet.WindowsMDMEnrollmentType
	orbitNodeKey       string
	lastManagementResp *fleet.SyncML
	responses          map[string]fleet.ProtoCmdOperation
}

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

func NewTestMDMClientWindowsProgramatic(serverURL string, orbitNodeKey string, opts ...TestWindowsMDMClientOption) *TestWindowsMDMClient {
	c := TestWindowsMDMClient{
		fleetServerURL: serverURL,
		deviceID:       uuid.NewString(),
		enrollmentType: fleet.WindowsMDMProgrammaticEnrollmentType,
		orbitNodeKey:   orbitNodeKey,
		hardwareID:     uuid.NewString(),
	}
	for _, fn := range opts {
		fn(&c)
	}
	return &c
}

func (c *TestWindowsMDMClient) StartManagementSession() error {
	managementReq := []byte(`
<SyncML xmlns="SYNCML:SYNCML1.2">
<SyncHdr>
	<VerDTD>1.2</VerDTD>
	<VerProto>DM/1.2</VerProto>
	<SessionID>1</SessionID>
	<MsgID>1</MsgID>
	<Target>
	<LocURI>` + c.fleetServerURL + microsoft_mdm.MDE2ManagementPath + `</LocURI>
	</Target>
	<Source>
	<LocURI>` + c.deviceID + `</LocURI>
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
		<Data>` + c.deviceID + `</Data>
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

	// TODO: this request works because we're allowing devices without
	// certificates to communicate with the server. We will need to include the
	// certificate we generated during enrollment when we fix that.
	managementResp, err := c.request(microsoft_mdm.MDE2ManagementPath, managementReq)
	if err != nil {
		return err
	}

	rawXML, err := io.ReadAll(managementResp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	var syncML fleet.SyncML
	if err := xml.Unmarshal(rawXML, &syncML); err != nil {
		return fmt.Errorf("unmarshalling response body: %w", err)
	}
	c.lastManagementResp = &syncML

	return nil
}

func (c *TestWindowsMDMClient) Respond() error {
	// Get SessionID
	sessionID, err := c.lastManagementResp.GetSessionID()
	if err != nil {
		return fmt.Errorf("session ID processing error %w", err)
	}

	// Get MessageID
	messageID, err := c.lastManagementResp.GetMessageID()
	if err != nil {
		return fmt.Errorf("message ID processing error %w", err)
	}

	var msg fleet.SyncML
	msg.Xmlns = microsoft_mdm.SyncCmdNamespace
	msg.SyncHdr = fleet.SyncHdr{
		VerDTD:    microsoft_mdm.SyncMLSupportedVersion,
		VerProto:  microsoft_mdm.SyncMLVerProto,
		SessionID: sessionID,
		MsgID:     messageID,
		Target:    &fleet.LocURI{LocURI: &c.deviceID},
		Source: &fleet.LocURI{
			LocURI: ptr.String(c.fleetServerURL + microsoft_mdm.MDE2ManagementPath),
		},
	}

	// iterate over operations and append them to the SyncML message
	for _, protoCmd := range c.responses {
		msg.AppendCommand(fleet.MDMRaw, protoCmd)
	}
	return nil
}

// SetResponse sets a response for a specific command UUID.
func (d *TestWindowsMDMClient) SetResponse(commandUUID string, op fleet.ProtoCmdOperation) {
	d.responses[commandUUID] = op
}

func (c *TestWindowsMDMClient) Enroll() error {
	if err := c.Discovery(); err != nil {
		return err
	}

	if err := c.Policy(); err != nil {
		return err
	}

	binarySecToken, err := fleet.GetEncodedBinarySecurityToken(c.enrollmentType, c.orbitNodeKey)
	if err != nil {
		return fmt.Errorf("generating encoded security token: %w", err)
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
        <a:To s:mustUnderstand="1">https://roperzh-fleet.ngrok.io/api/mdm/microsoft/enroll</a:To>
        <wsse:Security s:mustUnderstand="1">
            <wsse:BinarySecurityToken ValueType="http://schemas.microsoft.com/5.0.0.0/ConfigurationManager/Enrollment/DeviceEnrollmentUserToken" EncodingType="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd#base64binary">` + binarySecToken + `</wsse:BinarySecurityToken>
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
                    <ac:Value>` + c.hardwareID + `</ac:Value>
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
                    <ac:Value>` + c.deviceID + `</ac:Value>
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

	if _, err := c.request(microsoft_mdm.MDE2EnrollPath, enrollReq); err != nil {
		return err
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
	if _, err := c.request(microsoft_mdm.MDE2DiscoveryPath, discoveryReq); err != nil {
		return err
	}

	return nil
}

func (c *TestWindowsMDMClient) Policy() error {
	binarySecToken, err := fleet.GetEncodedBinarySecurityToken(c.enrollmentType, c.orbitNodeKey)
	fmt.Println(binarySecToken)
	if err != nil {
		return fmt.Errorf("generating encoded security token: %w", err)
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
            <wsse:BinarySecurityToken ValueType="http://schemas.microsoft.com/5.0.0.0/ConfigurationManager/Enrollment/DeviceEnrollmentUserToken" EncodingType="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd#base64binary">` + binarySecToken + `</wsse:BinarySecurityToken>
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
	if _, err := c.request(microsoft_mdm.MDE2PolicyPath, policyReq); err != nil {
		return err
	}

	return nil
}

func (c *TestWindowsMDMClient) request(path string, reqBody []byte) (*http.Response, error) {
	request, err := http.NewRequest("POST", c.fleetServerURL+path, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	cc := fleethttp.NewClient(fleethttp.WithTLSClientConfig(&tls.Config{
		InsecureSkipVerify: true,
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

//func (c *TestWindowsMDMClient) InitiateSession() error {
//	req := []byte(`
//			 <SyncML xmlns="SYNCML:SYNCML1.2">
//			<SyncHdr>
//				<VerDTD>1.2</VerDTD>
//				<VerProto>DM/1.2</VerProto>
//				<SessionID>1</SessionID>
//				<MsgID>1</MsgID>
//				<Target>
//				<LocURI>` + c.managementURL + `</LocURI>
//				</Target>
//				<Source>
//				<LocURI>` + c.deviceID + `</LocURI>
//				</Source>
//			</SyncHdr>
//			<SyncBody>
//				<Alert>
//				<CmdID>2</CmdID>
//				<Data>1201</Data>
//				</Alert>
//				<Alert>
//				<CmdID>3</CmdID>
//				<Data>1224</Data>
//				<Item>
//					<Meta>
//					<Type xmlns="syncml:metinf">com.microsoft/MDM/LoginStatus</Type>
//					</Meta>
//					<Data>user</Data>
//				</Item>
//				</Alert>
//				<Replace>
//				<CmdID>4</CmdID>
//				<Item>
//					<Source>
//					<LocURI>./DevInfo/DevId</LocURI>
//					</Source>
//					<Data>` + c.deviceID + `</Data>
//				</Item>
//				<Item>
//					<Source>
//					<LocURI>./DevInfo/Man</LocURI>
//					</Source>
//					<Data>VMware, Inc.</Data>
//				</Item>
//				<Item>
//					<Source>
//					<LocURI>./DevInfo/Mod</LocURI>
//					</Source>
//					<Data>VMware7,1</Data>
//				</Item>
//				<Item>
//					<Source>
//					<LocURI>./DevInfo/DmV</LocURI>
//					</Source>
//					<Data>1.3</Data>
//				</Item>
//				<Item>
//					<Source>
//					<LocURI>./DevInfo/Lang</LocURI>
//					</Source>
//					<Data>en-US</Data>
//				</Item>
//				</Replace>
//				<Final/>
//			</SyncBody>
//			</SyncML>`)
//
//	fmt.Println(req)
//	return nil
//}

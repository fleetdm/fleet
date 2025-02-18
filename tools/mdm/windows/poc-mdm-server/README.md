
# Windows MDM Server Demo

This project is a working and minimal implementation of the Windows device enrollment and management protocols. It was based on an initial implementation of the MS-MDE enrollment protocols [here](https://github.com/oscartbeaumont/windows_mdm).

This project uses the protocols:

- [MS-MDE](https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-mde/d9e18701-cd4c-4fdb-8a3e-c1ddd33b1307)
- [MS-MDM](https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-mdm/33769a92-ac31-47ef-ae7b-dc8501f7104f)
- [MS-WSTEP](https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-wstep/4766a85d-0d18-4fa1-a51f-e5cb98b752ea)
- [MS-XCEP](https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-xcep/08ec4475-32c2-457d-8c27-5a176660a210)
- [OMA Device Management Protocol](https://www.openmobilealliance.org/release/DM/V1_2_1-20080617-A/OMA-TS-DM_Protocol-V1_2_1-20080617-A.pdf)


The steps for MDE device enrollment correspond to five phases as shown in the following diagram:

![enter image description here](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-mde/ms-mde_files/image002.png)

## License

This code is MIT licensed and it was forked from [here](https://github.com/oscartbeaumont/windows_mdm). Initial implementation credit goes to [Oscar Beaumont](https://github.com/oscartbeaumont).

## Usage

On the server side, you just need to run the project using the already provided cert and keys. The certificate is in `.pfx` file format, so you need to extract the certificate and key first, see https://stackoverflow.com/a/59120388/1094941.
The "Import password" is "testpassword", and the names of the output files matter, on Linux something like this works (assuming you are in the certs/ directory):

```
# for the cert
$ openssl pkcs12 -in dev_cert_mdmwindows_com.pfx -clcerts -nokeys -out dev_cert_mdmwindows_com_cert.pem

# for the key
$ openssl pkcs12 -in dev_cert_mdmwindows_com.pfx -out dev_cert_mdmwindows_com.key -nocerts -nodes
```

Note that an asn1 error might occur when running the server, if that's the case you need to patch your local Go toolchain by running `$ go run ./patch/patch.go` (`GOROOT` env var must be set to point to your `go env GOROOT` directory). It may require `sudo` depending on where your `go` installation is (due to https://github.com/golang/go/issues/14017).

Next go to the project folder and run.

```bash
go run .
```

Note that the server binds to the standard and usually firewall-protected `443` port, so you may need to configure your firewall to allow connections to it for the duration of your test.

On the Windows client side, you need to import the custom CA certificate to the certificate store, and populate the `hosts` file before running the Windows Enrollment. The certificate to import is on the certs directory and it is called `dev_cert_mdmwindows_com.pfx`. You need to copy this certificate to the client machine and run the powershell command below (in the console, not in a powershell terminal). This is required because the project uses a local dev https endpoint.

    1) Import certificate to Trusted CAs repository (be sure to update the path to the pfx certificate)

    powershell -ep bypass "$mypwd = ConvertTo-SecureString -String 'testpassword' -Force -AsPlainText ; Import-PfxCertificate -FilePath c:\path\to\dev_cert_mdmwindows_com.pfx -CertStoreLocation Cert:\LocalMachine\Root -Password $mypwd"

    2) Add mdmwindows.com to the list of static DNS

    echo <server_ip> mdmwindows.com >> %SystemRoot%\System32\drivers\etc\hosts
    echo <server_ip> autodiscovery.mdmwindows.com >> %SystemRoot%\System32\drivers\etc\hosts
    echo <server_ip> enterpriseenrollment.mdmwindows.com >> %SystemRoot%\System32\drivers\etc\hosts

To enroll the device into this MDM server, go to `Settings > Accounts > Access work or school` and click the connect button, enter the email provided to the server when you ran `go run .` (default: `demo@mdmwindows.com`) and it should automatically detect the server and proceed with enrollment. This is why the server must run on port `:443`, because it uses automatic discovery and will not attempt a custom port.

## Protocol Details

Below is the raw https exchange of the MS-MDE and MS-MDM protocols when run using the -verbose mode:


### MDM Server HTTP Endpoints Auto Discovery Flow


    ============================= Input Request =============================
    ----------- Input Header -----------
     GET /EnrollmentServer/Discovery.svc HTTP/2.0
    Host: enterpriseenrollment.mdmwindows.com
    Cache-Control: no-cache
    Pragma: no-cache
    User-Agent: ENROLLClient


    ----------- Empty Input Body -----------
    =========================================================================



    ============================= Output Response =============================
    ----------- Response Header -----------
     HTTP/1.1 200 OK
    Connection: close


    ----------- Empty Response Body -----------
    =========================================================================

    ============================= Input Request =============================
    ----------- Input Header -----------
     POST /EnrollmentServer/Discovery.svc HTTP/2.0
    Host: enterpriseenrollment.mdmwindows.com
    Content-Length: 1042
    Content-Type: application/soap+xml; charset=utf-8
    User-Agent: ENROLLClient


    ----------- Input Body -----------

            <s:Envelope xmlns:a="http://www.w3.org/2005/08/addressing" xmlns:s="http://www.w3.org/2003/05/soap-envelope">
              <s:Header>
                <a:Action s:mustUnderstand="1">http://schemas.microsoft.com/windows/management/2012/01/enrollment/IDiscoveryService/Discover</a:Action>
                <a:MessageID>urn:uuid:748132ec-a575-4329-b01b-6171a9cf8478</a:MessageID>
                <a:ReplyTo>
                  <a:Address>http://www.w3.org/2005/08/addressing/anonymous</a:Address>
                </a:ReplyTo>
                <a:To s:mustUnderstand="1">https://EnterpriseEnrollment.mdmwindows.com:443/EnrollmentServer/Discovery.svc</a:To>
              </s:Header>
              <s:Body>
                <Discover xmlns="http://schemas.microsoft.com/windows/management/2012/01/enrollment">
                  <request xmlns:i="http://www.w3.org/2001/XMLSchema-instance">
                    <EmailAddress>demo@mdmwindows.com</EmailAddress>
                    <RequestVersion>4.0</RequestVersion>
                    <DeviceType>CIMClient_Windows</DeviceType>
                    <ApplicationVersion>10.0.19043.2364</ApplicationVersion>
                    <OSEdition>72</OSEdition>
                    <AuthPolicies>
                      <AuthPolicy>OnPremise</AuthPolicy>
                      <AuthPolicy>Federated</AuthPolicy>
                    </AuthPolicies>
                  </request>
                </Discover>
              </s:Body>
            </s:Envelope>
    =========================================================================




    ============================= Output Response =============================
    ----------- Response Header -----------
     HTTP/1.1 200 OK
    Content-Length: 1107
    Content-Type: application/soap+xml; charset=utf-8


    ----------- Response Body -----------


            <s:Envelope
                            xmlns:s="http://www.w3.org/2003/05/soap-envelope"
                            xmlns:a="http://www.w3.org/2005/08/addressing">
              <s:Header>
                <a:Action s:mustUnderstand="1">http://schemas.microsoft.com/windows/management/2012/01/enrollment/IDiscoveryService/DiscoverResponse</a:Action>
                <ActivityId CorrelationId="8c6060c4-3d78-4d73-ae17-e8bce88426ee"
                                            xmlns="http://schemas.microsoft.com/2004/09/ServiceModel/Diagnostics">8c6060c4-3d78-4d73-ae17-e8bce88426ee
                                    </ActivityId>
                <a:RelatesTo>urn:uuid:748132ec-a575-4329-b01b-6171a9cf8478</a:RelatesTo>
              </s:Header>
              <s:Body>
                <DiscoverResponse
                                            xmlns="http://schemas.microsoft.com/windows/management/2012/01/enrollment">
                  <DiscoverResult>
                    <AuthPolicy>OnPremise</AuthPolicy>
                    <EnrollmentVersion>4.0</EnrollmentVersion>
                    <EnrollmentPolicyServiceUrl>https://mdmwindows.com/EnrollmentServer/Policy.svc</EnrollmentPolicyServiceUrl>
                    <EnrollmentServiceUrl>https://mdmwindows.com/EnrollmentServer/Enrollment.svc</EnrollmentServiceUrl>
                  </DiscoverResult>
                </DiscoverResponse>
              </s:Body>
            </s:Envelope>
    =========================================================================

## MDM Certificate Enrollment Policy Flow (MS-XCEP)


============================= Input Request =============================
----------- Input Header -----------
 POST /EnrollmentServer/Policy.svc HTTP/2.0
Host: mdmwindows.com
Content-Length: 1495
Content-Type: application/soap+xml; charset=utf-8
User-Agent: ENROLLClient


----------- Input Body -----------

        <s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope" xmlns:a="http://www.w3.org/2005/08/addressing" xmlns:u="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd" xmlns:wsse="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd" xmlns:wst="http://docs.oasis-open.org/ws-sx/ws-trust/200512" xmlns:ac="http://schemas.xmlsoap.org/ws/2006/12/authorization">
          <s:Header>
            <a:Action s:mustUnderstand="1">http://schemas.microsoft.com/windows/pki/2009/01/enrollmentpolicy/IPolicy/GetPolicies</a:Action>
            <a:MessageID>urn:uuid:72048B64-0F19-448F-8C2E-B4C661860AA0</a:MessageID>
            <a:ReplyTo>
              <a:Address>http://www.w3.org/2005/08/addressing/anonymous</a:Address>
            </a:ReplyTo>
            <a:To s:mustUnderstand="1">https://mdmwindows.com/EnrollmentServer/Policy.svc</a:To>
            <wsse:Security s:mustUnderstand="1">
              <wsse:UsernameToken u:Id="uuid-cc1ccc1f-2fba-4bcf-b063-ffc0cac77917-4">
                <wsse:Username>demo@mdmwindows.com</wsse:Username>
                <wsse:Password wsse:Type="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-username-token-profile-1.0#PasswordText">demo</wsse:Password>
              </wsse:UsernameToken>
            </wsse:Security>
          </s:Header>
          <s:Body xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema">
            <GetPolicies xmlns="http://schemas.microsoft.com/windows/pki/2009/01/enrollmentpolicy">
              <client>
                <lastUpdate xsi:nil="true"/>
                <preferredLanguage xsi:nil="true"/>
              </client>
              <requestFilter xsi:nil="true"/>
            </GetPolicies>
          </s:Body>
        </s:Envelope>
=========================================================================




============================= Output Response =============================
----------- Response Header -----------
 HTTP/1.1 200 OK
Content-Length: 1378
Content-Type: application/soap+xml; charset=utf-8


----------- Response Body -----------


        <s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope" xmlns:a="http://www.w3.org/2005/08/addressing" xmlns:u="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd">
          <s:Header>
            <a:Action s:mustUnderstand="1">http://schemas.microsoft.com/windows/pki/2009/01/enrollmentpolicy/IPolicy/GetPoliciesResponse</a:Action>
            <a:RelatesTo>urn:uuid:72048B64-0F19-448F-8C2E-B4C661860AA0</a:RelatesTo>
          </s:Header>
          <s:Body
                                xmlns:xsd="http://www.w3.org/2001/XMLSchema"
                                xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
            <GetPoliciesResponse
                                        xmlns="http://schemas.microsoft.com/windows/pki/2009/01/enrollmentpolicy">
              <response>
                <policies>
                  <policy>
                    <attributes>
                      <policySchema>3</policySchema>
                      <privateKeyAttributes>
                        <minimalKeyLength>2048</minimalKeyLength>
                        <algorithmOIDReferencexsi:nil="true"/>
                      </privateKeyAttributes>
                      <hashAlgorithmOIDReference xsi:nil="true"></hashAlgorithmOIDReference>
                    </attributes>
                  </policy>
                </policies>
              </response>
              <oIDs>
                <oID>
                  <value>1.3.6.1.4.1.311.20.2</value>
                  <group>1</group>
                  <oIDReferenceID>5</oIDReferenceID>
                  <defaultName>Certificate Template Name</defaultName>
                </oID>
              </oIDs>
            </GetPoliciesResponse>
          </s:Body>
        </s:Envelope>
=========================================================================



### MDM Certificate Enrollment Extensions Flow (MS-WSTEP)


    ============================= Input Request =============================
    ----------- Input Header -----------
     POST /EnrollmentServer/Enrollment.svc HTTP/2.0
    Host: mdmwindows.com
    Content-Length: 4295
    Content-Type: application/soap+xml; charset=utf-8
    User-Agent: ENROLLClient


    ----------- Input Body -----------

            <s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope" xmlns:a="http://www.w3.org/2005/08/addressing" xmlns:u="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd" xmlns:wsse="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd" xmlns:wst="http://docs.oasis-open.org/ws-sx/ws-trust/200512" xmlns:ac="http://schemas.xmlsoap.org/ws/2006/12/authorization">
              <s:Header>
                <a:Action s:mustUnderstand="1">http://schemas.microsoft.com/windows/pki/2009/01/enrollment/RST/wstep</a:Action>
                <a:MessageID>urn:uuid:0d5a1441-5891-453b-becf-a2e5f6ea3749</a:MessageID>
                <a:ReplyTo>
                  <a:Address>http://www.w3.org/2005/08/addressing/anonymous</a:Address>
                </a:ReplyTo>
                <a:To s:mustUnderstand="1">https://mdmwindows.com/EnrollmentServer/Enrollment.svc</a:To>
                <wsse:Security s:mustUnderstand="1">
                  <wsse:UsernameToken u:Id="uuid-cc1ccc1f-2fba-4bcf-b063-ffc0cac77917-4">
                    <wsse:Username>demo@mdmwindows.com</wsse:Username>
                    <wsse:Password wsse:Type="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-username-token-profile-1.0#PasswordText">demo</wsse:Password>
                  </wsse:UsernameToken>
                </wsse:Security>
              </s:Header>
              <s:Body>
                <wst:RequestSecurityToken>
                  <wst:TokenType>http://schemas.microsoft.com/5.0.0.0/ConfigurationManager/Enrollment/DeviceEnrollmentToken</wst:TokenType>
                  <wst:RequestType>http://docs.oasis-open.org/ws-sx/ws-trust/200512/Issue</wst:RequestType>
                  <wsse:BinarySecurityToken ValueType="http://schemas.microsoft.com/windows/pki/2009/01/enrollment#PKCS10" EncodingType="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd#base64binary">MIICzjCCAboCAQAwSzFJMEcGA1UEAxNAQkE0MDVBNUQtQjFDNy00NzM3LTgxQzgtOEYzNzlBITFFMDhDNkU5NUQ4QkI4NDNCMTI3OEZGNDVCQzYwQ0M2ADCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAJiaMezO2srqSMWGm702z0XoaScezbTDbC7oLEFRTBe20cXUduHZXyR2UJvrbztQQhuzy8Fie/Y8FNOvBfs6qb+/S/iYGvK9Ju0SJaz5KLthuUK0BLj5GtAvHnEIKk1RkYZPBTMVwS8n+iJTged8C0XhOAVWk8pDsthlrLuUlURSOMji5ftN+dsygDfAWLbJc2imikdKx1sWwDNdSNDph+RjhZHWroABKBrLbmGatRMyxt+xdV/GYvd1rLl9HWZt+IIYPtMBWlnSjVnEMs6UdU7spK7FOr6lhkuQ5wXN1uelLdpj7Zl2pKL1iJgOMLn7N23dzjDzTRfjpUNL/rNvlg0CAwEAAaBCMEAGCSqGSIb3DQEJDjEzMDEwLwYKKwYBBAGCN0IBAAQhMUUwOEM2RTk1RDhCQjg0M0IxMjc4RkY0NUJDNjBDQzYAMAkGBSsOAwIdBQADggEBADYVlS6XuWXSBFjRSGQmKJmVe1a+8TQfRUVpakKKkMDlH7aqyOKZB00nL1vNNXO6xdaWb5ViyKdsTNwnXz/BhSmZaLYGS8Qi2N5HPo1XQOdAGj+Nee0R4Nun+q9b+zfNFXo8fJuNiUaOCaDrKX5pOcALRSJBF2Kv1mBxkixJNJQgWj/JPiCr76llqH06ODf9zbmofOdEYwa2XpT43mWD7gecI8zIi+N3KxJue6hL1sLHDa5nIJR1QLjr0BddJLSm5DKiHDyIGQqPFfQXgjlyPZC9X48noxizgwv8/pwkjoRCM/dvR0QyZb3jm0Ah4MWyTlnWZp6Kl9LAUWSJoJXBMK0=</wsse:BinarySecurityToken>
                  <ac:AdditionalContext xmlns="http://schemas.xmlsoap.org/ws/2006/12/authorization">
                    <ac:ContextItem Name="UXInitiated">
                      <ac:Value>true</ac:Value>
                    </ac:ContextItem>
                    <ac:ContextItem Name="HWDevID">
                      <ac:Value>3B3ED6D0EA88CBFCF37D36F90F22FE61172348C0162FC3840D6703149870CE76</ac:Value>
                    </ac:ContextItem>
                    <ac:ContextItem Name="Locale">
                      <ac:Value>en-US</ac:Value>
                    </ac:ContextItem>
                    <ac:ContextItem Name="TargetedUserLoggedIn">
                      <ac:Value>true</ac:Value>
                    </ac:ContextItem>
                    <ac:ContextItem Name="OSEdition">
                      <ac:Value>72</ac:Value>
                    </ac:ContextItem>
                    <ac:ContextItem Name="DeviceName">
                      <ac:Value>DESKTOP-28FGAI6</ac:Value>
                    </ac:ContextItem>
                    <ac:ContextItem Name="MAC">
                      <ac:Value>00-0C-29-51-60-9D</ac:Value>
                    </ac:ContextItem>
                    <ac:ContextItem Name="MAC">
                      <ac:Value>1A-77-20-52-41-53</ac:Value>
                    </ac:ContextItem>
                    <ac:ContextItem Name="MAC">
                      <ac:Value>1A-77-20-52-41-53</ac:Value>
                    </ac:ContextItem>
                    <ac:ContextItem Name="MAC">
                      <ac:Value>00-0C-29-51-60-A7</ac:Value>
                    </ac:ContextItem>
                    <ac:ContextItem Name="MAC">
                      <ac:Value>18-14-20-52-41-53</ac:Value>
                    </ac:ContextItem>
                    <ac:ContextItem Name="MAC">
                      <ac:Value>00-0C-29-51-60-93</ac:Value>
                    </ac:ContextItem>
                    <ac:ContextItem Name="DeviceID">
                      <ac:Value>1E08C6E95D8BB843B1278FF45BC60CC6</ac:Value>
                    </ac:ContextItem>
                    <ac:ContextItem Name="EnrollmentType">
                      <ac:Value>Full</ac:Value>
                    </ac:ContextItem>
                    <ac:ContextItem Name="DeviceType">
                      <ac:Value>CIMClient_Windows</ac:Value>
                    </ac:ContextItem>
                    <ac:ContextItem Name="OSVersion">
                      <ac:Value>10.0.19043.2364</ac:Value>
                    </ac:ContextItem>
                    <ac:ContextItem Name="ApplicationVersion">
                      <ac:Value>10.0.19043.2364</ac:Value>
                    </ac:ContextItem>
                    <ac:ContextItem Name="NotInOobe">
                      <ac:Value>false</ac:Value>
                    </ac:ContextItem>
                  </ac:AdditionalContext>
                </wst:RequestSecurityToken>
              </s:Body>
            </s:Envelope>
    =========================================================================




    ============================= Output Response =============================
    ----------- Response Header -----------
     HTTP/1.1 200 OK
    Content-Length: 8598
    Content-Type: application/soap+xml; charset=utf-8


    ----------- Response Body -----------


            <s:Envelope
                            xmlns:s="http://www.w3.org/2003/05/soap-envelope"
                            xmlns:a="http://www.w3.org/2005/08/addressing"
                            xmlns:u="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd">
              <s:Header>
                <a:Action s:mustUnderstand="1">http://schemas.microsoft.com/windows/pki/2009/01/enrollment/RSTRC/wstep</a:Action>
                <a:RelatesTo>urn:uuid:0d5a1441-5891-453b-becf-a2e5f6ea3749</a:RelatesTo>
                <o:Security
                                            xmlns:o="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd" s:mustUnderstand="1">
                  <u:Timestamp u:Id="_0">
                    <u:Created>2018-11-30T00:32:59.420Z</u:Created>
                    <u:Expires>2018-12-30T00:37:59.420Z</u:Expires>
                  </u:Timestamp>
                </o:Security>
              </s:Header>
              <s:Body>
                <RequestSecurityTokenResponseCollection
                                            xmlns="http://docs.oasis-open.org/ws-sx/ws-trust/200512">
                  <RequestSecurityTokenResponse>
                    <TokenType>http://schemas.microsoft.com/5.0.0.0/ConfigurationManager/Enrollment/DeviceEnrollmentToken</TokenType>
                    <DispositionMessage
                                                            xmlns="http://schemas.microsoft.com/windows/pki/2009/01/enrollment"></DispositionMessage>
                    <RequestedSecurityToken>
                      <BinarySecurityToken
                                                                    xmlns="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd" ValueType="http://schemas.microsoft.com/5.0.0.0/ConfigurationManager/Enrollment/DeviceEnrollmentProvisionDoc" EncodingType="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd#base64binary">PD94bWwgdmVyc2lvbj0iMS4wIiBlbmNvZGluZz0iVVRGLTgiPz48d2FwLXByb3Zpc2lvbmluZ2RvYyB2ZXJzaW9uPSIxLjEiPjxjaGFyYWN0ZXJpc3RpYyB0eXBlPSJDZXJ0aWZpY2F0ZVN0b3JlIj48Y2hhcmFjdGVyaXN0aWMgdHlwZT0iUm9vdCI+PGNoYXJhY3RlcmlzdGljIHR5cGU9IlN5c3RlbSI+PGNoYXJhY3RlcmlzdGljIHR5cGU9IkQ5QTg4RTA0QUYxOEE0RDM5OUNFRTYyRjJDNzE0NjlDM0FFMUU2NzUiPjxwYXJtIG5hbWU9IkVuY29kZWRDZXJ0aWZpY2F0ZSIgdmFsdWU9Ik1JSUZUakNDQXphZ0F3SUJBZ0lVQU1sQkJEYjU2bUZGVVpPaDM1TW1QVHVWNkpjd0RRWUpLb1pJaHZjTkFRRUxCUUF3UHpFWk1CY0dBMVVFQ2d3UVRXRjBkSEpoZUNCSlpHVnVkR2wwZVRFaU1DQUdBMVVFQXd3WlYybHVaRzkzY3lCTlJFMGdSR1Z0YnlCSlpHVnVkR2wwZVRBZUZ3MHlNVEF4TURNd01qUTBNREphRncweU5EQXhNRE13TWpRME1ESmFNRDh4R1RBWEJnTlZCQW9NRUUxaGRIUnlZWGdnU1dSbGJuUnBkSGt4SWpBZ0JnTlZCQU1NR1ZkcGJtUnZkM01nVFVSTklFUmxiVzhnU1dSbGJuUnBkSGt3Z2dJaU1BMEdDU3FHU0liM0RRRUJBUVVBQTRJQ0R3QXdnZ0lLQW9JQ0FRQytnNjJHaHNFR2U0WGYvNWw4MG1POEZDOHNNWTZxR0MwZEI4YXZjSlhQdVIxTjREUVpBRkhIS2pnTTFMcFk0NVB6eHhTbUQxWTBSZFF3YUpMejAvV1F6c0RBRmhQRTdCeEI1SjBSVU1ZaVg5Yk01cCsyZmlmMFhua2xCUjE2RG5vNi9aeHdsdnZtMW1TN1RQUkZNcUhGZFB5WW0wZVc4RzAxUXBkMWVhVDdKQVhEcjN1a25yeXpmTjUxN3hzaGxJSmhVYUJtTTZRWng2L3UrS3ZhWkRGWmk1akdTekVJVHFFcy8zcFU4UFcvQm1OR1pYUkNWd2NHVGJwSG9IejczVlg3VlNEb1poWTVQNXp0VzUvZ29wOVJEQ0dxU0lJck5rNGJhOTlGd1liTnJPWDVnYktQOHJJN3VEdXBLRVlTaE5xQ250VC9ETjZXTUVSVWhkYkgzVExXeWhMSzhrbmxQTWVOSG9QTjFXK0pSZUowZVk4d0JWUVBHUENxdmJnZ3lYZ1drOTRHT3ExdDhiTmljSkRXVFpaQy9nTzRlV2FyU0RlVEJoRS80TlhWTDF5YVpkVEY0TUdBa1VLN24xYkJWT0MzTFQ4dzFEWWJIc290NmRvUTNEQ1M3NGVia1d6aHNKbjRxLyswUXFZTkREaG5FRWxRYWhvSmtCNEgrWGxLNktIeE9WNlpQQlpaTVRMVHNLTDZXRjgxb1k5N2lEc3hhNTd0d1J5a04raXoyYkxwQlBTa1I4ajU2Nnhhb3U4VGo4T2t5ODZCeG42V25MWUxvWFpld0M0VkhQRFYwUDRHQXVBeXZhWXpyM3owV20ydW9FZDBBT1Eyb3dteTAwNnduWW1yWVF4NGtqUGZVakF0UExjZE9iallvTWszTzNZaE5iUWU0TUtOWDdXL1R3SURBUUFCbzBJd1FEQU9CZ05WSFE4QkFmOEVCQU1DQVFZd0R3WURWUjBUQVFIL0JBVXdBd0VCL3pBZEJnTlZIUTRFRmdRVWNsME15cjlpNjAzKytXM3BPMm5WVlZXdmNZc3dEUVlKS29aSWh2Y05BUUVMQlFBRGdnSUJBR01GQXByVmgrK3dXU0NOakl0RkF6bW5qcFRwRmZuVnpQbHZrNXJyU2xrajVTMHlYbk9hU3VOQ25kekhwdURhYzZLd1IwY0NEUVVXNjdnWEgxdUZ3ZTE0MGtOTy92ajkycFFqcUgzTmR4ek51YkE5cXBsRzFqRXN2NXVyNEpWY1NjT002RzlxY2FHUEhTbTRkRFNBazdBUWFDQnV2RUV6Qno3L2o2QTlqS0Y4RHJ4bzU2MkYxb0xIWHVjdTJIU0VuSXJxZWdadDAwbjg3WEpnUXNVTGxoMHB1ejFkRk9FYWNMZHdvM1oxTnpOOUxEamt2Q01NUi9wbFJZVUx1cGhiaEdaL3JkME0wYzdIT0k5MGMyaS82dFlXeDM2TjZiWC9LMTlzQTE1N2ZjY1piQzhFb05iYVI2RlJzUlpQN25RSGtRR204M29kT0cza2tQelJ4b3lTbStIL1ZhM0YyRVZ6VlhRUk9vRHArMktRSThJUmpRMjVwTWxDSCs1Qm5OVmpSMkZ3cHZFU0FKZ0tWZGQ4RkVPQkJPV0dKZ2xaamx3Rm1ZQnVETWE4UnZmeStEU2NMNGxCYTFPMEx1N0xwRjBpNkRyYUZHajBxS3k5SjRkc1FOaXB5elRsR3dpczF1M0E4RFNSbXphNWxzMEtlalQzaXQ5OWQva1A4L2lVam5XOVdvSDRYcVZMMHlCaDUzMExCV1F3QktBck5zenRSNzAvT01mQ0ZnbWFVOEN3VGdrU0dQNFdyK0UzVXd1QWxhQThnWERTYndmM2x4OGlnTUpmRGtPVDVxNWNrb3BNcHpCMGJrbVhVVk9YcUVCRjVwOTA2c3o1UmNzdTRkNnMwZDQ1MnVPSTJnQTBGOXVrWEFKd1A4UTVlUS9PSnBwanF1S1ByQzJnSzhRTDB1THIiIC8+PC9jaGFyYWN0ZXJpc3RpYz48L2NoYXJhY3RlcmlzdGljPjwvY2hhcmFjdGVyaXN0aWM+PGNoYXJhY3RlcmlzdGljIHR5cGU9Ik15Ij48Y2hhcmFjdGVyaXN0aWMgdHlwZT0iVXNlciI+PGNoYXJhY3RlcmlzdGljIHR5cGU9IkFEQTdCMzVBMzBGOTg2NDY1Q0U0QzE3NkQ1MjdENEI4MDg0OEVFRjQiPjxwYXJtIG5hbWU9IkVuY29kZWRDZXJ0aWZpY2F0ZSIgdmFsdWU9Ik1JSUVMVENDQWhXZ0F3SUJBZ0lCQWpBTkJna3Foa2lHOXcwQkFRVUZBREEvTVJrd0Z3WURWUVFLREJCTllYUjBjbUY0SUVsa1pXNTBhWFI1TVNJd0lBWURWUVFEREJsWGFXNWtiM2R6SUUxRVRTQkVaVzF2SUVsa1pXNTBhWFI1TUI0WERUSXlNVEl6TURFNU1UZzBNMW9YRFRJek1USXpNREU1TVRnME0xb3dLekVwTUNjR0ExVUVBeE1nTVVVd09FTTJSVGsxUkRoQ1FqZzBNMEl4TWpjNFJrWTBOVUpETmpCRFF6WXdnZ0VpTUEwR0NTcUdTSWIzRFFFQkFRVUFBNElCRHdBd2dnRUtBb0lCQVFDWW1qSHN6dHJLNmtqRmhwdTlOczlGNkdrbkhzMjB3Mnd1NkN4QlVVd1h0dEhGMUhiaDJWOGtkbENiNjI4N1VFSWJzOHZCWW52MlBCVFRyd1g3T3FtL3YwdjRtQnJ5dlNidEVpV3MrU2k3WWJsQ3RBUzQrUnJRTHg1eENDcE5VWkdHVHdVekZjRXZKL29pVTRIbmZBdEY0VGdGVnBQS1E3TFlaYXk3bEpWRVVqakk0dVg3VGZuYk1vQTN3RmkyeVhOb3BvcEhTc2RiRnNBelhValE2WWZrWTRXUjFxNkFBU2dheTI1aG1yVVRNc2Jmc1hWZnhtTDNkYXk1ZlIxbWJmaUNHRDdUQVZwWjBvMVp4RExPbEhWTzdLU3V4VHErcFlaTGtPY0Z6ZGJucFMzYVkrMlpkcVNpOVlpWURqQzUremR0M2M0dzgwMFg0NlZEUy82emI1WU5BZ01CQUFHalNEQkdNQTRHQTFVZER3RUIvd1FFQXdJSGdEQVRCZ05WSFNVRUREQUtCZ2dyQmdFRkJRY0RBakFmQmdOVkhTTUVHREFXZ0JSeVhRekt2MkxyVGY3NWJlazdhZFZWVmE5eGl6QU5CZ2txaGtpRzl3MEJBUVVGQUFPQ0FnRUFWOFlRNHErL2xXVUJVVHdEOVIwMXhmcEphQW5FUm4wSmxIdUMzU2tRY05XN3I4cC83L2tDdENjVDY5VFR2QlVzRENNZy9PdXVCODQxTGdraFhRRWtQdHZqa3l4aUVYaytkRzZiUXQ1QWpLSWtIWUtjZEY0TzA5M0NZTm9xWElpZTg4SnduRzNzUDl3S3kyQWZwNXhYTXlJRVFqdkdNYVlneUxQUDAyd21uMmp2QU5PTlpDUHVFbTV4VnlLRzltTU1ZWWFFNXVIenB5TU5JK3BzUnVKMU8zbFlBSFhMMjkwTlBiSjl2QzRnMDRDUWRhWUZzRjl6M3F2dGlEOWVCNE9PRHRabnNrZFpsVURQSE1RYWMrRjdVMVhPdHJuU3VmN0tYOXo2U3U2VEllOWFVa3hobVgxeCtmNW9QczdVSzREWHB1NWJzTGZUMDVFN1RpclA4OTErWHR1Q3BCUnJOOFVDTlZOV2J3aFFGOE1IUXkrRk5oQm43OGJLcnhpT1FDZGM1ZWVtdVlhaVBaTHNhYnFhNExadGlIODlOeEJpVmVtbDA3aDY3RjdXNE9VeVpGWUlpdVFxUzAvdjcxaWdmd3dQVUkyd0RhWGo2b3J5K3NxV3grV3JJdFVyNWxSMTdPVU4xVUpMVkNmcCtXaldCSlhEYnRueVcvWFdzd29UcHVDMXI5bHhDWEZpaFJJSGZGV2p0OWprbHJCRFF6dDYvMmphVWprR2V2SFhieWlsd3B6djlYcUczMDBjdG9mUHdwb2RjaVVFVmZtaHFzeG1NRHhabzNpODZiMGZuS3dhN3JUSmlVZWFpTkRuSFJoMzlKcVpGVVNOZElVLzFJVzlUall1NXZQOEovV2l1akFnTnNyZ1AyN2xBYTBzZUFDNFJqWUQyN0Y3S0o1WjhBTT0iIC8+PC9jaGFyYWN0ZXJpc3RpYz48Y2hhcmFjdGVyaXN0aWMgdHlwZT0iUHJpdmF0ZUtleUNvbnRhaW5lciIgLz48L2NoYXJhY3RlcmlzdGljPjwvY2hhcmFjdGVyaXN0aWM+PC9jaGFyYWN0ZXJpc3RpYz48Y2hhcmFjdGVyaXN0aWMgdHlwZT0iQVBQTElDQVRJT04iPjxwYXJtIG5hbWU9IkFQUElEIiB2YWx1ZT0idzciIC8+PHBhcm0gbmFtZT0iUFJPVklERVItSUQiIHZhbHVlPSJERU1PIE1ETSIgLz48cGFybSBuYW1lPSJOQU1FIiB2YWx1ZT0iRmxlZXRETSBEZW1vIFNlcnZlciAtIFdpbmRvd3MiIC8+PHBhcm0gbmFtZT0iQUREUiIgdmFsdWU9Imh0dHBzOi8vbWRtd2luZG93cy5jb20vTWFuYWdlbWVudFNlcnZlci9NRE0uc3ZjIiAvPjxwYXJtIG5hbWU9IlNlcnZlckxpc3QiIHZhbHVlPSJodHRwczovL21kbXdpbmRvd3MuY29tL01hbmFnZW1lbnRTZXJ2ZXIvU2VydmVyTGlzdC5zdmMiIC8+PHBhcm0gbmFtZT0iUk9MRSIgdmFsdWU9IjQyOTQ5NjcyOTUiIC8+PHBhcm0gbmFtZT0iQkFDS0NPTVBBVFJFVFJZRElTQUJMRUQiIC8+PHBhcm0gbmFtZT0iREVGQVVMVEVOQ09ESU5HIiB2YWx1ZT0iYXBwbGljYXRpb24vdm5kLnN5bmNtbC5kbSt4bWwiIC8+PGNoYXJhY3RlcmlzdGljIHR5cGU9IkFQUEFVVEgiPjxwYXJtIG5hbWU9IkFBVVRITEVWRUwiIHZhbHVlPSJDTElFTlQiIC8+PHBhcm0gbmFtZT0iQUFVVEhUWVBFIiB2YWx1ZT0iRElHRVNUIiAvPjxwYXJtIG5hbWU9IkFBVVRIU0VDUkVUIiB2YWx1ZT0iZHVtbXkiIC8+PHBhcm0gbmFtZT0iQUFVVEhEQVRBIiB2YWx1ZT0ibm9uY2UiIC8+PC9jaGFyYWN0ZXJpc3RpYz48Y2hhcmFjdGVyaXN0aWMgdHlwZT0iQVBQQVVUSCI+PHBhcm0gbmFtZT0iQUFVVEhMRVZFTCIgdmFsdWU9IkFQUFNSViIgLz48cGFybSBuYW1lPSJBQVVUSFRZUEUiIHZhbHVlPSJESUdFU1QiIC8+PHBhcm0gbmFtZT0iQUFVVEhOQU1FIiB2YWx1ZT0iZHVtbXkiIC8+PHBhcm0gbmFtZT0iQUFVVEhTRUNSRVQiIHZhbHVlPSJkdW1teSIgLz48cGFybSBuYW1lPSJBQVVUSERBVEEiIHZhbHVlPSJub25jZSIgLz48L2NoYXJhY3RlcmlzdGljPjwvY2hhcmFjdGVyaXN0aWM+PGNoYXJhY3RlcmlzdGljIHR5cGU9IkRNQ2xpZW50Ij48Y2hhcmFjdGVyaXN0aWMgdHlwZT0iUHJvdmlkZXIiPjxjaGFyYWN0ZXJpc3RpYyB0eXBlPSJERU1PIE1ETSI+PGNoYXJhY3RlcmlzdGljIHR5cGU9IlBvbGwiPjxwYXJtIG5hbWU9Ik51bWJlck9mRmlyc3RSZXRyaWVzIiB2YWx1ZT0iOCIgZGF0YXR5cGU9ImludGVnZXIiIC8+PC9jaGFyYWN0ZXJpc3RpYz48L2NoYXJhY3RlcmlzdGljPjwvY2hhcmFjdGVyaXN0aWM+PC9jaGFyYWN0ZXJpc3RpYz48L3dhcC1wcm92aXNpb25pbmdkb2M+
                                                            </BinarySecurityToken>
                    </RequestedSecurityToken>
                    <RequestID
                                                            xmlns="http://schemas.microsoft.com/windows/pki/2009/01/enrollment">0
                                                    </RequestID>
                  </RequestSecurityTokenResponse>
                </RequestSecurityTokenResponseCollection>
              </s:Body>
            </s:Envelope>
    =========================================================================




### MDM - Device Management Flow (MS-MDM)

    ============================= Input Request =============================
    ----------- Input Header -----------
     POST /ManagementServer/MDM.svc?mode=Maintenance&Platform=WoA HTTP/2.0
    Host: mdmwindows.com
    Accept: application/vnd.syncml.dm+xml, application/vnd.syncml.dm+wbxml, application/octet-stream
    Accept-Charset: UTF-8
    Client-Request-Id: 0
    Content-Length: 991
    Content-Type: application/vnd.syncml.dm+xml
    Ms-Cv: a/tCeBgffEqA5408.0.0.0
    User-Agent: MSFT OMA DM Client/1.2.0.1


    ----------- Input Body -----------

            <SyncML xmlns="SYNCML:SYNCML1.2">
              <SyncHdr>
                <VerDTD>1.2</VerDTD>
                <VerProto>DM/1.2</VerProto>
                <SessionID>1</SessionID>
                <MsgID>1</MsgID>
                <Target>
                  <LocURI>https://mdmwindows.com/ManagementServer/MDM.svc</LocURI>
                </Target>
                <Source>
                  <LocURI>1E08C6E95D8BB843B1278FF45BC60CC6</LocURI>
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
                    <Data>1E08C6E95D8BB843B1278FF45BC60CC6</Data>
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
    =========================================================================




    ============================= Output Response =============================
    ----------- Response Header -----------
     HTTP/1.1 200 OK
    Content-Length: 1736
    Content-Type: application/vnd.syncml.dm+xml


    ----------- Response Body -----------

            <?xml version="1.0" encoding="UTF-8"?>
            <SyncML xmlns="SYNCML:SYNCML1.2">
              <SyncHdr>
                <VerDTD>1.2</VerDTD>
                <VerProto>DM/1.2</VerProto>
                <SessionID>1</SessionID>
                <MsgID>1</MsgID>
                <Target>
                  <LocURI>1E08C6E95D8BB843B1278FF45BC60CC6</LocURI>
                </Target>
                <Source>
                  <LocURI>https://mdmwindows.com/ManagementServer/MDM.svc</LocURI>
                </Source>
              </SyncHdr>
              <SyncBody>
                <Status>
                  <CmdID>1</CmdID>
                  <MsgRef>1</MsgRef>
                  <CmdRef>0</CmdRef>
                  <Cmd>SyncHdr</Cmd>
                  <Data>200</Data>
                </Status>
                <Status>
                  <CmdID>2</CmdID>
                  <MsgRef>1</MsgRef>
                  <CmdRef>2</CmdRef>
                  <Cmd>Alert</Cmd>
                  <Data>200</Data>
                </Status>
                <Status>
                  <CmdID>3</CmdID>
                  <MsgRef>1</MsgRef>
                  <CmdRef>3</CmdRef>
                  <Cmd>Alert</Cmd>
                  <Data>200</Data>
                </Status>
                <Status>
                  <CmdID>4</CmdID>
                  <MsgRef>1</MsgRef>
                  <CmdRef>4</CmdRef>
                  <Cmd>Replace</Cmd>
                  <Data>200</Data>
                </Status>
                <Replace>
                  <CmdID>5</CmdID>
                  <Item>
                    <Target>
                      <LocURI>./Vendor/MSFT/Personalization/DesktopImageUrl</LocURI>
                    </Target>
                    <Meta>
                      <Format xmlns="syncml:metinf">chr</Format>
                      <Type>text/plain</Type>
                    </Meta>
                    <Data>https://fleetdm.com/images/articles/fleet-4.24.0-cover-1600x900@2x.jpg</Data>
                  </Item>
                </Replace>
                <Replace>
                  <CmdID>6</CmdID>
                  <Item>
                    <Target>
                      <LocURI>./Vendor/MSFT/Personalization/LockScreenImageUrl</LocURI>
                    </Target>
                    <Meta>
                      <Format xmlns="syncml:metinf">chr</Format>
                      <Type>text/plain</Type>
                    </Meta>
                    <Data>https://fleetdm.com/images/articles/fleet-4.24.0-cover-1600x900@2x.jpg</Data>
                  </Item>
                </Replace>
                <Final />
              </SyncBody>
            </SyncML>
    =========================================================================


    192.168.8.10 - - [30/Dec/2022:16:59:44 -0300] "POST /ManagementServer/MDM.svc?mode=Maintenance&Platform=WoA HTTP/2.0" 200 1400


    ============================= Input Request =============================
    ----------- Input Header -----------
     POST /ManagementServer/MDM.svc?mode=Maintenance&Platform=WoA HTTP/2.0
    Host: mdmwindows.com
    Accept: application/vnd.syncml.dm+xml, application/vnd.syncml.dm+wbxml, application/octet-stream
    Accept-Charset: UTF-8
    Client-Request-Id: 0
    Content-Length: 633
    Content-Type: application/vnd.syncml.dm+xml
    Ms-Cv: a/tCeBgffEqA5408.0.0.0
    User-Agent: MSFT OMA DM Client/1.2.0.1


    ----------- Input Body -----------

            <SyncML xmlns="SYNCML:SYNCML1.2">
              <SyncHdr>
                <VerDTD>1.2</VerDTD>
                <VerProto>DM/1.2</VerProto>
                <SessionID>1</SessionID>
                <MsgID>2</MsgID>
                <Target>
                  <LocURI>https://mdmwindows.com/ManagementServer/MDM.svc</LocURI>
                </Target>
                <Source>
                  <LocURI>1E08C6E95D8BB843B1278FF45BC60CC6</LocURI>
                </Source>
              </SyncHdr>
              <SyncBody>
                <Status>
                  <CmdID>1</CmdID>
                  <MsgRef>1</MsgRef>
                  <CmdRef>0</CmdRef>
                  <Cmd>SyncHdr</Cmd>
                  <Data>200</Data>
                </Status>
                <Status>
                  <CmdID>2</CmdID>
                  <MsgRef>1</MsgRef>
                  <CmdRef>5</CmdRef>
                  <Cmd>Replace</Cmd>
                  <Data>202</Data>
                </Status>
                <Status>
                  <CmdID>3</CmdID>
                  <MsgRef>1</MsgRef>
                  <CmdRef>6</CmdRef>
                  <Cmd>Replace</Cmd>
                  <Data>202</Data>
                </Status>
                <Final/>
              </SyncBody>
            </SyncML>
    =========================================================================




    ============================= Output Response =============================
    ----------- Response Header -----------
     HTTP/1.1 200 OK
    Content-Type: application/vnd.syncml.dm+xml
    Content-Length: 0


    ----------- Response Body -----------

    =========================================================================



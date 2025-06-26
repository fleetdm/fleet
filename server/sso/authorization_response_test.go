package sso

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/beevik/etree"
	"github.com/crewjam/saml"
	"github.com/fleetdm/fleet/v4/server/fleet"
	dsig "github.com/russellhaering/goxmldsig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeSuccessfulSalesforceResponse(t *testing.T) {
	const samlResponse = `<?xml version="1.0" encoding="UTF-8"?>
<samlp:Response xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" Destination="https://localhost:8080/api/v1/kolide/sso/callback" ID="_52f2515c5319f2adf3f072d9d9f2b6881493305396746" InResponseTo="4982b430-73e1-4ad2-885a-4a775a91f820" IssueInstant="2017-04-27T15:03:16.747Z" Version="2.0">
  <saml:Issuer xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" Format="urn:oasis:names:tc:SAML:2.0:nameid-format:entity">https://kolide-dev-ed.my.salesforce.com</saml:Issuer>
  <ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
    <ds:SignedInfo>
      <ds:CanonicalizationMethod Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/>
      <ds:SignatureMethod Algorithm="http://www.w3.org/2000/09/xmldsig#rsa-sha1"/>
      <ds:Reference URI="#_52f2515c5319f2adf3f072d9d9f2b6881493305396746">
        <ds:Transforms>
          <ds:Transform Algorithm="http://www.w3.org/2000/09/xmldsig#enveloped-signature"/>
          <ds:Transform Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#">
            <ec:InclusiveNamespaces xmlns:ec="http://www.w3.org/2001/10/xml-exc-c14n#" PrefixList="ds saml samlp xs xsi"/>
          </ds:Transform>
        </ds:Transforms>
        <ds:DigestMethod Algorithm="http://www.w3.org/2000/09/xmldsig#sha1"/>
        <ds:DigestValue>syKA9xe4vLJ+W1WxYrTDV8gjXdc=</ds:DigestValue>
      </ds:Reference>
    </ds:SignedInfo>
    <ds:SignatureValue>
SHXNEnRlRmTOpgfAtS14VNwAFmzR8u23rLNcr/K8Oh5e3l9LTtbL9QtvLsyYNUFoizDs4fbHfyBH
DcBD3zCEXFVnvS+3TA3SpUMH+4usdTsLkRhS1K5Ira/MK/aumR43IdMFilMcecF8J4YAblRtJIyh
KvSd1VKukUoTDv7YOMEwco4hxzL+gVrE9HzHfAv/fSyxOMXohEHLPO8QedBsX4ZKIr4ZuOPuViiJ
Au+01A8AO01gbZWuXmTKmI/WDH66tBQUcPRF2RBWwvzirpY86N4Sdv58VLdM5IMa/hhvLHMOHlGM
+kERr7KqLhMNFTTw9VnoybBmniR0ioAg2lwpZA==
</ds:SignatureValue>
    <ds:KeyInfo>
      <ds:X509Data>
        <ds:X509Certificate>MIIErDCCA5SgAwIBAgIOAVuhH3WkAAAAAB5NpvIwDQYJKoZIhvcNAQELBQAwgZAxKDAmBgNVBAMM
H1NlbGZTaWduZWRDZXJ0XzI0QXByMjAxN18xODAwNDQxGDAWBgNVBAsMDzAwRDZBMDAwMDAwMTd0
ODEXMBUGA1UECgwOU2FsZXNmb3JjZS5jb20xFjAUBgNVBAcMDVNhbiBGcmFuY2lzY28xCzAJBgNV
BAgMAkNBMQwwCgYDVQQGEwNVU0EwHhcNMTcwNDI0MTgwMDQ1WhcNMTgwNDI0MTIwMDAwWjCBkDEo
MCYGA1UEAwwfU2VsZlNpZ25lZENlcnRfMjRBcHIyMDE3XzE4MDA0NDEYMBYGA1UECwwPMDBENkEw
MDAwMDAxN3Q4MRcwFQYDVQQKDA5TYWxlc2ZvcmNlLmNvbTEWMBQGA1UEBwwNU2FuIEZyYW5jaXNj
bzELMAkGA1UECAwCQ0ExDDAKBgNVBAYTA1VTQTCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoC
ggEBAIOR7h8BF2eFOlQHhV/1S7uOBN22Jv7PDCXMz2fU0uLc+mrv9xDGj6ElfW+9dSdXaCbQzD3+
Xq4reS4pYRafJZ/27OtygXl3rpoPjSlhRiW+oYVuDcCURJpu0KuZ4I0fm5q1BDYqxcBxNPSe85OH
E3+ucmKqvPozhQgYLPCregMIomC3yyANZnLCoGfCv9TpQl6/+I182tST4WPNhVPxKxijoPU4Rh6x
Y34Ez8+Jr8KdmzmYSNe4ukkIASplpvG7rKka824Hf8zI1BWnjWLDxb5IAxgUBbdr4x8d8C3kPfTf
+3/6yC5wSOm9NSs0BA4OJNowtXZFryMzFfXzDzjl69kCAwEAAaOCAQAwgf0wHQYDVR0OBBYEFO+D
koP6qkysi9ZC74yTPuJVVg2yMA8GA1UdEwEB/wQFMAMBAf8wgcoGA1UdIwSBwjCBv4AU74OSg/qq
TKyL1kLvjJM+4lVWDbKhgZakgZMwgZAxKDAmBgNVBAMMH1NlbGZTaWduZWRDZXJ0XzI0QXByMjAx
N18xODAwNDQxGDAWBgNVBAsMDzAwRDZBMDAwMDAwMTd0ODEXMBUGA1UECgwOU2FsZXNmb3JjZS5j
b20xFjAUBgNVBAcMDVNhbiBGcmFuY2lzY28xCzAJBgNVBAgMAkNBMQwwCgYDVQQGEwNVU0GCDgFb
oR91pAAAAAAeTabyMA0GCSqGSIb3DQEBCwUAA4IBAQAVhYBv5GJvhltks2j7Zc9wdFHW7yB4/hPF
o05y0yiOf71tLjOlBucSyxtmXLPjrECJvIJwKhsAIgYXnVp7ditxfauCcxczJgfeL1/dxH/Ge8eP
kmH6SdsO71cJL8dXEzOsoF+PAVQzUhqh8zxIipntL0wwNGTD0zIVQeTSozm0KF0SsSHIfbNy279u
ReGonC61i4Ouk5AMKA7Re9fVeUs6tqM2at22h9Zaj/r/OhXoDcZhzkd8Wq0ER/UKLZA1CyJHgwOC
7REEZOuKrqgfWcYt4dGo5q6gqGHHPMv0N7s/MxqCvJCwGA8eJGvOO56I321vhWHQ6ZSJDWUqQFM/
Ze7A</ds:X509Certificate>
      </ds:X509Data>
    </ds:KeyInfo>
  </ds:Signature>
  <samlp:Status>
    <samlp:StatusCode Value="urn:oasis:names:tc:SAML:2.0:status:Success"/>
  </samlp:Status>
  <saml:Assertion xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" ID="_9ffad90ab367f32a52b749d5c4b2b7df1493305396749" IssueInstant="2017-04-27T15:03:16.749Z" Version="2.0">
    <saml:Issuer Format="urn:oasis:names:tc:SAML:2.0:nameid-format:entity">https://kolide-dev-ed.my.salesforce.com</saml:Issuer>
    <ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
      <ds:SignedInfo>
        <ds:CanonicalizationMethod Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/>
        <ds:SignatureMethod Algorithm="http://www.w3.org/2000/09/xmldsig#rsa-sha1"/>
        <ds:Reference URI="#_9ffad90ab367f32a52b749d5c4b2b7df1493305396749">
          <ds:Transforms>
            <ds:Transform Algorithm="http://www.w3.org/2000/09/xmldsig#enveloped-signature"/>
            <ds:Transform Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#">
              <ec:InclusiveNamespaces xmlns:ec="http://www.w3.org/2001/10/xml-exc-c14n#" PrefixList="ds saml xs xsi"/>
            </ds:Transform>
          </ds:Transforms>
          <ds:DigestMethod Algorithm="http://www.w3.org/2000/09/xmldsig#sha1"/>
          <ds:DigestValue>BOhmqkd//KYBmBJIZfUqgEx6iLc=</ds:DigestValue>
        </ds:Reference>
      </ds:SignedInfo>
      <ds:SignatureValue>
UaSyePoQdNc8ApcL7Ak7NhWuZY9ilmqbJDbkIFjfYoikPWpiqq0Z5DHxPVCHgRi42KC9oclXPjWh
v8acBZrMZlqn0yVaEeVwozcKYGwxh7mhnWnU2zrd4hnnfDZbwyU3pchUVNXyndPmfwnRR8wBDcID
+/uL10u6zBzGbtzvx1rG33Od8f4h+RDDOTRVX1iVKw5pbnvjrrYcY1gqI5OQKBoki1X6LhZE4qk5
77DG3U9Z3qut2GTYzupRp9nszbOv1l0jXuavy+94zZ3K3oqeLNH3ZW1fB8XG8b3nX9rFEYzto5CQ
SSQaUypAljmg9XrmfVoljUDpabRWKWi0eiEvxQ==
</ds:SignatureValue>
      <ds:KeyInfo>
        <ds:X509Data>
          <ds:X509Certificate>MIIErDCCA5SgAwIBAgIOAVuhH3WkAAAAAB5NpvIwDQYJKoZIhvcNAQELBQAwgZAxKDAmBgNVBAMM
H1NlbGZTaWduZWRDZXJ0XzI0QXByMjAxN18xODAwNDQxGDAWBgNVBAsMDzAwRDZBMDAwMDAwMTd0
ODEXMBUGA1UECgwOU2FsZXNmb3JjZS5jb20xFjAUBgNVBAcMDVNhbiBGcmFuY2lzY28xCzAJBgNV
BAgMAkNBMQwwCgYDVQQGEwNVU0EwHhcNMTcwNDI0MTgwMDQ1WhcNMTgwNDI0MTIwMDAwWjCBkDEo
MCYGA1UEAwwfU2VsZlNpZ25lZENlcnRfMjRBcHIyMDE3XzE4MDA0NDEYMBYGA1UECwwPMDBENkEw
MDAwMDAxN3Q4MRcwFQYDVQQKDA5TYWxlc2ZvcmNlLmNvbTEWMBQGA1UEBwwNU2FuIEZyYW5jaXNj
bzELMAkGA1UECAwCQ0ExDDAKBgNVBAYTA1VTQTCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoC
ggEBAIOR7h8BF2eFOlQHhV/1S7uOBN22Jv7PDCXMz2fU0uLc+mrv9xDGj6ElfW+9dSdXaCbQzD3+
Xq4reS4pYRafJZ/27OtygXl3rpoPjSlhRiW+oYVuDcCURJpu0KuZ4I0fm5q1BDYqxcBxNPSe85OH
E3+ucmKqvPozhQgYLPCregMIomC3yyANZnLCoGfCv9TpQl6/+I182tST4WPNhVPxKxijoPU4Rh6x
Y34Ez8+Jr8KdmzmYSNe4ukkIASplpvG7rKka824Hf8zI1BWnjWLDxb5IAxgUBbdr4x8d8C3kPfTf
+3/6yC5wSOm9NSs0BA4OJNowtXZFryMzFfXzDzjl69kCAwEAAaOCAQAwgf0wHQYDVR0OBBYEFO+D
koP6qkysi9ZC74yTPuJVVg2yMA8GA1UdEwEB/wQFMAMBAf8wgcoGA1UdIwSBwjCBv4AU74OSg/qq
TKyL1kLvjJM+4lVWDbKhgZakgZMwgZAxKDAmBgNVBAMMH1NlbGZTaWduZWRDZXJ0XzI0QXByMjAx
N18xODAwNDQxGDAWBgNVBAsMDzAwRDZBMDAwMDAwMTd0ODEXMBUGA1UECgwOU2FsZXNmb3JjZS5j
b20xFjAUBgNVBAcMDVNhbiBGcmFuY2lzY28xCzAJBgNVBAgMAkNBMQwwCgYDVQQGEwNVU0GCDgFb
oR91pAAAAAAeTabyMA0GCSqGSIb3DQEBCwUAA4IBAQAVhYBv5GJvhltks2j7Zc9wdFHW7yB4/hPF
o05y0yiOf71tLjOlBucSyxtmXLPjrECJvIJwKhsAIgYXnVp7ditxfauCcxczJgfeL1/dxH/Ge8eP
kmH6SdsO71cJL8dXEzOsoF+PAVQzUhqh8zxIipntL0wwNGTD0zIVQeTSozm0KF0SsSHIfbNy279u
ReGonC61i4Ouk5AMKA7Re9fVeUs6tqM2at22h9Zaj/r/OhXoDcZhzkd8Wq0ER/UKLZA1CyJHgwOC
7REEZOuKrqgfWcYt4dGo5q6gqGHHPMv0N7s/MxqCvJCwGA8eJGvOO56I321vhWHQ6ZSJDWUqQFM/
Ze7A</ds:X509Certificate>
        </ds:X509Data>
      </ds:KeyInfo>
    </ds:Signature>
    <saml:Subject>
      <saml:NameID Format="urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified">john@kolide.co</saml:NameID>
      <saml:SubjectConfirmation Method="urn:oasis:names:tc:SAML:2.0:cm:bearer">
        <saml:SubjectConfirmationData InResponseTo="4982b430-73e1-4ad2-885a-4a775a91f820" NotOnOrAfter="2017-04-27T15:08:16.760Z" Recipient="https://localhost:8080/api/v1/kolide/sso/callback"/>
      </saml:SubjectConfirmation>
    </saml:Subject>
    <saml:Conditions NotBefore="2017-04-27T15:02:46.760Z" NotOnOrAfter="2017-04-27T15:08:16.760Z">
      <saml:AudienceRestriction>
        <saml:Audience>kolide</saml:Audience>
      </saml:AudienceRestriction>
    </saml:Conditions>
    <saml:AuthnStatement AuthnInstant="2017-04-27T15:03:16.750Z">
      <saml:AuthnContext>
        <saml:AuthnContextClassRef>urn:oasis:names:tc:SAML:2.0:ac:classes:unspecified</saml:AuthnContextClassRef>
      </saml:AuthnContext>
    </saml:AuthnStatement>
    <saml:AttributeStatement>
      <saml:Attribute Name="userId" NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:unspecified">
        <saml:AttributeValue xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="xs:anyType">0056A000000Q6Rl</saml:AttributeValue>
      </saml:Attribute>
      <saml:Attribute Name="username" NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:unspecified">
        <saml:AttributeValue xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="xs:anyType">john@kolide.co</saml:AttributeValue>
      </saml:Attribute>
      <saml:Attribute Name="email" NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:unspecified">
        <saml:AttributeValue xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="xs:anyType">john@kolide.co</saml:AttributeValue>
      </saml:Attribute>
      <saml:Attribute Name="is_portal_user" NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:unspecified">
        <saml:AttributeValue xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="xs:anyType">false</saml:AttributeValue>
      </saml:Attribute>
    </saml:AttributeStatement>
  </saml:Assertion>
</samlp:Response>`

	setFakeClock(t, "2017-04-27T15:03:16.749Z")
	acsURL, err := url.Parse("https://localhost:8080/api/v1/kolide/sso/callback")
	require.NoError(t, err)
	p := dummySAMLProvider(acsURL)
	p.IDPMetadata = testSalesforceMetadata()
	p.IDPMetadata.EntityID = "https://kolide-dev-ed.my.salesforce.com"

	auth, err := ParseAndVerifySAMLResponse(p, []byte(samlResponse), "4982b430-73e1-4ad2-885a-4a775a91f820", acsURL)
	require.NoError(t, err)
	require.NotNil(t, auth)
	assert.Equal(t, "john@kolide.co", auth.UserID())
}

func TestDecodeWithCommentInName(t *testing.T) {
	t.Skip("TODO(lucas): Must be fixed on the crewjam/saml upstream, PR in review: https://github.com/crewjam/saml/pull/611")

	// Testing for vuln described at
	// https://duo.com/blog/duo-finds-saml-vulnerabilities-affecting-multiple-implementations
	// Relevant XML snippets:
	// <ds:CanonicalizationMethod Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/>
	// <saml:NameID Format="urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified">john@kol<!---->ide.co</saml:NameID>
	const samlResponse = `<?xml version="1.0" encoding="UTF-8"?>
<samlp:Response xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" Destination="https://localhost:8080/api/v1/kolide/sso/callback" ID="_52f2515c5319f2adf3f072d9d9f2b6881493305396746" InResponseTo="4982b430-73e1-4ad2-885a-4a775a91f820" IssueInstant="2017-04-27T15:03:16.747Z" Version="2.0">
  <saml:Issuer xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" Format="urn:oasis:names:tc:SAML:2.0:nameid-format:entity">https://kolide-dev-ed.my.salesforce.com</saml:Issuer>
  <ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
    <ds:SignedInfo>
      <ds:CanonicalizationMethod Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/>
      <ds:SignatureMethod Algorithm="http://www.w3.org/2000/09/xmldsig#rsa-sha1"/>
      <ds:Reference URI="#_52f2515c5319f2adf3f072d9d9f2b6881493305396746">
        <ds:Transforms>
          <ds:Transform Algorithm="http://www.w3.org/2000/09/xmldsig#enveloped-signature"/>
          <ds:Transform Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#">
            <ec:InclusiveNamespaces xmlns:ec="http://www.w3.org/2001/10/xml-exc-c14n#" PrefixList="ds saml samlp xs xsi"/>
          </ds:Transform>
        </ds:Transforms>
        <ds:DigestMethod Algorithm="http://www.w3.org/2000/09/xmldsig#sha1"/>
        <ds:DigestValue>syKA9xe4vLJ+W1WxYrTDV8gjXdc=</ds:DigestValue>
      </ds:Reference>
    </ds:SignedInfo>
    <ds:SignatureValue>
SHXNEnRlRmTOpgfAtS14VNwAFmzR8u23rLNcr/K8Oh5e3l9LTtbL9QtvLsyYNUFoizDs4fbHfyBH
DcBD3zCEXFVnvS+3TA3SpUMH+4usdTsLkRhS1K5Ira/MK/aumR43IdMFilMcecF8J4YAblRtJIyh
KvSd1VKukUoTDv7YOMEwco4hxzL+gVrE9HzHfAv/fSyxOMXohEHLPO8QedBsX4ZKIr4ZuOPuViiJ
Au+01A8AO01gbZWuXmTKmI/WDH66tBQUcPRF2RBWwvzirpY86N4Sdv58VLdM5IMa/hhvLHMOHlGM
+kERr7KqLhMNFTTw9VnoybBmniR0ioAg2lwpZA==
</ds:SignatureValue>
    <ds:KeyInfo>
      <ds:X509Data>
        <ds:X509Certificate>MIIErDCCA5SgAwIBAgIOAVuhH3WkAAAAAB5NpvIwDQYJKoZIhvcNAQELBQAwgZAxKDAmBgNVBAMM
H1NlbGZTaWduZWRDZXJ0XzI0QXByMjAxN18xODAwNDQxGDAWBgNVBAsMDzAwRDZBMDAwMDAwMTd0
ODEXMBUGA1UECgwOU2FsZXNmb3JjZS5jb20xFjAUBgNVBAcMDVNhbiBGcmFuY2lzY28xCzAJBgNV
BAgMAkNBMQwwCgYDVQQGEwNVU0EwHhcNMTcwNDI0MTgwMDQ1WhcNMTgwNDI0MTIwMDAwWjCBkDEo
MCYGA1UEAwwfU2VsZlNpZ25lZENlcnRfMjRBcHIyMDE3XzE4MDA0NDEYMBYGA1UECwwPMDBENkEw
MDAwMDAxN3Q4MRcwFQYDVQQKDA5TYWxlc2ZvcmNlLmNvbTEWMBQGA1UEBwwNU2FuIEZyYW5jaXNj
bzELMAkGA1UECAwCQ0ExDDAKBgNVBAYTA1VTQTCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoC
ggEBAIOR7h8BF2eFOlQHhV/1S7uOBN22Jv7PDCXMz2fU0uLc+mrv9xDGj6ElfW+9dSdXaCbQzD3+
Xq4reS4pYRafJZ/27OtygXl3rpoPjSlhRiW+oYVuDcCURJpu0KuZ4I0fm5q1BDYqxcBxNPSe85OH
E3+ucmKqvPozhQgYLPCregMIomC3yyANZnLCoGfCv9TpQl6/+I182tST4WPNhVPxKxijoPU4Rh6x
Y34Ez8+Jr8KdmzmYSNe4ukkIASplpvG7rKka824Hf8zI1BWnjWLDxb5IAxgUBbdr4x8d8C3kPfTf
+3/6yC5wSOm9NSs0BA4OJNowtXZFryMzFfXzDzjl69kCAwEAAaOCAQAwgf0wHQYDVR0OBBYEFO+D
koP6qkysi9ZC74yTPuJVVg2yMA8GA1UdEwEB/wQFMAMBAf8wgcoGA1UdIwSBwjCBv4AU74OSg/qq
TKyL1kLvjJM+4lVWDbKhgZakgZMwgZAxKDAmBgNVBAMMH1NlbGZTaWduZWRDZXJ0XzI0QXByMjAx
N18xODAwNDQxGDAWBgNVBAsMDzAwRDZBMDAwMDAwMTd0ODEXMBUGA1UECgwOU2FsZXNmb3JjZS5j
b20xFjAUBgNVBAcMDVNhbiBGcmFuY2lzY28xCzAJBgNVBAgMAkNBMQwwCgYDVQQGEwNVU0GCDgFb
oR91pAAAAAAeTabyMA0GCSqGSIb3DQEBCwUAA4IBAQAVhYBv5GJvhltks2j7Zc9wdFHW7yB4/hPF
o05y0yiOf71tLjOlBucSyxtmXLPjrECJvIJwKhsAIgYXnVp7ditxfauCcxczJgfeL1/dxH/Ge8eP
kmH6SdsO71cJL8dXEzOsoF+PAVQzUhqh8zxIipntL0wwNGTD0zIVQeTSozm0KF0SsSHIfbNy279u
ReGonC61i4Ouk5AMKA7Re9fVeUs6tqM2at22h9Zaj/r/OhXoDcZhzkd8Wq0ER/UKLZA1CyJHgwOC
7REEZOuKrqgfWcYt4dGo5q6gqGHHPMv0N7s/MxqCvJCwGA8eJGvOO56I321vhWHQ6ZSJDWUqQFM/
Ze7A</ds:X509Certificate>
      </ds:X509Data>
    </ds:KeyInfo>
  </ds:Signature>
  <samlp:Status>
    <samlp:StatusCode Value="urn:oasis:names:tc:SAML:2.0:status:Success"/>
  </samlp:Status>
  <saml:Assertion xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" ID="_9ffad90ab367f32a52b749d5c4b2b7df1493305396749" IssueInstant="2017-04-27T15:03:16.749Z" Version="2.0">
    <saml:Issuer Format="urn:oasis:names:tc:SAML:2.0:nameid-format:entity">https://kolide-dev-ed.my.salesforce.com</saml:Issuer>
    <ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
      <ds:SignedInfo>
        <ds:CanonicalizationMethod Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/>
        <ds:SignatureMethod Algorithm="http://www.w3.org/2000/09/xmldsig#rsa-sha1"/>
        <ds:Reference URI="#_9ffad90ab367f32a52b749d5c4b2b7df1493305396749">
          <ds:Transforms>
            <ds:Transform Algorithm="http://www.w3.org/2000/09/xmldsig#enveloped-signature"/>
            <ds:Transform Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#">
              <ec:InclusiveNamespaces xmlns:ec="http://www.w3.org/2001/10/xml-exc-c14n#" PrefixList="ds saml xs xsi"/>
            </ds:Transform>
          </ds:Transforms>
          <ds:DigestMethod Algorithm="http://www.w3.org/2000/09/xmldsig#sha1"/>
          <ds:DigestValue>BOhmqkd//KYBmBJIZfUqgEx6iLc=</ds:DigestValue>
        </ds:Reference>
      </ds:SignedInfo>
      <ds:SignatureValue>
UaSyePoQdNc8ApcL7Ak7NhWuZY9ilmqbJDbkIFjfYoikPWpiqq0Z5DHxPVCHgRi42KC9oclXPjWh
v8acBZrMZlqn0yVaEeVwozcKYGwxh7mhnWnU2zrd4hnnfDZbwyU3pchUVNXyndPmfwnRR8wBDcID
+/uL10u6zBzGbtzvx1rG33Od8f4h+RDDOTRVX1iVKw5pbnvjrrYcY1gqI5OQKBoki1X6LhZE4qk5
77DG3U9Z3qut2GTYzupRp9nszbOv1l0jXuavy+94zZ3K3oqeLNH3ZW1fB8XG8b3nX9rFEYzto5CQ
SSQaUypAljmg9XrmfVoljUDpabRWKWi0eiEvxQ==
</ds:SignatureValue>
      <ds:KeyInfo>
        <ds:X509Data>
          <ds:X509Certificate>MIIErDCCA5SgAwIBAgIOAVuhH3WkAAAAAB5NpvIwDQYJKoZIhvcNAQELBQAwgZAxKDAmBgNVBAMM
H1NlbGZTaWduZWRDZXJ0XzI0QXByMjAxN18xODAwNDQxGDAWBgNVBAsMDzAwRDZBMDAwMDAwMTd0
ODEXMBUGA1UECgwOU2FsZXNmb3JjZS5jb20xFjAUBgNVBAcMDVNhbiBGcmFuY2lzY28xCzAJBgNV
BAgMAkNBMQwwCgYDVQQGEwNVU0EwHhcNMTcwNDI0MTgwMDQ1WhcNMTgwNDI0MTIwMDAwWjCBkDEo
MCYGA1UEAwwfU2VsZlNpZ25lZENlcnRfMjRBcHIyMDE3XzE4MDA0NDEYMBYGA1UECwwPMDBENkEw
MDAwMDAxN3Q4MRcwFQYDVQQKDA5TYWxlc2ZvcmNlLmNvbTEWMBQGA1UEBwwNU2FuIEZyYW5jaXNj
bzELMAkGA1UECAwCQ0ExDDAKBgNVBAYTA1VTQTCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoC
ggEBAIOR7h8BF2eFOlQHhV/1S7uOBN22Jv7PDCXMz2fU0uLc+mrv9xDGj6ElfW+9dSdXaCbQzD3+
Xq4reS4pYRafJZ/27OtygXl3rpoPjSlhRiW+oYVuDcCURJpu0KuZ4I0fm5q1BDYqxcBxNPSe85OH
E3+ucmKqvPozhQgYLPCregMIomC3yyANZnLCoGfCv9TpQl6/+I182tST4WPNhVPxKxijoPU4Rh6x
Y34Ez8+Jr8KdmzmYSNe4ukkIASplpvG7rKka824Hf8zI1BWnjWLDxb5IAxgUBbdr4x8d8C3kPfTf
+3/6yC5wSOm9NSs0BA4OJNowtXZFryMzFfXzDzjl69kCAwEAAaOCAQAwgf0wHQYDVR0OBBYEFO+D
koP6qkysi9ZC74yTPuJVVg2yMA8GA1UdEwEB/wQFMAMBAf8wgcoGA1UdIwSBwjCBv4AU74OSg/qq
TKyL1kLvjJM+4lVWDbKhgZakgZMwgZAxKDAmBgNVBAMMH1NlbGZTaWduZWRDZXJ0XzI0QXByMjAx
N18xODAwNDQxGDAWBgNVBAsMDzAwRDZBMDAwMDAwMTd0ODEXMBUGA1UECgwOU2FsZXNmb3JjZS5j
b20xFjAUBgNVBAcMDVNhbiBGcmFuY2lzY28xCzAJBgNVBAgMAkNBMQwwCgYDVQQGEwNVU0GCDgFb
oR91pAAAAAAeTabyMA0GCSqGSIb3DQEBCwUAA4IBAQAVhYBv5GJvhltks2j7Zc9wdFHW7yB4/hPF
o05y0yiOf71tLjOlBucSyxtmXLPjrECJvIJwKhsAIgYXnVp7ditxfauCcxczJgfeL1/dxH/Ge8eP
kmH6SdsO71cJL8dXEzOsoF+PAVQzUhqh8zxIipntL0wwNGTD0zIVQeTSozm0KF0SsSHIfbNy279u
ReGonC61i4Ouk5AMKA7Re9fVeUs6tqM2at22h9Zaj/r/OhXoDcZhzkd8Wq0ER/UKLZA1CyJHgwOC
7REEZOuKrqgfWcYt4dGo5q6gqGHHPMv0N7s/MxqCvJCwGA8eJGvOO56I321vhWHQ6ZSJDWUqQFM/
Ze7A</ds:X509Certificate>
        </ds:X509Data>
      </ds:KeyInfo>
    </ds:Signature>
    <saml:Subject>
      <saml:NameID Format="urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified">john@kol<!---->ide.co</saml:NameID>
      <saml:SubjectConfirmation Method="urn:oasis:names:tc:SAML:2.0:cm:bearer">
        <saml:SubjectConfirmationData InResponseTo="4982b430-73e1-4ad2-885a-4a775a91f820" NotOnOrAfter="2017-04-27T15:08:16.760Z" Recipient="https://localhost:8080/api/v1/kolide/sso/callback"/>
      </saml:SubjectConfirmation>
    </saml:Subject>
    <saml:Conditions NotBefore="2017-04-27T15:02:46.760Z" NotOnOrAfter="2017-04-27T15:08:16.760Z">
      <saml:AudienceRestriction>
        <saml:Audience>kolide</saml:Audience>
      </saml:AudienceRestriction>
    </saml:Conditions>
    <saml:AuthnStatement AuthnInstant="2017-04-27T15:03:16.750Z">
      <saml:AuthnContext>
        <saml:AuthnContextClassRef>urn:oasis:names:tc:SAML:2.0:ac:classes:unspecified</saml:AuthnContextClassRef>
      </saml:AuthnContext>
    </saml:AuthnStatement>
    <saml:AttributeStatement>
      <saml:Attribute Name="userId" NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:unspecified">
        <saml:AttributeValue xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="xs:anyType">0056A000000Q6Rl</saml:AttributeValue>
      </saml:Attribute>
      <saml:Attribute Name="username" NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:unspecified">
        <saml:AttributeValue xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="xs:anyType">john@kolide.co</saml:AttributeValue>
      </saml:Attribute>
      <saml:Attribute Name="email" NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:unspecified">
        <saml:AttributeValue xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="xs:anyType">john@kolide.co</saml:AttributeValue>
      </saml:Attribute>
      <saml:Attribute Name="is_portal_user" NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:unspecified">
        <saml:AttributeValue xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="xs:anyType">false</saml:AttributeValue>
      </saml:Attribute>
    </saml:AttributeStatement>
  </saml:Assertion>
</samlp:Response>`

	setFakeClock(t, "2017-04-27T15:03:16.747Z")
	acsURL, err := url.Parse("https://localhost:8080/api/v1/kolide/sso/callback")
	require.NoError(t, err)
	p := dummySAMLProvider(acsURL)
	p.IDPMetadata = testSalesforceMetadata()
	p.IDPMetadata.EntityID = "https://kolide-dev-ed.my.salesforce.com"

	auth, err := ParseAndVerifySAMLResponse(p, []byte(samlResponse), "4982b430-73e1-4ad2-885a-4a775a91f820", acsURL)
	require.NoError(t, err)
	require.NotNil(t, auth)
	assert.Equal(t, "john@kol<!---->ide.co", auth.UserID())
}

type nopSignatureVerified struct{}

func (n *nopSignatureVerified) VerifySignature(validationContext *dsig.ValidationContext, el *etree.Element) error {
	return nil
}

func dummySAMLProvider(acsURL *url.URL) *saml.ServiceProvider {
	return &saml.ServiceProvider{
		ValidateAudienceRestriction: func(assertion *saml.Assertion) error {
			return nil
		},
		SignatureVerifier: &nopSignatureVerified{},
		AcsURL:            *acsURL,
		// We set the Salesforce test metadata because the library requires setting certificates,
		// but the dummySAMLProvider performs no signature verification.
		IDPMetadata: testSalesforceMetadata(),
	}
}

func setFakeClock(t *testing.T, date string) {
	tm, err := time.Parse("2006-01-02T15:04:05.000Z", date)
	require.NoError(t, err)
	saml.TimeNow = func() time.Time {
		return tm
	}
	clock := dsig.NewFakeClockAt(tm)
	saml.Clock = clock
}

func TestDecodeSuccessfulGoogleResponse(t *testing.T) {
	const samlResponse = `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<saml2p:Response xmlns:saml2p="urn:oasis:names:tc:SAML:2.0:protocol" Destination="https://localhost:8080/api/v1/kolide/sso/callback" ID="_83579a9008ef726f87c52aad4b6dcc04" InResponseTo="SGJhi1g5D4/npOwXaw8t6A==" IssueInstant="2017-07-18T14:47:08.035Z" Version="2.0">
  <saml2:Issuer xmlns:saml2="urn:oasis:names:tc:SAML:2.0:assertion">https://accounts.google.com/o/saml2?idpid=C0171bstf</saml2:Issuer>
  <saml2p:Status>
    <saml2p:StatusCode Value="urn:oasis:names:tc:SAML:2.0:status:Success"/>
  </saml2p:Status>
  <saml2:Assertion xmlns:saml2="urn:oasis:names:tc:SAML:2.0:assertion" ID="_500061990acc00723288833a327cc986" IssueInstant="2017-07-18T14:47:08.035Z" Version="2.0">
    <saml2:Issuer>https://accounts.google.com/o/saml2?idpid=C0171bstf</saml2:Issuer>
    <ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
      <ds:SignedInfo>
        <ds:CanonicalizationMethod Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/>
        <ds:SignatureMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"/>
        <ds:Reference URI="#_500061990acc00723288833a327cc986">
          <ds:Transforms>
            <ds:Transform Algorithm="http://www.w3.org/2000/09/xmldsig#enveloped-signature"/>
            <ds:Transform Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/>
          </ds:Transforms>
          <ds:DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"/>
          <ds:DigestValue>nZmgK9XtjyT7sBApU0tyZbUE4WWMwCsDz8j6IZE5Ixw=</ds:DigestValue>
        </ds:Reference>
      </ds:SignedInfo>
      <ds:SignatureValue>DHdU+LnOX/u8Hujx+IpDmozt9u2ROD9UU2Ob5El0ZjEpAESqyY2Pj9Y4Kd01IsDTf/gFKJWOyVMz
PP3io5P4eiA96p+0g0YNuO6ickVF9BHAJyjET38C3pB95rgqUb7rLaD6XdfAXFQ7l2dalHS9yLa/
KBtT3f3ykYPb74NrAhihV8Z0gvPpyWqBDg23B76tIerWn26LooZkPNXPTGv/sy8ocY5oz56plKvZ
OmVdwpzwH7/7i/UEnNv6sis3/es0Omm5gxeKLP40vWb9lTm1HmvLTV3sZiHZQQmUwmfcsZL6gyVE
eaJNDQP4yOw+vXKdeyAlVC6jtt06MgY9V0zj5g==</ds:SignatureValue>
      <ds:KeyInfo>
        <ds:X509Data>
          <ds:X509SubjectName>ST=California,C=US,OU=Google For Work,CN=Google,L=Mountain View,O=Google Inc.</ds:X509SubjectName>
          <ds:X509Certificate>MIIDdDCCAlygAwIBAgIGAV1SKeijMA0GCSqGSIb3DQEBCwUAMHsxFDASBgNVBAoTC0dvb2dsZSBJ
bmMuMRYwFAYDVQQHEw1Nb3VudGFpbiBWaWV3MQ8wDQYDVQQDEwZHb29nbGUxGDAWBgNVBAsTD0dv
b2dsZSBGb3IgV29yazELMAkGA1UEBhMCVVMxEzARBgNVBAgTCkNhbGlmb3JuaWEwHhcNMTcwNzE3
MjAwNzQzWhcNMjIwNzE2MjAwNzQzWjB7MRQwEgYDVQQKEwtHb29nbGUgSW5jLjEWMBQGA1UEBxMN
TW91bnRhaW4gVmlldzEPMA0GA1UEAxMGR29vZ2xlMRgwFgYDVQQLEw9Hb29nbGUgRm9yIFdvcmsx
CzAJBgNVBAYTAlVTMRMwEQYDVQQIEwpDYWxpZm9ybmlhMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A
MIIBCgKCAQEAzLXNn7VmJBkvVNYHffTzDoow/8eSklauVeYjhELY6dtFv56wAQsFNeMovFUPxPeG
7Fci50/KStvoNZOdKqZFCwYkfI2ssXuMpBP37x2iprV7moVwGdGJb52elMNe0DesgTPbJ/IWIvzF
3GYxqYCHUlHuzJEzBYsdtvM8T/PClBxiLXRNbnjotzleFqb25w3XRfayOZg5GdQPeEmceWXDBhCa
eQyEPOrUTZ+//pZXSuKnOyaFfESNFNgvQJlYQQukjnhPtf674eWT6OdgZHyq8EBbZKfEhs5+KiAN
U43bDh9rpTJCB7rAKk1BFAW3r72pggwN9Z/sfp/C5B7uKAM5hwIDAQABMA0GCSqGSIb3DQEBCwUA
A4IBAQAZXypikbbRzichNXLdK96M/do9nGS5Q3xVgA2uxTzm/6qNkAfOSGSk8OcLrppPonbohkeZ
WVnNB5VZZava4DoSZ6OZsvKc1FM0wKvPJd83KUb7Syk1bV7TkT8DPEclfsLnn5s5g0oHlhsqkNly
0WPFTAoGHXYyOKGEARPoC/o+ZfgfvoMNyZkSQHiRboVVP2cT1ckJt4iCA65hNGXte29hSGmnX7QG
QyrBRp8n4UR9PjoeIy0tTCmG0tqu/NackFH4PkamY84Etxe9uH0StmkhID46QTT4Cv2+jqCaklg+
7VYqXbY64Wc/k0sK7WI1o3IVLWAPNb8ajV6Eo0Y8u+1N</ds:X509Certificate>
        </ds:X509Data>
      </ds:KeyInfo>
    </ds:Signature>
    <saml2:Subject>
      <saml2:NameID Format="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress">john@edilok.net</saml2:NameID>
      <saml2:SubjectConfirmation Method="urn:oasis:names:tc:SAML:2.0:cm:bearer">
        <saml2:SubjectConfirmationData InResponseTo="SGJhi1g5D4/npOwXaw8t6A==" NotOnOrAfter="2017-07-18T14:52:08.035Z" Recipient="https://localhost:8080/api/v1/kolide/sso/callback"/>
      </saml2:SubjectConfirmation>
    </saml2:Subject>
    <saml2:Conditions NotBefore="2017-07-18T14:42:08.035Z" NotOnOrAfter="2017-07-18T14:52:08.035Z">
      <saml2:AudienceRestriction>
        <saml2:Audience>kolide.edilok.net</saml2:Audience>
      </saml2:AudienceRestriction>
    </saml2:Conditions>
    <saml2:AttributeStatement>
      <saml2:Attribute Name="myattribute">
        <saml2:AttributeValue xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="xs:anyType">john@edilok.net</saml2:AttributeValue>
      </saml2:Attribute>
    </saml2:AttributeStatement>
    <saml2:AuthnStatement AuthnInstant="2017-07-18T14:33:41.000Z" SessionIndex="_500061990acc00723288833a327cc986">
      <saml2:AuthnContext>
        <saml2:AuthnContextClassRef>urn:oasis:names:tc:SAML:2.0:ac:classes:unspecified</saml2:AuthnContextClassRef>
      </saml2:AuthnContext>
    </saml2:AuthnStatement>
  </saml2:Assertion>
</saml2p:Response>`

	setFakeClock(t, "2017-07-18T14:47:08.035Z")
	acsURL, err := url.Parse("https://localhost:8080/api/v1/kolide/sso/callback")
	require.NoError(t, err)
	p := dummySAMLProvider(acsURL)
	p.IDPMetadata.EntityID = "https://accounts.google.com/o/saml2?idpid=C0171bstf"

	auth, err := ParseAndVerifySAMLResponse(p, []byte(samlResponse), "SGJhi1g5D4/npOwXaw8t6A==", acsURL)
	require.NoError(t, err)
	require.NotNil(t, auth)
	assert.Equal(t, "john@edilok.net", auth.UserID())
}

func TestDecodeAttributes(t *testing.T) {
	const samlResponse = `<samlp:Response xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" ID="_8419c6009d891c39b454f53b34454818bdab19bd3a" Version="2.0" IssueInstant="2022-08-10T19:42:34Z" Destination="https://localhost:8080/api/v1/fleet/sso/callback" InResponseTo="idJpL4A6IM6MMBQ9eH">
  <saml:Issuer>http://localhost:9080/simplesaml/saml2/idp/metadata.php</saml:Issuer>
  <ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
    <ds:SignedInfo>
      <ds:CanonicalizationMethod Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/>
      <ds:SignatureMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"/>
      <ds:Reference URI="#_8419c6009d891c39b454f53b34454818bdab19bd3a">
        <ds:Transforms>
          <ds:Transform Algorithm="http://www.w3.org/2000/09/xmldsig#enveloped-signature"/>
          <ds:Transform Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/>
        </ds:Transforms>
        <ds:DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"/>
        <ds:DigestValue>HtQ1uTkyzTwwNZ6XJQT8ROuQsfaw99TiQktm2EhKPWk=</ds:DigestValue>
      </ds:Reference>
    </ds:SignedInfo>
    <ds:SignatureValue>EawypjAfab8xnhmF994nIq8oSqd/DenA4HGj7A85MRlAGa5W3fftFwmovXPhbGZeDZsQtAGssnvtJgZ7FEraWF9oYGxqdZQZ2Oethx6uKxaryxKmp22erd8ctd0JMMmK5vu2PFqLCoP2qFvmruM4ruYHbee5sl8EMWHJ/FExuq4bWoeAa08ZCYdJnPawO2AyIzJtAku2nXQwz1756VrV1mX7tenaADxqMexCJdyXSuDG1I1nXp/qnRIOjUbfsfIN0WkG8jY9MDdUUi8ah0UYgdv2kQn1d/FYoI1/AuPXRl72QLtN4jFbg0ZZJAX5wtP66vvbC29cK6SN3h1ghSFFCg==</ds:SignatureValue>
    <ds:KeyInfo>
      <ds:X509Data>
        <ds:X509Certificate>MIIDXTCCAkWgAwIBAgIJALmVVuDWu4NYMA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQwHhcNMTYxMjMxMTQzNDQ3WhcNNDgwNjI1MTQzNDQ3WjBFMQswCQYDVQQGEwJBVTETMBEGA1UECAwKU29tZS1TdGF0ZTEhMB8GA1UECgwYSW50ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAzUCFozgNb1h1M0jzNRSCjhOBnR+uVbVpaWfXYIR+AhWDdEe5ryY+CgavOg8bfLybyzFdehlYdDRgkedEB/GjG8aJw06l0qF4jDOAw0kEygWCu2mcH7XOxRt+YAH3TVHa/Hu1W3WjzkobqqqLQ8gkKWWM27fOgAZ6GieaJBN6VBSMMcPey3HWLBmc+TYJmv1dbaO2jHhKh8pfKw0W12VM8P1PIO8gv4Phu/uuJYieBWKixBEyy0lHjyixYFCR12xdh4CA47q958ZRGnnDUGFVE1QhgRacJCOZ9bd5t9mr8KLaVBYTCJo5ERE8jymab5dPqe5qKfJsCZiqWglbjUo9twIDAQABo1AwTjAdBgNVHQ4EFgQUxpuwcs/CYQOyui+r1G+3KxBNhxkwHwYDVR0jBBgwFoAUxpuwcs/CYQOyui+r1G+3KxBNhxkwDAYDVR0TBAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAQEAAiWUKs/2x/viNCKi3Y6blEuCtAGhzOOZ9EjrvJ8+COH3Rag3tVBWrcBZ3/uhhPq5gy9lqw4OkvEws99/5jFsX1FJ6MKBgqfuy7yh5s1YfM0ANHYczMmYpZeAcQf2CGAaVfwTTfSlzNLsF2lW/ly7yapFzlYSJLGoVE+OHEu8g5SlNACUEfkXw+5Eghh+KzlIN7R6Q7r2ixWNFBC/jWf7NKUfJyX8qIG5md1YUeT6GBW9Bm2/1/RiO24JTaYlfLdKK9TYb8sG5B+OLab2DImG99CJ25RkAcSobWNF5zD0O6lgOo3cEdB/ksCq3hmtlC/DlLZ/D8CJ+7VuZnS1rR2naQ==</ds:X509Certificate>
      </ds:X509Data>
    </ds:KeyInfo>
  </ds:Signature>
  <samlp:Status>
    <samlp:StatusCode Value="urn:oasis:names:tc:SAML:2.0:status:Success"/>
  </samlp:Status>
  <saml:Assertion xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xs="http://www.w3.org/2001/XMLSchema" ID="_ec34771d0b777b553f5e0363dbbb36fc973e183f29" Version="2.0" IssueInstant="2022-08-10T19:42:34Z">
    <saml:Issuer>http://localhost:9080/simplesaml/saml2/idp/metadata.php</saml:Issuer>
    <ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
      <ds:SignedInfo>
        <ds:CanonicalizationMethod Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/>
        <ds:SignatureMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"/>
        <ds:Reference URI="#_ec34771d0b777b553f5e0363dbbb36fc973e183f29">
          <ds:Transforms>
            <ds:Transform Algorithm="http://www.w3.org/2000/09/xmldsig#enveloped-signature"/>
            <ds:Transform Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/>
          </ds:Transforms>
          <ds:DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"/>
          <ds:DigestValue>AuiXXRBMG03hyrsjxHQeaI82pfb6JQ/8FJMqu1zC46o=</ds:DigestValue>
        </ds:Reference>
      </ds:SignedInfo>
      <ds:SignatureValue>Tsjsa0xPER51uAOlM+smsX+kBkqy/0xXlFEohzdflfP6N0qtjEla/jOC9M5j1F+bfkjeP3Lo2cp5OvU2PeQRWpfp+CTfF1tZCuP+F/Du0n4wfKMAU6bjNhq8LYEjC/LbNIyLnDNxT33lZp9udBWPmXwe4U92jcCEY54Tnql7xdqh/ez7J1A3TjNbsbTc9D9NWD9ZG7QR8+GcNVAq5fOxcElxSFrgZJGhoUO+PPes4C9oiB44f6FTcEoKHD9gYycMB9kich1lguG0DO9v+daZZ9HQyeoUaF50W231TM2Iax1YGD3QLZg674X90HGG7YKuRIenocVCyZgljzS2tAeNog==</ds:SignatureValue>
      <ds:KeyInfo>
        <ds:X509Data>
          <ds:X509Certificate>MIIDXTCCAkWgAwIBAgIJALmVVuDWu4NYMA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQwHhcNMTYxMjMxMTQzNDQ3WhcNNDgwNjI1MTQzNDQ3WjBFMQswCQYDVQQGEwJBVTETMBEGA1UECAwKU29tZS1TdGF0ZTEhMB8GA1UECgwYSW50ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAzUCFozgNb1h1M0jzNRSCjhOBnR+uVbVpaWfXYIR+AhWDdEe5ryY+CgavOg8bfLybyzFdehlYdDRgkedEB/GjG8aJw06l0qF4jDOAw0kEygWCu2mcH7XOxRt+YAH3TVHa/Hu1W3WjzkobqqqLQ8gkKWWM27fOgAZ6GieaJBN6VBSMMcPey3HWLBmc+TYJmv1dbaO2jHhKh8pfKw0W12VM8P1PIO8gv4Phu/uuJYieBWKixBEyy0lHjyixYFCR12xdh4CA47q958ZRGnnDUGFVE1QhgRacJCOZ9bd5t9mr8KLaVBYTCJo5ERE8jymab5dPqe5qKfJsCZiqWglbjUo9twIDAQABo1AwTjAdBgNVHQ4EFgQUxpuwcs/CYQOyui+r1G+3KxBNhxkwHwYDVR0jBBgwFoAUxpuwcs/CYQOyui+r1G+3KxBNhxkwDAYDVR0TBAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAQEAAiWUKs/2x/viNCKi3Y6blEuCtAGhzOOZ9EjrvJ8+COH3Rag3tVBWrcBZ3/uhhPq5gy9lqw4OkvEws99/5jFsX1FJ6MKBgqfuy7yh5s1YfM0ANHYczMmYpZeAcQf2CGAaVfwTTfSlzNLsF2lW/ly7yapFzlYSJLGoVE+OHEu8g5SlNACUEfkXw+5Eghh+KzlIN7R6Q7r2ixWNFBC/jWf7NKUfJyX8qIG5md1YUeT6GBW9Bm2/1/RiO24JTaYlfLdKK9TYb8sG5B+OLab2DImG99CJ25RkAcSobWNF5zD0O6lgOo3cEdB/ksCq3hmtlC/DlLZ/D8CJ+7VuZnS1rR2naQ==</ds:X509Certificate>
        </ds:X509Data>
      </ds:KeyInfo>
    </ds:Signature>
    <saml:Subject>
      <saml:NameID SPNameQualifier="https://localhost:8080" Format="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress">sso_user@example.com</saml:NameID>
      <saml:SubjectConfirmation Method="urn:oasis:names:tc:SAML:2.0:cm:bearer">
        <saml:SubjectConfirmationData NotOnOrAfter="2022-08-10T19:47:34Z" Recipient="https://localhost:8080/api/v1/fleet/sso/callback" InResponseTo="idJpL4A6IM6MMBQ9eH"/>
      </saml:SubjectConfirmation>
    </saml:Subject>
    <saml:Conditions NotBefore="2022-08-10T19:42:04Z" NotOnOrAfter="2022-08-10T19:47:34Z">
      <saml:AudienceRestriction>
        <saml:Audience>https://localhost:8080</saml:Audience>
      </saml:AudienceRestriction>
    </saml:Conditions>
    <saml:AuthnStatement AuthnInstant="2022-08-10T19:42:34Z" SessionNotOnOrAfter="2022-08-11T03:42:34Z" SessionIndex="_4d73872e0cb05ec4ca29cc1c7ea86b0f731234f675">
      <saml:AuthnContext>
        <saml:AuthnContextClassRef>urn:oasis:names:tc:SAML:2.0:ac:classes:Password</saml:AuthnContextClassRef>
      </saml:AuthnContext>
    </saml:AuthnStatement>
    <saml:AttributeStatement>
      <saml:Attribute Name="uid" NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:basic">
        <saml:AttributeValue xsi:type="xs:string">1</saml:AttributeValue>
      </saml:Attribute>
      <saml:Attribute Name="eduPersonAffiliation" NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:basic">
        <saml:AttributeValue xsi:type="xs:string">group1</saml:AttributeValue>
      </saml:Attribute>
      <saml:Attribute Name="displayname" NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:basic">
        <saml:AttributeValue xsi:type="xs:string">SSO User 1</saml:AttributeValue>
      </saml:Attribute>
      <saml:Attribute Name="email" NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:basic">
        <saml:AttributeValue xsi:type="xs:string">sso_user@example.com</saml:AttributeValue>
      </saml:Attribute>
    </saml:AttributeStatement>
  </saml:Assertion>
</samlp:Response>`

	setFakeClock(t, "2022-08-10T19:42:34.000Z")
	acsURL, err := url.Parse("https://localhost:8080/api/v1/fleet/sso/callback")
	require.NoError(t, err)
	p := dummySAMLProvider(acsURL)
	p.IDPMetadata.EntityID = "http://localhost:9080/simplesaml/saml2/idp/metadata.php"

	auth, err := ParseAndVerifySAMLResponse(p, []byte(samlResponse), "idJpL4A6IM6MMBQ9eH", acsURL)
	require.NoError(t, err)
	require.NotNil(t, auth)
	assert.Equal(t, "sso_user@example.com", auth.UserID())
	assert.Equal(t, "SSO User 1", auth.UserDisplayName())
}

func TestEmptyResponse(t *testing.T) {
	var auth resp
	assert.Equal(t, "", auth.UserID())
	assert.Equal(t, "", auth.UserDisplayName())
	assert.Empty(t, auth.AssertionAttributes())
}

func TestUserDisplayName(t *testing.T) {
	cases := []struct {
		attrName string
		values   []saml.AttributeValue
		out      string
	}{
		{"name", []saml.AttributeValue{{Value: "Name Surname"}}, "Name Surname"},
		{"displayname", []saml.AttributeValue{{Value: "Name Surname"}}, "Name Surname"},
		{"displayName", []saml.AttributeValue{{Value: "Name Surname"}}, "Name Surname"},
		{"cn", []saml.AttributeValue{{Value: "Name Surname"}}, "Name Surname"},
		{"urn:oid:2.5.4.3", []saml.AttributeValue{{Value: "Name Surname"}}, "Name Surname"},
		{"http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name", []saml.AttributeValue{{Value: "Name Surname"}}, "Name Surname"},
		{"name", []saml.AttributeValue{{Value: ""}}, ""},
		{"", []saml.AttributeValue{{Value: ""}}, ""},
		{"displayname", []saml.AttributeValue{{Value: ""}, {Value: "Name Surname"}}, "Name Surname"},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%s attribute with %v values", c.attrName, c.values), func(t *testing.T) {
			var auth resp
			auth.assertion = &saml.Assertion{
				AttributeStatements: []saml.AttributeStatement{{Attributes: []saml.Attribute{{Name: c.attrName, Values: c.values}}}},
			}
			assert.Equal(t, c.out, auth.UserDisplayName())
		})
	}
}

func TestDecodeOktaResponseWithCustomAttrs(t *testing.T) {
	const samlResponse = `<?xml version="1.0" encoding="UTF-8"?>
<saml2p:Response Destination="https://example.com/api/v1/fleet/sso/callback" ID="id163119768052374872101569422" InResponseTo="id1Juy6Mx2IHYxLwsi" IssueInstant="2023-02-27T17:41:53.505Z" Version="2.0" xmlns:saml2p="urn:oasis:names:tc:SAML:2.0:protocol" xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <saml2:Issuer Format="urn:oasis:names:tc:SAML:2.0:nameid-format:entity" xmlns:saml2="urn:oasis:names:tc:SAML:2.0:assertion">http://www.okta.com/exk8glknbnr9Lpdkl5d7</saml2:Issuer>
  <ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
    <ds:SignedInfo>
      <ds:CanonicalizationMethod Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/>
      <ds:SignatureMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"/>
      <ds:Reference URI="#id163119768052374872101569422">
        <ds:Transforms>
          <ds:Transform Algorithm="http://www.w3.org/2000/09/xmldsig#enveloped-signature"/>
          <ds:Transform Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#">
            <ec:InclusiveNamespaces PrefixList="xs" xmlns:ec="http://www.w3.org/2001/10/xml-exc-c14n#"/>
          </ds:Transform>
        </ds:Transforms>
        <ds:DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"/>
        <ds:DigestValue>YjhYEzGsKy0vrLtgepjdmeAYf8ZkSiPMsKvSV/QSXZY=</ds:DigestValue>
      </ds:Reference>
    </ds:SignedInfo>
    <ds:SignatureValue>REDACTED</ds:SignatureValue>
    <ds:KeyInfo>
      <ds:X509Data>
        <ds:X509Certificate>REDACTED</ds:X509Certificate>
      </ds:X509Data>
    </ds:KeyInfo>
  </ds:Signature>
  <saml2p:Status xmlns:saml2p="urn:oasis:names:tc:SAML:2.0:protocol">
    <saml2p:StatusCode Value="urn:oasis:names:tc:SAML:2.0:status:Success"/>
  </saml2p:Status>
  <saml2:Assertion ID="id16311976805446352575023709" IssueInstant="2023-02-27T17:41:53.505Z" Version="2.0" xmlns:saml2="urn:oasis:names:tc:SAML:2.0:assertion" xmlns:xs="http://www.w3.org/2001/XMLSchema">
    <saml2:Issuer Format="urn:oasis:names:tc:SAML:2.0:nameid-format:entity" xmlns:saml2="urn:oasis:names:tc:SAML:2.0:assertion">http://www.okta.com/exk8glknbnr9Lpdkl5d7</saml2:Issuer>
    <ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
      <ds:SignedInfo>
        <ds:CanonicalizationMethod Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/>
        <ds:SignatureMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"/>
        <ds:Reference URI="#id16311976805446352575023709">
          <ds:Transforms>
            <ds:Transform Algorithm="http://www.w3.org/2000/09/xmldsig#enveloped-signature"/>
            <ds:Transform Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#">
              <ec:InclusiveNamespaces PrefixList="xs" xmlns:ec="http://www.w3.org/2001/10/xml-exc-c14n#"/>
            </ds:Transform>
          </ds:Transforms>
          <ds:DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"/>
          <ds:DigestValue>/W9NXsxZTI5GljYgaUNnl21ZlgF4L4A/npt/mpOyOeg=</ds:DigestValue>
        </ds:Reference>
      </ds:SignedInfo>
      <ds:SignatureValue>REDACTED</ds:SignatureValue>
      <ds:KeyInfo>
        <ds:X509Data>
          <ds:X509Certificate>REDACTED</ds:X509Certificate>
        </ds:X509Data>
      </ds:KeyInfo>
    </ds:Signature>
    <saml2:Subject xmlns:saml2="urn:oasis:names:tc:SAML:2.0:assertion">
      <saml2:NameID Format="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress">foo@example.com</saml2:NameID>
      <saml2:SubjectConfirmation Method="urn:oasis:names:tc:SAML:2.0:cm:bearer">
        <saml2:SubjectConfirmationData InResponseTo="id1Juy6Mx2IHYxLwsi" NotOnOrAfter="2023-02-27T17:46:53.506Z" Recipient="https://example.com/api/v1/fleet/sso/callback"/>
      </saml2:SubjectConfirmation>
    </saml2:Subject>
    <saml2:Conditions NotBefore="2023-02-27T17:36:53.506Z" NotOnOrAfter="2023-02-27T17:46:53.506Z" xmlns:saml2="urn:oasis:names:tc:SAML:2.0:assertion">
      <saml2:AudienceRestriction>
        <saml2:Audience>example.com</saml2:Audience>
      </saml2:AudienceRestriction>
    </saml2:Conditions>
    <saml2:AuthnStatement AuthnInstant="2023-02-27T17:41:53.505Z" SessionIndex="id1Juy6Mx2IHYxLwsi" xmlns:saml2="urn:oasis:names:tc:SAML:2.0:assertion">
      <saml2:AuthnContext>
        <saml2:AuthnContextClassRef>urn:oasis:names:tc:SAML:2.0:ac:classes:PasswordProtectedTransport</saml2:AuthnContextClassRef>
      </saml2:AuthnContext>
    </saml2:AuthnStatement>
    <saml2:AttributeStatement xmlns:saml2="urn:oasis:names:tc:SAML:2.0:assertion">
      <saml2:Attribute Name="FLEET_JIT_USER_ROLE_TEAM_1" NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:unspecified">
        <saml2:AttributeValue xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="xs:string">observer</saml2:AttributeValue>
      </saml2:Attribute>
      <saml2:Attribute Name="FLEET_JIT_USER_ROLE_TEAM_2" NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:unspecified">
        <saml2:AttributeValue xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="xs:string">maintainer</saml2:AttributeValue>
      </saml2:Attribute>
      <saml2:Attribute Name="foo" NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:unspecified">
        <saml2:AttributeValue xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="xs:string">bar</saml2:AttributeValue>
      </saml2:Attribute>
    </saml2:AttributeStatement>
  </saml2:Assertion>
</saml2p:Response>`

	setFakeClock(t, "2023-02-27T17:41:53.505Z")
	acsURL, err := url.Parse("https://example.com/api/v1/fleet/sso/callback")
	require.NoError(t, err)
	p := dummySAMLProvider(acsURL)
	p.IDPMetadata.EntityID = "http://www.okta.com/exk8glknbnr9Lpdkl5d7"

	auth, err := ParseAndVerifySAMLResponse(p, []byte(samlResponse), "id1Juy6Mx2IHYxLwsi", acsURL)
	require.NoError(t, err)
	require.NotNil(t, auth)
	assert.Equal(t, "foo@example.com", auth.UserID())
	attrs := auth.AssertionAttributes()
	assert.Equal(t, []fleet.SAMLAttribute{
		{
			Name: "FLEET_JIT_USER_ROLE_TEAM_1",
			Values: []fleet.SAMLAttributeValue{{
				Type:  "xs:string",
				Value: "observer",
			}},
		},
		{
			Name: "FLEET_JIT_USER_ROLE_TEAM_2",
			Values: []fleet.SAMLAttributeValue{{
				Type:  "xs:string",
				Value: "maintainer",
			}},
		},
		{
			Name: "foo",
			Values: []fleet.SAMLAttributeValue{{
				Type:  "xs:string",
				Value: "bar",
			}},
		},
	}, attrs)
}

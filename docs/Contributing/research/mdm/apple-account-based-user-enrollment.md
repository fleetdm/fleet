# Account-driven user enrollment for Apple MDM

Account-driven user enrollment better supports BYOD (Bring Your Own Device) scenarios while maintaining strong security and privacy boundaries between
work and personal data.

## Key differences from traditional MDM enrollment

- **User-Centric**: Enrollment is tied to the user's Managed Apple ID (required) rather than just the device
  - MDM server only sees EnrollmentID and does not see device UDID or hardware serial number
  - EnrollmentID changes with each enrollment, so it is impossible to link enrollment to a specific device
- **Streamlined Authentication**: Leverages modern authentication flows (SAML/OAuth2) for user verification
- **Limited Management**: Provides specific restrictions on what MDM can manage, protecting user privacy
  - Profiles are user-scoped (not system-scoped)

## Questions?

- Which Fleet team will the enrolling device be assigned to?

## Links

- [Apple developer docs](https://developer.apple.com/documentation/devicemanagement/onboarding-users-with-account-sign-in)

## Enrollment flow

The enrollment process follows these main steps:

1. User initiates enrollment on their iOS device
2. Device contacts Fleet server for enrollment
3. User authenticates via SSO (SAML or OAuth2)
4. Fleet server validates authentication and provides the enrollment profile
- Fleet server can link the SAML username/email to the device via the access-token it sends back to the device
5. User authenticates with managed Apple ID
6. Device gets SCEP certificate
7. Device performs MDM check-in protocol steps

```mermaid
---
title: Account-driven user enrollment
---
sequenceDiagram
    autonumber
    participant ios as iOS host
    participant fleet as Fleet server
    participant IdP
    activate ios
    ios->>+fleet: POST /account_driven_enroll
    fleet-->>-ios: 401 WWW-Authenticate
    alt SAML
        ios->>+fleet: Fleet SSO URL
        fleet-->>-ios: HTML page
        ios->>+fleet: [HTML page]<br>POST /api/v1/fleet/mdm/sso
        fleet-->>-ios: IdP URL (SAML request, HTTP redirect binding)
        ios->>+IdP: SAML request
        IdP-->>-ios: IdP login page
        ios->>ios: Enter IdP credentials
        ios->>+IdP: IdP login
        IdP-->>-ios: SAML Response
        ios->>+fleet: [HTML page]<br>POST /api/v1/fleet/mdm/sso/callback<br>(SAML Response)
        fleet-->>-ios: 308 (with access token)
    else OAuth2
        ios->>IdP: Authorization URL
        ios->>ios: Enter IdP credentials
        ios->>fleet: Token URL
        activate fleet
        fleet->>+IdP: Token URL (with client secret)
        IdP-->>-fleet: Access token
        fleet-->>ios: Forward access token
        deactivate fleet
    end
    ios->>+fleet: POST /account_driven_enroll (with Bearer token)
    fleet-->>-ios: Enrollment profile
    ios->>ios: Enter Apple credentials
    ios->>fleet: Get SCEP certificate flow
    ios->>+fleet: PUT /mdm/apple/mdm<br/>[check-in protocol]<br/>MessageType: Authenticate
    fleet-->>-ios: OK
    ios->>+fleet: PUT /mdm/apple/mdm<br/>[check-in protocol]<br/>MessageType: TokenUpdate
    fleet-->>-ios: OK
    deactivate ios
```

### Detailed example of SAML enrollment flow

This example is based on the 2025/04/29 POC, so URLs and details may have changed.

```
POST /api/mdm/apple/enroll`

Headers
Accept	*/*
Accept-Encoding	gzip, deflate, br
Accept-Language	en-US,en;q=0.9
Content-Length	3596
Content-Type	application/pkcs7-signature
Host	<Fleet server>
User-Agent	DeviceManagementClient/1.0
X-Forwarded-For	<Client IP>
X-Forwarded-Host	<Fleet server>
X-Forwarded-Proto	https
```

```
401 Unauthorized

Headers
Content-Length	0
Content-Type	application/json; charset=utf-8
Date	Tue, 29 Apr 2025 18:58:21 GMT
Www-Authenticate	Bearer method="apple-as-web" url="<Fleet server URL>/mdm/sso"
```

```
GET /mdm/sso

Accept	text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8
Accept-Encoding	gzip, deflate, br
Accept-Language	en-US,en;q=0.9
Host	<Fleet server>
Priority	u=0, i
Sec-Fetch-Dest	document
Sec-Fetch-Mode	navigate
Sec-Fetch-Site	none
User-Agent	Mozilla/5.0 (iPhone; CPU iPhone OS 18_2_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.2 Mobile/15E148 Safari/604.1
X-Forwarded-For	<Client IP>
X-Forwarded-Host	<Fleet server>
X-Forwarded-Proto	https
```

```
200 OK

HTML page for orchestrating SAML login
```

```
POST /api/latest/fleet/mdm/sso

Headers
Accept	application/json, text/plain, */*
Accept-Encoding	gzip, deflate, br
Accept-Language	en-US,en;q=0.9
Authorization	Bearer null
Content-Length	2
Content-Type	application/json
Host	<Fleet server>
Origin	https://<Fleet server>
Priority	u=3, i
Referer	https://<Fleet server>/mdm/sso?user-identifier=<Managed Apple ID>
Sec-Fetch-Dest	empty
Sec-Fetch-Mode	cors
Sec-Fetch-Site	same-origin
User-Agent	Mozilla/5.0 (iPhone; CPU iPhone OS 18_2_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.2 Mobile/15E148 Safari/604.1
X-Forwarded-For	<Client IP>
X-Forwarded-Host	<Fleet server>
X-Forwarded-Proto	https
```

```
200 OK

Headers
Content-Length	560
Content-Type	application/json; charset=utf-8
Date	Tue, 29 Apr 2025 18:58:24 GMT

{
    "url": "https://dev-XXXXXX.okta.com/app/dev-XXXXXX_saml_1/XXXXX/sso/saml?SAMLRequest=<SAML HTTP request binding>"
}
```

```
POST /api/v1/fleet/mdm/sso/callback

Headers
Accept	text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8
Accept-Encoding	gzip, deflate, br
Accept-Language	en-US,en;q=0.9
Content-Length	9323
Content-Type	application/x-www-form-urlencoded
Host	<Fleet server>
Origin	https://dev-XXXXXX.okta.com
Priority	u=0, i
Referer	https://dev-XXXXXX.okta.com/
Sec-Fetch-Dest	document
Sec-Fetch-Mode	navigate
Sec-Fetch-Site	cross-site
User-Agent	Mozilla/5.0 (iPhone; CPU iPhone OS 18_2_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.2 Mobile/15E148 Safari/604.1
X-Forwarded-For	<Client IP>
X-Forwarded-Host	<Fleet server>
X-Forwarded-Proto	https

SAMLResponse: <data>
```

```
308 Permanent Redirect

Headers
Content-Length	0
Content-Type	application/json; charset=utf-8
Date	Tue, 29 Apr 2025 18:58:58 GMT
Location	apple-remotemanagement-user-login://authentication-results?access-token=cccca116-2527-11f0-a903-0242ac120002
```

```
POST /api/mdm/apple/enroll

Headers
Accept	*/*
Accept-Encoding	gzip, deflate, br
Accept-Language	en-US,en;q=0.9
Authorization	Bearer cccca116-2527-11f0-a903-0242ac120002
Content-Length	3596
Content-Type	application/pkcs7-signature
Host	<Fleet server>
User-Agent	DeviceManagementClient/1.0
X-Forwarded-For	<Client IP>
X-Forwarded-Host	<Fleet server>
X-Forwarded-Proto	https
```

```
200 OK

Headers
Content-Disposition	attachment;fleet-enrollment-profile.mobileconfig
Content-Length	3896
Content-Type	application/x-apple-aspen-config
Date	Tue, 29 Apr 2025 18:58:58 GMT
X-Content-Type-Options	nosniff

<fleet-enrollment-profile>
```

```
<SCEP protocol>
```

```
PUT /mdm/apple/mdm

Headers
Accept	*/*
Accept-Encoding	gzip, deflate, br
Accept-Language	en-US,en;q=0.9
Authorization	Bearer cccca116-2527-11f0-a903-0242ac120002
Cache-Control	no-cache
Content-Length	570
Content-Type	application/x-apple-aspen-mdm-checkin
Host	<Fleet server>
Mdm-Signature	<signature>
User-Agent	MDM/1.0
X-Forwarded-For	<Client IP>
X-Forwarded-Host	<Fleet server>
X-Forwarded-Proto	https
```

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
    <dict>
        <key>BuildVersion</key>
        <string>22C161</string>
        <key>EnrollmentID</key>
        <string>FB833D84-50C0-41DD-99DE-F748E44FFEF5</string>
        <key>MessageType</key>
        <string>Authenticate</string>
        <key>OSVersion</key>
        <string>18.2.1</string>
        <key>ProductName</key>
        <string>iPhone12,1</string>
        <key>Topic</key>
        <string>com.apple.mgmt.External.XXXXX</string>
    </dict>
</plist>
```

```
200 OK

Headers
Content-Length	0
Date	Tue, 29 Apr 2025 18:59:50 GMT
```

```
PUT /mdm/apple/mdm

Headers
Accept	*/*
Accept-Encoding	gzip, deflate, br
Accept-Language	en-US,en;q=0.9
Authorization	Bearer cccca116-2527-11f0-a903-0242ac120002
Cache-Control	no-cache
Content-Length	621
Content-Type	application/x-apple-aspen-mdm-checkin
Host	<Fleet server>
Mdm-Signature	<signature>
User-Agent	MDM/1.0
X-Forwarded-For	<Client IP>
X-Forwarded-Host	<Fleet server>
X-Forwarded-Proto	https
```

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>AwaitingConfiguration</key>
	<false/>
	<key>EnrollmentID</key>
	<string>FB833D84-50C0-41DD-99DE-F748E44FFEF5</string>
	<key>MessageType</key>
	<string>TokenUpdate</string>
	<key>PushMagic</key>
	<string>7B8A04B0-E9B8-4BB3-8A85-4FD5FA1F70E9</string>
	<key>Token</key>
	<data>
	ftAoCo4gNk3mxQgFVRXRJfO8qYsMpS46kHClaugCnt8=
	</data>
	<key>Topic</key>
	<string>com.apple.mgmt.External.XXXXX</string>
</dict>
</plist>
```

```
200 OK

Headers
Content-Length	0
Date	Tue, 29 Apr 2025 18:59:50 GMT
```

```
PUT /mdm/apple/mdm

Headers
Accept	*/*
Accept-Encoding	gzip, deflate, br
Accept-Language	en-US,en;q=0.9
Authorization	Bearer cccca116-2527-11f0-a903-0242ac120002
Cache-Control	no-cache
Content-Length	310
Content-Type	application/x-apple-aspen-mdm
Host	<Fleet server>
Mdm-Signature	<signature>>
User-Agent	MDM/1.0
X-Forwarded-For	<Client IP>
X-Forwarded-Host	<Fleet server>
X-Forwarded-Proto	https
```

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>EnrollmentID</key>
	<string>FB833D84-50C0-41DD-99DE-F748E44FFEF5</string>
	<key>Status</key>
	<string>Idle</string>
</dict>
</plist>
```

```
200 OK

Headers
Content-Type	text/xml; charset=utf-8
Date	Tue, 29 Apr 2025 19:00:00 GMT

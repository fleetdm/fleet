# Spec Delta

## Purpose

Lets Fleet Premium admins connect an EJBCA certificate authority and have Fleet issue a unique PKCS#12 certificate for each Apple host through EJBCA's REST API, delivered by configuration profile, so hosts can join Wi-Fi and VPN with certificate authentication.

## ADDED Requirements

### Requirement: EJBCA CA type
Fleet SHALL support a certificate authority of type `ejbca` on Fleet Premium, creatable and editable through the UI, the REST API, and GitOps. The type SHALL be rejected on Fleet Free.

#### Scenario: Create on Premium
- **WHEN** an admin creates an EJBCA CA with valid fields
- **THEN** the CA is stored, appears in the certificate authority list with type `ejbca`, and an `added_ejbca` activity is recorded with the CA name

#### Scenario: Create on Free
- **WHEN** an EJBCA CA is created on a Fleet Free server
- **THEN** the request is rejected with the same Premium-required error used by other certificate authority types

### Requirement: Required configuration fields
An EJBCA CA SHALL require a name, the EJBCA REST base URL, a client certificate PKCS#12 and its password, the EJBCA CA name, the certificate profile name, the end entity profile name, and a common name template. The name SHALL contain only letters, numbers, and underscores and SHALL be unique among certificate authorities of the same type.

#### Scenario: Missing field on create
- **WHEN** a create request omits a required field
- **THEN** the request is rejected with a 422 that names the missing field

#### Scenario: Invalid name
- **WHEN** the name contains characters other than letters, numbers, and underscores
- **THEN** the request is rejected with a 422 and nothing is stored

#### Scenario: Duplicate name
- **WHEN** an EJBCA CA with the same name already exists
- **THEN** the request is rejected with a 409 conflict

### Requirement: Partial updates keep stored credentials
An update SHALL change only the fields it supplies. Omitted fields and secret fields set to the masked placeholder SHALL keep their stored values, and the merged result SHALL be validated as a whole. Supplying a PKCS#12 SHALL require its password and SHALL replace the stored client certificate and key after verification succeeds.

#### Scenario: Edit without credentials
- **WHEN** an admin changes the certificate profile name and sends no PKCS#12 or password
- **THEN** the change is saved, the stored client certificate and key are unchanged, and the connection is verified with the stored credential

#### Scenario: Credential rotation
- **WHEN** an admin supplies a new PKCS#12 and its password
- **THEN** the new certificate and key replace the stored ones only after verification succeeds

#### Scenario: Invalid replacement credential
- **WHEN** the supplied PKCS#12 has the wrong password or EJBCA rejects its certificate
- **THEN** the update is rejected and the stored credential is unchanged

#### Scenario: PKCS#12 without password
- **WHEN** an update supplies a PKCS#12 but no password
- **THEN** the update is rejected with a 422 naming the password field

### Requirement: Optional configuration fields
An EJBCA CA SHALL accept an optional PEM trust CA bundle used to verify EJBCA's HTTPS certificate, an optional list of user principal names (UPNs), and an optional username template. When no bundle is given, Fleet SHALL verify EJBCA's certificate against the system root store.

#### Scenario: Private TLS certificate
- **WHEN** EJBCA's HTTPS certificate is issued by a private CA and the trust CA bundle contains that CA
- **THEN** the connection is accepted

#### Scenario: Untrusted TLS certificate
- **WHEN** EJBCA's HTTPS certificate is not trusted by the bundle or the system roots
- **THEN** the save is rejected with a message that the server certificate could not be verified

### Requirement: Client credential handling
Fleet SHALL accept the client certificate as a PKCS#12 file in either BER or DER encoding, SHALL decode it at submission without running an external program, and SHALL store the certificate and private key without storing the PKCS#12 password. The private key SHALL be encrypted at rest.

#### Scenario: EJBCA-exported PKCS#12
- **WHEN** an admin uploads a PKCS#12 exported from EJBCA RA Web (BER encoded) with the correct password
- **THEN** the upload is accepted and the certificate and key are stored

#### Scenario: Wrong password
- **WHEN** the PKCS#12 password is wrong or the file is not a PKCS#12
- **THEN** the request is rejected with a 422 and nothing is stored

#### Scenario: Password not persisted
- **WHEN** a CA is saved
- **THEN** the PKCS#12 password is not stored anywhere

### Requirement: Secret masking
Reads of an EJBCA CA SHALL NOT return the private key or the PKCS#12 password. API and GitOps responses that echo secret fields SHALL return the masked placeholder used by other certificate authority types. Reads SHALL return `client_cert_expires_at`, a read-only RFC 3339 timestamp derived from the stored client certificate's expiry on every read; the field is ignored if supplied.

#### Scenario: List response
- **WHEN** an admin lists or gets certificate authorities
- **THEN** no EJBCA secret value appears in the response and `client_cert_expires_at` equals the stored client certificate's expiry

#### Scenario: Expiry after rotation
- **WHEN** an admin rotates the client certificate
- **THEN** subsequent reads return the new certificate's expiry

### Requirement: Connection verification on save
On create and update of connection settings, Fleet SHALL perform an authenticated request to EJBCA and reject the save if EJBCA does not accept the client certificate or if the configured EJBCA CA name is not among the CAs the client certificate may access. An unauthenticated status call alone SHALL NOT count as verification.

#### Scenario: Client certificate rejected
- **WHEN** EJBCA rejects the client certificate or the REST protocol is disabled
- **THEN** the save is rejected and the message tells the admin to check the certificate, its EJBCA role binding, and that REST Certificate Management is enabled

#### Scenario: CA name not visible
- **WHEN** the EJBCA CA name does not match a CA visible to the client certificate
- **THEN** the save is rejected and the message names the CA name that was not found

#### Scenario: Unreachable server
- **WHEN** the URL cannot be reached within the timeout
- **THEN** the save is rejected with a connection error and nothing is stored

### Requirement: Certificate enrollment
For each host, Fleet SHALL generate an RSA 2048 key and CSR in memory, set the CSR common name to the expanded common name template, include the expanded UPNs as Microsoft UPN subject alternative name entries when configured, request the certificate from EJBCA's REST API, and return the certificate with its key as a PKCS#12 with a random password.

#### Scenario: Successful enrollment
- **WHEN** a host receives a profile that references the EJBCA CA
- **THEN** EJBCA issues a certificate whose subject common name is the expanded common name template, and the host receives a PKCS#12 containing it and the generated key

#### Scenario: UPN included
- **WHEN** the CA has a UPN configured
- **THEN** the issued certificate contains that UPN as a subject alternative name

#### Scenario: Repeat enrollment for the same host
- **WHEN** the same username is enrolled again, for example after a resend
- **THEN** EJBCA issues a new certificate without requiring a pre-created or reset end entity

### Requirement: EJBCA username
Fleet SHALL enroll with the expanded username template as the EJBCA username, or with the expanded common name template when no username template is set. The username SHALL be independent of the certificate's common name.

#### Scenario: Username defaults to the common name
- **WHEN** the CA has a common name template and no username template
- **THEN** each host enrolls with a username equal to its expanded common name

#### Scenario: Username differs from the common name
- **WHEN** the username template is `$FLEET_VAR_HOST_HARDWARE_SERIAL` and the common name template is `$FLEET_VAR_HOST_END_USER_EMAIL_IDP`
- **THEN** each host enrolls under its serial as the username and receives a certificate whose common name is the user's email

### Requirement: Profile variables
Apple configuration profiles SHALL support `$FLEET_VAR_EJBCA_DATA_<CA_NAME>` (base64 PKCS#12) and `$FLEET_VAR_EJBCA_PASSWORD_<CA_NAME>` (its password) in a `com.apple.security.pkcs12` payload. Both variables for a CA SHALL be present together, and the CA SHALL exist.

#### Scenario: Valid profile
- **WHEN** an admin uploads a profile that uses both variables for an existing EJBCA CA in a pkcs12 payload
- **THEN** the profile is accepted

#### Scenario: Missing pair or unknown CA
- **WHEN** a profile uses only one of the two variables, or names a CA that does not exist
- **THEN** the upload is rejected with a message naming the missing variable or CA

#### Scenario: Wrong payload
- **WHEN** the variables appear outside a pkcs12 payload's data and password fields
- **THEN** the upload is rejected

#### Scenario: Delivery
- **WHEN** Fleet delivers the profile to a host
- **THEN** both variables are replaced with that host's certificate data and password

### Requirement: Fleet variables in CA fields
The common name template, username template, and UPNs SHALL accept the same Fleet variables DigiCert accepts for its certificate fields: host hardware serial, platform, UUID, end user IdP email, and the IdP variables. Other variables SHALL be rejected.

#### Scenario: Supported variable
- **WHEN** the common name template is `$FLEET_VAR_HOST_HARDWARE_SERIAL`
- **THEN** each host's certificate common name is that host's serial

#### Scenario: Unsupported variable
- **WHEN** a field contains an unsupported Fleet variable
- **THEN** the save is rejected naming the variable

### Requirement: Enrollment failure reporting
When EJBCA refuses an enrollment, Fleet SHALL mark the host's profile as failed with a detail that states the cause. Messages SHALL distinguish unknown CA, certificate profile, or end entity profile names (HTTP 400), missing role access (HTTP 403), and other EJBCA errors, and SHALL include EJBCA's error message.

#### Scenario: Unknown profile name
- **WHEN** EJBCA responds 400 that the certificate profile does not exist
- **THEN** the host profile detail reports that profile name does not exist in EJBCA

#### Scenario: Missing access rule
- **WHEN** EJBCA responds 403 not authorized to an end entity profile
- **THEN** the detail says the Fleet client certificate's EJBCA role lacks access, and does not claim the certificate is expired or revoked

### Requirement: Reissue on change
When an update changes the common name template, the username template, the UPNs, the EJBCA CA name, the certificate profile name, or the end entity profile name, Fleet SHALL resend the matching Apple profiles to hosts that use the CA so new certificates are issued. Updates to other fields SHALL NOT resend profiles. Profiles that use an IdP variable in these fields SHALL be resent when the host's IdP values change.

#### Scenario: Certificate field changed
- **WHEN** an admin changes the certificate profile name, through the UI, API, or GitOps
- **THEN** Fleet resends the profiles that reference the CA and each host receives a new certificate

#### Scenario: Connection field changed
- **WHEN** an admin rotates the client certificate or edits the trust bundle
- **THEN** no profile is resent and existing host certificates are unaffected

#### Scenario: IdP value changed
- **WHEN** a host's IdP email changes and the common name or username template uses it
- **THEN** Fleet resends the profile and the host gets a new certificate

### Requirement: Activities
Creating, editing, and deleting an EJBCA CA SHALL record `added_ejbca`, `edited_ejbca`, and `deleted_ejbca` respectively, each with the CA name, from the UI, API, and GitOps.

#### Scenario: Edit and delete
- **WHEN** an admin edits or deletes an EJBCA CA
- **THEN** the matching activity is recorded with the CA name, including when done through GitOps

### Requirement: GitOps
Fleet SHALL accept EJBCA CAs under `org_settings.certificate_authorities.ejbca`, apply them by name through batch apply with the same add, update, and delete semantics as DigiCert, and `fleetctl generate-gitops` SHALL emit them with the credential fields as explicit placeholders the admin must fill, since stored credentials cannot be exported.

#### Scenario: Apply
- **WHEN** a GitOps run adds, changes, or removes an `ejbca` entry
- **THEN** the CA is created, updated, or deleted and the corresponding activity is recorded

#### Scenario: Apply with credentials
- **WHEN** a GitOps run applies an `ejbca` entry
- **THEN** the entry supplies `client_p12` (base64) and `password`, normally as environment variable references, because Fleet cannot reconstruct them from stored material, and the connection is verified on every apply that changes the entry

#### Scenario: Generate
- **WHEN** an admin runs `fleetctl generate-gitops` on a server with an EJBCA CA
- **THEN** the entry is emitted with explicit TODO placeholders for `client_p12` and `password` named `$EJBCA_<CA_NAME>_CLIENT_P12` and `$EJBCA_<CA_NAME>_PASSWORD`, and no secret values

### Requirement: Host certificate tracking
Certificates issued through an EJBCA CA SHALL be recorded as managed by Fleet for the host, with the CA type `ejbca`, so the host's certificate list identifies them as Fleet-managed and renewal logic applies as it does for DigiCert.

#### Scenario: Issued certificate recorded
- **WHEN** a host receives an EJBCA certificate
- **THEN** a managed certificate record exists for the host and profile with type `ejbca`, serial, and validity dates

#### Scenario: Automatic renewal
- **WHEN** an EJBCA certificate nears expiry
- **THEN** Fleet reissues it through the same renewal job that renews DigiCert certificates, with no renewal ID needed in the profile

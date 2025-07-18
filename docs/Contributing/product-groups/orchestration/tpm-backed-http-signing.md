# TPM-backed HTTP signing for fleetd requests

## Overview

TPM-backed HTTP signing is a security feature that uses the device’s TPM 2.0 (Trusted Platform Module) hardware to securely generate and store cryptographic keys for signing HTTP requests. By ensuring that private keys never leave the TPM's secure boundary, this feature provides hardware-backed assurance that requests to the Fleet server originate from the same physical device that initially enrolled.

A **device identity certificate** is an X.509 certificate whose private key is bound to the TPM, enabling cryptographic proof of device identity.

This feature includes:

* **Device identity certificate enrollment** via SCEP (Simple Certificate Enrollment Protocol)
  * **Rate limiting** for certificate enrollment to prevent abuse
* **HTTP request signing** using TPM-protected keys
  * **Server-side signature verification** with certificate validation

Together, these mechanisms establish a strong trust foundation for authenticated communication between `fleetd` and the Fleet server.

**Also known as:**

* Fleet host identity
* Hardware-backed device identity
* TPM-based request signing
* Secure request signing with TPM
* Trusted device authentication

## Architecture

### Reference links

- [TPM 2.0 Library specification](https://trustedcomputinggroup.org/resource/tpm-library-specification/)
- [TPM 2.0 Key Files](https://www.hansenpartnership.com/draft-bottomley-tpm2-keys.html) - de facto standard used by OpenConnect VPN and other tools
- [RFC 9421 - HTTP Message Signatures](https://datatracker.ietf.org/doc/html/rfc9421)
- [RFC 8894 - Simple Certificate Enrolment Protocol](https://datatracker.ietf.org/doc/html/rfc8894) (SCEP)

### Components

The TPM-backed HTTP signing feature consists of several key components:

#### orbit components

fleetd is the Fleet agent that includes orbit (the main agent process), osquery, and Fleet Desktop.

1. **Secure hardware interface** - Hardware-agnostic Go abstraction for TPM (and, in the future, Apple's Secure Enclave)
2. **TPM 2.0 implementation** - Linux-specific TPM 2.0 integration with automatic ECC curve selection
3. **SCEP client** - Certificate enrollment client for obtaining device identity certificates
4. **HTTP signing proxy** - Proxy component that intercepts osquery traffic and adds HTTP signature headers
5. **HTTP signing integration** - Direct HTTP signature support for orbit's own communications

#### Server components
1. **SCEP server interface** - Certificate Authority (CA) with dedicated keys for issuing device identity certificates
   * **Rate limiting** - Configurable cooldown periods to prevent certificate enrollment abuse
   * **Certificate management** - Automatic certificate lifecycle management including revocation and cleanup
2. **HTTP signature verification** - Server-side verification of TPM-signed HTTP requests and associated certificates

### Architecture diagrams

```mermaid
---
title: TPM-backed HTTP signing (high level)
---
flowchart TD
    subgraph Host
        TPM[TPM 2.0 hardware]
        Cert[Device identity certificate]
        fleetd[fleetd agent]
    end

    subgraph Fleet server
        Server[Fleet server]
        Validate[Validate signed request]
        CA
    end

    TPM --> Cert
    Cert --> fleetd
    fleetd -->|Signs HTTP request| Server
    Server --> Validate

    subgraph CA
       SCEP[SCEP endpoint]
    end

    fleetd -->|Request cert via SCEP| SCEP
    SCEP -->|Issues cert| Cert
```

```mermaid
---
title: TPM-backed HTTP signing
---
sequenceDiagram
    autonumber
    participant orbit
    participant tpm as TPM 2.0
    participant osquery
    participant server as Fleet server
    orbit->>orbit: Load cert if exists
    alt no cert
       orbit->>+tpm: Create private key
       tpm-->>-orbit: Private key handle
       orbit->>+tpm: Sign CSR
       tpm-->>-orbit: Signature
       orbit->>+server: Get cert using SCEP
       server-->>-orbit: Cert singed by CA
       orbit->>orbit: Save cert and TPM keys
    end
    par orbit requests
       orbit->>+tpm: Sign HTTP request
       tpm-->>-orbit: Signature
       orbit->>+server: Signed HTTP request
       server->>server: Verify cert is valid and not expired
       server->>server: Verify signature using cert pub key
       server-->>-orbit: Response (unsigned)
    and osquery requests
       osquery->>+orbit: HTTP request to proxy
       orbit->>+tpm: Sign HTTP request
       tpm-->>-orbit: Signature
       orbit->>+server: Signed HTTP request
       server->>server: Verify cert is valid and not expired
       server->>server: Verify signature using cert pub key
       server-->>-orbit: Response (unsigned)
       orbit-->>-osquery: Response from proxy
    end
```

## TPM 2.0 implementation

### Hardware requirements

- **Linux Platform**: TPM 2.0 support is currently Linux-only
- **TPM Device**: Requires `/dev/tpmrm0` (resource manager), which was added in Linux kernel 4.12 (July 2, 2017) and adopted in enterprise around 2018-2019. Compatible with TPM 2.0 hardware, firmware, or virtual implementations (vTPM).

### Key generation

The TPM implementation creates a transient parent key, which must use the same template when the key is loaded.

The TPM implementation automatically selects the best available ECC curve for the child key:

1. **Preferred**: ECC P-384 (NIST P-384) with SHA-384 - modern, fast, and stronger than Apple MDM's RSA 2048
2. **Fallback**: ECC P-256 (NIST P-256) with SHA-256 - still stronger than RSA 2048

The implementation determines TPM's P-384 support by attempting to create a test key, and falling back to P-256 if unsupported.

### Key storage

Keys are saved as to the filesystem using [TPM 2.0 Key Files](https://www.hansenpartnership.com/draft-bottomley-tpm2-keys.html) format, which includes:
- Private key blob
- Public key blob
- Parent key template

Filename used is `host_identity_tpm.pem`

## SCEP certificate enrollment

### Overview

The SCEP (Simple Certificate Enrollment Protocol) client enables fleetd to obtain device identity certificates from a Certificate Authority. The certificates are used to establish device identity and can be used in conjunction with HTTP signing for enhanced authentication.

### Certificate enrollment process

The SCEP enrollment process follows these steps:

1. **CA Certificate Retrieval**: Fetch the CA certificate from the SCEP server
2. **Key Generation**: Create an ECC key pair in the TPM (P-384 preferred, P-256 fallback)
3. **CSR Creation**: Generate a Certificate Signing Request using the TPM key
4. **Temporary RSA Key**: Create a temporary RSA key for SCEP protocol encryption/decryption
5. **SCEP Request**: Send the CSR to the SCEP server with challenge authentication
   * challenge is the enrollment secret
6. **Rate Limit Check**: Server validates that the host is not requesting certificates too frequently
7. **Certificate Retrieval**: Decrypt and parse the issued certificate
8. **Certificate Storage**: Save the certificate as `host_identity.crt`

#### Key usage separation

The SCEP implementation uses a hybrid approach for cryptographic operations:

- **ECC Key (TPM)**: Used for signing the Certificate Signing Request (CSR)
- **RSA Key (Temporary)**: Used for SCEP protocol encryption and decryption
- **Final Certificate**: Contains the ECC public key but is signed by the CA

This separation is necessary because:
- ECC keys cannot perform encryption/decryption operations required by SCEP
- The TPM-generated ECC key provides the actual device identity
- The temporary RSA key is only used for SCEP protocol compliance

## HTTP signature

### Architecture overview

The TPM-backed HTTP signing operates at two levels within fleetd:

1. **Direct Integration**: orbit's own HTTP communications are signed directly using TPM keys
2. **Proxy Integration**: A proxy component intercepts osquery traffic and adds HTTP signature headers

This proxy approach allows osquery (which doesn't natively support HTTP signatures or TPM) to benefit from TPM-backed authentication without requiring modifications to osquery itself.

The TPM implementation produces RFC 9421-compatible ECDSA signatures.

### Implementation details

The HTTP signing implementation uses a `signerWrapper` pattern that wraps HTTP clients to automatically sign requests:

```go
// signerWrapper wraps an HTTP client to add signing capabilities
signerWrapper := func(client *http.Client) *http.Client {
    return httpsig.NewHTTPClient(client, signer, nil)
}
```

This approach allows existing HTTP client code to be enhanced with signing capabilities without requiring extensive modifications to the codebase.

### HTTP signature fields

Both direct and proxy signing use the same HTTP signature fields:

- **`@method`**: HTTP method (i.e., GET, POST, etc.)
- **`@authority`**: Hostname (i.e., example.com)
- **`@path`**: URL path (i.e., /api/v1/resource)
- **`@query`**: Query params (i.e., foo=bar)
- **`content-digest`**: SHA-256 digest of request body

> **Note**: We did not include the scheme (e.g., http, https) as part of the signature to prevent potential hard-to-debug issues with proxies and HTTP forwarding. We did not include Content-Type header in the signature because not all requests have this header.

Additional metadata included:
- **`keyid`**: Identifier for the signing key, which maps to identity certificate's serial number in uppercase hexadecimal format
- **`created`**: Timestamp of signature creation
- **`nonce`**: Random value for replay protection

The `created` and `nonce` fields can be used in the future to prevent replay attacks. One way to use them would be:
- server checks that `created` is within 10 minutes of current server time (since these fields are included in the signature, we know they have not been tampered with)
- server checks that `nonce` value has not been used within the last 10 minutes

> **Note**: Apple MDM prevents most (but not all) replay attacks by using a unique CommandUUID.

### Traffic flow

```
┌─────────────┐    ┌──────────────┐    ┌─────────────┐
│   osquery   │───▶│ fleetd proxy │───▶│ Fleet Server│
└─────────────┘    └──────────────┘    └─────────────┘
                          │
                          ▼
                   ┌──────────────┐
                   │  TPM Signing │
                   └──────────────┘

┌─────────────┐    ┌──────────────┐    ┌─────────────┐
│    orbit    │───▶│ HTTP Client  │───▶│ Fleet Server│
└─────────────┘    └──────────────┘    └─────────────┘
                          │
                          ▼
                   ┌──────────────┐
                   │  TPM Signing │
                   └──────────────┘
```

- **osquery → fleetd proxy**: osquery sends unsigned requests to the local proxy
- **fleetd proxy → TPM**: Proxy uses TPM to sign the intercepted requests
- **fleetd proxy → Fleet Server**: Proxy forwards signed requests to the server

## Configuration

### Client configuration

Enable TPM-backed HTTP signing when packaging or running fleetd:

```bash
# Package with TPM signing enabled
fleetctl package --fleet-managed-client-certificate ...

# Run orbit with TPM signing enabled
orbit --fleet-managed-client-certificate ...
```

### Server configuration

No additional server configuration is required. The SCEP endpoint is automatically available on Fleet servers with:
- **Premium license**
- **Configured server private key**

The server automatically verifies that:
- Requests with HTTP message signatures match the certificate public key and the host node key
- Requests without HTTP message signatures do not have associated host identity certificates

#### Rate limiting configuration

Certificate enrollment rate limiting is configurable through the Fleet server configuration, using the same setting as for host enrollment rate limiting:

```yaml
osquery:
  enroll_cooldown: 5m
```

When rate limiting is enabled:
- Hosts requesting certificates too frequently receive HTTP 429 (Too Many Requests) responses
- Rate limiting applies per host based on the certificate Common Name (CN)
- Different hosts are not affected by each other's rate limits
- Rate limiting uses the same configuration as host enrollment cooldown

## Future enhancements

Additional features that may be implemented in future releases:

1. **Key Rotation/Renewal**: Automatic key rotation policies and certificate renewal
2. **One-time enrollment secret**: This provides additional security to make sure an unauthorized device cannot get an identity certificate and enroll in Fleet.
3. **Windows Support**: TPM support for Windows platforms using TBS (TPM Base Services)
4. **Apple Secure Enclave**: Integration with Apple's Secure Enclave for macOS devices
5. **Fleet server visibility**: Allow IT admin to see which hosts have host identity certificates. For example, we can add a field to `orbit_info` table and IT admin could set up a policy to make sure all hosts have certificates.
6. **Multiple Key Support**: Support for multiple signing keys and certificates, like a separate key for WiFi/VPN.
7. **Hardware Attestation**: TPM-based device attestation and platform integrity
8. **SCEP Extensions**: Support for additional SCEP features and external CA integrations
9. **ACME**: Use ACME protocol instead of SCEP to get a certificate.

## Troubleshooting

### TPM hardware issues

1. **TPM device not found**
   - Verify TPM is enabled in BIOS/UEFI
   - Check kernel TPM driver is loaded
   - Ensure device files exist with proper permissions

2. **Permission denied**
   - Add user to `tss` group for TPM access
   - Check device file permissions (`/dev/tpmrm0`)

3. **Key creation failures**
   - Verify TPM is not locked or in failure mode
   - Clear TPM if necessary (will lose existing keys)
   - Check available TPM resources

### Certificate enrollment issues

1. **SCEP server connection issues**
   - Verify SCEP server URL is accessible (and your Fleet server has this feature)
   - Check network connectivity and firewall rules

2. **Challenge password authentication**
   - Confirm challenge password is correct (a valid enrollment key)

3. **Certificate enrollment failures**
   - Review SCEP server logs for rejection reasons
   - Check if rate limiting is causing HTTP 429 responses

4. **Rate limiting issues**
   - Check if the host is requesting certificates too frequently
   - Verify the configured cooldown period in server configuration
   - Monitor for HTTP 429 responses indicating rate limiting

### General debugging

Enable fleetd/server debug logging to troubleshoot issues.

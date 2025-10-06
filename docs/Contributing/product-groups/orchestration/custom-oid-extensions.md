# Custom OID Extensions

## Overview

Object Identifiers (OIDs) are globally unique identifiers used in various cryptographic standards, particularly in X.509 certificates and ASN.1 structures. Custom OID extensions allow organizations to embed proprietary or application-specific data within certificates and other cryptographic objects while maintaining standards compliance.

Fleet uses IANA private enterprise number **63991** for all custom extensions. This ensures our extensions don't conflict with other organizations' implementations.

## Background

### What are OIDs?

An OID is a hierarchical identifier written as a dot-separated sequence of numbers (e.g., `1.3.6.1.4.1.63991`). The hierarchy ensures global uniqueness:

- `1.3.6.1` - ISO/ITU-T jointly assigned, Internet subtree
- `1.3.6.1.4` - Private enterprises
- `1.3.6.1.4.1` - IANA-assigned enterprise numbers
- `1.3.6.1.4.1.63991` - Fleet's private enterprise number

### Why use custom OID extensions?

Custom OID extensions are valuable for:

1. **Embedding application-specific metadata** in certificates without breaking standards compliance
2. **Extending standard protocols** with proprietary features while maintaining interoperability
3. **Creating domain-specific certificate policies** that standard extensions don't cover
4. **Implementing custom authentication mechanisms** that require certificate-bound data
5. **Enabling certificate lifecycle management** features like custom renewal processes

### X.509 certificate extensions

In X.509 certificates, extensions appear in the v3 extensions field. Each extension contains:
- An OID identifying the extension type
- A critical flag (indicating whether the extension must be understood)
- The extension value (encoded data)

## Fleet's Custom Extensions

All Fleet-specific OID extensions follow the pattern `1.3.6.1.4.1.63991.x.y`.

### Certificate Renewal Extension

- **OID**: `1.3.6.1.4.1.63991.1.1`
- **Purpose**: Proves possession of the current ECC private key during host identity certificate renewal
- **Critical**: No
- **Format**: JSON object containing:
  ```json
  {
    "sn": "0x1b",      // Hex-encoded serial number of the old certificate
    "sig": "MEUCIQ..." // Base64-encoded ECDSA signature
  }
  ```
- **Usage**: SCEP certificate renewal (because our SCEP library doesn't support ECC certificate renewal)
- **Implementation**: `RenewalExtensionOID` in `ee/server/service/hostidentity/types/host_identity_certificates.go`

## Adding New Custom Extensions

### 1. Assigning an OID

Use the next available number under Fleet's namespace:
- `1.3.6.1.4.1.63991.1.x` - Host identity certificate-related extensions
- `1.3.6.1.4.1.63991.2.x` - Future: ??? extensions

### 2. Documentation

Document new extensions in this file with:
- OID value
- Purpose and use case
- Critical flag setting
- Data format (ASN.1 structure, JSON, etc.)
- Validation rules
- Implementation location in codebase
- Example usage

## Security Considerations

1. **Validation**: Always validate extension data to prevent injection attacks
2. **Size limits**: Enforce reasonable size limits on extension data
3. **Critical flag**: Use sparingly (only when the extension must be understood)
4. **Privacy**: Don't embed sensitive information that shouldn't be in certificates
5. **Compatibility**: Test with various certificate parsers to ensure compatibility

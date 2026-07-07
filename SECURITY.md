# Security Policy

## Reporting a Vulnerability

Fleet runs a Vulnerability Disclosure Program (VDP) on Bugbop:

**[https://bugbop.com/programs/b5f2f20e-fe4d-466b-a474-6db65b4d2bb3](https://bugbop.com/programs/b5f2f20e-fe4d-466b-a474-6db65b4d2bb3)**

Please review the program scope and rules of engagement on Bugbop before submitting. Researchers can also report vulnerabilities directly to security **at** fleetdm.com (PGP key below) for coordinated, non-public disclosure.

Fleet endeavors to acknowledge and fix any reported vulnerabilities ASAP. Acknowledgement is typically within 1 business day, and patches usually go out within 5 business days (depending on severity and timing).

### Scope

In scope:
- Fleet product source code: [github.com/fleetdm/fleet](https://github.com/fleetdm/fleet)
- Fleet REST API documentation: [fleetdm.com/docs/api/rest-api](https://fleetdm.com/docs/api/rest-api)

Out of scope:
- Marketing pages, blogs, and landing pages on fleetdm.com
- Third-party hosted services (unless they directly impact a primary in-scope asset)
- Physical offices and infrastructure
- Employee social media accounts

Reports that are typically not eligible:
- Missing HTTP security headers (unless they lead to a proven, demonstrated vulnerability)
- Theoretical vulnerabilities without proof of exploitation
- Automated tool output without clear impact evidence
- Self-XSS requiring significant user interaction
- Issues solely affecting outdated browsers

### PGP Key

To encrypt vulnerability reports before sending them, please use this [PGP key](https://keys.openpgp.org/vks/v1/by-fingerprint/82F2AF19547E462A4605D53801B2575E46766EBE).

The fingerprint of the key is `82F2 AF19 547E 462A 4605  D538 01B2 575E 4676 6EBE`.

### Vulnerability tracking

GitHub issues concerning vulnerabilities will be tagged with the **security** label to differentiate them from other issues and maintain SOC2 compliance.  

See [security/README.md](./security/README.md) for more information on our process to keep Fleet products secure.

### Compatibility

Fleet reserves the right to make breaking changes for security. Security fixes may introduce backward-incompatible changes and may be released in minor or patch versions.

# ISO 27001 compliance policies

Fleet's ISO 27001 policies help organizations demonstrate compliance with the ISO/IEC 27001:2022 Information Security Management System (ISMS) standard. These policies verify endpoint-level security controls that map to ISO 27001 Annex A controls.

These policies are intended to assist companies selling software in Europe and other markets where ISO 27001 certification is expected or required.

## Annex A control coverage

The policies cover the following ISO 27001:2022 Annex A control areas:

| Annex A Control | Description | Policies |
|---|---|---|
| A.5.14 | Information transfer | iCloud Desktop and Document sync |
| A.5.15 | Access control | Guest account, Guest access to shared folders |
| A.5.17 | Authentication information | Unencrypted SSH keys |
| A.8.1 | User endpoint devices | Screen lock, Lock screen after inactivity, MDM enrollment |
| A.8.2 | Privileged access rights | Guest account disabled |
| A.8.5 | Secure authentication | Screen lock, Password minimum length, Automatic login disabled |
| A.8.7 | Protection against malware | Antivirus healthy, Gatekeeper enabled |
| A.8.8 | Management of technical vulnerabilities | OS up to date, Automatic updates, SMBv1 disabled, Security updates |
| A.8.9 | Configuration management | System Integrity Protection |
| A.8.11 | Data masking | iCloud sync disabled |
| A.8.15 | Logging | Firewall logging |
| A.8.20 | Networks security | Firewall enabled, Internet sharing blocked, LLMNR disabled, Remote login disabled |
| A.8.21 | Security of network services | Firewall enabled |
| A.8.24 | Use of cryptography | Full disk encryption, SSH keys encrypted |

## Platform coverage

- **macOS**: 22 policies
- **Windows**: 10 policies
- **Linux**: 4 policies
- **Cross-platform** (macOS, Windows, Linux): 2 policies

## Usage

Apply these policies using `fleetctl`:

```sh
fleetctl apply -f ee/iso27001/iso27001-policy-queries.yml
```

### Template policies

Policies tagged with `template` contain version numbers or other values that should be updated to match your organization's requirements. For example, the "Operating system up to date" policy should be updated with the minimum OS version your organization requires.

## Limitations

ISO 27001 includes many organizational and procedural controls (e.g., risk assessment processes, supplier management, incident management procedures) that cannot be verified through device policies alone. These policies cover only the technical endpoint controls that can be checked via osquery. A complete ISO 27001 compliance program requires additional organizational measures beyond what Fleet policies can verify.

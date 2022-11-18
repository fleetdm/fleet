## Glossary

### SCEP: Simple Certificate Enrollment Protocol

SCEP is a protocol that allows devices to get a certificate from a CA (in our
case, the Fleet server). This certificate is used later on to authenticate to
the MDM server.

Resources:

- [SCEP summary](#scep-summary) below
- [RFC 8894][https://datatracker.ietf.org/doc/html/rfc8894]

### DEP: Device Enrollment Program

A device enrolled via DEP prompts the user to enroll in MDM during the initial
device setup process (right after macOS is installed.)

DEP is also called "automatic" enrollment because it doesn't require user
action to download and activate a profile like [manual
enrollment](#manual-enrollment) does.

### Manual enrollment

It's a method to enroll a device to an MDM server by manually getting
(generally by downloading from an URL) an [enrollment
profile](#enrollment-profile) and installing it.


### ABM: Apple Business Manager

Interface to administer Devices and MDM servers, mainly used for [DEP
enrollment](#dep-enrollment).

Can be accessed at https://business.apple.com/ .

### APNs: Apple Push Notification Service 

MDM uses the Apple Push Notification Service (APNs) to deliver a "wake up"
message to managed devices. The device then connects to the MDM server to
retrieve [commands](#commands) and return results.

APNs are servers managed by Apple, the MDM server needs a certificate signed by
Apple to authenticate with them.

Resources:

- [MDM Protocol sumary](#mdm-protocol-summary)
- [MDM protocol specification][https://developer.apple.com/business/documentation/MDM-Protocol-Reference.pdf].

### Profile

A configuration profile is an XML file that allows to distribute configuration
information, for example: restrictions on device features, VPN settings, etc.

Profiles have the `.mobileconfig` extension and can be administered in a device
from "Settings > Profiles."

Resources:

- [Configuration Profile Reference](https://developer.apple.com/business/documentation/Configuration-Profile-Reference.pdf).

### Enrollment profile

An enrollment profile is a [profile](#profile) that contains special directives
to enroll a device to an MDM server.

For [DEP enrollment](#dep-device-enrollment-program) this profile is
automatically sent an installed into the device.

For [manual enrollment](#manual-enrollment) the profile needs to be downloaded
and installed by the user.

### Commands

After a device is enrolled, an MDM server can send commands to be executed in
the device (e.g: install an application, shut down the device, etc.)

The server first sends a [push
notification](#apons-apple-push-notification-service), then the device polls
the server for the command, processes the command, and reports the command
results to the server.

### CSR: Certificate Signing Request 

Issued by the server that needs validation from a signing authority, the
request has the public key in the pair and information about the server
(organization name, etc)

The CSR itself is usually created in a Base-64 based PEM format

### PKI: Public Key Infrastructure

Allows authenticating users and devices. The basic idea is to have one or more
trusted parties digitally sign documents certifying that a particular
cryptographic key belongs to a particular user or device. The key can then be
used as an identity for the user.

Resources:

- https://www.ssh.com/academy/pki
- https://en.wikipedia.org/wiki/Public_key_infrastructure

### CA: Certificate Authority

The primary role of the CA is to digitally sign and publish a public key
bound to a given user.

This is done using the CA's own private key, so that
trust in the user key relies on one's trust in the validity of the CA's key.

Resources:

- https://en.wikipedia.org/wiki/Public_key_infrastructure#Certificate_authorities

## SCEP summary

SCEP is a [PKI](#pki-public-key-infrastructure) protocol that allows devices to
request certificates from a [CA](#ca-certificate-authoriy) (in our context, the
Fleet server acts as a CA) that will be later used to authenticate with the MDM
server (in our context, the Fleet server also acts as the MDM server.)

To enroll, a client provides a distinguished name and a public key, and the
server responds with a X.509 certificate.

The CA server might also request a `challengePassword`, which is a shared
secret that the server uses gate access to certificates, its omission allows
for unauthenticated authorisation of enrolment requests.

More generally, the protocol is specified in [RFC
8894](https://datatracker.ietf.org/doc/html/rfc8894) and allows:

- [CA](#ca-certificate-authority) public key distribution
- Certificate enrolment and issue
- Certificate renewal
- Certificate query
- Query (not perform) certificate revocation

## MDM Protocol summary

This is a rough summary of the [MDM Protocol
Reference](https://developer.apple.com/business/documentation/MDM-Protocol-Reference.pdf).

The protocol is composed by the MDM Check-in protocol and the main MDM protocol.

### MDM Check-in Protocol

Used during initialization, validates if the device can be enrolled and
notifies the server.

TODO: finish summary

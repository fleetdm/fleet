# What is device attestation, and why does it matter for Apple enrollment?

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/o1YFhlzsRlg?si=OKiMLcbc9OyK2AAJ" title="0" allowfullscreen></iframe>
</div>
<p></p>

If you've set up Automated Device Enrollment (ADE) for your organization, you've already solved
the "how do devices get enrolled" problem. But there's a separate question worth asking: how do
you know the device that enrolled is actually the device it claims to be?

That's where device attestation comes in. Fleet 4.84.0 adds support for hardware-attested MDM
enrollment for Apple Silicon Macs via ADE, so let's talk about what attestation means
and why it's worth enabling.

## The problem attestation solves

ADE ties a device to your MDM server before it ever reaches an end user. When someone powers on 
a new Mac, it checks in with Apple. The device learns which MDM it belongs to and enrolls automatically.
It's a smooth experience, but enrollment itself doesn't cryptographically prove the device's
identity.

Historically, MDMs have relied on identifiers like serial numbers and UDIDs to track devices.
Those values are accurate in normal operation, but they're also attributes that software can
misrepresent. A device can claim to have any serial number it wants.

Device attestation closes that gap. Instead of trusting what a device tells you, you verify
what Apple can prove about it.

## How Apple device attestation works

Apple Silicon Macs include a Secure Enclave, a dedicated coprocessor that stores cryptographic
keys in hardware. Keys generated in the Secure Enclave can't be exported. Operations that use
them happen inside the enclave itself.

During attestation, the device generates a key pair inside the Secure Enclave. It then requests
an attestation certificate from Apple. That certificate binds the key to hardware attributes
like serial number, UDID, and chip type. Apple can verify these attributes because the device
was manufactured by Apple and its provenance is known.

The attestation certificate Apple issues is rooted in Apple's Enterprise Attestation CA. Your
MDM verifies that chain to confirm the device is genuine. The attestation certificate itself
isn't the client certificate the device uses to identify itself to MDM. Once the attestation
checks out, the MDM's own ACME server issues that client certificate.

The result: you know the device is a genuine Apple device, you know the hardware attributes
are accurate, and the client certificate your MDM issued is bound to a hardware key that can't
leave the Secure Enclave.

## The ACME protocol

Apple uses the ACME (Automated Certificate Management Environment) protocol to handle certificate
issuance for device attestation. ACME is a standard protocol originally designed for automating
TLS certificate management. Apple adapted it for device identity.

Before ACME, MDM enrollment certificates were issued using SCEP (Simple Certificate Enrollment
Protocol), which doesn't support hardware-bound keys or attestation. ACME replaces SCEP for
devices that qualify, and it's the mechanism that makes hardware-attested enrollment possible.

When Fleet sends an ACME configuration to an enrolling device, the device:

1. Requests a challenge token from Fleet's ACME server
2. Generates a key pair in the Secure Enclave
3. Requests an attestation certificate from Apple that binds the key to the device's hardware
   attributes
4. Presents the attestation certificate back to Fleet for validation

Fleet validates the response a few ways: the leaf certificate must chain up to Apple's
Enterprise Attestation CA (however many intermediates sit in between), the certificate
extensions must include the challenge token Fleet issued, and the hardware attributes in the
extensions (like serial number) must match what Fleet expects. Once those checks pass, Fleet's
ACME server issues the client certificate the device uses for MDM identity going forward.

## What attestation proves (and what it doesn't)

Device attestation tells you:

- This is a real Apple device, not a virtual machine misrepresenting itself
- The serial number, UDID, and hardware attributes in the certificate are accurate
- The private key is hardware-bound and can't leave the device

It doesn't tell you:

- Whether the device is in a good security posture (that's what osquery and MDM compliance
  checks are for)
- Whether the right user is operating the device
- Anything about software running on the device

Attestation establishes hardware identity. Posture checks, access controls, and compliance 
policies all benefit from that trustworthy foundation. But attestation itself is specifically 
about proving the device is what it says it is.

## How Fleet supports device attestation

Starting in Fleet 4.84.0, Fleet Premium customers can require hardware attestation for ADE
enrollments on Apple Silicon Macs running macOS 14 or later.

When you enable **Require hardware attestation** in Fleet's MDM settings, Fleet does two things:

1. Sends an enrollment profile that includes an ACME hardware-bound certificate configuration
2. Requires the device to pass an Apple device attestation challenge before enrollment completes

Devices that fail the attestation challenge aren't allowed to enroll. This is an explicit gate,
not a soft check.

Devices that already enrolled via SCEP don't need to re-enroll. Qualifying devices (Apple
Silicon Macs on macOS 14+ from ADE) receive ACME certificates on their next renewal cycle.

Intel Macs fall back to SCEP because they don't have a Secure Enclave to bind keys to. Apple's
ACME protocol also supports iPhones and iPads with an A11 Bionic chip or later, but Fleet's
current implementation only covers Apple Silicon Macs. iPhones and iPads continue to enroll
via SCEP for now.

When a device enrolls with a hardware-attested certificate, Fleet shows **MDM attestation: Yes**
in host vitals. If a host isn't attested, the field doesn't appear. That keeps the UI clear
for devices where attestation applies.

The setting is available in Fleet's UI and can be managed via GitOps. When GitOps mode is
enabled, the checkbox in the UI is disabled, which keeps your configuration source of truth in
version control.

## Why this matters in practice

If your organization is moving toward zero trust access models, device identity is foundational.
You can't make meaningful trust decisions about a device if you can't verify its identity.
Hardware attestation gives you an identity that's rooted in silicon, not just MDM enrollment
records.

It also matters for regulated environments. Some environments require proof that only managed, 
genuine Apple devices access certain resources. Attestation provides evidence anchored in hardware.

For Mac admins, attestation is most meaningful when paired with something that acts on it. That 
could be an identity provider that gates access based on device posture. It could also be a 
network access control system that requires proof of enrollment. Fleet now gives you the verified 
device identity to feed those systems.

For devices that already enrolled, you don't have to do anything disruptive.
Enable the setting and qualifying devices upgrade to ACME on their next renewal cycle.


---
* [Learn more about the Fleet 4.84.0 release](https://fleetdm.com/releases/fleet-4-84-0)
* [Learn more about Fleet](https://fleetdm.com)



<meta name="articleTitle" value="What is device attestation, and why does it matter for Apple enrollment?">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-04-24">
<meta name="description" value="Fleet 4.84.0 adds hardware-attested MDM enrollment for Apple Silicon Macs. Here's what device attestation is and why it matters.">

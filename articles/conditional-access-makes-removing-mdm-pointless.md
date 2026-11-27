# Conditional access makes removing the MDM pointless

*Mac admins used to worry about employees removing MDM. Conditional access mostly solved that. The problem worth worrying about now is quieter: a work certificate copied onto a personal laptop.*

## Key takeaways

- **Removing MDM only locks the user out.** Once single sign-on (SSO) checks enrollment, a Mac that leaves MDM loses email, chat, and every other work app at the next sign-in.
- **Absolute solves PC problems.** Its firmware persistence, lost-laptop tracking, and blue screen recovery are built for PCs. On a Mac, Apple's Automated Device Enrollment already keeps MDM in place.
- **Jamf, Intune, and Fleet can all enforce this.** They differ in which operating systems and identity providers they support, and in how a device proves who it is.
- **The real gap is a copied certificate.** An employee who copies their sign-in certificate onto a personal Mac never removes MDM. Depending on your setup, the personal Mac can still pass.
- **A key in the Secure Enclave closes it.** Platform SSO with Entra, or Okta FastPass, keeps the device's key in hardware, where it can't be copied.

<a purpose="cta-button" href="https://fleetdm.com/guides/conditional-access">Explore conditional access</a>

Every few months, someone in a Mac admins forum asks how to stop employees from removing MDM. A developer with admin rights deletes the enrollment profile, or a contractor wipes their MacBook and skips enrollment, and IT finds out weeks later.

The replies split two ways: take away admin rights, or buy something that can't be removed, like Absolute. Both made sense once. Most IT teams I talk to have since stopped worrying about MDM removal, and they're right to. But it took me a while to see what they should be worrying about instead.

## Why removing MDM used to matter

MDM is how IT knows a Mac is encrypted, patched, and running the right security tools. Remove it, and IT is blind.

Apple solved most of this years ago. Macs bought through Apple Business enroll during setup with [Automated Device Enrollment](https://support.apple.com/guide/deployment/automated-device-enrollment-management-dep73069dd57/web), and the enrollment can be made non-removable. Fleet's default enrollment profile does this. Wipe the Mac, and Setup Assistant enrolls it again. The gap is Macs enrolled by hand, and Fleet can [block those entirely](https://fleetdm.com/docs/configuration/yaml-files#controls) with `only_allow_apple_business_enrollment`.

Absolute's pitch is firmware persistence: an agent [built into PCs from close to 30 manufacturers](https://www.absolute.com/en/why/persistence) that reinstalls itself if someone deletes it. Teams mostly buy it to find, freeze, and recover lost or stolen laptops, which Absolute has done since it was called Computrace. It can also [repair PCs stuck on a blue screen](https://www.absolute.com/blog/absolute-security-automates-remediation-for-crowdstrike-and-other-windows-bsod-incidents), like the [8.5 million](https://blogs.microsoft.com/blog/2024/07/20/helping-our-customers-through-the-crowdstrike-outage/) that crashed during the 2024 CrowdStrike outage. Those are PC problems. Absolute runs on Macs as a regular agent, and Apple's own tools already keep Macs enrolled and recoverable.

## Conditional access made removal pointless

Conditional access flips the question. Instead of stopping someone from removing MDM, it makes removal useless. Each time someone signs in to a work app, your identity provider (IdP) checks that the device is enrolled. Remove MDM, and Slack stops working.

There are three common ways to check:

1. **A device certificate.** MDM installs a certificate, and the IdP asks the browser for it at sign-in.
2. **The IdP's own app.** Okta Verify or Microsoft's Company Portal looks for a signal only MDM delivers, and reports back.
3. **A direct question.** The IdP asks the MDM whether the device is enrolled and compliant. Entra works this way with Intune and Jamf, and Okta works this way with Fleet.

Here's where the two big names in Mac management, and Fleet, stand:

| | Mac | Windows | Linux |
|---|---|---|---|
| Jamf | Entra or Okta | No (Jamf doesn't manage Windows) | No |
| Microsoft Intune | Entra | Entra | Entra, for web apps in Edge on Ubuntu and Red Hat |
| Fleet | Okta, Entra, Duo, or PingFederate | Okta, Entra, Duo, or PingFederate | Duo or PingFederate, with Okta [coming soon](https://github.com/fleetdm/fleet/issues/53286) |

Sources: [Jamf](https://trusted.jamf.com/docs/win-modern-device-management), [Intune](https://learn.microsoft.com/en-us/intune/fundamentals/platform-guide-linux), [Fleet](https://fleetdm.com/guides/conditional-access).

Any of these turns MDM removal from a security incident into a help desk ticket. So what's left to worry about?

## The problem nobody's talking about

Picture an engineer with a work MacBook and a personal one. They'd rather read email on the personal one. They don't remove MDM from anything. They copy the sign-in certificate from the work Mac to the personal Mac.

The threat was never a Mac leaving MDM. It's a personal Mac borrowing the identity of an enrolled one. Whether that works depends on how your devices prove who they are.

**Fleet as an Okta factor.** Fleet finds the host by the certificate's serial number and checks that host's policies. A copied certificate points back to the work Mac, which is enrolled and patched, so the personal Mac passes every check. Copying takes effort: Fleet marks the key non-exportable on macOS and keeps it in the TPM on Windows. But on a Mac, the key sits in the login keychain, not the Secure Enclave, and the employee knows their own keychain password. We've filed this [as a bug](https://github.com/fleetdm/fleet/issues/60032). Fleet is open source, and anyone with an AI assistant could find it in the code in minutes, so we'd rather tell you ourselves.

**Jamf, or any MDM, with Okta management attestation.** Jamf installs a certificate from Okta into the login keychain, and Okta [recommends making it non-exportable](https://help.okta.com/oie/en-us/content/topics/identity-engine/devices/okta-ca-dynamic-scep-macos-jamf.htm). A copy makes the personal Mac look managed. Okta's own posture checks, like OS version, do run on the personal Mac, so it has to pass those itself.

**Jamf, Intune, or Fleet with Entra.** Entra identifies the Mac by a key created at Platform SSO registration. Since August 2025, new registrations [keep that key in the Secure Enclave](https://learn.microsoft.com/en-us/entra/identity-platform/apple-sso-plugin#upcoming-changes-to-device-identity-key-storage), which never lets it out. On Windows, Entra keeps the device key [in the TPM](https://learn.microsoft.com/en-us/entra/identity/devices/concept-primary-refresh-token). There's nothing to copy.

Is this a big deal? For most companies, no. The employee still needs a real, enrolled, compliant work computer, and deleting that host in Fleet kills its certificate. Most people who try it want convenience, not your data. I've been that person. But it means conditional access isn't always checking what it seems to check, and you should know that before you rely on it.

## How to close the gap on a Mac

**Put the device key in the Secure Enclave.** With Entra, deploy [Platform SSO with a Secure Enclave key](https://fleetdm.com/guides/deploying-entra-platform-sso-with-fleet). With Okta, require [FastPass](https://support.okta.com/help/s/blog/a67WR00000BlpvDYAR/experts-helping-customers-a-technical-deep-dive-into-okta-fastpass) alongside Fleet. Okta Verify keeps its key in the Secure Enclave, so a copied certificate alone won't get anyone in.

**Attest the MDM connection.** Turn on [`apple_require_hardware_attestation`](https://fleetdm.com/docs/configuration/yaml-files#controls). Apple silicon Macs enrolling through Apple Business then prove, with a Secure Enclave key, that they're the Mac you bought. Everything Fleet tells Okta about that Mac comes over that connection.

**Fail Macs that leave MDM.** Add this policy to conditional access:

```sql
SELECT 1 FROM mdm WHERE enrolled = 'true' AND server_url LIKE '%fleet.example.com%';
```

Replace `fleet.example.com` with your Fleet server's hostname.

**Watch where sign-ins come from.** Okta's System Log records each sign-in's IP address, and Fleet records each host's public IP. A sign-in from somewhere the host has never been deserves a look.

On Windows, Fleet's key already lives in the TPM. Linux is the weakest platform for every vendor, because browsers keep certificates in files, so pair any Linux certificate with policy checks and a second factor.

## Set it up

- **Macs and PCs with Okta.** Add Fleet to Okta as a factor, deploy the certificate profiles, and choose which policies block sign-in. [Guide](https://fleetdm.com/guides/okta-conditional-access-integration)
- **Macs and PCs with Entra.** Add Fleet as a compliance partner and deploy Platform SSO. [Guide](https://fleetdm.com/guides/entra-conditional-access-integration)
- **Personal iPhones and Android phones.** Use Okta management attestation, delivered to Okta Verify through Fleet. When a phone leaves Fleet, Okta Verify goes with it. [Guide](https://fleetdm.com/guides/configure-okta-management-attestation-for-mobile-devices)
- **Linux.** Use [Duo](https://fleetdm.com/guides/duo-conditional-access-integration) or [PingFederate](https://fleetdm.com/guides/pingfederate-conditional-access-integration) today. Okta support is in progress.

## Guard the right door

Stop asking whether employees can remove MDM. With conditional access, it doesn't matter. Ask whether a computer you've never seen can borrow the identity of one you have. On a Mac, the answer comes down to where the key lives.

## See it live

Start with the [conditional access guide](https://fleetdm.com/guides/conditional-access), which covers every IdP Fleet supports.

- **Get a demo:** [See conditional access with your IdP](https://fleetdm.com/contact).
- **Compare plans:** Conditional access is part of [Fleet Premium](https://fleetdm.com/pricing).

<meta name="articleTitle" value="Conditional access makes removing the MDM pointless">
<meta name="authorFullName" value="Mike McNeil">
<meta name="authorGitHubUsername" value="mikermcneil">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-11-27">
<meta name="description" value="Conditional access makes removing MDM pointless. The real risk is a copied certificate. How Jamf, Intune, and Fleet compare on Macs.">

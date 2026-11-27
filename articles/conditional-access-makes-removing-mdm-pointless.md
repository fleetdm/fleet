# Conditional access makes removing the MDM pointless

*Mac admins used to worry about employees removing MDM. The problem worth worrying about now is quieter: a work certificate copied onto a personal laptop.*

## Key takeaways

- **Removing MDM locks the user out.** Once single sign-on (SSO) checks enrollment, a Mac that leaves MDM loses email, chat, and every other work app at the next sign-in.
- **The real gap is a copied certificate.** An employee who copies their sign-in certificate onto a personal Mac never removes MDM. Depending on your setup, the personal Mac can still pass.
- **A key in the Secure Enclave closes it.** Entra's Platform SSO keeps the Mac's identity in hardware, where it can't be copied. Okta setups, including Fleet's, aren't there yet.

<a purpose="cta-button" href="https://fleetdm.com/guides/conditional-access">Explore conditional access</a>

Every few months, someone on [Apple's forums](https://discussions.apple.com/thread/253448874) or [Jamf Nation](https://community.jamf.com/t5/jamf-pro/preventing-removal-of-mdm-profile/m-p/147549) asks how to stop employees from removing MDM. A developer with admin rights deletes the enrollment profile, or a contractor wipes their MacBook and skips enrollment, and IT finds out weeks later.

The replies split two ways: take away admin rights, or buy something that can't be removed, like Absolute. Most IT teams I talk to have since stopped worrying about MDM removal, and they're right to.

## Why removing MDM used to matter

MDM is how IT knows a Mac is encrypted, patched, and running the right security tools. Remove it, and IT loses control.

Apple solved most of this years ago. Macs bought through Apple Business enroll during setup with [Automated Device Enrollment](https://support.apple.com/guide/deployment/automated-device-enrollment-management-dep73069dd57/web), and the enrollment can be made non-removable. Fleet's default enrollment profile does this. Wipe the Mac, and Setup Assistant enrolls it again. The gap is Macs enrolled by hand, and Fleet can [block those](https://fleetdm.com/docs/configuration/yaml-files#controls) with `only_allow_apple_business_enrollment`.

Absolute takes the PC approach: an agent [built into the firmware](https://www.absolute.com/en/why/persistence) of most major PC brands that reinstalls itself if deleted. Teams buy it mainly to find and freeze lost or stolen laptops, and lately to [repair PCs stuck on a blue screen](https://www.absolute.com/blog/absolute-security-automates-remediation-for-crowdstrike-and-other-windows-bsod-incidents) after outages like CrowdStrike's in 2024. On a Mac, it runs as a regular agent, and Apple's own tools already do the persistence part.

## Conditional access made removal pointless

When someone signs in to a work app, your identity provider (IdP) checks that the device is enrolled. Remove MDM, and Slack stops working at the next sign-in. There are three common ways to check:

1. **A device certificate.** MDM installs a certificate, and the IdP asks the browser for it.
2. **The IdP's own app.** Okta Verify or Microsoft's Company Portal looks for a signal only MDM delivers, and reports back.
3. **A direct question.** The IdP asks the MDM whether the device is enrolled and compliant. Entra works this way with Intune and Jamf, and Okta works this way with Fleet.

Here's how Jamf, Intune, and Fleet compare:

| | Mac | Windows | Linux |
|---|---|---|---|
| Jamf | Entra or Okta | No | No |
| Microsoft Intune | Entra | Entra | Entra, for web apps in Edge on Ubuntu and Red Hat |
| Fleet | Okta, Entra, Duo, or PingFederate | Okta, Entra, Duo, or PingFederate | Duo or PingFederate, with Okta [coming soon](https://github.com/fleetdm/fleet/issues/53286) |

Sources: [Jamf](https://trusted.jamf.com/docs/win-modern-device-management), [Intune](https://learn.microsoft.com/en-us/intune/fundamentals/platform-guide-linux), [Fleet](https://fleetdm.com/guides/conditional-access).

Removing Fleet's MDM also removes the sign-in certificate it installed. As a backstop, add this conditional access policy, with your Fleet server's hostname:

```sql
SELECT 1 FROM mdm WHERE enrolled = 'true' AND server_url LIKE '%fleet.example.com%';
```

Any of these turns MDM removal from a security incident into a help desk ticket.

## The real gap: a copied certificate

At past jobs, I've had a work laptop and a personal one, and just wanted to check something quickly without switching computers. An engineer in that spot might copy the sign-in certificate from their work Mac to their personal one. They never remove MDM.

Whether it works depends on how your devices prove who they are.

**Fleet with Okta.** Fleet matches the certificate to a host and checks that host's policies. A copied certificate points back to the work Mac, so the personal Mac passes. Fleet marks the key non-exportable, but macOS enforces that in software. The key sits in the login keychain, locked with the employee's own password, and open source tools like [chainbreaker](https://github.com/n0fate/chainbreaker) decrypt every key in that file, exportable or not. macOS Tahoe 26.4 [raised the bar](https://eclecticlight.co/2026/10/02/how-to-copy-login-keychains-that-can-be-unlocked/) by also requiring a secret protected by System Integrity Protection (SIP), so the employee would have to turn SIP off first. On Windows, the key stays in the TPM. We've [filed this as a bug](https://github.com/fleetdm/fleet/issues/60032), because we'd rather you hear it from us.

**Jamf, or any MDM, with Okta management attestation.** Jamf installs a certificate from Okta in the login keychain, and Okta [recommends making it non-exportable](https://help.okta.com/oie/en-us/content/topics/identity-engine/devices/okta-ca-dynamic-scep-macos-jamf.htm). A copy makes the personal Mac look managed. Okta's own posture checks, like OS version, do run on the personal Mac, so it has to pass those itself.

**Jamf, Intune, or Fleet with Entra.** Entra identifies the Mac by a key created at Platform SSO registration. Since August 2025, new registrations [keep that key in the Secure Enclave](https://learn.microsoft.com/en-us/entra/identity-platform/apple-sso-plugin#device-identity-key-storage), which never lets it out. Macs registered before then need to register again, and after that there's nothing to copy.

Is this a big deal? For most companies, no. The employee still needs a real, enrolled, compliant work computer, and if you use Fleet, deleting that host kills its certificate. Most people who try it want convenience, not your data. But it means conditional access isn't always checking what it seems to check.

## How to close the gap on a Mac

**Keep Macs on macOS 26.4 or later, and require SIP.** Copying the key then means turning SIP off, which this conditional access policy catches:

```sql
SELECT 1 FROM sip_config WHERE config_flag = 'sip' AND enabled = 1;
```

An employee could turn SIP back on afterward, so this raises the bar without closing the gap.

**Use Entra with a Secure Enclave key.** If you're on Entra, deploy [Platform SSO with a Secure Enclave key](https://fleetdm.com/guides/deploying-entra-platform-sso-with-fleet). It works with Jamf, Intune, or Fleet.

**On Okta, FastPass only partly helps.** Okta Verify keeps its [FastPass](https://support.okta.com/help/s/blog/a67WR00000BlpvDYAR/experts-helping-customers-a-technical-deep-dive-into-okta-fastpass) key in the Secure Enclave, but Okta decides which Macs count as managed using the same certificate an employee can copy. Okta can't fully close this gap yet.

## Set it up

- **Macs and PCs with Entra.** Add Fleet as a compliance partner and deploy Platform SSO. [Guide](https://fleetdm.com/guides/entra-conditional-access-integration)
- **Macs and PCs with Okta.** Add Fleet to Okta as a factor, deploy the certificate profiles, and choose which policies block sign-in. [Guide](https://fleetdm.com/guides/okta-conditional-access-integration)
- **Personal iPhones and Android phones.** Use Okta management attestation, delivered to Okta Verify through Fleet. When a phone leaves Fleet, Okta Verify goes with it. [Guide](https://fleetdm.com/guides/configure-okta-management-attestation-for-mobile-devices)
- **Linux.** Use [Duo](https://fleetdm.com/guides/duo-conditional-access-integration) or [PingFederate](https://fleetdm.com/guides/pingfederate-conditional-access-integration) today. Browsers keep Linux certificates in files, so pair them with policy checks.

## Guard the right door

The question isn't whether employees can remove MDM. It's whether a computer you've never seen can borrow the identity of one you have. On a Mac, the answer comes down to where the key lives.

And the person copying a certificate isn't your enemy. They're telling you the secure path takes longer than the shortcut. So shorten it. With [User Enrollment](https://fleetdm.com/guides/enroll-personal-byod-ios-ipad-hosts-with-managed-apple-account), IT manages a work-only slice of a personal Mac, in its own encrypted volume, and leaves the rest alone. Give people a front door, and fewer of them will look for a window.

<a href="https://fleetdm.com/contact">Get a demo</a> to see conditional access with your IdP.

<meta name="articleTitle" value="Conditional access makes removing the MDM pointless">
<meta name="authorFullName" value="Mike McNeil">
<meta name="authorGitHubUsername" value="mikermcneil">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-11-27">
<meta name="description" value="Conditional access makes removing MDM pointless. The real risk is a copied certificate. How Jamf, Intune, and Fleet compare on Macs.">

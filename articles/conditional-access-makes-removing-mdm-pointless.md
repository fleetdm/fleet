# Conditional access makes removing the MDM pointless

*Mac admins used to worry about employees removing MDM. Conditional access mostly solved that. The problem worth worrying about now is quieter: a work certificate copied onto a personal laptop.*

## Key takeaways

- **Removing MDM only locks the user out.** Once single sign-on (SSO) checks enrollment, a Mac that leaves MDM loses email, chat, and every other work app at the next sign-in.
- **On a Mac, you don't need firmware tricks.** Absolute's persistence is built for PCs. Apple's Automated Device Enrollment already keeps MDM in place.
- **Jamf, Intune, and Fleet can all enforce this.** They differ in which platforms and identity providers they support, and in how a device proves who it is.
- **The real gap is a copied certificate.** An employee who copies their sign-in certificate onto a personal Mac never removes MDM. Depending on your setup, the personal Mac can still pass.
- **A key in the Secure Enclave closes it.** Entra's Platform SSO keeps the Mac's identity in hardware, where it can't be copied. Okta setups, including Fleet's, aren't there yet.

<a purpose="cta-button" href="https://fleetdm.com/guides/conditional-access">Explore conditional access</a>

Every few months, someone in a Mac admins forum asks how to stop employees from removing MDM. A developer with admin rights deletes the enrollment profile, or a contractor wipes their MacBook and skips enrollment, and IT finds out weeks later.

The replies split two ways: take away admin rights, or buy something that can't be removed, like Absolute. Both made sense once. Most IT teams I talk to have since stopped worrying about MDM removal, and they're right to. They should be worrying about something else.

## Why removing MDM used to matter

MDM is how IT knows a Mac is encrypted, patched, and running the right security tools. Remove it, and IT is blind.

Apple solved most of this years ago. Macs bought through Apple Business enroll during setup with [Automated Device Enrollment](https://support.apple.com/guide/deployment/automated-device-enrollment-management-dep73069dd57/web), and the enrollment can be made non-removable. Fleet's default enrollment profile does this. Wipe the Mac, and Setup Assistant enrolls it again. The gap is Macs enrolled by hand, and Fleet can [block those](https://fleetdm.com/docs/configuration/yaml-files#controls) with `only_allow_apple_business_enrollment`.

Absolute takes the PC approach: an agent [built into the firmware](https://www.absolute.com/en/why/persistence) of most major PC brands that reinstalls itself if deleted. Teams buy it mainly to find and freeze lost or stolen laptops, and lately to [repair PCs stuck on a blue screen](https://www.absolute.com/blog/absolute-security-automates-remediation-for-crowdstrike-and-other-windows-bsod-incidents) after outages like CrowdStrike's in 2024. On a Mac, it runs as a regular agent, and Apple's own tools already do the persistence part.

## Conditional access made removal pointless

Conditional access flips the question. Instead of stopping someone from removing MDM, it makes removal useless. When someone signs in to a work app, your identity provider (IdP) checks that the device is enrolled. Remove MDM, and Slack stops working at the next sign-in.

There are three common ways to check:

1. **A device certificate.** MDM installs a certificate, and the IdP asks the browser for it.
2. **The IdP's own app.** Okta Verify or Microsoft's Company Portal looks for a signal only MDM delivers, and reports back.
3. **A direct question.** The IdP asks the MDM whether the device is enrolled and compliant. Entra works this way with Intune and Jamf, and Okta works this way with Fleet.

With Fleet, add this policy to conditional access, and a Mac that leaves MDM fails it:

```sql
SELECT 1 FROM mdm WHERE enrolled = 'true' AND server_url LIKE '%fleet.example.com%';
```

Replace `fleet.example.com` with your Fleet server's hostname. Here's where the two big names in Mac management, and Fleet, stand:

| | Mac | Windows | Linux |
|---|---|---|---|
| Jamf | Entra or Okta | No (Jamf doesn't manage Windows) | No |
| Microsoft Intune | Entra | Entra | Entra, for web apps in Edge on Ubuntu and Red Hat |
| Fleet | Okta, Entra, Duo, or PingFederate | Okta, Entra, Duo, or PingFederate | Duo or PingFederate, with Okta [coming soon](https://github.com/fleetdm/fleet/issues/53286) |

Sources: [Jamf](https://trusted.jamf.com/docs/win-modern-device-management), [Intune](https://learn.microsoft.com/en-us/intune/fundamentals/platform-guide-linux), [Fleet](https://fleetdm.com/guides/conditional-access).

Any of these turns MDM removal from a security incident into a help desk ticket. So what's left?

## The real gap: a copied certificate

Picture an engineer with a work MacBook and a personal one. They'd rather read email on the personal one. They don't remove MDM from anything. They copy the sign-in certificate from the work Mac to the personal Mac.

The threat was never a Mac leaving MDM. It's a personal Mac borrowing the identity of an enrolled one. Whether that works depends on how your devices prove who they are.

**Fleet with Okta.** Okta hands each sign-in to Fleet, which finds the host by its certificate and checks that host's policies. A copied certificate points back to the work Mac, which is enrolled and patched, so the personal Mac passes. Fleet marks the key non-exportable, so macOS won't hand it over. But the key sits in the login keychain, not the Secure Enclave, and the keychain is locked with a password the employee knows, so a determined employee can get it out with third-party tools. (On Windows, the key stays in the TPM.) We've [filed this as a bug](https://github.com/fleetdm/fleet/issues/60032), because we'd rather you hear it from us.

**Jamf, or any MDM, with Okta management attestation.** Jamf installs a certificate from Okta in the login keychain, and Okta [recommends making it non-exportable](https://help.okta.com/oie/en-us/content/topics/identity-engine/devices/okta-ca-dynamic-scep-macos-jamf.htm). A copy makes the personal Mac look managed. Okta's own posture checks, like OS version, do run on the personal Mac, so it has to pass those itself.

**Jamf, Intune, or Fleet with Entra.** Entra identifies the Mac by a key created at Platform SSO registration. Since August 2025, new registrations [keep that key in the Secure Enclave](https://learn.microsoft.com/en-us/entra/identity-platform/apple-sso-plugin#upcoming-changes-to-device-identity-key-storage), which never lets it out. There's nothing to copy.

Is this a big deal? For most companies, no. The employee still needs a real, enrolled, compliant work computer, and deleting that host in Fleet kills its certificate. Most people who try it want convenience, not your data. I've been that person. But it means conditional access isn't always checking what it seems to check.

## How to close the gap on a Mac

**Use Entra with a Secure Enclave key.** If you're on Entra, deploy [Platform SSO with a Secure Enclave key](https://fleetdm.com/guides/deploying-entra-platform-sso-with-fleet). This is the strongest option on a Mac today, with Jamf, Intune, or Fleet.

**On Okta, add a second check.** Requiring [FastPass](https://support.okta.com/help/s/blog/a67WR00000BlpvDYAR/experts-helping-customers-a-technical-deep-dive-into-okta-fastpass) helps, because Okta Verify keeps its own key in the Secure Enclave. It only helps if Okta Verify can't be set up on personal Macs, and today that restriction leans on the same copyable certificate. A fully hardware-bound answer for Okta is what our [bug](https://github.com/fleetdm/fleet/issues/60032) tracks.

**Watch where sign-ins come from.** Okta's System Log records each sign-in's IP address, and Fleet records each host's public IP. A sign-in from somewhere the host has never been deserves a look.

## Set it up

- **Macs and PCs with Entra.** Add Fleet as a compliance partner and deploy Platform SSO. [Guide](https://fleetdm.com/guides/entra-conditional-access-integration)
- **Macs and PCs with Okta.** Add Fleet to Okta as a factor, deploy the certificate profiles, and choose which policies block sign-in. [Guide](https://fleetdm.com/guides/okta-conditional-access-integration)
- **Personal iPhones and Android phones.** Use Okta management attestation, delivered to Okta Verify through Fleet. When a phone leaves Fleet, Okta Verify goes with it. [Guide](https://fleetdm.com/guides/configure-okta-management-attestation-for-mobile-devices)
- **Linux.** Use [Duo](https://fleetdm.com/guides/duo-conditional-access-integration) or [PingFederate](https://fleetdm.com/guides/pingfederate-conditional-access-integration) today. Browsers keep Linux certificates in files, so pair them with policy checks.

## Guard the right door

Stop asking whether employees can remove MDM. With conditional access, it doesn't matter. Ask whether a computer you've never seen can borrow the identity of one you have. On a Mac, the answer comes down to where the key lives.

<a href="https://fleetdm.com/contact">Get a demo</a> to see conditional access with your IdP.

<meta name="articleTitle" value="Conditional access makes removing the MDM pointless">
<meta name="authorFullName" value="Mike McNeil">
<meta name="authorGitHubUsername" value="mikermcneil">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-11-27">
<meta name="description" value="Conditional access makes removing MDM pointless. The real risk is a copied certificate. How Jamf, Intune, and Fleet compare on Macs.">

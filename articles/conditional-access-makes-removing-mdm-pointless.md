# Conditional access makes removing the MDM pointless

*Mac admins used to worry about employees removing MDM. There's a quieter problem to worry about now: someone copying a work certificate onto their personal laptop.*

## Key takeaways

- **Removing MDM locks the user out.** When your sign-in system checks that a Mac is enrolled, a Mac without MDM can't open email, chat, or other work apps.
- **The bigger risk is a copied certificate.** An employee can copy the certificate from their work Mac to their personal Mac without removing MDM. With some setups, the personal Mac then gets in.
- **Keeping the key in hardware fixes it.** Entra can store a Mac's identity in the Secure Enclave, a chip that won't let the key out. Okta setups, including Fleet's, can't do this yet.

<a purpose="cta-button" href="https://fleetdm.com/guides/conditional-access">Explore conditional access</a>

Every few months, someone on [Apple's forums](https://discussions.apple.com/thread/253448874) or [Jamf Nation](https://community.jamf.com/t5/jamf-pro/preventing-removal-of-mdm-profile/m-p/147549) asks how to stop employees from removing MDM. A developer with admin rights deletes the enrollment profile. A contractor wipes their MacBook and skips enrollment. IT finds out weeks later.

The usual answers are to take away admin rights, or to buy a tool that can't be removed, like Absolute. Most IT teams I talk to have stopped worrying about this, and they're right.

## Why removing MDM used to matter

MDM is how IT makes sure a Mac is encrypted, up to date, and running the right security tools. If someone removes it, IT loses control of that Mac.

Apple fixed most of this years ago. Macs bought through Apple Business enroll in MDM during setup, using [Automated Device Enrollment](https://support.apple.com/guide/deployment/automated-device-enrollment-management-dep73069dd57/web), and you can stop users from removing it. Fleet does this by default. If someone wipes the Mac, it enrolls again during setup. The only gap is Macs enrolled by hand, and Fleet can [block those](https://fleetdm.com/docs/configuration/yaml-files#controls) with the `only_allow_apple_business_enrollment` setting.

Absolute works differently. Most major PC brands [build its agent into their firmware](https://www.absolute.com/en/why/persistence), so it reinstalls itself if someone deletes it. Companies mostly use Absolute to find and freeze lost or stolen laptops. It can also [fix PCs stuck on a blue screen](https://www.absolute.com/blog/absolute-security-automates-remediation-for-crowdstrike-and-other-windows-bsod-incidents), like the ones that crashed during the CrowdStrike outage in 2024. On a Mac, Absolute is a regular app, and Apple's tools already keep Macs enrolled.

## Conditional access made removal pointless

Conditional access checks the device every time someone signs in to a work app. Your sign-in provider (like Okta or Microsoft Entra) asks whether the device is enrolled, and blocks it if not. Remove MDM, and Slack stops working the next time you sign in.

There are three common ways to check:

1. **A certificate.** MDM puts a certificate on the device, and the sign-in provider asks the browser for it.
2. **The sign-in provider's app.** An app like Okta Verify or Microsoft's Company Portal looks for something only MDM installs, and reports back.
3. **Asking the MDM.** The sign-in provider asks the MDM whether the device is enrolled and meets your rules. Entra does this with Intune and Jamf, and Okta does this with Fleet.

Here's how Jamf, Intune, and Fleet compare:

| | Mac | Windows | Linux |
|---|---|---|---|
| Jamf | Entra or Okta | No | No |
| Microsoft Intune | Entra | Entra | Entra, for web apps in Edge on Ubuntu and Red Hat |
| Fleet | Okta, Entra, Duo, or PingFederate | Okta, Entra, Duo, or PingFederate | Duo or PingFederate, with Okta [coming soon](https://github.com/fleetdm/fleet/issues/53286) |

Sources: [Jamf](https://trusted.jamf.com/docs/win-modern-device-management), [Intune](https://learn.microsoft.com/en-us/intune/fundamentals/platform-guide-linux), [Fleet](https://fleetdm.com/guides/conditional-access).

When someone removes Fleet's MDM, the sign-in certificate goes with it. To be safe, also add this policy to conditional access. Replace `fleet.example.com` with your Fleet server's address:

```sql
SELECT 1 FROM mdm WHERE enrolled = 'true' AND server_url LIKE '%fleet.example.com%';
```

With any of these, removing MDM stops being a security problem. It becomes a help desk ticket.

## The bigger risk: a copied certificate

At past jobs, I had a work laptop and a personal one, and sometimes I just wanted to check something quickly without switching. Someone in that spot might copy the sign-in certificate from their work Mac to their personal Mac. They never remove MDM.

Whether that works depends on how your setup recognizes a device.

**Fleet with Okta.** Fleet looks up which computer the certificate belongs to and checks that computer. A copied certificate points to the work Mac, which is enrolled and up to date, so the personal Mac gets in. Fleet tells macOS not to let anyone export the key, but that's only a software rule. The key is stored in the login keychain, which is protected by the employee's own password. Free tools like [chainbreaker](https://github.com/n0fate/chainbreaker) can read that file and pull out every key in it. Starting with macOS Tahoe 26.4, this [got harder](https://eclecticlight.co/2026/10/02/how-to-copy-login-keychains-that-can-be-unlocked/): the employee would first have to turn off System Integrity Protection (SIP), a macOS safety feature. On Windows, the key is stored in the TPM, a security chip, so it's much harder to copy. We've [filed this as a bug](https://github.com/fleetdm/fleet/issues/60032). We'd rather you hear it from us.

**Jamf, or any MDM, with Okta's managed-device check.** Jamf puts a certificate from Okta in the login keychain, and Okta [recommends blocking exports](https://help.okta.com/oie/en-us/content/topics/identity-engine/devices/okta-ca-dynamic-scep-macos-jamf.htm). A copied certificate makes the personal Mac look managed. Okta's own checks, like the macOS version, still run on the personal Mac, so it has to pass those.

**Jamf, Intune, or Fleet with Entra.** Entra recognizes a Mac by a key it creates when the Mac registers. Since August 2025, Entra [stores that key in the Secure Enclave](https://learn.microsoft.com/en-us/entra/identity-platform/apple-sso-plugin#device-identity-key-storage), a chip inside the Mac that never lets it out. Macs that registered earlier need to register again. After that, there's nothing to copy.

Is this a big deal? For most companies, no. The employee still needs a real work computer that's enrolled and passing your checks. If you use Fleet, deleting that computer from Fleet stops its certificate from working. Most people who try this want convenience, not your data. But it means conditional access isn't always checking the computer you think it is.

## How to close the gap on a Mac

**Keep Macs on macOS 26.4 or later, and require SIP.** Then copying the key means turning SIP off first. This conditional access policy blocks any Mac with SIP off:

```sql
SELECT 1 FROM sip_config WHERE config_flag = 'sip' AND enabled = 1;
```

Someone could turn SIP back on after copying the key, so this makes copying harder but doesn't stop it.

**Use Entra with a Secure Enclave key.** If you use Entra, set up [Platform SSO with a Secure Enclave key](https://fleetdm.com/guides/deploying-entra-platform-sso-with-fleet). This works with Jamf, Intune, or Fleet.

**On Okta, FastPass only partly helps.** Okta Verify keeps its [FastPass](https://support.okta.com/help/s/blog/a67WR00000BlpvDYAR/experts-helping-customers-a-technical-deep-dive-into-okta-fastpass) key in the Secure Enclave. But Okta decides which Macs are managed using the same certificate someone can copy. So Okta can't fully close this gap yet.

## Set it up

- **Macs and PCs with Entra.** Add Fleet as a compliance partner and set up Platform SSO. [Guide](https://fleetdm.com/guides/entra-conditional-access-integration)
- **Macs and PCs with Okta.** Add Fleet to Okta, install the certificates, and choose which Fleet policies block sign-in. [Guide](https://fleetdm.com/guides/okta-conditional-access-integration)
- **Personal iPhones and Android phones.** Use Okta's managed-device check, set up through Fleet. If a phone leaves Fleet, Okta Verify is removed with it. [Guide](https://fleetdm.com/guides/configure-okta-management-attestation-for-mobile-devices)
- **Linux.** Use [Duo](https://fleetdm.com/guides/duo-conditional-access-integration) or [PingFederate](https://fleetdm.com/guides/pingfederate-conditional-access-integration) today. Linux browsers store certificates in regular files, so also use Fleet policies to check the computer.

## What to worry about instead

With conditional access, you don't need to worry much about employees removing MDM. Worry instead about a personal computer using a certificate copied from a work one. On a Mac, the fix is keeping the device's key in the Secure Enclave, where it can't be copied.

Someone who copies a certificate usually isn't trying to cause harm. They just want a faster way to get their work done. So give them a supported way to do it. For example, let them enroll their personal iPhone or iPad with [User Enrollment](https://fleetdm.com/guides/enroll-personal-byod-ios-ipad-hosts-with-managed-apple-account), which lets IT manage work apps and data while personal data stays private.

<a href="https://fleetdm.com/contact">Get a demo</a> to see conditional access with your sign-in provider.

<meta name="articleTitle" value="Conditional access makes removing the MDM pointless">
<meta name="authorFullName" value="Mike McNeil">
<meta name="authorGitHubUsername" value="mikermcneil">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-11-27">
<meta name="description" value="Conditional access makes removing MDM pointless. The real risk is a copied certificate. How Jamf, Intune, and Fleet compare on Macs.">

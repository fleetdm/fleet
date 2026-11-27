# Conditional access makes removing the MDM pointless

*An end user who removes MDM still has a working laptop. What they lose is access to email, chat, and every other app behind single sign-on. Here's how that works on Macs, PCs, Linux, and personal phones.*

## Key takeaways

- **Removing MDM isn't the risk it used to be.** If single sign-on (SSO) checks enrollment, a device that drops out of MDM loses access to company apps at the next sign-in. The incentive to remove it goes away.
- **There are three common ways to prove enrollment.** A device certificate, a check by an agent your identity provider (IdP) installs, or a direct question from the IdP to the MDM. Most setups use one, and some combine two.
- **Certificates are only as strong as their private key.** A key locked in the Secure Enclave or TPM can't be copied to another computer. An exportable key can, so check how your certificates are issued before trusting them.
- **Macs and PCs take about an afternoon to set up.** Fleet acts as a sign-in factor in Okta, checks a hardware-bound certificate, and blocks hosts that fail the policies you pick. Entra, Duo, and PingFederate are supported too.
- **Personal phones can prove they're managed without handing over the whole device.** Okta management attestation marks enrolled iPhones and Android devices as managed, including personal devices enrolled with User Enrollment or a work profile.
- **Linux is next.** Okta conditional access for Linux is in progress at Fleet, and the steps below show what setup will look like. Duo and PingFederate already work on Linux today.

<a purpose="cta-button" href="https://fleetdm.com/guides/conditional-access">Explore conditional access</a>

Every few months, a thread pops up on Reddit or in a Mac admins Slack about end users removing MDM. Someone with admin rights deletes the enrollment profile, or a contractor wipes the device and skips enrollment, and IT finds out weeks later. The replies usually go two ways: lock down local admin rights and make the MDM profile non-removable, or buy a firmware-level tool like Absolute Security that reinstalls agents after they're deleted.

Both are reasonable. But most of the IT and security teams I talk to have stopped worrying about this, because they've changed the question. Instead of "how do we stop someone from removing MDM?", they ask "what happens when they do?" If the answer is "they can't sign in to anything," removing MDM stops being a security problem and becomes a help desk ticket. The trick is making SSO check enrollment, and there are three common ways to do it.

## Three ways to prove a device is enrolled

### 1. A device certificate

The MDM installs a client certificate on the device. At sign-in, the IdP or VPN asks the browser for it and checks that your certificate authority (CA) issued it. No certificate, no access.

When the private key is generated inside the Secure Enclave on a Mac or the TPM on a Windows PC, it can't be exported. Removing MDM removes the certificate along with the MDM's other profiles, and the user can't take it with them. This is the strongest and simplest of the three methods, when it's done right.

### 2. A check by the IdP's own agent

The IdP installs its own app on the device, such as Okta Verify, Microsoft's Company Portal, or Google's Endpoint Verification. The app looks for a signal that the device is managed, like a certificate or a key that only the MDM delivers, and reports the device's status back to the IdP. Okta calls this management attestation.

### 3. A direct MDM-to-IdP link

The IdP asks the MDM directly whether the device is enrolled and compliant before it lets the user in. Microsoft Intune and Entra work this way, and Fleet can play the same role for Entra as a compliance partner. Fleet's Okta integration also works this way: Okta hands the sign-in to Fleet as a factor, Fleet identifies the host by its certificate, and Fleet checks the host's policies before sending the user back to Okta.

That last part matters. With a direct link, the IdP isn't just checking that a device was once enrolled. It's checking that the device is passing the policies you care about right now, like disk encryption, a current OS version, or a running EDR agent.

## The weak spot: certificates that can be copied

Method 1 has one failure mode, and it's a big one. If the private key isn't tied to hardware, someone can export the certificate and import it on an unmanaged computer. The IdP sees a valid certificate and lets them in.

This happens more often than you'd think. The usual causes:

- **A PKCS #12 identity pushed to every device.** Some teams generate one certificate and private key, wrap them in a `.p12` file, and deploy the same file everywhere. Anyone who can read the file has a certificate that works on any computer.
- **SCEP with an exportable key.** SCEP generates the key on the device, which is good, but Apple's SCEP payload allows the key to be exported unless `KeyIsExtractable` is set to `false`.
- **Software key storage on Windows.** A SCEP profile that stores the key in the software key storage provider (KSP) instead of the TPM leaves it exportable by anyone with admin rights.
- **Linux.** Linux browsers keep certificates in an NSS database in the user's home folder. There's no MDM-managed hardware keystore by default, so the key is a file that can be copied.

### Ways around it

**Generate keys in hardware and make them non-exportable.** This is the real fix on Macs and PCs. The SCEP profile Fleet generates for Okta conditional access on macOS sets `KeyIsExtractable` to `false`, so the key never leaves the keychain. On Windows, Fleet's profile sets the `KeyProtection` option of the `ClientCertificateInstall` CSP to use the TPM. <!-- TODO: Confirm with the MDM team how Windows behaves when KeyProtection=2 and the host has no TPM (fall back to software vs. fail enrollment), and update this sentence. -->

**Issue one certificate per device, and check it against the device.** A certificate that maps to exactly one host can be traced and revoked. Fleet looks up the host by the certificate's serial number at every sign-in, then checks that host's policies. A copied certificate only works while the original host is enrolled, current, and passing. Delete the host in Fleet and its certificate stops working right away.

**Keep certificates short-lived.** The less time a copied certificate stays valid, the less it's worth stealing. Fleet renews conditional access certificates automatically, so you can shorten their lifetime without creating tickets.

**Combine methods.** Pair a certificate with a second signal the IdP checks on its own, like Okta FastPass or management attestation, so a stolen certificate alone isn't enough. Add your IdP's network zones and behavior detection on top, so a sign-in from an unfamiliar location triggers extra verification.

**On Linux, put the key in the TPM if you can.** Most modern laptops ship with a TPM 2.0 chip, and tools like `tpm2-pkcs11` can expose a TPM-backed key to browsers through a PKCS #11 module. This takes more work than on macOS or Windows, and it's the main reason Linux has lagged on hardware-bound certificates. <!-- TODO: Validate a tpm2-pkcs11 + NSS setup on Ubuntu and RHEL with Chrome and Firefox before publishing, or soften this to "worth exploring." -->

**Watch for the same certificate in two places.** Your IdP's sign-in logs show where each sign-in came from. If the same device shows up from two networks at once, or from somewhere its host in Fleet has never been, investigate. <!-- TODO: Add an example Okta System Log query (or SIEM query) that flags the same Fleet factor used from multiple IPs. -->

## Set up conditional access with Fleet and Okta

The examples below use Okta, because it's the IdP we hear about most. Here's where the other major IdPs stand with Fleet:

| IdP | macOS | Windows | Linux | iPhone, iPad, and Android |
|---|---|---|---|---|
| Okta | Yes, with policy checks | Yes, with policy checks | Coming soon | Yes, with management attestation |
| Microsoft Entra | Yes, with policy checks | Yes, with policy checks | No | Not through Fleet <!-- TODO: confirm --> |
| Cisco Duo | Managed check | Managed check | Managed check | Not through Fleet <!-- TODO: confirm --> |
| PingFederate | Yes, with critical policy checks | Yes, with critical policy checks | Yes, with critical policy checks | Not through Fleet <!-- TODO: confirm --> |
| Google | No | No | No | Yes, company-owned only |

"Managed check" means the IdP blocks hosts that aren't managed by Fleet. "Policy checks" means it also blocks hosts that fail the Fleet policies you choose. Ping's integration uses PingFederate, not PingID on its own. <!-- TODO: Confirm whether PingOne/PingID customers without PingFederate have a supported path, and say so here. -->

These steps are abridged. Each section links to the full guide, which covers the details you'll need in production.

### Macs and PCs

Fleet's Okta integration covers macOS and Windows. Fleet adds itself to Okta as an identity provider used only as a factor, deploys a hardware-bound certificate to each host, and checks the host's policies every time someone signs in. Okta Adaptive MFA is required.

1. **Set up the mTLS proxy (self-hosted only).** Okta reaches Fleet through a reverse proxy on an `okta.` subdomain that requires the host's client certificate. Fleet's managed cloud handles this for you, and the [fleet-terraform addon](https://github.com/fleetdm/fleet-terraform/tree/main/addons/okta-conditional-access) sets it up on AWS.
2. **Connect Fleet and Okta.** In Fleet, head to **Settings > Integrations > Conditional access** and select **Connect** on the Okta card. Download the IdP signature certificate, add Fleet as a **SAML 2.0 IdP** in Okta with **IdP Usage** set to **Factor only**, then copy Okta's IdP ID, ACS URL, audience URI, and certificate back into Fleet.

<!-- TODO(screenshot): Fleet > Settings > Integrations > Conditional access, with the Okta card's "Connect" modal open. Save as website/assets/images/articles/conditional-access-makes-removing-mdm-pointless-okta-connect-[width]x[height]@2x.png -->

3. **Deploy the certificate profiles.** On the Okta card, select **Profiles**. Deploy the macOS profile, and the Windows certificate and browser profiles, as [custom OS settings](https://fleetdm.com/guides/custom-os-settings).
4. **Add Fleet as an authenticator in Okta.** In Okta, head to **Security > Authenticators**, add an **IdP Authenticator**, and select **Fleet**.
5. **Require Fleet in an authentication policy.** Add a rule to the authentication policy for the apps you want to protect. Scope it to **macOS** and **Windows**, and add the Fleet authenticator to the authentication method chain.

<!-- TODO(screenshot): Okta Admin Console > Security > Authentication policies > Add rule, showing "Device platform is" set to macOS and Windows and the Fleet IdP authenticator in the method chain. -->

6. **Choose which policies block sign-in.** In Fleet, head to **Policies**, choose a fleet, and select **Manage automations > Conditional access**. Turn it on and select the policies that should block sign-in.

<!-- TODO(screenshot): Fleet > Policies > Manage automations > Conditional access modal with a few policies selected. -->

Here are the policies that close the loop on MDM removal. Select them in step 6, and a host that leaves MDM fails its policy and loses access on its next sign-in. On macOS:

```sql
SELECT 1 FROM mdm WHERE enrolled = 'true' AND server_url LIKE '%fleet.example.com%';
```

Replace `fleet.example.com` with your Fleet server's hostname. On Windows, use the policy from Fleet's [MDM turned on](https://github.com/fleetdm/fleet/blob/main/docs/solutions/windows/policies/mdm-turned-on.yml) example:

```sql
SELECT 1 FROM mdm_bridge WHERE enrollment_status = 'device_enrolled';
```

On macOS and Windows, removing MDM also removes the certificate Fleet's profiles installed, so the host can't reach Fleet's sign-in check at all. The policy covers the edge cases, like a host that's still reporting through Fleet's agent but has left MDM. <!-- TODO: Confirm Windows removes ClientCertificateInstall SCEP certificates on MDM unenrollment. -->

When a host fails a policy, the end user lands on their **My device** page with instructions to fix it. Once they fix it and select **Refetch**, they can sign in again.

<!-- TODO(screenshot): End user's browser showing the Fleet "My device" page after a blocked Okta sign-in, with the failing policy and fix instructions. -->

Full guide: [Conditional access: Okta](https://fleetdm.com/guides/okta-conditional-access-integration)

Using a different IdP? See [Entra](https://fleetdm.com/guides/entra-conditional-access-integration), [Duo](https://fleetdm.com/guides/duo-conditional-access-integration), or [PingFederate](https://fleetdm.com/guides/pingfederate-conditional-access-integration). If you use Okta and only need to require a managed Windows device, without Fleet's policy checks, see [Enable Okta Verify on Windows](https://fleetdm.com/guides/enable-okta-verify-on-windows-using-a-scep-configuration-profile).

### Personal phones and tablets (BYOD)

Personal devices are where the "just make MDM non-removable" answer falls apart, because the device isn't yours to lock down. End users can remove a personal enrollment whenever they want, and they should be able to. Conditional access makes that a fair trade: keep the work enrollment and keep access to work apps, or remove it and lose both.

For personal iPhones, iPads, and Android devices, use Okta management attestation. Okta generates a secret key, Fleet delivers it to Okta Verify through managed app configuration, and Okta marks the device as managed. When the device is unenrolled, Okta Verify loses the managed configuration, so the device stops counting as managed.

1. **Create an app hint in Okta.** Head to **Security > Device integrations > Management attestation**, select **Create app hint**, and choose **iOS**. Copy the secret key, set **Device management provider** to `Fleet`, and set the enrollment link to your Fleet BYOD enrollment URL. Repeat for Android, which uses its own key.
2. **Store the secret key in Fleet.** Add it as a [custom variable](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles) instead of pasting it into the app configuration.
3. **Deploy Okta Verify with managed app configuration.** Add Okta Verify from the App Store (iOS) or Google Play (Android) in Fleet, then add the app configuration with the `managementHint` key.

<!-- TODO(screenshot): Fleet > Software > Okta Verify (iOS) > edit app configuration, showing the managementHint key with a $FLEET_SECRET_ variable. -->

4. **Require a managed device in Okta.** In the authentication policy for your apps, add a rule for **iOS** and **Android** that requires the device to be **Managed**.

<!-- TODO(screenshot): Okta authentication policy rule for iOS and Android with "Device state is: Registered" and "Device management is: Managed". -->

On iPhones and iPads with Account-driven User Enrollment, Okta marks the whole device as managed, not just the work partition. Android work profiles don't have this limitation.

Full guide: [Configure Okta management attestation for mobile devices with Fleet](https://fleetdm.com/guides/configure-okta-management-attestation-for-mobile-devices)

To enroll personal devices, see [Enroll personal iPhones and iPads](https://fleetdm.com/guides/enroll-byod-ios-ipados-hosts). If your phones are company-owned and you use Google Workspace, [Google conditional access](https://fleetdm.com/guides/google-conditional-access-integration) blocks devices that aren't company-owned and managed by Fleet, which means it blocks personal devices by design.

### Linux (coming soon)

> **Note:** Okta conditional access for Linux isn't released yet. It's [in progress](https://github.com/fleetdm/fleet/issues/53286), starting with Ubuntu and RHEL. The steps below show the expected setup and may change before release. Until then, [Duo](https://fleetdm.com/guides/duo-conditional-access-integration) and [PingFederate](https://fleetdm.com/guides/pingfederate-conditional-access-integration) support Linux today.

Linux doesn't have MDM configuration profiles, so Fleet delivers the certificate with a script instead. Everything on the Okta side stays the same as for Macs and PCs.

1. **Connect Fleet and Okta.** If you already set up Okta conditional access for macOS or Windows, skip this step. Otherwise, follow steps 1 and 2 in [Macs and PCs](#macs-and-pcs).
2. **Deploy the certificate with a script.** Add the Linux certificate script as a [script-only package](https://fleetdm.com/guides/deploy-software-packages#script-only-packages), and install it during [setup experience](https://fleetdm.com/guides/setup-experience) so new hosts get the certificate as they enroll. The script requests a certificate from Fleet and imports it into each user's Chrome and Firefox certificate stores, because Linux browsers don't read certificates from the filesystem. Hosts need `libnss3-tools` (Ubuntu) or `nss-tools` (RHEL).

<!-- TODO: Link the Linux conditional access certificate script once it lands in docs/solutions/linux/scripts/, and confirm whether Fleet surfaces it on the Okta card's "Profiles" modal (new "Linux" tab) or in the guide only. Update step 2 to match. -->

3. **Renew the certificate automatically.** Add a policy that fails when the certificate is missing or close to expiring, and use a [policy automation](https://fleetdm.com/guides/automatic-software-install-in-fleet) to rerun the script on failing hosts. This also catches users who open Firefox for the first time after the script ran.

<!-- TODO: Add the renewal policy query once the certificate path and subject are final. Model it on the Okta FastPass for Linux renewal policy. -->

4. **Add Linux to your Okta authentication policy.** Edit the rule from [Macs and PCs](#macs-and-pcs) step 5 so it also applies to Linux. <!-- TODO: Confirm how Okta's "Device platform is" condition matches Linux desktops (Linux option vs. "Other desktop"), and update this step with the exact option name. -->
5. **Choose which Linux policies block sign-in.** In Fleet, head to **Policies > Manage automations > Conditional access** and select policies that target Linux, like disk encryption or a required OS version.

<!-- TODO(screenshot): Fleet > Policies > Manage automations > Conditional access with a Linux policy (e.g. "Disk encryption enabled (Linux)") selected. Capture once the feature ships. -->

As covered above, the certificate's private key on Linux lives in a browser's software keystore, so it's the easiest platform to copy a certificate from. Keep certificate lifetimes short and pair the certificate with a policy check that only the real host can pass.

Full guide: Coming soon. <!-- TODO: Link the Linux section of the Okta conditional access guide once it ships. -->

Want to require managed Linux hosts in Okta through Okta Verify instead? See [Deploy Okta FastPass for Linux](https://fleetdm.com/guides/deploy-okta-fastpass-for-linux).

## Stop chasing removals and start checking sign-ins

You can't make MDM impossible to remove on a device someone has admin rights to, and on a personal phone you shouldn't try. What you can do is make sure removing it doesn't get anyone anything. When every sign-in checks a hardware-bound certificate against a host that's enrolled and passing policies, a device that leaves MDM just stops working for work, on every platform you manage.

## See it live

Start with the [conditional access guide](https://fleetdm.com/guides/conditional-access), which covers every IdP Fleet supports.

- **Get a demo:** [See conditional access with your IdP](https://fleetdm.com/contact).
- **Try Fleet:** [Get started for free](https://fleetdm.com/register). <!-- TODO: Confirm this is the right second CTA, since conditional access requires Fleet Premium. -->

<meta name="articleTitle" value="Conditional access makes removing the MDM pointless">
<meta name="authorFullName" value="TODO">
<meta name="authorGitHubUsername" value="TODO">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-12-01">
<meta name="description" value="If SSO checks MDM enrollment, removing MDM just locks the user out. How to set up conditional access for Macs, PCs, Linux, and BYOD phones.">

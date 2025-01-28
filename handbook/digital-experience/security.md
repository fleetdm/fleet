# Security


## Fleet security


### Account recovery process

As an all-remote company, we do not have the luxury of seeing each other or being able to ask for help in person. Instead, we require live video confirmation of someone's identity before performing recovery, and this applies to all Fleet company accounts, from internal systems to SaaS accounts.

| Participant | Role                                                                                                                                                 |
| ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requester   | Requests recovery for their own account                                                                |
| Recoverer   | Person with access to perform the recovery who monitors `#g-digital-experience`                                                                                                                               |
| Identifier  | Person that visually identifies the requester in a video call. The identifier can be the recoverer or a person the recoverer can recognize visually |


### Preparing for recovery

1. If the requester still has access to GitHub and/or Slack, they [ask for
   help](https://fleetdm.com/handbook/digital-experience#contact-us). For non-urgent requests, please
   prefer filing an issue with the Digital Experience team. If they do not have access,
   they can contact their manager or a teammate over the phone via voice or texting, and they will
   [ask for help](https://fleetdm.com/handbook/digital-experience#contact-us) on behalf of the
   requester.
2. The recoverer identifies the requester through a live video call.
* If the recoverer does not know the requester well enough to positively identify them visually, the
  recoverer can ask a colleague whom they recognize to act as the identifier. **All three must be
  live on a video call at the same time.**
*  For example, if the recoverer does not recognize Sam but can recognize Zach, they should ask Zach to identify Sam. Using the requester's manager or a direct teammate is recommended, as it increases the chances they frequently see each other on video.
3. If the recoverer recognizes the requester or has the identity confirmed by the person acting as
   the identifier, they can perform the recovery and update the login recovery issue.
* If the recoverer is not 100% satisfied with identification, they do **NOT** proceed and post to
  `#_security` to engage the security team immediately.

After the identity confirmation, the recovery can be performed while still on the video call, or asynchronously.


### Performing recovery

Before any account recovery, the recoverer must send a message to `#_security` announcing that the
recovery will take place. Then, perform the necessary recovery steps.


#### Google

The recoverer (who must be a Google admin) can follow [the instructions](https://support.google.com/a/answer/9176734) to
get backup verification codes. Provide a code to the requester, which they can use in place of
2-step verification at login.

After recovery, the requester should reset their 2-step verification.


#### 1Password

The recoverer (who must be a 1Password admin/owner) can follow [the
instructions](https://support.1password.com/recovery/) to perform a recovery. An email will be sent
to the requester allowing them to log back into their 1Password account.

After recovery, the requester may need to reinitialize 1Password on their devices.


## How we protect end-user devices

At Fleet, we believe that a good user experience empowers contributors.

We follow the guiding principles below to secure our company-owned devices.

* Our devices should give contributors the freedom to work from anywhere.
* To allow maximum freedom in where and how we work, we assume that "Safe" networks do not exist. Contributors should be able to work on a coffee shop's Wi-Fi as if it were their home or work network.
* To limit the impact on user experience, we do not dictate security configurations unless the security benefit is significant (only if it dramatically reduces the risk for the company, customers, or open source users).
* By using techniques such as Two-Factor Authentication (2FA), code reviews, and more, we can further empower contributors to work comfortably from anywhere - on any network.


### macOS devices

> *Find more information about the process of implementing security on the Fleet blog. The first [Tales from Fleet security: securing the startup](https://blog.fleetdm.com/tales-from-fleet-security-securing-the-startup-448ea590ea3a) article covers the process of securing our laptops.*

We use configuration profiles to standardize security settings for our Mac devices. We use [CIS Benchmark for macOS 12](https://www.cisecurity.org/benchmark/apple_os) as our configuration baseline and adapt it to
* suit a remote team.
* balance the need for productivity and security.
* limit the impact on the daily use of our devices.

> *Note: Details of your Mac’s configuration profile can be viewed anytime from the **Profiles** app under **System Preferences**.*



Our policy applies to Fleet-owned laptops purchased via Apple's DEP (Device Enrollment Program), which will retroactively be applied to every company-owned Mac, consists of the below. 


#### Enabling automatic updates

| #   | Setting                                                                                |
| --- | -------------------------------------------------------------------------------------- |
| 1.1 | Ensure all Apple-provided software is current                                          |
| 1.2 | Ensure auto-update is enabled                                                          |
| 1.4 | Ensure installation of app updates is enabled                                          |
| 1.5 | Ensure system data files and security updates are downloaded automatically is enabled |
| 1.6 | Ensure install of macOS updates is enabled                             |

*Note: the setting numbers included in the tables throughout this section are the recommended numbers from the CIS Benchmark for macOS12 document referenced above.*

**Why?**

Keeping software up-to-date helps to improve the resilience of our Mac fleet. Software updates include security updates that fix vulnerabilities that could otherwise be exploited. Browsers, for example, are often exposed to untrusted code, have a significant attack surface, and are frequently attacked.

macOS includes [malware protection tools](https://support.apple.com/en-ca/guide/security/sec469d47bd8/web) such as *Xprotect*. This is an antivirus technology based on [YARA](https://github.com/VirusTotal/yara) and MRT (Malware Removal Tool), a tool built by Apple to remove common malware from systems that are infected.
By enabling these settings, we:

* Ensure the operating system is kept up to date.
* Ensure XProtect and MRT are as up-to-date as possible.
* Ensure that Safari is kept up to date. 

This improves the resilience of our Mac fleet. 

**User experience impacts**

* Updates are required, which can be disruptive. For this reason, we allow the user to **postpone the installation five times**.
* Critical security updates are automatically downloaded, which could result in bandwidth use on slow or expensive links. For this reason, we limit automatic downloads to critical security updates only, while feature updates, which are typically larger, are downloaded at the time of installation selected by the user.
* Enforced updates **do not** include significant macOS releases (e.g., 11➡️12). Those updates are tracked and enforced separately, as the impact can be more significant. We require installing the latest macOS version within three months of release or when known vulnerabilities remain unpatched on the older version.


#### Time and date

| #     | Setting                                             |
| ----- | --------------------------------------------------- |
| 2.2.1 | Ensure "Set time and date automatically" is enabled |

**Why?**

An accurate time is important for two main reasons
1. Authentication. Many authentication systems like [Kerberos](https://en.wikipedia.org/wiki/Kerberos_(protocol)) and [SAML](https://en.wikipedia.org/wiki/Security_Assertion_Markup_Language) require the time between clients and servers to be [close](http://web.mit.edu/Kerberos/krb5-1.5/krb5-1.5.4/doc/krb5-admin/Clock-Skew.html). Keeping accurate time allows those protocols to prevent attacks that leverage old authentication sessions. 
2. Logging. Performing troubleshooting or incident response is much easier when all the logs involved have close to perfectly synchronized timestamps.

**User experience impact**

* Minimal. Inability to set the wrong time. Time zones remain user-configurable.


#### Passwords

| #     | Setting                                                                                  |
| ----- | ---------------------------------------------------------------------------------------- |
| 5.2.2 | Ensure  minimum password length is configured (our minimum: eight characters)                                             |
| 5.2.3 | Ensure complex password must contain alphabetic characters is configured                 |
| 5.8   | Ensure a password is required to wake the computer from sleep or screen saver is enabled |

**Why?**

This category of settings is unique because there are more settings that we do *not* configure than those we do.

We follow the CIS benchmark where it makes sense and, in this case, take guidance from [NIST SP800-63B - Digital Identity Guidelines](https://pages.nist.gov/800-63-3/sp800-63b.html), especially [Appendix A -Strength of Memorized Secrets](https://pages.nist.gov/800-63-3/sp800-63b.html#appA).

* We do NOT enforce special complexity beyond requiring letters to be in the password.

Length is the most important factor when determining a secure password; while enforcing password expiration, special characters and other restrictive patterns are not as effective as previously believed and provide little benefit at the cost of hurting the user experience.

* We do NOT enforce exceptionally long passwords. 

As we use recent Macs with T2 chips or Apple Silicon, brute-force attacks against the hardware are [mitigated](https://www.apple.com/mideast/mac/docs/Apple_T2_Security_Chip_Overview.pdf).

* We DO require passwords to be a minimum of eight characters long with letters.

Since we can't eliminate the risk of passwords being cracked remotely, we require passwords to be a minimum of eight characters long with letters, a length reasonably hard to crack over the network and the minimum recommendation by SP800-63B.


**User experience impact**

* A password is required to boot and unlock a laptop. Touch ID and Apple Watch unlock are allowed, and we recommend using a longer password combined with TouchID or Apple Watch to reduce password annoyances throughout the day.



#### Disabling various services

| #      | Setting                                           |
| ------ | ------------------------------------------------- |
| 2.4.2  | Ensure internet sharing is disabled               |
| 2.4.4  | Ensure printer sharing is disabled                |
| 2.4.10 | Ensure content caching is disabled                |
| 2.4.12 | Ensure media sharing is disabled                  |
| 6.1.4  | Ensure guest access to shared folders is disabled |

**Why?**

* Any service listening on a port expands the attack surface, especially when working on unsafe networks, to which we assume all laptops are connected.
* Laptops with tunnels connecting to internal systems (TLS tunnel, SSH tunnel, VPN.) or multiple network interfaces could be turned into a bridge and exposed to an attack if internet sharing is enabled. 
* Guest access to shared data could lead to accidental exposure of confidential work files.

**User experience impacts**

* The inability to use the computer as a server to share internet access, printers, content caching of macOS and iOS updates, and streaming iTunes media to devices on the local network.
* File shares require an account.


#### Encryption, Gatekeeper, and firewall

| #       | Setting                                           |
| ------- | ------------------------------------------------- |
| 2.5.1.1 | Ensure FileVault is enabled                       |
| 2.5.2.1 | Ensure Gatekeeper is enabled                      |
| 2.5.2.2 | Ensure firewall is enabled                        |
| 2.5.2.3 | Ensure firewall Stealth Mode is enabled           |
| 3.6     | Ensure firewall logging is enabled and configured |

**Why?**

* Using FileVault protects the data on our laptops, including confidential data and session material (browser cookies), SSH keys, and more. Using FileVault makes sure a lost laptop is a minor inconvenience and not an incident. We escrow the keys to be sure we can recover the data if needed.
* [Gatekeeper](https://support.apple.com/en-ca/HT202491) is a macOS feature that makes sure users can safely open software on their Mac. With Gatekeeper enabled, users may execute only trustworthy apps (signed by the software developer and/or checked for malicious software by Apple). This is a useful first line of defense to have.
* Using the firewall will make sure that we limit the exposure to our devices, while stealth mode makes them more challenging to discover. 
* Firewall logging allows us to troubleshoot and investigate whether the firewall blocks applications or connections.

**User experience impacts**

* Due to FileVault's encryption process, a password is needed as soon as the laptop is turned on, instead of once it has booted.
* There is no performance impact macOS encrypts the system drive by default. 
* With Gatekeeper enabled, unsigned or unnotarized (not checked for malware by Apple) applications require extra steps to execute.
* With the firewall enabled, unsigned applications cannot open a firewall port for inbound connections.


#### Screen saver and automatic locking

| #     | Setting                                                                             |
| ----- | ----------------------------------------------------------------------------------- |
| 2.3.1 | Ensure an inactivity interval of 20 minutes or less for the screen saver to be enabled |
| 6.1.2 | Ensure show password hint is disabled                                              |
| 6.1.3 | Ensure guest account is disabled                                                    |
| NA    | Prevent the use of automatic login                                                  |

**Why?**

* Fleet contributors are free to work from wherever they choose. Automatic login exposes sensitive company data and poses a critical security risk if a laptop is lost or stolen. 
* Password hints can sometimes be easier to guess than the password itself. Since we support contributors remotely via MDM and do not require users to change passwords frequently, we eliminate the need for password hints and their associated risk.
* Since company laptops are issued primarily for work and tied to a single contributor's identity, guest accounts are not permitted.
* Automatic login would defeat the purpose of even requiring passwords to unlock computers.

**User experience impacts**

* Laptops lock after 20 minutes of inactivity. To voluntarily pause this, a [hot corner](https://support.apple.com/en-mo/guide/mac-help/mchlp3000/mac) can be configured to disable the screen saver. This is useful if you are, for example, watching an online meeting without moving the mouse and want to be sure the laptop will not lock.
* Forgotten passwords can be fixed via MDM instead of relying on potentially dangerous hints.
* Guest accounts are not available.


#### iCloud

We do not apply ultra restrictive Data Loss Prevention style policies to our devices. Instead, by using our company Google Drive, we make sure that the most critical company data never reaches our laptops, so it can remain secure while our laptops can remain productive.


| #       | Setting                                                   |
| ------- | --------------------------------------------------------- |
| 2.6.1.4 | Ensure iCloud Drive Documents and Desktop sync is disabled |

**Why?**
* We do not use managed Apple IDs and allow contributors to use their own iCloud accounts. We disable iCloud Documents and Desktop sync to avoid accidentally copying data to iCloud, but we do allow iCloud drive.

**User experience impact**

* iCloud remains permitted, but the Desktop and Documents folders will not be synchronized. Make sure you put your documents in our Google Drive, so you do not lose them if your laptop has an issue.


#### Miscellaneous security settings

| #     | Setting                                                      |
| ----- | ------------------------------------------------------------ |
| 2.5.6 | Ensure limit ad tracking is enabled                          |
| 2.10  | Ensure secure keyboard entry Terminal.app is enabled         |
| 5.1.4 | Ensure library validation is enabled                         |
| 6.3   | Ensure automatic opening of safe files in Safari is disabled |

**Why?**

* Limiting ad tracking has privacy benefits and no downside.
* Protecting keyboard entry into Terminal.app could prevent malicious or non-malicious but inappropriate applications from receiving passwords.
* Library validation makes sure that an attacker can't trick applications into loading a software library in a different location, leaving it open to abuse.
* Safari opening files automatically can lead to negative scenarios where files are downloaded and automatically opened in another application. Though the setting relates to files deemed "safe," it includes PDFs and other file formats where malicious documents exploiting vulnerabilities have been seen before.

**User experience impact**

* There is minimal to no user experience impact for these settings. However, applications used to create custom keyboard macros will not receive keystrokes when Terminal.app is the active application window.


#### Enforce DNS-over-HTTPs (DoH)

| #  | Setting                |
| -- | ---------------------- |
| NA | Enforce [DNS over HTTPS](https://en.wikipedia.org/wiki/DNS_over_HTTPS) |

**Why?**

* We assume that no network is "safe." Therefore, DNS queries could be exposed and leak private data. An attacker on the same wireless network could see DNS queries, determine who your employer is, or even intercept them and [respond with malicious answers](https://github.com/iphelix/dnschef). Using DoH protects the DNS queries from eavesdropping and tampering.
* We use Cloudflare's DoH servers with basic malware blocking. No censorship should be applied on these servers, except towards destinations known as malware-related.


**User experience impacts**

* Some misconfigured "captive portals," typically used in hotels and airports, might be unusable with DoH due to how they are configured. This can be worked around by using the hotspot on your phone, and if you have to use this network for an extended period of time, there are usually workarounds to perform to connect to them. Navigating to http://1.1.1.1 often resolves the issue.
* If you are trying to reach a site and believe it is being blocked accidentally, please submit it to Cloudflare. This should be extremely rare. If it is not, please let the security team know.
* If your ISP's DNS service goes down, you'll be able to continue working. 😎

*Note: If you are from another organization, reading this to help create your own configuration, remember implementing DoH in an office environment where other network controls are in place has other downsides than it would for a remote company. **Disabling** DoH makes more sense in those cases so that network controls can retain visibility. Please evaluate your situation before implementing any of our recommendations at your organization, especially DoH.*


#### Deploy osquery

| #  | Setting                |
| -- | ---------------------- |
| NA | Deploy [osquery](https://osquery.io/) pointed to our dogfood instance |

***Why?***

We use osquery and Fleet to monitor our own devices. This is used for vulnerability detection, security posture tracking, and incident response when necessary.


### Chrome configuration

We configure Chrome on company-owned devices with a basic policy.

| Setting                                                   |
| --------------------------------------------------------- |
| Enforce Chrome updates and Chrome restart within 48 hours |
| Block intrusive ads                                       |
| uBlock Origin adblocker extension deployed               |
| Password manager extension deployed                       |
| Chrome Endpoint Verification extension deployed           |

**Why?**

* Browsers have a large attack surface, and their updates contain critical security updates. 

**User experience impact**

* Chrome must be restarted within 48 hours of patch installation. The automatic restart happens after 19:00 and before 6:00 if the computer is running and tabs are restored (except for incognito tabs).
* Ads considered intrusive are blocked.
* uBlock Origin is enabled by default, and is 100% configurable, improving security and browsing performance.
* Endpoint Verification is used to make access decisions based on the security posture of the device. For example, an outdated Mac could be prevented access to Google Drive.


### Personal mobile devices

The use of personal devices is allowed for some applications, so long as the iOS or Android device's OS
is kept up to date.


### Hardware security keys

We strongly recommend using hardware security keys. Fleet configures privileged user accounts with a policy that enforces the use of hardware security keys. This prevents credential theft better than other methods of 2FA/2-SV. If you do not already have a pair of hardware security keys, order [two YubiKey 5C NFC security
keys](https://www.yubico.com/us/product/yubikey-5-nfc/) with your company card, or ask
for help in [#help-login](https://fleetdm.com/handbook/digital-experience/security#slack-channels) to get one if you do not have a company card.


#### Are they YubiKeys or security keys?

We use YubiKeys, a hardware security key brand that supports the FIDO U2F protocol. You can use
both terms interchangeably at Fleet. We use YubiKeys because they support more authentication protocols than regular
security keys.


#### Who has to use security keys and why?

Security keys are **strongly recommended** for everyone and **required** for team members with elevated privilege access. 

Because they are the only type of Two-Factor Authentication (2FA) that protects credentials from
phishing, we will make them **mandatory for everyone** soon. 

See the [Google Workspace security
section](https://fleetdm.com/handbook/digital-experience/security#google-workspace-security-authentication) for more
information on the security of different types of 2FA.


#### Goals

Our goals with security keys are to

1. eliminate the risk of credential phishing.
2. maintain the best user experience possible.
3. make sure team members can access systems as needed, and that recovery procedures exist in case of a lost key.
4. make sure recovery mechanisms are safe to prevent attackers from bypassing 2FA completely.


#### Setting up security keys on Google

We recommend setting up **three** security keys on your Google account for redundancy purposes: two
YubiKeys and your phone as the third key.

If you get a warning during this process about your keyboard not being identified, this is due to
YubiKeys having a feature that can simulate a keyboard. Ignore the "Your keyboard cannot be
identified" warning.

1. Set up your first YubiKey by following [Google's
   instructions](https://support.google.com/accounts/answer/6103523?hl=En). The instructions make
   you enroll the key by following [this
   link](https://myaccount.google.com/signinoptions/two-step-verification?flow=sk&opendialog=addsk).
   When it comes to naming your keys, that is a name only used so you can identify which key was
   registered. You can name them Key1 and Key2.
2. Repeat the process with your 2nd YubiKey. 
3. Configure your phone as [a security key](https://support.google.com/accounts/answer/9289445)  


#### Optional: getting rid of keyboard warnings

1. Install YubiKey manager. You can do this from the **Managed Software Center** on managed Macs.
   On other platforms, download it [from the official
   website](https://www.yubico.com/support/download/yubikey-manager/#h-downloads).
2. Open the YubiKey manager with one of your keys connected.
3. Go to the **Interfaces** tab.
4. Uncheck the **OTP** checkboxes under **USB** and click *Save Interfaces*.
5. Unplug your key and connect your 2nd one to repeat the process.


#### Optional: setting up security keys on GitHub

1. Configure your two security keys to [access
   GitHub](https://github.com/settings/two_factor_authentication/configure).
2. If you use a Mac, feel free to add it as a security key on GitHub. This brings most of the
   advantages of the hardware security key but allows you to log in by simply touching Touch ID as
   your second factor.


## FAQ

1. Can I use my Fleet YubiKeys with personal accounts?

**Answer**: We highly recommend that you do so. Facebook accounts, personal email, Twitter accounts,
cryptocurrency trading sites, and many more support FIDO U2F authentication, the standard used by
security keys. Fleet will **never ask for your keys back**. They are yours to use everywhere you
can.

2. Can I use my phone as a security key?

**Answer**: Yes. Google [provides
instructions](https://support.google.com/accounts/answer/6103523?hl=En&co=GENIE.Platform%3DiOS&oco=1),
and it works on Android devices as well as iPhones. When doing this, you will still need the YubiKey
to access Google applications from your phone. 
Since it requires Bluetooth, this option is also less reliable than the USB-C security key.

3. Can I leave my YubiKey connected to my laptop?

**Answer**: Yes, unless you are traveling. We use security keys to eliminate the ability of
attackers to phish our credentials remotely, not as any type of local security improvement. That
being said, keeping it separate from the laptop when traveling means they are unlikely to be
lost or stolen simultaneously.

4. I've lost one of my keys, what do I do?

**Answer**: Post in the `#g-security` channel ASAP so we can disable the key. IF you find it later, no
worries, just enroll it again!

5. I lost all of my keys, and I'm locked out! What do I do?

**Answer**: Post in the `#help-login` channel, or contact your manager if you find yourself locked out of Slack. You will be provided a way to log back in and make your phone your security key until you
receive new ones.

6. Can I use security keys to log in from any device?

**Answer**: The keys we use, YubiKeys 5C NFC, work over USB-C as well as NFC. They can be used on
Mac/PC, Android, iPhone, and iPad Pro with USB-C port. If some application or device does
not support it, you can always browse to [g.co/sc](https://g.co/sc) from a device that supports
security keys to generate a temporary code for the device that does not.

7. Will I need my YubiKey every time I want to check my email?

**Answer**: No. Using them does not make sessions shorter. For example, if using the GMail app on
mobile, you'd need the keys to set up the app only.


## GitHub security

Since Fleet makes open source software; we need to host and collaborate on code. We do this using GitHub.

This section covers our GitHub configuration. Like everything we do, we aim for the right level of security and productivity.

Because our code is open source, we are much more concerned about its integrity than its confidentiality.
This is why our configuration aims to protect what is in the code, but we spend no
effort preventing "leaks" since almost everything is public anyway.

If you are reading this from another organization that makes code that is not open source, we
recommend checking out [this guide](https://oops.computer/posts/safer-github-setup/).


### Authentication

Authentication is the lynchpin of security on Software-as-a-Service (SaaS) applications such as GitHub. It
is also one of the few controls we have to secure SaaS apps in general.

GitHub authentication differs from many SaaS products in one crucial way: accounts are global.
Developers can carry their accounts from company to company and use them for open source projects.
There is no reason to require company-specific GitHub accounts, as our code is public, and if it were
not, we would enforce Single Sign-On (SSO) to access our organization.

We enable *Require two-factor authentication* for everyone in the organization.

Fleet requires two-factor authentication for everyone in the organization. We do not require Single Sign-on (SSO) -
as most of the software we work on is open source and accessible to external collaborators. If you can imagine, GitHub
charges a [4x premium](https://sso.tax/) for this feature.


### Code security and analysis

| Code security and analysis feature | Setting                                                            | Note                                                                        |
| ---------------------------------- | ------------------------------------------------------------------ | --------------------------------------------------------------------------- |
| Dependency graph                   | Automatically enable for new private repositories + enabled on all | Default on all public repositories.                                               |
| Dependabot alerts                  | Automatically enable for new repositories + enabled for all        | We want to be alerted if any dependency is vulnerable.                       |
| Dependabot security updates        | Automatically enable for new repositories                          | This automatically creates PRs to fix vulnerable dependencies when possible. |


### Member privileges

| Member privileges feature | Setting | Note                                                                                                                         |
| ------------------------- | ------- | ---------------------------------------------------------------------------------------------------------------------------- |
| Base permissions          | Write   | Admin is too powerful, as it allows reconfiguring the repositories themselves. Selecting *Write* provides the perfect balance!                 |
| Repository creation       | None    | We want to limit repository creation and eventually automate it with the [GitHub Terraform provider](https://github.com/integrations/terraform-provider-github).     |
| Repository forking        | ✅  | By default, we allow repository forking.                                                                                      |
| Pages creation            | None    | We do not use GitHub pages, so we disable them to make certain people use our actual website or handbook, which are also in GitHub. |


#### Admin repository permissions

| Admin privileges feature                                                   | Member privileges feature | Note                                                                                                                                                                          |
| -------------------------------------------------------------------------- | ------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Allow members to change repository visibilities for this organization      | 🚫                   | Most of our repos are public, but for the few that are private, we want to require org admin privileges to make them public                                                    |
| Allow members to delete or transfer repositories for this organization     | 🚫                   | We want to require org admin privileges to be able to delete or transfer any repository.                                                                                       |
| Allow repository administrators to delete issues for this organization     | 🚫                   | We want to require org admin privileges to be able to delete issues, which is something that is very rarely needed but could be, for example, if we received GitHub issue spam. |
| Allow members to see the comment author's profile name in private repositories | 🚫                   | We barely use private repositories and do not need this.                                                                                                                |
| Allow users with read access to create discussions                         | 🚫                   | We do not currently use discussions and want people to use issues as much as possible.                                                                                       |
| Allow members to create teams                                              | 🚫                   | We automate the management of GitHub teams with the [GitHub Terraform provider](https://github.com/integrations/terraform-provider-github).                            |


### Team discussions

We do not use team discussions and therefore have disabled them. This is simply to avoid discussions
located in too many places and not security-related.



### Repository security



#### Branch protection

Branch protection is one of the most important settings to configure and the main reason we should not have members with administrative privileges on the repositories.

By default, Fleet protects branches with these names: `main`, `patch[_-*]`, `feature[_-*]`, `minor[_-*]`, `rc-minor[_-*]`, `rc-patch[_-*]`, and `fleet-v*`.

To see the rules for protected branches, go tothe Branches section of repository settings.


### Scanning tools

Though not technically a part of GitHub itself, we feel like the security tools we use to scan our code, workflows, and GitHub configuration are part of our overall GitHub configuration.


#### SAST and configuration scanning

| Scanning Tool                                       | Purpose                                                                                                                                              | Configuration                                                                                                  |
| --------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------- |
| [OSSF Scorecard](https://github.com/ossf/scorecard) | Scan our GitHub repository for best practices and send problems to GitHub Security.                                                                  | [scorecard-analysis.yml](https://github.com/fleetdm/fleet/blob/main/.github/workflows/scorecards-analysis.yml) |
| [CodeQL](https://codeql.github.com/)                | Discover vulnerabilities across our codebase, both in the backend and frontend code.                                                                 | [codeql-analysis.yml](https://github.com/fleetdm/fleet/blob/main/.github/workflows/codeql-analysis.yml)        |
| [gosec](https://github.com/securego/gosec)          | Scan golang code for common security mistakes. We use gosec as one of the linters(static analysis tools used to identify problems in code) used by [golangci-lint](https://github.com/golangci/golangci-lint) | [golangci-lint.yml](https://github.com/fleetdm/fleet/blob/main/.github/workflows/golangci-lint.yml)             |

We are planning on adding [tfsec](https://github.com/aquasecurity/tfsec) to scan for configuration vulnerabilities in the Terraform code provided to deploy Fleet infrastructure in the cloud. 
Once we have full coverage from a static analysis point of view, we will evaluate dynamic analysis
and fuzzing options.


#### Dependabot

As described in *Code security and analysis*, we use Dependabot for security updates to libraries.
Our [dependabot.yml](https://github.com/fleetdm/fleet/blob/main/.github/dependabot.yml) only
mentions GitHub actions. Security updates to all other dependencies are performed by Dependabot automatically, even though we do not configure all package managers explicitly in the configuration file, as specified in the repository configuration. As GitHub actions have no impact on the Fleet software itself, we are
simply more aggressive in updating actions even if the update does not resolve a vulnerability.


### Actions configuration

We configure GitHub Actions to have *Read repository contents permission* by default. This is
located in *organization/settings/actions*. As our code is open source, we allow all GitHub actions
but limit their default privileges so they do not create any additional risk. Additional permissions
needed can be configured in the YAML file for each workflow.

We pin actions to specific versions using a complete hash.


### Automation

We manage our GitHub configuration, creation of repositories, and team memberships manually. In the
future, we will consider automating most of it using the [Terraform
provider](https://github.com/integrations/terraform-provider-github) for GitHub. Our strategy for
this will be similar to what [this blog post](https://oops.computer/posts/github_automation/) describes.


## Google Workspace security

Google Workspace is our collaboration tool and the source of truth for our user identities.
A Google Workspace account gives access to email, calendar, files, and external applications integrated with Google Authentication or SAML.
At the same time, third-party applications installed by users can access the same data.

We configure Google Workspace beyond the default settings to reduce the risk of malicious or vulnerable apps being used to steal data. Our current configuration balances security and productivity and is a starting point for any organization looking to improve the security of Google Workspace.

As Google frequently adds new features, feel free to submit a PR to edit this file if you discover a new one we should use!


### Authentication

We cannot overstate the importance of securing authentication, especially in a platform that includes email and is used as a directory to log in to multiple applications.


#### 2-Step Verification 

Google's name for Two-Factor Authentication (2FA) or Multi-Factor Authentication (MFA) is 2-Step Verification (2-SV). No matter what we call it, it is the most critical feature to protect user accounts on Google Workspace or any other system.

| 2FA Authentication methods from least to most secure                              | Weaknesses                                                                                                |
| ----------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------- |
| No 2FA                                                                        | Credential theft is easy, and passwords are often leaked or easy to guess.                                |
| SMS/Phone-based 2FA                                                           | Puts trust in the phone number itself, which attackers can hijack by [social engineering phone companies](https://www.vice.com/en/topic/sim-hijacking).      |
| Time-based one-time password (TOTP - Google Authenticator type six digit codes) | Phishable as long as the attacker uses it within its short lifetime by intercepting the login form. |
| App-based push notifications                                                  | These are harder to phish than TOTP, but by sending a lot of prompts to a phone, a user might accidentally accept a nefarious notification.       |
| Hardware security keys                                                        | [Most secure](https://krebsonsecurity.com/2018/07/google-security-keys-neutralized-employee-phishing/) but requires extra hardware or a recent smartphone. Configure this as soon as you receive your Fleet YubiKeys                                                                |


##### 2-Step verification in Google Workspace

We apply the following settings to *Security/2-Step Verification* to all users as the minimum baseline.

| Setting name                               | Value                                              |
| ------------------------------------------ | -------------------------------------------------- |
| Allow users to turn on 2-Step Verification | On                                                 |
| Enforcement                                | On                                                 |
| New user enrollment period                 | 1-week                                             |
| Frequency: Allow user to trust the device  | Off                                                |
| Methods                                    | Any except verification codes via text, phone call |




#### Passwords

As we enforce the use of 2-SV, passwords are less critical to the security of our accounts. We base our settings on [NIST 800-63B](https://pages.nist.gov/800-63-3/sp800-63b.html).

Enforcing 2FA is a much more valuable control than enforcing the expiration of passwords, which usually results in users changing only a small portion of the password and following predictable patterns.

We apply the following *Security/Password management* settings to all users as the minimum baseline.


| Setting name                                                            | Value         |
| ----------------------------------------------------------------------- | ------------- |
| Enforce strong password                                                 | Enabled       |
| Length                                                                  | 8-100         |
| Strength and length enforcement/enforce password policy at next sign-in | Enabled       |
| Allow password reuse                                                    | Disabled      |
| Expiration                                                              | Never expires |

We also configure [Password Alert](https://support.google.com/chrome/a/answer/9696707?visit_id=637806265550953415-394435698&rd=1#zippy=) to warn users of password re-use. See [How we protect end-user devices](https://fleetdm.com/handbook/security#how-we-protect-end-user-devices).


#### Account recovery

Self-service account recovery is a feature we do not need, as we have enough Google administrators to support Fleet employees. As we secure accounts beyond the security level of most personal email accounts, it would not be logical to trust those personal accounts for recovery.

We apply the following settings to *Security/Account Recovery* to all users as the minimum baseline.

| Setting name                                               | Value |
| ---------------------------------------------------------- | ----- |
| Allow super admins to recover their account                | Off   |
| Allow users and non-super admins to recover their account | Off   |

First, we make sure we have a handful of administrators. Then, by not requiring password expiration, the number of issues related to passwords is reduced. Lastly, we can support locked-out users manually as the volume of issues is minimal.


#### Less secure apps

Less secure apps use legacy protocols that do not support secure authentication methods. We disable them, and as they are becoming rare, we have not noticed any issues from this setting.

We apply the following *Security/Less Secure Apps* settings to all users as the minimum baseline.

| Setting name                                                                                            | Value                                            |
| ------------------------------------------------------------------------------------------------------- | ------------------------------------------------ |
| Control user access to apps that use less secure sign-in technology makes accounts more vulnerable.  | Disable access to less secure apps (Recommended) |


#### API access

Google Workspace makes it easy for users to add tools to their workflows while having these tools authenticate to their Google applications and data via OAuth. We mark all Google services as *restricted* but do allow the use of OAuth for simple authentication and the use of less dangerous privileges on Gmail and Drive. We then approve applications that require more privileges on a case-by-case basis.

This level of security allows users to authenticate to web applications with their Google accounts. This exposes little information beyond what they would provide in a form to create an account, and it protects confidential data while keeping everything managed.

>To get an application added to Fleet's Google Workspace security configuration, create an issue and assign it to the security team in [this repository](https://github.com/fleetdm/confidential/issues). You'll need to include: the client ID in text (not a screenshot) in your issue. This is processed quickly (about 1-2 days) by the Head of Security. The Head of Security will do the research on permissions the app is requesting and determine approval for the app.

We mark every Google Service as *restricted* and recommend that anyone using Google Workspace mark at least the following as restricted in *Security/API Control/Google Services*:
* Google Drive
* Gmail
* Calendar (Invites include sensitive info such as external participants, attachments, links to meetings, etc.)
* Google Workspace Admin

When marked as *trusted* applications that need access to data in our Google Workspace.


### Rules and alerts

Google provides many useful built-in alerts in *Security/Rules*. We enable most and tweak their severity levels as needed. When necessary, we visit the [Alert Center](https://admin.google.com/ac/ac) to investigate and close alerts.

We have also created the following custom alerts. 

| Alert On                                    | Created on                          | Purpose                                                                                                                                                                  | Notification         |
| ------------------------------------------- | ----------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------------------- |
| Out of domain email forwarding              | Login audit log, filtered by event  | Attackers in control of an email account often configure forwarding to establish persistence.                                                              | Alert Center + Email |
| 2-step Verification disable                 | Login audit log, filtered by event  | Though we enforce 2-SV, if we accidentally allow removing it, we want to know as soon as someone does so.                                                               | Alert Center + Email |
| 2-step Verification Scratch Codes Generated | Admin audit log, filtered by event  | Use scratch codes to bypass 2-SV. An attacker with elevated privileges could leverage this to log in as a user.                           | Alert Center + Email |
| Change Allowed 2-step Verification Methods  | Admin audit log, filtered by event  | We want to detect accidental or malicious downgrades of 2-SV configuration.                                                                                              | Alert Center + Email |
| Change 2-Step Verification Start Date       | Admin audit log, filtered by event  | We want to detect accidental or malicious "downgrades" of the 2-SV configuration.                                                                                              | Alert Center + Email |
| Alert Deletion                              | Admin audit log, filtered by event  | For alerts to be a reliable control, we need to alert on alerts being disabled or changed.                                                                                | Alert Center + Email |
| Alert Criteria Change                       | Admin audit log, filtered by event  | For alerts to be a reliable control, we need to alert on alerts being disabled or changed.                                                                                | Alert Center + Email |
| Alert Receivers Change                      | Admin audit log, filtered by event  | For alerts to be a reliable control, we need to alert on alerts being disabled or changed.                                                                                | Alert Center + Email |
| Dangerous download warning                  | Chrome audit log, filtered by event | As we roll out more Chrome security features, we want to track the things getting blocked to evaluate the usefulness of the feature and potential false positives. | Alert Center         |
| Malware transfer                            | Chrome audit log, filtered by event | As we roll out more Chrome security features, we want to track the things getting blocked to evaluate the usefulness of the feature and potential false positives. | Alert Center         |
| Password reuse                              | Chrome audit log, filtered by event | As we roll out more Chrome security features, we want to track the things getting blocked to evaluate the usefulness of the feature and potential false positives | Alert Center         |



### Gmail


#### Email authentication

Email authentication makes it harder for other senders to pretend to be from Fleet. This improves trust in emails from fleetdm.com and makes it more difficult for anyone attempting to impersonate Fleet.

We authenticate email with [DKIM](https://support.google.com/a/answer/174124?product_name=UnuFlow&hl=en&visit_id=637806265550953415-394435698&rd=1&src=supportwidget0&hl=en) and have a [DMARC](https://support.google.com/a/answer/2466580) policy to decide how our outgoing email should be defined.

The DKIM configuration under *Apps/Google Workspace/Settings for Gmail/Authenticate Email* simply consists of generating the key, publishing it to DNS, then enabling the feature 48-hours later.

[DMARC](https://support.google.com/a/answer/2466580) is configured separately at the DNS level once DKIM is enforced.


#### Email security

Google Workspace includes multiple options in *Apps/Google Workspace/Settings for Gmail/Safety* related to how it handles inbound email.

As email is one of the main vectors used by attackers, we make certain we protect it as much as possible. Attachments are frequently used to send malware. We apply the following settings to block common tactics.

| Category                    | Setting name                                                    | Value   | Action                               | Note                                                                                                   |
| --------------------------- | --------------------------------------------------------------- | ------- | ------------------------------------ | ------------------------------------------------------------------------------------------------------ |
| Attachments                 | Protect against encrypted attachments from untrusted senders    | Enabled | Quarantine                           |                                                                                                        |
| Attachments                 | Protect against attachments with scripts from untrusted senders | Enabled | Quarantine                           |                                                                                                        |
| Attachments                 | Protect against anomalous attachment types in emails            | Enabled | Quarantine                           |                                                                                                        |
| Attachments                 | Whitelist (*Google's term for allow-list*) the following uncommon filetypes                      | Empty   |                                    |                                                                                                        |
| Attachments                 | Apply future recommended settings automatically                 | On      |                                    |                                                                                                        |
| IMAP View time protections  | Enable IMAP link protection                                     | On      |                                   |  |
| Links and external images   | Identify links behind shortened URLs                            | On      |                                      |                                                                                                        |
| Links and external images   | Scan linked images                                              | On      |                                      |                                                                                                        |
| Links and external images   | Show warning prompt for any click on links to untrusted domains | On      |                                      |                                                                                                        |
| Links and external images   | Apply future recommended settings automatically                 | On      |                                      |                                                                                                        |
| Spoofing and authentication | Protect against domain spoofing based on similar domain names   | On      | Keep email in the inbox and show warning |                                                                                                        |
| Spoofing and authentication | Protect against spoofing of employee names                      | On      | Keep email in the inbox and show warning |                                                                                                        |
| Spoofing and authentication | Protect against inbound emails spoofing your domain             | On      | Quarantine                           |                                                                                                        |
| Spoofing and authentication | Protect against any unauthenticated emails                      | On      | Keep email in the inbox and show warning |                                                                                                        |
| Spoofing and authentication | Protect your Groups from inbound emails spoofing your domain    | On      | Quarantine                           |                                                                                                        |
| Spoofing and authentication | Apply future recommended settings automatically                 | On      |                                      |                                                                                                        |
| Manage quarantines | Notify periodically when messages are quarantine                   | On      |                                      |                                                                                                        |

We enable *Apply future recommended settings automatically* to make certain we are secure by default. We would prefer to adjust this after seeing emails quarantined accidentally rather than missing out on new security features for email security.


#### End-user access

We recommend using the Gmail web interface on computers and the Gmail app on mobile devices. The user interface on the official applications includes security information not visible in standard mail clients (e.g., Mail on macOS). We do allow a few of them at the moment for specific workflows. 

| Category                         | Setting name                                                                                                                                      | Value                                                                                                                                                                                                                        | Note                                                                                                                                                                                                                                                  |
| -------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| POP and IMAP access              | Enable IMAP access for all users                                                                                                                  | Restrict which mail clients users can use (OAuth mail clients only)                                                                                                                                                          |                                                                                                                                                                                                                                                       |
|                                  | Clients                                                                                                                                           | (450232826690-0rm6bs9d2fps9tifvk2oodh3tasd7vl7.apps.googleusercontent.com, 946018238758-bi6ni53dfoddlgn97pk3b8i7nphige40.apps.googleusercontent.com, 406964657835-aq8lmia8j95dhl1a2bvharmfk3t1hgqj.apps.googleusercontent.com) | Those are the iOS, macOS built-in clients as well as Thunderbird. We plan to eventually only allow iOS,\ to limit the data cached on Macs and PCs.                                                                                         |
|                                  | Enable POP access for all users                                                                                                                   | Disabled                                                                                                                                                                                                                     |                                                                                                                                                                                                                                                       |
| Google Workspace Sync            | Enable Google Workspace Sync for Microsoft Outlook for my users                                                                                   | Disabled                                                                                                                                                                                                                     |                                                                                                                                                                                                                                                       |
| Automatic forwarding             | Allow users to automatically forward incoming email to another address                                                                            | Enabled                                                                                                                                                                                                                      | We will eventually disable this in favor of custom routing rules for domains where we want to allow forwarding. There is no mechanism for allow-listing destination domains, so we rely on alerts when new forwarding rules are added. |
| Allow per-user outbound gateways | Allow users to send mail through an external SMTP server when configuring a "from" address hosted outside your email domain                       | Disabled                                                                                                                                                                                                                     |                                                                                                                                                                                                                                                       |
| Warn for external recipients     | Highlight any external recipients in a conversation. Warn users before they reply to email messages with external recipients who aren't in their contacts. | Enabled                                                                                                                                                                                                                      |                                                                                                                                                                                                                                                       |



### Google Drive and Docs

We use Google Drive and related applications for internal and external collaboration.



#### Sharing settings

| Category                  | Setting name                                                                                                                                                          | Value                                       | Note                                                                                                                                                                                                                                                                                                                                                                                         |
| ------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Sharing options           | Sharing outside of Fleet Device Management                                                                                                                            | On                                          |                                                                                                                                                                                                                                                                                                                                                                                              |
| Sharing options           | For files owned by users in Fleet Device Management warn when sharing outside of Fleet Device Management                                                              | Enabled                                     |                                                                                                                                                                                                                                                                                                                                                                                              |
| Sharing options           | Allow users in Fleet Device Management to send invitations to non-Google accounts outside Fleet Device Management                                                     | Enabled                                     |                                                                                                                                                                                                                                                                                                                                                                                              |
| Sharing options           | When sharing outside of Fleet Device Management is allowed, users in Fleet Device Management can make files and published web content visible to anyone with the link | Enabled                                     |                                                                                                                                                                                                                                                                                                                                                                                              |
| Sharing options           | Access Checker                                                                                                                                                        | Recipients only, or Fleet Device Management |                                                                                                                                                                                                                                                                                                                                                                                              |
| Sharing options           | Distributing content outside of Fleet Device Management                                                                                                               | Only users in Fleet Device Management       | This prevents external contributors from sharing to other external contributors                                                                                                                                                                                                                                                                                                              |
| Link sharing default      | When users in Fleet Device Management create items, the default link sharing access will be:                                                                          | Off                                         | We want the owners of new files to make a conscious decision around sharing and to be secure by default                                                                                                                                                                                                                                                                                     |
| Security update for files | Security update                                                                                                                                                       | Apply security update to all affected files |                                                                                                                                                                                                                                                                                                                                                                                              |
| Security update for files | Allow users to remove/apply the security update for files they own or manage                                                                                          | Enabled                                     | We have very few files impacted by [updates to link sharing](https://support.google.com/a/answer/10685032?amp;visit_id=637807141073031168-526258799&amp;rd=1&product_name=UnuFlow&p=update_drives&visit_id=637807141073031168-526258799&rd=2&src=supportwidget0). For some files meant to be public, we want users to be able to revert to the old URL that is more easily guessed.  |


#### Features and applications

| Category                             | Setting name                                                             | Value                                                                                                                           | Note                                                                                                                   |
| ------------------------------------ | ------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------- |
| Offline                              | Control offline access using device policies                             | Enabled                                                                                                                         |                                                                                                                        |
| Smart Compose                        | Allow users to see Smart Compose suggestions                             | Enabled                                                                                                                         |                                                                                                                        |
| Google Drive for desktop             | Allow Google Drive for desktop in your organization                      | Off                                                                                                                             | To limit the amount of data stored on computers, we currently do not allow local sync. We may enable it in the future.  |
| Drive                                | Drive                                                                    | Do not allow Backup and Sync in your organization                                                                               |                                                                                                                        |
| Drive SDK                            | Allow users to access Google Drive with the Drive SDK API                | Enabled                                                                                                                         | The applications trusted for access to Drive are controlled but require this to work.                                 |
| Add-Ons                              | Allow users to install Google Docs add-ons from add-ons store            | Enabled                                                                                                                         | The applications trusted for access to Drive are controlled but require this to work.                                 |
| Surface suggestions in Google Chrome | Surface suggestions in Google Chrome                                     | Allow Google Drive file suggestions for signed-in users whenever a new search is performed or a new tab is opened (recommended) |                                                                                                                        |
| Creating new files on Drive          | Allow users to create and upload any file                                | On                                                                                                                              |                                                                                                                        |
| Creating new files on Drive          | Allow users to create new Docs, Sheets, Slides, Drawings and Forms files | On                                                                                                                              |                                                                                                                        |


## Vendor questionnaires 


## Scoping

| Question | Answer                                                                                                                                                 |
| ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| Will Fleet allow us to conduct our own penetration test?   | Yes                                                               |


## Service monitoring and logging

| Question | Answer                                                                                                                                                 |
| ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| Does your service system/application write/export logs to a SIEM or cloud-based log management solution?    |   Yes, Fleet Cloud service logs are written to AWS Cloudwatch |
| How are logs managed (stored, secured, retained)?    |   Alerting triggers manual review of the logs on an as-needed basis. Logs are retained for a period of 30 days by default. Logging access is enabled by IAM rules within AWS.   |
| Can Fleet customers access service logs?    |    Logs will not be accessible by default, but can be provided upon request. |


## Governance and risk management

| Question | Answer                                                                                                                                                 |
| ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| Does Fleet have documented information security baselines for every component of the infrastructure (e.g., hypervisors, operating systems, routers, DNS servers, etc.)?  | Fleet follows best practices for the given system. For instance, with AWS we utilize AWS best practices for security including GuardDuty, CloudTrail, etc.                                                                |


## Network security

| Question | Answer                                                                                                                                                 |
| ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| Does Fleet have the following employed in their production environment? File integrity Monitoring (FIM), Host Intrusion Detection Systems (HIDS), Network Based Indrusion Detection Systems (NIDS), OTHER?   | Fleet utilizes several security monitoring solutions depending on the requirements of the system. For instance, given the highly containerized and serverless environment, FIM would not apply. But, we do use tools such as (but not limited to) AWS GuardDuty, AWS CloudTrail, and VPC Flow Logs to actively monitor the security of our environments.                                                               |


## Privacy

Please also see [privacy](https://fleetdm.com/legal/privacy)
| Question | Answer                                                                                                                                                 |
| ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| Is Fleet a processor, controller, or joint controller in its relationship with its customer?  | Fleet is a processor.                                                               |


## Sub-processors

| Question | Answer |
| ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| Does Fleet possess an APEC PRP certification issued by a certification body (or Accountability Agent)? If not, is Fleet able to provide any evidence that the PRP requirements are being met as it relates to the Scoped Services provided to its customers? | Fleet has not undergone APEC PRP certification but has undergone an external security audit that included pen testing. For a complete list of subprocessors, please refer to our [trust page](https://trust.fleetdm.com/subprocessors). |




## Security policies

Security policies are the foundation of our security program and guide team members in understanding the who, what, and why regarding security at Fleet.


### Information security policy and acceptable use policy

This Information Security Policy is intended to protect Fleet Device Management Inc's employees, contractors, partners, customers, and the company from illegal or damaging actions by individuals, either knowingly or unknowingly.

Internet/Intranet/Extranet-related systems are the property of Fleet Device Management Inc. This includes, but is not limited to

- computer equipment.
- software.
- operating systems.
- storage media.
- network accounts providing electronic mail.
- web browsing.
- file transfers

These systems are to be used for business purposes, serving the interests of the company, and of our clients and customers in the course of normal operations.

Effective security is a team effort. This involves the participation and support of every Fleet Device Management Inc employee or contractor who deals with information and/or information systems. It is every team member's responsibility to read and understand this policy so they know how to conduct their activities accordingly.

All Fleet employees and long-term collaborators are expected to read and electronically sign the *acceptable use of end-user computing* policy. They should also be aware of the others and consult them as needed. This is to make sure systems built and used are done in a compliant manner.


### Acceptable use of end-user computing

> _Created from [JupiterOne/security-policy-templates](https://github.com/JupiterOne/security-policy-templates). [CC BY-SA 4 license](https://creativecommons.org/licenses/by-sa/4.0/)_

| Policy owner   | Effective date |
| -------------- | -------------- |
| @sampfluger88 | 2024-03-14   |

Fleet requires all team members to comply with the following acceptable use requirements and procedures:

- The use of Fleet computing systems is subject to monitoring by Fleet IT and/or Security teams.

- Fleet team members must not leave computing devices (including laptops and smart devices) used for business purposes, including company-provided and BYOD devices, unattended in public. Unattended devices (even in private spaces) must be locked with the lid closed or through the OS screen lock mechanism.

- Device encryption must be enabled for all mobile devices accessing company data, such as whole-disk encryption for all laptops. This is automatically enforced on Fleet-managed macOS devices and must be manually configured for any unmanaged workstations.

- Anti-malware or equivalent protection and monitoring must be installed and enabled on all endpoint systems that may be affected by malware, including workstations, laptops, and servers. This is automatically enforced on Fleet-managed macOS devices and must be manually configured for any unmanaged workstations.

- Teams must exclusively use legal software with a valid license installed through the "app store" or trusted sources. Well-documented open source software can be used. If in doubt, ask in [#g-security](https://fleetdm.slack.com/archives/C037Q8UJ0CC).  

- Avoid sharing credentials. Secrets must be stored safely, using features such as GitHub secrets. For accounts and other sensitive data that need to be shared, use the company-provided password manager (1Password). If you don't know how to use the password manager or safely access secrets, please ask in [#g-security](https://fleetdm.slack.com/archives/C037Q8UJ0CC)!

- Sanitize and remove any sensitive or confidential information prior to posting. At Fleet, we are public by default. Sensitive information from logs, screenshots, or other types of data (eg. debug profiles) should not be shared publicly.

- Fleet team members must not let anyone else use Fleet-provided and managed workstations unsupervised, including family members and support personnel of vendors. Use screen sharing instead of allowing them to access your system directly, and never allow unattended screen sharing.

- Device operating systems must be kept up to date. Fleet-managed macOS workstations will receive prompts for updates to be installed, and unmanaged devices are to be updated by the team member using them. Access may be revoked for devices not kept up to date.

- Team members must not store sensitive data on external storage devices (USB sticks, external hard drives).

- The use of Fleet company accounts on "shared" computers, such as hotel kiosk systems, is strictly prohibited.

- Lost or stolen devices (laptops, or any other company-owned or personal devices used for work purposes) must be reported as soon as possible. Minutes count when responding to security incidents triggered by missing devices. Report a lost, stolen, or missing device by posting in [#g-security](https://fleetdm.slack.com/archives/C037Q8UJ0CC), or use the security@ (fleetdm.com) email alias if you no longer have access to Slack. Include your name, the type of device, timeline (when were you last in control of the device?), whether the device was locked, whether any sensitive information is on the device, and any other relevant information in the report.

When in doubt, **ASK!** (in [#g-security](https://fleetdm.slack.com/archives/C037Q8UJ0CC))


### Access control policy

> _Created from [JupiterOne/security-policy-templates](https://github.com/JupiterOne/security-policy-templates). [CC BY-SA 4 license](https://creativecommons.org/licenses/by-sa/4.0/)_

| Policy owner   | Effective date |
| -------------- | -------------- |
| @sampfluger88 | 2024-03-14      |

Fleet requires all workforce members to comply with the following acceptable use requirements and procedures, such that:

- Access to all computing resources, including servers, end-user computing devices, network equipment, services, and applications, must be protected by strong authentication, authorization, and auditing.

- Interactive user access to production systems must be associated with an account or login unique to each user.

- All credentials, including user passwords, service accounts, and access keys, must meet the length, complexity, age, and rotation requirements defined in Fleet security standards.

- Use a strong password and two-factor authentication (2FA) whenever possible to authenticate to all computing resources (including both devices and applications).

- 2FA is required to access any critical system or resource, including but not limited to resources in Fleet production environments.

- Unused accounts, passwords, and access keys must be removed within 30 days.

- A unique access key or service account must be used for different applications or user access.

- Authenticated sessions must time out after a defined period of inactivity.



- [Asset management policy](https://fleetdm.com/handbook/digital-experience/security#asset-management-policy)
- [Business continuity and disaster recovery policy](https://fleetdm.com/handbook/digital-experience/security#business-continuity-and-disaster-recovery-policy)
- [Data management policy](https://fleetdm.com/handbook/digital-experience/security#data-management-policy)
- [Encryption policy](https://fleetdm.com/handbook/digital-experience/security#encryption-policy)
- [Human resources security policy](https://fleetdm.com/handbook/digital-experience/security#human-resources-security-policy)
- [Incident response policy](https://fleetdm.com/handbook/digital-experience/security#incident-response-policy)
- [Operations security and change management policy](https://fleetdm.com/handbook/digital-experience/security#operations-security-and-change-management-policy)
- [Risk management policy](https://fleetdm.com/handbook/digital-experience/security#risk-management-policy)
- [Secure software development and product security policy](https://fleetdm.com/handbook/digital-experience/security#secure-software-development-and-product-security-policy)
- [Security policy management policy](https://fleetdm.com/handbook/digital-experience/security#security-policy-management-policy)
- [Third-party management policy](https://fleetdm.com/handbook/digital-experience/security#third-party-management-policy)


### Access authorization and termination

Fleet policy requires that:

- Access authorization shall be implemented using role-based access control (RBAC) or a similar mechanism.

- Standard access based on a user's job role may be pre-provisioned during employee onboarding. All subsequent access requests to computing resources must be approved by the requestor’s manager prior to granting and provisioning of access.

- Access to critical resources, such as production environments, must be approved by the security team in addition to the requestor’s manager.

- Access must be reviewed regularly and revoked if no longer needed.

- Upon the termination of employment, all system access must be revoked, and user accounts terminated within 24-hours or one business day, whichever is shorter.

- All system access must be reviewed at least annually and whenever a user's job role changes.


### Shared secrets management


Fleet policy requires that:

- Use of shared credentials/secrets must be minimized.

- If required by Digital Experience, secrets/credentials must be shared securely and stored in encrypted vaults that meet the Fleet data encryption standards.


### Privileged access management


Fleet policy requires that:

- Automation with service accounts must be used to configure production systems when technically feasible.

- Use of high privilege accounts must only be performed when absolutely necessary.


### Asset management policy

> _Created from [JupiterOne/security-policy-templates](https://github.com/JupiterOne/security-policy-templates). [CC BY-SA 4 license](https://creativecommons.org/licenses/by-sa/4.0/)_

| Policy owner   | Effective date |
| -------------- | -------------- |
| @sampfluger88 | 2024-03-14       |

You can't protect what you can't see. Therefore, Fleet must maintain an accurate and up-to-date inventory of its physical and digital assets.

Fleet policy requires that:

- IT and/or security must maintain an inventory of all critical company assets, both physical and logical.

- All assets should have identified owners and a risk/data classification tag.

- All company-owned computer purchases must be tracked.


### Business continuity and disaster recovery policy

| Question | Answer                                                                                                                                                 |
| ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| Please provide your application/solution disaster recovery RTO/RPO | RTO and RPO intervals differ depending on the service that is impacted. Please refer to https://fleetdm.com/handbook/digital-experience/security-policies#business-continuity-and-disaster-recovery-policy


> _Created from [JupiterOne/security-policy-templates](https://github.com/JupiterOne/security-policy-templates). [CC BY-SA 4 license](https://creativecommons.org/licenses/by-sa/4.0/)_

| Policy owner   | Effective date |
| -------------- | -------------- |
| @sampfluger88 | 2024-03-14       |

The Fleet business continuity and disaster recovery plan establishes procedures to recover Fleet following a disruption resulting from a disaster. 

Fleet policy requires that:

- A plan and process for business continuity and disaster recovery (BCDR), will be defined and documented including the backup and recovery of critical systems and data,.

- BCDR shall be simulated and tested at least once a year. 

- Security controls and requirements will be maintained during all BCDR activities.


## Business continuity plan


#### Line of Succession


The following order of succession to make sure that decision-making authority for the Fleet Contingency Plan is uninterrupted. The Chief Executive Officer (CEO) is responsible for ensuring the safety of personnel and the execution of procedures documented within this Fleet Contingency Plan. The CTO is responsible for the recovery of Fleet technical environments. If the CEO or Head of Engineering cannot function as the overall authority or choose to delegate this responsibility to a successor, the board of directors shall serve as that authority or choose an alternative delegate.

For technical incidents:
- CTO (Luke Heath)
- CEO (Mike McNeil)

For business/operational incidents:
- CEO (Mike McNeil)
- Head of Digital Experience (Sam Pfluger)


### Response Teams and Responsibilities


The following teams have been developed and trained to respond to a contingency event affecting Fleet infrastructure and systems.

- **Infrastructure** is responsible for recovering the Fleet automatic update service hosted environment. The team includes personnel responsible for the daily IT operations and maintenance. The team reports to the CTO.

- **People Ops** is responsible for ensuring the physical safety of all Fleet personnel and coordinating the response to incidents that could impact it. Fleet has no physical site to recover. The team reports to the CEO.

- **Security** is responsible for assessing and responding to all cybersecurity-related incidents according to Fleet Incident Response policy and procedures. The security team shall assist the above teams in recovery as needed in non-cybersecurity events. The team leader is the CTO.

Members of the above teams must maintain local copies of the contact information of the BCDR succession team. Additionally, the team leads must maintain a local copy of this policy in the event Internet access is not available during a disaster scenario.

All executive leadership shall be informed of any and all contingency events.

Current Fleet continuity leadership team members include the CEO and CTO.

### General Disaster Recovery Procedures


#### Recovery objectives

Our Recovery Time Objective (RTO) is the goal we set for the maximum length of time it should take to restore normal operations following an outage or data loss. Our Recovery Point Objective (RPO) is the goal we set for the maximum amount of time we can tolerate losing data.

- RTO: 1 hour
- RPO: 24 hours


#### Notification and Activation Phase

This phase addresses the initial actions taken to detect and assess the damage inflicted by a disruption to Fleet Device Management. Based on the assessment of the Event, sometimes, according to the Fleet Incident Response Policy, the Contingency Plan may be activated by either the CEO or CTO.  The Contingency Plan may also be triggered by the Head of Security in the event of a cyber disaster.

The notification sequence is listed below:

1. The first responder is to notify the CTO. All known information must be relayed.
2. The CTO is to contact the Response Teams and inform them of the event. The CTO or delegate is responsible to beginning the assessment procedures.
3. The CTO is to notify team members and direct them to complete the assessment procedures outlined below to determine the extent of the issue and estimated recovery time. 
4. The Fleet Contingency Plan is to be activated if one or more of the following criteria are met:
    - Fleet automatic update service will be unavailable for more than 48 hours.
    - Cloud infrastructure service is damaged and will be unavailable for more than 24 hours.
    - Other criteria, as appropriate and as defined by Fleet.
5. If the plan is to be activated, the CTO is to notify and inform team members of the event details.
6. Upon notification from the CTO, group leaders and managers must notify their respective teams. Team members are to be informed of all applicable information and prepared to respond and relocate if necessary.
7. The CTO is to notify the remaining personnel and executive leadership on the general status of the incident.
8. Notification can be via Slack, email, or phone.
9. The CTO posts a blog post explaining that the service is down and recovery is in progress.


#### Reconstitution Phase

This section discusses activities necessary for restoring full Fleet operations at the original or new site. The goal is to restore full operations within 24 hours of a disaster or outage. The goal is to provide a seamless transition of operations.

1. Contact Partners and Customers affected to begin initial communication - CTO
2. Assess damage to the environment - Infrastructure
3. Create a new production environment using new environment bootstrap automation - Infrastructure
4. Make sure secure access to the new environment - Security
5. Begin code deployment and data replication using pre-established automation - DevOps
6. Test new environment and applications using pre-written tests - DevOps
7. Test logging, security, and alerting functionality - DevOps and Security
8. Assure systems and applications are appropriately patched and up to date -DevOps
9. Update DNS and other necessary records to point to the new environment - DevOps
10. Update Partners and Customers affected through established channels - DevOps


#### Plan Deactivation

If the Fleet environment has been restored, the continuity plan can be deactivated. If the disaster impacted the company and not the service or both, make sure that any leftover systems created temporarily are destroyed.


## Data management policy

| Question | Answer                                                                                                                                                 |
| ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| Should the need arise during an active relationship, how can our Data be removed from the Fleet's environment?   | Customer data is primarily stored in RDS, S3, and Cloudwatch logs. Deleting these resources will remove the vast majority of customer data. Fleet can take further steps to remove data on demand, including deleting individual records in monitoring systems if requested.                                                                                                                              |
| Does Fleet support secure deletion (e.g., degaussing/cryptographic wiping) of archived and backed-up data as determined by the tenant? | Since all data is encrypted at rest, Fleet's secure deletion practice is to delete the encryption key. Fleet does not host customer services on-premise, so hardware specific deletion methods (such as degaussing) do not apply. |
| Does Fleet have a Data Loss Prevention (DLP) solution or compensating controls established to mitigate the risk of data leakage? | In addition to data controls enforced by Google Workspace on corporate endpoints, Fleet applies appropiate security controls for data depending on the requirements of the data, including but not limited to minimum access requirements. |
| Can your organization provide a certificate of data destruction if required?    |     No, physical media related to a certificate of data destruction  is managed by AWS. Media storage devices used to store customer data are classified by AWS as critical and treated accordingly, as high impact, throughout their life-cycles. AWS has exacting standards on how to install, service, and eventually destroy the devices when they are no longer useful. When a storage device has reached the end of its useful life, AWS decommissions media using techniques detailed in NIST 800-88. Media that stored customer data is not removed from AWS control until it has been securely decommissioned.   |
| Who has access to authentication tokens? And does the access gets monitored on a regular basis?  | Users of Fleet software have access to their own authentication tokens. Fleet engineers and support staff may be approved for access to these tokens with consent from the customer. All access to customer production data generates logs in Fleet's infrastructure.  |
| Does Fleet have in house rules in place for weak passwords or are they using some 3rd party solution?  | SAML SSO is used for production infrastructure. The IdP (Google) enforces password complexity requirements.  |


> _Created from [JupiterOne/security-policy-templates](https://github.com/JupiterOne/security-policy-templates). [CC BY-SA 4 license](https://creativecommons.org/licenses/by-sa/4.0/)_

This policy outlines the requirements and controls/procedures Fleet has implemented to manage the end-to-end data lifecycle, from data creation/acquisition to retention and deletion.

Additionally, this policy outlines requirements and procedures to create and maintain retrievable exact copies of electronically protected health information(ePHI), PII, and other critical customer/business data.

Data backup is an important part of the day-to-day operations of Fleet. To protect the confidentiality, integrity, and availability of sensitive and critical data, both for Fleet and Fleet Customers, complete backups are done daily to assure that data remains available when needed and in case of a disaster.

Fleet policy requires that:
- Data should be classified at the time of creation or acquisition.
- Fleet must maintain an up-to-date inventory and data flows mapping of all critical data.
- All business data should be stored or replicated to a company-controlled repository.
- Data must be backed up according to the level defined in Fleet data classification.
- Data backup must be validated for integrity.
- The data retention period must be defined and comply with any and all applicable regulatory and contractual requirements.  More specifically, **data and records belonging to Fleet platform customers must be retained per Fleet product terms and conditions and/or specific contractual agreements.**
- By default, all security documentation and audit trails are kept for a minimum of seven years unless otherwise specified by Fleet data classification, specific regulations, or contractual agreement.


### Data classification model

Fleet defines the following four data classifications:

- **Critical**
- **Confidential**
- **Internal**
- **Public**

As Fleet is an open company by default, most of our data falls into **public**.


#### Definitions and Examples

**Critical** data includes data that must be protected due to regulatory requirements, privacy, and/or security sensitivities.

Unauthorized disclosure of critical data may result in major disruption to business operations, significant cost, irreparable reputation damage, and/or legal prosecution of the company.

External disclosure of critical data is strictly prohibited without an approved process and agreement in place.

*Example Critical Data Types* include

- PII (personal identifiable information)
- ePHI (electronically protected health information)
- Production security data, such as
    - Production secrets, passwords, access keys, certificates, etc.
    - Production security audit logs, events, and incident data
- Production customer data


**Confidential** and proprietary data represents company secrets and is of significant value to the company.

Unauthorized disclosure may result in disruption to business operations and loss of value.

Disclosure requires the signing of NDA and management approval.

*Example Confidential Data Types* include

- Business plans
- Employee/HR data
- News and public announcements (pre-announcement)
- Patents (pre-filing)
- Production metadata (server logs, non-secret configurations, etc.)
- Non-production security data, including
  - Non-prod secrets, passwords, access keys, certificates, etc.
  - Non-prod security audit logs, events, and incident data

**Internal** data contains information used for internal operations.

Unauthorized disclosure may cause undesirable outcomes to business operations.

Disclosure requires management approval.  NDA is usually required but may be waived on a case-by-case basis.

**Public** data is Information intended for public consumption. Although
non-confidential, the integrity and availability of public data should be
protected.

*Example Internal Data Types* include

- Fleet source code.
- news and public announcements (post-announcement).
- marketing materials.
- product documentation.
- content posted on the company website(s) and social media channel(s).


#### Data Handling Requirements Matrix

Requirements for data handling, such as the need for encryption and the duration of retention, are defined according to the Fleet data classification.

| Data             | Labeling or Tagging | Segregated Storage | Endpoint Storage | Encrypt At Rest | Encrypt In Transit | Encrypt In Use | Controlled Access | Monitoring | Destruction at Disposal | Retention Period | Backup Recovery |
|------------------|---------------------|--------------------|------------------|-----------------|--------------------|----------------|-------------------|------------|------------------------|------------------|-----------------|
| **Critical**     | Required            | Required           | Prohibited       | Required        | Required           | Required       | Access is blocked to end users by default; Temporary access for privileged users only | Required   | Required   | seven years for audit trails; Varies for customer-owned data† | Required   |
| **Confidential** | Required            | N/R                | Allowed          | Required        | Required           | Required       | All access is based on need-to-know | Required   | Required   | Seven years for official documentation; Others vary based on business need | Required   |
| **Internal**     | Required            | N/R                | Allowed          | N/R             | N/R                | N/R            | All employees and contractors (read); Data owners and authorized individuals (write) | N/R | N/R | Varies based on business need | Optional   |
| **Public**       | N/R                 | N/R                | Allowed          | N/R             | N/R                | N/R            | Everyone (read); Data owners and authorized individuals (write) | N/R     | N/R     | Varies based on business need | Optional   |

N/R = Not Required

† Customer-owned data is stored for as long as they remain as a Fleet customer, or as required by regulations, whichever is longer. Customers may request their data to be deleted at any time; unless retention is required by law.

Most Fleet data is **public** yet retained and backed up not due to our data handling requirements but simply business requirements.


#### Customer data deletion

This process is followed when offboarding a customer and deleting all of the production customer data.

1. `terraform destroy` the infrastructure for the customer. This triggers immediate deletion of the RDS database and all automated snapshots, along with immediate deletion of the ElastiCache Redis instance. Secrets are marked for deletion with a 7 day recovery window. Cloudwatch (server) logs are automatically deleted after the retention window expires.
2. Manually delete any manual database snapshots. The engineer should verify that there are no manual snapshots remaining for this customer.
3. Commit a removal of all the Terraform files for the customer.


## Encryption policy

| Question | Answer                                                                                                                                                 |
| ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| Does Fleet have a cryptographic key management process (generation, exchange, storage, safeguards, use, vetting, and replacement), that is documented and currently implemented, for all system components? (e.g. database, system, web, etc.)   | All data is encrypted at rest using methods appropriate for the system (ie KMS for AWS based resources). Data going over the internet is encrypted using TLS or other appropiate transport security. |
| Does Fleet allow customers to bring and their own encryption keys? | By default, Fleet does not allow for this, but if absolutely required, Fleet can accommodate this request. |
| Does Fleet have policy regarding key rotation ? Does rotation happens after every fixed time period or only when there is evidence of key leak ?  | TLS certificates are managed by AWS Certificate Manager and are rotated automatically annually.  |


> _Created from [JupiterOne/security-policy-templates](https://github.com/JupiterOne/security-policy-templates). [CC BY-SA 4 license](https://creativecommons.org/licenses/by-sa/4.0/)_

| Policy owner   | Effective date |
| -------------- | -------------- |
| @sampfluger88 | 2024-03-14     |

Fleet requires all workforce members to comply with the encryption policy, such that:

- The storage drives of all Fleet-owned workstations must be encrypted and enforced by the IT and/or security team.
- Confidential data must be stored in a manner that supports user access logs.
- All Production Data at rest is stored on encrypted volumes.
- Volume encryption keys and machines that generate volume encryption keys are protected from unauthorized access. Volume encryption key material is protected with access controls such that the key material is only accessible by privileged accounts.
- Encrypted volumes use strong cipher algorithms, key strength, and key management process as defined below.
- Data is protected in transit using recent TLS versions with ciphers recognized as secure.


### Local disk/volume encryption

Encryption and key management for local disk encryption of end-user devices follow the defined best practices for Windows, macOS, and Linux/Unix operating systems, such as Bitlocker and FileVault. 


### Protecting data in transit

- All external data transmission is encrypted end-to-end. This includes, but is not limited to, cloud infrastructure and third-party vendors and applications.
- Transmission encryption keys and systems that generate keys are protected from unauthorized access.
- Transmission encryption key materials are protected with access controls and may only be accessed by privileged accounts.
- TLS endpoints must score at least an "A" on SSLLabs.com.
- Transmission encryption keys are limited to use for one year and then must be regenerated.


### Authorized Sub-Processors for Fleet Cloud services

| Sub-processor Name | Purpose | Location |
| ------------------ | ------- | -------- |
| Amazon Web Services, Inc. and sub-processors located at https://aws.amazon.com/compliance/sub-processors/ | Database hosting platform | USA |


### Human resources security policy

> _Created from [JupiterOne/security-policy-templates](https://github.com/JupiterOne/security-policy-templates). [CC BY-SA 4 license](https://creativecommons.org/licenses/by-sa/4.0/)_

| Policy owner   | Effective date |
| -------------- | -------------- |
| @mikermcneil | 2022-06-01     |


Fleet is committed to ensuring all workforce members participate in security and compliance in their roles at Fleet. We encourage self-management and reward the right behaviors. 

Fleet policy requires all workforce members to comply with the HR Security Policy.

Fleet policy requires that:


- Background verification checks on candidates for all Fleet employees and contractors must be carried out in accordance with relevant laws, regulations, and ethics. These checks should be proportional to the business requirements, the classification of the information to be accessed, and the perceived risk.
- Employees, contractors, and third-party users must agree to and sign the terms and conditions of their employment contract and comply with acceptable use.
- Employees will perform an onboarding process that familiarizes them with the environments, systems, security requirements, and procedures that Fleet already has in place. Employees will also have ongoing security awareness training that is audited.
- Employee offboarding will include reiterating any duties and responsibilities still valid after terminations, verifying that access to any Fleet systems has been removed, and ensuring that all company-owned assets are returned.
- Fleet and its employees will take reasonable measures to make sure no sensitive data is transmitted via digital communications such as email or posted on social media outlets.
- Fleet will maintain a list of prohibited activities that will be part of onboarding procedures and have training available if/when the list of those activities changes.
- A fair disciplinary process will be used for employees suspected of committing security breaches. Fleet will consider multiple factors when deciding the response, such as whether or not this was a first offense, training, business contracts, etc. Fleet reserves the right to terminate employees in the case of severe cases of misconduct.
- Fleet will maintain a reporting structure that aligns with the organization's business lines and/or individual's functional roles. The list of employees and reporting structure must be available to [all employees](https://docs.google.com/spreadsheets/d/1OSLn-ZCbGSjPusHPiR5dwQhheH1K8-xqyZdsOe9y7qc/edit#gid=0).
- Employees will receive regular feedback and acknowledgment from their managers and peers. Managers will give constant feedback on performance, including but not limited to during regular one-on-one meetings.
- Fleet will publish job descriptions for available positions and conduct interviews to assess a candidate's technical skills as well as soft skills prior to hiring.
- Background checks of an employee or contractor must be performed by operations and/or the hiring team before we grant the new employee or contractor access to the Fleet production environment.
- A list of employees and contractors will be maintained, including their titles and managers, and made available to everyone internally.
- An [anonymous](https://docs.google.com/forms/d/e/1FAIpQLSdv2abLfCUUSxFCrSwh4Ou5yF80c4V2K_POoYbHt3EU1IY-sQ/viewform?vc=0&c=0&w=1&flr=0&fbzx=4276110450338060288) form to report unethical behavior will be provided to employees.


### Incident response policy

> _Created from [JupiterOne/security-policy-templates](https://github.com/JupiterOne/security-policy-templates). [CC BY-SA 4 license](https://creativecommons.org/licenses/by-sa/4.0/). Based on the SANS incident response process._

Fleet policy requires that:

- All computing environments and systems must be monitored in accordance with Fleet policies and procedures specified in the Fleet handbook.
- Alerts must be reviewed to identify security incidents.
- Incident response procedures are invoked upon discovery of a valid security incident.
- Incident response team and management must comply with any additional requests by law enforcement in the event of a criminal investigation or national security, including but not limited to warranted data requests, subpoenas, and breach notifications.


### Incident response plan


#### Security Incident Response Team (SIRT)

The Security Incident Response Team (SIRT) is responsible for

- Reviewing analyzing, and logging all received reports and tracking their statuses.
- Performing investigations, creating and executing action plans, and post-incident activities.
- Collaboration with law enforcement agencies.

Current members of the Fleet SIRT:
- CTO
- CEO
- VP of Customer Success


#### Incident Management Process

Fleet's incident response classifies security-related events into the following categories:
- **Events** - Any observable computer security-related occurrence in a system or network with a negative consequence. Examples:
  - Hardware component failing, causing service outages.
  - Software error causing service outages.
  - General network or system instability.

- **Precursors** - A sign that an incident may occur in the future. Examples:
  - Monitoring system showing unusual behavior.
  - Audit log alerts indicated several failed login attempts.
  - Suspicious emails that target specific Fleet staff members with administrative access to production systems.
  - Alerts raised from a security control source based on its monitoring policy, such as:
    - Google Workspace (user authentication activities)
    - Fleet (internal instance)
    - Syslog events from servers

- **Indications** - A sign that an incident may have occurred or may be occurring at the present time. Examples:
  - Alerts for modified system files or unusual system accesses.
  - Antivirus alerts for infected files or devices.
  - Excessive network traffic directed at unexpected geographic locations.

- **Incidents** - A confirmed attack/indicator of compromise or a validated violation of computer security policies or acceptable use policies, often resulting in data breaches. Examples:
  - Unauthorized disclosure of sensitive data
  - Unauthorized change or destruction of sensitive data
  - A data breach accomplished by an internal or external entity
  - A Denial-of-Service (DoS) attack causing a critical service to become
      unreachable

Fleet employees must report any unauthorized or suspicious activity seen on
production systems or associated with related communication systems (such as
email or Slack). In practice, this means keeping an eye out for security events
and letting the Security team know about any observed precursors or indications
as soon as they are discovered.

Incidents of a severity/impact rating higher than **MINOR** shall trigger the response process.


#### I - Identification and Triage

1. Immediately upon observation, Fleet members report suspected and known Events, Precursors, Indications, and Incidents in one of the following ways:
  - Direct report to management, CTO, CEO, or other
  - Email
  - Phone call
  - Slack
2. The individual receiving the report facilitates the collection of additional information about the incident, as needed, and notifies the CTO (if not already done).
3. The CTO determines if the issue is an Event, Precursor, Indication, or Incident.
  - If the issue is an event, indication, or precursor, the CTO forwards it to the appropriate resource for resolution.
    - Non-Technical Event (minor infringement): the CTO of the designee creates an appropriate issue in GitHub and further investigates the incident as needed.
    - Technical Event: Assign the issue to a technical resource for resolution. This resource may also be a contractor or outsourced technical resource in the event of a lack of resource or expertise in the area.
  - If the issue is a security incident, the CTO activates the Security Incident Response Team (SIRT) and notifies senior leadership by email.
    - If a non-technical security incident is discovered, the SIRT completes the investigation, implements preventative measures, and resolves the security incident.
    - Once the investigation is completed, progress to Phase V, Follow-up.
    - If the issue is a technical security incident, commence to Phase II: Containment.
    - The Containment, Eradication, and Recovery Phases are highly technical. It is important to have them completed by a highly qualified technical security resource with oversight by the SIRT team.
    - Each individual on the SIRT and the technical security resource document all measures taken during each phase, including the start and end times of all efforts.
    - The lead member of the SIRT team facilitates the initiation of an Incident ticket in GitHub Security Project and documents all findings and details in the ticket.

           * The intent of the Incident ticket is to provide a summary of all
             events, efforts, and conclusions of each Phase of this policy and
             procedures.
           * Each Incident ticket should contain sufficient details following
             the [SANS Security Incident Forms templates](https://www.sans.org/score/incident-forms/),
             as appropriate.

3. The CTO, Privacy Officer, or Fleet representative appointed
   notifies any affected Customers and Partners. If no Customers and Partners
   are affected, notification is at the discretion of the Security and Privacy
   Officer.
   
   Fleet’s incident response policy is to report significant cyber incidents within 
   24 hours.
    - Reporting Timeline – 24 hours after determining a cyber incident has occurred.
    - Definitions – Significant cyber incidents are defined as an incident or group 
         of incidents that are likely to result in demonstrable harm to Fleet or Fleet’s 
         customers.
    - Reporting Mechanism – Reports to be provided to customers via email 
         correspondence and Slack.

4. In the case of a threat identified, the Head of Security is to form a team to
   investigate and involve necessary resources, both internal to Fleet and
   potentially external.


#### II - Containment (Technical)

In this Phase, Fleet's engineers and security team attempt to contain the
security incident. It is essential to take detailed notes during the
security incident response process. This provides that the evidence gathered
during the security incident can be used successfully during prosecution, if
appropriate.

1. Review any information that has been collected by the Security team or any
   other individual investigating the security incident.
2. Secure the blast radius (i.e., a physical or logical network perimeter or
   access zone).
3. Perform the following forensic analysis preparation, as needed:
    - Securely connect to the affected system over a trusted connection.
    - Retrieve any volatile data from the affected system.
    - Determine the relative integrity and the appropriateness of backing the system up.
    - As necessary, take a snapshot of the disk image for further forensic, and if appropriate, back up the system.
    - Change the password(s) to the affected system(s).
    - Determine whether it is safe to continue operations with the affected system(s).
    - If it is safe, allow the system to continue to functioning; and move to Phase V, Post Incident Analysis and Follow-up.
    - If it is NOT safe to allow the system to continue operations, discontinue the system(s) operation and move to Phase III, Eradication.
    - The individual completing this phase provides written communication to the SIRT.

4. Complete any documentation relative to the security incident containment on the Incident ticket, using [SANS IH Containment Form](https://www.sans.org/media/score/incident-forms/IH-Containment.pdf) as a template.
5. Continuously apprise Senior Management of progress.
6. Continue to notify affected Customers and Partners with relevant updates as
   needed.


#### III - Eradication (Technical)

The Eradication Phase represents the SIRT's effort to remove the cause and the
resulting security exposures that are now on the affected system(s).

1. Determine symptoms and cause related to the affected system(s).
2. Strengthen the defenses surrounding the affected system(s), where possible (a
   risk assessment may be needed and can be determined by the Head of Security).
   This may include the following:
     - An increase in network perimeter defenses.
     - An increase in system monitoring defenses.
     - Remediation ("fixing") any security issues within the affected system, such as removing unused services/general host hardening techniques.

3. Conduct a detailed vulnerability assessment to verify all the holes/gaps that can be exploited are addressed.
    - If additional issues or symptoms are identified, take appropriate preventative measures to eliminate or minimize potential future compromises.

4. Update the Incident ticket with Eradication details, using [SANS IH Eradication Form](https://www.sans.org/media/score/incident-forms/IH-Eradication.pdf) as a template.
5. Update the documentation with the information learned from the vulnerability assessment, including the cause, symptoms, and the method used to fix the problem with the affected system(s).
6. Apprise Senior Management of the progress.
7. Continue to notify affected Customers and Partners with relevant updates as needed.
8. Move to Phase IV, Recovery.


#### IV - Recovery (Technical)

The Recovery Phase represents the SIRT's effort to restore the affected
system(s) to operation after the resulting security exposures, if any, have
been corrected.

The technical team determines if the affected system(s) have been changed in any way.
1. If they have, the technical team restores the system to its proper, intended functioning ("last known good").
2. Once restored, the team validates that the system functions the way it was intended/had functioned in the past. This may require the involvement of the business unit that owns the affected system(s).
3. If the operation of the system(s) had been interrupted (i.e., the system(s) had been taken offline or dropped from the network while triaged), restart the restored and validated system(s) and monitor for behavior.
4. If the system had not been changed in any way but was taken offline (i.e., operations had been interrupted), restart the system and monitor for proper behavior.
5. Update the documentation with the detail that was determined during this phase.
6. Apprise Senior Management of progress.
7. Continue to notify affected Customers and Partners with relevant updates as needed. 
8. Move to Phase V, Follow-up.


#### V - Post-Incident Analysis (Technical and Non-Technical)

The Follow-up phase represents the review of the security incident to look for
"lessons learned" and determine whether the process could have
been improved. It is recommended all security incidents be reviewed
shortly after resolution to determine where response could be improved.
Timeframes may extend to one to two weeks post-incident.

1. Responders to the security incident (SIRT Team and technical security resource) meet to review the documentation collected during the security incident.
2. A "lessons learned" section is written and attached to the Incident ticket.
    - Evaluate the cost and impact of the security incident on Fleet using the documents provided by the SIRT and the technical security resource.
    - Determine what could be improved. This may include:
        - Systems and processes adjustments
        - Awareness training and documentation
        - Implementation of additional controls
    - Communicate these findings to Senior Management for approval and implementation of any recommendations made post-review of the security incident.
    - Carry out recommendations approved by Senior Management; sufficient budget, time, and resources should be committed to this activity.
3. Ensure all incident-related information is recorded and retained as described in Fleet Auditing requirements and Data Retention standards.
4. Close the security incident.


#### Periodic Evaluation

It is important to note that the security incident response processes
should be periodically reviewed and evaluated for effectiveness. This
also involves appropriate training of resources expected to respond to security
incidents, as well as the training of the general population regarding
Fleet's expectations for them relative to security responsibilities. We test the
incident response plan annually.


## Information security roles and responsibilities

> _Created from [Vanta](https://www.vanta.com/) policy templates._

| Policy owner   | Effective date |
| -------------- | -------------- |
| @sampfluger88 | 2024-03-14     |

Fleet Device Management is committed to conducting business in compliance with all applicable laws, regulations, and company policies. Fleet has adopted this policy to outline the security measures required to protect electronic information systems and related equipment from unauthorized use.

| Role                                            | Responsibilities                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| ----------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Board of directors                              | Oversight over risk and internal control for information security, privacy, and compliance<br/> Consults with executive leadership to understand Fleet's security mission and risks and provides guidance to bring them into alignment                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| Executive leadership                            | Approves capital expenditures for information security<br/> Oversight over the execution of the information security risk management program<br/> Communication path to Fleet's board of directors. Meets with the board regularly, including at least one official meeting a year<br/> Aligns information security policy and posture based on Fleet's mission, strategic objectives, and risk appetite                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
CTO                                             | Oversight over information security in the software development process<br/>  Responsible for the design, development, implementation, operation, maintenance and monitoring of development and commercial cloud hosting security controls<br/> Responsible for oversight over policy development <br/>Responsible for implementing risk management in the development process                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| Head of Security                                | Oversight over the implementation of information security controls for infrastructure and IT processes<br/>  Responsible for the design, development, implementation, operation, maintenance, and monitoring of IT security controls<br/> Communicate information security risks to executive leadership<br/> Report information security risks annually to Fleet's leadership and gains approvals to bring risks to acceptable levels<br/>  Coordinate the development and maintenance of information security policies and standards<br/> Work with applicable executive leadership to establish an information security framework and awareness program<br/>  Serve as liaison to the board of directors, law enforcement and legal department.<br/>  Oversight over identity management and access control processes |
| System owners                                   | Manage the confidentiality, integrity, and availability of the information systems for which they are responsible in compliance with Fleet policies on information security and privacy.<br/>  Approve of technical access and change requests for non-standard access                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| Employees, contractors, temporary workers, etc. | Acting at all times in a manner that does not place at risk the security of themselves, colleagues, and the information and resources they have use of<br/>  Helping to identify areas where risk management practices should be adopted<br/>  Adhering to company policies and standards of conduct Reporting incidents and observed anomalies or weaknesses                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| Head of People Operations                       | Ensuring employees and contractors are qualified and competent for their roles<br/>  Ensuring appropriate testing and background checks are completed<br/>  Ensuring that employees and relevant contractors are presented with company policies <br/>  Ensuring that employee performance and adherence to values is evaluated<br/>  Ensuring that employees receive appropriate security training                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         |
| Head of Digital Experience                     | Responsible for oversight over third-party risk management process; responsible for review of vendor service contracts                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
## Network and system hardening standards

Fleet leverages industry best practices for network hardening, which involves implementing a layered defense strategy called defense in depth. This approach ensures multiple security controls protect data and systems from internal and external threats.

1. Network Segmentation:

Objective: Limit the spread of potential threats and control access to sensitive data.

How we implement: 
  - Divide our network into distinct segments or subnets, each with its security controls. 
  - Use VPNs and firewalls to enforce segmentation policies. 
  - Restrict communication between segments to only what is necessary for business operations.

2. Firewall Configuration:

Objective: Control incoming and outgoing network traffic based on predetermined security rules.

How we implement: 
  - Implement a default-deny policy, where all traffic is blocked unless explicitly allowed. 
  - Regularly review and update firewall rules to ensure they align with current security policies and threat landscape.

3. Intrusion Detection and Prevention Systems (IDPS):

Objective: Detect and respond to malicious activity on the network.

How we implement: 
  - Install and configure IDPS to monitor network traffic for signs of malicious activity or policy violations. 
  - Use both signature-based and anomaly-based detection methods.
  - Regularly update IDPS signatures and rules to keep up with emerging threats.

4. Patch Management:

Objective: Ensure all network devices and systems are updated with the latest security patches.

How we implement: 
  - Establish a patch management policy that includes regular scanning for vulnerabilities.
  - Prioritize and apply patches based on the vulnerabilities' severity and the affected systems' criticality.
  - Verify and test patches in a controlled environment before deployment to production systems.

5. Access Control:

Objective: Limit authorized users and devices access to network resources.

How we implement:
  - Implement strong authentication mechanisms, such as multi-factor authentication (MFA).
  - Enforce the principle of least privilege, granting users only the access necessary for their roles.

6. Encryption:

Objective: Protect data in transit and at rest from unauthorized access.

How we implement:
  - Strong encryption protocols like TLS secure data transmitted over the network and at rest.
  - Encrypt sensitive data stored on physical devices, databases, servers, or other object storage.
  - Regularly review and update encryption standards to align with industry best practices.

7. Monitoring and Logging:

Objective: Maintain visibility into network activities and detect potential security incidents.

How we implement:
  - Enable logging on all critical network devices and systems.
  - Use centralized logging solutions to aggregate and analyze log data.
  - Implement real-time monitoring and alerting for suspicious activities or policy violations.

8. Regular Security Assessments:

Objective: Identify and remediate security weaknesses in the network.

How we implement:
  - Regular vulnerability assessments and penetration testing are conducted to evaluate the network's security posture.
  - Address identified vulnerabilities promptly and reassess to verify remediation.
  - Perform regular audits of security policies and procedures to ensure they are effective and up to date.


### Operations security and change management policy

> _Created from [JupiterOne/security-policy-templates](https://github.com/JupiterOne/security-policy-templates). [CC BY-SA 4 license](https://creativecommons.org/licenses/by-sa/4.0/)_

| Policy owner   | Effective date |
| -------------- | -------------- |
| @sampfluger88 | 2024-03-14     |

Fleet policy requires

- All production changes, including but not limited to software deployment, feature toggle enablement, network infrastructure changes, and access control authorization updates, must be invoked through the approved change management process.
- Each production change must maintain complete traceability to fully document the request, including the requestor, date/time of change, actions taken, and results.
- Each production change must include proper approval.
  -  The approvers are determined based on the type of change.
  -  Approvers must be someone other than the author/executor of the change unless they are the DRI for that system.
  -  Approvals may be automatically granted if specific criteria are met.
  -  The auto-approval criteria must be pre-approved by the Head of Security and fully documented and validated for each request.


### Risk management policy

> _Created from [JupiterOne/security-policy-templates](https://github.com/JupiterOne/security-policy-templates). [CC BY-SA 4 license](https://creativecommons.org/licenses/by-sa/4.0/)_

| Policy owner   | Effective date |
| -------------- | -------------- |
| @sampfluger88 | 2024-03-14    |

Fleet policy requires:

- A thorough risk assessment must be conducted to evaluate potential threats and vulnerabilities to the confidentiality, integrity, and availability of sensitive, confidential, and proprietary electronic information Fleet stores, transmits, and/or processes.
- Risk assessments must be performed with any major change to Fleet's business or technical operations and/or supporting infrastructure no less than once per year.
- Strategies shall be developed to mitigate or accept the risks identified in the risk assessment process.
- The risk register is monitored quarterly to assess compliance with the above policy, and document newly discovered or created risks.


### Acceptable Risk Levels

Risks that are either low impact or low probability are generally considered acceptable.

All other risks must be individually reviewed and managed.


### Risk corrective action timelines

| Risk Level | Corrective action timeline |
| ---------- | ------------------- |
| Low        | Best effort         |
| Medium     | 120 days            |
| High       | 30 days             |


### Secure software development and product security policy 

> _Created from [JupiterOne/security-policy-templates](https://github.com/JupiterOne/security-policy-templates). [CC BY-SA 4 license](https://creativecommons.org/licenses/by-sa/4.0/)_

Fleet policy requires that:

1. Fleet software engineering and product development are required to follow security best practices. The product should be "Secure by Design" and "Secure by Default."
2. Fleet performs quality assurance activities. This may include:
    - Peer code reviews prior to merging new code into the main development branch (e.g., main branch)
    - Thorough product testing before releasing it to production (e.g., unit testing and integration testing)
3. Risk assessment activities (i.e., threat modeling) must be performed for a new product or extensive changes to an existing product.
4. Security requirements must be defined, tracked, and implemented.
5. Security analysis must be performed for any open source software and/or third-party components and dependencies included in Fleet software products.
6. Static application security testing (SAST) must be performed throughout development and before each release.
7. Dynamic application security testing (DAST) must be performed before each release.
8. All critical or high severity security findings must be remediated before each release.
9. All critical or high severity vulnerabilities discovered post-release must be remediated in the next release or as per the Fleet vulnerability management policy SLAs, whichever is sooner.
10. Any exception to the remediation of a finding must be documented and approved by the security team or CTO.


### Security policy management policy

> _Created from [JupiterOne/security-policy-templates](https://github.com/JupiterOne/security-policy-templates). [CC BY-SA 4 license](https://creativecommons.org/licenses/by-sa/4.0/)_

| Policy owner   | Effective date |
| -------------- | -------------- |
| @sampfluger88 | 2024-03-14      |

Fleet policy requires that:
- Fleet policies must be developed and maintained to meet all applicable compliance requirements and adhere to security best practices, including but not limited to:
  - SOC 2
- Fleet must annually review all policies.
  - Fleet maintains all policy changes must be approved by Fleet's CTO or CEO. Additionally:
    - Major changes may require approval by Fleet CEO or designee;
    - Changes to policies and procedures related to product development may require approval by the CTO.
- Fleet maintains all policy documents with version control.
- Policy exceptions are handled on a case-by-case basis.
  - All exceptions must be fully documented with business purpose and reasons why the policy requirement cannot be met.
    - All policy exceptions must be approved by Fleet Head of Security and CEO.
    - An exception must have an expiration date no longer than one year from date of exception approval and it must be reviewed and re-evaluated on or before the expiration date.


### Third-party management policy

> _Created from [JupiterOne/security-policy-templates](https://github.com/JupiterOne/security-policy-templates). [CC BY-SA 4 license](https://creativecommons.org/licenses/by-sa/4.0/)_

| Policy owner   | Effective date |
| -------------- | -------------- |
| @mikermcneil | 2022-06-01     |

Fleet makes every effort to assure all third-party organizations are compliant and do not compromise the integrity, security, and privacy of Fleet or Fleet Customer data. Third Parties include Vendors, Customers, Partners, Subcontractors, and Contracted Developers.

- A list of approved vendors/partners must be maintained and reviewed annually.
- Approval from management, procurement, and security must be in place before onboarding any new vendor or contractor that impacts Fleet production systems. Additionally, all changes to existing contract agreements must be reviewed and approved before implementation.
- For any technology solution that needs to be integrated with Fleet production environment or operations, the security team must perform a Vendor Technology Review to understand and approve the risk. Periodic compliance assessment and SLA review may be required.
- Fleet Customers or Partners should not be allowed access outside of their own environment, meaning they cannot access, modify, or delete any data belonging to other third parties.
- Additional vendor agreements are obtained as required by applicable regulatory compliance requirements.


### Anti-corruption policy

> Fleet is committed to ethical business practices and compliance with the law.  All Fleeties are required to comply with the "Foreign Corrupt Practices Act" and anti-bribery laws and regulations in applicable jurisdictions including, but not limited to, the "UK Bribery Act 2010", "European Commission on Anti-Corruption" and others.  The policies set forth in [this document](https://docs.google.com/document/d/16iHhLhAV0GS2mBrDKIBaIRe_pmXJrA1y7-gTWNxSR6c/edit?usp=sharing) go over Fleet's anti-corruption policy in detail.


## Application security

The Fleet community follows best practices when coding. Here are some of the ways we mitigate against the OWASP top 10 issues:

| Question | Answer                                                                                                                                                 |
| ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| Does Fleet use any third party code, including open source code in the development of the scoped application(s)? If yes, please explain.   | Yes. All third party code is managed through standard dependency management tools (Go, Yarn, NPM) and audited for vulnerabilities using GitHub vulnerability scanning.                    |
| Does Fleet have security tooling in place which will enumerate all files and directories to check for appropriate permissions ?  | No. Fleet Cloud does not use VMs and instead uses containers for the Fleet server and AWS hosted MySQL and Redis to reduce surface area for this kind of misconfiguration.  |
| Does Fleet have tooling in place which will provide insights into all API endpoints they have in prod?  | Our load balancer logs/metrics provide insights into all API endpoints that are accessed.  |
| In order to prevent IDOR related bulbs does Fleet plan to have API fuzzer in place?  | No API fuzzer is in place. Instead, IDOR is prevented through explicit authorization checks in each API endpoint and manually tested in regular penetration tests.  |


### Describe your secure coding practices, including code reviews, use of static/dynamic security testing tools, 3rd party scans/reviews.

Code commits to Fleet go through a series of tests, including SAST (static application security
testing). We use a combination of tools, including [gosec](https://github.com/securego/gosec) and
[CodeQL](https://codeql.github.com/) for this purpose.

At least one other engineer reviews every piece of code before merging it to Fleet.
This is enforced via branch protection on the main branch.

The server backend is built in Golang, which (besides for language-level vulnerabilities) eliminates buffer overflow and other memory related attacks.

We use standard library cryptography wherever possible, and all cryptography is using well-known standards.


### SQL injection

All queries are parameterized with MySQL placeholders, so MySQL itself guards against SQL injection and the Fleet code does not need to perform any escaping.


### Broken authentication – authentication, session management flaws that compromise passwords, keys, session tokens etc.


#### Passwords

Fleet supports SAML auth which means that it can be configured such that it never sees passwords.

Passwords are never stored in plaintext in the database. We store a `bcrypt`ed hash of the password along with a randomly generated salt. The `bcrypt` iteration count and salt key size are admin-configurable.


#### Authentication tokens

The size and expiration time of session tokens is admin-configurable. See [The documentation on session duration](https://fleetdm.com/docs/deploying/configuration#session-duration).

It is possible to revoke all session tokens for a user by forcing a password reset.


### Sensitive data exposure – encryption in transit, at rest, improperly implemented APIs.

By default, all traffic between user clients (such as the web browser and fleetctl) and the Fleet server is encrypted with TLS. By default, all traffic between osqueryd clients and the Fleet server is encrypted with TLS. Fleet does not by itself encrypt any data at rest (_however a user may separately configure encryption for the MySQL database and logs that Fleet writes_).


### Broken access controls – how restrictions on what authorized users are allowed to do/access are enforced.

Each session is associated with a viewer context that is used to determine the access granted to that user. Access controls can easily be applied as middleware in the routing table, so the access to a route is clearly defined in the same place where the route is attached to the server see [https://github.com/fleetdm/fleet/blob/main/server/service/handler.go#L114-L189](https://github.com/fleetdm/fleet/blob/main/server/service/handler.go#L114-L189).


### Cross-site scripting – ensure an attacker can’t execute scripts in the user’s browser

We render the frontend with React and benefit from built-in XSS protection in React's rendering. This is not sufficient to prevent all XSS, so we also follow additional best practices as discussed in [https://stackoverflow.com/a/51852579/491710](https://stackoverflow.com/a/51852579/491710).


### Components with known vulnerabilities – prevent the use of libraries, frameworks, other software with existing vulnerabilities.

We rely on GitHub's automated vulnerability checks, community news, and direct reports to discover
vulnerabilities in our dependencies. We endeavor to fix these immediately and would almost always do
so within a week of a report.

Libraries are inventoried and monitored for vulnerabilities. Our process for fixing vulnerable
libraries and other vulnerabilities is available in our
[handbook](https://fleetdm.com/handbook/digital-experience/security#vulnerability-management). We use
[Dependabot](https://github.com/dependabot) to automatically open PRs to update vulnerable dependencies.



## Vulnerability management

At Fleet, we handle software vulnerabilities no matter what their source is.

The process is simple:

1. A person or tool discovers a vulnerability and informs us.
2. Fleet determines if we must fix this vulnerability, and if not, documents why.
3. As long as it respects our remediation timelines and enough time remains for implementation and testing, Fleet fixes vulnerabilities in the next scheduled release. Else, Fleet creates a special release to address the vulnerabilities.



### Timeline

Fleet commits to remediating vulnerabilities on Fleet according to the following:


| Severity                           | Triage | Mitigation | Remediation                               |
| ---------------------------------- | ---------------- | ---------------- | ------------------------------------------------ |
| Critical+ In-the-wild exploitation | 2 business hours | 1 business day         | 3 business days (unless mitigation downgrades severity) |
| Critical                           | 4 business hours | 7 business days           | 30 days                                          |
| High                               | 2 business days  | 14 days          | 30 days                                          |
| Medium                             | 1 week           | 60 days          | 60 days                                          |
| Low                                | Best effort      | Best effort      | Best effort                                      |
| Unspecified                        | 2 business days  | N/A              | N/A                                              |

Refer to our commercial SLAs for more information on the definition of "business hours" and
"business days."

Other resources present in the Fleet repo but not as part of the Fleet product, like our website,
are fixed on a case-by-case scenario depending on the risk.


### Exceptions and extended timelines

We may not be able to fix all vulnerabilities or fix them as rapidly as we would like. For example,
a complex vulnerability reported to us that would require redesigning core parts of the Fleet
architecture would not be fixable in 3 business days.

We ask for vulnerabilities reported by researchers and prefer to perform coordinated disclosure
with the researcher. In some cases, we may take up to 90 days to fix complex issues, in which case
we ask that the vulnerability remains private.

For other vulnerabilities affecting Fleet or code used in Fleet, the Head of Security, CTO and CEO
can accept the risk of patching them according to custom timelines, depending on the risk and
possible temporary mitigations.


### Mapping of CVSSv3 scores to Fleet severity

Fleet adapts the severity assigned to vulnerabilities when needed.

The features we use in a library, for example, can mean that some vulnerabilities in the library are unexploitable. In other cases, it might make the vulnerability easier to exploit. In those cases, Fleet would first categorize the vulnerability using publicly available information, then lower or increase the severity based on additional context.

When using externally provided CVSSv3 scores, Fleet maps them like this:

| CVSSv3 score                       | Fleet severity                      |
| ---------------------------------- | ----------------------------------- |
| 0.0                                | None                                |
| 0.1-3.9                            | Low                                 |
| 4-6.9                              | Medium                              |
| 7-8.9                              | High                                |
| 9-10                               | Critical                            |
| Determined on a case by case basis | Critical + in-the-wild-exploitation |


### Disclosure

Researchers who discover vulnerabilities in Fleet can disclose them as per the [Fleet repository security policy](https://github.com/fleetdm/fleet/security/policy).

If Fleet confirms the vulnerability:

1. Fleet's security team creates a private GitHub security advisory.
2. Fleet asks the researcher if they want credit or anonymity. If the researcher wishes to be credited, we invite them to the private advisory on GitHub.
3. We request a CVE through GitHub.
4. Developers address the issue in a private branch.
5. As we release the fix, we make the advisory public.

Example Fleet vulnerability advisory: [CVE-2022-23600](https://github.com/fleetdm/fleet/security/advisories/GHSA-ch68-7cf4-35vr)


### Vulnerabilities in dependencies

Fleet remediates vulnerabilities related to vulnerable dependencies, but we do not create security advisories on the Fleet repository unless we believe that the vulnerability could impact Fleet. In some situations where we think it is warranted, we mention the updates in release notes. The best way of knowing what dependencies are required to use Fleet is to look at them directly  [in the repository](https://github.com/fleetdm/fleet/blob/main/package.json).

We use [Dependabot](https://github.com/dependabot) to create pull requests to update vulnerable dependencies. You can find these PRs by filtering on the [*Dependabot*](https://github.com/fleetdm/fleet/pulls?q=is%3Apr+author%3Aapp%2Fdependabot+) author in the repository.

We make sure the fixes to vulnerable dependencies are also performed according to our remediation timeline. We fix as many dependencies as possible in a single release.


## Trust report

We publish a trust report that includes automated checking of controls, answers to frequently asked
questions and more on [https://fleetdm.com/trust](https://fleetdm.com/trust)


## Securtiy audits

This section contains explanations of the latest external security audits performed on Fleet software.


### June 2024 penetration testing of Fleet 4.50.1

In June 2024, [Latacora](https://www.latacora.com/) performed an application penetration assessment of the application from Fleet. 

An application penetration test captures a point-in-time assessment of vulnerabilities, misconfigurations, and gaps in applications that could allow an attacker to compromise the security, availability, processing integrity, confidentiality, and privacy (SAPCP) of sensitive data and application resources. An application penetration test simulates the capabilities of a real adversary, but accelerates testing by using information provided by the target company.

Latacora identified a few medium and low severity risks, and Fleet is prioritizing and responding to those within SLAs. Once all action has been taken, a summary will be provided.

You can find the full report here: [2024-06-14-fleet-penetration-test.pdf](https://github.com/fleetdm/fleet/raw/main/docs/files/2024-06-14-fleet-penetration-test.pdf).

### June 2023 penetration testing of Fleet 4.32 

In June 2023, [Latacora](https://www.latacora.com/) performed an application penetration assessment of the application from Fleet. 

An application penetration test captures a point-in-time assessment of vulnerabilities, misconfigurations, and gaps in applications that could allow an attacker to compromise the security, availability, processing integrity, confidentiality, and privacy (SAPCP) of sensitive data and application resources. An application penetration test simulates the capabilities of a real adversary, but accelerates testing by using information provided by the target company.

Latacora identified a few issues, the most critical ones we have addressed in 4.33. These are described below.

You can find the full report here: [2023-06-09-fleet-penetration-test.pdf](https://github.com/fleetdm/fleet/raw/main/docs/files/2023-06-09-fleet-penetration-test.pdf).

### Findings


#### 1 - Stored cross-site scripting (XSS) in tooltip

| Type                | Latacora Severity |
| ------------------- | -------------- |
| Cross-site scripting| High risk      |

All tooltips using the "tipContent" tag are set using "dangerouslySetInnerHTML". This allows manipulation of the DOM without sanitization. If a user can control the content sent to this function, it can lead to a cross-site scripting vulnerability. 

This was resolved in version release [4.33.0](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.33.0) with [implementation of DOMPurify library](https://github.com/fleetdm/fleet/pull/12229) to remove dangerous dataset.


#### 2 - Broken authorization leads to observers able to add hosts

| Type                | Latacora Severity |
| ------------------- | -------------- |
| Authorization issue | High risk      |

Observers are not supposed to be able to add hosts to Fleet. Via specific endpoints, it becomes possible to retrieve the certificate chains and the secrets for all teams, and these are the information required to add a host. 

This was resolvedin version release [4.33.0](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.33.0) with [updating the observer permissions](https://github.com/fleetdm/fleet/pull/12216).


### April 2022 penetration testing of Fleet 4.12 

In April 2022, we worked with [Lares](https://www.lares.com/) to perform penetration testing on our Fleet instance, which was running 4.12 at the time. 

Lares identified a few issues, the most critical ones we have addressed in 4.13. Other less impactful items remain. These are described below.

As usual, we have made the full report (minus redacted details such as email addresses and tokens) available.

You can find the full report here: [2022-04-29-fleet-penetration-test.pdf](https://github.com/fleetdm/fleet/raw/main/docs/files/2022-04-29-fleet-penetration-test.pdf).


### Findings


#### 1 - Broken access control & 2 - Insecure direct object reference

| Type                | Lares Severity |
| ------------------- | -------------- |
| Authorization issue | High risk      |

This section contains a few different authorization issues, allowing team members to access APIs out of the scope of their teams. The most significant problem was that a team administrator was able to add themselves to other teams. 

This is resolved in 4.13, and an [advisory](https://github.com/fleetdm/fleet/security/advisories/GHSA-pr2g-j78h-84cr) has been published before this report was made public.
We are also planning to add [more testing](https://github.com/fleetdm/fleet/issues/5457) to catch potential future mistakes related to authorization.


#### 3 - CSV injection in export functionality

| Type      | Lares Severity |
| --------- | -------------- |
| Injection | Medium risk    |

It is possible to create or rename an existing team with a malicious name, which, once exported to CSV, could trigger code execution in Microsoft Excel. We assume there are other ways that inserting this type of data could have similar effects, including via osquery data. For this reason, we will evaluate the feasibility of [escaping CSV output](https://github.com/fleetdm/fleet/issues/5460).

Our current recommendation is to review CSV contents before opening in Excel or other programs that may execute commands.


#### 4 - Insecure storage of authentication tokens

| Type                   | Lares Severity |
| ---------------------- | -------------- |
| Authentication storage | Medium risk    |

This issue is not as straightforward as it may seem. While it is true that Fleet stores authentication tokens in local storage as opposed to cookies, we do not believe the security impact from that is significant. Local storage is immune to CSRF attacks, and cookie protection is not particularly strong. For these reasons, we are not planning to change this at this time, as the changes would bring minimal security improvement, if any, and change always carries the risk of creating new vulnerabilities.


#### 5 - No account lockout

| Type           | Lares Severity |
| -------------- | -------------- |
| Authentication | Medium risk    |

Account lockouts on Fleet are handled as a “leaky bucket” with 10 available slots. Once the bucket is full, a four second timeout must expire before another login attempt is allowed. We believe that any longer, including full account lockout, could bring user experience issues as well as denial of service issues without improving security, as brute-forcing passwords at a rate of one password per 4 seconds is very unlikely.

We have additionally added very prominent activity feed notifications of failed logins that make brute forcing attempts apparent to Fleet admins.


#### 6 - Session timeout - insufficient session expiration

| Type               | Lares Severity |
| ------------------ | -------------- |
| Session expiration | Medium risk    |

Fleet sessions are currently [configurable](https://fleetdm.com/docs/deploying/configuration#session-duration). However, the actual behavior, is different than the expected one. We [will switch](https://github.com/fleetdm/fleet/issues/5476) the behavior so the session timeout is based on the length of the session, not on how long it has been idle. The default will remain five days, which will result in users having to log in at least once a week, while the current behavior would allow someone to remain logged in forever. If you have any reason to want a shorter session duration, simply configure it to a lower value.


#### 7 - Weak passwords allowed

| Type           | Lares Severity |
| -------------- | -------------- |
| Weak passwords | Medium risk    |

The default password policy in Fleet requires passwords that are seven characters long. We have [increased this to 12](https://github.com/fleetdm/fleet/issues/5477) while leaving all other requirements the same. As per NIST [SP 800-63B](https://pages.nist.gov/800-63-3/sp800-63b.html), we believe password length is the most important requirement. If you have additional requirements for passwords, we highly recommend implementing them in your identity provider and setting up [SSO](https://fleetdm.com/docs/deploying/configuration#configuring-single-sign-on-sso).


#### 8 - User enumeration

| Type        | Lares Severity |
| ----------- | -------------- |
| Enumeration | Low risk       |

User enumeration by a logged-in user is not a critical issue. Still, when done by a user with minimal privileges (such as a team observer), it is a leak of information, and might be a problem depending on how you use teams. For this reason, only team administrators are able to enumerate users as of Fleet 4.31.0.


#### 9 - Information disclosure via default content

| Type                   | Lares Severity |
| ---------------------- | -------------- |
| Information disclosure | Informational  |

This finding has two distinct issues. 

The first one is the /metrics endpoint, which contains a lot of information that could potentially be leveraged for attacks. We had identified this issue previously, and it was [fixed in 4.13](https://github.com/fleetdm/fleet/issues/2322) by adding authentication in front of it.

The second one is /version. While it provides some minimal information, such as the version of Fleet and go that is used, it is information similar to a TCP banner on a typical network service. For this reason, we are leaving this endpoint available. 

If this endpoint is a concern in your Fleet environment, consider that the information it contains could be gleaned from the HTML and JavaScript delivered on the main page. If you still would like to block it, we recommend using an application load balancer.


#### The GitHub issues that relate to this test are:

[Security advisory fixed in Fleet 4.13](https://github.com/fleetdm/fleet/security/advisories/GHSA-pr2g-j78h-84cr)

[Add manual and automated test cases for authorization #5457](https://github.com/fleetdm/fleet/issues/5457)

[Evaluate current CSV escaping and feasibility of adding if missing #5460](https://github.com/fleetdm/fleet/issues/5460)

[Set session duration to total session length #5476](https://github.com/fleetdm/fleet/issues/5476)

[Increase default minimum password length to 12 #5477](https://github.com/fleetdm/fleet/issues/5477)

[Add basic auth to /metrics endpoint #2322](https://github.com/fleetdm/fleet/issues/2322)

[Ensure only team admins can list other users #5657](https://github.com/fleetdm/fleet/issues/5657)


### August 2021 security of Orbit auto-updater

Back in 2021, when Orbit was still new, alpha, and likely not used by anyone but us here at Fleet, we contracted Trail of Bits (ToB) to have them review the security of the auto-updater portion of it.

For more context around why we did this, please see this [post](https://blog.fleetdm.com/security-testing-at-fleet-orbit-auto-updater-audit-7e3e99152a25) on the Fleet blog.

You can find the full report here: [2021-04-26-orbit-auto-updater-assessment.pdf](https://github.com/fleetdm/fleet/raw/3ad02fc697e196b5628bc07e807fbc2db3086393/docs/files/2021-04-26-orbit-auto-updater-assessment.pdf)


### Findings


#### 1 - Unhandled deferred file close operations

| Type               | ToB Severity |
| ------------------ | ------------ |
| Undefined Behavior | Low          |

This issue was addressed in PR [1679](https://github.com/fleetdm/fleet/issues/1679) and merged on August 17, 2021.

The fix is an improvement to cleanliness, and though the odds of exploitation were very low, there is no downside to improving it. 

This finding did not impact the auto-update mechanism but did impact Orbit installations.


#### 2 - Files and directories may pre-exist with too broad permissions

| Type            | ToB Severity |
| --------------- | ------------ |
| Data Validation | High         |

This issue was addressed in PR [1566](https://github.com/fleetdm/fleet/pull/1566) and merged on August 11, 2021

Packaging files with permissions that are too broad can be hazardous. We fixed this in August 2021. We also recently added a [configuration](https://github.com/fleetdm/fleet/blob/f32c1668ae3bc57d33c31eb30eb1959f65963a0a/.golangci.yml#L29) to our [linters](https://en.wikipedia.org/wiki/Lint_(software)) and static analysis tools to throw an error any time permissions on a file are above 0644 to help avoid future similar issues. We rarely change these permissions. When we do, they will have to be carefully code-reviewed no matter what, so we have also enforced code reviews on the Fleet repository.

This finding did not impact the auto-update mechanism but did impact Orbit installations.


#### 3 - Possible nil pointer dereference 

| Type            | ToB Severity  |
| --------------- | ------------- |
| Data Validation | Informational |

We did not do anything specific for this informational recommendation. However, we did deploy multiple SAST tools, such as [gosec](https://github.com/securego/gosec), mentioned in the previous issue, and [CodeQL](https://codeql.github.com/), to catch these issues in the development process.

This finding did not impact the auto-update mechanism but did impact Orbit installations.


#### 4 - Forcing empty passphrase for keys encryption

| Type         | ToB Severity |
| ------------ | ------------ |
| Cryptography | Medium       |

This issue was addressed in PR [1538](https://github.com/fleetdm/fleet/pull/1538) and merged on August 9, 2021.

We now ensure that keys do not have empty passphrases to prevent accidents.


#### 5 - Signature verification in fleetctl commands

| Type            | ToB Severity |
| --------------- | ------------ |
| Data Validation | High         |

Our threat model for the Fleet updater does not include the TUF repository itself being malicious. We currently assume that if the TUF repository is compromised and that the resulting package could be malicious. For this reason, we keep the local repository used with TUF offline (except for the version we publish and never re-sign) with the relevant keys, and why we add target files directly rather than adding entire directories to mitigate this risk. 

We consider the security of the TUF repository itself out of the threat model of the Orbit auto-updater at the moment, similarly to how we consider the GitHub repository out of scope. We understand that if the repository was compromised, an attacker could get malicious code to be signed, and so we have controls at the GitHub level to prevent this from happening. For TUF, currently, our mitigation is to keep the files offline.

We plan to document our update process, including the signature steps, and improve them to reduce risk as much as possible. 


#### 6 - Redundant online keys in documentation

| Type            | ToB Severity |
| --------------- | ------------ |
| Access Controls | Medium       |

Using the right key in the right place and only in the right place is critical to the security of the update process. 

This issue was addressed in PR [1678](https://github.com/fleetdm/fleet/pull/1678) and merged on August 15, 2021. 


#### 7 - Lack of alerting mechanism 

| Type          | ToB Severity |
| ------------- | ------------ |
| Configuration | Medium       |

We will make future improvements, always getting better at detecting potential attacks, including the infrastructure and processes used for the auto-updater.


#### 8 - Key rotation methodology is not documented

| Type         | ToB Severity |
| ------------ | ------------ |
| Cryptography | Medium       |

This issue was addressed in PR [2831](https://github.com/fleetdm/fleet/pull/2831) and merged on November 15, 2021


#### 9 - Threshold and redundant keys 

| Type         | ToB Severity  |
| ------------ | ------------- |
| Cryptography | Informational |


We plan to document our update process, including the signature steps, and improve them to reduce risk as much as possible. We will consider multiple role keys and thresholds, so specific actions require a quorum, so the leak of a single key is less critical.


#### 10 - Database compaction function could be called more times than expected

| Type               | ToB Severity  |
| ------------------ | ------------- |
| Undefined Behavior | Informational |

This database was not part of the update system, and we [deleted](http://hrwiki.org/wiki/DELETED) it.


#### 11 - All Windows users have read access to Fleet server secret

| Type            | ToB Severity |
| --------------- | ------------ |
| Access Controls | High         |

While this did not impact the security of the update process, it did affect the security of the Fleet enrollment secrets if used on a system where non-administrator accounts were in use. 

This issue was addressed in PR [21](https://github.com/fleetdm/orbit/pull/21) of the old Orbit repository and merged on April 26, 2021. As mentioned in finding #2, we also deployed tools to detect weak permissions on files.


#### 12 - Insufficient documentation of SDDL permissions

| Type                 | ToB Severity |
| -------------------- | ------------ |
| Auditing and Logging | Low          |

While SDDL strings are somewhat cryptic, we can decode them with [PowerShell](https://docs.microsoft.com/en-us/powershell/module/microsoft.powershell.utility/convertfrom-sddlstring?view=powershell-7.2). We obtained SDDL strings from a clean Windows installation with a new osquery installation. We then ensure that users do not have access to secret.txt, to resolve finding #11. 

We have documented the actual permissions expected on April 26, 2021, as you can see in this [commit](https://github.com/fleetdm/fleet/commit/79e82ebcb653b435c6753c68a42cadaa083115f7) in the same PR [21](https://github.com/fleetdm/orbit/pull/21) of the old Orbit repository as for #11.


### Summary

ToB identified a few issues, and we addressed most of them. Most of these impacted the security of the resulting agent installation, such as permission-related issues.

Our goal with this audit was to ensure that our auto-updater mechanism, built with
[TUF](https://theupdateframework.io/), was sound. We believe it is, and we are planning future
improvements to make it more robust and resilient to compromise.




<meta name="maintainedBy" value="hollidayn">
<meta name="title" value="Security">

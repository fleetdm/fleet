# Security

## Account recovery process

As an all-remote company, we do not have the luxury of seeing each other or are able to ask for help in person. Instead, we require live video confirmation of someone's identity before performing recovery and this applies to all Fleet company accounts, from internal systems to SaaS accounts.

| Participant | Role                                                                                                                                                 |
| ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| Requester   | Requests recovery for their own account                                                                |
| Recoverer   | Person with access to perform the recovery who monitors *#help-login*                                                                                                                               |
| Identifier  | Person that visually identifies the requester in a video call. The identifier can be the recoverer or a person the recoverer can recognize visually |


Here are the steps we take for the recovery process:


1. If the requester still has access to Slack, they ask for help in *#help-login*. If they do not have access to Slack, they can contact their manager or a teammate over the phone via voice or texting, and they will post in *#help-login* for the requester.
2. A recoverer acknowledges the request in #help-login using the "eyes" emoji 👀.
3. The recoverer identifies the requester through a live video call.
* If the recoverer does not know the requester well enough to positively identify them visually, the
  recoverer can ask a colleague whom they recognize to act as the identifier. **All three must be
  live on a video call at the same time.**
*  For example, if the recoverer does not recognize Guillaume but can recognize Zach, they should ask Zach to identify Guillaume. Using the requester's manager or a direct teammate is recommended, as it increases the chances they see each other on video frequently.
4. If the recoverer recognizes the requester, or has the identity confirmed by the person acting as
   the identifier, they can perform the recovery and update the thread in *#help-login*.
* If the recoverer is not 100% satisfied with identification, they do **NOT** proceed, and post to #g-security to engage the security team immediately.


## How we protect end-user devices

At Fleet, we believe that a good user experience empowers contributors.

We follow the guiding principles below to secure our company-owned devices.

* Our devices should give contributors the freedom to work from anywhere.
* To allow maximum freedom in where and how we work, we assume that "Safe" networks do not exist. Contributors should be able to work on a coffee shop's Wi-Fi as if it were their home or work network.
* To limit the impact on user experience, we do not dictate security configurations unless the security benefit is significant (only if it dramatically reduces risk for the company, customers, or open source users).
* By using techniques such as Two-Factor Authentication (2FA), code reviews, and more, we can further empower contributors to work comfortably from any location - on any network.


### macOS devices
> *Find more information about the process of implementing security on the Fleet blog. The first [Tales from Fleet security: securing the startup](https://blog.fleetdm.com/tales-from-fleet-security-securing-the-startup-448ea590ea3a) article covers the process of securing our laptops.*

We use configuration profiles to standardize security settings for our Mac devices. We use [CIS Benchmark for macOS 12](https://www.cisecurity.org/benchmark/apple_os) as our configuration baseline and adapt it to:
* suit a remote team.
* balance the need for productivity and security.
* limit the impact on the daily use of our devices.

> *Note: Details of your Mac’s configuration profile can be viewed anytime from the **Profiles** app under **System Preferences**.*



Our policy applies to Fleet-owned laptops purchased via Apple's DEP (Device Enrollment Program) which will retroactively be applied to every company-owned Mac consists of the below. 

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

macOS includes [malware protection tools](https://support.apple.com/en-ca/guide/security/sec469d47bd8/web) such as *Xprotect*. This is an antivirus technology based on [YARA](https://github.com/VirusTotal/yara), and MRT (Malware Removal Tool), a tool built by Apple to remove common malware from systems that are infected.
By enabling these settings, we:

* Ensure the operating system is kept up to date.
* Ensure XProtect and MRT are as up-to-date as possible.
* Ensure that Safari is kept up to date. 

This improves the resilience of our Mac fleet. 

**User experience impacts**

* Updates are required, which can be disruptive. For this reason, we allow the user to **postpone the installation 5 times**.
* Critical security updates are automatically downloaded, which could result in bandwidth use on slow or expensive links. For this reason, we limit automatic downloads to critical security updates only, while feature updates, which are typically larger, are downloaded at the time of installation selected by the user.
* Enforced updates **do not** include significant macOS releases (e.g., 11➡️12). Those updates are tracked and enforced separately, as the impact can be more significant. We require installing the latest macOS version within 3 months of release or when known vulnerabilities remain unpatched on the older version.

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
| 5.2.2 | Ensure password minimum length is configured (our minimum: 8 characters)                                             |
| 5.2.3 | Ensure complex password must contain alphabetic characters is configured                 |
| 5.8   | Ensure a password is required to wake the computer from sleep or screen saver is enabled |

**Why?**

This category of settings is special because there are more settings that we do *not* configure than those we do.

We follow the CIS benchmark where it makes sense and in this case, take guidance from [NIST SP800-63B - Digital Identity Guidelines](https://pages.nist.gov/800-63-3/sp800-63b.html), especially [Appendix A -Strength of Memorized Secrets](https://pages.nist.gov/800-63-3/sp800-63b.html#appA).

* We do NOT enforce special complexity beyond requiring letters to be in the password.

Length is the most important factor when determining a secure password; while enforcing password expiration, special characters and other restrictive patterns are not as effective as previously believed and provide little benefit at the cost of hurting the user experience.

* We do NOT enforce exceptionally long passwords. 

As we use recent Macs with T2 chips or Apple Silicon, brute-force attacks against the hardware are [mitigated](https://www.apple.com/mideast/mac/docs/Apple_T2_Security_Chip_Overview.pdf).

* We DO require passwords to be a minimum of 8 characters long with letters.

Since we can't eliminate the risk of passwords being cracked remotely, we require passwords to be a minimum of 8 characters long with letters, a length reasonably hard to crack over the network, and the minimum recommendation by SP800-63B.


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

* Using FileVault protects the data on our laptops, including confidential data and session material (browser cookies), SSH keys, and more. Using FileVault ensures a lost laptop is a minor inconvenience and not an incident. We escrow the keys to be sure we can recover the data if needed.
* [Gatekeeper](https://support.apple.com/en-ca/HT202491) is a macOS feature that ensures users can safely open software on their Mac. With Gatekeeper enabled, users may execute only trustworthy apps (signed by the software developer and/or checked for malicious software by Apple). This is a useful first line of defense to have.
* Using the firewall will ensure that we limit the exposure to our devices, while stealth mode makes them more challeging to discover. 
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

* Fleet contributors are free to work from wherever they choose. If a laptop is lost or forgotten, automatic login exposes sensitive company data and poses a critical security risk. 
* Password hints can sometimes be easier to guess than the password itself. Since we support contributors remotely via MDM and do not require users to change passwords frequently, we eliminate the need for passwords hints and their associated risk.
* Since company laptops are issued primarily for work and tied to a single contributor's identity, guest accounts are not permitted.
* Automatic login would defeat the purpose of even requiring passwords to unlock computers.

**User experience impacts**

* Laptops lock after 20 minutes of inactivity. To voluntarily pause this, a [hot corner](https://support.apple.com/en-mo/guide/mac-help/mchlp3000/mac) can be configured to disable the screen saver. This is useful if you are, for example, watching an online meeting without moving the mouse and want to be sure the laptop will not lock.
* Forgotten passwords can be fixed via MDM instead of relying on potentially dangerous hints.
* Guest accounts are not available.

#### iCloud
We do not apply ultra restrictive Data Loss Prevention style policies to our devices. Instead, by using our company Google Drive, we ensure that the most critical company data never reaches our laptops, so it can remain secure while our laptops can remain productive.


| #       | Setting                                                   |
| ------- | --------------------------------------------------------- |
| 2.6.1.4 | Ensure iCloud Drive Documents and Desktop sync is disabled |

**Why?**
* We do not use managed Apple IDs, and allow contributors to use their own iCloud accounts. We disable iCloud Documents and Desktop sync to avoid accidental copying of data to iCloud, but we do allow iCloud drive.

**User experience impact**

* iCloud remains permitted, but the Desktop and Documents folders will not be synchronized. Ensure you put your documents in our Google Drive so you do not lose them if your laptop has an issue.

#### Miscellaneous security settings

| #     | Setting                                                      |
| ----- | ------------------------------------------------------------ |
| 2.5.6 | Ensure limit ad tracking is enabled                          |
| 2.10  | Ensure secure keyboard entry Terminal.app is enabled         |
| 5.1.4 | Ensure library validation is enabled                         |
| 6.3   | Ensure automatic opening of safe files in Safari is disabled |

**Why?**

* Limiting ad tracking has privacy benefits and no downside.
* Protecting keyboard entry into Terminal.app could prevent malicious applications or non-malicious but inappropriate applications from receiving passwords.
* Library validation ensures that an attacker can't trick applications into loading a software library in a different location, leaving it open to abuse.
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

* Some misconfigured "captive portals," typically used in hotels and airports, might be unusable with DoH due to how they are configured. This can be worked around by using the hotspot on your phone, and if you have to use this network for an extended period of time, there are usually workarounds to preform to connect to them. Navigating to http://1.1.1.1 often resolves the issue.
* If you are trying to reach a site and believe it is being blocked accidentally, please submit it to Cloudflare. This should be extremely rare. If it is not, please let the security team know.
* If your ISP's DNS service goes down, you'll be able to continue working. 😎

*Note: If you are from another organization, reading this to help create your own configuration, remember implementing DoH in an office environment where other network controls are in place has other downsides than it would for a remote company. In those cases, **disabling** DoH makes more sense, so network controls can retain visibility. Please evaluate your situation before implementing any of our recommendations at your organization, especially DoH.*

#### Deploy osquery
| #  | Setting                |
| -- | ---------------------- |
| NA | Deploy [osquery](https://osquery.io/) pointed to our dogfood instance |

***Why?***

We use osquery and Fleet to monitor our own devices. This is used for vulnerability detection, security posture tracking, and for incident response when necessary.


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

* Chrome needs to be restarted within 48 hours of patch installation. The automatic restart happens after 19:00 and before 6:00 if the computer is running and tabs are restored (except for incognito tabs).
* Ads considered intrusive are blocked.
* uBlock Origin is enabled by default, and is 100% configurable, improving security and the performance of browsing.
* Endpoint Verification is used to make access decisions based on the security posture of the device. For example, an outdated Mac could be prevented access to Google Drive.

### Personal mobile devices

The use of personal devices is allowed for some applications, so long as the iOS or Android device's OS
is kept up to date.

## Hardware security keys

If you do not already have a pair of hardware security keys, order [YubiKey 5C NFC security
keys](https://www.yubico.com/ca/product/yubikey-5c-nfc-pack-of-2/) with your company card, or ask
for help in [#help-login](https://fleetdm.com/handbook/security#slack-channels) to get you one if you do not have a company card.

### Are they YubiKeys or security keys?

We use YubiKeys, a brand of hardware security keys that support the FIDO U2F protocol. You can use
both terms interchangeably at Fleet. We use YubiKeys because they support more authentication protocols than regular
security keys.

### Who has to use security keys and why?

Security keys are **strongly recommended** for everyone and **required** for team members with elevated privilege access. 

Because they are the only type of Two-Factor Authentication (2FA) that protects credentials from
phishing, we will make them **mandatory for everyone** soon. 

See the [Google Workspace security
section](https://fleetdm.com/handbook/security#google-workspace-security-authentication) for more
information on the security of different types of 2FA.

### Goals

Our goals with security keys are to:

1. eliminate the risk of credential phishing.
2. maintain the best user experience possible.
3. ensure team members can access systems as needed, and that recovery procedures exist in case of a lost key.
4. ensure recovery mechanisms are safe to prevent attackers from bypassing 2FA completely.

### Setting up security keys on Google

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


### Optional: getting rid of keyboard warnings

1. Install YubiKey manager. You can do this from the **Managed Software Center** on managed Macs.
   On other platforms, download it [from the official
   website](https://www.yubico.com/support/download/yubikey-manager/#h-downloads).
2. Open the YubiKey manager with one of your keys connected.
3. Go to the **Interfaces** tab.
4. Uncheck the **OTP** checkboxes under **USB** and click *Save Interfaces*.
5. Unplug your key and connect your 2nd one to repeat the process.


### Optional: setting up security keys on GitHub

1. Configure your two security keys to [access
   GitHub](https://github.com/settings/two_factor_authentication/configure).
2. If you are using a Mac, feel free to add it as a security key on GitHub. This brings most of the
   advantages of the hardware security key but allows you to log in by simply touching Touch ID as
   your second factor.

### FAQ

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

**Answer**: Post in the `#help-login` channel, or if you are locked out of Slack, contact your
manager. You will be provided a way to log back in and make your phone your security key until you
receive new ones.

6. Can I use security keys to log in from any device?

**Answer**: The keys we use, YubiKeys 5C NFC, work over USB-C as well as NFC. They can be used on
Mac/PC, Android and iPhone and iPad Pro with USB-C port. If some application or device does
not support it, you can always browse to [g.co/sc](https://g.co/sc) from a device that supports
security keys to generate a temporary code for the device that does not.

7. Will I need my YubiKey every time I want to check my email?

**Answer**: No. Using them does not make sessions shorter. For example, if using the GMail app on
mobile, you'd need the keys to set up the app only.

## GitHub security
Since Fleet makes open source software, we need to host and collaborate on code. We do this using GitHub.

This section covers our GitHub configuration. Like everything we do, we aim for the right level of security and productivity.

Because our code is open source, we are much more concerned about the integrity of the code than its confidentiality.
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
| Pages creation            | None    | We do not use GitHub pages, so we disable them to ensure people use our actual website or handbook, which are also in GitHub. |

#### Admin repository permissions

| Admin privileges feature                                                   | Member privileges feature | Note                                                                                                                                                                          |
| -------------------------------------------------------------------------- | ------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Allow members to change repository visibilities for this organization      | 🚫                   | Most of our repos are public, but for the few that are private we want to require org admin privileges to make them public                                                    |
| Allow members to delete or transfer repositories for this organization     | 🚫                   | We want to require org admin privileges to be able to delete or transfer any repository.                                                                                       |
| Allow repository administrators to delete issues for this organization     | 🚫                   | We want to require org admin privileges to be able to delete issues, which is something that is very rarely needed but could be, for example, if we received GitHub issue spam. |
| Allow members to see the comment author's profile name in private repositories | 🚫                   | We barely use private repositories and have no need for this.                                                                                                                |
| Allow users with read access to create discussions                         | 🚫                   | We do not currently use discussions and want people to use issues as much as possible.                                                                                       |
| Allow members to create teams                                              | 🚫                   | We automate the management of GitHub teams with the [GitHub Terraform provider](https://github.com/integrations/terraform-provider-github).                            |

### Team discussions
We do not use team discussions and therefore have disabled them. This is simply to avoid discussions
located in too many places and not security-related.

### Repository security

#### Branch protection
Branch protection is one of the most important settings to configure and the main reason we should not have members with administrative privileges on the repositories.

Located in the Branches section of repository settings, we create a rule for **main** that applies:

| Setting                                                          | Value | Note                                                                                                                  |
| ---------------------------------------------------------------- | ----- | --------------------------------------------------------------------------------------------------------------------- |
| Require a pull request before merging                            | ✅     | We enforce code reviews, which require PRs.                                                                           |
| Require approvals                                                | 1️⃣   | We require approval from one person in the team.                                                                      |
| Dismiss stale pull request approvals when new commits are pushed | ✅     | Without this, someone could get approval for a small, very nice PR and then change everything about it!                   |
| Require review from Code Owners                                  | 🗓     | We are working towards enabling this as our team grows and allows for more flexibility                                    |
| Restrict who can dismiss pull request reviews                    | 🚫     | As we are a team working in multiple timezones, we want to allow dismissing reviews and getting another one.          |
| Allow specified actors to bypass required pull requests          | 🚫     | We do not want anyone pushing directly to main.                                                                       |
| Require status checks to pass before merging                     | ✅     | Because of our [monorepo](https://en.wikipedia.org/wiki/Monorepo#:~:text=In%20version%20control%20systems%2C%20a,as%20a%20'shared%20codebase'.), it is hard to pick many checks that work for all types of PRs, but we still enable this.     |
| Require conversation resolution before merging                   | 🚫     | Reviewers should not approve a pull request if they do not think it's ready for merging.                                             |
| Require signed commits                                           | 🗓     | We are working towards enabling this, manually keeping track of unverified commits.                                   |
| Require linear history                                           | 🚫     | We do not currently use or enforce practices to generate a linear history.                                                                                                                      |
| Include administrators                                           | ✅     | We want these rules to apply to *everyone*.                                                                           |
| Restrict who can push to matching branches                       | 🚫     | Anyone in our organization should be able to merge PRs that get reviewed, and nobody should be able to push directly. |
| Allow force pushes                                               | 🚫     | We do not have a need this, so we do not allow it.                                                             |
| Allow deletions                                                  | 🚫     | We do not want ANYONE to be able to delete the *main* branch.                                                         |

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
mentions GitHub actions. Security updates to all other dependencies are performed by Dependabot automatically, even though we do not configure all package managers explicitly in the configuration file, as is specified in the repository configuration. As GitHub actions have no impact on the Fleet software itself, we are
simply more aggressive in updating actions even if the update does not resolve a vulnerability.

### Actions configuration
We configure GitHub Actions to have *Read repository contents permission* by default. This is
located in *organization/settings/actions*. As our code is open source, we allow all GitHub actions
but limit their default privileges, so they do not create any additional risk. Additional permissions
needed can be configured in the YAML file for each workflow.

We pin actions to specific versions using a complete hash.

### Automation
We manage our GitHub configuration, creation of repositories, and team memberships manually. In the
future, we will consider automating most of it using the [Terraform
provider](https://github.com/integrations/terraform-provider-github) for GitHub. Our strategy for
this will be similar to what is described in [this blog post](https://oops.computer/posts/github_automation/).

## Google Workspace security
Google Workspace is our collaboration tool and the source of truth for our user identities.
A Google Workspace account has access to email, calendar, files, and external applications integrated with Google Authentication or SAML.
At the same time, third-party applications installed by users can access the same data.

To reduce the risk of malicious or vulnerable apps being used to steal data, we configure Google Workspace beyond the default settings. Our current configuration balances security and productivity and is a starting point for any organization looking to improve the security of Google Workspace.

As Google frequently adds new features, feel free to submit a PR to edit this file if you discover a new one that we should use!

### Authentication
We cannot overstate the importance of securing authentication, especially in a platform that includes email and is used as a directory to log in to multiple applications.

#### 2-Step Verification 
Google's name for Two-Factor Authentication (2FA) or Multi-Factor Authentication (MFA) is 2-Step Verification (2-SV). No matter what we call it, it is the most critical feature to protect user accounts on Google Workspace or any other system.

| 2FA Authentication methods from least to most secure                              | Weaknesses                                                                                                |
| ----------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------- |
| No 2FA                                                                        | Credential theft is easy, and passwords are often leaked or easy to guess.                                |
| SMS/Phone-based 2FA                                                           | Puts trust in the phone number itself, which attackers can hijack by [social engineering phone companies](https://www.vice.com/en/topic/sim-hijacking).      |
| Time-based one-time password (TOTP - Google Authenticator type 6 digit codes) | Phishable as long as the attacker uses it within its short lifetime by intercepting the login form. |
| App-based push notifications                                                  | Harder to phish than TOTP, but by sending a lot of prompts to a phone, a user might accidentally accept a nefarious notification.       |
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

##### Hardware security keys

We strongly recommend using hardware security keys. 

Fleet configures privileged user accounts with a policy that enforces the use of hardware security
keys. This prevents credential theft better than other methods of 2FA/2-SV. See [hardware security
keys](https://fleetdm.com/handbook/security#hardware-security-keys) for information about the model we use, why, and how to set
them up.


#### Passwords
As we enforce the use of 2-SV, passwords are less critical to the security of our accounts. We base our settings on [NIST 800-63B](https://pages.nist.gov/800-63-3/sp800-63b.html).

Enforcing 2FA is a much more valuable control than enforcing the expiration of passwords, which usually results in users changing only a small portion of the password and following predictable patterns.

We apply the following settings to *Security/Password management* to all users as the minimum baseline.


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

First, we ensure we have a handful of administrators. Then, by not requiring password expiration, the number of issues related to passwords is reduced. Lastly, we can support locked-out users manually as the volume of issues is minimal.

#### Less secure apps
Less secure apps use legacy protocols that do not support secure authentication methods. We disable them, and as they are becoming rare, we have not noticed any issues from this setting.

We apply the following settings to *Security/Less Secure Apps* to all users as the minimum baseline.

| Setting name                                                                                            | Value                                            |
| ------------------------------------------------------------------------------------------------------- | ------------------------------------------------ |
| Control user access to apps that use less secure sign-in technology and make accounts more vulnerable.  | Disable access to less secure apps (Recommended) |

#### API access
Google Workspace makes it easy for users to add tools to their workflows while having these tools authenticate to their Google applications and data via OAuth. We mark all Google services as *restricted* but do allow the use of OAuth for simple authentication and the use of less dangerous privileges on Gmail and Drive. We then approve applications that require more privileges on a case-by-case basis.

This level of security allows users to authenticate to web applications with their Google accounts. This exposes little information beyond what they would provide in a form to create an account, and it protects confidential data while keeping everything managed.

>To get an application added to Fleet's Google Workspace security configuration, create an issue assigned to the security team in [this repository](https://github.com/fleetdm/confidential/issues).

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
| 2-step Verification Scratch Codes Generated | Admin audit log, filtered by event  | Scratch codes can be used to bypass 2-SV. An attacker with elevated privileges could leverage this to log in as a user.                           | Alert Center + Email |
| Change Allowed 2-step Verification Methods  | Admin audit log, filtered by event  | We want to detect accidental or malicious downgrades of 2-SV configuration.                                                                                              | Alert Center + Email |
| Change 2-Step Verification Start Date       | Admin audit log, filtered by event  | We want to detect accidental or malicious "downgrades" of 2-SV configuration.                                                                                              | Alert Center + Email |
| Alert Deletion                              | Admin audit log, filtered by event  | For alerts to be a reliable control, we need to alert on alerts being disabled or changed.                                                                                | Alert Center + Email |
| Alert Criteria Change                       | Admin audit log, filtered by event  | For alerts to be a reliable control, we need to alert on alerts being disabled or changed.                                                                                | Alert Center + Email |
| Alert Receivers Change                      | Admin audit log, filtered by event  | For alerts to be a reliable control, we need to alert on alerts being disabled or changed.                                                                                | Alert Center + Email |
| Dangerous download warning                  | Chrome audit log, filtered by event | As we roll out more Chrome security features we want to track the things getting blocked to evaluate the usefulness of the feature and potential false positives. | Alert Center         |
| Malware transfer                            | Chrome audit log, filtered by event | As we roll out more Chrome security features we want to track the things getting blocked to evaluate the usefulness of the feature and potential false positives. | Alert Center         |
| Password reuse                              | Chrome audit log, filtered by event | As we roll out more Chrome security features we want to track the things getting blocked to evaluate the usefulness of the feature and potential false positives | Alert Center         |


### Gmail

#### Email authentication
Email authentication makes it harder for other senders to pretend to be from Fleet. This improves trust in emails from fleetdm.com and makes it more difficult for anyone attempting to impersonate Fleet.

We authenticate email with [DKIM](https://support.google.com/a/answer/174124?product_name=UnuFlow&hl=en&visit_id=637806265550953415-394435698&rd=1&src=supportwidget0&hl=en) and have a [DMARC](https://support.google.com/a/answer/2466580) policy to define how our outgoing email should be defined.

The DKIM configuration under *Apps/Google Workspace/Settings for Gmail/Authenticate Email* simply consists of generating the key, publishing it to DNS, then enabling the feature 48 hours later.

[DMARC](https://support.google.com/a/answer/2466580) is configured separately at the DNS level once DKIM is enforced.

#### Email security

Google Workspace includes multiple options in *Apps/Google Workspace/Settings for Gmail/Safety* related to how inbound email is handled.

As email is one of the main vectors used by attackers, we ensure we protect it as much as possible. Attachments are frequently used to send malware. We apply the following settings to block common tactics.

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

We enable *Apply future recommended settings automatically* to ensure we are secure by default. We would prefer to adjust this after seeing emails quarantined accidentally rather than missing out on new security features for email security.

#### End-user access

We recommend using the Gmail web interface on computers and the Gmail app on mobile devices. The user interface on the official applications includes security information that is not visible in standard mail clients (e.g., Mail on macOS). We do allow a few of them at the moment for specific workflows. 

| Category                         | Setting name                                                                                                                                      | Value                                                                                                                                                                                                                        | Note                                                                                                                                                                                                                                                  |
| -------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| POP and IMAP access              | Enable IMAP access for all users                                                                                                                  | Restrict which mail clients users can use (OAuth mail clients only)                                                                                                                                                          |                                                                                                                                                                                                                                                       |
|                                  | Clients                                                                                                                                           | (450232826690-0rm6bs9d2fps9tifvk2oodh3tasd7vl7.apps.googleusercontent.com, 946018238758-bi6ni53dfoddlgn97pk3b8i7nphige40.apps.googleusercontent.com, 406964657835-aq8lmia8j95dhl1a2bvharmfk3t1hgqj.apps.googleusercontent.com) | Those are the iOS, macOS built-in clients as well as Thunderbird. We plan to eventually only allow iOS,\ to limit the data cached on Macs and PCs.                                                                                         |
|                                  | Enable POP access for all users                                                                                                                   | Disabled                                                                                                                                                                                                                     |                                                                                                                                                                                                                                                       |
| Google Workspace Sync            | Enable Google Workspace Sync for Microsoft Outlook for my users                                                                                   | Disabled                                                                                                                                                                                                                     |                                                                                                                                                                                                                                                       |
| Automatic forwarding             | Allow users to automatically forward incoming email to another address                                                                            | Enabled                                                                                                                                                                                                                      | We will eventually disable this in favor of custom routing rules for domains where we want to allow forwarding. There is no mechanism for allow-listing destination domains, so we rely on alerts when new forwarding rules are added. |
| Allow per-user outbound gateways | Allow users to send mail through an external SMTP server when configuring a "from" address hosted outside your email domain                       | Disabled                                                                                                                                                                                                                     |                                                                                                                                                                                                                                                       |
| Warn for external recipients     | Highlight any external recipients in a conversation. Warn users before they reply to email messages with external recipients who aren't in their contacts. | Enabled                                                                                                                                                                                                                      |                                                                                                                                                                                                                                                       |


### Drive and Docs

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
| Link sharing default      | When users in Fleet Device Management create items, the default link sharing access will be:                                                                          | Off                                         | We want the owners of new files to make a conscious decision around sharing, and to be secure by default                                                                                                                                                                                                                                                                                     |
| Security update for files | Security update                                                                                                                                                       | Apply security update to all affected files |                                                                                                                                                                                                                                                                                                                                                                                              |
| Security update for files | Allow users to remove/apply the security update for files they own or manage                                                                                          | Enabled                                     | We have very few files impacted by [updates to link sharing](https://support.google.com/a/answer/10685032?amp;visit_id=637807141073031168-526258799&amp;rd=1&product_name=UnuFlow&p=update_drives&visit_id=637807141073031168-526258799&rd=2&src=supportwidget0). For some files meant to be public, we want users to be able to revert to the old URL that is more easily guessed.  |

#### Features and applications

| Category                             | Setting name                                                             | Value                                                                                                                           | Note                                                                                                                   |
| ------------------------------------ | ------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------- |
| Offline                              | Control offline access using device policies                             | Enabled                                                                                                                         |                                                                                                                        |
| Smart Compose                        | Allow users to see Smart Compose suggestions                             | Enabled                                                                                                                         |                                                                                                                        |
| Google Drive for desktop             | Allow Google Drive for desktop in your organization                      | Off                                                                                                                             | To limit the amount of data stored on computers, we currently do not allow local sync. We may enable it in the future  |
| Drive                                | Drive                                                                    | Do not allow Backup and Sync in your organization                                                                               |                                                                                                                        |
| Drive SDK                            | Allow users to access Google Drive with the Drive SDK API                | Enabled                                                                                                                         | The applications trusted for access to Drive are controlled but require this to work.                                 |
| Add-Ons                              | Allow users to install Google Docs add-ons from add-ons store            | Enabled                                                                                                                         | The applications trusted for access to Drive are controlled but require this to work.                                 |
| Surface suggestions in Google Chrome | Surface suggestions in Google Chrome                                     | Allow Google Drive file suggestions for signed-in users whenever a new search is performed or a new tab is opened (recommended) |                                                                                                                        |
| Creating new files on Drive          | Allow users to create and upload any file                                | On                                                                                                                              |                                                                                                                        |
| Creating new files on Drive          | Allow users to create new Docs, Sheets, Slides, Drawings and Forms files | On                                                                                                                              |                                                                                                                        |

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

We ensure the fixes to vulnerable dependencies are also performed according to our remediation timeline. We fix as many dependencies as possible in a single release.

## Trust report

We publish a trust report that includes automated checking of controls, answers to frequently asked
questions and more on [https://fleetdm.com/trust](https://fleetdm.com/trust)

## Rituals

The following table lists the Security group's rituals, frequency, and Directly Responsible Individual (DRI).

| Ritual                       | Frequency                | Description                                         | DRI               |
|:-----------------------------|:-----------------------------|:----------------------------------------------------|-------------------|
| Security notifications check | Daily | Check Slack, Google, Vanta, and Fleet dogfood for security-related notifications. | Guillaume Ross |
| GitHub check | Weekly | Check GitHub security issues for anything pending analysis or prioritization. | Guillaume Ross |
| Iteration planning | Every three weeks | Prioritize issues in the GitHub security project for the next iteration. | Guillaume Ross |
| YubiKey adoption | Monthly | Track YubiKey adoption in Google workspace and follow up with those that aren't using it. | Guillaume Ross |
| Dogfood policy update | Monthly | Edit Fleet dogfood policies that check for Chrome, Docker, and macOS version numbers. | Guillaume Ross |
| Security blog post | Monthly | Publish a security-related blog post to Fleet's blog. | Guillaume Ross |
| Security lunch & learn | Monthly | Educational live stream session on cybersecurity for Fleet employees, sessions are later shared for public consumption on YouTube. | Guillaume Ross |
| MDM device enrollment | Quarterly | Provide export of MDM enrolled devices to ops team. | Guillaume Ross |
| Access revalidation | Quarterly | Review critical access groups to make sure they contain only relevant people. | Guillaume Ross |
| Snyk scan | Quarterly | Scan Fleet repo with Snyk to identify important vulnerabilities that GitHub did not identify. | Guillaume Ross |
| Security conference review | Quarterly | Look at upcoming security conferences and work with the growth team to determine whether Fleet should submit to attend. | Guillaume Ross |
| Security policy update | Annually | Update security policies and have them approved by the CEO. | Guillaume Ross |


## Slack channels

This group maintains the following [Slack channels](https://fleetdm.com/handbook/company#group-slack-channels):

| Slack channel                       | [DRI](https://fleetdm.com/handbook/company#group-slack-channels)    |
|:------------------------------------|:--------------------------------------------------------------------|
| `#g-security`                       | Guillaume Ross
| `#help-login`                       | Guillaume Ross
| `#help-tines`                       | Guillaume Ross


<meta name="maintainedBy" value="guillaumeross">
<meta name="title" value="⛑ Security">

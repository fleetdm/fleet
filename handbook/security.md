# Security

## How we protect end-user devices

At Fleet, we believe that a good user experience empowers employees.

We follow the guiding principles below to secure our company-owned devices:

* Our devices should give employees the freedom to work from anywhere.
* To allow maximum freedom in where and how we work, we assume that "Safe" networks do not exist. Employees should be able to work on a coffee shop's Wi-Fi as if it were their home or work network.
* To limit the impact on user experience, we do not dictate security configurations unless the security benefit is significant.
* By using techniques such as Two-Factor Authentication (2FA), code reviews, and more, we can further empower employees to work comfortably from any location - on any network.


### macOS devices

We use configuration profiles to standardize security settings for our Mac devices. We use [CIS Benchmark for macOS 12](https://www.cisecurity.org/benchmark/apple_os), as our configuration baseline, and adapt it to:
* Suit a remote team
* Balance the need for productivity and security
* Limit the impact on the daily use of our devices

*Note: details of your Mac’s configuration profile can be viewed anytime from the **Profiles** app under **System Preferences**.)*



Our policy, which applies to Fleet owned laptops purchased via Apple's DEP (Device Enrollment Program), and which will retroactively be applied to every company owned Mac consists of: 

#### Enabling automatic updates

| #   | Setting                                                                                |
| --- | -------------------------------------------------------------------------------------- |
| 1.1 | Ensure all Apple-provided software is current                                          |
| 1.2 | Ensure auto update is enabled                                                          |                          |
| 1.4 | Ensure installation of app updates is enabled                                          |
| 1.5 | Ensure system data files and security updates are downloaded automatically is enabled |
| 1.6 | Ensure install of macOS updates is enabled                             |

*Note: the setting numbers included in the tables throughout this section are the recommended numbers from the CIS Benchmark for macOS12 document referenced above.*

**Why?**

Keeping software up-to-date helps to improve the resilience of our Mac fleet. Software updates include security updates that fix vulnerabilities that could otherwise be exploited. Browsers, for example, are often exposed to untrusted code, have a significant attack surface, and are frequently attacked.

macOS includes [malware protection tools](https://support.apple.com/en-ca/guide/security/sec469d47bd8/web) such as *Xprotect*, which is antivirus technology based on [YARA](https://github.com/VirusTotal/yara), and MRT (Malware Removal Tool), which is a tool built by Apple to remove common malware from systems that are infected.
By enabling these settings we:

* Ensure the operating system is kept up to date.
* Ensure XProtect and MRT are as up to date as possible.
* Ensure that Safari is kept up to date. 

This improves the resilience of our Mac fleet. 

**User experience impact**

* Updates are required, which can be disruptive. For this reason, we allow the user to **postpone the installation 5 times**.
* Critical security updates are automatically downloaded, which could result in bandwidth use on slow or expensive links. For this reason, we limit automatic downloads to critical security updates only, while feature updates, that are typically larger, are downloaded at the time of installation selected by the user.
* Enforced updates **do not** include major macOS releases (e.g., 11➡️12). Those updates are tracked and enforced separately, as the impact can be more significant. We require installation of the latest macOS version within 3 months of release, or when known vulnerabilities have remained unpatched on the older version.

#### Time and date

| #     | Setting                                             |
| ----- | --------------------------------------------------- |
| 2.2.1 | Ensure "Set time and date automatically" is enabled |

**Why?**

Accurate time is important for two main reasons:
1. Authentication. Many authentication systems like [Kerberos](https://en.wikipedia.org/wiki/Kerberos_(protocol)) and [SAML](https://en.wikipedia.org/wiki/Security_Assertion_Markup_Language) require the time between clients and servers to be [close](http://web.mit.edu/Kerberos/krb5-1.5/krb5-1.5.4/doc/krb5-admin/Clock-Skew.html). Keeping accurate time allows those protocols to prevent attacks that would leverage old authentication sessions. 
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

This category of settings is special because there are more settings that we do *not* configure than ones we do.

We follow the CIS benchmark where it makes sense, and in this case, take guidance from [NIST SP800-63B - Digital Identity Guidelines](https://pages.nist.gov/800-63-3/sp800-63b.html), especially [Appendix A -Strength of Memorized Secrets](https://pages.nist.gov/800-63-3/sp800-63b.html#appA).

* We do NOT enforce special complexity beyond requiring letters to be in the password.

Length is the most important factor when determining a secure password; while enforcing password expiration, special characters and other restrictive patterns are not as effective as previously believed and provide little benefit at the cost of hurting the user experience.

* We do NOT enforce extremely long passwords. 

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

**User experience impact**

* Inability to use the computer as a server to share internet access, printers, content caching of macOS and iOS updates, and streaming iTunes media to devices on the local network.
* File shares require an account.

#### Encryption, Gatekeeper and firewall

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
* Using the firewall will ensure that we limit the exposure to our devices, while Stealth mode makes them more difficult to discover. 
* Firewall logging allows us to troubleshoot and investigate whether the firewall blocks applications or connections.

**User experience impact**

* Due to FileVault's encryption process, a password is needed as soon as the laptop is turned on, instead of once it has booted.
* No performance impact - macOS encrypts the system drive by default. 
* With Gatekeeper enabled, unsigned or unnotarized (not checked for malware by Apple) applications require extra steps to execute.
* With the firewall enabled, unsigned applications cannot open a firewall port for inbound connections.

#### Screen saver and automatic locking

| #     | Setting                                                                             |
| ----- | ----------------------------------------------------------------------------------- |
| 2.3.1 | Ensure an inactivity interval of 20 minutes or less for the screen saver is enabled |
| 6.1.2 | Ensure show password hint is disabled                                              |
| 6.1.3 | Ensure guest account is disabled                                                    |
| NA    | Prevent the use of automatic logon                                                  |

**Why?**

* Fleet employees are free to work from wherever they choose. If a laptop is lost or forgotten, automatic login exposes sensitive company data and poses a critical security risk. 
* Password hints can sometimes be easier to guess than the password itself. Since we support employees remotely via MDM and do not require users to change passwords frequently, we eliminate the need for passwords hints and their associated risk.
* Only a single employee should use a company laptop. Therefore, guest accounts are not permitted.
* Automatic logon would defeat the purpose of even requiring passwords to unlock computers.

**User experience impact**

* Laptops lock after 20 minutes of inactivity. To voluntarily pause this, a [hot corner](https://support.apple.com/en-mo/guide/mac-help/mchlp3000/mac) can be configured to disable the screen saver. This is useful if you are, for example, watching an online meeting without moving the mouse and want to be sure the laptop will not lock.
* Forgotten passwords can be fixed via MDM, instead of relying on potentially dangerous hints.
* Guest accounts are not available.

#### iCloud
We do not apply ultra restrictive Data Loss Prevention style policies to our devices. Instead, by using different web-based tools we have, we ensure that the most critical company data is not stored on our laptops, so it can remain secure, while our laptops can remain productive.


| #       | Setting                                                   |
| ------- | --------------------------------------------------------- |
| 2.6.1.4 | Ensure iCloud Drive Documents and Desktop sync is disabled |

**Why?**
* We do not use managed Apple IDs, and allow employees to use their own iCloud accounts. We disable iCloud Documents and Desktop sync to avoid "accidental" copying of data to iCloud, but we do allow iCloud drive.

**User experience impact**

* iCloud remains allowed, but the Desktop and Documents folders will not be synchronized. Ensure you put your documents in our Google Drive, so you do not lose them if your laptop has an issue.

#### Miscellaneous security settings

| #     | Setting                                                      |
| ----- | ------------------------------------------------------------ |
| 2.5.6 | Ensure limit ad tracking is enabled                          |
| 2.10  | Ensure secure keyboard entry Terminal.app is enabled         |
| 5.1.4 | Ensure library validation is enabled                         |
| 6.3   | Ensure automatic opening of safe files in Safari is disabled |

**Why?**

* Limiting ad tracking has privacy benefits, and no downside.
* Protecting keyboard entry into Terminal.app could prevent malicious applications or non-malicious but inappropriate applications from receiving passwords.
* Library validation ensures that an attacker can't trick applications into loading a software library in a different location, leaving it open to abuse.
* Safari opening files automatically can lead to negative scenarios where files are downloaded and automatically opened in another application. Though the setting relates to files deemed "safe", it includes PDFs and other file formats where malicious documents exploiting vulnerabilities have been seen before.

**User experience impact**

* There is minimal to no user experience impact for these settings. However, applications used to create custom keyboard macros will not receive keystrokes when Terminal.app is the active application window.


#### Enforce DNS-over-HTTPs (DoH)

| #  | Setting                |
| -- | ---------------------- |
| NA | Enforce [DNS over HTTPS](https://en.wikipedia.org/wiki/DNS_over_HTTPS) |

**Why?**

* We assume that no network is "safe." Therefore, DNS queries could be exposed and leak private data. An attacker on the same wireless network could see DNS queries, determine who your employer is, or even intercept them and [respond with malicious answers](https://github.com/iphelix/dnschef). Using DoH protects the DNS queries from eavesdropping and tampering.
* We use Cloudflare's DoH servers with basic malware blocking. No censorship should be applied on these servers, except towards destinations known as malware related.


**User experience impact**

* Some misconfigured "captive portals", typically used in hotels and airports, might be unusable with DoH due to how they are configured. This can be worked around by using the hotspot on your phone, and if you really have to use this network for an extended period of time, there are usually workarounds that can be performed to connect to them. Navigating to http://1.1.1.1 often resolves the issue.
* If you are trying to reach a site, and you believe it is being blocked accidentally, please submit it to Cloudflare. This should be extremely rare. If it is not, please let the security team know.
* If your ISP's DNS service goes down, you'll be able to continue working 😎

*Note: If you from another organization, reading this to help create your own configuration, remember that implementing DoH in an office environment where other network controls are in place has different downsides than doing it for a remote company. In those cases, **disabling** DoH makes more sense, so network controls can retain visibility. Please evaluate your situation before implementing any of our recommendations at your organization, especially DoH.*

#### Deploy osquery
| #  | Setting                |
| -- | ---------------------- |
| NA | Deploy [osquery](https://osquery.io/) pointed to our dogfood instance |

***Why?***

We use osquery and Fleet to monitor our own devices. This is used for vulnerability detection, security posture tracking, and can be used for incident response when necessary.


### Chrome configuration
We configure Chrome on company-owned devices with a basic policy.

| Setting                                                   |
| --------------------------------------------------------- |
| Enforce Chrome updates and Chrome restart within 48 hours |
| Block intrusive ads                                       |
| uBlock Origin ad blocker extension deployed               |
| Password manager extension deployed                       |
| Chrome Endpoint Verification extension deployed           |

**Why?**

* Browsers have a large attack surface, and their updates contain critical security updates. 

**User experience impact**

* Chrome needs to be restarted within 48 hours of patches being installed. The automatic restart happens after 19:00 and before 6:00 if the computer is running, and tabs are restored (except for incognito tabs).
* Ads considered intrusive are blocked.
* uBlock Origin is enabled by default, and is 100% configurable, improving security and performance of browsing.
* Endpoint Verification is used to make access decisions based on the security posture of the device. For example, an outdated Mac could be prevented access to Google Drive.

### Personal mobile devices

The use of personal devices is allowed for some applications, as long as the iOS or Android device is kept up to date.

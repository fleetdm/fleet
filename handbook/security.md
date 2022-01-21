# Security

## End-user devices

At Fleet, we believe employees should be empowered to work with devices that provide a good experience. 

We follow the following guiding principles to secure our company-owned devices:

* Laptops and mobile devices are used from anywhere, and we must be able to manage and monitor them no matter where they are.
* Assume they are being used on a dangerous network at all time. There is no such thing as a safe network for the device to be on, therefore, network communications must be protected.
* Do not dictate the configuration of preferences unless the security benefit is significant, to limit the impact on user experience.
* Put as little trust as possible in these endpoints, by using techniques such as hardware security keys.

### macOS configuration baseline

Our macOS configuration baseline is simple, and has limited impact on daily use of the device. It is based on the [CIS Benchmark for macOS 12](https://www.cisecurity.org/benchmark/apple_os), adapted for a fully remote team and balancing the need for productivity and security.

The setting number in this document are the recommendation numbers from the CIS document.

The detailed configuration deployed to your Mac can always be inspected by using the *Profiles* app under *System Preferences*.

Our policy, which applies to Fleet owned laptops purchased via Apple's DEP (Device Enrollment Program), consists of: 

#### Enabling automatic updates

**What are the settings?**

| #   | Setting                                                                                |
| --- | -------------------------------------------------------------------------------------- |
| 1.1 | Ensure all Apple-provided software is current                                          |
| 1.2 | Ensure auto update is enabled                                                          |                          |
| 1.4 | Ensure Installation of app updates is enabled                                          |
| 1.5 | Ensure system data files and security updates are downloaded automatically is enabled |
| 1.6 | Ensure install of macOS updates is enabled                             |

**Why?**

Software updates frequently include security updates. These fix vulnerabilities, which might already have been publicly disclosed, could be exploited in the wild, and if not, could be in a few days. Updates are also released through the same system for Safari. Browsers are exposed to untrusted code all day, and have a significant attack surface, and are frequently attacked when new vulnerabilities are discovered.

macOS also includes [malware protection toos](https://support.apple.com/en-ca/guide/security/sec469d47bd8/web) such as *Xprotect*, which is antivirus technology based on [YARA](https://github.com/VirusTotal/yara), and MRT (Malware Removal Tool) which is a tool built by Apple to remove common malware from systems that are infected.


By enabling these settings we:

* Ensure the operating system is kept up to date
* Ensure XProtect and MRT are updated as frequently as possible
* Ensure that Safari is kept up to date. 

This improves the resilience of our Mac fleet significantly. 

**User experience impact**

* Updates will have to be downloaded, and installation will be required, which could be disruptive. For this reason, we allow the user to **postpone the installation** 5 times, to pick an ideal time.
* We do not enforce the download of all updates, only security ones, to reduce the risk of large updates being downloaded while on tethering or other slow or expensive links. This means that when prompted to install an update, they first have to be downloaded.
* The updates that are enforced **do not** include major macOS releases (11➡️12). Those updates are tracked and enforced separately, as the impact can be more important. Generally, we require installing the latest macOS release within 3 months of release, or when known vulnerabilities have remained unpatched on the older version.

#### Time and date

| #     | Setting                                             |
| ----- | --------------------------------------------------- |
| 2.2.1 | Ensure "Set Time and date automatically" is enabled |

**Why?**

Accurate time is important for two main reasons.
1. Authentication. Many authentication systems like [Kerberos](https://en.wikipedia.org/wiki/Kerberos_(protocol)) and [SAML](https://en.wikipedia.org/wiki/Security_Assertion_Markup_Language) require the time between clients and servers to be [relatively close](http://web.mit.edu/Kerberos/krb5-1.5/krb5-1.5.4/doc/krb5-admin/Clock-Skew.html). This allows those protocols to prevent attacks that would leverage old authentication sessions. 
2. Logging. Performing troubleshooting or incident response is much easier when all the logs involved have timestamps that are close to perfectly synchronized.

**User experience impact**

* Minimal. Inability to set the wrong time. Time zones remain user configurable.

#### Passwords

| #     | Setting                                                                                  |
| ----- | ---------------------------------------------------------------------------------------- |
| 5.2.2 | Ensure Password Minimum Length Is Configured (our minimum: 8 characters)                                             |
| 5.2.3 | Ensure Complex Password Must Contain Alphabetic Characters is Configured                 |
| 5.8   | Ensure a password is required to wake the computer from sleep or screen saver is enabled |

**Why?**

This category of setting is special, because there are more settings that we do *not* configure than ones we do.

We follow the CIS benchmark where it makes sense, and in this case, take guidance from [NIST SP800-63B - Digital Identity Guidelines](https://pages.nist.gov/800-63-3/sp800-63b.html), especially [Appendix A -Strength of Memorized Secrets](https://pages.nist.gov/800-63-3/sp800-63b.html#appA).

Essentially, length the most important factor, while enforcing password expiration, special characters and other restrictive patterns not as effective as previously believed. Everyone has updated a password by simply changing a number at the end of it, or capitalized the first letter of a password because they had to use at least one uppercase character.

* As we only use recent Macs with T2 chips or Apple Silicon, brute-force attacks against the hardware are [mitigated](https://www.apple.com/mideast/mac/docs/Apple_T2_Security_Chip_Overview.pdf), we do not need to enforce an extremely long password. Therefore, we will use the minimum recommended by SP800-63B, **8** characters.
* We will NOT enforce special complexity beyond requiring letters to be in the password, as that leads to people updating passwords in predictable ways, or storing them insecurely, and generates headaches and support cost.
* We disable many services, and enable the firewall, reducing the odds that the passwords could be broken online by attempting to connect to a service, such as a file share, or HTTP server using local accounts. Since we can't eliminate this 100%, we still require 8 character long passwords with letters, to be safe.

**User experience impact**

* A password is required to boot, as well as to unlock the laptop. Touch ID and Apple Watch unlock are allowed, and we recommend using a longer password and using one of those techniques to reduce the annoyances through the day.



#### Disabling various services

| #      | Setting                                           |
| ------ | ------------------------------------------------- |
| 2.4.2  | Ensure Internet Sharing is Disabled               |
| 2.4.4  | Ensure Printer Sharing is Disabled                |
| 2.4.10 | Ensure Content Caching is disabled                |
| 2.4.12 | Ensure Media Sharing is Disabled                  |
| 6.1.4  | Ensure Guest Access to Shared Folders is Disabled |

**Why?**

* Any service listening on a port expands the attack surface, especially when working on unsafe networks, which is where we assume all laptops are located.
* Internet sharing could turn a laptop into a bridge, if it had some kind of tunnel connecting it to internal systems (TLS tunnel, SSH tunnel, VPN.)
* Guest access to shared data could lead to accidental exposure of files.

**User experience impact**

* Inability to use the computer as a server to share Internet access, printers, content caching of macOS and iOS updates as well as to stream iTunes media content to devices on the LAN.
* File shares are only accessible with a real account.

#### Encryption, Gatekeeper and firewall

| #       | Setting                                           |
| ------- | ------------------------------------------------- |
| 2.5.1.1 | Ensure FileVault is enabled                       |
| 2.5.2.1 | Ensure Gatekeeper is Enabled                      |
| 2.5.2.2 | Ensure firewall is enabled                        |
| 2.5.2.3 | Ensure Firewall Stealth Mode is Enabled           |
| 3.6     | Ensure firewall logging is enabled and configured |

**Why?**

* Using FileVault protects the data on our laptops, which includes not only confidential data but also session material (browser cookies), SSH keys, and more. By enforcing the use of FileVault, we ensure a lost laptop is a minor inconvenience and not an incident. We also escrow the keys, to be sure we can recover the data if needed.
* [Gatekeeper](https://support.apple.com/en-ca/HT202491) is a feature on macOS that verifies if applications are properly signed by the developer, and notarized by Apple, a process where they do some testing on the application before "stamping" it. The certificates used by applications can be revoked, for example, if a vendor is discovered to be bundling malware in legitimate applications. With Gatekeeper enabled, unsigned and/or unnotarized applications will not be executed with the standard double-click of the icon. This is a very useful first line of defense to have.
* Using the firewall will ensure that we limit the exposure to our computers, especially in dangerous network environmnets, which is where we assume they always are. Stealth mode will make it more difficult to discover. 
* Firewall logging allows us to troubleshoot and investigate, by letting us know if some applications or connections are being blocked by it.

**User experience impact**

* A password will be needed as soon as the laptop is turned on, due to the encryption process, instead of once the laptop has booted.
* No performance impact - macOS encrypts the system drive by default. Enabling FileVault makes it so the password of an authorized user is necessary to boot and decrypt, and does not change throughput or latency negatively.
* Unsigned or unnotarized applications will require extra steps to execute.
* Unsigned applications will not be allowed to open a firewall port for inbound connections.

#### Screen saver and automatic locking

| #     | Setting                                                                             |
| ----- | ----------------------------------------------------------------------------------- |
| 2.3.1 | Ensure an Inactivity Interval of 20 minutes or less for the screen saver is enabled |
| 6.1.2 | Ensure Show Password Hints is Disabled                                              |
| 6.1.3 | Ensure Guest Account is Disabled                                                    |
| NA    | Prevent the use of automatic logon                                                  |

**Why?**

* Our workstations are mostly laptops, which by definition are portable and used in many different areas. If a laptop is forgotten, while logged in, it defeats the purpose of other controls and exposes data that could be critical. By enforcing the screen to lock, we reduce the odds that a laptop forgotten unlocked in a home, office, hotel room could be accessed.
* Password hints can be dangerous, as they can sometimes be easier to guess than the password itself. Since we can support employees remotely via MDM, and since we do not require passwords that expire, we eliminate that risk by hiding the hints.
* Guest accounts are not useful for systems that are used by a single employee, therefore, we disable them.
* Automatic logon would defeat the purpose of even requiring passwords to unlock computers, so we ensure it can't be used.

**User experience impact**

* Laptops will lock after 20 minutes of inactivity. To voluntarily pause this, you can configure a [hot corner](https://support.apple.com/en-mo/guide/mac-help/mchlp3000/mac) to disable the screen saver. This can be useful if you are, for example, watching an online meeting without moving the mouse and want to be sure the laptop will not lock.
* Forgotten passwords will have to be fixed via MDM, instead of relying on potentially dangerous hints.
* Guest accounts will not be available.

#### iCloud
We do not apply ultra restrictive [Data Loss Prevention](https://en.wikipedia.org/wiki/Data_loss_prevention_software) style policies to our workstations. A computer that is used for day to day work, with full access to the Internet can never be protected from voluntary malicious actors willing to upload data. Instead, we focus on ensuring the most critical data never reaches our laptops, so it can remain secure, while our laptops can remain productive.


| #       | Setting                                                   |
| ------- | --------------------------------------------------------- |
| 2.6.1.4 | Ensure iCloud Drive Document and Desktop Sync is Disabled |

**Why?**
* We do not use managed Apple IDs, and allow employees to use their own iCloud accounts. We disable this to avoid "accidental" copying of data to iCloud, but still allow iCloud drive.

**User experience impact**

* iCloud remains allowed, but the defautl Desktop and Documents folders will not be synchronized. Ensure you put your documents in our Google Drive, so you do not lose them if your laptop has an issue.

#### Miscellaneous security settings

| #     | Setting                                                      |
| ----- | ------------------------------------------------------------ |
| 2.5.6 | Ensure Limit Ad Tracking is Enabled                          |
| 2.10  | Ensure Secure Keyboard Entry terminal.app is Enabled         |
| 5.1.4 | Ensure Library Validation is Enabled                         |
| 6.3   | Ensure Automatic Opening of Safe FIles in Safari is Disabled |

**Why?**

* Limiting ad tracking has privacy benefits, and no downside beyond receiving less targeted advertising.
* Protecting keyboard entry into the terminal app could prevent malicious applications, or non-malicious but inappropriate applications from receiving passwords and other secrets.
* Library validation ensures that applications can't be tricked into loading a library in a different location, leaving it open to abuse.
* Safari opening files automatically can lead to negative scenarios where files are downloaded and automatically opened in another application. Though the setting specifically relates to files deemed "safe", it includes PDFs and other file formats where malicious documents exploiting vulnerabilities have been seen before, and it is just plain annoying in any case when a website is able to pop-up multiple Preview application windows.

**User experience impact**

* Minimal to invisible for most of these settings, however, applications used to create custom keyboard macros will not be able to receive keystrokes when Terminal.app is the active application window.

### Personal mobile devices

The use of personal devices is allowed for some applications. Your iOS or Android device must however be kept up to date.

#### Enforce DNS-over-HTTPs (DoH)

| #  | Setting                |
| -- | ---------------------- |
| NA | Enforce DNS over HTTPS |

**Why?**

* We assume laptops are used on dangerous networks at all times. Therefore, DNS queries could be exposed, and leak private data. An attacker on the same wireless network could easily see DNS queries, determine who your employer is, or even intercept them and respond with malicious answers. Using DoH protects the DNS queries from eavesdropping and tampering.
* We use Cloudflare's DoH servers with basic malware blocking. No censorship should be applied on these servers, except towards destinations known as malware related.


**User experience impact**

* Some misconfigured captive portals, such as in hotels, could be unreachable, as they essentially perform an attack on DNS traffic to redirect you, and some of them misuse IP addresses such as *1.1.1.1*. In some cases, you can work around this by performing a *nslookup* in the terminal,f or any domain, and manually browsing to the IP being provided as the response. **This should be relatively rare, and getting rarer by the day. The best workaround is to use tethering on a phone or hotspot**.
* Some rare false positives could happen, preventing access to a site. Please report those, as if they become frequent, we will define a strategy for handling them.
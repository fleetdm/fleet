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

**Why?**

* Using FileVault protects the data on our laptops, which includes not only confidential data but also session material (browser cookies), SSH keys, and more. By enforcing the use of FileVault, we ensure a lost laptop is a minor inconvenience and not an incident. We also escrow the keys, to be sure we can recover the data if needed.
* [Gatekeeper](https://support.apple.com/en-ca/HT202491) is a feature on macOS that verifies if applications are properly signed by the developer, and notarized by Apple, a process where they do some testing on the application before "stamping" it. The certificates used by applications can be revoked, for example, if a vendor is discovered to be bundling malware in legitimate applications. With Gatekeeper enabled, unsigned and/or unnotarized applications will not be executed with the standard double-click of the icon. This is a very useful first line of defense to have.
* Using the firewall will ensure that we limit the exposure to our computers, especially in dangerous network environmnets, which is where we assume they always are. Stealth mode will make it more difficult to discover. 

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



---

* Ensures the operating system and applications are kept as up-to-date as possible, while allowing leeway to avoid a situation where an enforced update prevents you from working on Monday morning!
* Encrypts the disk and escrows the key, to ensure we can recover data if needed, and to ensure that lost or stolen devices are a minor inconvenience and not a breach.
* Prevents the use of Kernel Extensions that have not been explicitly allow-listed. 
* Ensures that "server" features of macOS are disabled, to avoid accidentally exposing data from the laptop to other devices on the network. These features include: Web server, file sharing, caching server, Internet sharing and more.
* Disables guest accounts.
* Enables the firewall.
* Enables Gatekeeper, to protect the Mac from unsigned software.
* Enforces DNS-over-HTTPS, to protect your DNS queries on dangerous networks, for privacy and security reasons.
* Enforces other built-in operating system security features, such as library validation.
* Requires a 10 character long password. We recommend using a longer one and leveraging Touch ID, to have to type it less.

The detailed configuration deployed to your Mac can always be inspected by using the *Profiles* app under *System Preferences*, but at a high level, the configuration does the following:


### Personal mobile devices

The use of personal devices is allowed for some applications. Your iOS or Android device must however be kept up to date.
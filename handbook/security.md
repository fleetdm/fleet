# Security

## End-user devices

At Fleet, we believe employees should be empowered to work with devices that provide a good experience. 

We follow the following guiding principles to secure our company-owned devices:

* Laptops and mobile devices are used from anywhere, and we must be able to manage and monitor them no matter where they are.
* Assume they are being used on a dangerous network at all time. There is no such thing as a safe network for the device to be on, therefore, network communications must be protected.
* Do not dictate the configuration of preferences unless the security benefit is significant, to limit the impact on user experience.
* Put as little trust as possible in these endpoints, by using techniques such as hardware security keys.

### macOS configuration baseline

Our macOS configuration baseline is simple, and has limited impact on daily use of the device.

Our policy, which applies to Fleet owned laptops purchased via Apple's DEP (Device Enrollment Program), consists of: 

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
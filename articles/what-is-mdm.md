Organizations shipping laptops to remote employees, enforcing encryption across operating systems, and preparing for compliance audits often find themselves juggling disconnected tools that each cover only part of the fleet. Mobile device management (MDM) ties those workflows together under a single framework. This guide covers what MDM is, how it works, and the key decisions that shape a successful deployment.

## What is mobile device management?

MDM is a system for enrolling, configuring, and managing organizational devices from a central console. Once devices are enrolled, administrators can enforce security settings, deploy applications, manage OS updates, and take remote actions like locking or wiping a device without physical access. The same management channel provides ongoing visibility into device state across the fleet, whether devices are in an office or shipped directly to remote employees.

## Key features and benefits of MDM

MDM differs from general endpoint management in that the control plane sits in OS-level management protocols rather than a vendor-installed agent. That distinction shapes what MDM does well.

- Zero-touch provisioning: IT pre-stages device assignments before shipment. When an employee first powers on a new device, vendor-side enrollment flows handle registration and initial configuration without IT physically handling the hardware. Apple's Setup Assistant via Apple Business Manager, Windows Autopilot via Microsoft Entra ID, Android Enterprise zero-touch enrollment, and ChromeOS zero-touch enrollment all follow this model.
- Configuration profile delivery: Profiles delivered over the management channel enforce settings like encryption requirements, Wi-Fi configurations, virtual private network (VPN) profiles, and passcode requirements across the fleet.
- Remote actions: Commands like lock and wipe travel the same management channel, so IT can recover or retire a device without physical access.
- Near real-time visibility: MDM reports device attributes like OS version, encryption status, installed software, and certificate validity. That reporting feeds compliance audits and conditional access decisions.

All four capabilities ride the same protocol channel between the MDM server and the device, which the next section walks through.

## How mobile device management works

MDM uses a command-and-response model where the server queues instructions and enrolled devices execute them when they connect. Each major platform has its own management protocol:

- Apple (iOS, iPadOS, macOS): Apple's MDM protocol over HTTPS, with Apple Push Notification service (APNs) as the wake-up channel.
- Windows: Open Mobile Alliance Device Management (OMA-DM) over HTTPS, with Windows Notification Service (WNS) for check-in signals.
- Android: managed under the Android Enterprise framework, accessed through either the Google-hosted Android Management API or EMM-provided Device Policy Controller (DPC) apps. The original Device Administration API is deprecated.
- ChromeOS: managed through the cloud-based Google Admin console, which delivers device-level policies that apply at boot and user-level policies that apply at sign-in.
- Linux: no native MDM protocol. Management relies on installed agents that pull configuration from a central server.

The term "mobile" is historical: these protocols now govern laptops, desktops, tablets, and phones.

Push notification services rather than persistent connections trigger check-ins. When a device receives the signal, it connects to the MDM server, retrieves queued commands, executes them, and reports results. Push notifications carry no commands or data. They're wake-up signals only.

Enrollment sets up the trust relationship between a device and the MDM server before any commands can flow. The path varies by platform:

- Apple: Apple Business Manager (ABM) combined with Automated Device Enrollment (ADE) automates enrollment at first boot.
- Windows: Autopilot joins the device to Microsoft Entra ID and triggers MDM enrollment.
- Android: Android Enterprise zero-touch enrollment registers corporate-owned devices before shipment, and Knox Mobile Enrollment serves the same role for Samsung hardware. QR code or Google account flows handle BYOD enrollment.
- ChromeOS: devices register to a Google Admin domain through enterprise enrollment at first boot. Zero-touch enrollment automates this for compatible hardware bought through authorized resellers.
- Linux: no equivalent native flow exists. Devices come under management by installing and configuring an agent.

Apple is also moving toward a declarative model with Declarative Device Management (DDM), which applies to Apple devices only. Rather than processing server-issued commands in sequence, the device evaluates declarations and applies configurations autonomously, reporting its state back proactively. Microsoft's Declared Configuration follows a similar desired-state approach for Windows.

## Best practices for implementing MDM

These practices apply regardless of fleet size or platform mix.

- Define configuration as code from the start: When configuration lives in version-controlled files rather than GUI settings, you get an audit trail and a single source of truth. Teams can review changes through pull requests and deploy them automatically, preventing the configuration drift that shows up when some changes happen through a console and others through scripts. Fleet integrates natively with git repositories through [GitOps workflows](https://fleetdm.com/guides/what-i-have-learned-from-managing-devices-with-gitops).
- Connect your identity provider early: MDM reports device compliance state, and the identity provider uses that state for conditional access decisions. Without that integration, you end up managing devices and access in separate silos where neither system has the full picture.
- Verify device state, not command acknowledgment: Traditional MDM confirms that a device received a command, but that doesn't confirm the configuration was applied correctly. Fleet pairs MDM-based configuration with osquery-based [device reports](https://fleetdm.com/guides/queries) to compare intended state with observed state, catching the gap between "command delivered" and "setting enforced."

The design decisions below (ownership model, compliance requirements, and platform mix) shape how straightforward each of these practices is to implement.

## Key MDM design decisions

Three decisions shape most MDM deployments: who owns the devices, what compliance frameworks apply, and which platforms need management.

### BYOD and ownership models

Bring Your Own Device (BYOD) and corporate-owned hardware require different management scopes. Some organizations use Mobile Application Management (MAM) for personal devices, protecting corporate data at the app layer without any device enrollment. For enrollment-based management of personal devices, MDM's lighter-touch enrollment modes cover most requirements. On Apple devices, User Enrollment and Account-Driven Enrollment provide cryptographic separation between corporate and personal data. On Windows, adding a work account via Microsoft Entra ID can trigger scoped enrollment where IT manages corporate apps and configuration profiles while personal data stays separate.

A BYOD policy also benefits from early decisions about app ownership, remote wipe scope on personal devices, and data handling after employment ends. Those are organizational determinations that shape your enrollment model rather than settings you configure after the fact.

### Compliance and regulatory alignment

MDM provides configuration state data, which is the evidence compliance frameworks require. If your organization is subject to federal guidance on enterprise MDM, that guidance covers both organization-provided and personally-owned deployment scenarios and addresses device and EMM configurations.

For HIPAA, MDM-enforced [encryption status](https://fleetdm.com/tables/disk_encryption) supports the §164.312(a)(2)(iv) encryption specification. That covers FileVault on macOS, BitLocker on Windows, and platform-native device encryption on iOS, iPadOS, Android, and ChromeOS, with key escrow on the platforms that support it. Under HIPAA, that specification is addressable: your organization assesses whether encryption is appropriate for your environment, not a fixed requirement. Healthcare organizations subject to HIPAA often list MDM as a formal cybersecurity practice for protecting devices that access patient data.

MDM also works alongside other security tools. Mobile Threat Defense solutions are often deployed as managed apps via MDM, showing how MDM serves as the delivery mechanism for adjacent security capabilities. For zero-trust controls, MDM provides the device trust signal that identity providers evaluate when granting or restricting access to corporate resources.

### Platform coverage

The platform mix shapes what to prioritize in your evaluation. If your environment spans macOS, Windows, Linux, iOS, iPadOS, Android, and ChromeOS, consistent multi-platform coverage matters more than depth on any single platform. Linux has no native MDM protocol or central device registry, so Linux management relies on agent-based approaches. The depth of that coverage varies across solutions and is worth testing against your actual workflows.

## How Fleet handles multi-platform MDM

The platform coverage question above is where Fleet was designed from the start, rather than added later. [Device management](https://fleetdm.com/device-management) for macOS, Windows, Linux, iOS, iPadOS, ChromeOS, and Android runs through a single console and API. Configuration profiles, OS updates, and software deployment share one workflow regardless of platform.

Configuration lives in version-controlled files via GitOps, which keeps the audit trail and pull-request workflow consistent across operating systems rather than fragmented across per-platform consoles. The compliance state data MDM produces feeds the same reporting pipeline regardless of which platform generated it.

osquery runs on macOS, Windows, and Linux to provide SQL-queryable device data alongside MDM-based configuration on those platforms. iOS, iPadOS, Android, and ChromeOS are managed through MDM protocol channels without an agent. Both cloud-hosted and self-hosted deployments use the same code, so the depth of management doesn't depend on the hosting model.

## Closing the gap between configuration and verification

The recurring theme across MDM implementation is the gap between sending a configuration and confirming it took effect. Closing that gap means verifying observed state against intended state, not only acknowledging that a command was delivered.

Fleet pairs MDM-based configuration with [device reports](https://fleetdm.com/guides/queries) built on osquery to compare what was sent with what is in place. The same verification workflow runs across macOS, Windows, and Linux for the agent layer, and across all supported platforms for MDM-delivered configuration. Teams don't have to stitch together evidence from separate tools at audit time.

That workflow applies whether you're onboarding a first group of devices or maintaining an established fleet across departments. To see how Fleet handles configuration and verification in your environment, [contact us](https://fleetdm.com/contact).

## Frequently asked questions

### Does MDM work without internet connectivity?

Configuration profiles and settings that are already installed on a device continue to apply locally while the device is offline. What stops working is the command channel: new configuration profiles, MDM commands, and status reports all require connectivity between the device and the MDM server.

On Apple devices, if the APNs push notification can't reach the device, no check-in occurs and queued commands wait until the device reconnects. On Windows, the same applies to WNS-triggered check-ins. Once Apple's Declarative Device Management (DDM) declarations have been delivered, devices can re-evaluate and reapply them locally as state changes, without waiting for a server command. New declarations and status uploads still require connectivity. Devices that stay offline for extended periods can also miss certificate renewals, which may require re-enrollment when connectivity returns.

### How long does MDM enrollment take for new devices?

The enrollment handshake itself typically completes in under a minute, but first-day readiness depends on what happens after enrollment. Apps, configuration profiles, certificates, and OS updates all queue up once the device registers with the server, and the total delivery time depends on payload size and network speed.

Teams that pre-stage smaller configuration profiles before larger app installs can often get devices to a usable state faster. Security-critical settings land first while heavier downloads continue in the background.

### Can MDM manage virtual machines and cloud-hosted desktops?

MDM is typically not the right fit for virtual machines and cloud-hosted desktops. Enrollment depends on a persistent device identity and ongoing trust relationship that short-lived or frequently rebuilt environments often don't preserve. Hardware-backed security features MDM relies on (Trusted Platform Module on Windows, Secure Enclave on Apple devices) may also not be exposed inside a virtual machine.

Virtual desktop infrastructure (VDI) environments that rebuild images on each login compound the problem. Enrollment state doesn't survive an image refresh unless the environment is explicitly configured to preserve it. Teams managing virtual desktops typically rely on the hypervisor's own management plane or session-based controls rather than MDM.

### What causes MDM settings to revert after applying?

Several platform-specific behaviors can cause a setting to revert even after the device acknowledges the command. On macOS, a user with admin privileges can modify managed preferences through Terminal unless the configuration profile restricts that access. On Windows, group policy objects (GPOs) from Active Directory can overwrite CSP-delivered settings if both target the same configuration key, with the last write winning.

Profile conflicts are another common cause. If two configuration profiles target the same setting with different values, the result depends on profile priority and installation order, which varies by platform. Fleet's osquery-based device reporting can help you catch these reversions by comparing observed state against intended configuration. [Get a demo](https://fleetdm.com/contact) to see how that works across your fleet.

<meta name="articleTitle" value="What is MDM? How mobile device management works and what to evaluate">
<meta name="authorFullName" value="Ashish Kuthiala">
<meta name="authorGitHubUsername" value="akuthiala">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-05-15">
<meta name="description" value="Learn what mobile device management (MDM) is, how it works across macOS, Windows, Linux, iOS, Android, and ChromeOS, and best practices. ">

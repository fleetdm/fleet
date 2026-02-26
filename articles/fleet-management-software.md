# Fleet vs. Jamf vs. Microsoft Intune: How to choose the right fleet management software

Managing a fleet of corporate devices means juggling inventory, configuration, security, and compliance across potentially thousands of endpoints. The tools IT teams use for this vary dramatically in platform coverage, deployment flexibility, and what's included versus sold separately.

This guide compares Fleet, Jamf Pro, and Microsoft Intune as device management software options, covering platform support, security capabilities, and automation features.

## Overview

### Fleet

Fleet is a multi-platform device management solution built on top of osquery for data collection. Fleet provides the following capabilities from a single console:

* Supports macOS, iOS, iPadOS, Windows, Linux, ChromeOS, and Android devices  
* Device data is returned in near real-time  
* API-first, GitOps-native design  
* Can be deployed on-prem, in almost any cloud-hosting platform (Amazon Web Services (AWS), Google Cloud (GCP), Render, Digital Ocean, Proxmox, etc.) on Docker, Kubernetes, or natively on almost any server hardware  
* Also available as a fully-hosted, cloud-managed SaaS offering on Fleet's own AWS infrastructure

By integrating Fleet with a git repository management solution like GitHub or GitLab, the fleetctl command line (CLI) binary is capable of running as part of a continuous integration (CI/CD) and delivery pipeline for complete, declarative control of the Fleet UI. Managing devices enrolled into Fleet, along with security controls and software with infrastructure-as-code techniques is extremely powerful and straightforward to set up.

### Jamf Pro

Jamf Pro is a device management solution focused on Apple devices. Key characteristics include:

* Supports macOS, iOS, iPadOS, tvOS, visionOS, and watchOS using Apple's management frameworks  
* Identity management (Jamf Connect) and endpoint security (Jamf Protect) are separate products at additional cost  
* Jamf's broader portfolio includes Android enrollment (Jamf for Mobile), but Jamf does not support Windows or Linux  
* Only available as a cloud-hosted deployment

Jamf Pro has a large customer base and long history in the Apple device management space. Organizations with Windows and Linux endpoints alongside Apple devices need additional tools to manage their full fleet.

### Microsoft Intune

Microsoft Intune is a cloud-based endpoint management solution and part of the Microsoft Endpoint Manager suite. Key characteristics include:

* Supports Windows, macOS, iOS/iPadOS, Android, Linux, and ChromeOS  
* Licensing is per user as part of Microsoft's tiered SKU scheme, with Plan 1, Plan 2, and Intune Suite options  
* Intune Plan 1 is included in Microsoft 365 E3/E5, Business Premium, and Enterprise Mobility \+ Security (EMS) licensing, though full endpoint security requires additional Microsoft Defender licensing  
* Only available as a cloud offering. There is no on-premises deployment option

Intune integrates with Microsoft Entra ID and Microsoft Defender. Intune's Windows capabilities are more developed than its macOS and Linux support, which have fewer features and configuration options.

## Key differences in fleet management software

| Attribute | Fleet | Jamf Pro | Intune |
| ----- | ----- | ----- | ----- |
| Architecture | API-first, GitOps-native, unified REST API, osquery data collection | GUI-first, multiple APIs | GUI-first, multiple APIs, |
| OS Coverage | macOS, iOS, iPadOS, Windows, Linux, ChromeOS, Android | macOS, iOS, iPadOS, tvOS, visionOS, Android | Windows, macOS, iOS, iPadOS, Android, Linux, ChromeOS |
| Deployment model | Cloud or self-hosting with full feature parity | Cloud-only | Cloud-only |
| GitOps support | GitOps-native, requires git repository management solution integration | Requires third-party tools (e.g., terraform provider) | Requires third-party tools (e.g., IntuneCD) |
| Security features | Vulnerability detection, YARA rules, file integrity monitoring, CIS benchmarks included | Requires Jamf Protect | Required Microsoft Defender |
| Device reporting | Near real-time osquery-backed telemetry | Device check-ins at 5m, 15m, 30m, 1h. Default inventory every 24h. | Inventory cadence every 6-8h. |
| Queries | SQL-based queries across all devices | Extension attributes for inventory | Limited query capabilities |
| Pricing model | Per-device premium subscription, open-source model | Per-device subscription plus add-ons | Bundled with Microsoft 365 E3 / E5 license |

## Device management workflow comparisons

Fleet, Jamf, and Intune all handle core device management functions: enrollment, configuration, software deployment, and compliance. The differences show up in automation capability, platform consistency, and which capabilities require additional purchases.

### Enrollment and provisioning

All three solutions support zero-touch deployment for the platforms they support. Apple devices enroll through Apple Business Manager. Windows devices use Windows Autopilot. All three can configure supervised settings that prevent users from removing management profiles.

Jamf Pro's PreStage enrollment configures Apple device onboarding, including signed package deployment during setup. Intune's Autopilot configures Windows device onboarding with comparable automation. Fleet supports both Apple Business Manager and Windows Autopilot, providing consistent zero-touch enrollment across a mixed device fleet.

### Configuration management

Jamf Pro uses Smart Groups and Static Groups to scope Configuration Profiles and policies. It uses extension attributes to collect custom inventory data. Intune uses device groups and user groups for policy assignment, with conditional access policies tied to Microsoft Entra ID. Configuration profiles work differently across Windows and macOS, with more mature controls on Windows.

Fleet uses Teams and Labels for scoping across all platforms. A label can be as simple as a manual tag or as sophisticated as a live osquery result. Devices dynamically match labels as their state changes, keeping policy assignments current without manual intervention. The same scoping model works identically whether targeting macOS, Windows, or Linux devices. When using Fleet with GitOps, configurations are version-controlled and auditable, with changes tracked through pull requests rather than GUI edits that lack audit trails.

Fleet supports Declarative Device Management (DDM) natively for Apple devices. Jamf Pro supports DDM, but availability of specific DDM-powered workflows can vary by hosting tier and subscription. Intune supports Apple DDM for software update policies and is moving Apple update management toward DDM.

### Software and patch management

All three solutions deploy applications and handle operating system updates.

Jamf Pro offers App Installers (Jamf App Catalog) for third-party macOS apps, custom package uploads, and Apps and Books (VPP) for iOS/iPadOS app distribution. OS updates can be enforced through MDM commands.

Intune deploys Windows applications through the Company Portal and manages Microsoft Store apps. macOS app deployment exists but with fewer options than Windows. Windows Update for Business handles OS patching.

Fleet provides Fleet-maintained apps and custom installer package installer uploads. Software deployment works consistently across macOS, Windows, and Linux. OS update enforcement uses the same policy model regardless of platform.

### Security and compliance

Jamf Pro manages FileVault encryption, Gatekeeper, and Apple security baselines. Advanced capabilities require purchasing Jamf Protect separately for EDR and threat detection.

Intune integrates with Microsoft Defender for endpoint protection, conditional access, and compliance policies. The full security stack requires Microsoft Defender licensing beyond base Intune.

Fleet includes security capabilities in the base product. Vulnerability detection pulls data from the National Vulnerability Database (NVD) to enable EPSS scoring, and the Known Exploited Vulnerabilities (KEV) catalog. File integrity monitoring and YARA-based threat detection are built in to osquery. CIS benchmark queries are publicly available and easily uploaded. SIEM integration for shipping log data is standard.

Fleet can deliver an arbitrary MDM command to a managed device without limitations. Custom MDM commands are not available in Jamf or Intune. MDM commands are limited to what’s made available via the Jamf Pro or Intune GUI.

## Single-platform vs. multi-platform fleet management

Jamf Pro is Apple-first. Jamf's broader portfolio includes Android enrollment, but organizations with Windows and Linux endpoints need additional tools.

Intune covers Windows, macOS, iOS, iPadOS, Android, Linux, and ChromeOS, but Windows management is emphasized. macOS management capabilities are less mature than Windows equivalents, and Linux/ChromeOS support has platform-specific limitations.

Fleet manages macOS, iOS, iPadOS, Windows, Linux, ChromeOS, and Android with consistent capabilities across platforms. The same queries work everywhere. The same policies apply regardless of operating system. IT teams learn one console for the entire device fleet rather than context-switching between tools.

Many organizations manage more than one operating system. Separate tools for each platform means separate training, separate policies, separate compliance evidence. Fleet provides device management that works across every platform from a single console, with security features included rather than requiring additional purchases.

## FAQ

### What is device management software for IT teams?

Device management solutions (i.e., device management software) centralizes reporting, inventory collection, configuration, security, and compliance for devices that are owned and managed by organizations. IT teams use device management solutions to manage laptops, desktops, servers and mobile devices.

### Can one device management solution handle Mac, Windows, and Linux devices?

Fleet and Intune manage macOS, Windows, Linux, iOS, iPadOS, ChromeOS, and Android from the same console. Jamf Pro is Apple-first with Android support, but Windows and Linux require additional tools. Intune supports multiple platforms but with uneven feature depth across operating systems.

### What factors matter most when comparing fleet management software?

Platform coverage determines whether you need one tool or several. Security and compliance features affect whether you need additional products. Automation capabilities (GitOps, APIs, scripting) impact how efficiently your team operates. Deployment model (cloud-only vs. self-hosted) matters for organizations with data residency requirements.

### How does Fleet differ from Jamf and Intune as device management software?

Fleet provides consistent multi-platform capabilities where Jamf Pro is Apple-first and Intune is Windows-centric. Fleet includes security features like vulnerability detection and file integrity monitoring in the base product rather than as add-ons. Fleet's osquery foundation enables SQL-based queries across the entire device fleet with near real-time reporting. [Try Fleet](https://fleetdm.com/try-fleet) to see multi-platform device management in action.

### How long does it take to roll out device management software?

Timelines depends on fleet size and configuration complexity. Fleet supports zero-touch enrollment through Apple Business Manager and Windows Autopilot for automated onboarding. Fleet also provides MDM migration workflows and professional services for organizations transitioning from other solutions. [Schedule a demo](https://fleetdm.com/contact) to discuss your rollout timeline.

<meta name="articleTitle" value="Fleet vs. Jamf vs. Intune: Comparing fleet management software">
<meta name="authorFullName" value="Brock Walters">
<meta name="authorGitHubUsername" value="nonpunctual">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-02-26">
<meta name="description" value="Compare the Fleet, Jamf Pro, and Microsoft Intune product offerings.">

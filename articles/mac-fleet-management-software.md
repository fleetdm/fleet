# Fleet vs. Jamf vs. Microsoft Intune: How to choose the right fleet management software

Managing a fleet of corporate devices means juggling inventory, configuration, security, and compliance across potentially thousands of endpoints. The tools IT teams use for this vary dramatically in platform coverage, deployment flexibility, and what's included versus sold separately.

This guide compares Fleet, Jamf Pro, and Microsoft Intune as fleet management software options, covering platform support, security capabilities, and automation features.

## Overview

Fleet is a cross-platform device management solution built on osquery that manages macOS, iOS, iPadOS, Windows, Linux, ChromeOS, and Android from a single console. Device data returns in under 30 seconds through osquery-backed telemetry. GitOps is native to Fleet's architecture. 

Teams store device policies in version control, review changes through standard code review processes, and deploy configurations automatically when commits merge. Every policy change has an audit trail showing who approved what and when.

Jamf Pro is a device management solution focused on Apple devices. It manages Mac, iPhone, iPad, and Apple TV using Apple's management frameworks. Deployment options include on-premises, cloud, and hybrid architectures. Identity management (Jamf Connect) and endpoint security (Jamf Protect) are separate products at additional cost. Jamf's broader portfolio also includes Android enrollment (Jamf for Mobile), but organizations with Windows and Linux endpoints need additional tools.

Microsoft Intune is part of the Microsoft Endpoint Manager suite, bundled with Microsoft 365 E3/E5 licensing. It supports Windows, macOS, iOS/iPadOS, Android, Linux, and ChromeOS (with varying depth across platforms). Organizations using Microsoft 365 E3/E5 licensing have Intune included, though full endpoint security requires additional Microsoft Defender licensing.

Intune integrates with Azure Active Directory and Microsoft Defender. Intune's Windows capabilities are more developed than its macOS and Linux support, which have fewer features and configuration options.

## Key differences in fleet management software

| Attribute | Fleet | Jamf Pro | Intune |
| ----- | ----- | ----- | ----- |
| Architecture | API-first, unified REST API, osquery data collection | GUI-first, multiple APIs | GUI-first, multiple APIs, Azure integration |
| OS Coverage | macOS, iOS, iPadOS, Windows, Linux, ChromeOS, Android | macOS, iOS, iPadOS, tvOS, visionOS, Android | Windows, macOS, iOS, iPadOS, Android, Linux, ChromeOS |
| Deployment model | Cloud or self-hosting with full feature parity | On-premises, cloud, or hybrid | Cloud-only |
| GitOps support | Native GitOps workflow management | Requires third-party tools | Requires third-party tools (IntuneCD) |
| Security features | Vulnerability detection, YARA rules, file integrity monitoring, CIS benchmarks included | Requires Jamf Protect purchase | Integrates with Microsoft Defender (separate license) |
| Device reporting | Sub-30-second osquery-backed telemetry | MDM push-based commands; inventory cadence varies | MDM push-based commands; inventory cadence varies |
| Queries | SQL-based queries across all devices | Extension attributes for inventory | Limited query capabilities |
| Pricing model | Per-device subscription, open-source option | Per-device subscription plus add-ons | Bundled with Microsoft 365 E3/E5 |

## Device management workflow comparisons

Fleet, Jamf, and Intune all handle core fleet management functions: enrollment, configuration, software deployment, and compliance. The differences show up in automation depth, platform consistency, and what requires additional purchases.

### Enrollment and provisioning

All three solutions support zero-touch deployment for their respective platforms. Apple devices enroll through Apple Business Manager. Windows devices use Windows Autopilot. All three can configure supervised settings that prevent users from removing management profiles.

Jamf Pro's PreStage enrollment configures Apple device onboarding, including signed package deployment during setup. Intune's Autopilot configures Windows device onboarding with comparable automation. Fleet supports both Apple Business Manager and Windows Autopilot, providing consistent zero-touch enrollment across a mixed device fleet.

### Configuration management

Jamf Pro uses Smart Groups and Static Groups to scope Configuration Profiles and policies. It uses extension attributes to collect custom inventory data, though these are limited compared to osquery's 300+ queryable data tables

Intune uses device groups and user groups for policy assignment, with conditional access policies tied to Azure AD. Configuration profiles work differently across Windows and macOS, with more mature controls on Windows.

Fleet uses Teams and Labels for scoping across all platforms. A label can be as simple as a manual tag or as sophisticated as a live query result—devices automatically gain or lose labels as their state changes, keeping policy assignments current without manual intervention. The same scoping model works identically whether targeting macOS, Windows, or Linux devices. Fleet's GitOps approach means configurations are version-controlled and auditable, with changes tracked through pull requests rather than GUI edits that lack audit trails.

Fleet supports Declarative Device Management (DDM) natively. Jamf Pro supports DDM, but availability of specific DDM-powered workflows can vary by hosting tier and subscription. Intune supports Apple DDM for software update policies and is moving Apple update management toward DDM.

### Software and patch management

All three solutions deploy applications and handle operating system updates.

Jamf Pro offers App Installers (Jamf App Catalog) for third-party macOS apps, plus Apps and Books (VPP) for iOS/iPadOS app distribution. OS updates can be enforced through MDM commands.

Intune deploys Windows applications through the Company Portal and manages Microsoft Store apps. macOS app deployment exists but with fewer options than Windows. Windows Update for Business handles OS patching.

Fleet provides Fleet-maintained apps alongside custom installer support. Software deployment works consistently across macOS, Windows, and Linux. OS update enforcement uses the same policy model regardless of platform.

### Security and compliance

Jamf Pro manages FileVault encryption, Gatekeeper, and Apple security baselines. Advanced capabilities require purchasing Jamf Protect separately for EDR and threat detection.

Intune integrates with Microsoft Defender for endpoint protection, conditional access, and compliance policies. The full security stack requires Microsoft Defender licensing beyond base Intune.

Fleet includes security capabilities in the base product. Vulnerability detection pulls from the National Vulnerability Database, EPSS scoring, and the Known Exploited Vulnerabilities catalog. File integrity monitoring and YARA-based threat detection are built in. CIS benchmark queries are publicly available. SIEM integration ships standard.

MDM command delivery works similarly across platforms. Fleet's advantage shows up in visibility: security teams can query any device attribute across the fleet and get results back fast enough to act on during an active incident, rather than working from stale inventory data.

## Single-platform vs. cross-platform fleet management

Jamf Pro is Apple-first. Jamf's broader portfolio includes Android enrollment, but organizations with Windows and Linux endpoints need additional tools.

Intune covers Windows, macOS, iOS, iPadOS, Android, Linux, and ChromeOS, but Windows management is notably more capable. macOS management capabilities are less mature than Windows equivalents, and Linux/ChromeOS support has platform-specific limitations.

Fleet manages macOS, iOS, iPadOS, Windows, Linux, ChromeOS, and Android with consistent capabilities across platforms. The same queries work everywhere. The same policies apply regardless of operating system. IT teams learn one console for the entire device fleet rather than context-switching between tools.

Many organizations manage more than one operating system. Separate tools for each platform means separate training, separate policies, separate compliance evidence. Fleet provides device management that works across every platform from a single console, with security features included rather than requiring additional purchases.

## FAQ

### What is fleet management software for IT teams?

Fleet management software centralizes inventory, configuration, security, and compliance for corporate devices. IT teams use it to manage laptops, desktops, and mobile devices from a single console rather than touching each device individually. This is device fleet management, distinct from vehicle fleet management used in logistics.

### Can one device management solution handle Mac, Windows, and Linux devices?

Fleet manages macOS, Windows, Linux, iOS, iPadOS, ChromeOS, and Android from the same console with consistent capabilities across platforms. Jamf Pro is Apple-first with Android support through Jamf's broader portfolio, but Windows and Linux require additional tools. Intune supports multiple platforms but with uneven feature depth across operating systems.

### What factors matter most when comparing fleet management software?

Platform coverage determines whether you need one tool or several. Security and compliance features affect whether you need additional products. Automation capabilities (GitOps, APIs, scripting) affect how efficiently your team operates. Deployment model (cloud-only vs. self-hosted) matters for organizations with data residency requirements.

### How does Fleet differ from Jamf and Intune as fleet management software?

Fleet provides consistent cross-platform capabilities where Jamf Pro is Apple-first and Intune is Windows-centric. Fleet includes security features like vulnerability detection and file integrity monitoring in the base product rather than as add-ons. Fleet's osquery foundation enables SQL-based queries across the entire device fleet with sub-30-second reporting. [Try Fleet](https://fleetdm.com/try-fleet) to see cross-platform fleet management in action.

### How long does it take to roll out fleet management software across an entire device fleet?

Timeline depends on fleet size and configuration complexity. Fleet supports zero-touch enrollment through Apple Business Manager and Windows Autopilot for automated onboarding. Fleet also provides MDM migration workflows and professional services for organizations transitioning from other solutions. [Schedule a demo](https://fleetdm.com/contact) to discuss your rollout timeline.

<meta name="articleTitle" value="Fleet vs. Jamf vs. Intune: Comparing fleet management software">
<meta name="authorFullName" value="Brock Walters">
<meta name="authorGitHubUsername" value="nonpunctual">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-01-14">
<meta name="description" value="Compare Fleet, Jamf Pro, and Microsoft Intune for device fleet management. Platform coverage, security features, GitOps support, and pricing differences explained.">

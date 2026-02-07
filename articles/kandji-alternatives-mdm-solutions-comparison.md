# Alternatives to Iru: Fleet vs Kandji vs Jamf Pro

Whether you're hitting cloud-only limitations, managing devices beyond Apple, or looking for more predictable pricing, this guide compares Fleet, Kandji (now Iru), and Jamf Pro to help you find the right fit.

## Overview

Fleet is an open-source device management platform with support for macOS, iOS, iPadOS, Windows, Linux, Android, and ChromeOS. Fleet can be self-hosted or run as a managed cloud service, with identical features in both deployment models. Built on osquery, Fleet provides sub-30-second device reporting, native GitOps workflow management, and includes security capabilities like vulnerability detection, file integrity monitoring, and SIEM integration within its core product.

Kandji started as an Apple-focused MDM using Blueprint-based workflows and pre-built Library items. In October 2025, the company rebranded as Iru and added Windows and Android support. Iru operates exclusively as a cloud platform with no self-hosting option and uses a proprietary agent.

Jamf Pro provides Apple device management for macOS, iOS, iPadOS, tvOS, and visionOS. Jamf's product line requires separate purchases for advanced capabilities: Jamf Protect for endpoint security, Jamf Connect for identity management, and Jamf Executive Threat Protection for threat detection. Jamf offers cloud and self-hosted deployment, though the company encourages cloud adoption.

## Key differences

| Attribute | Fleet | Iru | Jamf Pro |
| ----- | ----- | ----- | ----- |
| Architecture | API-first, unified REST API, osquery-based data collection | Proprietary agent, GUI-based | GUI-first, multiple APIs |
| Source model | Open-core, public codebase | Proprietary, closed source | Proprietary, closed source |
| Console management | GUI or GitOps with YAML configuration | GUI only | GUI-first, GitOps requires third-party tools |
| Deployment options | Self-hosted or managed cloud (identical features) | Cloud-only | Cloud or self-hosted |
| Platform support | macOS, iOS, iPadOS, Windows, Linux, Android, ChromeOS | macOS, iOS, iPadOS, tvOS, Windows, Android (Windows/Android newly added October 2025\) | macOS, iOS, iPadOS, tvOS, visionOS (Apple only) |
| Device reporting speed | Sub-30-second reporting | Standard sync intervals | Standard sync intervals |
| Declarative Device Management | Full DDM support | DDM support | DDM supported; availability of specific DDM-powered workflows can vary by plan or hosting |
| GitOps support | Native | Not available | Requires third-party projects |
| MDM migration | Native support | Available | Professional services only |
| Security features | Vulnerability detection, queries, YARA rules, file integrity monitoring, SIEM integration included | EDR and vulnerability management are separate products at extra cost | Advanced security requires Jamf Protect (separate purchase) |

## Device management workflow comparisons

### Enrollment and provisioning

Fleet, Iru, and Jamf Pro all support Apple Business Manager and School Manager for zero-touch deployment. For Apple devices enrolled via Automated Device Enrollment (ABM/ASM), admins can optionally prevent users from removing the MDM enrollment profile.

Fleet also supports Windows Autopilot for zero-touch Windows enrollment and provides native MDM migration workflows for organizations switching from other MDM solutions. Iru offers migration assistance. Jamf provides migration only through professional services engagements.

### Configuration management

All three platforms support delivery of MDM Configuration Profiles and provide mechanisms for scoping configurations to specific device groups.

Iru organizes configurations through Blueprints and a Library of pre-built items. Jamf uses Smart Groups and Static Groups for scoping and includes Configuration Profile templates.

Fleet scopes Configuration Profiles using Teams for organizational boundaries and Labels for device grouping. Labels update dynamically: a device that installs prohibited software or falls out of compliance automatically moves into the appropriate label, triggering remediation policies without admin intervention. Fleet directs admins to iMazing Profile Creator for building Configuration Profiles. 

GitOps workflows let teams define fleet configurations in YAML files, enforce peer review through pull requests, and trigger automated deployments when changes merge. The Git history becomes the audit log. Fleet validates Configuration Profile delivery through osquery independently from MDM reporting, providing verification that profiles are actually applied rather than just sent.

### Software management

All three platforms support custom software package uploads and scripting for installation automation.

Iru provides an Auto Apps catalog and Apps and Books distribution for volume purchasing. Jamf Pro provides an App Catalog and Apps and Books distribution with scoping through Smart and Static Groups. Self-service installation uses the Jamf Self Service app.

Fleet provides software management through Fleet-maintained apps and Apps and Books distribution. Self-service installation uses Fleet Desktop.

### Security and compliance

All three platforms support FileVault disk encryption management, Gatekeeper, and other Apple security settings through MDM.

Iru's device management product handles baseline security features, but EDR and Vulnerability Management require additional products at extra cost. The Iru rebrand combines these previously separate products into a single platform, increasing per-device costs compared to Iru's original MDM-only pricing.

Jamf Pro handles baseline Apple security settings, but advanced capabilities require separate products: Jamf Protect for endpoint detection, Jamf Executive Threat Protection for threat response. These separate purchases increase total cost.

Fleet includes vulnerability detection, osquery-based queries and policies, file integrity monitoring, YARA-based malware detection, and SIEM integration within the core product. These capabilities don't require separate purchases or additional per-device fees. Fleet's osquery foundation provides visibility into software inventories, running processes, network configurations, file system events, and hardware details that competitors don't expose without add-on products.

### Cross-platform coverage and deployment flexibility

Iru added Windows and Android management in October 2025 with its Iru rebrand. Iru remains cloud-only with no self-hosting option.

Jamf Pro manages Apple devices only. Organizations with mixed fleets must run Jamf alongside separate Windows and Linux management tools.

Fleet manages macOS, iOS, iPadOS, Windows, Linux, Android, and ChromeOS from a single console, with consistent query and policy capabilities across operating systems. Teams can deploy Fleet as a managed cloud service or self-host on their own infrastructure. Both deployment models use identical code and provide identical features, addressing data residency requirements and eliminating vendor lock-in.

### Migration considerations

Switching MDM providers requires planning, but recent Apple platform changes have simplified the process.

* **Apple's 2025 migration workflow:** Apple introduced ABM/ASM device-management migration (requires iOS/iPadOS/macOS 26 and ADE-owned devices) that can move devices to a new MDM with prompts and reenrollment. In some iPhone/iPad cases, managed apps and data can be preserved if the destination MDM delivers apps before DeviceConfigured.  
* **macOS migrations involve end-user interaction:** Apple's migration flow can be user-initiated or enforced by the organization, but it still involves a restart and user-facing prompts (including a nondismissible full-screen prompt on Mac). Organizations typically plan 1-2 months for mid-size deployments, with actual enrollment taking 1-2 weeks once users begin the process.  
* **iOS and iPadOS migrations:** For ADE-owned iPhone/iPad on iOS/iPadOS 26, ABM migration can move devices to a new MDM without a factory reset in many cases. Return to Service is still an erase-and-reenroll workflow, though on iOS/iPadOS 26 it can preserve managed apps (user data is erased).  
* **Apps and Books managed distribution:** Apps and Books depends on the location content token used by your MDM. During migration you typically remove the token from the old MDM and upload it to the new one. App access may continue for a grace period, and you'll need to reconcile assignments in the destination MDM. Fleet, Iru, and Jamf Pro all support Apps and Books integration.

Fleet provides native migration workflows for macOS and Windows that automate much of the transition process. For organizations evaluating Iru alternatives, Fleet's migration support and professional services can help plan and execute the switch with minimal disruption to end users.

## FAQ

### What should I look for in alternatives to Iru?

Evaluate platform coverage and maturity across the operating systems you manage. Consider deployment flexibility if you have data residency requirements or prefer self-hosting (Iru is cloud-only). Look at automation capabilities: Fleet supports native GitOps workflows and a unified REST API. Check whether security features like vulnerability detection and SIEM integration require separate product purchases or are included in base pricing.

### Does Fleet provide Apple device management at parity with Iru?

Fleet provides Apple device management at parity with Iru for zero-touch enrollment through Apple Business Manager or School Manager, Configuration Profile delivery, MDM commands, Declarative Device Management, software deployment, script execution, and scoping controls. Fleet's osquery foundation adds capabilities Iru doesn't offer, including SQL-based device queries, file integrity monitoring, and independent verification of configuration state.

### How does cross-platform support compare across these three platforms?

Fleet supports macOS, iOS, iPadOS, Windows, Linux, Android, and ChromeOS with consistent query and policy capabilities across all operating systems. Iru added Windows and Android in October 2025\. Jamf Pro manages Apple devices only and doesn't provide Windows or Linux management.

### What pricing differences should I consider?

Iru pricing isn't publicly listed. Jamf's public pricing is published for packaged offerings (e.g., Jamf for Mac, Jamf for Mobile), rather than a simple standalone Jamf Pro list price.Fleet's core product includes vulnerability scanning that identifies outdated software with known security issues, custom detection through osquery queries, file integrity monitoring for sensitive system paths, YARA-based malware identification, and direct integration with SIEM platforms. These aren't upsells or separate SKUs.

### How long does migration to a new MDM take?

Timelines depend on fleet size and complexity. Fleet provides native migration workflows for all computer platforms and offers professional services to assist with transitions. Mobile device MDM migrations are more constrained due to platform vendor limitations. [Schedule a demo](https://fleetdm.com/contact) to discuss implementation timelines for your environment.

<meta name="articleTitle" value="Alternatives to Iru (Kandji): Fleet vs Iru vs Jamf Pro">
<meta name="authorFullName" value="Brock Walters">
<meta name="authorGitHubUsername" value="nonpunctual">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-01-16">
<meta name="description" value="Compare Fleet, Iru (Kandji), and Jamf Pro for cross-platform MDM, security features, and deployment options.">

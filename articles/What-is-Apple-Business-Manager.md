# What is Apple Business Manager? A complete guide

This guide explains what Apple Business Manager (ABM) does and its key features.

## Apple Business Manager defined

Apple Business Manager (ABM) is Apple's free service for tracking the provenance of an organization's purchased devices from Apple or from authorized Apple resellers. By validating institutional ownership of Apple devices, ABM allows organizations to:

- Connect their MDM servers to ABM to enable Automated Device Enrollment (ADE) for zero-touch enrollment workflows
- Control licensed software and content distribution via Apps and Books management
- Administer Managed Apple Accounts

The backend of the ABM infrastructure handles connecting institutionally-owned devices to Apple activation servers after first boot. Devices negotiate with ABM and become associated to an organization's MDM server to receive an enrollment profile. When a device is enrolled, users are walked through a partially or fully automated Setup Assistant process, usually followed by a customized provisioning workflow. The "Out of the Box" (OOB) experience for a new employee is easy to understand and hassle-free, resulting in a completely configured, fully-managed device that is ready for work without manual or IT intervention.

Apple consolidated ABM in 2018 from two previously separate programs: the Device Enrollment Program (DEP) for automated enrollment and the Volume Purchase Program (VPP) for app licensing. The consolidation eliminated operational silos of managing device enrollment in one system and app distribution in another.

ABM is a single-portal for:

- Device enrollment automation (ADE)
- Software and content purchasing (Apps and Books)
- Managed Apple Account provisioning

integrated with your chosen mobile device management (MDM) solution for complete control of your devices across the Apple platform.

## Is Apple Business Manager an MDM?

ABM alone is not a substitute for an MDM solution. MDM servers connected to ABM are responsible for performing management actions on devices after enrollment.

Pairing ABM with an MDM solution results in a comprehensive management system where automated enrollment feeds devices to the MDM server. The MDM server deploys MDM commands and configuration profiles to enforce settings and controls post-enrollment. 

MDM is possible without ABM, but manual enrollment can create bottlenecks that slow provisioning and increase IT overhead. ABM enables ADE which allows for Apple devices to be fully managed and supervised, giving organizations access to the full range of Apple device management capabilities.

ABM can be integrated with all major MDM solutions, including [Fleet](http://fleetdm.com). This flexibility allows organizations to evaluate MDM vendors based on specific requirements or features like REST API capabilities, cross-platform support, data collection, security compliance, remediation capabilities, GitOps automation, and self-hosting options. 

If your current MDM fails to meet your needs, you can switch MDM vendors without having to create a new ABM instance. At WWDC 2025 Apple introduced [Managed Device Migration](https://fleetdm.com/announcements/mdm-just-got-better) making the move from your current MDM vendor to any other easier than ever.

### What is Apple Business Essentials?

Apple Business Essentials (ABE) is Apple's MDM solution for small businesses that bundles enrollment and settings enforcement into a single paid subscription ($2.99 to $24.99 per device per month). Unlike ABM which requires a separate MDM, ABE packages ABM functionality with built-in MDM, 24/7 support, and iCloud storage in a simplified offering.

ABE works best for organizations with specific characteristics and constraints:

* Organizations with fewer than 500 employees  
* US-based operations that don't require multi-region deployment  
* Apple-only device fleets without Windows, Linux, or cross-platform management needs  
* Limited IT staff who benefit from simplified administration and built-in support  
* Basic security requirements that don't demand advanced compliance

ABE lacks advanced controls like conditional access, dynamic grouping, and sophisticated automation, while app deployment is via App Store software distribution custom package installation capabilities.

Complex organizations with strict management requirements should consider using ABM paired with a third-party MDM solution for greater flexibility and capabilities.

## What are the key features of Apple Business Manager?

ABM has three core capabilities that work together to automate device provisioning at scale: Automated Device Enrollment (ADE), volume purchasing for Apps and Books, and identity management through Managed Apple Accounts. 

### Automated Device Enrollment

Automated Device Enrollment (ADE) streamlines MDM enrollment by handling initial device setup through Apple's activation infrastructure.

### Volume purchasing and app distribution

ABM provides bulk app and content purchases with remote distribution that doesn't require employees to use personal Apple Accounts on-device. This gives you complete license management with remote deployment capabilities even if users or your organization has disabled the App Store on managed devices.

Admins can push app updates and revoke compromised app access during security incidents across their entire fleet without requiring physical access to devices.

When employees leave, licenses are maintained by your organization rather than leaving with them. This means licenses are available for reassignment to current employees and licensing costs can be more easily managed. For organizations with proprietary software, ABM allows custom, in-house app distribution directly to managed devices without requiring App Store distribution or public availability.

### Managed Apple Accounts

Managed Apple Accounts allow organizations to separate work and personal identities. ABM People Managers can control all identities created within an organization's domain. This solves the problem of employees controlling Apple Accounts created with an organizational identity but intended for personal use outside the organization's control.

Managed Apple Accounts also enable role-based access control within ABM. This allows an organization to delegate specific ABM administrative responsibilities. A Content Manager, for example, can distribute apps without accessing device configurations. A People Manager can provision Managed Apple Accounts without access to change MDM assignments.

Role-based access control typically follows the security principle of "least privilege" while distributing workloads across your IT team, preventing any single ABM administrator from having unnecessary access to sensitive configurations.

## Benefits of Apple Business Manager

ABM eliminates manual device setup, reduces administrative overhead through automation, and provides vendor flexibility at zero platform cost.

* **Deployment efficiency:** Automated enrollment delivers devices ready for work on first boot, allowing IT teams to shift their focus from repetitive device configuration to designing policies and maintaining infrastructure.

* **Operational scale:** ABM supports organizational growth without requiring architectural changes, keeping your provisioning process consistent as your device fleet expands from dozens to thousands of devices.

* **Cost savings:** ABM itself costs nothing, and volume app purchasing keeps software and content license expenses predictable.

* **Security and compliance:** ABM enables good security posture at many different levels: 

  - Device supervision via ADE enforce security settings from first boot and enables remote lock/wipe capabilities
  - Managed Apple Accounts can help to keep work data separate from personal information
  - Role-based access controls in ABM allow organizations to engage in best practices 

* **Vendor flexibility:** ABM integrates with any MDM solution, preventing vendor lock-in. Select the MDM solution that fits your organization best and switch MDM vendors without rebuilding your enrollment infrastructure via Managed Device Migration.

## Who should use Apple Business Manager?

Small US-only teams with fewer than 500 employees and basic security needs might be able to use ABE. Organizations managing Apple devices at scale should use ABM paired with a third-party MDM solution, such as [Fleet](https://fleetdm.com/device-management).

Value emerges for enterprises with distributed teams, international operations, or those planning to exceed 500 employees. Large enterprises with multi-location operations will appreciate ABM's global availability and unlimited scale since Apple Business Essentials has strict size and geographic limits.

Fleet pairs well with ABM. Fleet's MDM features are built on top of [osquery](https://fleetdm.com/guides/osquery-a-tool-to-easily-ask-questions-about-operating-systems). Fleet provides deep endpoint visibility through 300+ queryable data tables and delivers device reporting in under 30 seconds. Its cross-platform support extends beyond the Apple ecosystem to Windows, Linux, ChromeOS, and Android devices. For organizations with data residency requirements, Fleet offers both hosted, cloud-managed, and self-hosted server deployment options, while native GitOps and API-first design integrate with the modern, infrastructure-as-code practices large enterprises are adopting to thrive.

## What about Apple devices in education and the public sector?

Educational institutions should use Apple School Manager (ASM) instead of ABM. ASM has all the features of ABM with additional features designed for K-12 and higher education like Shared iPad management, student information management tools, and student account provisioning. Schools don't need both portals and only need to pair ASM with a third-party MDM solution like [Fleet](http://fleetdm.com).

ABM works with all MDM solutions and provides detailed procurement, enrollment and compliance data. Government and public sector agencies managing Apple devices that:

- Must meet strict vendor flexibility and audit requirements
- Operate under strict procurement and regulatory frameworks

should require ABM as part of their technology stack.

Fleet helps these organizations [meet compliance requirements](https://fleetdm.com/securing/get-and-stay-compliant-across-your-devices-with-fleet) through automated vulnerability detection, policy enforcement, continuous monitoring, and deployment flexibility with both cloud-hosted and self-hosted options that address data residency requirements.

## Pairing ABM with the right MDM

ABM provides the enrollment infrastructure that makes automated Apple device provisioning possible. Pairing ABM's free platform with your chosen MDM solution eliminates manual device setup and gives you a flexible foundation that scales with your organization.

For comprehensive device management across Mac, iPhone, iPad, Windows, and Linux, Fleet provides open-source MDM that integrates with ABM. Once ABM handles enrollment, Fleet manages your devices with an API-first architecture that supports GitOps workflows and configuration as code. [Schedule a Fleet demo](https://fleetdm.com/contact) to explore how ABM and Fleet work together.
<meta name="articleTitle" value="What is Apple Business Manager? A complete guide">
<meta name="authorFullName" value="Brock Walters">
<meta name="authorGitHubUsername" value="nonpunctual">
<meta name="category" value="articles">
<meta name="publishedOn" value="2025-11-21">
<meta name="description" value="Learn what Apple Business Manager is, how it works with MDM, its key features, benefits, and who should use it.">

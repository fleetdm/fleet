# Alternatives to Jamf for cross-platform device management

Organizations often look for Jamf alternatives when their device fleet expands beyond Apple. Jamf focuses on macOS and iOS, so managing Windows, Linux, or ChromeOS devices usually means adding more tools and dealing with visibility gaps across the fleet. This guide compares Fleet, JumpCloud, and Omnissa Workspace ONE UEM as cross-platform alternatives to Jamf.

## Overview

Fleet expands on Jamf's Apple device management capabilities while adding support for Windows, Linux, ChromeOS, and Android. Fleet includes Apple Business Manager integration, Declarative Device Management, and configuration profiles, plus GitOps workflows, sub-30-second device reporting through osquery, and full self-hosting capability. Fleet integrates with identity providers including Okta, Microsoft Entra ID, and Google Workspace.

JumpCloud combines cloud directory services with cross-platform device management, unifying identity and endpoint control in a single platform. JumpCloud supports Windows, macOS, Linux, iOS, iPadOS, and Android, with MDM capabilities alongside SSO, LDAP, and RADIUS services.

Workspace ONE is Omnissa's enterprise UEM platform, offering cloud, on-premises, and hybrid deployment options. Workspace ONE manages Windows, macOS, Linux, iOS, iPadOS, Android, and ChromeOS, with additional capabilities for rugged devices and virtual desktops through the broader Omnissa ecosystem.

## Key differences

| Attribute | Fleet | JumpCloud | Omnissa Workspace ONE UEM |
| ----- | ----- | ----- | ----- |
| Platform support | Windows, macOS, Linux, iOS, iPadOS, Android, ChromeOS | Windows, macOS, Linux, iOS, iPadOS, Android | Windows, macOS, Linux, iOS, iPadOS, Android, ChromeOS |
| GitOps support | Native GitOps workflow management | No native GitOps | No native GitOps |
| Self-hosting | Full self-hosting support with no restrictions | Cloud-based primary | On-premises requires professional services |
| Device reporting | Sub-30-second reporting | Standard sync cycles | Standard sync cycles |
| Queries | SQL-based queries across all devices via osquery | Policy-based monitoring | Reporting capabilities |
| Open-source | Open-source foundation | Proprietary | Proprietary |

## Device management workflow comparisons

### Enrollment and provisioning

All three solutions support zero-touch enrollment through Apple Business Manager and Windows Autopilot, letting new hires power on devices and start working without IT manually configuring each machine. Workspace ONE adds Android Enterprise and Samsung Knox for Android zero-touch deployments.

Fleet includes two enrollment capabilities the others lack: MDM migration and GitOps. MDM migration lets you transition devices from existing management tools without re-imaging, so end-users don't need to visit IT to switch solutions. GitOps support means enrollment configurations live in version control with full audit trails.

### Configuration management

Jamf admins use configuration profiles and Smart Groups to target policies to specific devices. All three alternatives provide equivalent targeting: Fleet uses Labels and Teams, JumpCloud uses device groups, and Workspace ONE uses assignment groups. JumpCloud and Workspace ONE deploy configurations through GUI-based workflows across their respective platforms.

Fleet matches Jamf's Apple configuration capabilities, including support for Apple's [Declarative Device Management (DDM)](https://fleetdm.com/articles/declarative-device-management-a-primer) framework for macOS and iOS. Workspace ONE also supports DDM on its Modern SaaS Architecture, while JumpCloud does not currently offer DDM support.

Where Fleet goes beyond Jamf is in verification and automation. Native GitOps workflows let teams manage configurations as code with full version control, so policy changes go through review processes and can roll back when something breaks. The osquery foundation lets you verify that configurations actually applied through SQL-based queries, rather than assuming success after sending a profile. Fleet also provides device remediation workflows that automatically fix configuration drift and maintenance windows for scheduling changes during off-hours, capabilities that Jamf, JumpCloud, and Workspace ONE don't offer.

### Software management

Organizations moving from Jamf need software deployment and patching that works beyond Apple devices. Fleet provides [software deployment and patching](https://fleetdm.com/software-management) across all supported platforms, with GitOps-managed deployment configurations that let you track changes and roll back when needed.

JumpCloud includes software deployment and an app catalog for macOS, with on-demand script execution across platforms. Workspace ONE offers application deployment and patch management through its app catalog.

Fleet's GitOps approach means software deployments go through the same review and approval workflows as infrastructure changes, with full audit trails for compliance reporting.

### Security and compliance

All three solutions provide security monitoring and compliance capabilities, but they take different approaches. JumpCloud focuses on identity-based security through multi-factor authentication, conditional access policies, and SaaS management for application visibility. Workspace ONE offers compliance policies with Omnissa Intelligence integration for analytics, plus Workspace ONE Vulnerability Defense for vulnerability management through CrowdStrike integration.

Fleet's osquery foundation provides SQL-based queries for deep visibility into device security state across all platforms. The key differentiators: sub-30-second reporting for rapid incident response (compared to 1-6 hour sync delays common with other MDM tools), policies with compliance scoring that track organizational security posture over time, and built-in vulnerability detection through software inventory analysis. Fleet also supports YARA-based malware detection and file integrity monitoring for organizations that need forensic capabilities without adding another tool.

### API and integration capabilities

Organizations expanding beyond Apple-focused management need APIs that cover all their devices equally. Fleet, JumpCloud, and Workspace ONE all provide REST APIs for automation. Fleet's API covers all platforms through a single endpoint, with native GitOps support for managing configurations as code. JumpCloud offers API access alongside Cloud LDAP and RADIUS services. Workspace ONE provides REST APIs with additional endpoints for Intelligence analytics.

Fleet's osquery foundation adds a capability the others lack: you can query devices to verify configurations actually applied, rather than assuming success after sending a command.

## Pricing and licensing

Organizations evaluating Jamf alternatives often find that cross-platform management changes the cost equation. Jamf charges per device for Apple management, but adding Windows and Linux devices typically means paying for additional tools.

JumpCloud offers a free tier for up to 10 users and 10 devices, with paid packages starting at $9 per user per month for device management. SSO and identity features are available in separate or bundled packages.

Workspace ONE uses tiered pricing starting at $5.25 per device per month for UEM Essentials. Advanced features like Omnissa Intelligence for analytics and automation require the Enterprise edition or add-on purchases.

Fleet's open-source foundation means you can evaluate Fleet fully before committing to paid tiers. Premium features are available through per-device licensing with predictable costs across all platforms. Fleet integrates with identity providers at no additional cost.

## Cross-platform device management with Fleet

Organizations moving beyond Jamf typically need more than just Apple device management. Fleet provides the same Apple device management capabilities you expect from Jamf, including Apple Business Manager integration, Declarative Device Management, and configuration profiles, while extending those capabilities equally across all supported operating systems.

The osquery foundation delivers sub-30-second device reporting across all platforms, giving your security and IT teams the speed they need during incident investigations. Native GitOps workflows let you manage device configurations as code, and the open-source foundation means you can inspect exactly how monitoring works.

[Try Fleet](https://fleetdm.com/try-fleet) to see how cross-platform device management works with your existing infrastructure.

## FAQ

### What's the main difference between an identity-first solution and a device management-first solution?

Identity-first solutions prioritize authentication and directory services, with device management added as a secondary feature. Device management-first solutions like Fleet prioritize endpoint control, configuration, and security visibility with complete feature parity. Fleet integrates with leading identity providers including Okta, Azure AD, and Google Workspace, so you don't have to choose between identity management and deep device visibility.

### How does open-source device management compare to proprietary solutions?

Open-source solutions like Fleet let you inspect code, understand data handling practices, and modify functionality according to your specific needs. The open-source model also enables community contributions that accelerate feature development and security improvements. Fleet's self-hosting capability means you can maintain complete data sovereignty, keeping all device telemetry within your own infrastructure.

### Can I query device state in real-time across different operating systems?

Fleet provides SQL-based queries across all devices with sub-30-second reporting cycles. During a security investigation, your team can immediately verify whether a specific vulnerability exists across your entire fleet, identify which devices have a particular application installed, or confirm compliance status without waiting for scheduled sync cycles. Fleet also supports maintenance windows for scheduling updates during off-hours.

### What support is available for migrating from an existing device management solution to Fleet?

Fleet supports MDM migration from existing solutions without requiring device re-imaging, preserving device enrollment state so end-users experience minimal disruption. Most organizations complete migrations in phases, starting with a pilot group before expanding fleet-wide. [Schedule a demo](https://fleetdm.com/demo) to learn how migration works for your environment.

### How do maintenance and configuration updates work with GitOps workflows?

Fleet provides native GitOps workflow support, letting your team manage device configurations as code in version-controlled repositories. Configuration changes go through standard review processes before deployment, with full audit trails and rollback capabilities. Teams can define Fleet configurations in YAML files, submit changes through pull requests, and automatically deploy approved configurations. [Schedule a demo](https://fleetdm.com/demo) to see Fleet's GitOps workflows in action.

<meta name="articleTitle" value="Alternatives to Jamf for cross-platform device management">
<meta name="authorFullName" value="Brock Walters">
<meta name="authorGitHubUsername" value="nonpunctual">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-01-29">
<meta name="description" value="Compare Fleet, JumpCloud, and Workspace ONE as Jamf alternatives for managing macOS, Windows, Linux, and mobile devices.">

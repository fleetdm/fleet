## How Jamf approaches Apple device management

Jamf Pro targets large, complex Apple environments where customization requirements outweigh ease-of-use considerations. The solution extends Apple's native MDM protocol through a binary framework requiring root privileges, giving it capabilities beyond pure MDM: custom script execution, advanced policy enforcement, and granular inventory collection through extension attributes (custom inventory fields beyond standard system information).

This flexibility shows up most clearly in how Jamf handles device segmentation. Smart Groups dynamically organize devices based on hardware specifications, installed software, user attributes, and custom criteria. Policies then target these groups with precise configurations.

The trade-off for this flexibility is complexity. Implementation typically involves dedicated Mac admin staff comfortable with shell scripting, API integration, and macOS architecture. Organizations often hire consultants to handle deployment, and time-to-productivity runs longer compared to automation-first alternatives.

## How Iru (Kandji) approaches Apple device management

Iru (Kandji) targets organizations prioritizing rapid onboarding with pre-built automations and continuous remediation, trading flexibility for ease of deployment. Where Jamf emphasizes customization, Iru (Kandji) emphasizes consistency and self-healing.

The architecture combines Apple's MDM protocol with a proprietary agent that includes automatic recovery mechanisms. If a device checks in via MDM within 7 days but the agent hasn't communicated, Iru (Kandji) automatically triggers agent reinstallation, preventing the "lost sheep" phenomenon where devices lose management connectivity.

Blueprints organize devices into policy groups with pre-built automations and compliance templates. Auto Apps provides automated patch management with reduced lag time between vendor releases and deployment. Liftoff handles zero-touch onboarding, converting new devices into enterprise-ready endpoints on first boot.

Iru (Kandji)'s proprietary agent also integrates endpoint detection and response capabilities, providing threat detection and real-time vulnerability scanning as optional add-ons priced per device. When purchased alongside base MDM licensing, this consolidates endpoint software footprint. The solution suits lean IT teams with standard compliance frameworks and no need for highly customized workflows. However, if you require deep customization or complex integration scenarios, you may find the Iru (Kandji) opinionated approach limiting.

## How does Jamf compare to Iru (Kandji) in practice?

Five key dimensions separate these solutions in ways that affect daily work. Understanding where each solution excels helps match capabilities to your operational requirements and team structure.

### Enrollment and onboarding

Both solutions integrate with Apple Business Manager for automated device enrollment, but the experience diverges after initial MDM connection.

Jamf's PreStage Enrollment system provides extensive customization options, supporting deployment of signed packages during enrollment and sophisticated onboarding workflows including department-specific configurations and role-based application deployment.

Iru (Kandji) Liftoff prioritizes simplicity. The guided onboarding experience works well for standardized deployments where all users receive similar baseline configurations, converting devices from factory state to production-ready without IT intervention.

### Automation, scripting, and API support

This dimension creates the starkest difference between solutions. Jamf Pro supports extensive scripting through custom script deployment, parameter passing, and execution triggered by policy, self-service, or webhook events. The API architecture supports both Classic and Jamf Pro API endpoints with complete solution coverage. Webhooks enable real-time, event-driven capabilities.

Iru (Kandji) takes a no-code approach through pre-built automations. The API provides programmatic access with token-based security and granular role management, though webhook and eventing capabilities are less extensive and less publicly documented than Jamf's mature Events API ecosystem.

### Security and compliance capabilities

Both solutions offer compliance and security features through different approaches. Jamf Pro offers compliance benchmarks derived from government security standards, implementing [CIS Benchmarks](https://www.cisecurity.org/cis-benchmarks) and [STIG baselines](https://www.cyber.mil/stigs/) with explicit mapping to NIST 800-53 controls. The [Jamf Compliance Editor](https://trusted.jamf.com/docs/establishing-compliance-baselines) is an application that allows admins to build controls tied to specific framework requirements for macOS, iOS, iPadOS, and visionOS. Once controls are created on an admin’s computer they can be uploaded to the Jamf console for use.

Iru (Kandji) provides automated security controls designed to meet compliance requirements across frameworks like SOC 2 and HIPAA. The solution emphasizes continuous monitoring with automated remediation when devices drift from compliant states. Compliance reports generate automatically but focus on proving compliance rather than detailed control citation mapping.

For security tooling, Jamf follows a modular approach. Jamf Protect is an Endpoint, Detection & Response (EDR) based on the macOS Endpoint Security framework. It integrates with SIEM solutions like Splunk. Iru (Kandji) bundles EDR and vulnerability management into its proprietary agent architecture when both capabilities are purchased.

The choice often comes down to your compliance framework requirements. If auditors expect detailed control mapping and customization for unique regulatory interpretations, Jamf's explicit framework references provide clearer audit trails. If continuous automated remediation matters more, the Iru (Kandji) approach works well.

### Support and community resources

Jamf Nation provides an active community forum where Mac admins share scripts, troubleshoot configurations, and discuss implementation patterns. The ecosystem includes third-party tools, consulting services, and extensive documentation developed over 20+ years.

Iru (Kandji) offers direct support focused on implementing pre-built features. The vendor provides implementation guidance, regular updates, and customer success engagement, though the community is smaller and less established than Jamf's decades-old ecosystem.

## Key differences at a glance

The architectural differences between Jamf and Iru (Kandji) translate into practical trade-offs that affect daily operations.
| Dimension | Jamf Pro | Iru (Kandji) |
|---|---|---|
| Team technical depth | Teams comfortable with shell scripting, API integration, and custom automation development | IT generalists managing multiple responsibilities beyond Mac administration |
| Customization needs | Complex regulatory requirements; custom integrations via API workflows and scripts | Standard compliance frameworks with no need for highly customized workflows |
| Compliance approach | Framework templates from CIS, NIST, STIG standards; audit trails with control citation mapping | One-click compliance templates for SOC 2, HIPAA; automated controls with continuous remediation |
| Security tooling | Optional Jamf Protect for EDR; Jamf Connect for identity. 'Jamf for Mac' bundles Pro + Connect + Protect; documented SIEM integrations | EDR and vulnerability management as add-ons (priced per device) via single agent; consolidates endpoint tools when purchased |
| Device ecosystem | Apple and Android support | Cross-platform solution with support for Apple, Windows, and Android |
| Implementation timeline | Longer time-to-productivity; often requires consultants | Faster deployment through pre-built automations and Liftoff onboarding |

## What are the limitations of MDM solutions like Jamf and Iru (Kandji)?

Mixed environment challenges emerge immediately when you manage Windows computers alongside Macs, maintain Linux servers, or support ChromeOS devices. Many organizations end up with separate tools for each platform: Intune or SCCM for Windows, Ansible or Puppet for Linux, and Google Workspace Admin Console for Chromebooks. Each requires separate admin interfaces, policy definitions, and compliance reporting.

Tool sprawl compounds costs beyond licensing. The overhead includes:

- Training burden: Teams need expertise across multiple platforms.
- Maintenance overhead: Each tool requires ongoing updates and configuration management.
- Policy fragmentation: Configuration policies don't translate between systems, creating security gaps where FileVault is mandated on macOS while BitLocker remains unmanaged on Windows.
- Reporting silos: When device inventory lives in separate systems, answering "how many devices are running vulnerable software?" requires custom integration work or manual consolidation.

Both solutions also create vendor lock-in through proprietary architectures. Migrating between MDMs means translating configurations with realistic migration timelines extending months rather than weeks. Device telemetry remains within each solution's ecosystem, though both offer API access for organizations willing to build custom pipelines.

## When should you look at multi-platform MDM alternatives?

Organizations running Windows computers alongside Macs, maintaining Linux servers, or supporting Chromebooks in specific departments face a key question: is your current device management setup actually solving your problem, or just part of it?

When your fleet includes a meaningful proportion of non-Apple devices, managing everything through a single interface cuts the time your team spends switching between tools, reconciling reports, and maintaining separate policy definitions. Multi-platform solutions become worth evaluating when tool sprawl creates real operational friction through duplicate compliance reporting, inconsistent security enforcement, or training overhead.

Security-focused teams particularly value solutions that provide infrastructure-as-code workflows, REST API access, and native support across macOS, Windows, Linux, iOS/iPadOS, and Android. If your security strategy depends on integrating device management with existing SIEM and security infrastructure across all operating systems, solutions like Jamf Pro and Iru (Kandji) will need additional tooling to cover the full scope.

## How Fleet addresses gaps in Jamf and Iru (Kandji)

Fleet can provide near [real-time device reporting](https://fleetdm.com/releases/fleet-4-78-0) across macOS, Windows, Linux, iOS, Android, and ChromeOS. During incident response, your security teams get immediate answers rather than waiting for scheduled device check-ins.

Automated evidence collection for compliance frameworks happens through scheduled queries that capture required data points continuously. Rather than scrambling before audits, your teams can export compliance evidence covering the entire review period.

## Unified device management across platforms

Choosing between Jamf and Iru (Kandji) addresses Apple device management, but for many organizations, this solves only part of the problem.

[Fleet](https://fleetdm.com/device-management) is an open-source device management solution that provides unified visibility and control across macOS, iOS, Windows, Linux, ChromeOS, and Android. The solution integrates with existing identity providers, enforces security policies through [GitOps workflows](https://fleetdm.com/fleet-gitops), and delivers near real-time device reporting during incident response. [Schedule a demo](https://fleetdm.com/demo) to see how Fleet handles device management across entire fleets.

## Frequently asked questions

#### How hard is it to migrate from Jamf to Iru (Kandji) or from Iru (Kandji) to Jamf?

MDM migrations benefit from careful planning and phased implementation. Apple's Managed Device Migration features built into their operating systems for all devices running an Apple OS 26.0+ makes MDM migration easier than ever. By reassigning devices to a new MDM server in Apple Business Manager and setting an enforcement deadline, Apple handles notifying users and, if they don't act in time, automatically enforcing the migration. The bigger challenge isn't the migration itself. It's translating configurations between solutions. Jamf's policy-based architecture fundamentally differs from the Iru (Kandji) Blueprint system, meaning you'll need to reimplement policies rather than simply export and import them.

#### Do I still need separate security and compliance tools if I use Jamf or Iru (Kandji)?

Iru (Kandji) includes native EDR and automated compliance controls that can reduce the need for separate endpoint security tools. Jamf requires add-on modules (Jamf Protect for EDR, Jamf Compliance Editor/Jamf Pro Compliance Benchmarks for compliance framework mapping) that together provide similar coverage. However, both solutions have important limitations: your security teams typically still need SIEM integration, threat intelligence platforms, and broader security monitoring beyond endpoint management. Additionally, organizations with multi-platform device fleets may face coverage gaps with either solution alone, as neither fully covers Linux or ChromeOS.

#### What are the best alternatives for mixed macOS, Windows, and Linux fleets?

Note: Jamf supports Apple and Android, and Iru (Kandji) supports Apple, Windows, and Android. Organizations with mixed-platform fleets may need different approaches. Unified device management solutions that support multiple operating systems through a single interface work better than maintaining separate tools for each platform. [Fleet](https://fleetdm.com/device-management) provides open-source multi-platform device management with support for macOS, iOS, Windows, Linux, ChromeOS, and Android through API, UI, or GitOps workflows, alongside zero-touch device deployment across all platforms along with unified device inventory and configuration management capabilities.

#### When does it make sense to use Jamf or Iru (Kandji) alongside a multi-platform solution?

You can maintain Jamf or Iru (Kandji) for core device management while adding multi-platform solutions for unified querying, automated compliance evidence collection, and near real-time visibility across all platforms. This layered approach works when your teams have expertise with the existing MDM solution and need multi-platform capabilities without disrupting established workflows. Multi-platform solutions complement rather than conflict with existing MDM infrastructure. [Try Fleet](https://fleetdm.com/demo) to see how it integrates with your current device management setup.

<meta name="articleTitle" value="Fleet vs. Jamf vs. Iru: How to Choose the Right MDM">
<meta name="authorFullName" value="Ashish Kuthiala">
<meta name="authorGitHubUsername" value="akuthiala">
<meta name="category" value="comparison">
<meta name="articleSlugInCategory" value="jamf-vs-iru-vs-fleet">
<meta name="introductionTextBlockOne" value="This guide covers how Jamf Pro and Iru (formerly Kandji) differ in architecture, compliance workflows, and day-to-day management, and when multi-platform alternatives become worth evaluating.">
<meta name="publishedOn" value="2026-03-24">
<meta name="description" value="Compare Fleet, Jamf, and Iru (fka Kandji) for platform support, pricing, GitOps, security, and API capabilities to find the right MDM for your team.">

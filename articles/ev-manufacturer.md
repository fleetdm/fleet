# EV manufacturer brings Linux workstations under centralized management with Fleet

An electric vehicle manufacturer builds software and hardware that power modern vehicles. Its engineering teams rely on a mix of macOS, Windows, and Linux systems to design, test, and ship complex automotive technology.

As the company’s engineering footprint grew, Linux workstations became increasingly critical. Managing those systems with the same rigor as corporate laptops required a more flexible device management platform.

## At a glance

* **Industry:** Automotive and electric vehicles

* **Devices managed:** 500+ Linux workstations plus macOS and Windows devices

* **Primary requirements:** Centralized Linux management, automated remediation

* **Previous challenge:** Limited visibility and management for Linux engineering systems

## The challenge

The company’s engineering teams rely heavily on Linux systems, especially Ubuntu-based workstations used for development and testing.

Legacy device management tools struggled to support these environments. Linux devices and servers were often unmanaged, leaving the IT and security teams without a reliable way to verify configuration or enforce policies.

This created blind spots across the organization. Engineering workstations that played a critical role in development pipelines lacked visibility and compliance tracking.

The team needed a system that could manage Linux devices with the same consistency and automation as macOS and Windows systems.

## Evaluation criteria

1. **Centralized Linux management:** Provide strong support for Ubuntu-based engineering workstations.

2. **Policy automation and script execution:** Detect configuration drift and automatically remediate issues at scale.

3. **GitOps workflows:** Manage device configurations using version-controlled processes similar to the company’s vehicle software pipelines.

A unified platform across macOS, Windows, and Linux was also critical. The team wanted to avoid maintaining separate tools for each operating system.

## The solution:

Fleet gave the team a unified system to manage engineering and corporate devices.

Linux workstations that were previously unmanaged are now fully visible and monitored. The platform allows security teams to query system state in real time and enforce consistent policies across the entire fleet.

Automation plays a key role. Fleet policies detect configuration drift and automatically run remediation scripts when issues appear. For example, the team implemented automated monitoring of DNS configuration. If a device’s DNS settings drift from the company’s standard configuration, Fleet automatically runs a remediation script every hour until the issue is corrected.

Fleet’s open-source model also provides transparency and flexibility. Security teams can inspect how the system works and adapt it to meet the needs of a highly technical engineering environment.

### A phased rollout with minimal disruption

Devices were gradually enrolled into Fleet while maintaining the uptime required for automotive development operations. This careful approach allowed the organization to expand coverage without interrupting engineering workflows.

In some cases, Fleet actually improved the user experience. Automated agent updates and self-service software installation helped reduce friction for engineers working on development systems.

## The results

Fleet introduced centralized visibility across Linux, macOS, and Windows systems.

Security teams can now track patch cadence, monitor configuration drift, and generate compliance reports using real-time device data. Vulnerabilities can be detected and remediated much faster than before.

Telemetry from devices also streams directly into the company’s internal data platforms. This integration allows the team to build custom dashboards that track device health and security trends across the entire fleet.

The shift from fragmented tools to a unified platform also improved operational efficiency. By automating routine compliance checks and remediation tasks, the IT team can focus on higher-impact infrastructure work.

### Why they recommend Fleet

For technology leaders in engineering-heavy organizations, their recommendation is straightforward:

Fleet provides the granularity and flexibility needed to manage modern development environments.

The platform allows teams to customize policies, automate remediation, and maintain visibility across diverse operating systems without relying on rigid, one-size-fits-all tooling.


<meta name="articleTitle" value="EV manufacturer brings Linux workstations under centralized management with Fleet">
<meta name="authorFullName" value="Irena Reedy">
<meta name="authorGitHubUsername" value="irenareedy">
<meta name="category" value="case study">
<meta name="publishedOn" value="2026-02-22">
<meta name="description" value="How an EV manufacturer uses Fleet to manage Linux workstations with centralized visibility and automated remediation."> 
<meta name="useBasicArticleTemplate" value="true">
<meta name="cardTitleForCustomersPage" value="Electric vehicle manufacturer">
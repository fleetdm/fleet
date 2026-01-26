# Open source MDM software: How to choose the right solution

Organizations seeking transparent, auditable device management have several open source options to evaluate. MicroMDM ended official support in 2025, leaving three actively maintained projects: Fleet, NanoMDM, and Zentral. Each takes a different approach to open source MDM, from multi-OS coverage to minimalist protocol implementations.

This guide compares these tools across deployment models, security capabilities, and OS support.

## Overview

Fleet lets you manage and protect laptops and mobile devices across macOS, iOS, iPadOS, Windows, Linux, ChromeOS, and Android from a single platform. Fleet has an API-first design with built-in GitOps console management and provides sub-30-second device reporting, comprehensive device control, and automated remediation capabilities.

Fleet can return osquery-backed telemetry in seconds to minutes depending on configuration, whereas many tools rely on periodic inventory and reporting intervals. Teams can manage Fleet through a web console, REST API, or infrastructure-as-code workflows where YAML configurations in a Git repository define the desired state of the fleet.

NanoMDM is an Apple-only MDM server limited to core protocol implementation for macOS, iOS, and iPadOS. NanoMDM operates through CLI and API only, without a web console. Organizations using NanoMDM integrate it with external tools for configuration management, security monitoring, and compliance workflows.

Zentral provides Apple MDM across macOS, iOS, iPadOS, and tvOS, plus an osquery server for visibility beyond Apple devices. Zentral integrates with external tools (Munki for patch management, Santa for allowlisting) rather than providing these capabilities natively.

## Key differences

| Attribute | Fleet | NanoMDM | Zentral |
| ----- | ----- | ----- | ----- |
| OS support | macOS, Windows, Linux, iOS, iPadOS, ChromeOS, Android | macOS, iOS, iPadOS only | macOS, iOS, iPadOS, tvOS; others via osquery |
| Architecture | API-first, unified REST API, osquery foundation | CLI/API only, pure MDM protocol | Web dashboard, event-driven |
| Management interface | Web UI, API, GitOps, IaC  | CLI and API only | Web UI, Terraform provider (limited scope) |
| Built-in security | Vulnerability detection, YARA rules, CIS benchmarks, SIEM integration | None (external integration required) | Relies on Santa (macOS-only), compliance checks |
| Device reporting | Sub-30-second reporting | N/A (protocol implementation) | Configurable (depends on osquery schedule) |
| Deployment options | Cloud-hosted or self-hosted | Self-hosted only | Cloud-hosted or self-hosted |
| Commercial support | Yes | No | Yes |
| License | Core open source (MIT); some features source-available | MIT | Core open source (Apache 2.0); some modules source-available |

## Device management workflow comparisons

### Enrollment and provisioning

Fleet supports zero-touch enrollment through Apple Business Manager for Apple devices and Windows Autopilot for Windows devices. Fleet provides enrollment options across all supported platforms with settings that prevent end users from removing management without authorization.

NanoMDM supports Apple Business Manager enrollment for zero-touch deployment of Apple devices. NanoMDM implements the core MDM protocol; production enrollment typically requires additional infrastructure and setup (APNs, ABM/DEP server assignment, certificates, and optional SCEP), and teams often pair it with other tools for higher-level workflows.

Zentral supports Apple Business Manager integration for Apple device enrollment across macOS, iOS, iPadOS, and tvOS. Zentral uses MDM Blueprints for scoping configurations to devices based on tags.

### Configuration management

Fleet scopes Configuration Profiles using Teams and Labels. Labels can be assigned manually, generated dynamically from osquery results, or derived from server-side attributes like IdP group membership. Because Fleet runs osquery on every device, administrators can verify that profiles were actually applied rather than assuming delivery succeeded.

NanoMDM provides raw MDM command queuing and delivery. Configuration profile creation, distribution, and verification require external tooling. Organizations typically pair NanoMDM with configuration management systems like Ansible or Puppet.

Zentral provides MDM Blueprints for grouping settings and profiles across Apple devices. Tags can be synced from IdP groups or set via conditions.

### Software management

Fleet handles software deployment through Fleet-maintained apps, Apps and Books (VPP) distribution, custom package uploads, and scripting across all supported operating systems.

NanoMDM doesn't include software management capabilities. Organizations integrate external tools for application deployment and updates.

Zentral relies on Munki for software distribution and patching, which limits software management to macOS only.

### Security and compliance

Fleet includes built-in software vulnerability detection integrating the National Vulnerability Database, Known Exploited Vulnerabilities catalog, and EPSS scoring. Fleet provides CIS Benchmark queries, STIG compliance automation, threat detection via YARA rules, file integrity monitoring, and SIEM integration. These capabilities are included under a single license at no additional cost.

NanoMDM doesn't include security or compliance features. Organizations bring their own vulnerability scanning, threat detection, and compliance monitoring tools.

Zentral provides compliance checks based on inventory data and osquery. Zentral relies on Santa for application allowlisting, which is macOS-only.

## Multi-OS vs. single-OS support

OS coverage determines whether organizations can consolidate device management into one tool or need multiple tools. For open source evaluators specifically, code transparency applies consistently across operating systems: Fleet's osquery-based approach provides the same queryable visibility on Windows and Linux that it does on macOS.

Fleet manages macOS, Windows, Linux, iOS, iPadOS, ChromeOS, and Android from a single console, with capabilities that vary by operating system.

NanoMDM supports Apple devices only (macOS, iOS, iPadOS). Organizations with Windows, Linux, or Android devices need additional tools.

Zentral provides Apple MDM for macOS, iOS, iPadOS, and tvOS, with osquery-based visibility for other operating systems. Organizations with significant non-Apple device populations may need supplementary tools for full management capabilities.

## FAQ

### What distinguishes multi-OS open source MDM from Apple-focused alternatives?

Multi-OS tools manage multiple operating systems from one console, reducing tool sprawl and consolidating visibility. Apple-focused tools like NanoMDM specialize in macOS, iOS, and iPadOS. Zentral covers macOS, iOS, iPadOS, and tvOS with MDM, plus osquery for other operating systems. For organizations with mixed device fleets, multi-OS open source MDM avoids maintaining multiple codebases and learning multiple interfaces. [Try Fleet](https://fleetdm.com/try-fleet/register) to evaluate multi-OS management in your environment.

### Can open source MDM match commercial tools for Apple device management?

Fleet supports Apple Business Manager for zero-touch enrollment, Configuration Profiles, Declarative Device Management, Apps and Books distribution, and remote commands. The osquery foundation adds capabilities commercial Apple MDMs lack: SQL-based device queries, real-time policy verification, and sub-30-second reporting.

### What should I consider when evaluating open source MDM sustainability?

Prioritize projects with active development, commercial backing, and enterprise adoption. MicroMDM ended official support in 2025, demonstrating the risk of project abandonment. Fleet has commercial support options and validates at enterprise scale with over 2 million managed devices globally for organizations including Stripe, Dropbox, and Fastly. NanoMDM is actively maintained but has no commercial support. Zentral is actively maintained with paid support available.

### How do security capabilities differ between open source MDM solutions?

Fleet includes built-in vulnerability detection, compliance automation for CIS and STIG, threat detection via YARA rules, and SIEM integration under a single license. NanoMDM provides no built-in security features, requiring external tool integration. Zentral relies on Santa for macOS-only application allowlisting and provides osquery-based compliance checks.

### What technical expertise is required for each open source MDM solution?

Fleet supports web UI, API, and GitOps workflows, accommodating teams with varying technical backgrounds. NanoMDM requires strong development or DevOps skills for CLI/API-only operation and integration with external tools. Zentral provides a web dashboard and Terraform provider, requiring familiarity with Apple administration and potentially infrastructure-as-code workflows.

### How long does it take to implement open source MDM?

Implementation timelines vary based on fleet size and existing infrastructure. Fleet provides customer support and professional services to assist with migration, including enrollment workflows for all supported platforms. [Schedule a demo](https://fleetdm.com/try-fleet/register) to discuss implementation timelines for your environment.

<meta name="articleTitle" value="Open source MDM software: Fleet vs. NanoMDM vs. Zentral">
<meta name="authorFullName" value="Brock Walters">
<meta name="authorGitHubUsername" value="nonpunctual">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-01-14">
<meta name="description" value="Compare open source MDM solutions for device management. See how Fleet, NanoMDM, and Zentral differ in platform support, security features, and deployment options. ">

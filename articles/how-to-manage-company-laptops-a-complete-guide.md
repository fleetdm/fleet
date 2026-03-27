# How to manage company laptops: a complete guide

Distributed workforces have scattered company laptops across home offices, coworking spaces, and airports, while platform diversity adds complexity: macOS, Windows, and Linux each bring different enrollment protocols, update mechanisms, and security controls. This guide covers how to onboard laptops securely across platforms, maintain real-time visibility and control, and keep devices protected through policies, patching, and least privilege.

## What does it mean to manage laptops today?

Managing company laptops goes well beyond tracking serial numbers and pushing software updates. It means establishing a consistent security baseline across every device, enforcing compliance with frameworks like SOC 2, HIPAA, or PCI DSS, and giving IT teams the ability to configure, monitor, and remediate devices regardless of where they connect to the internet.

Where IT teams once configured laptops manually before handing them to employees, modern device management relies on Mobile Device Management (MDM) protocols, zero-touch enrollment, and policy-based configuration. These approaches let IT define settings once and apply them automatically across the fleet, turning device management into something closer to infrastructure engineering than traditional desktop support.

## Why unify laptop management across Mac, Windows, and Linux?

Many organizations run separate tools for each platform: one for macOS, another for Windows, and a patchwork of scripts or configuration management tools for Linux. This fragmentation tends to create operational silos, inconsistent security posture, and a growing maintenance burden. Unifying management can help, though unified tools may not match the platform-specific depth of dedicated ones.

Here's what organizations gain when laptops come under one management approach:

* **Consistent security baselines:** A single approach lets teams enforce encryption, password policies, and firewall rules across all platforms, reducing the gaps attackers look for when targeting the weakest link in a mixed fleet. Consistency depends on the tool actually supporting equivalent controls on each platform.  
* **Simplified compliance reporting:** Pulling compliance evidence from one console is typically far faster than assembling reports from multiple tools. Many frameworks expect proof of uniform controls across all devices that access sensitive data.  
* **Reduced tool sprawl and cost:** Consolidation can free budget and reduce time spent switching between dashboards, though teams should verify a unified tool covers their platform-specific needs before retiring specialized ones.  
* **Faster onboarding and offboarding:** Standardized enrollment workflows make it easier to get a configured, secure laptop to a new hire on day one and to deprovision departing employees consistently.

These benefits tend to compound as fleets grow.

## How to onboard laptops securely and consistently

Secure onboarding starts before the device reaches the employee. The goal is zero-touch enrollment: the laptop is pre-registered with an enrollment service, then configures itself during initial setup without IT physically handling it. Before diving into platform-specific enrollment, align on fleet-wide decisions (ownership model, identity provider, encryption escrow, admin rights policy, and update strategy) so the organization doesn't end up with three different definitions of "managed."

### Apple devices: Automated Device Enrollment

Apple Business (AB) links device serial numbers to an organization's MDM server. When a new Mac powers on, it contacts Apple's activation servers, identifies the assigned MDM, and enrolls automatically during Setup Assistant. Devices enrolled through Automated Device Enrollment (ADE) can be configured as supervised, which prevents users from removing the MDM profile and enables additional management capabilities. Once enrolled, configuration profiles, encryption settings, and applications can deploy automatically.

A few details that often trip teams up: Apple Push Notification service (APNs) certificates need renewal before they expire or enrolled devices may stop receiving new commands, Setup Assistant flows should be tested on the same macOS version being shipped, and splitting profiles by function (security restrictions separate from VPN settings) reduces change risk.

### Windows devices: Autopilot and Microsoft Entra ID

Windows Autopilot enables zero-touch deployment for new devices, using pre-registered hardware hashes and Microsoft Entra ID join. Device hardware hashes are registered with the Autopilot service, typically by the OEM at purchase. When a registered device first boots, it joins Microsoft Entra ID and auto-enrolls into device management. User-driven mode is common for employee laptops, while self-deploying mode can work for shared devices.

For organizations migrating from on-premises Active Directory, Group Policy Objects often need conversion to MDM policies since Group Policy doesn't apply the same way on Microsoft Entra joined devices. Also worth planning early: confirm that Trusted Platform Module (TPM) is enabled in firmware for BitLocker, establish naming conventions before cleanup work accumulates, and test update rings on a canary group before broad rollout.

### Linux devices: Configuration management

Linux lacks a standardized, OS-level MDM enrollment protocol, so teams typically rely on configuration management or custom agents. Organizations supporting multiple distros often need separate playbooks for package managers, service management, and security controls.

To make Linux onboarding less ad hoc, treat provisioning as a repeatable checklist. Linux Unified Key Setup (LUKS) is typically configured during OS installation when full disk encryption is required, so the provisioning flow becomes a key control point, whereas macOS and Windows commonly enable FileVault and BitLocker during or shortly after initial enrollment. 

Access to configuration management should use SSH certificates or rotate keys on a schedule to avoid long-lived static keys. A baseline package set (osquery agent, VPN client, log forwarder, and configuration management agent) keeps visibility consistent across laptops.

For organizations that want to bring Linux into the same console as macOS and Windows, device management solutions built on osquery can collect device data across all three platforms, though enforcement capabilities vary depending on the tool.

## How to maintain visibility and control over all laptops in real time

Once devices are enrolled, the next challenge is keeping track of what's actually happening on them. Remote devices may never connect to a corporate network directly, so most programs combine continuous inventory with automated compliance checks and clear exception handling.

### Continuous device inventory

Ideally, a device management tool provides near real-time or frequently updated data on hardware specs, installed software, OS versions, and security posture for enrolled devices. This isn't the same as asset management (dedicated tools handle procurement and lifecycle tracking), but device management feeds those systems with accurate, current device data.

Think in terms of questions your team needs to answer quickly: which laptops are running unsupported OS builds, are security agents actually running right now, and has any unauthorized remote access tooling appeared? If current tooling can't answer these consistently, gaps tend to show up during incidents.

Devices that go offline for extended periods create blind spots. Setting thresholds for acceptable check-in intervals lets IT flag devices that haven't reported in, and conditional access can block sensitive apps until compliance is confirmed.

### Compliance monitoring

Rather than scrambling before audits, teams can set up continuous compliance checks that compare device state against established security baselines. Automated monitoring catches drift as it happens, and IT can decide whether to remediate automatically or require a manual review.

It also helps to define what's acceptable as a documented exception. Some teams treat exceptions as time-bound waivers, such as "this developer laptop can defer feature updates for 30 days, but not security updates." When tracked centrally, auditors can see why a device deviates and when it's scheduled to return to baseline.

## How to keep laptops secure with policies, patching, and least privilege

Security for company laptops rests on three pillars: enforcing baseline policies, keeping software current, and limiting access to what each user actually needs.

* **Encryption and security policies:** Full disk encryption protects data on lost or stolen devices. Beyond encryption, common baseline controls include screen lock timeouts, firewall defaults, and restrictions on removable storage. If users need exceptions, a clear approval process helps keep the baseline enforceable.  
* **OS and application patching:** Prioritize OS patches first, then application patches, and keep a canary group to catch breakages before broad rollout. Each platform handles patching differently: Declarative Device Management (DDM) for macOS 15 and later, MDM update policies for Windows, and environment-specific tooling for Linux. A clear patch policy with communicated reboot expectations keeps fewer devices in a half-patched state.  
* **Least privilege access:** Create unique accounts per employee, assign access based on role, and keep users as standard accounts by default. Multi-factor authentication for remote access and regular access reviews help catch privilege creep before it becomes a problem.

These three controls reinforce each other. Encryption protects data at rest, patching closes known vulnerabilities, and least privilege limits the damage if a device or account is compromised.

## How to troubleshoot, handle incidents, and offboard

Managing laptops doesn't stop at configuration. Teams also need the ability to act on devices when things go wrong or when employees leave.

### Remote troubleshooting

When a remote employee reports an issue, support teams need the ability to run [live queries](https://fleetdm.com/guides/queries), check installed software versions, verify network configuration, and review logs without asking the user to run terminal commands.

To keep remote troubleshooting fast and consistent, standardize what the help desk can collect:

* **Basic health signals:** Capture disk space, battery health, OS build, and last reboot to rule out common failure modes early.  
* **Security posture:** Check encryption status, firewall status, and agent status so troubleshooting doesn't accidentally bypass security controls.  
* **Network configuration:** Verify VPN status, DNS servers, and active interfaces to catch split-tunnel issues and misrouted traffic.

When that checklist is consistent, support teams reduce back-and-forth with users and avoid "works on my machine" loops.

### Incident response

If a device is compromised, time matters. For macOS and Windows devices, remote lock and remote wipe can be issued via MDM and will execute as soon as the device next connects and the management service can deliver the command. Devices offline for extended periods will wipe once they reconnect, which can surprise users who borrowed equipment during outages, so wipe policies should be communicated clearly. For Linux, organizations often rely on full-disk encryption plus key revocation, account disablement, and re-provisioning workflows.

For investigations, having near real-time device data (running processes, network connections, installed software) lets security teams triage faster. Logging and audit trails support both internal investigations and regulatory requirements like GDPR, which commonly expects notification of certain personal data breaches to authorities within about 72 hours where feasible.

### Offboarding

When an employee leaves, an offboarding workflow should revoke access, unenroll the device from management, and, depending on policy, wipe the device remotely.

In practice, teams get fewer surprises when access removal is separated from device recovery:

* **Revoke identity access first:** Disable the user in your identity provider and enforce conditional access so the laptop can't reach corporate resources even if it stays online.  
* **Handle legal hold before wiping:** Decide whether data needs to be preserved for legal hold or investigation. A full wipe is final, so the wipe step should align with your HR and legal processes.  
* **Reset for reassignment:** Prepare the laptop for reassignment. For Macs, this may involve a manual wipe-and-reenroll cycle, or the device can be reassigned through Apple Business.

If device management integrates with the identity provider, offboarding becomes less dependent on someone remembering a checklist. For example, disabling a user account in Microsoft Entra ID or Okta can trigger conditional access policies that block the device from corporate resources.

## Manage company across platforms with Fleet

The practices above work best when your device management tool supports all platforms from a single console, while still providing sufficient depth on each operating system for your requirements.

Fleet is an open-source device management solution that manages macOS, Windows, and Linux alongside iOS, iPadOS, ChromeOS, and Android. By integrating Fleet with a git repository like GitHub, IT teams can use Infrastructure-as-Code (IaC) / GitOps workflows to create and push policy configurations in version-controlled YAML files, review changes through pull requests, and apply them using Fleet's `fleetctl` CLI in CI/CD pipelines. 

Because Fleet is built on [osquery](https://www.osquery.io/), it can provide more detailed device data than typical MDM inventory syncs, with near real-time reporting for many use cases and a self-hosted deployment option for organizations with data residency requirements. [Schedule a demo](https://fleetdm.com/contact) to see how Fleet fits your laptop management workflows.

## Frequently asked questions

### How long does it take to set up laptop management for a new organization?

The timeline depends on fleet size, platform mix, and existing infrastructure. Many organizations complete initial enrollment and baseline policy deployment within a few weeks for a pilot group, then roll out in waves of 20–30% of remaining devices. Full production deployment across all platforms usually takes a minimum of at least 30-60 days or longer depending on the number of devices and configuration complexity.

### Can company laptops be managed without an on-premises server?

Fleet is fully committed to on-prem installations of the Fleet infrastructure and can be flexibly installed just about anywhere. Modern device management tools can operate entirely through cloud-based infrastructure. Apple devices communicate through Apple Push Notification service (APNs), Windows devices enroll through Microsoft Entra ID. Devices only need internet access, not corporate network access, to receive management commands.

### What happens if a managed laptop goes offline for an extended period?

Pending commands such as configuration updates and remote wipe orders typically queue on the management server and execute when the device reconnects. Most tools flag devices that haven't checked in within a configurable threshold so IT teams can investigate, and conditional access can restrict high-risk access until the device reports healthy again.

### How do you manage laptops running different operating systems from one place?

Multi-platform device management tools use OS-native protocols and management frameworks behind a single administrative console. The best fit depends on operating systems in the fleet, reporting requirements, and whether configuration-as-code workflows are a priority. Fleet supports multi-platform laptop management from one console, so teams can standardize many policies and visibility workflows across macOS, Windows, and Linux. [Explore Fleet](https://fleetdm.com/device-management) to see how it works in practice.

<meta name="articleTitle" value="How to manage company laptops: a complete guide">
<meta name="authorFullName" value="Brock Walters">
<meta name="authorGitHubUsername" value="nonpunctual">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-03-07">
<meta name="description" value="An overview of laptop management concepts and best practices for the enterprise.">

# Digital bank strengthens security and compliance with Fleet

The company serves millions of customers through a mobile-first platform designed to expand access to financial services. Supporting that mission requires a secure and reliable device environment across thousands of employees. The company manages more than 10,000 devices across macOS, Windows, Linux, and ChromeOS, all of which must meet strict security and regulatory requirements.

## At a glance

* **Industry:** Financial services and digital banking

* **Devices managed:** 10,000+ devices across macOS, Windows, Linux, and ChromeOS

* **Primary requirements:** On-premise control, osquery visibility, GitOps automation

* **Previous challenge:** Limited visibility across non-Mac devices and concerns around proprietary management tools

## The challenge

Legacy device management tools created several challenges. Port configurations were complex, visibility across non-Mac devices was limited, and proprietary agents acted as black boxes. That lack of transparency created risk in an environment where security teams must understand exactly how device policies are enforced.

Linux servers and remote laptops were also difficult to monitor consistently. Without a unified way to verify device state, maintaining a consistent security baseline across the fleet became increasingly difficult.

The team needed a platform that provided full visibility, strong automation, and complete control over how the system operated.

## Evaluation criteria

During the evaluation process, Fleet had to meet three key requirements:

1. **On-premise hosting:** The company required full control of infrastructure to satisfy financial and regulatory compliance.

2. **osquery integration:** Security teams needed deep, SQL-based visibility into device state across operating systems.

3. **GitOps and automation:** Device management had to integrate with existing CI/CD workflows and automation pipelines.

The team also wanted a unified approach to managing macOS, Windows, Linux, and ChromeOS devices instead of maintaining separate management silos.

## The solution

Fleet provided a platform that aligned with both security and engineering requirements.

The company deployed Fleet across multiple instances to support its large-scale environment. Using osquery through Fleet, the team can now query device state across the entire fleet and verify compliance in real time.

Fleet also integrates with the company’s identity and security systems. For example, the team developed custom multi-factor authentication (MFA) workflows that connect Okta identity policies with Fleet device checks.

Fleet’s API allows the team to automatically generate and prioritize vulnerability tickets, ensuring that security teams focus on the most critical risks rather than reviewing thousands of alerts manually.

Device telemetry also streams directly into the company’s internal monitoring tools, providing real-time visibility into device health and software changes.

### A phased rollout across a global fleet

The company used a phased deployment strategy across multiple Fleet instances. Each segment of the fleet was migrated gradually to ensure stability and maintain regulatory compliance throughout the process.

Despite the scale of the transition, end-user disruption remained minimal. Automated policies and carefully managed update cycles allowed employees to continue working without interruption.

## The results

Fleet introduced a unified view of the company’s device environment.

Security teams now monitor macOS, Windows, Linux, and ChromeOS systems through a single platform. This consistency helps ensure every device meets the same security baseline required.

Real-time telemetry also improves response time to vulnerabilities and compliance requests. Automated dashboards and prioritized alerts allow the team to identify and remediate risks as soon as they appear.

Operational efficiency has improved as well. By consolidating device management into a unified platform, the team reduced management overhead and improved license management across the organization.

### Why they recommend Fleet

For other technology leaders in regulated industries, their recommendation focuses on transparency and control.

Fleet provides an open platform that allows security teams to understand exactly how device policies work. That transparency, combined with automation and real-time visibility, makes it easier to operate a secure device fleet at global scale.

For organizations managing thousands of devices in regulated environments, that level of insight and control is essential.


<meta name="articleTitle" value="Digital bank strengthens security and compliance with Fleet">
<meta name="authorFullName" value="Irena Reedy">
<meta name="authorGitHubUsername" value="irenareedy">
<meta name="category" value="case study">
<meta name="publishedOn" value="2026-03-03">
<meta name="description" value="Fleet helps a digital bank manage 10,000+ devices with on-prem control, real-time visibility, and automation across macOS, Windows, Linux, ChromeOS."> 
<meta name="useBasicArticleTemplate" value="true">
<meta name="cardTitleForCustomersPage" value="Digital bank">

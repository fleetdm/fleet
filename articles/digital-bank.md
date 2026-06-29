# Digital bank strengthens security and compliance with Fleet

The company serves millions of customers through a mobile-first platform designed to expand access to financial services. Supporting that mission requires strong endpoint security and visibility across thousands of employees. With more than 10,000 endpoints spanning macOS, Windows, Linux, and ChromeOS — all subject to strict regulatory requirements — the company needed deep, real-time visibility into device state across its entire environment.

## At a glance

- **Industry:** Financial services and digital banking
- **Endpoints with Fleet visibility:** 10,000+ across macOS, Windows, Linux, and ChromeOS
- **Primary requirements:** Real-time osquery visibility, multi-OS coverage, API-driven automation
- **Previous challenge:** Limited visibility across non-Mac systems and concerns around proprietary, opaque agents

## The challenge

Legacy tools created several challenges. Visibility across non-Mac systems was limited, and proprietary agents acted as black boxes. That lack of transparency created risk in an environment where security teams need to understand exactly how endpoint data is collected and how compliance is verified.

Linux servers and remote laptops were especially difficult to monitor consistently. Without a unified way to verify device state across operating systems, maintaining a consistent security baseline across the fleet became increasingly difficult.

The team needed a platform that provided full visibility, strong automation, and clear, transparent control over how endpoint telemetry was collected and used.

## Evaluation criteria

During the evaluation process, Fleet had to meet three key requirements:

1. **osquery integration:** Security teams needed deep, SQL-based visibility into device state across every operating system.
2. **Multi-OS coverage:** A single platform spanning macOS, Windows, Linux, and ChromeOS, replacing fragmented per-OS tools.
3. **API and GitOps automation:** Integration with existing CI/CD workflows and security automation pipelines, so endpoint telemetry could feed into the systems the team already operates.

## The solution: real-time visibility across a global fleet

Fleet provided a platform that aligned with both security and engineering requirements.

The company runs on Fleet Cloud, taking advantage of fully managed SaaS while keeping endpoint telemetry and security workflows in their team's hands. Using osquery through Fleet, the team can query device state across the entire fleet and verify compliance in real time, across every operating system in the environment.

Fleet integrates with the company's identity and security systems. For example, the team developed custom multi-factor authentication (MFA) workflows that connect Okta identity policies with Fleet device checks, allowing access decisions to incorporate real-time device posture.

Fleet's API also allows the team to automatically generate and prioritize vulnerability tickets, ensuring that security teams focus on the most critical risks rather than reviewing thousands of alerts manually. Device telemetry streams directly into the company's internal monitoring tools, giving security and IT teams a live view of endpoint health and software changes.

### A phased rollout across a global fleet

The company used a phased rollout strategy across its global environment. Each segment of the fleet was onboarded gradually to ensure stability and maintain regulatory compliance throughout the process.

Despite the scale of the transition, end-user disruption remained minimal. Carefully managed onboarding and update cycles allowed employees to continue working without interruption.

## The results

Fleet introduced a unified view of the company's endpoint environment.

Security teams now monitor macOS, Windows, Linux, and ChromeOS systems through a single platform. This consistency helps ensure every device meets the same security baseline.

Real-time telemetry also improves response time to vulnerabilities and compliance requests. Automated dashboards and prioritized alerts allow the team to identify and remediate risks as soon as they appear.

Operational efficiency improved as well. By consolidating endpoint visibility into a unified platform, the team reduced overhead and simplified licensing across the organization.

### Why they recommend Fleet

For other technology leaders in regulated industries, their recommendations focus on transparency and control. Fleet provides an open platform that allows security teams to understand exactly how endpoint data is collected and how compliance is verified. That transparency, combined with automation and real-time visibility, makes it easier to operate a secure endpoint environment at a global scale.

For organizations operating thousands of endpoints in regulated environments, that level of insight is essential.


<meta name="articleTitle" value="Digital bank strengthens security and compliance with Fleet">
<meta name="authorFullName" value="Irena Reedy">
<meta name="authorGitHubUsername" value="irenareedy">
<meta name="category" value="case study">
<meta name="publishedOn" value="2026-03-03">
<meta name="description" value="Fleet helps a digital bank manage 10,000+ devices with on-prem control, real-time visibility, and automation across macOS, Windows, Linux, ChromeOS."> 
<meta name="useBasicArticleTemplate" value="true">
<meta name="cardTitleForCustomersPage" value="Digital bank">

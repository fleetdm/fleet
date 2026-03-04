# Global collaboration platform consolidates device management with Fleet

A global technology company provides a collaboration platform that helps people and businesses securely store, organize, and share files. Supporting millions of users requires reliable infrastructure and strong internal security practices.

To support its distributed workforce, the company manages a large fleet of devices across macOS, Windows, Linux, ChromeOS, and mobile platforms. As the environment grew, the team needed a simpler way to manage devices across operating systems while maintaining consistent visibility and security controls.

## At a glance

* **Industry:** Technology and cloud collaboration

* **Devices managed:** Tens of thousands across macOS, Windows, Linux, ChromeOS, and mobile

* **Primary requirements:** Unified device management, GitOps workflows, osquery visibility

* **Previous challenge:** Fragmented tooling and inconsistent visibility across platforms

## The challenge

The company previously relied on multiple device management tools such as Jamf and Intune. This fragmentation created operational complexity and increased costs. Teams managing different platforms had to maintain separate systems, workflows, and expertise.

Visibility was also inconsistent. Linux servers and remote laptops lacked a reliable system for verifying device security and compliance across the organization.

As the workforce became more distributed, the company needed a single platform that could provide unified visibility and simplify device management operations.

## Evaluation criteria

During the evaluation process, Fleet needed to meet three core requirements:

1. **Hosting flexibility:** Support both on-premise and cloud-hosted deployments.

2. **GitOps workflows:** Allow device configurations and policies to be managed through version-controlled code.

3. **Strong osquery integration:** Provide deep, real-time visibility into device state across all operating systems.

The team also wanted a platform capable of managing macOS, Windows, and Linux devices through a single API rather than separate tools.

## The solution

Fleet provided the company with a single platform for managing its diverse device environment.

The team consolidated multiple device management workflows into Fleet, reducing the complexity of managing separate systems. This unified approach helped eliminate silos between operating system management teams.

Fleet’s API and GitOps workflows enabled deeper automation. Using GitHub Actions, the team now automates software updates and policy deployments across the fleet. Device configurations are version-controlled and applied through automated pipelines.

The company also began onboarding Linux endpoints into Fleet. Starting with an initial group of power users, Linux systems are gradually being integrated into the same device management framework used for other platforms.

Fleet’s open-source model was also important. The ability to inspect code and extend the platform reduces vendor lock-in and allows the team to adapt the system to their infrastructure.

### A gradual migration across a massive fleet

Core components of the Fleet environment were deployed over roughly two years. This gradual rollout allowed the team to transition systems without disrupting employees or critical infrastructure.

During the transition, automatic updates and self-service software installation options improved the user experience. In many cases, employees experienced fewer interruptions compared to previous management systems.

## The results

Fleet introduced a single source of truth for device data across the organization.

Security teams now have real-time visibility into device state across operating systems. Vulnerability investigations can often be completed without contacting users directly, allowing security teams to detect and respond to threats faster.

Streaming device telemetry into internal monitoring tools also improves threat detection. Security teams can now investigate issues across macOS, Windows, and Linux simultaneously.

### Why they recommend Fleet

Fleet provides a unified and extensible platform.

Instead of maintaining separate management systems for each operating system, organizations can operate from a single control plane. This reduces operational complexity and allows IT and security teams to work together more effectively.

## About Fleet

Fleet is the single endpoint management platform for macOS, iOS, Android, Windows, Linux, ChromeOS, and cloud infrastructure. Trusted by over 1,300 organizations, Fleet empowers IT and security teams to accelerate productivity, build verifiable trust, and optimize costs.

By bringing infrastructure-as-code (IaC) practices to device management, Fleet ensures endpoints remain secure and operational, freeing engineering teams to focus on strategic initiatives.

Fleet offers total deployment flexibility: on-premises, air-gapped, container-native (Docker and Kubernetes), or cloud-agnostic (AWS, Azure, GCP, DigitalOcean). Organizations can also choose fully managed SaaS via Fleet Cloud, ensuring complete control over data residency and legal jurisdiction.

<meta name="articleTitle" value="Global collaboration platform consolidates device management with Fleet">
<meta name="authorFullName" value="Irena Reedy">
<meta name="authorGitHubUsername" value="irenareedy">
<meta name="category" value="case study">
<meta name="publishedOn" value="2026-03-03">
<meta name="description" value="A global collaboration platform uses Fleet and osquery to simplify device management and improve visibility across tens of thousands of devices.>
<meta name="useBasicArticleTemplate" value="true">

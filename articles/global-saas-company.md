# Global SaaS company modernizes device management with Fleet

A global SaaS company builds software that helps businesses manage marketing, sales, and customer service. Its platform supports thousands of employees and engineers worldwide.

To support that workforce, the company manages more than 10,000 devices across macOS, Windows, and Linux. As their engineering culture matured, they needed device management that matched the same automation and transparency standards used across the rest of their infrastructure.

## At a glance

* **Industry:** SaaS and software engineering

* **Devices managed:** 10,300+ devices (about 9,000 macOS, 1,300 Windows, and Linux engineering systems)

* **Primary requirements:** GitOps automation, real-time device visibility, precise policy targeting

* **Previous challenge:** Slow inventory updates and limited automation from legacy tooling

## The challenge

The team previously relied on Jamf for macOS management.

Over time, the platform became difficult to operate at scale. Inventory updates were slow, real-time visibility was limited, and the interface relied heavily on point-and-click workflows. These limitations made it harder for the team to automate configuration management and maintain up-to-date device data.

Linux systems were another challenge. Engineering servers and remote Linux workstations were largely unmanaged and invisible to the IT team.

The company needed a system that could manage devices with the same engineering-first approach they use to build software.

## Evaluation criteria

During their evaluation, Fleet needed to meet three requirements:

1. **GitOps and automation:** Device configurations needed to be managed through code, with version control and peer review.

2. **osquery integration:** The team required deep, SQL-based visibility into device state for compliance and security monitoring.

3. **Policy scoping and labeling:** With more than 10,000 devices, the team needed precise ways to target policies and queries across different groups.

They also wanted a single system that could manage macOS, Windows, and Linux rather than separate tools for each platform.

## The solution

Using Fleet, they now manage device policies and labels through GitOps workflows. Configuration changes are tracked in version control and applied automatically across the fleet.

The Fleet API enables deeper automation. The team runs custom queries to generate real-time compliance and inventory reports, something that was difficult with slower inventory cycles in their previous system.

Fleet also allowed them to bring Linux systems into standard device management for the first time. Engineering servers and remote Linux users are now visible and monitored alongside macOS and Windows devices.

### A careful migration at a global scale

Migrating more than 10,000 devices required a deliberate rollout.

The team took a phased approach over several weeks and months, prioritizing stability and communication with employees. Despite the scale, the transition created very little disruption.

Support tickets during the rollout remained low, demonstrating that large device fleets can migrate smoothly when the process is carefully planned.

## The results

The most immediate improvement has been real-time device visibility.

Fleet allows the team to query devices instantly and enforce compliance policies as soon as devices check in. Security teams no longer rely on stale inventory data when investigating vulnerabilities.

The platform also enables more precise device management. Labels and scoped policies allow the team to apply configurations to specific subsets of devices across their global fleet.

By consolidating tooling and moving away from Jamf’s pricing structure, the company also reduced licensing costs and redirected those resources into security automation.

### Why they recommend Fleet

For leaders evaluating device management platforms, their advice focuses on two themes: scalability and collaboration.

Fleet scales to tens of thousands of devices while maintaining a transparent development process. The open-source model allows teams to collaborate directly with the maintainers and ensure the platform evolves alongside their infrastructure.

Combined with GitOps workflows, Fleet enables device management that feels consistent with modern engineering practices.

## About Fleet

Fleet is the single endpoint management platform for macOS, iOS, Android, Windows, Linux, ChromeOS, and cloud infrastructure. Trusted by over 1,300 organizations, Fleet empowers IT and security teams to accelerate productivity, build verifiable trust, and optimize costs.

By bringing infrastructure-as-code (IaC) practices to device management, Fleet ensures endpoints remain secure and operational, freeing engineering teams to focus on strategic initiatives.

Fleet offers total deployment flexibility: on-premises, air-gapped, container-native (Docker and Kubernetes), or cloud-agnostic (AWS, Azure, GCP, DigitalOcean). Organizations can also choose fully managed SaaS via Fleet Cloud, ensuring complete control over data residency and legal jurisdiction.

<meta name="articleTitle" value="Global SaaS company modernizes device management with Fleet">
<meta name="authorFullName" value="Irena Reedy">
<meta name="authorGitHubUsername" value="irenareedy">
<meta name="category" value="case study">
<meta name="publishedOn" value="2026-03-03">
<meta name="description" value="How a global SaaS company manages 10,000+ devices with Fleet, using GitOps automation and real-time visibility across macOS, Windows, and Linux."> 
<meta name="useBasicArticleTemplate" value="true">
<meta name="cardTitleForCustomersPage" value="Global SaaS company">

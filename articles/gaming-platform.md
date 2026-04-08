# Gaming platform gains production visibility with Fleet

A global technology company operates a large-scale platform where millions of people create and share immersive digital experiences. Supporting this platform requires a distributed infrastructure that includes production servers, developer systems, and corporate devices.

To support its operations, the company manages more than 135,000 hosts across macOS, Windows, and Linux. As the platform grows, the team needs better visibility into its production infrastructure without introducing performance overhead.


## At a glance

* **Industry:** Gaming and technology

* **Devices managed:** 135,000+ hosts across macOS, Windows, and Linux

* **Primary requirements:** Infrastructure visibility, GitOps workflows, container-level telemetry

* **Previous challenge:** Limited visibility into Linux servers and containerized production environments


## The challenge

The company already used tools like Jamf to manage corporate devices. However, those tools were not designed for the scale or performance requirements of production server environments.

Linux servers and containerized workloads represented major visibility gaps. Security teams lacked reliable access to real-time data.

Gathering detailed telemetry from production systems was difficult without introducing performance overhead. The team needed a way to observe the infrastructure state without affecting the performance of game servers.

They also wanted a system that could operate consistently across macOS, Windows, and Linux systems.


## The evaluation criteria

During evaluation, Fleet needed to meet three key requirements:

1. **Infrastructure visibility**  
    Provide real-time telemetry from production servers and container environments.

2. **GitOps workflows**  
    Support configuration-as-code approaches suitable for high-stakes infrastructure environments.

3. **Advanced osquery integration**  
    Enable querying across Kubernetes and container-level workloads.

The team also prioritized a unified platform that could manage multiple operating systems through a single API.


## The solution

Fleet now provides a unified source of telemetry across the company’s infrastructure.

Using osquery through Fleet, the team gathers detailed system data from macOS, Windows, and Linux hosts. The platform also provides visibility into container environments, allowing engineers to query system state across Kubernetes clusters.

Fleet operates in a read-only GitOps configuration for sensitive production environments. This approach allows the team to gather critical telemetry and enforce compliance visibility without introducing operational risk.

Fleet telemetry feeds directly into internal security and compliance systems. Vulnerability tracking across server clusters is now automated, replacing manual processes that were previously impractical at this scale.

The platform’s open-source model also aligns with the company’s engineering culture. Security teams can inspect the source code and collaborate directly with maintainers, ensuring the system operates transparently.


## A phased rollout across production infrastructure

Fleet adoption began with pilot deployments across selected infrastructure segments.

Over time, the team integrated Fleet deeper into its DevOps workflows and internal tooling. This incremental approach allowed the organization to expand coverage without disrupting production environments.

As integrations matured, Fleet scaled to handle telemetry from hundreds of thousands of infrastructure data points in near real time.

The transition created minimal disruption for engineers, as Fleet was introduced as a natural extension of the existing infrastructure platform.


## The results

Fleet introduced near-instant visibility into infrastructure health and compliance.

Security teams can now query large volumes of device data and analyze infrastructure state within seconds. This capability dramatically improves vulnerability investigation and compliance reporting.

The platform also helps unify telemetry across corporate endpoints and production systems. Instead of maintaining separate monitoring approaches for different environments, teams now operate from a single data source.

Operational complexity has also decreased. Fleet provides a scalable way to collect telemetry across a global infrastructure without introducing heavy agents or performance penalties.


## Why they recommend Fleet

For organizations operating large-scale infrastructure, their recommendation centers on visibility and scale.

Fleet provides unified telemetry across endpoints, servers, and container environments. This visibility allows security, operations, and compliance teams to work from the same data source.

For organizations managing tens or hundreds of thousands of hosts, that level of observability becomes critical.


<meta name="articleTitle" value="Gaming platform gains production visibility with Fleet">
<meta name="authorFullName" value="Irena Reedy">
<meta name="authorGitHubUsername" value="irenareedy">
<meta name="category" value="case study">
<meta name="publishedOn" value="2026-03-04">
<meta name="description" value="How a global gaming platform uses Fleet to gain infrastructure visibility across 135,000+ hosts and container environments."> 
<meta name="useBasicArticleTemplate" value="true">
<meta name="cardTitleForCustomersPage" value="Gaming platform">
<meta name="cardBodyForCustomersPage" value="How a gaming platform uses Fleet to gain infrastructure visibility across 135,000+ hosts.">
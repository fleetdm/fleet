# Financial data company scales endpoint visibility with Fleet

A financial data and media company provides business intelligence, analytics, and global news. Its products support financial institutions, governments, and enterprises that rely on accurate, real-time information.

Supporting this infrastructure requires strong internal security and operational visibility. The company manages approximately 140,000 hosts across macOS, Windows, and Linux. At this scale, endpoint observability is critical.


## At a glance

- **Industry:** Financial data and media

- **Devices managed:** ~140,000 hosts across macOS, Windows, and Linux

- **Primary requirements:** Scalable endpoint observability, on-premise control, deep telemetry across operating systems

- **Previous challenge:** Limited visibility across Linux environments and difficult-to-deploy systems


## The challenge

The company needed a platform capable of delivering deep telemetry across its global infrastructure without introducing performance bottlenecks. Traditional endpoint tools often relied on proprietary agents that limited transparency and flexibility, making it harder to trust and verify the data being collected.

Coverage gaps also created visibility challenges. Some systems, especially Linux hosts and other difficult-to-deploy devices, lacked consistent telemetry. The team needed a platform capable of collecting reliable, real-time device data across every operating system in their environment.


## The evaluation criteria

During evaluation, Fleet needed to meet three key requirements:

1. **On-premise hosting**  
    Maintain full control of infrastructure and data to satisfy compliance requirements.

2. **osquery integration**  
    Provide SQL-based visibility across a global fleet.

3. **Open, scalable telemetry**  
    Deliver consistent endpoint data across macOS, Windows, and Linux through a single API, replacing fragmented, proprietary agents.


## The solution

Fleet now provides a unified telemetry layer across the company's device infrastructure. The security team can run real-time queries against any host in the world and retrieve system data instantly, without relying on slow legacy scanning cycles or manual IT intervention.

The platform integrates directly with internal security tooling. Endpoint telemetry from Fleet flows into vulnerability management and security monitoring systems, giving teams a continuous view of device health and compliance across the entire fleet.

Fleet's API also enables custom automation around observability. Security teams use it to run scheduled queries, collect device information at scale, and feed that data into the workflows they already rely on.

The open-source nature of Fleet was equally important. Being able to inspect and extend the platform allows the company to adapt how it collects and uses endpoint data to fit its complex, large-scale infrastructure.


### Careful rollout across 140,000 hosts

Deploying and upgrading a telemetry platform across a fleet of this size requires careful coordination.

Major migration and upgrade cycles are treated as long-term projects. One large upgrade cycle took roughly a year to complete, prioritizing stability and service continuity throughout.

During large check-in events, the system occasionally experienced high traffic spikes. The infrastructure was designed to recover quickly, typically stabilizing within 45 to 90 minutes.

This careful rollout strategy allowed the company to maintain uptime while expanding observability coverage across the organization.


## The results

Fleet introduced comprehensive endpoint visibility across the global fleet.

Security teams now access real-time telemetry instead of relying on scheduled reports. Vulnerabilities can be investigated immediately, allowing the company to respond faster to new threats and compliance requests.

The platform also reduced the need for multiple proprietary agents on each device. Consolidating endpoint telemetry into a single open platform simplified the security stack and improved operational efficiency.

With macOS, Windows, and Linux observable through a single API, teams can maintain a consistent visibility baseline across the organization, regardless of operating system.


## Why they recommend Fleet

For organizations managing large and complex infrastructures, their recommendation centers on visibility and scalability.

Fleet provides the data depth of osquery while scaling reliably across hundreds of thousands of hosts. This combination allows security teams to operate with real-time insight into device state across global environments.

For a financial data company operating in a high-compliance industry, that level of observability is essential.

<meta name="articleTitle" value="Financial data company scales endpoint visibility with Fleet">
<meta name="authorFullName" value="Irena Reedy">
<meta name="authorGitHubUsername" value="irenareedy">
<meta name="category" value="case study">
<meta name="publishedOn" value="2026-03-04">
<meta name="description" value="A global financial data company uses Fleet to gain real-time visibility across 140,000 hosts running macOS, Windows, and Linux."> 
<meta name="useBasicArticleTemplate" value="true">
<meta name="cardTitleForCustomersPage" value="Financial data company">

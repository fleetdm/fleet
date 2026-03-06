# Financial data company scales endpoint visibility with Fleet

A financial data and media company provides business intelligence, analytics, and global news. Its products support financial institutions, governments, and enterprises that rely on accurate, real-time information.

Supporting this infrastructure requires strong internal security and operational visibility. The company manages approximately 140,000 hosts across macOS, Windows, and Linux. At this scale, device visibility and compliance are critical.

---

## **At a glance**

* **Industry:** Financial data and media

* **Devices managed:** \~140,000 hosts across macOS, Windows, and Linux

* **Primary requirements:** Scalable visibility, on-premise control, GitOps workflows

* **Previous challenge:** Limited visibility across Linux environments and difficult-to-deploy systems

---

## **The challenge: maintaining visibility across a massive fleet**

The company needed a platform capable of delivering deep telemetry across its global infrastructure without introducing performance bottlenecks. Traditional device management approaches often relied on proprietary agents that limited transparency and flexibility.

Deployment gaps also created visibility challenges. Some systems, especially Linux hosts and difficult-to-deploy devices, lacked consistent coverage. The team needed a system capable of collecting reliable device telemetry across every operating system.

---

## **The evaluation criteria**

During evaluation, Fleet needed to meet three key requirements:

1. **On-premise hosting**  
    Maintain full control of infrastructure and data to satisfy compliance requirements.

2. **osquery integration**  
    Provide SQL-based visibility across a global fleet.

3. **GitOps and infrastructure-as-code workflows**  
    Enable configuration management using repeatable, engineering-driven processes.

The team also wanted a unified platform capable of managing macOS, Windows, and Linux through a single API instead of maintaining separate management systems.

---

## **The solution: scalable telemetry across the global fleet**

Fleet now provides a unified telemetry layer across the company’s device infrastructure. The security team can run real-time queries across any host worldwide. This allows engineers to retrieve system data instantly without relying on slow legacy scanning cycles or manual IT intervention.

The platform integrates directly with internal security tooling. Telemetry data flows into vulnerability management and security monitoring systems, enabling teams to analyze device health and compliance across the entire fleet.

Fleet’s API also enables custom automation. Security teams use it to run automated queries and scripts that collect device information and enforce policy checks across the organization.

The open-source nature of Fleet was also important. Being able to inspect and extend the platform allows the company to adapt the system to its complex infrastructure.

---

## **Careful rollout across 140,000 hosts**

Deploying and upgrading a platform across a fleet of this size requires careful coordination.

Major migration and upgrade cycles are treated as long-term projects. One large upgrade cycle took roughly a year to complete, prioritizing stability and continuity of service.

During large check-in events, the system occasionally experienced high traffic spikes. The infrastructure was designed to recover quickly, typically stabilizing within 45 to 90 minutes.

This careful rollout strategy allowed the company to maintain uptime while expanding device coverage across the organization.

---

## **The results: real-time visibility and stronger compliance**

Fleet introduced comprehensive device visibility across the global fleet.

Security teams now access real-time telemetry instead of relying on scheduled reports. Vulnerabilities can be investigated immediately, allowing the company to respond faster to new threats and compliance requests.

The platform also eliminated the need for multiple proprietary agents. Consolidating telemetry into a single platform simplified the device management stack and improved operational efficiency.

Fleet also helps unify security practices across operating systems. With macOS, Windows, and Linux managed through a single API, teams can maintain a consistent security baseline across the organization.

---

## **Why they recommend Fleet**

For organizations managing large and complex infrastructures, their recommendation centers on visibility and scalability.

Fleet provides the data depth of osquery while scaling reliably across hundreds of thousands of hosts. This combination allows security teams to operate with real-time insight into device state across global environments.

For a financial data company operating in a high-compliance industry, that level of observability is essential.

---

## **About Fleet**

Fleet is the single endpoint management platform for macOS, iOS, Android, Windows, Linux, ChromeOS, and cloud infrastructure. Trusted by over 1,300 organizations, Fleet empowers IT and security teams to accelerate productivity, build verifiable trust, and optimize costs.

By bringing infrastructure-as-code (IaC) practices to device management, Fleet ensures endpoints remain secure and operational, freeing engineering teams to focus on strategic initiatives.

Fleet offers total deployment flexibility: on-premises, air-gapped, container-native (Docker and Kubernetes), or cloud-agnostic (AWS, Azure, GCP, DigitalOcean). Organizations can also choose fully managed SaaS via Fleet Cloud, ensuring complete control over data residency and legal jurisdiction.

<meta name="articleTitle" value="Financial data company scales endpoint visibility with Fleet">
<meta name="authorFullName" value="Irena Reedy">
<meta name="authorGitHubUsername" value="irenareedy">
<meta name="category" value="case study">
<meta name="publishedOn" value="2026-03-04">
<meta name="description" value="A global financial data company uses Fleet to gain real-time visibility across 140,000 hosts running macOS, Windows, and Linux."> 
<meta name="useBasicArticleTemplate" value="true">

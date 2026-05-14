Untracked software creates security and compliance gaps, particularly for distributed teams managing devices across multiple operating systems. This guide covers what shadow IT discovery involves, why it matters, and how to build a discovery program using device inventory and MDM.

## What is shadow IT discovery?

Shadow IT refers to any hardware, software, or cloud service used for work without IT's knowledge or approval. Employees install tools, sign up for cloud services, and add browser extensions to get work done, often without going through an approval process. Over time, these untracked applications create gaps in security coverage and compliance documentation. Shadow IT discovery is the practice of systematically identifying these unauthorized technologies across an organization's device fleet and network.

The scope goes well beyond someone running an unauthorized server in a closet. Modern shadow IT also includes personal cloud storage, rogue APIs, and AI tools accessed through personal accounts. Major compliance frameworks like NIST CSF 2.0 call for identifying authorized and unauthorized technology as part of asset management, which means discovery often has both a security and an audit dimension.

Shadow IT typically emerges not from malicious intent but from friction. When enterprise-provided tools feel cumbersome or IT response times lag, employees find alternatives. That makes discovery not just a technical challenge but an organizational one, and understanding why people reach for unapproved tools often matters as much as finding the tools themselves.

## Why shadow IT discovery matters for security and compliance

Untracked software creates practical problems for IT and security teams. If you don't know something is installed, you can't patch it, monitor it, or account for it during audits. For teams supporting incident response or compliance programs, discovery work is one of the fastest ways to close those gaps.

Several categories of risk make shadow IT discovery worth prioritizing:

- Compliance gaps: Many security and compliance programs expect accurate inventories and clear ownership of in-scope assets and services. Frameworks like NIST CSF 2.0, NIST 800-53, and ISO 27001 explicitly call for asset inventories, and auditors may ask you to reconcile documented inventories with evidence from device or network controls. Discrepancies can trigger follow-up.
- Unpatched software: You can't patch what you don't know about. Applications outside your management tooling tend to fall outside vulnerability management processes too, leaving known exploits open.
- Data leaving your controls: Unauthorized cloud storage, AI tools, and browser extensions can move sensitive data outside organizational boundaries. When employees paste customer data into personal AI accounts, you lose visibility into how that data gets stored or shared.
- Incident response blind spots: If you don't know an application exists on a device, you can't monitor it for suspicious behavior or correlate its logs during an investigation. These gaps tend to slow response times, especially across remote workforces.
- Regulatory exposure: Some regulations, such as GDPR, carry penalties tied to global revenue. Shadow IT that processes personal data outside approved channels may not surface until an audit or breach forces it into the open.

Each of these risks compounds when left unaddressed, and organizations with distributed workforces often find that their actual cloud service count far exceeds what IT tracks. Building visibility into shadow IT is the first step toward managing it.

## How does shadow IT discovery actually work?

Effective shadow IT discovery requires layering multiple detection methods because no single approach covers every blind spot. If you only use one signal (like network logs or an MDM app list), you'll miss entire categories of shadow usage.

The right mix depends on how devices connect to the internet, which devices you manage, and whether traffic regularly traverses corporate infrastructure.

### Device-based discovery

Agent-based discovery deploys software directly on managed devices to capture application and network activity at the operating system level. For remote and hybrid workforces, device-based discovery is often the most reliable way to maintain consistent visibility, especially when your workforce rarely connects through corporate VPN.

Device agents can detect installed applications, running processes, browser extensions, listening ports, and outbound network connections. Because the agent sits on the device itself, it can capture shadow IT activity even when a device operates outside the corporate network. The tradeoff is operational: you have to deploy and maintain agents on every device that needs coverage, and unmanaged or bring your own device (BYOD) devices remain invisible unless enrollment is required or an agent can be installed.

### Network-based traffic analysis

Network-based discovery collects and analyzes logs from firewalls, proxies, VPN gateways, and routers. This provides infrastructure-wide visibility into which cloud services and external applications users access. When an organization routes web traffic through a proxy or VPN, analysts can often identify SaaS applications by analyzing DNS queries and HTTP/HTTPS traffic patterns.

The limitation is architectural: network-based discovery can't see traffic that never crosses corporate infrastructure. If you have remote employees connecting directly to cloud services from home networks, they bypass these detection points unless traffic is routed through a VPN. Encrypted HTTPS traffic also requires SSL inspection for deep visibility, which introduces its own complexity and privacy considerations.

### Identity and SaaS audit logs

A third signal often gets overlooked: identity provider and SaaS admin logs. When you enforce single sign-on (SSO) for approved tools, authentication logs can surface sign-ins to unexpected applications, risky personal-account usage patterns, and sudden spikes in OAuth grants.

This approach won't show everything, especially when employees log into shadow apps with personal email accounts. Still, it can answer questions that device inventory can't, like which departments are actually using a new SaaS tool and which accounts are accessing it.

### Hybrid approaches

The most effective discovery programs combine multiple signals rather than relying on any single method. Device-based discovery is often the critical primary method because it covers devices that operate off-network, while network analysis provides infrastructure-wide visibility into cloud services being accessed. Identity and authentication logs add context about which teams and accounts are involved.

Starting with device-based visibility means teams can start building coverage even when employees work off corporate networks or use SaaS apps outside the identity provider. Layering network monitoring on top adds a map of external services in use. Each method compensates for the gaps the others have, which is why organizations with effective discovery programs typically use at least two approaches together.

## When shadow IT discovery becomes a priority

Not every organization needs a comprehensive discovery program on day one. For teams handling regulated data under frameworks like HIPAA, GDPR, SOC 2, or PCI-DSS, discovery is usually a compliance requirement since auditors may ask you to reconcile what's documented with what's actually running. Distributed workforces accelerate the need because visibility gaps grow quickly when employees work from home networks and personal devices.

Growing IT complexity is another signal: when teams adopt cloud services independently and security investigations regularly turn up unknown applications mid-incident, reactive discovery has already become the default. The general pattern for getting started is to establish a device inventory baseline, layer in deeper querying to find what falls outside your approved software list, classify findings by risk, and then address root causes rather than just blocking tools. If employees reach for shadow IT because approved tools are slow or poorly supported, blocking alone tends to push the behavior further underground.

## How device inventory and MDM support shadow IT discovery

MDM enrollment data and device inventory create the baseline against which shadow IT becomes visible. If you can't reliably answer which devices are managed and which ones aren't, discovery results become harder to interpret. Without that baseline, it's also harder to decide where to focus your discovery effort.

### What device management tells you

MDM solutions like Fleet collect installed application lists, device configurations, and policy or compliance state from enrolled devices. On macOS, iOS, and iPadOS, MDM can provide application inventory, though the scope varies by enrollment type. On Windows, MDM can surface app inventory details like name, version, and publisher. This data gives you a starting point for comparing what's approved against what's actually present.

MDM visibility is scoped to what's necessary for policy enforcement and security, not everything the device does. Most organizations want to know whether a device is compliant and whether sensitive data is leaving approved channels, not a full record of every website an employee visits.

### Going deeper with osquery

Fleet combines MDM with `osquery` to extend visibility beyond standard application inventory. SQL-based querying tools like `osquery` treat the operating system as a database, letting teams ask questions about what's installed, what's running, and what's communicating across macOS, Windows, and Linux devices. Instead of relying on what an MDM tool reports, you can query directly for installed applications, browser extensions, running processes, and outbound network connections using consistent syntax across platforms.

This approach is particularly useful for shadow IT discovery because it covers categories that traditional device management often misses. For example, you can inventory extensions and, where available, review their declared permissions or access patterns. You can also correlate running processes with active network connections to identify software communicating with external services that aren't on an approved list. These kinds of queries surface tools that often don't appear in standard application inventories, such as portable apps, user-installed utilities, or extensions that quietly move data outside your controls.

When an organization maintains an approved software list, SQL-based querying makes it possible to compare what's actually present on devices against that list on a recurring basis, turning discovery from a one-time effort into a continuous process. Fleet handles this through policies: yes/no SQL-based checks that run on a configurable cadence (hourly by default) across enrolled hosts. When a policy fails (for example, detecting unapproved software), Fleet can trigger automatic remediation. That might mean installing approved software, running a script to remove unapproved software, or opening a ticket in Jira, Zendesk, or ServiceNow.

For known vulnerabilities, Fleet matches installed software versions against the National Vulnerability Database to flag known CVEs, including for browser extensions and Python packages. Patch policies generate version-check queries for Fleet-maintained apps and trigger software installation when a device falls behind.

[Fleet's device management](https://fleetdm.com/device-management) provides this visibility across macOS, Windows, and Linux from a single console. This deeper visibility is especially relevant on Linux, where no standardized MDM protocol exists. Organizations managing Linux laptops or developer workstations typically deal with multiple package managers, manual installations from source, and containerized applications. Each represents a different vector for shadow IT.

## Shadow IT discovery across your device fleet

Combining device management with queryable device data makes shadow IT discovery practical across mixed operating system environments. Fleet uses `osquery` to provide SQL-based visibility across macOS, Windows, and Linux from a single console. That includes installed applications, browser extensions for Chromium-based browsers (Chrome, Edge, Brave, Opera), Safari, and Firefox, running processes, and network connections. For SaaS discovery, combine device signals with identity provider and network logs.

Fleet supports [GitOps workflows](https://fleetdm.com/fleet-gitops) through declarative YAML configuration applied via `fleetctl gitops` in CI/CD pipelines (GitHub Actions, GitLab), with built-in drift correction. [Get a demo](https://fleetdm.com/contact) to see how Fleet supports shadow IT discovery across your fleet.

## Frequently asked questions

### What types of software count as shadow IT?

Any technology used for work without IT's knowledge or approval qualifies. The most common categories today are SaaS applications, browser extensions, personal cloud storage, AI tools accessed through personal accounts, and developer utilities installed outside official channels. Browser extensions are often overlooked but can pose significant risk because of the data access permissions they request.

### Can shadow IT discovery work without VPN or corporate network access?

Yes, but the discovery method matters. Network-based approaches only see traffic that crosses corporate infrastructure, so they lose visibility for remote workers. Device-based discovery through agents on managed devices captures activity regardless of network, which is why most organizations with distributed workforces rely on agent-based approaches as their primary signal.

### How often should shadow IT discovery run?

It depends on the asset type. Many teams run extension and app inventories daily; higher-risk environments may run more frequently. Network traffic analysis typically works best as a continuous feed rather than periodic scans, since short-lived connections can disappear between scheduled checks, though some teams use sampling or scheduled collection to balance cost and privacy constraints.

### Does shadow IT discovery require MDM enrollment on all devices?

No. MDM provides a useful starting point, but effective discovery doesn't depend on it. On Linux, SQL-based querying through tools like `osquery` is often the primary discovery method since there's no standardized MDM protocol. For macOS and Windows, layering device querying on top of MDM covers the gaps in standard application inventories. Fleet combines device management with `osquery` to provide both. To see how the pieces fit together, [contact us](https://fleetdm.com/contact).

<meta name="articleTitle" value="Shadow IT discovery: Finding unapproved software across your device fleet">
<meta name="authorFullName" value="Ashish Kuthiala">
<meta name="authorGitHubUsername" value="akuthiala">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-05-14">
<meta name="description" value=" Learn how shadow IT discovery works, why it matters for security and compliance & how to build a discovery program using device inventory and MDM.">

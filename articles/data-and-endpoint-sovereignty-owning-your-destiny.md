# Data and Endpoint Sovereignty: Owning Your Destiny

### Links to article series:

- Part 1: [Why enterprise Linux is important in 2026](https://fleetdm.com/articles/why-enterprise-linux-is-important-in-2026)
- Part 2: [Automated provisioning for Linux desktop in the enterprise](https://fleetdm.com/articles/automated-provisioning-for-linux-desktop-in-the-enterprise)
- Part 3: [Security baselines for Linux: closing the gap on exemptions](https://fleetdm.com/articles/security-baselines-for-linux)
- Part 4: [Unlocking Linux productivity: securing apps and updating certificates](https://fleetdm.com/articles/unlocking-linux-productivity-securing-apps-and-updating-certificates)
- Part 5: [Protecting the Linux device: remote wipe, USB and sudo](https://fleetdm.com/articles/protecting-the-linux-device-remote-wipe-usb-sudo)
- Part 6: Data and Endpoint Sovereignty: Owning Your Destiny

-----

The SolarWinds attack, uncovered in December 2020, was a highly sophisticated supply chain cyberattack in which threat actors compromised updates to SolarWinds’ Orion network management software. By inserting malicious code into legitimate software updates, the attackers (widely attributed to a state-sponsored group) gained covert access to the networks of thousands of organizations, including U.S. government agencies and major corporations.

The incident exposed critical vulnerabilities in the software supply chain generally and challenged core assumptions around software security. The malicious software was distributed by Solarwinds. The pacakge had been signed and deployed with valid certificates. Victims followed the best practice of deploying vendor updates. So, what went wrong?

Development is increasingly complex. Package managers like `npm`, `pip`, `brew`, et al, along with the complex toolchains and dependencies required to build modern software have resulted in a broad, attackable surface that can only be defended with purposeful strategy.

There are two basic approaches to manage and contain this type of risk.

The first is to use software “bills of material” (or an SBOM). Using a BOM explicitly specifies a software build and the components used in it. BOM files can be audited for conformance to specifications and for optimization. This is strict and potentially limits developer freedom and creativity.

The second is a movement towards full transparency by selecting fully open-source software. Open-source software lets teams inspect, audit, and modify the code they depend on. Open-source optimizes for transparency by publishing all source code and configuration files required to build software in public view. Instead of relying on a vendor to specify the Software BOM, organizations can audit and inspect open-source code directly as needed or even suggest changes to it.

## Data Sovereignty

With the emergence of cloud computing and the always-connected nature of endpoints, the control over data and its security is no longer limited to physical presence. Instead, companies have to trust software vendors who are accountable to multiple stakeholders and who operate in numerous legal jurisdictions. What happens if a national government or intelligence agency mandates access to proprietary and confidential information from a software vendor? What happens if countries sanction technology from being used by their adversaries?

The concept of data sovereignty arises from the desire to make sure that companies can trust the software that their servers and endpoints run on. Open-source software like Linux is attractive because it is free from much of the encumbrance around proprietary software. There are no vendors that control access to the OS and its dependencies. 

### The open source security advantage

If an organization uses proprietary products it faces a fundamental constraint: teams can trust vendors, but they can't always verify vendor claims. Proprietary operating systems and management tools are black boxes. When a vulnerability surfaces, organizations often have to wait for the vendor to acknowledge it, patch it, and release a fix on the vendor's timeline.

Open source changes this dynamic. Teams can audit source code, verify behavior, and identify vulnerabilities independently. If a critical flaw needs patching before an upstream fix arrives, it can be patched internally or by forking the project.

### Transparency enables detection but isn't enough on its own

In 2021, a critical remote code execution vulnerability known as [Log4Shell](https://www.cisa.gov/news-events/cybersecurity-advisories/aa21-356a) was discovered in Apache Log4j, a widely used open source Java logging library. The flaw had been present in the codebase since 2013 and affected a very large share of internet-connected systems. Because Log4j is open-source, researchers could immediately inspect the vulnerable code, understand how the attack worked, and share detection tooling publicly within days. Organizations didn't have to wait for a single vendor's response.

This kind of rapid, distributed response is harder to achieve when the affected software is closed source and only the vendor can inspect or patch it. But the incident also illustrates that openness alone isn't sufficient. It highlighted ecosystem issues like under-resourced maintainers, uneven security practices across projects, and the difficulty of coordinating response across a sprawling dependency tree. Open-source provides the conditions for detection, but realizing that benefit requires active investment in review processes and contributor trust models.

Community involvement adds another layer of defense. When organizations rely on widely adopted projects like the Linux kernel, they benefit from large numbers of contributors and reviewers examining changes. Industry initiatives focused on open-source security invest in vulnerability identification, supply chain tooling, and developer education.

### Why transparency matters for data sovereignty

Outcomes still depend on active maintenance, supply chain verification, and an organization's investment in security practices. But open source allows for direct access to verify code behavior, including independent source code review and vulnerability analysis, in a way that most proprietary software does not.

For organizations where data sovereignty matters, this is critical. During audits, teams may need to demonstrate what software is doing, where data goes, and what evidence supports those claims. Open source makes those questions answerable. Closed-source software typically does not.

## Linux follows the enterprise trend

Linux on the workstation isn't a departure from enterprise norms. It's a continuation of choices most organizations have already made.

### Environment parity from development to production

When organizations choose Linux workstations for developers, the local environment can align more closely with what gets deployed to production. If servers run Linux, containers run Linux, and CI/CD pipelines run on Linux, there's an argument for developer workstations to match. 

This can reduce friction between development and deployment and give engineering teams a more consistent toolchain from local development through production. That said, environment parity isn't the only factor in platform decisions. Application compatibility, end user support burden, and the maturity of management tooling all play a role.

### Open source runs the enterprise

Many organizations already depend on these projects:

- Nginx and Apache together handle a large share of the world's web traffic
- Chromium is the open-source browser project on which Chrome, Edge, Brave, and others are built
- OpenSSL secures encrypted connections on the internet
- PostgreSQL and MySQL and SQLite are commonly used in applications and enterprise database deployments
- Kubernetes is a common choice for orchestrating container workloads

These are familiar, foundational technologies, and they're all open-source. The broader pattern worth noting is that open-source already underpins critical workloads, everywhere. 

When that pattern extends to the desktop, Linux adoption can look less like a departure and more like a continuation of choices organizations and technology practitioners have already made elsewhere. This doesn't mean every organization is ready for it, or that the transition is frictionless, but it does mean the conversation has shifted from whether open source belongs in the enterprise to how far it can be extended.

## Managing Linux with Linux values

Linux is often chosen for openness and transparency. For organizations that value those qualities, it's worth considering whether the management tooling applied to Linux devices reflects the same principles.

The marketplace has proven Linux adoption in the enterprise. The trend is growing and will likely increase. The only question is how will organizations plan to manage Linux adoption with Linux values at enterprise scale and what tools are available to achieve it?

### The visibility trade-off with proprietary management tools

Proprietary management tools can create the same black-box problem that Linux avoids at the operating system level. Teams gain visibility into the OS through open source, but may lose that visibility in a closed management layer where they can't inspect how devices are monitored, verify what data is collected, or modify workflows to fit their environment. For organizations, choosing a closed management tool may also work against the flexibility that drew them to Linux in the first place.

Open source management tools shift this equation. Teams can inspect monitoring logic, verify data collection practices, extend functionality for specific needs, and export data if they ever decide to move to a different device management solution. For organizations that treat auditability and data control as requirements, this matters.

[Fleet](https://fleetdm.com/device-management) is built for organizations taking this approach. Fleet is [open source](https://fleetdm.com/handbook/company/why-this-way#why-open-source). Source code is publicly available for both its free and premium versions. Anyone can verify how it works. Fleet is built on [`osquery`](https://fleetdm.com/tables/account_policy_data), an open source project with broad community adoption that expresses operating system data as queryable database tables, helping turn Linux workstations from a potential blind spot into a rich source of device telemetry. Multi-platform support means Fleet can manage Linux devices alongside macOS, Windows, Chromebook, iOS, iPadOS, and Android from a single console, so IT teams aren't forced to have a segregated Linux management silo.

Fleet offers both [self-hosted and cloud-hosted deployment](https://fleetdm.com/docs/deploy/deploy-fleet), giving teams more control over where device data lives, who can access it, and how it's processed. For teams operating under GDPR, HIPAA, or FedRAMP requirements, that control directly supports data sovereignty obligations. Native [GitOps workflows](https://fleetdm.com/fleet-gitops) built into the [fleetctl](https://fleetdm.com/guides/fleetctl#basic-article) binary let teams manage device policies through version-controlled `YAML` files, aligning with how many Linux administrators already work and providing the kind of audit trail that compliance teams increasingly require.

## Design your Linux management strategy for the future

This article series started with a simple premise: Linux desktop adoption has grown significantly. Organizations need to understand that it is happening, why Linux desktop matters and how to build a management strategy around provisioning automation, security baselines, software and certificate deployment and device data.

The throughline is the philosophy that makes Linux valuable in the first place: openness, transparency, and control. Fleet brings enterprise device management to Linux while honoring those principles through transparent code, flexible deployment, and GitOps-friendly workflows. [Talk to Fleet](https://fleetdm.com/contact) today about defining your Linux device management strategy.

<meta name="articleTitle" value="Data and Endpoint Sovereignty: Owning Your Destiny">
<meta name="authorFullName" value="Ashish Kuthiala">
<meta name="authorGitHubUsername" value="akuthiala">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-03-16">
<meta name="description" value="Part 6 of 6 in the 'Protecting Linux endpoints with modern device management' article series.">

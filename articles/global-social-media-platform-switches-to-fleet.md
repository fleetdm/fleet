# Global social media platform switches to Fleet for workstation telemetry

<div purpose="attribution-quote">

Context is king for device data, and Fleet provides a way to surface that information to our other teams and partners.

**- Systems and infrastructure manager**
</div>

##  Challenge

One of the largest social media platforms sought to enhance its telemetry capabilities to maintain strict compliance and security without compromising data accessibility and operational efficiency. Managing thousands of devices across multiple platforms had led to fragmented visibility and reliance on manual data handling processes, which were time-consuming and error-prone. Additionally, the existing solution failed to provide actionable insights without significant customization, hindering proactive operations and complicating compliance with ongoing [ISO27002](https://www.iso.org/standard/75652.html) and [SOC audit](https://en.wikipedia.org/wiki/System_and_Organization_Controls) requirements.

## Solution

The social media platform transitioned to Fleet, consolidating under a single, [multi-platform](https://fleetdm.com/orchestration) system that supports macOS, Linux, and Windows. Fleet's compliance features made it easier to meet regulatory standards and protect sensitive data. Leveraging Fleet's real-time reporting and flexible data management capabilities eliminated the need for manual workflows and extensive customizations. Additionally, deploying [osquery](https://osquery.io/) independently from their existing EDR enhanced data accuracy and reliability, providing more accurate measurement against other benchmarks like [CIS](https://www.cisecurity.org/cis-benchmarks), and standardized osquery operations across their entire fleet.

## Results

<div purpose="checklist">

Verifiable compliance

Cross-team data accessibility

Real-time insights

Standardized processes
</div>

By switching to Fleet, they were able to institute more stringent compliance policies, verify security posture, gather real-time insight into ongoing operations, and standardize these processes across their growing fleet of diverse devices and teams.


## Their story

This social media platform is one of the largest globally, connecting thousands of communities and millions of users. With a vast user base and a significant global presence, effective visibility is essential to maintaining their required performance, security, and compliance standards.

The decision to switch to Fleet was driven by a few key factors. Strict adherence to compliance standards, balancing proactive and reactive security measures to protect sensitive data, and streamlining device data accessibility. By making data easily accessible and parsable, Fleet eliminated the inefficiencies of manual workflows and addressed communication gaps across teams, enabling better, faster decision-making.


With Fleet, they achieved this through:

- Eliminating tool overlap

- Definitive data for compliance and real-time reporting

- Robust API and webhook support

- A Unified reporting language

### Eliminate tool overlap

Fleet’s centralized platform enabled the combination of device operations across macOS, Windows, and Linux with a [unified reporting language](https://fleetdm.com/docs/deploy/reference-architectures#mysql) that provides flexibility and contextualized data. By adhering to standard data shapes and formats, Fleet makes sure that data is easily interpretable and usable across various teams and applications while serving as the central hub for security data.

### Definitive data for compliance

Fleet’s live query engine streams easily accessible and parsable [data](https://fleetdm.com/tables/account_policy_data), eliminating the need for manual exports and data consolidation from multiple tools. With accurate visibility and access across teams, remediations based on information directly from each device lead to fewer infrastructure failures and auditing errors.

### Robust API and webhook support

Fleet’s API facilitates real-time [compliance](https://fleetdm.com/queries) auditing and reporting, allowing the team to respond promptly to potential issues by combining data from different tools. The [API](https://fleetdm.com/docs/rest-api/rest-api) and webhook features enable automation and integration with existing systems, eliminating the need for extra middleware and reducing reliance on manual configurations.

### Unified reporting language

Fleet's straightforward deployment of the osquery agent across their devices as an independent element ensured data accuracy and reliability while standardizing its operations across macOS, Windows, and Linux. This allowed the social media platform to inspect, collect, fix, install, patch, and program just about anything, every minute of the day, on any computer in their infrastructure, with an unnoticeable performance impact


## Conclusion

Transitioning to Fleet provided the platform with a strategic solution that addressed its critical needs for compliance, security, data accessibility, and operational efficiency. Fleet's cross-platform support and open-source transparency set it apart from competitors, providing a single source of truth for all devices.

<call-to-action></call-to-action>

<meta name="category" value="announcements">
<meta name="authorGitHubUsername" value="Drew-P-drawers">
<meta name="authorFullName" value="Andrew Baker">
<meta name="publishedOn" value="2024-12-16">
<meta name="articleTitle" value="Global social media platform migrates to Fleet">
<meta name="description" value="Global social media platform migrates to Fleet">

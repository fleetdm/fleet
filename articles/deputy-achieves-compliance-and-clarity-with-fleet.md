# How Deputy achieved compliance and clarity with Fleet—keeping shift work in sync

<div purpose="attribution-quote">

“We were using Fleet to get some accurate reporting on browsers that people are using - it's useful to have that clear picture when we go and talk to SLT and can back those decisions up with some actual stats” 

**- John Howell, Director of IT**
</div>

## Challenge

[Deputy](https://www.deputy.com/), a global leader in workforce management software, needed a reliable way to capture device telemetry, troubleshoot issues, and ensure accurate reporting on OS and software updates to maintain SLA compliance. The increasing number of software applications and browser extensions introduced additional complexity, leading to compliance challenges and gaps across cross-functional teams.

## Solution
Deputy immediately leveraged Fleet’s robust [API](https://fleetdm.com/docs/rest-api/rest-api) to streamline reporting and enhance visibility across their infrastructure. The engineering team quickly automated reporting processes, delivering regular snapshots of their hosts directly into [Slack](https://slack.com/) channels. This provided security and operations teams with the transparency needed to monitor system health effectively. Using creative solutions, the team built a ‘rolling’ delta to track changes as OS updates were released and patched, enabling real-time updates to the Director of Security.

Previously reliant on [Kolide](https://www.kolide.com/), Deputy reduced costs by transitioning to Fleet while benefiting from hands-on support and direct access to Fleet’s engineers. They spun up a [dedicated Fleet instance](https://fleetdm.com/docs/deploy/deploy-fleet) on their managed infrastructure, tailoring configurations and deployments to meet the unique needs of their organization.

<div>

“We want to use Fleet to specifically build a catalog of what's currently in use across our hosts. I've said to the team, get that reporting out of Fleet. Let's see what people are using. if we found something that we weren't happy with through that reporting, it'd be quite useful to pick that up."

**- John Howell, Director of IT**
</div>

## Results

<div purpose="checklist">

Automated reporting and transparency

OS change tracking 

Quick troubleshooting of host issues

Cost savings and efficiency
</div>

Fleet provided [real-time visibility](https://fleetdm.com/orchestration) into security posture and operational performance, enabling the IT operations team to proactively address issues and stay ahead of potential risks. Fleet also streamlined processes, allowing Deputy to maintain consistency and control across their rapidly expanding fleet of global devices, supporting their diverse teams with a unified approach to security and compliance. End user experience is always top of mind at Deputy, and Fleets lightweight agent and minimal performance impact allowed the agent to be deployed quickly and confidently.


## Deputy’s Story

Headquartered in Australia, Deputy is rapidly expanding its global presence with offices in Sydney, San Francisco, and London. With a growing, diverse workforce, they needed a centralized platform to provide comprehensive insights into the health and security posture of their operations worldwide. By switching to Fleet, Deputy gained a new level of visibility and control over their devices, enabling them to save time on implementing new processes and proactively managing their fleet.

They achieved this through:

- API-Driven reporting and automation
- Comprehensive device health querying
- Enhanced endpoint visibility
- Flexible deployment options

### API-driven reporting and automation

Deputy’s Corporate Engineering team recognized the potential of automating routine compliance and reporting tasks. With Fleet, they streamlined their reporting workflows, enabling quick generation of [compliance](https://fleetdm.com/queries) reports and real-time tracking of device status. This automation significantly reduced manual effort and made it easier to respond to auditor requests for [ISO27001](https://www.iso.org/standard/75652.html) and [SOC2](https://en.wikipedia.org/wiki/System_and_Organization_Controls) compliance documentation.

### Comprehensive device health querying

With Fleet’s robust osquery capabilities and extensive library of pre-built queries, Deputy was able to ask questions about a device that was previously not available as easily. Engineers could now easily check the status of EDR tools, monitor memory-intensive processes, assess battery health and cycle counts, and much more - enabling them to quickly address issues as soon as they appeared in the helpdesk.

### Enhanced endpoint visibility

For Deputy’s CorpEng and Trust teams, having visibility into the software and packages installed on every device is essential for proactive security. Fleet’s aggregation of installed software helped Deputy quickly identify and [mitigate vulnerabilities](https://fleetdm.com/software-management), including high-priority zero-day exploits like the [XZ Utils issue](https://en.wikipedia.org/wiki/XZ_Utils_backdoor), ensuring a rapid response to threats.

### Flexible deployment options

When evaluating tools, Deputy wanted the ability to manage their own infrastructure in AWS, ensuring a flexible deployment path that aligned with their infrastructure-as-code approach. This allowed them to right-size their deployment, optimizing costs and resources. The self-hosting option allowed Deputy security teams to wire in existing Cloud Security Posture Management tools to observe misconfiguration detection and continuous monitoring of cloud resources.


## Conclusion

By switching to Fleet, Deputy gained a powerful, flexible solution that addressed their need for centralized device visibility, streamlined compliance reporting, and proactive security management. Fleet’s robust API, real-time telemetry, and flexible deployment options empowered Deputy to automate processes, reduce operational overhead, and improve their security posture. With greater insight into their devices, Deputy can confidently support their growing global workforce.

<call-to-action></call-to-action>

<meta name="category" value="announcements">
<meta name="authorGitHubUsername" value="harrisonravazzolo">
<meta name="authorFullName" value="Harrison Ravazzolo">
<meta name="publishedOn" value="2024-12-17">
<meta name="articleTitle" value="How Deputy achieved compliance and clarity with Fleet—keeping shift work in sync">
<meta name="description" value="How Deputy achieved compliance and clarity with Fleet—keeping shift work in sync">
<meta name="showOnTestimonialsPageWithEmoji" value="🚪">

# How Fleet helps federal agencies meet CISA BOD 23-01

![BOD 23-01](../website/assets/images/articles/BOD-23-01-800x450@2x.jpg)

Recently, the Cybersecurity and Infrastructure Security Agency (CISA) published [Binding Operational Directive 23-01](https://www.cisa.gov/binding-operational-directive-23-01). The directive’s goal is to improve asset visibility and vulnerability detection for the Federal Civilian Executive Branch (FCEB) enterprise. FCEB agencies have until April 3, 2023 to meet or exceed BOD 23-01’s requirements.

What does this mean for FCEB agencies? Stronger security postures. That’s reassuring considering these agencies include the Department of Energy, the Department of the Treasury, and the Department of Health and Human Services.

What will it take to get there? Comprehensive, continuous reporting. The frequency and scope of these reports might seem daunting. But BOD 23-01 doesn’t have to disrupt your agency’s operations. Fleet will help you meet these requirements quickly and easily. Yes, really.

## Wrangle roaming devices

BOD 23-01 requires agencies to begin vulnerability enumeration across all discovered assets every 14 days. This includes roaming and remote devices.

Roaming devices pose a problem for traditional vulnerability scanners. A server can only scan online, connected devices. This isn’t too high a hurdle for computers at the office, but remote devices must connect to the server with a VPN. And VPNs diminish internet speeds. Since nobody likes working with slow internet, remote employees might avoid using a VPN. This could cause the server to overlook roaming devices.

You don’t have to worry about missing remote devices with Fleet. Our agent was designed to fetch data from hundreds of thousands of devices across the globe. No VPN necessary. [Read our deploying documentation](https://fleetdm.com/docs/deploying/faq#what-api-endpoints-should-i-expose-to-the-public-internet) to see which endpoints you should expose to manage devices outside your VPN or intranet.

## Receive regular updates

Slowed remote devices aren’t the only inconvenience that comes with traditional vulnerability scanners. Your security team has to schedule each scan. That’s easier said than done. 

Servers only scan online devices, but these scans can strain CPUs. To avoid performance issues, you could set up desktops to be on when employees are off the clock. But on which day (or night) of the week should you run your scan? Operating systems and software applications release updates on their own schedules. You need to plan around these releases to make sure you’re pulling the latest data.

Fleet can save you the trouble. Once a device has been added to Fleet, you’ll automatically receive data every hour. That’s right. Forget about scheduling scans at 2 am. Of course, you have the freedom to schedule queries whenever you want. You can even fetch data in real time — without the CPU strain.

## Find vulnerabilities easily

Fleet makes it easy to collect the exact endpoint data you need, whether you want to write your own queries or [copy and paste queries from our library](https://fleetdm.com/queries). Identifying vulnerabilities in particular should be as simple as possible.

Fleet Premium provides rich software vulnerability data by querying devices against Common Vulnerabilities and Exposures (CVEs). This includes CVSS and EPSS scores, as well as CISA’s Known Exploited Vulnerabilities (KEV) Catalog.

This data not only helps your agency satisfy BOD 23-01’s requirements, but it also helps your security team prioritize devices that require remediation — so engineers, analysts, and admins can make the most of their time.

## Deliver data automatically

Enumerating assets and vulnerabilities isn’t enough to comply with BOD 23-01. FCEB agencies must send this data to CISA’s CDM Federal Dashboard. Making sure agencies have done their homework isn’t the only purpose of this requirement. This information empowers CISA to make future recommendations to further improve federal cybersecurity.

That still means more work for federal security teams. Let Fleet lighten the load. Fleet integrates with leading log destinations, like AWS Kinesis, Snowflake, and Splunk. Schedule automated queries to retrieve data as often as you’d like. Say, every 7 or 14 days. Your agent sends this data to your log destination, which submits it to the CDM Federal Dashboard.

What if your agency wants to limit the number of third-party vendors? Fleet has you covered. The REST API gives you the option to create your own pipeline — providing more control and peace of mind.

## Protect device performance

Every organization wants to ensure stability. But device performance takes on greater importance for agencies that deal with public health or power plants. We’ve mentioned the shortcomings of traditional vulnerability scanners. Those performance hits hurt a lot more if your systems need to be online and fast 24/7.

Fleet’s security agent, osquery, has a lightweight resource footprint. If a query is set to exceed a certain RAM threshold, then the query will be canceled before any devices are affected. We call this the osquery watchdog. Under the default configuration, the watchdog will ensure that utilization stays below 200 MB of memory and 10% CPU. If a query is canceled, you’ll receive a notification that offers suggestions to lower its impact.

For queries that have been run before, Fleet also gives you the ability to measure an estimated performance impact directly on the Queries page. You’ll be able to see the average impact rating across all hosts where this query was scheduled.

## Enjoy simple implementation

Fleet can fit into the security ecosystem of any federal agency. Some SaaS solutions have limitations about where they can be deployed. That’s non-negotiable for agencies handling highly sensitive information. And that isn’t a problem for Fleet. You can deploy Fleet anywhere — including cloud.gov. [Learn how to deploy Fleet to cloud.gov](https://fleetdm.com/docs/deploying/cloudgov) in our documentation.

The scope of just one FCEB agency can be quite broad. Complex organizational structures are a natural result. Fleet lets you assign devices to specific teams. Each team can have its own queries, schedules, and policies. So, you can tailor compliance standards to specific departments.

You shouldn’t have to trade one tool for another, only to discover it can’t do the job. Fleet provides visibility into servers and workstations at scale. You can easily identify vulnerabilities and misconfigurations on every laptop in your agency. But you can’t scan routers. That’s why we recommend deploying Fleet in addition to software you already use, like Rapid7 or Tableau. Fleet’s up-to-the-minute data can supplement reports from these trusted platforms.

## Comply with BOD 23-01

Fleet helps Fortune 1000 companies achieve compliance with internal guidelines and government regulations. The ability to log historical data and run real-time queries lets you address CISA requests quickly and accurately. Custom [policies](https://fleetdm.com/securing/what-are-fleet-policies) allow your agency to adjust enforcement as federal requirements change. This puts your agency in the position to comply with BOD 23-01 and any other directives to come.

There’s no better way to vet a vendor than to use the platform yourself. See how Fleet can help federal agencies. [Try Fleet on your device for free](https://fleetdm.com/try-fleet/register).

<meta name="category" value="security">
<meta name="authorFullName" value="Chris McGillicuddy">
<meta name="authorGitHubUsername" value="chris-mcgillicuddy">
<meta name="publishedOn" value="2022-10-28">
<meta name="articleTitle" value="How Fleet helps federal agencies meet CISA BOD 23-01">
<meta name="articleImageUrl" value="../website/assets/images/articles/BOD-23-01-800x450@2x.jpg">

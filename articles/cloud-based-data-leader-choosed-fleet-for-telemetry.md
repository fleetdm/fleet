Cloud-based data leader chooses Fleet for telemetry

<div purpose="attribution-quote">

I wanted an easy way to control osquery configurations, and I wanted to stream data as fast as possible. No other solution jumped out to solve those things except for Fleet.

**- IT Engineering Manager**
</div>

## Challenge

A leader in cloud-based data platforms, needed to modernize device management for tens of thousands of endpoints while maintaining performance and cost efficiency. Legacy device management tools caused bottlenecks by delivering data updates only every 24 hours, limiting their ability to monitor and optimize device performance. Additionally, a lack of seamless cross-platform compatibility and dependency on proprietary systems increased operational complexity and hindered their IT and operations teams.

## Solution

They transitioned to Fleet for centralized, high-frequency data collection without reliance on traditional MDMs. By leveraging Fleet’s seamless integration with its existing infrastructure, including AWS Kinesis Firehose, they gained the ability to process [osquery](https://osquery.io/) logs and device telemetry at scale. The IT team also implemented Fleet’s flexible [JSON](https://en.wikipedia.org/wiki/JSON)-based data reporting, empowering teams to access data faster and enabling smarter decision-making across the organization.

## Results

<div purpose="checklist">

A 96% reduction in telemetry collection latency, from 24 hours to every 15 minutes.
Cost savings through better device refresh planning, supported by historical data insights.
Enhanced compliance management with automated checks on security configurations.
Greater operational agility, empowering teams to run live queries for near real-time data access.
</div>

By switching to Fleet, it transformed its device management strategy, improving performance, reducing costs, and enabling cross-functional collaboration.


## Their Story

 This cloud-based data company automatically manages all parts of the data storage process, including organization, structure, metadata, file size, compression, and statistics. It sought a modern solution to manage tens of thousands of devices by providing thorough endpoint telemetry, faster incident response, threat-hunting capabilities, enhanced [software patching](https://fleetdm.com/software-management) workflows, and easy data sharing across internal teams.

With Fleet, they achieved this with:

- Definitive data
- Unified reporting language
- Instant audits
- Portability

### Definitive data
<div purpose="attribution-quote">

This is mind-blowing to me, as I have never had a setup at any job where I can get data from our end-user device fleet this fast. I don’t know how to describe this other than it is just pure magic.

**— IT Engineering Manager**
</div>

Fleet’s configurable [data update cycle](https://fleetdm.com/docs/configuration/fleet-server-configuration#osquery-detail-update-interval) revolutionized their endpoint management. This allowed them to choose a 15-minute frequency, enabling precise device performance tracking without triggering their internal rate limits. Unlike other legacy systems, Fleet gives you complete control over how frequent and labor-intensive the scanning is with [performance impact](https://fleetdm.com/releases/fleet-4.5.0) being automatically reported.

### Unified reporting language

Fleet integrated directly, using AWS Kinesis Firehose to stream osquery logs at high speeds. This ensured its teams could ingest and model large datasets effortlessly with standard formats without requiring the standard programming languages or variations across macOS, Windows, and Linux.

### Instant audits

Fleet enables teams to run live queries and gain insights in near real-time, enabling faster incident responses, threat hunting, and compliance reporting. Scheduling these queries to run in the background, meant that compliance policies would always check against certain states of security settings to stay ahead of audits.

### Portability

Portability with Fleet extends beyond data—it enhances the flexibility of your entire tool stack. Fleet and osquery function as standalone solutions, free from reliance on traditional MDM systems, and enable you to [ship data](https://fleetdm.com/guides/log-destinations) to any platform like Splunk, Snowflake, or any streaming infrastructure like AWS Kinesis and Apache Kafka. This independence means that if an organization chooses to switch MDM providers, the Fleet + osquery stack can easily integrate with the new solution avoiding disruptions to data collection. 


## Conclusion
The cloud data platform's adoption of Fleet Device Management exemplifies how modern IT organizations can achieve operational excellence with the right tools. By delivering timely, actionable data and integrating seamlessly with their existing ecosystem, Fleet enabled them to reduce costs, improve performance, and foster innovation across teams.

<call-to-action></call-to-action>

<meta name="category" value="announcements">
<meta name="authorGitHubUsername" value="Drew-P-drawers">
<meta name="authorFullName" value="Andrew Baker">
<meta name="publishedOn" value="2024-12-20">
<meta name="articleTitle" value="Cloud-based data leader chooses Fleet for telemetry">
<meta name="description" value="Cloud-based data leader chooses Fleet for telemetry">
<meta name="showOnTestimonialsPageWithEmoji" value="🚪">

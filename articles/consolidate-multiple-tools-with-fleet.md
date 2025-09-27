# Stripe consolidates multiple tools with Fleet

<div purpose="attribution-quote">

We've been using Fleet for a few years and we couldn't be happier. The fact that it's also open-source made it easy for us to try it out, customize it to our needs, and seamlessly integrate it into our existing environment.

**- Head of Developer Infrastructure & Corporate Technology, Stripe**
</div>

## Their story

Stripe is a global technology company, building the economic infrastructure for the current and future internet. Enterprises of every size, from startups to publicly-traded bohemoths, use Stripe for payment transaction processing and managing their businesses.

As Stripe expanded, they faced growing complexity in managing devices across teams and locations. Existing device management solutions were either too cumbersome or lacked the flexibility to align with the high-availability, fast, secure infrastructure Stripe has built for itself.

To address these challenges, Stripe set out to achieve four key device management goals:

- Reduce tool overlap
- Adopt next-generation change management
- Streamline device health assessments
- Empower end-user self-service

## Challenge

Stripe wanted to simplify device management. They also wanted to reduce the load on their engineering teams without sacrificing control over the critical devices they manage. Their use of multiple device management solutions created operational silos, required specialized expertise for legacy systems, and led to engineering inefficiencies.

## Solution

Stripe replaced their legacy device management tooling with Fleet: a cross-platform device management solution that supports macOS, mulitple Linux flavors, Windows, iOS / iPadOS, Chromebook and Android.

The company was already using Fleet in early 2023 for managing [osquery](https://www.osquery.io/) in threat detection and compliance use cases with [scheduled queries](https://fleetdm.com/guides/queries).

Not long after, Fleet Device Management announced open-source [cross-platform MDM capabilities](https://www.computerworld.com/article/1622574/fleet-announces-open-source-cross-platform-mdm-solution.html). Fleet added these MDM features on top of osquery's powerful capabilities. Stripe saw the addition of the new features as an opportunity to leverage Fleet for device management and to consolidate their tools. Fleet's combination of cross-platform support, open-source transparency, and scalability made it the right choice.

## Results

<div purpose="checklist">

- Consolidated multiple legacy device management solutions, improving efficiency and reducing SaaS spending without compromising functionality

- Reduced mistakes through peer reviews and robust automation using the Fleet API

- Used Fleet to get reliable, live access to their infrastructure for verifying device data, driving better decisions around end-user access and auditing

- Elected to self-host Fleet for complete control of their data and security posture while maintaining their impressive 99.99% uptime
</div>

### Agent deployment

<div purpose="attribution-quote">

Mad props to how easy making a deploy pkg of the agent was. I wish everyone made stuff that easy.

**— Staff Client Platform Engineer, Stripe**
</div>

THe ability to easily build Fleet's agent deployment packages allowed a quick install of `fleetd` across all of Stripe's devices. By supporting all of Stripe's platforms, Fleet enabled Stripe to deploy `fleetd` for managing osquery and device management, elimintating legacy tooling in the process.

### Audits

By switching to Fleet, Stripe wasted less time around audits by unblocking data collection and overcame change management inertia, allowing IT to move faster with less manual intervention. At Stripe this means expanded employee device choice without adding risk. 

### Device health

Fleet can pull detailed information from every operating system in near real-time, allowing quick assessments of device health, installed applications, and verified configurations. Because Fleet is API-first and built for automation, Stripe configures all of their devices to access its network but only if they've passed conditional access checks.

### End-user empowerment

By providing self-service instructions in [Fleet Desktop](https://fleetdm.com/guides/fleet-desktop#basic-article), end-users can resolve common policy issues without IT intervention, reducing support tickets and improving IT help desk response. This optimizes resources and allows Stripe's teams to spend less time reacting and more time focused on strategic initiatives.

### Next-gen change management

Being [open-source](http://fleetdm.com/handbook/company/why-this-way?utm_content=eo-security#why-open-source) Fleet provides transparency and flexibility, allowing Stripe to customize it to their requirements. This builds trust among peers on Stripe's engineering teams allowing them to audit and extend Fleet as needed.

## Conclusion

By choosing Fleet, Stripe streamlined device management, unified their device management strategy, and empowered their end-users while leveraging the benefits of an open-source solution with an API-first design. Fleet's device management features compliment the advanced data collection and real-time insights available via osquery, enabling proactive management, improved decision-making and enhancing operational efficiency.

For Stripe, Fleet's capabilities and design philosophy set it apart from its competitors. Fleet has become an integral part of Stripe's infrastructure, offering the scalability, transparency, and flexibility needed to support their growth.

To learn more about how Fleet can support your organization, visit [fleetdm.com/mdm](https://fleetdm.com/mdm).

<call-to-action></call-to-action>

<meta name="category" value="announcements">
<meta name="authorGitHubUsername" value="nonpunctual">
<meta name="authorFullName" value="Brock Walters">
<meta name="publishedOn" value="2025-09-26">
<meta name="articleTitle" value="Stripe consolidates multiple tools with Fleet">
<meta name="description" value="Stripe consolidates multiple tools with Fleet">
<meta name="showOnTestimonialsPageWithEmoji" value="🥀">

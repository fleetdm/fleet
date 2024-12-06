# Leading financial company consolidates multiple tools with Fleet


## Fleet’s impact

* Eliminate tool overlap:
Fleet reduced tool overlap by consolidating multiple legacy solutions - improving efficiency and reducing SaaS spending without compromising functionality.

* Next-gen change management: 
GitOps capabilities reduce mistakes through peer reviews and keep track of changes for faster auditing.

* Definitive data: 
Reliable, live access to their infrastructure to verify device data for better decisions surrounding end-user access and auditing context.

* Seamless customization and integration:
Electing to self-host Fleet, and as the company continued to scale, they did so without a single point to their impressive 99.99% uptime.

"We've been using Fleet for a few years and we couldn't be happier. The fact that it's also open-source made it easy for us to try it out, customize it to our needs, and seamlessly integrate it into our existing environment." - Head of Developer Infrastructure & Corporate Technology

**Challenge:** They looked to simplify how they manage devices and reduce tool overlap without sacrificing control over their infrastructure.  The use of multiple proprietary device management tools was creating operational silos, and it required specialized expertise for different legacy systems, leading to inefficiencies.

**Solution:** The leading financial company migrated to Fleet without sacrificing a single point of their 99.99% uptime, replacing multiple device management suppliers with a single multi-platform system supporting macOS, desktop Linux, and Windows. They also implemented next-change management, reducing mistakes through peer reviews, and using the user interface and Fleet API for reporting, automation, and to enable smarter end-user self-service.

**Impact:** They saw a reduction in wasted time by unblocking data collection for audits and overcame change inertia, allowing IT to move faster with less maintenance through convention over configuration and bare metal access to every supported platform, including Apple and desktop Linux. In this way, they were able to offer employees device choice without adding to their risk register. 


## The challenge

This company is global technology company building economic infrastructure for the Internet. Businesses of every size, from new startups to public companies, use it's software to accept payments and manage their businesses online and in person.

As they expanded, it faced a growing complexity in managing a vast array of devices across teams and locations. Existing solutions were either too cumbersome to deploy or lacked the flexibility and cross-platform support needed to align with existing infrastructure and workflows.

To address these challenges, they set out to achieve four key goals:

- **Reduce tool overlap:** Replace multiple tools with a single solution that supports macOS, Windows, and Linux, ensuring consistency across all platforms.
- **Adopt next-generation change management:** Leverage GitOps workflows for tasks such as deploying configuration profiles, delivering MDM commands, updating custom settings, and reporting on application installations.
- **Streamline device health assessments:** Enable quick access to asset data to evaluate device health and make informed network access decisions efficiently.
- **Empower end-user self-service:** Provide users with clear instructions to resolve common issues independently, reducing dependence on IT teams.


## The solution

The company was already using Fleet in early 2023 to manage osquery from a threat detection and compliance perspective with [scheduled queries](https://fleetdm.com/guides/queries). However, they mentioned the growing need to quickly reach out to users to educate them on enabling compliance checks.

Not soon after in April 2023, Fleet announced open-source, [cross-platform MDM capabilities](https://www.computerworld.com/article/1622574/fleet-announces-open-source-cross-platform-mdm-solution.html) building on top of osquery which they were already familiar with.  Seeing this as an opportunity to leverage Fleet and reduce the amount of tools they had to manage. Fleet's combination of cross-platform support, open-source transparency, and scalability made it worthwhile to migrate MDMs.

### Eliminate tool overlap with easy deployment

"Mad props to how easy making a deploy pkg of the agent was. I wish everyone made stuff that easy."
— Staff Client Platform Engineer

Fleet's straightforward deployment package allowed a quick install of the agent across all of its devices. By supporting macOS, Windows, and Linux, Fleet enabled them to not only continue managing osquery but also consolidate its legacy device management tools into a single self-hosted MDM without sacrificing existing control. 

### Next-gen change management and open-source flexibility

"We've been using Fleet for a few years and we couldn't be happier. The fact that it's also open-source made it easy for us to try it out, customize it to our needs, and seamlessly integrate it into our existing environment." — Head of Developer Infrastructure & Corporate Technology

Being [open-source](http://fleetdm.com/handbook/company/why-this-way?utm_content=eo-security#why-open-source), Fleet provided the transparency and flexibility to tailor the platform to their specific requirements. This fostered trust among engineering teams and allows them to audit, customize, and extend the platform as needed.


**Fleet's next-gen change management capabilities:**

- Enforce custom settings updates and deployment: Manage custom settings across all devices using GitOps workflows.
- Implement change control: Reduce mistakes through peer reviews.
- Deploy configuration profiles to macOS devices: Update system settings and controls on macOS devices efficiently.
- Deliver MDM commands: Manage and execute MDM commands like lock, sleep, and wipe.
- Report on installed applications and versions: Generate comprehensive reports on installed applications, aiding in software management and compliance checks.


### Definitive data and end-user empowerment

Fleet can pull detailed information on assets across every operating system in seconds, allowing quick assessments of device health, installed applications, and verified configurations. Because Fleet is API-first, programmable, and built for automation, they configure all of their devices to access its network but only if they passed its predetermined policies.

By providing self-service instructions in [Fleet Desktop](https://fleetdm.com/guides/fleet-desktop#basic-article), end-users can resolve common policy issues without IT intervention, reducing support tickets and increasing efficiency. This empowerment optimizes resources and allowed their IT teams to focus on more strategic initiatives.



## Conclusion:


Fleet has become an integral part of the financial company's infrastructure, offering the scalability, transparency, and flexibility needed to support their growth. By choosing Fleet, they improved endpoint operations, streamlined device management, unified their device management strategy across all platforms, and empowered end-users—all while leveraging the benefits of an open-source solution with advanced [GitOps capabilities](https://github.com/fleetdm/fleet-gitops).

Fleet's cross-platform support and open-source transparency set it apart from competitors, providing a single source of truth for all devices. The advanced data collection and real-time insights enable proactive management and decision-making, enhancing operational efficiency.


To learn more about how Fleet can support your organization, visit [fleetdm.com/mdm](https://fleetdm.com/mdm).

<meta name="category" value="announcements">
<meta name="authorGitHubUsername" value="Drew-P-drawers">
<meta name="authorFullName" value="Andrew Baker">
<meta name="publishedOn" value="2024-12-06">
<meta name="articleTitle" value="Leading financial company consolidates multiple tools with Fleet">
<meta name="description" value="Leading financial company consolidates multiple tools with Fleet">
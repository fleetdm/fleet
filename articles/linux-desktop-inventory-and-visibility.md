# Linux desktop inventory and visibility

*You can't manage a Linux desktop you can't see. Here's how to build a live inventory of every host and query its real state, without adding a separate tool for every distribution.*

## Key takeaways

- **Visibility comes before management.** Setting policy, spotting drift, and remediating misconfigurations all depend on a continuously updated picture of your hosts, and Linux's mix of kernels, distributions, and configurations makes that picture harder to hold.
- **Group hosts by business logic, not by operating system.** Fleets let you organize Windows, Mac, and Linux hosts together around real requirements instead of splitting them across separate tools that drift apart.
- **Labels keep themselves current.** Dynamic labels apply automatically based on live host state, so a group like "workstations running Docker" stays accurate as the environment changes.
- **One SQL dialect answers static and live questions.** Query OS versions and installed packages alongside running processes and open ports, using the same cross-platform syntax on every distribution, with no per-platform tooling to learn.
- **Ad-hoc, scheduled, and continuous checks share one workflow.** Run a one-off report during an incident, schedule a monthly disk-space report, or continuously evaluate a compliance policy, all defined the same way.
- **A failed check can trigger a fix.** Policies answer yes-or-no compliance questions and can run a script, install software, or fire a webhook when a host falls out of line, turning noncompliance into an action instead of a ticket.

<a purpose="cta-button" href="/linux-management">See Linux management in Fleet</a>

The first step in any desktop management strategy is understanding your infrastructure: what hosts you have, what operating systems and software versions they run, and how their real configuration compares to your organization's policies. Taking that inventory is hard, because modern environments are both heterogeneous and constantly changing, and nowhere more so than on the Linux desktop, where the range of choices creates a growing management burden.

This article covers why inventory and visibility matter for Linux desktops and how Fleet delivers both. It starts with a simple premise: you can't manage what you can't see.

## Background

You can't manage any infrastructure without an accurate picture of your environment. Developing policy, identifying drift, remediating misconfigurations, and enforcing security controls all rely on continuous visibility.

Linux adds its own challenges. The heterogeneous nature of Linux devices means dealing with different kernels, distributions, and configurations, so you have to think carefully about how you track your hosts and gain visibility into them.

## Tracking inventory

Modern environments are dynamic. Users switch roles, IT policies change, and devices aren't always connected to your network. A management platform has to maintain an accurate host inventory despite constant change.

An ideal Linux desktop management platform tracks all of your devices, not just a subset, in one location. IT administrators don't need yet another tool to log into. They need a unified experience that provides visibility into their Windows, Mac, and Linux hosts from one place, along with a common language to query devices across that heterogeneous ecosystem.

A management platform also has to group and label hosts for visibility and control. Some hosts may carry restrictive security requirements based on the data they access, and grouping should reflect dynamic, continuously updated device state rather than a static snapshot. That's what lets the platform keep up with the constantly changing environment IT teams are expected to support.

Consider the following questions when evaluating whether a platform meets your needs:

1. Can you manage all devices (Windows, Mac, and Linux) from a single location, or do you need multiple tools?
2. Does the tool support devices, such as end-user workstations, that frequently disconnect and reconnect to the network?
3. Can you group or label hosts in ways that make sense for your organization, or does the system impose rigid restrictions around those groupings?
4. How difficult is it to add a new host, and can that be easily automated?

## Visibility

Once you have an accurate inventory, you need visibility into host configuration and state, and that visibility has to be comprehensive. It should include static, configured information such as OS versions, installed packages, and configured users, but it must also include dynamic information such as running processes and open ports.

A good platform lets you query your environment on both a scheduled and an as-needed basis. Not every aspect of your environment needs continuous monitoring. Sometimes you have a specific question you need answered once, or on a schedule. A common example is a newly discovered software vulnerability: you need to query your environment, find vulnerable hosts, and take action, which means executing an ad-hoc query across everything you manage. Other needs are recurring, like determining free disk space every month so you can proactively upgrade hosts before they run out.

Policy-based requirements are different. They need regular, ongoing visibility to confirm systems keep meeting external or organizational rules. For example, you may have a security requirement that forbids any workstation from running a listening service on ports 80 or 443. Your platform should confirm the policy is met, tell you which hosts are failing it, and, ideally, remediate automatically. Unlike ad-hoc or scheduled reports, policy-based requirements must be continuously monitored to ensure compliance.

Consider the following questions when evaluating a platform's visibility features:

1. Can the system provide visibility into static characteristics, configuration, and dynamic elements of your hosts?
2. Does the system support continuous policy evaluation as well as ad-hoc and scheduled reports, using consistent tooling rather than a different approach for each?
3. Can you query heterogeneous systems (Windows, Mac, and different Linux distributions) using a consistent language and framework?

## One query language for every host

Fleet's agent is built on osquery, a cross-platform tool that exposes information about your systems as a SQL database. That gives you a common, consistent language for inventory and visibility across your environment, with a rich schema that exposes hundreds of tables and thousands of attributes about your devices.

For example, the query below looks for any users named "docker" on a system. It works equally well across Windows, Mac, and Linux.

```sql
SELECT uid, uuid, gid, username FROM users WHERE username = 'docker';
```

This approach is uniquely suited to heterogeneous environments. Platform-agnostic SQL gives you a single interface for querying every device, so you don't have to learn a different tool for each operating system, and the extensive set of tables, many of them cross-platform, lets you determine virtually anything about your hosts.

osquery is a mature, actively maintained open-source project that has been around for over a decade and has more than 20,000 stars on GitHub. It runs in a lightweight footprint and imposes minimal overhead, but on its own it reports on one host at a time. Fleet is what scales that visibility to your whole environment.

## Inventory and visibility in Fleet

### Host inventory

Fleet makes it easy to track host inventory over time and across tens, hundreds, or thousands of hosts. Fleet's agent is a lightweight software package installed on every device in your environment, with packages for Windows, Mac, and Linux. It has a very small footprint and communicates with your Fleet server over TLS.

Once a host is connected to your Fleet environment, you can begin managing it. Fleet provides two key features for tracking host inventory: fleets and labels.

#### Fleets

Fleets let you organize hosts into groups that you can report on, apply policies to, and configure. They're tailored to your organization's specific tasks and compliance requirements, and because Fleet is cross-platform, you can manage Windows, Mac, and Linux workstations within a single fleet.

This lets you define fleets around business logic rather than arbitrary technical requirements. You might have one fleet for all your workstations, another for employee-owned devices, and a third for company-issued mobile devices. It contrasts with tools that require you to separate devices by operating system, an approach that leads to duplicated effort and configuration drift. Fleet lets you manage all of your systems in one place.

Manage fleets by clicking your user icon in the top-right corner and navigating to **Settings > Fleets**. New hosts can be added to a fleet automatically based on their enrollment secret, or you can manually move hosts between fleets by clicking a host and selecting **Actions > Transfer**. To move several at once, go to the **Hosts** page, select the hosts you want, and click **Transfer**.

![Hosts in a Workstations fleet](../website/assets/images/articles/linux-desktop-inventory-and-visibility-1-947x269@2x.png)
*Hosts in a Workstations fleet*

#### Labels

Fleet also lets you label hosts for targeted reporting and policy enforcement. For example, you can apply a "Docker" label to every workstation that has Docker installed, then target reports or policies at that label (for instance, ensuring all Docker workstations run the latest version from your internal repositories).

Administrators can apply labels statically, but their real power is dynamic labeling: labels applied automatically based on report results. Because Fleet's agent can inspect virtually any system characteristic, you can label hosts on almost anything, such as automatically labeling every host with SSH enabled and running.

Create new labels by clicking your user icon in the top-right corner and navigating to **Labels**, then clicking **Add label**. A manual label lets you add specific hosts, while a dynamic label applies automatically based on a report, giving you the flexibility of report-based grouping.

![Labels can be dynamically applied to hosts based on reports](../website/assets/images/articles/linux-desktop-inventory-and-visibility-2-955x306@2x.png)
*Labels can be dynamically applied to hosts based on reports*

### Reports and policies

#### Reports

Fleet's reporting runs on the same agent, so you can query your environment using one common language. You can define reports to run on demand or on a schedule, or run ad-hoc reports directly against your environment without saving them for later.

Reports can target static system information, such as the operating system version, and dynamic runtime information, such as running processes or open ports. Because the query language is cross-platform, one report can work across your Windows, Mac, and Linux devices, which reduces the cognitive burden of a heterogeneous environment and lets you build standardized reports across different systems.

Navigate to **Reports > Add report** to define a new report. The **New report** window prompts you for a report to run against your environment, offers a helpful reference for table information, and automatically checks your report for operating system compatibility.

![Creating a new report in Fleet](../website/assets/images/articles/linux-desktop-inventory-and-visibility-3-684x238@2x.png)

You can **Save** the report for later use. The **Save report** window lets you set an interval to run it on a schedule, or you can specify **Never** to keep it manual and run it yourself when needed from the **Reports** page by clicking its name and choosing **Live report**.

You don't have to save a report at all. A **Live report** runs immediately against your environment without saving, which is useful for exploratory or ad-hoc work you don't plan to reuse.

#### Policies

Policies are similar to reports, and both are just queries, but a policy is designed to answer a yes-or-no question about your environment. A regular report returns detailed information, while a policy returns pass or fail. That lets you define organizational policies and identify when hosts are failing them.

Fleet continuously monitors policy compliance and can take action when a violation occurs. For example, Fleet can run a script, install software, block single sign-on, or trigger a webhook when a policy fails.

![Fleet policies allow you to monitor compliance with organizational rules](../website/assets/images/articles/linux-desktop-inventory-and-visibility-4-1024x256@2x.png)
*Fleet policies allow you to monitor compliance with organizational rules*

Defining a policy is much like defining a report. Navigate to **Policies > Add policy**; the **New policy** window is nearly identical to the **New report** window. Policy queries are evaluated differently from regular reports: if the query returns any result, the policy passes, and if it returns no result, the policy fails. You'll often see policy queries start with `SELECT 1…` to ensure they return a result when the check succeeds.

![Saving a policy in Fleet](../website/assets/images/articles/linux-desktop-inventory-and-visibility-5-1020x247@2x.png)

Once you've refined the query, you can **Save** the policy. The **Save policy** window prompts you for a name, description, and resolution; the description and resolution help whoever later investigates a compliance issue. You can even use Fleet's AI capabilities to generate a description and resolution automatically from the query. This is also where you specify which hosts the policy applies to, and because Fleet combines inventory and visibility, you can target policies by operating system or by dynamic label.

Policies are versatile. Tying an action such as running a script or triggering a webhook to a compliance result lets your IT teams automate common workflows and address noncompliance quickly.

## Wrapping up

The first step in managing an environment is understanding it, and nowhere is that more true than in Linux desktop management, where a heterogeneous environment introduces unique challenges.

Robust Linux desktop management requires a complete inventory of your hosts and visibility into their current state. You need to report on a range of system characteristics, know when hosts aren't meeting your policies, and do it all through a common language and framework that doesn't force your IT teams to learn yet another tool. Fleet's cross-platform agent provides exactly that depth of insight.

Inventory and visibility are only the first step in a complete Linux management strategy. Later articles build on these foundations to cover drift management, automated software installation, and automatic remediation.

To learn more about Fleet or to get a demo, [contact us](https://fleetdm.com/contact).

<meta name="articleTitle" value="Linux desktop inventory and visibility">
<meta name="authorFullName" value="Anthony Critelli">
<meta name="authorGitHubUsername" value="acritelli">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-05-18">
<meta name="description" value="Discover how to track and query Linux desktops at scale using Fleet and osquery: covering fleets, labels, ad-hoc reports, and compliance policies.">

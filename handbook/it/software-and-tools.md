# Software and Tools

This page guides you through software procurement, licensing, and tool management at Fleet. Whether you're evaluating a new SaaS tool, managing licenses for your team, or need to cancel a subscription, you'll find the processes and best practices here.

We take a thoughtful approach to tool selection, prioritizing data portability, programmability, and intentional integration with our existing workflows. This helps us build a cohesive technology stack while maintaining cost efficiency and avoiding tool sprawl.

## Individual Tool Documentation

Each major software tool at Fleet has its own dedicated page with detailed information about what the tool is, what systems it integrates with, who is the DRI, how to get access, and how to get support:

### Go-To-Market (GTM) Tools
- **[Salesforce](tools/salesforce.md)** - Customer relationship management (CRM) system
- **[Gong.io](tools/gong.md)** - Conversation intelligence and revenue analytics
- **[Dripify](tools/dripify.md)** - LinkedIn automation and outreach
- **[Clay](tools/clay.md)** - Data enrichment and lead generation
- **[LinkedIn Sales Navigator](tools/linkedin-sales-navigator.md)** - LinkedIn's premium sales tool

### Design and Development Tools
- **[Figma](tools/figma.md)** - Collaborative design and prototyping
- **[GitHub](tools/github.md)** - Version control and code collaboration
- **[Slack](tools/slack.md)** - Internal communication platform

### Automation and Workflow Tools
- **[Zapier](tools/zapier.md)** - Workflow automation platform
- **[DocuSign](tools/docusign.md)** - Electronic signature platform

> **Note:** This list will be updated as new tools are added or existing ones are retired. For access requests or questions about any tool, [contact IT](https://fleetdm.com/handbook/it#contact-us).

## Purchase a SaaS tool

When procuring SaaS tools and services, analyze the purchase of these subscription services look for these way to help the company:
- Get product demos whenever possible.  Does the product do what it's supposed to do in the way that it is supposed to do it?
- Avoid extra features you don't need, and if they're there anyway, avoid using them.
- Data portability: is it possible for Fleet to export it's data if we stop using it? Is it easy to pull that data in an understandable format?
- Programmability: Does it have a publicly documented legible REST API that requires at most a single API token?
- Intentionality: The product fits into other tools and processes that Fleet uses today. Avoid [unintended consequences](https://en.wikipedia.org/wiki/Midas). The tool will change to fit the company, or we won't use it. 

## Cancel a vendor or subscription

Once the decision has been made not to renew a tool or subscription on Fleet's behalf, use the following steps to churn/cancel a vendor or subscription:

1. Cancel the subscription, including recurring billing. If invoiced, then send churn notice.
2. Update ["¶ 🧮 The numbers" spreadsheet (confidential doc)](https://docs.google.com/spreadsheets/d/1X-brkmUK7_Rgp7aq42drNcUg8ZipzEiS153uKZSabWc/edit?gid=2112277278#gid=2112277278).
  - Prepend the recurring expense title with "CANCELLED - ".
  - Zero-out "Projected monthly burn" and "Projected invoice amount".
3. Remove references from integrated systems and references (i.e. unplug the tool from any other integrations)
4. Remove any shared access from 1Password vaults.
5. Update any reference to the tool or subscription and afterwards communicate the change (e.g. by linking to your merged PR in Slack).

## Grant role-specific license to a team member

Certain new team members, especially in go-to-market (GTM) roles, will need paid access to tools like Salesforce and LinkedIn Sales Navigator immediately on their first day with the company. Gong licenses that other departments need may [request them from IT](https://fleetdm.com/handbook/it#contact-us) and we will make sure there is no license redundancy in that department.

## Process a tool upgrade request from a team member

- A Fleetie may request an upgraded license seat for Fleet tools by submitting an issue through GitHub.
- IT will upgrade or add the license seat as needed and let the requesting team member know they did it.

## Downgrade an unused license seat

- On the first Wednesday of every quarter, the CEO and Head of Digital Workplace & GTM Systems will meet for 30 minutes to audit license seats in Figma, Slack, GitHub, Salesforce and other tools.
- During this meeting, as many seats will be downgraded as possible. When doubt exists, downgrade.
- Afterward, post in #random letting folks know that the quarterly tool reconciliation and seat clearing is complete, and that any members who lost access to anything they still need can submit a GitHub issue to IT to have their access restored.
- The goal is to build deep, integrated knowledge of tool usage across Fleet and cut costs whenever possible. It will also force conversations on redundancies and decisions that aren't helping the business that otherwise might not be looked at a second time.  

## Add a seat to Salesforce

Here are the steps we take to grant appropriate Salesforce licenses to a new hire:
- Go to ["My Account"](https://fleetdm.lightning.force.com/lightning/n/standard-OnlineSalesHome).
- View contracts -> pick current contract.
- Add the desired number of licenses.
- Sign DocuSign sent to the email.
- The order will be processed in ~30m.
- Once the basic license has been added, you can create a new user using the new team member's `@fleetdm.com` email and assign a license to it.
  - To enable email sync for a user:
    - Navigate to the [user's record](https://fleetdm.lightning.force.com/lightning/setup/ManageUsers/home) and scroll to the bottom of the permission set section.
    - Add the "Inbox with Einstein Activity Capture" permission set and save.
    - Navigate to the ["Einstein Activity Capture Settings"](https://fleetdm.lightning.force.com/lightning/setup/ActivitySyncEngineSettingsMain/home) and click the "Configurations" tab.
    - Select "Edit", under "User and Profile Assignments" move the new user's name from "Available" to "Selected", scroll all the way down and click save.
   

<meta name="maintainedBy" value="allenhouchins">
<meta name="title" value="Software and Tools">


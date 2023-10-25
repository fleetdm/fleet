# Customer Success

This handbook page details processes specific to working [with](#contact-us) and [within](#responsibilities) this department.

## What we do
The customer success department is directly responsible for ensuring that customers and community members of Fleet achieve their desired outcomes with Fleet products and services.


## Team
| Role                                  | Contributor(s)           |
|:--------------------------------------|:------------------------------------------------------------------------------------------------------------------------|
| VP of Customer Success                | [Zay Hanlon](https://www.linkedin.com/in/zayhanlon/) _([@zayhanlon](https://github.com/zayhanlon))_
| Customer Success Manager (CSM)        | [Jason Lewis](https://www.linkedin.com/in/jlewis0451/) _([@patagonia121](https://github.com/patagonia121))_
| Customer Support Engineer (CSE)       | [Kathy Satterlee](https://www.linkedin.com/in/ksatter/) _([@ksatter](https://github.com/ksatter))_, [Grant Bilstad](https://www.linkedin.com/in/grantbilstad/) _([@Pacamaster](https://github.com/Pacamaster))_, Ben Edwards _([@edwardsb](https://github.com/edwardsb))_
| Infrastructure Engineer               | [Robert Fairburn](https://www.linkedin.com/in/robert-fairburn/) _([@rfairburn](https://github.com/rfairburn))_

## Contact us
- To make a request of this department, [create an issue](https://github.com/fleetdm/confidential/issues/new?assignees=&labels=%23g-customer-success&projects=&template=custom-request.md&title=Request%3A+_______________________) and a team member will get back to you within one business day.
  - Any Fleet team member can [view the kanban board](https://github.com/fleetdm/fleet#workspaces/-g-customer-success-642c83a53e96760014c978bd/board) for this department, including pending tasks and the status of new requests.
  - Please use issue comments and GitHub mentions to communicate follow-ups or answer questions related to your request.
- If urgent, or if you need help submiting your request, mention a [team member](#team) in the [#g-customer-success](https://fleetdm.slack.com/archives/C062D0THVV1) Slack channel.


## Customer support
Customer support engineers (CSEs) serve as Fleet's first line of communication related to technical support questions or bug reports from the customer and community base.  

### Customer support service level agreements (SLAs)

#### Fleet Free
| Impact Level | Definition | Preferred Contact | Response Time |
|:---|:---|:---|:---|
| All Inquiries | Any request regardless of impact level or severity | Osquery #fleet Slack channel | No guaranteed resolution |

Note: If you're using Fleet Free, you can also access community support by [opening a bug](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=bug%2C%3Areproduce&projects=&template=bug-report.md&title=) in the [Fleet GitHub](https://github.com/fleetdm/fleet/) repository.

#### Fleet Premium
| Impact Level | Definition | Preferred Contact | Response Time |
|:-----|:----|:----|:-----|
| Emergency (P0) | Your production instance of Fleet is unavailable or completely unusable. For example, if Fleet is showing 502 errors for all users. | Expedited phone/chat/email support during business hours. </br></br>Email the contact address provided in your Fleet contract or chat with us via your dedicated private Slack channel | **≤4 hours** |
| High (P1) | Fleet is highly degraded with significant business impact. | Expedited phone/chat/email support during business hours. </br></br>Email the contact address provided in your Fleet contract or chat with us via your dedicated private Slack channel | **≤4 business hours** |
| Medium (P2) | Something is preventing normal Fleet operation, and there may or may not be minor business impact. | Standard email/chat support | ≤1 business day | 
| Low (P3) | Questions or clarifications around features, documentation, deployments, or 'how to's'. | Standard email/chat support | 1-2 business days | 

Note: Fleet business hours for support are Monday-Friday, 6AM-4PM Pacific Time, excluding current U.S. federal holidays during which responses may be delayed for Medium and Low impact issues. Fleeties can find Fleet general contact information [here](https://docs.google.com/document/d/1tE-NpNfw1icmU2MjYuBRib0VWBPVAdmq4NiCrpuI0F0/edit).


#### Emergency (P0) request communications:
![Screen Shot 2022-12-05 at 10 22 43 AM](https://user-images.githubusercontent.com/114112018/205676145-38491aa2-288d-4a6c-a611-a96b5a87a0f0.png)


### Workflow outside business hours:
1. A new message is posted in any Slack channel
2. (Zapier filter) The automation will continue if the message is:
    - Not from a Fleet team member
    - Posted outside of Fleet’s business hours
    - In a specific customer channel (manually designated by customer success)   
3. (Slack) Notify the sender that the request has been submitted outside of business hours and provide them with options for escalation in the event of a P0 or P1 incident.
4. (Zapier) Send a text to the VP of CS to begin the emergency request flow if triggered by the original sender. 

##### Things to note: 
- New customer channels that the automation will run in must be configured manually. Submit requests for additions to the Zapier administator. 

### Bug report
Any customer or community member can file a 🦟 ["Bug report"](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=bug%2C%3Areproduce&projects=&template=bug-report.md&title=).

#### Customer codenames
Occasionally, we will need to track public issues for customers that wish to remain anonymous on our public issue tracker. To do this, we choose an appropriate minor planet name from this [Wikipedia page](https://en.wikipedia.org/wiki/List_of_named_minor_planets_(alphabetical)) and create a label which we attach to the issue and any future issues for this customer.

#### Create customer support issue
Locate the relevant issue or create it if it doesn't already exist (to avoid duplication, be creative when searching GitHub for issues - it can often take a couple of tries with different keywords to find an existing issue). 

When creating a new issue, make sure the following:
- Make sure the issue has a "customer request" label or "customer-codename" label.
- "+" prefixed labels (e.g., "+more info please") indicate we are waiting on an answer from an external community member who does not work at Fleet or that no further action is needed from the Fleet team until an external community member, who doesn't work at Fleet, replies with a comment. At this point, our bot will automatically remove the +-prefixed label.
- Is the issue straightforward and easy to understand, with appropriate context (default to public: declassify into public issues in fleetdm/fleet whenever possible)?
- Have we provided a link to that issue for the customer to remind everyone of the plan and for the sake of visibility, so other folks who weren't directly involved are up to speed  (e.g., "Hi everyone, here's a link to the issue we discussed on today's call: […link…](https://omfgdogs.com)")?

#### Troubleshooting questions
1. Required details that will help speed up time to resolution:
    - Fleet server version
    - Agent version 
        - Osquery or fleetd?
    - Operating system
    - Web browser
    - Expected behavior
    - Actual behavior
2. Details that are nice to have but not required. These may be requested by Fleet support as needed:
    - Amount of total hosts
    - Amount of online hosts
    - Amount of scheduled queries
    - Amount and size (CPU/Mem) of the Fleet instances
    - Fleet instances CPU and Memory usage while the issue has been happening
    - MySQL flavor/version in use
    - MySQL server capacity (CPU/Mem)
    - MySQL CPU and Memory usage while the issue has been happening
    - Are MySQL read replicas configured? If so, how many?
    - Redis version and server capacity (CPU/Mem)
    - Is Redis running in cluster mode?
    - Redis CPU and Memory usage while the issue has been happening
    - The output of fleetctl debug archive

#### Assistance from engineering
Customer team members can [reach the engineering oncall](https://fleetdm.com/handbook/engineering#how-to-reach-the-oncall-engineer) for assistance by writing a message with `@oncall` in the [#help-engineering](https://fleetdm.slack.com/archives/C019WG4GH0A) channel of the Fleet Slack. Additional help can be obtained by messaging your friendly Solutions Consultant in the [#help-solutions-consulting channel](https://fleetdm.slack.com/archives/C05HZ2LHEL8).


### Customer support process 
This section outlines Fleet's customer and community support process.
- The customer support engineering (CSE) team handles basic help desk resolution and service delivery issues (P2 and P3) with assistance from on-call and the solutions consulting team as needed.
- The CSE team handles in depth technical issues (P0 and P1) in conjunction with on-call.
- The CSE team handles expert technical product and services support in coordination with the on-call technicians, the customer success manager (CSM), and the product team via [#help-product-design](https://fleetdm.slack.com/archives/C02A8BRABB5).

CSE's track Fleet Premium customer support conversations via the external tool [Unthread](https://app.unthread.io/login?redirect=dashboard). 

The on-call engineer [holds responsibility](https://fleetdm.com/handbook/engineering#responsibilities) for responses to technical Slack comments, Slack threads, and GitHub issues raised by customers and the community, which the CSE team cannot address.

Support issues should be handled in the relevant Slack channel rather than Direct Messages (DMs). This will ensure that questions and solutions can be easily referenced in the future. If it is necessary to use DMs to share sensitive information, a summary of the conversation should be posted in the Slack channel as well. 

An automated weekly [on-call handoff](https://fleetdm.com/handbook/engineering#handoff) Slack thread in #g-engineering provides the opportunity to discuss highlights, improvements, and hand off ongoing issues.

## Customer success 
Customer success managers (CSMs) serve as the primary point of contact for Fleet Premium customers and are responsible for ensuring that customer's desired outcomes are achieved.  

### Invite new customer DRI
Sometimes there is a change in the champion within the customer's organization.
1. Get an introduction to the new DRIs including names, roles, contact information.
2. Make sure they're in the Slack channel.
3. Invite them to the *Success* meetings.
4. In the first meeting understand their proficiency level of osquery.
    1. Make sure the meeting time is still convenient for their team. 
    2. Understand their needs and goals for visibility.
    3. Offer training to get them up to speed.
    4. Provide a white glove experience.

### Schedule a customer call
To schedule an [ad hoc meeting](https://www.vocabulary.com/dictionary/ad%20hoc) with a Fleet customer, use the ["Customer meeting" Calendly link](https://calendly.com/fleetdm/customer).

To schedule a recurring meeting, follow the instructions in the customer success section of the ['sale' issue template](https://github.com/fleetdm/confidential/issues/new?assignees=hughestaylor&labels=%23g-business-operations&projects=&template=3-sale.md&title=New+customer%3A+_____________)

- **Before a customer call(48hrs):** Check the calendar invite 48hrs before the meeting to determine if the customer has accepted the invitation.
  - If the customer has not accepted the invitation, reach out to confirm attendance (e.g., EAs, email, Slack).
  - Test the Zoom Meeting link to make sure that it is working.
  - Make sure that agenda documents are attached and accessible to meeting attendees (as appropriate to the situation).

- **Day of the customer call:** Join the meeting two to three minutes before the start time.

- **Missed customer call:** If the customer does not join the call after five minutes, contact the customer with
  - Slack, if we have a shared channel.
  - email, using the email address from the calendar invite.
  - LinkedIn, send a direct message.
  - an alternative date and time. Suggest two to three options from which the customer can choose.
    - Confirm that contact information is accurate and that the customer can receive and access meeting invites.

### Generate a trial license key
1. Fleet's self-service license key creator is the best way to generate a proof of concept (POC) or renewal/expansion Fleet Premium license key. 
    - [Here is a tutorial on using the self-service method](https://www.loom.com/share/b519e6a42a7d479fa628e394ee1d1517) (internal video)
    - Pre-sales license key DRI is the Director of Solutions Consulting
    - Post-sales license key DRI is the VP of Customer Success

2. Legacy method: [create an opportunity issue](https://github.com/fleetdm/confidential/issues/new/choose) for the customer and follow the instructions in the issue for generating a trial license key.

## Managed Cloud 

Managed Cloud is our hosted Fleet offering for Fleet Premium and Fleet Ultimate customers. All managed Fleet instances receive 24/7 monitoring and on-call support from Customer Success. 

### 24/7 on-call

The 24/7 on-call (aka infrastructure on-call) responsibility happens in shifts of one week.

First responders:

- Robert Fairburn
- Kathy Satterlee

Escalations (in order):

- Eric Shaw (fleetdm.com)
- Zay Hanlon
- Luke Heath
- Mike McNeil

Expected response times: during business hours, 1 hour. Outside of business hours <4 hours.

The first responder on-call for Managed Cloud will take ownership of the @infrastructure-oncall alias in Slack first thing Monday morning. The previous week's on-call will provide a summary in the #g-customer-success Slack channel with an update on alarms that came up the week before, open issues with or without direct end-user impact, and other issues to keep an eye out for.

Escalation of alarms will be done manually by the first responder according to the escalation contacts mentioned above. A [suspected outage issue](https://github.com/fleetdm/confidential/issues/new?assignees=&labels=%23outage%2C%23g-cx%2C%3Arelease&projects=&template=outage.md&title=Suspected+outage%3A+YYYY-MM-DD) should be created to track the escalation and determine root cause. 

All infrastructure alarms (fleetdm.com and Fleet managed cloud) will go to #help-p1.

When the current infrastructure on-call engineer is unable to meet the response time SLAs, it is their responsibility to arrange and designate a replacement who will assume the @oncall-infrastructure Slack alias.

#### Runbooks

Use [on-call runbooks](https://github.com/fleetdm/confidential/tree/main/infrastructure/runbooks#readme) to guide your response. Runbooks provided detailed, step-by-step instructions to quickly and effectively respond to and resolve most 24/7 on-call alerts. 

## Responsibilities

#### First-line response

Customer Success is responsible for the first response to messages in private customer Slack channels.

Customer Support Engineers are responsible for the first response to Slack messages in the [#fleet channel](https://osquery.slack.com/archives/C01DXJL16D8) of osquery Slack, and other public Slacks. 

The 24/7 on-call is responsible for alarms related to fleetdm.com and Fleet Managed Cloud, as well as delivering 24/7 support for Fleet Ultimate customers.

We respond within 1-hour during business hours and 4 hours outside business hours. Note that we do not need to have answers within 1 hour -- we need to at least acknowledge and collect any additional necessary information, while researching/escalating to find answers internally.

> Response SLAs help us measure and guarantee the responsiveness that a customer [can expect](https://fleetdm.com/handbook/company#values) from Fleet.  But SLAs aside, when a Fleet customer has an emergency or other time-sensitive situation ongoing, it is Fleet's priority to help them find them a solution quickly.

### Onboard a customer support engineer
What do you do every day? What does the path to success look like in this role and what can you do to contribute quickly at Fleet? To onboard a customer support engineer at Fleet it's important to understand the [continued training needed](https://docs.google.com/document/d/1GB8i_VMaFxeb9ipLock9MVWGJ2RqIW8lZ5n3MLiXG4s/edit).

### Onboard a customer success manager
What do you do every day? What does the path to success look like in this role and what can you do to contribute quickly at Fleet? To onboard a customer success manager at Fleet it's important to understand the [continued training needed](https://docs.google.com/document/d/1itrBeztwjK253Q548wbveVWdDaldBYCEOS6Cbz5Z4Uc/edit).

### Onboard a customer solutions architect
What do you do every day? What does the path to success look like in this role and what can you do to contribute quickly at Fleet? To onboard a customer solutions architect at Fleet it's important to understand the [continued training needed](https://docs.google.com/document/d/1G26Aqmn4tSKa7s0jMcSRqNTtz6h47Tvf8Ddi2-cP1ek/edit#heading=h.2i16pc77rnb7).

### Onboard an infrastructure engineer
What do you do every day? What does the path to success look like in this role and what can you do to contribute quickly at Fleet? To onboard a customer solutions architect at Fleet it's important to understand the [continued training needed](https://docs.google.com/document/d/1G26Aqmn4tSKa7s0jMcSRqNTtz6h47Tvf8Ddi2-cP1ek/edit#heading=h.2i16pc77rnb7).


## Rituals

<rituals :rituals="rituals['handbook/customers/sales.rituals.yml']"></rituals>



# Customer Success

This handbook page details processes specific to working [with](#contact-us) and [within](#responsibilities) this department.


## Team

| Role                                  | Contributor(s)           |
|:--------------------------------------|:------------------------------------------------------------------------------------------------------------------------|
| VP of Customer Success                | [Zay Hanlon](https://www.linkedin.com/in/zayhanlon/) _([@zayhanlon](https://github.com/zayhanlon))_
| Infrastructure Engineer               | [Robert Fairburn](https://www.linkedin.com/in/robert-fairburn/) _([@rfairburn](https://github.com/rfairburn))_
| Customer Support (CSE/CSA)            | [Kathy Satterlee](https://www.linkedin.com/in/ksatter/) _([@ksatter](https://github.com/ksatter))_ <br> [Rebecca Cowart](https://www.linkedin.com/in/rebeccaui/) _([@rebeccaui](https://github.com/rebeccaui))_ <br> [Brock Walters (CSA)](https://www.linkedin.com/in/brock-walters-247a2990/) _([@nonpunctual](https://github.com/nonpunctual))_ <br> [Dale Ribeiro (CSA)](https://www.linkedin.com/in/daleribeiro/) _([@ddribeiro](https://github.com/ddribeiro))_ <br> Ben Edwards _([@edwardsb](https://github.com/edwardsb))_ 
| Customer Success Manager (CSM)        | [Jason Lewis](https://www.linkedin.com/in/jlewis0451/) _([@patagonia121](https://github.com/patagonia121))_ <br> [Michael Pinto](https://www.linkedin.com/in/michael-pinto-a06b4515a/) _([@pintomi1989](https://github.com/pintomi1989))_
| Technical Evangelist                  | [Zach Wasserman](https://www.linkedin.com/in/zacharywasserman/) _([@zwass](https://github.com/zwass))_


## Contact us

- To **make a request** of this department, [create an issue](https://github.com/fleetdm/confidential/issues/new?assignees=&labels=%23g-customer-success&projects=&template=custom-request.md&title=Request%3A+_______________________) and a team member will get back to you within one business day (If urgent, mention a [team member](#team) in the [#g-customer-success](https://fleetdm.slack.com/archives/C062D0THVV1)).
  - Any Fleet team member can [view the kanban board](https://github.com/fleetdm/fleet#workspaces/-g-customer-success-642c83a53e96760014c978bd/board) for this department, including pending tasks and the status of new requests.
  - Please **use issue comments and GitHub mentions** to communicate follow-ups or answer questions related to your request. 


## Responsibilities

The customer success department is directly responsible for ensuring that customers and community members of Fleet achieve their desired outcomes with Fleet products and services.


### Assign a customer codename

Occasionally, we will need to track public issues for customers and prospects who wish to remain anonymous on our public issue tracker. To do this: 

1. The team member creating the issue will choose an appropriate minor planet name from this [minor planets page](https://minorplanetcenter.net//iau/lists/MPNames.html) (alphabetical).
2. Create a label in the fleetdm/fleet and fleetdm/confidential repos which can be attached to current and future issues for the customer or prospect. As part of the label description in the fleetdm/confidential repo, add the customer or prospect name. This way, we maintain a confidential mapping of codename to customer or prospect.


### Prepare for routine customer meeting

Before a routine customer call, the CSM prepares an agenda including the following items:
1. Customer and Fleet expected attendees
2. Release notes for the latest version of Fleet
3. Update notes for which version of Fleet the customer is running (if self-hosted)
4. Follow ups to the agenda from the previous call or Slack
5. Provide updates to open feature requests (can be done monthly or quarterly)
6. Provide updates to open bug reports


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


### Generate an expansion opportunity in Salesforce

[Customer Success Managers (CSMs)](https://fleetdm.com/handbook/customer-success#team) are responsible for developing customer expansion opportunities that are not being worked on in conjunction with an Account Executive (AE). An AE may be assigned by the [Chief Revenue Officer (CRO)](https://fleetdm.com/handbook/sales#team) for large-scale expansion opportunities such as bringing on a new Fleet use case or bringing on a new group of hosts to an existing Fleet use case. CSMs manage expansion opportunities for things like host count increases for customer growth and price increases on renewals. Discuss examples of these scenarios with your manager to learn more. Moving forward, CSM's are responsible for keeping the stage, next steps, and date of next steps fields updated as the opportunity progresses through the sales cycle. Take the steps below when creating an expansion opportunity in Salesforce:

1. Navigate to the customer account record in [Salesforce](https://fleetdm.lightning.force.com/lightning/page/home).
2. Scroll down to the "Upcoming renewal (± expansions)" table (in the "Customer" section) and click "New".
3. Change the opportunity name to reflect the following naming structure: CustomerName_FleetProduct_Expansion_QuarterDue
    - Example: ABCTestCompany_FleetPremiumMDM_HostExpansion_Q12025
    - Example: ABCTestCompany_FleetPremium_PriceIncreaseExpansion_Q12025
4. Fill out all the required fields making sure to pick "Expansion" in the  "Type" dropdown menu and then click "Save".


### Conduct a health check

Health checks are conducted quarterly or bi-annually, in preparation for a quarterly business review (QBR). The purpose of a health check is to understand what features and functionality the customer is currently using in Fleet. This information will be used to provide guidance to the customer during their QBR. For more information around QBRs, please see the section below, titled "Conduct a quarterly business review".

1. Work with your champion to schedule the health check at a time when their Fleet admins and daily users are available. Be sure to take notes, and record the meeting if possible.
2. During the meeting, ask the customer to share their screen and walk through their day-to-day use of Fleet.
3. Ask the customer questions about the features they are using to understand the "why" behind their use cases for Fleet. Try not to provide guidance directly on this call.
4. Review your notes after the meeting, and find areas of improvement that you can highlight to help your partner more thoroughly utilize Fleet and add your findings to the QBR deck.


### Conduct a quarterly business review (QBR)

Business reviews are conducted quarterly or bi-annually to ensure initial success criteria completion, ongoing adoption, alignment on goals, and delivery of value as a vendor. Use the meeting to assess customer priorities for the coming year, review performance metrics, address any challenges and showcase value in upcoming and unutilized features.
1. Work with your champion to schedule the business review at a time their stakeholders are available (typically 90 days after kickoff and again, 90 days before renewal).
2. Collect usage metrics from the [usage data report](https://docs.google.com/spreadsheets/d/1Mh7Vf4kJL8b5TWlHxcX7mYwaakZMg_ZGNLY3kl1VI-c/edit?gid=0#gid=0) (internal Fleet document) and the following:
    - Optionally schedule a health check with day to day admins prior to the QBR to better understand how the product is being used and which features have been adopted.
    - Have a support engineer collect data on open and closed bugs from the previous quarter and highlight any P0 or P1 incidents along with a summary of the postmortem (search Unthread and GitHub for issues tagged with the customer codename and ':bug').
    - Summarize status updates for open feature requests and highlight delivered feature requests.
    - For managed cloud customers, reach out to #help-infrastructure to collect information on cloud uptime and any outages or alarms.
    - Provide one slide with information on the latest Fleet release and any upcoming big ticket features which can be found on the product board and current release board for #g-mdm and #g-endpoint-ops
3. After the business review, save the presentation as a PDF and share it with your customer.


### Create customer support issue

Locate the relevant issue or create it if it doesn't already exist (to avoid duplication, be creative when searching GitHub for issues - it can often take a couple of tries with different keywords to find an existing issue). When creating a new issue, make sure the following:
- Make sure the issue has a "customer request" label or "customer-codename" label.
  - Occasionally, we will need to track public issues for customers that wish to remain anonymous on our public issue tracker. To do this, we choose an appropriate minor planet name from this [Wikipedia page](https://en.wikipedia.org/wiki/List_of_named_minor_planets_(alphabetical)) and create a label which we attach to the issue and any future issues for this customer.
- "+" prefixed labels (e.g., "+more info please") indicate we are waiting on an answer from an external community member who does not work at Fleet or that no further action is needed from the Fleet team until an external community member, who doesn't work at Fleet, replies with a comment. At this point, our bot will automatically remove the +-prefixed label.
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
- Have we provided a link to that issue for the customer to remind everyone of the plan and for the sake of visibility, so other folks who weren't directly involved are up to speed  (e.g., "Hi everyone, here's a link to the issue we discussed on today's call: […link…](https://omfgdogs.com)")?


### Contact the developer on-call

The acting developer on-call rotation is reflected in the [📈KPIs spreadsheet (confidential Google sheet)](https://docs.google.com/spreadsheets/d/1Hso0LxqwrRVINCyW_n436bNHmoqhoLhC8bcbvLPOs9A/edit#gid=0&range=F2 ). The developer on-call is responsible for responses to technical Slack comments, Slack threads, and GitHub issues raised by customers and the community, which the CSE team cannot address.
- To reach the developer on-call for assistance, mention them in Fleet Slack using `@oncall` in the [#help-engineering](https://fleetdm.slack.com/archives/C019WG4GH0A) channel. 
  - Support issues should be handled in the relevant Slack channel rather than Direct Messages (DMs). This will ensure that questions and solutions can be easily referenced in the future. If it is necessary to use DMs to share sensitive information, a summary of the conversation should be posted in the Slack channel as well. 

> **Note:** Additional help can be obtained by messaging a [Solutions Consultant](https://fleetdm.com/handbook/sales#team) in the [#help-solutions-consulting channel](https://fleetdm.slack.com/archives/C05HZ2LHEL8).

- An automated weekly [on-call handoff](https://fleetdm.com/handbook/engineering#handoff) Slack thread in #g-engineering provides the opportunity to discuss highlights, improvements, and hand off ongoing issues.


### Onboard a customer success team member

- **Customer Success Manager:** Follow the [training steps for this role](https://docs.google.com/document/d/1itrBeztwjK253Q548wbveVWdDaldBYCEOS6Cbz5Z4Uc/edit).
- **Customer Solutions Architect (CSA):** Follow the [training steps for this role](https://docs.google.com/document/d/1G26Aqmn4tSKa7s0jMcSRqNTtz6h47Tvf8Ddi2-cP1ek/edit#heading=h.2i16pc77rnb7).
- **Customer Support Engineer (CSE):** Follow the [training steps for this role](https://docs.google.com/document/d/1GB8i_VMaFxeb9ipLock9MVWGJ2RqIW8lZ5n3MLiXG4s/edit).
- **Infrastructure Engineer:** Follow the [training steps for this role](https://docs.google.com/document/d/1G26Aqmn4tSKa7s0jMcSRqNTtz6h47Tvf8Ddi2-cP1ek/edit#heading=h.2i16pc77rnb7).


### Manage automation of customer slack

1. A new message is posted in any Slack channel
2. (Zapier filter) The automation will continue if the message is:
    - Not from a Fleet team member
    - Posted outside of Fleet’s business hours
    - In a specific customer channel (manually designated by customer success)   
3. (Slack) Notify the sender that the request has been submitted outside of business hours and provide them with options for escalation in the event of a P0 or P1 incident.
4. (Zapier) Send a text to the VP of CS to begin the emergency request flow if triggered by the original sender. 

> **Note:** New customer channels that the automation will run in must be configured manually. Submit requests for additions to the Zapier administrator. 


### Generate a trial license key

1. Fleet's self-service license key creator is the best way to generate a proof of concept (POC) or renewal/expansion Fleet Premium license key. 
    - [Here is a tutorial on using the self-service method](https://www.loom.com/share/048474d7199048e1bf0c4fc106632129) (internal video)
    - Pre-sales license key DRI is the Director of Solutions Consulting
    - Post-sales license key DRI is the VP of Customer Success

2. Legacy method: [create an opportunity issue](https://github.com/fleetdm/confidential/issues/new/choose) for the customer and follow the instructions in the issue for generating a trial license key.


### Respond to messages and alerts 

Customer Support and 24/7 on-call Engineers are responsible for the first response to Slack messages in the [#fleet channel](https://osquery.slack.com/archives/C01DXJL16D8) of osquery Slack, and other public Slacks. 
- The 24/7 on-call is responsible for alarms related to fleetdm.com and Fleet Managed Cloud, as well as delivering 24/7 support for Fleet Premium customers. Use [on-call runbooks](https://github.com/fleetdm/confidential/tree/main/infrastructure/runbooks#readme) to guide your response. Runbooks provided detailed, step-by-step instructions to quickly and effectively respond to and resolve most 24/7 on-call alerts.
- We respond within 1-hour during business hours and 4 hours outside business hours. Note that we do not need to have answers within 1 hour -- we need to at least acknowledge and collect any additional necessary information while researching/escalating to find answers internally.


### Maintain first responder SLA

The first responder on-call for Managed Cloud will take ownership of the @infrastructure-oncall alias in Slack first thing Monday morning. The previous week's on-call will provide a summary in the #g-customer-success Slack channel with an update on alarms that came up the week before, open issues with or without direct end-user impact, and other issues to keep an eye out for.
- **First responders:** Robert Fairburn, Kathy Satterlee

Escalation of alarms will be done manually by the first responder according to the escalation contacts mentioned above. A [suspected outage issue](https://github.com/fleetdm/confidential/issues/new?assignees=&labels=%23outage%2C%23g-cx%2C%3Arelease&projects=&template=outage.md&title=Suspected+outage%3A+YYYY-MM-DD) should be created to track the escalation and determine root cause. 
- **Escalations (in order):** » Eric Shaw (fleetdm.com) » Zay Hanlon » Luke Heath » Mike McNeil

All infrastructure alarms (fleetdm.com and Managed Cloud) will go to #help-p1. When the current 24/7 on-call engineer is unable to meet the response time SLAs, it is their responsibility to arrange and designate a replacement who will assume the @oncall-infrastructure Slack alias.

## Rituals

<rituals :rituals="rituals['handbook/customer-success/customer-success.rituals.yml']"></rituals>

#### Stubs
The following stubs are included only to make links backward compatible.

##### Runbooks
Please see [Handbook/customer-success#respond-to-messages-and-alerts](https://www.fleetdm.com/handbook/customer-success#respond-to-messages-and-alerts)

<meta name="maintainedBy" value="zayhanlon">
<meta name="title" value="🌦️ Customer Success">

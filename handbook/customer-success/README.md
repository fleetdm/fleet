# Customer Success

This handbook page details processes specific to working [with](#contact-us) and [within](#responsibilities) this department.


## Team

| Role                                  | Contributor(s)           |
|:--------------------------------------|:------------------------------------------------------------------------------------------------------------------------|
| Chief Customer Officer                   | [Alex Mitchell](https://www.linkedin.com/in/alexandercmitchell/) _([@alexmitchelliii](https://github.com/alexmitchelliii))_
| SVP of Customer Success                | [Zay Hanlon](https://www.linkedin.com/in/zayhanlon/) _([@zayhanlon](https://github.com/zayhanlon))_
| VP of Security Solutions              | [Dhruv Majumdar](https://www.linkedin.com/in/neondhruv/) _([@karmine05](https://github.com/karmine05))_
| Infrastructure Engineer               | [Robert Fairburn](https://www.linkedin.com/in/robert-fairburn/) _([@rfairburn](https://github.com/rfairburn))_ <br> [Jorge Falcon](https://www.linkedin.com/in/falcon-jorge/) _([@BCTBB](https://github.com/bctbb))_
| Technical Evangelist                  | [Zach Wasserman](https://www.linkedin.com/in/zacharywasserman/) _([@zwass](https://github.com/zwass))_
| Manager of Customer Support and Solutions Architecture | [Dale Ribeiro](https://www.linkedin.com/in/daleribeiro/) _([@ddribeiro](https://github.com/ddribeiro))_
| Customer Solutions Architect (CSA)    | [Jake Stenger](https://www.linkedin.com/in/jakestenger) _([@jakestenger](https://github.com/jakestenger))_ <br> [Adam Baali](https://uk.linkedin.com/in/adambaali) _([@AdamBaali](https://github.com/AdamBaali))_ <br> Steven Palmesano _([@spalmesano0](https://github.com/spalmesano0))_ <br> [Kitzy](https://linkedin.com/in/kitzy) _([@kitzy](https://github.com/kitzy))_ 
| Customer Success Manager (CSM)        | <sup><sub> _See [🦄 Go-To-Market groups](https://fleetdm.com/handbook/company/go-to-market-groups#current-gtm-groups)
| Customer Support Engineer (CSE)       | [Kathy Satterlee](https://www.linkedin.com/in/ksatter/) _([@ksatter](https://github.com/ksatter))_ <br> [Mason Buettner](https://www.linkedin.com/in/mason-buettner-b72959175/) _([@mason-buettner](https://github.com/mason-buettner))_ <br> Ben Edwards _([@edwardsb](https://github.com/edwardsb))_ <br> [Gray Williams](https://linkedin.com/in/gwilliamsuk) _([@grayw](https://github.com/grayw))_

## Contact us

- To **make a request** of this department, [create an issue](https://github.com/fleetdm/confidential/issues/new?assignees=&labels=:help-customers&projects=&template=1-custom-request.md&title=Request%3A+_______________________) and a team member will get back to you within one business day. (If urgent, mention a [team member](#team) in the [#help-customers](https://fleetdm.slack.com/archives/C062D0THVV1) Slack channel.)
  - Any Fleet team member can [view the kanban board](https://github.com/orgs/fleetdm/projects/79) for this department, including pending tasks and the status of new requests.
  - Please **use issue comments and GitHub mentions** to communicate follow-ups or answer questions related to your request. 


## Responsibilities

The customer success department is directly responsible for ensuring that customers and community members of Fleet achieve their desired outcomes with Fleet products and services.


### Respond to a "Contact us" job application submission

1. Applications for open roles at Fleet come through the "Contact us" form.
2. The contact form generates a new ticket in Unthread. 
3. Within 4 business hours, the assigned CSE sends an email from their Gmail account to the applicant using the suggested response template under "Applying for open position" [in this internal file](https://docs.google.com/spreadsheets/d/1-wsYunAfr-BQZMBYizY4TMavi3X071D5KZ3mCYX4Uqs/edit?gid=695748028#gid=695748028). Remember that contact form messages cannot be replied to in Unthread. 
5. CC the Interim Head of People on all job application emails.
6. Remember to include the title of the position being applied for, as is requested in the response template.
7. Add a closure note or include copy of your response as an internal note in the unthread ticket, and close the ticket.


### Assign a customer codename

We track public issues for customers and prospects who wish to remain anonymous on our public issue tracker. To do this: 

1. The team member creating the issue will choose an appropriate minor planet name from this [minor planets page](https://minorplanetcenter.net//iau/lists/MPNames.html) (alphabetical).
2. Create a label in the fleetdm/fleet and fleetdm/confidential repos which can be attached to current and future issues for the customer or prospect. As part of the label description in the fleetdm/confidential repo, add the customer or prospect name. This way, we maintain a confidential mapping of codename to customer or prospect.
3. Navigate to the account in Salesforce. Edit the "GitHub label" field to include the customer or prospect label and save the record. This enables the "fleetdm/fleet" and "fleetdm/confidential" GitHub issue searches. 


### Prepare for routine customer meeting

Before a routine customer call, the CSM prepares an agenda including the following items:
1. Customer and Fleet expected attendees
2. Release notes for the latest version of Fleet
3. Update notes for which version of Fleet the customer is running (if self-hosted)
4. Follow ups to the agenda from the previous call or Slack
5. Provide updates to open feature requests (can be done monthly or quarterly)


### Gather status updates for open issues

When on call, CSEs/CSAs will start their day by following these steps to gather status updates for open issues:
1. Search Unthread for open conversations by the customer name.
2. Search GitHub issues for `label:bug` and `label:customer-codename`.
3. Debrief with any internal resources in order to gather information if needed, and be prepared to provide a status update.

### Perform morning triage

The first CSE to sign on for the day is responsible for triaging new support issues that were reported after hours. The following actions are a general guideline for what should be checked during morning triage:
1. Look at all new support requests and immediately respond to any urgent or high-priority issues.
2. Check the osquery Slack channel/Unthread for support issues.
3. Check the MacAdmins Slack channel for support issues.
   > FYI: MacAdmins Slack messages are not populated in Unthread.
4. Check the "Unassigned" queue in Unthread and re-assign any issues from after hours to the appropriate resource.
5. Check the "All" queue in Unthread for potential after-hours mis-assigned issues and re-assign them to the appropriate resource.
6. Look at all customer meetings for the day to check that they can be attended by a CSE/CSA and that there are no scheduling conflicts.
7. Update the [#help-customers](https://fleetdm.slack.com/archives/C062D0THVV1) Slack channel that morning triage is complete. Report any escalations or conflicts with customer meetings to the [Manager of Customer Success and Solutions Architecture](https://fleetdm.com/handbook/customer-success#team).

Other CSEs that sign on after morning triage has been completed should check the morning triage thread in the #help-customers Slack channel to learn what items are still outstanding.


### Invite new customer DRI

Sometimes there is a change in the champion within the customer's organization.
1. Get an introduction to the new DRIs including names, roles, contact information.
2. Make sure they're in the Slack channel.
3. Invite them to the *Success* meetings.
4. In the first meeting understand their proficiency level with Fleet.
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


### Schedule a Fast-track engagement

Fast-track is Fleet's service delivery package for new MDM customers. Check with your team to learn about the options available and the differences between them (virtual vs on site, migration vs no migration). If your customer has a Fast-track engagement, it will be included in their contract. Follow the directions below to get a Fast-track set up and collect the training pre-requisites.

1. When a deal including Fast-track closes, add a TODO on the final page of the partnership kickoff presentation, to confirm the details around their services purchase and to coordinate scheduling. Be sure to make the customer aware that delays in confirming service delivery date can cause the date to move out further.
2. Prior to the Fast-track kickoff, schedule a Pre-requisite planning meeting with the customer and the assigned CSA to collect the following information:
- What is the target migration date and when does the previous MDM contract end?
- Which critical workflows will Fleet be used for?
  - Onboarding workflow?
  - Offboarding workflow?
  - Automated device enrollment (ADE)? Autopilot?
  - Setup experience?
  - Self-service software?
- Which integrations will be required for migration? Which integrations will be required post-migration (no hard timeline)?
  - IAM?
  - Log shipping to SIEM?
  - Zendesk/JIRA?
  - Others?
- Gather a list of which policies and profiles need to be replicated or replaced
3. For managed cloud customers, send a request to the [:help-customers board](https://github.com/orgs/fleetdm/projects/79/views/1?filterQuery=) requesting that an infrastructure engineer double check the configuration variables to ensure they support the size and scale of the upcoming deployment. For self-hosted customers, schedule a dedicated session with the customer and the assigned CSA to review their server configuration and ensure that it supports the size and scale of the upcoming deployment.


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

### Track a customer promise

Customer promises are contractually obligated feature requests, with guaranteed completion in specific timeframes. These are always represented in a signed contract with the customer. The [Customer promises internal spreadsheet](https://docs.google.com/spreadsheets/d/11Z6zDD4UQktc34IZTtRpNmYZMqndjfLDgFP92P4s7cI/edit?gid=0#gid=0) is our current source of truth. It will be maintained and updated by the Customer Success team. Each CSM is responsible for adding in the following when they are assigned a new customer with contractual promises: 
1. The name of the customer
2. The feature that was promised
3. The date that it's due
4. The current status of that feature

The SVP of Customer Success is the DRI for ensuring customer promise delivery and communicating delays in delivery to the CSM team, in conjunction with the Head of Product Design. Any potential delays in customer promises are addressed with a contract amendment highlighting the updated delivery timeline and is to be signed by the customer via DocuSign, or via an email highlighting the updated delivery time which requires a written response from the customer acknowledging the new timeline. 

### Close out a completed customer promise

Document the completion of a customer promise through the following steps:
1. When a customer promise is thought to be complete, Fleet's product team will reach out and ask the assigned CSM for confirmation from the customer.
2. Once notified, reach out to your customer and schedule a meeting to review the work that has been done, and to make sure it meets their requirements.
3. At the end of the customer promise review meeting, tell your customer that you will be sending over an email going over the discussion and completion of their promise.
4. Get a verbal agreement from your customer to respond to that follow up email, with a confirmation that the promise was completed in a satisfactory manner.
5. Once you have received email confirmation of the completed promise, note this via a comment in the GitHub issue. If all other customers have confirmed completion, then you may close out the issue as well.


### File a customer bug report

Locate the relevant issue or create it if it doesn't already exist (to avoid duplication, be creative when searching GitHub for issues - it can often take a couple of tries with different keywords to find an existing issue). When creating a new issue, make sure to do the following:
- Include a "customer-codename" label.
  - [Search the confidential repo labels](https://github.com/fleetdm/confidential/labels) for an existing codename or [create a new one](https://github.com/fleetdm/confidential/labels) if one does not exist.
- Include required details that will help speed up time to resolution:
    - Fleet server version
    - Agent version 
        - Osquery or fleetd?
    - Operating system
    - Web browser
    - Expected behavior
    - Actual behavior
- Mandatory to include reproduction steps. If a Fleet team member is unable to reproduce the issue, include the steps that were taken by the customer that resulted in the issue occurring. It is also helpful to grab a Gong snippet of the issue as experienced by the customer. 
- Include additional details that are nice to have but not required. These may be requested by Fleet engineering as needed:
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


### Timebox an investigation

During the window of time available to investigate an issue, use the resources at your disposal such as:
  - Request applicable logs from the customer.
  - Jump on a Zoom call with the customer if it would help gather reproduction steps (coordinate with the CSM).
  - Block time on your calendar (maximum 1 hour at a time) to dig into the issue further.
  - Escalate to other CSE's or CSA's.
  - Contact the developer on-call.

Note: For non-CSA engaged customer requests, CSE's are responsible for escalations to a CSA as needed. 

### Contact the developer on-call

The acting developer on-call rotation is reflected in the [📈KPIs spreadsheet (confidential Google sheet)](https://docs.google.com/spreadsheets/d/1Hso0LxqwrRVINCyW_n436bNHmoqhoLhC8bcbvLPOs9A/edit#gid=0&range=F2 ). The developer on-call is responsible for responses to technical Slack comments, Slack threads, and GitHub issues raised by customers and the community, which the CSE team cannot address.
- To reach the developer on-call for assistance, mention them in Fleet Slack using `@oncall` in the [#help-engineering](https://fleetdm.slack.com/archives/C019WG4GH0A) channel. 
  - Support issues should be handled in the relevant Slack channel rather than Direct Messages (DMs). This will ensure that questions and solutions can be easily referenced in the future. If it is necessary to use DMs to share sensitive information, a summary of the conversation should be posted in the Slack channel as well. 

- An automated weekly [on-call handoff](https://fleetdm.com/handbook/engineering#handoff) Slack thread in #g-engineering provides the opportunity to discuss highlights, improvements, and hand off ongoing issues.

### Contact the infrastructure engineer on-call

The acting infrastructure engineer on-call rotation is reflected in the [📈KPIs spreadsheet (confidential Google sheet)](https://docs.google.com/spreadsheets/d/1Hso0LxqwrRVINCyW_n436bNHmoqhoLhC8bcbvLPOs9A/edit#gid=0&range=F2 ). The individual on-call is responsible for responding to infrastructure-related Slack comments, Slack threads, and GitHub issues raised by customers and the community that the CSE team cannot address. These may be related to self-hosted or Fleet Managed cloud bugs or performance issues, which are suspected to be infrastructure-related. 
- To reach the infrastructure engineer on-call for assistance, a CSE or developer should mention them in Slack using `@infrastructure-oncall` in the [#help-infrastructure](https://fleetdm.slack.com/archives/C051QJU3D0V) channel or in the customer channel where the original request lives. 
  - Support issues must be handled in the relevant customer or internal Slack channel rather than Direct Messages (DMs). This will ensure that questions and solutions can be easily referenced in the future and help the infrastructure engineering team focused on their planned work.
  - A CSE or CSA must always triage and process suspected infrastructure issues before tagging in the infrastructure engineer on-call.
  - If your request for infrastructure is not urgent and/or not related to a suspected bug or performance issue impacting a customer, please create an issue on the [#help-customers kanban board](https://github.com/orgs/fleetdm/projects/79/views/1?filterQuery=) and @ mention the SVP of Customer Success to request prioritization. 


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
4. (Zapier) Send a text to the SVP of CS to begin the emergency request flow if triggered by the original sender. 

> **Note:** New customer channels that the automation will run in must be configured manually. Submit requests for additions to the Zapier administrator. 


### Generate a trial license key

1. Fleet's self-service license key creator is the best way to generate a proof of concept (POC) or renewal/expansion Fleet Premium license key. 
    - [Here is a tutorial on using the self-service method](https://www.loom.com/share/048474d7199048e1bf0c4fc106632129) (internal video)
    - Pre-sales license key DRI is the Director of Solutions Consulting
    - Post-sales license key DRI is the SVP of Customer Success

2. Legacy method: [create an opportunity issue](https://github.com/fleetdm/confidential/issues/new/choose) for the customer and follow the instructions in the issue for generating a trial license key.


### Respond to messages and alerts 

Customer Support Engineers (CSEs) are responsible for the first response to Slack messages in the [#fleet channel](https://osquery.slack.com/archives/C01DXJL16D8) of osquery Slack, MacAdmins Slack and dedicated customer Slack channels. 
- The 24/7 infrastructure on-call engineer is responsible for alarms related to fleetdm.com and Fleet Managed Cloud, as well as delivering 24/7 support for Fleet Premium customers when tagged in for assistance. Use [on-call runbooks](https://github.com/fleetdm/confidential/tree/main/infrastructure/runbooks#readme) to guide your response. Runbooks provide detailed, step-by-step instructions to quickly and effectively respond to and resolve most 24/7 on-call alerts.
- We respond within 1-hour or less during business hours and 4 hours outside business hours. Note that we do not need to have answers within 1 hour -- we need to at least acknowledge and collect any additional necessary information while researching/escalating to find answers internally.


### Maintain first responder SLA

The first responder on-call for Managed Cloud will take ownership of the @infrastructure-oncall alias in Slack first thing Monday morning. The previous week's on-call will provide a summary in the #help-customers Slack channel with an update on alarms that came up the week before, open issues with or without direct end-user impact, and other issues to keep an eye out for.
- **First responders:** Robert Fairburn, Jorge Falcon

Escalation of alarms will be done manually by the first responder according to the escalation contacts mentioned above. A [suspected outage issue](https://github.com/fleetdm/confidential/issues/new?assignees=&labels=%23outage%2C%23g-cx%2C%3Arelease&projects=&template=outage.md&title=Suspected+outage%3A+YYYY-MM-DD) should be created to track the escalation and determine root cause. 
- **Escalations (in order):** » Eric Shaw (fleetdm.com) » Zay Hanlon » Luke Heath » Mike McNeil

All infrastructure alarms (fleetdm.com and Managed Cloud) will go to #help-p1. When the current 24/7 on-call engineer is unable to meet the response time SLAs, it is their responsibility to arrange and designate a replacement who will assume the @infrastructure-oncall Slack alias.


### Communicate feedback on prioritized customer requests

When Fleet [prioritizes](https://fleetdm.com/handbook/company/product-groups#feature-fest) a new customer request, the Product Designer (PD) files a user story that's brought through [drafting](https://fleetdm.com/handbook/product-design#drafting).

After the user story is released, the PD will ask the appropriate Customer Success Manager (CSM) to bring the released improvements to the customer for feedback. When this happens, PD assigns the CSM and adds the `:help-customers` label.

If the improvements meet the customer's needs, the request issue is closed with a comment that @ mentions the PD. If the improvements are missing something in order to meet the customer's needs, the CSM adds feedback as comment (Gong snippet, Slack thread, or meetings notes), @ mention the PD, and unsassign themselves from the request issue.

### Manage DNS records

Fleet-managed DNS records are maintained in Cloudflare using Terraform.  
See [DNS management](https://github.com/fleetdm/confidential/tree/main/infrastructure/dns/dns-management.md) for how changes are reviewed, validated, and applied automatically.


### Process a self-service license dispenser refund

Refunds for Fleet Premium licenses purchased on the self-service license dispenser on fleetdm.com are processed in [Stripe](https://dashboard.stripe.com/). To refund a subscription: 
1. Log in to Stripe using the shared credentials from 1Password. 
2. Search for the user's email address, and select the subscription associated with their Stripe customer account. 
3. On the page for the user's subscription, select the "Actions" dropdown in the top right and choose "Cancel subscription". 
4. In the cancellation options, select the options to *cancel the subscription immediately*, *refund the last payment*, and *send the user a refund receipt*. 

Once you submit the form, Stripe will refund the user's payment and cancel their subscription.


### Respond to a data-deletion request

When a user requests that we delete all data we have stored about them, their data will need to be removed from the following places:
1. **fleetdm.com**
    - Create a confidential website request issue
    - If the user signed up for an account on fleetdm.com, you will need to create a confidential website request issue. A member of the #g-website working group will delete the account and let you know in a comment when the user account is deleted.
2.  **Salesforce**
    1. Search Salesforce for the user's email address, delete the contact record, and any related historical event records associated with the user's contact record.
3. **Stripe** 
    - If the user created an account on the Fleet website, a Stripe customer profile will have been created for their email address.
    - Follow these steps to delete the profile: 
        1. Log in to Stripe using the shared credentials in 1Password 
        2. Search for the user's email address
        3. Select the user's Stripe customer record
        4. Click the "Actions" dropdown in the upper right corner of the customer profile page and select delete.
 
## Rituals

<rituals :rituals="rituals['handbook/customer-success/customer-success.rituals.yml']"></rituals>

#### Stubs
The following stubs are included only to make links backward compatible.

##### Customer support service level agreements (SLAs)
Please see 📖[handbook/company/go-to-market-groups#customer-support-service-level-objectives-slos](https://fleetdm.com/handbook/company/go-to-market-groups#customer-support-service-level-objectives-slos).

##### Runbooks
Please see [Handbook/customer-success#respond-to-messages-and-alerts](https://www.fleetdm.com/handbook/customer-success#respond-to-messages-and-alerts)

<meta name="maintainedBy" value="zayhanlon">
<meta name="title" value="🌦️ Customer Success">

# Sales

This handbook page details processes specific to working [with](#contact-us) and [within](#responsibilities) this department.


## Team

| Role                                  | Contributor(s)           |
|:--------------------------------------|:------------------------------------------------------------------------------------------------------------------------|
| Chief Revenue Officer (CRO)                  | [Alex Mitchell](https://www.linkedin.com/in/alexandercmitchell/) _([@alexmitchelliii](https://github.com/alexmitchelliii))_
| Solutions Consulting (SC)                    | [Allen Houchins](https://www.linkedin.com/in/allenhouchins/) _([@allenhouchins](https://github.com/allenhouchins))_ <br> [Harrison Ravazzolo](https://www.linkedin.com/in/harrison-ravazzolo/) _([@harrisonravazzolo](https://github.com/harrisonravazzolo))_
| Channel Sales                                | [Tom Ostertag](https://www.linkedin.com/in/tom-ostertag-77212791/) _([@tomostertag](https://github.com/TomOstertag))_
| Account Executive (AE)                       | [Patricia Ambrus](https://www.linkedin.com/in/pambrus/) _([@ambrusps](https://github.com/ambrusps))_ <br> [Anthony Snyder](https://www.linkedin.com/in/anthonysnyder8/) _([@anthonysnyder8](https://github.com/AnthonySnyder8))_ <br> [Paul Tardif](https://www.linkedin.com/in/paul-t-750833/) _([@phtardif1](https://github.com/phtardif1))_ <br> [Kendra McKeever](https://www.linkedin.com/in/kendramckeever/) _([@KendraAtFleet](https://github.com/KendraAtFleet))_


## Contact us

- To **make a request** of this department, [create an issue](https://github.com/fleetdm/confidential/issues/new?assignees=&labels=%23g-sales&projects=&template=custom-request.md&title=Request%3A+_______________________) and a team member will get back to you within one business day (If urgent, mention a [team member](#team) in the [#g-sales](https://fleetdm.slack.com/archives/C030A767HQV)).
  - Any Fleet team member can [view the kanban board](https://app.zenhub.com/workspaces/g-sales-64fbb46c65f9ff003a1530a8/board?sprints=none) for this department, including pending tasks and the status of new requests.
  - Please **use issue comments and GitHub mentions** to communicate follow-ups or answer questions related to your request.


## Responsibilities

The Sales department is directly responsible for attaining the revenue goals of Fleet and helping to deliver upon our customers' objectives.


### Set up a Fleet trial

You can set up a Fleet Managed Cloud environment for a prospect with >300 hosts, or you can help them generate a trial license key to configure on their own self-managed Fleet server.

- **To set up a new Fleet Managed Cloud environment** for a user: First, [create a "New customer environment" issue](https://fleetdm.com/docs/configuration/fleet-server-configuration#license-key).  Then, once the environment is set up, you'll get a notification and you can let the user know.
- **To set up only a trial license key** for a user's self-managed Fleet server: Point the user towards fleetdm.com/start, where they can sign up and choose to "Run your own trial with Docker".  On that page, they'll see a license key located in the `fleectl preview` CLI instructions, and they can configure this by copying and pasting it as the [`FLEET_LICENSE_KEY`](https://fleetdm.com/docs/configuration/fleet-server-configuration#license-key)  environment variable on the server(s) where Fleet is deployed.

### Demo Fleet to a prospect

To run a demo for a prospect, follow the relevant steps in ["Why Fleet?"](https://docs.google.com/document/d/1E0VU4AcB6UTVRd4JKD45Saxh9Gz-mkO3LnGSTBDLEZo/edit#heading=h.vfxwnwufxzzi)

### Introduce Fleet's CEO

To get the CEO's attention and introduce him to an account, follow the relevant steps in ["Why Fleet?"](https://docs.google.com/document/d/1E0VU4AcB6UTVRd4JKD45Saxh9Gz-mkO3LnGSTBDLEZo/edit#heading=h.vfxwnwufxzzi)


### Track an objection

To track an objection you heard from a prospect, follow the relevant steps in ["Why Fleet?"](https://docs.google.com/document/d/1E0VU4AcB6UTVRd4JKD45Saxh9Gz-mkO3LnGSTBDLEZo/edit#heading=h.vfxwnwufxzzi)


### Change a contact's organization in Salesforce

Use the following steps to change a contact's organization in Salesforce:
- If the contact's organization in Salesforce is incorrect but their new organization is unknown, navigate to the contact in Salesforce and change the "Account name" to "?" and save.
- If the contact's organization in Salesforce is incorrect and we know where they're moving to, navigate to the contact in Salesforce, change the "Account name" to the contact's new organization, and save.


### Send an order form

In order to be transparent, Fleet sends order forms within 30 days of opportunity creation in most cases. All quotes and purchase orders must be approved by the CRO and 🌐 [Head of Digital Experience](https://fleetdm.com/handbook/digital-experience#team) before being sent to the prospect or customer. Often, the CRO will request legal review of any unique terms required. To prepare and send a subscription order form the Fleet owner of the opportunity (usually AE or CSM) will: 

1. Navigate to the ["Template gallery"](https://docs.google.com/document/u/0/?tgif=d&ftv=1) in Google Docs and create a copy of the "TEMPLATE - Order form".
2. Add/remove table rows as needed for multi-year deals.
3. Where possible, include a graphic of the customer's logo. Use good judgment and omit if a high-quality graphic is unavailable. If in doubt, ask Digital Experience for help.

> **Important**
> - All changes to the [subscription agreement template](https://docs.google.com/document/d/1X4fh2LsuFtAVyQDnU1ZGBggqg-Ec00vYHACyckEooqA/edit?tab=t.0), or [standard terms](http://fleetdm.com/terms) must be brought to ["🦢🗣 Design review (#g-digital-experience)"](https://app.zenhub.com/workspaces/-g-digital-experience-6451748b4eb15200131d4bab/board?sprints=none) for approval.
> - All non-standard (from another party) subscription agreements, NDAs, and similar contracts require legal review from Digital Experience before being signed. [Create an issue to request legal review](https://github.com/fleetdm/confidential/blob/main/.github/ISSUE_TEMPLATE/contract-review.md).

4. In the internal Slack channel for the deal, at-mention the CRO and the Head of Digital Experience with a link to the docx version of the order and ask them to approve the order form.
5. Once approved, send the order to the prospect. 


### Send an NDA to a customer

- Fleet uses "Non-Disclosure Agreements" (NDAs) to protect the company and the companies we collaborate with. Always offer to send Fleet's NDA and, whenever possible, default to using the company's version. To send an NDA to a customer, follow these steps: 
1. If a customer has no objections to using Fleet's NDA, route the NDA to them for signature using the "🙊 NDA (Non-disclosure agreement)" template in [DocuSign](https://apps.docusign.com/send/home).
> If a customer would like to review the NDA first, download a .docx of [Fleet's NDA](https://docs.google.com/document/d/1gQCrF3silBFG9dJgyCvpmLa6hPhX_T4V7pL3XAwgqEU/edit?usp=sharing) and send it to the customer.
2. If the customer has no objections, route the NDA using the template in DocuSign (do not upload and use the copy you emailed to the customer).
3. If the customer "redlines" (i.e. wants to change) the NDA, follow the [contract review process](https://fleetdm.com/handbook/company/communications#getting-a-contract-reviewed) so that Digital Experience can look over any proposed changes and provide guidance on how to proceed.


### Close a new customer deal

To close a deal with a new customer (non-self-service), create and complete a GitHub issue using the ["Sale" issue template](https://github.com/fleetdm/confidential/issues/new?assignees=alexmitchelliii&labels=%23g-sales&projects=&template=3-sale.md&title=New+customer%3A+_____________).


### Process a security questionnaire

- The AE will [use the handbook](https://fleetdm.com/handbook/company/communications#vendor-questionnaires) to answer most of the questions with links to appropriate sections in the handbook. After this first pass has been completed, and if there are outstanding questions, the AE will [assign the issue to Digital Experience (#g-digital-experience)](https://fleetdm.com/handbook/digital-experience#contact-us)  with a requested timeline for completion defined.
- Digital Experience consults the handbook to validate that nothing was missed by the AE. After the second pass has been completed, and if there are outstanding questions, Digital Experience will [reassign the issue to Sales (#g-sales)](https://fleetdm.com/handbook/sales#contact-us) for intake.
- The issue will be assigned to the Solutions Consultant (SC) associated to the opportunity in order to complete any unanswered questions.
- The SC will search for unanswered questions and confirm again that nothing was missed from the handbook. Content missing from the handbook will need to be added via PR by the SC. Any unanswered questions after this pass has been completed by the SC will need to be [escalated to the Infrastructure team (#g-customer-success)](https://fleetdm.com/handbook/customer-success#contact-us) with the requested timeline for completion defined in the issue. Once complete, the infra team will assign the issue back to the #g-sales board.
- Any questions answered by the infra team will be added to the handbook by the SC.


<!-- 2024-11-16 We noticed some content in these sections was outdated, so we're using this opportunity to try out a different structure ± altitude level for the content for on this page


### Review rep activity

Following up with people interested in Fleet is an important part of finding out whether or not they'd like to continue the process of buying the product.  It is also very important not to be annoying.  At Fleet, team members follow up with people, but not too often.

To help coach reps and avoid being annoying to Fleet users, Fleet reviews rep activity on a regular basis following these steps:
1. In Salesforce, visit the activity report on your dashboard.
2. For each rep, review recent activity from the last 30 days across all of that rep's accounts.
3. If outreach is too frequent or doesn't fit the company's strategy, then set up a 30 minute coaching session to discuss with the rep.

Every week, AEs will review the status of all qualified opportunities with leadership in an opportunity pipeline review meeting. For this meeting, reps will:
1. Update the following information in Salesforce for every opp:
  - Contacts (and Roles)
  - Amount
  - Close date
  - Stage
  - Next steps
2. Make sure all contacts have been sent a connection request from Mike McNeil.
3. Identify and discuss where gaps are in [MEDDPICC](https://handbook.gitlab.com/handbook/sales/meddppicc/).
4. Relay how many meetings they had with attendees from both IT and security this week.


### Conduct a POV

We use the "tech eval test plan" as a guide when conducting a "POV" (Proof of Value) with a prospect. This planning helps us avoid costly detours that can take a long time, and result in folks getting lost. The tech eval test plan is the main document that will track success criteria for the tech eval.

When we have had sufficient meetings and demos, including an overview demo and a customized demo, and we have qualified the prospect, when the prospect asks to "kick the tires/do a POC/do a technical evaluation", the AE moves the opportuity to "Stage 3 - Requested POV" phase in Salesforce. Automation will generate the tech eval test plan. This doc will exist in Google Drive> Sales> Opportunities> "Account Name". 

The AE and SC will work together to scope the POV with the prospect in this stage. The AE and SC will work together to answer the following questions:

1. Do we have a well-defined set of technical criteria to test and are we confident that Fleet can meet this criteria to achieve a technical win?
2. Do we have a timeline agreed upon?
3. What are the key business outcomes that will be verified as a result of completing the tech eval?

If the above questions are answered successfully, the opportunity should progress to tech eval. If we cannot answer the questions above successfully, then the POV should not start unless approved by the CRO.

During Stage 4, follow this process:
1. SC creates a [tech eval issue](https://github.com/fleetdm/confidential/issues/new?assignees=&labels=%23g-sales&projects=&template=technical-evaluation.md&title=Technical+evaluation%3A+___________________).
2. SC updates the issue labels to include: "~sc, :tech-eval" and the obfuscated "prospect-codename" label. See [Assign a customer a codename](https://fleetdm.com/handbook/customer-success#assign-a-customer-codename). Instead of
   "customer-codename", prospects are labeled "prospect-codename". When a prospect purchases Fleet, the SC will edit this label from "prospect-codename" to "customer-codename".
3. SC sets the appropriate sprint duration based on the defined timelines and an estimation of effort in points.
4. SC converts the issue to an Epic. All issues related to this prospect tech eval (ie: cloud instance deployments, etc.) should be added to the newly created epic.
5. All check-in meetings and notes taken are documented in the tech eval test plan document. Any TODO item will be added as a comment to the tech eval issue epic.
6. The SC presents the tech eval test plan and feature tracker used for the tech eval to the CS team upon the prospect's transition to Fleet customer.


### Hand off a technical evaluation to a temporary DRI

Tech evals will have a DRI at all times; should the DRI be unavailable (ie: vacation), a hand off process to a temporary DRI will be required. In advance of vacation time (target one week in advance), refer to the following examples and review with each individual that will act as the temporary DRI for the technical evaluation while you are away. This can be documented as a google doc or can be added to the relevant tech eval epic issue in github.

Ensure that our valued customers know that you will be away and that the temporary DRI has been debriefed on their setup and can handle any technical questions that come up. 

```
Active Technical Evaluations (TechEvals), workshops that need monitoring:

Account Name:
Issue link:
Status: Cloud instance deployed, 1st enablement session complete, MDM assets generated, need to create infrastructure request to get deployed
AE: 
Background: First workshop completed <date>
Documentation: link to Tech eval plan
Slack Channel (external): #fleet-at-
Slack Channel (internal): #op-
Temp Transfer to: Temp technical DRI


Likely to convert to demo:

Account name:
Issue Link:
Status: RFP complete, video content delivered via consensus, awaiting further requests for Demo (Live)..
AE: 
Background: 
Documentation: gong links, meeting minutes links, summary 
Slack Channel (external): n/a
Slack Channel (internal): #op-
Temp Transfer to: Temp technical DRI

```

- **Documenting a prospect call:** When we do prospect calls, add the prospect's name in both the google doc title and the heading, ex. "Alex + Natalie (Fleet + Acme Co)."  This makes it easier when searching for the document later. 
- **Before a prospect call(48hrs):** Check the calendar invite 48hrs before the meeting to determine if the prospect has accepted the invitation.
  - If the prospect has not accepted the invitation, reach out to confirm attendance (e.g., EAs, email, Slack).
  - Test the Zoom Meeting link to make sure that it is working.
  - Make sure that agenda documents are attached and accessible to meeting attendees (as appropriate to the situation).
- **Day of the prospect call:** Join the meeting two to three minutes before the start time.

- **Missed prospect call:** If the prospect does not join the call after three minutes, contact the prospect with
  - Slack, if we have a shared channel.
  - email, using the email address from the calendar invite.
  - LinkedIn, send a direct message.
  - phone, try finding their number to text and/or call (as appropriate to the device type: landline vs. cell phone).
  - an alternative date and time. Suggest two to three options from which the prospect can choose.
    - Confirm that contact information is accurate and that the prospect can receive and access meeting invites.
-->  




## Rituals

<rituals :rituals="rituals['handbook/sales/sales.rituals.yml']"></rituals>


#### Stubs
The following stubs are included only to make links backward compatible.

##### Fleet Premium
##### Fleet Free
##### Emergency (P0) request communications
Please see [handbook/company/communications#customer-support-service-level-agreements-slas](https://fleetdm.com/handbook/company/communications#customer-support-service-level-agreements-slas) for all sections above.

##### Submit a customer contract for legal review
##### Standard terms
##### Non-standard NDAs
##### Reviewing subscription agreement
##### Submit a customer contract
Please see [handbook/sales#create-a-customer-agreement](https://fleetdm.com/handbook/sales#create-a-customer-agreement) for all sections above.


<meta name="maintainedBy" value="alexmitchelliii">
<meta name="title" value="🐋 Sales">

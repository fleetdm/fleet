# Sales

This handbook page details processes specific to working [with](#contact-us) and [within](#responsibilities) this department.


## Team

| Role                                  | Contributor(s)           |
|:--------------------------------------|:------------------------------------------------------------------------------------------|
| Head of Customers                     | [Alex Mitchell](https://www.linkedin.com/in/alexandercmitchell/) _([@alexmitchelliii](https://github.com/alexmitchelliii))_
| Account Executive (AE)                | <sup><sub> _See [🦄 Go-To-Market groups](https://fleetdm.com/handbook/company/go-to-market-groups#current-gtm-groups)


## Contact us

- To **make a request** of this department, [create an issue](https://github.com/fleetdm/confidential/issues/new?assignees=&labels=%23g-sales&projects=&template=custom-request.md&title=Request%3A+_______________________) and a team member will get back to you within one business day (If urgent, mention a [team member](#team) in the [#g-sales](https://fleetdm.slack.com/archives/C030A767HQV)).
  - Any Fleet team member can [view the kanban board](https://app.zenhub.com/workspaces/g-sales-64fbb46c65f9ff003a1530a8/board?sprints=none) for this department, including pending tasks and the status of new requests.
  - Please **use issue comments and GitHub mentions** to communicate follow-ups or answer questions related to your request.


## Responsibilities

The Sales department is directly responsible for attaining the revenue goals of Fleet and helping to deliver upon our customers' objectives.


### Demo Fleet to a prospect

To run a demo for a prospect, follow the relevant steps in ["Why Fleet?"](https://docs.google.com/document/d/1E0VU4AcB6UTVRd4JKD45Saxh9Gz-mkO3LnGSTBDLEZo/edit#heading=h.vfxwnwufxzzi)

### Introduce Fleet's CEO

To get the CEO's attention and introduce him to an account, follow the relevant steps in ["Why Fleet?"](https://docs.google.com/document/d/1E0VU4AcB6UTVRd4JKD45Saxh9Gz-mkO3LnGSTBDLEZo/edit#heading=h.vfxwnwufxzzi)


### Track an objection

To track an objection you heard from a prospect, follow the relevant steps in ["Why Fleet?"](https://docs.google.com/document/d/1E0VU4AcB6UTVRd4JKD45Saxh9Gz-mkO3LnGSTBDLEZo/edit#heading=h.vfxwnwufxzzi)


### Change a contact's organization in Salesforce

Use the following steps to change a contact's organization in Salesforce:
- If the contact's organization in Salesforce is incorrect but their new organization is unknown, navigate to the contact in Salesforce and change the "Account name" to "[? (Unsure)](https://fleetdm.lightning.force.com/lightning/r/Account/0014x000025JC8DAAW/view)" and save.
- If the contact's organization in Salesforce is incorrect and we know where they're moving to, navigate to the contact in Salesforce, change the "Account name" to the contact's new organization, and save.


### Review Salesforce opportunities

Every week, the sales manager will review the necessary opportunities with internal stakeholders. The AE or CSM who owned the deal, their manager, the Head of Marketing, Head of Digital Experience, and the CEO will review all opportunities on the "[Ω Ops for review](https://fleetdm.lightning.force.com/lightning/r/Report/00OUG000001qyE12AI/view?queryScope=userFolders)" report in Salesforce. Opportunities will be reviewed for a number of reasons including: 
- If the opportunity is older than 30 days but the prospect hasn't been sent an order form yet.
- Any closed lost new business or expansion opportunity from the previous week.
- Any opportunity with a closed date pushed into a different quarter.

If no opportunities meet these criteria, the meeting is used to discuss the oldest opportunities and close any that are stalled.


### Send a subscription quote

Reseller partners occasionally reach out and ask Fleet for a quote on behalf of customers. Use the following steps to provide a subscription quote:
1. Navigate to the [Google Docs template gallery](https://docs.google.com/document/u/0/?ftv=1&tgif=d) and make a copy of the "TEMPLATE - 3EYE - Subscription quote" document.
2. Assign the "Quote #" by combining the "ISO date" (YYYYMMDD) and "Total amount" (e.g. "YYYYMMDD-000000" or "20250212-128400").
3. Make sure the "Customer", "Customer contact", "Total term", "Effective dates", "Billing frequency/timing", and "Payment terms" fields are correctly reflected on the subscription quote
4. Insert the correct "Quantity", "Total list price", "Distributor price", and "Effective price" in the table. If a discount is applied to the quote, insert the appropriate ISO date in the "**Discount.**" term at the bottom of the page.
5. Insert the total effective price in the "Total amount (USD)" field and send the quote. 


### Send an order form

In order to be transparent, Fleet sends order forms within 30 days of opportunity creation in most cases. All quotes and purchase orders must be approved by the CRO and 🌐 [Head of Digital Experience](https://fleetdm.com/handbook/digital-experience#team) before being sent to the prospect or customer. Often, the CRO will [request legal review](https://fleetdm.com/handbook/company/communications#getting-a-contract-reviewed) of any unique terms required. To prepare and send a subscription order form the Fleet owner of the opportunity (usually AE or CSM) will: 

1. Navigate to the Salesforce opportunity and click the "Create a subscription order form" link (top-right corner of the op page) to copy the "[TEMPLATE - Order form](https://docs.google.com/document/u/0/?tgif=d&ftv=1). 
2. Move the order for to the "[Contract drafts](https://drive.google.com/drive/u/0/folders/1G1JTpFxhKZZzmn2L2RppohCX5Bv_CQ9c)" folder in Google Drive.
3. Edit the order form to be specific to the customer (e.g. add/remove table rows as needed for multi-year deals). Where possible, include a graphic of the customer's logo. Use good judgment and omit if a high-quality graphic is unavailable. If in doubt, ask Digital Experience for help.

> **IMPORTANT** To ensure product language is consistent, any changes to the standard order form template (including subscription appendix) must be submitted to ["🦢🗣 Design review (#g-digital-experience)"](https://app.zenhub.com/workspaces/-g-digital-experience-6451748b4eb15200131d4bab/board?sprints=none) for approval.

4. In the internal Slack channel for the deal, at-mention the CRO and the Head of Digital Experience with a link to the docx version of the order and ask them to approve the order form.
5. Once approved, copy the Google Doc URL to the "Order form URL" field on the Salesforce opportunity and send the order to the prospect. 

> Every week, any proposal not sent within 30 days of its creation in Salesforce should be reviewed and closed lost. The review of these opportunities and exceptions for them of one (1) week or less are the responsibility of the sales manager.  For exceptions of more than one week, escalate to the CEO.


### Send an NDA to a customer

- Fleet uses "Non-Disclosure Agreements" (NDAs) to protect the company and the companies we collaborate with. Always offer to send Fleet's NDA and, whenever possible, default to using the company's version. To send an NDA to a customer, follow these steps: 
1. If a customer has no objections to using Fleet's NDA, route the NDA to them for signature using the "🙊 NDA (Non-disclosure agreement)" template in [DocuSign](https://apps.docusign.com/send/home).
> If a customer would like to review the NDA first, download a .docx of [Fleet's NDA](https://docs.google.com/document/d/1gQCrF3silBFG9dJgyCvpmLa6hPhX_T4V7pL3XAwgqEU/edit?usp=sharing) and send it to the customer.
2. If the customer has no objections, route the NDA using the template in DocuSign (do not upload and use the copy you emailed to the customer).
3. If the customer "redlines" (i.e. wants to change) the NDA, follow the [contract review process](https://fleetdm.com/handbook/company/communications#getting-a-contract-reviewed) so that Digital Experience can look over any proposed changes and provide guidance on how to proceed.


### Close a new customer deal

Once an order form has been signed by both Fleet and the new customer, create and complete a ["🤝 Sale" issue](https://github.com/fleetdm/confidential/issues/new?assignees=alexmitchelliii&labels=%23g-sales&projects=&template=3-sale.md&title=New+customer%3A+_____________).


### Process a security questionnaire

- The AE will [use the handbook](https://fleetdm.com/handbook/company/communications#vendor-questionnaires) to answer most of the questions with links to appropriate sections in the handbook. After this first pass has been completed, and if there are outstanding questions, the AE will [assign the issue to Digital Experience (#g-digital-experience)](https://fleetdm.com/handbook/digital-experience#contact-us) with a requested timeline for completion defined.
- Digital Experience consults the handbook to validate that nothing was missed by the AE. After the second pass has been completed, and if there are outstanding questions, Digital Experience will [reassign the issue to Sales (#g-sales)](https://fleetdm.com/handbook/sales#contact-us) for intake.
- The issue will be assigned to the Solutions Consultant (SC) associated to the opportunity in order to complete any unanswered questions.
- The SC will search for unanswered questions and confirm again that nothing was missed from the handbook. Content missing from the handbook will need to be added via PR by the SC. Any unanswered questions after this pass has been completed by the SC will need to be [escalated to the Infrastructure team (#g-customer-success)](https://fleetdm.com/handbook/customer-success#contact-us) with the requested timeline for completion defined in the issue. Once complete, the infra team will assign the issue back to the #g-sales board.
- Any questions answered by the infra team will be added to the handbook by the SC.



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

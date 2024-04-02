# Sales
This handbook page details processes specific to working [with](#contact-us) and [within](#responsibilities) this department.

## Team
| Role                                  | Contributor(s)           |
|:--------------------------------------|:------------------------------------------------------------------------------------------------------------------------|
| Chief Revenue Officer (CRO)           | [Alex Mitchell](https://www.linkedin.com/in/alexandercmitchell/) _([@alexmitchelliii](https://github.com/alexmitchelliii))_
| 🏹 [Customer Success](https://www.fleetdm.com/handbook/customer-success#responsibilities) | [Customer Success team members](https://www.fleetdm.com/handbook/customer-success#team)
| Director of Solutions Consulting      | [Dave Herder](https://www.linkedin.com/in/daveherder/) _([@dherder](https://github.com/dherder))_
| Solutions Consultant (SC)             | [Will Mayhone](https://www.linkedin.com/in/william-mayhone-671977b6/) _([@willmayhone88](https://github.com/willmayhone88))_
| Head of Public Sector                 | [Keith Barnes](https://www.linkedin.com/in/keith-barnes-8b666/) _([@KAB703](https://github.com/KAB703))_
| Account Executive (AE)                | [Tom Ostertag](https://www.linkedin.com/in/tom-ostertag-77212791/) _([@TomOstertag](https://github.com/TomOstertag))_, [Patricia Ambrus](https://www.linkedin.com/in/pambrus/) _([@ambrusps](https://github.com/ambrusps))_, [Anthony Snyder](https://www.linkedin.com/in/anthonysnyder8/) _([@AnthonySnyder8](https://github.com/AnthonySnyder8))_, [Paul Tardif](https://www.linkedin.com/in/paul-t-750833/) _([@phtardif1](https://github.com/phtardif1))_


## Contact us
- To **make a request** of this department, [create an issue](https://github.com/fleetdm/confidential/issues/new?assignees=&labels=%23g-sales&projects=&template=custom-request.md&title=Request%3A+_______________________) and a team member will get back to you within one business day (If urgent, mention a [team member](#team) in the [#g-sales](https://fleetdm.slack.com/archives/C030A767HQV)).
  - Any Fleet team member can [view the kanban board](https://app.zenhub.com/workspaces/g-sales-64fbb46c65f9ff003a1530a8/board?sprints=none) for this department, including pending tasks and the status of new requests.
  - Please **use issue comments and GitHub mentions** to communicate follow-ups or answer questions related to your request.

<!---
Rituals (out-dated 2023-10-19)

| Opportunity pipeline review | Weekly | Agenda: Go through every [open opportunity](https://fleetdm.lightning.force.com/lightning/o/Opportunity/list?filterName=00B4x00000CTHZIEA5) and update the next steps, amounts, dates, and status (including choosing Closed Lost if no communications for >= 45 days). | Alex Mitchell

| Lead pipeline review  | Weekly | Agenda: Review leads by status/stage; make sure SLAs are met. Clean up Open MQL list. Ask CRO if questions. | Alex Mitchell |
--->

## Responsibilities
The Sales department is directly responsible for attaining the revenue goals of Fleet and helping to deliver upon our customers' objectives.

### Onboard a new sales team member
Once the standard Fleetie onboarding issue is complete, create a new ["Sales team onboarding"](https://github.com/fleetdm/confidential/issues/new?assignees=&labels=%23g-sales&projects=&template=sales-team-onboarding.md&title=Sales%20onboarding%3A_____________) issue and complete it.

### Send a quote
During the buying cycle, the champion will need to start the process to secure funding in cooperation with the economic buyer and the finance org.

All quotes and purchase orders must be approved by CRO before being sent to the prospect or customer. Often, the CRO will request Fleet business operations/legal of any unique terms required.

The Fleet owner of the opportunity (usually AE or CSM) will prepare a quote and/or a Purchase Order when requested.
- Because the champion may need to socialize "what is Fleet" or "what are we getting when buying Fleet," it is most often best to send the quote in [slide form](https://docs.google.com/presentation/d/15kbqm0OYPf1OmmTZvDp4F7VvMERnX4K6TMYqCYNr-wI/edit?usp=sharing).
- Docusign can be used to create a [standard Purchase Order](https://www.loom.com/share/Loom-Message-16-January-2023-2ba8cf195ec645ebabac267d7df59823?sid=214f8c6b-beb3-427a-a3a8-e8c20b5dc350) if no special terms or pricing are needed.

### Obtain a copy of Fleet's W-9
A recent signed copy of Fleet's W-9 form can be found in [this confidential PDF in Google Drive](https://drive.google.com/file/d/1ugXazEBk1oVm_LqGbYNsIFECcv5jXLA9/view?usp=drivesdk).

### Provide payment information to a prospect
For customers with large deployments, Fleet accepts payment via wire transfer or electronic debit (ACH/SWIFT).

Provide remittance information to customers by exporting ["💸 Paying Fleet"](https://docs.google.com/document/d/1KP_-x9c1x3sS1X9Q8Wlib2H7tq69xRONn1KMA3nVFQc/edit) into a PDF, then sending that to the prospect.

### Review rep activity
Following up with people interested in Fleet is an important part of finding out whether or not they'd like to continue the process of buying the product.  It is also very important not to be annoying.  At Fleet, team members follow up with people, but not too often.

To help coach reps and avoid being annoying to Fleet users, Fleet reviews rep activity on a regular basis following these steps:
1. In Salesforce, visit the activity report on your dashboard.  (TODO: taylor will replace this and/or link it)
2. For each rep, review recent activity from the last 30 days across all of that rep's accounts.
3. If outreach is too frequent or doesn't fit the company's strategy, then set up a 30 minute coaching session to discuss with the rep.


### Validate Salesforce data (RevOps) 
In order to maintain a consistent contributor experience in Salesforce, we log in to make sure the structure of Salesforce data continues to look correct based on processes started elsewhere. Then we can look and see that the goals we want to achieve as a business are in line with our view inside Salesforce by conducting the following checkup. Any discrepancies between how information is presented in Salesforce and what should be in there per this ritual should be flagged so that they can be fixed or discussed.

1. Make sure the default tabs for a standard user include a detailed view of contacts, opportunities, accounts, and leads. No other tabs should exist.

2. Click the accounts tab and check for the following: 

* The default filter is Customers when you click on the accounts tab. Click on an account to continue.
* Click on a customer and make sure billing address, parent account, LinkedIn company URL, CISO employees (#), employees, and industry appear first at the top of the account. 
* Useful links section should appear in the top right section of the account page. It includes links to purchase orders (POs), signed subscription agreements, invoices sent, meeting notes, and signed NDA. Clicking these links should search the appropriate repository for the requested information pertaining to the customer. All meeting notes should be saved in the [Meeting notes](https://drive.google.com/drive/folders/18e-rVadHG0T5w98OKMngM-yv-K9SXaOq) folder in Google Drive with the account name and date in the title. We do not use the notes feature on "accounts" or "opportunities" in Salesforce. 
* Additional information section should include fields for account (customer) name first, account rating, LinkedIn sales navigator URL, LinkedIn company URL, and my LinkedIn overlaps. Make sure the LinkedIn links work.
* Accounting section should include the following fields: invoice sent (latest), the payment received on (latest), subscription end date (latest), press approval field, license key, total opportunities (#), deals won (#), close date (first deal), cumulative revenue, payment terms, billing address, and shipping address. 
* Opportunities, meeting notes, and activity feed should appear on the right.  

3. Click on the opportunities tab and check for the following:

* Default filter should be all opportunities. Open an opportunity to continue.
* Section at the top of the page should include fields for account name, amount, close date, next step, and opportunity owner.
* Opportunity information section should include fields for account name, opportunity name (should have the year on it), amount, next step, next step's due date, close date, and stage.
* The accounting section here should include: up to # of hosts, type, payment terms, billing process, term, reseller, effective date, subscription end date, invoice sent, and the date payment was received.
* Stage history, activity feed, and LinkedIn sales navigator should appear at the right.  

4. Click on the contacts tab and check for the following:

* Default filter should be all contacts. Open a contact to continue.
* Top section should have fields for the contact's name, job title, department, account name, LinkedIn, and Orbit feed. 
* The second section should have fields for LinkedIn URL, account name, name, title, is champion, and reports to
* Additional information should have fields for email, personal email, Twitter, GitHub, mobile, website, orbit feed, and description.
* Related contacts section should exist at the bottom, activity feed, meeting notes reminder, and manager information should appear on the right. 

5. Click on the leads tab and check for the following:

* Default filter should be all leads. Open a lead to continue.
* There should be fields for name, lead source, lead status, and rating.


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

### To schedule Solutions Consultant (SC) for a prospect meeting
To schedule an [ad hoc meeting](https://www.vocabulary.com/dictionary/ad%20hoc) with a Fleet prospect, the Account Executive (AE) will [open an issue](https://github.com/fleetdm/confidential/issues/new?assignees=&labels=%23g-sales%2C%23solutions-consultant%2C%3Adiscovery%2C%3Ademo%2C%3Ascoping%2C%3Atech-eval&projects=&template=custom-request.md&title=prospect+name+-+prep+%28date%29+-+discovery%2Cdemo%2Cscoping+%28date%29). 
 - Use [this calendly link](https://calendly.com/fleetdm/talk-to-a-solutions-consultant) to obtain SC availability.
 - The AE will populate this issue with the appropriate dates for an internal prep meeting as well as the dates for the external prospect meeting.
 - Do not assign the issue. The Director of Solutions Consulting will assign the issue.
 - Ensure that the product category is defined ("Endpoint ops", "Device management", or "Vulnerability management") in the description of the issue.

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

### Create a customer agreement
- **Contract terms:** Fleet's subscription agreement is available at [fleetdm.com/terms](https://fleetdm.com/terms). 
  - **Effective date:** The start date for the subscription service.
  - **Close date:** The date the last party to the contract signed the agreement.
  - **Invoice date:** The date that Fleet sent the invoice to the customer.
- Fleeties can find a summary of contract terms in the relevant [customer's Salesforce opportunity.](https://fleetdm.lightning.force.com/lightning/o/Opportunity/list?filterName=Recent)

- **Standard terms:** For all subscription agreements, NDAs, and similar contracts, Fleet maintains a [standard set of terms and maximum allowable adjustments for those terms](https://docs.google.com/spreadsheets/d/1gAenC948YWG2NwcaVHleUvX0LzS8suyMFpjaBqxHQNg/edit#gid=1136345578). Exceptions to these maximum allowable adjustments always require CEO approval, whether in the form of redlines to Fleet's agreements or in terms on a prospective customer's own contract.

> All non-standard (from another party) subscription agreements, NDAs, and similar contracts require legal review from the Business Operations department before being signed. [Create an issue to request legal review](https://github.com/fleetdm/confidential/blob/main/.github/ISSUE_TEMPLATE/contract-review.md).

### Close a new customer deal
To close a deal with a new customer (non-self-service), create and complete a GitHub issue using the ["Sale" issue template](https://github.com/fleetdm/confidential/issues/new?assignees=hughestaylor&labels=%23g-business-operations&projects=&template=3-sale.md&title=New+customer%3A+_____________).

### Change customer credit card number
You can help a Premium license dispenser customers change their credit card by directing them to their [account dashboard](https://fleetdm.com/customers/dashboard). On that page, the customer can update their billing card by clicking the pencil icon next to their billing information.


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

##### Customer codenames
Please see [Handbook/sales#assign-a-customer-codename](https://www.fleetdm.com/handbook/sales#assign-a-customer-codename)

##### Document customer requests
Please see [handbook/customer-success#document-customer-requests](https://fleetdm.com/handbook/customer-success#document-customer-requests)

##### Generate a trial license key
Please see [handbook/customer-success#generate-a-trial-license-key](https://fleetdm.com/handbook/customer-success#generate-a-trial-license-key)

#### Create customer support Issue
Please see [handbook/customer-success#create-customer-support-issue](https://fleetdm.com/handbook/customer-success#create-customer-support-issue)

<meta name="maintainedBy" value="alexmitchelliii">
<meta name="title" value="🐋 Sales">

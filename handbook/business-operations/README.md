# Business Operations
This handbook page details processes specific to working [with](#contact-us) and [within](#responsibilities) the Business Operations (BizOps) department.

## What we do
The BizOps department is directly responsible for these traditional functions: People, Finance, Legal, IT, and Revenue Operations (RevOps).  

## Team
| Role                          | Contributor(s)           |
|:------------------------------|:-----------------------------------------------------------------------------------------------------------|
| Head of Business Operations   | [Joanne Stableford](https://www.linkedin.com/in/joanne-stableford/) _([@jostableford](https://github.com/JoStableford))_
| Business Operations Engineer  | [Nathan Holliday](https://www.linkedin.com/in/nathanael-holliday/) _([@hollidayn](https://github.com/hollidayn))_
| Head of Revenue Operations    | [Taylor Hughes](https://www.linkedin.com/in/taylorhughes834/) _([@hughestaylor](https://github.com/hughestaylor))_


## Contact us
- To make a request of this department, [create an issue](https://github.com/fleetdm/confidential/issues/new?assignees=&labels=%23g-business-operations&projects=&template=custom-request.md&title=Request%3A+_______________________) and a team member will get back to you within one business day.
  - Please use issue comments and GitHub mentions to communicate follow-ups or answer questions related to your request.
  - Any Fleet team member can [view the kanban board](https://app.zenhub.com/workspaces/-g-business-operations-63f3dc3cc931f6247fcf55a9/board?sprints=none) for this department, including pending tasks and the status of new requests.
  - Submit legal questions and requests to Business Operations department. (Business Operations will escalate first-of-its-kind agreements to the CEO.  The CEO will review business terms and consult with lawyers as necessary.)
- If urgent, or if you need help submiting your request, mention a [team member](#team) in the [#g-business-operations](https://fleetdm.slack.com/archives/C047N5L6EGH) Slack channel.
  - If you need help during onboarding ask in [#help-onboarding](https://fleetdm.slack.com/archives/C02QG7P5571).
  - For all questions related to your company credit card, ask in [#help-brex](https://fleetdm.slack.com/archives/C0396TYH4EP).


## Tools we use
There are a number of tools that are used throughout Fleet. Some of these tools are used company-wide, while others are department-specific. You can see a list of those tools in ["Tools we use" (private Google doc)](https://docs.google.com/spreadsheets/d/170qjzvyGjmbFhwS4Mucotxnw_JvyAjYv4qpwBrS6Gl8/edit?usp=sharing).

### Role-specific licenses
Certain new team members, especially in go-to-market (GTM) roles, will need paid access to paid tools like Salesforce and LinkedIn Sales Navigator immediately on their first day with the company. Gong licenses that other departments need may [request them from BizOps](https://fleetdm.com/handbook/business-operations#contact-us) and we will make sure there is no license redundancy in that department. The table below can be used to determine which paid licenses they will need, based on their role:

| Role                 | Salesforce CRM | Salesforce "Inbox" | LinkedIn _(paid)_ | Gong _(paid)_ | Zoom _(paid)_ |
|:-----------------|:---|:---|:----|:---|:---|
| 🐋 AE            | ✅ | ✅ | ✅ | ✅ | ✅
| 🐋 CSM           | ✅ | ✅ | ❌ | ✅ | ✅
| 🐋 SC            | ✅ | ✅ | ❌ | ❌ | ✅
| 🫧 SDR           | ✅ | ✅ | ✅ | ❌ | ❌
| ⚗️ PM            | ❌ | ❌ | ❌ | ✅ | ✅
| 🔦 CEO           | ✅ | ✅ | ✅ | ✅ | ✅
|   Other roles    | ❌ | ❌ | ❌ | ❌ | ❌

> **Warning:** Do NOT buy LinkedIn Recruiter. AEs and SDRs should use their personal Brex card to purchase the monthly [Core Sales Navigator](https://business.linkedin.com/sales-solutions/compare-plans) plan. Fleet does not use a company wide Sales Navigator account. The goal of Sales Navigator is to access to profile views and data, not InMail.  Fleet does not send InMail. 

### Namecheap
Domain name registrations are handled through Namecheap. Access is managed via 1Password. <!-- TODO: Move into "tools we use" sheet -->

### Vetty
Fleet team members with access to Fleet's infrastructure undergo a background check provided through [Vetty](https://vetty.co/). Only the most recent background checks appear on the home page of Vetty's dashboard. To access a complete list of background checks run in Vetty, scroll down to the bottom of the candidates page and click "View Historical". <!-- TODO: Move into "tools we use" sheet -->

## Recurring expenses
Recurring monthly or annual expenses are tracked as recurring, non-personnel expenses in ["🧮 The Numbers"](https://docs.google.com/spreadsheets/d/1X-brkmUK7_Rgp7aq42drNcUg8ZipzEiS153uKZSabWc/edit#gid=2112277278) _(¶confidential Google Sheet)_, along with their payment source. Reconciliation of recurring expenses happens monthly.

> Use this spreadsheet as the source of truth.  Always make changes to it first before adding or removing a recurring expense. Only track significant expenses. (Other things besides amount can make a payment significant; like it being an individualized expense, for example.)

## Payroll
Many of these processes are automated, but it's vital to check Gusto and Plane manually for accuracy.
 - Salaried fleeties are automated in Gusto and Plane.
 - Hourly fleeties and consultants are a manual process each month in Gusto and Plane.

| Payroll type                 | What to use                  | DRI                          |
|:-----------------------------|:-----------------------------|:-----------------------------|
| [Commissions and ramp](https://fleetdm.com/handbook/business-operations#run-us-commission-payroll)         | "Off-cycle" payroll          | Head of Revenue Operations
| Sign-on bonus                | "Bonus" payroll              | Head of Business Operations
| Performance bonus            | "Bonus" payroll              | Head of Business Operations     
| Accelerations (quarterly)    | "Off-cycle" payroll          | Head of Revenue Operations
| [US contractor payroll](https://fleetdm.com/handbook/business-operations#run-us-contractor-payroll) | "Off-cycle" payroll | Head of Business Operations


## Responsibilities

### Add a seat to Salesforce
Here are the steps we take to grant appropriate Salesforce licenses to a new hire:
1. Go to ["My Account"](https://fleetdm.lightning.force.com/lightning/n/standard-OnlineSalesHome).
2. View contracts -> pick current contract.
3. Add the desired number of licenses.
4. Sign DocuSign sent to the email.
5. The order will be processed in ~30m.
6. Once the basic license has been added, you can create a new user using the new team member's `@fleetdm.com` email and assign a license to it.
7. To also assign a user an "Inbox license", go to the ["Setup" page](https://fleetdm.lightning.force.com/lightning/setup/SetupOneHome/home) and select "User > Permission sets". Find the [inbox permission set](https://fleetdm.lightning.force.com/lightning/setup/PermSets/page?address=%2F005%3Fid%3D0PS4x000002uUn2%26isUserEntityOverride%3D1%26SetupNode%3DPermSets%26sfdcIFrameOrigin%3Dhttps%253A%252F%252Ffleetdm.lightning.force.com%26clc%3D1) and assign it to the new team member.

### Process an email from a state agency
From time to time, you may get notices via email (or in the mail) from state agencies regarding Fleet's withholding and/or unemployment tax accounts. You can resolve some of these notices on your own by verifying and/or updating the settings in your Gusto account.

If the notice is regarding an upcoming change to your deposit schedule or unemployment tax rate, make the required change in Gusto, such as:
- Update your unemployment tax rate.
- Update your federal deposit schedule.
- Update your state deposit schedule.

In Gusto, you can click **How to review your notice** to help you understand what kind of notice you received and what additional action you can take to help speed up the time it takes to resolve the issue.

> **Note:** Many agencies do not send notices to Gusto directly, so it’s important that you read and take action before any listed deadlines or effective dates of requested changes, in case you have to do something.  If you can't resolve the notice on your own, are unsure what the notice is in reference to, or the tax notice has a missing payment or balance owed, follow the steps in the Report and upload a tax notice in Gusto.

<!-- 2023-08-19: I commented this out because I wasn't sure if we were still doing it this way and I didn't want to leave infomration that was potentially incorrect or contradictory.  -mikermcneil
#### Process an email from a state agency with help from CorpNet
In CorpNet, select "place an order for an existing business":
- Email the CorpNet account rep "Subject: Fleet Device Management: State - Foreign Registration and Payroll Tax Registration" (this takes about two weeks).
- Select "Foreign Qualification," place the order and email the confirmation to the CorpNet rep for "Payroll registration" (this is a shorter turnaround).
  - You can also do this on your own by visiting the state's "Secretary of State" website and checking that the company name is available. To register online, you'll need the EIN, business address, information about the owners and their percentages, the first date of business, sales within the state, and the business type (usually get an email right away for approval ~24-48 hrs). 
-->

<!-- 2023-08-19: This linked sheet is out of date so I'm commenting it out.  -mikermcneil
For more information about how Fleet and our accounting team work together, check out [Fleet - who does what](https://docs.google.com/spreadsheets/d/1FFOudmHmfVFIk-hdIWoPFsvMPmsjnRB8/edit#gid=829046836) (private doc). -->

<!-- 2023-08-19: I commented this out because I wasn't sure how to map it to an imperative mood verb phrase (responsibility) that bizops is doing either ad hoc or as part of a ritual (recurring task). -mikermcneil
Every quarter, payroll and tax filings are due for each state. Gusto can handle these automatically if Third-party authorization (TPA) is enabled. Each state is unique and Gusto has a library of [State registration and resources](https://support.gusto.com/hub/Employers-and-admins/Taxes-forms-and-compliance/State-registration-and-resources) available to review.  You will need to grant Third-party authorization (TPA) per state and this should be checked quarterly before the filing due dates to ensure that Gusto can file on time. -->


### Inform managers about hours worked
Every Friday at 1:00pm CT, we gather hours worked for anyone who gets paid hourly by Fleet. This includes core team members and consultants, regardless of employment classification, and regardless whether inside or outside of the United States.

Here's how:
- For every hourly core team member in Gusto or Pilot.co, look up their manager ([who they report to](https://fleetdm.com/handbook/company#org-chart)).
- If any direct report is hourly in Pilot.co and does not submit their hours until the end of the month, still list them, but explain.  (See example below.)
- [Consultants](https://fleetdm.com/handbook/business-operations#hiring) don't have a formal reporting structure or manager. Instead, send their hours worked to the CEO, no matter who the consultant is.

Then, send **the CEO** and **each manager** a direct message in Slack by copying and pasting the following template:

> Here are the hours worked by your direct reports since last Saturday at midnight (YYYY-MM-DD):
> - 🧑‍🚀 Alice Bobberson: 21.25
> - 🧑‍🚀 Charles David: 3.5
> - 🧑‍🚀 Philippe Timebender: (hours not available until they invoice at the end of the month)
>
> And here are the hours worked by consultants:
> - 💁 Bombalurina: 0
> - 💁 Jennyanydots: 0
> - 💁 Skimbleshanks: 19
> - 💁 Griza Bella: 0
> 
> More info: https://fleetdm.com/handbook/business-operations#inform-managers-about-hours-worked


### Run US contractor payroll
For Fleet's US contractors, running payroll is a manual process:
1. Add the amount to be paid to the "Gross" line.
2. Review hours _("Time tools > Time tracking")_
3. Adjust time frame to match current payroll period (the 27th through 26th of the month)
4. Sync hours and run contractor payroll.

### Run US commission payroll
- Update [commission calculator](https://docs.google.com/spreadsheets/d/1vw6Q7kCC7-FdG5Fgx3ghgUdQiF2qwxk6njgK6z8_O9U/edit) with new revenue from any deals that are closed/won (have a subscription agreement signed by both parties) and have an **effective start date** within the previous month.
  - Find detailed notes on this process in [Notes - Run commission payroll in Gusto](https://docs.google.com/document/d/1FQLpGxvHPW6X801HYYLPs5y8o943mmasQD3m9k_c0so/edit#). 
- Let the Head of Business Operations know they can run the commission payroll. Use the off-cycle payroll option in Gusto. Be sure to classify the payment as "Commission" in the "other earnings" field and not the generic "Bonus."
- Once commission payroll has been run, update the [commission calculator](https://docs.google.com/spreadsheets/d/1vw6Q7kCC7-FdG5Fgx3ghgUdQiF2qwxk6njgK6z8_O9U/edit) to mark the commission as paid. 

### Process monthly accounting
Create a [new montly accounting issue](https://github.com/fleetdm/confidential/issues/new/choose) for the current month and year named "Closing out YYYY-MM" in GitHub and complete all of the tasks in the issue. (This uses the [monthly accounting issue template](https://github.com/fleetdm/confidential/blob/main/.github/ISSUE_TEMPLATE/5-monthly-accounting.md).

- **SLA:** The monthly accounting issue should be completed and closed before the 7th of the month.
- The close date is tracked each month in [KPIs](https://docs.google.com/spreadsheets/d/1Hso0LxqwrRVINCyW_n436bNHmoqhoLhC8bcbvLPOs9A/edit).
- **When is the issue created?** We create and close the monthly accounting issue for the previous month within the first 7 days of the following month.  For example, the monthly accounting issue to close out the month of January is created promptly in February and closed before the end of the day, Feb 7th.  A convenient trick is to create the issue on the first Friday of the month and close it ASAP.

### Check finances for quirks
Every quarter, we check Quickbooks Online (QBO) for discrepancies and follow up on quirks.
- Check to make sure [bookkeeping quirks](https://docs.google.com/spreadsheets/d/1nuUPMZb1z_lrbaQEcgjnxppnYv_GWOTTo4FMqLOlsWg/edit?usp=sharing) are all accounted for and resolved or in progress toward resolution.
- Check balance sheet and profit and loss statements (P&Ls) in QBO against the [monthly workbooks](https://drive.google.com/drive/folders/1ben-xJgL5MlMJhIl2OeQpDjbk-pF6eJM) in Google Drive.

### Report quarterly numbers in Chronograph
Follow these steps to perform quarterly reporting for Fleet's investors:
- Login to Chronograph and upload our profit and loss statement (P&L), balance sheet and cash flow statements for CRV (all in one book saved in [Google Drive](https://drive.google.com/drive/folders/1ben-xJgL5MlMJhIl2OeQpDjbk-pF6eJM).
- Provide updated metrics for the following items using Fleet's [KPI spreadsheet](https://docs.google.com/spreadsheets/d/1Hso0LxqwrRVINCyW_n436bNHmoqhoLhC8bcbvLPOs9A/edit#gid=0).
  - Headcount at end of the previous quarter.
  - Starting ARR for the previous quarter.
  - Total new ARR for the previous quarter.
  - "Upsell ARR" (new ARR from expansions only- Chronograph defines "upsell" as price increases for any reason.
    **- Fleet does not "upsell" anything; we deliver more value and customers enroll more hosts), downgrade ARR and churn ARR (if any) for the previous quarter.**
  - Ending ARR for the previous quarter.
  - Starting number of customers, churned customers, and the number of new customers Fleet gained during the previous quarter.
  - Total amount of Fleet customers at the end of the previous quarter.
  - Gross margin % 
    - How to calculate: (total revenue for the quarter - cost of goods sold for the quarter)/total revenue for the quarter (these metrics can be found in our books from Pilot). Chronograph will automatically conver this number to a %.
  - Net dollar retention rate
    - How to calculate: (starting ARR + new subscriptions and expansions - churn)/starting ARR. 
  - Cash burn
    - How to calculate: (start of quarter runway - end of quarter runway)/3. 


### Grant equity
Equity grants for new hires are queued up as part of the [hiring process](https://fleetdm.com/handbook/business-operations#hiring), then grants and consents are [batched and processed quarterly](https://github.com/fleetdm/confidential/issues/new/choose).

Doing an equity grant involves:
1. executing a board consent
2. the recipient and CEO signing paperwork about the stock options
3. updating the number of shares for the recipient in the equity plan
4. updating Carta to reflect the grant

For the status of stock option grants, exercises, and all other _common stock_ including advisor, founder, and team member equity ownership, see [Fleet's equity plan](https://docs.google.com/spreadsheets/d/1_GJlqnWWIQBiZFOoyl9YbTr72bg5qdSSp4O3kuKm1Jc/edit#gid=0).  For information about investor ownership, see [Carta](https://app.carta.com/corporations/1234715/summary/).

> Fleet's [equity plan](https://docs.google.com/spreadsheets/d/1_GJlqnWWIQBiZFOoyl9YbTr72bg5qdSSp4O3kuKm1Jc/edit#gid=0) is the source of truth, not Carta.  Neither are pro formas sent in an email attachment, even if they come from lawyers.
> 
> Anyone can make mistakes, and none of us are perfect.  Even when we triple check.  Small mistakes in share counts can be hard to attribute, and can cause headaches and eat up nights of our CEO's and operations team's time.  If you notice what might be a discrepancy between the equity plan and any other secondary source of information, please speak up and let Fleet's CEO know ASAP.  Even if you're wrong, your note will be appreciated.


### Deliver annual report for venture line
Within 60 days of the end of the year, follow these steps:
  - Provide Silicon Valley Bank (SVB) with our balance sheet and profit and loss statement (P&L, sometimes called a cashflow statement) for the past twelve months.  
  - Provide SVB with our annual operating budgets and projections (on a quarterly basis) for the coming year.
  - Deliver this as early as possible in case they have questions.



## Rituals
The following table lists this department's rituals, frequency, and Directly Responsible Individual (DRI).

> TODO: Verify each of these points to the correct link above.  Then extrapolate into rituals.yml.  (See CEO handbook page for an example.)

| Ritual                       | Frequency                | Description                                         | DRI               |
|:-----------------------------|:-----------------------------|:----------------------------------------------------|-------------------|
| Key review | every three weeks | Every release cycle, a key review deck is prepared and presented. | Head of Business Operations |
| Informing managers about hours worked | Weekly |  See [Informing managers about hours worked"](https://fleetdm.com/handbook/business-operations#informing-managers-about-hours-worked). | Head of Business Operations |
| Payroll | Monthly before payroll runs | Every month, Mike McNeil audits the payroll platforms for accuracy. | Head of Business Operations |
| US contractor payroll | Monthly | Sync contractor hours to payments in Gusto and run payroll for the month. | Head of Business Operations |
| Commission payroll | Monthly | Use the [commission calculator](https://docs.google.com/spreadsheets/d/1vw6Q7kCC7-FdG5Fgx3ghgUdQiF2qwxk6njgK6z8_O9U/edit#gid=0) to determine the commission payroll to be run in Gusto. | Taylor Hughes |
| Revenue report | Weekly | At the start of every week, check the Salesforce reports for past due invoices, non-invoiced opportunities, and past due renewals.  Report any findings to in the `#g-sales` channel by mentioning Alex Mitchell and Mike McNeil. | Taylor Hughes |
| Monthly accounting | Monthly | Create [the monthly close GitHub issue](https://fleetdm.com/handbook/business-operations#intake) and walk through the steps. | Nathanael Holliday |
| Quarterly grants | Quarterly | Create [the quarterly close GitHub issue](https://fleetdm.com/handbook/business-operations#intake) and walk through the steps. | Nathanael Holliday |
| AP invoice monitoring | Weekly | Look for new accounts payable invoices and make sure that Fleet's suppliers are paid. | Nathanael Holliday | 
| Tax preparation | Annually on the first week of March | Provide information to tax team with Deloitte and assist with filing and paying state and federal returns | Nathanael Holliday | 
| Vanta check | Monthly | Look for any new actions in Vanta due in the upcoming months and create issues to ensure they're done on time. | Nathan Holliday |
| Investor reporting | Quarterly | Provide updated metrics for CRV in Chronograph. | Nathanael Holliday |
| Applicant forwarding | Daily | Whenever an application notification arrives in the BizOps slack channel, forward this notification to the hiring channel for that position. | Joanne Stableford |
| KPI roundup + weekly update | Weekly | Update KPI spreadsheet with BizOps KPI data by 5pm US central time every Friday.  At 5pm check other department KPIs to make sure they have been updated, and if not, notify DRIs and the apprentice to the CEO which KPIs have not been updated. | Nathanael Holliday |


<!--
Note: These are out of date, but retained for future reference.  TODO: Deal with them and delete them

| Weekly update reminder | Weekly | Early Friday mornings (US time), a Slack bot posts in the `#g-e` channel reminding directly responsible individuals for KPIs to add their metrics for the current week in ["KPIs"](https://docs.google.com/spreadsheets/d/1Hso0LxqwrRVINCyW_n436bNHmoqhoLhC8bcbvLPOs9A/edit) before the end of the day. | N/A |
| Access revalidation | Quarterly | Review critical access groups to make sure they contain only relevant people. | Mike McNeil |
| 550C update | Annually | File California 550C. | Mike McNeil |
| TPA verifications | Quarterly | Every quarter before tax filing due dates, Mike McNeil audits state accounts to ensure TPA is set up or renewed. | Mike McNeil |
| Brex reconciliation | Monthly | Make sure all company-issued credit card transactions include memos. | Nathanael Holliday |
| Hours update | Weekly | Screenshots of contractor hours as shown in Gusto are sent via Slack to each contractor's manager with no further action necessary if everything appears normal. | Mike McNeil |
| QBO check | Quarterly | The first month after the previous quarter has closed, make sure that QBO is accurate compared to Fleet's records. | Nathanael Holliday | 
| Capital credit reporting | Annually | Within 60 days of the new year, provide financial statements to SVB. | Nathanael Holliday |
| YubiKey adoption | Monthly | Track YubiKey adoption in Google workspace and follow up with those that aren't using it. | Mike McNeil |
| Security policy update | Annually | Update security policies and have them approved by the CEO. | Nathanael Holliday |
| Security notifications check | Daily | Check Slack, Google, Vanta, and Fleet dogfood for security-related notifications. | Nathanael Holliday |
| Changeset for onboarding issue template | Quarterly | pull up the changeset in the onboarding issue template and send out a link to the diff to all team members by posting in Slack's `#general` channel. | Mike McNeil |
| Recruiting progress checkup | Weekly | Mike McNeil looks in the [Fleeties spreadsheet](https://docs.google.com/spreadsheets/d/1OSLn-ZCbGSjPusHPiR5dwQhheH1K8-xqyZdsOe9y7qc/edit#gid=0) and reports on each open position. | Mike McNeil |
| Investor and advisor updates | PRN | Mike McNeil tracks the last contact with investors and coordinates outreach with CEO. | Mike McNeil |
| MDM device enrollment | Quarterly | Provide export of MDM enrolled devices to the ops team. | Luke Heath |

-->





#### Stubs
The following stubs are included only to make links backward compatible.

##### Open positions
Please see 📖[handbook/company#open-positions](https://fleetdm.com/handbook/company#open-positions) for a list of open job postings at Fleet.

##### Weekly updates
Please see 📖[handbook/ceo#weekly-updates](https://fleetdm.com/handbook/ceo#send-the-weekly-update)

##### Directly responsible individuals
Please see 📖[handbook/company/why-this-way#why-direct-responsibility](https://fleetdm.com/handbook/company/why-this-way#why-direct-responsibility) to learn more about DRIs.

##### Security
Please see [📖handbook/company/communications#security](https://fleetdm.com/handbook/company/communications#security).

##### Vendor questionnaires
Please see [📖handbook/company/communications#vendor-questionnaires](https://fleetdm.com/handbook/company/communications#vendor-questionnaires)

##### Getting a contract signed
Please see [📖handbook/company/communications#getting-a-contract-signed](https://fleetdm.com/handbook/company/communications#getting-a-contract-signed)

##### Getting a contract reviewed
Please see [📖handbook/company/communications#getting-a-contract-reviewed](https://fleetdm.com/handbook/company/communications#getting-a-contract-reviewed)

#### Zapier and DocuSign
Please see [📖handbook/ceo#zapier-and-docusign](https://fleetdm.com/handbook/ceo#zapier-and-docusign)

#### Gong
Please see [📖handbook/company/communications#gong](https://fleetdm.com/handbook/company/communications#gong).

#### Troubleshooting Gong
Please see [📖handbook/company/communications#troubleshooting-gong](https://fleetdm.com/handbook/company/communications#troubleshooting-gong).

##### Excluding calls from being recorded
Please see [📖handbook/company/communications#excluding-calls-from-being-recorted](https://fleetdm.com/handbook/company/communications#excluding-calls-from-being-recorted).

##### Salesforce
Please see [📖handbook/business-operations#add-a-seat-to-salesforce](https://fleetdm.com/handbook/business-operations#add-a-seat-to-salesforce).

##### Intake
##### Kanban
##### Legal Operations
##### IT Operations
##### Finance Operations
##### Taxes and compliance
##### State quarterly payroll and tax filings
##### CorpNet state registration process
##### Annual reporting for capital credit line
##### Equity grants
##### Finance rituals
##### Monthly rituals
##### Quarterly rituals
##### Informing managers about hours worked
##### Monthly accounting
Please see above on this page for the latest content for these topics within the new framework for departmental handbook pages.

##### Zoom
##### Slack
##### Email relays
##### Meetings
##### Scheduling a meeting
##### Internal meeting scheduling
##### Modifying an event organized by someone else
##### External meeting scheduling
##### Performance feedback
##### Levels of confidentiality
##### Relocating
##### Celebrations
##### Workiversaries
##### Spending company money
##### Brex
##### Travel
##### Attending conferences or company travel
##### Non-travel purchases that exceed a Brex cardholder's limit
##### Reimbursements
##### Individualized expenses
##### Benefits
##### Paid time off
##### Taking time off 
##### Holidays
##### New parent leave
##### Retirement contributions
##### US based team members
##### Non-US team members
##### Coworking
##### Team member onboarding
##### Before the start date
##### Recommendations for new teammates
##### Training expectations
##### Sightseeing tour
##### Contributor experience training
##### Onboarding retrospective
##### Compensation
##### Compensation changes
##### Reprovisioning equipment
##### Equipment retention and replacement
##### Equipment
##### Laptops
##### Purchasing a company-issued device
##### Selecting a laptop
##### Buying other new equipment
##### Tracking equipment
##### Returning equipment
##### Github
##### GitHub labels
Please see 📖[handbook/company/communications](https://fleetdm.com/handbook/company/communications) for all sections above.

##### Hiring
##### Consultants
##### Hiring a consultant
##### Who ISN'T a consultant?
##### Sending a consulting agreement
##### Updating a consultant's fee
##### Advisor
##### Adding an advisor
##### Finalizing a new advisor
##### Core team member
##### Creating a new position
##### Approving a new position
##### Recruiting
##### Receiving job applications
##### Candidate correspondence email templates
##### Hiring restrictions
##### Incompatible former employers
##### Incompatible locations
##### Interviewing
##### Hiring a new team member
##### Making an offer
##### Steps after an offer is accepted
##### Key reviews
##### Tracking hours
##### Informing managers about hours worked
##### Departures
##### Communicating departures
Please see 📖[handbook/company/leadership](https://fleetdm.com/handbook/company/leadership) for all sections above.



<meta name="maintainedBy" value="jostableford">
<meta name="title" value="🔦 Business Operations">

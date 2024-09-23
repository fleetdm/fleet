# Finance
This handbook page details processes specific to working [with](#contact-us) and [within](#responsibilities) this department.

## Team
| Role                          | Contributor(s)           |
|:------------------------------|:-----------------------------------------------------------------------------------------------------------|
| Head of Finance   | [Joanne Stableford](https://www.linkedin.com/in/joanne-stableford/) _([@jostableford](https://github.com/JoStableford))_
| Finance Engineer | [Isabell Reedy](https://www.linkedin.com/in/isabell-reedy-202aa3123/) _([@ireedy](https://github.com/ireedy))_


## Contact us
- To **make a request** of this department, [create an issue](https://github.com/fleetdm/confidential/issues/new?assignees=&labels=%23g-finance&projects=&template=custom-request.md) and a team member will get back to you within one business day (If urgent, mention a [team member](#team) in [#g-finance](https://fleetdm.slack.com/archives/C047N5L6EGH).
  - Please **use issue comments and GitHub mentions** to communicate follow-ups or answer questions related to your request.
  - Any Fleet team member can [view the kanban board](https://app.zenhub.com/workspaces/-g-finance-63f3dc3cc931f6247fcf55a9/board?sprints=none) for this department, including pending tasks and the status of new requests.


## Responsibilities
The Finance department is directly responsible for accounts receivable including invoicing, accounts payable including commision calculations, exspense reporting including Brex memos and maintaining accurate spend projections in "🧮The numbers", sales taxes, payroll taxes, corporate income/franchise taxes, and financial operations including bank accounts and cash flow management.


### Run payroll
Many of these processes are automated, but it's vital to check Gusto and Plane manually for accuracy.
 - Salaried fleeties are automated in Gusto and Plane.
 - Hourly fleeties and consultants are a manual process each month in Gusto and Plane.

| Payroll type                 | What to use                  | DRI                          |
|:-----------------------------|:-----------------------------|:-----------------------------|
| [Commissions and ramp](https://fleetdm.com/handbook/finance#run-us-commission-payroll)         | "Off-cycle - Commission" payroll          | Head of Finance
| Sign-on bonus                | "Bonus" payroll              | Head of Finance
| Performance bonus            | "Bonus" payroll              | Head of Finance     
| Accelerations (quarterly)    | "Off-cycle - Commission" payroll          | Head of Finance
| [US contractor payroll](https://fleetdm.com/handbook/finance#run-us-contractor-payroll) | "Off-cycle" payroll | Head of Finance

### Reconcile monthly recurring expenses
Recurring monthly or annual expenses, such as the tools we use throughout Fleet, are tracked as recurring, non-personnel expenses in ["🧮 The Numbers"](https://docs.google.com/spreadsheets/d/1X-brkmUK7_Rgp7aq42drNcUg8ZipzEiS153uKZSabWc/edit#gid=2112277278) _(¶confidential Google Sheet)_, along with their payment source. Reconciliation of recurring expenses happens monthly. <!-- TODO: Merge "🧮 The Numbers" and  ["Tools we use" (private Google doc)](https://docs.google.com/spreadsheets/d/170qjzvyGjmbFhwS4Mucotxnw_JvyAjYv4qpwBrS6Gl8/edit?usp=sharing) -->

> Use this spreadsheet as the source of truth.  Always make changes to it first before adding or removing a recurring expense. Only track significant expenses. (Other things besides amount can make a payment significant; like it being an individualized expense, for example.)


### Register Fleet as an employer with a new state 
Fleet must register as an employer in any state where we hire new teammates. To do this, complete the following steps in Gusto:
1. After a new teammate completes their Gusto profile, the Finance department will be prompted to approve it for payroll. Sign in to your Gusto admin account and begin the approval process.
2. Select "yes" when prompted to file a new hire report and complete the approval process.
3. Once the profile is approved, navigate to Tax setup and select the state you’d like to register Fleet in.
4. Select “Have us register for you” and then “Start registration.”
5. Verify, add, and amend any company information to ensure accuracy.
6. Select “Send registration” and authorize payment for the specified amount. CorpNet will then send an email with next steps, which vary by state.
7. Update the [list of states that Fleet is currently registered with as an employer](https://fleetdm.com/handbook/finance#review-state-employment-tax-filings-for-the-previous-quarter).
    

### Process an email from a state agency
From time to time, you may get notices via email (or in the mail) from state agencies regarding Fleet's withholding and/or unemployment tax accounts. You can resolve some of these notices on your own by verifying and/or updating the settings in your Gusto account.

If the notice is regarding an upcoming change to your deposit schedule or unemployment tax rate, make the required change in Gusto, such as:
- Update your unemployment tax rate.
- Update your federal deposit schedule.
- Update your state deposit schedule.

In Gusto, you can click **How to review your notice** to help you understand what kind of notice you received and what additional action you can take to help speed up the time it takes to resolve the issue.

> **Note:** Many agencies do not send notices to Gusto directly, so it’s important that you read and take action before any listed deadlines or effective dates of requested changes, in case you have to do something.  If you can't resolve the notice on your own, are unsure what the notice is in reference to, or the tax notice has a missing payment or balance owed, follow the steps in the Report and upload a tax notice in Gusto.

Every quarter, payroll and tax filings are due for each state. Gusto can handle these automatically if Third-party authorization (TPA) is enabled. Each state is unique and Gusto has a library of [State registration and resources](https://support.gusto.com/hub/Employers-and-admins/Taxes-forms-and-compliance/State-registration-and-resources) available to review.  You will need to grant Third-party authorization (TPA) per state and this should be checked quarterly before the filing due dates to ensure that Gusto can file on time. -->


### Review state employment tax filings for the previous quarter

Every quarter, payroll and tax filings are due for each state. Gusto automates this process, however there are often delays or quirks between Gusto's submission and the state receiving the filings.
To mitigate the risk of penalties and to ensure filings occur as expected, follow these steps in the first month of the new quarter, verifying past quarter submission:
1. Create an issue to "Review state filings for the previous quarter".
2. Copy this text block into the issue to track progress by state:


```
States checked:
- [ ] California
- [ ] Colorado
- [ ] Connecticut
- [ ] Florida
- [ ] Georgia
- [ ] Hawaii
- [ ] Illinois
- [ ] Kansas
- [ ] Maryland
- [ ] Massachusetts
- [ ] New York
- [ ] Ohio
- [ ] Oregon
- [ ] Pennsylvania
- [ ] Rhode Island
- [ ] Tennessee
- [ ] Texas
- [ ] Utah
- [ ] Virginia
- [ ] Washington
- [ ] Washington, DC
- [ ] West Virginia
- [ ] Wisconsin
```
 

3. Login to Gusto and navigate to "Taxes and compliance", then "Tax documents".
4. Login to each State portal (using the details saved in 1Password) and verify that the portal has received the automated submission from Gusto.
5. Check off states that are correct, and use comments to explain any quirks or remediation that's needed.


### Run US contractor payroll
For Fleet's US contractors, running payroll is a manual process:
1. Add the amount to be paid to the "Gross" line.
2. Review hours _("Time tools > Time tracking")_
3. Adjust time frame to match current payroll period (the 27th through 26th of the month)
4. Sync hours and run contractor payroll.

### Create an invoice
To create a new invoice for a Fleet customer, follow these steps:
1. Go to the [invoice folder in google drive](https://drive.google.com/drive/folders/11limC_KQYNYQPApPoXN0CplHo_5Qgi2b?usp=drive_link).
2. Create a copy of the invoice template, and title the copy `[invoice number] Fleet invoice - [customer name]`.
    - The invoice number follows the format of `YYMMDD[daily issued invoice number]`, where the daily issued invoice number should equal `01` if it's the first invoice issued that day, `02` if it's the second, etc.
3.  Edit the new invoice to reflect details from the signed subscription agreement (and PO if required).
    - Enter the invoice number (and PO number if required) into the top right section of the invoice.
    - Update the date of the invoice to reflect the current date.
    - Make sure the payment terms match the signed subscription agreement.
    - Copy the customer address from the signed subscription agreement and input it in the "Bill to" section of the invoice.
    - Copy the "Billing contact" email from the signed subscription agreement and add it to the last line of the "Bill to" address.
    - Make sure the start and end dates of the contract and amount match the subscription agreement.
    - If professional services are included in the subscription agreement, include as a separate line in the invoice, and ensure the amounts total correctly.
    - Ensure the "Notes" section has wiring instructions for payment via SVB.
4.  Download the completed invoice as a PDF.
5.  Send the PDF to the billing contact from the "Bill to" section of the invoice and cc [Fleet's billing email address](https://fleetdm.com/handbook/company/communications#email-relays). Use the following template for the email:

```
Subject: Invoice for Fleet Device Management [invoice number]
Hello,

I've attached the invoice for [customer name]'s purchase of Fleet Device Management's premium subscription.
For payment instructions please refer to your invoice, and reach out to [insert Fleet's billing address] with any questions.

Thanks,
[name]
```

6. Update the opportunity and the opportunity billing cycle in Salesforce to include the "Invoice date" as the day the invoice was sent.
8. Notify the AE/CSM that the invoice has been sent.

> Certain vendors require invoices submitted via a payment portal (such as Coupa). Once you've generated the invoice using the steps above, upload it to the relevant payment portal and email the billing contact to let them know you've submitted the invoice.


### Communicate the status of customer financial actions
This reporting is performed to update the status of open or upcoming customer actions regarding the financial health of the opportunity. To complete the report:
1. Check [SVB](https://connect.svb.com/#/) and [Brex](https://accounts.brex.com/login) for any recently received payments from customers and record them in SFDC.
2. Go to this [report folder](https://fleetdm.lightning.force.com/lightning/r/Folder/00lUG000000DstpYAC/view?queryScope=userFolders) in SFDC. The three reports will provide the data used in the report.
3. Copy the template below and paste it into the [#g-sales slack channel](https://fleetdm.slack.com/archives/C030A767HQV) and complete all "todos" using the data from Salesforce before sending.

```
Weekly revenue report - [@`todo: CRO` and @`todo: CEO`]
- Number accounts with outstanding balances = `todo`
- Number of customers awaiting invoices = `todo`
- Number of past-due renewals = `todo`
```

4. Send payment reminders via email to all outstanding accounts by responding to the invoice email initially sent to the customer.

```
Hello,
This is a reminder that you have an outstanding balance due for your Fleet Device Management premium subscription.
We have included the invoice here for your convenience.
For payment instructions please refer to your invoice, and reach out to [Fleet's billing contact] with any questions.

Thanks,
[name]
```

5. If any accounts will become overdue within a week, reply in thread to the slack post, mention the opportunity owner of the account, and ask them to notify their contact that Fleet is still awaiting payment.
6. Review the [billing cycles](https://fleetdm.lightning.force.com/lightning/r/Report/00OUG000000yGjR2AU/view) report in SFDC for customers on multiyear deals. For any customers due for invoicing within the next week, create an issue on the Finance board.


### Run US commission payroll
1. Update individual teammates commission calculators (linked from [main commission calculator](https://docs.google.com/spreadsheets/d/1PuqUbfPGos87TfcHWgUd05TRJgQLlBmhyz1euj79m2A/edit?usp=sharing)) with new revenue from any deals that are closed-won (have a subscription agreement signed by both parties) and have a **close date** within the previous month.
    - Verify closed-won deal numbers with CRO to ensure any agreed upon exceptions are captured (eg: CRO approves an AE to receive commission on a renewal deal due to cross-sell).
2. In the "Monthly commission payroll party" meeting, present the commission calculations for Fleeties receiving commission for approval.
    - If there are any quarterly accelerators due for the teammate receiving commission, ensure the individual total includes both the monthly and the quarterly amount.
3. After the amounts are approved in the meeting, process the commission payroll.
    - Use the off-cycle payroll option in Gusto. Be sure to classify the payment as "Commission" in the "other earnings" field and not the generic "Bonus."
4. Once commission payroll has been run, update the [main commission calculator](https://docs.google.com/spreadsheets/d/1PuqUbfPGos87TfcHWgUd05TRJgQLlBmhyz1euj79m2A/edit?usp=sharing) to mark the commission as paid.

### Run international commission payroll
1. Follow the steps in [run US commission payroll](https://fleetdm.com/handbook/finance#run-us-commission-payroll) to have the commission amounts approved by the CRO.
2. After the amounts are approved in the "Monthly commission payroll party", navigate to Help > Ask a question in Plane to request a commission payment for the teammate.
3. Send a message using the following template

    ```
    Hello,
    I’d like to run an off-cycle commission payment for [teammate’s full name] for the period of [commission period].
    The amount of [USD amount] should be paid with their next payroll.
    Please let me know if you need any additional information to process this request.
    
    Thanks,
    [name]
    ```

4. Once Plane confirms the payroll change has been actioned, update the [main commission calculator](https://docs.google.com/spreadsheets/d/1PuqUbfPGos87TfcHWgUd05TRJgQLlBmhyz1euj79m2A/edit#gid=928324236) to mark the commission as paid. 


### Run quarterly or annual employee bonus payroll
1. Update individual teammate bonus calculator (linked from [main commission calculator](https://docs.google.com/spreadsheets/d/1PuqUbfPGos87TfcHWgUd05TRJgQLlBmhyz1euj79m2A/edit?usp=sharing)) with relevant metrics.
    - Bonus plans will have details specified on how to measure success, with most drawing from the [KPI spreadsheet](https://docs.google.com/spreadsheets/d/1Hso0LxqwrRVINCyW_n436bNHmoqhoLhC8bcbvLPOs9A/edit?usp=sharing) or from linked SFDC reports. If unsure where to pull achievement metrics from, contact teammate's manager to clarify.
2. In the "Monthly commission payroll party" meeting, present the bonus calculations for Fleeties receiving bonus for approval.
3. After the amounts are approved in the meeting, process the bonus payroll.
    - Use the off-cycle payroll option in Gusto and be sure to classify the payment as "Bonus".
    - For international teammates, you may need to use the "Help" function, or email support to notify Plane of the amount needing to be paid.
4. Once bonus payroll has been run, update the [main commission calculator](https://docs.google.com/spreadsheets/d/1PuqUbfPGos87TfcHWgUd05TRJgQLlBmhyz1euj79m2A/edit?usp=sharing) to mark the bonus as paid. 
 

### Process monthly accounting
Create a [new montly accounting issue](https://github.com/fleetdm/confidential/issues/new/choose) for the current month and year named "Closing out YYYY-MM" in GitHub and complete all of the tasks in the issue. (This uses the [monthly accounting issue template](https://github.com/fleetdm/confidential/blob/main/.github/ISSUE_TEMPLATE/5-monthly-accounting.md).

- **SLA:** The monthly accounting issue should be completed and closed before the 7th of the month.
- The close date is tracked each month in [KPIs](https://docs.google.com/spreadsheets/d/1Hso0LxqwrRVINCyW_n436bNHmoqhoLhC8bcbvLPOs9A/edit).
- **When is the issue created?** We create and close the monthly accounting issue for the previous month within the first 7 days of the following month.  For example, the monthly accounting issue to close out the month of January is created promptly in February and closed before the end of the day, Feb 7th.  A convenient trick is to create the issue on the first Friday of the month and close it ASAP.


### Respond to low credit alert
Fleet admins will receive an email alert when the usage of company cards for the month is aproaching the company credit limit. To avoid the limit being exceeded, a Brex admin will follow these steps:
1. Sign in to Fleet's Brex account.
2. On the landing page, use the "Move money" button to "Add funds to your Brex business accounts".
3. Select "Transfer from a connected account" and select the primary business account.
4. Choose the "One time" transfer option and process the transfer.

No further action needs to be taken, the amount available for use will increase without disruption to regular processes.

### Check franchise tax status
No later than the second month of every quarter, we check [Delaware divison of corporations](https://icis.corp.delaware.gov) to ensure that Fleet has paid the quarterly franchise tax amounts to remain in good standing with the state of Delaware.
- Go to the [DCIS - eCorp website](https://icis.corp.delaware.gov/ecorp/logintax.aspx?FilingType=FranchiseTax) and use the details in 1Password to look up Fleet's status.
- If no outstanding amounts: the tax has been paid.
- If outstanding amounts shown: ensure payment before due date to avoid penalties, interest, and entering bad standing.


### Check finances for quirks
Every quarter, we check Quickbooks Online (QBO) for discrepancies and follow up on quirks.
1. Check to make sure [bookkeeping quirks](https://docs.google.com/spreadsheets/d/1nuUPMZb1z_lrbaQEcgjnxppnYv_GWOTTo4FMqLOlsWg/edit?usp=sharing) are all accounted for and resolved or in progress toward resolution.
2. Check balance sheet and profit and loss statements (P&Ls) in QBO against the latest [monthly workbooks](https://drive.google.com/drive/folders/1ben-xJgL5MlMJhIl2OeQpDjbk-pF6eJM) in Google Drive. Ensure reports are in the "accural" accounting method.
3. Reach out to Pilot with any differences or quirks, and ask them to resolve/provide clarity.  This often will need to happen over a call to review sycnhronously.
4. Once quirks are resolved, note the day it was resolved in the spreadsheet.


### Report quarterly numbers in Chronograph
Follow these steps to perform quarterly reporting for Fleet's investors:
1. Login to Chronograph and upload our profit and loss statement (P&L), balance sheet and cash flow statements for CRV (all in one book saved in [Google Drive](https://drive.google.com/drive/folders/1ben-xJgL5MlMJhIl2OeQpDjbk-pF6eJM).
2. Provide updated metrics for the following items using Fleet's [KPI spreadsheet](https://docs.google.com/spreadsheets/d/1Hso0LxqwrRVINCyW_n436bNHmoqhoLhC8bcbvLPOs9A/edit#gid=0).
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
      - How to calculate: start of quarter runway - end of quarter runway. 


### Deliver annual report for venture line
Within 60 days of the end of the year, follow these steps:
1. Provide Silicon Valley Bank (SVB) with our balance sheet and profit and loss statement (P&L, sometimes called a cashflow statement) for the past twelve months.  
2. Provide SVB with our board-approved annual operating budgets and projections (on a quarterly granularity) for the new year.
3. Deliver this as early as possible in case they have questions.


### Process a new vendor invoice
Fleet pays its vendors in less than 15 business days in most cases. All invoices and tax documents should be submitted to the Finance department using the [appropriate Fleet email address (confidential Google Doc)](https://docs.google.com/document/d/1tE-NpNfw1icmU2MjYuBRib0VWBPVAdmq4NiCrpuI0F0/edit#heading=h.wqalwz1je6rq).
- After making sure the invoice received from a new vendor is valid, add the new vendor to the recurring expenses section of ["The numbers"](https://docs.google.com/spreadsheets/d/1X-brkmUK7_Rgp7aq42drNcUg8ZipzEiS153uKZSabWc/edit#gid=2112277278) before paying the invoice.
- If we have not paid this vendor before, make sure we have received the required W-9 or W-8 form from the vendor. **Accounting cannot process a payment without these tax forms for compliance reasons.**
  - **US-based vendors** are required to complete a [W-9 form](https://www.irs.gov/pub/irs-pdf/fw9.pdf).
  - **Non-US based vendors and individuals** are required to follow these [instructions](https://www.irs.gov/instructions/iw8bene) and provide a completed [W-8BEN-E](https://www.irs.gov/pub/irs-pdf/fw8bene.pdf) form.


### Process a request to cancel a vendor
- Make the cancellation notification in accordance with the contract terms between Fleet and the vendor, typically these notifications are made via email and may have a specific address that notice must be sent to. If the vendor has an autorenew contract with Fleet there will often be a window of time in which Fleet can cancel, if notification is made after this time period Fleet may be obligated to pay for the subsequent year even if we don't use the vendor during the next contract term.  
- Once cancelled, update the recurring expenses section of [The Numbers](https://docs.google.com/spreadsheets/d/1X-brkmUK7_Rgp7aq42drNcUg8ZipzEiS153uKZSabWc/edit#gid=2112277278) to reflect the cancellation by changing the projected monthly burn in column G to $0 and adding "CANCELLED" in front of the vendor's name in column C.


### Update weekly KPIs
- Create the weekly update issue from the template in ZenHub every Friday and update the [KPIs for finance](https://docs.google.com/spreadsheets/d/1Hso0LxqwrRVINCyW_n436bNHmoqhoLhC8bcbvLPOs9A/edit#gid=0) by 5pm US central time.


## Rituals

The following table lists this department's rituals, frequency, and Directly Responsible Individual (DRI).

<rituals :rituals="rituals['handbook/finance/finance.rituals.yml']"></rituals>

<!--
Note: These are out of date, but retained for future reference.  TODO: Deal with them and delete them

| Access revalidation | Quarterly | Review critical access groups to make sure they contain only relevant people. | Mike McNeil |
| 550C update | Annually | File California 550C. | Mike McNeil |
| TPA verifications | Quarterly | Every quarter before tax filing due dates, Mike McNeil audits state accounts to ensure TPA is set up or renewed. | Mike McNeil |
| YubiKey adoption | Monthly | Track YubiKey adoption in Google workspace and follow up with those that aren't using it. | Mike McNeil |
| Security policy update | Annually | Update security policies and have them approved by the CEO. | Nathanael Holliday |
| Security notifications check | Daily | Check Slack, Google, Vanta, and Fleet dogfood for security-related notifications. | Nathanael Holliday |
| Changeset for onboarding issue template | Quarterly | pull up the changeset in the onboarding issue template and send out a link to the diff to all team members by posting in Slack's `#general` channel. | Mike McNeil |
| MDM device enrollment | Quarterly | Provide export of MDM enrolled devices to the ops team. | Luke Heath |
-->

#### Stubs
The following stubs are included only to make links backward compatible.

##### Secure company-issued equipment for a team member
Please see [handbook/engineering#secure-company-issued-equipment-for-a-team-member](https://www.fleetdm.com/handbook/engineering#secure-company-issued-equipment-for-a-team-member).

##### Register a domain for Fleet
Please see [handbook/register-a-domain-for-fleet](https://www.fleetdm.com/handbook/engineering#register-a-domain-for-fleet).

##### Updating personnel details
Please see [handbook/engineering#update-personnel-details](https://www.fleetdm.com/handbook/engineering#update-personnel-details).

##### Fix a laptop that's not checking in
Please see [handbook/engineering#fix-a-laptop-thats-not-checking-in](https://www.fleetdm.com/handbook/engineering#fix-a-laptop-thats-not-checking-in)

##### Enroll a macOS host in dogfood
Please see [handbook/engineering#enroll-a-macos-host-in-dogfood](https://www.fleetdm.com/handbook/engineering#enroll-a-macos-host-in-dogfood)

##### Enroll a Windows or Ubuntu Linux device in dogfood
Please see [handbook/engineering#enroll-a-windows-or-ubuntu-linux-device-in-dogfood](https://www.fleetdm.com/handbook/engineering#enroll-a-windows-or-ubuntu-linux-device-in-dogfood)

##### Enroll a ChromeOS device in dogfood
Please see [handbook/engineering#enroll-a-chromeos-device-in-dogfood](https://www.fleetdm.com/handbook/engineering#enroll-a-chromeos-device-in-dogfood)

##### Lock a macOS host in dogfood using fleetctl CLI tool
Please see [handbook/engineering#lock-a-macos-host-in-dogfood-using-fleetctl-cli-tool](https://www.fleetdm.com/handbook/engineering#lock-a-macos-host-in-dogfood-using-fleetctl-cli-tool)

##### Book an event
Please see [handbook/engineering#book-an-event](https://www.fleetdm.com/handbook/engineering#book-an-event)

##### Order SWAG
Please see [handbook/engineering#order-swag](https://www.fleetdm.com/handbook/engineering#order-swag)


<meta name="maintainedBy" value="jostableford">
<meta name="title" value="💸 Finance">

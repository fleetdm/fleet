# Business Operations
This handbook page details processes specific to working [with](#contact-us) and [within](#responsibilities) this department.

## Team
| Role                          | Contributor(s)           |
|:------------------------------|:-----------------------------------------------------------------------------------------------------------|
| Head of Business Operations   | [Joanne Stableford](https://www.linkedin.com/in/joanne-stableford/) _([@jostableford](https://github.com/JoStableford))_
| Business Operations Engineer  | [Nathan Holliday](https://www.linkedin.com/in/nathanael-holliday/) _([@hollidayn](https://github.com/hollidayn))_ <br> [Isabell Reedy](https://www.linkedin.com/in/isabell-reedy-202aa3123/) _([@ireedy](https://github.com/ireedy))_

## Contact us
- To **make a request** of this department, [create an issue](https://github.com/fleetdm/confidential/issues/new?assignees=&labels=%23g-business-operations&projects=&template=custom-request.md&title=Request%3A+_______________________) and a team member will get back to you within one business day (If urgent, mention a [team member](#team) in [#g-business-operations](https://fleetdm.slack.com/archives/C047N5L6EGH).
  - Please **use issue comments and GitHub mentions** to communicate follow-ups or answer questions related to your request.
  - Any Fleet team member can [view the kanban board](https://app.zenhub.com/workspaces/-g-business-operations-63f3dc3cc931f6247fcf55a9/board?sprints=none) for this department, including pending tasks and the status of new requests.


## Responsibilities
The Business Operations department is directly responsible for people operations, finance + invoicing, tax, compliance, and legal + deal desk.


### Run payroll
Many of these processes are automated, but it's vital to check Gusto and Plane manually for accuracy.
 - Salaried fleeties are automated in Gusto and Plane.
 - Hourly fleeties and consultants are a manual process each month in Gusto and Plane.

| Payroll type                 | What to use                  | DRI                          |
|:-----------------------------|:-----------------------------|:-----------------------------|
| [Commissions and ramp](https://fleetdm.com/handbook/business-operations#run-us-commission-payroll)         | "Off-cycle - Commission" payroll          | Head of Business Operations
| Sign-on bonus                | "Bonus" payroll              | Head of Business Operations
| Performance bonus            | "Bonus" payroll              | Head of Business Operations     
| Accelerations (quarterly)    | "Off-cycle - Commission" payroll          | Head of Business Operations
| [US contractor payroll](https://fleetdm.com/handbook/business-operations#run-us-contractor-payroll) | "Off-cycle" payroll | Head of Business Operations

### Reconcile monthly recurring expenses
Recurring monthly or annual expenses, such as the tools we use throughout Fleet, are tracked as recurring, non-personnel expenses in ["🧮 The Numbers"](https://docs.google.com/spreadsheets/d/1X-brkmUK7_Rgp7aq42drNcUg8ZipzEiS153uKZSabWc/edit#gid=2112277278) _(¶confidential Google Sheet)_, along with their payment source. Reconciliation of recurring expenses happens monthly. <!-- TODO: Merge "🧮 The Numbers" and  ["Tools we use" (private Google doc)](https://docs.google.com/spreadsheets/d/170qjzvyGjmbFhwS4Mucotxnw_JvyAjYv4qpwBrS6Gl8/edit?usp=sharing) -->

> Use this spreadsheet as the source of truth.  Always make changes to it first before adding or removing a recurring expense. Only track significant expenses. (Other things besides amount can make a payment significant; like it being an individualized expense, for example.)


### Access a background check
All Fleet team members undergo a background check provided through [Vetty](https://vetty.co/). Only the most recent background checks appear on the home page of Vetty's dashboard. To access a complete list of background checks run in Vetty, scroll down to the bottom of the candidates page and click "View Historical".


### Register Fleet as an employer with a new state 
Fleet must register as an employer in any state where we hire new teammates. To do this, complete the following steps in Gusto:
1. After a new teammate completes their Gusto profile, the Business Operations department will be prompted to approve it for payroll. Sign in to your Gusto admin account and begin the approval process.
2. Select "yes" when prompted to file a new hire report and complete the approval process.
3. Once the profile is approved, navigate to Tax setup and select the state you’d like to register Fleet in.
4. Select “Have us register for you” and then “Start registration.”
5. Verify, add, and amend any company information to ensure accuracy.
6. Select “Send registration” and authorize payment for the specified amount. CorpNet will then send an email with next steps, which vary by state.
7. Update the [list of states that Fleet is currently registered with as an employer](https://fleetdm.com/handbook/business-operations#review-state-employment-tax-filings-for-the-previous-quarter).
    

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


### Inform managers about hours worked

Every Friday at 2:00 PM CT, we collect hours worked for all hourly employees at Fleet, including core team members and consultants, regardless of their location.

Here's how:

1. For each hourly core team member in Gusto or Plane.com, find the DRI by checking [who they report to](https://docs.google.com/spreadsheets/d/1OSLn-ZCbGSjPusHPiR5dwQhheH1K8-xqyZdsOe9y7qc/edit#gid=0).
   - If any direct report is hourly in Plane.com and submits hours monthly, still list them and provide an explanation.
2. Consultants submit their hours through Gusto (US consultants) or Plane.com (international consultants) and require DRI approval. Find the DRI using the [Business Operations KPIs](https://docs.google.com/spreadsheets/d/1Hso0LxqwrRVINCyW_n436bNHmoqhoLhC8bcbvLPOs9A/edit#gid=0).
3. Send the teammate's DRI a direct message in Slack with a screenshot of the HRIS portal, showing hours logged since last Saturday at midnight, and ask them to confirm the hours are expected. Ensure the screenshot does not include compensation information.
4. The following Monday, check for updates to logged hours and ensure the KPI sheet aligns with HRIS records.
   - If there are discrepancies between what was previously reported, reconfirm logged hours with the teammate's DRI and update the KPI sheet to reflect the correct amount.


### Change the DRI of a consultant

1. In the [KPIs](https://docs.google.com/spreadsheets/d/1Hso0LxqwrRVINCyW_n436bNHmoqhoLhC8bcbvLPOs9A/edit#gid=0) sheet, find the consultant's column.
2. Change the DRI documented there to the new DRI who will receive information about the consultant's hours.

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
1. Go to this [report folder](https://fleetdm.lightning.force.com/lightning/r/Folder/00lUG000000DstpYAC/view?queryScope=userFolders) in SFDC. The three reports will provide the data used in the report.
2. Copy the template below and paste it into the [#g-sales slack channel](https://fleetdm.slack.com/archives/C030A767HQV) and complete all "todos" using the data from Salesforce before sending. 

```
Weekly revenue report - [@`todo: CRO` and @`todo: CEO`]
- Number accounts with outstanding balances = `todo`
- Number of customers awaiting invoices = `todo`
- Number of past-due renewals = `todo`
```

3. Send payment reminders via email to all outstanding accounts by responding to the invoice email initially sent to the customer.

```
Hello,
This is a reminder that you have an outstanding balance due for your Fleet Device Management premium subscription.
We have included the invoice here for your convenience.
For payment instructions please refer to your invoice, and reach out to [Fleet's billing contact] with any questions.

Thanks,
[name]
```
4. If any accounts will become overdue within a week, reply in thread to the slack post, mention the opportunity owner of the account, and ask them to notify their contact that Fleet is still awaiting payment.
5. Review the [billing cycles](https://fleetdm.lightning.force.com/lightning/r/Report/00OUG000000yGjR2AU/view) report in SFDC for customers on multiyear deals. For any customers due for invoicing within the next week, create an issue on the Business Operations board.


### Run US commission payroll
1. Update individual teammates commission calculators (linked from [main commission calculator](https://docs.google.com/spreadsheets/d/1PuqUbfPGos87TfcHWgUd05TRJgQLlBmhyz1euj79m2A/edit?usp=sharing)) with new revenue from any deals that are closed-won (have a subscription agreement signed by both parties) and have a **close date** within the previous month.
    - Verify closed-won deal numbers with CRO to ensure any agreed upon exceptions are captured (eg: CRO approves an AE to receive commission on a renewal deal due to cross-sell).
2. In the "Monthly commission payroll party" meeting, present the commission calculations for Fleeties receiving commission for approval.
    - If there are any quarterly accelerators due for the teammate receiving commission, ensure the individual total includes both the monthly and the quarterly amount.
3. After the amounts are approved in the meeting, process the commission payroll.
    - Use the off-cycle payroll option in Gusto. Be sure to classify the payment as "Commission" in the "other earnings" field and not the generic "Bonus."
4. Once commission payroll has been run, update the [main commission calculator](https://docs.google.com/spreadsheets/d/1PuqUbfPGos87TfcHWgUd05TRJgQLlBmhyz1euj79m2A/edit?usp=sharing) to mark the commission as paid.

### Run international commission payroll
1. Follow the steps in [run US commission payroll](https://fleetdm.com/handbook/business-operations#run-us-commission-payroll) to have the commission amounts approved by the CRO.
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


### Convert a Fleetie to a consultant
If a Fleetie decides they want to move to being a [consultant](https://fleetdm.com/handbook/company/leadership#consultants), either the Fleetie or their manager need to create a [custom issue for the BizOps team](https://github.com/fleetdm/confidential/issues/new?assignees=&labels=%23g-business-operations&projects=&template=custom-request.md&title=Request%3A+_______________________) to notify them of the change.
Once notified, BizOps takes the following steps:
1. Confirm the following details with the Fleetie:
    - Date of change
    - Term of consultancy (time period)
    - Hours/capacity expected (hours per week or month)
    - Confirm hourly rate
2. Once details are confirmed, use the information given to create the consulting agreement for the Fleetie (either in docusign (US-based) or via Plane (international)), and send to their personal email for signature. Once signed, save in Fleetie's [employee file](https://drive.google.com/drive/folders/1UL7o3BzkTKnpvIS4hm_RtbOilSABo3oG?usp=drive_link).
3. Schedule the Fleetie's final day in HRIS (Gusto or Plane).
4. Update final day in ["🧑‍🚀 Fleeties"](https://docs.google.com/spreadsheets/d/1OSLn-ZCbGSjPusHPiR5dwQhheH1K8-xqyZdsOe9y7qc/edit#gid=0) spreadsheet.
5. Create an [offboarding issue](https://github.com/fleetdm/classified/blob/main/.github/ISSUE_TEMPLATE/%F0%9F%9A%AA-offboarding-____________.md) for the Fleetie converting to a consultant, and confirm with their manager if there is a need to retain any tools or access while they are a consultant (default to removing all access from Fleet email, and migrating to personal email for Slack and other tools unless there is a business case to retain the Fleet email and associated tool access).
6. Follow the offboarding issue for next steps, including communicating to teammates and updating equity plan.


### Update personnel details
When a Fleetie, consultant or advisor requests an update to their personnel details (name, location, phone, etc), follow these steps to ensure accurate representation across systems.
1. Team member submits a [custom issue](https://github.com/fleetdm/confidential/issues/new?assignees=&labels=%23g-business-operations&projects=&template=custom-request.md&title=Request%3A+_______________________) to update their personnel details (or BizOps team creates if the request comes via email or is sensitive and needs a classified issue).
    - If change is for a primary identification or contact method, ask for evidence of change and capture in [employee's personnel file](https://drive.google.com/drive/folders/1UL7o3BzkTKnpvIS4hm_RtbOilSABo3oG?usp=drive_link).
2. BizOps makes change to HRIS (Gusto or Plane) to reflect change. 
    - Note: if making the change requires follow up steps, resolve those steps to action the change.
3. Once change is effected in HRIS, BizOps makes changes to ["🧑‍🚀 Fleeties"](https://docs.google.com/spreadsheets/d/1OSLn-ZCbGSjPusHPiR5dwQhheH1K8-xqyZdsOe9y7qc/edit#gid=0) spreadsheet.
4. If required, BizOps makes any relevant changes to [Fleet's equity plan](https://docs.google.com/spreadsheets/d/1_GJlqnWWIQBiZFOoyl9YbTr72bg5qdSSp4O3kuKm1Jc/edit#gid=0).
5. If required, BizOps makes any relevant changes to the ["🗺️ Geographical factors"](https://docs.google.com/spreadsheets/d/1rCVCs-eOo-VSEG7fPLgdq5l7oSaActl5bewaWP7PnSE/edit#gid=1533353559) spreadsheet and follows through on any action items involving tax implications (i.e. registering with a new state for employer taxes).
6. If required, BizOps also makes changes to other core systems (e.g: creating a new email alias in google workspace; updating details in Carta; etc).
7. The change is now actioned, notify the team member and close the issue.

> Note: if the Fleetie is US based and has a qualifying life event that impacts benefit coverage, they can [follow the Gusto steps](https://support.gusto.com/article/100895878100000/Change-your-benefits-with-a-qualifying-life-event) to update their coverage elections.


### Change a Fleetie's job title
When BizOps receives notification of a Fleetie's job title changing, follow these steps to ensure accurate recording of the change across our systems.
1. Update ["🧑‍🚀 Fleeties"](https://docs.google.com/spreadsheets/d/1OSLn-ZCbGSjPusHPiR5dwQhheH1K8-xqyZdsOe9y7qc/edit#gid=0):
    - Search the spreadsheet for the Fleetie in need of a job title change.
    - Input the new job title in the Fleetie's row in the "Job title" cell.
    - Navigate to the "Org chart" tab of the spreadsheet, and verify that the Fleetie's title appears correctly in the org chart.
2. Update the departmental handbook page with the change of job title
3. Update the relevant payroll/HRIS system.
    - For updating Gusto (US-based Fleeties):
      - Login to Gusto and navigate to "People > Team members".
      - Find the Fleetie and select them to see their profile page.
      - Under the "Compensation" heading, select edit and update the "Job title" and input the specific date the change happened. Save the changes.
    - For updating Plane (non-US Fleeties):
      - Login to Plane and navigate to "People > Team".
      - Find the Fleetie and select them to see their profile page.
      - Use the "Help" function, or email support@plane.com to notify Plane of the need to change the job title for the Fleetie. Include the Fleetie's name, current title, new title, and effective date.
      - Take any relevant steps as directed by Plane in order to make the required changes to the Fleetie's profile.


### Change a Fleetie's manager
When BizOps receives notification of a Fleetie's manager changing, follow these steps to ensure correct recording in our systems.
1. Update [🧑‍🚀 Fleeties](https://docs.google.com/spreadsheets/d/1OSLn-ZCbGSjPusHPiR5dwQhheH1K8-xqyZdsOe9y7qc/edit#gid=0):
    - Search for the Fleetie's new manager, and copy the new manager's unique ID from the far left "Unique ID" column.
    - Search for the Fleetie who's manager is changing, and paste (without formatting) their new manager's unique ID in the "Reports to: (manager unique ID)" cell in the Fleetie's row.
    - Verify that the "Reports to (auto: manager name and job title)" cell in the Fleetie's row reflects the new manager's details.
    - Verify that in the new manager's row, the "# direct reports" cell reflect the correct number.
    - Navigate to the "Org chart" tab in the spreadsheet, and verify that the Fleetie now appears in the correct place in the org chart.
2. If the person's department is changing, then update both departmental handbook pages to move the person to their new department:
    - Remove the person from the "Team" section of the old department and add them to the "Team" section of the new department.
3. If the person's level of confidential access will change along with the change to their manager, then update that level of access:
    - Update Google Workspace to make sure this person lives in the correct Google Group, removing them from the old and/or adding them to the new.
    - Update 1password to remove this person from old vaults and/or add them to new vaults.
    - For a team member moving from "classified" to "confidential" access, check Gusto, Plane, and other systems to remove their access.

> **Note:** The Fleeties spreadsheet is the source of truth for who everyone's manager is and their job titles.

### Recognize employee workiversaries

At Fleet, everyone is recognized on their [workiversary](https://fleetdm.com/handbook/company/communications#workiversaries). To ensure this happens, take the following steps:

1. Bimonthly, use [Fleeties (private google doc)](https://docs.google.com/spreadsheets/d/1OSLn-ZCbGSjPusHPiR5dwQhheH1K8-xqyZdsOe9y7qc/edit#gid=0) to determine who is celebrating their workiversary in the following two months.
2. Post in the #help-classifed Slack channel and cc the Head of Business Operations. Use the following template:

   
    ```
    [Month]
    [workiversary date (DD-MMM)] - [teammate name] - [number of years at Fleet]
    ```

    
   The Apprentice to the CEO will also use this post to update the [All hands](https://fleetdm.com/handbook/company/communications#all-hands) deck.
3. On the day prior to a workiversary, send the teammate’s manager a DM on Slack:


    ```
    Hey! Just a heads up, tomorrow is [teammate’s name] [number of years at Fleet] workiversary at Fleet.
    BizOps were planning on posting something in the #random channel to recognize them, but I was wondering if you would like to instead?
    ```

    
   > If a manager elects to post and hasn't done so by 2pm ET on the day of the workiversary, send them a friendly reminder and offer to post instead.

4. If the manager has deferred to BizOps, schedule a Slack post for the following day to recognize the teammate's contributions at Fleet. If you’re unsure about what to post, take a look at what’s been [posted previously](https://docs.google.com/document/d/1Va4TYAs9Tb0soDQPeoeMr-qHxk0Xrlf-DUlBe4jn29Q/edit).



### Prepare salary benchmarking information
1. Use the relevant template text in the README section of the [¶¶ 💌 Compensation decisions document](https://docs.google.com/document/d/1NQ-IjcOTbyFluCWqsFLMfP4SvnopoXDcX0civ-STS5c/edit?usp=sharing) for a current Fleetie, a new role, a prospective hire, or other benchmarking use case.
2. Copy the template text and paste at the end of the document.
3. Fill in details as required, pulling from [🧑‍🚀 Fleeties spreadsheet](https://docs.google.com/spreadsheets/d/1OSLn-ZCbGSjPusHPiR5dwQhheH1K8-xqyZdsOe9y7qc/edit#gid=0) and [equity spreadsheet](https://docs.google.com/spreadsheets/d/1_GJlqnWWIQBiZFOoyl9YbTr72bg5qdSSp4O3kuKm1Jc/edit?usp=sharing) as required.
4. Use the teammate's information to benchmark in [Pave](https://www.pave.com/) (login details in 1Password). You can pattern match from previous benchmarking entries, and include all company assumtions. Add the direct link to the Pave benchmark.


### Update a team member's compensation
To [change a team member's compensation](https://fleetdm.com/handbook/company/communications#compensation-changes), follow these steps:
1. Create a copy of the ["Values assessment" template](https://docs.google.com/spreadsheets/d/1P5TyRV2v-YN0aR_X8vd8GksKcr3uHfUDdshqpVzamV8/edit?usp=drive_link) and move it to the team member's [personnel folder in Google Drive](https://drive.google.com/drive/folders/1UL7o3BzkTKnpvIS4hm_RtbOilSABo3oG?usp=drive_link).
2. Share the values assessment document with the manager and ask them to perform the values assessment.
3. Once the values assessment is complete, [prepare salary benchmarking information](#prepare-salary-benchmarking-information) and set a meeting between the manager, head of department, and Head of Business Operations about level of job skill in relation to compensation benchmarking levels.
4. Schedule a new calendar event for the Head of Business Operations with the founders over an existing founders' 1:1 to discuss if an adjustment needs to be made to team member's compensation to align with benchmarking. During the 1:1 call, founders review values assessment, benchmarking for role and geography, and decide if there will be an adjustment.
5. Head of Business Operations will post in slack to `#help-classified` with the decision on compensation changes and effective date, if any.
6. Communicate via Slack DM the decision to the teammate's people manager, who will then communicate to their teammate.
7. Update the respective payroll platform (Gusto or Plane) by navigating to the personnel page, selecting salary field, and updating with an effective date that makes the next payroll.
8. Update the [equity spreadsheet](https://docs.google.com/spreadsheets/d/1_GJlqnWWIQBiZFOoyl9YbTr72bg5qdSSp4O3kuKm1Jc/edit?usp=sharing) (internal doc) by copying existing OTE to the bottom of the "Notes" cell, updating the OTE column with the new compensation information, and updating the "Last compensation change" column with the effective date from payroll platform.

> If the company decides on an additional equity grant as part of a compensation change, note the previous equity and new situation in detail in the "Notes" column of the equity plan. Update the "Grant started?" column to "todo" which adds it to the queue for the next time grants are processed (quarterly).

### Review Fleet's US company benefits

Annually, around mid-year, Fleet will be prompted by Gusto to review company benefits. The goal is to keep changes minimal. Follow these steps:
1. Log in to your [Gusto admin account](https://gusto.com/).
2. Navigate to "Benefits" and select "Renewal survey".
3. Complete the survey questions, aiming for minimal changes.
4. Approximately 2-3 months after survery completion, Gusto will suggest plans based on Fleet's responses. Choose plans with minimal changes.
5. Gusto will offer these plans to employees during open enrollment, with new coverage starting 3-4 weeks afterward.
   

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


### Grant equity
Equity grants for new hires are queued up as part of the [hiring process](https://fleetdm.com/handbook/business-operations#hiring), then grants and consents are [batched and processed quarterly](https://github.com/fleetdm/confidential/issues/new/choose).

Doing an equity grant involves:
- Executing a board consent
- The recipient and CEO signing paperwork about the stock options
- Updating the number of shares for the recipient in the equity plan
- Updating Carta to reflect the grant

For the status of stock option grants, exercises, and all other _common stock_ including advisor, founder, and team member equity ownership, see [Fleet's equity plan](https://docs.google.com/spreadsheets/d/1_GJlqnWWIQBiZFOoyl9YbTr72bg5qdSSp4O3kuKm1Jc/edit#gid=0).  For information about investor ownership, see [Carta](https://app.carta.com/corporations/1234715/summary/).

> Fleet's [equity plan](https://docs.google.com/spreadsheets/d/1_GJlqnWWIQBiZFOoyl9YbTr72bg5qdSSp4O3kuKm1Jc/edit#gid=0) is the source of truth, not Carta.  Neither are pro formas sent in an email attachment, even if they come from lawyers.
> 
> Anyone can make mistakes, and none of us are perfect.  Even when we triple check.  Small mistakes in share counts can be hard to attribute, and can cause headaches and eat up nights of our CEO's and operations team's time.  If you notice what might be a discrepancy between the equity plan and any other secondary source of information, please speak up and let Fleet's CEO know ASAP.  Even if you're wrong, your note will be appreciated.


### Deliver annual report for venture line
Within 60 days of the end of the year, follow these steps:
1. Provide Silicon Valley Bank (SVB) with our balance sheet and profit and loss statement (P&L, sometimes called a cashflow statement) for the past twelve months.  
2. Provide SVB with our board-approved annual operating budgets and projections (on a quarterly granularity) for the new year.
3. Deliver this as early as possible in case they have questions.


### Process a new vendor invoice
Fleet pays its vendors in less than 15 business days in most cases. All invoices and tax documents should be submitted to the Business Operations department using the [appropriate Fleet email address (confidential Google Doc)](https://docs.google.com/document/d/1tE-NpNfw1icmU2MjYuBRib0VWBPVAdmq4NiCrpuI0F0/edit#heading=h.wqalwz1je6rq).
- After making sure the invoice received from a new vendor is valid, add the new vendor to the recurring expenses section of ["The numbers"](https://docs.google.com/spreadsheets/d/1X-brkmUK7_Rgp7aq42drNcUg8ZipzEiS153uKZSabWc/edit#gid=2112277278) before paying the invoice.
- If we have not paid this vendor before, make sure we have received the required W-9 or W-8 form from the vendor. **Accounting cannot process a payment without these tax forms for compliance reasons.**
  - **US-based vendors** are required to complete a [W-9 form](https://www.irs.gov/pub/irs-pdf/fw9.pdf).
  - **Non-US based vendors and individuals** are required to follow these [instructions](https://www.irs.gov/instructions/iw8bene) and provide a completed [W-8BEN-E](https://www.irs.gov/pub/irs-pdf/fw8bene.pdf) form.



### Process a request to cancel a vendor
- Make the cancellation notification in accordance with the contract terms between Fleet and the vendor, typically these notifications are made via email and may have a specific address that notice must be sent to. If the vendor has an autorenew contract with Fleet there will often be a window of time in which Fleet can cancel, if notification is made after this time period Fleet may be obligated to pay for the subsequent year even if we don't use the vendor during the next contract term.  
- Once cancelled, update the recurring expenses section of [The Numbers](https://docs.google.com/spreadsheets/d/1X-brkmUK7_Rgp7aq42drNcUg8ZipzEiS153uKZSabWc/edit#gid=2112277278) to reflect the cancellation by changing the projected monthly burn in column G to $0 and adding "CANCELLED" in front of the vendor's name in column C.


### Review an NDA
We need to review an NDA anytime a vendor, customer or other party wants to:
- Use their own NDA rather than Fleet's standard NDA, or
- "Redline" (modify) Fleet's NDA by removing, adding or altering its terms.

We should always seek to use Fleet's own NDA first, without alteration. 

When reading an NDA, we want to pay close attention to the following:
- We want to be sure that the confidentiality obligations of the NDA are reciprocal.  Fleet and the other party to the agreement should be bound to the same standards of confidentiality toward the handling of each other's confidential information.
- Fleet does not agree to _"do not compete"_ or _"do not solicit clauses"_. An NDA should not contain provisions beyond the scope of an NDA. The two most commonly encountered examples of this are the "do not compete" and "do not solicit" clauses. We want to be free to hire the best people and make the best products, so when reading through an NDA it is important to keep an eye out for language that prohibits Fleet from hiring or soliciting current or former employees of other companies or that prohibit Fleet from independently developing products that compete with another company's products.  Using the `cmd + f` function to search for "solici", "compet" and "hir" and reading through the results is a helpful method to quickly scan for these clauses.
- Look for any language that discusses a transfer of property rights. Rarely, you may find a clause snuck into an agreement that discusses the transfer of intellectual property rights.  _We want to avoid any situation where Fleet transfers its intellectual property to another party as part of an NDA_.  
- Should you find any clauses in steps 2 or 3 that are beyond the scope of protecting both party's confidential information in a customer NDA or an altered version of Fleet's NDA, reject this language and communicate that Fleet cannot agree to those terms.
- Any concerns or uncertainty over _any_ provisions in an NDA should be brought to Nathanael Holliday in BizOps, who will consult legal counsel if necessary to resolve any concerns.

### Review a vendor agreement
When reviewing contracts from a vendor, Fleet is concerned about the following:
- If there are confidentiality provisions in the agreement in place of a stand-alone NDA, verify the confidentiality provisions are appropriate and protect Fleet when sensitive data is involved that isn't otherwise available to the public.
- We want to make sure there are no _do not solicit_ or _do not compete_ clauses in the contract.  To aid in this search, we double check by using the cmd + f function and searching for "solici", "compet" and "hir" and then looking through the results to be sure that nothing prohibits Fleet from independently developing competing products or from hiring personnel with ties to the vendor.
- We want to make sure that contracts can be terminated relatively easily and be aware of what the process is for terminating them, avoiding commitments over 12 months in length.
- We want to make sure the payment terms work for us (i.e. being able to pay via wire transfer, credit card or bill.com) and that the price in any contract or order form is what we have agreed to.  While almost never malicious, mistakes often occur in the steps between agreeing on a price, negotiating a contract, and receiving an invoice.  We want to be sure at every step that the dollar amount and service provided is consistent with what has been negotiated and agreed upon.
- Remember, once we have signed the agreement - we're stuck with it. If any clause in the agreement appears strange or gives you pause or concern, it is better to seek clarification than to commit to something that might be detrimental to Fleet.  Contracts are fairly standardized, and you'll quickly learn what is normal and what feels out of place.  Unusual clauses or wording that seems out of the ordinary should get a second set of eyes just to be sure, do not hesitate to reach out to Nathanael Holliday with questions, who will reach out to legal counsel as necessary.

### Review an order form
- We should always check order forms for additional terms that go beyond the scope of the order form (caps on price increases, for example).
- Be sure the order form includes contact information + billing address and information so that Fleet knows how and who to invoice for payment.
- Verify that the payment terms are correct and matches what's in the agreement. This is a frequent common mistake as companies usually have default payment terms and overlook changing them to match atypical payment terms.
- Make sure the effective term of the order matches what was agreed upon (usually a one year term) and that the order form includes the correct number of hosts and whether or not it should contain professional services (usually, it does not). 
- Check that the amount on the order form reflects what Fleet agreed to, as this is the amount that the customer will expect to be invoiced for.
- Lastly, double check one more time to make sure there are no sneaky, unusual terms snuck in at the bottom of an order form or stashed away in fine print.  Common things that are included in order forms and not always communicated to Fleet are caps on price increases upon renewal, new SLAs, or a product roadmap or milestones we may not have agreed upon.  Any clauses on an order form that appear beyond the scope of simply elaborating on the services being provided, the purchase cost, the contract that the purchase is being made under, how Fleet will bill and how the customer will pay deserves a careful look.  Reach out to Nathanael Holliday in BizOps with concerns.

### Review a non-standard subscription agreement
We want to use our standard terms whenever possible with our customers, but it is common that customers want to use their own agreement or redline (modify) Fleet's terms.  
When reviewing subscription agreements on customer paper or when a customer has made changes to Fleet's terms, we review it using [these guidelines](https://docs.google.com/document/d/1aGgN5It1i3fdsBF37vWSbvukO_gQhy5vCp4fINg191Q/edit?usp=sharing).
   

### Update weekly KPIs
- Create the weekly update issue from the template in ZenHub every Friday and update the [KPIs for BizOps](https://docs.google.com/spreadsheets/d/1Hso0LxqwrRVINCyW_n436bNHmoqhoLhC8bcbvLPOs9A/edit#gid=0) by 5pm US central time.
- Check the KPI sheet at 5pm US central time to ensure all departments have updated their KPIs on time.  If any departments are delinquent, notify the department head and let the [Apprentice](https://fleetdm.com/handbook/digital-experience#team) know so they can put it on the agenda for their next one-on-one with the CEO.

### Tech stack admins
| Role | Google Workspace | Slack | GitHub | Gusto | Pilot | Plane | 1Password |
|:----------------------|------------------:|------------------:|------------------:|------------------:|------------------:|------------------:|------------------:|
| CEO | ✅&nbsp;Super&nbsp;admin | ✅&nbsp;Primary workspace owner | ✅&nbsp;Owner | ✅&nbsp;Primary admin | ✅&nbsp;Admin| ✅&nbsp;Owner | ✅ Owner |
| CTO | ❌ | ❌ | ✅ Owner | ❌ | ✅ Admin | ❌ | ❌ |
| Head of BizOps | ✅ Super admin | ✅ Owner | ✅ Owner| ✅ Admin | ✅ Admin| ✅ Admin | ✅ Admin |
| BizOps engineer | ✅ Super admin| ✅ Owner | ✅ Owner| ✅ Admin | ✅ Admin| ✅ Admin | ✅ Admin|
| Head of Digital&nbsp;Experience | ✅ Super admin| ✅ Owner | ✅ Owner| ❌ | ✅ Admin| ❌ | ✅ Admin|
| Apprentice | ❌ | ❌ | ❌ | ❌ | ✅ Admin| ❌ | ❌ |
| Digital&nbsp;Experience Engineer | ✅ Super admin | ✅ Admin | ❌ | ❌ | ❌ | ❌ | ✅ Admin|
| Head of Product&nbsp;Design | ❌ | ✅ Admin | ❌ | ❌ | ❌ | ❌ | ❌ |
| VP of CX | ❌ | ✅ Owner | ❌ | ❌ | ❌ | ❌ | ❌ |
| CX Sr. Suppoert Engineer | ❌ | ✅ Admin | ❌ | ❌ | ❌ | ❌ | ❌ |
| Client Platform Engineer & Community&nbsp;Advocate | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ Admin|
| Pilot bookkeeper | ❌ | ❌ | ❌ | ✅ Admin  | ❌ | ✅ Admin | ❌ |


## Rituals

The following table lists this department's rituals, frequency, and Directly Responsible Individual (DRI).

<rituals :rituals="rituals['handbook/business-operations/business-operations.rituals.yml']"></rituals>

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

##### Vetty
Please see [hanbook/business-operations#access-a-background-check](https://www.fleetdm.com/handbook/business-operations#access-a-background-check).

##### Role-specific licenses  
Please see [hanbook/business-operations#grant-role-specific-license-to-a-team member](https://www.fleetdm.com/handbook/business-operations#grant-role-specific-license-to-a-team-member).

##### Recurring expenses
##### Tools we use
Please see [hanbook/business-operations#grant-role-specific-license-to-a-team member](https://www.fleetdm.com/handbook/business-operations#reconcile-monthly-recurring-expenses).

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
<meta name="title" value="🔦 Business Operations">

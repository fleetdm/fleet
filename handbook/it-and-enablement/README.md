# 🌐 IT and Enablement 

This page details processes specific to working [with](#contact-us) and [within](#responsibilities) this department.


## Team

| Role                                    | Contributor(s)
|:----------------------------------------|:----------------------------------------------------------------------|
| Head of IT & Enablement                 | [Allen Houchins](https://www.linkedin.com/in/allenhouchins/) _([@allenhouchins](https://github.com/allenhouchins))_
| Head of Digital Workplace & GTM Systems | [Sam Pfluger](https://www.linkedin.com/in/sampfluger88/) _([@sampfluger88](https://github.com/sampfluger88))_ 
| Manager of Training and Enablement      | [Brock Walters](https://www.linkedin.com/in/brock-walters-247a2990/) _([@nonpunctual](https://github.com/nonpunctual))_
| Solutions Consultant (SC)               | <sup><sub> _See [🦄 Go-To-Market groups](https://fleetdm.com/handbook/company/go-to-market-groups#current-gtm-groups)


## Contact us

- To **make a request** of this department, [create an issue](https://github.com/fleetdm/confidential/issues/new?assignees=&labels=%3Ahelp-it-and-enablement&projects=&template=1-custom-request.md&title=) and a team member will get back to you within one business day (If urgent, mention a [team member](#team) in the [#help-it-and-enablement](https://fleetdm.slack.com/archives/C09861YJUJ2) Slack channel.
  - Any Fleet team member can [view the kanban board](https://github.com/orgs/fleetdm/projects/69) for this department, including pending tasks and the status of new requests.
  - Please **use issue comments and GitHub mentions** to communicate follow-ups or answer questions related to your request.


## Responsibilities

The IT & Enablement department is directly responsible for solutions consulting, customer training curriculum, prospect enablement, and dogfooding, as well as the framework, schema, equipment, internal tooling, automation, and technology behind Fleet's Go-To-Market (GTM) systems, remote work, the handbook, issue templates, Zapier flows, Docusign templates, key spreadsheets, and project management tools.


### Manage duplicate accounts in CRM

1. Navigate to ["Ω Possible duplicate accounts" report](https://fleetdm.lightning.force.com/lightning/r/Report/00OUG000001FA1h2AG/view?queryScope=userFolders).
2. Verify that each potential duplicate account is indeed a duplicate of the account's it has been paired with.
3. Open duplicate accounts and compare duplicate accounts to select the best account to "Use as principal" (the account all other duplicates will be merged into). Consider the following:
  - Is there an open opportunity on any of the accounts? If so, this is your "principal" account.
  - Do any of the accounts not have contacts? If no contacts found on the account and no significant activity, delete the account. 
  - Do any of these accounts have activity that the others don't have (e.g. a rep sent an email or logged a call)? Be sure to preserve the maximum amount of historical activity on the principal account.
4. Click view duplicates, select all relevant accounts that appear. Click next.
5. Select the best and most up-to-date data to combine into the single principal account.

> Do *NOT* change account owners if you can help it during this process. For "non-sales-ready" accounts default to the Integrations Admin. If the account is owned by an active user, be sure they maintain ownership of the principal account. 

6. YOU CAN NOT UNDO THIS NEXT PART! Click next, click merge. 
7. Verify that the principal account details match exactly what is on LinkedIn. The end result should be as follows:
  - LinkedIn company url
  - Website
  - Employees


### Register a domain for Fleet

Domain name registrations are handled through Namecheap. Access is managed via 1Password.


### Purchase a SaaS tool

When procuring SaaS tools and services, analyze the purchase of these subscription services look for these way to help the company:
- Get product demos whenever possible.  Does the product do what it's supposed to do in the way that it is supposed to do it?
- Avoid extra features you don't need, and if they're there anyway, avoid using them.
- Data portability: is it possible for Fleet to export it's data if we stop using it? Is it easy to pull that data in an understandable format?
- Programability: Does it have a publicly documented legible REST API that requires at most a single API token?
- Intentionality: The product fits into other tools and processes that Fleet uses today. Avoid [unintended consequences](https://en.wikipedia.org/wiki/Midas). The tool will change to fit the company, or we won't use it. 


### Secure company-issued equipment for a team member

As soon as an offer is accepted, Fleet provides laptops and YubiKey security keys for core team members to use while working at Fleet. The IT engineer will work with the new team member to get their equipment requested and shipped to them on time.

- [**Check the "📦 Warehouse" team in dogfood**](https://dogfood.fleetdm.com/dashboard?team_id=279) before purchasing any equipment including laptops, to ensure we efficiently [utilize existing assets before spending money](https://fleetdm.com/handbook/company/why-this-way#why-spend-less). If Fleet IT warehouse inventory can meet the needs of the request, file a [warehouse request](https://github.com/fleetdm/confidential/issues/new?assignees=sampfluger88&labels=&projects=&template=warehouse-request.md&title=%F0%9F%92%BB+Warehouse+request).

- Apple computers shipping to the United States and Canada are ordered using the Apple [eCommerce Portal](https://ecommerce2.apple.com/asb2bstorefront/asb2b/en/USD/?accountselected=true), or by contacting the business team at an Apple Store or contacting the online sales team at [800-854-3680](tel:18008543680). The IT engineer can arrange for same-day pickup at a store local to the Fleetie if needed.
  - **Note:** Most Fleeties use 16-inch MacBook Pros. Team members are free to choose any laptop or operating system that works for them, as long as the price [is within reason](https://www.fleetdm.com/handbook/communications#spending-company-money). 

  - When ordering through the Apple eCommerce Portal, look for a banner with *Apple Store for FLEET DEVICE MANAGEMENT | Welcome [Your Name].* Hovering over *Welcome* should display *Your Profile.* If Fleet's account number is displayed, purchases will be automatically made available in Apple Business Manager (ABM).

- Apple computers for Fleeties in other countries should be purchased through an authorized reseller to ensure the device is enrolled in ADE. In countries that Apple does not operate or that do not allow ADE, work with the authorized reseller to find the best solution, or consider shipping to a US based Fleetie and then shipping on to the teammate. 

 > A 3-year AppleCare+ Protection Plan (APP) should be considered default for Apple computers >$1500. Base MacBook Airs, Mac minis, etc. do not need APP unless configured beyond the $1500 price point. APP provides 24/7 support, and global repair coverage in case of accidental screen damage or liquid spill, and battery service.

 - Order a pack of two [YubiKey 5C NFC security keys](https://www.yubico.com/product/yubikey-5-series/yubikey-5c-nfc/) for new team member, shipped to them directly.

- Include delivery tracking information when closing the support request so the new employee can be notified.


### Process incoming equipment

Upon receiving any device, follow these steps to process incoming equipment.
1. Find the device in ["🍽️ Dogfood"](https://dogfood.fleetdm.com/dashboard) to confirm the correct equipment was received.
2. Visibly inspect equipment and all related components (e.g. laptop charger) for damage.
3. Remove any stickers and clean devices and components.
4. Using the device's charger, plug in the device.
5. Using your company laptop, navigate to the host in dogfood, and click `actions` » `Unlock` and copy the unlock code. 
6. Turn on the device and enter the unlock code.
7. If the previous user has not wiped the device, navigate to the host in dogfood, and click `actions` » `wipe` and wait until the device is finished and restarts.

**If you need to manually recover a device or reinstall macOS**
1. Enter recovery mode using the [appropriate method](https://support.apple.com/en-us/HT204904).
2. Connect the device to WIFI.
3. Using the "Recovery assistant" tab (In the top left corner), select "Delete this Mac".
4. Follow the prompts to activate the device and reinstall the appropriate version of macOS.


### Ship approved equipment

Once the Digital Experience department approves inventory to be shipped from Fleet IT, follow these step to ship the equipment.
1. Compare the equipment request issue with the ["📦 Warehouse" team](https://dogfood.fleetdm.com/settings/teams/users?team_id=279) and verify physical inventory.
2. Plug in the device and ensure inventory has been correctly processed and all components are present (e.g. charger cord, power converter).
3. Package equipment for shipment and include Yubikeys (if requested).
4. Change the "host" info to reflect the new user. If you encounter any issues, repeat the [process incoming equipment steps](https://fleetdm.com/handbook/it-and-enablement#process-incoming-equipment).
6. Ship via FedEx to the address listed in the equipment request.
7. Add a comment to the equipment request issue, at-mentioning the requestor with the FedEx tracking info and close the issue.


### Grant role-specific license to a team member

Certain new team members, especially in go-to-market (GTM) roles, will need paid access to paid tools like Salesforce and LinkedIn Sales Navigator immediately on their first day with the company. Gong licenses that other departments need may [request them from IT & Enablement](https://fleetdm.com/handbook/it-and-enablement#contact-us) and we will make sure there is no license redundancy in that department.


### Process a tool upgrade request from a team member

- A Fleetie may request an upgraded license seat for Fleet tools by submitting an issue through GitHub.
- Digital Experience will upgrade or add the license seat as needed and let the requesting team member know they did it.


### Downgrade an unused license seat

- On the first Wednesday of every quarter, the CEO and Head of Digital Workplace & GTM Systems will meet for 30 minutes to audit license seats in Figma, Slack, GitHub, Salesforce and other tools.
- During this meeting, as many seats will be downgraded as possible. When doubt exists, downgrade.
- Afterward, post in #random letting folks know that the quarterly tool reconciliation and seat clearing is complete, and that any members who lost access to anything they still need can submit a GitHub issue to Digital Experience to have their access restored.
- The goal is to build deep, integrated knowledge of tool usage across Fleet and cut costs whenever possible. It will also force conversations on redundancies and decisions that aren't helping the business that otherwise might not be looked at a second time.  


### Add a seat to Salesforce

Here are the steps we take to grant appropriate Salesforce licenses to a new hire:
- Go to ["My Account"](https://fleetdm.lightning.force.com/lightning/n/standard-OnlineSalesHome).
- View contracts -> pick current contract.
- Add the desired number of licenses.
- Sign DocuSign sent to the email.
- The order will be processed in ~30m.
- Once the basic license has been added, you can create a new user using the new team member's `@fleetdm.com` email and assign a license to it.
  - To enable email sync for a user:
    - Navigate to the [user’s record](https://fleetdm.lightning.force.com/lightning/setup/ManageUsers/home) and scroll to the bottom of the permission set section.
    - Add the “Inbox with Einstein Activity Capture” permission set and save.
    - Navigate to the ["Einstein Activity Capture Settings"](https://fleetdm.lightning.force.com/lightning/setup/ActivitySyncEngineSettingsMain/home) and click the "Configurations" tab.
    - Select "Edit", under "User and Profile Assignments" move the new user's name from "Available" to "Selected", scroll all the way down and click save.
   

## Rituals

<rituals :rituals="rituals['handbook/it-and-enablement/it-and-enablement.rituals.yml']"></rituals>


#### Stubs
The following stubs are included only to make links backward compatible.

##### Update a company brand front
Please see [handbook/product-design#update-a-company-brand-front](https://fleetdm.com/handbook/product-design#update-a-company-brand-front)

##### Prepare "Let's get you set up!" meeting notes
Please see [handbook/marketing#prepare-lets-get-you-set-up-meeting-notes](https://fleetdm.com/handbook/marketing#prepare-lets-get-you-set-up-meeting-notes)

### Process the CEO's inbox
Please see [handbook/ceo#process-the-ceos-inbox](https://fleetdm.com/handbook/ceo#process-the-ceos-inbox)

### Process the CEO's calendar
Please see [handbook/ceo#process-the-ceos-calendar](https://fleetdm.com/handbook/ceo#process-the-ceos-calendar)

### Check LinkedIn for new activity 
Please see [handbook/ceo#check-linkedin-for-new-activity](https://fleetdm.com/handbook/ceo#check-linkedin-for-new-activity)

### Add LinkedIn connections to CRM
Please see [handbook/ceo#add-linkedin-connections-to-crm](https://fleetdm.com/handbook/ceo#add-linkedin-connections-to-crm)

### Connect with active community members
Please see [handbook/ceo#connect-with-active-community-members](https://fleetdm.com/handbook/ceo#connect-with-active-community-members)

### Schedule travel for the CEO
Please see [handbook/ceo#schedule-travel-for-the-ceo](https://fleetdm.com/handbook/ceo#schedule-travel-for-the-ceo)

### Schedule CEO interview
Please see [handbook/ceo#schedule-ceo-interview](https://fleetdm.com/handbook/ceo#schedule-ceo-interview)

### Confirm CEO shadow dates
Please see [handbook/ceo#confirm-ceo-shadow-dates](https://fleetdm.com/handbook/ceo#confirm-ceo-shadow-dates)

### Program the CEO to do something
Please see [handbook/ceo#program-the-ceo-to-do-something](https://fleetdm.com/handbook/ceo#program-the-ceo-to-do-something)

### Process and backup Sid agenda
Please see [handbook/ceo#process-and-backup-sid-agenda](https://fleetdm.com/handbook/ceo#process-and-backup-sid-agenda)

### Process and backup E-group agenda 
Please see [handbook/ceo#process-and-backup-e-group-agenda](https://fleetdm.com/handbook/ceo#process-and-backup-e-group-agenda)

### Process the help-being-ceo Slack channel
Please see [handbook/ceo#process-the-help-being-ceo-slack-channel](https://fleetdm.com/handbook/ceo#process-the-help-being-ceo-slack-channel)

### Unroll a Slack thread
Please see [handbook/ceo#unroll-a-slack-thread](https://fleetdm.com/handbook/ceo#unroll-a-slack-thread)

### Delete an accidental meeting recording
Please see [handbook/ceo#delete-an-accidental-meeting-recording](https://fleetdm.com/handbook/ceo#delete-an-accidental-meeting-recording)

### Communicate Fleet's potential energy to stakeholders
Please see [handbook/ceo#communicate-fleets-potential-energy-to-stakeholders](https://fleetdm.com/handbook/ceo#communicate-fleets-potential-energy-to-stakeholders)

### Archive a document
Please see [handbook/ceo#archive-a-document](https://fleetdm.com/handbook/ceo#archive-a-document)

### Approve a new position
Please see [handbook/people#approve-a-new-position](https://fleetdm.com/handbook/people#approve-a-new-position)

### Inform managers about hours worked
Please see [handbook/people#inform-managers-about-hours-worked](https://fleetdm.com/handbook/people#inform-managers-about-hours-worked)

### Prepare for the All hands
Please see [handbook/people#prepare-for-the-all-hands](https://fleetdm.com/handbook/people#prepare-for-the-all-hands)

### Share recording of all hands meeting
Please see [handbook/people#share-recording-of-all-hands-meeting](https://fleetdm.com/handbook/people#share-recording-of-all-hands-meeting)

### Update personnel details
Please see [handbook/people#update-personnel-details](https://fleetdm.com/handbook/people#update-personnel-details)

### Change a Fleetie's role
Please see [handbook/people#change-a-fleeties-role](https://fleetdm.com/handbook/people#change-a-fleeties-role)

### Change a Fleetie's manager
Please see [handbook/people#change-a-fleeties-manager](https://fleetdm.com/handbook/people#change-a-fleeties-manager)

### Prepare salary benchmarking information
Please see [handbook/people#prepare-salary-benchmarking-information](https://fleetdm.com/handbook/people#prepare-salary-benchmarking-information)

### Recognize employee workiversaries
Please see [handbook/people#recognize-employee-workiversaries](https://fleetdm.com/handbook/people#recognize-employee-workiversaries)

### Update a team member's compensation
Please see [handbook/people#update-a-team-members-compensation](https://fleetdm.com/handbook/people#update-a-team-members-compensation)

### Change the DRI of a consultant
Please see [handbook/people#change-the-dri-of-a-consultant](https://fleetdm.com/handbook/people#change-the-dri-of-a-consultant)

### Add an advisor
Please see [handbook/people#add-an-advisor](https://fleetdm.com/handbook/people#add-an-advisor)

### Convert a Fleetie to a consultant
Please see [handbook/people#convert-a-fleetie-to-a-consultant](https://fleetdm.com/handbook/people#convert-a-fleetie-to-a-consultant)

### Review Fleet's US company benefits
Please see [handbook/people#review-Fleets-us-company-benefits](https://fleetdm.com/handbook/people#review-Fleets-us-company-benefits)


<meta name="maintainedBy" value="allenhouchins">
<meta name="title" value="🌐 IT & Enablement">

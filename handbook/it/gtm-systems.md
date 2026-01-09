# GTM Systems

This page documents processes for managing Fleet's Go-To-Market (GTM) systems and automation. These processes help ensure our CRM data is accurate, intent signals are captured and acted upon, and our sales and marketing systems work together effectively.

Whether you're measuring intent signals, cleaning up duplicate records in Salesforce, or working with other GTM automation, you'll find the step-by-step procedures here.

## Supported GTM Systems

IT & Enablement manages and supports the following GTM systems and tools:

- **Salesforce** - Customer relationship management (CRM) system for tracking accounts, contacts, opportunities, and sales activities
- **Gong.io** - Conversation intelligence and revenue analytics platform for recording and analyzing sales calls
- **Dripify** - LinkedIn automation and outreach platform
- **Clay** - Data enrichment and lead generation platform
- **LinkedIn Sales Navigator** - LinkedIn's sales tool for finding and connecting with prospects

### Responsibilities

**IT & Enablement** handles account provisioning, access management, and technical setup for all GTM systems. This includes creating user accounts, managing licenses, and configuring integrations.

**GTM Team** handles application administration and day-to-day configuration changes within these systems. This includes workflow configuration, field customization, report creation, and other operational changes that affect how the systems are used.

> **Note:** This list will be updated as new systems are added or existing ones are retired. For access requests or questions about any GTM system, [contact IT & Enablement](https://fleetdm.com/handbook/it#contact-us).

## Measure intent signals

Daily, follow the steps in the [🦄⚡️🌐 Go-To-Market strategy doc (confidential)](https://github.com/fleetdm/confidential/blob/main/go-to-market-strategy.md#daily) to measure and process intent signals.

## Manage duplicates in CRM

1. For accounts, navigate to the ["Ω Possible duplicate accounts" report](https://fleetdm.lightning.force.com/lightning/r/Report/00OUG000001FA1h2AG/view). For Contacts, navigate to the ["Ω Possible duplicate contacts" report](https://fleetdm.lightning.force.com/lightning/r/Report/00OUG000002qAoX2AU/view).
2. Verify that each potential duplicate record is indeed a duplicate of the record it has been paired with.
3. Open and compare the duplicate records to select the most up-to-date record to "Use as principal" (the record all other duplicates will be merged into). Consider the following:
  - Is there an open opportunity on any of the records? If so, this is your "principal" account/contact.
  - Do any of the accounts not have contacts? If no contacts found on the account and no significant activity, delete the account. 
  - Do any of these accounts/contacts have activity that the others don't have (e.g. a rep sent an email or logged a call)? Be sure to preserve the maximum amount of historical activity on the principal record.
4. Click "View duplicates", select all relevant records that appear. Click next.
5. Select the best and most up-to-date data to combine into the single principal account/contact.

> Do *NOT* change account owners if you can help it during this process. For "non-sales-ready" accounts default to the Integrations Admin. If the account is owned by an active user, be sure they maintain ownership of the principal account. 

6. YOU CAN NOT UNDO THIS NEXT PART! Click next, click merge. 
7. Verify that the principal record details match exactly what is on LinkedIn.


<!-- 
### Research an account

To research an account, follow the steps in the follow the steps in the [🦄⚡️🌐 Go-To-Market strategy doc (confidential)](https://github.com/fleetdm/confidential/edit/main/go-to-market-strategy.md#research-an-account) and move it toward sales-readiness **after** discovering [relevant intent signals](https://fleetdm.com/handbook/marketing#measure-intent-signals).
-->


<meta name="maintainedBy" value="allenhouchins">
<meta name="title" value="GTM Systems">


# Go-To-Market Architecture


## Automation

### Capture Eventbrite attendees in Salesforce campaigns

> ***TL;DR: It's not working, Who should I call and what can I check?***
> 
> DRI: @Sampfluger88 (`@`-mention the DRI in [#help-gtm-ops](https://fleetdm.slack.com/archives/C08BTMFTUCR))
> - Does the Eventbrite page have an "order form" attached? If so, remove it! « This breaks the flow by adding another required form submission not tied to the `New Attendee Registered` action. Attendee name and email will be returned as "Info Requested".
> - Does the SFDC campaign exists?
> - Is the `Event_key` populated correctly on the corresponding SFDC campaign?


***Purpose***

Create a reliable, repeatable way to associate Eventbrite registrations with the correct Salesforce contact and campaign. Each event has a unique identifier (`event_key`). We store that identifier on the corresponding Salesforce campaign creating a 1:1 relationship between the published event and the Salesforce campaign. 

This approach “connects” Eventbrite to Salesforce campaigns by using the **`Event_key` as the system-of-record key**. Salesforce Campaigns store that key, and Clay uses it to automatically route registrations to the right Campaign and create/update Campaign Members—cleanly, invisibly, and in a way that can later support additional event platforms.


***High-level workflow***

1. A new registration occurs and is captured by Zapier (workflow: [Eventbrite - Event registration » Clay](https://zapier.com/editor/355884186/published)).
2. Zap captures and sends the following info to Clay:
    - `fullName`
    - `firstName`
    - `lastName` 
    - `Email` 
    - `providedNotes`: "`EVENT_NAME` - `EVENT_URL`" 
    - `Event_key`: "Eventbrite-"`EVENT_ID` (This is used to identify the correct Salesforce campaign to add the contact to.)
    - `campaignMemberStatus`: "Registered" « (Hardcoded)
3. Clay (table: [Events - Historical event creation](https://app.clay.com/workspaces/315782/workbooks/wb_0t4mlesfmwB8E6W357B/tables/t_0t90w56wNMpfCnCnfFm/views/gv_0t90w56hCPwZrpWtyC6)) receives the payload.
    - The `Event_key` is used to find the correct campaign.
    - A [historical event](https://fleetdm.com/handbook/finance/gtm-architecture#historical-events-sfdc) gets created with a `relatedCampaign` matching the `Event_key`. Creating a historical event will also create the contact/account if it doesn't already exist.
    - The name and email is used to pull the correct LinkedIn. If a LinkedIn profile is found, Clay updates the following data in Salesforce:
        - Job title
        - Mailing address: (City, State/Province, Country)
        - Primary buying situation « TODO Document
        - Role « TODO Document
    - Sends the following message to the [#help-gitops-workshops](https://fleetdm.slack.com/archives/C0ALY0LJD39) Slack channel.
    
    ```
        NEW GITOPS REGISTRATION
        _*`fullName`*_ signed up for `proviededNotes`

        - CONTACT: 
        _*`fullName`*_ (`finalLinkedInProfile`)
        `CRMLink`

        - ACCOUNT:
        `Rating` - _*`accountName`*_ (`finalLinkedInCompanyUrl`)
    ```


### LinkedIn comments from tracked posts

We track certian social posts from the [LinkedIn company page](https://www.linkedin.com/company/fleetdm/) using the following workflow:
- LinkedIn post URL provided to Clay.
- Clay enriches the data from any reactions or shares.
- Clay sends webhook to webhooks/receive-from-clay.js
- fleetdm.com sends a webhook to Salesforce.
- Salesforce will create/update the contact and account, and creates a "Historical event" for each contact.
- Clay then sends a webhook to Zapier.
- Zapier posts a message to the [_linkedin-comments-from-tracked-posts](https://fleetdm.slack.com/archives/C0AP1FM3ES2).


<img width="1410" height="1174" alt="image" src="https://github.com/user-attachments/assets/da2dccaa-e5ac-4373-9d93-d02b2a1bd8cd" />


## Salesforce

### SFDC access

Fleet uses Okta SSO for Salesforce authentication. All Fleet employees (`@fleetdm.com`) authenticate through Okta — Salesforce credential login is disabled for SSO-enabled profiles. All Fleet employees must login at our custom domain [fleetdm.my.salesforce.com](fleetdm.my.salesforce.com) or by clicking the Salesforce app tile in Okta. For users and accounts that cannot use SSO (e.g., integration users, external collaborators), Fleet has created custom cloned profiles with SSO disabled that must login at [login.salesforce.com](login.salesforce.com).


#### Profiles and when to use them

| Profile | SSO | Who gets this | When to assign |
|:---|:---|:---|:---|
| **Fleet User** | Yes | All `@fleetdm.com` employees (standard users). | Assign to any new Fleet employee who needs Salesforce access. |
| **System Administrator** | Yes | Fleet employees who need admin-level access. | Assign to any new Fleet employee who needs full admin privileges in Salesforce. |
| **externalNonSSOEnabledSystemAdmin** | No | UTTR (integration) users and the Integrations admin account. | Assign to integration/service accounts or external admin users that authenticate with Salesforce credentials instead of Okta. |
| **externalNonSSOEnabledFleetUser** | No | External non-admin users who do not use SSO. | Assign to any external collaborator or non-Fleet user who needs standard (non-admin) Salesforce access without SSO. |

- **Adding an SSO user:** Assign the **Fleet User** profile (or **System Administrator** if they need admin privileges). The user will authenticate via Okta and Salesforce credential login will be disabled.
- **Adding a non-SSO user (e.g., an integration account or external collaborator):** Assign **externalNonSSOEnabledSystemAdmin** for admin-level access or **externalNonSSOEnabledFleetUser** for standard access. These users authenticate with Salesforce credentials directly.


### Campaigns (SFDC)

TODO

#### For event campaigns (SFDC)

- **Event platform** (Picklist) – identifies the source platform
  - Options: `Eventbrite`, `Luma`, etc.

- **External event ID** (Text) – stores the platform-specific event identifier
  - Example: Eventbrite event ID `123456789`

- **Event key** (Formula) – composite key for matching integrations
  - Formula: `"Event platform"&"-"&"External event ID"`
  - Example output: `Eventbrite-123456789`


### Historical events (SFDC)

Historical events (`fleet_website_page_views__c`) is a custom Salesforce object that records timestamped interactions a contact has with Fleet across the website and other channels. Each Historical event record is associated with both a **Contact** and an **Account** in Salesforce, creating a per-contact activity log that the GTM team uses to understand engagement over time.


#### What historical events do

Historical events serve as the single source of truth for tracking how contacts engage with Fleet. Every time a meaningful interaction occurs — whether it's a website page view, a LinkedIn reaction, a newsletter subscription, or a form submission — a Historical event record is created in Salesforce. This gives GTM teams a chronological view of engagement that helps with:

- Measuring psychological progression of contacts and accounts.
- Prioritizing accounts for [research](https://fleetdm.com/handbook/marketing#research-an-account) and outreach.
- Identifying contacts that would benefit from a [POV conversation](https://fleetdm.com/handbook/company/go-to-market-operations#proof-of-value-pov).


#### Historical event types and intent signals

There are two types of Historical event records:

| Event type | Description |
|:---|:---|
| **Website page view** | Logged when a signed-in user visits a page on fleetdm.com. Includes the page URL and, when available, the ad attribution that brought them to the site. |
| **Intent signal** | Logged when a contact takes a specific high-value action. |
| **Warm-up action** | Logged when a Fleetie takes a specific high-value action toward a contact. |

The following intent signals are tracked:

- Followed the Fleet LinkedIn company page
- LinkedIn comment, share, or reaction
- Fleet channel member in MacAdmins Slack or osquery Slack
- Implemented a trial key
- Signed up for a Fleet event
- Registered for a conference
- Engaged with Fleetie at event
- Attended a Fleet happy hour
- Starred, forked, or contributed to the fleetdm/fleet repo on GitHub
- Subscribed to the Fleet newsletter
- Attended a Fleet training course
- Submitted the "Send a message" form
- Scheduled a "Talk to us" or "Let's get you set up" meeting
- Submitted the "GitOps workshop request" form
- Signed up for a fleetdm.com account
- Requested whitepaper download
- Created a quote for a self-service Fleet Premium license


#### How historical events are triggered

Historical event records are created automatically by the Fleet website backend (`website/api/helpers/salesforce/create-historical-event.js`). The helper is called from several code paths:

| Trigger | Code path | Event type |
|:---|:---|:---|
| Signed-in user views a page on fleetdm.com | `website/api/hooks/custom/index.js` | Website page view |
| Clay webhook receives LinkedIn activity data | `website/api/controllers/webhooks/receive-from-clay.js` | Intent signal |
| User subscribes to the Fleet newsletter | `website/api/controllers/create-or-update-one-newsletter-subscription.js` | Intent signal |
| User submits the "Send a message" contact form | `website/api/controllers/deliver-contact-form-message.js` | Intent signal |
| User requests a whitepaper download | `website/api/controllers/deliver-whitepaper-download-request.js` | Intent signal |
| User creates a self-service quote | `website/api/controllers/customers/create-quote.js` | Intent signal |
| User submits the "GitOps workshop request" form | `website/api/controllers/deliver-gitops-workshop-request.js` | Intent signal |
| User signs up for a fleetdm.com account | `website/api/controllers/entrance/signup.js` | Intent signal |

In every case, the website first calls `updateOrCreateContactAndAccount` to ensure the contact and account exist in Salesforce, then calls `createHistoricalEvent` with the returned `salesforceContactId` and `salesforceAccountId`.


#### Historical event fields

| Salesforce field API name | Description |
|:---|:---|
| `Contact__c` | Lookup to the related Contact record. |
| `Account__c` | Lookup to the related Account record. |
| `Event_type__c` | The type of event: "Website page view" or "Intent signal". |
| `Intent_signal__c` | The specific intent signal (only for Intent signal events). |
| `Content__c` | Free-text content associated with the event (e.g. a LinkedIn comment or form message). |
| `Content_url__c` | URL of the content (e.g. a LinkedIn post URL). |
| `Interactor_profile_url__c` | The LinkedIn profile URL of the person who interacted. |
| `Page_URL__c` | The fleetdm.com page URL (only for Website page view events). |
| `Website_visit_reason__c` | Ad attribution string, if the user arrived via an ad within the last 30 minutes. |
| `Related_campaign__c` | Related Salesforce campaign, if applicable. |

> Historical event records are only created in the production environment. When deleting a contact's data (e.g. for a data deletion request), any related Historical event records associated with that contact are also automaticly deleted.




<meta name="maintainedBy" value="sampfluger88">
<meta name="title" value="🚂 GTM architecture">

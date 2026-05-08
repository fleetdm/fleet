# Marketing ops

Drive efficient, scalable pipeline growth by building and optimizing the systems, processes, and campaigns that attract, nurture, and convert high-quality leads into revenue—enabling the sales team to focus on closing while we focus on filling the funnel.


##  Go-to-market attribution

Our go-to-market (GTM) approach is built on a foundation of end-to-end visibility. We want to track touchpoints from first engagement through closed revenue, connecting marketing activity to pipeline and revenue. This means instrumenting our campaigns, content, and channels with consistent attribution, maintaining clean data flow between our marketing automation and CRM systems, and building reporting that ties spend and effort directly to outcomes. The goal isn't data for data's sake—it's to create a feedback loop where we can see what's working, double down on high-performing channels, cut what isn't delivering, and continuously refine our targeting, messaging, and timing. Every campaign we run should make us smarter about the next one.


### Key resources

1. [Conversion rates](https://fleetdm.com/handbook/marketing/marketing-ops#conversion-rates)
2. [GTM Model](https://fleetdm.com/handbook/marketing/marketing-ops#GTM-model)
3. [Attribution framework (aka contact source)](https://fleetdm.com/handbook/marketing/marketing-ops#attribution-framework)  
4. [Unified campaign member status framework](https://fleetdm.com/handbook/marketing/marketing-ops#unified-campaign-member-status-framework)


## Conversion rates

Conversion rates help us to plan, forecast, and improve. There are several key comparisons that we want to understand:

- **Win rate**: From stage X to closed won.  For closed opportunities, this tells us what percentage of opportunities historically will be won for a given stage in the sales cycle.  
- **Stage to win cycle time**: tbd/todo  
- **Stage to stage**:tbd/todo  
- **Stage to stage cycle time**: tbd/todo


## GTM model

We can build a reverse funnel using the conversion rates and an estimated ASP, which will indicate the business demand for top-of-funnel leads/contacts and opportunities in order to attain future revenue targets. 
See our current model in google docs [link](link to google docs) <Tbd/todo>


## Attribution framework



To scale demand generation effectively, we need to have a trusted source of data about what works in generating new leads, opportunities, pipeline, and business. Without a consistent framework, our data is messy, reporting is unreliable, and we cannot confidently measure the ROI of our marketing or sales efforts. This framework solves three core problems:

1. **Inconsistent data**  
2. **Lack of visibility**  
3. **Inaccurate ROI**

This outlines a simple, scalable, and non-negotiable system for tracking all lead-generating activities at Fleet.


### First-touch vs. converting-touch

This framework is **not** just for the Original Lead/Contact Source field. It should be applied to **two separate, critical moments** in the customer journey.


#### 🌎 First-touch: Original contact source

- **What it is:** The "birth certificate" of a contact. It is the very first marketing or sales touch that brought this person into our database.  
- **The rule:** This field is **set once and is locked forever**. It should *never* be overwritten.  
- **It answers:** "Which of our channels are best at generating *net-new names* and filling the top of our funnel?"


#### 🏁 Converting-touch: Opportunity creation source 🟡

- **What it is:** The "final push." It is the specific campaign that caused a known lead or contact to convert into a sales-qualified opportunity (i.e., they booked a demo or engaged with sales).  
- **The rule:** This field is set *at the moment of opportunity creation*.  
- **It answers:** "Which of our channels are best at generating *pipeline and revenue*?"

Example:

- A MacAdmin first discovers FleetDM by attending an OSQuery 101 webinar in Oct 2025 (2025\_10-WH-osquery\_101).   
  - Their First-Touch is Event \> Webinar (Hosted). 

- Six months later, an SDR emails them (2026\_04-SDR-q2\_fintech\_sequence), and they reply to book a demo.   
  - Their Most Recent/Converting-Touch is Prospecting \> SDR Outbound.

This allows us to see that our webinars are great for *finding* leads, and our SDR team is great at *converting* them.


### Attribution hierarchy

Our model is a simple 3-level hierarchy. Every report can be rolled up to Level 1 for an executive summary or drilled down to Level 3 for granular analysis.

| Level | Name | Purpose | Example | Control/Type |
| :---- | :---- | :---- | :---- | :---- |
| **Level 1** | **Source** | The high-level budget bucket or media channel. (Max 6-8) | Event | PickList |
| **Level 2** | **Source detail** | The specific *tactic* or *program* within that source. | Major conference | PickList (variable) Tied to Source |
| **Level 3** | **Campaign** | The specific, unique, and trackable initiative. | 2026\_08-MC-blackhat\_booth | Text Field (naming convention) |


### Source

At the top of the hierarchy, there are 6 “Source” buckets, where all our contacts and new logo opportunities will align.

- **🌳 Organic/web**: All unpaid, inbound traffic and brand-driven interest.
- **🗣️ Word-of-mouth**: All manually tracked, human-to-human recommendations.
- **🗓️ Event**: All in-person or virtual events, sponsored or hosted.
- **💻 Digital**: All paid and owned online media and content.
- **🎯 Prospecting**: All outbound activities initiated by sales or a 3rd-party vendor.
- **🤝 Partner**: All co-marketing and leads generated from formal channel partners.


#### 🌳 Organic/web

For all unpaid, inbound traffic and brand-driven interest.

| Source detail | Code | Campaign examples (always-on) |
| :---- | :---- | :---- |
| Organic search | OS | Default-OS |
| Direct traffic | DT | Default-DT |
| Web referral | WR | Default-WR |
| Organic social | SOC | Default-SOC |


#### 🗣️ Word-of-mouth

For all manually tracked, human-to-human recommendations.

| Source detail | Code | Campaign examples |
| :---- | :---- | :---- |
| Customer referral | CR | Default-CR  |
| Employee referral | ER | Default-ER  |
| Analyst/influencer | AR | AR-gartner\_mention |


#### 🗓️ Event

For all in-person or virtual events, sponsored or hosted.

| Source detail | Code | Campaign examples (discreet) |
| :---- | :---- | :---- |
| Major conference (global, 10k+) | MC | 2026\_08-MC-blackhat |
| Regional conference | RC | 2026\_03-RC-secureworld\_boston |
| Local event / meetup | LE | 2026\_02-LE-osquery\_meetup\_nyc |
| Executive community (Evanta, etc.) | EC | 2026\_01-EC-evanta\_ciso\_summit |
| Field event / sales event (workshop, hosted dinner, HH) | FE | 2026\_04-FE-nyc\_fintech\_dinner |
| Partner event (sponsoring) | PE | 2025\_11-PE-aws\_reinvent |
| Speaking engagement | SE | 2026_06-SE-macadmins\_keynote |
| Webinar (hosted) | WH | 2026\_02-WH-fleet\_v5\_launch |
| Webinar (sponsored) | WS | 2026\_03-WS-darkreading\_webinar |


#### 💻 Digital

For all paid and owned online media and content.

| Source detail | Code | Campaign examples |
| :---- | :---- | :---- |
| Paid search | PS | 2025\_11-PS-google\_brand\_usa |
| Paid social | SO | 2025\_11-SO-linkedin\_video\_ciso |
| Paid media | PM | 2025\_11-PM-riskybiz\_podcast |
| Content syndication & 3rd-party | CS | 2025\_12-CS-techtarget\_survey |
| Email marketing (owned list) | EM | 2025\_11-EM-newsletter\_promo |


#### 🎯 Prospecting

For all outbound activities initiated by sales or a 3rd-party vendor.

| Source detail | Code | Campaign examples |
| :---- | :---- | :---- |
| SDR outbound | SDR | SDR-General\_Prospecting (Always-On) 2025\_11-SDR-q4\_fintech\_sequence (Discreet) |
| AE outbound | AE | AE-General\_Prospecting |
| Meeting Service | MS | 2025\_11-MS-VIB 2026\_01-MS-SageTap |


#### 🤝 Partner

For all co-marketing and leads generated from formal channel partners.

| Source detail | Code | Campaign examples |
| :---- | :---- | :---- |
| Tech partner | TP | 2025\_11-TP-aws\_marketplace |
| Reseller / VAR | RE | RE-General\_Referrals |
| Co-marketing | CM | 2026\_01-CM-crowdstrike\_whitepaper |


### Campaigns

**The golden rule:** Every single lead-generating activity *must* have a unique campaign in the CRM before it launches. 

There are only two types of campaigns:
1. "Always-on" campaigns (continuous)
2. Discrete campaigns (time-based)


#### Discrete campaigns

Discreet campaigns have a specific start, end, and budget (e.g., webinar, trade show, quarterly ad). Use the following naming convention when naming a "Discrete" campaign: 
- **Structure:** YYYY\_MM-\[Code\]-\[Name\]  
  - **YYYY\_MM:** The start month. (e.g., 2026\_02)  
  - **\[Code\]:** The 2-4 letter code from the table above. (e.g., MC, PS, WH)  
  - **\[Name\]:** A short, URL-friendly name. (e.g., blackhat, google\_brand)  
- **Full example:** 2026\_08-MC-blackhat


#### Always-on campaigns

These are generic "buckets" for continuous inbound channels that don't have a start/end date.  They are “Default” campaigns, since they do not have a start or stop date. Use the following naming convention when naming an "Always-on" campaign:
- **Structure:** Default-\[Code\]  
  - **\[Name\]:** Default, Always\_On, or General.
  - **\[Code\]:** The 2-4 letter code.
- **Full example:** Default-OS (for all Organic Search)


## SFDC campaign hierarchy

### Campaign hierarchy

Salesforce campaigns should live inside a parent-child hierarchy that mirrors the attribution framework. This allows us to roll up ROI, pipeline, and engagement at any level — from an individual campaign all the way up to a Source bucket — without building custom reports from scratch.

There are two types of campaigns in Salesforce, determined by the **campaign record type**:
- **Working campaigns:** Traditional campaigns with content and activities associated with them.
- **Parent campaigns:** Buckets that group related working campaigns together.

The campaign record type is the controlling field that determines whether a campaign is a working campaign or a parent campaign.

Use the following list views to navigate campaigns in Salesforce:
1. [Parent campaigns list](https://fleetdm.lightning.force.com/lightning/o/Campaign/list?filterName=Parent_campaigns)
2. [Active working campaigns list](https://fleetdm.lightning.force.com/lightning/o/Campaign/list?filterName=Active_working_campaigns)

#### How the hierarchy maps to attribution

| Hierarchy level | Attribution level | What it represents | Example |
|----------------|-------------------|--------------------|---------|
| L1 — Top parent | Source | The 6 high-level budget buckets | `1_Event` |
| L2 — Sub-parent | Source Detail | The specific tactic or program type within a Source | `2_Field_Event` |
| L3 — Program parent (optional) | — | A recurring initiative that runs multiple times | `3_GitOps_Workshops` |
| Leaf — Individual campaign | Campaign | The specific, trackable activity with campaign members | `2026_03-FE-GitOps_Workshop_Chicago` |

L1, L2, and L3 parent campaigns are structural — they exist only for rollup and never have campaign members directly attached to them. Only leaf campaigns contain campaign members.

#### Parent campaign naming

Parent campaigns (nodes in the tree) use a numerical prefix that indicates their hierarchy level. Leaf campaigns keep their existing naming convention unchanged.

L1 Source parents use the prefix `1_` followed by the Source name: `1_Organic_Web`, `1_Word_of_Mouth`, `1_Event`, `1_Digital`, `1_Prospecting`, `1_Partner`.

L2 Source Detail parents use the prefix `2_` followed by the Source Detail name: `2_Paid_Search`, `2_Field_Event`, `2_Major_Conference`, `2_Webinar`.

L3 Program parents use the prefix `3_` followed by a descriptive name: `3_GitOps_Workshops`.

The numerical prefix makes the hierarchy level immediately visible in any SFDC list view and sorts parent campaigns naturally above leaf campaigns.

#### When to create a program parent

Create an L3 program parent when a tactic is repeated three or more times and you want to see the collective impact separately from other campaigns in the same Source Detail. For example, if we run six GitOps Workshops under Field Event (FE), a `3_` program parent lets us see the total pipeline from GitOps Workshops without mixing in happy hours or dinners.

If a tactic only runs once or twice, keep it directly under the `2_` Source Detail parent — no program parent needed.

#### Where always-on campaigns live

Always-on "Default" campaigns sit inside the hierarchy under their corresponding `2_` Source Detail parent, just like discrete campaigns. For example, `Default-OS-Organic` lives under `2_Organic_Search`, which lives under `1_Organic_Web`. This ensures that rollup numbers at every level are complete.

To isolate discrete campaigns for time-bound analysis, filter on campaign name — anything starting with `Default-` is always-on.

#### Fiscal year

Fiscal year is not a layer in the campaign hierarchy. Use the `Fiscal_Year__c` formula field on the Campaign record (derived from Start Date) to filter any report by fiscal year. The YYYY_MM prefix in campaign names also provides a natural date anchor for sorting and filtering.

#### Adding a new campaign to the hierarchy

When creating a new campaign in Salesforce:

1. Identify the Source Detail code from the attribution framework table above (e.g., FE for Field Event).
2. Find the corresponding `2_` parent campaign (e.g., `2_Field_Event`).
3. If the campaign belongs to an existing program series, set the parent to the `3_` program parent instead (e.g., `3_GitOps_Workshops`).
4. If no `2_` parent exists yet for that Source Detail, create one under the correct `1_` Source parent first.
5. Set the Parent Campaign field on your new campaign before launch.

#### Example

A new GitOps Workshop in Nashville in May 2026:

```
1_Event
└── 2_Field_Event
    └── 3_GitOps_Workshops
        └── 2026_05-FE-GitOps_Workshop_Nashville   ← new campaign here
```

The workshop's pipeline will automatically roll up into the GitOps Workshops total, the Field Event total, and the overall Event total.



## Unified campaign member status framework

To accurately measure marketing ROI and attribution, we must standardize how we track prospect progression through our campaigns. This framework establishes a *unified status hierarchy* for Salesforce campaigns. 

**Key objectives:**
1. **Standardization:** Use the same language across all campaign types.  
2. **Attribution:** Ensure only meaningful interactions trigger attribution models.  
3. **Social integration:** Capture top-of-funnel social intent without inflating pipeline metrics.


### Unified hierarchy

All campaigns must utilize the following status values. Custom statuses outside this list are to be avoided.

| Status value | Responded? | Funnel stage | Psystage | Definition |
| ----- | ----- | ----- | ----- | ----- |
| **Targeted** | No | Unaware | 1 \- Unaware | The individual is on a list or in an audience segment but has taken no action. |
| **Sent** | No | Awareness | 2 \- Aware | The email was sent, the ad was displayed, or the post was published. |
| **Interacted** | **Yes** | Interest | **3 \- Intrigued**  | **(Light Touch)** Passive engagement. They clicked a link, liked a post, or visited a high-value page, but **did not exchange contact** info. |
| **Registered** | **Yes** | Consideration | **3 \- Intrigued** | **(Conversion)** The individual explicitly exchanged data for access (Form Fill, Sign Up, RSVP). |
| **Attended** | **Yes** | Evaluation | 3 \- Intrigued | The individual showed up to a synchronous event (Booth Scan, Webinar, Live Event, Dinner). |
| **Engaged** | **Yes** | Intent | **4 \- Has use case** | **(Deep Interaction)** High-effort engagement. They asked a question, made a meaningful comment, or engaged in a conversation. Hot Lead from Event |
| **Meeting Requested** | **Yes** | Purchase | 5 \- Personally confident | The individual explicitly requested a sales contact or a demo. |


### Operational definitions by channel

#### Social media and content

*Goal: Distinguish between vanity metrics (Likes) and true leads.*

- **Interacted:** User "Likes" a post, "Follows" the page, or clicks a link to ungated content.  
- **Engaged:** User comments on a post, shares/retweets with their own commentary, or sends a Direct Message (DM).  
- **Registered:** User fills out a specific lead gen form (e.g., LinkedIn lead gen) or clicks through to a landing page and converts.

#### Webinars and virtual events

*Goal: Track the drop-off between sign-up and attendance.*

- **Interacted:** Clicked the invitation link but did not complete registration.  
- **Registered:** Completed the registration form.  
- **Attended:** Logged into the webinar platform for \>1 minute.  
- **Engaged:** Attended **AND** asked a question in Q\&A, answered a poll, or stayed for the entire duration.

#### Physical events 

*Goal: Differentiate between booth traffic and serious conversations.*

- **Interacted:** Visited the booth, took swag, a COLD LEAD  
- **Registered:** RSVP’d to the event (if hosted by us) or pre-booked a meeting.  
- **Attended:** Badge scanned at booth.  
- **Engaged:** HOT lead. Had a meaningful conversation with a rep; notes added to CRM.  
- **Meeting Requested**


#### Meeting service

*Goal: Qualify and move to become an opportunity*

- **Targeted:** A prospect is in the pool of potential targets  
- **Interacted:** Introductory meeting requested/ scheduled  
- **Attended:** Introductory meeting completed.  
- **Meeting Requested:** The prospect has asked for a follow-up engagement/discussion 


#### Email marketing

*Goal: Move beyond "Open Rates."*

- **Sent:** Email delivered.  
- **Interacted:** Clicked a link in the email (click-through).  
- **Registered:** Clicked a link and filled out the resulting form.  
- **Engaged:** Replied to the email directly.

#### Website chat (qualified)
- **Interacted:** We chatted and learned about the prospect
- **Engaged:** We chatted long enough to offer a meeting 
- **Meeting Requested:** The prospect has booked a meeting

## Contact source (Lead source)
At Fleet, we also keep track of the specific form or activity that a contact completed when they were created. This way we keep track of "Where" they came from (the attribution framework), but also have data about what they did.  Historically, we've had the field *Contact source*, which effectively told us what form or activity a person did.  This is good data, and works alongside the overall attribution framework.

Here are the values for the contact source:


| **Contact Source Value** | **Category** | **Status** | **Definition** |
| :---- | :---- | :---- | :---- |
| Website \- Sign up | Website | Existing | Contact created an account/signed up for the Fleet platform. |
| Website \- Contact forms \- Demo | Website | Existing | Contact requested a standard demo via the website. |
| Website \- Contact forms \- Demo \- ICP | Website | Existing | Contact requested a demo and was routed/flagged as an Ideal Customer Profile. |
| Website \- Contact forms | Website | Existing | Contact submitted a general inquiry via the website. |
| Website \- Gated document | Website | NEW | Contact filled out a form to download a whitepaper, report, or guide. |
| Website \- Newsletter | Website | Existing | Contact explicitly subscribed to the Fleet blog or newsletter. |
| Website \- Swag request | Website | Existing | Contact filled out a form specifically to request Fleet merchandise. |
| Website \- GitOps | Website | Existing | Contact converted via a specific GitOps-related form or landing page flow. |
| Website \- Chat | Website | Existing | Contact engaged and provided their email via the website chatbot. |
| Website \- Partner sign up | Website | NEW | Contact submitted a form to apply for or join the Fleet partner program. |
| Webinar | Events | NEW | Contact registered for or attended a webinar (hosted by Fleet or a 3rd-party). Note: The specific host/campaign is captured in the 3-tier attribution. |
| Event | Events | Existing | Contact was scanned, uploaded, or registered from a live physical or virtual event. |
| LinkedIn \- Native lead form | Third-Party | NEW | Contact submitted their info directly inside LinkedIn via a Document Ad or Lead Gen form. |
| Content syndication | Third-Party | NEW | Contact info was acquired via a 3rd-party vendor promoting Fleet's content. |
| Partner \- Deal registration | Third-Party | NEW | Contact was formally registered as a lead by an authorized partner/reseller. |
| GitHub \- Stared fleetdm/fleet | Community | Existing | Contact starred the Fleet repository. |
| GitHub \- Forked fleetdm/fleet | Community | Existing | Contact forked the Fleet repository. |
| GitHub \- Contributed to fleetdm/fleet | Community | Existing | Contact made a code/documentation contribution to the Fleet repository. |
| LinkedIn \- Liked the LinkedIn company page | Social | Existing | Contact followed or liked the official Fleet LinkedIn page. |
| LinkedIn \- Reaction | Social | Existing | Contact reacted (like, celebrate, etc.) to a Fleet post. |
| LinkedIn \- Comment | Social | Existing | Contact commented on a Fleet post. |
| LinkedIn \- Share | Social | Existing | Contact shared a Fleet post. |
| Prospecting \- AE | Outbound | Existing | Contact was sourced directly via outbound efforts by an Account Executive. |
| Prospecting \- Specialist | Outbound | Existing | Contact was sourced directly via outbound efforts by a Sales Specialist. |
| Prospecting \- Meeting service | Outbound | Existing | Contact was sourced/booked via an outsourced meeting-setting agency. |
| Dripify \- AE | Outbound | Existing | Contact was sourced via Dripify automation by an AE. |
| Dripify \- Specialist | Outbound | Existing | Contact was sourced via Dripify automation by a Specialist. |
| Attended a call with Fleet | Outbound | Existing | Contact was added to the system after attending a calendar invite/call with the team. |



## 📧 Contact marketability & compliance

At Fleet, we maintain a strict separation between contacts we *can* legally email (Marketable) and those we are prospecting cold (Non-Marketable). This ensures we honor opt-outs, protect our domain reputation, and comply with GDPR/CAN-SPAM.

We do not rely on "implied" logic (e.g., "If they have an email, email them"). Instead, we use a dedicated status field on the Contact object to act as the single source of truth.

### The "marketing status" definitions

The `Marketing_Status__c` picklist is the master switch for a contact's eligibility. Every contact in Salesforce must fall into one of the following buckets:

| Status value | Definition | Can marketing email? | Can sales email? |
| --- | --- | --- | --- |
| **Marketable** | The contact has **explicitly opted in** (e.g., Trial signup, Webinar reg, Newsletter form) or is an active customer with marketing consent. | ✅ **Yes** | ✅ **Yes** |
| **Transactional Only** | The contact is a user or customer (e.g., Fleet Free tier) but has **not** opted into marketing. They receive *only* critical system alerts, billing, or security notices. | ❌ **No** | ✅ **Yes** (Contextual) |
| **Cold / Prospect** | The contact was identified via enrichment (Clay, Snitcher, ZoomInfo) or outbound sourcing. We have a valid email, but they have **no prior relationship** with us. | ❌ **No** (Risk of Spam Trap) | ✅ **Yes** (1:1 Outbound Only) |
| **Unsubscribed** | The contact has clicked "Unsubscribe" or explicitly asked to be removed from lists. This is a **legal compliance** flag. | 🛑 **NEVER** | 🛑 **NEVER** |
| **Bounced / Invalid** | The email address is known to be dead or a hard bounce. | 🛑 **NEVER** | 🛑 **NEVER** |
| **Do Not Contact** | The "Nuclear Option." Used for competitors, angry prospects, or disqualifications. Blocks all automated and manual outreach. | 🛑 **NEVER** | 🛑 **NEVER** |

### Data structure

To support this status, we use three additional fields to track the "Who, When, and Why."

| Field | API Name | Purpose |
| --- | --- | --- |
| **Status Date** | `Marketing_Status_Last_Updated__c` | **The Timestamp.** <br>

<br>Records exactly *when* the status changed. Required for compliance auditing (e.g., "Proof of Consent"). |
| **Status Reason** | `Marketing_Status_Detail__c` | **The Audit Trail.** <br>

<br>Explains *why* the status changed. <br>

<br>*Examples:* "Inbound Demo Request," "Clay Enrichment Import," "User Clicked Unsubscribe." |
| **Is Marketable?** | `Is_Marketable__c` | **The Integration Gatekeeper.** <br>

<br>A generic checkbox formula (`TRUE` only if Status = `Marketable`). Our marketing platform (ActiveCampaign) only syncs contacts where this is Checked. |

### The "Traffic Cop" automation

We generally do not manually update these fields. A Salesforce Flow acts as a "Traffic Cop" to standardize data entering from different sources.

**1. Inbound sources (marketable)**

- **Triggers:** Website forms, Trial signups, Event badge scans.
- **Result:** Status  `Marketable`.
- **Reason Stamped:** "Inbound Form Fill: [Form Name]"

**2. Outbound/enrichment sources (cold)**

- **Triggers:** Clay enrichment, Snitcher identification, ZoomInfo imports.
- **Result:** Status  `Cold/Prospect`.
- **Reason stamped:** "Enriched via Clay - Cold"
- **Note:** These contacts are synced to sales tools (Outreach/Apollo) for 1:1 prospecting but are **excluded** from marketing newsletters.

**3. Opt-Outs (Unsubscribed)**

- **Triggers:** User clicks "Unsubscribe" in email, or `HasOptedOutOfEmail` is checked in SFDC.
- **Result:** Status  `Unsubscribed`.
- **Rule:** This is permanent. A "Cold" lead can become "Marketable" (by filling a form), but an "Unsubscribed" contact is locked unless they manually re-subscribe.

### Why this matters

- **Compliance:** We must be able to prove *when* and *how* someone consented to receive emails.
- **Deliverability:** Sending marketing blasts to "Cold" data (Clay lists) ruins domain reputation. We keep those lists separate for low-volume, high-relevance sales outreach only.
- **Debugging:** If a VIP prospect stops receiving emails, the `Status Reason` tells us if it was a system error (Bounce) or human error (Sales marked "Do Not Contact").


## ActiveCampaign

Fleet uses ActiveCampaign as its marketing automation platform for email marketing, contact lifecycle management, lead nurturing, and segmentation. ActiveCampaign is integrated with Salesforce (SFDC) as the system of record; key contact fields sync from SFDC into ActiveCampaign, and lifecycle transitions driven by sales activity in SFDC are reflected in ActiveCampaign automations.

### Lists

ActiveCampaign lists represent permission groups — the type of communication a contact has consented to receive. Fleet maintains four lists:

| List | Channel | Who belongs here | Purpose |
|---|---|---|---|
| Marketing Contacts | Email | All opted-in prospects and trial users | All marketing emails: nurture sequences, product announcements, event follow-ups, and campaigns |
| Newsletter | Email | Contacts who have explicitly opted into the Fleet newsletter | Newsletter sends only. A contact can be on this list without being on Marketing Contacts. |
| Master SMS List | SMS | Contacts who have opted into SMS communications | SMS campaigns and notifications |
| Customers | Email | Active Fleet customers on a paid plan | Customer-specific communications: onboarding, product updates, and renewals |

Unsubscribing from a list removes the contact from all automations tied to that list. Contacts may appear on multiple lists (e.g., a customer who also receives the newsletter).

### Segmentation

Segmentation in ActiveCampaign is driven by two mechanisms: **contact fields** for stable attributes and **tags** for dynamic, behavioral signals.

#### Contact fields

The following fields are available on every contact record. Fields in the Attribution group are set automatically and should not be manually edited.

**General Details**

| Field | Type | Personalization tag | Notes |
|---|---|---|---|
| First Name | Text | `%FIRSTNAME%` | |
| Last Name | Text | `%LASTNAME%` | |
| Email | Text | `%EMAIL%` | |
| Phone | Text | `%PHONE%` | |
| Account | Text | `%ACCT_NAME%` | Company name, synced from SFDC |
| Job Title | Text | `%CONTACT_JOBTITLE%` | |
| LinkedInURL | Text | `%LINKEDINURL%` | |
| Role | Dropdown | `%ROLE%` | Synced from SFDC. Values: 🧝 Niche individual contributor, 🦌 Program owner, 🧑‍🎄 Leadership, ⛄️ Non-prospect |
| Primary buying situation | Dropdown | `%PRIMARY_BUYING_SITU%` | Synced from SFDC |
| Contact Status | Dropdown | `%CONTACT_STATUS%` | Synced from SFDC |
| Contact stage | Dropdown | `%CONTACT_STAGE%` | Current lifecycle stage. Synced from SFDC and updated by AC automations. See lifecycle stages below. |
| Marketing Email | Dropdown | `%MARKETING_EMAIL%` | Gives sales the ability to block emails for a contact |
| State | Text | `%STATE%` | |
| Country | Text | `%COUNTRY%` | |
| GCLID | Text | `%GCLID%` | Google Click ID, captured from paid search landing pages |

**Attribution**

Attribution fields map directly to Fleet's [attribution framework](#attribution-framework). First-touch fields capture the original source when a contact enters the database and are never overwritten. Most recent fields capture the last known touch and are updated at opportunity creation.

| Field | Type | Personalization tag | Maps to |
|---|---|---|---|
| Source channel | Dropdown | `%SOURCE_CHANNEL%` | First-touch → Level 1 (Source bucket) |
| Source channel detail | Dropdown | `%SOURCE_CHANNEL_DET%` | First-touch → Level 2 (Source detail) |
| Source campaign | Text | `%SOURCE_CAMPAIGN%` | First-touch → Level 3 (Campaign code) |
| Most recent channel | Dropdown | `%MOST_RECENT_CHANNE%` | Converting-touch → Level 1 |
| Most recent channel detail | Dropdown | `%MOST_RECENT_CHANNE%` | Converting-touch → Level 2 |
| Most recent campaign | Text | `%MOST_RECENT_CAMPAI%` | Converting-touch → Level 3 (Campaign code) |
| Contact source | Dropdown | `%CONTACT_SOURCE%` | Summary source field for reporting |

#### Tags

Tags handle behavioral and automation state signals that change over time. They follow a `namespace: value` naming convention and are used to enroll, pause, and exit contacts from automations.

| Tag | Category | Description |
|---|---|---|
| `ls: prospect` | Lifecycle Stage | Opted-in contact with no qualification yet. Starting point for all contacts regardless of source. |
| `ls: mql` | Lifecycle Stage | Marketing Qualified Lead. Right-fit demographics/firmographics plus minor intent signal. Qualifies for increased marketing investment. |
| `ls: srl` | Lifecycle Stage | Sales Ready Lead. MQL threshold met plus sufficient intent to hand off. Triggers sales notification and pauses marketing nurture. |
| `ls: sal` | Lifecycle Stage | Sales Accepted Lead. Sales has accepted and is actively working the contact. SFDC is system of record from this point. |
| `ls: sql` | Lifecycle Stage | Sales Qualified Lead. Sales has met with the contact and is moving forward toward a deal. |
| `ls: customer` | Lifecycle Stage | Has purchased Fleet. Contact is added to the Customers list. |
| `ls: churned` | Lifecycle Stage | Former customer who has cancelled or not renewed. |
| `ls: non-prospect` | Lifecycle Stage | In the database but will never enter the funnel (press, media, analysts, students, community members). Excluded from all nurture automations. |
| `interest: mdm` | Interest | Has shown interest in Fleet's MDM / device management use case. |
| `interest: vuln-management` | Interest | Has shown interest in Fleet's vulnerability management use case. |
| `interest: compliance` | Interest | Has shown interest in Fleet's compliance use case. |
| `interest: osquery` | Interest | Has shown interest in Fleet as an osquery management platform. |
| `nurture: enrolled` | Nurture State | Currently active in a marketing nurture sequence. |
| `nurture: completed` | Nurture State | Finished a nurture sequence without advancing to the next lifecycle stage. |
| `nurture: paused` | Nurture State | Sales is actively working this contact. All marketing sequences are suppressed. |
| `demo: requested` | Demo | Contact has requested a demo. |
| `demo: completed` | Demo | Contact has completed a demo with the sales team. |
| `engaged: hot` | Engagement | 3 or more email opens or clicks in the last 30 days. |
| `engaged: cold` | Engagement | No email opens in 90 or more days. Candidate for re-engagement sequence or suppression. |

`ls:` tags are mutually exclusive — when a contact advances to a new lifecycle stage, the previous `ls:` tag must be removed in the same automation step. The `ls:` tag and the **Contact stage** field should always be kept in sync; any automation that updates one must update the other.

### Email marketing

Fleet uses ActiveCampaign for all owned-list email marketing. This includes the newsletter, product announcements, event follow-ups, and lead nurture sequences.

All campaign names in ActiveCampaign must follow the Level 3 attribution naming convention so that email-driven conversions are correctly attributed in SFDC:

```
YYYY_MM-EM-description
```

For example: `2026_04-EM-trial_nurture_wk1` or `2026_03-EM-q1_newsletter`.

This maps to the **Digital > Email marketing (owned list)** source detail in the attribution framework. When a contact clicks through an email and subsequently books a demo, the ActiveCampaign campaign name is passed to SFDC as the converting-touch via the **Most recent campaign** field.

Newsletter sends are targeted to the **Newsletter** list. All other marketing campaigns target the **Marketing Contacts** list, or a segment within it.

### Opt-in and opt-out

Fleet uses an **opt-in** model for all marketing communications.

- Contacts must explicitly consent before being added to the Marketing Contacts or Newsletter lists. Consent is captured at the point of form submission (newsletter sign-up, content downloads, event registration) or when a contact responds to an SDR or AE outreach.
- Contacts collected at events (e.g., badge scans) are considered opted-in at the point of scanning and are added to Marketing Contacts with `ls: prospect`.
- Contacts with Role = ⛄️ Non-prospect are tagged `ls: non-prospect` and excluded from marketing sends even if they are on a marketing list.
- The **Marketing Email** field is used to give the sales team the ability to signal to Marketing(ActiveCampaign) that they want to STOP marketing from emailing a contact.  The defualt value is **"no restrictions"**, the two optional values are **"Block Nurture Email""** and **"Block All Email"**
- Unsubscribing removes a contact from the relevant list and halts all active automations tied to it. Unsubscribed contacts should not be re-subscribed without explicit re-consent.
- Transactional and product emails (e.g., billing notifications, security alerts) are not managed in ActiveCampaign and are not subject to marketing list opt-in requirements.

### Automation

ActiveCampaign automations manage lifecycle progression, nurture enrollment, and sales handoff. The following rules govern all automations.

**Lifecycle updates are always paired.** Any automation that advances a lifecycle stage must simultaneously: (1) add the new `ls:` tag, (2) remove the previous `ls:` tag, and (3) update the **Contact stage** field to match. These three actions happen in a single automation step.

**Marketing defers to sales at SRL.** When a contact reaches `ls: srl`, ActiveCampaign notifies sales and sets `nurture: paused`. All nurture sequences include an exclusion condition for `nurture: paused`. ActiveCampaign does not set `ls: sal` or `ls: sql` — those transitions are driven by SFDC and synced back into ActiveCampaign.

**Sales rejection handling.** If sales declines an SRL (wrong timing, poor fit, incomplete data), the contact is returned to `ls: mql`, `nurture: paused` is removed, and the contact re-enters the appropriate nurture sequence based on their `interest:` tags.

**DRAFT**
| Trigger | Actions |
|---|---|
| Contact added to Marketing Contacts | Add `ls: prospect`, update Contact stage, enroll in welcome sequence |
| Role syncs as ⛄️ Non-prospect | Add `ls: non-prospect`, remove active `ls:` tag, exit all nurture sequences |
| Contact meets MQL criteria | Remove `ls: prospect`, add `ls: mql`, update Contact stage, enroll in MQL nurture sequence |
| Contact meets SRL criteria | Remove `ls: mql`, add `ls: srl`, update Contact stage, add `nurture: paused`, notify SDR |
| SFDC sync: SAL | Remove `ls: srl`, add `ls: sal`, update Contact stage |
| SFDC sync: SQL | Remove `ls: sal`, add `ls: sql`, update Contact stage |
| SFDC sync: Closed Won | Remove `ls: sql`, add `ls: customer`, update Contact stage, add to Customers list |
| SFDC sync: Churned | Add `ls: churned`, update Contact stage |
| Demo booked | Add `demo: requested` |
| Demo completed (SFDC sync) | Add `demo: completed` |
| No email opens in 90 days | Add `engaged: cold`, trigger re-engagement sequence |
| 3+ opens or clicks in 30 days | Add `engaged: hot` |


## 🐰 Video hosting

### Why do we host videos on a service other than YouTube?

We use a dedicated video hosting platform instead of YouTube for several important reasons:

- **Higher quality** — Videos are delivered at higher fidelity without compression trade-offs.
- **No ads** — Viewers are never interrupted by pre-roll or mid-roll advertisements.
- **Control over content** — We maintain full ownership and control over how our videos are presented and distributed.

### Platform

We use [Bunny.net](https://dash.bunny.net/stream) for video hosting. Credentials are stored in 1Password.

### Uploading a video

1. **Rename the video file** so it is easy to identify. Use the format: `YYYY-MM-title` as a prefix to the video's technical filename (e.g., `2026-04-fleet-webinar-mdm-deep-dive.mp4`).
2. Go to [Bunny.net Stream](https://dash.bunny.net/stream).
3. Select **Stream** and then the appropriate **Video Library** (e.g., `FleetWebinars`).
4. **Upload** the video.
5. **Edit** the video details:
   - Set the **Video Title**.
   - Update the **Metadata** with the title, speakers, and abstract.
6. **Set the desired thumbnail** — take a screenshot of the start of the video and upload it as the thumbnail image.

> **Note for webinars:** You can append a `t=xxs` parameter to the embed code URL to make the video start at a specific timestamp (e.g., `t=90s` to start at 1 minute 30 seconds). This parameter is **not** saved in Bunny.net — it must be added manually each time you use the embed code.

## Virtual persona for email automation

### What it is

We use a virtual team member — **"Grace"** — as the sender identity for our automated email campaigns, nurture sequences, and lifecycle communications. Grace has a name, a headshot, and a consistent voice, but she is not a real employee. She is a purpose-built persona that represents our marketing team.

People engage with people, not logos. Emails from a named individual consistently outperform emails sent from a brand name or a generic address like `marketing@` or `no-reply@`. A virtual persona gives us the warmth and approachability of a personal sender without the operational problems that come with tying automation to a real employee.

- Turnover risk: When a real person is the face of automated email, their departure creates a jarring experience for recipients and a scramble to update templates, signatures, and sender addresses across every platform.
- Scalability: No single employee can realistically "own" the relationship with every lead and contact in the database. A persona can.
- Consistency: A virtual identity stays on-brand across every touchpoint — tone, title, photo, and signature never drift.
- Privacy for the team: Real employees don't have their name and likeness attached to thousands of cold or automated emails they didn't personally write.

### Who is Grace?

Grace is named after pioneering women in science and technology, grounding the persona in values we admire — curiosity, precision, and breaking new ground.

| **Persona name**   | **Named after**                                                                 |
|----------------|-----------------------------------------------------------------------------|
| **Grace West**   | [Adm. **Grace** Hopper](https://en.wikipedia.org/wiki/Grace_Hopper) (She was a pioneer of computer programming. Hopper was the first to devise the theory of machine-independent programming languages) and </br>[Gladys **West**](https://en.wikipedia.org/wiki/Gladys_West) (She was known for her contributions to mathematical modeling of the shape of the Earth, and her work on the development of satellite geodesy models, which were later incorporated into the Global Positioning System (GPS) |


What grace is not:

- She is **not** a chatbot or AI assistant. She is a sender identity for outbound email.
- She is **not** used to deceive. We disclose her nature in every message.
- She is **not** a replacement for real human interaction. When a recipient replies, a real team member responds.


### How it works in practice

- **Sender name and address:** Emails come from Grace west with a dedicated email address (e.g., `grace.west at company.com`).
- Headshot: Use an AI-generated or stock portrait that looks professional and approachable. Keep it consistent across all channels.
- Title: Something credible but not senior enough to create false expectations — e.g., *virtual Marketing Assistnat*
- Voice: Friendly, helpful, knowledgeable. Grace writes the way a sharp colleague would, not the way a press release reads.
- Transparency: Every automated email includes a brief disclaimer identifying Grace as a virtual team member. Honesty builds trust; deception erodes it.

Example disclaimers:
> *P.S. Full transparency — Grace is a virtual member of our team who helps us stay in touch. Hit reply and a real human will be on the other end.*

Alternative
> *Grace is our virtual team member, named after trailblazing women in science and tech. She's not a real person, but every reply goes straight to one.*


### Set up and configuration

Fleet uses a virtual persona — Grace West (gracewest at fleetdm.com) — as the sender for ActiveCampaign automated marketing emails. Using a realistic-looking sender improves open rates and engagement.

**How it's set up:** gracewest at fleetdm.com is a Google Group, not a licensed Gmail user. This avoids license fees and SSO/Okta complications, and allows multiple marketing team members to send as Grace when needed.

**Who has access:** Members of the marketing team are added to the Google Group with email delivery on. Anything sent to gracewest@fleetdm.com lands in their inboxes, and replies route back to the group rather than to individuals.

**To send as Grace from your own Gmail:**
1. In Gmail, click the gear icon → **See all settings**.
2. Go to the **Accounts and Import** tab.
3. Under **Send mail as**, click **Add another email address**.
4. Set **Name:** Grace West, **Email:** gracewest at fleetdm.com.
5. Uncheck "Treat as an alias" — this ensures replies from prospects route to the Google Group rather than your personal inbox.
6. Click **Next Step** → **Send Verification**.
7. The verification email lands in your inbox (since you're a group member). Click the link or copy the code.

Once verified, the **From** dropdown in Gmail's compose window lets you switch to "Grace West <gracewest at fleetdm.com>" when sending.

<meta name="maintainedBy" value="johnjeremiah">
<meta name="title" value="🫧 Marketing ops">

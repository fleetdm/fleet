# Demand gen/marketing operations

Drive efficient, scalable pipeline growth by building and optimizing the systems, processes, and campaigns that attract, nurture, and convert high-quality leads into revenue—enabling the sales team to focus on closing while we focus on filling the funnel.

##  Go to market model

Our go-to-market approach is built on a foundation of end-to-end visibility. We want to track touchpoints from first engagement through closed revenue, connecting marketing activity to pipeline and revenue. This means instrumenting our campaigns, content, and channels with consistent attribution, maintaining clean data flow between our marketing automation and CRM systems, and building reporting that ties spend and effort directly to outcomes. The goal isn't data for data's sake—it's to create a feedback loop where we can see what's working, double down on high-performing channels, cut what isn't delivering, and continuously refine our targeting, messaging, and timing. Every campaign we run should make us smarter about the next one.

### Key processes/resources (see below)

1. [Conversion rates](#Conversion rates)
2. [GTM Model](#GTM-model)
2. [Attribution framework (aka Lead Source)](#Attribution-framework-(aka-Lead-Source))  
3. Unified Campaign member status framework  

## Conversion rates

Conversion rates help us to plan, forecast, and improve.    
There are several key comparisons that we want to understand:

- **Win Rate**: From Stage X to Closed Won.  For closed opportunities, this tells us what percentage of opportunities historically will be won for a given stage in the sales cycle.  
- **Stage to win cyce time**: tbd/todo  
- **Stage to Stage**:tbd/todo  
- **Stage to stage cycle time**: tbd/todo

## GTM model
We can build a reverse funnel hsing the conversion rates and an estimated ASP which will indicate the business demand for top of funnel leads/contacts, and opportunities in order to attain future revenue targets. 
See our current model in google docs [link](link to google docs)

## Attribution framework (aka Lead Source)

### **1\. Introduction**

To scale demand generation effectively, we need to have a trusted source of data about what works in generating new leads, opportunities, pipeline, and business. 

Without a consistent framework, our data is messy, reporting is unreliable, and we cannot confidently measure the ROI of our marketing or sales efforts.

This framework solves three core problems:

1. **Inconsistent data**  
2. **Lack of visibility**  
3. **Inaccurate ROI**

This outlines a simple, scalable, and non-negotiable system for tracking all lead-generating activities at Fleet.

### **First-touch  vs. converting-touch**

This framework is **not** just for the Original Lead/Contact Source field.   
It should be applied to **two separate, critical moments** in the customer journey.

#### **🌎 First-touch: Original lead/contact source  			🟢**

* **What it is:** The "birth certificate" of a contact. It is the very first marketing or sales touch that brought this person into our database.  
* **The rule:** This field is **set once and is locked forever**. It should *never* be overwritten.  
* **It answers:** "Which of our channels are best at generating *net-new names* and filling the top of our funnel?"
#### **🏁 Most recent/converting-touch: Opportunity creation source 🟡**

* **What it is:** The "final push." It is the specific campaign that caused a known lead or contact to convert into a sales-qualified opportunity (i.e., they booked a demo or engaged with sales).  
* **The rule:** This field is set *at the moment of opportunity creation*.  
* **It answers:** "Which of our channels are best at generating *pipeline and revenue*?"

Example:

- A MacAdmin first discovers FleetDM by attending an OSQuery 101 webinar in Oct 2025 (2025\_10-WH-osquery\_101).   
  - Their First-Touch is Event \> Webinar (Hosted). 

- Six months later, an SDR emails them (2026\_04-SDR-q2\_fintech\_sequence), and they reply to book a demo.   
  - Their Most Recent/Converting-Touch is Prospecting \> SDR Outbound.

This allows us to see that our webinars are great for *finding* leads, and our SDR team is great at *converting* them.

### **4\. 3-tier attribution hierarchy**

Our model is a simple 3-level hierarchy. Every report can be rolled up to Level 1 for an executive summary or drilled down to Level 3 for granular analysis.

| Level | Name | Purpose | Example | Control/Type |
| :---- | :---- | :---- | :---- | :---- |
| **Level 1** | **Source** | The high-level budget bucket or media channel. (Max 6-8) | Event | PickList |
| **Level 2** | **Source detail** | The specific *tactic* or *program* within that source. | Major conference | PickList (variable) Tied to Source |
| **Level 3** | **Campaign** | The specific, unique, and trackable initiative. | 2026\_08-MC-blackhat\_booth | Text Field (naming convention) |

---

### 

### **5\. Level 1:  The main buckets:  “SOURCES.”**

At the top of the hierarchy, there are 6 “Source” buckets, where all our leads and new logo opportunities will align.

**🌳 Organic/web**	*All unpaid, inbound traffic and brand-driven interest.*

**🗣️  Word-of-mouth** 	*All manually tracked, human-to-human recommendations.*

**🗓️ Event**		*All in-person or virtual events, sponsored or hosted.*

**💻 Digital**		*All paid and owned online media and content.*	  
			*(Need to think about difference between LinkedIn, GitHub, Web)*

**🎯 Prospecting** 	*All outbound activities initiated by sales or a 3rd-party vendor.*

**🤝 Partner**		*All co-marketing and leads generated from formal channel partners.*

### **6\. The detailed attribution model (level 2 and 3 of the model)**

Here is the complete, definitive list of all Sources and Source Details.

#### **🌳 Source: Organic/Web**

*For all unpaid, inbound traffic and brand-driven interest.*

| Level 2: Source detail | Code | Level 3: Campaign examples (always-on) |
| :---- | :---- | :---- |
| Organic search | OS | OS-Default |
| Direct traffic | DT | DT-Default |
| Web referral | WR | WR-Default |
| Organic social | SOC | SOC-Default |

#### 

#### **🗣️ Source: Word-of-mouth**

*For all manually tracked, human-to-human recommendations.*

| Level 2: Source detail | Code | Level 3: Campaign examples |
| :---- | :---- | :---- |
| Customer referral | CR | CR-Default  |
| Employee referral | ER | ER-Default  |
| Analyst/influencer | AR | AR-gartner\_mention |

#### **🗓️ Source: Event**

*For all in-person or virtual events, sponsored or hosted.*

| Level 2: Source detail | Code | Level 3: Campaign examples (discreet) |
| :---- | :---- | :---- |
| Major conference (global, 10k+) | MC | 2026\_08-MC-blackhat\_booth\_scans |
| Regional conference | RC | 2026\_03-RC-secureworld\_boston |
| Local event / meetup | LE | 2026\_02-LE-osquery\_meetup\_nyc |
| Executive community (Evanta, etc.) | EC | 2026\_01-EC-evanta\_ciso\_summit |
| Field event / sales event (hosted dinner, HH) | FE | 2026\_04-FE-nyc\_fintech\_dinner |
| Partner event (sponsoring) | PE | 2025\_11-PE-aws\_reinvent\_booth |
| Webinar (hosted) | WH | 2026\_02-WH-fleet\_v5\_launch |
| Webinar (sponsored) | WS | 2026\_03-WS-darkreading\_webinar |

#### **💻 Source: Digital**

*For all paid and owned online media and content.*

| Level 2: Source detail | Code | Level 3: Campaign examples |
| :---- | :---- | :---- |
| Paid search | PS | 2025\_11-PS-google\_brand\_usa |
| Paid social | SO | 2025\_11-SO-linkedin\_video\_ciso |
| Paid media | PM | 2025\_11-PM-riskybiz\_podcast |
| Content syndication & 3rd-party | CS | 2025\_12-CS-techtarget\_survey |
| Email marketing (owned list) | EM | 2025\_11-EM-newsletter\_promo |

#### **🎯 Source: Prospecting**

*For all outbound activities initiated by sales or a 3rd-party vendor.*

| Level 2: Source detail | Code | Level 3: Campaign examples |
| :---- | :---- | :---- |
| SDR outbound | SDR | SDR-General\_Prospecting (Always-On) 2025\_11-SDR-q4\_fintech\_sequence (Discreet) |
| AE outbound | AE | AE-General\_Prospecting |
| Outsourced BDR / CPL | OB | 2025\_11-OB-revshara\_meetings |

#### **🤝 Source: Partner**

*For all co-marketing and leads generated from formal channel partners.*

| Level 2: Source detail | Code | Level 3: Campaign examples |
| :---- | :---- | :---- |
| Tech partner | TP | 2025\_11-TP-aws\_marketplace |
| Reseller / VAR | RE | RE-General\_Referrals |
| Co-marketing | CM | 2026\_01-CM-crowdstrike\_whitepaper |

---

### 

### **5\. Campaign naming convention: The "how-to"**

**The golden rule:** Every single lead-generating activity *must* have a unique campaign in the CRM before it launches.

There are only two types of campaigns:

#### **1\. Discreet campaigns (time-based)**

These have a specific start, end, and budget (e.g., Webinar, Trade Show, Quarterly Ad).

* **Structure:** YYYY\_MM-\[Code\]-\[Name\]  
* **YYYY\_MM:** The start month. (e.g., 2026\_02)  
* **\[Code\]:** The 2-4 letter code from the table above. (e.g., MC, PS, WH)  
* **\[Name\]:** A short, URL-friendly name. (e.g., blackhat\_booth, google\_brand)  
* **Full example:** 2026\_08-MC-blackhat\_booth\_scans

#### **2\. "Always-on" campaigns (continuous)**

These are generic "buckets" for continuous inbound channels that don't have a start/end date.  They are “Default” campaigns, since they do not have a start or stop date.

* **Structure:** \[Code\]-Default  
* **\[Code\]:** The 2-4 letter code.  
* **\[Name\]:** Default, Always\_On, or General.  
* **Full example:** OS-Default (for all Organic Search)

## Unified campaign member status framework

### 1\. Executive summary

To accurately measure Marketing ROI and Attribution, we must standardize how we track prospect progression through our campaigns.   
This framework establishes a **Unified Status Hierarchy** for Salesforce campaigns. It introduces specific tiers for social engagement while maintaining rigorous definitions for high-intent conversions (Webinars, Events, Demos).

**Key Objectives:**

1. **Standardization:** Use the same language across all campaign types.  
2. **Attribution:** Ensure only meaningful interactions trigger attribution models.  
3. **Social Integration:** Capture top-of-funnel social intent without inflating pipeline metrics.

---

### 2\. Unified hierarchy

All campaigns must utilize the following status values. Custom statuses outside this list are to be avoided.

| Status Value | Responded? | Funnel Stage | PsyStage | Definition |
| ----- | ----- | ----- | ----- | ----- |
| **Targeted** | No | Unaware | 1 \- Unaware | The individual is on a list or in an audience segment but has taken no action. |
| **Sent** | No | Awareness | 2 \- Aware | The email was sent, the ad was displayed, or the post was published. |
| **Interacted** | **Yes** | Interest | **3 \- Intrigued**  | **(Light Touch)** Passive engagement. They clicked a link, liked a post, or visited a high-value page, but **did not exchange contact** info. |
| **Registered** | **Yes** | Consideration | **3 \- Intrigued** | **(Conversion)** The individual explicitly exchanged data for access (Form Fill, Sign Up, RSVP). |
| **Attended** | **Yes** | Evaluation | 3 \- Intrigued | The individual showed up to a synchronous event (Booth Scan, Webinar, Live Event, Dinner). |
| **Engaged** | **Yes** | Intent | **4 \- Has use case** | **(Deep Interaction)** High-effort engagement. They asked a question, made a meaningful comment, or engaged in a conversation. Hot Lead from Event |
| **Meeting Requested** | **Yes** | Purchase | 5 \- Personally confident | The individual explicitly requested a sales contact or a demo. |

---

### 3\. Operational definitions by channel

#### A. Social Media & Content

*Goal: Distinguish between vanity metrics (Likes) and true leads.*

* **Interacted:** User "Likes" a post, "Follows" the page, or clicks a link to ungated content.  
* **Engaged:** User comments on a post, shares/retweets with their own commentary, or sends a Direct Message (DM).  
* **Registered:** User fills out a specific lead gen form (e.g., LinkedIn Lead Gen) or clicks through to a landing page and converts.

#### B. Webinars & Virtual Events

*Goal: Track the drop-off between sign-up and attendance.*

* **Interacted:** Clicked the invitation link but did not complete registration.  
* **Registered:** Completed the registration form.  
* **Attended:** Logged into the webinar platform for \>1 minute.  
* **Engaged:** Attended **AND** asked a question in Q\&A, answered a poll, or stayed for the entire duration.

#### C. Physical Events (Trade Shows & Field Marketing)

*Goal: Differentiate between booth traffic and serious conversations.*

* **Interacted:** Visited the booth, took swag, a COLD LEAD  
* **Registered:** RSVP’d to the event (if hosted by us) or Pre-booked a meeting.  
* **Attended:** Badge scanned at booth.  
* **Engaged:** HOT lead. Had a meaningful conversation with a rep; notes added to CRM.  
* **Meeting Requested**

#### D. Meeting Service

*Goal: Qualify and move to become an opportunity*

* **Targeted:** A prospect is in the pool of potential targets  
* **Interacted:** Introductory meeting requested/ scheduled  
* **Attended:** Introductory meeting completed.  
* **Meeting Requested:** The prospect has asked for a follow-up engagement/discussion 

#### E. Email Marketing

*Goal: Move beyond "Open Rates."*

* **Sent:** Email delivered.  
* **Interacted:** Clicked a link in the email (Click-through).  
* **Registered:** Clicked a link and filled out the resulting form.  
* **Engaged:** Replied to the email directly.


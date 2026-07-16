# 🚂 Go-To-Market operations

This page covers the journey from prospect to customer and details what contributors need to know in order to make changes to the Go-To-Market (GTM) [philosophy](https://fleetdm.com/handbook/company/why-this-way#why-dont-we-sell-like-everyone-else), [strategy](https://fleetdm.com/handbook/company/go-to-market-operations#gtm-strategy), and [actions](https://fleetdm.com/handbook/company/go-to-market-operations#warm-up-actions) at Fleet.


## Cross-functional GTM processes

When communicating with future or current customers, hand-offs [between departments](https://fleetdm.com/handbook/company#org-chart), contributors, or external organizations can negatively effect the "Return On Investment" (ROI) for Fleet, our customers, and our friends in the community. Cross-functional GTM groups minimize hand-offs between internal and external stakeholders and maximize iteration and efficiency in the way we engage with the market.


> Use this "🚂 Go-To-Market operations" page to write down philosophies and show how the different pieces of the GTM process fit together.
> Use the dedicated departmental handbook pages for [🫧 Marketing](https://fleetdm.com/handbook/marketing), 💻 [IT](https://fleetdm.com/handbook/it), [🐋 Sales](https://fleetdm.com/handbook/sales), [🌦️ Customer Success](https://fleetdm.com/handbook/customer-success), and [💸 Finance](https://fleetdm.com/handbook/finance) to keep track of specific responsibilities and recurring rituals designed to be read and used within those departments.


## Current GTM motions

| GTM group                            | Goal |
|:-------------------------------------|:------------------------------------|
| [🤝Enterprise](https://fleetdm.com/handbook/company/go-to-market-operations#enterprise)             | Provide the best possible customer experience for organizations with 700+ hosts. 
| [🌐Buy online](https://fleetdm.com/handbook/company/go-to-market-operations#buy-online)         | Provide the best possible customer experience for organizations and contributors with less than 700 hosts that prefer a more self-service experience.


### 🤝Enterprise

The goal of the 🤝Enterprise group is to provide the best possible customer experience for organizations with 700+ hosts.

| Responsibility                    | Human(s)                  |
|:----------------------------------|:--------------------------|
| Revenue DRI                       | [Chaz MacLaughlin](https://www.linkedin.com/in/chazmaclaughlin/) _([@chazmac6](https://github.com/chazmac6))_
| Solutions Consultant (SC)         | [Allen Houchins](https://www.linkedin.com/in/allenhouchins/) _([@allenhouchins](https://github.com/allenhouchins))_ <br> [Harrison Ravazzolo](https://www.linkedin.com/in/harrison-ravazzolo/) _([@harrisonravazzolo](https://github.com/harrisonravazzolo))_ <br> [Mitch Francese](https://www.linkedin.com/in/mitchell-francese/) _([@tux234](https://github.com/tux234))_ <br> [Dave Siederer](https://www.linkedin.com/in/siederer/) _([@ds0x](https://github.com/ds0x))_ <br> [Henry Stamerjohann](https://www.linkedin.com/in/henry-st/) _([@headmin](https://github.com/headmin))_
| Account Executive (AE)           | [Patricia Ambrus](https://www.linkedin.com/in/pambrus/) _([@ambrusps](https://github.com/ambrusps))_ <br> [Anthony Snyder](https://www.linkedin.com/in/anthonysnyder8/) _([@anthonysnyder8](https://github.com/AnthonySnyder8))_  <br> [Nick Blee](https://www.linkedin.com/in/nickablee/) _([@NickBlee](https://github.com/NickBlee))_ <br> [Manny Mendoza](https://www.linkedin.com/in/mannymendoza1/) _([@mmendm](https://github.com/mmendm))_ 
| Solutions Specialist              | [Thomas Salomon](https://www.linkedin.com/in/thomassalomon4/) _([@ThomasSalomon4](https://github.com/ThomasSalomon4))_ <br> [Maribell Morales](https://www.linkedin.com/in/maribell-morales-056647139/) _([@maribell-fleetdm](https://github.com/maribell-fleetdm))_
| Pipeline DRI                      | [Ashish Kuthiala](https://www.linkedin.com/in/ashishkuthiala/) _([@akuthiala](https://github.com/akuthiala))_
| Customer Success DRI              | [Zay Hanlon](https://www.linkedin.com/in/zayhanlon/) _([@zayhanlon](https://github.com/zayhanlon))_
| Customer Success Manager (CSM)    | [Michael Pinto](https://www.linkedin.com/in/michael-pinto-a06b4515a/) _([@pintomi1989](https://github.com/pintomi1989)_) <br> [Joshua Roskos](https://www.linkedin.com/in/jroskos/) _([@kc9wwh](https://github.com/kc9wwh))_ 
| Customer Support Engineer (CSE)       | <sup><sub> _See [Customer_Success](https://fleetdm.com/handbook/customer-success)_

> The [Slack channel](https://fleetdm.slack.com/archives/C08BTMFTUCR), [kanban release board](https://github.com/orgs/fleetdm/projects/81/views/1), and [GitHub label](https://github.com/fleetdm/confidential/labels/%3Ahelp-gtm-ops) for this GTM group is `:help-gtm-ops`.


### 🌐 Buy online

The goal of the 🌐 Buy online group is to provide the best possible customer experience for organizations and contributors with less than 700 hosts that prefer a more self-service experience.

| Responsibility                    | Human(s)                  |
|:----------------------------------|:--------------------------|
| DRI                               | [Sam Pfluger](https://www.linkedin.com/in/sampfluger88/) _([@sampfluger88](https://github.com/sampfluger88))_


## Customer support service level objectives (SLOs)

**Fleet Free:**

| Impact level | Definition | Preferred contact | Response time |
|:---|:---|:---|:---|
| All inquiries | Any request regardless of impact level or severity | Osquery #fleet Slack channel | No guaranteed resolution |

> **Note:** If you're using Fleet Free, you can also access community support by [opening a bug](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=bug%2C%3Areproduce&projects=&template=bug-report.md&title=) in the [Fleet GitHub](https://github.com/fleetdm/fleet/) repository.

**Fleet Premium:**

| Impact level | Definition | Preferred contact | Response time |
|:-----|:----|:----|:-----|
| Emergency (P0) | Your production instance of Fleet is unavailable or completely unusable. For example, if Fleet is showing 502 errors for all users. | Expedited phone/chat/email support during business hours. </br></br>Email the contact address provided in your Fleet contract or chat with us via your dedicated private Slack channel | **≤4 hours** |
| High (P1) | Fleet is highly degraded with significant business impact. | Expedited phone/chat/email support during business hours. </br></br>Email the contact address provided in your Fleet contract or chat with us via your dedicated private Slack channel | **≤4 business hours** |
| Medium (P2) | Something is preventing normal Fleet operation, and there may or may not be minor business impact. | Standard email/chat support | ≤1 business day | 
| Low (P3) | Questions or clarifications around features, documentation, deployments, or "how to's". | Standard email/chat support | 1-2 business days | 

> **Note:** Fleet business hours for support are Monday-Friday, 6AM-4PM Pacific Time, excluding current U.S. federal holidays during which responses may be delayed for Medium and Low impact issues. Refer to your welcome email for contact and support info. 

**Emergency (P0) request communications:**

![Screen Shot 2022-12-05 at 10 22 43 AM](https://user-images.githubusercontent.com/114112018/205676145-38491aa2-288d-4a6c-a611-a96b5a87a0f0.png)


## Go-To-Market tools

Go-To-Market tools at Fleet will be vetted by the Head of GTM Architecture, onboarded by IT, and made available to all necessary stakeholders.

Any GTM tool, automation, or functionality that someone wants to explore using in Sales, Marketing, or Customer Success, including any tool we want to integrate with Salesforce or use its data in Salesforce, must first be evaluated and approved by the Head of GTM Architecture before being used by anyone at the company. This includes demos of GTM tools — any demo of a tool used in Sales, Marketing, Customer Success, or that integrates with/uses data from Salesforce must be kicked off by the Head of GTM Architecture.

To request approval for a new GTM tool/functionality, [create a GitHub issue](https://github.com/fleetdm/confidential/issues/new?assignees=sampfluger88&template=1-custom-request.md&labels=%3Ahelp-gtm-ops) and include a user story describing the goal of the added tool/automation.


## GTM strategy

At Fleet, our GTM strategy consists of:
- [Outreach](https://fleetdm.com/handbook/company/go-to-market-operations#gtm-outreach)
- [Processing intent signals](https://fleetdm.com/handbook/company/go-to-market-operations#processing-intent-signals)
- [Research](https://fleetdm.com/handbook/company/go-to-market-operations#research)
- [Warm-up actions](https://fleetdm.com/handbook/company/go-to-market-operations#warm-up-actions)
- [Executive Messaging Framework - Communicating the value of Fleet to the CIO and his direct team](https://docs.google.com/document/d/17u8z_aeZiatOBzcMfRtYewxMWZjRt-UKc0gQftcg7XM/edit?tab=t.0) (private Google doc)
- [Proof of Value (POV)](https://fleetdm.com/handbook/company/go-to-market-operations#proof-of-value-pov)
- [Signatures](https://fleetdm.com/handbook/company/go-to-market-operations#signatures)


### Competition

We track competitors' capabilities and adjacent (or commonly integrated) products in [Competition](https://docs.google.com/spreadsheets/d/1zwr59MpruIw4dsV-Qbk8xFbMrbHAV3qaRJDWM7-YrwU/edit?gid=611626809#gid=611626809) (private document).

> Due to legislation by the U.S. Department of Commerce, we are unable to initiate business with [certain countries and territories including specific U.S. sanction programs.](https://ofac.treasury.gov/sanctions-programs-and-country-information)


## GTM outreach 

Go-To-Market (GTM) strategy at Fleet is [always evolving](https://handbook.gitlab.com/handbook/values/#everything-is-in-draft), but the [philosophy behind the strategy](https://fleetdm.com/handbook/company/why-this-way#why-dont-we-sell-like-everyone-else) remains consistent. 

When you reach out to a prospect or customer, make sure you're the right person:
- **🐋 "Sales-ready" prospect**: The rep (AE) determines the best way to contact the prospective customer, which is often the Solutions Consultant reaching out to help.
- **🌦️ Customers**: The Customer Success Manager (CSM) reaches out.
- **🫧 Any other prospects**: The Solutions Consultant (SC) reaches out.


### Programs

Fleet's community programs are rooted in several areas created to nurture communication between all current and future Fleet users through [events](https://fleetdm.com/handbook/company/go-to-market-operations#events), community support, [social media](https://fleetdm.com/handbook/company/go-to-market-operations#social-media), [ads](https://fleetdm.com/handbook/company/go-to-market-operations#ads), [video](https://fleetdm.com/handbook/company/go-to-market-operations#video), and articles.


#### Social media

Fleet's largest asset is our user community, the people actually using Fleet. Public conversations on social media create valuable opportunities for contributors to answer technical questions and collect feedback.

Fleet [does not self-promote](https://www.audible.com/pd/The-Impact-Equation-Audiobook/B00AR1VFBU).  (Great brands are [magnanimous](https://en.wikipedia.org/wiki/Magnanimity).) Conversations are already happening in our social spaces that open up opportunities for Fleet to [engage with the community](https://fleetdm.com/handbook/marketing#engage-with-the-community).

Here are some topics for social media posts:
- Fleet the product
- Internal progress
- Highlighting community contributions
- Highlighting Fleet, osquery, and Mac Admins open-source accomplishments
- Industry news about osquery 
- Industry news about device management
- Upcoming events, interviews, and podcasts

> **Is there a post that you would like to see on the company page?**
>
> Original posts on LinkedIn by Fleet employees [can be promoted using the Fleet company page](https://fleetdm.com/handbook/marketing#promote-a-post-on-linkedin). If you think your post would make sense in front of a bigger audience, reach out in the [#help-marketing channel](https://fleetdm.slack.com/archives/C01ALP02RB5) linking the team to your personal LinkedIn post (only original posts please, re-posts and quotes of other posts can not be promoted). Include any context in the Slack message and keep an eye on your inbox. The Marketing team will request permission to use the post in a promoted post.   


##### Fleet on LinkedIn

> **Warning:** Do NOT buy LinkedIn Recruiter. AEs should use their personal Brex card to purchase the monthly [Core Sales Navigator](https://business.linkedin.com/sales-solutions/compare-plans) plan. Fleet does not use a company wide Sales Navigator account. The goal of Sales Navigator is to access to profile views and data, not InMail.  Fleet does not send InMail.


#### Ads

Fleet uses advertising to spread awareness through a broader audience and foster greater engagement within user communities. The more people actively using Fleet, or contributing, the better Fleet will be.


#### Events

We sponsor and participate in events so that we can support, connect, engage, and grow the Fleet community. We need to be thoughtful about what events we sponsor or host, and we need to be disciplined in how we run events so we can be efficient and effective.

> Want to propose an event:  
> Add a row to the ["🫧 Proposed events (not yet settled)" tab](https://docs.google.com/spreadsheets/d/1YQXAX2Q_WnGkAwMYjMbQpV3nbCj7gOBbv7Y0u4twxzQ/edit?gid=1411322737#gid=1411322737) of the 🎪 Events spreadsheet (confidential doc) and fill in all necessary information.


### Field event follow-up

#### Solution Specialist: Tradeshows, GitOps & field events

##### Post-event goals
- Immediate outreach to all scans and attendees within 24 hours
- Assign Partner, ICP, and pipeline accounts to the correct DRI
- Direct non-ICP accounts to the "Let's Get You Set Up" engagement

##### Follow-up process

**Immediate**

| Phase | Action | Timing |
|-------|--------|--------|
| Prioritize | Sort contacts by status. Tag each as ICP prospect, existing pipeline, partner, or non-ICP. Assign DRI immediately. | Within 2 hrs |
| Pull from SF | Open recent events dashboard. Pull all "Attended", "Engaged", "Interacted", and "Registered" members from the campaign record. | Within 2 hrs |

**Outreach**

| Phase | Action | Timing |
|-------|--------|--------|
| Email | Send a personalized email to every scan/attendee. Reference the event. Lead with value — not a generic follow-up. | Within 24 hrs |
| Call to Action | ICP/pipeline: request a meeting or demo with a booking link. Non-ICP: direct to self-serve or "Let's Get You Set Up." | Within 24 hrs |
| LinkedIn | Send a personalized connection request referencing the event. Follow up with a message after acceptance. | Within 24 hrs |
| Log in SFDC | Update campaign member status. Log all tasks, activities, meetings booked, and opportunities identified. | Within 24 hrs |

**Routing (concurrent with outreach)**

| Phase | Action | Timing |
|-------|--------|--------|
| ICP Prospect | Begin discovery. Book intro/demo. Create opportunity in SFDC. | Immediate |
| Pipeline Account | Notify DRI. Use event touch to accelerate the deal. Log against the open opportunity. | Immediate |
| Partner Account | Route to channel DRI. Log in CRM. | Immediate |
| Non-ICP | Send "Let's Get You Set Up." Point to self-serve. Log disposition in CRM. | Immediate |


#### SWAG

Bulk SWAG (Stuff-We-All-Get) orders of any kind are reviewed and placed by the [🫧 Content Specialist](https://fleetdm.com/handbook/marketing#team). If we're ordering a new SWAG item, the Content Specialist will work with the [🦢 Head of Design](https://fleetdm.com/handbook/product-design#team) to obtain an approved product template. 

##### Request swag

There are many times in which community members, customers, and contributors are in need of some cool Fleet swag. To request swag:
1. [Create an issue](https://github.com/orgs/fleetdm/projects/65) on the :help-marketing board.
2. Provide order details (e.g. expected shirt size, name, and shipping details).
3. Decide if you'd like to include a personalized message and attach it to the issue.


#### Video

Fleet uses YouTube to help keep the community up-to-date and informed. These videos facilitate community engagement, provide educational resources, and help share essential information about Fleet and the people using it. Meetings regularly uploaded to YouTube will have a "▶️" emoji prepended to the calendar event title (e.g. "▶️ ☁️🌈 Sprint demos!").  


## Processing intent signals

Intent signals help measure an individual's/organization's current level of engagement with the Fleet brand and help us:  
- Create/update contact and/or accounts in Fleet's CRM. 
- Prioritize accounts for [research](https://fleetdm.com/handbook/marketing#research-an-account).
- Identify accounts/contacts that would benefit from a POV conversation.

When processing intent signals, prioritize accounts in the following order:
1. Sales-ready: Accounts currently assigned to reps.
2. Ads running: Accounts with trending psychological progression (as measured by fleetdm.com website signups (i.e. new contacts ± contacts that have increased their psystage to a certain point).
3. Researched: Key accounts that fleeties have suggested to prioritize.


## Research

ADRs research accounts to ensure there's a practical need Fleet can solve before we attempt to reach out. By doing the groundwork of validating account data and gauging initial intent, we can ensure the organizations that we reach out to [benefit from deeper discussions](https://fleetdm.com/handbook/company/go-to-market-operations#proof-of-value-pov) around making Fleet a practical solution.


## Warm-up actions

Warm-up actions are actions that will take at any point in time to help move the psychological progress of contacts on any account.
Our ADRs stay connected with Fleet’s engineering team to keep technical knowledge current and to coordinate any outreach. Everyone’s time is valuable, and this approach ensures that prospects have direct access to engineers who speak their language. (Munki, DDM, patch management, EPSS, etc.)


### Solution Specialist inbound lead follow-up

### Objective

Convert high-intent inbound leads into qualified first meetings through fast engagement and strict response SLAs.

### Source queue

Salesforce Report: Solution Specialist — includes:
- Demo Requests - ICP
- Demo Requests (evaluate)
- "Talk to Us" form submissions
- Webinar Sign-ups
- Document Downloads
- Sign-ups and trial starts
- Swag requests (lower priority)
- Other website conversions

Monitor hourly, sorted by Created Date (newest first). No lead untouched > 60 minutes. If a Solution Specialist is out of office, etc. another Solution Specialist will assume ownership of the follow-up.

### Lead prioritization

| Tier | SLA | Leads |
|------|-----|-------|
| Tier 1 | Immediate | "Talk to Us" forms, demo requests, 1K+ employee accounts |
| Tier 2 | ≤ 1 hour | Sign-ups and trial starts |
| Tier 3 | Same day | Swag requests, low-intent conversions — qualify fast or disqualify |

### Workflow

**Check for live engagement (last 10 min)**
- Attempt real-time engagement via Qualified — SLA: ≤ 5 minutes

**Immediate outreach (same hour, if not live)**
- Send a personalized email referencing company name, use case, and source
- LinkedIn connection request
- Maximum three attempts at contact, if no response

**Qualify quickly**
- Clear use case? Relevant company size? Relevant role?
- If unsure → treat as Tier 2 and follow up once
- If no → disqualify and move on

**Book meeting**
- Schedule directly on AE calendar
- Include: use case summary, lead source, and any urgency signals

**Update Salesforce — before moving to next lead**
- Status updated, activity logged, notes added (use case + tier)
- Assign to AE if qualified

### Response SLAs

| Scenario | SLA |
|----------|-----|
| Live inbound (chat / Qualified) | ≤ 5 minutes |
| New Tier 1 & 2 inbound | ≤ 1 hour |
| Tier 3 (swag, low intent) | Same day |
| Meeting scheduled after qualification | ≤ 48 hours |

### Outreach standards

- Always reference company name and action taken
- No generic messaging. Keep it direct, outcome-focused, and short
- No lead ends the day without a clear disposition

### Escalate immediately if

- 2,000+ employees in report
- Known or high-value target
- "Talk to Us" submission combined with strong buying signals (e.g., Calendly intent)

### Daily standard

- Zero backlog — every lead has an action taken and clear disposition
- All priority leads engaged within SLA
- Salesforce updated before moving to next lead



## Proof of value (POV)

When the prospect is ready to "kick the tires/do a POC", the opportunity is moved to "Stage 3 - Requested POV"  in Salesforce. The AE and SC work together with the prospect to define a timeline and the "definition of done" in order to scope the POV. This planning helps us avoid costly detours that can take a long time, and result in folks getting lost. 


### Spin up a POV

You can set up a Fleet Managed Cloud environment for a prospect with >700 hosts, or you can help them generate a trial license key to configure on their own self-managed Fleet server.

- **To set up a new Fleet Managed Cloud environment** for a user: First, [create a "New customer environment" issue](https://github.com/fleetdm/confidential/issues/new?template=new-fleet-instance.md).  Then, once the environment is set up, you'll get a notification and you can let the user know.
- **To set up only a trial license key** for a user's self-managed Fleet server: Point the user towards fleetdm.com/try, where they can sign up and choose to "Run your own trial with Docker".  On that page, they'll see a license key located in the `fleectl preview` CLI instructions, and they can configure this by copying and pasting it as the [`FLEET_LICENSE_KEY`](https://fleetdm.com/docs/configuration/fleet-server-configuration#license-key)  environment variable on the server(s) where Fleet is deployed.


### NFR instances

NFR (Not For Resale) instances are Fleet environments deployed for partners and resellers who need to demo Fleet functionality or test integrations. Solutions Consulting sets up these instances to support partner enablement and evaluation activities outside of the standard sales process.

#### Deploy an NFR instance

**To deploy an NFR instance:** Create a [new NFR instance issue](https://github.com/fleetdm/confidential/issues/new?template=new-nfr-request.yml). Solutions Consulting will deploy the instance. The infrastructure team will then configure DNS and email, and the requester will be notified in #help-solutions-consulting when the instance is ready.


## Quoting

### Generate a quote

Navigate to the opportunity you are creating a quote for, then follow the steps below.

> Are you generating a quote for a customer?
>
> If so, be sure you're doing it from the renewal oppty. Do not generate quotes from an expansion opportunity. 

1. Advance through each pipeline stage sequentially using the stage progression bar. Salesforce enforces sequential progression — skipping a stage will trigger an error.

2. When you attempt to advance to the **"Justification"** stage, Salesforce will block the move with an error indicating an approved quote is required. Scroll down to the **"Quotes"** section and click **"New Quote"**.

3. Enter a proxy name for the quote (the system auto-generates the final name). Review the pre-populated defaults — Billing Frequency is Annual and Payment Terms are Net 30 — and update if needed. Click **"Save"**, then open the newly created quote record.

4. Scroll to **"Related Lists"** and locate **"Quote Line Items"**. Click **"Add Products"** to open the Fleet Price Book, select all products to include, and click **"Next"**.

5. For each line item, set the following fields:
   - **"Unit Price"**: update if applying a discount from list price
   - **"Quantity"**: number of endpoints or units
   - **"Contract Start Date"**: effective date of the contract
   - **"Contract Term in Months"**: duration (e.g., 12 for one year, 24 for two years)

   The **"Discount Percentage"** field is locked and updates automatically based on the difference between list price and unit price. Click **"Save"**. MRR, ARR, Total Price, and Grand Total will update automatically.

6. Click **"Generate PDF"**, select the appropriate pre-approved template using the table below, and click **"Create PDF"**. Review the preview for accuracy, and save it to the quote.

   | Template | When to use |
   | -------- | ----------- |
   | Direct | Renewing an existing non-channel customer with no partner involved |
   | Direct w/ custom terms | Selling directly to the customer with special commitments included |
   | Authorized partner | Selling through an authorized channel partner |
   | Unauthorized partner | Selling through a channel partner pending authorization |

> Custom terms:
>
> If you're adding custom terms to a quote, be sure to add all necessary language in the **"Terms"** field. No "General terms" are added to the template by default.
> Any quote with custom terms must be reviewed by the CFO in addition to other approvers.

7. Click **"Submit for Approval"**. Complete any remaining required fields:
   - Billing address
   - Contact name (selected from associated Salesforce contact records)
   - Quote expiration date
   - Contract term in months
   - Billing frequency and payment terms

   Add any relevant notes in the submission comments field and click **"Submit"**.

   > Chaz is the approver for new business opportunities. Zay is the approver for renewals and upsells.

8. If the quote is rejected, review the feedback, make the necessary changes to the quote or line items, and resubmit.

9. Once the quote is approved, send it to the relevant stakeholders.
 

### Remove a contact from the "Top contacts" list in Salesforce

1. Navigate to the contact.
2. Uncheck the ⭐ field in the system info section at the bottom and save the record.

<img width="2489" height="612" alt="image" src="https://github.com/user-attachments/assets/d67e6890-4eb8-485b-8faf-4eed67a14ce1" />


### Mark an account as a "Top target"

Navigate to the account you would like to label as a "Top target". Edit the account, and check the box, and save the record.

> Want to stack rank your target accounts?
>
> Once you check the "Top target" box, the "Target tier" field will appear. You can split your target accounts into 3 different tiers. "Tier 1" being the best, "Tier 3" being the least priority but still worth calling out.


### Fill out the "Pre-meeting context" field

After a meeting is booked, the Solutions Specialist is responsible for filling out the **"Pre-meeting context"** field on the Salesforce opportunity before the meeting takes place.

To fill out this field:

1. Reach out to the prospect who set the meeting (e.g. via email or LinkedIn) to gather additional context about what they're hoping to get out of the call.
2. Ask about their current environment, key pain points, and any specific goals or topics they want to cover.
3. Navigate to the Salesforce opportunity and edit the **"Pre-meeting context"** field.
4. Summarize the context you gathered from the prospect and save the record.

This helps the full meeting team arrive prepared and aligned on the prospect's priorities.



## Signatures


### Getting a contract reviewed

The [Finance team](https://fleetdm.com/handbook/finance#team) will review all contracts within **2 business days**. 

> If a document is ready for signature and does not need to be reviewed or negotiated, you can skip the review process and [get the contract signed](https://fleetdm.com/handbook/company/communications#getting-a-contract-signed). Please submit other legal questions and requests to 💸 [Finance](https://fleetdm.com/handbook/finance#contact-us).

To get a contract reviewed, complete the [contract review issue template in GitHub](https://github.com/fleetdm/confidential/issues/new?assignees=hollidayn&labels=%3Ahelp-finance&projects=&template=contract-review.md&title=Review%3A++%F0%9F%96%8B%EF%B8%8F+__________________________). Upload the docx version whenever possible and be sure to include the link to the document in the issue. Follow-up comments should be made in the GitHub issue and in the document itself to avoid losing context.

If an agreement requires additional review during the negotiation process, the requestor will need to upload the new draft agreement and repeat the process. When no further review or action is required, the requestor is responsible for [routing the document](https://fleetdm.com/handbook/company/communications#getting-a-contract-signed) for signature.


### Getting a contract signed

The SLA for contract signature is **2 business days**. Please do not follow up on signatures unless this time has elapsed. If a contract is ready for signature and **DOES NOT** require [review or revision](https://fleetdm.com/handbook/company/communications#getting-a-contract-reviewed) (i.e. no contract review issue necessary), follow the steps below:

First, log into DocuSign (credentials in 1Password) and route the agreement to the CFO for signature via [Fleet's Sales email address](https://docs.google.com/document/d/1tE-NpNfw1icmU2MjYuBRib0VWBPVAdmq4NiCrpuI0F0/edit?tab=t.0). 

> When a contract is going to be routed for signature by someone outside of Fleet (i.e. the vendor or customer), the requestor is responsible for working with the other party to make sure the document gets routed correctly. Please use [Fleet's Sales email address](https://fleetdm.com/handbook/company/communications#email-relays) for all contracts and never include individual emails in any company agreement. If the agreement includes any individual emails, remove them before routing the agreement to the CFO for signature.

Once the signature SLA has expired you can [contact Finance](https://fleetdm.com/handbook/finance#contact-us) to follow up. 


## Vendor questionnaires 

Occasionally, prospective customers will ask us to complete a questionnaire. In responding to security questionnaires, Fleet endeavors to provide full transparency via our [security policies](https://fleetdm.com/handbook/it/security#security-policies), [trust](https://trust.fleetdm.com/), and [application security](https://fleetdm.com/handbook/it/security#application-security) documentation. In addition to this documentation, please refer to [the vendor questionnaires page](https://fleetdm.com/handbook/it/security#vendor-questionnaires). [Contact the Sales department](https://fleetdm.com/handbook/sales#contact-us) to address any pending questionnaires.


### Fleet's vendor collateral

Use the following steps to send Fleet's vendor collateral to a prospect or customer:
1. Be sure that there's a signed NDA between Fleet and the requesting organization [saved in Google Drive](https://drive.google.com/drive/folders/1ee6E2wwhUL8F5qTRGUleJ9HeWjSRj5xm?usp=drive_link). If not, [send an NDA](https://fleetdm.com/handbook/sales#send-an-nda).
2. Navigate to the ["🗃️ Vendor collateral" folder (confidential)](https://drive.google.com/drive/folders/18_Q7Q9Qwu7a8uFyHIS9QZo7iVA0B9YED?usp=drive_link) in Google Drive and to download the necessary documents.

> 🧑‍🚀 Attention Fleeties:
>
> Can't find what you're looking for in Google Drive? 🧐 Reach out in the [🚂 :help-gtm-ops Slack channel](https://fleetdm.slack.com/archives/C08BTMFTUCR) for help. Any collateral documents (e.g. SOC2, Pen test, etc.) you send to a prospect or customer should be downloaded from the [🗃️ Vendor collateral folder](https://drive.google.com/drive/folders/18_Q7Q9Qwu7a8uFyHIS9QZo7iVA0B9YED?usp=drive_link) in Google Drive. If it's not in the "🗃️ Vendor collateral folder", it's not ready to be sent out.


## Slide decks

The goal of a slide deck is not necessarily to walk every customer through it.  It's to make sure we're presenting the most impactful outcomes of Fleet to the right people, and standardizing how we talk about the products and customer experience to give people evaluating Fleet the opportunity to understand it, fairly evaluate it, and present it in the best light internally to other people at their organization.

Even if you never show these decks on a screenshare, use them to keep the conversation on track, or to send as a teaser.

- [Fleet for IT engineers and IT admins](https://docs.google.com/presentation/d/1WTyGrmA4pSB7H8BeT14BF7peozBceToW8TK__doyQTg/edit?slide=id.g3d7b8aeb1bc_1_182#slide=id.g3d7b8aeb1bc_1_182)
- [Fleet for digital workplace leaders](https://drive.google.com/file/d/1JlIV1PY5lECQQmq2H_eR35haeKefHXIf/view?usp=sharing)
- [Fleet for partners](https://docs.google.com/presentation/d/1iNvn5EYnkklKxguYzrOh6ZNvZee53OqAlF3rc_Da_Us/edit?slide=id.g3871afd58d8_0_0#slide=id.g3871afd58d8_0_0)

<!--
- [Fleet for digital workplace leaders](https://docs.google.com/presentation/d/1G8BtuhYRX92He3AifA5TAW4YlZO3jlcj8OeCqcSHmOM/edit?slide=id.g3d28ee536a1_2_37#slide=id.g3d28ee536a1_2_37)
- [Fleet for CISOs](https://docs.google.com/presentation/d/17PUAqa63jTb5yFT3hGg3F5mgGyPtUmg8OlTGyxS6vLI/edit?slide=id.g3d28ee536a1_2_0#slide=id.g3d28ee536a1_2_0)
- [Fleet for CIOs](https://docs.google.com/presentation/d/14GpQs83B_nxTe2hbf2eOJDU6i0OaGnNvXSX7b47boBA/edit?slide=id.g3e7bfd82431_0_29#slide=id.g3e7bfd82431_0_29)
-->


## Go-To-Market architecture and automation

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

Fleet uses Okta SSO for Salesforce authentication. All Fleet employees (`@fleetdm.com`) authenticate through Okta — Salesforce credential login is disabled for SSO-enabled profiles. All Fleet employees must login at our custom domain [fleetdm.my.salesforce.com](https://fleetdm.my.salesforce.com) or by clicking the Salesforce app tile in Okta. For users and accounts that cannot use SSO (e.g., integration users, external collaborators), Fleet has created custom cloned profiles with SSO disabled that must login at [login.salesforce.com](login.salesforce.com).


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


##  Go-to-market attribution

Our go-to-market (GTM) approach is built on a foundation of end-to-end visibility. We want to track touchpoints from first engagement through closed revenue, connecting marketing activity to pipeline and revenue. This means instrumenting our campaigns, content, and channels with consistent attribution, maintaining clean data flow between our marketing automation and CRM systems, and building reporting that ties spend and effort directly to outcomes. The goal isn't data for data's sake—it's to create a feedback loop where we can see what's working, double down on high-performing channels, cut what isn't delivering, and continuously refine our targeting, messaging, and timing. Every campaign we run should make us smarter about the next one.


## Conversion rates

Conversion rates help us to plan, forecast, and improve. There are several key comparisons that we want to understand:

- **Win rate**: From stage X to closed won.  For closed opportunities, this tells us what percentage of opportunities historically will be won for a given stage in the sales cycle.  
- **Stage to win cycle time**:   
- **Stage to stage**:tbd/todo  
- **Stage to stage cycle time**: 

## GTM model

We can build a reverse funnel using the conversion rates and an estimated ASP, which will indicate the business demand for top-of-funnel contacts and opportunities in order to attain future revenue targets. 


## Contact source
At Fleet, we also keep track of the specific form or activity that a contact completed when they were created. This way we keep track of "Where" they came from (the attribution framework), but also have data about what they did.  We have a field *Contact source*, which is the same as the first historical event that took place causing us to create the contact.

Here are the values for the contact source:

| Contact source value | Definition |
| :---- | :---- |
| Attended a call with Fleet | Contact was added to the system after attending a calendar invite/call with the team. |
| Website - Contact forms - Demo | Contact requested a standard demo via the website. |
| Website - Contact forms - Demo - ICP | Contact requested a demo and was routed/flagged as an Ideal Customer Profile. |
| Website - Contact forms | Contact submitted a general inquiry via the website. |
| Website - Chat | Contact engaged and provided their email via the website chatbot. |
| Website - Sign up | Contact created an account/signed up for the Fleet platform. |
| Website - Gated document | Contact filled out a form to download a whitepaper, report, or guide. |
| Website - Newsletter | Contact explicitly subscribed to the Fleet blog or newsletter. |
| Website - Workshop request | Contact filled out a form on the website requesting a workshop in a city near them. |
| Website - Swag request | Contact filled out a form specifically to request Fleet merchandise. |
| Event | Contact was scanned, uploaded, or registered from a live physical or virtual event. |
| Event - Webinar | Contact registered for or attended a webinar (hosted by Fleet or a 3rd-party). |
| Event - Workshop | Contact registered for or attended a workshop hosted by Fleet, such as a [GitOps workshop](https://fleetdm.com/gitops-workshop). |
| LinkedIn - Liked the LinkedIn company page | Contact followed or liked the official Fleet LinkedIn page. |
| LinkedIn - Reaction | Contact reacted (like, celebrate, etc.) to a Fleet post. |
| LinkedIn - Comment | Contact commented on a Fleet post. |
| LinkedIn - Share | Contact shared a Fleet post. |
| LinkedIn - Native lead form | Contact submitted their info directly inside LinkedIn via a Document Ad or lead gen form. |
| Prospecting - AE | Contact was sourced directly via outbound efforts by an Account Executive and added to Linkedin via Dripify webhook. |
| Prospecting - Specialist | Contact was sourced directly via outbound efforts by a Solution Specialist. |
| Prospecting - Meeting service | Contact was sourced/booked via an outsourced meeting-setting agency. |
| GitHub - Stared fleetdm/fleet | Contact starred the Fleet repository. |
| GitHub - Forked fleetdm/fleet | Contact forked the Fleet repository. |
| GitHub - Contributed to fleetdm/fleet | Contact made a code/documentation contribution to the Fleet repository. |

<!-- FUTURE:

| Website - Partner sign up | Contact submitted a form to apply for or join the Fleet partner program. |
| Website - Deal registration | Contact was tracked by an authorized partner/reseller filling out a form on the website as part of a formal deal registration. |

-->


<!--

## Attribution framework

To scale demand generation effectively, we need to have a trusted source of data about what works in generating new contacts, opportunities, pipeline, and business. Without a consistent framework, our data is messy, reporting is unreliable, and we cannot confidently measure the ROI of our marketing or sales efforts. This framework solves three core problems:

1. **Inconsistent data**  
2. **Lack of visibility**  
3. **Inaccurate ROI**

This outlines a simple, scalable, and non-negotiable system for tracking all contact-generating activities at Fleet.



### First-touch vs. converting-touch

This framework is **not** just for the Contact Source field. It should be applied to **two separate, critical moments** in the customer journey.  At some point, we may want to look at multi-touch attribution, this model is our starting point and foundation.


#### 🌎 First-touch: Original contact source

- **What it is:** The "birth certificate" of a contact. It is the very first marketing or sales touch that brought this person into our database.  
- **The rule:** This field is **set once and is locked forever**. It should *never* be overwritten.  
- **It answers:** "Which of our channels are best at generating *net-new names* and filling the top of our funnel?"

#### 🏁 Converting-touch: Opportunity creation source 🟡

- **What it is:** The "final push." It is the specific campaign that caused a known contact to convert into a sales-qualified opportunity (i.e., they booked a demo or engaged with sales).  
- **The rule:** This field is set *at the moment of opportunity creation*.  
- **It answers:** "Which of our channels are best at generating *pipeline and revenue*?"

Example:

- A MacAdmin first discovers FleetDM by attending an OSQuery 101 webinar in Oct 2025 (2025\_10-WH-osquery\_101).   
  - Their First-Touch is Event \> Webinar (Hosted). 

- Six months later, an SDR emails them (2026\_04-SDR-q2\_fintech\_sequence), and they reply to book a demo.   
  - Their Most Recent/Converting-Touch is Prospecting \> SDR Outbound.
  
Converting-touch is always stamped fresh at the moment of opportunity creation. If a contact re-engages after a prior opportunity has closed, the new opportunity's Converting-touch reflects whatever campaign or activity drove the current re-engagement,not any historical value. The prior opportunity record retains its own Converting-touch data. If a closed-lost opportunity is re-engaged within 90 days, we typically should reopen the original opportunity rather than creating a new one.

**Converting-touch** allows us to see that our webinars are great for *finding* contacts, and our SDR team is great at *converting* them.


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
- **🤝 Partner**: All co-marketing and contacts generated from formal channel partners.


#### 🌳 Organic/web

For all unpaid, inbound traffic and brand-driven interest.

| Source detail | Code | Campaign examples (always-on) |
| :---- | :---- | :---- |
| Organic search | OS | Default-OS |
| Direct traffic | DT | Default-DT |
| Web referral | WR | Default-WR |
| Organic social | SOC | Default-SOC |
| Organic AI | AI | Default-AI, Default-AI-ChatGPT |


#### 🗣️ Word-of-mouth

For all manually tracked, human-to-human recommendations.

| Source detail | Code | Campaign examples |
| :---- | :---- | :---- |
| Customer referral | CR | Default-CR  |
| Employee referral | ER | Default-ER  |
| Analyst/influencer | AR | AR-gartner\_mention |


#### 🗓️ Event

For all in-person or virtual events, sponsored or hosted.

| Source detail | Code | Campaign examples (discrete) |
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
| Press Release | PR | 2025\_11-PR-Abc\_launch |  


#### 🎯 Prospecting

For all outbound activities initiated by sales or a 3rd-party vendor.

| Source detail | Code | Campaign examples |
| :---- | :---- | :---- |
| SDR outbound | SDR | Default-SDR-General\_Prospecting (Always-On) or 2025\_11-SDR-q4\_fintech\_sequence (a discrete campaign) |
| AE outbound | AE | Default-AE-General\_Prospecting |
| Meeting Service | MS | 2025\_11-MS-VIB 2026\_01-MS-SageTap |

Other exmaples of campaigns:
Default-AE-Dripify_LinkedIn
Default-SDR-Dripify_LinkedIn


#### 🤝 Partner

For all co-marketing and contacts generated from formal channel partners.

| Source detail | Code | Campaign examples |
| :---- | :---- | :---- |
| Tech partner | TP | Default-TP-TechPartner\_Referral or 2025\_11-TP-aws\_marketplace |
| Reseller / VAR | RE | Default-RE-Reseller\_Referral |
| Co-marketing | CM | 2026\_01-CM-crowdstrike\_whitepaper |


### Campaigns

**The golden rule:** Every single contact-generating activity *must* have a unique campaign in the CRM before it launches. 

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


## SFDC field mapping

The attribution framework is implemented across two record types in Salesforce: Contact and Opportunity. Understanding which fields store which attribution values — and how they behave — is essential for building accurate reports and debugging data issues.

### Contact fields

There are nine attribution fields on the Contact record, organized into two groups: **Source** (first-touch, locked forever) and **Most Recent** (updated on every new campaign touch).

| Field label | API name | Attribution level | Behavior |
|---|---|---|---|
| Source campaign initial URL | `Source_campaign_initial_url__c` | — | The landing page URL from the contact's first touch. Set once, never overwritten. |
| Source channel | `Source_channel__c` | L1 | The high-level source bucket (e.g., Event, Prospecting). Set once, never overwritten. |
| Source channel detail | `Source_channel_detail__c` | L2 | The specific tactic (e.g., Webinar Hosted, SDR Outbound). Set once, never overwritten. |
| Source campaign | `Source_campaign__c` | L3 | The specific campaign name (e.g., `2026_02-WH-fleet_v5_launch`). Set once, never overwritten. |
| Most recent campaign initial URL | `Most_recent_campaign_initial_url__c` | — | The landing page URL from the contact's most recent touch. Updated on every new touch. |
| Most recent channel | `Most_recent_channel__c` | L1 | Updated on every new campaign touch. |
| Most recent channel detail | `Most_recent_channel_detail__c` | L2 | Updated on every new campaign touch. |
| Most recent campaign | `Most_recent_campaign__c` | L3 | The primary trigger field for the attribution automation. Updated on every new campaign touch. |
| Most recent campaign member status | `Most_recent_campaign_member_status__c` | — | Reflects the contact's engagement level on their most recent campaign. Updated on every new touch. |

### Opportunity fields

When an opportunity is created from a Contact, the Most Recent values at that moment are copied into the Opportunity's Converting fields. These represent the converting-touch — the campaign that drove this specific pipeline event.

| Field label | API name | Attribution level | Behavior |
|---|---|---|---|
| Converting contact | `Converting_Contact__c` | — | Lookup to the Contact record that triggered opportunity creation. |
| GCLID | `GCLID__c` | — | Google Click ID. Captured for paid search attribution. |
| Converting channel | `Converting_channel__c` | L1 | Copied from Most Recent Channel at opportunity creation. |
| Converting channel detail | `Converting_channel_detail__c` | L2 | Copied from Most Recent Channel Detail at opportunity creation. |
| Converting campaign | `Converting_campaign__c` | L3 | Copied from Most Recent Campaign at opportunity creation. |
| Primary Campaign Source | `CampaignId` | L3 | Standard SFDC lookup to the Campaign record. Set at opportunity creation. |

### How the automation works

The attribution system is driven by a single trigger: **when Most Recent Campaign is populated**, a Salesforce Flow fires and handles everything downstream.

**Step 1 — Derive L1 and L2 from the campaign name.** The two-character code embedded in every campaign name (e.g., `WH` in `2026_02-WH-fleet_v5_launch`) is used to look up the correct Source Channel Detail (L2) and Source Channel (L1) values automatically. This is why the campaign naming convention is non-negotiable — the automation depends on it.

**Step 2 — Stamp first-touch if Source fields are blank.** If the Source Channel field is empty, the flow copies the Most Recent values into the Source fields. This happens exactly once per contact — the moment they are first known to us. After that, the Source fields are locked and never overwritten.

**Step 3 — Add to campaign and set member status.** The flow adds the contact as a Campaign Member on the corresponding SFDC Campaign record and sets their member status based on the Most Recent Campaign Member Status field.

**Step 4 — Populate Opportunity on creation.** When an opportunity is created, the Most Recent Channel, Most Recent Channel Detail, and Most Recent Campaign values are copied to the Converting fields on the Opportunity record, capturing the converting-touch at that exact moment.

Note: The Most Recent values on the contact are updated with each engagemet with the contact, overwriting historical values.

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

| Status value | Responded? | Funnel stage | Psystage (legacy) | Definition |
| ----- | ----- | ----- | ----- | ----- |
| **Targeted** | No | Unaware | 1 \- Unaware | The individual is on a list or in an audience segment but has taken no action. |
| **Sent** | No | Awareness | 2 \- Aware | The email was sent, the ad was displayed, or the post was published. |
| **Interacted** | **Yes** | Interest | **3 \- Intrigued**  | **(Light Touch)** Passive engagement. They clicked a link, liked a post, or visited a high-value page, but **did not exchange contact** info. |
| **Registered** | **Yes** | Consideration | **3 \- Intrigued** | **(Conversion)** The individual explicitly exchanged data for access (Form Fill, Sign Up, RSVP). |
| **Attended** | **Yes** | Evaluation | 3 \- Intrigued | The individual showed up to a synchronous event (Booth Scan, Webinar, Live Event, Dinner). |
| **Engaged** | **Yes** | Intent | **4 \- Has use case** | **(Deep Interaction)** High-effort engagement. They asked a question, made a meaningful comment, or engaged in a conversation. Hot contact from Event |
| **Meeting Requested** | **Yes** | Purchase | 5 \- Personally confident | The individual explicitly requested a sales contact or a demo. |


### Operational definitions by channel

#### Social media and content

*Goal: Distinguish between vanity metrics (Likes) and true prospects.*

- **Interacted:** User "Likes" a post, "Follows" the page, or clicks a link to ungated content.  
- **Engaged:** User comments on a post, shares/retweets with their own commentary, or sends a Direct Message (DM).  
- **Registered:** User fills out a specific lead gen form (e.g., LinkedIn lead gen form) or clicks through to a landing page and converts.

#### Webinars and virtual events

*Goal: Track the drop-off between sign-up and attendance.*

- **Interacted:** Clicked the invitation link but did not complete registration.  
- **Registered:** Completed the registration form.  
- **Attended:** Logged into the webinar platform for \>1 minute.  
- **Engaged:** Attended **AND** asked a question in Q\&A, answered a poll, or stayed for the entire duration.

#### Physical events 

*Goal: Differentiate between booth traffic and serious conversations.*

- **Interacted:** Visited the booth, took swag, a COLD contact  
- **Registered:** RSVP’d to the event (if hosted by us) or pre-booked a meeting.  
- **Attended:** Badge scanned at booth.  
- **Engaged:** HOT contact. Had a meaningful conversation with a rep; notes added to CRM.  
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
- **Engaged:** We chatted and learned about the prospect
- **Meeting Requested:** The prospect has booked a meeting

-->


## 📧 Contact marketability & compliance

At Fleet, we maintain a strict separation between contacts we *can* legally email (Marketable) and those we are prospecting cold (Non-Marketable). This ensures we honor opt-outs, protect our domain reputation, and comply with GDPR/CAN-SPAM.

We do not rely on "implied" logic (e.g., "If they have an email, email them"). Instead, we use a dedicated status field on the Contact object to act as the single source of truth.

### The "marketing status" definitions

The `Marketing_Email_Status__c` picklist is the master switch for a contact's eligibility. Every contact in Salesforce must fall into one of the following buckets:

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
| **Status Reason** | `Marketing_Status_Detail__c` | **The Audit Trail.** <br>

<br>Explains *why* the status changed. <br>


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
- **Rule:** This is permanent. A "Cold" contact can become "Marketable" (by filling a form), but an "Unsubscribed" contact is locked unless they manually re-subscribe.

### Why this matters

- **Compliance:** We must be able to prove *when* and *how* someone consented to receive emails.
- **Deliverability:** Sending marketing blasts to "Cold" data (Clay lists) ruins domain reputation. We keep those lists separate for low-volume, high-relevance sales outreach only.
- **Debugging:** If a VIP prospect stops receiving emails, the `Status Reason` tells us if it was a system error (Bounce) or human error (Sales marked "Do Not Contact").


## ActiveCampaign

Fleet uses ActiveCampaign as its marketing automation platform for email marketing, contact lifecycle management, nurturing, and segmentation. ActiveCampaign is integrated with Salesforce (SFDC) as the system of record; key contact fields sync from SFDC into ActiveCampaign, and lifecycle transitions driven by sales activity in SFDC are reflected in ActiveCampaign automations.

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
| `ls: mql` | Lifecycle Stage | Marketing Qualified. Right-fit demographics/firmographics plus minor intent signal. Qualifies for increased marketing investment. |
| `ls: srl` | Lifecycle Stage | Sales Ready. MQL threshold met plus sufficient intent to hand off. Triggers sales notification and pauses marketing nurture. |
| `ls: sal` | Lifecycle Stage | Sales Accepted. Sales has accepted and is actively working the contact. SFDC is system of record from this point. |
| `ls: sql` | Lifecycle Stage | Sales Qualified. Sales has met with the contact and is moving forward toward a deal. |
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

Fleet uses ActiveCampaign for all owned-list email marketing. This includes the newsletter, product announcements, event follow-ups, and nurture sequences.

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


<meta name="maintainedBy" value="sampfluger88">
<meta name="title" value="🚂 Go-To-Market operations">

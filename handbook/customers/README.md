# Customers

## Contacting Fleet

If you're using a free version of Fleet, you can access free community support by opening an issue in the [Fleet GitHub repository](https://github.com/fleetdm/fleet/).

Customers on a paid tier of Fleet can get in touch directly for commercial support:

| Level of impact | Type of support |
| :--- | :--- |
| Low to medium impact | Email/chat support during business hours </br> Email: support @ fleetdm.com </br> Chat: Dedicated Slack channel (confidential) </br> Response time: **≤1 business day** |
| High to emergency impact | Expedited phone/chat/email support during business hours </br> Call or text: **(415) 651-2575** </br> Email: emergency @ fleetdm.com </br> Response time: **≤4 hours** |

## Customer success calls

### Scheduling a customer call

To schedule an [ad hoc meeting](https://www.vocabulary.com/dictionary/ad%20hoc) with a Fleet customer, use the ["Customer meeting" Calendly link](https://calendly.com/fleetdm/customer).

### Documenting a customer call

When we do prospect calls, add the customer's name in both the google doc title and the heading, ex. "Charlie (Fleet)."  This makes it easier when searching for the document later. 

### How to conduct a customer meeting

#### Before a customer call (48hrs)

- Check the calendar invite 48hrs before the meeting to determine if the customer has accepted the invitation.
- If the customer has not accepted the invitation, reach out to confirm attendance (e.g., EAs, email, Slack).
- Test the Zoom Meeting link to make sure that it is working.
- Make sure that agenda documents are attached and accessible to meeting attendees (as appropriate to the situation).

#### Day of the customer call

- Join the meeting two to three minutes before the start time.

#### Missed customer call

- If the customer does not join the call after three minutes, contact the customer with
  - Slack, if we have a shared channel.
  - email, using the email address from the calendar invite.
  - LinkedIn, send a direct message.
  - phone, try finding their number to text and/or call (as appropriate to the device type: landline vs. cell phone).
  - an alternative date and time. Suggest two to three options from which the customer can choose.
    - Confirm that contact information is accurate and that the customer can receive and access meeting invites.

> **Customer etiquette**. When communicating with the customer, remember to approach the situation with empathy. Anything could have happened.

### Next steps after a customer conversation 

After a customer conversation, it can sometimes feel like there are 1,001 things to do, but it can be hard to know where to start.  Here are some tips: 

## Support process

This section outlines Fleet's customer and community support process.
- Basic help desk resolution and service delivery -> the CS team handles these with occasional support from L2.
- In-depth technical support -> the CS team with L2 oncall technician.
- Expert product and service support -> the CS team liaises with L2 and L3 on-call technicians.


If possible, the resulting solution should be more straightforward in each case in the documentation and/or the FAQs.

The support process is accomplished via on-call rotation and the weekly on-call retro meeting.

The on-call engineer holds responsibility for responses to technical Slack comments, Slack threads, and GitHub issues raised by customers and the community, which the Customer Success team cannot handle.

Slack messages have a 24-hour SLA, and the Slack channel should have a notice at the top explaining so.

The weekly on-call retro at Fleet provides time to discuss highlights and answer the following questions about the previous week's on-call:

1. What went well?

2. What could have gone better?

3. What should we remember next time?

This way, the Fleet team can constantly improve the effectiveness and experience during future on-call rotations.

### Customer support responses

When a customer encounters an unexpected error on fleetdm.com, it is important that we contact them quickly to resolve the issue. 

If you need to reach out to a customer to resolve an error, you can use one of the templates below. The full list of response email templates is available in this [Google doc](https://docs.google.com/document/d/1-DvPSBnFbsa2hlr02rAGy2GBTwE0Gx03jV94AStTYf4/edit).

#### Error while purchasing a Fleet Premium license

"Hi [company name] team, I just noticed you ran into an error signing up for a license key on fleetdm.com. 

I’m so sorry about that! Were fixing the issue now and I’ve refunded your payment and bumped your license to add an additional three hosts for the year as well.

I’ll let you know when your account is sorted and follow up with instructions to access your Fleet Premium licenses.

Thanks for bearing with us, please don’t hesitate to reach out if you have any questions about this, Fleet, osquery, or anything else!"


#### Error while running a live query in Fleet Sandbox

"Hey there, the team and I just noticed you experienced a 500 error that may have affected your experience running a live query on https://fleetdm.com/.

I wanted to personally apologize for our mistake and let you know we're looking into the problem. We’ll provide an update on the underlying fix and track an (anonymized) bug on https://github.com/fleetdm/fleet so you can follow the progress in the open.

Thank you for trying Fleet!"

## Customer requests
Locate the relevant issue or create it if it doesn't already exist (to avoid duplication, be creative when searching GitHub for issues - it can often take a couple of tries with different keywords to find an existing issue). 

When creating a new issue, make sure the following:
- Make sure the issue has a "customer request" label.
- "+" prefixed labels (e.g., "+more info please") indicate we are waiting on an answer from an external community member who does not work at Fleet or that no further action is needed from the Fleet team until an external community member, who doesn't work at Fleet, replies with a comment. At this point, our bot will automatically remove the +-prefixed label.
- Is the issue straightforward and easy to understand, with appropriate context (default to public: declassify into public issues in fleetdm/fleet whenever possible)?
- Is there a key date or timeframe that the customer hopes to meet?  If so, please post about that in #g-product with a link to the issue, so the team can discuss it before committing to a time frame.
- Have we provided a link to that issue for the customer to remind everyone of the plan and for the sake of visibility, so other folks who weren't directly involved are up to speed  (e.g., "Hi everyone, here's a link to the issue we discussed on today's call: […link…](https://omfgdogs.com)")?

## Assistance from engineering

Customer team members can reach the engineering oncall for assistance by writing a message with `@oncall` in the `#help-engineering` channel of the Fleet Slack.

## Runbook

### Responding to a request to change a credit card number
To change a customer's credit card number, you identify the customer's account email, log into Stripe, and choose the subscriptions associated with that account. You can then email the customer an invoice, and they can update the payment method on file.

## Customer codenames
Occasionally, we will need to track public issues for customers that wish to remain anonymous on our public issue tracker. To do this, we choose an appropriate minor planet name from this [Wikipedia page](https://en.wikipedia.org/wiki/List_of_named_minor_planets_(alphabetical)) and create a label which we attach to the issue and any future issues for this customer.

## Generating a trial license key
Fleet's self-service license dispenser is the best way to generate trial license keys for small deployments of Fleet Premium.

To generate a trial license key for a larger deployment, [create an opportunity issue](https://github.com/fleetdm/confidential/issues/new/choose) for the customer and follow the instructions in the issue for generating a trial license key.

## Documentation updates
Occasionally, users will email or Slack questions about product usage. We will track these requests and occasionally update our documentation to simplify things for our users. We have a Zapier integration that will automatically create an entry in our customer questions Google doc (in Slack, right-click on the customer question and select send to Zapier). At the end of the week, one of our team members will take each request in the spreadsheet and make any helpful documentation updates to help prevent similar questions in the future.

> **Note** When submitting any pull request that changes Markdown files in the docs, request an editor review from Desmi Dizney.

## Customer contracts
Fleet's subscription agreement is available at [fleetdm.com/terms](https://fleetdm.com/terms). 

Fleeties can find a summary of contract terms in the relevant [customer's Salesforce opportunity.](https://fleetdm.lightning.force.com/lightning/o/Opportunity/list?filterName=Recent)

## Customer DRI change
Sometimes there is a change in the champion within the customer's organization.
1. Get an introduction to the new DRIs including names, roles, contact information.
2. Make sure they're in the Slack channel.
3. Invite them to the *Success* meetings.
4. In the first meeting understand their proficiency level of osquery.
    1. Make sure the meeting time is still convenient for their team. 
    2. Understand their needs and goals for visibility.
    3. Offer training to get them up to speed.
    4. Provide a white glove experience.


## Contract glossary

| Term           | Definition                                                |
|:---------------|:----------------------------------------------------------|
| Effective date | The start date for the subscription service. |
| Close date | The date the last party to the contract signed the agreement. |
| Invoice date | The date that Fleet sent the invoice to the customer. |

## Sales

The Fleet sales team embodies [our values](https://fleetdm.com/handbook/company#values) in every aspect of our work. Specifically, we continuously work to overperform and achieve strong results. We prioritize efficiency in our processes and operations. We succeed because of transparent, cross-functional collaboration. We are committed to hiring for and celebrating diversity, and we strive to create an environment of inclusiveness and belonging for all. We embrace a spirit of iteration, understanding that we can always improve.

### Outreach one-pager

Our one-pager offers a summary of what Fleet does. It can help stakeholders become familiar with the company and product while also being a useful tool the Growth team uses for sales outreach. Find Fleet's outreach one-pager in this [Google Doc](https://docs.google.com/presentation/d/1GzSjUZj1RrRBpa_yHJjOrvOTsldQQKfq927vpKP1lpU/edit?usp=sharing).

### Intro deck

Fleet's intro deck adds additional detail to our pitch. Find it in [Google Slides](https://docs.google.com/presentation/d/1GzSjUZj1RrRBpa_yHJjOrvOTsldQQKfq927vpKP1lpU/edit?usp=sharing).

### Intro video

Fleet's intro video shows how to get started with Fleet as an admin. Find it on [YouTube](https://www.youtube.com/watch?v=rVxSgvKjrWo).

### SOC 2

You can find a copy of Fleet's SOC 2 report in [Google Drive](https://drive.google.com/file/d/1B-Xb4ZVmZk7Fk0IA1eCr8tCVJ-cfipid/view?usp=drivesdk).  In its current form, this SOC 2 report is intended to be shared only with parties who have signed a non-disclosure agreement with Fleet.

You can learn more about how Fleet approaches security in the [security handbook](https://fleetdm.com/handbook/security) or in [Fleet's trust report](https://fleetdm.com/trust).

### Our lead handling and outreach approach

Fleet's main source for prospects to learn about the company and its offerings is our website, fleetdm.com. There are many places across the website for prospects to ask for more information, request merchandise, try the product and even purchase licenses directly. If the user experience in any of these locations asks for an email address or other contact information, Fleet may use that contact information for follow-up, including sales and marketing purposes. That contact information is for Fleet's sole use, and we do not give or sell that information to any third parties.

In the case of a prospect or customer request, we strive to adhere to the following response times:
- Web chat: 1 hour response during working hours, 8 hours otherwise
- Talk to an expert: prospects can schedule chats via our calendar tool
- All other enquiries: 1-2 days

Fleet employees can find other expectations for action and response times in this [internal document](https://docs.google.com/presentation/d/104-TRXlY55g303q2xazY1bpcDx4dHqS5O5VdJ05OwzE/edit?usp=sharing)

### Salesforce lead status flow

To track the stage of the sales cycle that a lead is at, we use the following standardized lead statuses to indicate which stage of the sales process a lead is at.
|Lead status                 | Description                                         |
|:-----------------------------|:----------------------------------------------------|
| New | Default status for all new leads when initially entered into Salesforce. We have an email or LinkedIn profile URL for the lead, but no established intent. The lead is just a relevant person to reach out to.|
| New enriched | Fleet enriched the lead with additional contact info.|
| New MQL | Lead has been established as a marketing qualified lead, meeting company size criteria.|
| Working to engage | Fleet (often Sales development representative-SDR) is working to engage the lead. |
| Engaged | Fleet has successfully made contact with the lead |
| Meeting scheduled | Fleet has scheduled a meeting with the lead. |
| Working to convert | Not enough info on Lead's Budget, Authority, Need and Timing (BANT) to be converted into an opportunity. |
| Closed nurture | Lead does not meet BANT criteria to be converted to an opportunity, but we should maintain contact with the lead as it may be fruitful in the future. |
| Closed do not contact | Lead does not meet BANT criteria for conversion, and we should not reach out to them again. |
| SAO Converted | Lead has met BANT criteria and successfully converted to an opportunity. |

At times, our sales team will reach out to prospective customers before they come to Fleet for information. Our cold approach is inspired by Daniel Grzelak’s (Founder, investor, advisor, hacker, CISO) [LinkedIn post](https://www.linkedin.com/posts/danielgrzelak_if-you-are-going-to-do-a-cold-approach-be-activity-6940518616459022336-iYE7). The following are the keys to an engaging cold approach. Since cold approaches like these can be easily ignored as mass emails, it’s important to personalize each one. 

- Research each prospect.
- Praise what’s great about their company.
- Avoid just stating facts about our product.
- State why we would love to work with them.
- Ask questions about their company and current device management experience.
- Keep an enthusiastic and warm tone.
- Be personable.
- Ask for the meeting with a proposed time.

Importantly, when we interact with CISOs or, for that matter, any member of a prospective customer organization, we adhere to the principles in this [LinkedIn post](https://www.linkedin.com/pulse/selling-ciso-james-turner). Specifically:

- Be curteous
- Be honest
- Show respect
- Build trust
- Grow relationships
- Help people

### Sales team writing principles

When writing for the Sales team, we want to abide by the following principles in our communications.

#### Maintain naming conventions

Maintain naming conventions so people can expect what fields will look like when revisiting automations outside of Salesforce. This helps them avoid misunderstanding jargon and making mistakes that break automated integrations and cause business problems. One way we do this is by using sentence case where only the first word is capitalized (unless it’s a proper noun). See the below examples.

| Good job! ✅          | Don't do this. ❌    |
|:----------------------|:---------------------|
| Bad data              | Bad Data

#### Be explicit

Being explicit helps people to understand what they are reading and how to use terms for proper use of automations outside of Salesforce. In the case of acronyms, that means expanding and treating them as proper nouns. Note the template for including acronyms is in the first column below.

| Good job! ✅          | Don't do this. ❌    |
|:----------------------|:---------------------|
| Do Not Contact (DNC)  | DNC



### Salesforce contributor experience checkups

In order to maintain a consistent contributor experience in Salesforce, we log in to make sure the structure of Salesforce data continues to look correct based on processes started elsewhere. Then we can look and see that the goals we want to achieve as a business are in line with our view inside Salesforce by conducting the following checkup. Any discrepancies between how information is presented in Salesforce and what should be in there per this ritual should be flagged so that they can be fixed or discussed.

1. Make sure the default tabs for a standard user include a detailed view of contacts, opportunities, accounts, and leads. No other tabs should exist.

2. Click the accounts tab and check for the following: 

* The default filter is Customers when you click on the accounts tab. Click on an account to continue.
* Click on a customer and make sure billing address, parent account, LinkedIn company URL, CISO employees (#), employees, and industry appear first at the top of the account.
* "Looking for meeting notes" reminder should appear on the right of the screen.  
* Useful links section should include links to Purchase Orders (POs), signed subscription agreements, invoices sent, meeting notes, and signed NDA. Clicking these links should search the appropriate repository for the requested information pertaining to the customer.
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


## Rituals

The following table lists the Customer's group's rituals, frequency, and Directly Responsible Individual (DRI).

| Ritual                       | Frequency                | Description                                         | DRI               |
|:-----------------------------|:-----------------------------|:----------------------------------------------------|-------------------|
| Overnight customer feedback  | Daily | Check Slack for customer feedback that occurred outside of usual business hours.| Kathy Satterlee       |  
| Customer Slack channel monitoring | Daily | Continuously monitor Slack for customer feedback, feature requests, reported bugs, etc., and respond in less than an hour.   | Kathy Satterlee        |
| Customer follow-up | Daily | Follow-up and tag appropriate personnel for follow-up on customer items in progress and items that remain unresolved. | Kathy Satterlee |
| Internal follow-up | Daily | Go through Fleet's internal Slack channels to check for any relevant new information or tasks from other teams. | Kathy Satterlee |
| [Customer voice](https://docs.google.com/document/d/15Zn6qdm9NyNM7C9kLKtvgMKsuY4Hpgo7lABOBhw7olI/edit?usp=sharing) | Weekly | Prepare and review the health and latest updates from Fleet's key customers and active proof of concepts (POCs), plus other active support items related to community support, community engagement efforts, contact form or chat requests, self-service customers, outages, and more. | Kathy Satterlee  |
| Stand-up | Weekly | Meet with the Engineering team three to four times a week to share information and prioritize issues. | Kathy Satterlee |
| Customer request backlog | Weekly | Check-in before product office hours to make sure that all information necessary has been gathered before presenting customer requests and feedback to the Product team. | Kathy Satterlee |
| Product office hours | Weekly | Present and advocate for requests and ideas brought to Fleet's attention by customers that are interesting from a product perspective. | Kathy Satterlee |
| Customer meetings | Weekly | Check-in on how product and company are performing, provide updates on new product features or progress on customer requests.  These are private meetings with one meeting for each individual commercial customer. | Kathy Satterlee |
| Release announcements | Every three weeks | Update customers on new features and resolve issues in an upcoming release. | Kathy Satterlee        |
| Sales huddle | Weekly | Agenda: Go through every [open opportunity](https://fleetdm.lightning.force.com/lightning/o/Opportunity/list?filterName=00B4x00000CTHZIEA5) and update the next steps. | Alex Mitchell
[Salesforce contributor experience checkup](#salesforce-contributor-experience-checkups)| Monthly | Make sure all users see a detailed view of contacts, opportunities, accounts, and leads. | Nathan Holliday |
| Lead pipeline review  | Weekly | Agenda: Review leads by status/stage; make sure SLAs are met. | Alex Mitchell 


## Slack channels
The following [Slack channels are maintained](https://fleetdm.com/handbook/company#group-slack-channels) by this group:

| Slack channel                       | [DRI](https://fleetdm.com/handbook/company#group-slack-channels)    |
|:------------------------------------|:--------------------------------------------------------------------|
| `#g-customers`           | Zay Hanlon                                                     |
| `#fleet-at-*` _(customer channels)_ | Kathy Satterlee                                                     |
| `#g-sales`                     | Alex Mitchell |
| `#_from-prospective-customers` | Alex Mitchell |


<meta name="maintainedBy" value="alexmitchelliii">
<meta name="title" value="🐋 Customers">



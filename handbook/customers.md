# Customers

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

## Customer contracts
Fleet's subscription agreement is available at [fleetdm.com/terms](https://fleetdm.com/terms). 

Fleet employees can find a summary of contract terms [here](https://docs.google.com/spreadsheets/d/1gAenC948YWG2NwcaVHleUvX0LzS8suyMFpjaBqxHQNg/edit?usp=sharing).

## Rituals

The following table lists the Customer's group's rituals, frequency, and Directly Responsible Individual (DRI).

| Ritual                       | Frequency                | Description                                         | DRI               |
|:-----------------------------|:-----------------------------|:----------------------------------------------------|-------------------|
| Overnight customer feedback  | Daily | Check Slack for customer feedback that occurred outside of usual business hours.| Tony Gauda       |  
| Customer Slack channel monitoring | Daily | Continuously monitor Slack for customer feedback, feature requests, reported bugs, etc., and respond in less than an hour.   | Tony Gauda        |
| Customer follow-up | Daily | Follow-up and tag appropriate personnel for follow-up on customer items in progress and items that remain unresolved. | Tony Gauda |
| Internal follow-up | Daily | Go through Fleet's internal Slack channels to check for any relevant new information or tasks from other teams. | Tony Gauda |
| Customer debriefs | Weekly | Discuss customer questions, requests, and issues with the Product team. | Tony Gauda  |
| Stand-up | Weekly | Meet with the Engineering team three to four times a week to share information and prioritize issues. | Tony Gauda |
| Customer request backlog | Weekly | Check-in before product office hours to make sure that all information necessary has been gathered before presenting customer requests and feedback to the Product team. | Tony Gauda |
| Product office hours | Weekly | Present tickets and items brought to Fleet's attention by customers that are interesting from a product perspective and advocate for customer requests. | Tony Gauda |
| Customer meetings | Weekly | Check-in on how product and company are performing, provide updates on new product features or progress on customer requests.  These are private meetings with one meeting for each individual commercial customer. | Tony Gauda |
| Product review | Every three weeks | Meet with the Product team to gain product pipeline visibility in order to gather info on new features and fixes in the next release. | Tony Gauda |
| Release announcements | Every three weeks | Update customers on new features and resolve issues in an upcoming release. | Tony Gauda        |
| Sales huddle | Bi-monthly | Meet with Sales team to gain sales pipeline visibility for business intelligence and product development purposes, such as testing scalability for potential customer's needs, predicting product success obstacles, etc. | Tony Gauda |
| Advisory meetings | Quarterly | Peer network feedback and Q& with other industry professionals. Mostly discussions on the refining process. | Tony Gauda |

## Slack channels
The following [Slack channels are maintained](https://fleetdm.com/handbook/company#group-slack-channels) by this group:

| Slack channel                       | [DRI](https://fleetdm.com/handbook/company#group-slack-channels)    |
|:------------------------------------|:--------------------------------------------------------------------|
| `#g-customer-engineering`           | Tony Gauda                                                          |
| `#fleet-at-*` _(customer channels)_ | Tony Gauda                                                          |



<meta name="maintainedBy" value="tgauda">
<meta name="title" value="🎈 Customers">


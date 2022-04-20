# Customer engineering

## Customer success

### Scheduling meetings with customers
To schedule an ad hoc meeting with a Fleet customer, use the ["Customer meeting" Calendly link](https://docs.google.com/document/d/1tE-NpNfw1icmU2MjYuBRib0VWBPVAdmq4NiCrpuI0F0/edit#heading=h.v47bs6uo0jpk).

### Missed Zoom Meeting | Etiquette with Customers
This is a tutorial on how to respond when someone from outside the company misses a call.

#### 48 hours before a meeting

L1: Check the calendar invite to determine if the customer has accepted the invitation.

L2: If the customer has not accepted the invitation please reach out through appropriate channels to confirm attendance (Ex: EAs, Email, Slack).

L3: Test the Zoom Meeting link to ensure that it is working.

L4: Ensure that agenda documents are attached and accessible to meeting attendees (as appropriate to the situation).

#### Day of meeting

L1: Join meeting 2-3 minutes prior to the meeting start time.

L2: If the customer does not join the call after 3-minutes, contact the customer with:

- Slack, if we have a shared channel.
- Email, using the email address from the calendar invite.
- LinkedIn, send a direct message.
- Phone, try finding their number to text and/or call (as appropriate to the device type: landline vs cell phone).

L3: In these communications to the customer remember to approach the situation with empathy. Anything could have happened.

L4: Be ready to supply an alternative date and time to reschedule the call. Suggest 2-3 options for the customer to choose from.

L5: Ensure that contact information is accurate and that meeting invites can be received and accessed by the customer.

L6: Repeat back to the customer the newly agreed upon date and time, as well as the contact information.

L7: Congratulations, you’re ready to set up a new call.

### Next steps after a customer conversation
After a customer conversation, it can sometimes feel like there are 1001 things to do, but it can be hard to know where to start.  Here are some tips:


## Support process

This section outlines the customer and community support process at Fleet.

- L1: Basic help desk resolution and service delivery -> CS team handles these with occasional support from L2
- L2: In-depth technical suppport -> CS team with L2 oncall technician
- L3: Expert product and service support -> CS team liases with L2 and L3 oncall technician.

In each case, if possible, the resulting solution should be made more clear in the documentation and/or the FAQs.

The support process is accomplished via on-call rotation and the weekly on-call retro meeting.

The on-call engineer is responsible for responding to technical Slack comments, Slack threads, and GitHub issues raised by customers and the community which cannot handled by the Customer Success team.

Slack messages have a 24 hour SLA and the Slack channel should have a notice at the top explaining so.

The weekly on-call retro at Fleet provides time to discuss highlights and answer the following questions about the previous week's on-call:

1. What went well?

2. What could have gone better?

3. What should we remember next time?

This way, the Fleet team can constantly improve the effectiveness and experience during future on-call rotations.

### For customer requests
Locate the appropriate issue, or create it if it doesn't already exist. (To avoid duplication, be creative when searching GitHub for issues - it can often take a couple of tries with different keywords to find an existing issue.) 

When creating a new issue, ensure the following:

- Make sure the issue has a "customer request" label.
- "+" prefixed labels (e.g., "+more info please") indicate we are waiting on an answer from an external community member who does not work at Fleet, or otherwise that no further action is needed from the Fleet team until an external community member, who doesn't work at Fleet, replies with a comment. (At which point our bot will automatically remove the +-prefixed label.)
- Is the issue clear and easy to understand, with appropriate context?  (Default to public: declassify into public issues in fleetdm/fleet whenever possible)
- Is there a key date or timeframe that the customer is hoping to meet?  If so, please post about that in #g-product with a link to the issue, so the team can discuss before committing to a time frame.
- Have we provided a link to that issue for the customer to remind everyone of the plan, and for the sake of visibility, so other folks who weren't directly involved are up to speed?  (e.g. "Hi everyone, here's a link to the issue we discussed on today's call: […link…](https://omfgdogs.com)")


## Runbook

### Responding to a request to change a credit card number
To change a customer credit card number you identify the customer's account email, log into Stripe, and choose the subscriptions associated with that account. You can then email the customer an invoice and they can update the payment method on file.


## Incident postmortems
At Fleet, we take customer incidents very seriously. After working with customers to resolve issues, we will conduct an internal postmortem to determine any documentation or coding changes to prevent similar incidents from happening in the future. Why? We strive to make Fleet the best osquery management platform globally, and we sincerely believe that starts with sharing lessons learned with the community to become stronger together.

At Fleet, we do postmortem meetings for every production incident, whether it's a customer's environment or on fleetdm.com.


## Customer codenames
Occasionally we will need to track public issues for customers that wish to remain anonymous on our public issue tracker. To do this, we choose an appropriate minor planet name from this [Wikipedia page](https://en.wikipedia.org/wiki/List_of_named_minor_planets_(alphabetical)) and create a label which we attach to the issue and any future issues for this customer.


## Generating a trial license key
Fleet's self-service license dispenser is the best way to generate trial license keys for small deployments of Fleet Premium.

To generate a trial license key for a larger deployment, [create an opportunity issue](https://github.com/fleetdm/confidential/issues/new/choose) for the customer and follow the instructions in the issue for generating a trial license key.


## Documentation updates

Occasionally users will either email or Slack questions about product usage. We will track these requests and occasionally update our documentation to simplify things for our users. We have a Zapier integration that will automatically create an entry in our customer questions Google doc (in Slack, right-click on the customer question and select send to Zapier). At the end of the week, one of our team members will take each request in the spreadsheet and make any helpful documentation updates to help prevent similar questions in the future.


## Fleet rituals for customers

The following rituals are engaged in by the  directly responsible individual (DRI) and at the frequency specified for the ritual.

| Ritual                       | Description                                         | DRI               |
|:-----------------------------|:----------------------------------------------------|-------------------|
| Overnight customer feedback  | Check Slack daily for customer feedback that occurred outside of usual business hours.| Tony Gauda       |  
| Customer Slack channel monitoring | Continuously monitor Slack for customer feedback, feature requests, reported bugs, etc. and respond <1 hour.       | Tony Gauda        |
| Weekly customer debrief | Discuss customer questions, requests and issues with the product team. | Tony Gauda  |
| Stand up ritual | Information sharing and issue prioritization meeting with the engineering team, occurs 3 - 4 times per week. | Tony Gauda |
| Daily follow up | Follow up and tag appropriate personnel for follow up on customer items in progress and items that remain unresolved. | Tony Gauda |
| Customer request backlog | Check-in before product office hours to ensure that all information necessary has been gathered before presenting customer requests and feedback to the product team. Occurs every week on the same day as the product office hours ritual. | Tony Gauda |
| Product office hours | Present tickets and items brought to Fleet's attention by customers that are interesting from a product perspective and advocate for customer requests. Occurs weekly. | Tony Gauda |
| Sales huddle | Meet with sales team every two weeks to gain sales pipeline visibility for business intelligence and product development purposes, such as testing scalability for potential customer's needs, predicting product success obstacles, etc. | Tony Gauda |
| Product review | Occurs every three weeks, meet with product team to gain product pipeline visibility in order to gather info on new features and fixes in the next release. | Tony Gauda |
| Release announcements | Update customers on new features and resolved issues in an upcoming release, happens every three weeks. | Tony Gauda        |
| Customer meetings | Check-in on how product and company is performing, provide updates on new product features or progress on customer requests.  Typically occur every Thursday, these are private meetings with one meeting for each individual commercial customer. | Tony Gauda |
| Advisory meetings | Peer network feedback and Q& with other industry professionals. Mostly discussions on refining process. Vary somewhat in frequency, but occur at least once per quarter. | Tony Gauda |
| Internal follow up ritual | Go through Fleet's internal Slack channels to check for any relevant new information or tasks from other teams. | Tony Gauda |


## Slack channels

The following [Slack channels are maintained](https://fleetdm.com/handbook/company#group-slack-channels) by this group:

| Slack channel                       | [DRI](https://fleetdm.com/handbook/company#group-slack-channels)    |
|:------------------------------------|:--------------------------------------------------------------------|
| `#g-customer-engineering`           | Tony Gauda                                                          |
| `#fleet-at-*` _(customer channels)_ | Tony Gauda                                                          |
| `#help-sell`                        | Andrew Bare                                                         |
| `#_from-prospective-customers`      | Andrew Bare                                                         |


<meta name="maintainedBy" value="tgauda">


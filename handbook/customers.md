# Customer experience

## Customer success

### Scheduling meetings with customers
To schedule an ad hoc meeting with a Fleet customer, use the ["Customer meeting" Calendly link](https://docs.google.com/document/d/1tE-NpNfw1icmU2MjYuBRib0VWBPVAdmq4NiCrpuI0F0/edit#heading=h.v47bs6uo0jpk).

### Next steps after a customer conversation
After a customer conversation, it can sometimes feel like there are 1001 things to do, but it can be hard to know where to start.  Here are some tips:

## Support process

This section outlines the customer and community support process at Fleet.

L1: Basic help desk resolution and service delivery -> CS team handles these with occasional support from L2
L2: In-depth technical suppport -> CS Team with L2 Oncall Technician
L3: Expert product and service support -> CS team liases with L2 and L3 Oncall Technician.

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

## Incident Postmortems
At Fleet, we take customer incidents very seriously. After working with customers to resolve issues, we will conduct an internal postmortem to determine any documentation or coding changes to prevent similar incidents from happening in the future. Why? We strive to make Fleet the best osquery management platform globally, and we sincerely believe that starts with sharing lessons learned with the community to become stronger together.

## Customer codenames
Occasionally we will need to track public issues for customers that wish to remain anonymous on our public issue tracker. To do this, we choose an appropriate minor planet name from this [Wikipedia page](https://en.wikipedia.org/wiki/List_of_named_minor_planets_(alphabetical)) and create a label which we attach to the issue and any future issues for this customer.

## Generating a trial license key
Fleet's self-service license dispenser (coming soon) is the best way to generate trial license keys for Fleet Premium.

In the meantime, to generate a trial license key, [create an opportunity issue](https://github.com/fleetdm/confidential/issues/new/choose) for the customer and follow the instructions in the issue for generating a trial license key.

<meta name="maintainedBy" value="tgauda">


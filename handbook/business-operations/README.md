# Business Operations
Welcome to the Business Operations (or BizOps) handbook! The BizOps handbook serves as an informational, how-to guide for Fleeties, and supports Fleet's mission by enabling team members to make meaningful contributions in alignment with our values. If you can not find what you are looking for please use the [intake process](https://fleetdm.com/handbook/business-operations#intake) to make a request of the team.

The BizOps group works together as one team, made up of People Operations (POps), Finance Operations (FinOps), Legal Operations (LegalOps), IT Operations (ITOps), and Revenue Operations (RevOps).

## Intake
To make a request of the business operations department, [create an issue using one of our issue templates](https://github.com/fleetdm/confidential/issues/new/choose).  If you don't see what you need, or you are unsure, [create a custom request issue](https://github.com/fleetdm/confidential/issues/new/choose) and someone in business operations will reply within 1 business day.

> If you're not sure that your request can wait that long, then please ask for urgent help in our group Slack channel: `#g-business-operations`.  Only use this approach or at-mention contributors in business operations directly in urgent situations.  Otherwise, create an issue.

## Kanban
Any Fleet team member can [view the 🔦#g-business-operations kanban board](https://app.zenhub.com/workspaces/-g-business-operations-63f3dc3cc931f6247fcf55a9/board?sprints=none) (confidential) for this department, including pending tasks in the active sprint and any new requests.

## Levels of confidentiality

- *Public*   _(share with anyone, anywhere in the world)_
- *Confidential*  _(share only with team members who've signed an NDA, consulting agreement, or employment agreement)_
- *Classified*  _(share only with founders of Fleet, business operations, and/or the people involved.  e.g., US social security numbers during hiring)_

> TODO: extrapolate to "why this way" page

## Rituals
The following table lists this group's rituals, frequency, and Directly Responsible Individual (DRI).

| Ritual                       | Frequency                | Description                                         | [DRI](https://fleetdm.com/handbook/company/why-this-way#why-group-slack-channels)               |
|:-----------------------------|:-----------------------------|:----------------------------------------------------|-------------------|
| Key review | every three weeks | Every release cycle, a key review deck is prepared and presented. | Head of Business Operations |
| Informing managers about hours worked | Weekly |  See [Informing managers about hours worked"](https://fleetdm.com/handbook/business-operations#informing-managers-about-hours-worked). | Head of Business Operations |
| Payroll | Monthly before payroll runs | Every month, Mike McNeil audits the payroll platforms for accuracy. | Head of Business Operations |
| US contractor payroll | Monthly | Sync contractor hours to payments in Gusto and run payroll for the month. | Head of Business Operations |
| Commission payroll | Monthly | Use the [commission calculator](https://docs.google.com/spreadsheets/d/1vw6Q7kCC7-FdG5Fgx3ghgUdQiF2qwxk6njgK6z8_O9U/edit#gid=0) to determine the commission payroll to be run in Gusto. | Taylor Hughes |
| Revenue report | Weekly | At the start of every week, check the Salesforce reports for past due invoices, non-invoiced opportunities, and past due renewals.  Report any findings to in the `#g-sales` channel by mentioning Alex Mitchell and Mike McNeil. | Taylor Hughes |
| Monthly accounting | Monthly | Create [the monthly close GitHub issue](https://fleetdm.com/handbook/business-operations#intake) and walk through the steps. | Nathanael Holliday |
| Quarterly grants | Quarterly | Create [the quarterly close GitHub issue](https://fleetdm.com/handbook/business-operations#intake) and walk through the steps. | Nathanael Holliday |
| AP invoice monitoring | Weekly | Look for new accounts payable invoices and make sure that Fleet's suppliers are paid. | Nathanael Holliday | 
| Tax preparation | Annually on the first week of March | Provide information to tax team with Deloitte and assist with filing and paying state and federal returns | Nathanael Holliday | 
| Vanta check | Monthly | Look for any new actions in Vanta due in the upcoming months and create issues to ensure they're done on time. | Nathan Holliday |
| Investor reporting | Quarterly | Provide updated metrics for CRV in Chronograph. | Nathanael Holliday |
| Applicant forwarding | Daily | Whenever an application notification arrives in the BizOps slack channel, forward this notification to the hiring channel for that position. | Joanne Stableford |

<!--
TODO: Move to CEO handbook page

| Weekly update | Weekly | On Friday, Mike McNeil posts a single message in #general [based on the message from the previous week, saving a copy for reference _next_ week](https://docs.google.com/spreadsheets/d/1Hso0LxqwrRVINCyW_n436bNHmoqhoLhC8bcbvLPOs9A/edit#gid=0).  This weekly update recognizeseach of the week's on-duty people for each of the on-call rotations, along with any hiring and departure announcements, and information about ongoing onboardings and open positions.  | Mike McNeil |
| CEO inbox sweep | Daily unless OOO | Mike McNeil does a morning sweep of the CEO's inbox to remove spam and grab action items. | Mike McNeil |
| Calendar audit | Daily | Daily Mike McNeil audits CEOs calendar and set notes for meetings. | Mike McNeil |
| [Workiversaries](#workiversaries) | Weekly/PRN | Mike McNeil posts in `#g-people` and tags @mikermcneil about any upcoming workiversaries. | Mike McNeil |
| Prepare Mike and Sid's 1:1 doc | Bi-weekly | Run through the document preparation GitHub issue for Mike's call with Sid. | Nathanael Holliday |
-->

<!--
Note: These are out of date, but retained for future reference:

| Weekly update reminder | Weekly | Early Friday mornings (US time), a Slack bot posts in the `#g-e` channel reminding directly responsible individuals for KPIs to add their metrics for the current week in ["KPIs"](https://docs.google.com/spreadsheets/d/1Hso0LxqwrRVINCyW_n436bNHmoqhoLhC8bcbvLPOs9A/edit) before the end of the day. | N/A |
| Access revalidation | Quarterly | Review critical access groups to make sure they contain only relevant people. | Mike McNeil |
| 550C update | Annually | File California 550C. | Mike McNeil |
| TPA verifications | Quarterly | Every quarter before tax filing due dates, Mike McNeil audits state accounts to ensure TPA is set up or renewed. | Mike McNeil |
| Brex reconciliation | Monthly | Make sure all company-issued credit card transactions include memos. | Nathanael Holliday |
| Hours update | Weekly | Screenshots of contractor hours as shown in Gusto are sent via Slack to each contractor's manager with no further action necessary if everything appears normal. | Mike McNeil |
| QBO check | Quarterly | The first month after the previous quarter has closed, make sure that QBO is accurate compared to Fleet's records. | Nathanael Holliday | 
| Capital credit reporting | Annually | Within 60 days of the new year, provide financial statements to SVB. | Nathanael Holliday |
| YubiKey adoption | Monthly | Track YubiKey adoption in Google workspace and follow up with those that aren't using it. | Mike McNeil |
| Security policy update | Annually | Update security policies and have them approved by the CEO. | Nathanael Holliday |
| Security notifications check | Daily | Check Slack, Google, Vanta, and Fleet dogfood for security-related notifications. | Nathanael Holliday |
| Changeset for onboarding issue template | Quarterly | pull up the changeset in the onboarding issue template and send out a link to the diff to all team members by posting in Slack's `#general` channel. | Mike McNeil |
| Recruiting progress checkup | Weekly | Mike McNeil looks in the [Fleeties spreadsheet](https://docs.google.com/spreadsheets/d/1OSLn-ZCbGSjPusHPiR5dwQhheH1K8-xqyZdsOe9y7qc/edit#gid=0) and reports on each open position. | Mike McNeil |
| Investor and advisor updates | PRN | Mike McNeil tracks the last contact with investors and coordinates outreach with CEO. | Mike McNeil |
| MDM device enrollment | Quarterly | Provide export of MDM enrolled devices to the ops team. | Luke Heath |

-->

### Weekly updates

Please see [handbook/company/ceo-handbook#weekly-updates](https://fleetdm.com/handbook/company/ceo-handbook#weekly-updates)

### Key reviews
Every release cycle, each department leader prepares a [key review deck](https://about.gitlab.com/handbook/key-review/#purpose) and presents it to the CEO. In this deck, the department will highlight KPI metrics (numbers measuring everyday excellence) and progress of timebound goals for a particular quarter (OKRs). The information for creating this deck is located in the ["🌈 Fleet" Google drive](https://drive.google.com/drive/folders/1lizTSi7YotG_zA7zJeHuOXTg_KF1Ji8k) using ["How to create key review"](https://docs.google.com/document/d/1PDwJL0HiCz-KbEGZMfldAYX_aLk5OVAU1MMSgMYYF2A/edit?usp=sharing)(internal doc).

> TODO: extrapolate "key reviews" to fleetdm.com/handbook/company/why-this-way -- maybe a section on "why measure KPIs and set goals?"

### All hands
Every month, Fleet holds a company-wide meeting called the "All hands".

All team members should attend the "All hands" every month.  Team members who cannot attend are expected to watch the [recording](https://us-65885.app.gong.io/conversations?workspace-id=9148397688380544352&callSearch=%7B%22search%22%3A%7B%22type%22%3A%22And%22%2C%22filters%22%3A%5B%7B%22type%22%3A%22CallTitle%22%2C%22phrase%22%3A%22all%20hands%22%7D%5D%7D%7D) within a few days.

"All hands" meetings are [recorded](https://us-65885.app.gong.io/conversations?workspace-id=9148397688380544352&callSearch=%7B%22search%22%3A%7B%22type%22%3A%22And%22%2C%22filters%22%3A%5B%7B%22type%22%3A%22CallTitle%22%2C%22phrase%22%3A%22all%20hands%22%7D%5D%7D%7D) and have [slides](https://drive.google.com/drive/folders/1cw_lL3_Xu9ZOXKGPghh8F4tc0ND9kQeY) available.

## Trust
Fleet is successful because of our customers and community, and those relationships are built on trust.

### Security
Security policies are best when they're alive, in context in how an organization operates.  Fleeties carry Yubikeys, and change control of policies and access control is driven primarily through GitOps and SSO.

Here are a few different entry points for a tour of Fleet's security policies and best practices:
1. [Security policies](https://fleetdm.com/handbook/security/security-policies#security-policies)
2. [Human resources security policy](https://fleetdm.com/handbook/security/security-policies#human-resources-security-policy)
3. [Account recovery process](https://fleetdm.com/handbook/security#account-recovery-process)
4. [Personal mobile devices](https://fleetdm.com/handbook/security#personal-mobile-devices)
5. [Hardware security keys](https://fleetdm.com/handbook/security#hardware-security-keys)
6. More details about internal security processes at Fleet are located on [the Security page](./security.md).

## Directly responsible individuals
Please read ["Why direct responsibility?"](https://fleetdm.com/handbook/company/why-this-way#why-direct-responsibility) to learn more about DRIs.

## Slack channels
These groups maintain the following [Slack channels](https://fleetdm.com/handbook/company/why-this-way#why-group-slack-channels):

| Slack channel                           | [DRI](https://fleetdm.com/handbook/company/why-this-way#why-group-slack-channels)    |
|:----------------------------------------|:--------------------------------------------------------------------|
| `#g-business-operations`                | Joanne Stableford
| `#help-brex`                            | Nathan Holliday
| `#help-classified` _(¶¶)_               | Joanne Stableford
| `#help-onboarding`                      | Mike McNeil
| `#help-manage`                          | Mike McNeil
| `#help-open-core-ventures` _(¶¶)_       | Mike McNeil
| `#_security`                            | Zach Wasserman

## Email relays

There are several special email addresses that automatically relay messages to the appropriate people at Fleet. Each email address meets a minimum response time ("Min RT"), expressed in business hours/days, and has a dedicated, directly responsible individual (DRI) who is responsible for reading and replying to emails sent to that address.  You can see a list of those email addresses in ["Contacting Fleet" (private Google doc)](https://docs.google.com/document/d/1tE-NpNfw1icmU2MjYuBRib0VWBPVAdmq4NiCrpuI0F0/edit#).

> TODO: extrapolate to "why this way" page

## People Operations

### Relocating
When Fleeties relocate, there are vendors that need to be notified of the change. 

Before relocating, please [let the company know in advance](#intake) by following the directions listed in the relevant issue template ("Moving").

### People Operations rituals
#### Celebrations
At Fleet we like to celebrate sinificant milestones for our teammates! We do this in a variety of ways including company-wide slack messaging. If you would prefer we didn't celebrate your milestone, please submit an [intake issue](#intake) to ensure the team has record of your preference.

##### Workiversaries
We're happy you've ventured a trip around the sun with Fleet- let's celebrate! The POps team will post in Slack to highlight your dedication and contribution to Fleet, giving an opportunity for teammates to thare their appreciation of your contribution!
Fleet also [evaluates and (if necessary) updates compensation decisions yearly](#compensation-changes), shortly after the anniversary of a team member's start date.

### Benefits
In this section, you can find information about Fleet's benefit strategies and decisions.

#### Paid time off
What matters most is your results, which are driven by your focus, your availability to collaborate, and the time and consideration you put into your work. Fleet offers all team members unlimited time off. Whether you're sick, you want to take a trip, you are eager for some time to relax, or you need to get some chores done around the house, any reason is a good reason.
For team members working in jurisdictions that require certain mandatory sick leave or PTO policies, Fleet complies to the extent required by law.

##### Taking time off
When you take any time off, you should follow this process:
- Let your manager and team know as soon as possible (i.e., post a message in your team's Slack channel with when and how long).
- Find someone to cover anything that needs covering while you're out and communicate what they need to take over the responsibilities as well as who to refer to for help (e.g., meetings, planned tasks, unfinished business, important Slack/email threads, anything where someone might be depending on you).
- Mark an all-day "Out of office" event in Google Calendar for the day(s) you're taking off or for the hours that you will be off if less than a day. Google Calendar recognizes the event title "OOO" and will give you the option to decline existing and new meetings or just new meetings. You are expected to attend any meetings that you have accepted, so be sure to decline meetings you are not going to attend.
If you can’t complete the above because you need to take the day off quickly due to an emergency, let your manager know and they will help you complete the handoff.
If you ever want to take a day off, and the only thing stopping you is internal (Fleetie-only) meetings, don’t stress. Consider, “Is this a meeting that I can reschedule to another day, or is this a meeting that can go on without me and not interfere with the company’s plans?” Talk to your manager if you’re unsure, but it is perfectly OK to reschedule internal meetings that can wait so that you can take a day off.
This process is the same for any days you take off, whether it's a holiday or you just need a break.
   
##### Holidays
At Fleet, we have team members with various employment classifications in many different countries worldwide. Fleet is a US company, but we think you should choose the days you want to work and what days you are on holiday, rather than being locked into any particular nation or culture's expectation about when to take time off.
When a team member joins Fleet, they pick one of the following holiday schedules:
 - **Traditional**: This is based on the country where you work. Non-US team members should let their managers know the dates of national holidays.
 **Or**
 - **Freestyle**: You have no set schedule and start with no holidays. Then you add the days that are holidays to you.

Either way, it's up to you to make sure that your responsibilities are covered, and that your team knows you're out of the office.

##### New parent leave
Fleet gives new parents six weeks of paid leave. After six weeks, if you don't feel ready to return yet, we'll set up a quick call to discuss and work together to come up with a plan to help you return to work gradually or when you're ready.

#### Retirement contributions
##### US based team members
Commencing in August 2023, Fleet offers the ability for US based team members to contribute to a 401(k) retirement plan directly from their salary. Team members will be auto-enrolled in our plan with Guideline at a default 1% contribution unless they opt out or change their contribution amount within 30 days of commencement. Fleet currently does not match any contributions made by team members to 401(k) plans.

##### Non-US team members
Fleet meets the relevant country's retirement contribution requirements for team members outside the US.

#### Coworking
Your Brex card may be used for up to $500 USD per month in coworking costs. Please get prior approval by making a [custom request to the business operations team](#intake).

### Compensation
Compensation at Fleet is determined by benchmarking using [Pave](https://pave.com). Annual raises are not guaranteed, instead we ensure teammates are compensated fairly based on the role, experience, location, and performance relative to benchmarks.

#### Compensation changes
Fleet evaluates and (if necessary) updates compensation decisions yearly, shortly after the anniversary of a team member's start date. The process for that evaluation and update is:
- On the first Friday of the month, the Head of BizOps posts in the `#help-classified` channel with the list of teammates celebrating anniversaries over the next month.
- On the day of the fleetiversary, the Head of BizOps or executive team member will post in `#random` celebrating the tenure of the teammate.
- The Head of BizOps confers with manager or head of department and prepares compensation benchmarking data and schedules time with the CEO and CTO over an existing 1:1 to discuss if an adjustment needs to be made to compensation.
- During the 1:1 call, founders review benchmarking for role and geography, and decide if there will be an adjustment.
- Head of BizOps posts to slack in `#help-classified` with the decision on compensation changes and effective date, if any.
- If a change is to be made, the Head of BizOps communicates decision to the teammate's people manager, who then communicates to their teammate.
- Head of BizOps updates the respective payroll platform (Gusto or Plane) and [equity spreadsheet](https://docs.google.com/spreadsheets/d/1_GJlqnWWIQBiZFOoyl9YbTr72bg5qdSSp4O3kuKm1Jc/edit?usp=sharing) (internal doc).
- If an additional equity grant is part of this compensation change, the previous equity and new situation is noted in detail in the "Notes" column of the equity plan, and the "Grant started?" column is set back to "todo" to add it to the queue for the next time grants are processed (quarterly).

### Team member onboarding

#### Before the start date

Fleet is open source and anyone can contribute at any time.  Before a core team member's start date, they are welcome to contribute, but not expected to.

##### Recommendations for new teammates
Welcome to Fleet!

1. Understand the company
2. Take the time to get trained
3. Get comfortable with the tools
4. Immerse yourself in the world of device management and cybersecurity.

> Please see Fleet's ["🥽 Recommendations for new teammates"](https://docs.google.com/document/d/1xcnqKB9HHPd94POnZ_7LATiy_VjO2kJdbYx0SAgKVao/edit#).

#### Training expectations
It's [important](https://fleetdm.com/handbook/company/why-this-way#why-the-emphasis-on-training) that every team member at Fleet takes the time to get fully trained and onboarded. 
When a new team member joins Fleet, we create an onboarding issue for them in the [fleetdm/confidential](https://github.com/fleetdm/confidential) repo using this [issue template](https://github.com/fleetdm/confidential/blob/main/.github/ISSUE_TEMPLATE/onboarding.md). 
We want to make sure that the new team member will be able to complete every task in their issue. To make sure the new team member is successful in their onboarding, we customize their issue by commenting on any tasks they won't need to complete.
We believe in taking onboarding and training seriously and that the onboarding template is an essential source of truth and good use of time for every single new hire. If managers see a step that they don't feel is necessary, they should make a pull request to the [onboarding template](https://github.com/fleetdm/confidential/blob/main/.github/ISSUE_TEMPLATE/onboarding.md).

Expectations during onboarding:
- Onboarding time (all checkboxes checked) is a KPI for the business operations team.  Our goal is 14 days or less.
- The first 3 weekdays (excluding days off) for **every new team member** at Fleet is reserved for completing onboarding tasks from the checkboxes in their onboarding issue.  New team members **should not work on anything else during this time**, whether or not other tasks are stacking up or assigned.  It is OK, expected, and appreciated for new team members to **remind their manager and colleagues** of this [important](https://fleetdm.com/handbook/company/why-this-way#why-the-emphasis-on-training) responsibility.
- Even after the first 3 days, during the rest of their first 2 weeks, completing onboarding tasks on time is a new team member's [highest priority](https://fleetdm.com/handbook/company/why-this-way#why-the-emphasis-on-training).

#### Sightseeing tour
During their first day at Fleet, new team members join a sightseeing tour call with the acting Head of People (CEO). During this call, the new team member will participate in an interactive tour of the seven main attractions in our all-remote company, including the primary tools used company-wide, what the human experience is like, and when/why we use them at Fleet.

In this meeting, we'll take a look at:
- Handbook: values, purpose, key pages to pay special attention to
- GitHub issues: the living bloodstream of the company.
- Kanban boards: the bulletin board of quests you can get and how you update status and let folks know things are done.
- Google Calendar: the future.
- Gmail: like any mailbox, full of junk mail, plus some important things, so it is important to check carefully.
- Salesforce: the Rolodex.
- Google Docs: the archives.
- Slack:
  - The "office" (#g-, #general).
  - The walkie talkies (DMs).
  - The watering hole (#oooh-, #random, #news, #help-).

#### Contributor experience training
During their first week at Fleet, every new team member schedules a contributor experience training call with the acting Head of People (CEO). During this call, the new team member will share their screen, and the acting Head of People will:
- make sure emails will get seen and responded to quickly.
- make sure Slack messages will get seen and responded to quickly.
- make sure you know where your issues are tracked, which kanban board you use, and what the columns mean.
- make sure you can succeed with submitting a PR with the GitHub web editor, modifying docs or handbook, and working with Markdown.
- talk about Google calendar.
- give you a quick tour of the Fleet Google drive folder.

<!-- 
TODO: Merge this commented-out stuff with the above

Agenda:
A 60-minute call with Mike where you will share your screen, and she will work with you to...
Make sure Slack messages are going to get seen and responded to quickly and disable email notifications in Slack
Make sure you know where your issues are tracked, which kanban board you use, what the columns mean
Make sure you can succeed with submitting a PR in github.com, modifying docs or handbook, working with Markdown
Make sure emails are going to get seen and responded to quickly (make sure inbox management is going to be productive, talk about filters, unsubscribe)
Make sure you know how to see and subscribe to other team members' calendars and that you can add yourself to an event on someone else's calendar.
A quick tour of the Google drive folder (access look correct? Ok. Give access to executed documents on the shared drive as needed) show how to use “Add to drive” or “favorite,” or just a browser bookmark, so the folder is easily accessible. This is where things go. It's the archive.)
Make sure you know how to share a google doc into the folder for all fleeties to see and access.
A high level overview of the Company values
-->

#### Onboarding retrospective
At the end of their first two weeks of onboarding at Fleet, every new team member schedules a onboarding retro call with the acting Head of People (CEO).  Agenda: 
> Welcome once again to the team! Please tell me about your first few weeks at Fleet. How did your onboarding/training go? What didn't you manage to get to? Anything you weren't sure how to do? Any feedback on how we can make the experience better for Fleet's next hire?

Fleet prioritizes a [bias for action](https://fleetdm.com/handbook/company#ownership).  If possible, apply onboarding feedback to the handbook and issue templates in realtime, during this call.  This avoids backlogging tasks that may just get out of date before we get around to them anyway.

### Tracking hours
Fleet asks US-based hourly contributors to track hours in Gusto, and contributors outside the US to track hours via Pilot.co.

This applies to anyone who gets paid by the hour, including consultants and hourly core team members of any employment classification, inside or outside of the US.

> _**Note:** If a contributor uses their own time tracking process or tools, then it is OK to track the extra time spent tracking!  Contributors at Fleet are evaluated based on their results, not the number of hours they work._

#### Informing managers about hours worked
Every Friday at 1:00pm CT, we gather hours worked for anyone who gets paid hourly by Fleet. This includes core team members and consultants, regardless of employment classification, and regardless whether inside or outside of the United States.

Here's how:
- For every hourly core team member in Gusto or Pilot.co, look up their manager ([who they report to](https://fleetdm.com/handbook/company#org-chart)).
- If any direct report is hourly in Pilot.co and does not submit their hours until the end of the month, still list them, but explain.  (See example below.)
- [Consultants](https://fleetdm.com/handbook/business-operations#hiring) don't have a formal reporting structure or manager. Instead, send their hours worked to the CEO, no matter who the consultant is.

Then, send **the CEO** and **each manager** a direct message in Slack by copying and pasting the following template:

> Here are the hours worked by your direct reports since last Saturday at midnight (YYYY-MM-DD):
> - 🧑‍🚀 Alice Bobberson: 21.25
> - 🧑‍🚀 Charles David: 3.5
> - 🧑‍🚀 Philippe Timebender: (this person's hours will not be available until they invoice at the end of the month)
>
> And here are the hours worked by consultants:
> - 💁 Bombalurina: 0
> - 💁 Jennyanydots: 0
> - 💁 Skimbleshanks: 19
> - 💁 Grizabella: 0
> 
> More info: https://fleetdm.com/handbook/business-operations#informing-managers-about-hours-worked

### Performance feedback
At Fleet, performance feedback is a continuous process. We give feedback (particularly negative) as soon as possible. Most feedback will happen during 1:1 meetings, if not sooner.

### Hiring

At Fleet, we collaborate with [core team members](#creating-a-new-position), [consultants](#hiring-a-consultant), [advisors](#adding-an-advisor), and [outside contributors](https://github.com/fleetdm/fleet/graphs/contributors) from the community.  

> Are you a new fleetie joining the Business Operations team?  For Loom recordings demonstrating how to make offers, hire, onboard, and more please see [this classified Google Doc](https://docs.google.com/document/d/1fimxQguPOtK-2YLAVjWRNCYqs5TszAHJslhtT_23Ly0/edit).

#### Consultants

##### Hiring a consultant

In addition to [core team members](#hiring-a-new-team-member), from time to time Fleet hires consultants who may work for only a handful of hours on short projects.

A consultant is someone who we expect to either:
- complete their relationship with the company in less than 6 weeks
- or have a longer-term relationship with the company, but never work more than 10 hours per week.

Consultants:
- do NOT receive company-issued laptops
- do NOT receive Yubikeys
- do NOT get a "Hiring" issue created for them
- do NOT get a company email address, nor everyone's calendars, nor the shared drive _(with occasional exceptions)_
- do NOT go through training using the contributor onboarding issue.
- do NOT fill any existing [open position](#creating-a-new-position)

Consultants [track time using the company's tools](#tracking-hours) and sign [Fleet's consulting agreement](#sending-a-consulting-agreement).

To hire a consultant, [submit a custom request](#intake) to the business operations team.

> TODO: replace this w/  issue template (see also commented-out notes in hiring.md for some other steps)

##### Who ISN'T a consultant?

If a consultant plans to work _more_ than 10 hours per week, or for _longer_ than 6 weeks, they should instead be hired as a [core team member](#hiring-a-new-team-member).

Core team members:
- are hired for an existing [open position](#creating-a-new-position)
- are hired using Fleet's "Hiring" issue template, including receiving a company-issued laptop and Yubikeys
- must be onboarded (complete the entire, unabridged onboarding process in Fleet's "Onboarding" issue template)
- must be offboarded
- get an email address
- have a manager and a formal place in the company [org chart](https://fleetdm.com/handbook/company#org-chart)
- are listed in ["🧑‍🚀 Fleeties"](https://docs.google.com/spreadsheets/d/1OSLn-ZCbGSjPusHPiR5dwQhheH1K8-xqyZdsOe9y7qc/edit#gid=0)
- are paid as part of the standard payroll ritual for the place they work and their employment classification.

Consultants aren't required to do any of those things.

##### Sending a consulting agreement

To hire a non-US consultant, please [submit a custom request](#intake).

To hire a US-based consultant, send them an agreement using the "Contractor agreement (US)" template in [DocuSign](https://www.docusign.com/).
(This template is located in the "¶¶ Classified templates" folder, which is only accessible via certain Docusign accounts in 1Password.)

> _**Note:** The Docusign template labeled "Contractor agreement (US)" is actually used for both consultants and [core team members in the US who are classified as 1099 contractors or billed corp-to-corp as vendors](#hiring-a-new-team-member).  You may also sometimes hear this referred to as Fleet's "Consulting agreement". Same thing._

To send a US consulting agreement, you'll need the new consultant's name, the term of the service, a summary of the services provided, and the consultant's fee. 

There are some defaults that we use for these agreements:
   - Term: Default to one month unless otherwise discussed.
   - Services rendered: Copy and paste from the [language in this doc](https://docs.google.com/document/d/1b5SGgYEHqDmq5QF8p29WWN3it3XJh3xRT3zG0RdXARo/edit)
   - Work will commence and complete by dates: Start date and end of term date
   - Fee: Get from the consultant.
   - Hours: Default to 10 hr/week.

Then hit send!  After all of the signatures are there, the completed document will automatically be uploaded to the appropriate Google Drive folder, and a Slack message will appear in the `#help-classified` channel.

##### Updating a consultant's fee
 - Direct message Mike McNeil with hourly rate change information.
 - After CEO approval, Mike McNeil will issue a new contractor agreement with the updated fee via DocuSign.

#### Advisor

##### Adding an advisor
Advisor agreements are sent through [DocuSign](https://www.docusign.com/), using the "Advisor Agreement"
template.
- Send the advisor agreement. To send a new advisor agreement, you'll need the new advisor's name and the number of shares they are offered. 
- Once you send the agreement, locate an existing empty row and available ID in ["Advisors"](https://docs.google.com/spreadsheets/d/15knBE2-PrQ1Ad-QcIk0mxCN-xFsATKK9hcifqrm0qFQ/edit#gid=1803674483) and enter the new advisor's information.
   >**_Note:_** *Be sure to mark any columns that haven't been completed yet as "TODO"*

##### Finalizing a new advisor
- Update the ["Advisors"](https://docs.google.com/spreadsheets/d/15knBE2-PrQ1Ad-QcIk0mxCN-xFsATKK9hcifqrm0qFQ/edit#gid=1803674483) to show that the agreement has been signed, and ask the new advisor to add us on [LinkedIn](https://www.linkedin.com/company/71111416), [Crunchbase](https://www.crunchbase.com/organization/fleet-device-management), and [Angellist](https://angel.co/company/fleetdm).
- Update "Equity plan" to reflect updated status and equity grant for this advisor, and to ensure the advisor's equity is queued up for the next quarterly equity grant ritual.

#### Core team member
This section is about creating a core team member role, and the hiring process for a new core team member, or Fleetie.

##### Creating a new position

Want to hire?  Here's how to open up a new position on the core team:

> Use these steps to hire a [fleetie, not a consultant](https://fleetdm.com/handbook/business-operations#who-isnt-a-consultant).

<!--
> If you think this job posting may need to stay temporarily classified (¶¶) and not shared company-wide or publicly yet, for any reason, then stop here and send a Slack DM with your proposal to the CEO instead of modifying ["🧑‍🚀 Fleeties"](https://docs.google.com/spreadsheets/d/1OSLn-ZCbGSjPusHPiR5dwQhheH1K8-xqyZdsOe9y7qc/edit) (which is visible company-wide) or submitting a draft pull request to "Open positions" (which is public).
-->

1. **Propose headcount:** Add the proposed position to ["🧑‍🚀 Fleeties"](https://docs.google.com/spreadsheets/d/1OSLn-ZCbGSjPusHPiR5dwQhheH1K8-xqyZdsOe9y7qc/edit#gid=0) in an empty row (but using one of the existing IDs.  Unsure?  Ask for help.)  Be sure to include job title, manager, and department. Set the start date to the first Monday of the next month (This position is still only proposed (not approved), but would make it easier for the approver to have the date set).
2. **Propose job description:** Copy, personalize, and publish the job description:
   - _Duplicate file:_ Locate [one of the existing job description files inside `handbook/company/`](https://github.com/fleetdm/fleet/tree/main/handbook/company) and duplicate it into a new handbook subpage.  If no other open job descriptions currently exist, you can [copy and paste the raw text](https://raw.githubusercontent.com/fleetdm/fleet/586194b771aa4ff7aa18072bd061720f94719d29/handbook/company/product-designer.md) from an [old job description](https://github.com/fleetdm/fleet/blob/586194b771aa4ff7aa18072bd061720f94719d29/handbook/company/product-designer.md).
     - _Filename:_ Use the [same style of filename](https://github.com/fleetdm/fleet/blob/586194b771aa4ff7aa18072bd061720f94719d29/handbook/company/product-designer.md), but based on the new job title.  (This filename will determine the living URL on fleetdm.com where candidates can apply.)
     - _Contents:_ Keep the structure of the document [identical](https://raw.githubusercontent.com/fleetdm/fleet/586194b771aa4ff7aa18072bd061720f94719d29/handbook/company/product-designer.md).  Change only the job title, "Responsibilities", and "Experience".
   - _Add to list of open positions:_ In [the same pull request](https://www.loom.com/share/75da64632a93415cbe0e7752107c1af2), add a link to your new job posting to the bottom of the list of ["📖 Company#Open positions"](https://fleetdm.com/handbook/company#open-positions) in the handbook.
     - State the proposed job title, include the appropriate departmental emoji, and link to the "living" fleetdm.com URL; not the GitHub URL.

> _**Note:** The "living" URL where the new page will eventually exist on fleetdm.com won't ACTUALLY exist until your pull request is merged.  For now, if you were to visit this URL, you'd just see a 404 error.  So how can you determine this URL?  To understand the pattern, visit other job description pages from the [live handbook](https://fleetdm.com/handbook/company#open-positions), and examine their URLs in your browser._

3. **Get it approved and merged:**  When you submit your proposed job description, the CEO will be automatically tagged for review and get a notification.  He will consider where this role fits into Fleet's strategy and decide whether Fleet will open this position at this time.  He will review the data carefully to try and catch any simple mistakes, then tentatively budget cash and equity compensation and document this compensation research.  He will set a tentative start date (which also indicates this position is no longer just "proposed"; it's now part of the hiring plan.)  Then the CEO will start a `#hiring-xxxxx-YYYY` Slack channel, at-mentioning the original proposer and letting them know their position is approved.  (Unless it isn't.)

> _**Why bother with approvals?**  We avoid cancelling or significantly changing a role after opening it.  It hurts candidates too much.  Instead, get the position approved first, before you start recruiting and interviewing.  This gives you a sounding board and avoids misunderstandings._

##### Approving a new position
When review is requested on a proposal to open a new position, the 🐈‍⬛ CEO will complete the following steps when reviewing the pull request:

1. **Consider role and reporting structure:** Confirm the new row in "Fleeties" has a manager, job title, and department, that it doesn't have any corrupted spreadsheet formulas or formatting, and that the start date is set to the first Monday of the next month.
2. **Read job description:** Confirm the job description consists only of changes to "Responsibilities" and "Experience," with an appropriate filename, and that the content looks accurate, is grammatically correct, and is otherwise ready to post in a public job description on fleetdm.com.
3. **Budget compensation:** Ballpark and document compensation research for the role based on 
   - _Add screenshot:_ Scroll to the very bottom of ["¶¶ 💌 Compensation decisions (offer math)"](https://docs.google.com/document/d/1NQ-IjcOTbyFluCWqsFLMfP4SvnopoXDcX0civ-STS5c/edit#heading=h.slomq4whmyas) and add a new heading for the role, pattern-matching off of the names of other nearby role headings. Then create written documentation of your research for future reference.  The easiest way to do this is to take screenshots of the [relevant benchmarks in Pave](https://pave.com) and paste those screenshots under the new heading.
   - _Update team database:_ Update the row in ["¶¶ 🥧 Equity plan"](https://docs.google.com/spreadsheets/d/1_GJlqnWWIQBiZFOoyl9YbTr72bg5qdSSp4O3kuKm1Jc/edit#gid=0) using the benchmarked compensation and share count.
     - _Salary:_ Enter the salary: If the role has variable compensation, use the role's OTE (on-target earning estimate) as the budgeted salary amount, and leave a note in the "Notes (¶¶)" cell clarifying the role's bonus or commission structure.
     - _Equity:_ Enter the equity as a number of shares, watching the percentage that is automatically calculated in the next cell.  Keep guessing different numbers of shares until you get the derived percentage looking like what you want to see.

4. **Decide**: Decide whether to approve this role or to consider it a different time.  If approving, then:
   - _Create Slack channel:_ Create a private "#hiring-xxxxxx-YYYY" Slack channel (where "xxxxxx" is the job title and YYYY is the current year) for discussion and invite the hiring manager.
   - _Publish opening:_ Approve and merge the pull request.  The job posting go live within ≤10 minutes.
   - _Reply to requestor:_ Post a comment on the pull request, being sure to include a direct link to their live job description on fleetdm.com.  (This is the URL where candidates can go to read about the job and apply.  For example: `fleetdm.com/handbook/company/product-designer`):
     ```
     The new opening is now live!  Candidates can apply at fleetdm.com/handbook/company/railway-conductor.
     ```

> _**Note:** Most columns of the "Equity plan" are updated automatically when "Fleeties" is, based on the unique identifier of each row, like `🧑‍🚀890`.  (Advisors have their own flavor of unique IDs, such as `🦉755`, which are defined in ["Advisors and investors"](https://docs.google.com/spreadsheets/d/15knBE2-PrQ1Ad-QcIk0mxCN-xFsATKK9hcifqrm0qFQ/edit).)_

##### Recruiting
Fleet accepts job applications, but the company does not list positions on general purpose job boards.  This prevents us being overwhelmed with candidates so we can fulfill our goal of responding promptly to every applicant.

This means that outbound recruiting, 3rd party recruiters, and references from team members are important aspect of the company's hiring strategy.  Fleet's CEO is happy to assist with outreach, intros, and recruiting strategy for candidates.

##### Receiving job applications
Every job description page ends with a "call to action", including a link that candidates can click to apply for the job.  Fleet replies to all candidates within **1 business day** and always provides either a **rejection** or **decisive next steps**; even if the next step is just a promise.  For example:

> "We are still working our way through applications and _still_ have not been able to review yours yet.  We think we will be able to review and give you an update about your application by Thursday at the latest.  I'll let you know as soon as I have news.  I'll assume we're both still in the running if I don't hear from you, so please let me know if anything comes up."

When a candidate clicks applies for a job at Fleet, they are taken to a generic Typeform.  When they submit their job application, the Typeform triggers a Zapier automation that will posts the submission to `g-business-operations` in Slack.  The candidate's job application answers are then forwarded to the applicable `#hiring-xxxxx-202x` Slack channel and the hiring manager is @mentioned.

##### Candidate correspondence email templates
Fleet uses [certain email templates](https://docs.google.com/document/d/1E_gTunZBMNF4AhsOFuDVi9EnvsIGbAYrmmEzdGmnc9U) when responding to candidates.  This helps us live our value of [🔴 empathy](https://fleetdm.com/handbook/company#empathy) and helps the company meet the aspiration of replying to all applications within one business day.

##### Hiring restrictions

###### Incompatible former employers
Fleet maintains a list of companies with whom Fleet has do-not-solicit terms that prevents us from making offers to employees of these companies.  The list is in the Do Not Solicit tab of the [BizOps spreadsheet](https://docs.google.com/spreadsheets/d/1lp3OugxfPfMjAgQWRi_rbyL_3opILq-duHmlng_pwyo/edit#gid=0).

###### Incompatible locations
Fleet is unable to hire team members in some countries. See [this internal document](https://docs.google.com/document/d/1jHHJqShIyvlVwzx1C-FB9GC74Di_Rfdgmhpai1SPC0g/edit) for the list.

##### Interviewing
We're glad you're interested in joining the team! 
Here are some of the things you can anticipate throughout this process:
  - We will reply by email within one business day from the time when the application arrives.
  - You may receive a rejection email (Bummer, consider applying again in the future).
  - You may receive an invitation to "book with us."
If you've been invited to "book with us," you'll have a Zoom meeting with the hiring team to discuss the next steps. 


##### Hiring a new team member

This section is about the hiring process a new core team member, or fleetie.

> **_Note:_** _Employment classification isn't what makes someone a fleetie.  Some Fleet team members are contractors and others are employees.  The distinction between "contractor" and "employee" varies in different geographies, and the appropriate employment classification and agreement for any given team member and the place where they work is determined by Head of Business Operations during the process of making an offer._

Here are the steps hiring managers follow to get an offer out to a candidate:
1. **Add to team database:** Update the [Fleeties](https://docs.google.com/spreadsheets/d/1OSLn-ZCbGSjPusHPiR5dwQhheH1K8-xqyZdsOe9y7qc/edit#gid=0) doc to accurately reflect the candidate's:
   - Start date
     > _**Tip:** No need to check with the candidate if you haven't already.  Just guess.  First Mondays tend to make good start dates.  When hiring an international employee, Pilot.co recommends starting the hiring process a month before the new employee's start date._
   - First and last name
   - Preferred pronoun _("them", "her", or "him")_
   - LinkedIn URL _(If the fleetie does not have a LinkedIn account, enter `N/A`)_
   - GitHub username _(Every candidate must have a GitHub account in "Fleeties" before the company makes them an offer.  If the the candidate does not have a GitHub account, ask them to create one, and make sure it's tracked in "Fleeties".)_
     > _**Tip:** A revealing live interview question can be to ask a candidate to quickly share their screen, sign up for GitHub, and then hit the "Edit" button on one of the pages in [the Fleet handbook](https://fleetdm.com/handbook) to make their first pull request.  This should not take more than 5 minutes._
2. **Call references:** Ask the candidate for at least 2+ references and contact each reference in parallel using the instructions and tips in [Fleet's reference check template](https://docs.google.com/document/d/1LMOUkLJlAohuFykdgxTPL0RjAQxWkypzEYP_AT-bUAw/edit?usp=sharing).  Be respectful and keep these calls very short.
3. **Schedule CEO interview:** Book a quick chat so our CEO can get to know the future Fleetie.
   - No need to check with the CEO first.  You can [book the meeting directly](https://fleetdm.com/handbook/company/communitcation#internal-meeting-scheduling) on the CEO's calendar during a time they and the candidate are both available.
   - Set the Google Calendar description of the calendar event to: `Agenda: https://docs.google.com/document/d/1yARlH6iZY-cP9cQbmL3z6TbMy-Ii7lO64RbuolpWQzI/edit`.
   - The personal email you use for the candidate in this calendar event is where they will receive their offer or rejection email.
4. **Confirm intent to offer:** Compile feedback about the candidate into a single document and share that document (the "interview packet") with the Head of Business Operations via Google Drive.  _This will be interpreted as a signal that you are ready for them to make an offer to this candidate._
   - _Compile feedback into a single doc:_ Include feedback from interviews, reference checks, and challenge submissions.  Include any other notes you can think of offhand, and embed links to any supporting documents that were impactful in your final decision-making, such as portfolios or challenge submissions.
   - _Share_ this single document with the Head of Business Operations via email.
     - Share only _one, single Google Doc, please_; with a short, formulaic name that's easy to understand in an instant from just an email subject line.  For example, you could title it:
       >Why hire Jane Doe ("Train Conductor") - 2023-03-21
     - When the Head of Business Operations receives this doc shared doc in their email with the compiled feedback about the candidate, they will understand that to mean that it is time for Fleet to make an offer to the candidate.

##### Making an offer
After receiving the interview packet, the Head of Business Operations uses the following steps to make an offer:
1. **Adjust compensation:** 🔦 Head of Business Operations [re-benchmarks salary](https://www.pave.com), adjusting for cost of living where the candidate will do the work.
   - _Paste a screenshot_ from Pave showing the amount of cash and equity in the offer (or write 1-2 sentences about what is being offered to this candidate and why) under the [heading for this position in " 💌 Compensation decisions"](https://docs.google.com/document/d/1NQ-IjcOTbyFluCWqsFLMfP4SvnopoXDcX0civ-STS5c/edit)
   - _Update the ["🥧 Equity plan"](https://docs.google.com/spreadsheets/d/1_GJlqnWWIQBiZFOoyl9YbTr72bg5qdSSp4O3kuKm1Jc/edit#gid=0)_ to reflect the offer that is about to be sent:
     -  Salary _(OTE actually offered)_
     -  Equity _(stock options actually offered)_
     -  "Notes" _(include base salary versus commission or bonus plan, if relevant)_
     -  "Offer sent?" _(set this to `TRUE`)_
     - …and make sure the other status columns are set to `todo`.
2. **Prepare the "exit scenarios" spreadsheet:** 🔦 Head of Business Operations [copies the "Exit scenarios (template)"](https://docs.google.com/spreadsheets/d/1k2TzsFYR0QxlD-KGPxuhuvvlJMrCvLPo2z8s8oGChT0/copy) for the candidate, and renames the copy to e.g. "Exit scenarios for Jane Doe".
   - _Edit the candidate's copy of the exit scenarios spreadsheet_ to reflect the number of shares in ["🥧 Equity plan"](https://docs.google.com/spreadsheets/d/1_GJlqnWWIQBiZFOoyl9YbTr72bg5qdSSp4O3kuKm1Jc/edit#gid=0), and the spreadsheet will update automatically to reflect their approximate ownership percentage.
     > _**Note:** Don't play with numbers in the exit scenarios spreadsheet. The revision history is visible to the candidate, and they might misunderstand._
3. **Prepare offer:** 🔦 Head of Business Operations [copies "Offer email (template)"](https://docs.google.com/document/d/1zpNN2LWzAj-dVBC8iOg9jLurNlSe7XWKU69j7ntWtbY/copy) and renames to e.g. "Offer email for Jane Doe".  Edit the candidate's copy of the offer email template doc and fill in the missing information:
   - _Benefits:_ If candidate will work outside the US, [change the "Benefits" bullet](https://docs.google.com/document/d/1zpNN2LWzAj-dVBC8iOg9jLurNlSe7XWKU69j7ntWtbY/edit) to reflect what will be included through Fleet's international payroll provider, depending on the candidate's location.
   - _Equity:_ Highlight the number of shares with a link to the candidate's custom "exit scenarios" spreadsheet.
   - _Hand off:_ Share the offer email doc with the [Apprentice to the CEO](https://fleetdm.com/handbook/company/ceo#team).
4. **Draft email:** 🦿 Apprentice to the CEO drafts the offer email in the CEO's inbox, reviews one more time, and then brings it to their next daily meeting for CEO's approval:
   - To: The candidate's personal email address _(use the email from the CEO interview calendar event)_
   - Cc: Zach Wasserman and Head of Business Operations _(neither participate in the email thread until after the offer is accepted)_
   - Subject: "Full time?"
   - Body: _(The offer email is copied verbatim from Google doc into Gmail as the body of the message, formatting and all.  Check all links in offer letter for accuracy, and also click the surrounding areas to ensure no "ghost links" are left from previous edits... which has happened before.  Re-read the offer email one last time, and especially double-check that salary, number of shares, and start date match the equity plan.)_
5. **Send offer:** 🐈‍⬛ CEO reviews and sends the offer to the candidate:
   - _Grant the candidate "edit" access_ to their "exit scenarios" spreadsheet.
   - _Send_ the email.

##### Steps after an offer is accepted

Once the new team member replies and accepts their offer in writing, 🔦 Head of Business Operations follows these steps:
1. **Verify, track, and reply:** Reply to the candidate:
   - _Verify the candidate replied with their physical address… or else keep asking._  If they did not reply with their physical address, then we are not done.  No offer is "accepted" until we've received a physical address.
   - _Review and update the team database_ to be sure everything is accurate, **one last time**.  Remember to read the column headers and precisely follow the instructions about how to format the data:
     - The new team member's role in ["🧑‍🚀 Fleeties"](https://docs.google.com/spreadsheets/d/1OSLn-ZCbGSjPusHPiR5dwQhheH1K8-xqyZdsOe9y7qc/edit#gid=0) now includes:
       - **Start date** _(The new fleetie's first day, YYYY-MM-DD)_
       - **Location** _(Derive this from the physical address)_
       - **GitHub username**  _(Username of 2FA-enabled GitHub account)_
       - **@fleetdm.com email** _(Set this to whatever email you think this person should have)_
     - The new team member's row in ["🥧 Equity plan"](https://docs.google.com/spreadsheets/d/1_GJlqnWWIQBiZFOoyl9YbTr72bg5qdSSp4O3kuKm1Jc/edit#gid=0) now includes:
       - **OTE** _("On-target earnings", i.e. anticipated total annual cash compensation)_
       - **Equity** _(Stock options)_
       - **"Notes"** _(Track base salary here, as well as a very short explanation of commission or bonus structure.)_
       - **Physical address** _(The full street address of the location where work will typically be performed.)_
       - **Personal email** _(Use the personal email they're replying from, e.g. `@gmail.com`)_
       - **"Offer accepted?"** _(Set this to `TRUE`)_
   - _[Create a "Hiring" issue](https://github.com/fleetdm/confidential/issues/new/choose)_ for the new team member.  (This issue will keep track of the hiring tasks for the new team member.)
   - _Send a reply_ welcoming the team member to Fleet and letting them know to expect a separate email with next steps for getting the team member's laptop, Yubikeys, and agreement going ASAP so they can start on time.  For example:
     ```
     \o/  It's official!
     
     Be on the lookout for an email in a separate thread with next steps for quickly signing the paperwork and getting your company laptop and hardware 2FA keys (Yubikeys), which we recommend setting up ASAP.
     
     Thanks, and welcome to the team!
     
     -Joanne
     ```
2. **Ask hiring manager to send rejections:** Post to the `hiring-xxxxx-yyyy` Slack channel to let folks know the offer was accepted, and at-mention the _hiring manager_ to ask them to communicate with [all other interviewees](https://fleetdm.com/handbook/company#empathy) who are still in the running and [let them know that we chose a different person](https://fleetdm.com/handbook/business-operations#candidate-correspondence-email-templates).
   >_**Note:** Send rejection emails quickly, within 1 business day.  It only gets harder if you wait._
3. **Remove open position:** Take down the newly-filled position from the fleetdm.com website by making the following two changes:  (please only submit [one, single pull request that changes both of these files](https://www.loom.com/share/75da64632a93415cbe0e7752107c1af2):
   - Edit the [list of open positions](https://fleetdm.com/handbook/company#open-positions) to remove the newly-filled position from the list.
   - Remove the [job description file](https://github.com/fleetdm/fleet/tree/main/handbook/company) that corresponds with the newly-filled position.  (This is a Markdown file named after the role, with a filename ending in `.md`.)
5. **Close Slack channel:** Then archive and close the channel.

Now what happens?  🔦 Business Operations will then follow the steps in the "Hiring" issue, which includes reaching out to the new team member within 1 business day from a separate email thread to get additional information as needed, prepare their agreement, add them to the company's payroll system, and get their new laptop and hardware security keys ordered so that everything is ready for them to start on their first day.

### Departures

#### Communicating departures
Although it's sad to see someone go, Fleet understands that not everything is meant to be forever [like open-source is](https://fleetdm.com/handbook/company/why-this-way#why-open-source). There are a few steps that the company needs to take to facilitate a departure. 
1. **Departing team member's manager:** Before speaking further with the team member, inform business operations about the departure via direct message in Slack to the acting Head of People (`@mikermcneil`), who will coordinate the team member's last day, offboarding, and exit meeting.
2. **Business Operations**: Create and begin completing [offboarding issue](https://github.com/fleetdm/classified/blob/main/.github/ISSUE_TEMPLATE/%F0%9F%9A%AA-offboarding-____________.md).
   > After finding out at the next standup (or sooner), Business Operations will post in `#g-e` to inform the E-group of the team member's departure and ask E-group members will inform any other managers on their teams.
3. **CEO**: The CEO will make an announcement during the "🌈 Weekly Update" post on Friday in the `#general` channel on Slack. 


## Finance Operations

### Spending company money
As we continue to expand our company policies, we use [GitLab's open expense policy](https://about.gitlab.com/handbook/spending-company-money/) as a guide for company spending.
In brief, this means that as a Fleet team member, you may:
* Spend company money like it is your own money.
* Be responsible for what you need to purchase or expense to do your job effectively.
* Feel free to make purchases __in the company's interest__ without asking for permission beforehand (when in doubt, do __inform__ your manager prior to purchase or as soon as possible after the purchase).
For more developed thoughts about __spending guidelines and limits__, please read [GitLab's open expense policy](https://about.gitlab.com/handbook/spending-company-money/).

#### Brex

##### Travel
###### Attending conferences or company travel
When attending a conference or traveling for Fleet, keep the following in mind:
- $100 allowance per day for your own personal food and beverage. **(There are many good reasons to make exceptions to this guideline, such as dinners with customers.  Before proceeding, please [request approval from the Head of Business Operations](https://fleetdm.com/handbook/business-operations#intake).**
- We highly recommend you order a physical Brex card if you do not have one before attending the conference.
- The monthly limit on your Brex card may need to be increased temporarily as necessary to accommodate the increased spending associated with the conference, such as travel.  You can [request that here](https://fleetdm.com/handbook/business-operations#intake) by providing the following information:
  - The start and end dates for your trip.
  - The price of your flight (feel free to optimize a direct flight if there is one that is less than double the price of the cheapest non-direct flight).
  - The price of your hotel per night (dry cleaning is allowable if the stay is over 3 days).
  - The price of the admission fees if attending a conference.
- Please use your personal credit card for movies, mini bars, and entertainment.  These expenses _will not_ be reimbursed.

##### Non-travel purchases that exceed a Brex cardholder's limit
For non-travel purchases that would require an increase in the Brex cardholder's limit, please [make a request](https://fleetdm.com/handbook/business-operations#intake) with following information:
- The nature of the purchase (i.e. SaaS subscription and what it's used for)
- The cost of the purchase and whether it is a fixed or variable (i.e. use-based) cost.
- Whether it is a one time purchase or a recurring purchase and at what frequency the purchase will re-occur (annually, monthly, etc.)
- If there are more ideal options to pay for the purchase (i.e. bill.com, the Fleet AP Brex card, etc.) that method will be used instead.  
- In general, recurring purchases such as subscription services that will continually stretch the spend limit on a cardholder's Brex card should be paid through other means. 
- For one time purchases where payment via credit card is the most convenient then the card limit will be temporarily increased to accomodate the purchase.  

#### Reimbursements
Fleet does not reimburse expenses. We provide all of our team members with Brex cards for making purchases for the company. For company expenses, **use your Brex card.**  If there was an extreme accident, [get help](https://fleetdm.com/handbook/business-operations#intake).

<!-- 
No longer supported.  -mike, CEO, 2023-04-26.

Fleet will reimburse team members who pay for work-related expenses with their personal funds.
Team members can request reimbursement through [Gusto]([https://app.gusto.com/expenses](https://support.gusto.com/article/209831449100000/Get-reimbursed-for-expenses-as-an-employee)) if they're in the US or [Pilot]([https://pilot.co/](https://help.pilot.co/en/articles/4658204-how-to-request-a-reimbursement#:~:text=If%20you%20made%20a%20purchase,and%20click%20'Add%20new%20expense.)) if they are an international team member. When submitting an expense report, team members need to provide the receipt and a description of the expense.
Operations will review the expense and reach out to the team member if they have any questions. The reimbursement will be added to the team member's next payroll when an expense is approved.
>Pilot handles reimbursements differently depending on if the international team member is classified as an employee or a contractor. If the reimbursement is for a contractor, Operations will need to add the expense reimbursement to an upcoming recurring payment or schedule the reimbursement as an off-cycle payment. If the reimbursement is for an employee, no other action is needed; Pilot will add the reimbursement to the team member's next payroll.  -->

### Individualized expenses
Recurring expenses related to a particular team member, such as coworking fees, are called _individualized expenses_.  These expenses are still considered [non-personnel expenses](https://docs.google.com/spreadsheets/d/1X-brkmUK7_Rgp7aq42drNcUg8ZipzEiS153uKZSabWc/edit#gid=2112277278), with a few extra considerations:

- Non-recurring (one-off) expenses such as an Uber ride from the airport are NOT considered "individualized".
- Seat licenses for tools like Salesforce or GitHub are NOT considered "individualized".
- Individualized expenses should include the team member's name explicitly in the name of the expense.
- If multiple team members use the same vendor for an individualized expense (for example, "WeWork"), use a separate row for each individualized expense.  (For example: "Coworking, Mungojerry (WeWork)" and "Coworking, Jennyanydots (WeWork)")
- Individualized expenses are always attributed to the  "🔦 Business operations" department.
- These expenses are still considered non-personnel expenses, in the same way seat licenses for tools like Salesforce or GitHub are considered non-personnel expenses.

### Taxes and compliance

From time to time, you may get notices in the mail from the IRS and/or state agencies regarding your company’s withholding and/or unemployment tax accounts. You can resolve many of these notices on your own by verifying and/or updating the settings in your Gusto account. 
If the notice is regarding an upcoming change to your deposit schedule or unemployment tax rate, Mike McNeil will make the change in Gusto. Including: 
 - Update your unemployment tax rate.
 - Update your federal deposit schedule.
 - Update your state deposit schedule.
**Important** Agencies do not send notices to Gusto directly, so it’s important that you read and take action before any listed deadlines or effective dates of requested changes.
Notices you should report to Gusto.
If you can't resolve the notice on your own, are unsure what the notice is in reference to, or the tax notice has a missing payment or balance owed, follow the steps in the Report and upload a tax notice in Gusto.
In Gusto, click **How to review your notice** to help you understand what kind of notice you received and what additional action you can take to help speed up the time it takes to resolve the issue.
For more information about how Fleet and our accounting team work together, check out [Fleet - who does what](https://docs.google.com/spreadsheets/d/1FFOudmHmfVFIk-hdIWoPFsvMPmsjnRB8/edit#gid=829046836) (private doc).

#### State quarterly payroll and tax filings
Every quarter, payroll and tax filings are due for each state. Gusto can handle these automatically if Third-party authorization (TPA) is enabled. Each state is unique and Gusto has a library of [State registration and resources](https://support.gusto.com/hub/Employers-and-admins/Taxes-forms-and-compliance/State-registration-and-resources) available to review. 
You will need to grant Third-party authorization (TPA) per state and this should be checked quarterly before the filing due dates to ensure that Gusto can file on time. 

#### CorpNet state registration process
In CorpNet, select "place an order for an existing business" we’ll need to have Foreign Registration and Payroll Tax Registration done.
  - You can have CorpNet do this by emailing the account rep "Subject: Fleet Device Management: State - Foreign Registration and Payroll Tax Registration" (this takes about two weeks).
  - You can do this between you and CorpNet by selecting "Foreign Qualification," placing the order and emailing the confirmation to the rep for Payroll registration (this is a short turnaround).
  - You can do this on your own by visiting the state's "Secretary of State" website and checking that the company name is available. To register online, you'll need the EIN, business address, information about the owners and their percentages, the first date of business, sales within the state, and the business type (usually get an email right away for approval ~24-48 hrs). 
For more information, check out [Fleet - who does what](https://docs.google.com/spreadsheets/d/1FFOudmHmfVFIk-hdIWoPFsvMPmsjnRB8/edit?usp=sharing&ouid=102440584423243016963&rtpof=true&sd=true).

### Finance rituals

#### Payroll
Many of these processes are automated, but it's vital to check Gusto and Pilot manually for accuracy.
 - Salaried fleeties are automated in Gusto and Pilot
 - Hourly fleeties and consultants are a manual process in Gusto and Pilot.

| Unique payrolls              | Action                       | DRI                          |
|:-----------------------------|:-----------------------------|:-----------------------------|
| Commissions and ramp         | "Off-cycle" payroll          | Nathan
| Sign-on bonus                | "Bonus" payroll              | Mike McNeil
| Performance bonus            | "Bonus" payroll              | Mike McNeil                     
| Accelerations (quarterly)    | "Off-cycle" payroll          | Nathan

Add the amount to be paid to the "Gross" line.
For Fleet's US contractors, running payroll is a manual process. 
The steps for doing this are highlighted in this loom, TODO. 
1. Time tools
2. Time tracking
3. Review hours
4. Adjust time frame to match current payroll period (the 27th through 26th of the month)
5. Sync hours
6. Run contractor payroll

##### Commission payroll

> TODO: bit more process here.  Maybe revops is DRI of commission calculator, creates "2023-03 commission payroll", transfered to Nathan when it's time to run?  SLA == payroll run by the 7th, with commission sheet 100% accurate.

- Update [commission calculator](https://docs.google.com/spreadsheets/d/1vw6Q7kCC7-FdG5Fgx3ghgUdQiF2qwxk6njgK6z8_O9U/edit) with new revenue from any deals that are closed/won (have a subscription agreement signed by both parties) and have an **effective start date** within the previous month.
  - Find detailed notes on this process in [Notes - Run commission payroll in Gusto](https://docs.google.com/document/d/1FQLpGxvHPW6X801HYYLPs5y8o943mmasQD3m9k_c0so/edit#). 
- Contact Mike McNeil in Slack and let her know he can run the commission payroll. Use the off-cycle payroll option in Gusto. Be sure to classify the payment as "Commission" in the "other earnings" field and not the generic "Bonus."
- Once commission payroll has been run, update the [commission calculator](https://docs.google.com/spreadsheets/d/1vw6Q7kCC7-FdG5Fgx3ghgUdQiF2qwxk6njgK6z8_O9U/edit) to mark the commission as paid. 

#### Monthly rituals

##### Monthly accounting
Create a [new montly accounting issue](https://github.com/fleetdm/confidential/issues/new/choose) for the current month and year named "Closing out YYYY-MM" in GitHub and complete all of the tasks in the issue. (This uses the [monthly accounting issue template](https://github.com/fleetdm/confidential/blob/main/.github/ISSUE_TEMPLATE/5-monthly-accounting.md).

###### SLA
The monthly accounting issue should be completed and closed before the 7th of the month.
The close date is tracked each month in [KPIs](https://docs.google.com/spreadsheets/d/1Hso0LxqwrRVINCyW_n436bNHmoqhoLhC8bcbvLPOs9A/edit).

###### When is the issue created?

We create and close the monthly accounting issue for the previous month within the first 7 days of the following month.  For example, the monthly accounting issue to close out the month of January is created promptly in February and closed before the end of the day, Feb 7th.

A convenient trick is to create the issue on the first Friday of the month and close it ASAP.

##### Recurring expenses
Recurring monthly or annual expenses are tracked as recurring, non-personnel expenses in ["🧮 The Numbers"](https://docs.google.com/spreadsheets/d/1X-brkmUK7_Rgp7aq42drNcUg8ZipzEiS153uKZSabWc/edit#gid=2112277278) _(classified Google Sheet)_, along with their payment source. Reconciliation of recurring expenses happens monthly.

> Use this spreadsheet as the source of truth.  Always make changes to it first before adding or removing a recurring expense. Only track significant expenses. (Other things besides amount can make a payment significant; like it being an individualized expense, for example.)

#### Quarterly rituals

##### Quarterly Quickbooks Online (QBO) check
- Check to make sure [bookkeeping quirks](https://docs.google.com/spreadsheets/d/1nuUPMZb1z_lrbaQEcgjnxppnYv_GWOTTo4FMqLOlsWg/edit?usp=sharing) are all accounted for and resolved or in progress toward resolution.
- Check balance sheet and profit and loss statements (P&Ls) in QBO against the [monthly workbooks](https://drive.google.com/drive/folders/1ben-xJgL5MlMJhIl2OeQpDjbk-pF6eJM) in Google Drive.

##### Quarterly investor reporting
- Login to Chronograph and upload our profit and loss statement (P&L), balance sheet and cash flow statements for CRV (all in one book saved in [Google Drive](https://drive.google.com/drive/folders/1ben-xJgL5MlMJhIl2OeQpDjbk-pF6eJM).
- Provide updated metrics for the following items using Fleet's [KPI spreadsheet](https://docs.google.com/spreadsheets/d/1Hso0LxqwrRVINCyW_n436bNHmoqhoLhC8bcbvLPOs9A/edit#gid=0).
  - Headcount at end of the previous quarter.
  - Starting ARR for the previous quarter.
  - Total new ARR for the previous quarter.
  - "Upsell ARR" (new ARR from expansions only- Chronograph defines "upsell" as price increases for any reason.
    **- Fleet does not "upsell" anything; we deliver more value and customers enroll more hosts), downgrade ARR and churn ARR (if any) for the previous quarter.**
  - Ending ARR for the previous quarter.
  - Starting number of customers, churned customers, and the number of new customers Fleet gained during the previous quarter.
  - Total amount of Fleet customers at the end of the previous quarter.
  - Gross margin % 
    - How to calculate: total revenue for the quarter - cost of goods sold for the quarter (these metrics can be found in our books from Pilot). Chronograph will automatically conver this number to a %.
  - Net dollar retention rate
    - How to calculate: (starting ARR + new subscriptions and expansions - churn)/starting ARR. 
  - Cash burn
    - How to calculate: (start of quarter runway - end of quarter runway)/3. 

##### Equity grants
Equity grants for new hires are queued up as part of the [hiring process](https://fleetdm.com/handbook/business-operations#hiring), then grants and consents are [batched and processed quarterly](https://github.com/fleetdm/confidential/issues/new/choose).

Doing an equity grant involves:
1. executing a board consent
2. the recipient and CEO signing paperwork about the stock options
3. updating the number of shares for the recipient in the equity plan
4. updating Carta to reflect the grant

For the status of stock option grants, exercises, and all other _common stock_ including advisor, founder, and team member equity ownership, see [Fleet's equity plan](https://docs.google.com/spreadsheets/d/1_GJlqnWWIQBiZFOoyl9YbTr72bg5qdSSp4O3kuKm1Jc/edit#gid=0).  For information about investor ownership, see [Carta](https://app.carta.com/corporations/1234715/summary/).

> Fleet's [equity plan](https://docs.google.com/spreadsheets/d/1_GJlqnWWIQBiZFOoyl9YbTr72bg5qdSSp4O3kuKm1Jc/edit#gid=0) is the source of truth, not Carta.  Neither are pro formas sent in an email attachment, even if they come from lawyers.
> 
> Anyone can make mistakes, and none of us are perfect.  Even when we triple check.  Small mistakes in share counts can be hard to attribute, and can cause headaches and eat up nights of our CEO's and operations team's time.  If you notice what might be a discrepancy between the equity plan and any other secondary source of information, please speak up and let Fleet's CEO know ASAP.  Even if you're wrong, your note will be appreciated.

#### Annual rituals
##### Annual reporting for capital credit line
- Within 60 days of the end of the year:
  - Provide Silicon Valley Bank (SVB) with our balance sheet and profit and loss statement (P&L, sometimes called a cashflow statement) for the past twelve months.  
  - Provide SVB with our annual operating budgets and projections (on a quarterly basis) for the coming year.
  - Deliver this as early as possible in case they have questions.

## Legal Operations

Please submit legal questions and requests to [Business Operations department](https://fleetdm.com/handbook/business-operations#intake).
> **Note:** Escalate first-of-its-kind agreements to the CEO. Mike will review business terms and consult with lawyers as necessary.


### Getting a contract signed

If a contract is ready for signature and requires no review or revision, the requestor logins into DocuSign using hello@ from the 1Password vault and routes the agreement to the CEO for signature. 

When a contract is going to be routed for signature by someone outside of Fleet (i.e. the vendor or customer), the requestor is responsible for working with the other party to make sure the document gets routed to the CEO for signature.

The SLA for contract signature is **2 business days**. Please do not follow up on signature unless this time has elapsed.

> _**Note:** Signature open time for the CEO is not currently measured, to avoid the overhead of creating separate signature issues to measure open and close time. This may change as signature volume increases._

### Getting a contract reviewed

> If a document is ready for signature and does not need to be reviewed or negotiated, you can skip the review process and use the signature process documented above.

To get a contract reviewed, upload the agreement to [Google Drive](https://drive.google.com/drive/folders/1G1JTpFxhKZZzmn2L2RppohCX5Bv_CQ9c).

Complete the [contract review issue template in GitHub](https://fleetdm.com/handbook/business-operations#intake), being sure to include the link to the document you uploaded and using the Calendly link in the issue template to schedule time to discuss the agreement with Nathan Holliday (allowing for sufficient time for him to have reviewed the contract prior to the call).

Follow up comments should be made in the GitHub issue and in the document itself so it is all in the same place.

The SLA for contract review is **2 business days**.

Once the review is complete, the issue will be closed.

If an agreement requires an additional review during the negotiation process, the requestor will need to follow these steps again. Uploading the new draft and creating a new issue in GitHub. 

When no further review or action is required for an agreement and the document is ready to be signed, the requestor is then responsible for routing the document for signature.

### Vendor questionnaires 
In responding to security questionnaires, Fleet endeavors to provide full transparency via our [security policies](https://fleetdm.com/handbook/security/security-policies#security-policies) and [application security](https://fleetdm.com/handbook/business-operations/application-security) documentation. In addition to this documentation, please refer to [the vendor questionnaires page](./vendor-questionnaires.md) 

## IT Operations

### Tools we use

There are a number of tools that are used throughout Fleet. Some of these tools are used company-wide, while others are department-specific. You can see a list of those tools in ["Tools we use" (private Google doc)](https://docs.google.com/spreadsheets/d/170qjzvyGjmbFhwS4Mucotxnw_JvyAjYv4qpwBrS6Gl8/edit?usp=sharing).

#### Slack
At Fleet, we do not send internal emails to each other. Instead, we prefer to use [Slack](https://www.linkedin.com/pulse/remote-work-how-set-boundaries-when-office-your-house-lora-vaughn/) to communicate with other folks who work at Fleet.
We use threads in Slack as much as possible. Threads help limit noise for other people following the channel and reduce notification overload.
We configure our [working hours in Slack](https://slack.com/help/articles/360025054173-Set-up-Slack-for-work-hours-) to make sure everyone knows when they can get in touch with others.

##### Slack channel prefixes
We have specific channels for various topics, but we also have more general channels for the teams at Fleet.
We use these prefixes to organize the Fleet Slack:
 * ***g-***: for team/group channels *(Note: "g-" is short for "grupo" or "group")*.
 * ***oooh-***: used to discuss and share interesting information about a topic.
 * ***help-***: for asking for help on specific topics.
 * ***at*** or ***fleet-at***: for customer channels.
 * ***2023-***: for temporary channels _(Note: specify the relevant year in four digits, like "YYYY-`)_

##### Slack communications and best practices
In consideration of our team, Fleet avoids using global tags in channels (i.e. @here, @channel, etc). 
      1. What about polls? Good question, Fleeties are asked to post their poll in the channel and @mention the teammates they would like to hear from. 
      2. Why does this matter? Great question! The Fleet [culture](https://fleetdm.com/handbook/company#culture) is pretty simple: think of others, and remember the company [Values](https://fleetdm.com/handbook/company#values).

#### Zoom
We use [Zoom](https://zoom.us) for virtual meetings at Fleet, and it is important that every team member feels comfortable hosting, joining, and scheduling Zoom meetings.
By default, Zoom settings are the same for all Fleet team members, but you can change your personal settings on your [profile settings](https://zoom.us/profile/setting) page. 
Settings that have a lock icon next to them have been locked by an administrator and cannot be changed. Zoom administrators can change settings for all team members on the [account settings page](https://zoom.us/account/setting) or for individual accounts on the [user management page](https://zoom.us/account/user#/).

#### Role-specific licenses
Certain new team members, especially in go-to-market (GTM) roles, will need paid access to paid tools like Salesforce and LinkedIn Sales Navigator immediately on their first day with the company. Gong licenses that other departments need may [request them from BizOps](https://fleetdm.com/handbook/business-operations#intake) and we will make sure there is no license redundancy in that department. The table below can be used to determine which paid licenses they will need, based on their role:

| Role                 | Salesforce CRM | Salesforce "Inbox" | LinkedIn _(paid)_ | Gong _(paid)_ | Zoom _(paid)_ |
|:-----------------|:--|:---|:---|:---|:--|
| 🐋 AE            | ✅ | ✅ | ✅ | ✅ | ✅
| 🐋 CSM           | ✅ | ✅ | ❌ | ✅ | ✅
| 🐋 SA            | ✅ | ✅ | ❌ | ❌ | ✅
| 🫧 SDR           | ✅ | ✅ | ✅ | ❌ | ❌
| ⚗️ PM             | ❌ | ❌ | ❌ | ✅ | ✅
| 🔦 CEO           | ✅ | ✅ | ✅ | ✅ | ✅
|   Other roles    | ❌ | ❌ | ❌ | ❌ | ❌

> **Warning:** Do NOT buy LinkedIn Recruiter. AEs and SDRs should use their personal Brex card to purchase the monthly [Core Sales Navigator](https://business.linkedin.com/sales-solutions/compare-plans) plan. Fleet does not use a company wide Sales Navigator account. The goal of Sales Navigator is to access to profile views and data, not InMail.  Fleet does not send InMail. 

#### Salesforce 
We consider Salesforce to be our Rolodex for customer information.

Here are the steps we take to grant appropriate Salesforce licenses to a new hire:
1. Go to ["My Account"](https://fleetdm.lightning.force.com/lightning/n/standard-OnlineSalesHome).
2. View contracts -> pick current contract.
3. Add the desired number of licenses.
4. Sign DocuSign sent to the email.
5. The order will be processed in ~30m.
6. Once the basic license has been added, you can create a new user using the new team member's `@fleetdm.com` email and assign a license to it.
7. To also assign a user an "Inbox license", go to the ["Setup" page](https://fleetdm.lightning.force.com/lightning/setup/SetupOneHome/home) and select "User > Permission sets". Find the [inbox permission set](https://fleetdm.lightning.force.com/lightning/setup/PermSets/page?address=%2F005%3Fid%3D0PS4x000002uUn2%26isUserEntityOverride%3D1%26SetupNode%3DPermSets%26sfdcIFrameOrigin%3Dhttps%253A%252F%252Ffleetdm.lightning.force.com%26clc%3D1) and assign it to the new team member.


#### Gong
Capturing video from meetings with customers, prospects, and community members outside the company is an important part of building world-class sales and customer success teams and is a widespread practice across the industry. At Fleet, we use Gong to capture Zoom meetings and share them company-wide. If a team member with a Gong license attends certain meetings, generally those with at least one person from outside of Fleet in attendance.  
  - While some Fleeties may have a Gong seat that is necessary in their work, the typical use case at Fleet is for employees on the company's sales, customer success, or customer support teams. 
  - You should be notified anytime you join a recorded call with an audio message announcing "this meeting is being recorded" or "recording in progress."  To stop a recording, the host of the call can press "Stop." 
  - If the call has external participants and is recorded, this call is stored in Gong for future use. 
To access a recording saved in Gong, visit [app.gong.io](https://app.gong.io) and sign in with SSO. 
  - Everyone at Fleet has access, whether they have a Gong seat or not, and you can explore and search through any uploaded call transcripts unless someone marks them as private (though the best practice would be not to record any calls you don't want to be captured). 
If you ever make a mistake and need to delete something, you can delete the video in Gong or reach out to Nathan Holliday or Mike McNeil for help. They will delete it immediately without watching the video. 
  - Note that any recording stopped within 60 seconds of the start of the recording is not saved in Gong, and there will be no saved record of it. 

Most folks at Fleet should see no difference in their meetings if they aren't interfacing with external parties. 
Our goal in using Gong and recording calls is to capture insights from sales, customer, and community meetings and improve how we position and sell our product. We never intend to make anyone uncomfortable, and we hope you reach out to our DRI for Gong, Nathan Holliday, or Mike McNeil if you have questions or concerns.  

#### Troubleshooting Gong
  - In order to use Gong, the Zoom call must be hosted by someone with a Fleet email address.  
  - You cannot use Gong to record calls hosted by external parties.
  - Cloud recording in Zoom has to be turned on and unlocked company wide for Gong to function properly, because of this, there is a chance that some Gong recordings may still save in Zoom's cloud storage even if they aren't uploaded into Gong.
  - To counter this, Nathan Holliday will periodically delete all recordings found in Zoom's storage without viewing them.

>If you need help using Gong, please check out Gong Academy at [https://academy.gong.io/](https://academy.gong.io/).

##### Excluding calls from being recorded
For those with a Gong seat or scheduling a call with someone in attendance that has a Gong seat who does not wish for their Zoom call with an external party to record, make sure your calendar event title contains `[no shadows]`.  You can also read the [complete list of exclusion rules](https://docs.google.com/document/d/1OOxLajvqf-on5I8viN7k6aCzqEWS2B24_mE47OefutE/edit?usp=sharing).


#### Zapier and DocuSign
We use Zapier to automate how completed DocuSign envelopes are formatted and stored. This process ensures we store signed documents in the correct folder and that filenames are formatted consistently. 
When the final signature is added to an envelope in DocuSign, it is marked as completed and sent to Zapier, where it goes through these steps:
1. Zapier sends the following information about the DocuSign envelope to our Hydroplane webhook:
   - **`emailSubject`** - The subject of the envelope sent by DocuSign. Our DocuSign templates are configured to format the email subject as `[type of document] for [signer's name]`.
   - **`emailCsv`** - A comma-separated list of signers' email addresses.
2. The Hydroplane webhook matches the document type to the correct Google Drive folder, orders the list of signers, creates a timestamp, and sends that data back to Zapier as
   - **`destinationFolderID`** - The slug for the Google Drive folder where we store this type of document.
   - **`emailCsv`** - A sorted list of signers' email addresses.
   - **`date`** - The date the document was completed in DocuSign, formatted YYYY-MM-DD.
3. Zapier uses this information to upload the file to the matched Google Drive folder, with the filename formatted as `[date] - [emailSubject] - [emailCvs].PDF`.
4. Once the file is uploaded, Zapier uses the Slack integration to post in the #peepops channel with the message:
   ```
   Now complete with all signatures:
      [email subject]
      link: drive.google.com/[destinationFolderID]
   ```

#### Namecheap

Domain name registrations are handled through Namecheap. Access is managed via 1Password.

#### Github
##### GitHub labels

We use special characters to define different types of GitHub labels. By combining labels, we
organize and categorize GitHub issues. This reduces the total number of labels required while
maintaining an expressive labeling system. For example, instead of a label called
`platform-dev-backend`, we use `#platform :dev ~backend`.

| Special character | Label type  | Examples                            |
|:------------------|:------------|:------------------------------------|
| `#`               | Noun        | `#platform`, `#interface`, `#agent`
| `:`               | Verb        | `:dev`, `:research`, `:design`
| `~`               | Adjective   | `~blocked`, `~frontend`, `~backend`

> TODO: extrapolate to "why this way" page

### Equipment
#### Laptops
##### Purchasing a company-issued device
Fleet provides laptops for core team members to use while working at Fleet. As soon as an offer is accepted, Business Operations will reach out to the new team member to start this process and will work with the new team member to get their laptop purchased and shipped to them on time.

Apple computers shipping to the United States and Canada are ordered using the Apple [eCommerce Portal](https://ecommerce2.apple.com/asb2bstorefront/asb2b/en/USD/?accountselected=true), or by contacting the business team at an Apple Store or contacting the online sales team at [800-854-3680](tel:18008543680). The business team can arrange for same-day pickup at a store local to the Fleetie if needed.

When ordering through the Apple eCommerce Portal, look for a banner with *Apple Store for FLEET DEVICE MANAGEMENT | Welcome [Your Name].* Hovering over *Welcome* should display *Your Profile.* If Fleet's account number is displayed, purchases will be automatically made available in Apple Business Manager (ABM).

Apple computers for Fleeties in other countries should be purchased through an authorized reseller to ensure the device is enrolled in ADE. In countries that Apple does not operate or that do not allow ADE, work with the authorized reseller to find the best solution, or consider shipping to a US based Fleetie and then shipping on to the teammate. 

##### Selecting a laptop
Most Fleeties use 16-inch MacBook Pros. Team members are free to choose any laptop or operating system that works for them, as long as the price [is within reason](#spending-company-money) and supported by our device management solution.  (Good news: Since Fleet uses Fleet for device management, every operating system is supported!)

When selecting a new laptop for a team member, optimize their configuration to:
1. Have a reasonably large storage (at least 512GB of storage, and if there's any concern go bigger)
2. Look for pre-configured models with the desired memory and storage requirements. These tend to be available for delivery or pickup as quickly as possible and before the start date.

> If delivery timelines are a concern with no devices in stock, play around with build until it ships as quickly as possible.  Sometimes small changes lead to much faster ship times.  More standard configurations (with fewer customizations) usually ship more quickly.  Sometimes MacBook Pros ship more quickly than MacBook Airs, and vice versa.  This varies.  Remember: Always play around with the build and optimize for something that will **ship quickly**!

For example, someone in sales, marketing, or business operations might like to use a 14-inch Macbook Air, whereas someone in an engineering, product, or design role might use a 16-inch MacBook Pro.  **Default to a 16-inch MacBook Pro.**

> A 3-year AppleCare+ Protection Plan (APP) should be considered default for Apple computers >$1500. Base MacBook Airs, Mac minis, etc. do not need APP unless configured beyond the $1500 price point. APP provides 24/7 support, and global repair coverage in case of accidental screen damage or liquid spill, and battery service.

Windows and Linux devices are available upon request for team members in product and engineering.  (See [Buying other new equipment](#buying-other-new-equipment).)

#### Buying other new equipment
At Fleet, we [spend company money like it's our own money](https://fleetdm.com/handbook/business-operations#spending-company-money).  If you need equipment above and beyond those standard guidelines, you can request new equipment by creating a GitHub issue in fleetdm/fleet and attaching the `#g-business-operations`.  Please include a link to the requested equipment (including any specs), the reason for the request, and a timeline for when the device is needed. 

#### Tracking equipment
When a device has been purchased, it's added to the [spreadsheet of company equipment](https://docs.google.com/spreadsheets/d/1hFlymLlRWIaWeVh14IRz03yE-ytBLfUaqVz0VVmmoGI/edit#gid=0) where we keep track of devices and equipment, purchased by Fleet.

When you receive your new computer, complete the entry by adding a description, model, and serial number to the spreadsheet.

#### Returning equipment
Equipment should be returned once offboarded for reprovisioning. Coordinate offboarding and return with the Head of Business Operations. 

#### Reprovisioning equipment
Apple computers with remaining AppleCare Protection Plans should be reprovisioned to other Fleeties who may have older or less-capable computers.

#### Equipment retention and replacement
Older equipment results in lost productivity of Fleeties and should be considered for replacement. Replacement candidates are computers that are no longer under an AppleCare+ Protection Plan (or another warranty plan), are >3 years from the [discontinued date](https://everymac.com/systems/apple/macbook_pro/index-macbookpro.html#specs), or when the "Battery condition" status in Fleet is less than "Normal". The old equipment should be evaluated for return or retention as a test environment.

> If your Apple device is less than 3 years old, has normal battery condition, but is experiencing operating difficulties, you should first contact Apple support and troubleshoot performance issues before requesting a new device.


##### Open positions

Please see [handbook/company#open-positions](https://fleetdm.com/handbook/company#open-positions) for a list of open job postings at Fleet.

#### Stubs
The following stubs are included only to make links backward compatible.

##### Meetings
##### Scheduling a meeting
##### Internal meeting scheduling
##### Modifying an event organized by someone else
##### External meeting scheduling

Please see [handbook/company/communication](https://fleetdm.com/handbook/company/communication#meetings) for all meetings sections


<meta name="maintainedBy" value="jostableford">
<meta name="title" value="🔦 Business Operations">

# Leadership

## Vision for dept handbook pages
The idea here is to get this vision implemented on a single departmental handbook page first, starting with handbook/company/ceo.  It's hard to know what the philosophy should be until we can see it.  So we need to shorten the feedback loop so we can see it change live in one place.  That way we can iterate in one place instead of having things go a bunch of different directions, and adding in all the complexity of extra redirects to keep track of and all that stuff.  Then once we've got that looking good and have iterated a bit, we'll spread it out.

Another thing is that we need to get a better intuitive understanding of who these pages are designed to serve.  So in order to put ourselves in their shoes (get behind their eyeballs), we need something to look at.  So doing a particular page first provides us with that canvas.

From Mike - 2023-08-21

<blockquote purpose="large-quote">
Biggest learning is that we federated carte blanche edit permissions a bit too early back in 2021, and it’s resulted in the need for a lot of cleanup as different people have had their hands in the content prior to introducing a framework for organizing that content.  

  For reference, Sid at Gitlab didn’t delegate ownership over pages away from a single individual (him) until they were close to 100 employees, whereas at Fleet we did it in the 15 employee stage, and are dealing with the consequences.
It meant that until recently, about 1/3 of the Fleet handbook was completely wrong, duplicated, or out of date.  (We’re probably down to only 25% now, and falling!)
Joanne and team did some planning during the bizops offsite, and Sam and I took that and applied it to the ceo and bizops handbook pages yesterday.

We’re going to do the same thing gradually for marketing, then sales, then engineering, then product.
Content related to onboarding and policies like vacation is now in: https://fleetdm.com/handbook/company#every-day

The audience for the “Communications” page is every fleetie.

The audience for the “Leadership” page is every manager.

The audience for individual department pages are the people working with and within that department (in that order, with “Contact us” and other generally useful information and intake channels listed first)
This pass through the handbook has also eliminated several pages in favor of getting more onto single pages.  This is because there is still a lot of duplication, and it’s easier to deal with when everything is on a single page.

Dear onboardees: could you update broken links in the onboarding issue template as you find them? Everything should still redirect correctly, or provide a path to get to the right place through “Stubs”, but it’s helpful to have the links point directly to the right place.
If you have any questions or feedback, please contact us: https://fleetdm.com/handbook/ceo#contact-us
</blockquote>

### Outline of departmental page structure

- `# Name of department`
  - "This handbook page details processes specific to working `[with](#contact-us)` and `[within](#responsibilities)` this department." 

  - `## Team`
    - Table that displays each position and the team member(s) that fill that position, linking each Fleetie's LinkedIn to their name and GitHub to GitHub user name. See [handbook/ceo#team](https://fleetdm.com/handbook/ceo#team) for example.

  - `## Contact us`
    - "To make a request of this department, `[create an issue](https://github.com/fleetdm/confidential/issues/new?assignees=&labels=%23{DEPARTMENTAL-GITHUB-LABEL}&projects=&template=custom-request.md&title=Request%3A+_______________________)` and a team member will get back to you within one business day (If urgent, mention a `[team member](#team)` in `[Slack]({DEPARTMENTAL-SLACK-CHANNEL-LINK})`)."
      - "Please **use issue comments and GitHub mentions** to communicate follow-ups or answer questions related to your request."
      - "Any Fleet team member can `[view the kanban board](https://app.zenhub.com/workspaces/{DEPARTMENTAL-KANBAN-BOARD-LINK}/board?sprints=none)` for this department, including pending tasks and the status of new requests."

- `## What we do`
    - Outline the direct responsibilities of the department. What value do you provide to Fleet's contributors.  

- `## Responsibilities`
  - The "Responsibilities" section consists of sub-headings written in the imperative mood (e.g. "Process CEO inbox") and designed to be the internal "How-to" of each department.  


- `## Rituals`


### Key reviews
Every release cycle, each department leader discusses their [key performance indicators (KPIs)](https://docs.google.com/spreadsheets/d/1Hso0LxqwrRVINCyW_n436bNHmoqhoLhC8bcbvLPOs9A/edit#gid=0) (confidential) with the CEO.  KPIs are numbers measuring results and everyday excellence, usually accompanied by time-bound goals.

In this meeting, the department leader discusses actual week-over-week progress toward the goals for a particular quarter with the CEO.

- Key reviews are scheduled during the e-group time slot every three weeks and are not moved or rescheduled.  If a department leader is not available to lead a particular key review, another team member from their department will join the meeting and discuss their department's key performance indicators (KPIs).
- Use this meeting to add, remove, or change the definitions or ownership of KPIs.  Otherwise, KPI definitions do not change, even if those definitions have problems.  For help with KPIs, get [input from the CEO](https://fleetdm.com/handbook/ceo#contact-us).



## Hiring

At Fleet, we collaborate with [core team members](#creating-a-new-position), [consultants](#hiring-a-consultant), [advisors](#adding-an-advisor), and [outside contributors](https://github.com/fleetdm/fleet/graphs/contributors) from the community.  

> Are you a new fleetie joining the Business Operations team?  For Loom recordings demonstrating how to make offers, hire, onboard, and more please see [this classified Google Doc](https://docs.google.com/document/d/1fimxQguPOtK-2YLAVjWRNCYqs5TszAHJslhtT_23Ly0/edit).

### Consultants

#### Hiring a consultant

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

To hire a consultant, [submit a custom request](https://fleetdm.com/handbook/business-operations#intake) to the business operations team.

> TODO: replace this w/  issue template (see also commented-out notes in hiring.md for some other steps)

#### Who ISN'T a consultant?

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

#### Sending a consulting agreement

To hire a non-US consultant, please [submit a custom request](https://fleetdm.com/handbook/business-operations#intake).

To hire a US-based consultant, send them an agreement using the "Contractor agreement (US)" template in [DocuSign](https://www.docusign.com/).
(This template is located in the "¶¶ Classified templates" folder, which is only accessible via certain Docusign accounts in 1Password.)

> _**Note:** The Docusign template labeled "Contractor agreement (US)" is actually used for both consultants and [core team members in the US who are classified as 1099 contractors or billed corp-to-corp as vendors](#hiring-a-new-team-member).  You may also sometimes hear this referred to as Fleet's "Consulting agreement". Same thing._

To send a US consulting agreement, you'll need the new consultant's name, the term of the service, a summary of the services provided, and the consultant's fee. 

There are some defaults that we use for these agreements:
   - Term: Default to one month unless otherwise discussed.
   - Services rendered: Copy and paste from the [language in this doc](https://docs.google.com/document/d/1b5SGgYEHqDmq5QF8p29WWN3it3XJh3xRT3zG0RdXARo/edit)
   - Work will commence and complete by dates: Start date and end of term date
   - Fee: Get from the consultant.
   - Hours: Default to 10 hr/week.
   - All US consultants track their hours weekly in Gusto.

Then hit send!  After all of the signatures are there in Docusign, automation will trigger that uploads the completed document to the appropriate Google Drive folder, and that makes a Slack message appear in the `#help-classified` channel.

Finally, create a [custom request](https://fleetdm.com/handbook/business-operations#intake) titled "New US consultant: _____________" and request that this new consultant be registered with Fleet.  (Business Operations will receive this request and take care of next steps, which include things like providing a place for the company to report their hours weekly in the KPIs sheet, and providing access to Slack and any relevant company tools.)

#### Updating a consultant's fee
 - Direct message Mike McNeil with hourly rate change information.
 - After CEO approval, Mike McNeil will issue a new contractor agreement with the updated fee via DocuSign.

### Advisor

#### Adding an advisor
Advisor agreements are sent through [DocuSign](https://www.docusign.com/), using the "Advisor Agreement"
template.
- Send the advisor agreement. To send a new advisor agreement, you'll need the new advisor's name and the number of shares they are offered. 
- Once you send the agreement, locate an existing empty row and available ID in ["Advisors"](https://docs.google.com/spreadsheets/d/15knBE2-PrQ1Ad-QcIk0mxCN-xFsATKK9hcifqrm0qFQ/edit#gid=1803674483) and enter the new advisor's information.
   >**_Note:_** *Be sure to mark any columns that haven't been completed yet as "TODO"*

#### Finalizing a new advisor
- Update the ["Advisors"](https://docs.google.com/spreadsheets/d/15knBE2-PrQ1Ad-QcIk0mxCN-xFsATKK9hcifqrm0qFQ/edit#gid=1803674483) to show that the agreement has been signed, and ask the new advisor to add us on [LinkedIn](https://www.linkedin.com/company/71111416), [Crunchbase](https://www.crunchbase.com/organization/fleet-device-management), and [Angellist](https://angel.co/company/fleetdm).
- Update "Equity plan" to reflect updated status and equity grant for this advisor, and to ensure the advisor's equity is queued up for the next quarterly equity grant ritual.

### Core team member
This section is about creating a core team member role, and the hiring process for a new core team member, or Fleetie.

#### Creating a new position

Want to hire?  Here's how to open up a new position on the core team:

> Use these steps to hire a [fleetie, not a consultant](https://fleetdm.com/handbook/business-operations#who-isnt-a-consultant).

<!--
> If you think this job posting may need to stay temporarily classified (¶¶) and not shared company-wide or publicly yet, for any reason, then stop here and send a Slack DM with your proposal to the CEO instead of modifying ["🧑‍🚀 Fleeties"](https://docs.google.com/spreadsheets/d/1OSLn-ZCbGSjPusHPiR5dwQhheH1K8-xqyZdsOe9y7qc/edit) (which is visible company-wide) or submitting a draft pull request to "Open positions" (which is public).
-->

1. **Propose headcount:** Add the proposed position to ["🧑‍🚀 Fleeties"](https://docs.google.com/spreadsheets/d/1OSLn-ZCbGSjPusHPiR5dwQhheH1K8-xqyZdsOe9y7qc/edit#gid=0) in an empty row (but using one of the existing IDs.  Unsure?  Ask for help.)  Be sure to include job title, manager, and department. Set the start date to the first Monday of the next month (This position is still only proposed (not approved), but would make it easier for the approver to have the date set).
2. **Propose job description:** Copy, personalize, and publish the job description:
  - Create an entry for the proposed position in the [open positions YAML file](https://github.com/fleetdm/fleet/tree/main/handbook/company/open-positions.yml). To do this, you can either duplicate an existing open position and update the values, or you can copy and paste the commented out template at the top of the file.

  - Update the required values for the new entry:
    - `jobTitle`: The job title of the proposed position. This will determine the living URL of the page on the Fleet website.
    - `department`: The department of the proposed position.
    - `hiringManagerName`: The full name of this proposed position's hiring manager.
    - `hiringManagerGithubUsername`: The GitHub username of the proposed position's hiring manger. This is used to add the hiring manager as the open position page's maintainer.
    - `hiringManagerLinkedInUrl`: The url of the hiring manger's LinkedIn profile. People applying for this position will be asked to reach out to the manager on LinkedIn.
    - `responsibilities`: A Markdown list of the responsibilities of this proposed position.
    - `experience`: A Markdown list of the experience that applicants should have when applying for the proposed position.


A completed open position entry should look something like this:

```
- jobTitle: 🐈 Railway cat
  department: Jellicle cats
  hiringManagerName: Skimbleshanks
  hiringManagerLinkedInUrl: https://www.linkedin.com/in/skimbleshanks-the-railway-cat
  hiringManagerGithubUsername: skimbieshanks
  responsibilities: |
    - ⏫ Elevate the standard of train travel
    - 📖 Learn the ins and outs of rail operations
    - 🏃‍♂️ Dash through stations to ensure punctuality
  experience: |
    - 🎯 Punctuality is crucial
    - 🌐 Familiarity with the Northern Line
    - 👥 Excellent at commanding attention
    - 🤝 Adept at coordinating with the Night Mail
    - 🦉 Skilled at nocturnal operations
    - 🛠️ Proficient in tap-dance communication
    - 🟣 Ability to maintain railway order and standards
    - 🐭 Can swiftly and silently eliminate any rodent problems
    - 💭 Speak the language of timetable jargon
    - 💖 Sing praises of a smooth rail journey
    - 🐭 Can articulate effective rodent control strategies
```

  - Create a pull request to add the new position to the YAML file.

- _**Note:** The "living" URL where the new page will eventually exist on fleetdm.com won't ACTUALLY exist until your pull request is merged. A link will be added in the ["Open positions" section](https://fleetdm.com/handbook/company#open-positions) of the company handbook page.

3. **Link to pull request in "Fleeties:"** Include a link to your GitHub pull request in the "Job description" column for the new row you just added in "Fleeties".

4. **Get it approved and merged:**  When you submit your proposed job description, the CEO will be automatically tagged for review and get a notification.  He will consider where this role fits into Fleet's strategy and decide whether Fleet will open this position at this time.  He will review the data carefully to try and catch any simple mistakes, then tentatively budget cash and equity compensation and document this compensation research.  He will set a tentative start date (which also indicates this position is no longer just "proposed"; it's now part of the hiring plan.)  Then the CEO will start a `#hiring-xxxxx-YYYY` Slack channel, at-mentioning the original proposer and letting them know their position is approved.  (Unless it isn't.)

- _**Why bother with approvals?**  We avoid cancelling or significantly changing a role after opening it.  It hurts candidates too much.  Instead, get the position approved first, before you start recruiting and interviewing.  This gives you a sounding board and avoids misunderstandings._

#### Approving a new position
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
   - _Publish opening:_ Approve and merge the pull request.  The job posting will go live within ≤10 minutes.
   - _Track as approved in "Fleeties":_ In the "Fleeties" spreadsheet, find the row for the new position and update the "Job description" column and replace the URL of the pull request that originally proposed this new position with the URL of the GitHub merge commit when that PR was merged.
   - _Reply to requestor:_ Post a comment on the pull request, being sure to include a direct link to their live job description on fleetdm.com.  (This is the URL where candidates can go to read about the job and apply.  For example: `fleetdm.com/handbook/company/product-designer`):
     ```
     The new opening is now live!  Candidates can apply at fleetdm.com/handbook/company/railway-conductor.
     ```

> _**Note:** Most columns of the "Equity plan" are updated automatically when "Fleeties" is, based on the unique identifier of each row, like `🧑‍🚀890`.  (Advisors have their own flavor of unique IDs, such as `🦉755`, which are defined in ["Advisors and investors"](https://docs.google.com/spreadsheets/d/15knBE2-PrQ1Ad-QcIk0mxCN-xFsATKK9hcifqrm0qFQ/edit).)_

#### Recruiting
Fleet accepts job applications, but the company does not list positions on general purpose job boards.  This prevents us being overwhelmed with candidates so we can fulfill our goal of responding promptly to every applicant.

This means that outbound recruiting, 3rd party recruiters, and references from team members are important aspect of the company's hiring strategy.  Fleet's CEO is happy to assist with outreach, intros, and recruiting strategy for candidates.

#### Receiving job applications
Every job description page ends with a "call to action", including a link that candidates can click to apply for the job.  Fleet replies to all candidates within **1 business day** and always provides either a **rejection** or **decisive next steps**; even if the next step is just a promise.  For example:

> "We are still working our way through applications and _still_ have not been able to review yours yet.  We think we will be able to review and give you an update about your application by Thursday at the latest.  I'll let you know as soon as I have news.  I'll assume we're both still in the running if I don't hear from you, so please let me know if anything comes up."

When a candidate clicks applies for a job at Fleet, they are taken to a generic Typeform.  When they submit their job application, the Typeform triggers a Zapier automation that will posts the submission to `g-business-operations` in Slack.  The candidate's job application answers are then forwarded to the applicable `#hiring-xxxxx-202x` Slack channel and the hiring manager is @mentioned.

#### Candidate correspondence email templates
Fleet uses [certain email templates](https://docs.google.com/document/d/1E_gTunZBMNF4AhsOFuDVi9EnvsIGbAYrmmEzdGmnc9U) when responding to candidates.  This helps us live our value of [🔴 empathy](https://fleetdm.com/handbook/company#empathy) and helps the company meet the aspiration of replying to all applications within one business day.

#### Hiring restrictions

##### Incompatible former employers
Fleet maintains a list of companies with whom Fleet has do-not-solicit terms that prevents us from making offers to employees of these companies.  The list is in the Do Not Solicit tab of the [BizOps spreadsheet](https://docs.google.com/spreadsheets/d/1lp3OugxfPfMjAgQWRi_rbyL_3opILq-duHmlng_pwyo/edit#gid=0).

##### Incompatible locations
Fleet is unable to hire team members in some countries. See [this internal document](https://docs.google.com/document/d/1jHHJqShIyvlVwzx1C-FB9GC74Di_Rfdgmhpai1SPC0g/edit) for the list.

#### Interviewing
> TODO: Rewrite this section for the hiring manager as our audience.

We're glad you're interested in joining the team! 
Here are some of the things you can anticipate throughout this process:
  - We will reply by email within one business day from the time when the application arrives.
  - You may receive a rejection email (Bummer, consider applying again in the future).
  - You may receive an invitation to "book with us."
If you've been invited to "book with us," you'll have a Zoom meeting with the hiring team to discuss the next steps.

Department specific interviewing instructions:
- [Engineering](https://fleetdm.com/handbook/engineering#interview-a-developer-candidate)

#### Hiring a new team member
This section is about the hiring process a new core team member, or fleetie.

> **_Note:_** _Employment classification isn't what makes someone a fleetie.  Some Fleet team members are contractors and others are employees.  The distinction between "contractor" and "employee" varies in different geographies, and the appropriate employment classification and agreement for any given team member and the place where they work is determined by Head of Business Operations during the process of making an offer._

Here are the steps hiring managers follow to get an offer out to a candidate:
1. **Call references:** Before proceeding, make sure you have 2-5+ references. Ask the candidate for at least 2-5+ references and contact each reference in parallel using the instructions in [Fleet's reference check template](https://docs.google.com/document/d/1LMOUkLJlAohuFykdgxTPL0RjAQxWkypzEYP_AT-bUAw/edit?usp=sharing).  Be respectful and keep these calls very short.
2. **Add to team database:** Update the [Fleeties](https://docs.google.com/spreadsheets/d/1OSLn-ZCbGSjPusHPiR5dwQhheH1K8-xqyZdsOe9y7qc/edit#gid=0) doc to accurately reflect the candidate's:
   - Start date
     > _**Tip:** No need to check with the candidate if you haven't already.  Just guess.  First Mondays tend to make good start dates.  When hiring an international employee, Pilot.co recommends starting the hiring process a month before the new employee's start date._
   - First and last name
   - Preferred pronoun _("them", "her", or "him")_
   - LinkedIn URL _(If the fleetie does not have a LinkedIn account, enter `N/A`)_
   - Location of candidate
3. **Schedule CEO interview:** [Book a quick chat](https://fleetdm.com/handbook/ceo#contact-us) so our CEO can get to know the future Fleetie.  (Please take care of all of the previous steps first.)
4. **Confirm intent to offer:** Compile feedback about the candidate into a single document and share that document (the "interview packet") with the Head of Business Operations via Google Drive.  _This will be interpreted as a signal that you are ready for them to make an offer to this candidate._
   - _Compile feedback into a single doc:_ Include feedback from interviews, reference checks, and challenge submissions.  Include any other notes you can think of offhand, and embed links to any supporting documents that were impactful in your final decision-making, such as portfolios or challenge submissions.
   - _Share_ this single document with the Head of Business Operations via email.
     - Share only _one, single Google Doc, please_; with a short, formulaic name that's easy to understand in an instant from just an email subject line.  For example, you could title it:
       >Why hire Jane Doe ("Train Conductor") - 2023-03-21
     - When the Head of Business Operations receives this doc shared doc in their email with the compiled feedback about the candidate, they will understand that to mean that it is time for Fleet to make an offer to the candidate.

#### Making an offer
After receiving the interview packet, the Head of Business Operations uses the following steps to make an offer:

<!-- For future use: some ready-to-go language around rebencharking compensation for cost of living: https://github.com/fleetdm/fleet/pulls/13499 -->
1. **Prepare the "exit scenarios" spreadsheet:** 🔦 Head of Business Operations [copies the "Exit scenarios (template)"](https://docs.google.com/spreadsheets/d/1k2TzsFYR0QxlD-KGPxuhuvvlJMrCvLPo2z8s8oGChT0/copy) for the candidate, and renames the copy to e.g. "Exit scenarios for Jane Doe".
   - _Edit the candidate's copy of the exit scenarios spreadsheet_ to reflect the number of shares in ["🥧 Equity plan"](https://docs.google.com/spreadsheets/d/1_GJlqnWWIQBiZFOoyl9YbTr72bg5qdSSp4O3kuKm1Jc/edit#gid=0), and the spreadsheet will update automatically to reflect their approximate ownership percentage.
     > _**Note:** Don't play with numbers in the exit scenarios spreadsheet. The revision history is visible to the candidate, and they might misunderstand._
2. **Prepare offer:** 🔦 Head of Business Operations [copies "Offer email (template)"](https://docs.google.com/document/d/1zpNN2LWzAj-dVBC8iOg9jLurNlSe7XWKU69j7ntWtbY/copy) and renames to e.g. "Offer email for Jane Doe".  Edit the candidate's copy of the offer email template doc and fill in the missing information:
   - _Benefits:_ If candidate will work outside the US, [change the "Benefits" bullet](https://docs.google.com/document/d/1zpNN2LWzAj-dVBC8iOg9jLurNlSe7XWKU69j7ntWtbY/edit) to reflect what will be included through Fleet's international payroll provider, depending on the candidate's location.
   - _Equity:_ Highlight the number of shares with a link to the candidate's custom "exit scenarios" spreadsheet.
   - _Hand off:_ Share the offer email doc with the [Apprentice to the CEO](https://fleetdm.com/handbook/company/ceo#team).
3. **Draft email:** 🦿 Apprentice to the CEO drafts the offer email in the CEO's inbox, reviews one more time, and then brings it to their next daily meeting for CEO's approval:
   - To: The candidate's personal email address _(use the email from the CEO interview calendar event)_
   - Cc: Zach Wasserman and Head of Business Operations _(neither participate in the email thread until after the offer is accepted)_
   - Subject: "Full time?"
   - Body: _Copy the offer email verbatim from the Google doc into Gmail as the body of the message, formatting and all, then:_
     - _Check all links in offer letter for accuracy (e.g. LinkedIn profile of hiring manager, etc.)_
     - _Click the surrounding areas to ensure no "ghost links" are left from previous edits... which has happened before._
     - _Re-read the offer email one last time, and especially double-check that the salary, number of shares, and start date match the numbers that are currently in the equity plan._
4. **Send offer:** 🐈‍⬛ CEO reviews and sends the offer to the candidate:
   - _Grant the candidate "edit" access_ to their "exit scenarios" spreadsheet.
   - _Send_ the email.

#### Steps after an offer is accepted
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
3. **Remove open position:** The hiring manager removes the newly-filled position from the fleetdm.com website by [making a pull request](https://fleetdm.com/handbook/company/communications#making-a-pull-request) to delete it from the [open-positions.yml](https://github.com/fleetdm/fleet/blob/main/handbook/company/open-positions.yml) file.
4. **Close Slack channel:** Then archive and close the channel.

Now what happens?  🔦 Business Operations will then follow the steps in the "Hiring" issue, which includes reaching out to the new team member within 1 business day from a separate email thread to get additional information as needed, prepare their agreement, add them to the company's payroll system, and get their new laptop and hardware security keys ordered so that everything is ready for them to start on their first day.



## Tracking hours
Fleet asks US-based hourly contributors to track hours in Gusto, and contributors outside the US to track hours via Pilot.co.

This applies to anyone who gets paid by the hour, including consultants and hourly core team members of any employment classification, inside or outside of the US.

> _**Note:** If a contributor uses their own time-tracking process or tools, then it is OK to track the extra time spent tracking!  Contributors at Fleet are evaluated based on their results, not the number of hours they work._


## Communicating departures
Although it's sad to see someone go, Fleet understands that not everything is meant to be forever [like open-source is](https://fleetdm.com/handbook/company/why-this-way#why-open-source). There are a few steps that the company needs to take to facilitate a departure. 
1. **Departing team member's manager:** Inform the Head of Business Operations about the departure via email and cc your manager. The Head of Business Operations will coordinate the team member's last day, offboarding, and exit meeting.
3. **Business Operations**: Will then create and begin completing [offboarding issue](https://github.com/fleetdm/classified/blob/main/.github/ISSUE_TEMPLATE/%F0%9F%9A%AA-offboarding-____________.md), to include coordinating team member's last day, offboarding, and exit meeting.
   > After finding out about the departure, the Head of Business Operations will post in #g-e to inform the E-group of the team member's departure, asking E-group members to inform any other managers on their teams.
4. **CEO**: The CEO will make an announcement during the "🌈 Weekly Update" post on Friday in the `#general` channel on Slack. 


## Changing someone's position

From time to time, someone's job title changes.  To do this, Business Operations follows these steps:

1. Change "Fleeties" to reflect the new job title, manager, and/or department.
2. If there is a compensation change, update "Equity plan".  Use the first day of a month as the date, and enter this in the corresponding column.
3. If applicable, schedule the change in the appropriate payroll system.  (Don't worry about updating job titles in the payroll system.)



<meta name="maintainedBy" value="mikermcneil">
<meta name="title" value="🛠️ Leadership">

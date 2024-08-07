# Digital Experience 

This page details processes specific to working [with](#contact-us) and [within](#responsibilities) this department.


## Team

| Role                            | Contributor(s)
|:--------------------------------|:----------------------------------------------------------------------|
| Head of Digital Experience      | [Sam Pfluger](https://www.linkedin.com/in/sampfluger88/) _([@sampfluger88](https://github.com/sampfluger88))_ 
| Head of Design                  | [Mike Thomas](https://www.linkedin.com/in/mike-thomas-52277938) _([@mike-j-thomas](https://github.com/mike-j-thomas))_
| Software Engineer               | [Eric Shaw](https://www.linkedin.com/in/eric-shaw-1423831a9/) _([@eashaw](https://github.com/eashaw))_
| Apprentice to the CEO           | <sup><sub> See [Head of Digital Experience](https://www.fleetdm.com/handbook/digital-experience#team) <sup><sub>
| Apprentice                      | [Savannah Friend](https://www.linkedin.com/in/savannah-friend-2b1a53148/) _([@sfriendlee](https://github.com/sfriendlee))_


## Contact us

- To **make a request** of this department, [create an issue](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=%23g-digital-experience&projects=&template=digital-experience-request.md&title=TODO%3A+) and a team member will get back to you within one business day (If urgent, mention a [team member](#team) in the [#g-digital-experience](https://fleetdm.slack.com/archives/C058S8PFSK0) Slack channel.
  - Any Fleet team member can [view the kanban board](https://app.zenhub.com/workspaces/g-digital-experience-6451748b4eb15200131d4bab/board?sprints=none) for this department, including pending tasks and the status of new requests.
  - Please **use issue comments and GitHub mentions** to communicate follow-ups or answer questions related to your request.


## Responsibilities

The Digital Experience department is directly responsible for the framework, content design, and technology behind Fleet's remote work culture, including fleetdm.com, the handbook, issue templates, UI style guides, internal tooling, Zapier flows, Docusign templates, key spreadsheets, and project management processes. 

> _**Note:** If a user story involves only changes to fleetdm.com, without changing the core product, then that user story is prioritized, drafted, implemented, and shipped by the [Digital Experience](https://fleetdm.com/handbook/digital-experience) department.  Otherwise, if the story **also** involves changes to the core product **as well as** fleetdm.com, then that user story is prioritized, drafted, implemented, and shipped by [the other relevant product group](https://fleetdm.com/handbook/company/product-groups#current-product-groups), and not by `#g-digital-experience`._


### QA a change to fleetdm.com

Each PR to the website is manually checked for quality and tested before going live on fleetdm.com. To test any change to fleetdm.com

1. Write clear step-by-step instructions to confirm that the change to the fleetdm.com functions as expected and doesn't break any possible automation. These steps should be simple and clear enough for anybody to follow.

2. [View the website locally](https://fleetdm.com/handbook/digital-experience#test-fleetdm-com-locally) and follow the QA steps in the request ticket to test changes.

3. Check the change in relation to all breakpoints and [browser compatibility](https://fleetdm.com/handbook/digital-experience#check-browser-compatibility-for-fleetdm-com), Tests are carried out on [supported browsers](https://fleetdm.com/docs/using-fleet/supported-browsers) before website changes go live.


### Update the host count of a premium subscription

When a self-service license dispenser customer reaches out to upgrade a license via the contact form, a member of the [Demand department](https://fleetdm.com/handbook/demand) will create a confidential issue detailing the request and add it to the new requests column of [Digital Experience kanban board](https://github.com/fleetdm/confidential/issues#workspaces/g-digital-experience-6451748b4eb15200131d4bab/board). A member of this team will then log into Stripe using the shared login, and upgrade the customer's subscription.

To update the host count on a user's subscription:

1. Log in to the [Stripe dashboard](https://dashboard.stripe.com/dashboard) and search for the customer's email address.
2. Click on their subscription and select the "Update subscription" option in the "Actions" dropdown
3. Update the quantity of the user's subscription to be their desired host count.
4. Turn the "Proration charges" option on and select the "Charge proration amount immediately" option.
5. Under "Payment" select "Email invoice to the customer", and set the payment due date to be 15 days, and make sure the "Invoice payment page" option is checked.
6. Select "Update subscription" to send the user an updated invoice for their subscription. Once the customer pays their new invoice, the Fleet website will update the user's subscription and generate a new Fleet Premium license with an updated host count.
7. Let the person who created the request know what actions were taken so they can communicate them to the customer.


### Test fleetdm.com locally 

When making changes to the Fleet website, you can test your changes by running the website locally. To do this, you'll need the following:

- A local copy of the [Fleet repo](https://github.com/fleetdm/fleet).
- [Node.js](https://nodejs.org/en/download/)
- (Optional) [Sails.js](https://sailsjs.com/) installed globally on your machine (`npm install sails -g`)

Once you have the above follow these steps:

1. Open your terminal program, and navigate to the `website/` folder of your local copy of the Fleet repo.
    
    > Note: If this is your first time running this script, you will need to run `npm install` inside of the website/ folder to install the website's dependencies.


2. Run the `build-static-content` script to generate HTML pages from our Markdown and YAML content.
  - **With Node**, you will need to use `node ./node_modules/sails/bin/sails run build-static-content` to execute the script.
  - **With Sails.js installed globally** you can use `sails run build-static-content` to execute the script.

    > When this script runs, the website's configuration file ([`website/.sailsrc`](https://github.com/fleetdm/fleet/blob/main/website/.sailsrc)) will automatically be updated with information the website uses to display content built from Markdown and YAML. Changes to this file should never be committed to the GitHub repo. If you want to exclude changes to this file in any PRs you make, you can run this terminal command in your local copy of the Fleet repo: `git update-index --assume-unchanged ./website/.sailsrc`.
    
    > Note: You can run `npm run start-dev` in the `website/` folder to run the `build-static-content` script and start the website server with a single command.

3. Once the script is complete, start the website server:
  - **With Node.js:** start the server by running `node ./node_modules/sails/bin/sails lift`
  - **With Sails.js installed globally:** start the server by running `sails lift`.

4. When the server has started, the Fleet website will be available at [http://localhost:2024](http://localhost:2024)
    
  > **Note:** Some features, such as self-service license dispenser and account creation, are not available when running the website locally. If you need help testing features on a local copy, reach out to `@eashaw` in the [#g-digital-experience](https://fleetdm.slack.com/archives/C058S8PFSK0) channel on Slack.


### Check production dependencies of fleetdm.com

Every week, we run `npm audit --only=prod` to check for vulnerabilities on the production dependencies of fleetdm.com. Once we have a solution to configure GitHub's Dependabot to ignore devDependencies, this manual process can be replaced with Dependabot.


### Respond to a 5xx error on fleetdm.com

Production systems can fail for various reasons, and it can be frustrating to users when they do, and customer experience is significant to Fleet. In the event of system failure, Fleet will:
- investigate the problem to determine the root cause.
- identify affected users.
- escalate if necessary.
- understand and remediate the problem.
- notify impacted users of any steps they need to take (if any).  If a customer paid with a credit card and had a bad experience, default to refunding their money.
- Conduct an incident post-mortem to determine any additional steps we need (including monitoring) to take to prevent this class of problems from happening in the future.


### Check browser compatibility for fleetdm.com

A browser compatibility check of [fleetdm.com](https://fleetdm.com/) should be carried out monthly to verify that the website looks and functions as expected across all [supported browsers](https://fleetdm.com/docs/using-fleet/supported-browsers).

- We use [BrowserStack](https://www.browserstack.com/users/sign_in) (logins can be found in [1Password](https://start.1password.com/open/i?a=N3F7LHAKQ5G3JPFPX234EC4ZDQ&v=3ycqkai6naxhqsylmsos6vairu&i=nwnxrrbpcwkuzaazh3rywzoh6e&h=fleetdevicemanagement.1password.com)) for our cross-browser checks.
- Check for issues against the latest version of Google Chrome (macOS). We use this as our baseline for quality assurance.
- Document any issues in GitHub as a [bug](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=bug%2C%3Areproduce&template=bug-report.md&title=), and assign them for fixing.
- If in doubt about anything regarding design or layout, please reach out to the [Head of Design](https://fleetdm.com/handbook/digital-experience#team).


### Export an image for fleetdm.com

In Figma:
1. Select the layers you want to export.
2. Confirm export settings and naming convention:
  - Item name - color variant - (CSS)size - @2x.fileformat (e.g., `os-macos-black-16x16@2x.png`)
  - Note that the dimensions in the filename are in CSS pixels.  In this example, if you opened it in preview, the image would actually have dimensions of 32x32px but in the filename, and in HTML/CSS, we'll size it as if it were 16x16.  This is so that we support retina displays by default.
  - File extension might be .jpg or .png.
  - Avoid using SVGs or icon fonts.
3. Click the __Export__ button.


### Generate a new landing page

Experimental pages are short-lived, temporary landing pages intended for a small audience. All experiments and landing pages need to go through the standard [drafting process](https://fleetdm.com/handbook/company/product-groups#making-changes) before they are created.

Website experiments and landing pages live behind `/imagine` url. Which is hidden from the sitemap and intended to be linked to from ads and marketing campaigns. Design experiments (flyers, swag, etc.) should be limited to small audiences (less than 500 people) to avoid damaging the brand or confusing our customers. In general, experiments that are of a design nature should be targeted at prospects and random users, never targeted at our customers.

Some examples of experiments that would live behind the `/imagine` url:
- A flyer for a meetup "Free shirt to the person who can solve this riddle!"
- A landing page for a movie screening presented by Fleet
- A landing page for a private event
- A landing page for an ad campaign that is running for 4 weeks.
- An A/B test on product positioning
- A giveaway page for a conference
- Table-top signage for a conference booth or meetup

The Fleet website has a built-in landing page generator that can be used to quickly create a new page that lives under the /imagine/ url.

To generate a new page, you'll need: 

- A local copy of the [Fleet repo](https://github.com/fleetdm/fleet).
- [Node.js](https://nodejs.org/en/download/)
- (Optional) [Sails.js](https://sailsjs.com/) installed globally on your machine (`npm install sails -g`)

1. Open your terminal program, and navigate to the `website/` folder of your local copy of the Fleet repo.
    
    > Note: If this is your first time running the website locally, you will need to run `npm install` inside of the website/ folder to install the website's dependencies.

2. Call the `landing-page` generator by running `node ./node_modules/sails/bin/sails generate landing-page [page-name]`, replacing `[page-name]` with the kebab-cased name (words seperated by dashes `-`) of your page.

3. After the files have been generated, you'll need to manually update the website's routes. To do this, copy and paste the generated route for the new page to the "Imagine" section of `website/config/routes.js`.

4. Next you need to update the stylesheets so that the page can inherit the correct styles. To do this, copy and paste the generated import statement to the "Imagine" section of `website/assets/styles/importer.less`.

5. Start the website by running `node ./node_modules/sails/bin/sails lift` (or `sails lift` if you have Sails installed globally). The new landing page will be availible at `http://localhost:1337/imagine/{page-name}`.

6. Replace the lorum ipsum and placeholder images on the generated page with the page's real content, and add a meta description and title by changing the `pageTitleForMeta` and `pageDescriptionForMeta in the page's `locals` in `website/config/routes.js`.


### Restart Algolia manually

At least once every hour, an Algolia crawler reindexes the Fleet website's content. If an error occurs while the website is being indexed, Algolia will block our crawler and respond to requests with this message: `"This action cannot be executed on a blocked crawler"`.

When this happens, you'll need to manually start the crawler in the [Algolia crawler dashboard](https://crawler.algolia.com/admin/) to unblock it. 
You can do this by logging into the crawler dashboard using the login saved in 1password and clicking the "Restart crawling" button on our crawler's "overview" page](https://crawler.algolia.com/admin/crawlers/497dd4fd-f8dd-4ffb-85c9-2a56b7fafe98/overview).

No further action is needed if the crawler successfully reindexes the Fleet website. If another error occurs while the crawler is running, take a screenshot of the error and add it to the GitHub issue created for the alert and @mention `eashaw` for help.


### Re-run the "Deploy Fleet Website" action

If the action fails, please complete the following steps:
1. Head to the fleetdm-website app in the [Heroku dashboard](https://heroku.com) and select the "Activity" tab.
2. Select "Roll back to here" on the second to most recent deploy.
3. Head to the fleetdm/fleet GitHub repository and re-run the Deploy Fleet Website action. 


### Communicate Fleet's potential energy to stakeholders

On the first business day of every month, the Head of Digital Experience will send an update to the stakeholders of Fleet using the following steps:
1. Copy the following template into an outgoing email with the subject line: "[Investor update] Fleet, YYYY-MM".

```
Hi investors and friends,


FYI we just updated the self-service investor update portal with the numbers from last month:  https://docs.google.com/spreadsheets/d/10T7Q9iuHA4vpfV7qZCm6oMd5U1bLftBSobYD0RR8RkM/edit#gid=0


Thanks for your support,
Mike and the Fleet team

```

2. Address the email to the executive team's Gmail.
3. Using the [🌧️🦉 Investors + advisors](https://docs.google.com/spreadsheets/d/15knBE2-PrQ1Ad-QcIk0mxCN-xFsATKK9hcifqrm0qFQ/edit#gid=1068113636) spreadsheet, bcc the correct individuals and send the email.


### Grant role-specific license to a team member

Certain new team members, especially in go-to-market (GTM) roles, will need paid access to paid tools like Salesforce and LinkedIn Sales Navigator immediately on their first day with the company. Gong licenses that other departments need may [request them from Digital Experience](https://fleetdm.com/handbook/digital-experience#contact-us) and we will make sure there is no license redundancy in that department. The table below can be used to determine which paid licenses they will need, based on their role:

| Role                 | Salesforce CRM | Salesforce "Inbox" | LinkedIn _(paid)_ | Gong _(paid)_ | Zoom _(paid)_|
|:-----------------|:---|:---|:----|:---|:---|
| 🐋 AE            | ✅ | ✅ | ✅ | ✅ | ✅
| 🐋 CSM           | ✅ | ✅ | ❌ | ✅ | ✅
| 🐋 SC            | ✅ | ✅ | ❌ | ❌ | ✅
| ⚗️ PM            | ❌ | ❌ | ❌ | ✅ | ✅
| ⚗️ PD            | ❌ | ❌ | ❌ | ✅ | ✅
| 🔦 CEO           | ✅ | ✅ | ✅ | ✅ | ✅
|   Other roles    | ❌ | ❌ | ❌ | ❌ | ✅

> **Warning:** Do NOT buy LinkedIn Recruiter. AEs should use their personal Brex card to purchase the monthly [Core Sales Navigator](https://business.linkedin.com/sales-solutions/compare-plans) plan. Fleet does not use a company wide Sales Navigator account. The goal of Sales Navigator is to access to profile views and data, not InMail.  Fleet does not send InMail. 


### Process a tool upgrade request from a team member

- A Fleetie may request an upgraded license seat for Fleet tools by submitting an issue through ZenHub.
- Digital Experience will upgrade or add the license seat as needed and let the requesting team member know they did it.


### Downgrade an unused license seat

- On the first Wednesday of every quarter, the CEO, head of BizOps and Head of Digital experience will meet for 30 minutes to audit license seats in Figma, Slack, GitHub, Salesforce and other tools.
- During this meeting, as many seats will be downgraded as possible. When doubt exists, downgrade.
- Afterward, post in #random letting folks know that the quarterly tool reconciliation and seat clearing is complete, and that any members who lost access to anything they still need can submit a ZenHub issue to Digital Experience to have their access restored.
- The goal is to build deep, integrated knowledge of tool usage across Fleet and cut costs whenever possible. It will also force conversations on redundancies and decisions that aren't helping the business that otherwise might not be looked at a second time.  


### Add a seat to Salesforce

Here are the steps we take to grant appropriate Salesforce licenses to a new hire:
- Go to ["My Account"](https://fleetdm.lightning.force.com/lightning/n/standard-OnlineSalesHome).
- View contracts -> pick current contract.
- Add the desired number of licenses.
- Sign DocuSign sent to the email.
- The order will be processed in ~30m.
- Once the basic license has been added, you can create a new user using the new team member's `@fleetdm.com` email and assign a license to it.
- To also assign a user an "Inbox license", go to the ["Setup" page](https://fleetdm.lightning.force.com/lightning/setup/SetupOneHome/home) and select "User > Permission sets". Find the [inbox permission set](https://fleetdm.lightning.force.com/lightning/setup/PermSets/page?address=%2F005%3Fid%3D0PS4x000002uUn2%26isUserEntityOverride%3D1%26SetupNode%3DPermSets%26sfdcIFrameOrigin%3Dhttps%253A%252F%252Ffleetdm.lightning.force.com%26clc%3D1) and assign it to the new team member.


### Schedule press release

Fleet will occasionally release information to the press regarding upcoming initiatives before updating the functionality of the core product. This process sUse the following steps to schedule a press release:  

1. Add context for the next press release to the [e-group agenda](https://docs.google.com/document/d/13fjq3T0bZGOUah9cqHVxngckv0EB2R24A3gfl5cH7eo/edit) as a "DISCUSS:" to be reviewed by Fleet's executive team for alignment and finalization of date.
2. Once a release date is set, at-mention our public relations firm in the [#help-public-relations-firm--mindshare-pr--brand-marketing](https://fleetdm.slack.com/archives/C04PC9H34LF) and schedule a 30m call for our CEO and to communicate the press release.

> The above must be completed 6 weeks before the press release date. 

3. Schedule a 1.5h discussion between the [Head of Digital Experience](https://fleetdm.com/handbook/digital-experience#team) and the CEO to review the first draft linked as "Agenda: LINK" to the calendar event description.
4. Schedule a 60m call with the CEO and public relations firm to review the first draft linked as above to the calendar event (first draft provided by the PR firm)
5. Schedule 2.5 hrs of async time for the CEO work on edits and a 60m followup postgame (solo) where CEO edits and then settles+sends final release.


### Process the help-being-ceo Slack channel

The Apprentice will perform the following steps to process all communication from the CEO in the [help-being-ceo Slack channel](https://fleetdm.slack.com/archives/C03U703J0G5).
1. As soon as the message is received in the channel, add the "`:eyes:` (👀)" emoji to the Slack message to signify that you have seen and understood the task or question.
2. Start a Slack thread to add any context or let the stakeholders know the status of the task. 
3. After each task is completed, apply the "`:white_check_mark:`" (✅) to the slack message.


### Archive a document

Follow these steps to archive any document:
1. Create a copy of the document prefixed with the date using the format "`YYYY-MM-DD` Backup of `DOCUMENT_NAME`" (e.g. "2024-03-22 Backup of 🪂🗞️ Customer voice").
2. Be sure to "Share it with the same people", "Copy comments and suggestions", and "Include resolved comments and suggestions" as shown below.

<img width="455" alt="Screenshot 2024-03-23 at 12 14 00 PM" src="https://github.com/fleetdm/fleet/assets/108141731/1c773069-11a7-4ef4-ab43-8f7c626e4b10">

3. Save this backup copy to the same location in Google Drive where the original is found.
4. Link to the backup copy at the top of the original document. Be sure to use the full URL, no abbreviated pill links (e.g. "Notes from last time: URL_OF_MOST_RECENT_BACKUP_DOCUMENT").
5. Delete all non-structural content from the original document, including past meeting notes and current answers to "evergreen" questions.


### Schedule CEO interview

Use the following steps to schedule an interview between a candidate and the CEO:
1. Reach out to the candidate via email to find a time when the CEO and the canidate are both available.
   > This entire process takes an hour for the CEO. A 30 minute interview followed by a 30 minute "¶¶ Postgame". Be sure to offer times that accommodate for this.
3. [Make a copy of the "¶¶ CEO interview template"](https://docs.google.com/document/d/1yARlH6iZY-cP9cQbmL3z6TbMy-Ii7lO64RbuolpWQzI/copy) (private Google doc).
4. Move the document to the "[🕵️ ¶±¶ Reference checks & hiring data](https://drive.google.com/drive/folders/1VgKT6_VrQ9zYMnDOwJGE1mT1WrrMFqJw?usp=drive_link)" folder in Google Drive. 
5. Change file name and heading of doc to `¶¶ CANDIDATE_NAME (CANDIDATE_TITLE) <> Mike McNeil, CEO final interview (YYYY-MM-DD)`.
   - Add candidate's personal email in the "👥" (attendees) section at the top of the doc.
   - Add candidate's [LinkedIn url](https://www.linkedin.com/search/results/all/?keywords=people) on the first bullet for Mike.
6. Create a Google Calendar event at a time when the CEO and the canidate are both available.
   - Create a Google Calendar event matching the title of the interview doc.
   - Add the interveiw doc to the calendar event desription as the agenda (i.e. `Agenda: INTERVIEW_DOC_FULL_URL`) and save the calendar event.
7. Schedule a 30 minute "¶¶ Postgame" working session for the CEO to evaluate the candidate and give his recommendation.


### Program the CEO to do something

1. If necessary or if unsure, immediately direct message the CEO on Slack to clarify priority level, timing, and level of effort.  (For example, whether to schedule 30m or 60m to complete in full, or 30m planning as an iterative step.)
2. If there is not room on the calendar to schedule this soon enough with both Mike and Sam as needed (erring on the side of sooner), then either immediately direct message the CEO with a backup plan, or if it can obviously wait, then discuss at the next roundup.
3. Create a calendar event with a Zoom meeting for the CEO and Apprentice.  Keep the title short.  For the description, keep it very brief and use this template:

```
Agenda:
1. Apprentice: Is there enough context for you (CEO) to accomplish this?
2. Apprentice: Is this still a priority for you (CEO) to do.. right now?  Or should it be "someday/maybe"?
3. Apprentice: Is there enough time for you (CEO) to do this live? (Right now during this meeting?)
4. Apprentice: What are the next steps after you (CEO) complete this?
5. Apprentice: LINK_TO_DOC_OR_ISSUE
```

### Obtain a signature from the CEO

When an agreement is routed to the CEO for signature, the [Apprentice](https://fleetdm.com/handbook/digital-experience#team) is responsible for obtaining a signature from the CEO using the following steps:
1. Drag the email to the ["🔏 SAM: Signature wanted"](https://mail.google.com/mail/u/0/#label/SAM%3A+Signature+wanted) label making sure to mark the email as unread.
2. A Business Operations Engineer will at-mention the Apprentice in a legal review issue, letting them know the contract is good to go. After that, move the email to the "[✍️ MIKE: Ready to sign](https://mail.google.com/mail/u/0/#label/%E2%9C%8D%EF%B8%8F+MIKE%3A+Ready+to+sign)" label

> If the agreement closes a deal, inform the CEO (via Slack DM) that a subscription agreement is ready for his review/signature. The SLA for CEO review and signature is 48hrs.

3. Comment in the issue once the CEO has signed the agreement and assign the issue to [Nathan Holiday](https://fleetdm.com/handbook/business-operations#team).


### Prepare for CEO office minutes 

Before the start of the meeting, the Apprentice will prepare the "CEO office minutes" meeting [agenda](https://docs.google.com/document/d/12cd0N8KvHkfJxYlo7ggdisrvqw4MCErDoIzLjmBIdj4/edit) such that the following is true:
1. All agenda items are prefixed with a date of when the item will be covered and name of the person requesting to discuss the issue.
2. All team members with an agenda item have added themselves **and their manager** to the correct calendar event. If the team member or manager hasn't been added to the calendar event before the meeting begins, the agenda item is de-prioritized in favor of others with representatives in attendance. 
3. If there are more that two team members attending, the Apprentice will work with the team members to schedule additional time to cover the agenda.  

> If the manager is unable to attend the scheduled time of the meeting, the Apprentice will work with the team member to schedule an adhoc meeting between them, their manager, and the CEO.


### Process the CEO's calendar

Time management for the CEO is essential.  The Apprentice processes the CEO's calendar multiple times per day.

- **Clear any unexpected new events or double-bookings.** Look for any new double-bookings, invites that haven't been accepted, or other events you don't recognize.
  1. Double-book temporarily with a "UNCONFIRMED" calendar block so that the CEO ignores it and doesn't spend time trying to figure out what it is.
  2. Go to the organizer (or nearest fleetie who's not the CEO):
    - Get full context on what the CEO should know as to the purpose of the meeting and why the organizer thinks it is helpful or necessary for the CEO to attend.
    - Remind the organizer with [this link to the handbook that all CEO events have times chosen by Sam before booking](https://fleetdm.com/handbook/company/communications#schedule-time-with-the-ceo).
  3. Bring prepped discussion item about this proposed event to the next CEO roundup, including the purpose of the event and why it is helpful or necessary for the CEO to attend (according to the person requesting the CEO's attendance).  The CEO will decide whether to attend.
  4. Delete the "UNCONFIRMED" block if the meeting is confirmed, or otherwise work with the organizer to pick a new time or let them know the decision.

- **Prepare the agenda for any newly-added meetings**: [Meeting agenda prep](https://docs.google.com/document/d/1gH3IRRgptrqSYzBFy-77g98JROTL8wqrazJIMkp-Gb4/edit#heading=h.i7mkhr6m123r) is especially important to help the CEO focus and transition quickly in and between meetings. Using the CEO's browser, prepare each document by including the following:

> If a meeting agenda has to be created from scratch, be sure to move it to the "Meeting notes" folder in Google Drive so that he isn't locked out of any documents.
> If preparing for a meeting with a current advisor, use the existing journal as the meeting agenda using these steps:
> 1. Search for the journal in Mike's browser using the advisor's name or email.
> 2. Update the journal by adding the date of the meeting as an H3 in the Google document (pattern matching the document) and link the document to the calendar description.
 
  1. LinkedIn profile url of all outside participants. Connect with any of the attendees that the CEO is not already connected to on LinkedIn, this should always be a blank connect request meaning "Send without note". Nest everything from prep under the LinkedIn url (ie all under #1)
  2. A screen-shot of LinkedIn profile pic
  3. Company name (in doc title, file name and Google calendar event title)
  4. Correct date (20XX-XX-XX in doc title and file name)
  5. Context that helps the CEO to understand the purpose of the meeting at a glance from:
    - CEO's email
    - LinkedIn messages (careful not to mark things as read!)
    - Google Drive 
  6. Edit the calendar event description, changing “Notes” to “Agenda” when you're finished preparing the document to signify that this meeting has been prepped.


### Process the CEO's inbox

- The Apprentice is [responsible](https://fleetdm.com/handbook/company/why-this-way#why-direct-responsibility) for [processing all email traffic](https://docs.google.com/document/d/1gH3IRRgptrqSYzBFy-77g98JROTL8wqrazJIMkp-Gb4/edit#heading=h.i7mkhr6m123r) prior to CEO review to reduce the scope of Mike's inbox to only include necessary and actionable communication.
 -  Marking spam emails as read (same for emails Mike doesn't actually need to read).
 -  Escalate actionable sales communication and update Mike directly.
 -  Ensure all calendar invites have the necessary documents included.
 -  Forward any emails from customers about paying Fleet to the Buisness Operations department using [Fleet's billing email](https://docs.google.com/document/d/1tE-NpNfw1icmU2MjYuBRib0VWBPVAdmq4NiCrpuI0F0/edit#heading=h.wqalwz1je6rq).


### Document performance feedback

Every Friday at 5PM a [Business Operations team member](https://fleetdm.com/handbook/business-operations#team) will look for missing data in the [KPIs spreadsheet](https://docs.google.com/spreadsheets/d/1Hso0LxqwrRVINCyW_n436bNHmoqhoLhC8bcbvLPOs9A/edit#gid=0). 
1. If KPIs are not reported on time, the BizOps Engineer will notify the Apprentice to the CEO and the DRI.
2. The Apprentice will update the "performance management" section of the appropriate individual's 1:1 doc so that the CEO can address during the next 1:1 meeting with the DRI.


### Send the weekly update

We like to be open about milestones and announcements. Every Friday, e-group members [report their KPIs for the week](https://docs.google.com/spreadsheets/d/1Hso0LxqwrRVINCyW_n436bNHmoqhoLhC8bcbvLPOs9A/edit) by 5:00pm U.S. CT. Every Friday at 6PM, the Apprentice will post a short update in [#general](https://fleetdm.slack.com/archives/C019FNQPA23) including:
    - A link to view KPIs
    - Who was on-call that week
    - Fleeties who are currently onboarding
    - Planned hires who haven't started yet
    - Fleeties that departed that week
  
  To send the weekly update follow these steps:  
  
1. Navigate to the current weeks row in the [KPIs Google Sheet](https://docs.google.com/spreadsheets/d/1Hso0LxqwrRVINCyW_n436bNHmoqhoLhC8bcbvLPOs9A/edit#gid=0).
2. Copy the entire formula in this weeks "Weekly update" update cell and paste without formating (CMD+⇧+V) back into the same cell. The formula will now look like this:

<img width="464" alt="image" src="https://github.com/fleetdm/fleet/assets/108141731/1f7c652c-955e-4e84-b16f-83bc48af71f1">

3. In the "Weekly update" column (column E) for that row, double-click into that cell and change the "⚡️" to "🔭" in the beginning of the formula.  
4. Paste the newly formatted message in the [#general Slack channel](https://fleetdm.slack.com/archives/C019FNQPA23) and delete the double quotes around the message and any links that unfurl from links in the weekly update message.
5. Use the drop-down next to the send button and select "Custom time" and schedule the message to send "Today" at 18:00 or 6pm CT.


### Troubleshoot signature automation

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
4. Once the file is uploaded, Zapier uses the Slack integration to post in the #help-classified channel with the message:
   ```
   Now complete with all signatures:
      [email subject]
      link: drive.google.com/[destinationFolderID]
   ```


### Schedule travel for the CEO

The Apprentice schedules all travel arrangements for the CEO including flights, hotel, and reservations if needed. CEO traveling preferences in descending order of importance are:
  - Direct flight whenever possible  (as long as the cost of the direct flight is ≤2x the cost of a reasonable non-direct flight)
  - Select a non-middle seat, whenever possible
  - Don't upgrade seats (unless there's a cheap upgrade that gets a non-middle seat, or if a flight is longer than 5 hours.  Even then, never buy a seat upgrade that costs >$100.)
  - The CEO does not like to be called "Michael".  Unfortunately, this is necessary when booking flights.  (He has missed flights before by not doing this.)
  - Default to carry-on only, no checked bags.  (For trips longer than 5 nights, add 1 checked bag.)
  - Use the Brex card.
  - Frequent flyer details of all (previously flown) airlines are in 1Password as well as important travel documents.


### Prepare for the All hands

- **Every month** the Apprentice will do the prep work for the monthly "✌️ All hands 🖐👋🤲👏🙌🤘" call.
  -  In the ["👋 All hands" folder](https://drive.google.com/drive/folders/1cw_lL3_Xu9ZOXKGPghh8F4tc0ND9kQeY?usp=sharing), create a new folder using "yyyy-mm - All hands".
  - Update "End of the quarter" slides to reflect the current countdown.
  - Download a copy of the previous month's keynote file and rename the copy pattern matching existing files.
  - Update the slides to reflect the current "All hands" date (e.g. cover slides month and the "You are here" slide)'
  - Update slides that contain metrics to reflect current information using the [📈 KPIs](https://docs.google.com/spreadsheets/d/1Hso0LxqwrRVINCyW_n436bNHmoqhoLhC8bcbvLPOs9A/edit#gid=0) doc.
  - Update the "Spotlight slide" for guest speakers.
  - Add new customer logos from Mike's bookmarks ["Customers list"](https://fleetdm.lightning.force.com/lightning/o/Account/list?filterName=00B4x00000CTHP8EAP) and Google "Company name" to find the current logo.

- **First "All hands" of the quarter**
  - Audit the "Strategy" slide.
  - Audit the "Goals" slide

The day before the All hands, Mike will prepare slides that reflect the CEO vision and focus. 


#### Share recording of all hands meeting

The Apprentice will post a link to the All hands Gong recording and slide deck in Slack.

Template to use:

```
Thanks to everyone who contributed to today's "All hands" call.

:tv: If you weren't able to attend, please *[watch the recording](Current-link-to-Gong-recording)* _(1.5x playback supported)_.

You can also grab a copy of the [original slides](https://fleetdm.com/handbook/company/communications#all-hands) for use in your own confidential presentations.
```

1. Copy and paste the template to the "[# general](https://fleetdm.slack.com/archives/C019FNQPA23)" Slack channel.
2. Open [Gong recording](https://us-65885.app.gong.io/home?workspace-id=9148397688380544352&r=m) and click `Share call`, then click `Share with customers`, then `Copy link`.
3. Paste the url `*[Watch the recording](`here-in-your-template-message`)*`.

<img width="464" alt="image" src="https://github.com/Sampfluger88/fleet/assets/108141731/c2002cfa-a0f6-4349-bb06-71104f6cdce1">

4. Schedule the Slack message to go out at 6pm CT (18:00).


### Process and backup Sid agenda

Every two weeks, our CEO Mike has a meeting with Sid Sijbrandij. The CEO uses dedicated (blocked, recurring) time to prepare for this meeting earlier in the week.
1. 30 minutes After each meeting [archive the "💻 Sid : Mike(Fleet)" agenda](https://fleetdm.com/handbook/digital-experience#archive-a-document), moving it to the [(¶¶) Sid archive](https://drive.google.com/drive/folders/1izVfIBt2nr4APlkm36E6DJg1k1PDjmae) folder in Google Drive.
2. **In the backup copy**, create Google Doc comments assigning all Fleet TODOs to the correct DRI.   
3. In the ¶¶¶¶🦿🌪️CEO Roundup doc, update the URL in `Sam: FYI: Agenda from last time:` [LINK](link).


### Process and backup E-group agenda 

Follow these steps to process and backup the E-group agenda: 
1. [Archive the E-group agenda](https://fleetdm.com/handbook/digital-experience#archive-a-document) after each meeting, moving it to the ["¶¶ E-group archive"](https://drive.google.com/drive/u/0/folders/1IsSGMgbt4pDcP8gSnLj8Z8NGY7_6UTt6) folder in Google Drive.
2. **In the backup copy**, leave Google Doc comments assigning all TODOs to the correct DRI.  
3. If the "All hands" meeting has happened today remove any spotlights covered in the current "All hands" presentation.

### Check LinkedIn for new activity 

Once a day the Apprentice will check LinkedIn for unread messages and pending connect request. 

  1. Log into the CEO's [LinkedIn](https://www.linkedin.com/search/results/all/?sid=s2%3A) and bring up the messaging window.
  2. Filter out all read messages by clicking "filter" and then "Unread".
  3. Bring all unreads to the CEO during the daily roundup.
     
To check for pending connect requests, perform the following steps:
  1. Log into the CEO's LinkedIn (if you're not already) and click "My Network".
  2. Bring all pending connect requests to the CEO during the daily roundup.  


### Unroll a Slack thread

From time to time the CEO will ask the Apprentice to unroll a Slack thread into a well-named whiteboard Google doc for safekeeping and future searching. 
  1. Start with a new doc.
  2. Name the file with "yyyy-mm-dd - topic" (something empathetic and easy to find).
  3. Use CMD+SHFT+V to paste the Slack convo into the doc.
  4. Reapply formatting manually (be mindful of quotes, links, and images).
      - To copy images right-click+copy and then paste in the doc (some resizing may be necessary to fit the page).


### Delete an accidental meeting recording

It's not enough to just "delete" a recording of a meeting in Gong.  Instead, use these steps:

- Wait for at least 30 minutes after the meeting has ended to ensure the recording and transcript exist and can be deleted.
- [Sign in to Gong](https://us-65885.app.gong.io/deals?company-id=2676443513846037003&workspace-id=9148397688380544352&board-id=8761946992754097113&view-mode=DEALS&tab-idx=0&account-activity=true&owner-ids=&owner-team-ids=5778354842532790437&timespan-id=34&sort-by=DealActivity&sort-field=%7B%22type%22%3A%22RegularField%22%2C%22name%22%3A%22DealActivity%22%7D&sort-order=DESC&owner-id=5778354842532790437&include-team=true) through the CEO's browser.
- Scroll down to `Conversations`
- Select the call recording no longer needed
- Click the "hotdog" menu in the right-hand corner
<img width="264" alt="image" src="https://github.com/fleetdm/fleet/assets/108141731/86948d02-a972-42ef-9a2d-1d93f24a1780">
- `Delete recording`
- Search for the title of the meeting Google Drive and delete the auto-generated Google Doc containing the transcript. 
- Always check back to ensure the recording **and** transcript were both deleted.

### Update a company brand front

Fleet has several brand fronts that need to be updated from time to time. Check each [brand front](https://docs.google.com/spreadsheets/d/1c15vwMZytpCLHUdGvXxi0d6WGgPcQU1UBMniC1F9oKk/edit?gid=0#gid=0) for consistency and update as needed with the following: 
- The current pitch, found in the blurbs section of the [🎐 Why Fleet?](https://docs.google.com/document/d/1E0VU4AcB6UTVRd4JKD45Saxh9Gz-mkO3LnGSTBDLEZo/edit#heading=h.uovxedjegxdc) doc. 
- The current [brand imagery](https://www.figma.com/design/1J2yxqH8Q7u8V7YTtA1iej/Social-media-(logos%2C-covers%2C-banners)?node-id=3962-65895). Check this [Loom video](https://www.loom.com/share/4432646cc9614046aaa4a74da1c0adb5?sid=2f84779f-f0bd-4055-be69-282c5a16f5c5) for more info.


## Rituals

- Note: Some rituals (⏰) are especially time-sensitive and require attention multiple times (3+) per day.  Set reminders for the following times (CT):
  - 9:30 AM _(/before first meeting)_
  - 12:30 PM CT _(/beginning of "reserved block")_
  - 6:30 PM CT _(/after last meeting, before roundup / Japan calls)_

<rituals :rituals="rituals['handbook/digital-experience/digital-experience.rituals.yml']"></rituals>


#### Stubs
The following stubs are included only to make links backward compatible.

##### Why not mention the CEO in Slack threads?
Please see [handbook/company/why-this-way/why-not-mention-the-ceo-in-slack-threads](https://www.fleetdm.com/handbook/company/why-this-way#why-not-mention-the-ceo-in-slack-threads)


<meta name="maintainedBy" value="Sampfluger88">
<meta name="title" value="🌐 Digital Experience">

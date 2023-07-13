# Website handbook

This page details processes related to maintaining and updating the Fleet website ([fleetdm.com](https://fleetdm.com)).

Website-related topics that are NOT included on this page:

- [Documentation](https://fleetdm.com/handbook/marketing#documentation)
- [Publishing an article](./how-to-submit-and-publish-an-article)
- [Markdown guide](./markdown-guide)

## Responsibilities

The [website group](https://fleetdm.com/handbook/company/product-groups#website-group) is responsible for production and maintenance of the Fleet website.

## Website roadmap

View planned changes to the website on the website group's [sprint board](https://app.zenhub.com/workspaces/g-website-6451748b4eb15200131d4bab/board?sprints=none).

## Requesting changes

See Marketing [intake](https://fleetdm.com/handbook/marketing#intake) for more information on how the website team prioritizes new requests. Bugs are always prioritized first.

## Wireframes

Before committing anything to code, we create wireframes (referred to as ["drafting"](https://fleetdm.com/handbook/company/development-groups#making-changes)) to illustrate all changes that affect the layout and structure of the user interface, design, or APIs of fleetdm.com.

See [Why do we use a wireframe first approach](https://fleetdm.com/handbook/company/why-this-way#why-do-we-use-a-wireframe-first-approach) for more information. 

## Design reviews

We hold regular design review sessions to evaluate, revise, and approve wireframes before moving into production.

Design review sessions are hosted by [Mike Thomas](https://calendar.google.com/calendar/u/0?cid=bXRob21hc0BmbGVldGRtLmNvbQ) and typically take place daily, late afternoon (CST). Anyone is welcome to join.

## Estimation sessions

We use estimation sessions to estimate the effort required to complete a prioritized task. 

Through these sessions, we can:

- Confirm that wireframes are complete before moving to production.
- Consider all edge cases and requirements that may have been with during wireframing.
- Avoid having the engineer make choices for “unknowns” during production.
- More accurately plan and prioritize upcoming tasks.

### Story points

Story points represent the effort required to complete a task. After accessing wireframes, we typically play planning poker, a gamified estimation technique, to determine the necessary story point value.

We use the following story points to estimate website tasks:

| Story point | Time |
|:---|:---|
| 1 | 1 to 2 hours |
| 2 | 2 to 4 hours |
| 3 | 1 day |
| 5 | 1 to 2 days |
| 8 | Up to a week |
| 13 | 1 to 2 weeks |

## Quality

Quality assurance (QA) checks must be completed before changes to the website can be merged. Read on to learn about the quality assurance process for the website.

> **Important:** A PR to the website should not be merged until the quality assurance process has been successfully completed.

### Manual QA

Before estimating changes to the website, the product manager of the website group is responsible for making sure that manual QA steps have been added to requests.

#### Writing QA steps

QA steps are step-by-step instructions used to confirm that changed to the website function as expected. They should be simple and clear enough for anybody to follow. Example steps are included in [the “Website request” issue template](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=%23g-website&template=website-request.md&title=Request%3A+__________________________).

#### Actioning QA steps

[View the website locally](#test-changes-to-the-website) and follow the QA steps in the request ticket to test changes.

QA steps should be actioned when a request has been moved into the “Review/QA” column of the website product board. PRs to the website should not be merged until QA has been completed.

A successful QA check can be indicated by leaving a comment in the conversation thread of the PR. 

### Additional QA

In addition to the steps above. All website changes must be thoroughly checked at all breakpoints and a [browser compatibility](#browser-compatibility) test should be carried out on [supported browsers](https://fleetdm.com/docs/using-fleet/supported-browsers) before website changes can go live.

## Testing changes

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
    
    > You can use the `--skipGithubRequests` flag to skip requests made to GitHub if you get rate-limited by GitHub’s API while running this script. 
    > 
    > e.g., `node ./node_modules/sails/bin/sails run build-static-content --skipGithubRequests`

3. Once the script is complete, start the website server. From the `website/` folder:
  - **With Node.js:** start the server by running `node ./node_modules/sails/bin/sails lift`
  - **With Sails.js installed globally:** start the server by running `sails lift`.
4. When the server has started, the Fleet website will be availible at [http://localhost:2024](http://localhost:2024)
    
  > **Note:** Some features, such as Fleet Sandbox, Self-service license dispenser, and account creation are not availible when running the website locally. If you need help testing features on a local copy, reach out to `@eashaw` in the [#g-website](https://fleetdm.slack.com/archives/C058S8PFSK0) channel on Slack..

## Merging changes
When merging a PR to the master branch of the [Fleet repo](https://github.com/fleetdm/fleet), remember that whatever you merge gets deployed live immediately. Ensure that the appropriate quality checks have been completed before merging. [Learn about the website QA process](#quality).

When merging changes to the [docs](https://fleetdm.com/docs), [handbook](https://fleetdm.com/handbook), and articles, make sure that the PR’s changes do not contain inappropriate content (goes without saying) or confidential information, and that the content represents our [brand](#brand) accordingly. When in doubt reach out to the product manager of the [website group](https://fleetdm.com/handbook/company/product-groups#website-group) in the [#g-website](https://fleetdm.slack.com/archives/C058S8PFSK0) channel on Slack.

### The "Deploy Fleet Website" GitHub action failed
If the action fails, please complete the following steps:
1. Head to the fleetdm-website app in the [Heroku dashboard](https://heroku.com) and select the "Activity" tab.
2. Select "Roll back to here" on the second to most recent deploy.
3. Head to the fleetdm/fleet GitHub repository and re-run the Deploy Fleet Website action. 

## Browser compatibility

A browser compatibility check of [fleetdm.com](https://fleetdm.com/) should be carried out monthly to verify that the website looks and functions as expected across all [supported browsers](../../docs/Using-Fleet/Supported-browsers.md).

- We use [BrowserStack](https://www.browserstack.com/users/sign_in) (logins can be found in [1Password](https://start.1password.com/open/i?a=N3F7LHAKQ5G3JPFPX234EC4ZDQ&v=3ycqkai6naxhqsylmsos6vairu&i=nwnxrrbpcwkuzaazh3rywzoh6e&h=fleetdevicemanagement.1password.com)) for our cross-browser checks.
- Check for issues against the latest version of Google Chrome (macOS). We use this as our baseline for quality assurance.
- Document any issues in GitHub as a [bug report](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=bug%2C%3Areproduce&template=bug-report.md&title=), and assign them for fixing.
- If in doubt about anything regarding design or layout, please reach out to the Design team.

## Error handling

### Responding to a 5xx error on fleetdm.com

Production systems can fail for various reasons, and it can be frustrating to users when they do, and customer experience is significant to Fleet. In the event of system failure, Fleet will:
* investigate the problem to determine the root cause.
* identify affected users.
* escalate if necessary.
* understand and remediate the problem.
* notify impacted users of any steps they need to take (if any).  If a customer paid with a credit card and had a bad experience, default to refunding their money.
* Conduct an incident post-mortem to determine any additional steps we need (including monitoring) to take to prevent this class of problems from happening in the future.

### Incident post-mortems

When conducting an incident post-mortem, answer the following three questions:

1. Impact: What impact did this error have? How many humans experienced this error, if any, and who were they?
2. Root Cause: Why did this error happen?
3. Side effects: did this error have any side effects? e.g., did it corrupt any data? Did code that was supposed to run afterward and “finish something up” not run, and did it leave anything in the database or other systems in a broken state requiring repair? This typically involves checking the line in the source code that threw the error. 

## Vulnerability monitoring

Every week, we run `npm audit --only=prod` to check for vulnerabilities on the production dependencies of fleetdm.com. Once we have a solution to configure GitHub's Dependabot to ignore devDependencies, this manual process can be replaced with Dependabot.

## Experimentation

In order for marketing to iterate rapidly we have created a process of experimentation. This will allow a small group of marketers to draft, review and publish a page or a flyer or an experiment without getting a series of approvals. Experiments should be short-lived, temporary things intended for a small audience. When an experiment succeeds it should immediately be turned into a part of Fleet'd rituals and then go through the proper wireframe-first approach. 

Website experiments and landing pages should live behind `/imagine` url. Which is hidden from the sitemap and intended to be linked to from ads and marketing campaigns. Design experiments (flyers, swag, etc.) should be limited to small audiences (less than 500 people) to avoid damaging the brand or confusing our customers. In general, experiments that are of a design nature should be targeted at prospects and random users, never targeted at our customers.

Some examples of experiments that would qualify to get a rapid approach:
- A flyer for a meetup "Free shirt to the person who can solve this riddle!"
- A landing page for a movie screening presented by Fleet
- A landing page for a private event
- A landing page for an ad campaign that is running for 4 weeks.
- An A/B test on product positioning
- A giveaway page for a conference
- Table-top signage for a conference booth or meetup

### Landing pages

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


## How to export images for the website
In Figma:
1. Select the layers you want to export.
2. Confirm export settings and naming convention:
  * Item name - color variant - (CSS)size - @2x.fileformat (e.g., `os-macos-black-16x16@2x.png`)
  * Note that the dimensions in the filename are in CSS pixels.  In this example, if you opened it in preview, the image would actually have dimensions of 32x32px but in the filename, and in HTML/CSS, we'll size it as if it were 16x16.  This is so that we support retina displays by default.
  * File extension might be .jpg or .png.
  * Avoid using SVGs or icon fonts.
3. Click the __Export__ button.

## Website services

### Cloudflare

We use Cloudflare to manage the DNS records of fleetdm.com and our other domains. Cloudflare offers a free, user-friendly, and cloud-agnostic interface that allows authorized team members to manage all our domains easily.
If you need access to Fleet's Cloudflare account, please ask the [DRI](https://fleetdm.com/handbook/company/why-this-way#why-direct-responsibility) Zach Wasserman in Slack for an invitation.


To make DNS changes in Cloudflare:
1. Log into your Cloudflare account and select the "Fleet" account.
2. Select the domain you want to change and go to the DNS panel on that domain's dashboard.
3. To add a record, click the "Add record" button, select the record's type, fill in the required values, and click "Save". If you're making changes to an existing record, you only need to click on the record, update the record's values, and save your changes.

### Heroku

TODO: Document.

### Algolia

TODO: Document.

## Rituals
| Ritual                       | Frequency                | Description                                         | DRI               |
|:-----------------------------|:-------------------------|:----------------------------------------------------|-------------------|
| Generate latest schema | once every 3 weeks | After each sprint, generate the latest tables json file to incorporate any new schema documentation. | Eric Shaw |


<meta name="maintainedBy" value="mike-j-thomas">
<meta name="title" value="Website handbook">

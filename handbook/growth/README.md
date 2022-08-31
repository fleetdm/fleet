# Growth

As an open-core company, Fleet endeavors to build a community of engaged users, customers, and
contributors. The purpose of the growth team is to own and improve the growth funnel to drive awareness, adoption, and referrals of Fleet while honoring the ideals and voice of the open source community and our company values.

## Positioning

Effective market positioning is crucial to the growth of any software product. Fleet needs to maintain a unique, valuable position in the minds of our users. We keep assertions on our positioning in this [Google Doc](https://docs.google.com/document/d/177Q4_2FY5Vm7Nd3ne32vOKTQqYfDG0p_ouklvl3PnWc/edit) (private). We will update it quarterly based on the feedback of users, customers, team members, and other stakeholders. Feedback can be provided as a comment in the document or by posting in the `#g-growth` Slack channel. 

## Marketing Qualified Opportunities (MQOs)

Growth's goal is to increase product usage. We value users of all sizes adopting Fleet Free or Fleet Premium. Companies purchasing under 100 device licenses should sign up for [self-service](https://fleetdm.com/pricing/). Companies that enroll more than 100 devices should [schedule a demo](https://fleetdm.com/). When these companies attend a demo, Fleet considers them Marketing Qualified Opportunities (MQOs).

## Lead enrichment

Fleet's lead enrichment process can be found in this [Google Doc](https://docs.google.com/document/d/1zOv39O989bPRNTIcLNNE4ESUI5Ry2XII3XuRpJqNN7g/edit?usp=sharing) (private).

## Posting on social media as Fleet

Posting to social media should follow a [personable tone](https://fleetdm.com/handbook/digital-experience#communicating-as-fleet) and strive to deliver useful information across our social accounts.

### Topics:

- Fleet the product
- Internal progress
- Highlighting [community contributions](https://fleetdm.com/handbook/community#community-contributions-pull-requests)
- Highlighting Fleet and osquery accomplishments
- Industry news about osquery
- Industry news about device management
- Upcoming events, interviews, and podcasts

### Guidelines:

In keeping with our tone, use hashtags in line and only when it feels natural. If it feels forced, don’t include any.

Self-promotional tweets are not ideal(Same goes for, to varying degrees, Reddit, HN, Quora, StackOverflow, LinkedIn, Slack, and almost anywhere else).  Also, see [The Impact Equation](https://www.audible.com/pd/The-Impact-Equation-Audiobook/B00AR1VFBU) by Chris Brogan and Julien Smith.

Great brands are [magnanimous](https://en.wikipedia.org/wiki/Magnanimity).

### Scheduling:

Once a post is drafted, deliver it to our three main platforms.

- [Twitter](https://twitter.com/fleetctl)
- [LinkedIn](https://www.linkedin.com/company/fleetdm/)
- [Facebook](https://www.facebook.com/fleetdm)

Log in to [Sprout Social](https://app.sproutsocial.com/publishing/) and use the compose tool to deliver the post to each platform. (credentials in 1Password).


## Promoting blog posts on social media

Once a blog post has been written, approved, and published, make sure that it is promoted on social media. Please refer to our [Publishing as Fleet](https://docs.google.com/document/d/1cmyVgUAqAWKZj1e_Sgt6eY-nNySAYHH3qoEnhQusph0/edit?usp=sharing) guide for more detailed information. 


## Press releases

If we are doing a press release, we are probably pitching it to one or more reporters as an exclusive story if they choose to take it.  Consider not sharing or publicizing any information related to the upcoming press release before the announcement.  Also, see [What is a press exclusive, and how does it work](https://www.quora.com/What-is-a-press-exclusive-and-how-does-it-work) on Quora.

### Press release boilerplate

Fleet gives teams fast, reliable access to data about the production servers, employee laptops, and other devices they manage - no matter the operating system. Users can search for any device data using SQL queries, making it faster to respond to incidents and automate IT. Fleet is also used to monitor vulnerabilities, battery health, and software. It can even monitor endpoint detection and response and mobile device management tools like Crowdstrike, Munki, Jamf, and Carbon Black, to help confirm that those platforms are working how administrators think they are. Fleet is open source software. It's easy to deploy and get started quickly, and it even comes with an enterprise-friendly free tier available under the MIT license.

IT and security teams love Fleet because of its flexibility and conventions. Instead of secretly collecting as much data as possible, Fleet defaults to privacy and transparency, capturing only the data your organization needs to meet its compliance, security, and management goals, with clearly-defined, flexible limits.   

That means better privacy, better device performance, and better data but with less noise.

## Sponsoring events

When reaching out for sponsorships, Fleet's goal is to expose potential hires, contributors, and users to Fleet and osquery.
Track prospective sponsorships in our [partnerships and outreach Google Sheet:](https://docs.google.com/spreadsheets/d/107AwHKqFjt7TWItnf8pFknSwwxb_gsp6awB66t7YE_w/edit#gid=2108184225)

Once a relevant sponsorship opportunity and its prospectus are reviewed:
1. Create a new [GitHub issue](https://github.com/fleetdm/fleet/issues/new).
 
2. Detail the important information of the event, such as date, name of the event, location, and page links to the relevant prospectus. 
 
3. Add the issue to the “Conferences/speaking” column of the [Growth plan project](https://github.com/orgs/fleetdm/projects/21).
 
4. Schedule a meeting with the representatives at the event to discuss pricing and sponsorship tiers.
 
5. Invoices should be received at billing@fleetdm.com and sent to Eric Shaw for approval.
 
6. Eric Shaw (Business Operations) will route the signatures required over to Mike McNeil (CEO) with DocuSign.
 
7. Once you complete the above steps, use the [Speaking events issue template](https://github.com/fleetdm/confidential/issues/new?assignees=mike-j-thomas&labels=&template=6-speaking-event.md&title=Speaking+event) to prepare speakers and participants for the event.

## Newsletter emails

The content for our newsletter emails comes from our articles. Because our HTML emails require that the styles are added inline, we generate HTML emails by using a script and manually QA them before sending them out to subscribers.

### Generating emails for the Fleet newsletter

To convert a Markdown article into an email for the newsletter, you'll need the following:

- A local copy of the [Fleet repo](https://github.com/fleetdm/fleet).
- [Node.js](https://nodejs.org/en/download/)
- (Optional) [Sails.js](https://sailsjs.com) installed globally on your machine (`npm install sails -g`)

Once you have the above follow these steps:

1. Open your terminal program, and navigate to the `website/` folder of the [Fleet repo](https://github.com/fleetdm/fleet).

>Note: If this is your first time running this script, you will need to run `npm install` inside of the `website/` folder to install the website's dependencies.

2. Run the `build-html-email` script and pass in the filename of the Markdown article you would like to convert with the `--articleFilename` flag.
	
	- **With Node**, you will need to use `node ./node_modules/sails/bin/sails run build-html-email` to execute the script. e.g., `node ./node_modules/sails/bin/sails run build-html-email --articleFilename="fleet-4.19.0.md"`
	- **With Sails.js installed globally** you can use `sails run build-html-email` to execute the script. e.g., `sails run build-html-email --articleFilename="fleet-4.19.0.md"`

> Note: Only Markdown (`.md`) files are supported by the build-html-email script. The file extension is optional when providing the articleFilename.

4. Once the script is complete, a new email partial will be added to the `website/views/emails/newsletter-partials/` folder.

> Note: If an email partial has already been created from the specified Markdown article, the old version will be overwritten by the new file.

5. Start the website server locally to preview the generated email. To test the changes locally, open a terminal window in the `website/` folder of the Fleet repo and run the following command:
	
	- **With Node.js:** start the server by running `node ./node_modules/sails/bin/sails lift`.
	- **With Sails.js installed globally:** start the server by running `sails lift`.

6. With the server lifted, navigate to http://localhost:2024/admin/email-preview and login with the test admin user credentials (email:`admin@example.com` pw: `abc123`). 

	Click on the generated email in the list of emails generated from Markdown content and navigate to the preview page. On this page, you can view the see how the email will look on a variety of screen sizes.

	When you've made sure the content of the email looks good at all screen sizes, commit the new email partial to a new branch and open a pull request to add the file. You can request a review from a member of the digital experience team.

**Things to keep in mind when generating newsletter emails:**

- The emails will be generated using the Markdown file locally, any changes present to the local Markdown file will be reflected in the generated email.
- HTML elements in the Markdown file can cause rendering issues when previewing the generated email. If you see a "Script error" overlay while trying to preview an email, reach out to the [Digital Experience] team for help.
- The filename of the generated email will have the article category added and any periods will be changed to dashes. e.g., The generated email partial for `fleet-4.19.0.md` would be `releases--fleet-4-19-0.ejs`

## Rituals

The following table lists the Growth group's rituals, frequency, and Directly Responsible Individual (DRI).


| Ritual                       | Frequency                | Description                                         | DRI               |
|:-----------------------------|:-----------------------------|:----------------------------------------------------|-------------------|
| Daily tweet         | Daily | Post Fleet content on Twitter.     | Drew Baker        |
| Daily LinkedIn post        | Daily | Post Fleet content to LinkedIn.   | Drew Baker        |
| Check Twitter messages | Daily | Check and reply to messages on the Fleet Twitter account. Disregard requests unrelated to Fleet. | Drew Baker | 
| Social engagement     | Weekly | Participate in 50 social media engagements per week.| Drew Baker        |  
| Osquery jobs          | Weekly | Post to @osqueryjobs twice a week.            | Drew Baker        |
| Enrich Salesforce leads       | Weekly | Follow the Salesforce lead enrichment process every Friday.    | Drew Baker        |
| Outside contributions | Weekly | Check pull requests for outside contributions every Monday. | Drew Baker|
| Weekly article       | Weekly | Publish an article and promote it on social media. | Drew Baker|
| Missed demo follow up | Weekly | Email all leads who missed a scheduled demo | Andrew Bare |
| Weekly ins and outs   | Weekly | Track Growth team ins and outs.        | Tim Kern          |
| Podcast outreach      | Weekly | Conduct podcast outreach twice a week.     | Drew Baker        |
| Weekly update      | Weekly | Update the Growth KPIs in the ["🌈 Weekly updates" spreadsheet](https://docs.google.com/spreadsheets/d/1Hso0LxqwrRVINCyW_n436bNHmoqhoLhC8bcbvLPOs9A/edit#gid=0). | Drew Baker        |
| Update the "Release" field on the #g-growth board   | Every 3 weeks | <ul><li>Go to the [Growth board](https://github.com/orgs/fleetdm/projects/38/settings/fields/2654827)</li><li>add a 3-week iteration with the correct release number</li></ul> | Tim Kern        |
| Monthly conference checks    | Monthly | Check for conference openings and sponsorship opportunities on the 1st of every month. | Drew Baker|
| Freshen up pinned posts | Quarterly | Swap out or remove pinned posts on the brand Twitter account and LinkedIn company page. | Drew Baker | 


## Slack channels

These groups maintain the following [Slack channels](https://fleetdm.com/handbook/company#group-slack-channels):

| Slack channel               | [DRI](https://fleetdm.com/handbook/company#group-slack-channels)    |
|:----------------------------|:--------------------------------------------------------------------|
| `#g-growth`                 | Tim Kern                                                            |
| `#help-public-relations`    | Tim Kern                                                            |
| `#help-promote`             | Tim Kern                                                            |
| `#help-swag`                | Drew Baker                                                          |


<meta name="maintainedBy" value="timmy-k">
<meta name="title" value="🪴 Growth">


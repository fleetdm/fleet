# Marketing

<!-- TODO: short preamble -->

## Community
[https://fleetdm.com/handbook/marketing/community](https://fleetdm.com/handbook/marketing/community)

# Community

As an open-core company, Fleet endeavors to build a community of engaged users, customers, and
contributors.

## Communities

Fleet's users and broader audience are spread across many online platforms. Here are the most active communities where Fleet's developer relations and social media team members participate at least once every weekday:

- [Osquery Slack](https://join.slack.com/t/osquery/shared_invite/zt-h29zm0gk-s2DBtGUTW4CFel0f0IjTEw) (`#fleet` channel)
- [MacAdmins Slack](https://www.macadmins.org/) (`#fleet` channel)
- [osquery discussions on LinkedIn](https://www.linkedin.com/search/results/all/?keywords=osquery)
- [osquery discussions on Twitter](https://twitter.com/search?q=osquery&src=typed_query)
- [reddit.com/r/sysadmins](https://www.reddit.com/r/sysadmin/)
- [reddit.com/r/SysAdminBlogs](https://www.reddit.com/r/SysAdminBlogs/)
- [r/sysadmin Discord](https://discord.gg/sysadmin)

### Goals

Our primary quality objectives are _customer service_ and _defect reduction_. We try to optimize the following:

- Customer response time
- The number of bugs resolved per release cycle
- Staying abreast of what our community wants and the problems they're having
- Making people feel heard and understood
- Celebrating contributions
- Triaging bugs, identifying community feature requests, community pull requests, and community questions

### How?

- Folks who post a new comment in Slack or issue on GitHub should receive a response **within one business day**. The response doesn't need to include an immediate answer.
- If you feel confused by a question or comment raised, [request more details](#requesting-more-details) to better your understanding of the next steps.
- If an appropriate response is outside of your scope, please post to `#help-engineering` (in the Fleet Slack)), tagging `@oncall`.
- If things get heated, remember to stay [positive and helpful](https://canny.io/blog/moderating-user-comments-diplomatically/). If you aren't sure how best to respond positively, or if you see behavior that violates the Fleet code of conduct, get help.

### Requesting more details

Typically, the _questions_, _bug reports_, and _feature requests_ raised by community members will be missing helpful context, recreation steps, or motivations.

❓ For questions that you don't immediately know the answer to, it's helpful to ask follow-up questions to receive additional context.

- Let's say a community member asks, "How do I do X in Fleet?" A follow-up question could be, "What are you attempting to accomplish by doing X?"
- This way, you have additional details when you bring this to the Roundup meeting. In addition, the community member receives a response and feels heard.

🦟 For bug reports, it's helpful to ask for re-creation steps so you're later able to verify the bug exists.

- Let's say a community member submits a bug report. An example follow-up question could be, "Can you please walk me through how you encountered this issue so that I can attempt to recreate it?"
- This way, you now have steps that verify whether the bug exists in Fleet or if the issue is specific to the community member's environment. If the latter, you now have additional information for further investigation and question-asking.

💡 For feature requests, it's helpful to ask follow-up questions to understand better the "Why?" or underlying motivation behind the request.

- Let's say a community member submits the feature request "I want the ability to do X in Fleet." A follow-up question could be "If you were able to do X in Fleet, what's the next action you would take?" or "Why do you want to do X in Fleet?."
- Both of these questions provide helpful context on the underlying motivation behind the feature request when brought to the Roundup meeting. In addition, the community member receives a response and feels heard.

### Closing issues

It is often good to let the original poster (OP) close their issue themselves since they are usually well equipped to decide to mark the issue as resolved. In some cases, circling back with the OP can be impractical, and for the sake of speed, issues might get closed.

Keep in mind that this can feel jarring to the OP. The effect is worse if issues are closed automatically by a bot (See [balderashy/sails#3423](https://github.com/balderdashy/sails/issues/3423#issuecomment-169751072) and [balderdashy/sails#4057](https://github.com/balderdashy/sails/issues/4057) for examples of this).

### Version support

To provide the most accurate and efficient support, Fleet will only target fixes based on the latest released version. In the current version fixes, Fleet will not backport to older releases.

Community version supported for bug fixes: **Latest version only**

Community support for support/troubleshooting: **Current major version**

Premium version supported for bug fixes: **Latest version only**

Premium support for support/troubleshooting: **All versions**

### Tools

Find the script in `scripts/oncall` for use during oncall rotation (only been tested on macOS and Linux).
Its use is optional but contains several useful commands for checking issues and PRs that may require attention.
You will need to install the following tools to use it:

- [GitHub CLI](https://cli.github.com/manual/installation)
- [JQ](https://stedolan.github.io/jq/download/)

### Resources

There are several locations in Fleet's public and internal documentation that can be helpful when answering questions raised by the community:

1. Find the frequently asked question (FAQ) documents in each section in the `/docs` folder. These documents are the [Using Fleet FAQ](./../../docs/Using-Fleet/FAQ.md), [Deploying FAQ](./../../docs/Deploying/FAQ.md), and [Contributing FAQ](./../../docs/Contributing/FAQ.md).

2. Use the [internal FAQ](https://docs.google.com/document/d/1I6pJ3vz0EE-qE13VmpE2G3gd5zA1m3bb_u8Q2G3Gmp0/edit#heading=h.ltavvjy511qv) document.

### Assistance from engineering

Community team members can reach the engineering oncall for assistance by writing a message with `@oncall` in the `#help-engineering` channel of the Fleet Slack.

## Pull requests

The most important thing when community members contribute to Fleet is to show them we value their time and effort. We need to get eyes on community pull requests quickly (within one business day) and get them merged or give feedback as soon as we can.

### Process for managing community contributions

The Community Engagement DRI is responsible for keeping an eye out for new community contributions, getting them merged if possible, and getting the right eyes on them if they require a review.

Each business day, the Community Engagement DRI will check open pull requests to

1. check for new pull requests (PRs) from the Fleet community.
2. approve and merge any community PRs that are ready to go.
3. make sure there aren't any existing community PRs waiting for a follow-up from Fleet.

#### Identify community contributions

When a new pull request is submitted by a community contributor (someone not a member of the Fleet organization):

- Add the `:community` label.
- Self-assign for review.
- Check whether the PR can be merged or needs to be reviewed by the Product team.

  - Things that generally don't need additional review:

    - Minor changes to the docs.
    - Small bug fixes.

    - Additions or fixes to the Standard Query Library (as long as the SQL works properly and is attributed correctly).

  - If a review is needed:
    - Request a review from the [Product DRI](../people/README.md#directly-responsible-individuals). They should approve extensive changes and new features. Ask in the #g-product channel in Fleet's Slack for more information.
    - Tag the DRI and the contributor in a comment on the PR, letting everyone know why an additional review is needed. Make sure to say thanks!
    - Find any related open issues and make a note in the comments.

> Please refer to our [PRs from the community](https://docs.google.com/document/d/13r0vEhs9LOBdxWQWdZ8n5Ff9cyB3hQkTjI5OhcrHjVo/edit?usp=sharing) guide and use your best judgment.

#### Communicate with contributors

Community contributions are fantastic, and it's important that the contributor knows how much they are appreciated. The best way to do that is to keep in touch while we're working on getting their PR approved.

While each team member is responsible for monitoring their active issues and pull requests, the Community Engagement DRI will check in on pull requests with the `:community ` label daily to make sure everything is moving along. If there's a comment or question from the contributor that hasn't been addressed, reach out on Slack to get more information and update the contributor.

#### Merge Community PRs

When merging a pull request from a community contributor:

- Ensure that the checklist for the submitter is complete.
- Verify that all necessary reviews have been approved.
- Merge the PR.
- Thank and congratulate the contributor.
- Share the merged PR with the team in the #help-promote channel of Fleet Slack to be publicized on social media. Those who contribute to Fleet and are recognized for their contributions often become great champions for the project.

### Reviewing PRs from the community

If you're assigned a community pull request for review, it is important to keep things moving for the contributor. The goal is to not go more than one business day without following up with the contributor.

A PR should be merged if:

- It's a change that is needed and useful.
- The CI is passing.
- Tests are in place.
- Documentation is updated.
- Changes file is created.

For PRs that aren't ready to merge:

- Thank the contributor for their hard work and explain why we can't merge the changes yet.
- Encourage the contributor to reach out in the #fleet channel of osquery Slack to get help from the rest of the community.
- Offer code review and coaching to help get the PR ready to go (see note below).
- Keep an eye out for any updates or responses.

> Sometimes (typically for Fleet customers), a Fleet team member may add tests and make any necessary changes to merge the PR.

If everything is good to go, approve the review.

For PRs that will not be merged:

- Thank the contributor for their effort and explain why the changes won't be merged.
- Close the PR.

## Updating docs and FAQ

When someone asks a question in a public channel, it's pretty safe to assume that they aren't the only person looking for an answer to the same question. To make our docs as helpful as possible, the Community team gathers these questions and uses them to make a weekly documentation update.

Our goal is to answer every question with a link to the docs and/or result in a documentation update.

> **Remember**, when submitting any pull request that changes Markdown files in the docs, request an editor review from Chris McGillicuddy, who will escalate to the [on-call engineer](https://fleetdm.com/handbook/engineering#oncall-rotation) as needed.

### Tracking

When responding to a question or issue in the [#fleet](https://osquery.slack.com/join/shared_invite/zt-h29zm0gk-s2DBtGUTW4CFel0f0IjTEw#/) channel of the osquery Slack workspace, push the thread to Zapier using the `TODO: Update docs` Zap. This will add information about the thread to the [Slack Questions Spreadsheet](https://docs.google.com/spreadsheets/d/15AgmjlnV4oRW5m94N5q7DjeBBix8MANV9XLWRktMDGE/edit#gid=336721544). In the `Notes` field, you can include any information that you think will be helpful when making weekly doc updates. That may be something like

- proposed change to the documentation.
- documentation link that was sent as a response.
- link to associated thread in [#help-oncall](https://fleetdm.slack.com/archives/C024DGVCABZ).

### Making the updates

Every week, the Community Engagement DRI will:

- Create a new `Weekly Doc Update` issue on Monday and add it to the [Community board](https://github.com/orgs/fleetdm/projects/36).
- Review the Slack Questions Spreadsheet and make sure that any necessary updates to the documentation are made.
  - Update the spreadsheet to indicate what action was taken (Doc change, FAQ added, or None) and add notes if need be.
- Set up a single PR to update the Docs.
  - In the notes, include a list of changes made as well as a link to the related thread.
- Bring any questions to DevRel Office Hours (time TBD).
- Submit the PR by the end of the day on Thursday.
- Once the PR is approved, share in the [#fleet channel](https://osquery.slack.com/archives/C01DXJL16D8) of Osquery Slack Workspace and thank the community for being involved and asking questions.

## Fleet swag

We want to recognize and congratulate community members for their contributions to Fleet. Nominating a contributor for Fleet swag is a great way to show our appreciation.

### How to order swag

We currently deliver Fleet swag and osquery stickers for those that request it through community contributions, [Fleet documentation](https://fleetdm.com/docs), and social media posts.

Our Typeform integrations automatically populate information within the #help-swag Slack channel for osquery sticker and shirt requests through TypeForm.

For community contributions, reach out to the contributor to thank them for their contribution, ask if they would like any swag, and fill out their information in the [Fleet swag request sheet](https://docs.google.com/spreadsheets/d/1bySsYVYHY8EjxWhhAKMLVAPLNjg3IYVNpyg50clfB6I/edit#gid=2028707729).

Once approved in the sheet, or submitted through [Typeform](https://admin.typeform.com/form/ZfA3sOu0/results#responses), place the order through our Printful account (credentials in 1Password) within 48 hours of submission. If available through the ordering process, add a thank you note for their contribution or request.

When an estimated shipping date is available, notify the requestor by email with an update on shipping, thank them for being a part of the community, and provide the tracking number once shipped.

Printful order information can be found on [Printful](https://www.printful.com/dashboard/default/orders) or billing@fleetdm.com.

At this time, double-check that information within Salesforce and Typeform is accurate according to the [enrichment process.](https://docs.google.com/document/d/1zOv39O989bPRNTIcLNNE4ESUI5Ry2XII3XuRpJqNN7g/edit?usp=sharing)

## Rituals

The following table lists the Community group's rituals, frequency, and Directly Responsible Individual (DRI).

| Ritual                        | Frequency | Description                                                                                                                                     | DRI                      |
| :---------------------------- | :-------- | :---------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------ |
| Community Slack               | Daily     | Check Fleet and osquery Slack channels for community questions, make sure questions are responded to and logged.                                | Kathy Satterlee          |
| Social media check-in         | Daily     | Check social media for community questions and make sure to respond to them. Generate dev advocacy-related content.                             | Kelvin Omereshone        |
| Issue check-in                | Daily     | Check GitHub for new issues submitted by the community, check the status of existing requests, and follow up when needed.                       | Kathy Satterlee          |
| Outside contributor follow-up | Weekly    | Bring pull requests from outside contributors to engineering and make sure they are merged promptly and promoted.                               | Kathy Satterlee          |
| Documentation update          | Weekly    | Turn questions answered from Fleet and osquery Slack into FAQs in Fleet’s docs.                                                                 | Kathy Satterlee          |
| StackOverflow                 | Weekly    | Search StackOverflow for “osquery,” answer questions with Grammarly, and find a way to feature Fleet in your StackOverflow profile prominently. | Rotation: Community team |

## Slack channels

This group maintains the following [Slack channels](https://fleetdm.com/handbook/company#group-slack-channels):

| Slack channel  | [DRI](https://fleetdm.com/handbook/company#group-slack-channels) |
| :------------- | :--------------------------------------------------------------- |
| `#g-community` | Kathy Satterlee                                                  |

## Digital Experience
[https://fleetdm.com/handbook/marketing/digital-experience](https://fleetdm.com/handbook/marketing/digital-experience)

# Digital Experience

## Publishing Fleet content 

The following describes how to go about publishing and editing content at Fleet.

### Publication methods

1. **Instant**: Content is published instantly. Content is approved by Digital Experience post-facto – see links in the table below to get the required training.
2. **Gated**: Submit content to Digital Experience for review – see specific instructions in the table below.
3. **Queued**: Communicate the publication date with the [DRI](https://fleetdm.com/handbook/people#directly-responsible-individuals) responsible for approving this content – refer to specific instructions linked in the table below. 

### Revision methods (for editors)

1. **Absorb**: Fix and publish yourself
2. **Feedback**: Request changes and wait
3. **Pairing**: Jump on Zoom to finalize with the original contributor.

> Consider: **Absorb** may be risky depending on how sure the editor is about their edits. **Feedback** can be forgotten. **Pairing** is underused but can eat up more time if not careful.

### Timeframe

Detail the minimum time needed for new or updated content to be live (published) and brand-approved (reviewed and revised, if necessary).

### Content types

| Content | To publish | To revise (for editors) | Timeframe |
|:------ |:-----------------|:-------------------------------|-----------|
| Articles | **Queued** – see [How to submit and publish an article](./how-to-submit-and-publish-an-article). | **Absorb** (pair or feedback as needed) – see [How to edit articles, release posts, and press releases](#how-to-edit-articles-release-posts-and-press-releases). | three business days |
| Ads | **Gated**. Request review from Digital Experience – see _(TODO: Creating an ad campaign)_. | **Feedback** or **pair** | five business days |
| Docs | **Gated**. Request review from Chris McGillicuddy – see _(TODO: Adding to the docs)_. | **Absorb** – see [How to edit Markdown pull requests for the docs](#how-to-edit-markdown-pull-requests-for-the-docs). For non-grammar-related revisions: **Feedback** or **pair** with contributor, and request review from the [on-call engineer](https://fleetdm.com/handbook/engineering#oncall-rotation). | two business days |
| Docs (REST API) | **Gated**. Request review from Luke Heath – see _(TODO: Adding to the docs (REST API))_. | **Absorb** – see [How to edit recently merged Pull Requests for the handbook and docs](#how-to-edit-recently-merged-pull-requests-for-the-handbook). For non-grammar-related revisions: **Feedback** or **pair** with contributor, and request review from Luke Heath. | two business days |
| Handbook | **Gated**. Request review from page DRI – see _(TODO: Adding to the handbook)_. | **Absorb** and request review from page DRI – see [How to edit recently merged Pull Requests for the handbook and docs](#how-to-edit-recently-merged-pull-requests-for-the-handbook-and-docs). | two business days |
| Social media (Twitter, FB, LinkedIn.) | **Instant** – see [Posting on social media as Fleet](https://fleetdm.com/handbook/growth#posting-on-social-media-as-fleet). | **Pair** or **absorb** (pair if possible otherwise, silently fix ASAP by editing or deleting the post. Consider that some or many people may have already seen the post, and decide accordingly – see [How to edit social media posts](#how-to-edit-social-media-posts).) | one business day |
| Newsletter/email blast | **Gated**. Request review from Digital Experience – see _(TODO: Creating an email campaign)_. | **Feedback** or **pair** | five business days |
| Press release | **Queued** – see _(TODO: Publishing a press release)_ | **Feedback** or **pair** – see [How to edit articles, release posts, and press releases](#how-to-edit-articles-release-posts-and-press-releases) | three business days |
| Release post | **Queued** – see _(TODO: Publishing release posts)_ | **Feedback** or **pair** – see [How to edit articles, release posts, and press releases](#how-to-edit-articles-release-posts-and-press-releases) | three business days |
| Website (text change) | **Gated** – see _(TODO: Adding content to fleetdm.com)_. | **Feedback** or **pair** | three business days |
| YouTube | **Queued** – see _(TODO: Publishing on YouTube as Fleet.)_ | **Absorb** for revisions to the description. **Pair** or **absorb** for video content (pair if possible otherwise, silently fix ASAP by deleting the post. Consider that the video may also have been promoted on social media – see Social media (Twitter, FB, LinkedIn) above. | three business days |
| Decks | **Instant**. Sales typically creates decks. Digital Experience shouldn't be a blocker. | **Feedback** | three business days |

## Fleet style guide

### Our voice

- **Clear.** Focus on what matters most. Details can clear things up, but too many are confusing. Use simple words and sentences, especially in technical conversations.
- **Thoughtful.** Try your best to understand the topic and your audience. Choose words carefully. Outdated terms may offend modern readers. 
- **Friendly.** Make people feel welcome. Let them know they’re talking to another human who cares. Relate to their struggles and offer solutions.
- **Inspiring.** Empower users with practical advice. Show them what success looks like. Manage risk, not fear. Help people feel confident about handling security threats.

### Our approach
Every piece of content we write should embody our values. To make sure we succeed, we apply a design thinking approach to our writing by following these steps:

- **Empathize.** Who is the reader? Why will they read it? What do they hope to get from it?
- **Define.** What is the subject? What action do you want from the reader?
- **Ideate and collaborate.** Create an outline of what you plan to write. Interview team members or friends of Fleet to help you.
- **Prototype.** Write a draft and see how it goes. Your first pass won’t be perfect. And that’s okay. If it isn’t working, try it again.
- **Test.** Revise, edit, proofread, repeat. Revise, edit, proofread, repeat. Revise, edit... You get the idea.

### What would Mr. Rogers say?

We should be clear, simple, friendly, and inspiring, like [Mr. Rogers](https://en.wikipedia.org/wiki/Fred_Rogers), who deeply understood these communication skills.

Here are some steps you can take to communicate like Mister Rogers:

- State the idea you want to express as clearly as possible.
- Rephrase the idea in a positive manner.
- Rephrase the idea, directing your reader to authorities they trust.
- Rephrase the idea to eliminate anything that may not apply to your reader.
- Add a motivational idea that gives your reader a reason to follow your advice.
- Rephrase the new statement, repeating the first step.
- Rephrase the idea a ﬁnal time, relating it to an important moment in your reader’s life.

Consider this example tweet.

*Distributed workforces aren’t going anywhere anytime soon. It’s past time to start engaging meaningfully with your workforce and getting them to work with your security team instead of around them.*
 
What would Mister Rogers say? The tweet could look something like this...

*Distributed workforces are here to stay. So, it’s a great time to help employees work with your security experts (and not around them). Because stronger teams get to celebrate more victories.*

By Mister Rogersing our writing, we can encourage our readers to succeed by emphasizing optimism. You might not be able to apply all of these steps every time. That’s fine. Think of these as guidelines to help you simplify complex topics.

### Writing documentation

You don’t have to be a “writer” to write documentation. Nobody knows Fleet better than the people who are building our product. That puts developers in the perfect position to show users what Fleet can do.

This guide will help you write docs that help users achieve their goals with Fleet.

#### Remember the reader
People come from different backgrounds. New users may not know terms that are common knowledge for seasoned developers. Since Fleet has users all over the world, English may not be their first language. Your writing must be easy for any user to understand.

- **Think of every user.** Define technical terms in the doc or include a link.
- **Strive for simplicity.** Avoid complex sentences and long paragraphs.
- **Be approachable.** Write like you’re meeting a new member of your team.

#### Answer the question

It’s what we’re all about at Fleet. People read docs in order to accomplish their goals. Those goals can vary from learning about Fleet for the first time to looking for troubleshooting tips. Make sure your doc meets the specific need of the user at that moment.

- **Understand the question.** Be clear about the topic you’re discussing.
- **Narrow your focus.** Avoid explanations that distract from the main topic.
- **No more, no less.** Use just enough information to give an accurate answer.

#### Follow a framework

Starting with a blank page can be scary. That’s why it helps to have a framework for your writing. Follow these four steps to write your docs: introduction, explanation, reference, and troubleshooting.

**Introduction**

Give an overview of the topic. You don’t need to mention everything at the beginning. Briefly establish the question you’re addressing. People want to get to the answer A.S.A.P.

**Explanation**

You’ve let users know why they’re reading your doc. It’s time to make sure they understand the topic. This will be most of your documentation. Don’t shy away from details.

**Reference**

Support your explanation with relevant references. This shows users how to put your explanation into practice. Such material will keep users coming back.

**Troubleshooting**

Nothing is perfect. Your readers understand this. Users will appreciate it if you identify common problems — and provide solutions — before they encounter these issues later.

#### Document every change

Any change to Fleet’s code should be documented, from adding patches to building features. This allows users and Fleeties to stay up to date with improvements to our product.

You don’t need to wait until a change has been made to write a new doc. Starting with documentation can help you discover ways to make Fleet even better.

Writing about how to use a new feature puts you in the shoes of the user. If something seems complicated, you have the opportunity to improve it — before commiting a line of code.

### Writing about Fleet

When talking about Fleet the company, we stylize our name as either "Fleet" or "Fleet Device Management." For Fleet the product, we say either “Fleet” or “Fleet for osquery.” Employees are “Fleeties.”

### Writing about osquery

Osquery should always be written in lowercase unless used to start a sentence or heading. For example:

- Open source software, built on osquery.
- Osquery and Fleet provide structured, convenient access to information about your devices.

### Open source vs. open core

For simplicity, Fleet is always described as "open source" in all writing and verbal communication. In specific situations, such as discussing the distinction between various kinds of open source, it can be appropriate to mention "open core" to clarify your meaning. When in doubt, go with "open source."

### Headings and subheadings

Headings help readers quickly scan content to find what they need. Organize page content using clear headings specific to the topic they describe.

Keep headings brief, organized, and in a logical order:

- H1: Page title
- H2: Main headings
- H3: Subheadings
- H4: Sub-subheadings

Try to stay within three or four heading levels. Complicated documents may use more, but pages with a simpler structure are easier to read.

#### Sentence case

Fleet uses sentence case capitalization for all headings across Fleet EE, fleetdm.com, our documentation, and our social media channels. In sentence case, we write titles as if they were sentences. For example:

*Ask questions about your servers, containers, and laptops running Linux, Windows, and macOS*

As we use sentence case, only the first word of a heading and subheading is capitalized. However, if a word would normally be capitalized in the sentence (e.g., a [proper noun](https://www.grammarly.com/blog/proper-nouns/?&utm_source=google&utm_medium=cpc&utm_campaign=11862361094&utm_targetid=dsa-1233402314764&gclid=Cj0KCQjwg7KJBhDyARIsAHrAXaFwpnEyL9qrS4z1PEAgFwh3RXmQ24zmwmowAyOQbHngsI8W_F730aAaArrwEALw_wcB&gclsrc=aw.ds)) it should remain capitalized in the heading.

> Note the capitalization of “macOS” in the example above. Although this is a proper noun, macOS uses its own style guide from Apple, to which we adhere.

You might’ve noticed that there isn’t a period at the end of the example heading. Fleet headings and subheadings do not use end punctuation unless the heading is a question. If the heading is a question, end the heading with a question mark.

### Bullet points

Bullet points are a clean and simple way to list information. But sticking to consistent rules for punctuation and capitalization can be tough. Here’s how we do it at Fleet.

#### How to introduce bullet points

**Do** use a colon if you introduce a list with a complete sentence.

**Do not** use a colon if you start a list right after a heading.

#### How to use end punctuation with bullet points

End punctuation refers to punctuation marks that are used to end sentences, such as periods, question marks, and exclamation points.

**Do** use end punctuation if your bullet points are complete sentences:

- Project confidence and be informative.
- Educate users about security threats positively.
- We never use fear as a marketing tactic.

**Do not** use end punctuation if your bullet points are sentence fragments, single words, or short phrases:

- Policies
- Enterprise support
- Self-hosted agent auto-updates

**Do not** mix complete sentences with sentence fragments, single words, or short phrases. Consistency makes lists easier to read.

**Do not** use commas or semicolons to end bullet points.

#### How to capitalize bullet points

**Do** use a capital letter at the beginning of every bullet point. The only exceptions are words that follow specific style guides (e.g., macOS).

### Commas

When listing three or more things, use commas to separate the words. This is called a serial comma.

**Do:** Fleet is for IT professionals, client platform engineers, and security practitioners.

**Don’t:** Fleet is for IT professionals, client platform engineers and security practitioners.

Aside from the serial comma, use commas, as usual, to break up your sentences. If you’re unsure whether you need a comma, saying the sentence aloud can give you a clue. If you pause or take a breath, that’s when you probably need a comma.

### Dashes and hyphens

Use a hyphen to link words that function as an adjective to modify a noun or to indicate a range:

- We release Fleet on a three-week cadence.
- Osquery is an open-source agent.
- Monday-Friday

A hyphen is unnecessary when not modifying a noun:

- The Fleet product is released every three weeks.
- Osquery is open source.

### SQL Statements

When adding SQL statements, all SQL reserved words should be uppercase, and all identifiers (such as tables and columns) should be lowercase. Here’s an example:

```sql
   SELECT days, hours, total_seconds FROM uptime;
```

### Grammarly

All of our writers and editors have access to Grammarly, which comes with a handy set of tools, including:

- **Style guide**, which helps us write consistently in the style of Fleet.
- **Brand tones** to keep the tone of our messaging consistent with just the right amount of confidence, optimism, and joy.
- **Snippets** to turn commonly used phrases, sentences, and paragraphs (such as calls to action, thank you messages, etc.) into consistent, reusable snippets to save time.

Our favorite Grammarly feature is the tone detector. It's excellent for keeping messaging on-brand and helps alleviate the doubt that often comes along for the ride during a writing assignment. Take a look at [their video](https://youtu.be/3Ct5Tgg9Imc) that sums it up better than this.

## For editors

### In this section

- [How to make edits with GitHub](#how-to-make-edits-with-git-hub)
- [How to edit recently merged pull requests for the handbook](#how-to-edit-recently-merged-pull-requests-for-the-handbook)
- [How to edit Markdown pull requests for the docs](#how-to-edit-markdown-pull-requests-for-the-docs)
- [How to edit articles, release posts, and press releases](#how-to-edit-articles-release-posts-and-press-releases)
- [How to edit social media posts](#how-to-edit-social-media-posts)

While we encourage and equip our writers to succeed by themselves in editing quests, tpyos are inevitable. Here's where the Fleet editor steps in.  

The following is our handy guide to editor bliss at Fleet, but first, let's start by listing common content types that require an editor pass. 

- Docs and Handbook pages.
- Articles, release posts, and press releases.
- Social media posts.

### How to make edits with GitHub

Our handbook and docs pages are written in Markdown and are editable from our website (via GitHub). Follow the instructions below to propose an edit to the handbook or docs.
1. Click the "Edit page" button from the relevant handbook or docs page on [fleetdm.com](https://www.fleetdm.com) (this will take you to the GitHub editor).
2. Make your suggested edits in the GitHub editor.
3. From the Propose changes dialog, at the bottom of the page, give your proposed edit a title and optional description (these help page maintainers quickly understand the proposed changes).
4. Hit Propose change which will open a new pull request (PR).
5. Request a review from the page maintainer, and finally, press “Create pull request.”
6. GitHub will run a series of automated checks and notify the reviewer. At this point, you are done and can safely close the browser page at any time.

> Keep PR titles short and clear. E.g., "Edit to handbook Product group" 
>
> Check the “Files changed” section on the Open a pull request page to double-check your proposed changes.

### How to edit recently merged pull requests for the handbook

We approach editing retrospectively for pull requests (PRs) to handbook pages. Remember our goal above about moving quickly and reducing time to value for our contributors? We avoid the editor becoming a bottleneck for merging quickly by editing for typos and grammatical errors after-the-fact. Here's how to do it:

> **Note:** Contributors are not required to request reviews from editors for handbook changes.

1. Check that the previous day's edits are formatted correctly on the website (more on this in the note below.)
2. Use the [Handbook history](https://github.com/fleetdm/fleet/commits/main/handbook) feed in GitHub to see a list of changes made to the handbook.
3. From the list of recently merged PRs, look at the files changed for each and then:
  - Scan for typos and grammatical errors.
  - Check that the tone aligns with our [Communicating as Fleet](https://fleetdm.com/handbook/brand#communicating-as-fleet) guidelines and that Grammarly's tone detector is on-brand.
  - Check that Markdown is formatted correctly.
  - **Remember**, Do not make edits to this page. It's already merged.
4. Instead, navigate to the page in question on the website and submit a new PR to make edits - making sure to request a review from the maintainer of that page.
5. Comment on the original PR to keep track of your progress. Comments made will show up on the history feed. E.g., `"Edited, PR incoming"` or `"LGTM, no edits required."`
6. Watch [this short video](https://www.loom.com/share/95d4525a7aae482b9f9a9470d446ce9c) to see this process in action.

> **Note:** The Fleet website may render Markdown differently from GitHub's rich text preview. It's essential to check that PRs merged by the editor are displaying as expected on the site. It can take a few minutes for merged PRs to appear on the live site, and therefore easy to move on and forget. It's good to start the ritual by looking at the site to check that the previous day's edits are displaying as they should.

### How to edit Markdown pull requests for the docs

- When someone creates a pull request for a doc that affects Markdown files, they’ll need to request a review from the editor. 
- If no edits are needed, the editor will merge the PR. 
- If an edit changes the meaning, or if unsure, the editor should request a review from the [on-call engineer](https://fleetdm.com/handbook/engineering#oncall-rotation) and remove themselves as a reviewer.

### How to edit articles, release posts, and press releases

Editing articles, release posts, and press releases usually comes in three flavors: a Google Docs draft, a new pull request, or an edit to an existing article.

* For unpublished articles, please read the review process in [How to submit and publish an article](https://fleetdm.com/handbook/digital-experience/how-to-submit-and-publish-an-article#review-process).

* To edit an existing article, see [How to make edits with GitHub](https://fleetdm.com/handbook/digital-experience#how-to-make-edits-with-git-hub).

### How to edit social media posts

In the world of the Fleet editor, there are two types of social media posts; those scheduled to be published and those published already. 

Refer to [Posting on social media as Fleet](https://fleetdm.com/handbook/growth#posting-on-social-media-as-fleet) for details on editing draft social media posts.

Making edits to published social media posts gets a little tricky. Twitter, for example, doesn't allow editing of tweets, so the only way to make an edit is to remove the tweet and post it again.

1. Post the tweet in the #g-growth Slack channel and tag the Digital Experience lead.
2. Decide whether to remove the tweet. There's a tradeoff between us striving for perfection vs. losing the engagements that the tweet may have already generated.
3. Suggest edits in the Slack thread for the Growth team to include and re-post.

## Commonly used terms

If you find yourself feeling a little overwhelmed by all the industry terms within our space, or if you just need to look something up, our glossary of [commonly used terms](./commonly-used-terms) is here to help.

## Brand resources

To download official Fleet logos, product screenshots, and wallpapers, head over to our [brand resources](https://fleetdm.com/logos) page.

See also [https://fleetdm.com/handbook/community#press-releases](https://fleetdm.com/handbook/community#press-releases) for our press-release boilerplate.

## Email blasts

Do you need to send out a branded email blast to multiple recipients?

### The manual way
Use "bcc" so recipients don't see each other's email addresses and send an email manually using Gmail.   (Good for small lists.  This is definitely a "thing that doesn't scale.")

### The automated way

- First, design the email and content. The preferred method is to base the design on one of our existing [email templates](https://www.figma.com/file/yLP0vJ8Ms4GbCoofLwptwS/?node-id=3609%3A12552) in Figma. If your Figma boots aren't comfortable (or you don't have edit access), your design could be a Google Drawing, Doc, or just a sketch on paper in a pinch.
- Bring your request to the digital experience team by posting it in [their primary Slack channel](./people.md#slack-channels), along with your urgency/timeline.  The digital experience team will finalize the design and language for consistency, then fork and customize [one of the existing email templates](https://github.com/fleetdm/fleet/blob/de280a478834a7f85772bea4f552f953c65bb29e/website/views/emails/email-order-confirmation.ejs) for you, and write a script to deliver it to your desired recipients. Then, digital experience will merge that, test it by hand to make sure it's attractive and links work, and then tell you how to run the script with e.g.;

  `sails run deliver-release-announcement --emailAddresses='["foo@example.com","bar@example.com"]'`

## Using Figma

We use Figma for most of our design work. This includes the Fleet product, our website, and our marketing collateral. 

### Which file should I use?

**Fleet product** All product design work is done in the [Fleet EE (scratchpad)](https://www.figma.com/file/hdALBDsrti77QuDNSzLdkx/%F0%9F%9A%A7-Fleet-EE-(dev-ready%2C-scratchpad)?node-id=9209%3A302838) Figma doc. Check out the [README](https://www.figma.com/file/hdALBDsrti77QuDNSzLdkx/%F0%9F%9A%A7-Fleet-EE-(dev-ready%2C-scratchpad)?node-id=2750%3A67203) for how to use this doc.

**Fleet website.** All website design work is done in the [fleetdm.com (current, dev-ready)](https://www.figma.com/file/yLP0vJ8Ms4GbCoofLwptwS/%E2%9C%85-fleetdm.com-(current%2C-dev-ready)?node-id=794%3A373) Figma file.

**Design system.** Shared logos, typography styles, and UI components can be found in [Design system](https://www.figma.com/files/project/15701210).

> The Figma docs in Design System contain the master components that are referenced throughout all other Figma files. Use caution when modifying these components, as changes will be reflected in the master Fleet EE (scratchpad) and fleetdm.com (current, dev-ready) Figma docs.

**Marketing assets.** Product screenshots and artwork for social media, articles, and other marketing assets can be found in [Collateral](https://www.figma.com/files/project/20798819).

> Looking for the official Fleet logo? Download it from: https://fleetdm.com/logos.


## Fleet website

The Digital Experience team is responsible for production and maintenance of the Fleet website.

#### In this section

- [Wireframes](#wireframes)
- [Design reviews](#design-reviews)
- [Estimation sessions](#estimation-sessions)
- [When can I merge changes to the website?](#when-can-i-merge-a-change-to-the-website)
- [How to export images for the website](#how-to-export-images-for-the-website)
- [Maintaining browser compatibility](#maintaining-browser-compatibility)
- [Responding to a 5xx error on fleetdm.com](#responding-to-a-5-xx-error-on-fleetdm-com)
- [The "Deploy Fleet Website" GitHub action failed](#the-deploy-fleet-website-git-hub-action-failed)
- [Vulnerability monitoring](#vulnerability-monitoring)
- [How to make usability changes to the website](#how-to-make-usability-changes-to-the-website)

### Wireframes

Before committing anything to code, we create wireframes to illustrate all changes that affect the website layout and structure.

See [Why do we use a wireframe first approach](https://fleetdm.com/handbook/company/why-this-way#why-do-we-use-a-wireframe-first-approach) for more information. 

### Design reviews

We hold regular design review sessions to evaluate, revise, and approve wireframes before moving into production.

Design review sessions are hosted by [Mike Thomas](https://calendar.google.com/calendar/u/0?cid=bXRob21hc0BmbGVldGRtLmNvbQ) and typically take place daily, late afternoon (CST). Anyone is welcome to join.   

### Estimation sessions

We use estimation sessions to estimate the effort required to complete a prioritized task. 

Through these sessions, we can:

- Confirm that wireframes are complete before moving to production
- Consider all edge cases and requirements that may have been with during wireframing.
- Avoid having the engineer make choices for “unknowns” during production.
- More accurately plan and prioritize upcoming tasks.

#### Story points

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

### When can I merge a change to the website?
When merging a PR to master, remember that whatever you merge to master gets deployed live immediately. So if the PR's changes contain anything that you don't think is appropriate to be seen publicly by all guests of [fleetdm.com](https://fleetdm.com/), please do not merge.

Merge a PR (aka deploy the website) when you think it is appropriately clean to represent our brand. When in doubt, use the standards and quality seen on existing pages, ensure correct functionality, and check responsive behavior - starting widescreen and resizing down to ≈320px width.

### How to export images for the website
In Figma:
1. Select the layers you want to export.
2. Confirm export settings and naming convention:
  * Item name - color variant - (CSS)size - @2x.fileformat (e.g., `os-macos-black-16x16@2x.png`)
  * Note that the dimensions in the filename are in CSS pixels.  In this example, if you opened it in preview, the image would actually have dimensions of 32x32px but in the filename, and in HTML/CSS, we'll size it as if it were 16x16.  This is so that we support retina displays by default.
  * File extension might be .jpg or .png.
  * Avoid using SVGs or icon fonts.
3. Click the __Export__ button.

### Maintaining browser compatibility

A browser compatibility check of [fleetdm.com](https://fleetdm.com/) should be carried out monthly to verify that the website looks and functions as expected across all [supported browsers](../docs/Using-Fleet/Supported-browsers.md).

- We use [BrowserStack](https://www.browserstack.com/users/sign_in) (logins can be found in [1Password](https://start.1password.com/open/i?a=N3F7LHAKQ5G3JPFPX234EC4ZDQ&v=3ycqkai6naxhqsylmsos6vairu&i=nwnxrrbpcwkuzaazh3rywzoh6e&h=fleetdevicemanagement.1password.com)) for our cross-browser checks.
- Check for issues against the latest version of Google Chrome (macOS). We use this as our baseline for quality assurance.
- Document any issues in GitHub as a [bug report](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=bug%2C%3Areproduce&template=bug-report.md&title=), and assign them for fixing.
- If in doubt about anything regarding design or layout, please reach out to the Design team.

### Responding to a 5xx error on fleetdm.com
Production systems can fail for various reasons, and it can be frustrating to users when they do, and customer experience is significant to Fleet. In the event of system failure, Fleet will:
* investigate the problem to determine the root cause.
* identify affected users.
* escalate if necessary.
* understand and remediate the problem.
* notify impacted users of any steps they need to take (if any).  If a customer paid with a credit card and had a bad experience, default to refunding their money.
* Conduct an incident post-mortem to determine any additional steps we need (including monitoring) to take to prevent this class of problems from happening in the future.

#### Incident post-mortems

When conducting an incident post-mortem, answer the following three questions:

1. Impact: What impact did this error have? How many humans experienced this error, if any, and who were they?
2. Root Cause: Why did this error happen?
3. Side effects: did this error have any side effects? e.g., did it corrupt any data? Did code that was supposed to run afterward and “finish something up” not run, and did it leave anything in the database or other systems in a broken state requiring repair? This typically involves checking the line in the source code that threw the error. 

### The "Deploy Fleet Website" GitHub action failed
If the action fails, please complete the following steps:
1. Head to the fleetdm-website app in the [Heroku dashboard](https://heroku.com) and select the "Activity" tab.
2. Select "Roll back to here" on the second to most recent deploy.
3. Head to the fleetdm/fleet GitHub repository and re-run the Deploy Fleet Website action.

### Vulnerability monitoring

Every week, we run `npm audit --only=prod` to check for vulnerabilities on the production dependencies of fleetdm.com. Once we have a solution to configure GitHub's Dependabot to ignore devDependencies, this manual process can be replaced with Dependabot.

### How to make usability changes to the website

We want to make it easy to learn how to manage devices with Fleet. Anyone inside or outside the company can suggest changes to the website to improve ease of use and clarity. 

To propose changes:
1. Decide what you want to change. A small change is the best place to start.
2. Wireframe the design. Usually, digital experience does this, but anyone can contribute.
3. Present your change to the website DRI. They will approve it or suggest revisions.
4. Code the website change. Again, digital experience often does this, but anyone can help.
5. Measure if the change made it easier to use. This can be tricky, but the growth team will have ideas on how to do this.


## Rituals

The following table lists the Brand group's rituals, frequency, and Directly Responsible Individual (DRI).

| Ritual                       | Frequency                | Description                                         | DRI               |
|:-----------------------------|:-----------------------------|:----------------------------------------------------|-------------------|
| Documentation quality | On request | Review pull requests to the docs for spelling, punctuation, and grammar. | Chris McGillicuddy |
| Handbook quality | Daily | Review pull requests to the handbook for spelling, punctuation, and grammar. | Chris McGillicuddy |
| Tweet review | Daily | Review tweets for tone and brand consistency. | Mike Thomas |
| Article review | Weekly | Review articles for tone and brand consistency. | Mike Thomas |
| Article graphic | Weekly | Create a graphic for the weekly article | Mike Thomas |
| Digital experience planning  | Three weeks | Prioritize and assigns issues to relevant personnel based on current goals and quarterly OKRs | Mike Thomas |
| OKR review  | Three weeks | Review the status of current OKRs. | Mike Thomas |
| Handbook editor pass | Monthly | Edit for copy and content. | Chris McGillicuddy |
| Browser compatibility check | Monthly | Check browser compatibility for the website | Eric Shaw |
| OKR planning  | Quarterly | Plan next quarter's OKRs | Mike Thomas |
| Website vulnerability check  | Weekly | Checking for vulnerabilities on fleetdm.com | Eric Shaw |
| Updating the extended osquery schema | Three weeks | Running the `generate-merged-schema` script and committing the merged schema json to the Fleet GitHub repo | Eric Shaw |

## Slack channels

These groups maintain the following [Slack channels](https://fleetdm.com/handbook/company#group-slack-channels):

| Slack channel               | [DRI](https://fleetdm.com/handbook/company#group-slack-channels)    |
|:----------------------------|:--------------------------------------------------------------------|
| `#g-digital-experience`     | Mike Thomas
| `#oooh-websites`            | Mike Thomas
| `#help-p1`		      | Mike McNeil



<meta name="maintainedBy" value="mike-j-thomas">
<meta name="title" value="💓 Digital Experience">

## Growth
# Growth

As an open-core company, Fleet endeavors to build a community of engaged users, customers, and
contributors. The purpose of the growth team is to own and improve the growth funnel to drive awareness, adoption, and referrals of Fleet while honoring the ideals and voice of the open source community and our company values.

## Positioning

Effective market positioning is crucial to the growth of any software product. Fleet needs to maintain a unique, valuable position in the minds of our users. We keep assertions on our positioning in this [Google Doc](https://docs.google.com/document/d/177Q4_2FY5Vm7Nd3ne32vOKTQqYfDG0p_ouklvl3PnWc/edit) (private). We will update it quarterly based on the feedback of users, customers, team members, and other stakeholders. Feedback can be provided as a comment in the document or by posting in the `#g-growth` Slack channel. 

## Marketing Qualified Opportunities (MQOs)

Growth's goal is to increase product usage. We value users of all sizes adopting Fleet Free or Fleet Premium. Companies purchasing under 100 device licenses should sign up for [self-service](https://fleetdm.com/pricing/). Companies that enroll more than 100 devices should [talk to an expert](https://fleetdm.com/). When these companies attend this meeting, Fleet considers them Marketing Qualified Opportunities (MQOs).

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

[https://fleetdm.com/handbook/marketing/growth](https://fleetdm.com/handbook/marketing/growth)


<meta name="maintainedBy" value="timmy-k">
<meta name="title" value="🫧 Marketing">

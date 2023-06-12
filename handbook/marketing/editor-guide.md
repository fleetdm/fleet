# Editor guide

While we encourage and equip our writers to succeed by themselves in editing quests, tpyos are inevitable. Here's where the Fleet editor steps in.  

The following is our handy guide to editor bliss at Fleet, but first, let's start by listing common content types that require an editor pass. 

- Docs and Handbook pages.
- Articles, release posts, and press releases.
- Social media posts.

#### How to make edits with GitHub

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

#### How to edit recently merged pull requests for the handbook

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

#### How to edit Markdown pull requests for the docs

- When someone creates a pull request for a doc that affects Markdown files, they’ll need to request a review from the editor. 
- If no edits are needed, the editor will merge the PR. 
- If an edit changes the meaning, or if unsure, the editor should request a review from the [on-call engineer](https://fleetdm.com/handbook/engineering#oncall-rotation) and remove themselves as a reviewer.

#### How to edit articles, release posts, and press releases

Editing articles, release posts, and press releases usually comes in three flavors: a Google Docs draft, a new pull request, or an edit to an existing article.

* For unpublished articles, please read the review process in [How to submit and publish an article](./how-to-submit-and-publish-an-article#review-process).

* To edit an existing article, see [How to make edits with GitHub](#how-to-make-edits-with-github).

#### How to edit social media posts

In the world of the Fleet editor, there are two types of social media posts; those scheduled to be published and those published already. 

Making edits to published social media posts gets a little tricky. Twitter, for example, doesn't allow editing of tweets, so the only way to make an edit is to remove the tweet and post it again.

1. Post the tweet in the #g-marketing Slack channel and tag the Brand team lead.
2. Decide whether to remove the tweet. There's a tradeoff between us striving for perfection vs. losing the engagements that the tweet may have already generated.
3. Suggest edits in the Slack thread for the Marketing team to include and re-post.


<meta name="maintainedBy" value="mike-j-thomas">
<meta name="title" value="Editor guide">

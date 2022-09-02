# Handbook

## About the handbook

The Fleet handbook is inspired by the [GitLab team handbook](https://about.gitlab.com/handbook/about/).  It shares the same [advantages](https://about.gitlab.com/handbook/about/#advantages) and will probably undergo a similar [evolution](https://about.gitlab.com/handbook/ceo/#evolution-of-the-handbook).

While [GitLab's handbook](https://about.gitlab.com/handbook/) inspires this handbook, it is nowhere near as complete (yet!)  We will continue adding and updating this handbook and gradually migrating information from [Fleet's shared Google Drive folder](https://drive.google.com/drive/u/0/folders/1StSOI3HNcsl9VleXxNWfUBT2co7h44OG) as time allows.

## Handbook style guide

### Headings

Headings help readers scan content to find what they need easily. Organize page content using clear headings specific to the topic they describe.

Keep headings brief and organize them in a logical order:

* H1: Page title
* H2: Main headings
* H3: Subheadings
* H4: Sub-subheadings (headings nested under subheadings)

Try to stay within three or four heading levels. Complicated documents may use more, but pages with a simpler structure are easier to read.

## Contributing to the handbook

To contribute to a handbook page:
1. Click "Edit this page."
2. Make your changes in the browser(The language is [Markdown](https://github.github.com/gfm/)).
3. Click "Propose changes."
4. Request a reviewer by clicking the gear and picking only one.  Choose the reviewer whose face is on the handbook page when you view it on fleetdm.com.
5. Click "Create pull request."

All done!

### Adding a new handbook page

To contribute a new handbook page:
1. Determine where the new page should live in the handbook.  That is, nested under either:
  a. [the "Company" handbook](https://fleetdm.com/handbook/company), or
  b. the handbook for a particular division (Security, Engineering, Product, Sales, Marketing, Business Operations)
2. Locate the appropriate folder for the new page in [the GitHub repository under `handbook/`](https://github.com/fleetdm/fleet/tree/main/handbook).
3. Create a new markdown file (like [one of these](https://github.com/fleetdm/fleet/tree/f90148abad96fccb6c5647a31877fa7e91b5ee57/handbook/digital-experience)).  A simple, easy way to do this is by clicking "Add file" on GitHub.com.
  a. Name your new file the kebab-cased, all lowercase version of your page title, with `.md` at the end.  (For example, a page titled "Why this way?" would have the file path: `handbook/company/why-this-way.md`.)
  b. At the top of your new page, include an H1 (`# Page title here`) with the same name as your page.
  c. At the bottom of your new page, include the appropriate `meta` tag to indicate the page maintainer.  (This is usually the same person who is the maintainer of the top-level page.  The easiest way to do this is to copy the tags from the bottom of the top-level page and paste them in to your new page, changing their values to suit, as-needed.)
4. Submit your change, requesting review from the maintainer of the top-level page.

> Note: GitHub _should_ automatically request review from the right person when submitting your merge request, thanks to CODEOWNERS.  Configuration for the auto-approval bot should also be taken care of automatically, so there's no further action needed from you.)


<meta name="maintainedBy" value="mike-j-thomas">
<meta name="title" value="Handbook">

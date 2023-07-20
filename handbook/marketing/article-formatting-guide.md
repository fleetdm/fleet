# Article formatting guide

> Before creating a pull request for an article, please submit a Google Doc draft (see [How to submit an article](./how-to-submit-and-publish-an-article.md#how-to-submit-an-article)). If you need help with Fleet-flavored Markdown, please read our handy [Markdown guide](./markdown-guide).

## Layout
The following layout guide aims to help you create consistently formatted articles. For an existing article example, check out the [Markdown](https://raw.githubusercontent.com/fleetdm/fleet/main/articles/tales-from-fleet-security-speeding-up-macos-updates-with-nudge.md) and the [finished result](https://fleetdm.com/securing/tales-from-fleet-security-speeding-up-macos-updates-with-nudge).

### Hero image
Consider adding a hero image for a more significant impact. Get in touch with Digital Experience in [#help-content-calendar](https://fleetdm.slack.com/archives/C03PH3BBVSM) on Slack to make a request. 

### Table of contents
For long articles or guides, consider adding a table of contents.

### Introduction
It’s good practice to start your article with a clear summary of what you will be discussing.

### Main content
The main body of your article.

### Conclusion
It’s a good idea to finish your article with a clear closing statement.

## Images and screenshots
Images are a great way to help engage your readers. But consider the following before including images or screenshots in your article:

- Does the image add value?
- Is your image likely to go out of date soon? (Consider the long-term maintenance of your article.)

## Meta tags
These tags help pass information to the website about the article to display and store it. 

```
<meta name="articleTitle" value="Deploying Fleet on Render">
<meta name="authorFullName" value="Ben Edwards">
<meta name="authorGitHubUsername" value="edwardsb">
<meta name="category" value="guides">
<meta name="publishedOn" value="2021-11-21">
<meta name="articleImageUrl" value="../website/assets/images/articles/deploying-fleet-on-render-cover-1600x900@2x.jpg">
<meta name="description" value="Learn how to configure and deploy Fleet on Render in 30 minutes">
```

### `articleTitle`
The title of your article. Avoid long titles. As a rule of thumb, your title should not exceed two lines at desktop browser resolution. This is roughly 75 characters (including spaces).

### `authorFullName`
Add the author’s full name here. Our system does not currently support multiple authors.

### `authorGitHubUsername`
Add the author’s GitHub username to populate the author’s headshot.

### `category`
Choose only __one__ of the following categories for your article.

- __Announcements__: company or product announcements (including breaking changes), news, or events.
- __Engineering__: posts about engineering at Fleet and other engineering-related topics.
- __Guides__: help articles for using and deploying Fleet.
- __Podcasts__: podcast-related posts.
- __Product__: posts related to Fleet features.
- __Releases__: release posts, security, and patch releases.
- __Reports__: posts about the industry, data, surveys, etc.
- __Security__: posts about how we approach security at Fleet and other security-related topics.
- Success stories: stories from users or customers successfully using Fleet. 

### `publishedOn`
The date that the article was published. Please follow the correct date format, e.g., __2021-09-29__.

### `articleImageUrl`
The relative url path for the article cover image. Article images are stored in `../website/assets/images/articles/` See [How to export images for the website](https://fleetdm.com/handbook/brand#how-to-export-images-for-the-website).

### `description`
The description meta-tag appears on social media posts when shared (e.g., on Twitter) and on browser results pages. It is also important for SEO purposes.

The description should be between 50 - 150 characters and provide a summary of your article to give context or information to readers. Do not repeat the title for the description.

> If you do not include a description, fleetdm.com will create a description using the articleTitle and the authorFullName meta tags

## Customizable CTA
Use the following code snippet to include an inline CTA (call to action) in your article:

```
<call-to-action 
  title="All the data you need, without the performance hit."
  text="Fleet is the lightweight management platform for laptops and servers."
  primary-button-text="Try Fleet Free" 
  primary-button-href="/get-started?try-it-now" 
  secondary-button-text="Schedule a demo"
  secondary-button-href="https://calendly.com/fleetdm/demo">
</call-to-action>
```

![Customizable CTA example](../../images/cta-example-1-900x320@2x.jpg)

__Tip__: paste the code-snippet at the end of your article, or, when creating long articles, consider adding a CTA mid-way through.

### How to modify the customizable CTA
You can customize the CTA to promote what's relevant to your article.

#### `title`
The main call to action text

#### `text`
The proposition statement for your call to action

#### `primary-button-text`
The main call to action interaction. E.g., “Get started.”

#### `primary-button-href`
The URL link for your primary CTA.

#### `secondary-button-text` (optional)
The secondary call to action interaction. E.g., “Schedule a demo.”

#### `secondary-button-href` 
The URL link for your secondary CTA.

#### `preset` (optional)
If provided, a `preset` will override all other values passed into the call to action component and the component will be rendered as a preset call to action. Check out our [preset examples](#preset-examples) to see our current presets.

### Example
In the following example we will modify `title`, `text`, `primary-button-text`, and also remove `secondary-button-text` and `secondary-button-href` to create a call to action that promotes a job opening at Fleet.

```
<call-to-action 
  title="We're hiring remote engineers, worldwide."
  text="Are you interested in working full time in Fleet's public GitHub repository?"
  primary-button-text="Apply now" 
  primary-button-href="https://fleetdm.com/jobs"> 
</call-to-action>
```

![Customizable CTA example](../../images/cta-example-2-900x280@2x.jpg)

### Preset examples

`<call-to-action preset="mdm-beta"></call-to-action>`

<call-to-action preset="mdm-beta">
</call-to-action>

`<call-to-action preset="premium-upgrade"></call-to-action>`

<call-to-action preset="premium-upgrade">
</call-to-action>

## Related pages
- [How to submit and publish an article](./how-to-submit-and-publish-an-article.md)
- [Markdown guide](./markdown-guide)
- Writing style guide (todo)

<meta name="maintainedBy" value="mike-j-thomas">
<meta name="title" value="Article formatting guide">
<meta name="description" value="A guide for formatting Markdown articles for use on the Fleet website">

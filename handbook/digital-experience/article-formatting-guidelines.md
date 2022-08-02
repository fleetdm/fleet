# Article formatting guidelines

To publish an article, you will need to create a Pull Request for a new file, formatted in Markdown (todo), in [https://github.com/fleetdm/fleet/tree/main/articles](https://github.com/fleetdm/fleet/tree/main/articles).

### On this page
- [Layout](#layout)
- [Images and screenshots](#images-and-screenshots)
- [Meta tags](#meta-tags)
- [Customizable CTA](#customizable-cta)
- [Other pages of interest](#other-pages-of-interest)

## Layout
The following layout guide aims to help you create consistently formatted articles. For an existing article example, check out the [Markdown](https://raw.githubusercontent.com/fleetdm/fleet/main/articles/tales-from-fleet-security-speeding-up-macos-updates-with-nudge.md) and the [finished result](https://fleetdm.com/securing/tales-from-fleet-security-speeding-up-macos-updates-with-nudge).

### Hero image
Consider adding a hero image for a more significant impact. Get in touch with Digital Experience via #content on Slack to make a request. 

### Table of contents
For long articles or guides, consider adding a table of contents.

### Introduction
It’s good practice to start your article with a clear summary of what you will be discussing.

### Main content
The main body of your article.

### Conclusion
It’s a good idea to finish your article with a clear closing statement.

### Add a customizable CTA
Add a CTA at the end of your article. See [Customizable CTA](#customizable-cta) below for instructions on creating a CTA tailored to your article topic.

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

## Customizable CTA
Use the following code snippet to include an inline CTA (call to action) in your article:

```
<call-to-action 
  title=”All the data you need, without the performance hit.”
  text=”Fleet is the lightweight telemetry platform for servers and workstations.”
  primary-button-text=”Try Fleet Free” 
  primary-button-href=”/get-started?try-it-now” 
  secondary-button-text=”Schedule a demo”
  secondary-button-href=”calendly.com/fleetdm/demo”>
</call-to-action>
```

![Customizable CTA example](../../images/cta-example-1-900x320@2x.jpg)

> __Tip__: paste the code-snippet at the end of your article, or, when creating long articles, consider adding a CTA mid-way through.

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

### Example
In the following example we will modify `title`, `text`, `primary-button-text`, and also remove `secondary-button-text` and `secondary-button-href` to create a call to action that promotes a job opening at Fleet.

```
<call-to-action 
  title=”We're hiring remote engineers, worldwide.”
  text=”Are you interested in working full time in Fleet's public GitHub repository?”
  primary-button-text=”Apply now” 
  primary-button-href=”https://fleetdm.com/jobs” 
</call-to-action>
```

![Customizable CTA example](../../images/cta-example-2-900x280@2x.jpg)

## Other pages of interest
- [Process for submitting and publishing articles](https://docs.google.com/document/d/1owejJ7PjCVm0e21QNXjzw7SRMa3FdkRxb8WoHkKxWRE/edit?usp=sharing)
- Markdown guide (todo)
- Writing style guide (todo)

<meta name="maintainedBy" value="mike-j-thomas">
<meta name="title" value="Article formatting guidelines">
## Promoting blog posts on social media

Once a blog post has been written, approved, and published, please ensure that it has been promoted on social media. Please refer to our [Publishing as Fleet](https://docs.google.com/document/d/1cmyVgUAqAWKZj1e_Sgt6eY-nNySAYHH3qoEnhQusph0/edit?usp=sharing) guide for more detailed information. 


## Fleet website

### How to export images
In Figma:
1. Select the layers you want to export.
2. Confirm export settings and naming convention:
  * item name - color variant - (css)size - @2x.fileformat (e.g., `os-macos-black-16x16@2x.png`)
  * note that the dimensions in the filename are in CSS pixels.  In this example, the image would actually have dimensions of 32x32px, if you opened it in preview.  But in the filename, and in HTML/CSS, we'll size it as if it were 16x16.  This is so that we support retina displays by default.
  * File extension might be .jpg or .png.
  * Avoid using SVGs or icon fonts.
3. Click the __Export__ button.

### When can I merge a change to the website?
When merging a PR to master, bear in mind that whatever you merge to master gets deployed live immediately. So if the PR's changes contain anything that you don't think is appropriate to be seen publicly by all guests of [fleetdm.com](https://fleetdm.com/), then please do not merge.

Merge a PR (aka deploy the website) when you think it is appropriately clean to represent our brand. When in doubt, use the standards and level of quality seen on existing pages, ensure correct functionality, and check responsive behavior - starting widescreen and resizing down to ≈320px width. 

### The "Deploy Fleet Website" GitHub action failed
If the action fails, please complete the following steps:
1. Head to the fleetdm-website app in the [Heroku dashboard](https://heroku.com) and select the "Activity" tab.
2. Select "Roll back to here" on the second to most recent deploy.
3. Head to the fleetdm/fleet GitHub repository and re-run the Deploy Fleet Website action.


### Browser compatibility checking

A browser compatibility check of [fleetdm.com](https://fleetdm.com/) should be carried out monthly to verify that the website looks, and functions as expected across all [supported browsers](../docs/01-Using-Fleet/12-Supported-browsers.md).

- We use [BrowserStack](https://www.browserstack.com/users/sign_in) (logins can be found in [1Password](https://start.1password.com/open/i?a=N3F7LHAKQ5G3JPFPX234EC4ZDQ&v=3ycqkai6naxhqsylmsos6vairu&i=nwnxrrbpcwkuzaazh3rywzoh6e&h=fleetdevicemanagement.1password.com)) for our cross-browser checks.
- Check for issues against the latest version of Google Chrome (macOS). We use this as our baseline for quality assurance.
- Document any issues in GitHub as a [bug report](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=bug%2C%3Areproduce&template=bug-report.md&title=), and assign for fixing.
- If in doubt about anything regarding design or layout, please reach out to the Design team.

## Grammar guidelines

### How to write headings & subheadings
Fleet uses sentence case capitalization for all headings across Fleet EE, fleetdm.com, our documentation, and our social media channels.
In sentence case, we write titles as if they were sentences. For example:
> **A**sk questions about your servers, containers, and laptops running **L**inux, **W**indows, and macOS

As we are using sentence case, only the first word of a heading and subheading is capitalized. However, if a word in the sentence would normally be capitalized (e.g. a [proper noun](https://www.grammarly.com/blog/proper-nouns/?&utm_source=google&utm_medium=cpc&utm_campaign=11862361094&utm_targetid=dsa-1233402314764&gclid=Cj0KCQjwg7KJBhDyARIsAHrAXaFwpnEyL9qrS4z1PEAgFwh3RXmQ24zmwmowAyOQbHngsI8W_F730aAaArrwEALw_wcB&gclsrc=aw.ds),) these words should also be capitalized in the heading.
> Note the capitalization of _“macOS”_ in the example above. Although this is a proper noun, macOS uses its own style guide from Apple, that we adhere to.

### How use osquery in sentences and headings
Osquery should always be written in lowercase, unless used to start a sentence or heading. For example:
> Open source software, built on osquery.

or

> Osquery and Fleet provide structured, convenient access to information about your devices.




<meta name="maintainedBy" value="mike-j-thomas">


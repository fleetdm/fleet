# Brand

## Communicating as Fleet

- **Sound positive**, and assume positive intent. A positive tone helps to empower our users and encourages them to succeed with Fleet.

- **Be relatable**, friendly and sincere. Being relatable reminds our users that they're talking to another human that cares. Use simple words and sentences, especially in technical conversations. 

- **Project confidence**, and be informative. Clearly tell users what they need to know, remembering to always stay positive so as NOT to sound overconfident.

- **Manage risk, not fear**. Educate users about security threats positively. Risk management is smart, but focusing on fear can lead to poor decisions. We NEVER use fear as a communication and marketing tactic. 

- **Consider the meaning of words**. We never want to offend people or sound judgemental. Industry jargon that was once commonly used may now be considered offensive and should be avoided.

### What would Mr Rogers say?

At Fleet, our voice and tone should be clear, simple, friendly, and inspiring - like [Mr. Rogers](https://en.wikipedia.org/wiki/Fred_Rogers) who had a deep understanding of these communication values.

Consider the example tweets below. What would Mr. Rogers say?

> *Distributed workforces aren’t going anywhere anytime soon. **It’s past time** to **start engaging meaningfully** with your **workforce** and **getting them** to work with your security team instead of around them.*
 
becomes...

> *Distributed workforces aren’t going anywhere anytime soon, **so it’s a great time** to **engage** with your **crew** and **help them** to work with your security team instead of around them.*

By Mr Rogering our writing we can emphasize positivity, optimism and encourage our readers to succeed. The example above also considers sentence flow and use of synonyms to reduce repetition.

Another example to consider is industry jargon that may now be inappropriate. While the term *"responsible vulnerability disclosure"* has been used for decades, it supposes that people who use a different process are *irresponsible*. Using *coordinated disclosure* is a more positive way to discuss the issue.

## Voice and tone guidelines

### How to use our name

When talking about Fleet the company, we stylize our name as either *"Fleet"* or *"Fleet Device Management"*.
For Fleet the product, we say either *“Fleet”* or *“Fleet for osquery”*.

### How to write headings & subheadings
Fleet uses **sentence case** capitalization for all headings across Fleet EE, fleetdm.com, our documentation, and our social media channels.
In **sentence case**, we write titles as if they were sentences. For example:
> **A**sk questions about your servers, containers, and laptops running **L**inux, **W**indows, and macOS

As we are using sentence case, only the first word of a heading and subheading is capitalized. However, if a word in the sentence would normally be capitalized (e.g. a [proper noun](https://www.grammarly.com/blog/proper-nouns/?&utm_source=google&utm_medium=cpc&utm_campaign=11862361094&utm_targetid=dsa-1233402314764&gclid=Cj0KCQjwg7KJBhDyARIsAHrAXaFwpnEyL9qrS4z1PEAgFwh3RXmQ24zmwmowAyOQbHngsI8W_F730aAaArrwEALw_wcB&gclsrc=aw.ds),) these words should also be capitalized in the heading.
> Note the capitalization of _“macOS”_ in the example above. Although this is a proper noun, macOS uses its own style guide from Apple, that we adhere to.

### How use osquery in sentences and headings
Osquery should always be written in lowercase, unless used to start a sentence or heading. For example:
> _Open source software, built on **o**squery._

or

> _**O**squery and Fleet provide structured, convenient access to information about your devices._

## Brand resources

To download official Fleet logos, product screenshots, and wallpapers, head over to our [brand resources](https://fleetdm.com/logos) page.

See also [https://fleetdm.com/handbook/community#press-releases](https://fleetdm.com/handbook/community#press-releases) for our press-release boilerplate.

## Fleet website

### Responding to a 5xx error on fleetdm.com
Production systems can fail for various reasons, and it can be frustrating to users when they do, and customer experience is significant to Fleet. In the event of system failure, Fleet will:
* Investigate the problem to determine the root cause
* Identify affected users
* Escalate if necessary
* Understand and remediate the problem
* Notify impacted users of any steps they need to take (if any).  If a customer paid with a credit card and had a bad experience, default to refunding their money.
* Conduct an incident post-mortem to determine any additional steps we need (including monitoring) to take to prevent this class of problems from happening in the future

### When can I merge a change to the website?
When merging a PR to master, bear in mind that whatever you merge to master gets deployed live immediately. So if the PR's changes contain anything that you don't think is appropriate to be seen publicly by all guests of [fleetdm.com](https://fleetdm.com/), then please do not merge.

Merge a PR (aka deploy the website) when you think it is appropriately clean to represent our brand. When in doubt, use the standards and level of quality seen on existing pages, ensure correct functionality, and check responsive behavior - starting widescreen and resizing down to ≈320px width. 

### The "Deploy Fleet Website" GitHub action failed
If the action fails, please complete the following steps:
1. Head to the fleetdm-website app in the [Heroku dashboard](https://heroku.com) and select the "Activity" tab.
2. Select "Roll back to here" on the second to most recent deploy.
3. Head to the fleetdm/fleet GitHub repository and re-run the Deploy Fleet Website action.

### Maintaining browser compatibility

A browser compatibility check of [fleetdm.com](https://fleetdm.com/) should be carried out monthly to verify that the website looks, and functions as expected across all [supported browsers](../docs/Using-Fleet/Supported-browsers.md).

- We use [BrowserStack](https://www.browserstack.com/users/sign_in) (logins can be found in [1Password](https://start.1password.com/open/i?a=N3F7LHAKQ5G3JPFPX234EC4ZDQ&v=3ycqkai6naxhqsylmsos6vairu&i=nwnxrrbpcwkuzaazh3rywzoh6e&h=fleetdevicemanagement.1password.com)) for our cross-browser checks.
- Check for issues against the latest version of Google Chrome (macOS). We use this as our baseline for quality assurance.
- Document any issues in GitHub as a [bug report](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=bug%2C%3Areproduce&template=bug-report.md&title=), and assign for fixing.
- If in doubt about anything regarding design or layout, please reach out to the Design team.

### How to make usability changes to the website

We want to make it as easy as possible to learn how to manage devices with Fleet. Anyone inside or outside the company can suggest changes to the website to improve ease of use and clarity. 

To propose changes:
1. Decide what you want to change. A small change is the best place to start.
2. Wireframe the design. Usually, digital experience does this but anyone can contribute.
3. Present your change to the website DRI. They will approve it or suggest revisions.
4. Code the website change. Again, digital experience often does this but anyone can help.
5. Measure if the change made it easier to use. This can be tricky, but the growth team will have ideas on how to do this.

### How to export images for the website
In Figma:
1. Select the layers you want to export.
2. Confirm export settings and naming convention:
  * item name - color variant - (css)size - @2x.fileformat (e.g., `os-macos-black-16x16@2x.png`)
  * note that the dimensions in the filename are in CSS pixels.  In this example, the image would actually have dimensions of 32x32px, if you opened it in preview.  But in the filename, and in HTML/CSS, we'll size it as if it were 16x16.  This is so that we support retina displays by default.
  * File extension might be .jpg or .png.
  * Avoid using SVGs or icon fonts.
3. Click the __Export__ button.

<meta name="maintainedBy" value="mike-j-thomas">


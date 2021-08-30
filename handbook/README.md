# Fleet handbook

The Fleet company handbook is the living knowledge base describing how we do things at Fleet Device Management, Inc.  Every Fleet team member uses and contributes to the handbook.  It is open to the world, and we welcome feedback.  Please make a pull request to suggest improvements or add clarifications.  Use [issues](https://github.com/fleetdm/fleet/issues) to ask questions.

## Company

### About Fleet

Fleet Device Management Inc is an open core company that sells subscriptions that offer more features and support for Fleet.

We are dedicated to making Fleet the best management platform for [osquery](https://osquery.io), the leading open source endpoint agent.

#### History

##### 2014: Origins of osquery
In 2014, our CTO Zach Wasserman, together with [Mike Arpaia](https://twitter.com/mikearpaia/status/1357455391588839424) and the rest of their team at Facebook, created an open source project called [osquery](https://osquery.io).

##### 2016: Origins of Fleet v1.0
A few years later, Zach, Mike Arpaia, and [Jason Meller](https://honest.security) founded [Kolide](https://kolide.com) and created Fleet: an open source platform that made it easier and more productive to use osquery in an enterprise setting.

##### 2019: The growing community
When Kolide's attention shifted away from Fleet and towards their separate, user-focused SaaS offering, the Fleet community took over maintenance of the open source project. After his time at Kolide, Zach continued as lead maintainer of Fleet.  He spent 2019 consulting and working with the growing open source community to support and extend the capabilities of the Fleet platform.

##### 2020: Fleet was incorporated
Zach partnered with our CEO, Mike McNeil, to found a new, independent company: Fleet Device Management Inc.  In November 2020, we [announced](https://medium.com/fleetdm/a-new-fleet-d4096c7de978) the transition and kicked off the logistics of moving the GitHub repository.


### Culture

##### All remote
Fleet Device Management Inc. is an all-remote company, with team members spread across 3 continents and 5 time zones.  The wider team of contributors from [all over the world](https://github.com/fleetdm/fleet/graphs/contributors) submit patches, bug reports, troubleshooting tips, improvements, and real-world insights to Fleet's open source code base, documentation, website, and company handbook.

##### Openness
The majority of the code, documentation, and content we create at Fleet is public and source-available, and we strive to be broadly open and transparent in the way we run the business; as much as confidentiality agreements (and time) allow.  We perform better with an audience, and our audience performs better with us.

##### Spending company money
As we continue to expand our own company policies, we use [GitLab's open expense policy](https://about.gitlab.com/handbook/spending-company-money/) as a guide for company spending.

In brief, this means that as a Fleet team member, you may:

* Spend company money like it is your own money.
* Be responsible for what you need to purchase or expense in order to do your job effectively.
* Feel free to make purchases __in the interest of the company__ without asking for permission beforehand (when in doubt, do __inform__ your manager prior to purchase, or as soon as possible after the purchase).

For more developed thoughts about __spending guidelines and limits__, please read [GitLab's open expense policy](https://about.gitlab.com/handbook/spending-company-money/).

##### Meetings

* At Fleet, meetings start whether you're there or not. Nevertheless, being even a few minutes late can make a big difference and slow your meeting counterparts down. When in doubt, show up a couple of minutes early.
* It's okay to spend the first minute or two of a meeting to be present and make small talk, if you want.  Being all-remote, it's easy to miss out on hallway chatter and human connections that happen in [meatspace](https://www.dictionary.com/browse/meatspace).  Why not use this time together during the first minute to say "hi".  Then you can jump right in to the topics being discussed?
* Turning on your camera allows for more complete and intuitive verbal and non-verbal communication.  When joining meetings with new participants who you might not be familiar with yet, feel free to leave your camera on or to turn it off.  When you lead or cohost a meeting, turn your camera on.

### Fleet EE

##### Communicating design changes to Engineering
For something NEW that has been added to [Figma Fleet EE (current, dev-ready)](https://www.figma.com/file/qpdty1e2n22uZntKUZKEJl/?node-id=0%3A1):
1. Create a new [GitHub issue](https://github.com/fleetdm/fleet/issues/new)
2. Detail the required changes (including page links to the relevant layouts), then assign it to the __“Initiatives”__ project.

<img src="https://user-images.githubusercontent.com/78363703/129840932-67d55b5b-8e0e-4fb9-9300-5d458e1b91e4.png" alt="Assign to Initiatives project" width="300"/>

> ___NOTE:__ Artwork and layouts in Figma Fleet EE (current, dev-ready) are final assets, ready for implementation. Therefore, it’s important NOT to use the “idea” label, as designs in this document are more than ideas - they are something that WILL be implemented._

3. Navigate to the [Initiatives project](https://github.com/orgs/fleetdm/projects/8), and hit “+ Add cards”, pick the new issue, and drag it into the “🤩Inspire me” column. 

<img src="https://user-images.githubusercontent.com/78363703/129840496-54ea4301-be20-46c2-9138-b70bff7198d0.png" alt="Add cards" width="600"/>

<img src="https://user-images.githubusercontent.com/78363703/129840735-3b270429-a92a-476d-87b4-86b93057b2dd.png" alt="Inspire me" width="300"/>

##### Communicating unplanned design changes

For issues related to something that was ALREADY in Figma Fleet EE (current, dev-ready), but __implemented differently__, e.g, padding/spacing inconsistency etc. Create a [bug issue](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=bug%2C%3Areproduce&template=bug-report.md&title=) and detail the required changes.


### Fleet website

##### How to export images
In Figma:
1. Select the layers you want to export.
2. Confirm export settings and naming convention:
  * item name - color variant - (css)size - @2x.fileformat (e.g., `os-macos-black-16x16@2x.png`)
  * note that the dimensions in the filename are in CSS pixels.  In this example, the image would actually have dimensions of 32x32px, if you opened it in preview.  But in the filename, and in HTML/CSS, we'll size it as if it were 16x16.  This is so that we support retina displays by default.
  * File extension might be .jpg or .png.
  * Avoid using SVGs or icon fonts.
3. Click the __Export__ button.

##### When can I merge a change to the website?
When merging a PR to master, bear in mind that whatever you merge to master gets deployed live immediately. So if the PR's changes contain anything that you don't think is appropriate to be seen publicly by all guests of [fleetdm.com](https://fleetdm.com/), then please do not merge.

Merge a PR (aka deploy the website) when you think it is appropriately clean to represent our brand. When in doubt, use the standards and level of quality seen on existing pages, ensure correct functionality, and check responsive behavior - starting widescreen and resizing down to ≈320px width. 

##### The "Deploy Fleet Website" GitHub action failed
If the action fails, please complete the following steps:
1. Head to the fleetdm-website app in the [Heroku dashboard](https://heroku.com) and select the "Activity" tab.
2. Select "Roll back to here" on the second to most recent deploy.
3. Head to the fleetdm/fleet GitHub repository and re-run the Deploy Fleet Website action.


##### Browser compatibility checking

A browser compatibility check of [fleetdm.com](https://fleetdm.com/) should be carried out monthly to verify that the website looks, and functions as expected across all [supported browsers](../docs/1-Using-Fleet/12-Supported-browsers.md).

- We use [BrowserStack](https://www.browserstack.com/users/sign_in) (logins can be found in [1Password](https://start.1password.com/open/i?a=N3F7LHAKQ5G3JPFPX234EC4ZDQ&v=3ycqkai6naxhqsylmsos6vairu&i=nwnxrrbpcwkuzaazh3rywzoh6e&h=fleetdevicemanagement.1password.com)) for our cross-browser checks.
- Check for issues against the latest version of Google Chrome (macOS). We use this as our baseline for quality assurance.
- Document any issues in GitHub as a [bug report](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=bug%2C%3Areproduce&template=bug-report.md&title=), and assign for fixing.
- If in doubt about anything regarding design or layout, please reach out to the Design team.

### Fleet docs

#### Adding a link to Fleet docs
The `fleetdm.com/docs` compilation process does not account for relative links to directories outside of `/docs`.
Therefore, when adding a link to Fleet docs, it is important to always use the absolute file path.

#### Linking to a location on GitHub
When adding a link to a location on GitHub that is outside of `/docs`, Be sure to use the canonical form of the URL.

To do this, navigate to the file's location on GitHub, and press "y" to transform the URL into its cononical form.

#### How to fix a broken link
For instances in which a broken link is discovered on fleetdm.com, check if the link is a relative link to a directory outside of `/docs`. 

An example of a link that lives outside of `/docs` is:

```
../../tools/app/prometheus
```

If the link lives outside `/docs`, head to the file's location on GitHub (in this case, [https://github.com/fleetdm/fleet/blob/main/tools/app/prometheus.yml)](https://github.com/fleetdm/fleet/blob/main/tools/app/prometheus.yml)), and press "y" to transform the URL into its canonical form ([https://github.com/fleetdm/fleet/blob/194ad5963b0d55bdf976aa93f3de6cabd590c97a/tools/app/prometheus.yml](https://github.com/fleetdm/fleet/blob/194ad5963b0d55bdf976aa93f3de6cabd590c97a/tools/app/prometheus.yml)). Replace the relative link with this link in the markdown file.

### About the handbook
#### Why bother?
The Fleet handbook is inspired by (and heavily influenced by) the [GitLab team handbook](https://about.gitlab.com/handbook/about/).  It shares the same [advantages](https://about.gitlab.com/handbook/about/#advantages) and will probably undergo a similar [evolution](https://about.gitlab.com/handbook/ceo/#evolution-of-the-handbook).

#### Where's the rest of the handbook?
While this handbook is inspired by [GitLab's handbook](https://about.gitlab.com/handbook/), it is nowhere near as complete (yet!)  We will continue to add and update information in this handbook, and gradually migrate information from [Fleet's shared Google Drive folder](https://drive.google.com/drive/u/0/folders/1StSOI3HNcsl9VleXxNWfUBT2co7h44OG) as time allows.

## Acknowledgements
This work, "Fleet Handbook", is licensed under CC BY-SA 4.0 by Fleet Device Management Inc.  It is, in part, a derivative of ["GitLab Handbook"](https://about.gitlab.com/handbook/), by [GitLab](https://about.gitlab.com/company/), used under [CC BY-SA 4.0](https://gitlab.com/gitlab-com/www-gitlab-com/-/blob/96c14468bbd29236dc1c3556bdf9514d966ca3d1/source/includes/cc-license.html.haml).


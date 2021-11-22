## Product DRIs

Below is a table of [directly responsible individuals (DRIs)](./people.md#directly-resonsible-individuals) for aspects of the Fleet product:

|    Aspect              										| DRI     		|
| ------------------------------------------------------------- | ------------- |
| Wireframes (figma)	 										| Noah Talerman	|
| How the product works 										| Noah Talerman |
| fleetctl CLI interface (and other tools) 						| Tomás Touceda |
| REST API interface, REST API docs 							| Luke Heath	|
| Postman 											| Ben Edwards (transitioning to Luke Heath) 	|
| Terraform 											| Ben Edwards 	|
| Customer deployments like acme.fleetdm.com 				| Ben Edwards 	|
| dogfood.fleetdm.com 											| Ben Edwards  	|
| Quality of core product UI 									| Luke Heath 	|
| Quality of tickets after Noah's done with them   				| Luke Heath 	|
| Quality of core product API 									| Tomás Touceda |
| Quality of fleetctl (and other tools)							| Tomás Touceda |
| Final cut of what goes into each release 						| Zach Wasserman|
| When we cut a release, version numbers, and whether to release| Zach Wasserman|
| Release notes 												| Noah Talerman |
| Publishing release blog post, and promoting releases 			| Mike Thomas  	|

## Feature flags
In Fleet, features are placed behind feature flags if the changes could affect Fleet's availability of existing functionalities.

The following highlights should be considered when deciding if feature flags should be leveraged:

- The feature flag must be disabled by default.
- The feature flag will not be permanent. This means that the individual who decides that a feature flag should be introduced is responsible for creating an issue to track the feature's progress towards removing the feature flag and including the feature in a stable release.
- The feature flag will not be advertised. For example, advertising in the documentation on fleetdm.com/docs, release notes, release blog posts, and Twitter.

Fleet's feature flag guidelines borrows from GitLab's ["When to use feature flags" section](https://about.gitlab.com/handbook/product-development-flow/feature-flag-lifecycle/#when-to-use-feature-flags) of their handbook. Check out [GitLab's "Feature flags only when needed" video](https://www.youtube.com/watch?v=DQaGqyolOd8) for an explanation on the costs of introducing feature flags.

## Fleet docs

### Adding a link to the Fleet docs
You can link documentation pages to each other using relative paths. For example, in `docs/01-Using-Fleet/01-Fleet-UI.md`, you can link to `docs/01-Using-Fleet/09-Permissions.md` by writing `[permissions](./09-Permissions.md)`. This will be automatically transformed into the appropriate URL for `fleetdm.com/docs`.

However, the `fleetdm.com/docs` compilation process does not account for relative links to directories **outside** of `/docs`.
Therefore, when adding a link to Fleet docs, it is important to always use the absolute file path.

When directly linking to a specific section within a page in the Fleet documentation, always format the spaces within a section name to use a hyphen `-` instead of an underscore `_`. For example, when linking to the `osquery_result_log_plugin` section of the configuration reference docs, use a relative link like the following: `./02-Configuration.md#osquery-result-log-plugin`.

### Linking to a location on GitHub
When adding a link to a location on GitHub that is outside of `/docs`, be sure to use the canonical form of the URL.

To do this, navigate to the file's location on GitHub, and press "y" to transform the URL into its canonical form.

### How to fix a broken link
For instances in which a broken link is discovered on fleetdm.com, check if the link is a relative link to a directory outside of `/docs`. 

An example of a link that lives outside of `/docs` is:

```
../../tools/app/prometheus
```

If the link lives outside `/docs`, head to the file's location on GitHub (in this case, [https://github.com/fleetdm/fleet/blob/main/tools/app/prometheus.yml)](https://github.com/fleetdm/fleet/blob/main/tools/app/prometheus.yml)), and press "y" to transform the URL into its canonical form ([https://github.com/fleetdm/fleet/blob/194ad5963b0d55bdf976aa93f3de6cabd590c97a/tools/app/prometheus.yml](https://github.com/fleetdm/fleet/blob/194ad5963b0d55bdf976aa93f3de6cabd590c97a/tools/app/prometheus.yml)). Replace the relative link with this link in the markdown file.

> Note that the instructions above also apply to adding links in the Fleet handbook.

### Adding an image to the Fleet docs
Try to keep images in the docs at a minimum. Images can be a quick way to help a user understand a concept or direct them towards a specific UI element, but too many can make the documentation feel cluttered and more difficult to maintain.

When adding images to the Fleet documentation, follow these guidelines:
- Keep the images as simple as possible to maintain (screenshots can get out of date quickly as UIs change)
- Exclude unnecessary images. An image should be used to help emphasize information in the docs, not replace it.
- Minimize images per doc page. More than one or two per page can get overwhelming, for doc maintainers and users.
- The goal is for the docs to look good on every form factor, from 320px window width all the way up to infinity and beyond. Full window screenshots and images with too much padding on the sides will be less than the width of the user's screen. When adding a large image, make sure that it is easily readable at all widths.

Images can be added to the docs using the Markdown image link format, e.g. `![Schedule Query Sidebar](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/schedule-query-sidebar.png)`
The images used in the docs live in `docs/images/`. Note that you must provide the url of the image in the Fleet Github repo for it to display properly on both Github and the Fleet website.

> Note that the instructions above also apply to adding images in the Fleet handbook.

## Manual QA

This living document outlines the manual quality assurance process conducted to ensure each release of Fleet meets organization standards.

All steps should be conducted during each QA pass. All steps are possible with `fleetctl preview`. In order to target a specific version of `fleetctl preview`, the tag argument can be used together with the commit you are targeting as long as that commit is represented by a tag in [docker hub](https://hub.docker.com/r/fleetdm/fleet/tags?page=1&ordering=last_updated). Without tag argument, `fleetctl preview` defaults to latest stable.

As new features are added to Fleet, new steps and flows will be added.

### Collecting bugs

The goal of manual QA is to catch unexpected behavior prior to release. All Manual QA steps should be possible using `fleetctl preview`. Please refer to [docs/03-Contributing/02-Testing.md](https://github.com/fleetdm/fleet/blob/main/docs/03-Contributing/02-Testing.md) for flows that cannot be completed using `fleetctl preview`.

Please start the manual QA process by creating a blank GitHub issue. As you complete each of the flows, record a list of the bugs you encounter in this new issue. Each item in this list should contain one sentence describing the bug and a screenshot if the item is a frontend bug.

### Fleet UI

For all following flows, please refer to the [permissions documentation](https://fleetdm.com/docs/using-fleet/permissions) to ensure that actions are limited to the appropriate user type. Any users with access beyond what this document lists as availale should be considered a bug and reported for either documentation updates or investigation.

#### Set up flow

Successfully set up `fleetctl preview` using the preview steps outlined [here](https://fleetdm.com/get-started)

#### Login and logout flow

Successfully logout and then login to your local Fleet.

#### Host details page

Select a host from the "Hosts" table as a global user with the Maintainer role. You may create a user with a fake email for this purpose.

You should be able to see and select the "Delete" button on this host's **Host details** page.

You should be able to see and select the "Query" button on this host's **Host details** page.

#### Label flow

`Flow is covered by e2e testing`

Create a new label by selecting "Add a new label" on the Hosts page. Make sure it correctly filters the host on the hosts page.

Edit this label. Confirm users can only edit the "Name" and "Description" fields for a label. Users cannot edit the "Query" field because label queries are immutable.

Delete this label.

#### Query flow

`Flow is covered by e2e testing`

Create a new saved query.

Run this query as a live query against your local machine.

Edit this query and then delete this query.

#### Pack flow

`Flow is covered by e2e testing`

Create a new pack (under Schedule/advanced).

Add a query as a saved query to the pack. Remove this query. Delete the pack.


#### My account flow

Head to the My account page by selecting the dropdown icon next to your avatar in the top navigation. Select "My account" and successfully update your password. Please do this with an extra user created for this purpose to maintain accessibility of `fleetctl preview` admin user.


### fleetctl CLI

#### Set up flow

Successfully set up Fleet by running the `fleetctl setup` command.

You may have to wipe your local MySQL database in order to successfully set up Fleet. Check out the [Clear your local MySQL database](#clear-your-local-mysql-database) section of this document for instructions.

#### Login and logout flow

Successfully login by running the `fleetctl login` command.

Successfully logout by running the `fleetctl logout` command. Then, log in again.

#### Hosts

Run the `fleetctl get hosts` command.

You should see your local machine returned. If your host isn't showing up, you may have to reenroll your local machine. Check out the [Orbit for osquery documentation](https://github.com/fleetdm/fleet/blob/main/orbit/README.md) for instructions on generating and installing an Orbit package.

#### Query flow

Apply the standard query library by running the following command:

`fleetctl apply -f docs/01-Using-Fleet/standard-query-library/standard-query-library.yml`

Make sure all queries were successfully added by running the following command:

`fleetctl get queries`

Run the "Get the version of the resident operating system" query against your local machine by running the following command:

`fleetctl query --hosts <your-local-machine-here> --query-name "Get the version of the resident operating system"`

#### Pack flow

Apply a pack by running the following commands:

`fleetctl apply -f docs/01-Using-Fleet/configuration-files/multi-file-configuration/queries.yml`

`fleetctl apply -f docs/01-Using-Fleet/configuration-files/multi-file-configuration/pack.yml`

Make sure the pack was successfully added by running the following command:

`fleetctl get packs`

#### Organization settings flow

Apply organization settings by running the following command:

`fleetctl apply -f docs/01-Using-Fleet/configuration-files/multi-file-configuration/organization-settings.yml`

#### Manage users flow

Create a new user by running the `fleetctl user create` command.

Logout of your current user and log in with the newly created user.


## Release process

This section outlines the release process at Fleet.

The current release cadence is once every 3 weeks and concentrated around Wednesdays. 

- [Blog post](#release-day)

### Blog post

Fleet posts a release blogpost, to the [Fleet blog](https://blog.fleetdm.com/ ), on the same day a new minor or major release goes out.

Patch releases do not have a release blogpost.

Check out the [Fleet 4.1.0 blog post](https://blog.fleetdm.com/fleet-4-1-0-57dfa25e89c1) for an example release blogpost. The suggested format of a release blogpost is the following:

**Title** - "Fleet `<insert Fleet version here>`

**Description** - "Fleet `<insert Fleet version here>` released with `<insert list of primary features here>`

**Main image** - This is the image that Medium will use as the thumbnail and link preview for the blogpost.

**Summary** - This section includes 3-4 sentences that answers the 'what?' and 'why should the user care?' questions for the primary features.

**Link to release notes** - One sentence that includes a link to the GitHub release page.

**Primary features** - Includes the each primary feature's name, availability (Free v. Premium), image/gif, and 3-4 sentences that answer the 'why should the user care?' and 'how do I find this feature?' questions.

**More improvements** - Includes each additional feature's name, availability (Free v. Premium), and 1-2 sentences that answer the 'why should the user care?' questions.

**Upgrade plan** - Once sentence that links to user to the upgrading Fleet documentation here: https://github.com/fleetdm/fleet/blob/main/docs/01-Using-Fleet/08-Updating-Fleet.md

#### Manual QA

After all changes required for release have been merged into the `main` branch, the individual tasked with managing the release should perform smoke tests. Manual smoke tests should be generated for a release via the [Release QA ticket template](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=&template=smoke-tests.md&title=) and assigned to the person responsible. 

Further ocumentation on conducting the manual QA pass can be found [here](#manual-qa). 

#### Release freeze period

In order to ensure quality releases, Fleet has a freeze period for testing prior to each release. Effective at the start of the freeze period, new feature work will not be merged. 

Release blocking bugs are exempt from the freeze period and are defined by the same rules as patch releases, which include:
1. Regressions
2. Security concerns
3. Issues with features targeted for current release

Non-release blocking bugs may include known issues that were not targeted for the current release, or newly documented behaviors that reproduce in older stable versions. These may be addressed during a release period by mutual agreement between Product and Engineering teams. 


### Release day

Documentation on completing the release process can be found [here](../docs/03-Contributing/05-Releasing-Fleet.md).  


## Support process

This section outlines the customer and community support process at Fleet.

The support process is accomplished via an on-call rotation and the weekly on-call retro meeting.

The on-call engineer is responsible for responding to Slack comments, Slack threads, and GitHub issues raised by customers and the community.

The weekly on-call retro at Fleet provides time to discuss highlights and answer the following questions about the previous week's on-call:

1. What went well?

2. What could have gone better?

3. What should we remember next time?

This way, the Fleet team can constantly improve the effectiveness and experience during future on-call rotations.

### Goals
At Fleet, our primary quality objectives are *customer service* and *defect reduction*. This entails Key Performance Indicators such as customer response time and number of bugs resolved per cycle and. 

- Get familiar with and stay abreast of what our community wants and the problems they're having.

- Make people feel heard and understood.  

- Celebrate contributions. 

- Triage bugs, identify community feature requests, community pull requests and community questions.

### Version support

In order to provide the most accurate and efficient support, Fleet will only target fixes based on the latest released version. Fixes in current versions will not be backported to older releases.

Community version supported for bug fixes: **Latest version only**
 
Community support for support/troubleshooting: **Current major version**

Premium version supported for bug fixes: **Latest version only**

Premium support for support/troubleshooting: **All versions**

### How?

- No matter what, folks who post a new comment in Slack or issue in GitHub get a **response** from the on-call engineer within 1 business day. The response doesn't need to include an immediate answer.

- The on-call engineer can discuss any items that require assistance at the end of the daily standup. They are also requested to attend the "Customer experience standup" where they can bring questions and stay abreast of what's happening with our customers.

- If you do not understand the question or comment raised, [request more details](#requesting-more-details) to best understand the next steps.

- If an appropriate response is outside your scope, please post to `#help-oncall`, a confidential Slack channel in the Fleet Slack workspace.

- If things get heated, remember to stay [positive and helpful](https://canny.io/blog/moderating-user-comments-diplomatically/).  If you aren't sure how best to respond in a positive way, or if you see behavior that violates the Fleet code of conduct, get help.

#### Requesting more details

Typically, the *questions*, *bug reports*, and *feature requests* raised by members of the community will be missing helpful context, recreation steps, or motivations respectively.

❓ For questions that you don't immediately know the answer to, it's helpful to ask follow up questions to receive additional context. 

- Let's say a community member asks the question "How do I do X in Fleet?" A follow question could be "what are you attempting to accomplish by doing X?" 
- This way, you now has additional details when the primary question is brought to the Roundup meeting. In addition, the community member receives a response and feels heard.

🦟 For bug reports, it's helpful to ask for recreation steps so you're later able to verify the bug exists.

- Let's say a community member submits a bug report. An example follow up question could be "Can you please walk me through how you encountered this issue so that I can attempt to recreate it?"
- This way, you now have steps the verify whether the bug exists in Fleet or if the issue is specific to the community member's environment. If the latter, you now have additional information for further investigation and question asking.

💡 For feature requests, it's helpful to ask follow up questions in an attempt to understand the "why?" or underlying motivation behind the request.

- Let's say a community member submits the feature request "I want the ability to do X in Fleet." A follow up question could be "If you were able to do X in Fleet, what's the next action you would take?" or "Why do you want to do X in Fleet?." 
- Both of these questions provide helpful context on the underlying motivation behind the feature request when it is brought to the Roundup meeting. In addition, the community member receives a response and feels heard.

#### Feature requests

If the feature is requested by a customer, the on-call engineer is requested to create a feature request issue and follow up with the customer by linking them to this issue. This way, the customer can add additional comments or feedback to the newly filed feature request issue.

If the feature is requested by anyone other than a customer (ex. user in #fleet Slack), the on-call engineer is requested to point to the user to the [feature request GitHub issue template](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=idea&template=feature-request.md&title=) and kindly ask the user to create a feature request.

#### Closing issues

It is often a good idea to let the original poster (OP) close their issue themselves, since they are usually well equipped to decide whether the issue is resolved.   In some cases, circling back with the OP can be impractical, and for the sake of speed issues might get closed.

Keep in mind that this can feel jarring to the OP.  The effect is worse if issues are closed automatically by a bot (See [balderashy/sails#3423](https://github.com/balderdashy/sails/issues/3423#issuecomment-169751072) and [balderdashy/sails#4057](https://github.com/balderdashy/sails/issues/4057) for examples of this.)

To provide another way of tracking status without closing issues altogether, consider using the green labels that begin with "+".  To explore them, type `+` from GitHub's label picker.


### Sources

There are four sources that the on-call engineer should monitor for activity:

1. Customer Slack channels - Found under the "Connections" section in Slack. These channels are usually titled "at-insert-customer-name-here"

2. Community chatroom - https://osquery.slack.com, #fleet channel

3. Reported bugs - [GitHub issues with the "bug" and ":reproduce" label](https://github.com/fleetdm/fleet/issues?q=is%3Aopen+is%3Aissue+label%3Abug+label%3A%3Areproduce). Please remove the ":reproduce" label after you've followed up in the issue.

4. Pull requests opened by the community - [GitHub open pull requests](https://github.com/fleetdm/fleet/pulls?q=is%3Aopen+is%3Apr)

### Resources

There are several locations in Fleet's public and internal documentation that can be helpful when answering questions raised by the community:

1. The frequently asked question (FAQ) documents in each section found in the `/docs` folder. These documents are the [Using Fleet FAQ](../docs/01-Using-Fleet/FAQ.md), [Deploying FAQ](../docs/02-Deploying/FAQ.md), and [Contributing FAQ](../docs/03-Contributing/FAQ.md).

2. The [Internal FAQ](https://docs.google.com/document/d/1I6pJ3vz0EE-qE13VmpE2G3gd5zA1m3bb_u8Q2G3Gmp0/edit#heading=h.ltavvjy511qv) document.


## UI Design

### Communicating design changes to Engineering
For something NEW that has been added to [Figma Fleet EE (current, dev-ready)](https://www.figma.com/file/qpdty1e2n22uZntKUZKEJl/?node-id=0%3A1):
1. Create a new [GitHub issue](https://github.com/fleetdm/fleet/issues/new)
2. Detail the required changes (including page links to the relevant layouts), then assign it to the __“Initiatives”__ project.

<img src="https://user-images.githubusercontent.com/78363703/129840932-67d55b5b-8e0e-4fb9-9300-5d458e1b91e4.png" alt="Assign to Initiatives project"/>

> ___NOTE:___ Artwork and layouts in Figma Fleet EE (current, dev-ready) are final assets, ready for implementation. Therefore, it’s important NOT to use the “idea” label, as designs in this document are more than ideas - they are something that WILL be implemented._

3. Navigate to the [Initiatives project](https://github.com/orgs/fleetdm/projects/8), and hit “+ Add cards”, pick the new issue, and drag it into the “🤩Inspire me” column. 

<img src="https://user-images.githubusercontent.com/78363703/129840496-54ea4301-be20-46c2-9138-b70bff7198d0.png" alt="Add cards"/>

<img src="https://user-images.githubusercontent.com/78363703/129840735-3b270429-a92a-476d-87b4-86b93057b2dd.png" alt="Inspire me"/>

### Communicating unplanned design changes

For issues related to something that was ALREADY in Figma Fleet EE (current, dev-ready), but __implemented differently__, e.g, padding/spacing inconsistency etc. Create a [bug issue](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=bug%2C%3Areproduce&template=bug-report.md&title=) and detail the required changes.

### Design conventions

We have certain design conventions that we include in Fleet. We will document more of these over time.

**Table empty states**

Use `---`, with color `$ui-fleet-black-50` as the default UI for empty columns.


<meta name="maintainedBy" value="noahtalerman">


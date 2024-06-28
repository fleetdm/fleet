# Engineering
This handbook page details processes specific to working [with](#contact-us) and [within](#responsibilities) this department.

## Team
| Role                            | Contributor(s)           |
|:--------------------------------|:-----------------------------------------------------------------------------------------------------------|
| Chief Technology Officer (CTO)  | [Luke Heath](https://www.linkedin.com/in/lukeheath/) _([@lukeheath](https://github.com/lukeheath))_
| Client Platform Engineer &amp; Community Advocate     | [JD Strong](https://www.linkedin.com/in/jackdaniyelstrong/) _([@spokanemac](https://github.com/spokanemac/spokanemac))_
| Engineering Manager (EM)        | <sup><sub> _See [🛩️ Product groups](https://fleetdm.com/handbook/company/product-groups#current-product-groups)_ </sup></sub>
| Quality Assurance Engineer (QA) | <sup><sub> _See [🛩️ Product groups](https://fleetdm.com/handbook/company/product-groups#current-product-groups)_ </sup></sub>
| Software Engineer               | <sup><sub> _See [🛩️ Product groups](https://fleetdm.com/handbook/company/product-groups#current-product-groups)_ </sup></sub>


## Contact us
- To **make a request** of this department, [create an issue](https://fleetdm.com/handbook/company/product-groups#current-product-groups) and a team member will get back to you within one business day (If urgent, mention a [team member](#team) in the [#help-engineering](https://fleetdm.slack.com/archives/C019WG4GH0A) Slack channel.
  - Any Fleet team member can [view the kanban boards](https://fleetdm.com/handbook/company/product-groups#current-product-groups) for this department, including pending tasks and the status of new requests.
  - Please **use issue comments and GitHub mentions** to communicate follow-ups or answer questions related to your request.

## Responsibilities
The 🚀 Engineering department at Fleet is directly responsible for writing and maintaining the [code](https://github.com/fleetdm/fleet) for Fleet's core product and infrastructure.

### Record engineering KPIs
We track the success of this process by observing the throughput of issues through the system and identifying where buildups (and therefore bottlenecks) are occurring.
The metrics are:
* Number of bugs opened this week
* Total # bugs open
* Bugs in each state (inbox, acknowledged, reproduced)
* Number of bugs closed this week

Each week these are tracked and shared in the weekly KPI sheet by Luke Heath.

### Create an engineering-initiated story
Engineering-initiated stories are types of user stories created by engineers to make technical changes to Fleet. Technical changes should improve the user experience or contributor experience. For example, optimizing SQL that improves the response time of an API endpoint improves user experience by reducing latency. A script that generates common boilerplate, or automated tests to cover important business logic, improves the quality of life for contributors, making them happier and more productive, resulting in faster delivery of features to our customers.

It is important to frame engineering-initiated user stories the same way we frame all user stories. Stay focused on how this technical change will drive value for our users.

To [create an engineering-initiated user story](https://fleetdm.com/handbook/engineering#creating-an-engineering-initiated-story), follow the [user story drafting process](https://fleetdm.com/handbook/company/development-groups#drafting). Once your user story is created using the [new story template](https://github.com/fleetdm/fleet/issues/new?assignees=lukeheath&labels=story,~engineering-initiated&projects=&template=story.md&title=), make sure the `~engineering-initiated` label is added and the engineering output and architecture DRI (@lukeheath) is assigned. 

What happens next? The engineering output and architecture DRI reviews engineering-initiated stories weekly. 

If there are product changes (i.e. interface, documentation, or dependency changes), the story is added to the "New requests" column on the drafting board.

If there are no product changes, and the DRI decides to prioritize the story, the story is added to the "Specified" column on the drafting board so that it can be estimated.

> We prefer the term engineering-initiated stories over technical debt because the user story format helps keep us focused on our users and contributors.

### Fix a bug

All bug fix pull requests should have a mention back to the issue they resolve with #<ticket_number> in the description or even in a comment. Please do not use any [automated words](https://docs.github.com/en/issues/tracking-your-work-with-issues/linking-a-pull-request-to-an-issue#linking-a-pull-request-to-an-issue-using-a-keyword) since we don't want the tickets auto-closing when PR's are merged. 
If the bug is labeled `~unreleased bug`, branch off and put your PR into `main`. These issues can be closed as soon as they complete QA.

If the bug is labeled `~released bug`, branch off and put your PR into `main`. After merging checkout the latest tag, for example `git checkout fleet-v4.48.2`, then `git fetch; git cherry-pick <commit>`. If the cherry-pick fails with a conflict call out in the ticket how to resolve or if it is sufficiently complicated call out this fix is not suited for the patch release process and should only be included in the end of sprint release. This approach makes sure the bug fix is not built on top of unreleased feature code, which can cause merge conflicts during patch releases. 

### Begin a merge freeze
To ensure release quality, Fleet has a freeze period for testing beginning the Tuesday before the release at 11:00 AM Pacific. Effective at the start of the freeze period, new feature work will not be merged into `main`.

Bugs are exempt from the release freeze period.

To begin the freeze, [open the repo on Merge Freeze](https://www.mergefreeze.com/installations/3704/branches/6847) and click the "Freeze now" button. This will freeze the `main` branch and require any PRs to be manually unfrozen before merging. PRs can be manually unfrozen in Merge Freeze using the PR number.

> Any Fleetie can [unfreeze PRs on Merge Freeze](https://www.mergefreeze.com/installations/3704/branches) if the PR contains documentation changes or bug fixes only. If the PR contains other changes, please confirm with your manager before unfreezing.

### Merge a pull request during the freeze period
We merge bug fixes, documentation changes, and website updates during the freeze period, but we do not merge other code changes. This minimizes code churn and helps ensure a stable release. To merge a bug fix, you must first unfreeze the PR in [Merge Freeze](https://app.mergefreeze.com/installations/3704/branches), and click the "Unfreeze 1 pull request" text link. 

> To allow a stable release test, the final 24 hours before release is a deep freeze when only bugs with the `~release-blocker` or `~unreleased-bug` labels are merged.

If there is partially merged feature work when freeze begins, the previously merged code must be reverted. If there is an exceptional, business-critical need to merge feature work during freeze, as determined by the [release ritual DRI](#rituals), the following exception process may be followed:

1. The engineer requesting the feature work merge exception during freeze notifies their Engineering Manager. 
2. The Engineering Manager notifies the QA lead for the product group and the [release ritual DRI](https://fleetdm.com/handbook/engineering#rituals). 
3. The Engineering Manager, QA lead, and [release ritual DRI](#rituals) must all approve the feature work PR before it is unfrozen and merged.

> This exception process should be avoided whenever possible. Any feature work merged during freeze will likely result in a significant release delay.

### Confirm latest versions of dependencies
Before kicking off release QA, confirm that we are using the latest versions of dependencies we want to keep up-to-date with each release. Currently, those dependencies are:

1. **Go**: Latest minor release
- Check the [version included in Fleet](https://github.com/fleetdm/fleet/settings/variables/actions).
- Check the [latest minor version of Go](https://go.dev/dl/). For example, if we are using `go1.19.8`, and there is a new minor version `go1.19.9`, we will upgrade.
- If the latest minor version is greater than the version included in Fleet, [file a bug](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=bug%2C%3Areproduce&projects=&template=bug-report.md&title=) and assign it to the [release ritual DRI](https://fleetdm.com/handbook/engineering#rituals) and the [current oncall engineer](https://fleetdm.com/handbook/engineering#how-to-reach-the-oncall-engineer). Add the `~release blocker` label. We must upgrade to the latest minor version before publishing the next release.
- If the latest major version is greater than the version included in Fleet, [create a story](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=story%2C%3Aproduct&projects=&template=story.md&title=) and assign it to the [release ritual DRI](https://fleetdm.com/handbook/engineering#rituals) and the [current oncall engineer](https://fleetdm.com/handbook/engineering#how-to-reach-the-oncall-engineer). This will be considered for an upcoming sprint. The release can proceed without upgrading the major version.
- Note that major version upgrades also require an [update to go.mod](https://github.com/fleetdm/fleet/blob/7b3134498873a31ba748ca27fabb0059cef70db9/go.mod#L3). 

> In Go versioning, the number after the first dot is the "major" version, while the number after the second dot is the "minor" version. For example, in Go 1.19.9, "19" is the major version and "9" is the minor version. Major version upgrades are assessed separately by engineering.

2. **macadmins-extension**: Latest release
- Check the [latest version of the macadmins-extension](https://github.com/macadmins/osquery-extension/releases).
- Check the [version included in Fleet](https://github.com/fleetdm/fleet/blob/main/go.mod#L60).
- If the latest stable version of the macadmins-extension is greater than the version included in Fleet, [file a bug](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=bug%2C%3Areproduce&projects=&template=bug-report.md&title=) and assign it to the [release ritual DRI](https://fleetdm.com/handbook/engineering#rituals) and the [current on-call engineer](https://fleetdm.com/handbook/engineering#how-to-reach-the-oncall-engineer).
- Add the `~release blocker` label.

>**Note:** Some new versions of the macadmins-extension include updates that require code changes in Fleet. Make sure to note in the bug that the update should be checked for any changes, like new tables, that require code changes in Fleet.

Our goal is to keep these dependencies up-to-date with each release of Fleet. If a release is going out with an old dependency version, it should be treated as a [critical bug](https://fleetdm.com/handbook/engineering#critical-bugs) to make sure it is updated before the release is published.

3. **osquery**: Latest release
- Check the [latest version of osquery](https://github.com/osquery/osquery/releases).
- Check the [version included in Fleet](https://github.com/fleetdm/fleet/blob/ceb4e4602ba9a90ebf0e33e1eddef770c9a8d8b5/.github/workflows/generate-osqueryd-targets.yml#L27).
- If the latest release of osquery is greater than the version included in Fleet, [file a bug](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=bug%2C%3Areproduce&projects=&template=bug-report.md&title=) and assign it to the [release ritual DRI](https://fleetdm.com/handbook/engineering#rituals) and the [current on-call engineer](https://fleetdm.com/handbook/engineering#how-to-reach-the-oncall-engineer).
- Do not add the `~release blocker` label. 
- Update the bug description to note that changes to [osquery command-line flags](https://osquery.readthedocs.io/en/stable/installation/cli-flags/) require updates to Fleet's flag validation and related documentation [as shown in this pull request](https://github.com/fleetdm/fleet/pull/16239/files). 

4. Vulnerability data sources
- Check the [NIST National Vulnerability Database website](https://nvd.nist.gov/) for any announcements that might impact our [NVD data feed](https://github.com/fleetdm/fleet/blob/5e22f1fb4647a6a387ca29db6dd75d492f1864d6/cmd/cpe/generate.go#L53). 
- Check the [CISA website](https://www.cisa.gov/) for any news or announcements that might impact our [CISA data feed](https://github.com/fleetdm/fleet/blob/5e22f1fb4647a6a387ca29db6dd75d492f1864d6/server/vulnerabilities/nvd/sync.go#L137). 

If an announcement is found for either data source that may impact data feed availability, notify the current [on-call engineer](https://fleetdm.com/handbook/engineering#how-to-reach-the-oncall-engineer). Notify them that it is their responsibility to investigate and file a bug or take further action as necessary. 

5. [Fleetd](https://fleetdm.com/docs/get-started/anatomy#fleetd) components
- Check for code changes to [Orbit](https://github.com/fleetdm/fleet/blob/main/orbit/) or [Desktop](https://github.com/fleetdm/fleet/tree/main/orbit/cmd/desktop) since the last `orbit-*` tag was published.
- Check for code changes to the [fleetd-chrome extension](https://github.com/fleetdm/fleet/tree/main/ee/fleetd-chrome) since the last `fleetd-chrome-*` tag was published.

If code changes are found for any `fleetd` components, create a new release QA issue to update `fleetd`. Delete the top section for Fleet core, and retain the bottom section for `fleetd`. Populate the necessary version changes for each `fleetd` component.

### Indicate your product group is release-ready
Once a product group completes its QA process during the freeze period, its QA lead moves the smoke testing ticket to the "Ready for release" column on their ZenHub board. They then notify the release ritual DRI by tagging them in a comment, indicating that their group is prepared for release. The release ritual DRI starts the [release process](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/Releasing-Fleet.md) after all QA leads have made these updates and confirmed their readiness for release.

### Prepare Fleet release
Documentation on completing the release process can be found [here](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/Releasing-Fleet.md).

### Deploy a new release to dogfood
After each Fleet release, the new release is deployed to Fleet's "dogfood" (internal) instance.

How to deploy a new release to dogfood:

1. Head to the **Tags** page on the fleetdm/fleet Docker Hub: https://hub.docker.com/r/fleetdm/fleet/tags
2. In the **Filter tags** search bar, type in the latest release (ex. v4.19.0).
3. Locate the tag for the new release and copy the image name. An example image name is "fleetdm/fleet:v4.19.0".
4. Head to the "Deploy Dogfood Environment" action on GitHub: https://github.com/fleetdm/fleet/actions/workflows/dogfood-deploy.yml
5. Select **Run workflow** and paste the image name in the **The image tag wished to be deployed.** field.

> Note that this action will not handle down migrations. Always deploy a newer version than is currently deployed.
>
> Note that "fleetdm/fleet:main" is not a image name, instead use the commit hash in place of "main".

### Conclude current milestone 
Immediately after publishing a new release, we close out the associated GitHub issues and milestones. 

1. **Rename current milestone**: In GitHub, [change the current milestone name](https://github.com/fleetdm/fleet/milestones) from `4.x.x-tentative` to `4.x.x`. `4.37.0-tentative` becomes `4.37.0`.

2. **Update product group boards**: In ZenHub, go to each product group board tracking the current release. Usually, these are [#g-endpoint-ops](https://app.zenhub.com/workspaces/-g-endpoint-ops-current-sprint-63bd7e0bf75dba002a2343ac/board) and [#g-mdm](https://app.zenhub.com/workspaces/-g-mdm-current-sprint-63bc507f6558550011840298/board).

3. **Remove milestone from unfinished items**: If you see any items in columns other than "Ready for release" tagged with the current milestone, remove that milestone tag. These items didn't make it into the release.

4. **Prep release items**: Make sure all items in the "Ready for release" column have the current milestone and sprint tags. If not, select all items in the column and apply the appropriate tags.

5. **Move user stories to drafting board**: Select all items in "Ready for release" that have the `story` label. Apply the `:product` label and remove the `:release` label. These items will move back to the product drafting board.

6. **Confirm and close**: Make sure that all items with the `story` label have left the "Ready for release" column. Select all remaining items in the "Ready for release" column and move them to the "Closed" column. This will close the related GitHub issues.

8. **Confirm and celebrate**: Now, head to the [Drafting](https://app.zenhub.com/workspaces/-drafting-ships-in-6-weeks-6192dd66ea2562000faea25c/board) board. Find all `story` issues with the current milestone (these are the ones you just moved). Move them to the "Confirm and celebrate" column. Product will close the issues during their [confirm and celebrate ritual](https://fleetdm.com/handbook/product#rituals).

9. **Close GitHub milestone**: Visit [GitHub's milestone page](https://github.com/fleetdm/fleet/milestones) and close the current milestone.

10. **Create next milestone**: Create a new milestone for the next versioned release, `4.x.x-tentative`.

11. **Remove the freeze**: [Open the repo in Merge Freeze](https://app.mergefreeze.com/installations/3704/branches/6847) and click the "Unfreeze" button. 

12. Announce that `main` is unfrozen and the milestone has been closed in #help-engineering.

13. Create a branch for the next patch and push it to GitHub. For example, if you release `fleet-v4.44.0`, create a branch called `patch-fleet-v4.44.1`. 

### Update the Fleet releases calendar
The [Fleet releases Google calendar](https://calendar.google.com/calendar/embed?src=c_v7943deqn1uns488a65v2d94bs%40group.calendar.google.com&ctz=America%2FChicago) is kept up-to-date by the [release ritual DRI](https://fleetdm.com/handbook/engineering#rituals). Any change to targeted release dates is reflected on this calendar.

### Secure company-issued equipment for a team member
As soon as an offer is accepted, Fleet provides laptops and YubiKey security keys for core team members to use while working at Fleet. The IT engineer will work with the new team member to get their equipment requested and shipped to them on time.

- [**Check the Fleet IT warehouse**](https://docs.google.com/spreadsheets/d/1hFlymLlRWIaWeVh14IRz03yE-ytBLfUaqVz0VVmmoGI/edit#gid=0) before purchasing any equipment including laptops, monitors, and Yubikeys to ensure we efficiently [utilize existing assets before spending money](https://fleetdm.com/handbook/company/why-this-way#why-spend-less). If Fleet IT warehouse inventory can meet the needs of the request, file a [warehouse request](https://github.com/fleetdm/confidential/issues/new?assignees=sampfluger88&labels=%23g-digital-experience&projects=&template=warehouse-request.md&title=%F0%9F%92%BB+Warehouse+request).

- Apple computers shipping to the United States and Canada are ordered using the Apple [eCommerce Portal](https://ecommerce2.apple.com/asb2bstorefront/asb2b/en/USD/?accountselected=true), or by contacting the business team at an Apple Store or contacting the online sales team at [800-854-3680](tel:18008543680). The IT engineer can arrange for same-day pickup at a store local to the Fleetie if needed.
  - **Note:** Most Fleeties use 16-inch MacBook Pros. Team members are free to choose any laptop or operating system that works for them, as long as the price [is within reason](https://www.fleetdm.com/handbook/communications#spending-company-money). 

  - When ordering through the Apple eCommerce Portal, look for a banner with *Apple Store for FLEET DEVICE MANAGEMENT | Welcome [Your Name].* Hovering over *Welcome* should display *Your Profile.* If Fleet's account number is displayed, purchases will be automatically made available in Apple Business Manager (ABM).

- Apple computers for Fleeties in other countries should be purchased through an authorized reseller to ensure the device is enrolled in ADE. In countries that Apple does not operate or that do not allow ADE, work with the authorized reseller to find the best solution, or consider shipping to a US based Fleetie and then shipping on to the teammate. 

 > A 3-year AppleCare+ Protection Plan (APP) should be considered default for Apple computers >$1500. Base MacBook Airs, Mac minis, etc. do not need APP unless configured beyond the $1500 price point. APP provides 24/7 support, and global repair coverage in case of accidental screen damage or liquid spill, and battery service.

 - Order a pack of two [YubiKey 5C NFC security keys](https://www.yubico.com/product/yubikey-5-series/yubikey-5c-nfc/) for new team member, shipped to them directly.

- Include delivery tracking information when closing the support request so the new employee can be notified.

### Register a domain for Fleet
Domain name registrations are handled through Namecheap. Access is managed via 1Password.

### Fix a laptop that's not checking in
It is [possible for end users to remove launch agents](https://github.com/fleetdm/confidential/issues/6088) (this is true not just for osquery, but for anything). 

If the host has MDM turned on, use the `fleetctl mdm run-command` CLI command to push the XML file located at https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/macos-send-fleetd.xml to the device, which will reinstall fleetd.

If the host doesn't have MDM turned on or isn't enrolled to dogfood, it is beyond our ability to control remotely.

### Enroll a macOS host in dogfood
When a device is purchased using the Apple eCommerce store, the device is automatically enrolled in Apple Business Manager (ABM) and assigned to the correct server to ensure the device is in dogfood.
You can confirm that the device has been ordered correctly by following these steps:
- Log into ABM
- Use the device serial number to find the device.
  - Note: if the device cannot be found, you will need to manually enroll the device.
- View device settings and ensure the "MDM Server" selected is "Fleet Dogfood".

On occasion there will be a need to manually enroll a macOS host in dogfood. This could be due to a BYOD arrangement, or because the Fleetie getting the device is in a country when DEP (automatic enrollment) isn't supported. To manually enroll a macOS host in dogfood, follow these steps:
- If you have physical access to the macOS host, use Apple Configurator (docs are [here](https://support.apple.com/guide/apple-business-manager/add-devices-from-apple-configurator-axm200a54d59/web)).
- If you do not have physical access to the device, the user will need to undertake the following steps:
  - Install the fleetd package for your device from shared drive folder [here](https://drive.google.com/drive/folders/1-hMwk4P7NRzCU5kDxkEcOo8Sluuaux1h?usp=drive_link).
  - Once fleetd is installed, click on Fleet desktop icon in top right menu bar, and select "My device".
  - In Fleet desktop, follow the instructions to turn on MDM.
  - Once complete, follow instructions to reset disk encryption key.
- Disk encryption key will now be stored in Fleet dogfood, which signifies that the device is now enrolled in dogfood.

### Enroll a Windows or Ubuntu Linux device in dogfood
To enroll a windows or Ubuntu Linux device in dogfood, instruct the user to install fleetd for their platform from internal shared drive folder [here](https://drive.google.com/drive/folders/1-hMwk4P7NRzCU5kDxkEcOo8Sluuaux1h?usp=drive_link).
Once the user has installed fleetd, verify the device is correctly enrolled by confirming the device encryption key is in dogfood.

### Enroll a ChromeOS device in dogfood
ChromeOS devices are automatically enrolled in dogfood after the IT admin sets up automatic enrollment. This is done in dogfood by following the steps found in the dialog popup when selecting "Add hosts > ChromeOS" from the dogfood Hosts page.

### Lock a macOS host in dogfood using fleetctl CLI tool
- Download the lock command XML file from Google Drive [here](https://drive.google.com/file/d/1o6vJ1fHilRtBmyKAj0I5URiKn77qe4gS/view?usp=drive_link).
- Customize any messaging that will appear on the locked device, and modify the pin for unlocking the device by editing the file in text editor.
  - Note you will need to safely store the recovery pin for the device, suggest using 1Password or other secure storage method
- Run this command with fleetctl CLI tool: `fleetctl mdm run-command --hosts=hostname --payload=Downloads/command-lock-macos-host.xml`
  - Note that `hostname` must be replaced with **Hostname** in Fleet (not the display name)
  - Note that the payload path may change based on where the file is stored once downloaded
  - Note that if you haven't logged into fleetctl recently, will need to follow authentication steps (see [Logging in with SAML (SSO) authentication](https://fleetdm.com/docs/using-fleet/fleetctl-cli#logging-in-with-saml-sso-authentication) ).
- Device will be locked
- When device needs to be unlocked, enter the security pin (from XML file) in the input field of the device
- The device will then open to the regular login screen, asking for password
  -  If you do not have the password available, you can choose the option to enter recovery key/disk encryption key (this option might be behind `?` icon).
  - Get disk encryption key from Fleet dogfood (using the action menu from the individual host page).
  - Enter disk encryption key on laptop. This should prompt you to create a new password.
- You will then be logged into the default device profile, and can complete any needed actions (wipe, recover data).

### Book an event
Fleet's Client Platform Engineer & Community Advocate is responsible for booking events that Fleet has chosen to attend and/or sponsor. To book an event, complete the steps in each event issue. Contact the [🫧 Digital Marketing Manager](https://fleetdm.com/handbook/demand#team) as needed with any questions or blockers to booking an event.

> Note: The Demand department [settles all event strategy](https://fleetdm.com/handbook/demand#settle-event-strategy) prior to booking an event. 

<!-- TODO Create Article issue template and update this section of the handbook to reflect reality and add corresponding rituals to engineering.rituals.yml

### Create an article
Article creation begins with creation of an issue using the "Article request" template.

1. Create a [new GitHub issue for the #g-demand board](https://github.com/fleetdm/confidential/issues/new?assignees=spokanemac&labels=%23g-demand=&template=custom-request.md&title=Article%20Idea%3A+_______________________) and select the "Event-preparation" template..
2. Move the issue into the "📃 Planned articles" column.

### Review ongoing articles
Check the "📃 Planned articles" column in [#g-demand board](https://app.zenhub.com/workspaces/g-demand-64e6c8e2d35c7f001a457b7f/board) and continue to work through steps in each event's issue.
-->

### Order SWAG

**To order T-shirts:**

  - Check [Postal](https://app.postal.io/items/postals) first and see if the warehouse has enough shirts.
  - Navigate to the [approved items page](https://app.postal.io/items/postals).
      - Hover over the shirt design and click on the airplane.
      - Click bulk send and choose one shirt size and the expected quantity of that particular shirt size.
      - Make sure the address matches the expected receiving address.
  - If the Postal warehouse can't fulfill the order or To order swag quickly: 
      - Login to [https://www.rushordertees.com/my-account/login/) (saved in 1Password).
      - Choose Fleet logo design t-shirt under [my designs](https://www.rushordertees.com/my-account/designs/).
      - Order shirts based on the pre-determined number (~5% of total event attendees).
      - Submit the order. Ensure the address matches the expected receiving address.

**To order stickers:**

  - Login to [StickerMule](https://www.stickermule.com/) (saved in 1Password).
  - Find the [brand kit](https://www.stickermule.com/studio/brand-kits) after logging in.
  - Click on the "Fleet Device Management" brand kit and order preapproved stickers from the templates.
  - Total sticker quantity should be ~10% of total event attendees.
  - Complete the checkout process. Ensure the address matches the expected receiving address.

**To order pens and sticky note pads**

  - Pens and sticky note pads are ordered through Everything Branded.
  - Email our sales representative Jake William (saved in 1Password) to order any of the following:
    - [Javalina™ Metallic Stylus Pen](https://www.everythingbranded.com/product/javalina-metallic-stylus-pen-us-pat-8847930-9092077-350220)
    - [Sharpie Fine Point Markers](https://www.everythingbranded.com/product/sharpie-fine-point-332908)
    - [Custom sticky note pads](https://www.everythingbranded.com/product/custom-sticky-notes-585601) (design is in the StickerMule [brand kit](https://www.stickermule.com/studio/brand-kits))

### Review another product group's pull request
Some code paths require pull request review from multiple product groups to confirm there are no
unintended side effects of the change for another product group. All code paths defined in
[CODEOWNERS](https://github.com/fleetdm/fleet/blob/main/CODEOWNERS) that are assigned to individual
engineers across multiple product groups must be approved by one engineer from each product group
before merging.

### Review a community pull request
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

### Merge a community pull request
When merging a pull request from a community contributor:

- Ensure that the checklist for the submitter is complete.
- Verify that all necessary reviews have been approved.
- Merge the PR.
- Thank and congratulate the contributor.
- Share the merged PR with the team in the #help-promote channel of Fleet Slack to be publicized on social media. Those who contribute to Fleet and are recognized for their contributions often become great champions for the project.

### Close a stale community issue
If a community member opens an issue that we can't reproduce leave a comment asking the author for more context. After one week with no reply, close the issue with a comment letting them know they are welcome to re-open it with any updates.

### Schedule developer on-call workload
Engineering managers are asked to be aware of the [on-call rotation](https://docs.google.com/document/d/1FNQdu23wc1S9Yo6x5k04uxT2RwT77CIMzLLeEI2U7JA/edit#) and schedule a light workload for engineers while they are on-call. While it varies week to week considerably, the on-call responsibilities can sometimes take up a substantial portion of the engineer's time.

We aspire to clear sprint work for the on-call engineer, but due to capacity or other constraints, sometimes the on-call engineer is required for sprint work. When this is the case, the EM will work with the on-call engineer to take over support requests or @oncall assignment completely when necessary.

The remaining time after fulfilling the responsibilities of on-call is free for the engineer to choose their own path. Please choose something relevant to your work or Fleet's goals to focus on. If unsure, speak with your manager.

Some ideas:

- Do training/learning relevant to your work.
- Improve the Fleet developer experience.
- Hack on a product idea. Note: Experiments are encouraged, but not all experiments will ship! Check in with the product team before shipping user-visible changes.
- Create a blog post (or other content) for fleetdm.com.
- Try out an experimental refactor. 

### Edit a DNS record
We use Cloudflare to manage the DNS records of fleetdm.com and our other domains. To make DNS changes in Cloudflare:
1. Log into your Cloudflare account and select the "Fleet" account.
2. Select the domain you want to change and go to the DNS panel on that domain's dashboard.
3. To add a record, click the "Add record" button, select the record's type, fill in the required values, and click "Save". If you're making changes to an existing record, you only need to click on the record, update the record's values, and save your changes.

> If you need access to Fleet's Cloudflare account, please ask the [DRI](https://fleetdm.com/handbook/company/why-this-way#why-direct-responsibility) [Luke Heath](https://fleetdm.com/handbook/engineering#team) in Slack for an invitation.

### Assume developer on-call alias
The on-call developer is responsible for: 
- Knowing [the on-call rotation](https://fleetdm.com/handbook/company/product-groups#the-developer-on-call-rotation).
- Preforming the [on-call responsibilities](https://fleetdm.com/handbook/company/product-groups#developer-on-call-responsibilities).
- [Escalating community questions and issues](https://fleetdm.com/handbook/company/product-groups#escalations).
- Successfully [transferring the on-call persona to the next developer](https://fleetdm.com/handbook/company/product-groups#changing-of-the-guard).

### Notify stakeholders when a user story is pushed to the next release
[User stories](https://fleetdm.com/handbook/company/product-groups#scrum-items) are intended to be completed in a single sprint. When a user story selected for a release has not merged into `main` by the time the [merge freeze](https://fleetdm.com/handbook/engineering#begin-a-merge-freeze) begins, it is the product group EM's responsibility to notify stakeholders:

1. Add the `~pushed` label to the user story.
2. Update the user story's milestone to the next minor version milestone.
3. Comment on the GitHub issue and at-mention the PM and anyone listed in the requester field.
4. If `customer-` labels are applied to the user story, at-mention the [VP of Customer Success](https://fleetdm.com/handbook/customer-success#team).

### Run Fleet locally for QA purposes
To try Fleet locally for QA purposes, run `fleetctl preview`, which defaults to running the latest stable release.

To target a different version of Fleet, use the `--tag` flag to target any tag in [Docker Hub](https://hub.docker.com/r/fleetdm/fleet/tags?page=1&ordering=last_updated), including any git commit hash or branch name. For example, to QA the latest code on the `main` branch of fleetdm/fleet, you can run: `fleetctl preview --tag=main`.

To start a preview without starting the simulated hosts, use the `--no-hosts` flag (e.g., `fleetctl preview --no-hosts`).

For each bug found, please use the [bug report template](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=bug%2C%3Areproduce&template=bug-report.md&title=) to create a new bug report issue.

For unreleased bugs in an active sprint, a new bug is created with the `~unreleased bug` label. The `:release` label and associated product group label is added, and the engineer responsible for the feature is assigned. If QA is unsure who the bug should be assigned to, it is assigned to the EM. Fixing the bug becomes part of the story.

### Accept new Apple developer account terms
Engineering is responsible for managing third-party accounts required to support engineering infrastructure. We use the official Fleet Apple developer account to notarize installers we generate for Apple devices. Whenever Apple releases new terms of service, we are unable to notarize new packages until the new terms are accepted.

When this occurs, we will begin receiving the following error message when attempting to notarize packages: "You must first sign the relevant contracts online." To resolve this error, follow the steps below.

1. Visit the [Apple developer account login page](https://appleid.apple.com/account?appId=632&returnUrl=https%3A%2F%2Fdeveloper.apple.com%2Fcontact%2F).

2. Log in using the credentials stored in 1Password under "Apple developer account".

3. Contact the Head of Business Operations to determine which phone number to use for 2FA.

4. Complete the 2FA process to log in.

5. Accept the new terms of service.

### Interview a developer candidate
As a hiring manager we want to ensure the interview process follows these steps in order. This process must follow [creating a new position](https://fleetdm.com/handbook/company/leadership#creating-a-new-position) through [receiving job applications](https://fleetdm.com/handbook/company/leadership#receiving-job-applications). Once the position is approved manage this process per candidate in [hiring pipeline](https://drive.google.com/drive/folders/1dLZaor9dQmAxcxyU6prm-MWNd-C-U8_1?usp=drive_link)

1. **Reach out**: If you are not already the primary contact with this candidate send an email or linkedin message introducing yourself and the intent that you would like the start the interview process including the link to the position and asking if they are comfortable with completing a coding exercise.
2. **Deliver code prompt**: After recieving confirmation that they are interested download the zip of the [code challenge](https://github.com/fleetdm/wordgame) and ask them to complete this and send their entry back within 5 business days.
3. **Test code prompt**: Verify the code runs and can complete the challenge correctly. Check the code for good style and tests that match our standards here at Fleet.
4. **Schedule manager interview**: Send the candidate a calendly link for 1hr to talk to you and screen them if they are a good fit for this role and our culture.
5. **Schedule technical interview**: Send the candidate a calendly link for 1hr to talk to a senior engineer on your team where the goal is to understand the thechnical capabilities of the candidate. An additional engineer can optionally join if available.
6. **Schedule DOPD interview**: Send the candidate a calendly link for 30m talk to the Director of Product Development @lukeheath.
7. **Schedule CTO interview**: Send the candidate a calendly link for 30m talk with our CTO @lukeheath.

If the candidate passes all of these steps then continue with [hiring a new team member](https://fleetdm.com/handbook/company/leadership#hiring-a-new-team-member).

### Renew MDM certificate signing request (CSR) 
The certificate signing request (CSR) certificate expires every year. It needs to be renewed prior to expiring. This is notified to the team by the MDM calendar event [IMPORTANT: Renew MDM CSR certificate](https://calendar.google.com/calendar/event?action=TEMPLATE&tmeid=NHM3YzZja2FoZTA4bm9jZTE3NWFvMTduMTlfMjAyNDA5MTFUMTczMDAwWiBjXzI0YjkwMjZiZmIzZDVkMDk1ZGQzNzA2ZTEzMWQ3ZjE2YTJmNDhjN2E1MDU1NDcxNzA1NjlmMDc4ODNiNmU3MzJAZw&tmsrc=c_24b9026bfb3d5d095dd3706e131d7f16a2f48c7a505547170569f07883b6e732%40group.calendar.google.com&scp=ALL)

Steps to renew the certificate:

1. Visit the [Apple developer account login page](https://appleid.apple.com/account?appId=632&returnUrl=https%3A%2F%2Fdeveloper.apple.com%2Fcontact%2F).
2. Log in using the credentials stored in 1Password under **Apple developer account**.
3. Verify you are using the **Enterprise** subaccount for Fleet Device Management Inc.
4. Generate a new certificate following the instructions in [MicroMDM](https://github.com/micromdm/micromdm/blob/c7e70b94d0cfc7710e5c92be20d4534d9d5a0640/docs/user-guide/quickstart.md?plain=1#L103-L118).
5. Note: `mdmctl` (a micromdm command for MDM vendors) will generate a `VendorPrivateKey.key` and `VendorCertificateRequest.csr` using an appropriate shared email relay and a passphrase (suggested generation method with pwgen available in brew / apt / yum `pwgen -s 32 -1vcy`)
6. Uploading `VendorCertificateRequest.csr` to Apple you will download a corresponding `mdm.cer` file
7. Convert the downloaded cert to PEM with `openssl x509 -inform DER -outform PEM -in mdm.cer -out server.crt.pem`
8. Update the **Config vars** in [Heroku](https://dashboard.heroku.com/apps/production-fleetdm-website/settings):
* Update `sails_custom__mdmVendorCertPem` with the results from step 7 `server.crt.pem`
* Update `sails_custom__mdmVendorKeyPassphrase` with the passphrase used in step 4
* Update `sails_custom__mdmVendorKeyPem` with `VendorPrivateKey.key` from step 4
9. Store updated values in [Confidential 1Password Vault](https://start.1password.com/open/i?a=N3F7LHAKQ5G3JPFPX234EC4ZDQ&v=lcvkjobeheaqdgnz33ontpuhxq&i=byyfn2knejwh42a2cbc5war5sa&h=fleetdevicemanagement.1password.com)
10. Verify by logging into a normal apple account (not billing@...) and Generate a new Push Certificate following our [setup MDM](https://fleetdm.com/docs/using-fleet/mdm-setup) steps and verify the Expiration date is 1 year from today.
11. Adjust calendar event to be between 2-4 weeks before the next expiration.

### Perform an incident postmortem
Conduct a postmortem meetings for every service or feature outage and every critical bug, whether it's a customer's environment or on fleetdm.com.

1. Copy this [postmortem template](https://docs.google.com/document/d/1Ajp2LfIclWfr4Bm77lnUggkYNQyfjePiWSnBv1b1nwM/edit?usp=sharing) document and pre-populate where possible.
2. Invite stakeholders. Typically the EM, PM, QA, and engineers involved. If a customer incident, include the CSM.
3. Follow and populate document topic by topic. Determine the root cause (why it happened), as well as why our controls did not catch it before release.
4. Assign each action item an owner that who is responsible for creating a Github issue promptly and working with with the relevant PM/EM to prioritize.

[Example Finished Document](https://docs.google.com/document/d/1YnETKhH9R7STAY-PaFnPy2qxhNht2EAFfkv-kyEwebQ/edit?usp=share_link)

### Process incoming equipment
Upon receiving any device, follow these steps to process incoming equipment.
1. Search for the SN of the physical device in the ["Company equipment" spreadsheet](https://docs.google.com/spreadsheets/d/1hFlymLlRWIaWeVh14IRz03yE-ytBLfUaqVz0VVmmoGI/edit#gid=0) to confirm the correct equipment was received.
3. Visibly inspect equipment and all related components (e.g. laptop charger) for damage.
4. Remove any stickers and clean devices and components.
5. Using the device's charger plug in the device.
6. Turn on the device and enter recovery mode using the [appropriate method](https://support.apple.com/en-us/HT204904).
7. Connect the device to WIFI.
8. Using the "Recovery assistant" tab (In the top left corner), select "Delete this Mac".
9. Follow the prompts to activate the device and reinstall the appropriate version of macOS.
> If you are prevented from completing the steps above, create a ["💻 IT support issue](https://github.com/fleetdm/confidential/issues/new?assignees=%40spokanemac&labels=%3Ahelp-it&projects=&template=request-it-support.md&title=%F0%9F%92%BB+Request+IT+support) for IT, for the device to be scheduled for troubleshooting and remediation. Please note in the issue where you encountered blockers to completing the steps.

### Ship approved equipment
Once the Business Operations department approves inventory to be shipped from Fleet IT, follow these step to ship the equipment.
1. Compare the equipment request issue with the ["Company equipment" spreadsheet](https://docs.google.com/spreadsheets/d/1hFlymLlRWIaWeVh14IRz03yE-ytBLfUaqVz0VVmmoGI/edit#gid=0) and verify physical inventory.
2. Plug in the device and ensure inventory has been correctly processed and all components are present (e.g. charger cord, power converter).
3. package equipment for shipment and include Yubikeys (if requested).
4. Change the "Company equipment" spreadsheet to reflect the new user.
  - If you encounter any issues, repeat the [process incoming equipment steps](https://fleetdm.com/handbook/engineering#process-incoming-equipment). If problems persist, create a ["💻 IT support issue](https://github.com/fleetdm/confidential/issues/new?assignees=%40spokanemac&labels=%3Ahelp-it&projects=&template=request-it-support.md&title=%F0%9F%92%BB+Request+IT+support) for IT to troubleshoot the device.
6. Ship via FedEx to the address listed in the equipment request.
7. Add a comment to the equipment request issue, at-mentioning the requestor with the FedEx tracking info and close the issue.

## Rituals
<rituals :rituals="rituals['handbook/engineering/engineering.rituals.yml']"></rituals>

#### Stubs
The following stubs are included only to make links backward compatible.

##### Weekly bug review
[handbook/company/product-groups#weekly-bug-review](https://fleetdm.com/handbook/company/product-groups#weekly-bug-review)

Please see [docs/contributing/infrastructure](https://fleetdm.com/docs/contributing/infrastructure) for **below**
##### Infrastructure
##### Infrastructure links
##### Best practices for containers
Please see [docs/contributing/infrastructure](https://fleetdm.com/docs/contributing/infrastructure) for **above**

##### Measurement
Please see [handbook/engineering#record-engineering-kpis](https://fleetdm.com/handbook/engineering#record-engineering-kpis)

##### Critical bug notification process
Please see [handbook/engineering#notify-community-members-about-a-critical-bug](https://fleetdm.com/handbook/engineering#notify-community-members-about-a-critical-bug)

##### Finding bugs
Please see [handbook/engineering#run-fleet-locally-for-qa-purposes](https://fleetdm.com/handbook/engineering#run-fleet-localy-for-qa-purposes)

##### Scrum at Fleet
Please see [handbook/company/product-groups#scrum-at-fleet](https://fleetdm.com/handbook/company/product-groups#scrum-at-fleet)

##### Scrum items
Please see [handbook/company/product-groups#scrum-items)](https://fleetdm.com/handbook/company/product-groups#scrum-items)

##### Sprint ceremonies
Please see [handbook/company/product-groups#sprint-ceremonies](https://fleetdm.com/handbook/company/product-groups#sprint-ceremonies)

##### Meetings
Please see [handbook/company/product-groups#meetings](https://fleetdm.com/handbook/company/product-groups#meetings)

##### Principles
Please see [handbook/company/product-groups#principles](https://fleetdm.com/handbook/company/product-groups#principles)

Please see [handbook/company/product-groups#eng-together](https://fleetdm.com/handbook/company/product-groups#eng-together) for **below**
##### Eng Together
##### Participants
##### Agenda
Please see [handbook/company/product-groups#eng-together](https://fleetdm.com/handbook/company/product-groups#eng-together) for **above**

Please see [handbook/company/product-groups#group-weeklies](https://fleetdm.com/handbook/company/product-groups#group-weeklies) for **below**
##### User story discovery
##### Participants
##### Agenda
Please see [handbook/company/product-groups#group-weeklies](https://fleetdm.com/handbook/company/product-groups#group-weeklies) for **above**

Please see [handbook/company/product-groups#group-weeklies](https://fleetdm.com/handbook/company/product-groups#group-weeklies) for **below**
##### Group weeklies
##### Participants
##### Sample agenda (Frontend weekly)
Please see [handbook/company/product-groups#group-weeklies](https://fleetdm.com/handbook/company/product-groups#group-weeklies) for **above**

##### Engineering-initiated stories
##### Creating an engineering-initiated story

Please see [handbook/engineering#create-an-engineering-initiated-story](https://fleetdm.com/handbook/engineering#create-an-engineering-initiated-story) for **above**

Please see [handbook/engineering#accept-new-apple-developer-account-terms](https://fleetdm.com/handbook/engineering#accept-new-apple-developer-account-terms) for **below**
##### Accounts  
##### Apple developer account
Please see [handbook/engineering#accept-new-apple-developer-account-terms](https://fleetdm.com/handbook/engineering#accept-new-apple-developer-account-terms) for **above**

##### Merging during the freeze period
Please see [handbook/engineering#merge-a-pull-request-during-the-freeze-period](https://fleetdm.com/handbook/engineering#merge-a-pull-request-during-the-freeze-period)

##### Scrum boards
Please see [handbook//product-groups#current-product-groups](https://fleetdm.com/handbook/engineering#contact-us)

Please see [handbook/engineering#begin-a-merge-freeze](https://fleetdm.com/handbook/engineering#begin-a-merge-freeze) for **below**
##### Release freeze period
##### Freeze day
Please see [handbook/engineering#begin-a-merge-freeze](https://fleetdm.com/handbook/engineering#begin-a-merge-freeze) for **above**

##### Release day
Please see [handbook/engineering#prepare-fleet-release](https://fleetdm.com/handbook/engineering#prepare-fleet-release)

##### Deploying to dogfood
Please see [handbook/engineering#deploy-a-new-release-to-dogfood](https://fleetdm.com/handbook/engineering#deploy-a-new-release-to-dogfood)

Please see [handbook/engineering#conclude-current-milestone](https://fleetdm.com/handbook/engineering#conclude-current-milestone) for **below**
##### Milestone release ritual
##### Update milestone in GitHub
##### ZenHub housekeeping
Please see [handbook/engineering#conclude-current-milestone](https://fleetdm.com/handbook/engineering#conclude-current-milestone) for **above**

##### Clearing the plate
Please see [handbook/engineering#schedule-developer-on-call-workload](https://fleetdm.com/handbook/engineering#schedule-developer-on-call-workload)

##### Check dependencies
Please see [handbook/engineering#confirm-latest-versions-of-dependencies](https://fleetdm.com/handbook/engineering#confirm-latest-versions-of-dependencies)

##### Release readiness
Please see [handbook/engineering#indicate-your-product-group-is-release-ready](https://fleetdm.com/handbook/engineering#indicate-your-product-group-is-release-ready)

##### Improve documentation
Please see [handbook/company/product-groups#documentation-for-contributors](https://fleetdm.com/handbook/company/product-groups#documentation-for-contributors)

##### How to reach the on-call engineer
Please see [handbook/company/product-groups#how-to-reach-the-developer-on-call](https://fleetdm.com/handbook/company/product-groups#how-to-reach-the-developer-on-call)

##### The rotation
Please see [handbook/company/product-groups#the-developer-on-call-rotation](https://fleetdm.com/handbook/company/product-groups#the-developer-on-call-rotation)

Please see [handbook/company/product-groups#the-developer-on-call-rotation](https://fleetdm.com/handbook/company/product-groups#developer-on-call-responsibilities) for **below**
##### Second-line response
##### PR reviews
##### Customer success meetings
Please see [handbook/company/product-groups#the-developer-on-call-rotation](https://fleetdm.com/handbook/company/product-groups#developer-on-call-responsibilities) for **above**

##### Escalations
Please see [handbook/company/product-groups#escalations](https://fleetdm.com/handbook/company/product-groups#escalations)

##### Handoff
Please see [handbook/company/product-groups#changing-of-the-guard](https://fleetdm.com/handbook/company/product-groups#changing-of-the-guard)

Please see [handbook/company/product-groups#quality](https://fleetdm.com/handbook/company/product-groups#quality) for **below**
##### Quality
##### Human-oriented QA
##### Bug process
##### Debugging
##### Bug states
Please see [handbook/company/product-groups#quality](https://fleetdm.com/handbook/company/product-groups#quality) for **above**

##### Inbox
Please see [handbook/company/product-groups#inbox](https://fleetdm.com/handbook/company/product-groups#inbox)

Please see [handbook/company/product-groups#reproduced](https://fleetdm.com/handbook/company/product-groups#reproduced) for **below**
##### Reproduced
##### Fast track for Fleeties
Please see [handbook/company/product-groups#reproduced](https://fleetdm.com/handbook/company/product-groups#reproduced) for **above**

##### In product drafting (as needed)
Please see [handbook/company/product-groups#in-product-drafting-as-needed](https://fleetdm.com/handbook/company/product-groups#in-product-drafting-as-needed)

##### In engineering
Please see [handbook/company/product-groups#in-engineering](https://fleetdm.com/handbook/company/product-groups#in-engineering)

##### Awaiting QA
Please see [handbook/company/product-groups#awaiting-qa](https://fleetdm.com/handbook/company/product-groups#awaiting-qa)

Please see [handbook/company/product-groups#all-bugs](https://fleetdm.com/handbook/company/product-groups#all-bugs) for **below**
##### All bugs
##### Bugs closed this week
##### Bugs closed this week
Please see [handbook/company/product-groups#all-bugs](https://fleetdm.com/handbook/company/product-groups#all-bugs) for **above**

Please see [handbook/company/product-groups#release-testing](https://fleetdm.com/handbook/company/product-groups#release-testing) for **below**
##### Release testing
##### Release blockers
##### Critical bugs
Please see [handbook/company/product-groups#release-testing](https://fleetdm.com/handbook/company/product-groups#release-testing) for **above**

##### Reviewing PRs from the community
Please see [handbook/engineering#review-a-community-pull-request](https://fleetdm.com/handbook/engineering#review-a-community-pull-request)

##### Merging community PRs
Please see [handbook/engineering#merge-a-community-pull-request](https://fleetdm.com/handbook/engineering#merge-a-community-pull-request)

##### Changes to tables' schema
Please see [handbook/company/product-groups#changes-to-tables-schema](https://fleetdm.com/handbook/company/product-groups#changes-to-tables-schema)

Please see [handbook/engineering#preform-an-incident-postmortem](https://fleetdm.com/handbook/engineering#preform-an-incident-postmortem) for **below**
##### Incident postmortems
##### Postmortem document
##### Postmortem meeting
##### Postmortem action items
Please see [handbook/engineering#preform-an-incident-postmortem](https://fleetdm.com/handbook/engineering#preform-an-incident-postmortem) for **below**

##### Outages
[handbook/company/product-groups#outages](https://fleetdm.com/handbook/company/product-groups#outages)

##### Scaling Fleet
[handbook/company/product-groups#scaling-fleet](https://fleetdm.com/handbook/company/product-groups#scaling-fleet)

##### Load testing
[handbook/company/product-groups#load-testing](https://fleetdm.com/handbook/company/product-groups#load-testing)

##### Version support
[handbook/company/product-groups#version-support](https://fleetdm.com/handbook/company/product-groups#version-support)

<meta name="maintainedBy" value="lukeheath">
<meta name="title" value="🚀 Engineering">

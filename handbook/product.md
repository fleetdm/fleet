# Product

## Product design process

The product team is responsible for product design tasks. These include drafting
changes to the Fleet product, reviewing and collecting feedback from engineering, sales, customer success, and marketing counterparts, and delivering
these changes to the engineering team.

### Drafting

* Move an issue that is assigned to you in the "Ready" column of the ["Product team" GitHub
  project](https://github.com/orgs/fleetdm/projects/17) to the "In progress" column.

* Create a page in the Fleet (scratchpad, dev-ready) Figma and combine your issue's number and
  title to name the Figma page.

* Draft changes to the Fleet product that solve the problem specified in the issue. Constantly place
  yourself in the shoes of a user while drafting changes.

* While drafting, reach out to sales, customer success, and marketing for a new perspective.

* While drafting, engage engineering to gain insight into technical costs and feasibility.

### Review

* Move the issue into the "Ready for review" column. The drafted changes that correspond to each
  issue in this column will be reviewed during the recurring product huddle meeting.

* During the product huddle meeting, record any feedback on the drafted changes for each
  of your issues.

### Deliver

* Once your work is complete and all feedback is addressed, make sure that the issue is updated with
  a link to the correct page in the Fleet EE (scratchpad) Figma. This page is where the design
  specifications live.

* Add the ":product" and ":architect" labels to the issue and remove all other labels. This will add the issue to the product
  board on ZenHub. In the ZenHub board, move the issue into the "Architect" column in the product
  board so an architect on the engineering team knows that this issue is ready for engineering specification and later,
  engineering estimation.

#### Priority drafting

Priority drafting is the revision of drafted changes that are currently being developed by
the engineering team. The goal of priority drafting is to quickly adapt to unknown edge cases and
changing specification while ensuring
that Fleet meets our brand and quality guidelines. 

Priority drafting occurs in the following scenarios:

* A drafted UI change is missing crucial information that prevents the engineering team from
  continuing the development task.

* Functionality included in a drafted UI change must be cut down in order to ship the improvement in
  the current scheduled release.

What happens during priority drafting?

1. Everyone on the product team and engineering team is made aware that a drafted change was brought back
   to drafting and prioritized. 

2. Drafts are updated to cover edge cases or reduce functionality.

3. UI changes are reviewed and the UI changes are brought back to the engineering team to continue
  the development task.

## Product quality

Fleet uses a human-oriented quality assurance (QA) process to ensure the product meets the standards of users and organizations.

To try Fleet locally for QA purposes, run `fleetctl preview`, which defaults to running the latest stable release.

To target a different version of Fleet, use the `--tag` flag to target any tag in [Docker Hub](https://hub.docker.com/r/fleetdm/fleet/tags?page=1&ordering=last_updated), including any git commit hash or branch name.  For example, to QA the latest code on the `main` branch of fleetdm/fleet, you can run: `fleetctl preview --tag=main`

To start preview without starting the simulated hosts, use the `--no-hosts` flag (eg. `fleetctl preview --no-hosts`).

### Why human-oriented QA?

Automated tests are important, but they can't catch everything.  Many issues are hard to notice until a human looks empathetically at the user experience, whether that's in the user interface, the REST API, or the command line.

The goal of quality assurance is to catch unexpected behavior prior to release:
- bugs
- edge cases
- error message UX
- developer experience using the API/CLI
- operator experience looking at logs
- API response time latency
- UI comprehensibility
- simplicity
- data accuracy
- perceived data freshness
- the product’s ability to save users from themselves


### Collecting bugs

All QA steps should be possible using `fleetctl preview`.  Please refer to [docs/03-Contributing/02-Testing.md](https://github.com/fleetdm/fleet/blob/main/docs/03-Contributing/02-Testing.md) for flows that cannot be completed using `fleetctl preview`.

Please start the manual QA process by creating a blank GitHub issue. As you complete each of the
flows, record a list of the bugs you encounter in this new issue. Each item in this list should
contain one sentence describing the bug and a screenshot if the item is a frontend bug.

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

## Fleet docs

### Docs style guide

#### Headings

Headings help readers scan content to easily find what they need. Organize page content using clear headings, specific to the topic they describe.

Keep headings brief and organize them in a logical order:

* H1: Page title
* H2: Main headings
* H3: Subheadings
* H4: Sub-subheadings (headings nested under subheadings)

Try to stay within 3 or 4 heading levels. Complicated documents may use more, but pages with a simpler structure are easier to read.

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

## UI design

### Communicating design changes to the engineering team
For something NEW that has been added to [Figma Fleet EE (current, dev-ready)](https://www.figma.com/file/qpdty1e2n22uZntKUZKEJl/?node-id=0%3A1):
1. Create a new [GitHub issue](https://github.com/fleetdm/fleet/issues/new)
2. Detail the required changes (including page links to the relevant layouts), then assign it to the __“Initiatives”__ project.

<img src="https://user-images.githubusercontent.com/78363703/129840932-67d55b5b-8e0e-4fb9-9300-5d458e1b91e4.png" alt="Assign to Initiatives project"/>

> ___NOTE:___ Artwork and layouts in Figma Fleet EE (current) are final assets, ready for implementation. Therefore, it’s important NOT to use the “idea” label, as designs in this document are more than ideas - they are something that WILL be implemented._

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

## Release 

This section outlines the communication between the product team and growth team and product team
and customer success team prior to a release of Fleet.

### Goal

Keep the business up to date with improvements and changes to the Fleet product so that all stakeholders are able to communicate
with customers and users.

### Blog post

The product team is responsible for providing the [growth team](./growth.md) with necessary information for writing
the release blog post. This is accomplished by filing a release blog post issue and adding
the issue to the growth board on GitHub.

The release blog post issue includes a list of the primary features included in the upcoming
release. This list of features should point the reader to the GitHub issue that explains each
feature in more detail.

An example release blog post issue can be found [here](https://github.com/fleetdm/fleet/issues/3465).

### Customer announcement

The product team is responsible for providing the [customer success team](./customer-experience.md) with necessary information
for writing a release customer announcement. This is accomplished by filing a release customer announcement issue and adding
the issue to the customer success board on GitHub. 


The release blog post issue is filed in the private fleetdm/confidential repository because the
comment section may contain private information about Fleet's customers.

An example release customer announcement blog post issue can be found [here](https://github.com/fleetdm/confidential/issues/747).

## Feature flags

In Fleet, features are placed behind feature flags if the changes could affect Fleet's availability of existing functionalities.

The following highlights should be considered when deciding if feature flags should be leveraged:

- The feature flag must be disabled by default.
- The feature flag will not be permanent. This means that the individual who decides that a feature flag should be introduced is responsible for creating an issue to track the feature's progress towards removing the feature flag and including the feature in a stable release.
- The feature flag will not be advertised. For example, advertising in the documentation on fleetdm.com/docs, release notes, release blog posts, and Twitter.

Fleet's feature flag guidelines borrows from GitLab's ["When to use feature flags" section](https://about.gitlab.com/handbook/product-development-flow/feature-flag-lifecycle/#when-to-use-feature-flags) of their handbook. Check out [GitLab's "Feature flags only when needed" video](https://www.youtube.com/watch?v=DQaGqyolOd8) for an explanation on the costs of introducing feature flags.

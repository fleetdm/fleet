# Community

As an open-core company, Fleet endeavors to build a community of engaged users, customers, and
contributors.

## Communities

Fleet's users and broader audience are spread across many online platforms.  Here are the most active communities where Fleet's developer relations and social media team members participate at least once every weekday:

- [Osquery Slack](https://join.slack.com/t/osquery/shared_invite/zt-h29zm0gk-s2DBtGUTW4CFel0f0IjTEw) (`#fleet` channel)
- [MacAdmins Slack](https://www.macadmins.org/) (`#fleet` channel)
- [osquery discussions on LinkedIn](https://www.linkedin.com/search/results/all/?keywords=osquery)
- [osquery discussions on Twitter](https://twitter.com/search?q=osquery&src=typed_query)
- [reddit.com/r/sysadmins](https://www.reddit.com/r/sysadmin/)
- [reddit.com/r/SysAdminBlogs](https://www.reddit.com/r/SysAdminBlogs/)
- [r/sysadmin Discord](https://discord.gg/sysadmin)

## Posting on social media as Fleet

Self-promotional tweets are non-ideal tweets.  (Same goes for, to varying degrees, Reddit, HN, Quora, StackOverflow, LinkedIn, Slack, and almost anywhere else.)  See also https://www.audible.com/pd/The-Impact-Equation-Audiobook/B00AR1VFBU

Great brands are [magnanimous](https://en.wikipedia.org/wiki/Magnanimity).

## Promoting blog posts on social media

Once a blog post has been written, approved, and published, please ensure that it has been promoted on social media. Please refer to our [Publishing as Fleet](https://docs.google.com/document/d/1cmyVgUAqAWKZj1e_Sgt6eY-nNySAYHH3qoEnhQusph0/edit?usp=sharing) guide for more detailed information. 

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
You can link documentation pages to each other using relative paths. For example, in `docs/Using-Fleet/Fleet-UI.md`, you can link to `docs/Using-Fleet/Permissions.md` by writing `[permissions](./09-Permissions.md)`. This will be automatically transformed into the appropriate URL for `fleetdm.com/docs`.

However, the `fleetdm.com/docs` compilation process does not account for relative links to directories **outside** of `/docs`.
Therefore, when adding a link to Fleet docs, it is important to always use the absolute file path.

When directly linking to a specific section within a page in the Fleet documentation, always format the spaces within a section name to use a hyphen `-` instead of an underscore `_`. For example, when linking to the `osquery_result_log_plugin` section of the configuration reference docs, use a relative link like the following: `./Configuration.md#osquery-result-log-plugin`.

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


## Press releases

If we are doing a press release, we are probably pitching it to one or more reporters as an exclusive story, if they choose to take it.  Consider not sharing or publicizing any information related to the upcoming press release before the announcement.  See also https://www.quora.com/What-is-a-press-exclusive-and-how-does-it-work

### Press release boilerplate

Fleet gives teams fast, reliable access to data about the production servers, employee laptops, and other devices they manage - no matter the operating system. Users can search for any device data using SQL queries, making it faster to respond to incidents and automate IT. Fleet can also be used to monitor vulnerabilities, battery health, software, and even EDR and MDM tools like Crowdstrike, Munki, Jamf, and Carbon Black, to help confirm that those platforms are working how administrators "think" they are. Fleet is open-source software. It's easy to get started quickly, easy to deploy, and it even comes with an enterprise-friendly free tier available under the MIT license.

IT and security teams love Fleet because of its flexibility and conventions. Instead of secretly collecting as much data as possible, Fleet defaults to privacy and transparency, capturing only the data your organization needs to meet its compliance, security, and management goals, with clearly-defined, flexible limits.   

That means better privacy. Better device performance. And better data, with less noise.

## Community contributions (pull requests)

The top priority when community members contribute PRs is to help the person feel engaged with
Fleet. This means acknowledging the contribution quickly (within 1 business day), and driving to a
resolution (close/merge) as soon as possible (may take longer than 1 business day).

#### Process

1. Decide whether the change is acceptable (see below). If this will take time, acknowledge the
   contribution and let the user know that the team will respond. For changes that are not
   acceptable, thank the contributor for their interest and encourage them to open an Issue, or
   discuss proposed changes in the `#fleet` channel of osquery Slack before working on any more
   code.
2. Help the contributor get the quality appropriate for merging. Ensure that the appropriate manual
   and automated testing has been performed, changes files and documentation are updated, etc.
   Usually this is best done by code review and coaching the user. Sometimes (typically for
   customers) a Fleet team member may take a PR to completion by adding the appropriate testing and
   code review improvements.
3. Any Fleet team member may merge a community PR after reviewing it and addressing any necessary
   changes. Before merging, double-check that the CI is passing, documentation is updated, and
   changes file is created. Please use your best judgement.
4. Once a PR has been approved and merged, thank and congratulate the contributor, then share with the team in the `#help-promote` channel of Fleet Slack to be publicized on social media. Folks who contribute to Fleet and are recognized for their contributions often become great champions for the project.

Please refer to our [PRs from the community](https://docs.google.com/document/d/13r0vEhs9LOBdxWQWdZ8n5Ff9cyB3hQkTjI5OhcrHjVo/edit?usp=sharing) guide for more detailed information.

#### What is acceptable?

Generally, any small documentation update or bugfix is acceptable, and can be merged by any member
of the Fleet team. Additions or fixes to the Standard Query Library are acceptable as long as the
SQL works properly and they attributed correctly. Please use your best judgement.

Larger changes and new features should be approved by the appropriate [Product
DRI](./product.md#product-dris). Ask in the `#g-product` channel in Fleet Slack.

<meta name="maintainedBy" value="mike-j-thomas">

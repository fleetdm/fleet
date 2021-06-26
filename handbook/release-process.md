# Releases

This document outlines the release process at Fleet.

The current release cadence is once every 3 weeks and concentrated around Wednesdays. 

- [Milestones](#milestones)
- [Release issue](#release-issue)
- [Social assets](#social-assets)
- [Release week](#release-week)
- [Release day](#release-day)

## Milestones

At Fleet, we use the Milestone GitHub feature to tag GitHub issues that include features and tasks to be included in a release.

The individual tasked with managing releases is responsible for creating a new milestone for the upcoming release. As soon as the release process for the previous release is complete, a new milestone should be created for the upcoming release. 

The title of the milestone corresponds to the version number of Fleet that is to be released. For example, if the upcoming version number of Fleet is 3.12.0, a milestone with the title “3.12.0” should be created.

## Release issue

One week before the release date, the individual tasked with managing the release will create a release issue to start preparing for the release. [Click here](https://github.com/fleetdm/confidential/issues/new?milestone=ASAP&body=%3E%20**This%20issue%20contains%20confidential%20information.**%0A%0A%23%23%20Fleet%20%3CVERSION%3E%0A%0A%23%23%23%20Release%20date%0A%0ATODO%0A%0A%23%23%23%20Summary%0A%0ATODO%0A%0A%23%23%23%20CHANGELOG%0A%0ATODO%0A%0A%23%23%23%20Core%20tasks%0A%0A-%20%5B%20%5D%20TODO%0A%0A%23%23%23%20Growth%20tasks%0A%0A-%20%5B%20%5D%20TODO) to open the release issue template.

### Changelog


The Changelog section of the release issue acts as a short term roadmap and will be used as the public facing Changelog included in the release.

To construct the Changelog, first, head to the [commit history for fleetdm/fleet](https://github.com/fleetdm/fleet/commits/main). Next, navigate to the commit made to prepare for the previous release. This commit is usually titled something like “Prepare for `<release number>`.” Finally, add a bullet point to the Changelog for each commit, according to the following:

1. Only include changes that are relevant to Fleet users. This is because the Changelog serves as a tool to both inform _and_ excite users of Fleet. This means that changes made to the development infrastructure, documentation, and contribution experience shouldn’t be included.
2. Each bullet should start with a verb. For example, “Add,” or “Fix.”
3. Each bullet should be a complete sentence.
4. The bulleted list should be ordered from most exciting items (on the top) to least exciting items (on the bottom). New features and performance improvements live at the top of the list, while bug fixes like to live at the bottom.

The number of the commits will largely outweigh the number of bullets included in the Changelog. This makes sense because as a Fleet user I want to see an informative collection of the changes I care about rather than a list of all the commits that have been made since the last release.

### Summary

The Summary section of the release issue servers as an outline for the release Medium blog post and fodder for the release Tweet. The summary should be one sentence and reflect the most exciting additions included in the release.

## Blog post

The individual tasked with managing the release is responsible for drafting the release Medium blog post.

Using the release summary as an outline, write one brief section (3-6 sentences) that describe each item. Check out Fleet’s Medium blog for example release blog posts: https://medium.com/fleetdm

## Social assets

The release Medium blog post and release Tweet require new assets. Three days prior to the release date, the individual managing the release will send a message to #grupo-growth channel that requests the release assets.

This message should include the release title, summary, and a link to the release issue. The summary serves as an outline for what the release assets should depict. 

If the Growth team requires additional information, they should reference the Changelog section of the release issue.

## Release week

On the first day of the week of the scheduled release, typically a Monday, the individual tasked with managing the release should send a message to the #grupo-core channel that includes the changes planned for the release that are still outstanding. This can be in the form of a list where each list item includes the GitHub issue’s title and link. This way, there the team can discuss which changes are still feasible.

Following discussion, any items that are now pushed to a later release should be removed from the current release milestone.

At this point, the individual managing the release should reserve 30 minutes to complete the release process. This meeting typically occurs on a Wednesday at 9a PST.

### Manual QA

After all changes required for release have been merged into the `master` branch, the individual tasked with managing the release should perform a manual quality assurance pass. 

Documentation on conducting the manual QA pass can be found [here](./manual-qa.md). 

## Release day

Documentation on completing the release process can be found [here](../docs/3-Contributing/5-Releasing-Fleet.md).  

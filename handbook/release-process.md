# Releases

This living document outlines the release process at Fleet.

The current release cadence is once every 3 weeks and concentrated around Wednesdays. 

- [Blog post](#release-day)

## Blog post

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

**Upgrade plan** - Once sentence that links to user to the upgrading Fleet documentation here: https://github.com/fleetdm/fleet/blob/main/docs/1-Using-Fleet/8-Updating-Fleet.md

### Manual QA

After all changes required for release have been merged into the `main` branch, the individual tasked with managing the release should perform a manual quality assurance pass. 

Documentation on conducting the manual QA pass can be found [here](./manual-qa.md). 

## Release day

Documentation on completing the release process can be found [here](../docs/3-Contributing/5-Releasing-Fleet.md).  

<meta name="maintainedBy" value="mike-j-thomas">
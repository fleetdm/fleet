# Manual QA

This living document outlines the manual quality assurance process conducted to ensure each release of Fleet meets organization standards.

All steps should be conducted during each QA pass. All steps are possible with `fleetctl preview`. In order to target a specific version of `fleetctl preview`, the tag argument can be used together with the commit you are targeting as long as that commit is represented by a tag in [docker hub](https://hub.docker.com/r/fleetdm/fleet/tags?page=1&ordering=last_updated). Without tag argument, `fleetctl preview` defaults to latest.

As new features are added to Fleet, new steps and flows will be added.

## Collecting bugs

The goal of manual QA is to catch unexpected behavior prior to release. All Manual QA steps should be possible using `fleetctl preview`. Please refer to [docs/03-Contributing/02-Testing.md](https://github.com/fleetdm/fleet/blob/main/docs/03-Contributing/02-Testing.md) for flows that cannot be completed using `fleetctl preview`.

Please start the manual QA process by creating a blank GitHub issue. As you complete each of the flows, record a list of the bugs you encounter in this new issue. Each item in this list should contain one sentence describing the bug and a screenshot if the item is a frontend bug.

## Fleet UI

For all following flows, please refer to the [permissions documentation](https://fleetdm.com/docs/using-fleet/permissions) to ensure that actions are limited to the appropriate user type. Any users with access beyond what this document lists as availale should be considered a bug and reported for either documentation updates or investigation.

### Set up flow

Successfully set up `fleetctl preview` using the preview steps outlined [here](https://fleetdm.com/get-started)

### Login and logout flow

Successfully logout and then login to your local Fleet.

### Host details page

Select a host from the "Hosts" table as a global user with the Maintainer role. You may create a user with a fake email for this purpose.

You should be able to see and select the "Delete" button on this host's **Host details** page.

You should be able to see and select the "Query" button on this host's **Host details** page.

### Label flow

`Flow is covered by e2e testing`

Create a new label by selecting "Add a new label" on the Hosts page. Make sure it correctly filters the host on the hosts page.

Edit this label. Confirm users can only edit the "Name" and "Description" fields for a label. Users cannot edit the "Query" field because label queries are immutable.

Delete this label.

### Query flow

`Flow is covered by e2e testing`

Create a new saved query.

Run this query as a live query against your local machine.

Edit this query and then delete this query.

### Pack flow

`Flow is covered by e2e testing`

Create a new pack (under Schedule/advanced).

Add a query as a saved query to the pack. Remove this query. Delete the pack.


### My account flow

Head to the My account page by selecting the dropdown icon next to your avatar in the top navigation. Select "My account" and successfully update your password. Please do this with an extra user created for this purpose to maintain accessibility of `fleetctl preview` admin user.


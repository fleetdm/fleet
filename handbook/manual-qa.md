# Manual QA

This living document outlines the manual quality assurance process conducted to ensure each release of Fleet meets organization standards.

All steps should be conducted during each QA pass.

As new features are added to Fleet, new steps and flows will be added.

> Note: Currently, the testing focuses on the Fleet UI. As this document grows, more testing procedures for the fleetctl CLI tool will be added.


## Collecting bugs

The goal of manual QA is to catch unexpected behavior prior to release. 

Please start the manual QA process by creating a blank GitHub issue. As you complete each of the flows, record a list of the bugs you encounter in this new issue. Each item in this list should contain one sentence describing the bug and a screenshot if the item is a frontend bug.

### Clear your local MySQL database

Before you fire up your local Fleet server, wipe your local MySQL database by running the following command:

```
docker volume rm fleet_mysql-persistent-volume
```

If you receive an error that says "No such volume," double check that the MySQL volume doesn't have a different name by running this command:

```
docker volume ls
```

### Start your development server

Next, fire up your local Fleet server. Check out [this Loom video](https://www.loom.com/share/e7439f058eb44c45af872abe8f8de4a1) for instructions on starting up your local development environment.

### Set up flow

Successfully set up Fleet.

### Login and logout flow

Successfully logout and then login to your local Fleet.

### Enroll host flow

Enroll your local machine to Fleet. Check out the [Orbit for osquery documentation](../docs/2-Orbit-osquery/README.md#packaging) for instructions on generating and installing an Orbit package.

### Host page

To populate the Fleet UI with more than just one host you'll need to use the [fleetdm/osquery-perf tool](https://github.com/fleetdm/osquery-perf/tree/629a7efb6097f9108f706ccd45828793ff73cf9c).

First, clone the fleetdm/osquery perf repo and then run the following commands from the top level of the cloned directory:

```
go run agent.go --host_count 200 --enroll_secret <your enroll secret goes here>
```

After about 10 seconds, the Fleet UI should be populated with 200 simulated hosts.

### Label flow
`Flow is covered by e2e testing`

Create a new label by selecting "Add a new label" on the Hosts page. Make sure it correctly filters the host on the hosts page.

Edit this label and then delete this label.

### Query flow
`Flow is covered by e2e testing`

Create a new saved query.

Run this query as a live query against your local machine.

Edit this query and then delete this query.

### Pack flow
`Flow is covered by e2e testing`

Create a new pack.

Add a query as a saved query to the pack. Remove this query. Delete the pack.

### Organization settings flow

As an admin user, select the "Settings" tab in the top navigation and then select "Organization settings". 

Follow [the instructions outlined in the Testing documentation](../docs/4-Contribution/2-Testing.md#email) to set up a local SMTP server.

Successfully edit your organization's name in Fleet.

### Manage users flow

Invite a new user. To be able to invite users, you must have your local SMTP server configured. Instructions for setting up a local SMTP server are outlined in [the Testing documentation](../docs/4-Contribution/2-TEsting.md#email)

Logout of your current admin user and accept the invitation for the newly invited user. With your local SMTP server configured, head to https://localhost:8025 to view and select the invitation link.

### Agent options flow

Head to the global agent options page and set the `distributed_iterval` field to `5`. 

Refresh the page to confirm that the agent options have been updated.

### My account flow

Head to the My account page by selecting the dropdown icon next to your avatar in the top navigation. Select "My account" and successfully update your password.
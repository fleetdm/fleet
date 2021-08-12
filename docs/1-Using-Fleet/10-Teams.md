# Teams

`Applies only to Fleet Basic`

```
ℹ️  In Fleet 4.0, Teams were introduced.
```

- [View teams](#view-teams)
- [Create a team](#create-a-teams)
- [Enroll hosts to a team](#enroll-hosts-to-a-team)
- [Transfer hosts to a team](#transfer-hosts-to-a-team)
- [Add users to a team](#add-users-to-a-team)
- [Remove a member from a team](#remove-a-member-from-a-team)
- [Remove a team](#remove-a-team)

In Fleet, you can group hosts together in a team.

With hosts segmented into exclusive teams, you can apply specific queries, packs, and agent options to each team.

For example, you might create a team for each type of system in your organization. You can name the teams `Workstations`, `Workstations - sandbox`, `Servers`, and `Servers - sandbox`.

> A popular pattern is to end a team’s name with “- sandbox”, then you can use this to test new queries and configuration with staging hosts or volunteers acting as canaries.

Then you can:

- Enroll hosts to one team using team specific enroll secrets

- Apply unique agent options to each team

- Schedule queries that target one or more teams

- Run live queries against one or more teams

- Grant users access to one or more

## View teams

To view teams:

In the top navigation select "Settings" and then "Teams."

## Create a team

To create a team:

1. In the top navigation select "Settings" and then, in the sub-navigation, select "Teams."

2. To the left of the search box, select "Create team."

3. Enter your new team's name and select "Save."

## Enroll hosts to a team

Hosts can only belong to one team in Fleet.

You can transfer hosts to a new team in Fleet by either enrolling the host with a team's enroll secret or by [transferring the host via the Fleet UI](#transfer-hosts-to-a-team) after the host has been enrolled to Fleet.

To enroll hosts to a team:

1. In the top navigation, select "Hosts" and the on the right side, select "Enroll new host."

2. In the "Enroll secret" section of the modal, select the team you'd like to transfer your hosts to.

3. Copy or download the team's enroll secret. Use this enroll secret when installing the osquery agents on your hosts to Fleet.

Orbit is the recommended agent for Fleet. Check out [the Orbit for osquery documentation](https://github.com/fleetdm/fleet/blob/main/orbit/README.md) for instructions for packaging and deploying Orbit to your hosts.

## Transfer hosts to a team

Hosts can be transferred to a different team they've has been enrolled to Fleet.

To transfer a host to a team:

1. In the top navigation, select "Hosts."

2. Using the checkboxes in the Hosts table, select the hosts you'd like to transfer.

3. In the Hosts table header select "Transfer to team."

4. Choose the team you'd like to transfer the hosts to and confirm the action.

## Add users to a team

Global users cannot be added to a team.

To add users to a team:

1. In the top navigation, select "Settings" and then, in the sub-navigation, select "Teams."

2. Find your team and select it.

3. To the left of the search box, select "Add member."

4. Select one or more users by searching for their full name and confirm the action.

Users will be given the [Observer role](./9-Permissions.md#team-member-permissions) when added to the team. The [Edit a member's role](#edit-a-members-role) provides instructions on changing the permission level of users on a team.

## Edit a member's role

To edit a member's role:

1. In the top navigation, select "Settings" and then, in the sub-navigation, select "Teams."

2. Find your team and select it.

3. In the Members table, select the "Actions" button for the user you'd like to edit and then select "Edit."

4. In the Teams section of the form, to the right of the team you'd like to change the users role on, select "Observer" (this may also say "Maintainer") and then select the new role.

5. Confirm the action.

## Remove a member from a team

To remove a member from a team:

1. In the top navigation, select "Settings" and then, in the sub-navigation, select "Teams."

2. Find your team and select it.

3. In the Members table, select the "Actions" button for the user you'd like to edit and then select "Remove."

4. Confirm the action.

## Delete a team

To delete a team:

1. In the top navigation, select "Settings" and then, in the sub-navigation, select "Teams."

2. Find your team and select it.

3. On the right side, select "Delete team" and confirm the action.

# Downgrade from Fleet Premium

Follow these steps to downgrade your Fleet instance from Fleet Premium.

> If you'd like to renew your Fleet Premium license key, please contact us [here](https://fleetdm.com/company/contact).

## Back up your users and update all team-level users to global users

1. Run the `fleetctl get user_roles > user_roles.yml` command. Save the `user_roles.yml` file so that you can restore user roles if you decide to upgrade later.
2. Head to the **Settings > Users** page in the Fleet UI.
3. For each user that has any team listed under the **Teams** column, select **Actions > Edit**, then select **Global user**, and then **Save**. Delete any users that shouldn't have global access.

## Move all team-level queries to the global level

1. Head to the **Queries** page in the Fleet UI and select a team from the teams dropdown at the top of the page. 
2. For each query that belongs to a team, select the query and select **Edit query** and copy the **Name**, **Description**, **Query**. Then expand the "advanced options" and take note of the values in the **Platforms**, **Minimum osquery version**, and **Logging** dropdowns.
3. On the Queries page select **All teams** in the top dropdown, select **Add query**, paste each item in the appropriate field, select the correct values from the advanced options dropdowns, and select **Save**.
4. **Optional:** Delete each query that belongs to a team because they will no longer be accessible in the Fleet UI following the downgrade process.

## Move all team-level policies to the global level

1. Head to the **Policies** page in the Fleet UI.
2. For each policy that belongs to a team, copy the **Name**, **Description**, **Resolve**, and **Query**. Then, select **All teams** in the top dropdown, select **Add a policy**, select **Create your own policy**, paste each item in the appropriate field, and select **Save**.
3. Delete each policy that belongs to a team because they will no longer run on any hosts following the downgrade process.

## Back up your teams

1. Run the `fleetctl get teams > teams.yml` command. Save the `teams.yml` file so you can restore your teams if you upgrade again later.
2. Head to the **Settings > Teams** page in the Fleet UI.
3. Delete all teams. This will move all hosts to the global level.

## Remove your Fleet Premium license key

1. Remove your license key from your Fleet configuration. Documentation on where the license key is located in your configuration is [here](https://fleetdm.com/docs/deploying/configuration#license).
2. Restart your Fleet server.

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="eashaw">
<meta name="authorFullName" value="Eric Shaw">
<meta name="publishedOn" value="2024-01-09">
<meta name="articleTitle" value="Downgrade from Fleet Premium">
<meta name="description" value="Learn how to downgrade from Fleet Premium.">
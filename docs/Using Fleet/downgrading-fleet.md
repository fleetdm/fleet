# Downgrading from Fleet Premium

Follow these steps to downgrade your Fleet instance from Fleet Premium to Fleet Free.

> If you'd like to renew your Fleet Premium license key, please contact us [here](https://fleetdm.com/company/contact).

## Back up your users and update all team-level users to global users

1. Run the `fleetctl get user_roles > user_roles.yml` command. Save the `user_roles.yml` file so that, if you choose to upgrade later, you can restore user roles.
2. Head to the **Settings > Users** page in the Fleet UI.
3. For each user that has any team listed under the **Teams** column, select **Actions > Edit**, then select **Global user**, and then select **Save**. If a user shouldn't have global access, delete this user.

## Move all team-level scheduled queries to the global level

1. Head to the **Schedule** page in the Fleet UI.
2. For each scheduled query that belongs to a team, copy the name in the **Query** column, select **All teams** in the top dropdown, select **Schedule a query**, past the name in the **Select query** field, choose the frequency, and select **Schedule**.
3. Delete each scheduled query that belongs to a team because they will no longer run on any hosts following the downgrade process.

## Move all team-level policies to the global level

1. Head to the **Policies** page in the Fleet UI.
2. For each policy that belongs to a team, copy the **Name**, **Description**, **Resolve**, and **Query**. Then, select **All teams** in the top dropdown, select **Add a policy**, select **Create your own policy**, paste each item in the appropriate field, and select **Save**.
3. Delete each policy that belongs to a team because they will no longer run on any hosts following the downgrade process.

## Back up your teams

1. Run the `fleetctl get teams > teams.yml` command. Save the `teams.yml` file so that, if you choose to upgrade later, you can restore teams.
2. Head to the **Settings > Teams** page in the Fleet UI.
3. Delete all teams. This will move all hosts to the global level.

## Remove your Fleet Premium license key

1. Remove your license key from your Fleet configuration. Documentation on where the license key is located in your configuration is [here](https://fleetdm.com/docs/deploying/configuration#license).
2. Restart your Fleet server.



<meta name="title" value="Downgrading Fleet">
<meta name="navSection" value="Dig deeper">
<meta name="pageOrderInSection" value="2000">
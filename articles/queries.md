# Queries

Queries in Fleet allow you to ask questions to help you manage, monitor, and identify threats on your devices. This guide will walk you through how to create, schedule, and run a query.

> Note: Unless a logging infrastructure is configured on your Fleet server, osquery-related logs will be stored locally on each device. Read more [here](https://fleetdm.com/guides/log-destinations)

> New users may find it helpful to start with Fleet's policies. You can find policies and queries from the community in Fleet's [query library](https://fleetdm.com/queries). To learn more about policies, see [What are Fleet policies?](https://fleetdm.com/securing/what-are-fleet-policies) and [Understanding the intricacies of Fleet policies](https://fleetdm.com/guides/understanding-the-intricacies-of-fleet-policies).

### In this guide:

- [Create a query](#create-a-query)
- [View a query report](#view-a-query-report)
- [Run a query](#run-a-query)
- [Schedule a query](#schedule-a-query)

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/1VNvg3_drow" allowfullscreen></iframe>
</div>



## Create a query

How to create a query:

1. In the top navigation, select **Queries**.

2. Select **Create new query** to navigate to the query console.

3. In the **Query** field, enter your query. Remember, you can find common queries in [Fleet's library](https://fleetdm.com/queries).
> Avoid using dot notation (".") for column names in your queries as it can cause results to render incorrectly in Fleet UI. Please see [issue #15446](https://github.com/fleetdm/fleet/issues/15446) for more details. 

4. Select **Save**, enter a name and description for your query, select the frequency that the query should run at, and select **Save query**.


## View a query report

Fleet will store up to 1000 results for each scheduled query to give users a snapshot of query results. If the number of results for a scheduled query is below 1000, then the results will continuously get updated every time the hosts send results to Fleet.

As you enable query reports, it is advisable to monitor your database to determine if it needs to be scaled up. As an alternative, you can disable query reports.

> To disable query reports globally, modify `server_settings.query_reports_disabled` field in the global configuration. To disable reports for individual queries, use the `discard_data` field.

How to view a query report:

1. In the top navigation, select **Queries**.

2. In the **Queries** table, find the query you'd like to run and select the query's name to navigate to the query console.

3. If you want to download the query report, select **Export results** to save it as a CSV.


## Run a query

Run a live query to get answers for all of your online hosts.

> Offline hosts won’t respond to a live query because they may be shut down, asleep, or not connected to the internet.

How to run a query:

1. In the top navigation, select **Queries**.

2. In the **Queries** table, find the query you'd like to run and select the query's name to navigate to the query console.

3. Select **Live query** to navigate to the target picker. Select **All hosts** and select **Run**. This will run the query against all your hosts.

4. If you want to download the live query results, select **Export results** to save it as a CSV.

> Fleet 4.24.0 and later versions provide notifications in the activity feed for live queries. 

The query may take several seconds to complete because Fleet has to wait for the hosts to respond with results.

> Fleet's query response time is inherently variable because of osquery's heartbeat response time. This helps prevent performance issues on hosts.

## Schedule a query

*In Fleet 4.35.0, the "Schedule" page was removed, and query automations are now configured on the "Queries" page. Instructions for scheduling queries in earlier versions of Fleet can be found [here](https://github.com/fleetdm/fleet/blob/ac797c8f81ede770853c25fd04102da9f5e109bf/docs/Using-Fleet/Fleet-UI.md#schedule-a-query).*

Fleet allows you to schedule queries to run at a set frequency. By default, queries that run on a schedule will only target platforms compatible with that query. This behavior can be overridden by setting the platforms in **Advanced options** when saving a query.

Scheduled queries will send data to Fleet and/or your [log destination](https://fleetdm.com/docs/using-fleet/log-destinations) automatically. Query automations can be turned off in **Advanced options** or using the bulk query automations UI.

How to configure query automations in bulk:

*Only users with the [admin role](https://fleetdm.com/docs/using-fleet/manage-access#admin) can manage query automations.*

1. In the top navigation, select **Queries**.

2. Select **Manage automations**.

3. Check the box next to the queries you want to send data to your log destination, and select **Save**. (The frequency that queries run at is set when a query is created.)

> Note: When viewing a specific [team](https://fleetdm.com/docs/using-fleet/segment-hosts) in Fleet Premium, only queries that belong to the selected team will be listed. When configuring query automations for all hosts, only global queries will be listed.

### Further reading

- [REST API documentation for queries](https://fleetdm.com/docs/rest-api/rest-api#queries)
- [Import and export queries in Fleet](https://fleetdm.com/guides/import-and-export-queries-in-fleet)
- [Using fleetctl to run a live query and how live queries work](https://fleetdm.com/guides/get-current-telemetry-from-your-devices-with-live-queries#basic-article)
- [Osquery: Consider joining against the users table](https://fleetdm.com/guides/osquery-consider-joining-against-the-users-table)


<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="authorFullName" value="Noah Talerman">
<meta name="publishedOn" value="2024-08-09">
<meta name="articleTitle" value="Queries">
<meta name="description" value="Learn how to create, run, and schedule queries, as well as update agent options in the Fleet user interface.">

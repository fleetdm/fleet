# Reports

Reports in Fleet allow you to ask questions to help you manage, monitor, and identify threats on your devices. This guide will walk you through how to create, schedule, and run a report.

> Unless a [log destination](https://fleetdm.com/guides/log-destinations) is configured, osquery logs will be stored locally on each device.

> New users may find it helpful to start with Fleet's policies. You can find policies and queries from the community in Fleet's [library](https://fleetdm.com/queries). To learn more about policies, see [What are Fleet policies?](https://fleetdm.com/securing/what-are-fleet-policies) and [Understanding the intricacies of Fleet policies](https://fleetdm.com/guides/understanding-the-intricacies-of-fleet-policies).

### In this guide:

- [Create a report](#create-a-report)
- [View a report](#view-a-report)
- [Run a report](#run-a-report)
- [Schedule a report](#schedule-a-report)

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/07ErAAahRsg" allowfullscreen></iframe>
</div>



## Create a report

How to create a report:

1. In the top navigation, select **Reports** and **Add report**.

2. In the **Query** field, enter your query. Remember, you can find common reports in [Fleet's library](https://fleetdm.com/queries).
> Avoid using dot notation (".") for column names in your queries as it can cause results to render incorrectly in Fleet UI. Please see [issue #15446](https://github.com/fleetdm/fleet/issues/15446) for more details. 

4. Select **Save**, enter a name and description for your report, select the interval that the report should run at, and select **Save**.

## Targeting hosts using labels

_Available in Fleet Premium._

When creating or editing a report, you can restrict the set of hosts that it will run on by using [labels](https://fleetdm.com/guides/managing-labels-in-fleet).  By default, a new report will target all hosts, indicated by the **All Hosts** option being selected beneath the **Targets** setting.  If you select **Custom** instead, you will be able to select one or more labels for the report to target. Note that the report will run on any host that matches __any__ of the selected labels. To learn more about labels, see [Managing labels in Fleet](https://fleetdm.com/guides/managing-labels-in-fleet).

## View a report

How to view a report:

1. In the top navigation, select **Reports**.

2. In the **Reports** table, find the report you'd like to run and select the reports's name.

3. If you want to download the report, select **Export results** to save it as a CSV.

Fleet stores up to 1,000 results per report. If the count stays below this limit, Fleet updates the report each time hosts send new data.

If the results exceed 1,000, Fleet stops updating the report. To start collecting data again, clear the stored results from the report's page. Go to **Advanced options**, check **Discard data**, and select **Save**. Then uncheck **Discard data** and select **Save** again.

> You can change the 1,000-result limit by setting [`server_settings.report_cap`](https://fleetdm.com/docs/rest-api/rest-api#server-settings).

Persisting results within Fleet creates load on the database, so you'll want to monitor database load as you add queries. If needed, you can disable stored results either globally or per-report.

* Globally via the UI: **Settings** > **Advanced options** > **Disable stored results**
* Globally via the API: set [`server_settings.discard_reports_data`](https://fleetdm.com/docs/rest-api/rest-api#server-settings)
* Per-report via the UI: **Edit report** > **Show advanced options** > **Discard data**
* Per-report via the API: Set the `discard_data` field when [creating](https://fleetdm.com/docs/rest-api/rest-api#create-query) or [modifying](https://fleetdm.com/docs/rest-api/rest-api#modify-query) the report

## Run a report

Run a live report to get answers for all of your online hosts.

> Offline hosts won’t respond to a live report because they may be shut down, asleep, or not connected to the internet.

How to run a report:

1. In the top navigation, select **Reports**.

2. In the **Reports** table, find the report you'd like to run and select the reports's name.

3. Select **Live report** to navigate to the target picker. Select **All hosts** and select **Run**. This will run the report against all your hosts.

4. If you want to download the results, select **Export results** to save it as a CSV.

The report may take several seconds to complete because Fleet has to wait for the hosts to respond with results.

> Response time is inherently variable because of osquery's heartbeat response time. This helps prevent performance issues on hosts.

## Schedule a report

Fleet allows you to schedule queries to run at a set interval. By default, queries that run on a schedule will only target platforms compatible with that report. This behavior can be overridden by setting the platforms in **Advanced options** when saving a report.

To create a scheduled report, set the interval to a value other than "Never" when [creating a report](#create-a-report). If the report has already been created, select the report and then select **Edit report** to set the interval.

Scheduled reports will send data to Fleet and/or your [log destination](https://fleetdm.com/docs/using-fleet/log-destinations) automatically. Automations can be turned off in **Advanced options** or using the bulk **Manage automations** UI.

How to configure automations in bulk:

*Only users with the [admin role](https://fleetdm.com/docs/using-fleet/manage-access#admin) can manage report automations.*

1. In the top navigation, select **Reports**.

2. Select **Manage automations**.

3. Check the box next to the queries you want to send data to your log destination, and select **Save**. (The interval that queries run at is set when a report is created.)

> Note: When viewing a specific [fleet](https://fleetdm.com/docs/using-fleet/segment-hosts) in Fleet Premium, only queries that belong to the selected fleet will be listed. When configuring automations for all hosts, only global reports will be listed.

### Further reading

- [REST API documentation for queries](https://fleetdm.com/docs/rest-api/rest-api#queries)
- [Import and export queries in Fleet](https://fleetdm.com/guides/import-and-export-queries-in-fleet)
- [Using fleetctl to run a live report and how live queries work](https://fleetdm.com/guides/get-current-telemetry-from-your-devices-with-live-queries#basic-article)
- [Osquery: Consider joining against the users table](https://fleetdm.com/guides/osquery-consider-joining-against-the-users-table)


<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="authorFullName" value="Noah Talerman">
<meta name="publishedOn" value="2025-01-01">
<meta name="articleTitle" value="Queries">
<meta name="description" value="Learn how to create, run, and schedule reports, as well as update agent options in the Fleet user interface.">

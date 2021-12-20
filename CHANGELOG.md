## Fleet 4.7.0 (Dec 14, 2021)

* Add ability to create, modify, or delete policies in Fleet without modifying saved queries. Fleet
  4.7.0 introduces breaking changes to the `/policies` API routes to separate policies from saved
  queries in Fleet. These changes will not affect any policies previously created or modified in the
  Fleet UI.

* Turn on vulnerability processing for all Fleet instances with software inventory enabled.
  [Vulnerability processing in Fleet](https://fleetdm.com/docs/using-fleet/vulnerability-processing)
  provides the ability to see all hosts with specific vulnerable software installed. 

* Improve the performance of the "Software" table on the **Home** page.

* Improve performance of the MySQL database by changing the way a host's users information   is saved.

* Add ability to select from a library of standard policy templates on the **Policies** page. These
  pre-made policies ask specific "yes" or "no" questions about your hosts. For example, one of
  these policy templates asks "Is Gatekeeper enabled on macOS devices?"

* Add ability to ask whether or not your hosts have a specific operating system installed by
  selecting an operating system policy on the **Host details** page. For example, a host that is
  running macOS 12.0.1 will present a policy that asks "Is macOS 12.0.1 installed on macOS devices?"

* Add ability to specify which platform(s) (macOS, Windows, and/or Linux) a policy is checked on.

* Add ability to generate a report that includes which hosts are answering "Yes" or "No" to a 
  specific policy by running a policy's query as a live query.

* Add ability to see the total number of installed software software items across all your hosts.

* Add ability to see an example scheduled query result that is sent to your configured log
  destination. Select "Schedule a query" > "Preview data" on the **Schedule** page to see the 
  example scheduled query result.

* Improve the host's users information by removing users without login shells and adding users 
  that are not associated with a system group.

* Add ability to see a Fleet instance's missing migrations with the `fleetctl debug migrations`
  command. The `fleet serve` and `fleet prepare db` commands will now fail if any unknown migrations
  are detected.

* Add ability to see syntax errors as your write a query in the Fleet UI.

* Add ability to record a policy's resolution steps that can be referenced when a host answers "No" 
  to this policy.

* Add server request errors to the Fleet server logs to allow for troubleshooting issues with the 
Fleet server in non-debug mode.

* Increase default login session length to 24 hours.

* Fix a bug in which software inventory and disk space information was not retrieved for Debian hosts.

* Fix a bug in which searching for targets on the **Edit pack** page negatively impacted performance of 
  the MySQL database.

* Fix a bug in which some Fleet migrations were incompatible with MySQL 8.

* Fix a bug that prevented the creation of osquery installers for Windows (.msi) when a non-default 
  update channel is specified.

* Fix a bug in which the "Software" table on the home page did not correctly filtering when a
  specific team was selected on the **Home** page.

* Fix a bug in which users with "No access" in Fleet were presented with a perpetual 
  loading state in the Fleet UI.

## Fleet 4.6.2 (Nov 30, 2021)

* Improve performance of the **Home** page by removing total hosts count from the "Software" table.

* Improve performance of the **Queries** page by adding pagination to the list of queries.

* Fix a bug in which the "Shell" column of the "Users" table on the **Host details** page would sometimes fail to update.

* Fix a bug in which a host's status could quickly alternate between "Online" and "Offline" by increasing the grace period for host status.

* Fix a bug in which some hosts would have a missing `host_seen_times` entry.

* Add an `after` parameter to the [`GET /hosts` API route](https://fleetdm.com/docs/using-fleet/rest-api#list-hosts) to allow for cursor pagination.

* Add a `disable_failing_policies` parameter to the [`GET /hosts` API route](https://fleetdm.com/docs/using-fleet/rest-api#list-hosts) to allow the API request to respond faster if failing policies count information is not needed.

## Fleet 4.6.1 (Nov 21, 2021)

* Fix a bug (introduced in 4.6.0) in which Fleet used progressively more CPU on Redis, resulting in API and UI slowdowns and inconsistency.

* Make `fleetctl apply` fail when the configuration contains invalid fields.

## Fleet 4.6.0 (Nov 18, 2021)

* Fleet Premium: Add ability to filter aggregate host data such as platforms (macOS, Windows, and Linux) and status (online, offline, and new) the **Home** page. The aggregate host data is also available in the [`GET /host_summary API route`](https://fleetdm.com/docs/using-fleet/rest-api#get-hosts-summary).

* Fleet Premium: Add ability to move pending invited users between teams.

* Fleet Premium: Add `fleetctl updates rotate` command for rotation of keys in the updates system. The `fleetctl updates` command provides the ability to [self-manage an agent update server](https://fleetdm.com/docs/deploying/fleetctl-agent-updates).

* Enable the software inventory by default for new Fleet instances. The software inventory feature can be turned on or off using the [`enable_software_inventory` configuration option](https://fleetdm.com/docs/using-fleet/vulnerability-processing#setup).

* Update the JSON payload for the host status webhook by renaming the `"message"` property to `"text"` so that the payload can be received and displayed in Slack.

* Remove the deprecated `app_configs` table from Fleet's MySQL database. The `app_config_json` table has replaced it.

* Improve performance of the policies feature for Fleet instances with over 100,000 hosts.

* Add instructions in the Fleet UI for generating an osquery installer for macOS, Linux, or Windows. Documentation for generating an osquery installer and distributing the installer to your hosts to add them to Fleet can be found here on [fleetdm.com/docs](https://fleetdm.com/docs/using-fleet/adding-hosts)

* Add ability to see all the software, and filter by vulnerable software, installed across all your hosts on the **Home** page. Each software's `name`, `version`, `hosts_count`, `vulnerabilities`, and more is also available in the [`GET /software` API route](https://fleetdm.com/docs/using-fleet/rest-api#software) and `fleetctl get software` command.

* Add ability to add, edit, and delete enroll secrets on the **Hosts** page.

* Add ability to see aggregate host data such as platforms (macOS, Windows, and Linux) and status (online, offline, and new) the **Home** page.

* Add ability to see all of the queries scheduled to run on a specific host on the **Host details** page immediately after a query is added to a schedule or pack.

* Add a "Shell" column to the "Users" table on the **Host details** page so users can now be filtered to see only those who have logged in.

* Package osquery's `certs.pem` in `fleetctl package` to improve TLS compatibility.

* Add support for packaging an osquery flagfile with `fleetctl package --osquery-flagfile`.

* Use "Fleet osquery" rather than "Orbit osquery" in packages generated by `fleetctl package`.

* Clarify that a policy in Fleet is a yes or no question you can ask about your hosts by replacing "Passing" and "Failing" text with "Yes" and "No" respectively on the **Policies** page and **Host details** page.

* Add ability to see the original author of a query on the **Query** page.

* Improve the UI for the "Software" table and "Policies" table on the **Host details** page so that it's easier to pivot to see all hosts with a specific software installed or answering "No" to a specific policy.

* Fix a bug in which modifying a specific target for a live query, in target selector UI, would deselect a different target.

* Fix a bug in which the user was navigated to a non existent page, in the Fleet UI, after saving a pack.

* Fix a bug in which long software names in the "Software" table caused the bundle identifier tooltip to be inaccessible.

## Fleet 4.5.1 (Nov 10, 2021)

* Fix performance issues with search filtering on manage queries page.

* Improve correctness and UX for query platform compatibility.

* Fleet Premium: Show correct hosts when a team is selected.

* Fix a bug preventing login for new SSO users.

* Always return the `disabled` value in the `GET /api/v1/fleet/packs/{id}` API (previously it was
  sometimes left out).

## Fleet 4.5.0 (Nov 1, 2021)

* Fleet Premium: Add a Team admin user role. This allows users to delegate the responsibility of managing team members in Fleet. Documentation for the permissions associated with the Team admin and other user roles can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/using-fleet/permissions).

* Add Apache Kafka logging plugin. Documentation for configuring Kafka as a logging plugin can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/deploying/configuration#kafka-rest-proxy-logging). Thank you to Joseph Macaulay for adding this capability.

* Add support for [MinIO](https://min.io/) as a file carving backend. Documentation for configuring MinIO as a file carving backend can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/using-fleet/fleetctl-cli#minio). Thank you to Chandra Majumdar and Ben Edwards for adding this capability.

* Add support for generating a `.pkg` osquery installer on Linux without dependencies (beyond Docker) with the `fleetctl package` command.

* Improve the performance of vulnerability processing by making the process consume less RAM. Documentation for the vulnerability processing feature can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/using-fleet/vulnerability-processing).

* Add the ability to run a live query and receive results using only the Fleet REST API with a `GET /api/v1/fleet/queries/run` API route. Documentation for this new API route can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/using-fleet/rest-api#run-live-query).

* Add ability to see whether a specific host is "Passing" or "Failing" a policy on the **Host details** page. This information is also exposed in the `GET api/v1/fleet/hosts/{id}` API route. In Fleet, a policy is a "yes" or "no" question you can ask of all your hosts.

* Add the ability to quickly see the total number of "Failing" policies for a particular host on the **Hosts** page with a new "Issues" column. Total "Issues" are also revealed on a specific host's **Host details** page.

* Add the ability to see which platforms (macOS, Windows, Linux) a specific query is compatible with. The compatibility detected by Fleet is estimated based on the osquery tables used in the query.

* Add the ability to see whether your queries have a "Minimal," "Considerable," or "Excessive" performance impact on your hosts. Query performance information is only collected when a query runs as a scheduled query.

  * Running a "Minimal" query, very frequently, has little to no impact on your host's performance.

  * Running a "Considerable" query, frequently, can have a noticeable impact on your host's performance.

  * Running an "Excessive" query, even infrequently, can have a significant impact on your host’s performance.

* Add the ability to see a list of hosts that have a specific software version installed by selecting a software version on a specific host's **Host details** page. Software inventory is currently under a feature flag. To enable this feature flag, check out the [feature flag documentation](./docs/02-Deploying/03-Configuration.md#feature-flags).

* Add the ability to see all vulnerable software detected across all your hosts with the `GET /api/v1/fleet/software` API route. Documentation for this new API route can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/using-fleet/rest-api#software).

* Add the ability to see the exact number of hosts that selected filters on the **Hosts** page. This ability is also available when using the `GET api/v1/fleet/hosts/count` API route.

* Add ability to automatically "Refetch" host vitals for a particular host without manually reloading the page.

* Add ability to connect to Redis with TLS. Documentation for configuring Fleet to use a TLS connection to the Redis server can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/deploying/configuration#redis-use-tls).

* Add `cluster_read_from_replica` Redis to specify whether or not to prefer readying from a replica when possible. Documentation for this configuration option can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/deploying/configuration#redis-cluster-read-from-replica).

* Improve experience of the Fleet UI by preventing autocomplete in forms.

* Fix a bug in which generating an `.msi` osquery installer on Windows would fail with a permission error.

* Fix a bug in which turning on the host expiry setting did not remove expired hosts from Fleet.

* Fix a bug in which the Software inventory for some host's was missing `bundle_identifier` information.

## Fleet 4.4.3 (Oct 21, 2021)

* Cache AppConfig in redis to speed up requests and reduce MySQL load.

* Fix migration compatibility with MySQL GTID replication.

* Improve performance of software listing query.

* Improve MSI generation compatibility (for macOS M1 and some Virtualization configurations) in `fleetctl package`.

## Fleet 4.4.2 (Oct 14, 2021)

* Fix migration errors under some MySQL configurations due to use of temporary tables.

* Fix pagination of hosts on host dashboard.

* Optimize HTTP requests on host search.

## Fleet 4.4.1 (Oct 8, 2021)

* Fix database migrations error when updating from 4.3.2 to 4.4.0. This did not effect upgrades
  between other versions and 4.4.0.

* Improve logging of errors in fleet serve.

## Fleet 4.4.0 (Oct 6, 2021)

* Fleet Premium: Teams Schedules show inherited queries from All teams (global) Schedule.

* Fleet Premium: Team Maintainers can modify and delete queries, and modify the Team Schedule.

* Fleet Premium: Team Maintainers can delete hosts from their teams.

* `fleetctl get hosts` now shows host additional queries if there are any.

* Update default homepage to new dashboard.

* Add ability to bulk delete hosts based on manual selection and applied filters.

* Display macOS bundle identifiers on software table if available.

* Fixed scroll position when navigating to different pages.

* Fleet Premium: When transferring a host from team to team, clear the Policy results for that host.

* Improve stability of host vitals (fix cases of dropping users table, disk space).

* Improve performance and reliability of Policy database migrations.

* Provide a more clear error when a user tries to delete a query that is set in a Policy.

* Fix query editor Delete key and horizontal scroll issues.

* Cleaner buttons and icons on Manage Hosts Page.

## Fleet 4.3.2 (Sept 29, 2021)

* Improve database performance by reducing the amount of MySQL database queries when a host checks in.

* Fix a bug in which users with the global maintainer role could not edit or save queries. In, Fleet 4.0.0, the Admin, Maintainer, and Observer user roles were introduced. Documentation for the permissions associated with each role can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/using-fleet/permissions). 

* Fix a bug in which policies were checked about every second and add a `policy_update_interval` osquery configuration option. Documentation for this configuration option can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/deploying/configuration#osquery-policy-update-interval).

* Fix a bug in which edits to a query’s name, description, SQL did not appear until the user refreshed the Edit query page.

* Fix a bug in which the hosts count for a label returned 0 after modifying a label’s name or description.

* Improve error message when attempting to create or edit a user with an email that already exists.

## Fleet 4.3.1 (Sept 21, 2021)

* Add `fleetctl get software` command to list all software and the detected vulnerabilities. The Vulnerable software feature is currently in Beta. For information on how to configure the Vulnerable software feature and how exactly Fleet processes vulnerabilities, check out the [Vulnerability processing documentation](https://fleetdm.com/docs/using-fleet/vulnerability-processing).

* Add `fleetctl vulnerability-data-stream` command to sync the vulnerabilities processing data streams by hand.

* Add `disable_data_sync` vulnerabilities configuration option to avoid downloading the data streams. Documentation for this configuration option can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/deploying/configuration#disable-data-sync).

* Only show observers the queries they have permissions to run on the **Queries** page. In, Fleet 4.0.0, the Admin, Maintainer, and Observer user roles were introduced. Documentation for the permissions associated with each role can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/using-fleet/permissions). 

* Add `connect_retry_attempts` Redis configuration option to retry failed connections. Documentation for this configuration option can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/deploying/configuration#redis-connect-retry-attempts).

* Add `cluster_follow_redirections` Redis configuration option to follow cluster redirections. Documentation for this configuration option can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/deploying/configuration#redis-cluster-follow-redirections).

* Add `max_jitter_percent` osquery configuration option to prevent all hosts from returning data at roughly the same time. Note that this improves the Fleet server performance, but it will now take longer for new labels to populate. Documentation for this configuration option can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/deploying/configuration#osquery-max-jitter-percent).

* Improve the performance of database migrations.

* Reduce database load for label membership recording.

* Fail early if the process does not have permissions to write to the logging file.

* Completely skip trying to save a host's users and software inventory if it's disabled to reduce database load. 

* Fix a bug in which team maintainers were unable to run live queries against the hosts assigned to their team(s).

* Fix a bug in which a blank screen would intermittently appear on the **Hosts** page.

* Fix a bug detecting disk space for hosts.

## Fleet 4.3.0 (Sept 13, 2021)

* Add Policies feature for detecting device compliance with organizational policies.

* Run/edit query experience has been completely redesigned.

* Add support for MySQL read replicas. This allows the Fleet server to scale to more hosts.

* Add configurable webhook to notify when a specified percentage of hosts have been offline for over the specified amount of days.

* Add `fleetctl package` command for building Orbit packages.

* Add enroll secret dialog on host dashboard.

* Expose free disk space in gigs and percentage for hosts.

* Add 15-minute interval option on Schedule page.

* Clean up advanced options UI.

* 404 and 500 page now include buttons for Osquery community Slack and to file an issue

* Update all empty and error states for cleaner UI.

* Add warning banners in Fleet UI and `fleetctl` for license expiration.

* Render query performance information on host vitals page pack section.

* Improved performance for app loading.

* Make team schedule names more user friendly and hide the stats for global and team schedules when showing host pack stats.

* Display `query_name` in when referencing scheduled queries for more consistent UI/UX.

* Query action added for observers on host vitals page.

* Add `server_settings.debug_host_ids` to gather more detailed information about what the specified hosts are sending to fleet.

* Allow deeper linking into the Fleet application by saving filters in URL parameters.

* Rename Basic Tier to Premium Tier, and Core Tier to Free Tier.

* Improve vulnerability detection compatibility with database configurations.

* MariaDB compatibility fixes: add explicit foreign key constraint and on cascade delete for host_software to allow for hosts with software to be deleted.

* Fix migration that was incompatible with MySQL primary key requirements (default on DigitalOcean MySQL 5.8).

* Add 30 second SMTP timeout for mail configuration.

* Fix display of platform Labels on manage hosts page

* Fix a bug recording scheduled query statistics.

* When a label is removed, ignore query executions for that label.

* Add fleet serve config to change the redis connection timeout and keep alive interval.

* Remove hardcoded limits in label searches when targeting queries.

* Allow host users to be readded.

* Move email template images from github to fleetdm.com.

* Fix bug rendering CPU in host vitals.

* Update the schema for host_users to allow for bulk inserts without locking, and allow for users without unique uid.

* When using dynamic vulnerability processing node, try to create the vulnerability.databases-path.

* Fix `fleetctl get host <hostname>` to properly output JSON when the command line flag is supplied i.e `fleetctl get host --json foobar`

## Fleet 4.2.4 (Sept 2, 2021)

* Fix a bug in which live queries would fail for deployments that use Redis Cluster.

* Fix a bug in which some new Fleet deployments don't include the default global agent options. Documentation for global and team agent options can be found [here](https://fleetdm.com/docs/using-fleet/configuration-files#agent-options).

* Improve how a host's `users` are stored in MySQL to prevent deadlocks. This information is available in the "Users" table on each host's **Host details** page and in the `GET /api/v1/fleet/hosts/{id}` API route.

## Fleet 4.2.3 (Aug 23, 2021)

* Add ability to troubleshoot connection issues with the `fleetctl debug connection` command.

* Improve compatibility with MySQL variants (MariaDB, Aurora, etc.) by removing usage of JSON_ARRAYAGG.

* Fix bug in which live queries would stop returning results if more than 5 seconds goes by without a result. This bug was introduced in 4.2.1.

* Eliminate double-logging of IP addresses in osquery endpoints.

* Update host details after transferring a host on the details page.

* Log errors in osquery endpoints to improve debugging.

## Fleet 4.2.2 (Aug 18, 2021)

* Add a new built in label "All Linux" to target all hosts that run any linux flavor.

* Allow finer grained configuration of the vulnerability processing capabilities.

* Fix performance issues when updating pack contents.

* Fix a build issue that caused external network access to panic in certain Linux distros (Ubuntu).

* Fix rendering of checkboxes in UI when modals appear.

* Orbit: sync critical file writes to disk.

* Add "-o" flag to fleetctl convert command to ensure consistent output rather than relying on shell redirection (this was causing issues with file encodings).

* Fix table column wrapping for manage queries page.

* Fix wrapping in Label pills.

* Side panels in UI have a fresher look, Teams/Roles UI greyed out conditionally.

* Improved sorting in UI tables.

* Improve detection of CentOS in label membership.

## Fleet 4.2.1 (Aug 14, 2021)

* Fix a database issue with MariaDB 10.5.4.

* Display updated team name after edit.

* When a connection from a live query websocket is closed, Fleet now timeouts the receive and handles the different cases correctly to not hold the connection to Redis.

* Read live query results from Redis in a thread safe manner.

* Allow observers and maintainers to refetch a host in a team they belong to.

## Fleet 4.2.0 (Aug 11, 2021)

* Add ability to simultaneously filter hosts by status (`online`, `offline`, `new`, `mia`) and by label on the **Hosts** page.

* Add ability to filter hosts by team in the Fleet UI, fleetctl CLI tool, and Fleet API. *Available for Fleet Basic customers*.

* Add ability to create a Team schedule in Fleet. The Schedule feature was released in Fleet 4.1.0. For more information on the new Schedule feature, check out the [Fleet 4.1.0 release blog post](https://blog.fleetdm.com/fleet-4-1-0-57dfa25e89c1). *Available for Fleet Basic customers*.

* Add Beta Vulnerable software feature which surfaces vulnerable software on the **Host details** page and the `GET /api/v1/fleet/hosts/{id}` API route. For information on how to configure the Vulnerable software feature and how exactly Fleet processes vulnerabilities, check out the [Vulnerability processing documentation](https://github.com/fleetdm/fleet/blob/main/docs/01-Using-Fleet/13-Vulnerability-Processing.md#vulnerability-processing).

* Add ability to see which logging destination is configured for Fleet in the Fleet UI. To see this information, head to the **Schedule** page and then select "Schedule a query." Configured logging destination information is also available in the `GET api/v1/fleet/config` API route.

* Improve the `fleetctl preview` experience by downloading Fleet's standard query library and loading the queries into the Fleet UI.

* Improve the user interface for the **Packs** page and **Queries** page in the Fleet UI.

* Add ability to modify scheduled queries in your Schedule in Fleet. The Schedule feature was released in Fleet 4.1.0. For more information on the new Schedule feature, check out the [Fleet 4.1.0 release blog post](https://blog.fleetdm.com/fleet-4-1-0-57dfa25e89c1).

* Add ability to disable the Users feature in Fleet by setting the new `enable_host_users` key to `true` in the `config` yaml, configuration file. For documentation on using configuration files in yaml syntax, check out the [Using yaml files in Fleet](https://github.com/fleetdm/fleet/tree/main/docs/01-Using-Fleet/configuration-files#using-yaml-files-in-fleet) documentation.

* Improve performance of the Software inventory feature. Software inventory is currently under a feature flag. To enable this feature flag, check out the [feature flag documentation](./docs/02-Deploying/03-Configuration.md#feature-flags).

* Improve performance of inserting `pack_stats` in the database. The `pack_stats` information is used to display "Frequency" and "Last run" information for a specific host's scheduled queries. You can find this information on the **Host details** page.

* Improve Fleet server logging so that it is more uniform.

* Fix a bug in which a user with the Observer role was unable to run a live query.

* Fix a bug that prevented the new **Home** page from being displayed in some Fleet instances.

* Fix a bug that prevented accurate sorting issues across multiple pages on the **Hosts** page.

## Fleet 4.1.0 (Jul 26, 2021)

The primary additions in Fleet 4.1.0 are the new Schedule and Activity feed features.

Schedule lets you add queries which are executed on your devices at regular intervals without having to understand or configure osquery query packs. For experienced Fleet and osquery users, the ability to create new, and modify existing, query packs is still available in the Fleet UI and fleetctl command-line tool. To reach the **Packs** page in the Fleet UI, head to **Schedule > Advanced**.

Activity feed adds the ability to observe when, and by whom, queries are changes, packs are created, live queries are run, and more. The Activity feed feature is located on the new Home page in the Fleet UI. Select the logo in the top right corner of the Fleet UI to navigate to the new **Home** page.

### New features breakdown

* Add ability to create teams and update their respective agent options and enroll secrets using the new `teams` yaml document and fleetctl. Available in Fleet Basic.

* Adds a new **Home** page to the Fleet UI. The **Home** page presents a breakdown of the enrolled hosts by operating system.

* Add a "Users" table on the **Host details** page. The `username` information displayed in the "Users" table, as well as the `uid`, `type`, and `groupname` are available in the Fleet REST API via the `/api/v1/fleet/hosts/{id}` API route.

* Add ability to create a user without an invitation. You can now create a new user by heading to **Settings > Users**, selecting "Create user," and then choosing the "Create user" option.

* Add ability to search and sort installed software items in the "Software" table on the **Host details** page. 

* Add ability to delete a user from Fleet using a new `fleetctl user delete` command.

* Add ability to retrieve hosts' `status`, `display_text`, and `labels` using the `fleetctl get hosts` command.

* Add a new `user_roles` yaml document that allows users to manage user roles via fleetctl. Available in Fleet Basic.

* Change default ordering of the "Hosts" table in the Fleet UI to ascending order (A-Z).

* Improve performance of the Software inventory feature by reducing the amount of inserts and deletes are done in the database when updating each host's
software inventory.

* Remove YUM and APT sources from Software inventory.

* Fix an issue in which disabling SSO at the organization level would not disable SSO for all users.

* Fix an issue with data migrations in which enroll secrets are duplicated after the `name` column was removed from the `enroll_secrets` table.

* Fix an issue in which it was not possible to clear host settings by applying the `config` yaml document. This allows users to successfully remove the `additional_queries` property after adding it.

* Fix printing of failed record count in AWS Kinesis/Firehose logging plugins.

* Fix compatibility with GCP Memorystore Redis due to missing CLUSTER command.


## Fleet 4.0.1 (Jul 01, 2021)

* Fix an issue in which migrations failed on MariaDB MySQL.

* Allow `http` to be used when configuring `fleetctl` for `localhost`.

* Fix a bug in which Team information was missing for hosts looked up by Label. 

## Fleet 4.0.0 (Jun 29, 2021)

The primary additions in Fleet 4.0.0 are the new Role-based access control (RBAC) and Teams features. 

RBAC adds the ability to define a user's access to features in Fleet. This way, more individuals in an organization can utilize Fleet with appropriate levels of access.

* Check out the [permissions documentation](https://github.com/fleetdm/fleet/blob/2f42c281f98e39a72ab4a5125ecd26d303a16a6b/docs/1-Using-Fleet/9-Permissions.md) for a breakdown of the new user roles.

Teams adds the ability to separate hosts into exclusive groups. This way, users can easily act on consistent groups of hosts. 

* Read more about the Teams feature in [the documentation here](https://github.com/fleetdm/fleet/blob/2f42c281f98e39a72ab4a5125ecd26d303a16a6b/docs/1-Using-Fleet/10-Teams.md).

### New features breakdown

* Add ability to define a user's access to features in Fleet by introducing the Admin, Maintainer, and Observer roles. Available in Fleet Core.

* Add ability to separate hosts into exclusive groups with the Teams feature. The Teams feature is available for Fleet Basic customers. Check out the list below for the new functionality included with Teams:

* Teams: Add ability to enroll hosts to one team using team specific enroll secrets.

* Teams: Add ability to manually transfer hosts to a different team in the Fleet UI.

* Teams: Add ability to apply unique agent options to each team. Note that "osquery options" have been renamed to "agent options."

* Teams: Add ability to grant users access to one or more teams. This allows you to define a user's access to specific groups of hosts in Fleet.

* Add ability to create an API-only user. API-only users cannot access the Fleet UI. These users can access all Fleet API endpoints and `fleetctl` features. Available in Fleet Core.

* Add Redis cluster support. Available in Fleet Core.

* Fix a bug that prevented the columns chosen for the "Hosts" table from persisting after logging out of Fleet.

### Upgrade plan

Fleet 4.0.0 is a major release and introduces several breaking changes and database migrations. The following sections call out changes to consider when upgrading to Fleet 4.0.0:

* The structure of Fleet's `.tar.gz` and `.zip` release archives have changed slightly. Deployments that use the binary artifacts may need to update scripts or tooling. The `fleetdm/fleet` Docker container maintains the same API.

* Use strictly `fleet` in Fleet's configuration, API routes, and environment variables. Users must update all usage of `kolide` in these items (deprecated since Fleet 3.8.0).

* Change your SAML SSO URI to use fleet instead of kolide . This is due to the changes to Fleet's API routes outlined in the section above.

* Change configuration option `server_tlsprofile` to `server_tls_compatibility`. This options previously had an inconsistent key name.

* Replace the use of the `api/v1/fleet/spec/osquery/options` with `api/v1/fleet/config`. In Fleet 4.0.0, "osquery options" are now called "agent options." The new agent options are moved to the Fleet application config spec file and the `api/v1/fleet/config` API endpoint.

* Enroll secrets no longer have "names" and are now either global or for a specific team. Hosts no longer store the “name” of the enroll secret that was used. Users that want to be able to segment hosts (for configuration, queries, etc.) based on the enrollment secret should use the Teams feature in Fleet Basic.

* JWT encoding is no longer used for session keys. Sessions now default to expiring in 4 hours of inactivity. `auth_jwt_key` and `auth_jwt_key_file` are no longer accepted as configuration.

* The `username` artifact has been removed in favor of the more recognizable `name` (Full name). As a result the `email` artifact is now used for uniqueness in Fleet. Upon upgrading to Fleet 4.0.0, existing users will have the `name` field populated with `username`. SAML users may need to update their username mapping to match user emails.

* As of Fleet 4.0.0, Fleet Device Management Inc. periodically collects anonymous information about your instance. Sending usage statistics is turned off by default for users upgrading from a previous version of Fleet. Read more about the exact information collected [here](https://github.com/fleetdm/fleet/blob/2f42c281f98e39a72ab4a5125ecd26d303a16a6b/docs/1-Using-Fleet/11-Usage-statistics.md).

## Fleet 4.0.0 RC3 (Jun 25, 2021)

Primarily a test of the new release workflows. Relevant changelog will be updated for Fleet 4.0. 

## Fleet 4.0.0 RC2 (Jun 18, 2021)

The primary additions in Fleet 4.0.0 are the new Role-based access control (RBAC) and Teams features. 

RBAC adds the ability to define a user's access to features in Fleet. This way, more individuals in an organization can utilize Fleet with appropriate levels of access.

* Check out the [permissions documentation](https://github.com/fleetdm/fleet/blob/5e40afa8ba28fc5cdee813dfca53b84ee0ee65cd/docs/1-Using-Fleet/8-Permissions.md) for a breakdown of the new user roles.

Teams adds the ability to separate hosts into exclusive groups. This way, users can easily act on consistent groups of hosts. 

* Read more about the Teams feature in [the documentation here](https://github.com/fleetdm/fleet/blob/5e40afa8ba28fc5cdee813dfca53b84ee0ee65cd/docs/1-Using-Fleet/9-Teams.md).

### New features breakdown

* Add ability to define a user's access to features in Fleet by introducing the Admin, Maintainer, and Observer roles. Available in Fleet Core.

* Add ability to separate hosts into exclusive groups with the Teams feature. The Teams feature is available for Fleet Basic customers. Check out the list below for the new functionality included with Teams:

* Teams: Add ability to enroll hosts to one team using team specific enroll secrets.

* Teams: Add ability to manually transfer hosts to a different team in the Fleet UI.

* Teams: Add ability to apply unique agent options to each team. Note that "osquery options" have been renamed to "agent options."

* Teams: Add ability to grant users access to one or more teams. This allows you to define a user's access to specific groups of hosts in Fleet.

* Add ability to create an API-only user. API-only users cannot access the Fleet UI. These users can access all Fleet API endpoints and `fleetctl` features. Available in Fleet Core.

* Add Redis cluster support. Available in Fleet Core.

* Fix a bug that prevented the columns chosen for the "Hosts" table from persisting after logging out of Fleet.

### Upgrade plan

Fleet 4.0.0 is a major release and introduces several breaking changes and database migrations. 

* Use strictly `fleet` in Fleet's configuration, API routes, and environment variables. Users must update all usage of `kolide` in these items (deprecated since Fleet 3.8.0).

* Change configuration option `server_tlsprofile` to `server_tls_compatability`. This option previously had an inconsistent key name.

* Replace the use of the `api/v1/fleet/spec/osquery/options` with `api/v1/fleet/config`. In Fleet 4.0.0, "osquery options" are now called "agent options." The new agent options are moved to the Fleet application config spec file and the `api/v1/fleet/config` API endpoint.

* Enroll secrets no longer have "names" and are now either global or for a specific team. Hosts no longer store the “name” of the enroll secret that was used. Users that want to be able to segment hosts (for configuration, queries, etc.) based on the enrollment secret should use the Teams feature in Fleet Basic.

* `auth_jwt_key` and `auth_jwt_key_file` are no longer accepted as configuration. 

* JWT encoding is no longer used for session keys. Sessions now default to expiring in 4 hours of inactivity.

### Known issues


There are currently no known issues in this release. However, we recommend only upgrading to Fleet 4.0.0-rc2 for testing purposes. Please file a GitHub issue for any issues discovered when testing Fleet 4.0.0!

## Fleet 4.0.0 RC1 (Jun 10, 2021)

The primary additions in Fleet 4.0.0 are the new Role-based access control (RBAC) and Teams features. 

RBAC adds the ability to define a user's access to information and features in Fleet. This way, more individuals in an organization can utilize Fleet with appropriate levels of access. Check out the [permissions documentation](https://github.com/fleetdm/fleet/blob/main/docs/01-Using-Fleet/09-Permissions.md) for a breakdown of the new user roles and their respective capabilities.

Teams adds the ability to separate hosts into exclusive groups. This way, users can easily observe and apply operations to consistent groups of hosts. Read more about the Teams feature in [the documentation here](https://github.com/fleetdm/fleet/blob/main/docs/01-Using-Fleet/10-Teams.md).

There are several known issues that will be fixed for the stable release of Fleet 4.0.0. Therefore, we recommend only upgrading to Fleet 4.0.0 RC1 for testing purposes. Please file a GitHub issue for any issues discovered when testing Fleet 4.0.0!

### New features breakdown

* Add ability to define a user's access to information and features in Fleet by introducing the Admin, Maintainer, and Observer roles.

* Add ability to separate hosts into exclusive groups with the Teams feature. The Teams feature is available for Fleet Basic customers. Check out the list below for the new functionality included with Teams:

* Add ability to enroll hosts to one team using team specific enroll secrets.

* Add ability to manually transfer hosts to a different team in the Fleet UI.

* Add ability to apply unique agent options to each team. Note that "osquery options" have been renamed to "agent options."

* Add ability to grant users access to one or more teams. This allows you to define a user's access to specific groups of hosts in Fleet.

### Upgrade plan

Fleet 4.0.0 is a major release and introduces several breaking changes and database migrations. 

* Use strictly `fleet` in Fleet's configuration, API routes, and environment variables. This means that you must update all usage of `kolide` in these items. The backwards compatibility introduced in Fleet 3.8.0 is no longer valid in Fleet 4.0.0.

* Change configuration option `server_tlsprofile` to `server_tls_compatability`. This options previously had an inconsistent key name.

* Replace the use of the `api/v1/fleet/spec/osquery/options` with `api/v1/fleet/config`. In Fleet 4.0.0, "osquery options" are now called "agent options." The new agent options are moved to the Fleet application config spec file and the `api/v1/fleet/config` API endpoint.

* Enroll secrets no longer have "names" and are now either global or for a specific team. Hosts no longer store the “name” of the enroll secret that was used. Users that want to be able to segment hosts (for configuration, queries, etc.) based on the enrollment secret should use the Teams feature in Fleet Basic.

* `auth_jwt_key` and `auth_jwt_key_file` are no longer accepted as configuration. 

* JWT encoding is no longer used for session keys. Sessions now default to expiring in 4 hours of inactivity.

### Known issues

* Query packs cannot be targeted to teams.

## Fleet 3.13.0 (Jun 3, 2021)

* Improve performance of the `additional_queries` feature by moving `additional` query results into a separate table in the MySQL database. Please note that the `/api/v1/fleet/hosts` API endpoint now return only the requested `additional` columns. See documentation on the changes to the hosts API endpoint [here](https://github.com/fleetdm/fleet/blob/06b2e564e657492bfbc647e07eb49fd4efca5a03/docs/1-Using-Fleet/3-REST-API.md#list-hosts).

* Fix a bug in which running a live query in the Fleet UI would return no results and the query would seem "hung" on a small number of devices.

* Improve viewing live query errors in the Fleet UI by including the “Errors” table in the full screen view.

* Improve `fleetctl preview` experience by adding the `fleetctl preview reset` and `fleetctl preview stop` commands to reset and stop simulated hosts running in Docker.

* Add several improvements to the Fleet UI including additional contrast on checkboxes and dropdown pills.

## Fleet 3.12.0 (May 19, 2021)

* Add scheduled queries to the _Host details_ page. Surface the "Name", "Description", "Frequency", and "Last run" information for each query in a pack that apply to a specific host.

* Improve the freshness of host vitals by adding the ability to "refetch" the data on the _Host details_ page.

* Add ability to copy log fields into Google Cloud Pub/Sub attributes. This allows users to use these values for subscription filters.

* Add ability to duplicate live query results in Redis. When the `redis_duplicate_results` configuration option is set to `true`, all live query results will be copied to an additional Redis Pub/Sub channel named LQDuplicate.

* Add ability to controls the server-side HTTP keepalive property. Turning off keepalives has helped reduce outstanding TCP connections in some deployments.

* Fix an issue on the _Packs_ page in which Fleet would incorrectly handle the configured `server_url_prefix`.

## Fleet 3.11.0 (Apr 28, 2021)

* Improve Fleet performance by batch updating host seen time instead of updating synchronously. This improvement reduces MySQL CPU usage by ~33% with 4,000 simulated hosts and MySQL running in Docker.

* Add support for software inventory, introducing a list of installed software items on each host's respective _Host details_ page. This feature is flagged off by default (for now). Check out [the feature flag documentation for instructions on how to turn this feature on](./docs/02-Deploying/03-Configuration.md#software-inventory).

* Add Windows support for `fleetctl` agent autoupdates. The `fleetctl updates` command provides the ability to self-manage an agent update server. Available for Fleet Basic customers.

* Make running common queries more convenient by adding the ability to select a saved query directly from a host's respective _Host details_ page.

* Fix an issue on the _Query_ page in which Fleet would override the CMD + L browser hotkey.

* Fix an issue in which a host would display an unreasonable time in the "Last fetched" column.

## Fleet 3.10.1 (Apr 6, 2021)

* Fix a frontend bug that prevented the "Pack" page and "Edit pack" page from rendering in the Fleet UI. This issue occurred when the `platform` key, in the requested pack's configuration, was set to any value other than `darwin`, `linux`, `windows`, or `all`.

## Fleet 3.10.0 (Mar 31, 2021)

* Add `fleetctl` agent auto-updates beta which introduces the ability to self-manage an agent update server. Available for Fleet Basic customers.

* Add option for Identity Provider-Initiated (IdP-initiated) Single Sign-On (SSO).

* Improve logging. All errors are logged regardless of log level, some non-errors are logged regardless of log level (agent enrollments, runs of live queries etc.), and all other non-errors are logged on debug level.

* Improve login resilience by adding rate-limiting to login and password reset attempts and preventing user enumeration.

* Add Fleet version and Go version in the My Account page of the Fleet UI.

* Improvements to `fleetctl preview` that ensure the latest version of Fleet is fired up on every run. In addition, the Fleet UI is now accessible without having to click through browser security warning messages.

* Prefer storing IPv4 addresses for host details.

## Fleet 3.9.0 (Mar 9, 2021)

* Add configurable host identifier to help with duplicate host enrollment scenarios. By default, Fleet's behavior does not change (it uses the identifier configured in osquery's `--host_identifier` flag), but for users with overlapping host UUIDs changing `--osquery_host_identifier` to `instance` may be helpful. 

* Make cool-down period for host enrollment configurable to control load on the database in scenarios in which hosts are using the same identifier. By default, the cooldown is off, reverting to the behavior of Fleet <=3.4.0. The cooldown can be enabled with `--osquery_enroll_cooldown`.

* Refresh the Fleet UI with a new layout and horizontal navigation bar.

* Trim down the size of Fleet binaries.

* Improve handling of config_refresh values from osquery clients.

* Fix an issue with IP addresses and host additional info dropping.

## Fleet 3.8.0 (Feb 25, 2021)

* Add search, sort, and column selection in the hosts dashboard.

* Add AWS Lambda logging plugin.

* Improve messaging about number of hosts responding to live query.

* Update host listing API endpoints to support search.

* Fixes to the `fleetctl preview` experience.

* Fix `denylist` parameter in scheduled queries.

* Fix an issue with errors table rendering on live query page.

* Deprecate `KOLIDE_` environment variable prefixes in favor of `FLEET_` prefixes. Deprecated prefixes continue to work and the Fleet server will log warnings if the deprecated variable names are used. 

* Deprecate `/api/v1/kolide` routes in favor of `/api/v1/fleet`. Deprecated routes continue to work and the Fleet server will log warnings if the deprecated routes are used. 

* Add Javascript source maps for development.

## Fleet 3.7.1 (Feb 3, 2021)

* Change the default `--server_tls_compatibility` to `intermediate`. The new settings caused TLS connectivity issues for users in some environments. This new default is a more appropriate balance of security and compatibility, as recommended by Mozilla.

## Fleet 3.7.0 (Feb 3, 2021)

### This is a security release.

* **Security**: Fixed a vulnerability in which a malicious actor with a valid node key can send a badly formatted request that causes the Fleet server to exit, resulting in denial of service. See https://github.com/fleetdm/fleet/security/advisories/GHSA-xwh8-9p3f-3x45 and the linked content within that advisory.

* Add new Host details page which includes a rich view of a specific host’s attributes.

* Reveal live query errors in the Fleet UI and `fleetctl` to help target and diagnose hosts that fail.

* Add Helm chart to make it easier for users to deploy to Kubernetes.

* Add support for `denylist` parameter in scheduled queries.

* Add debug flag to `fleetctl` that enables logging of HTTP requests and responses to stderr.

* Improvements to the `fleetctl preview` experience that include adding containerized osquery agents, displaying login information, creating a default directory, and checking for Docker daemon status.

* Add improved error handling in host enrollment to make debugging issues with the enrollment process easier.

* Upgrade TLS compatibility settings to match Mozilla.

* Add comments in generated flagfile to add clarity to different features being configured.

* Fix a bug in Fleet UI that allowed user to edit a scheduled query after it had been deleted from a pack.


## Fleet 3.6.0 (Jan 7, 2021)

* Add the option to set up an S3 bucket as the storage backend for file carving.

* Build Docker container with Fleet running as non-root user.

* Add support to read in the MySQL password and JWT key from a file.

* Improve the `fleetctl preview` experience by automatically completing the setup process and configuring fleetctl for users.

* Restructure the documentation into three top-level sections titled "Using Fleet," "Deployment," and "Contribution."

* Fix a bug that allowed hosts to enroll with an empty enroll secret in new installations before setup was completed.

* Fix a bug that made the query editor render strangely in Safari.

## Fleet 3.5.1 (Dec 14, 2020)

### This is a security release.

* **Security**: Introduce XML validation library to mitigate Go stdlib XML parsing vulnerability effecting SSO login. See https://github.com/fleetdm/fleet/security/advisories/GHSA-w3wf-cfx3-6gcx and the linked content within that advisory.

Follow up: Rotate `--auth_jwt_key` to invalidate existing sessions. Audit for suspicious activity in the Fleet server.

* **Security**: Prevent new queries from using the SQLite `ATTACH` command. This is a mitigation for the osquery vulnerability https://github.com/osquery/osquery/security/advisories/GHSA-4g56-2482-x7q8.

Follow up: Audit existing saved queries and logs of live query executions for possible malicious use of `ATTACH`. Upgrade osquery to 4.6.0 to prevent `ATTACH` queries from executing.

* Update icons and fix hosts dashboard for wide screen sizes.

## Fleet 3.5.0 (Dec 10, 2020)

* Refresh the Fleet UI with new colors, fonts, and Fleet logos.

* All releases going forward will have the fleectl.exe.zip on the release page.

* Add documentation for the authentication Fleet REST API endpoints.

* Add FAQ answers about the stress test results for Fleet, configuring labels, and resetting auth tokens.

* Fixed a performance issue users encountered when multiple hosts shared the same UUID by adding a one minute cooldown.

* Improve the `fleetctl preview` startup experience.

* Fix a bug preventing the same query from being added to a scheduled pack more than once in the Fleet UI.


## Fleet 3.4.0 (Nov 18, 2020)

* Add NPM installer for `fleetctl`. Install via `npm install -g osquery-fleetctl`.

* Add `fleetctl preview` command to start a local test instance of the Fleet server with Docker.

* Add `fleetctl debug` commands and API endpoints for debugging server performance.

* Add additional_info_filters parameter to get hosts API endpoint for filtering returned additional_info.

* Updated package import paths from github.com/kolide/fleet to github.com/fleetdm/fleet.

* Add first of the Fleet REST API documentation.

* Add documentation on monitoring with Prometheus.

* Add documentation to FAQ for debugging database connection errors.

* Fix fleetctl Windows compatibility issues.

* Fix a bug preventing usernames from containing the @ symbol.

* Fix a bug in 3.3.0 in which there was an unexpected database migration warning.

## Fleet 3.3.0 (Nov 05, 2020)

With this release, Fleet has moved to the new github.com/fleetdm/fleet
repository. Please follow changes and releases there.

* Add file carving functionality.

* Add `fleetctl user create` command.

* Add osquery options editor to admin pages in UI.

* Add `fleetctl query --pretty` option for pretty-printing query results. 

* Add ability to disable packs with `fleetctl apply`.

* Improve "Add New Host" dialog to walk the user step-by-step through host enrollment.

* Improve 500 error page by allowing display of the error.

* Partial transition of branding away from "Kolide Fleet".

* Fix an issue with case insensitive enroll secret and node key authentication.

* Fix an issue with `fleetctl query --quiet` flag not actually suppressing output.


## Fleet 3.2.0 (Aug 08, 2020)

* Add `stdout` logging plugin.

* Add AWS `kinesis` logging plugin.

* Add compression option for `filesystem` logging plugin.

* Add support for Redis TLS connections.

* Add osquery host identifier to EnrollAgent logs.

* Add osquery version information to output of `fleetctl get hosts`.

* Add hostname to UI delete host confirmation modal.

* Update osquery schema to 4.5.0.

* Update osquery versions available in schedule query UI.

* Update MySQL driver.

* Remove support for (previously deprecated) `old` TLS profile.

* Fix cleanup of queries in bad state. This should resolve issues in which users experienced old live queries repeatedly returned to hosts. 

* Fix output kind of `fleetctl get options`.

## Fleet 3.1.0 (Aug 06, 2020)

* Add configuration option to set Redis database (`--redis_database`).

* Add configuration option to set MySQL connection max lifetime (`--mysql_conn_max_lifetime`).

* Add support for printing a single enroll secret by name.

* Fix bug with label_type in older fleetctl yaml syntax.

* Fix bug with URL prefix and Edit Pack button. 

## Kolide Fleet 3.0.0 (Jul 23, 2020)

* Backend performance overhaul. The Fleet server can now handle hundreds of thousands of connected hosts.

* Pagination implemented in the web UI. This makes the UI usable for any host count supported by the backend.

* Add capability to collect "additional" information from hosts. Additional queries can be set to be updated along with the host detail queries. This additional information is returned by the API.

* Removed extraneous network interface information to optimize server performance. Users that require this information can use the additional queries functionality to retrieve it.

* Add "manual" labels implementation. Static labels can be set by providing a list of hostnames with `fleetctl`.

* Add JSON output for `fleetctl get` commands.

* Add `fleetctl get host` to retrieve details for a single host.

* Update table schema for osquery 4.4.0.

* Add support for multiple enroll secrets.

* Logging verbosity reduced by default. Logs are now much less noisy.

* Fix import of github.com/kolide/fleet Go packages for consumers outside of this repository.

## Kolide Fleet 2.6.0 (Mar 24, 2020)

* Add server logging for X-Forwarded-For header.

* Add `--osquery_detail_update_interval` to set interval of host detail updates.
  Set this (along with `--osquery_label_update_interval`) to a longer interval
  to reduce server load in large deployments.

* Fix MySQL deadlock errors by adding retries and backoff to transactions.

## Kolide Fleet 2.5.0 (Jan 26, 2020)

* Add `fleetctl goquery` command to bring up the github.com/AbGuthrie/goquery CLI.

* Add ability to disable live queries in web UI and `fleetctl`.

* Add `--query-name` option to `fleetctl query`. This allows using the SQL from a saved query.

* Add `--mysql-protocol` flag to allow connection to MySQL by domain socket.

* Improve server logging. Add logging for creation of live queries. Add username information to logging for other endpoints.

* Allow CREATE queries in the web UI.

* Fix a bug in which `fleetctl query` would exit before any results were returned when latency to the Fleet server was high.

* Fix an error initializing the Fleet database when MySQL does not have event permissions.

* Deprecate "old" TLS profile.

## Kolide Fleet 2.4.0 (Nov 12, 2019)

* Add `--server_url_prefix` flag to configure a URL prefix to prepend on all Fleet URLs. This can be useful to run fleet behind a reverse-proxy on a hostname shared with other services.

* Add option to automatically expire hosts that have not checked in within a certain number of days. Configure this in the "Advanced Options" of "App Settings" in the browser UI.

* Add ability to search for hosts by UUID when targeting queries.

* Allow SAML IdP name to be as short as 4 characters.

## Kolide Fleet 2.3.0 (Aug 14, 2019)

### This is a security release.

* Security: Upgrade Go to 1.12.8 to fix CVE-2019-9512, CVE-2019-9514, and CVE-2019-14809.

* Add capability to export packs, labels, and queries as yaml in `fleetctl get` with the `--yaml` flag. Include queries with a pack using `--with-queries`.

* Modify email templates to load image assets from Github CDN rather than Fleet server (fixes broken images in emails when Fleet server is not accessible from email clients).

* Add warning in query UI when Redis is not functioning.

* Fix minor bugs in frontend handling of scheduled queries.

* Minor styling changes to frontend.


## Kolide Fleet 2.2.0 (Jul 16, 2019)

* Add GCP PubSub logging plugin. Thanks to Michael Samuel for adding this capability.

* Improved escaping for target search in live query interface. It is now easier to target hosts with + and - characters in the name.

* Server and browser performance improvements by reduced loading of hosts in frontend. Host status will only update on page load when over 100 hosts are present.

* Utilize details sent by osquery in enrollment request to more quickly display details of new hosts. Also fixes a bug in which hosts could not complete enrollment if certain platform-dependent options were used.

* Fix a bug in which the default query runs after targets are edited.

## Kolide Fleet 2.1.2 (May 30, 2019)

* Prevent sending of SMTP credentials over insecure connection

* Prefix generated SAML IDs with 'id' (improves compatibility with some IdPs)

## Kolide Fleet 2.1.1 (Apr 25, 2019)

* Automatically pull AWS STS credentials for Firehose logging if they are not specified in config.

* Fix bug in which log output did not include newlines separating characters.

* Fix bug in which the default live query was run when navigating to a query by URL.

* Update logic for setting primary NIC to ignore link-local or loopback interfaces.

* Disable editing of logged in user email in admin panel (instead, use the "Account Settings" menu in top left).

* Fix a panic resulting from an invalid config file path.

## Kolide Fleet 2.1.0 (Apr 9, 2019)

* Add capability to log osquery status and results to AWS Firehose. Note that this deprecated some existing logging configuration (`--osquery_status_log_file` and `--osquery_result_log_file`). Existing configurations will continue to work, but will be removed at some point.

* Automatically clean up "incoming hosts" that do not complete enrollment.

* Fix bug with SSO requests that caused issues with some IdPs.

* Hide built-in platform labels that have no hosts.

* Fix references to Fleet documentation in emails.

* Minor improvements to UI in places where editing objects is disabled.

## Kolide Fleet 2.0.2 (Jan 17, 2019)

* Improve performance of `fleetctl query` with high host counts.

* Add `fleetctl get hosts` command to retrieve a list of enrolled hosts.

* Add support for Login SMTP authentication method (Used by Office365).

* Add `--timeout` flag to `fleetctl query`.

* Add query editor support for control-return shortcut to run query.

* Allow preselection of hosts by UUID in query page URL parameters.

* Allow username to be specified in `fleetctl setup`. Default behavior remains to use email as username.

* Fix conversion of integers in `fleetctl convert`.

* Upgrade major Javascript dependencies.

* Fix a bug in which query name had to be specified in pack yaml.

## Kolide Fleet 2.0.1 (Nov 26, 2018)

* Fix a bug in which deleted queries appeared in pack specs returned by fleetctl.

* Fix a bug getting entities with spaces in the name.

## Kolide Fleet 2.0.0 (Oct 16, 2018)

* Stable release of Fleet 2.0.

* Support custom certificate authorities in fleetctl client.

* Add support for MySQL 8 authentication methods.

* Allow INSERT queries in editor.

* Update UI styles.

* Fix a bug causing migration errors in certain environments.

See changelogs for release candidates below to get full differences from 1.0.9
to 2.0.0.

## Kolide Fleet 2.0.0 RC5 (Sep 18, 2018)

* Fix a security vulnerability that would allow a non-admin user to elevate privileges to admin level.

* Fix a security vulnerability that would allow a non-admin user to modify other user's details.

* Reduce the information that could be gained by an admin user trying to port scan the network through the SMTP configuration.

* Refactor and add testing to authorization code.

## Kolide Fleet 2.0.0 RC4 (August 14, 2018)

* Expose the API token (to be used with fleetctl) in the UI.

* Update autocompletion values in the query editor.

* Fix a longstanding bug that caused pack targets to sometimes update incorrectly in the UI.

* Fix a bug that prevented deletion of labels in the UI.

* Fix error some users encountered when migrating packs (due to deleted scheduled queries).

* Update favicon and UI styles.

* Handle newlines in pack JSON with `fleetctl convert`.

* Improve UX of fleetctl tool.

* Fix a bug in which the UI displayed the incorrect logging type for scheduled queries.

* Add support for SAML providers with whitespace in the X509 certificate.

* Fix targeting of packs to individual hosts in the UI.

## Kolide Fleet 2.0.0 RC3 (June 21, 2018)

* Fix a bug where duplicate queries were being created in the same pack but only one was ever delivered to osquery. A migration was added to delete duplicate queries in packs created by the UI.
  * It is possible to schedule the same query with different options in one pack, but only via the CLI.
  * If you thought you were relying on this functionality via the UI, note that duplicate queries will be deleted when you run migrations as apart of a cleanup fix. Please check your configurations and make sure to create any double-scheduled queries via the CLI moving forward.

* Fix a bug in which packs created in UI could not be loaded by fleetctl.

* Fix a bug where deleting a query would not delete it from the packs that the query was scheduled in.

## Kolide Fleet 2.0.0 RC2 (June 18, 2018)

* Fix errors when creating and modifying packs, queries and labels in UI.

* Fix an issue with the schema of returned config JSON.

* Handle newlines when converting query packs with fleetctl convert.

* Add last seen time hover tooltip in Fleet UI.

* Fix a null pointer error when live querying via fleetctl.

* Explicitly set timezone in MySQL connection (improves timestamp consistency).

* Allow native password auth for MySQL (improves compatibility with Amazon RDS).

## Kolide Fleet 2.0.0 (currently preparing for release)

The primary new addition in Fleet 2 is the new `fleetctl` CLI and file-format, which dramatically increases the flexibility and control that administrators have over their osquery deployment. The CLI and the file format are documented [in the Fleet documentation](https://github.com/fleetdm/fleet/blob/main/docs/01-Using-Fleet/02-fleetctl-CLI.md).

### New Features

* New `fleetctl` CLI for managing your entire osquery workflow via CLI, API, and source controlled files!
  * You can use `fleetctl` to manage osquery packs, queries, labels, and configuration.

* In addition to the CLI, Fleet 2.0.0 introduces a new file format for articulating labels, queries, packs, options, etc. This format is designed for composability, enabling more effective sharing and re-use of intelligence.

```yaml
apiVersion: v1
kind: query
spec:
  name: pending_updates
  query: >
    select value
    from plist
    where
      path = "/Library/Preferences/ManagedInstalls.plist" and
      key = "PendingUpdateCount" and
      value > "0";
```

* Run live osquery queries against arbitrary subsets of your infrastructure via the `fleetctl query` command.

* Use `fleetctl setup`, `fleetctl login`, and `fleetctl logout` to manage the authentication life-cycle via the CLI.

* Use `fleetctl get`, `fleetctl apply`, and `fleetctl delete` to manage the state of your Fleet data.

* Manage any osquery option you want and set platform-specific overrides with the `fleetctl` CLI and file format.

### Upgrade Plan

* Managing osquery options via the UI has been removed in favor of the more flexible solution provided by the CLI. If you have customized your osquery options with Fleet, there is [a database migration](./server/datastore/mysql/migrations/data/20171212182458_MigrateOsqueryOptions.go) which will port your existing data into the new format when you run `fleet prepare db`. To download your osquery options after migrating your database, run `fleetctl get options > options.yaml`. Further modifications to your options should occur in this file and it should be applied with `fleetctl apply -f ./options.yaml`.

## Kolide Fleet 1.0.8 (May 3, 2018)

* Osquery 3.0+ compatibility!

* Include RFC822 From header in emails (for email authentication)

## Kolide Fleet 1.0.7 (Mar 30, 2018)

* Support FileAccesses in FIM configuration.

* Populate network interfaces on windows hosts in host view.

* Add flags for configuring MySQL connection pooling limits.

* Fixed bug in which shard and removed keys are dropped in query packs returned to osquery clients.

* Fixed handling of status logs with unexpected fields.

## Kolide Fleet 1.0.6 (Dec 4, 2017)

* Added remote IP in the logs for all osqueryd/launcher requests. (#1653)

* Fixed bugs that caused logs to sometimes be omitted from the logwriter. (#1636, #1617)

* Fixed a bug where request bodies were not being explicitly closed. (#1613)

* Fixed a bug where SAML client would create too many HTTP connections. (#1587)

* Fixed bug in which default query was run instead of entered query. (#1611)

* Added pagination to the Host browser pages for increased performance. (#1594)

* Fixed bug rendering hosts when clock speed cannot be parsed. (#1604)

## Kolide Fleet 1.0.5 (Oct 17, 2017)

* Renamed the binary from kolide to fleet

* Add support for Kolide Launcher managed osquery nodes

* Remove license requirements

* Updated documentation link in the sidebar to point to public GitHub documentation

* Added FIM support

* Title on query page correctly reflects new or edit mode.

* Fixed issue on new query page where last query would be submitted instead of current.

* Fixed issue where user menu did not work on Firefox browser

* Fixed issue cause SSO to fail for ADFS

* Fixed issue validating signatures in nested SAML assertions.

## Kolide 1.0.4 (Jun 1, 2017)

* Added feature that allows users to import existing Osquery configuration files using the [configimporter](https://github.com/kolide/configimporter) utility.

* Added support for Osquery decorators.

* Added SAML single sign on support.

* Improved online status detection.

  The Kolide server now tracks the `distributed_interval` and `config_tls_refresh` values for each individual host (these can be different if they are set via flagfile and not through Kolide), to ensure that online status is represented as accurately as possible.

* Kolide server now requires `--auth_jwt_key` to be specified at startup.

  If no JWT key is provided by the user, the server will print a new suggested random JWT key for use.

* Fixed bug in which deleted packs were still displayed on the query sidebar.

* Fixed rounding error when showing % of online hosts.

* Removed --app_token_key flag.

* Fixed issue where heavily loaded database caused host authentication failures.

* Fixed issue where osquery sends empty strings for integer values in log results.

## Kolide 1.0.3 (April 3, 2017)

* Log rotation is no longer the default setting for Osquery status and results logs. To enable log rotation use the `--osquery_enable_log_rotation` flag.

* Add a debug endpoint for collecting performance statistics and profiles.

  When `kolide serve --debug` is used, additional handlers will be started to provide access to profiling tools. These endpoints are authenticated with a randomly generated token that is printed to the Kolide logs at startup. These profiling tools are not intended for general use, but they may be useful when providing performance-related bug reports to the Kolide developers.

* Add a workaround for CentOS6 detection.

  osquery 2.3.2 incorrectly reports an empty value for `platform` on CentOS6 hosts. We added a workaround to properly detect platform in Kolide, and also [submitted a fix](https://github.com/facebook/osquery/pull/3071) to upstream osquery.

* Ensure hosts enroll in labels immediately even when `distributed_interval` is set to a long interval.

* Optimizations reduce the CPU and DB usage of the manage hosts page.

* Manage packs page now loads much quicker when a large number of hosts are enrolled.

* Fixed bug with the "Reset Options" button.

* Fixed 500 error resulting from saving unchanged options.

* Improved validation for SMTP settings.

* Added command line support for `modern`, `intermediate`, and `old` TLS configuration
profiles. The profile is set using the following command line argument.
```
--server_tls_compatibility=modern
```
See https://wiki.mozilla.org/Security/Server_Side_TLS for more information on the different profile options.

* The Options Configuration item in the sidebar is now only available to admin users.

  Previously this item was visible to non-admin users and if selected, a blank options page would be displayed since server side authorization constraints prevent regular users from viewing or changing options.

* Improved validation for the Kolide server URL supplied in setup and configuration.

* Fixed an issue importing osquery configurations with numeric values represented as strings in JSON.

## Kolide 1.0.2 (March 14, 2017)

* Fix an issue adding additional targets when querying a host

* Show loading spinner while newly added Host Details are saved

* Show a generic computer icon when when referring to hosts with an unknown platform instead of the text "All"

* Kolide will now warn on startup if there are database migrations not yet completed.

* Kolide will prompt for confirmation before running database migrations.

  To disable this, use `kolide prepare db --no-prompt`.

* Kolide now supports emoji, so you can 🔥 to your heart's content.

* When setting the platform for a scheduled query, selecting "All" now clears individually selected platforms.

* Update Host details cards UI

* Lower HTTP timeout settings.

  In an effort to provide a more resilient web server, timeouts are more strictly enforced by the Kolide HTTP server (regardless of whether or not you're using the built-in TLS termination).

* Harden TLS server settings.

  For customers using Kolide's built-in TLS server (if the `server.tls` configuration is `true`), the server was hardened to only accept modern cipher suites as recommended by [Mozilla](https://wiki.mozilla.org/Security/Server_Side_TLS#Modern_compatibility).

* Improve the mechanism used to calculate whether or not hosts are online.

  Previously, hosts were categorized as "online" if they had been seen within the past 30 minutes. To make the "online" status more representative of reality, hosts are marked "online" if the Kolide server has heard from them within two times the lowest polling interval as described by the Kolide-managed osquery configuration. For example, if you've configured osqueryd to check-in with Kolide every 10 seconds, only hosts that Kolide has heard from within the last 20 seconds will be marked "online".

* Update Host details cards UI

* Add support for rotating the osquery status and result log files by sending a SIGHUP signal to the kolide process.

* Fix Distributed Query compatibility with load balancers and Safari.

  Customers running Kolide behind a web balancer lacking support for websockets were unable to use the distributed query feature. Also, in certain circumstances, Safari users with a self-signed cert for Kolide would receive an error. This release add a fallback mechanism from websockets using SockJS for improved compatibility.

* Fix issue with Distributed Query Pack results full screen feature that broke the browser scrolling abilities.

* Fix bug in which host counts in the sidebar did not match up with displayed hosts.

## Kolide 1.0.1 (February 27, 2017)

* Fix an issue that prevented users from replacing deleted labels with a new label of the same name.

* Improve the reliability of IP and MAC address data in the host cards and table.

* Add full screen support for distributed query results.

* Enable users to double click on queries and packs in a table to see their details.

* Reprompt for a password when a user attempts to change their email address.

* Automatically decorate the status and result logs with the host's UUID and hostname.

* Fix an issue where Kolide users on Safari were unable to delete queries or packs.

* Improve platform detection accuracy.

  Previously Kolide was determining platform based on the OS of the system osquery was built on instead of the OS it was running on. Please note: Offline hosts may continue to report an erroneous platform until they check-in with Kolide.

* Fix bugs where query links in the pack sidebar pointed to the wrong queries.

* Improve MySQL compatibility with stricter configurations.

* Allow users to edit the name and description of host labels.

* Add basic table autocompletion when typing in the query composer.

* Support MySQL client certificate authentication. More details can be found in the [Configuring the Fleet binary docs](./docs/infrastructure/configuring-the-fleet-binary.md).

* Improve security for user-initiated email address changes.

  This improvement ensures that only users who own an email address and are logged in as the user who initiated the change can confirm the new email.

  Previously it was possible for Administrators to also confirm these changes by clicking the confirmation link.

* Fix an issue where the setup form rejects passwords with certain characters.

  This change resolves an issue where certain special characters like "." where rejected by the client-side JS that controls the setup form.

* Automatically login the user once initial setup is completed.

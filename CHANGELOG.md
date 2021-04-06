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

The primary new addition in Fleet 2 is the new `fleetctl` CLI and file-format, which dramatically increases the flexibility and control that administrators have over their osquery deployment. The CLI and the file format are documented [in the Fleet documentation](https://github.com/fleetdm/fleet/blob/master/docs/1-Using-Fleet/2-fleetctl-CLI.md).

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

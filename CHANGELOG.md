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

The primary new addition in Fleet 2 is the new `fleetctl` CLI and file-format, which dramatically increases the flexibility and control that administrators have over their osquery deployment. The CLI and the file format are documented [in the Fleet documentation](https://github.com/kolide/fleet/blob/master/docs/cli/README.md).

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

* Managing osquery options via the UI has been removed in favor of the more flexible solution provided by the CLI. If you have customized your osquery options with Fleet, there is [a database migration](server/datastore/mysql/migrations/data/20171212182458_MigrateOsqueryOptions.go) which will port your existing data into the new format when you run `fleet prepare db`. To download your osquery options after migrating your database, run `fleetctl get options > options.yaml`. Further modifications to your options should occur in this file and it should be applied with `fleetctl apply -f ./options.yaml`.

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

  In an effort to provide a more resilient web server, timeouts are more strictly enforced by the Kolide HTTP server (regardless of whether or not you're using the built-in TLS termination). If your Kolide environment is particularly latent and you observe requests timing out, contact us at [help@kolide.co](mailto:help@kolide.co).

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

* Support MySQL client certificate authentication. More details can be found in the [Configuring the Kolide binary docs](https://docs.kolide.co/kolide/1.0.1/infrastructure/configuring-the-kolide-binary.html)

* Improve security for user-initiated email address changes.

  This improvement ensures that only users who own an email address and are logged in as the user who initiated the change can confirm the new email.

  Previously it was possible for Administrators to also confirm these changes by clicking the confirmation link.

* Fix an issue where the setup form rejects passwords with certain characters.

  This change resolves an issue where certain special characters like "." where rejected by the client-side JS that controls the setup form.

* Automatically login the user once initial setup is completed.

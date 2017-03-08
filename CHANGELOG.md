*   Show a generic computer icon when when referring to hosts with an unknown platform instead of the text "All"

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

  Previously Kolide was determing platform based on the OS of the system osquery was built on instead of the OS it was running on. Please note: Offline hosts may continue to report an erroneous platform until they check-in with Kolide.

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
